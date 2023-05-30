package sn

import (
	"sort"

	"cloud.google.com/go/datastore"
	"github.com/elliotchance/pie/v2"
)

type Player struct {
	ID              PID
	Passed          bool
	PerformedAction bool
	Colors          []Color
	Stats
}

func (p Player) GetPID() PID {
	return p.ID
}

func (p *Player) setPID(pid PID) {
	p.ID = pid
}

func (p Player) GetPassed() bool {
	return p.Passed
}

func (p Player) GetPerformedAction() bool {
	return p.PerformedAction
}

func (p Player) GetStats() *Stats {
	return &(p.Stats)
}

func (p *Player) Reset() {
	p.PerformedAction = false
	p.Passed = false
}

type Playerer interface {
	GetPID() PID
	GetPassed() bool
	GetPerformedAction() bool
	GetStats() *Stats
	Reset()
	Copy() Playerer

	compareByScore(Playerer) Comparison
	New() Playerer
	setPID(PID)
}

type Players[P Playerer] []P

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

func (pid PID) ToIndex() UIndex {
	return UIndex(pid - 1)
}

const NoPID PID = 0

func (h Header) NameFor(pid PID) string {
	return h.UserNames[pid.ToIndex()]
}

func (h Header) NameByUID(uid UID) string {
	return h.NameByIndex(h.IndexFor(uid))
}

func (h Header) NameByIndex(i UIndex) string {
	if i >= 0 && i < UIndex(len(h.UserNames)) {
		return h.UserNames[i]
	}
	return ""
}
func (h Header) IndexFor(uid UID) UIndex {
	return UIndex(pie.FindFirstUsing(h.UserIDS, func(id UID) bool { return id == uid }))
}

func (h Header) EmailFor(pid PID) string {
	return h.UserEmails[pid.ToIndex()]
}

func (h Header) EmailNotificationsFor(pid PID) bool {
	return h.UserEmailNotifications[pid.ToIndex()]
}

func (h Header) GravTypeFor(pid PID) string {
	return h.UserGravTypes[pid.ToIndex()]
}

// NotFound indicates a value (e.g., player) was not found in the collection.
const NotFound = -1
const UIndexNotFound UIndex = -1

func sortedByScore[P Playerer](ps []P, c Comparison) {
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].compareByScore(ps[j]) == c })
}

func (p Player) compareByScore(p2 Playerer) Comparison {
	switch {
	case p.Score < p2.GetStats().Score:
		return LessThan
	case p.Score > p2.GetStats().Score:
		return GreaterThan
	default:
		return EqualTo
	}
}

func (g Game[P]) determinePlaces() (Results, error) {
	rs := make(Results)
	sortedByScore(g.Players, Descending)
	ps := g.copyPlayers()

	place := 1
	for len(ps) != 0 {
		// Find all players tied at place
		found := pie.Filter(ps, func(p P) bool { return ps[0].compareByScore(p) == EqualTo })
		// Get user keys for found players
		rs[place] = pie.Map(found, func(p P) *datastore.Key { return g.UserKeyFor(p.GetPID()) })
		// Set ps to remaining players
		_, ps = diff(ps, found, func(p1, p2 P) bool { return p1.GetPID() == p2.GetPID() })
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
