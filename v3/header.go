package sn

import (
	"github.com/elliotchance/pie/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Phase represents a game phase
type Phase string

// Header provides fields common to all games.
type Header struct {
	ID                        string `firestore:"-"`
	Type                      Type
	Title                     string
	Turn                      int
	Phase                     Phase
	Round                     int
	NumPlayers                int
	CreatorID                 UID
	CreatorName               string
	CreatorEmail              string
	CreatorEmailNotifications bool
	CreatorEmailHash          string
	CreatorGravType           string
	UserIDS                   []UID
	UserNames                 []string
	UserEmails                []string
	UserEmailHashes           []string
	UserEmailNotifications    []bool
	UserGravTypes             []string
	OrderIDS                  []PID
	CPIDS                     []PID
	WinnerIDS                 []UID
	Places                    placesSMap
	Status                    Status
	Undo                      Stack `firestore:"-"`
	OptString                 string
	StartedAt                 *timestamppb.Timestamp
	EndedAt                   *timestamppb.Timestamp
	CreatedAt                 *timestamppb.Timestamp
	UpdatedAt                 *timestamppb.Timestamp
	Private                   bool
}

func (h *Header) users() []*User {
	us := make([]*User, len(h.UserIDS))
	for i := range us {
		us[i] = &User{
			ID: h.UserIDS[i],
			userData: userData{
				Name:               h.UserNames[i],
				Email:              h.UserEmails[i],
				EmailHash:          h.UserEmailHashes[i],
				EmailNotifications: h.UserEmailNotifications[i],
				GravType:           h.UserGravTypes[i],
			},
		}
	}
	Debugf("Users: %#v", us)
	return us
}

func (h *Header) id() string {
	return h.ID
}

func (h *Header) setID(id string) {
	h.ID = id
}

// EmailFor returns the user email for the player associated with the player id
func (h Header) EmailFor(pid PID) string {
	return h.UserEmails[pid.ToUIndex()]
}

// EmailNotificationsFor returns whether email notifications are to be sent for the player associated with the player id
func (h Header) EmailNotificationsFor(pid PID) bool {
	return h.UserEmailNotifications[pid.ToUIndex()]
}

// GravTypeFor returns the gravatar type for the player associated with the player id
func (h Header) GravTypeFor(pid PID) string {
	return h.UserGravTypes[pid.ToUIndex()]
}

// UIDFor returns the user id for the player associated with the player id
func (h Header) UIDFor(pid PID) UID {
	return h.UserIDS[pid.ToUIndex()]
}

// PIDFor returns the player id for the user associated with the user id
// If no user associated with player id, return 0
func (h Header) PIDFor(uid UID) PID {
	index, found := h.IndexFor(uid)
	if !found {
		return 0
	}
	return index.ToPID()
}

// IndexFor return the user index associated with the user id.
// Also, returns a boolean indicating whether a user index was found for the user id.
func (h Header) IndexFor(uid UID) (index UIndex, found bool) {
	const notFound = -1
	index = UIndex(pie.FindFirstUsing(h.UserIDS, func(id UID) bool { return id == uid }))
	if index == notFound {
		return notFound, false
	}
	return index, true
}

// NameFor returns the user name for the player associated with the player id
func (h Header) NameFor(pid PID) string {
	return h.UserNames[pid.ToUIndex()]
}
