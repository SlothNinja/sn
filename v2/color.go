package sn

import (
	"github.com/gin-gonic/gin"
)

type Color int
type Colors []Color

func (this *Colors) Append(colors ...Color) {
	*this = append(*this, colors...)
}

func (this *Colors) Slice(low, high int) {
	*this = (*this)[int(low):int(high)]
}

const (
	Red Color = iota
	Yellow
	Purple
	Black
	Brown
	White
	Green
	Orange

	None Color = -1
)

const (
	cmKey = "CMAP"
)

var colorString = map[Color]string{Red: "red", Yellow: "yellow", Purple: "purple", Black: "black", Brown: "brown", White: "white", Green: "green", Orange: "orange", None: "none"}

func (this Color) String() string {
	return colorString[this]
}

func (cs Colors) Strings() []string {
	var s []string
	for _, c := range cs {
		s = append(s, c.String())
	}
	return s
}

func (this Color) int() int {
	return int(this)
}

var textColor = map[Color]Color{Red: White, Yellow: Black, Purple: White, Black: White, Brown: White}

func TextColorFor(background Color) Color {
	return textColor[background]
}

type ColorMap map[int]Color

func ColorMapFrom(c *gin.Context) (cm ColorMap) {
	cm, _ = c.Value(cmKey).(ColorMap)
	return
}

func WithColorMap(c *gin.Context, cm ColorMap) *gin.Context {
	c.Set(cmKey, cm)
	return c
}
