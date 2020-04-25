package sn

import (
	"encoding/json"
	"strings"
)

// Status represents a status of game.
type Status int

// Statuses provides an array of game statuses.
type Statuses [6]Status

const (
	// NoStatus indicates a game having no status or an unknown status.
	NoStatus Status = iota
	// Recruiting indicates  game is recruiting players.
	Recruiting
	// Completed indicates a game is completed.
	Completed
	// Running indicates a game is running.
	Running
	// Abandoned indicates a player has abandoned the game.
	Abandoned
	// Aborted indicates the game has been aborted.
	Aborted
)

// ToStatus provides returns the appropriate status for the string key.
var ToStatus = map[string]Status{
	"none":       NoStatus,
	"recruiting": Recruiting,
	"completed":  Completed,
	"running":    Running,
	"abandoned":  Abandoned,
	"aborted":    Aborted,
}

// Statuses returns a slice of all supported game statuses.
func (h *Header) Statuses() Statuses {
	return Statuses{NoStatus, Recruiting, Completed, Running, Abandoned, Aborted}
}

var statusStrings = [6]string{"None", "Recruiting", "Completed", "Running", "Abandoned", "Aborted"}

// String returns a string representation of status.
func (s Status) String() string {
	return statusStrings[s]
}

// Int returns an integer representation of status.
func (s Status) Int() int {
	return int(s)
}

// MarshalJSON implements json.Marshaler interface to provide custom json marshalling.
func (s *Status) MarshalJSON() (b []byte, err error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler interface to provide custom json unmarshalling
func (s *Status) UnmarshalJSON(b []byte) (err error) {
	var (
		str string
	)

	if err = json.Unmarshal(b, &str); err == nil {
		*s, _ = ToStatus[strings.ToLower(str)]
	}

	return
}
