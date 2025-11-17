package sn

import (
	"math/rand/v2"
	"time"

	"github.com/elliotchance/pie/v2"
)

// Player represents a player of the game
type Player struct {
	ID              PID
	Passed          bool
	PerformedAction bool
	CanFinish       bool
	Colors          []Color
	Log             *entry
	Stats
}

// Stats represents players stats for a single game
type Stats struct {
	// Number of points scored by player
	Score int64
	// Number of games played at player count
	GamesPlayed int64
	// Number of games won at player count
	Won int64
	// Number of moves made by player
	Moves int64
	// Amount of time passed between player moves by player
	Think time.Duration
	// Position player finished (e.g., 1st, 2nd, etc.)
	Finish int
}

// PID returns a unique id for the player
func (p *Player) PID() PID {
	if p == nil {
		return NoPID
	}
	return p.ID
}

// UIndex returns the UIndex (user index) associated with the Player
func (p *Player) UIndex() UIndex {
	return UIndex(p.PID()) - 1
}

func (p *Player) setPID(pid PID) {
	if p != nil {
		p.ID = pid
	}
}

func (p *Player) getPassed() bool {
	return p.Passed
}

func (p *Player) getScore() int64 {
	return p.Score
}

func (p *Player) setLog(e *entry) {
	p.Log = e
}

func (p *Player) getPerformedAction() bool {
	return p.PerformedAction
}

func (p *Player) getCanFinish() bool {
	return p.CanFinish
}

func (p *Player) getStats() *Stats {
	if p == nil {
		return nil
	}
	return &(p.Stats)
}

// Clear sets PerformedAction and CanFinish to false
// Clear set Log to nil
// Clear panics if p == nil
func (p *Player) Clear() {
	p.PerformedAction = false
	p.CanFinish = false
	p.Log = nil
}

// Reset clears player via Clear() and sets Passed to false
// Reset panics if p == nil
func (p *Player) Reset() {
	p.Clear()
	p.Passed = false
}

// func (p *Player) equal(op *Player) bool {
// 	return p != nil && op != nil && p.ID == op.ID
// }

type ptr[T any] interface {
	*T
}

type playerer[T any] interface {
	PID() PID
	Reset()
	Clear()

	ptr[T]
	setPID(PID)
	getPerformedAction() bool
	getCanFinish() bool
	getStats() *Stats
	getScore() int64
	setLog(*entry)
}

// Players represents players of the game
type Players[T any, P playerer[T]] []P

// PID represent a unique id for a player
type PID int

// ToUIndex returns a unique index value for the user associated with the player id
func (pid PID) ToUIndex() UIndex {
	return UIndex(pid - 1)
}

// NoPID corresponds to the zero value for PID and represents the absence of a player id
const NoPID PID = 0

// PIDS returns the player identiers for the players slice
func (ps Players[T, P]) PIDS() []PID {
	return pie.Map(ps, func(p P) PID { return p.PID() })
}

// IndexFor returns the index for the given player in Players slice
// Also, return true if player found, and false if player not found in Players slice
func (ps Players[T, P]) IndexFor(p1 P) (int, bool) {
	const notFound = -1
	const found = true
	index := pie.FindFirstUsing(ps, func(p2 P) bool { return p1.PID() == p2.PID() })
	if index == notFound {
		return index, !found
	}
	return index, found
}

// Randomize randomizes the order of the players in the Players slice
func (ps Players[T, P]) Randomize() {
	rand.Shuffle(len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] })
}
