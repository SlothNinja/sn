package sn

type Color string

const (
	Red     Color = "red"
	Yellow  Color = "yellow"
	Purple  Color = "purple"
	Black   Color = "black"
	Brown   Color = "brown"
	White   Color = "white"
	Green   Color = "green"
	Orange  Color = "orange"
	NoColor Color = ""
)

var textColor = map[Color]Color{Red: White, Yellow: Black, Purple: White, Black: White, Brown: White}

func TextColorFor(background Color) Color {
	return textColor[background]
}

type ColorMap map[int]Color
