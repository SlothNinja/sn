package sn

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID UID
	Data
}

type Data struct {
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

func (cl Client) cuHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": nil})
			return
		}

		cl.Log.Debugf("cu: %#v", cu)
		tokenKey := getFSTokenKey()
		if tokenKey == "" {
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}

		token, err := getFBToken(ctx, cu.ID)
		if err != nil {
			cl.Log.Warningf(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"CU": cu})
			return
		}
		cl.Log.Debugf("token: %#v", token)
		ctx.JSON(http.StatusOK, gin.H{"CU": cu, tokenKey: token})
	}
}

const FSTokenKey = "FS_TOKEN_KEY"

func getFSTokenKey() string {
	return os.Getenv(FSTokenKey)
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
