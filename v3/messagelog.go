package sn

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

// Message represents a chat message
type Message struct {
	Text             string
	CreatorID        UID
	CreatorName      string
	CreatorEmailHash string
	CreatorGravType  string
	Read             []UID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func newMessage(u *User, text string) Message {
	t := time.Now()
	return Message{
		Text:             text,
		CreatorID:        u.ID,
		CreatorName:      u.Name,
		CreatorEmailHash: u.EmailHash,
		CreatorGravType:  u.GravType,
		Read:             []UID{u.ID},
		CreatedAt:        t,
		UpdatedAt:        t,
	}
}

func (cl *GameClient[GT, G]) updateReadHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		read, err := getRead(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.updateRead(ctx, cu.ID, read); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) updateRead(ctx *gin.Context, uid UID, read []string) error {
	for _, mid := range read {
		if _, err := cl.messageDocRef(getID(ctx), mid).Update(ctx, []firestore.Update{
			{Path: "Read", Value: firestore.ArrayUnion(uid)},
		}); err != nil {
			return err
		}
	}
	return nil
}

func getRead(ctx *gin.Context) ([]string, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	var obj struct {
		Read []string
	}
	err := ctx.ShouldBind(&obj)
	return obj.Read, err
}

func getMessage(ctx *gin.Context) (Message, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	var obj struct {
		Text    string
		Creator *User
		Read    []string
	}

	if err := ctx.ShouldBind(&obj); err != nil {
		return Message{}, err
	}
	return newMessage(obj.Creator, obj.Text), nil
}

func (cl *GameClient[GT, G]) addMessageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		m, err := cl.validateAddMessage(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.addMessage(ctx, m); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) validateAddMessage(ctx *gin.Context) (Message, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	cu, err := cl.RequireLogin(ctx)
	if err != nil {
		return Message{}, err
	}

	m, err := getMessage(ctx)
	if err != nil {
		return Message{}, err
	}

	if m.CreatorID != cu.ID {
		return Message{}, fmt.Errorf("invalid creator: %w", ErrValidation)
	}
	return m, nil
}

func (cl *GameClient[GT, G]) addMessage(ctx *gin.Context, m Message) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.messagesCollectionRef(getID(ctx)).NewDoc().Create(ctx, m)
	return err
}
