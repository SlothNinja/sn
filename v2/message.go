package sn

import (
	"html/template"
	"time"

	"github.com/SlothNinja/user"
)

type Message struct {
	Text             string    `json:"text"`
	CreatorID        int64     `json:"creatorId"`
	CreatorName      string    `json:"creatorName"`
	CreatorEmailHash string    `json:"creatorEmailHash"`
	CreatorGravType  string    `json:"creatorGravType"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func NewMessage(u *user.User, text string) *Message {
	t := time.Now()
	return &Message{
		Text:             text,
		CreatorID:        u.ID(),
		CreatorName:      u.Name,
		CreatorEmailHash: u.EmailHash,
		CreatorGravType:  u.GravType,
		CreatedAt:        t,
		UpdatedAt:        t,
	}
}

func (m *Message) Color(cm ColorMap) template.HTML {
	c, ok := cm[int(m.CreatorID)]
	if !ok {
		return template.HTML("default")
	}
	return template.HTML(c.String())
}

func (m *Message) Message() template.HTML {
	return template.HTML(template.HTMLEscapeString(m.Text))
}
