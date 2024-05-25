package sn

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("SessionToken", SessionToken{})
}

// ErrMissingToken signals that expected session token is missing
var ErrMissingToken = fmt.Errorf("missing token")

const (
	emailKey    = "email"
	nameKey     = "name"
	sessionName = "sng-oauth"
	sessionKey  = "session"
)

type session struct {
	sessions.Session
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

func (cl *Client) getCUID(ctx *gin.Context) (UID, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	token, ok := cl.session(ctx).GetUserToken()
	if !ok {
		return 0, ErrMissingToken
	}

	return token.ID, nil
}

func (cl *Client) getCU(ctx *gin.Context) (*User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	token, ok := cl.session(ctx).GetUserToken()
	cl.Log.Debugf("token: %#v", token)
	if !ok {
		return nil, ErrMissingToken
	}

	return &User{ID: token.ID, userData: token.Data}, nil
}

func (cl *Client) getAdmin(ctx *gin.Context) (bool, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	token, ok := cl.session(ctx).GetUserToken()
	if !ok {
		return false, ErrMissingToken
	}

	return token.Data.Admin, nil
}

type SessionToken struct {
	ID   UID
	Sub  string
	Data userData
}

func (s session) NewToken(uid UID, sub string, data userData) SessionToken {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return SessionToken{ID: uid, Sub: sub, Data: data}
}

func (s session) SaveToken(st SessionToken) error {
	s.Set(sessionKey, st)
	return s.Save()
}

func (s session) GetUserToken() (SessionToken, bool) {
	token, ok := s.Get(sessionKey).(SessionToken)
	return token, ok
}

func (s session) GetNewUser() (*User, error) {
	token, ok := s.GetUserToken()
	if !ok {
		return &User{}, errors.New("token not found")
	}

	if token.ID != 0 {
		return &User{}, errors.New("user present, no need for new one.")
	}

	var err error
	u := &User{ID: token.ID}
	u.Name, err = s.getNewUserName()
	if err != nil {
		return nil, err
	}
	u.Email, err = s.getNewUserEmail()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s session) SetUserEmail(email string) {
	s.Set(emailKey, email)
}

func (s session) getNewUserEmail() (string, error) {
	email, ok := s.Get(emailKey).(string)
	if !ok {
		return "", errors.New("email not found")
	}
	return email, nil
}

func (s session) SetUserName(name string) {
	s.Set(nameKey, name)
}

func (s session) getNewUserName() (string, error) {
	name, ok := s.Get(nameKey).(string)
	if !ok {
		return "", errors.New("name not found")
	}
	return name, nil
}

func (cl *Client) session(ctx *gin.Context) session {
	return session{sessions.Default(ctx)}
}

// NewStore generates a new secure cookie store
func (cl *Client) initSession(ctx context.Context) *Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	s, err := cl.getSessionSecrets(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to create cookie store: %v", err))
	}

	store := cookie.NewStore(s.HashKey, s.BlockKey)
	opts := sessions.Options{
		Domain: "fake-slothninja.com",
		Path:   "/",
	}
	if IsProduction() {
		opts = sessions.Options{
			Domain: "slothninja.com",
			Path:   "/",
			MaxAge: 60 * 60 * 24 * 30, // 1 Month in seconds
			Secure: true,
		}
	}
	store.Options(opts)
	cl.Router.Use(sessions.Sessions(sessionName, store))
	return cl
}
