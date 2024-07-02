package sn

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("SessionToken", new(SessionToken))
}

// ErrMissingToken signals that expected session token is missing
var ErrMissingToken = fmt.Errorf("missing token")

const (
	emailKey    = "email"
	nameKey     = "name"
	stateKey    = "state"
	redirectKey = "redirect"
	sessionName = "sng-oauth"
	sessionKey  = "session"
)

// type session struct {
// 	sessions.Session
// }

func getFBToken(ctx *gin.Context, uid UID) (string, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgEnter)

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

const notFound = false

func (cl *Client) getCUID(ctx *gin.Context) (UID, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	token := cl.GetSessionToken(ctx)
	if token == nil {
		return 0, ErrNotLoggedIn
	}

	return token.ID, nil
}

func (cl *Client) getCU(ctx *gin.Context) (*User, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	token := cl.GetSessionToken(ctx)
	if token == nil {
		return nil, ErrNotLoggedIn
	}

	return token.toUser(), nil
}

func (s *SessionToken) toUser() *User {
	return &User{ID: s.ID, userData: s.Data}
}

func TokenFrom(u *User, sub string) *SessionToken {
	return &SessionToken{ID: u.ID, Sub: sub, Data: u.userData}
}

// func (cl *Client) isAdminSession(ctx *gin.Context) bool {
// 	slog.Debug(msgEnter)
// 	defer slog.Debug(msgExit)
//
// 	token, found := cl.GetSessionToken(ctx)
// 	if !found {
// 		return false
// 	}
//
// 	return token.Data.Admin
// }

type SessionToken struct {
	ID   UID
	Sub  string
	Data userData
}

func (cl *Client) SetSessionToken(ctx *gin.Context, u *User, sub string) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if u == nil {
		return
	}

	cl.session(ctx).Set(sessionKey, TokenFrom(u, sub))
}

func (cl *Client) GetSessionToken(ctx *gin.Context) *SessionToken {
	token, ok := cl.session(ctx).Get(sessionKey).(*SessionToken)
	if !ok {
		return nil
	}
	return token
}

func (cl *Client) SaveSession(ctx *gin.Context) error {
	return cl.session(ctx).Save()
}

func (cl *Client) ClearSession(ctx *gin.Context) {
	cl.session(ctx).Clear()
}

func (cl *Client) GetNewUser(ctx *gin.Context) (*User, error) {
	token := cl.GetSessionToken(ctx)
	if token == nil {
		return nil, ErrNotLoggedIn
	}

	if token.ID != 0 {
		return nil, errors.New("user present, no need for new one.")
	}

	u := token.toUser()
	u.Name = cl.getSessionUserName(ctx)
	if u.Name == "" {
		return nil, errors.New("session missing user name.")
	}

	u.Email = cl.getSessionUserEmail(ctx)
	if u.Email == "" {
		return nil, errors.New("session missing user email.")
	}

	return u, nil
}

func (cl *Client) SetSessionUserEmail(ctx *gin.Context, email string) {
	cl.session(ctx).Set(emailKey, email)
}

func (cl *Client) getSessionUserEmail(ctx *gin.Context) string {
	email, ok := cl.session(ctx).Get(emailKey).(string)
	if !ok {
		return ""
	}
	return email
}

func (cl *Client) SetSessionUserName(ctx *gin.Context, name string) {
	cl.session(ctx).Set(nameKey, name)
}

func (cl *Client) SetSessionState(ctx *gin.Context, state string) {
	cl.session(ctx).Set(stateKey, state)
}

func (cl *Client) GetSessionState(ctx *gin.Context) string {
	state, ok := cl.session(ctx).Get(stateKey).(string)
	slog.Debug(fmt.Sprintf("state: %v, statekey: %v, ok: %v", state, stateKey, ok))
	if !ok {
		return ""
	}
	return state
}

func (cl *Client) SetSessionRedirect(ctx *gin.Context, redirect string) {
	cl.session(ctx).Set(redirectKey, redirect)
}

func (cl *Client) GetSessionRedirect(ctx *gin.Context) string {
	redirect, ok := cl.session(ctx).Get(redirectKey).(string)
	if !ok {
		return ""
	}
	return redirect
}

func (cl *Client) getSessionUserName(ctx *gin.Context) string {
	name, ok := cl.session(ctx).Get(nameKey).(string)
	if !ok {
		return ""
	}
	return name
}

func (cl *Client) session(ctx *gin.Context) sessions.Session {
	return sessions.Default(ctx)
}

// NewStore generates a new secure cookie store
func (cl *Client) initSession(ctx context.Context) *Client {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	s, err := cl.getSessionSecrets(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to create cookie store: %v", err))
	}

	store := cookie.NewStore(s.HashKey, s.BlockKey)
	// opts := sessions.Options{
	// 	Domain: "fake-slothninja.com",
	// 	Path:   "/",
	// }
	if IsProduction() {
		opts := sessions.Options{
			Domain: "slothninja.com",
			Path:   "/",
			MaxAge: 60 * 60 * 24 * 30, // 1 Month in seconds
			Secure: true,
		}
		store.Options(opts)
	}
	cl.Router.Use(sessions.Sessions(sessionName, store))
	return cl
}
