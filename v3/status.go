package sn

// Status represents a status of game.
type Status string

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

// statuses returns a slice of all supported game statuses.
func statuses() []Status {
	return []Status{NoStatus, Recruiting, Completed, Running, Abandoned, Aborted}
}

// String returns a string representation of status.
func (s Status) String() string {
	return string(s)
}
