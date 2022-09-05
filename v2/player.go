package sn

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/SlothNinja/user"
)

func init() {
	gob.Register(NewPlayer())
}

type Comparison int

const (
	EqualTo     Comparison = 0
	LessThan    Comparison = -1
	GreaterThan Comparison = 1
)

const NoPlayerID int = -1

type Player struct {
	gamer           Gamer
	user            *user.User
	rating          *CurrentRating
	IDF             int  `form:"idf"`
	PerformedAction bool `form:"performed-action"`
	Score           int  `form:"score"`
	Passed          bool `form:"passed"`
	ColorMapF       Colors
}
type Players []*Player

type jPlayer struct {
	User            *user.User     `json:"user"`
	Rating          *CurrentRating `json:"rating"`
	IDF             int            `json:"id"`
	PerformedAction bool           `json:"performedAction"`
	Score           int            `json:"score"`
	Passed          bool           `json:"passed"`
	ColorMap        []string       `json:"colorMap"`
}

func (p *Player) MarshalJSON() ([]byte, error) {
	j := &jPlayer{
		User:            p.user,
		Rating:          p.rating,
		IDF:             p.IDF,
		PerformedAction: p.PerformedAction,
		Score:           p.Score,
		Passed:          p.Passed,
		ColorMap:        p.ColorMap().Strings(),
	}
	return json.Marshal(j)
}

type Playerer interface {
	ID() int
	Index() int
	User() *user.User
	Name() string
	Color() Color
	ColorMap() Colors
}

func (p *Player) CompareByScore(p2 *Player) (c Comparison) {
	switch {
	case p.Score < p2.Score:
		c = LessThan
	case p.Score > p2.Score:
		c = GreaterThan
	default:
		c = EqualTo
	}
	return
}

type Playerers []Playerer

func (p *Player) Game() Gamer {
	return p.gamer
}

func (p *Player) SetGame(gamer Gamer) {
	p.gamer = gamer
}

func NewPlayer() (p *Player) {
	p = new(Player)
	return
}

func (p *Player) ID() int {
	return p.IDF
}

func (p *Player) SetID(id int) {
	p.IDF = id
}

func (p *Player) ColorMap() Colors {
	return p.ColorMapF
}

func (p *Player) SetColorMap(colors Colors) {
	p.ColorMapF = colors
}

func (p *Player) Equal(p2 Playerer) bool {
	return p2 != nil && p.ID() == p2.ID()
}

func (p *Player) NotEqual(p2 Playerer) bool {
	return !p.Equal(p2)
}

func (p *Player) User() *user.User {
	if p.user == nil {
		p.user = p.gamer.User(p.ID())
	}
	return p.user
}

func (h *Header) UserIDFor(p Playerer) (id int64) {
	if l, pid := len(h.UserIDS), p.ID(); pid >= 0 && pid < l {
		id = h.UserIDS[p.ID()]
	}
	return
}

func (h *Header) NameFor(p Playerer) (n string) {
	if p != nil {
		n = h.NameByPID(p.ID())
	}
	return
}

func (h *Header) NameByPID(pid int) (n string) {
	if l := len(h.UserNames); pid >= 0 && pid < l {
		n = h.UserNames[pid]
	}
	return
}

func (h *Header) NameByUID(uid int64) (n string) {
	var index int = NotFound
	for i := range h.UserIDS {
		if uid == h.UserIDS[i] {
			index = i
			break
		}
	}

	if index != NotFound {
		n = h.NameByPID(index)
	}
	return
}

func (h *Header) IndexFor(uid int64) int {
	for i := range h.UserIDS {
		if uid == h.UserIDS[i] {
			return i
		}
	}
	return NotFound
}

func (h *Header) EmailFor(p Playerer) (em string) {
	if l, pid := len(h.UserEmails), p.ID(); pid >= 0 && pid < l {
		em = h.UserEmails[p.ID()]
	}
	return
}

func (h *Header) GravTypeFor(p Playerer) string {
	l, pid := len(h.UserGravTypes), p.ID()
	if pid >= 0 && pid < l {
		return h.UserGravTypes[p.ID()]
	}
	return ""
}

// Name provides the name of the player.
// TODO: Deprecated in favor of NameFor.
func (p *Player) Name() (s string) {
	if p != nil && p.User() != nil {
		s = p.User().Name
	}
	return
}

// Index provides the index in players for the player.
// TODO: Deprecated in favor of IndexFor
func (p *Player) Index() (index int) {
	return IndexFor(p, p.Game().(GetPlayerers).GetPlayerers())
}

// IndexFor returns the index for the player in players, if present.
// Returns NotFound, the player not in players.
func IndexFor(p Playerer, ps Playerers) (index int) {
	index = NotFound
	for i, p2 := range ps {
		if p.ID() == p2.ID() {
			index = i
			break
		}
	}
	return
}

// NotFound indicates a value (e.g., player) was not found in the collection.
const NotFound = -1

func (p *Player) Color() Color {
	if p == nil {
		return None
	}
	colorMap := p.gamer.DefaultColorMap()
	// if cu != nil {
	// 	if player := p.gamer.PlayererByUserID(cu.ID()); player != nil {
	// 		colorMap = player.ColorMap()
	// 	}
	// }
	return colorMap[p.ID()]
}

func (ps Playerers) Colors() Colors {
	cs := make(Colors, len(ps))
	for i, p := range ps {
		cs[i] = p.Color()
	}
	return cs
}

var textColors = map[Color]Color{
	Yellow: Black,
	Purple: White,
	Green:  Yellow,
	White:  Black,
	Black:  White,
}

func (p *Player) TextColor() (c Color) {
	var ok bool
	if c, ok = textColors[p.Color()]; !ok {
		c = Black
	}
	return
}

// A bit of a misnomer
// Returns whether the current user is the same as the player's user
func (p *Player) IsCurrentUser(cu *user.User) bool {
	if p == nil {
		return false
	}
	return p.User().Equal(cu)
}

func (p *Player) IsCurrentPlayer() bool {
	for _, player := range p.gamer.CurrentPlayerers() {
		if player != nil && p.Equal(player) {
			return true
		}
	}
	return false
}

func (p *Player) IsWinner() (b bool) {
	for _, p2 := range p.gamer.Winnerers() {
		if b = p2 != nil && p.Equal(p2); b {
			break
		}
	}
	return
}

func (p *Player) Init(g Gamer) {
	p.SetGame(g)
}

func (p *Player) Gravatar() string {
	return fmt.Sprintf(`<a href="/user/show/%d" ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
		p.User().ID, p.User().Gravatar("80"), p.Color())
}

func (h *Header) GravatarFor(p Playerer) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href=%q ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
		h.UserPathFor(p), user.GravatarURL(h.EmailFor(p), "80", h.GravTypeFor(p)), p.Color()))
}

func (h *Header) UserPathFor(p Playerer) template.HTML {
	return user.PathFor(h.UserIDFor(p))
}
