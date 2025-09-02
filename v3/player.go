package sn

import (
	"cmp"
	"slices"
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

// UIndex represents a unique index value for a user
type UIndex int

// ToPID returns an id unique for the player associated with the user index value
func (uIndex UIndex) ToPID() PID {
	return PID(uIndex + 1)
}

// PID represent a unique id for a player
type PID int

// ToUIndex returns a unique index value for the user associated with the player id
func (pid PID) ToUIndex() UIndex {
	return UIndex(pid - 1)
}

// NoPID corresponds to the zero value for PID and represents the absence of a player id
const NoPID PID = 0

// NameFor returns the user name for the player associated with the player id
func (h Header) NameFor(pid PID) string {
	return h.UserNames[pid.ToUIndex()]
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

func (g *Game[S, T, P]) sortPlayers(compare func(PID, PID) int) {
	slices.SortFunc(g.Players, func(p1, p2 P) int { return compare(p1.PID(), p2.PID()) })
}

// Compare implements Comparer interface.
// Essentially, provides a fallback/default compare which ranks players
// in descending order of score. In other words, the more a player scores
// the earlier they are in finish order.
func (g *Game[S, T, P]) Compare(pid1, pid2 PID) int {
	return cmp.Compare(g.PlayerByPID(pid2).getScore(), g.PlayerByPID(pid1).getScore())
}
