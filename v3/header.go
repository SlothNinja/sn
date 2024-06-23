package sn

import (
	"time"
)

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
