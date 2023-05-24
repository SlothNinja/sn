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
	ErrInvalidCache       = errors.New("invalid cache value")
)

func JErr(c *gin.Context, err error) {
	Debugf(err.Error())
	if errors.Is(err, ErrValidation) {
		c.JSON(http.StatusOK, gin.H{"Message": err.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"Message": ErrUnexpected.Error()})
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
