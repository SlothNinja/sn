package sn

import (
	"html/template"
	"time"
)

type Entryer interface {
	Init(Gamer)
	PhaseName() string
	Turn() int
	Round() int
	Player() Playerer
	CreatedAt() time.Time
	HTML() template.HTML
}

type Entry struct {
	gamer         Gamer
	PlayerID      PID
	OtherPlayerID PID
	TurnF         int
	PhaseF        Phase
	SubPhaseF     SubPhase
	RoundF        int
	CreatedAtF    time.Time
}

func NewEntry(g Gamer) *Entry {
	h := g.GetHeader()
	return &Entry{
		gamer:         g,
		PlayerID:      NoPID,
		OtherPlayerID: NoPID,
		TurnF:         h.Turn,
		PhaseF:        h.Phase,
		SubPhaseF:     h.SubPhase,
		RoundF:        h.Round,
		CreatedAtF:    time.Now(),
	}
}

func NewEntryFor(p Playerer, g Gamer) *Entry {
	h := g.GetHeader()
	return &Entry{
		gamer:         g,
		PlayerID:      p.ID(),
		OtherPlayerID: NoPID,
		TurnF:         h.Turn,
		PhaseF:        h.Phase,
		SubPhaseF:     h.SubPhase,
		RoundF:        h.Round,
		CreatedAtF:    time.Now(),
	}
}

func (e *Entry) Init(g Gamer) {
	e.gamer = g
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

func (e *Entry) Player() Playerer {
	return e.gamer.PlayererByPID(e.PlayerID)
}

func (e *Entry) OtherPlayer() Playerer {
	return e.gamer.PlayererByPID(e.OtherPlayerID)
}

func (e *Entry) SetOtherPlayer(ps ...Playerer) {
	switch l := len(ps); {
	case l == 1 && ps[0] == nil:
		e.OtherPlayerID = NoPID
	case l == 1:
		e.OtherPlayerID = ps[0].ID()
	}
}

func (e *Entry) CreatedAt() time.Time {
	return e.CreatedAtF
}

func (e *Entry) Players() Playerers {
	return e.gamer.(GetPlayerers).GetPlayerers()
}

func (e *Entry) Winners() Playerers {
	return e.gamer.Winnerers()
}

type GameLog []Entryer

func (l GameLog) Last() Entryer {
	if length := len(l); length > 0 {
		return l[length-1]
	}
	return nil
}

func (e *Entry) Game() Gamer {
	return e.gamer
}

func (l GameLog) String() []string {
	strings := make([]string, len(l))
	for i, e := range l {
		strings[i] = string(e.HTML())
	}
	return strings
}
