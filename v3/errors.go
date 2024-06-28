package sn

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrValidation         = errors.New("validation error")
	ErrUnexpected         = errors.New("unexpected error")
	ErrPlayerNotFound     = fmt.Errorf("player not found: %w", ErrValidation)
	ErrActionNotPerformed = fmt.Errorf("player has yet to perform an action: %w", ErrValidation)
	ErrNotAdmin           = fmt.Errorf("current user is not admin: %w", ErrValidation)
	ErrNotCurrentPlayer   = fmt.Errorf("current user is not current player: %w", ErrValidation)
	ErrNotLoggedIn        = fmt.Errorf("must login to access resource: %w", ErrValidation)
)

func JErr(ctx *gin.Context, err error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	slog.Debug(err.Error())
	if errors.Is(err, ErrValidation) {
		ctx.JSON(http.StatusOK, gin.H{"Message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"Error": err.Error()})
}
