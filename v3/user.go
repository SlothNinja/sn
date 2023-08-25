package sn

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
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

func (cl Client) cuHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.currentUser(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		if cu.IsZero() {
			ctx.JSON(http.StatusOK, gin.H{"CU": nil})
			return
		}

		tokenKey := getFSTokenKey()
		if tokenKey == "" {
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}

		token, err := getFBToken(ctx, cu.ID())
		if err != nil {
			cl.Log.Warningf(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}
		cl.Log.Debugf("token: %#v", token)
		ctx.JSON(http.StatusOK, gin.H{"CU": cu, tokenKey: token})
	}
}

const FSTokenKey = "FS_TOKEN_KEY"

func getFSTokenKey() string {
	return os.Getenv(FSTokenKey)
}

// func (cl Client) fbTokenHandler() gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		cl.Log.Debugf(msgEnter)
// 		defer cl.Log.Debugf(msgExit)
//
// 		cu, err := cl.currentUser(ctx)
// 		if err != nil {
// 			cl.Log.Warningf(err.Error())
// 		}
//
// 		if cu.IsZero() {
// 			ctx.JSON(http.StatusOK, gin.H{"fbToken": nil})
// 			return
// 		}
//
// 		fbToken, err := getFBToken(ctx, cu.ID())
// 		if err != nil {
// 			cl.Log.Warningf(err.Error())
// 		}
// 		cl.Log.Debugf("fbToken: %#v", fbToken)
// 		ctx.JSON(http.StatusOK, gin.H{"fbToken": fbToken})
// 	}
// }

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

func AllUserQuery(ctx *gin.Context) *datastore.Query {
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

func (cl Client) updateUser(ctx *gin.Context, cu, u1, u2 User) (User, bool, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	changed := false
	if isAdmin(cu) {
		cl.Log.Debugf("is admin")
		if u2.Email != "" && u2.Email != u1.Email {
			cl.Log.Debugf("updating email")
			hash, err := EmailHash(u1.Email)
			if err != nil {
				return u1, changed, err
			}

			u1.Email = u2.Email
			u1.EmailHash = hash
			changed = true
		}

		u1, nameChanged, err := cl.updateUserName(ctx, u1, u2.Name)
		changed = changed || nameChanged
		if err != nil {
			return u1, changed, err
		}
	}

	if !isAdmin(cu) && (cu.ID() != u1.ID()) {
		return u1, changed, nil
	}
	cl.Log.Debugf("is admin or current")

	if u1.EmailReminders != u2.EmailReminders {
		cl.Log.Debugf("updating email reminders")
		u1.EmailReminders = u2.EmailReminders
		changed = true
	}
	if u1.EmailNotifications != u2.EmailNotifications {
		cl.Log.Debugf("updating email notifications")
		u1.EmailNotifications = u2.EmailNotifications
		changed = true
	}
	if u1.GravType != u2.GravType {
		cl.Log.Debugf("updating grav type")
		u1.GravType = u2.GravType
		changed = true
	}
	return u1, changed, nil
}

func (cl Client) updateUserName(ctx *gin.Context, u User, n string) (User, bool, error) {
	matcher := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._%+\-]+$`)

	switch {
	case n == u.Name:
		return u, false, nil
	case len(n) > 15:
		return u, false, fmt.Errorf("%q is too long.", n)
	case !matcher.MatchString(n):
		return u, false, fmt.Errorf("%q is not a valid user name.", n)
	default:
		uniq, err := cl.nameIsUnique(ctx, n)
		if err != nil {
			return u, false, err
		}
		if !uniq {
			return u, false, fmt.Errorf("%q is not a unique user name.", n)
		}
		u.Name = n
		u.LCName = strings.ToLower(n)
		return u, true, nil
	}
}

func (cl Client) nameIsUnique(ctx *gin.Context, name string) (bool, error) {
	LCName := strings.ToLower(name)

	q := datastore.NewQuery("User").Filter("LCName=", LCName)

	cnt, err := cl.User.Count(ctx, q)
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

func (cl GameClient[G, P]) getFiltered(ctx *gin.Context, start, length string) ([]User, int64, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	q := AllUserQuery(ctx).KeysOnly()
	icnt, err := cl.User.Count(ctx, q)
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

	ks, err := cl.User.GetAll(ctx, q, nil)
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

	err = cl.User.GetMulti(ctx, ks, us)
	return us, cnt, err
}

func getUID(ctx *gin.Context, param string) (UID, error) {
	id, err := strconv.ParseInt(ctx.Param(param), 10, 64)
	return UID(id), err
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

func (cl Client) currentUser(ctx *gin.Context) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	session := sessions.Default(ctx)
	token, ok := UserTokenFrom(session)
	if !ok {
		return User{}, ErrMissingToken
	}

	return cl.getUser(ctx, token.ID)
}

func (cl Client) userJSONHandler(uidParam string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.requireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		uid, err := getUID(ctx, uidParam)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if cu.ID() == uid {
			ctx.JSON(http.StatusOK, gin.H{"User": cu})
			return
		}

		u, err := cl.getUser(ctx, uid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"User": u})
	}
}

func (cl Client) NewUserHandler(ctx *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.requireLogin(ctx)
	if err != nil {
		JErr(ctx, fmt.Errorf("you already have an account"))
		return
	}

	session := sessions.Default(ctx)
	u, err := NewUserFrom(session)
	if err != nil {
		cl.Log.Errorf(err.Error())
		ctx.Redirect(http.StatusSeeOther, homePath)
	}

	u.EmailReminders = true
	u.EmailNotifications = true
	u.GravType = "monsterid"
	hash, err := EmailHash(u.Email)
	if err != nil {
		cl.Log.Warningf("email hash error: %v", err)
		ctx.Redirect(http.StatusSeeOther, homePath)
	}
	u.EmailHash = hash

	if !cu.Admin {
		cu = u
	}

	ctx.JSON(http.StatusOK, gin.H{
		"cu":   cu,
		"user": u,
	})
}

func (cl Client) CreateUserHandler(ctx *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	u, err := cl.createUser(ctx)
	if err != nil {
		cl.Log.Warningf("cannot create user: %w", err)
		ctx.Redirect(http.StatusSeeOther, homePath)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user":    u,
		"message": "account created for " + u.Name,
	})
}

func (cl Client) createUser(ctx *gin.Context) (User, error) {

	cu, err := cl.requireLogin(ctx)
	if err == nil && cu.ID() != 0 {
		cl.Log.Warningf("%s(%d) already has an account", cu.Name, cu.ID())
		return cu, err
	}

	session := sessions.Default(ctx)
	token, ok := UserTokenFrom(session)
	if !ok {
		return User{}, errors.New("missing token")
	}

	if token.ID != 0 {
		// ctx.Redirect(http.StatusSeeOther, homePath)
		return User{}, errors.New("user present, no need for new one.")
	}

	u := NewUser(0)
	err = ctx.ShouldBind(u)
	if err != nil {
		return User{}, err
	}

	u, _, err = cl.updateUser(ctx, u, u, u)
	if err != nil {
		return User{}, err
	}

	ks, err := cl.User.AllocateIDs(ctx, []*datastore.Key{u.Key})
	if err != nil {
		return User{}, err
	}

	u.Key = ks[0]
	u.LCName = strings.ToLower(u.Name)

	oaid := genOAuthID(token.Sub)
	oa := newOAuth(oaid)
	oa.ID = u.ID()
	oa.UpdatedAt = time.Now()
	_, err = cl.User.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		ks := []*datastore.Key{oa.Key, u.Key}
		es := []interface{}{&oa, u}
		_, err := tx.PutMulti(ks, es)
		return err

	})

	if err != nil {
		return User{}, err
	}

	if !isAdmin(cu) {
		cu = u
		token.ID = u.ID()

		err = token.SaveTo(session)
		if err != nil {
			return User{}, err
		}

	}

	return u, nil
}

func (cl Client) UpdateUser(uidParam string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.requireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		uid, err := getUID(ctx, uidParam)
		if err != nil {
			JErr(ctx, err)
			return
		}

		u := cu
		if cu.ID() != uid {
			u, err = cl.getUser(ctx, uid)
			if err != nil {
				JErr(ctx, err)
				return
			}
		}

		obj := NewUser(0)
		err = ctx.ShouldBind(&obj)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cl.Log.Debugf("before updateUser\nuser: %#v\nobj: %#v", u, obj)
		u, changed, err := cl.updateUser(ctx, cu, u, obj)
		if err != nil {
			JErr(ctx, err)
			return
		}
		cl.Log.Debugf("after updateUser\nuser: %#v\nobj: %#v", u, obj)
		cl.Log.Debugf("changed: %#v", changed)

		if !changed {
			ctx.JSON(http.StatusOK, gin.H{"Message": "no change to user"})
			return
		}

		_, err = cl.User.Put(ctx, u.Key, &u)
		if err != nil {
			JErr(ctx, err)
			return
		}

		session := sessions.Default(ctx)
		token, _ := UserTokenFrom(session)
		token.ID = u.ID()

		err = token.SaveTo(session)
		if err != nil {
			JErr(ctx, err)
			return
		}
		cl.Cache.SetDefault(u.Key.Encode(), u)

		if cu.ID() == u.ID() {
			ctx.JSON(http.StatusOK, gin.H{"CU": u, "Message": "user updated"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Message": "user updated"})
	}
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

func (cl Client) getUser(ctx *gin.Context, uid UID) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.get(ctx, newUserKey(uid))
}

func (cl Client) get(ctx *gin.Context, k *datastore.Key) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	u, err := cl.mcGet(k)
	if err == nil {
		return u, nil
	}

	return cl.dsGet(ctx, k)
}

func (cl Client) mcGet(k *datastore.Key) (User, error) {
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

func (cl Client) mcGetMulti(ks []*datastore.Key) ([]User, error) {
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

func (cl Client) dsGet(ctx *gin.Context, k *datastore.Key) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if k == nil {
		return User{}, ErrMissingKey
	}

	var u User
	err := cl.User.Get(ctx, k, &u)
	if err != nil {
		return User{}, err
	}
	cl.cacheUser(u)
	return u, nil
}

func (cl Client) dsGetMulti(ctx *gin.Context, ks []*datastore.Key) ([]User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	l := len(ks)
	if l == 0 {
		return nil, ErrMissingKey
	}

	us := make([]User, l)
	err := cl.User.GetMulti(ctx, ks, us)
	if err != nil {
		return us, err
	}
	for _, u := range us {
		cl.cacheUser(u)
	}
	return us, nil
}

func (cl Client) cacheUser(u User) {
	if u.Key == nil {
		return
	}
	cl.Cache.SetDefault(u.Key.Encode(), u)
}

func (cl GameClient[G, P]) GetMulti(ctx *gin.Context, uids []UID) ([]User, error) {
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

	return cl.dsGetMulti(ctx, ks)
}

// func (cl Client) AllocateIDs(c *gin.Context, ks []*datastore.Key) ([]*datastore.Key, error) {
// 	return cl.DS.AllocateIDs(c, ks)
// }

func (cl GameClient[G, P]) Put(ctx *gin.Context, k *datastore.Key, u User) (*datastore.Key, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.putUserByKey(ctx, k, u)
}

func (cl GameClient[G, P]) putUserByKey(ctx *gin.Context, k *datastore.Key, u User) (*datastore.Key, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	k, err := cl.User.Put(ctx, k, u)
	if err != nil {
		return nil, err
	}
	cl.cacheUser(u)
	return k, nil
}

// func (cl Client) RunInTransaction(c *gin.Context, f func(*datastore.Transaction) error, opts ...datastore.TransactionOption) (*datastore.Commit, error) {
// 	return cl.DS.RunInTransaction(c, f, opts...)
// }
