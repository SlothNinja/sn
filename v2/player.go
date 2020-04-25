package sn

import (
	"encoding/json"

	"github.com/SlothNinja/color"
	"github.com/SlothNinja/user/v2"
)

type Comparison int

const (
	EqualTo     Comparison = 0
	LessThan    Comparison = -1
	GreaterThan Comparison = 1
)

const NoPlayerID int = -1

type Player struct {
	User            *user.User   `json:"user"`
	ID              int          `form:"id" json:"id"`
	PerformedAction bool         `form:"performed-action" json:"performed-action"`
	Score           int          `form:"score" json:"score"`
	Passed          bool         `form:"passed" json:"passed"`
	ColorMapF       color.Colors `json:"colorMap"`
}
type Players []*Player

func (p *Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(p)
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
