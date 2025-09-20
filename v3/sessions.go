package sn

import (
	"context"
	"encoding/gob"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("SessionToken", new(SessionToken))
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

	token, err := client.CustomToken(ctx, uid.toString())
	if err != nil {
		return "", fmt.Errorf("error minting custom token: %w", err)
	}

	return token, err
}

func (cl *Client) getCUID(ctx *gin.Context) (UID, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	token := cl.GetSessionToken(ctx)
	if token == nil {
		return 0, ErrNotLoggedIn
	}

	return token.ID, nil
}

// ToUser returns a user from the session token
func (s *SessionToken) ToUser() *User {
	return &User{ID: s.ID, userData: s.Data}
}

func tokenFrom(u *User, sub string) *SessionToken {
	return &SessionToken{ID: u.ID, Sub: sub, Data: u.userData}
}

// SessionToken represents a session for a user
type SessionToken struct {
	ID   UID
	Sub  string
	Data userData
}

// SetSessionToken creates and stores a new session token for user and its associated subscription
func (cl *Client) setSessionToken(ctx *gin.Context, t *SessionToken) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if t == nil {
		return
	}

	cl.Session(ctx).Set("session", t)
}

// SetSessionToken creates and stores a new session token for user and its associated subscription
func (cl *Client) SetSessionToken(ctx *gin.Context, u *User, sub string) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if u == nil {
		return
	}

	cl.setSessionToken(ctx, tokenFrom(u, sub))
}

// GetSessionToken returns session token for user and its associated subscription
func (cl *Client) GetSessionToken(ctx *gin.Context) *SessionToken {
	token, ok := cl.Session(ctx).Get("session").(*SessionToken)
	if !ok {
		return nil
	}
	return token
}

// SaveSession saves the current session
func (cl *Client) SaveSession(ctx *gin.Context) error {
	return cl.Session(ctx).Save()
}

// ClearSession clears the current session
func (cl *Client) ClearSession(ctx *gin.Context) {
	cl.Session(ctx).Clear()
}

// Session returns the session tied to the client
func (cl *Client) Session(ctx *gin.Context) sessions.Session {
	return sessions.Default(ctx)
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
	cl.Router.Use(sessions.Sessions("sng-oauth", store))
	return cl
}
