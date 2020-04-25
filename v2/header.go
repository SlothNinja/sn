package sn

import (
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/user/v2"
	"github.com/gin-gonic/gin"
)

// Header provides fields common to all games.
type Header struct {
	Type          Type             `json:"type"`
	Title         string           `form:"title" json:"title"`
	Turn          int              `form:"turn" json:"turn" binding:"min=0"`
	Phase         Phase            `form:"phase" json:"phase" binding:"min=0"`
	SubPhase      SubPhase         `form:"sub-phase" json:"subPhase" binding:"min=0"`
	Round         int              `form:"round" json:"round" binding:"min=0"`
	NumPlayers    int              `form:"num-players" json:"numPlayers" binding"min=0,max=5"`
	Password      string           `form:"password" json:"-"`
	DefaultColors []string         `form:"default-colors" json:"defaultColors"`
	CreatorKey    *datastore.Key   `form:"creator-key" json:"creatorKey"`
	CreatorName   string           `form:"creator-name" json:"creatorName"`
	CreatorEmail  string           `form:"creator-emails" json:"creatorEmail"`
	UserKeys      []*datastore.Key `form:"user-keys" json:"userKeys"`
	UserNames     []string         `form:"user-names" json:"userNames"`
	UserEmails    []string         `form:"user-emails" json:"userEmails"`
	UserColors    []string         `form:"user-colors" json:"userColors"`
	OrderIDS      []int            `form:"order-ids" json:"-"`
	CPUserIndices []int            `form:"cp-user-indices" json:"cpUserIndices"`
	WinnerIDS     []int            `form:"winner-ids" json:"winnerIndices"`
	Status        Status           `form:"status" json:"status"`
	Progress      string           `form:"progress" json:"progress"`
	Options       []string         `form:"options" json:"options"`
	OptString     string           `form:"opt-string" json:"optString"`
	SavedState    []byte           `datastore:"SavedState,noindex" json:"-"`
	CreatedAt     time.Time        `form:"created-at" json:"createdAt"`
	UpdatedAt     time.Time        `form:"updated-at" json:"updatedAt"`
	UpdateCount   int              `json:"-"`
}

func (h *Header) AddCreator(u *user.User) {
	h.CreatorKey = u.Key
	h.CreatorName = u.Name
	h.CreatorEmail = u.Email
}

func (h *Header) AddUser(u *user.User) {
	h.UserKeys = append(h.UserKeys, u.Key)
	h.UserNames = append(h.UserNames, u.Name)
	h.UserEmails = append(h.UserEmails, u.Email)
}

// Returns (true, nil) if game should be started
func (h *Header) Accept(c *gin.Context, u *user.User) (bool, error) {
	log.Debugf("Entering")
	defer log.Debugf("Entering")

	err := h.validateAccept(c, u)
	if err != nil {
		return false, err
	}

	h.AddUser(u)
	if len(h.UserKeys) == h.NumPlayers {
		return true, nil
	}
	return false, nil
}

func (h *Header) validateAccept(c *gin.Context, u *user.User) error {
	switch {
	case len(h.UserKeys) >= h.NumPlayers:
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case h.HasUser(u):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case h.Password != "" && c.PostForm("password") != h.Password:
		return fmt.Errorf("%s provided incorrect password for Game %s: %w", u.Name, h.Title, ErrValidation)
	default:
		return nil
	}
}

func (h *Header) Drop(u *user.User) error {
	err := h.validateDrop(u)
	if err != nil {
		return err
	}

	h.RemoveUser(u)
	return nil
}

func (h *Header) validateDrop(u *user.User) error {
	switch {
	case h.Status != Recruiting:
		return fmt.Errorf("game is no longer recruiting, thus %s can't drop: %w", u.Name, ErrValidation)
	case !h.HasUser(u):
		return fmt.Errorf("%s has not joined this game, thus %s can't drop: %w", u.Name, u.Name, ErrValidation)
	default:
		return nil
	}
}

func (h *Header) RemoveUser(u *user.User) error {
	if u == nil {
		return ErrUserNotFound
	}

	index, found := h.IndexFor(u)
	if !found {
		return ErrUserNotFound
	}

	h.UserKeys = append(h.UserKeys[:index], h.UserKeys[index+1:]...)
	h.UserNames = append(h.UserNames[:index], h.UserNames[index+1:]...)
	h.UserEmails = append(h.UserEmails[:index], h.UserEmails[index+1:]...)

	return nil
}

func (h *Header) IndexFor(u *user.User) (int, bool) {
	if u == nil {
		return -1, false
	}

	for i, k2 := range h.UserKeys {
		if k2.Equal(u.Key) {
			return i, true
		}
	}
	return -1, false
}

func (h *Header) HasUser(u *user.User) bool {
	_, found := h.IndexFor(u)
	return found
}

type PhaseNameMap map[Phase]string
