package sn

import (
	"context"
	"fmt"
	"log"
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

type Client[G Game[G], I Invitation[G, I]] struct {
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

func NewClient[G Game[G], I Invitation[G, I]](ctx context.Context, opt Options) Client[G, I] {
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
		cl := Client[G, I]{
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
	cl := Client[G, I]{
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
func (cl Client[G, I]) addRoutes(prefix string) Client[G, I] {
	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", cl.NewInvitationHandler())

	// // Create
	iGroup.PUT("/new", cl.CreateInvitationHandler())

	// // Drop
	iGroup.PUT("/drop/:id", cl.DropHandler())

	// // Accept
	// // inv.PUT("/accept/:id", cl.acceptHandler)
	iGroup.PUT("/accept/:id", cl.AcceptHandler())

	// // Details
	iGroup.GET("/details/:id", cl.DetailsHandler())

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

	// login
	cl.Router.GET(prefix+"/login", cl.LoginHandler())
	//
	// logout
	cl.Router.GET(prefix+"/logout", cl.LogoutHandler())

	// // current user
	cl.Router.GET(prefix+"/cu", cl.CuHandler())
	// // 	// Message Log
	// // 	msg := cl.Router.Group("mlog")
	// //
	// // 	// Get
	// // 	msg.GET("/:id", cl.mlogHandler)
	// //
	// // 	// Add
	// // 	msg.PUT("/:id/add", cl.mlogAddHandler)
	// //
	return cl
}

func (cl Client[G, I]) Close() error {
	cl.FS.Close()
	return cl.User.Close()
}
