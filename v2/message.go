package sn

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Message struct {
	Text      string    `json:"text"`
	Creator   User      `json:"creator"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

// type Messages []*Message
//
// func (m *Message) Color(cm color.Map) template.HTML {
// 	if c, ok := cm[int(m.CreatorID)]; ok {
// 		return template.HTML(c.String())
// 	}
// 	return template.HTML("default")
// }
//
// func (m *Message) Message() template.HTML {
// 	return template.HTML(template.HTMLEscapeString(m.Text))
// }
