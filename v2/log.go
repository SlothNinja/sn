package sn

import (
	"html/template"
	"time"
)

type Entryer interface {
	PhaseName() string
	Turn() int
	Round() int
	CreatedAt() time.Time
	HTML() template.HTML
}

type Entry struct {
	PlayerID      int
	OtherPlayerID int
	TurnF         int
	PhaseF        Phase
	SubPhaseF     SubPhase
	RoundF        int
	CreatedAtF    time.Time
}

func NewEntry(h *Header) *Entry {
	return &Entry{
		PlayerID:      NoPlayerID,
		OtherPlayerID: NoPlayerID,
		TurnF:         h.Turn,
		PhaseF:        h.Phase,
		SubPhaseF:     h.SubPhase,
		RoundF:        h.Round,
		CreatedAtF:    time.Now(),
	}
}

func NewEntryFor(pid int, h *Header) *Entry {
	return &Entry{
		PlayerID:      pid,
		OtherPlayerID: NoPlayerID,
		TurnF:         h.Turn,
		PhaseF:        h.Phase,
		SubPhaseF:     h.SubPhase,
		RoundF:        h.Round,
		CreatedAtF:    time.Now(),
	}
}

func (e *Entry) Phase() Phase {
	return e.PhaseF
}

func (e *Entry) Turn() int {
	return e.TurnF
}

func (e *Entry) Round() int {
	return e.RoundF
}

func (e *Entry) CreatedAt() time.Time {
	return e.CreatedAtF
}

type GameLog []Entryer

func (l GameLog) Last() Entryer {
	if length := len(l); length > 0 {
		return l[length-1]
	}
	return nil
}

func (l GameLog) String() []string {
	strings := make([]string, len(l))
	for i, e := range l {
		strings[i] = string(e.HTML())
	}
	return strings
}
