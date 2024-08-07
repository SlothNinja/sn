package sn

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

// IsProduction returns true if NODE_ENV environment variable is equal to "production".
// GAE sets NODE_ENV environement to "production" on deployment.
// NODE_ENV can be overridden in app.yaml configuration.
func IsProduction() bool {
	return os.Getenv("NODE_ENV") == "production"
}

// RequireLogin returns the logged in User
// Otherwise, returns error
func (cl *Client) RequireLogin(ctx *gin.Context) (*User, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	cu, err := cl.getCU(ctx)
	if err != nil {
		return nil, err
	}
	return cu, nil
}

// RequireAdmin returns the logged in user if user is admin
// Otherwise, returns an error
func (cl *Client) RequireAdmin(ctx *gin.Context) (*User, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	token := cl.GetSessionToken(ctx)
	if token == nil {
		return nil, ErrNotLoggedIn
	}

	if !token.Data.Admin {
		return nil, ErrNotAdmin
	}
	return token.ToUser(), nil
}
