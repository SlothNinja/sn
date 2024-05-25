package sn

// Color provides predefined colors
type Color string

const (
	// Red color constant
	Red Color = "red"

	// Yellow color constant
	Yellow Color = "yellow"

	// Purple color constant
	Purple Color = "purple"

	// Black color constant
	Black Color = "black"

	// Brown color constant
	Brown Color = "brown"

	// White color constant
	White Color = "white"

	// Green color constant
	Green Color = "green"

	// Orange color constant
	Orange Color = "orange"

	// NoColor represents no, missing, or zero value of color
	NoColor Color = ""
)

var textColor = map[Color]Color{Red: White, Yellow: Black, Purple: White, Black: White, Brown: White}

// TextColorFor returns Black or White foreground text color for provided background color
func TextColorFor(background Color) Color {
	return textColor[background]
}
