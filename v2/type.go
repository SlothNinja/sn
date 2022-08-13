package sn

import "encoding/json"

type Type int

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

// String returns string name of game type constant
func (t Type) String() string {
	ss := map[Type]string{
		Confucius:  "Confucius",
		Tammany:    "Tammany Hall",
		ATF:        "After The Flood",
		GOT:        "Guild Of Thieves",
		Indonesia:  "Indonesia",
		Gettysburg: "Gettysburg",
		All:        "All",
	}
	s, ok := ss[t]
	if ok {
		return s
	}
	return ""
}

// SString returns short string name of game type constant
func (t Type) SString() string {
	ss := map[Type]string{
		Confucius:  "Confucius",
		Tammany:    "Tammany",
		ATF:        "ATF",
		GOT:        "GOT",
		Indonesia:  "Indonesia",
		Gettysburg: "Gettysburg",
		All:        "All",
	}
	s, ok := ss[t]
	if ok {
		return s
	}
	return ""
}

// MarshalJSON implements json.Marshaler interface to provide custom json marshalling.
func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.SString())
}

// UnmarshalJSON implements json.Unmarshaler interface to provide custom json unmarshalling
func (t *Type) UnmarshalJSON(b []byte) error {
	var s string

	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*t = ToType(s)
	return nil
}

func ToType(s string) Type {
	ss := map[string]Type{
		"confucius":  Confucius,
		"tammany":    Tammany,
		"atf":        ATF,
		"got":        GOT,
		"indonesia":  Indonesia,
		"gettysburg": Gettysburg,
		"all":        All,
	}
	v, ok := ss[s]
	if ok {
		return v
	}
	return NoType
}
