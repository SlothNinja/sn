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
	firebase "firebase.google.com/go"
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
	fbTokenKey     = "fbToken"
	redirectKey    = "redirect"
)

func getRedirectionPath(ctx *gin.Context) (string, bool) {
	return ctx.GetQuery("redirect")
}

func Login(path string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		state := randToken(tokenLength)
		session.Set(stateKey, state)

		redirect, found := getRedirectionPath(ctx)
		if !found {
			redirect = base64.StdEncoding.EncodeToString([]byte(ctx.Request.Header.Get("Referer")))
		}
		session.Set(redirectKey, redirect)
		session.Save()

		Debugf("path: %v\nstate: %v\nredirect: %v\n", path, state, getLoginURL(ctx, path, state))
		ctx.Redirect(http.StatusSeeOther, getLoginURL(ctx, path, state))
	}
}

func Logout(ctx *gin.Context) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	s := sessions.Default(ctx)
	s.Delete(sessionKey)
	err := s.Save()
	if err != nil {
		Warningf("unable to save session: %v", err)
	}

	path, found := getRedirectionPath(ctx)
	Debugf("path: %v\nfound: %v", path, found)
	if found {
		// bs, err := base64.StdEncoding.DecodeString(path)
		// if err == nil {
		// ctx.Redirect(http.StatusSeeOther, string(bs))
		ctx.Redirect(http.StatusSeeOther, path)
		return
		// }
		// Warningf("unable to decode path: %v", err)
	}
	ctx.Redirect(http.StatusSeeOther, homePath)
}

func randToken(length int) string {
	key := securecookie.GenerateRandomKey(length)
	return base64.StdEncoding.EncodeToString(key)
}

func getLoginURL(ctx *gin.Context, path, state string) string {
	// State can be some kind of random generated hash string.
	// See relevant RFC: http://tools.ietf.org/html/rfc6749#section-10.12
	return oauth2Config(ctx, path, scopes()...).AuthCodeURL(state)
}

func oauth2Config(ctx *gin.Context, path string, scopes ...string) *oauth2.Config {
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
func genOAuthID(s string) string {
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

func newOAuth(id string) OAuth {
	return OAuth{Key: NewOAuthKey(id)}
}

func redirectPathFrom(ctx *gin.Context) string {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	session := sessions.Default(ctx)
	retrievedPath, ok := session.Get(redirectKey).(string)
	if !ok {
		return ""
	}

	bs, err := base64.StdEncoding.DecodeString(retrievedPath)
	if err != nil {
		return ""
	}
	return string(bs)
}

// returns whether user present in database and any error resulting from trying to create session
func (cl UserServiceClient) loginSessionByOAuthSub(ctx *gin.Context, sub string) (bool, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	oaid := genOAuthID(sub)
	oa, err := cl.getOAuth(ctx, oaid)
	if err != nil {
		return false, err
	}

	// Succesfully pulled uid from datastore using OAuth Sub
	u, err := cl.getUser(ctx, oa.ID)
	if err != nil {
		return false, err
	}

	// created new token and save to session
	st := newSessionToken(u.ID(), sub)
	// st, err := newSessionToken(ctx, u.ID(), sub)
	// if err != nil {
	// 	return true, err
	// }

	return true, st.SaveTo(sessions.Default(ctx))
}

// returns whether user present in datastore and any error resulting for trying to create session
func (cl UserServiceClient) loginSessionByEmailAndSub(ctx *gin.Context, email, sub string) (bool, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	u, err := cl.getByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	oa := newOAuth(genOAuthID(sub))
	oa.ID = u.ID()

	_, err = cl.User.Put(ctx, oa.Key, &oa)
	if err != nil {
		return true, err
	}

	st := newSessionToken(u.ID(), sub)
	// st, err := newSessionToken(ctx, u.ID(), sub)
	// if err != nil {
	// 	return true, err
	// }

	return true, st.SaveTo(sessions.Default(ctx))
}

// returns error resulting for trying to create session
func (cl UserServiceClient) loginSessionNewUser(ctx *gin.Context, email, sub string) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	u := NewUser(0)
	u.Name = strings.Split(email, "@")[0]
	u.Email = email
	session := sessions.Default(ctx)
	session.Set(nameKey, u.Name)
	session.Set(emailKey, u.Email)
	st := newSessionToken(u.ID(), sub)
	// st, err := newSessionToken(ctx, u.ID(), sub)
	// if err != nil {
	// 	return err
	// }

	return st.SaveTo(sessions.Default(ctx))
}

func (cl UserServiceClient) Auth(path string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		uInfo, err := getUInfo(ctx, path)
		if err != nil {
			cl.Log.Errorf(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if userExists, err := cl.loginSessionByOAuthSub(ctx, uInfo.Sub); userExists && err != nil {
			cl.Log.Errorf(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else if err == nil {
			ctx.Redirect(http.StatusSeeOther, redirectPathFrom(ctx))
			return
		} else {
			cl.Log.Debugf(err.Error())
		}

		// OAuth sub not associated with UID in datastore
		// Check to see if other entities exist for same email address.
		// If so, use old entities for user
		if userExists, err := cl.loginSessionByEmailAndSub(ctx, uInfo.Email, uInfo.Sub); userExists && err != nil {
			cl.Log.Errorf(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else if err == nil {
			ctx.Redirect(http.StatusSeeOther, redirectPathFrom(ctx))
			return
		} else {
			cl.Log.Debugf(err.Error())
		}

		// Create New User
		if err := cl.loginSessionNewUser(ctx, uInfo.Email, uInfo.Sub); err != nil {
			cl.Log.Errorf(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.Redirect(http.StatusSeeOther, userNewPath)

	}
}

func getFBToken(ctx *gin.Context, uid UID) (string, error) {
	Debugf(msgEnter)
	defer Debugf(msgEnter)

	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error initializing app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting Auth client: %w", err)
	}

	token, err := client.CustomToken(ctx, fmt.Sprintf("%d", uid))
	if err != nil {
		return "", fmt.Errorf("error minting custom token: %w", err)
	}

	return token, err
}

func isAdmin(u User) bool {
	return u.Admin
}

func (cl UserServiceClient) As(ctx *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.currentUser(ctx)
	if err != nil {
		cl.Log.Errorf(err.Error())
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !isAdmin(cu) {
		cl.Log.Errorf("must be admin")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	id, err := strconv.ParseInt(ctx.Param("uid"), 10, 64)
	if err != nil {
		cl.Log.Errorf(err.Error())
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	u, err := cl.getUser(ctx, UID(id))
	if err != nil {
		cl.Log.Errorf(err.Error())
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	st := newSessionToken(u.ID(), "")
	// st, err := newSessionToken(ctx, u.ID(), "")
	// if err != nil {
	// 	cl.Log.Errorf(err.Error())
	// 	ctx.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }

	err = st.SaveTo(sessions.Default(ctx))
	if err != nil {
		cl.Log.Errorf(err.Error())
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	return
}

func getUInfo(ctx *gin.Context, path string) (OAInfo, error) {
	// Handle the exchange code to initiate a transport.
	session := sessions.Default(ctx)
	retrievedState := session.Get("state")
	if retrievedState != ctx.Query("state") {
		return OAInfo{}, fmt.Errorf("Invalid session state: %s", retrievedState)
	}

	conf := oauth2Config(ctx, path, scopes()...)
	tok, err := conf.Exchange(ctx, ctx.Query("code"))
	if err != nil {
		return OAInfo{}, fmt.Errorf("tok error: %s", err.Error())
	}

	client := conf.Client(ctx, tok)
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

func (cl UserServiceClient) getOAuth(ctx *gin.Context, id string) (OAuth, error) {
	return cl.getOAuthByKey(ctx, NewOAuthKey(id))
}

func (cl UserServiceClient) getOAuthByKey(ctx *gin.Context, k *datastore.Key) (OAuth, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	oauth, found := cl.getCachedOAuth(k)
	if found {
		return oauth, nil
	}

	oauth = newOAuth(k.Name)
	err := cl.User.Get(ctx, k, &oauth)
	if err != nil {
		return oauth, err
	}
	cl.cacheOAuth(oauth)
	return oauth, nil
}

func (cl UserServiceClient) getCachedOAuth(k *datastore.Key) (OAuth, bool) {
	oauth := newOAuth(k.Name)
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

func (cl UserServiceClient) cacheOAuth(oauth OAuth) {
	if oauth.Key == nil {
		return
	}
	cl.Cache.SetDefault(oauth.Key.Encode(), oauth)
}

// func saveToSessionAndReturnTo(c *gin.Context, st *sessionToken, path string) error {
// 	session := sessions.Default(c)
// 	err := st.SaveTo(session)
// 	if err != nil {
// 		Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusBadRequest)
// 		return
// 	}
// 	c.Redirect(http.StatusSeeOther, path)
// 	return
// }

func (cl UserServiceClient) getByEmail(ctx *gin.Context, email string) (User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	email = strings.ToLower(strings.TrimSpace(email))
	q := datastore.NewQuery(uKind).
		Ancestor(userRootKey()).
		Filter("Email=", email).
		KeysOnly()

	ks, err := cl.User.GetAll(ctx, q, nil)
	if err != nil {
		return User{}, err
	}

	for i := range ks {
		if ks[i].ID != 0 {
			return cl.getUser(ctx, UID(ks[i].ID))
		}
	}
	return User{}, errors.New("unable to find user")
}

type sessionToken struct {
	ID  UID
	Sub string
	// FireApp string
}

func newSessionToken(uid UID, sub string) sessionToken {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	// fbToken, err := getFBToken(ctx, uid)
	// if err != nil {
	// 	return sessionToken{}, err
	// }
	// return sessionToken{ID: uid, Sub: sub, FireApp: fbToken}, nil
	return sessionToken{ID: uid, Sub: sub}
}

func (st sessionToken) SaveTo(s sessions.Session) error {
	s.Set(sessionKey, st)
	return s.Save()
}

func UserTokenFrom(s sessions.Session) (*sessionToken, bool) {
	token, ok := s.Get(sessionKey).(*sessionToken)
	return token, ok
}

func NewUserFrom(s sessions.Session) (User, error) {
	token, ok := UserTokenFrom(s)
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

func (cl Client) loginHandler() gin.HandlerFunc {
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

func (cl Client) logoutHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		referer := ctx.Request.Referer()
		Logout(ctx)
		ctx.Redirect(http.StatusSeeOther, referer)
	}
}
