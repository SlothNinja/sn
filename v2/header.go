package sn

import (
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/user/v2"
	"golang.org/x/crypto/bcrypt"
)

// Header provides fields common to all games.
type Header struct {
	Type             Type             `json:"type"`
	Title            string           `form:"title" json:"title"`
	Turn             int              `form:"turn" json:"turn" binding:"min=0"`
	NumPlayers       int              `form:"num-players" json:"numPlayers" binding"min=0,max=5"`
	Password         []byte           `form:"password" json:"-"`
	DefaultColors    []string         `form:"default-colors" json:"defaultColors"`
	CreatorKey       *datastore.Key   `form:"creator-key" json:"creatorKey"`
	CreatorName      string           `form:"creator-name" json:"creatorName"`
	CreatorEmail     string           `form:"creator-email" json:"creatorEmail"`
	CreatorEmailHash string           `form:"creator-email-hash" json:"creatorEmailHash"`
	UserKeys         []*datastore.Key `form:"user-keys" json:"userKeys"`
	UserNames        []string         `form:"user-names" json:"userNames"`
	UserEmails       []string         `form:"user-emails" json:"userEmails"`
	UserEmailHashes  []string         `form:"user-emails-hashes" json:"userEmailHashes"`
	UserColors       []string         `form:"user-colors" json:"userColors"`
	OrderIDS         []int            `form:"order-ids" json:"-"`
	CPIDS            []int            `form:"cpIDS" json:"cpids"`
	WinnerIDS        []int            `form:"winner-ids" json:"winnerIndices"`
	Status           Status           `form:"status" json:"status"`
	Undo             Stack            `json:"undo"`
	SavedState       []byte           `datastore:"SavedState,noindex" json:"-"`
	CreatedAt        time.Time        `form:"created-at" json:"createdAt"`
	UpdatedAt        time.Time        `form:"updated-at" json:"updatedAt"`
	StartedAt        time.Time        `form:"started-at" json:"startedAt"`
}

func (h Header) MarshalJSON() ([]byte, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	type jHeader Header

	return json.Marshal(struct {
		jHeader
		Creator          *User   `json:"creator"`
		Users            []*User `json:"users"`
		LastUpdated      string  `json:"lastUpdated"`
		Public           bool    `json:"public"`
		CreatorEmail     omit    `json:"creatorEmail,omitempty"`
		CreatorEmailHash omit    `json:"creatorEmailHash,omitempty"`
		CreatorKey       omit    `json:"creatorKey,omitempty"`
		CreatorName      omit    `json:"creatorName,omitempty"`
		UserEmails       omit    `json:"userEmails,omitempty"`
		UserEmailHashes  omit    `json:"userEmailHashes,omitempty"`
		UserKeys         omit    `json:"userKeys,omitempty"`
		UserNames        omit    `json:"userNames,omitempty"`
	}{
		jHeader:     jHeader(h),
		Creator:     ToUser(h.CreatorKey, h.CreatorName, h.CreatorEmailHash),
		Users:       toUsers(h.UserKeys, h.UserNames, h.UserEmailHashes),
		LastUpdated: LastUpdated(h.UpdatedAt),
		Public:      len(h.Password) == 0,
	})
}

type omit *struct{}

type User struct {
	*user.User
}

func (u User) MarshalJSON() ([]byte, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	jUser, err := json.Marshal(u.User)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(jUser, &data)
	if err != nil {
		return nil, err
	}

	delete(data, "lcname")
	delete(data, "email")
	delete(data, "joined")
	delete(data, "createdat")
	delete(data, "updatedat")
	delete(data, "admin")

	return json.Marshal(data)
}

func ToUser(k *datastore.Key, name, hash string) *User {
	var id int64 = -1
	if k != nil {
		id = k.ID
	}
	u := &User{User: user.New(id)}
	u.Name = name
	u.EmailHash = hash
	return u
}

func toUsers(ks []*datastore.Key, names, hashes []string) []*User {
	us := make([]*User, len(ks))
	for i := range ks {
		us[i] = ToUser(ks[i], names[i], hashes[i])
	}
	return us
}

func (h *Header) AddCreator(u *user.User) {
	h.CreatorKey = u.Key
	h.CreatorName = u.Name
	h.CreatorEmail = u.Email
	h.CreatorEmailHash = u.EmailHash
}

func (h *Header) AddUser(u *user.User) {
	h.UserKeys = append(h.UserKeys, u.Key)
	h.UserNames = append(h.UserNames, u.Name)
	h.UserEmails = append(h.UserEmails, u.Email)
	h.UserEmailHashes = append(h.UserEmailHashes, u.EmailHash)
}

// Returns (true, nil) if game should be started
func (h *Header) Accept(u *user.User, pwd []byte) (bool, error) {
	err := h.validateAccept(u, pwd)
	if err != nil {
		return false, err
	}

	h.AddUser(u)
	if len(h.UserKeys) == int(h.NumPlayers) {
		return true, nil
	}
	return false, nil
}

func (h *Header) validateAccept(u *user.User, pwd []byte) error {
	switch {
	case len(h.UserKeys) >= int(h.NumPlayers):
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case h.HasUser(u.ID()):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case len(h.Password) != 0:
		err := bcrypt.CompareHashAndPassword(h.Password, pwd)
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

	h.RemoveUser(u.ID())
	return nil
}

func (h *Header) validateDrop(u *user.User) error {
	switch {
	case h.Status != Recruiting:
		return fmt.Errorf("game is no longer recruiting, thus %s can't drop: %w", u.Name, ErrValidation)
	case !h.HasUser(u.ID()):
		return fmt.Errorf("%s has not joined this game, thus %s can't drop: %w", u.Name, u.Name, ErrValidation)
	default:
		return nil
	}
}

func (h *Header) RemoveUser(uid int64) error {
	index, found := h.IndexFor(uid)
	if !found {
		return ErrUserNotFound
	}

	h.UserKeys = append(h.UserKeys[:index], h.UserKeys[index+1:]...)
	h.UserNames = append(h.UserNames[:index], h.UserNames[index+1:]...)
	h.UserEmails = append(h.UserEmails[:index], h.UserEmails[index+1:]...)

	return nil
}

func (h *Header) IndexFor(uid int64) (int, bool) {
	for i, k2 := range h.UserKeys {
		if k2.ID == uid {
			return i, true
		}
	}
	return -1, false
}

func (h *Header) HasUser(uid int64) bool {
	_, found := h.IndexFor(uid)
	return found
}

type PhaseNameMap map[Phase]string

func (h *Header) EmailFor(uid int64) string {
	i, found := h.IndexFor(uid)
	if !found {
		return ""
	}
	return h.UserEmails[i]
}
