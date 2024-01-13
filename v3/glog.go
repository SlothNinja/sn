package sn

type entry struct {
	Data  Entry
	Lines []line
}

type line struct {
	Template string
	Data     Line
}

type glog []entry

//	type Entryer interface {
//		Init(Gamer)
//		PhaseName() string
//		Turn() int
//		Round() int
//		Player() Playerer
//		CreatedAt() time.Time
//		HTML() template.HTML
//	}
//
//	type Entry struct {
//		gamer         Gamer
//		PlayerID      PID
//		OtherPlayerID PID
//		TurnF         int
//		PhaseF        Phase
//		SubPhaseF     SubPhase
//		RoundF        int
//		CreatedAtF    time.Time
//	}

type Entry map[string]any
type Line map[string]any

func (g *Game[S, T, P]) NewEntry(template string, e Entry, l Line) {
	g.Log = append(g.Log, entry{
		Data:  e,
		Lines: []line{line{Template: template, Data: l}},
	})
}

func (g *Game[S, T, P]) AppendEntry(template string, l Line) {
	lastIndex := len(g.Log) - 1
	if lastIndex < 0 {
		Warningf("no log entry to append to")
		return
	}
	g.Log[lastIndex].Lines = append(g.Log[lastIndex].Lines, line{Template: template, Data: l})
}

func (g *Game[S, T, P]) AppendLine(l Line) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		Warningf("no log line to append to")
		return
	}
	lastLineIndex := len(g.Log[lastEntryIndex].Lines) - 1
	if lastLineIndex < 0 {
		Warningf("no log line to append to")
		return
	}
	for k, v := range l {
		g.Log[lastEntryIndex].Lines[lastLineIndex].Data[k] = v
	}
}

// func NewEntry(g Gamer) *Entry {
// 	h := g.GetHeader()
// 	return &Entry{
// 		gamer:         g,
// 		PlayerID:      NoPID,
// 		OtherPlayerID: NoPID,
// 		TurnF:         h.Turn,
// 		PhaseF:        h.Phase,
// 		SubPhaseF:     h.SubPhase,
// 		RoundF:        h.Round,
// 		CreatedAtF:    time.Now(),
// 	}
// }
//
// func NewEntryFor(p Playerer, g Gamer) *Entry {
// 	h := g.GetHeader()
// 	return &Entry{
// 		gamer:         g,
// 		PlayerID:      p.ID(),
// 		OtherPlayerID: NoPID,
// 		TurnF:         h.Turn,
// 		PhaseF:        h.Phase,
// 		SubPhaseF:     h.SubPhase,
// 		RoundF:        h.Round,
// 		CreatedAtF:    time.Now(),
// 	}
// }
//
// func (e *Entry) Init(g Gamer) {
// 	e.gamer = g
// }
//
// func (e *Entry) Phase() Phase {
// 	return e.PhaseF
// }
//
// func (e *Entry) Turn() int {
// 	return e.TurnF
// }
//
// func (e *Entry) Round() int {
// 	return e.RoundF
// }
//
// func (e *Entry) Player() Playerer {
// 	return e.gamer.PlayererByPID(e.PlayerID)
// }
//
// func (e *Entry) OtherPlayer() Playerer {
// 	return e.gamer.PlayererByPID(e.OtherPlayerID)
// }
//
// func (e *Entry) SetOtherPlayer(ps ...Playerer) {
// 	switch l := len(ps); {
// 	case l == 1 && ps[0] == nil:
// 		e.OtherPlayerID = NoPID
// 	case l == 1:
// 		e.OtherPlayerID = ps[0].ID()
// 	}
// }
//
// func (e *Entry) CreatedAt() time.Time {
// 	return e.CreatedAtF
// }
//
// func (e *Entry) Players() Playerers {
// 	return e.gamer.(GetPlayerers).GetPlayerers()
// }
//
// func (e *Entry) Winners() Playerers {
// 	return e.gamer.Winnerers()
// }
//
// type GameLog []Entryer
//
// func (l GameLog) Last() Entryer {
// 	if length := len(l); length > 0 {
// 		return l[length-1]
// 	}
// 	return nil
// }
//
// func (e *Entry) Game() Gamer {
// 	return e.gamer
// }
//
// func (l GameLog) String() []string {
// 	strings := make([]string, len(l))
// 	for i, e := range l {
// 		strings[i] = string(e.HTML())
// 	}
// 	return strings
// }
