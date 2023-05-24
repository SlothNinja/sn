package sn

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/firestore"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	FS     *firestore.Client
	User   *datastore.Client
	Log    *Logger
	Cache  *cache.Cache
	Router *gin.Engine
}

type Options[G Game, I Invitation] struct {
	ProjectID     string
	UserProjectID string
	UserDSURL     string
	Logger        *Logger
	Cache         *cache.Cache
	Router        *gin.Engine
	CorsAllow     []string
	Prefix        string
	Game          G
	Invitation    I
}

func NewClient[G Game, I Invitation](ctx context.Context, opt Options[G, I]) Client {
	opt.Logger.Debugf(msgEnter)
	defer opt.Logger.Debugf(msgEnter)

	if IsProduction() {
		opt.Logger.Debugf("production")
		dsClient, err := datastore.NewClient(ctx, opt.UserProjectID)

		if err != nil {
			panic(fmt.Errorf("unable to connect to user database: %w", err))
		}
		fsClient, err := firestore.NewClient(ctx, opt.ProjectID)
		if err != nil {
			panic(fmt.Errorf("unable to connect to firestore database: %w", err))
		}
		client := Client{
			User:   dsClient,
			FS:     fsClient,
			Log:    opt.Logger,
			Cache:  opt.Cache,
			Router: opt.Router,
		}
		client.NewStore(ctx)
		return client
	}
	opt.Logger.Debugf("development")
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
	cl := Client{
		User:   dsClient,
		FS:     fsClient,
		Log:    opt.Logger,
		Cache:  opt.Cache,
		Router: opt.Router,
	}
	cl.NewStore(ctx)
	config := cors.DefaultConfig()
	config.AllowOrigins = opt.CorsAllow
	config.AllowCredentials = true
	config.AllowWildcard = true
	cl.Router.Use(cors.New(config))
	return addRoutes(cl, opt.Prefix, opt.Game, opt.Invitation)
}

func (cl *Client) Close() error {
	cl.FS.Close()
	return cl.User.Close()
}

// AddRoutes addes routing for game.
func addRoutes[G Game, I Invitation](cl Client, prefix string, g G, inv I) Client {
	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", NewInvitationHandler(cl, inv))

	// Create
	iGroup.PUT("/new", CreateInvitationHandler(cl, inv))

	// Drop
	iGroup.PUT("/drop/:id", DropHandler(cl, inv))

	// Accept
	// inv.PUT("/accept/:id", cl.acceptHandler)
	iGroup.PUT("/accept/:id", AcceptHandler(cl, inv))

	// Details
	iGroup.GET("/details/:id", DetailsHandler(cl, inv))

	/////////////////////////////////////////////
	// Game Group
	gGroup := cl.Router.Group(prefix + "/game")

	// Reset
	gGroup.PUT("reset/:id", ResetHandler(cl, g))

	// Undo
	gGroup.PUT("undo/:id", UndoHandler(cl, g))

	// Redo
	gGroup.PUT("redo/:id", RedoHandler(cl, g))

	// Rollback
	gGroup.PUT("rollback/:id", RollbackHandler(cl, g))

	// Rollforward
	gGroup.PUT("rollforward/:id", RollforwardHandler(cl, g))

	// login
	cl.Router.GET(prefix+"/login", LoginHandler(cl))
	//
	// logout
	cl.Router.GET(prefix+"/logout", LogoutHandler(cl))

	// current user
	cl.Router.GET(prefix+"/cu", CuHandler(cl))
	// 	// Message Log
	// 	msg := cl.Router.Group("mlog")
	//
	// 	// Get
	// 	msg.GET("/:id", cl.mlogHandler)
	//
	// 	// Add
	// 	msg.PUT("/:id/add", cl.mlogAddHandler)
	//
	return cl
}
