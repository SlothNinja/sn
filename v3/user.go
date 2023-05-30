package sn

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var ErrMissingKey = errors.New("missing key")

type User struct {
	Key *datastore.Key `datastore:"__key__"`
	Data
}

type Data struct {
	Name               string
	LCName             string
	Email              string
	EmailHash          string
	EmailNotifications bool
	EmailReminders     bool
	GoogleID           string
	XMPPNotifications  bool
	GravType           string
	Admin              bool
	Joined             time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type readMap map[string]int

func (cl Client[G, I, P]) CuHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.Current(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		if cu.IsZero() {
			ctx.JSON(http.StatusOK, gin.H{"CU": nil})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"CU": cu})
	}
}

func EmailHash(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.New()
	_, err := hash.Write([]byte(email))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

type UID int64

const noUID UID = 0

func (u User) IsZero() bool {
	return u == User{}
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Admin
}

const (
	kind            = "User"
	uidParam        = "uid"
	guserKey        = "guser"
	currentKey      = "current"
	userKey         = "User"
	salt            = "slothninja"
	usersKey        = "Users"
	USER_PROJECT_ID = "USER_PROJECT_ID"
	DS_USER_HOST    = "DS_USER_HOST"
)

type UserName struct {
	GoogleID string
}

var (
	ErrUserNotFound = errors.New("user not found.")
	ErrTooManyFound = errors.New("Found too many users.")
)

func userRootKey() *datastore.Key {
	return datastore.NameKey("Users", "root", nil)
}

func newUserKey(uid UID) *datastore.Key {
	return datastore.IDKey(kind, int64(uid), userRootKey())
}

func NewUser(uid UID) User {
	return User{Key: newUserKey(uid)}
}

func (u User) ID() UID {
	if u.Key == nil {
		return 0
	}
	return UID(u.Key.ID)
}

func GenID(gid string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(salt+gid)))
}

// func NewKeyFor(id int64) *datastore.Key {
// 	u := NewUser(id)
// 	return u.Key
// }

func AllUserQuery(c *gin.Context) *datastore.Query {
	return datastore.NewQuery(kind).Ancestor(userRootKey())
}

// func MCKey(c *gin.Context, gid string) string {
// 	return sn.VersionID() + gid
// }

func (u *User) Gravatar(size string) template.URL {
	return template.URL(GravatarURL(u.Email, size, u.GravType))
}

func GravatarURL(email, size, gravType string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.New()
	hash.Write([]byte(email))
	md5string := fmt.Sprintf("%x", hash.Sum(nil))
	if gravType == "" || gravType == "personal" {
		return fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=%s&d=monsterid", md5string, size)
	}
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=%s&d=%s&f=y", md5string, size, gravType)
}

func (cl Client[G, I, P]) Update(c *gin.Context, cu, u1, u2 User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if isAdmin(cu) {
		cl.Log.Debugf("is admin")
		if u2.Email != "" {
			cl.Log.Debugf("updating email")
			u1.Email = u2.Email

			hash, err := EmailHash(u1.Email)
			if err != nil {
				return err
			}
			u1.EmailHash = hash
		}

		err := cl.updateName(c, u1, u2.Name)
		if err != nil {
			return err
		}
	}

	if isAdmin(cu) || (cu.ID() == u1.ID()) {
		cl.Log.Debugf("is admin or current")
		cl.Log.Debugf("updating emailNotifications and gravType")
		u1.EmailReminders = u2.EmailReminders
		u1.EmailNotifications = u2.EmailNotifications
		u1.GravType = u2.GravType
		hash, err := EmailHash(u1.Email)
		if err != nil {
			return err
		}
		u1.EmailHash = hash
	}

	return nil
}

func (cl Client[G, I, P]) updateName(c *gin.Context, u User, n string) error {
	matcher := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._%+\-]+$`)

	switch {
	case n == u.Name:
		return nil
	case len(n) > 15:
		return fmt.Errorf("%q is too long.", n)
	case !matcher.MatchString(n):
		return fmt.Errorf("%q is not a valid user name.", n)
	default:
		uniq, err := cl.NameIsUnique(c, n)
		if err != nil {
			return err
		}
		if !uniq {
			return fmt.Errorf("%q is not a unique user name.", n)
		}
		u.Name = n
		u.LCName = strings.ToLower(n)
		return nil
	}
}

func (cl Client[G, I, P]) NameIsUnique(c *gin.Context, name string) (bool, error) {
	LCName := strings.ToLower(name)

	q := datastore.NewQuery("User").Filter("LCName=", LCName)

	cnt, err := cl.User.Count(c, q)
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func (u *User) Equal(u2 *User) bool {
	return u2 != nil && u != nil && u.ID() == u2.ID()
}

// func (u *User) Link() template.HTML {
// 	if u == nil {
// 		return ""
// 	}
// 	return LinkFor(u.ID(), u.Name)
// }
//
// func LinkFor(uid int64, name string) template.HTML {
// 	return template.HTML(fmt.Sprintf("<a href=%q>%s</a>", PathFor(uid), name))
// }
//
// func PathFor(uid int64) template.HTML {
// 	return template.HTML(fmt.Sprintf("%s/#/show/%d", getUserHostURL(), uid))
// }
//
// func (client *UserClient) Fetch(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	uid, err := getUID(c, uidParam)
// 	if err != nil || uid == NotFound {
// 		client.Log.Errorf(err.Error())
// 		c.Redirect(http.StatusSeeOther, "/")
// 		c.Abort()
// 		return
// 	}
//
// 	u, err := client.Get(c, uid)
// 	if err != nil {
// 		client.Log.Errorf("Unable to get user for id: %v", c.Param("uid"))
// 		c.Redirect(http.StatusSeeOther, "/")
// 		c.Abort()
// 		return
// 	}
// 	WithUser(c, u)
// }
//
// func (client *UserClient) FetchAll(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	us, cnt, err := client.getFiltered(c, c.PostForm("start"), c.PostForm("length"))
//
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		c.Abort()
// 	}
// 	withUsers(withCount(c, cnt), us)
// }

func (cl Client[G, I, P]) getFiltered(c *gin.Context, start, length string) ([]User, int64, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	q := AllUserQuery(c).KeysOnly()
	icnt, err := cl.User.Count(c, q)
	if err != nil {
		return nil, 0, err
	}
	cnt := int64(icnt)

	if start != "" {
		if st, err := strconv.ParseInt(start, 10, 32); err == nil {
			q = q.Offset(int(st))
		}
	}

	if length != "" {
		if l, err := strconv.ParseInt(length, 10, 32); err == nil {
			q = q.Limit(int(l))
		}
	}

	ks, err := cl.User.GetAll(c, q, nil)
	if err != nil {
		return nil, 0, err
	}

	var us []User
	for i := range ks {
		id := ks[i].ID
		if id != 0 {
			us = append(us, NewUser(UID(id)))
		}
	}

	err = cl.User.GetMulti(c, ks, us)
	return us, cnt, err
}

func getUID(c *gin.Context, param string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		return NotFound, err
	}
	return id, nil
}

// func Fetched(c *gin.Context) *User {
// 	return From(c)
// }
//
// func Gravatar(u *User, size string) template.HTML {
// 	return template.HTML(fmt.Sprintf(`<a href=%q ><img src=%q alt="Gravatar" class="black-border" /></a>`, PathFor(u.ID()), u.Gravatar(size)))
// }
//
// func from(c *gin.Context, key string) (u *User) {
// 	u, _ = c.Value(key).(*User)
// 	return
// }
//
// func From(c *gin.Context) *User {
// 	return from(c, userKey)
// }

var ErrMissingToken = fmt.Errorf("missing token")

func (cl Client[G, I, P]) Current(c *gin.Context) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	session := sessions.Default(c)
	token, ok := SessionTokenFrom(session)
	if !ok {
		return User{}, ErrMissingToken
	}

	return cl.Get(c, token.ID)
}

// func WithUser(c *gin.Context, u *User) {
// 	c.Set(userKey, u)
// }
//
// func WithCurrent(c *gin.Context, u *User) {
// 	c.Set(currentKey, u)
// }
//
// func UsersFrom(c *gin.Context) []*User {
// 	us, _ := c.Value(usersKey).([]*User)
// 	return us
// }
//
// func withUsers(c *gin.Context, us []*User) {
// 	c.Set(usersKey, us)
// }
//
// func withCount(c *gin.Context, cnt int64) *gin.Context {
// 	c.Set(countKey, cnt)
// 	return c
// }
//
// func CountFrom(c *gin.Context) (cnt int64) {
// 	cnt, _ = c.Value(countKey).(int64)
// 	return
// }

func (u User) MarshalJSON() ([]byte, error) {
	type usr User
	return json.Marshal(struct {
		usr
		ID UID
	}{
		usr: usr(u),
		ID:  u.ID(),
	})
}

func (cl Client[G, I, P]) Get(c *gin.Context, uid UID) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.get(c, newUserKey(uid))
}

func (cl Client[G, I, P]) get(c *gin.Context, k *datastore.Key) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	u, err := cl.mcGet(k)
	if err == nil {
		return u, nil
	}

	return cl.dsGet(c, k)
}

func (cl Client[G, I, P]) mcGet(k *datastore.Key) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if k == nil {
		return User{}, ErrMissingKey
	}

	item, found := cl.Cache.Get(k.Encode())
	if !found {
		return User{}, ErrUserNotFound
	}

	u, ok := item.(User)
	if !ok {
		return User{}, ErrInvalidCache
	}
	return u, nil
}

func (cl Client[G, I, P]) mcGetMulti(ks []*datastore.Key) ([]User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	l := len(ks)
	if l == 0 {
		return nil, ErrMissingKey
	}

	me := make(datastore.MultiError, l)
	us := make([]User, l)
	isNil := true
	for i, k := range ks {
		us[i], me[i] = cl.mcGet(k)
		if me[i] != nil {
			isNil = false
		}
	}

	if isNil {
		return us, nil
	}
	return us, me
}

func (cl Client[G, I, P]) dsGet(c *gin.Context, k *datastore.Key) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if k == nil {
		return User{}, ErrMissingKey
	}

	var u User
	err := cl.User.Get(c, k, &u)
	if err != nil {
		return User{}, err
	}
	cl.cacheUser(u)
	return u, nil
}

func (cl Client[G, I, P]) dsGetMulti(c *gin.Context, ks []*datastore.Key) ([]User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	l := len(ks)
	if l == 0 {
		return nil, ErrMissingKey
	}

	us := make([]User, l)
	err := cl.User.GetMulti(c, ks, us)
	if err != nil {
		return us, err
	}
	for _, u := range us {
		cl.cacheUser(u)
	}
	return us, nil
}

func (cl Client[G, I, P]) cacheUser(u User) {
	if u.Key == nil {
		return
	}
	cl.Cache.SetDefault(u.Key.Encode(), u)
}

func (cl Client[G, I, P]) GetMulti(c *gin.Context, uids []UID) ([]User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	ks := make([]*datastore.Key, len(uids))
	for i, id := range uids {
		ks[i] = newUserKey(id)
	}

	us, me := cl.mcGetMulti(ks)
	if me == nil {
		return us, nil
	}

	return cl.dsGetMulti(c, ks)
}

// func (cl Client) AllocateIDs(c *gin.Context, ks []*datastore.Key) ([]*datastore.Key, error) {
// 	return cl.DS.AllocateIDs(c, ks)
// }

func (cl Client[G, I, P]) Put(c *gin.Context, k *datastore.Key, u User) (*datastore.Key, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.putUserByKey(c, k, u)
}

func (cl Client[G, I, P]) putUserByKey(c *gin.Context, k *datastore.Key, u User) (*datastore.Key, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	k, err := cl.User.Put(c, k, u)
	if err != nil {
		return nil, err
	}
	cl.cacheUser(u)
	return k, nil
}

// func (cl Client) RunInTransaction(c *gin.Context, f func(*datastore.Transaction) error, opts ...datastore.TransactionOption) (*datastore.Commit, error) {
// 	return cl.DS.RunInTransaction(c, f, opts...)
// }
