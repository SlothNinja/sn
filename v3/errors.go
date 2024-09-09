package sn

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	// ErrValidation represents a validation error
	ErrValidation = errors.New("validation error")

	// ErrPlayerNotFound represents a player not found validation error
	ErrPlayerNotFound = fmt.Errorf("player not found: %w", ErrValidation)

	// ErrNotAdmin represents a user not admin validation error
	ErrNotAdmin = fmt.Errorf("current user is not admin: %w", ErrValidation)

	// ErrNotLoggedIn represents user not logged in validation error
	ErrNotLoggedIn = fmt.Errorf("must login to access resource: %w", ErrValidation)
)

// JErr returns an error message via JSON
// Error message returned with a 'Message' key if a validation error
// Error message returned with a 'Error' key if another type of error
func JErr(ctx *gin.Context, err error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	slog.Debug(err.Error())
	if errors.Is(err, ErrValidation) {
		ctx.JSON(http.StatusOK, gin.H{"Message": strings.TrimSuffix(err.Error(), ": "+ErrValidation.Error())})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"Error": err.Error()})
}
