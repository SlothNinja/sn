package sn

import (
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Gamers []Gamer
type Gamer interface {
	PhaseName() string
	FromParams(*gin.Context, *user.User, Type) error
	ColorMapFor(*user.User) ColorMap
	headerer
}

type GetPlayerers interface {
	GetPlayerers() Playerers
}

type hasUpdate interface {
	Update(*gin.Context) (string, ActionType, error)
}

func GamesRoot(c *gin.Context) *datastore.Key {
	return datastore.NameKey("Games", "root", nil)
}

func (h *Header) GetAcceptDialog() bool {
	return h.Private()
}

func (h *Header) RandomTurnOrder() {
	ps := h.gamer.(GetPlayerers).GetPlayerers()
	for i := 0; i < h.NumPlayers; i++ {
		ri := MyRand.Intn(h.NumPlayers)
		ps[i], ps[ri] = ps[ri], ps[i]
	}
	h.SetCurrentPlayerers(ps[0])

	h.OrderIDS = make(UserIndices, len(ps))
	for i, p := range ps {
		h.OrderIDS[i] = p.ID()
	}
}

// Returns (true, nil) if game should be started
func (h *Header) Accept(c *gin.Context, u *user.User) (start bool, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	err = h.validateAccept(c, u)
	if err != nil {
		return false, err
	}

	h.AddUser(u)
	log.Debugf("h: %#v", h)
	if len(h.UserIDS) == h.NumPlayers {
		return true, nil
	}
	return false, nil
}

func (h *Header) validateAccept(c *gin.Context, u *user.User) error {
	switch {
	case len(h.UserIDS) >= h.NumPlayers:
		return NewVError("Game already has the maximum number of players.")
	case h.HasUser(u):
		return NewVError("%s has already accepted this invitation.", u.Name)
	case h.Password != "" && c.PostForm("password") != h.Password:
		return NewVError("%s provided incorrect password for Game #%d: %s.", u.Name, h.ID, h.Title)
	}
	return nil
}

// Returns (true, nil) if game should be started
func (h *Header) AcceptWith(u *user.User, pwd []byte) (bool, error) {
	err := h.validateAcceptWith(u, pwd)
	if err != nil {
		return false, err
	}

	h.AddUser(u)
	if len(h.UserIDS) == int(h.NumPlayers) {
		return true, nil
	}
	return false, nil
}

func (h *Header) validateAcceptWith(u *user.User, pwd []byte) error {
	log.Debugf("PasswordHash: %v", h.PasswordHash)
	switch {
	case len(h.UserIDS) >= int(h.NumPlayers):
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case h.HasUser(u):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case len(h.PasswordHash) != 0:
		err := bcrypt.CompareHashAndPassword(h.PasswordHash, pwd)
		if err != nil {
			log.Warningf(err.Error())
			return fmt.Errorf("%s provided incorrect password for Game %s: %w",
				u.Name, h.Title, ErrValidation)
		}
		return nil
	default:
		return nil
	}
}

func (h *Header) Drop(u *user.User) (err error) {
	if err = h.validateDrop(u); err != nil {
		return
	}

	h.RemoveUser(u)
	return
}

func (h *Header) validateDrop(u *user.User) (err error) {
	switch {
	case h.Status != Recruiting:
		err = NewVError("Game is no longer recruiting, thus %s can't drop.", u.Name)
	case !h.HasUser(u):
		err = NewVError("%s has not joined this game, thus %s can't drop.", u.Name, u.Name)
	}
	return
}
