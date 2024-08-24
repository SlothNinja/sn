package sn

import (
	"fmt"
	"log/slog"
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

func (h Header) Users() []*User {
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
	slog.Debug(fmt.Sprintf("Users: %#v", us))
	return us
}
