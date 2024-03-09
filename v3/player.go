package sn

import (
	"sort"

	"github.com/elliotchance/pie/v2"
)

type Player struct {
	ID              PID
	Passed          bool
	PerformedAction bool
	Colors          []Color
	Stats
}

func (p *Player) PID() PID {
	if p == nil {
		return NoPID
	}
	return p.ID
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

func (p *Player) getPerformedAction() bool {
	return p.PerformedAction
}

func (p *Player) getStats() *Stats {
	if p == nil {
		return nil
	}
	return &(p.Stats)
}

func (p *Player) reset() {
	if p != nil {
		p.PerformedAction = false
		p.Passed = false
	}
}

func (p *Player) equal(op *Player) bool {
	return p != nil && op != nil && p.ID == op.ID
}

type Ptr[T any] interface {
	*T
}

type Playerer[T any] interface {
	Ptr[T]

	PID() PID
	setPID(PID)
	getPerformedAction() bool
	getStats() *Stats
	getScore() int64
	reset()
}

type Players[T any, P Playerer[T]] []P

type Comparison int

const (
	EqualTo     Comparison = 0
	LessThan    Comparison = -1
	GreaterThan Comparison = 1
	Ascending              = LessThan
	Descending             = GreaterThan
)

type UIndex int

func (uIndex UIndex) ToPID() PID {
	return PID(uIndex + 1)
}

type PID int

func (pid PID) ToUIndex() UIndex {
	return UIndex(pid - 1)
}

const NoPID PID = 0

func (h Header) NameFor(pid PID) string {
	return h.UserNames[pid.ToUIndex()]
}

func (h Header) UIDFor(pid PID) UID {
	return h.UserIDS[pid.ToUIndex()]
}

// func (h Header) NameByUID(uid UID) string {
// 	return h.NameByIndex(h.indexFor(uid))
// }

// func (h Header) NameByIndex(i UIndex) string {
// 	if i >= 0 && i < UIndex(len(h.UserNames)) {
// 		return h.UserNames[i]
// 	}
// 	return ""
// }

func (h Header) IndexFor(uid UID) UIndex {
	return UIndex(pie.FindFirstUsing(h.UserIDS, func(id UID) bool { return id == uid }))
}

func (h Header) EmailFor(pid PID) string {
	return h.UserEmails[pid.ToUIndex()]
}

func (h Header) EmailNotificationsFor(pid PID) bool {
	return h.UserEmailNotifications[pid.ToUIndex()]
}

func (h Header) GravTypeFor(pid PID) string {
	return h.UserGravTypes[pid.ToUIndex()]
}

// NotFound indicates a value (e.g., player) was not found in the collection.
const NotFound = -1
const UIndexNotFound UIndex = -1

func sortedByScore[T any, P Playerer[T]](ps []P, c Comparison) {
	sort.SliceStable(ps, func(i, j int) bool { return compareByScore(ps[i], ps[j]) == c })
}

func compareByScore[T any, P Playerer[T]](p1, p2 P) Comparison {
	switch {
	case p1.getScore() < p2.getStats().Score:
		return LessThan
	case p1.getScore() > p2.getStats().Score:
		return GreaterThan
	default:
		return EqualTo
	}
}

func (g *Game[S, T, P]) determinePlaces() (Results, error) {
	rs := make(Results)
	sortedByScore(g.Players, Descending)
	ps := g.copyPlayers()

	place := 1
	for len(ps) != 0 {
		// Find all players tied at place
		found := pie.Filter(ps, func(p2 P) bool { return compareByScore(ps[0], p2) == EqualTo })
		// Get user keys for found players
		rs[place] = pie.Map(found, func(p P) UID { return g.Header.UIDFor(p.PID()) })
		// Set ps to remaining players
		_, ps = diff(ps, found, func(p1, p2 P) bool { return p1.PID() == p2.PID() })
		// Above does not guaranty order so sort
		sortedByScore(ps, Descending)
		// Increase place by number of players added to current place
		place += len(rs[place])
	}
	// 	for _, p1 := range g.players {
	// 		rs[place] = append(rs[place], g.userKeyFor(p1.ID))
	// 		for _, p2 := range g.players {
	// 			if p1.ID != p2.ID && p1.compare(p2) != equalTo {
	// 				place++
	// 				break
	// 			}
	// 		}
	// 	}
	return rs, nil
}

func diff[T any](ss []T, against []T, equal func(T, T) bool) (added, removed []T) {
	// This is probably not the best way to do it. We do an O(n^2) between the
	// slices to see which items are missing in each direction.

	diffOneWay := func(ss1, ss2raw []T) (result []T) {
		ss2 := make([]T, len(ss2raw))
		copy(ss2, ss2raw)

		for _, s := range ss1 {
			found := false

			for i, element := range ss2 {
				if equal(s, element) {
					ss2 = append(ss2[:i], ss2[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				result = append(result, s)
			}
		}

		return
	}

	removed = diffOneWay(ss, against)
	added = diffOneWay(against, ss)

	return
}
