package sn

// Type represents a type of game.
type Type string

const (
	// NoType is the zero value and represents a missing game type
	NoType Type = ""

	// ATF specifies an "After the Flood" game
	ATF Type = "atf"

	// Confucius specifies a "Confucius" game
	Confucius Type = "confucius"

	// GOT specifies a "Guide of Thieves" game
	GOT Type = "got"

	// Indonesia specifies an "Indonesia" game
	Indonesia Type = "indonesia"

	// Plateau specifies a "Le Plateau" game
	Plateau Type = "plateau"

	// Tammany specifies a "Tammany Hall" game
	Tammany Type = "tammany"

	// All species all game times
	All Type = "all"
)

func types() []Type {
	return []Type{NoType, ATF, Confucius, GOT, Indonesia, Plateau, Tammany, All}
}

func (t Type) String() string {
	s, ok := map[Type]string{
		Confucius: "Confucius",
		Tammany:   "Tammany Hall",
		ATF:       "After The Flood",
		GOT:       "Guild Of Thieves",
		Indonesia: "Indonesia",
		Plateau:   "Le Plateau",
		All:       "All",
	}[t]
	if ok {
		return s
	}
	return ""
}
