package sn

import (
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/undo"
	"github.com/SlothNinja/user"
	"golang.org/x/crypto/bcrypt"
)

// Header provides fields common to all games.
type Header struct {
	Type                      Type             `json:"type"`
	Title                     string           `json:"title"`
	Turn                      int              `json:"turn"`
	Round                     int              `json:"round"`
	NumPlayers                int              `json:"numPlayers"`
	Password                  string           `json:"-"`
	PasswordHash              []byte           `json:"-"`
	CreatorID                 int64            `json:"creatorId"`
	CreatorName               string           `json:"creatorName"`
	CreatorEmail              string           `json:"creatorEmail"`
	CreatorEmailNotifications bool             `json:"creatorEmailNotifications"`
	CreatorEmailHash          string           `json:"creatorEmailHash"`
	CreatorGravType           string           `json:"creatorGravType"`
	UserIDS                   []int64          `json:"userIds"`
	UserKeys                  []*datastore.Key `json:"userKeys"`
	UserNames                 []string         `json:"userNames"`
	UserEmails                []string         `json:"userEmails"`
	UserEmailHashes           []string         `json:"userEmailHashes"`
	UserEmailNotifications    []bool           `json:"userEmailNotifications"`
	UserGravTypes             []string         `json:"userGravTypes"`
	OrderIDS                  []PID            `json:"-"`
	CPIDS                     []PID            `json:"cpids"`
	WinnerKeys                []*datastore.Key `json:"winnerKeys"`
	Status                    Status           `json:"status"`
	Undo                      undo.Stack       `json:"undo"`
	Progress                  string           `json:"progress"`
	Options                   []string         `json:"options"`
	OptString                 string           `json:"optString"`
	StartedAt                 time.Time        `json:"starteddAt"`
	CreatedAt                 time.Time        `json:"createdAt"`
	UpdatedAt                 time.Time        `json:"updatedAt"`
	EndedAt                   time.Time        `json:"endedAt"`
}

func (h Header) DMap() map[string]interface{} {
	dm := make(map[string]interface{})
	dm["type"] = h.Type
	dm["title"] = h.Title
	dm["turn"] = h.Turn
	dm["round"] = h.Round
	dm["numPlayers"] = h.NumPlayers
	dm["creatorId"] = h.CreatorID
	dm["creatorName"] = h.CreatorName
	dm["creatorEmail"] = h.CreatorEmail
	dm["creatorEmailNotifications"] = h.CreatorEmailNotifications
	dm["creatorEmailHash"] = h.CreatorEmailHash
	dm["creatorGravType"] = h.CreatorGravType
	dm["userIds"] = h.UserIDS
	dm["userKeys"] = h.UserKeys
	dm["userNames"] = h.UserNames
	dm["userEmails"] = h.UserEmails
	dm["userEmailHashes"] = h.UserEmailHashes
	dm["userEmailNotifications"] = h.UserEmailNotifications
	dm["userGravTypes"] = h.UserGravTypes
	dm["cpids"] = h.CPIDS
	dm["winnerKeys"] = h.WinnerKeys
	dm["status"] = h.Status
	dm["undo"] = h.Undo
	dm["progress"] = h.Progress
	dm["options"] = h.Options
	dm["optString"] = h.OptString
	dm["starteddAt"] = h.StartedAt
	dm["createdAt"] = h.CreatedAt
	dm["updatedAt"] = h.UpdatedAt
	dm["endedAt"] = h.EndedAt
	return dm
}

func include(ints []int64, i int64) bool {
	for _, v := range ints {
		if v == i {
			return true
		}
	}
	return false
}

func (h Header) hasUser(u *user.User) bool {
	return u != nil && include(h.UserIDS, u.ID())
}

func (h *Header) RemoveUser(u2 *user.User) {
	i, found := h.IndexFor(u2.ID())
	if !found {
		return
	}

	if i >= 0 && i < len(h.UserIDS) {
		h.UserIDS = append(h.UserIDS[:i], h.UserIDS[i+1:]...)
	}
	if i >= 0 && i < len(h.UserKeys) {
		h.UserKeys = append(h.UserKeys[:i], h.UserKeys[i+1:]...)
	}
	if i >= 0 && i < len(h.UserNames) {
		h.UserNames = append(h.UserNames[:i], h.UserNames[i+1:]...)
	}
	if i >= 0 && i < len(h.UserEmails) {
		h.UserEmails = append(h.UserEmails[:i], h.UserEmails[i+1:]...)
	}
	if i >= 0 && i < len(h.UserEmailHashes) {
		h.UserEmailHashes = append(h.UserEmailHashes[:i], h.UserEmailHashes[i+1:]...)
	}
	if i >= 0 && i < len(h.UserEmailNotifications) {
		h.UserEmailNotifications = append(h.UserEmailNotifications[:i], h.UserEmailNotifications[i+1:]...)
	}
	if i >= 0 && i < len(h.UserGravTypes) {
		h.UserGravTypes = append(h.UserGravTypes[:i], h.UserGravTypes[i+1:]...)
	}
}

func (h *Header) AddUser(u *user.User) {
	h.UserIDS = append(h.UserIDS, u.ID())
	h.UserKeys = append(h.UserKeys, u.Key)
	h.UserNames = append(h.UserNames, u.Name)
	h.UserEmails = append(h.UserEmails, u.Email)
	h.UserEmailHashes = append(h.UserEmailHashes, u.EmailHash)
	h.UserEmailNotifications = append(h.UserEmailNotifications, u.EmailNotifications)
	h.UserGravTypes = append(h.UserGravTypes, u.GravType)
}

func (h *Header) AddCreator(u *user.User) {
	h.CreatorID = u.ID()
	h.CreatorName = u.Name
	h.CreatorEmail = u.Email
	h.CreatorEmailHash = u.EmailHash
	h.CreatorEmailNotifications = u.EmailNotifications
	h.CreatorGravType = u.GravType
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
	case h.hasUser(u):
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
	case !h.hasUser(u):
		return fmt.Errorf("%s has not joined this game, thus %s can't drop: %w", u.Name, u.Name, ErrValidation)
	default:
		return nil
	}
}
