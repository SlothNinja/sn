package sn

import (
	"fmt"
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Message represents a chat message
type Message struct {
	Text             string
	CreatorID        UID
	CreatorName      string
	CreatorEmailHash string
	CreatorGravType  string
	Read             []UID
	CreatedAt        *timestamppb.Timestamp
	UpdatedAt        *timestamppb.Timestamp
}

func newMessageFor(m *message, user *User) Message {
	t := timestamppb.Now()
	return Message{
		Text:             m.Text,
		CreatorID:        user.ID,
		CreatorName:      user.Name,
		CreatorEmailHash: user.EmailHash,
		CreatorGravType:  user.GravType,
		Read:             []UID{user.ID},
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

type message struct {
	CreatorID UID
	Text      string
}

func getMessage(ctx *gin.Context) (*message, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	obj := new(message)
	if err := ctx.ShouldBind(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (cl *GameClient[GT, G]) addMessageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		m, err := cl.validateAddMessage(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.addMessage(ctx, newMessageFor(m, cu)); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) validateAddMessage(ctx *gin.Context, cu *User) (*message, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	m, err := getMessage(ctx)
	if err != nil {
		return nil, err
	}

	if m.CreatorID != cu.ID {
		return nil, fmt.Errorf("invalid creator: %w", ErrValidation)
	}
	return m, nil
}

func (cl *GameClient[GT, G]) addMessage(ctx *gin.Context, m Message) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.messagesCollectionRef(getID(ctx)).NewDoc().Create(ctx, m)
	return err
}
