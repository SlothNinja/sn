package sn

import (
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gofrs/uuid"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	gob.RegisterName("*user.sessionToken", new(sessionToken))
}

const (
	HOST           = "HOST"
	UserHostURLEnv = "USER_HOST_URL"
	authPath       = "/auth"
	sessionKey     = "session"
	userNewPath    = "#/new"
	tokenLength    = 32
	uKind          = "User"
	oauthsKind     = "OAuths"
	oauthKind      = "OAuth"
	root           = "root"
	stateKey       = "state"
	emailKey       = "email"
	nameKey        = "name"
	redirectKey    = "redirect"
)

func getRedirectionPath(c *gin.Context) (string, bool) {
	return c.GetQuery("redirect")
}

func Login(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		state := randToken(tokenLength)
		session.Set(stateKey, state)

		redirect, found := getRedirectionPath(c)
		if !found {
			redirect = base64.StdEncoding.EncodeToString([]byte(c.Request.Header.Get("Referer")))
		}
		session.Set(redirectKey, redirect)
		session.Save()

		c.Redirect(http.StatusSeeOther, getLoginURL(c, path, state))
	}
}

func Logout(c *gin.Context) {
	s := sessions.Default(c)
	s.Delete(sessionKey)
	err := s.Save()
	if err != nil {
		Warningf("unable to save session: %v", err)
	}

	path, found := getRedirectionPath(c)
	if found {
		bs, err := base64.StdEncoding.DecodeString(path)
		if err == nil {
			c.Redirect(http.StatusSeeOther, string(bs))
			return
		}
		Warningf("unable to decode path: %v", err)
	}
	c.Redirect(http.StatusSeeOther, homePath)
}

func randToken(length int) string {
	key := securecookie.GenerateRandomKey(length)
	return base64.StdEncoding.EncodeToString(key)
}

func getLoginURL(c *gin.Context, path, state string) string {
	// State can be some kind of random generated hash string.
	// See relevant RFC: http://tools.ietf.org/html/rfc6749#section-10.12
	return oauth2Config(c, path, scopes()...).AuthCodeURL(state)
}

func oauth2Config(c *gin.Context, path string, scopes ...string) *oauth2.Config {
	redirectURL := fmt.Sprintf("%s/%s", getHost(), strings.TrimPrefix(path, "/"))
	return &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  redirectURL,
	}
}

func scopes() []string {
	return []string{"email", "profile", "openid"}
}

func getHost() string {
	return os.Getenv(HOST)
}

func getUserHostURL() string {
	s := os.Getenv(UserHostURLEnv)
	if s != "" {
		return s
	}
	return getHost()
}

type OAInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	LoggedIn      bool
	Admin         bool
}

const fqdn = "www.slothninja.com"

var namespaceUUID = uuid.NewV5(uuid.NamespaceDNS, fqdn)

// Generates ID for User from ID obtained from OAuth OpenID Connect
func GenOAuthID(s string) string {
	return uuid.NewV5(namespaceUUID, s).String()
}

type OAuth struct {
	Key       *datastore.Key `datastore:"__key__"`
	ID        UID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (o *OAuth) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(o, ps)
}

func (o *OAuth) Save() ([]datastore.Property, error) {
	t := time.Now()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = t
	}
	o.UpdatedAt = t
	return datastore.SaveStruct(o)
}

func (o *OAuth) LoadKey(k *datastore.Key) error {
	o.Key = k
	return nil
}

func pk() *datastore.Key {
	return datastore.NameKey(oauthsKind, root, nil)
}

func NewOAuthKey(id string) *datastore.Key {
	return datastore.NameKey(oauthKind, id, pk())
}

func NewOAuth(id string) OAuth {
	return OAuth{Key: NewOAuthKey(id)}
}

func (cl Client[G, P]) Auth(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		uInfo, err := getUInfo(c, path)
		if err != nil {
			cl.Log.Errorf(err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		session := sessions.Default(c)
		retrievedPath, ok := session.Get(redirectKey).(string)
		var redirectPath string
		if ok {
			bs, err := base64.StdEncoding.DecodeString(retrievedPath)
			if err == nil {
				redirectPath = string(bs)
			}
		}

		oaid := GenOAuthID(uInfo.Sub)
		oa, err := cl.getOAuth(c, oaid)
		// Succesfully pulled oauth id from datastore
		if err == nil {
			u, err := cl.Get(c, oa.ID)
			if err != nil {
				cl.Log.Errorf(err.Error())
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			st := NewSessionToken(u.ID(), uInfo.Sub)
			saveToSessionAndReturnTo(c, st, redirectPath)
			return
		}

		// Datastore error other than missing entity.
		if err != datastore.ErrNoSuchEntity {
			cl.Log.Errorf("unable to get user for %#v", uInfo)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// oauth id not present in datastore
		// Check to see if other entities exist for same email address.
		// If so, use old entities for user
		u, err := cl.getByEmail(c, uInfo.Email)
		if err == nil {
			oa := NewOAuth(oaid)
			oa.ID = u.ID()
			_, err = cl.User.Put(c, oa.Key, &oa)
			if err != nil {
				cl.Log.Errorf(err.Error())
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			st := NewSessionToken(u.ID(), uInfo.Sub)
			saveToSessionAndReturnTo(c, st, redirectPath)
			return
		}

		u = NewUser(0)
		u.Name = strings.Split(uInfo.Email, "@")[0]
		u.Email = uInfo.Email
		session.Set(nameKey, u.Name)
		session.Set(emailKey, u.Email)
		saveToSessionAndReturnTo(c, NewSessionToken(u.ID(), uInfo.Sub), userNewPath)
	}
}

func isAdmin(u User) bool {
	return u.Admin
}

func (cl Client[G, P]) As(c *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.Current(c)
	if err != nil {
		cl.Log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !isAdmin(cu) {
		cl.Log.Errorf("must be admin")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	id, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		cl.Log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	u, err := cl.Get(c, UID(id))
	if err != nil {
		cl.Log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	st := NewSessionToken(u.ID(), "")
	saveToSessionAndReturnTo(c, st, homePath)
	return
}

func getUInfo(c *gin.Context, path string) (OAInfo, error) {
	// Handle the exchange code to initiate a transport.
	session := sessions.Default(c)
	retrievedState := session.Get("state")
	if retrievedState != c.Query("state") {
		return OAInfo{}, fmt.Errorf("Invalid session state: %s", retrievedState)
	}

	conf := oauth2Config(c, path, scopes()...)
	tok, err := conf.Exchange(c, c.Query("code"))
	if err != nil {
		return OAInfo{}, fmt.Errorf("tok error: %s", err.Error())
	}

	client := conf.Client(c, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return OAInfo{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return OAInfo{}, err
	}

	uInfo := OAInfo{}
	var b binding.BindingBody = binding.JSON
	err = b.BindBody(body, &uInfo)
	if err != nil {
		return OAInfo{}, err
	}
	return uInfo, nil
}

func (cl Client[G, P]) getOAuth(c *gin.Context, id string) (OAuth, error) {
	return cl.getOAuthByKey(c, NewOAuthKey(id))
}

func (cl Client[G, P]) getOAuthByKey(c *gin.Context, k *datastore.Key) (OAuth, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	oauth, found := cl.getCachedOAuth(k)
	if found {
		return oauth, nil
	}

	oauth = NewOAuth(k.Name)
	err := cl.User.Get(c, k, &oauth)
	if err != nil {
		return oauth, err
	}
	cl.cacheOAuth(oauth)
	return oauth, nil
}

func (cl Client[G, P]) getCachedOAuth(k *datastore.Key) (OAuth, bool) {
	oauth := NewOAuth(k.Name)
	if k == nil {
		return oauth, false
	}

	data, found := cl.Cache.Get(k.Encode())
	if !found {
		return oauth, false
	}

	oauth, ok := data.(OAuth)
	if !ok {
		return oauth, false
	}
	return oauth, true
}

func (cl Client[G, P]) cacheOAuth(oauth OAuth) {
	if oauth.Key == nil {
		return
	}
	cl.Cache.SetDefault(oauth.Key.Encode(), oauth)
}

func saveToSessionAndReturnTo(c *gin.Context, st *sessionToken, path string) {
	session := sessions.Default(c)
	err := st.SaveTo(session)
	if err != nil {
		Errorf(err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Redirect(http.StatusSeeOther, path)
	return
}

func (cl Client[G, P]) getByEmail(c *gin.Context, email string) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	email = strings.ToLower(strings.TrimSpace(email))
	q := datastore.NewQuery(uKind).
		Ancestor(userRootKey()).
		Filter("Email=", email).
		KeysOnly()

	ks, err := cl.User.GetAll(c, q, nil)
	if err != nil {
		return User{}, err
	}

	for i := range ks {
		if ks[i].ID != 0 {
			return cl.Get(c, UID(ks[i].ID))
		}
	}
	return User{}, errors.New("unable to find user")
}

type sessionToken struct {
	ID  UID
	Sub string
}

func NewSessionToken(uid UID, sub string) *sessionToken {
	return &sessionToken{
		ID:  uid,
		Sub: sub,
	}
}

func (st *sessionToken) SaveTo(s sessions.Session) error {
	s.Set(sessionKey, st)
	return s.Save()
}

func SessionTokenFrom(s sessions.Session) (*sessionToken, bool) {
	token, ok := s.Get(sessionKey).(*sessionToken)
	return token, ok
}

func NewFrom(s sessions.Session) (User, error) {
	token, ok := SessionTokenFrom(s)
	if !ok {
		return User{}, errors.New("token not found")
	}

	if token.ID != 0 {
		return User{}, errors.New("user present, no need for new one.")
	}

	var err error
	u := NewUser(token.ID)
	u.Name, err = nameFrom(s)
	if err != nil {
		return User{}, err
	}
	u.Email, err = emailFrom(s)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func emailFrom(s sessions.Session) (string, error) {
	email, ok := s.Get(emailKey).(string)
	if !ok {
		return "", errors.New("email not found")
	}
	return email, nil
}

func nameFrom(s sessions.Session) (string, error) {
	name, ok := s.Get(nameKey).(string)
	if !ok {
		return "", errors.New("name not found")
	}
	return name, nil
}

func (cl Client[G, P]) loginHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		referer := ctx.Request.Referer()
		encodedReferer := base64.StdEncoding.EncodeToString([]byte(referer))

		path := getUserHostURL() + "/login?redirect=" + encodedReferer
		cl.Log.Debugf("path: %q", path)
		ctx.Redirect(http.StatusSeeOther, path)
	}
}

func (cl Client[G, P]) logoutHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		referer := ctx.Request.Referer()
		Logout(ctx)
		ctx.Redirect(http.StatusSeeOther, referer)
	}
}
