package sn

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddRoutes adds routing for game.
func (cl *Client) addRoutes() *Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/current", cl.cuHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	return cl
}
