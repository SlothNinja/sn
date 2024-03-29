package sn

import (
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
)

var (
	ErrValidation = errors.New("validation error")
	ErrUnexpected = errors.New("unexpected error")
	// ErrUserNotFound       = fmt.Errorf("current user not found: %w", ErrValidation)
	ErrPlayerNotFound     = fmt.Errorf("player not found: %w", ErrValidation)
	ErrActionNotPerformed = fmt.Errorf("player has yet to perform an action: %w", ErrValidation)
	ErrNotAdmin           = fmt.Errorf("current user is not admin: %w", ErrValidation)
	ErrNotCurrentPlayer   = fmt.Errorf("current user is not current player: %w", ErrValidation)
	ErrNotLoggedIn        = fmt.Errorf("must login to access resource: %w", ErrValidation)
)

func JErr(ctx *gin.Context, err error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	Debugf(err.Error())
	if errors.Is(err, ErrValidation) {
		ctx.JSON(http.StatusOK, gin.H{"Message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"Error": err.Error()})
}

func singleError(err error) error {
	if err == nil {
		return err
	}
	if me, ok := err.(datastore.MultiError); ok {
		return me[0]
	}
	return err
}
