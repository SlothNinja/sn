package sn

import (
	"github.com/elliotchance/pie/v2"
)

//	func init() {
//		gob.Register(NewPlayer())
//	}
type Comparison int

const (
	EqualTo     Comparison = 0
	LessThan    Comparison = -1
	GreaterThan Comparison = 1
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

//	type Player struct {
//		gamer           Gamer
//		user            *User
//		rating          *CurrentRating
//		IDF             PID  `form:"idf"`
//		PerformedAction bool `form:"performed-action"`
//		Score           int  `form:"score"`
//		Passed          bool `form:"passed"`
//		ColorMapF       []Color
//	}
//
// type Players []*Player
//
//	type jPlayer struct {
//		User            *User          `json:"user"`
//		Rating          *CurrentRating `json:"rating"`
//		IDF             PID            `json:"id"`
//		PerformedAction bool           `json:"performedAction"`
//		Score           int            `json:"score"`
//		Passed          bool           `json:"passed"`
//		ColorMap        []Color        `json:"colorMap"`
//	}
//
//	func (p *Player) MarshalJSON() ([]byte, error) {
//		j := &jPlayer{
//			User:            p.user,
//			Rating:          p.rating,
//			IDF:             p.IDF,
//			PerformedAction: p.PerformedAction,
//			Score:           p.Score,
//			Passed:          p.Passed,
//			ColorMap:        p.ColorMap(),
//		}
//		return json.Marshal(j)
//	}
//
//	type Playerer interface {
//		ID() PID
//		UIndex() UIndex
//		User() *User
//		Name() string
//		Color() Color
//		ColorMap() []Color
//	}
//
//	func (p *Player) CompareByScore(p2 *Player) (c Comparison) {
//		switch {
//		case p.Score < p2.Score:
//			c = LessThan
//		case p.Score > p2.Score:
//			c = GreaterThan
//		default:
//			c = EqualTo
//		}
//		return
//	}
//
// type Playerers []Playerer
//
//	func (p *Player) Game() Gamer {
//		return p.gamer
//	}
//
//	func (p *Player) SetGame(gamer Gamer) {
//		p.gamer = gamer
//	}
//
//	func NewPlayer() (p *Player) {
//		p = new(Player)
//		return
//	}
//
//	func (p *Player) ID() PID {
//		return p.IDF
//	}
//
//	func (p *Player) SetID(id PID) {
//		p.IDF = id
//	}
//
//	func (p *Player) ColorMap() []Color {
//		return p.ColorMapF
//	}
//
//	func (p *Player) SetColorMap(colors []Color) {
//		p.ColorMapF = colors
//	}
//
//	func (p *Player) Equal(p2 Playerer) bool {
//		return p2 != nil && p.ID() == p2.ID()
//	}
//
//	func (p *Player) NotEqual(p2 Playerer) bool {
//		return !p.Equal(p2)
//	}
//
//	func (p *Player) User() *User {
//		if p.user == nil {
//			p.user = p.gamer.User(p.UIndex())
//		}
//		return p.user
//	}
//
//	func (p *Player) UIndex() UIndex {
//		return p.ID().ToIndex()
//	}
//
//	func (h *Header) UserIDFor(pid PID) int64 {
//		l, uIndex := UIndex(len(h.UserIDS)), pid.ToIndex()
//		if uIndex >= 0 && uIndex < l {
//			return h.UserIDS[uIndex]
//		}
//		return 0
//	}
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

// func (h Header) UKeyFor(pid PID) *datastore.Key {
// 	return h.UserKeys[pid.ToIndex()]
// }

func (h Header) GravTypeFor(pid PID) string {
	return h.UserGravTypes[pid.ToIndex()]
}

// // Name provides the name of the player.
// // TODO: Deprecated in favor of NameFor.
// func (p *Player) Name() (s string) {
// 	if p != nil && p.User() != nil {
// 		s = p.User().Name
// 	}
// 	return
// }
//
// // Index provides the index in players for the player.
// // TODO: Deprecated in favor of IndexFor
// func (p *Player) UIndex() (index UIndex) {
// 	return UIndexFor(p, p.Game().(GetPlayerers).GetPlayerers())
// }

// // IndexFor returns the index for the player in players, if present.
// // Returns NotFound, the player not in players.
// func UIndexFor(p Playerer, ps Playerers) (index UIndex) {
// 	index = NotFound
// 	for i, p2 := range ps {
// 		if p.ID() == p2.ID() {
// 			index = i
// 			break
// 		}
// 	}
// 	return
// }

// NotFound indicates a value (e.g., player) was not found in the collection.
const NotFound = -1
const UIndexNotFound UIndex = -1

//
//	func (p *Player) Color() Color {
//		if p == nil {
//			return NoColor
//		}
//		colorMap := p.gamer.DefaultColorMap()
//		return colorMap[p.ID()]
//	}
//
//	func (ps Playerers) Colors() []Color {
//		cs := make([]Color, len(ps))
//		for i, p := range ps {
//			cs[i] = p.Color()
//		}
//		return cs
//	}
//
//	var textColors = map[Color]Color{
//		Yellow: Black,
//		Purple: White,
//		Green:  Yellow,
//		White:  Black,
//		Black:  White,
//	}
//
//	func (p *Player) TextColor() (c Color) {
//		var ok bool
//		if c, ok = textColors[p.Color()]; !ok {
//			c = Black
//		}
//		return
//	}
//
// A bit of a misnomer
// Returns whether the current user is the same as the player's user
// func (p *Player) IsCurrentUser(cu *User) bool {
// 	if p == nil {
// 		return false
// 	}
// 	return p.User().Equal(cu)
// }
//
// func (p *Player) IsCurrentPlayer() bool {
// 	for _, player := range p.gamer.CurrentPlayerers() {
// 		if player != nil && p.Equal(player) {
// 			return true
// 		}
// 	}
// 	return false
// }
//
// func (p *Player) IsWinner() (b bool) {
// 	for _, p2 := range p.gamer.Winnerers() {
// 		if b = p2 != nil && p.Equal(p2); b {
// 			break
// 		}
// 	}
// 	return
// }
//
// func (p *Player) Init(g Gamer) {
// 	p.SetGame(g)
// }
//
// func (p *Player) Gravatar() string {
// 	return fmt.Sprintf(`<a href="/user/show/%d" ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
// 		p.User().ID, p.User().Gravatar("80"), p.Color())
// }
//
// func (h *Header) GravatarFor(pid PID) template.HTML {
// 	return template.HTML(fmt.Sprintf(`<a href=%q ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
// 		h.UserPathFor(pid), GravatarURL(h.EmailFor(pid), "80", h.GravTypeFor(pid)), h.ColorFor(pid)))
// }
//
// func (h *Header) UserPathFor(pid PID) template.HTML {
// 	return PathFor(h.UserIDFor(pid))
// }
//
// func (h *Header) ColorFor(pid PID) Color {
// 	return h.DefaultColorMap()[pid]
// }
