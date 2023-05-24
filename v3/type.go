package sn

import (
	"strings"

	"github.com/elliotchance/pie/v2"
)

type Type string

const (
	NoType     Type = ""
	ATF        Type = "atf"
	Confucius  Type = "confucius"
	GOT        Type = "got"
	Gettysburg Type = "gettysburg"
	Indonesia  Type = "indonesia"
	Plateau    Type = "plateau"
	Tammany    Type = "tammany"
	All        Type = "all"
)

func types() []Type {
	return []Type{NoType, ATF, Confucius, GOT, Gettysburg, Indonesia, Plateau, Tammany, All}
}

func (t Type) String() string {
	s, ok := map[Type]string{
		Confucius:  "Confucius",
		Tammany:    "Tammany Hall",
		ATF:        "After The Flood",
		GOT:        "Guild Of Thieves",
		Indonesia:  "Indonesia",
		Plateau:    "Le Plateau",
		Gettysburg: "Gettysburg",
		All:        "All",
	}[t]
	if ok {
		return s
	}
	return ""
}

func ToType(s string) Type {
	t := Type(strings.ToLower(s))
	if pie.Contains(types(), t) {
		return t
	}
	return NoType
}
