package sn

import (
	"errors"
	"fmt"
)

var (
	ErrValidation         = errors.New("validation error")
	ErrUnexpected         = errors.New("unexpected error")
	ErrUserNotFound       = fmt.Errorf("current user not found: %w", ErrValidation)
	ErrPlayerNotFound     = fmt.Errorf("player not found: %w", ErrValidation)
	ErrActionNotPerformed = fmt.Errorf("player has yet to perform an action: %w", ErrValidation)
	ErrNotAdmin           = fmt.Errorf("current user is not admin: %w", ErrValidation)
	ErrNotCurrentPlayer   = fmt.Errorf("current user is not current player: %w", ErrValidation)
)
