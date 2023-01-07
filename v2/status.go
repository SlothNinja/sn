package sn

import (
	"strings"

	"github.com/elliotchance/pie/v2"
)

// Status represents a status of game.
type Status string

// // Statuses provides an array of game statuses.
// type Statuses [6]Status

const (
	// NoStatus indicates a game having no status or an unknown status.
	NoStatus Status = ""
	// Recruiting indicates  game is recruiting players.
	Recruiting Status = "recruiting"
	// Completed indicates a game is completed.
	Completed Status = "completed"
	// Running indicates a game is running.
	Running Status = "running"
	// Abandoned indicates a player has abandoned the game.
	Abandoned Status = "abandoned"
	// Aborted indicates the game has been aborted.
	Aborted Status = "aborted"
)

func ToStatus(s string) Status {
	status := Status(strings.ToLower(s))
	if pie.Contains(statuses(), status) {
		return status
	}
	return NoStatus
}

// // ToStatus provides returns the appropriate status for the string key.
// var ToStatus = map[string]Status{
// 	"none":       NoStatus,
// 	"recruiting": Recruiting,
// 	"completed":  Completed,
// 	"running":    Running,
// 	"abandoned":  Abandoned,
// 	"aborted":    Aborted,
// }

// Statuses returns a slice of all supported game statuses.
func statuses() []Status {
	return []Status{NoStatus, Recruiting, Completed, Running, Abandoned, Aborted}
}

// var statusStrings = [6]string{"None", "Recruiting", "Completed", "Running", "Abandoned", "Aborted"}

// String returns a string representation of status.
func (s Status) String() string {
	return string(s)
	// return statusStrings[s]
}

// // Int returns an integer representation of status.
// func (s Status) Int() int {
// 	return int(s)
// }

// // MarshalJSON implements json.Marshaler interface to provide custom json marshalling.
// func (s *Status) MarshalJSON() (b []byte, err error) {
// 	return json.Marshal(s.String())
// }
//
// // UnmarshalJSON implements json.Unmarshaler interface to provide custom json unmarshalling
// func (s *Status) UnmarshalJSON(b []byte) (err error) {
// 	var (
// 		str string
// 	)
//
// 	if err = json.Unmarshal(b, &str); err == nil {
// 		*s, _ = ToStatus[strings.ToLower(str)]
// 	}
//
// 	return
// }
