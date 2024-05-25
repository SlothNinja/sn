// SN implements services for SlothNinja Games Website
package sn

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

var myRandomSource = rand.NewSource(time.Now().UnixNano())

type Client struct {
	Log    *Logger
	Cache  *cache.Cache
	Router *gin.Engine
	options
}

type GameClient[GT any, G Gamer[GT]] struct {
	*Client
	FS *firestore.Client
}

func defaultClient() *Client {
	cl := new(Client)
	cl.projectID = getProjectID()
	cl.url = getURL()
	cl.frontEndURL = getFrontEndURL()
	cl.backEndURL = getBackEndURL()
	cl.port = getPort()
	cl.backEndPort = getBackEndPort()
	cl.frontEndPort = getFrontEndPort()
	cl.secretsProjectID = getSecretsProjectID()
	cl.secretsDSURL = getSecretsDSURL()
	cl.loggerID = getLoggerID()
	cl.corsAllow = getCORSAllow()
	cl.prefix = getPrefix()
	cl.home = getHome()
	return cl
}

func NewClient(ctx context.Context, opts ...Option) *Client {
	cl := defaultClient()

	// Apply all functional options
	for _, opt := range opts {
		cl = opt(cl)
	}

	// Initalize
	return cl.initLogger().
		initCache().
		initRouter().
		initSession(ctx).
		initEnvironment().
		addRoutes()
}

func (cl *Client) initLogger() *Client {
	lClient, err := NewLogClient(cl.projectID)
	if err != nil {
		log.Panicf("unable to create logging client: %v", err)
	}
	cl.Log = lClient.Logger(cl.loggerID)
	return cl
}

func (cl *Client) initCache() *Client {
	cl.Cache = cache.New(30*time.Minute, 10*time.Minute)
	return cl
}

func (cl *Client) initRouter() *Client {
	cl.Router = gin.Default()
	return cl
}

func (cl *Client) initEnvironment() *Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if IsProduction() {
		cl.Router.TrustedPlatform = gin.PlatformGoogleAppEngine
		return cl
	}

	// Is development
	config := cors.DefaultConfig()
	config.AllowOrigins = cl.corsAllow
	config.AllowCredentials = true
	config.AllowWildcard = true
	cl.Router.Use(cors.New(config))
	cl.Router.SetTrustedProxies(nil)
	return cl
}

func NewGameClient[GT any, G Gamer[GT]](ctx context.Context, opts ...Option) *GameClient[GT, G] {
	cl := &GameClient[GT, G]{Client: NewClient(ctx, opts...)}

	var err error
	if cl.FS, err = firestore.NewClient(ctx, cl.projectID); err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}
	return cl.addRoutes(cl.prefix)
}

// AddRoutes addes routing for game.
func (cl *Client) addRoutes() *Client {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/current", cl.cuHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	return cl
}

// AddRoutes addes routing for game.
func (cl *GameClient[GT, G]) addRoutes(prefix string) *GameClient[GT, G] {
	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", cl.newInvitationHandler())

	// // Create
	iGroup.PUT("/new", cl.createInvitationHandler())

	// // Drop
	iGroup.PUT("/drop/:id", cl.dropHandler())

	// // Accept
	iGroup.PUT("/accept/:id", cl.acceptHandler())

	// // Details
	iGroup.GET("/details/:id", cl.detailsHandler())

	// Abort
	iGroup.PUT("abort/:id", cl.abortHandler())

	/////////////////////////////////////////////
	// Game Group
	gGroup := cl.Router.Group(prefix + "/game")

	// Reset
	gGroup.PUT("reset/:id", cl.resetHandler())

	// Undo
	gGroup.PUT("undo/:id", cl.undoHandler())

	// Redo
	gGroup.PUT("redo/:id", cl.redoHandler())

	// Rollback
	gGroup.PUT("rollback/:id", cl.rollbackHandler())

	// Rollforward
	gGroup.PUT("rollforward/:id", cl.rollforwardHandler())

	// Abandon
	gGroup.PUT("abandon/:id", cl.abandonHandler())

	/////////////////////////////////////////////
	// Message Log
	msg := cl.Router.Group(prefix + "/mlog")

	// Update Read
	msg.PUT("/updateRead/:id", cl.updateReadHandler())

	// Add
	msg.PUT("/add/:id", cl.addMessageHandler())

	return cl
}

func (cl *GameClient[GT, G]) Close() error {
	cl.FS.Close()
	return cl.Client.Close()
}

func (cl *Client) Close() error {
	return nil
}
