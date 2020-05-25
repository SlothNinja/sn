package sn

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

type Color int

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

var toColor = map[string]Color{"red": Red, "yellow": Yellow, "purple": Purple, "black": Black, "brown": Brown, "white": White, "green": Green, "orange": Orange, "none": None}

func (c Color) String() string {
	return colorString[c]
}

func (c Color) int() int {
	return int(c)
}

var textColor = map[Color]Color{Red: White, Yellow: Black, Purple: White, Black: White, Brown: White}

func TextColorFor(background Color) Color {
	return textColor[background]
}

type Map map[int]Color

func MapFrom(c *gin.Context) (cm Map) {
	cm, _ = c.Value(cmKey).(Map)
	return
}

func WithMap(c *gin.Context, cm Map) *gin.Context {
	c.Set(cmKey, cm)
	return c
}

func (c Color) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *Color) UnmarshalJSON(bs []byte) error {
	var s string
	err := json.Unmarshal(bs, &s)
	if err != nil {
		return err
	}

	c2, ok := toColor[s]
	if !ok {
		return fmt.Errorf("unable to unmarshal %s", s)
	}
	*c = c2
	return nil
}
