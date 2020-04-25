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
	ErrNotCPorAdmin       = fmt.Errorf("not current player or admin: %w", ErrValidation)
	ErrNotAdmin           = fmt.Errorf("not admin: %w", ErrValidation)
)
