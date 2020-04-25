package sn

import (
	"strings"

	"github.com/SlothNinja/user/v2"
	"github.com/gin-gonic/gin"
)

func WithType(c *gin.Context, t Type) *gin.Context {
	c.Set(typeKey, t)
	return c
}

func TypeFrom(c *gin.Context) (t Type) {
	t, _ = c.Value(typeKey).(Type)
	return
}

type Type int
type GTypes []Type

// Do not alphabetize or otherwise reorder the following
// Existing games in datastore rely upon the currently assigned values
const (
	NoType Type = iota

	Confucius
	Tammany
	ATF
	GOT
	Indonesia
	Gettysburg

	All Type = 10000
)

func (t Type) String() string   { return TypeStrings[t] }
func (t Type) SString() string  { return TypeSStrings[t] }
func (t Type) IDString() string { return strings.ToLower(t.SString()) }
func (t Type) Prefix() string   { return t.IDString() }

var Types = GTypes{
	ATF,
	Confucius,
	Gettysburg,
	GOT,
	Indonesia,
	Tammany,
}

type TypeMap map[Type]string

var TypeStrings = TypeMap{
	Confucius:  "Confucius",
	Tammany:    "Tammany Hall",
	ATF:        "After The Flood",
	GOT:        "Guild Of Thieves",
	Indonesia:  "Indonesia",
	Gettysburg: "Gettysburg",
	All:        "All",
}

var TypeSStrings = TypeMap{
	Confucius:  "Confucius",
	Tammany:    "Tammany",
	ATF:        "ATF",
	GOT:        "GOT",
	Indonesia:  "Indonesia",
	Gettysburg: "Gettysburg",
	All:        "All",
}

var ToType = map[string]Type{
	"confucius":  Confucius,
	"tammany":    Tammany,
	"atf":        ATF,
	"got":        GOT,
	"indonesia":  Indonesia,
	"gettysburg": Gettysburg,
	"all":        All,
}

var multiUndo = map[Type]bool{
	Confucius:  false,
	Tammany:    true,
	ATF:        false,
	GOT:        false,
	Indonesia:  false,
	Gettysburg: true,
}

func (t Type) MultiUndo() bool {
	if b, ok := multiUndo[t]; ok {
		return b
	}
	return false
}

var released = map[Type]bool{
	Confucius:  true,
	Tammany:    true,
	ATF:        true,
	GOT:        true,
	Indonesia:  true,
	Gettysburg: false,
}

func rtypes() GTypes {
	var gts GTypes
	for _, t := range Types {
		if released[t] {
			gts = append(gts, t)
		}
	}
	return gts
}

func SetTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch _, ok := c.Get("cuser"); {
		case !ok, !user.IsAdmin(c):
			c.Set("types", rtypes())
		default:
			c.Set("types", Types)
		}
	}
}

func GetType(c *gin.Context) Type {
	ltype := strings.ToLower(c.Param("type"))
	if t, ok := ToType[ltype]; ok {
		return t
	}
	return NoType
}
