package sn

import (
	"log/slog"
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
	Joined             time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (cl *Client) cuHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			slog.Warn(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": nil})
			return
		}

		tokenKey := getFSTokenKey()
		if tokenKey == "" {
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}

		token, err := getFBToken(ctx, cu.ID)
		if err != nil {
			slog.Warn(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"CU": cu, tokenKey: token})
	}
}

const fsTokenKey = "FS_TOKEN_KEY"

func getFSTokenKey() string {
	return os.Getenv(fsTokenKey)
}

type UID int64

const noUID UID = 0

const (
	uidParam        = "uid"
	guserKey        = "guser"
	currentKey      = "current"
	userKey         = "User"
	salt            = "slothninja"
	usersKey        = "Users"
	USER_PROJECT_ID = "USER_PROJECT_ID"
	DS_USER_HOST    = "DS_USER_HOST"
)

func getUID(ctx *gin.Context, param string) (UID, error) {
	id, err := strconv.ParseInt(ctx.Param(param), 10, 64)
	return UID(id), err
}

// func (u User) MarshalJSON() ([]byte, error) {
// 	type usr User
// 	return json.Marshal(struct {
// 		usr
// 		ID UID
// 	}{
// 		usr: usr(u),
// 		ID:  u.ID,
// 	})
// }
