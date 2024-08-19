package sn

import (
	"time"
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
	Status                    Status
	Undo                      Stack
	OptString                 string
	StartedAt                 time.Time
	EndedAt                   time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	Private                   bool
}

func (h Header) Users() []User {
	us := make([]User, len(h.UserIDS))
	for i, u := range us {
		u.Name = h.UserNames[i]
		u.Email = h.UserEmails[i]
		u.EmailHash = h.UserEmailHashes[i]
		u.EmailNotifications = h.UserEmailNotifications[i]
		u.GravType = h.UserGravTypes[i]
	}
	return us
}
