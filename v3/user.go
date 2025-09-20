package sn

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// User represents a user
type User struct {
	ID UID `datastore:"-"`
	userData
}

type userData struct {
	Name               string
	LCName             string
	Email              string
	EmailHash          string
	EmailNotifications bool
	EmailReminders     bool
	GoogleID           string
	XMPPNotifications  bool
	GravType           string
	Admin              bool
	GodMode            bool `datastore:"-"`
	Joined             time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (cl *Client) cuHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"CU": nil, "Error": err.Error()})
			return
		}

		tokenKey := getFSTokenKey()
		if tokenKey == "" {
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}

		token, err := getFBToken(ctx, cu.ID)
		if err != nil {
			Warnf("%v", err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"CU": cu, tokenKey: token})
	}
}

func (cl *Client) updateGodModeHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

		token := cl.GetSessionToken(ctx)
		if token == nil {
			ctx.JSON(http.StatusOK, gin.H{"Error": ErrNotLoggedIn})
			return
		}

		if !token.Data.Admin {
			ctx.JSON(http.StatusOK, gin.H{"Error": ErrNotAdmin})
			return
		}

		obj := new(struct {
			GodMode bool
		})

		if err := ctx.ShouldBind(obj); err != nil {
			ctx.JSON(http.StatusOK, gin.H{"Error": err})
			return
		}

		token.Data.GodMode = obj.GodMode
		cl.setSessionToken(ctx, token)

		if err := cl.SaveSession(ctx); err != nil {
			ctx.JSON(http.StatusOK, gin.H{"Error": err})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"CU": token.ToUser()})
	}
}

func getFSTokenKey() string {
	return os.Getenv("FS_TOKEN_KEY")
}

// UID represent a unique id of a user
type UID int64

const noUID UID = 0

func (uid UID) toString() string {
	return strconv.Itoa(int(uid))
}
