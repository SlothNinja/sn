package sn

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/firestore"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var myRandomSource = rand.NewSource(time.Now().UnixNano())

// type Client[G Gamer[G], I Invitation[I], P Playerer] struct {
type Client[G Gamer[G], P Playerer] struct {
	FS     *firestore.Client
	User   *datastore.Client
	Log    *Logger
	Cache  *cache.Cache
	Router *gin.Engine
}

type Options struct {
	ProjectID     string
	UserProjectID string
	UserDSURL     string
	LoggerID      string
	CorsAllow     []string
	Prefix        string
}

func NewClient[G Gamer[G], P Playerer](ctx context.Context, opt Options) Client[G, P] {
	lClient, err := NewLogClient(opt.ProjectID)
	if err != nil {
		log.Panicf("unable to create logging client: %v", err)
	}
	log := lClient.Logger(opt.LoggerID)
	if IsProduction() {
		log.Debugf("production")
		dsClient, err := datastore.NewClient(ctx, opt.UserProjectID)

		if err != nil {
			panic(fmt.Errorf("unable to connect to user database: %w", err))
		}
		fsClient, err := firestore.NewClient(ctx, opt.ProjectID)
		if err != nil {
			panic(fmt.Errorf("unable to connect to firestore database: %w", err))
		}
		cl := Client[G, P]{
			User:   dsClient,
			FS:     fsClient,
			Log:    log,
			Cache:  cache.New(30*time.Minute, 10*time.Minute),
			Router: gin.Default(),
		}
		cl.NewStore(ctx)
		return cl.addRoutes(opt.Prefix)
	}
	log.Debugf("development")
	dsClient, err := datastore.NewClient(
		ctx,
		opt.UserProjectID,
		option.WithEndpoint(opt.UserDSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCConnectionPool(50),
	)
	if err != nil {
		panic(fmt.Errorf("unable to connect to user database: %w", err))
	}
	fsClient, err := firestore.NewClient(ctx, opt.ProjectID)
	if err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}
	cl := Client[G, P]{
		User:   dsClient,
		FS:     fsClient,
		Log:    log,
		Cache:  cache.New(30*time.Minute, 10*time.Minute),
		Router: gin.Default(),
	}
	cl.NewStore(ctx)
	config := cors.DefaultConfig()
	config.AllowOrigins = opt.CorsAllow
	config.AllowCredentials = true
	config.AllowWildcard = true
	cl.Router.Use(cors.New(config))
	return cl.addRoutes(opt.Prefix)
}

// AddRoutes addes routing for game.
func (cl Client[G, P]) addRoutes(prefix string) Client[G, P] {
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
	// // inv.PUT("/accept/:id", cl.acceptHandler)
	iGroup.PUT("/accept/:id", cl.acceptHandler())

	// // Details
	iGroup.GET("/details/:id", cl.detailsHandler())

	/////////////////////////////////////////////
	// Game Group
	gGroup := cl.Router.Group(prefix + "/game")

	// Reset
	gGroup.PUT("reset/:id", cl.ResetHandler())

	// Undo
	gGroup.PUT("undo/:id", cl.UndoHandler())

	// Redo
	gGroup.PUT("redo/:id", cl.RedoHandler())

	// Rollback
	gGroup.PUT("rollback/:id", cl.RollbackHandler())

	// Rollforward
	gGroup.PUT("rollforward/:id", cl.RollforwardHandler())

	/////////////////////////////////////////////
	// Login/Logout
	// login
	cl.Router.GET(prefix+"/login", cl.LoginHandler())
	//
	// logout
	cl.Router.GET(prefix+"/logout", cl.LogoutHandler())

	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(prefix+"/cu", cl.CuHandler())

	/////////////////////////////////////////////
	// Message Log
	msg := cl.Router.Group(prefix + "/mlog")

	// Update Read
	msg.GET("/update/:id", cl.updateReadHandler())

	// Add
	msg.PUT("/add/:id", cl.addMessageHandler())

	// return cl.staticRoutes(prefix)
	return cl
}

// func (cl Client[G, I]) staticRoutes(prefix string) Client[G, I] {
// 	if IsProduction() {
// 		return cl
// 	}
// 	// cl.Router.StaticFile("/", "dist/index.html")
// 	// cl.Router.StaticFile("/index.html", "dist/index.html")
// 	// cl.Router.StaticFile("/firebase-messaging-sw.js", "dist/firebase-messaging-sw.js")
// 	// cl.Router.StaticFile("/manifest.json", "dist/manifest.json")
// 	// cl.Router.StaticFile("/robots.txt", "dist/robots.txt")
// 	// cl.Router.StaticFile("/precache-manifest.c0be88927a8120cb7373cf7df05f5688.js", "dist/precache-manifest.c0be88927a8120cb7373cf7df05f5688.js")
// 	// cl.Router.StaticFile("/app.js", "dist/app.js")
// 	// cl.Router.StaticFile("/favicon.ico", "dist/favicon.ico")
// 	// cl.Router.Static("/img", "dist/img")
// 	// cl.Router.Static("/js", "dist/js")
// 	// cl.Router.Static("/css", "dist/css")
// 	return cl
// }

func (cl Client[G, P]) Close() error {
	cl.FS.Close()
	return cl.User.Close()
}
