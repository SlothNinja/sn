package sn

import (
	"html/template"
	"time"

	"github.com/SlothNinja/color"
	"github.com/gin-gonic/gin"
)

type Message struct {
	Text      string
	CreatorID int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ml *MLog) NewMessage(c *gin.Context) *Message {
	t := time.Now()
	m := &Message{
		CreatedAt: t,
		UpdatedAt: t,
	}
	ml.Messages = append(ml.Messages, m)
	return m
}

type Messages []*Message

func (m *Message) Color(cm color.Map) template.HTML {
	if c, ok := cm[int(m.CreatorID)]; ok {
		return template.HTML(c.String())
	}
	return template.HTML("default")
}

func (m *Message) Message() template.HTML {
	return template.HTML(template.HTMLEscapeString(m.Text))
}