package sn

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
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

type Client struct {
	User   *datastore.Client
	Log    *Logger
	Cache  *cache.Cache
	Router *gin.Engine
	options
}

type UserServiceClient struct {
	Client
}

type GameClient[G Gamer[G], P Playerer] struct {
	UserServiceClient
	FS *firestore.Client
}

type options struct {
	projectID     string
	userProjectID string
	userDSURL     string
	loggerID      string
	corsAllow     []string
	prefix        string
}

func WithProjectID(id string) Option {
	return func(cl Client) Client {
		cl.projectID = id
		return cl
	}
}

func getProjectID() string {
	return os.Getenv("PROJECT_ID")
}

func WithUserProjectID(id string) Option {
	return func(cl Client) Client {
		cl.userProjectID = id
		return cl
	}
}

func getUserProjectID() string {
	if id, found := os.LookupEnv("USER_PROJECT_ID"); found {
		return id
	}
	return "user-slothninja-games"
}

func WithUserDSURL(url string) Option {
	return func(cl Client) Client {
		cl.userDSURL = url
		return cl
	}
}

func getUserDSURL() string {
	if url, found := os.LookupEnv("USER_DS_URL"); found {
		return url
	}
	if IsProduction() {
		return "https://user.slothninja.com/"
	}
	return "https://user.fake-slothninja.com:8086/"
}

func WithLoggerID(id string) Option {
	return func(cl Client) Client {
		cl.loggerID = id
		return cl
	}
}

func getLoggerID() string {
	if id, found := os.LookupEnv("LOGGER_ID"); found {
		return id
	}
	return getProjectID()
}

func WithCORSAllow(paths ...string) Option {
	return func(cl Client) Client {
		cl.corsAllow = paths
		return cl
	}
}

func getCORSAllow() []string {
	cors, found := os.LookupEnv("CORS_ALLOW")
	if !found {
		return nil
	}
	return strings.Split(cors, ",")
}

func WithPrefix(prefix string) Option {
	return func(cl Client) Client {
		cl.prefix = prefix
		return cl
	}
}

func getPrefix() string {
	if prefix, found := os.LookupEnv("PREFIX"); found {
		return prefix
	}
	return "sn"
}

type Option func(Client) Client

func NewUserServiceClient(ctx context.Context, opt ...Option) UserServiceClient {
	cl := UserServiceClient{NewClient(ctx, opt...)}
	return cl.addRoutes(cl.prefix)
}

func defaultClient() Client {
	var cl Client
	cl.projectID = getProjectID()
	cl.userProjectID = getUserProjectID()
	cl.userDSURL = getUserDSURL()
	cl.loggerID = getLoggerID()
	cl.corsAllow = getCORSAllow()
	cl.prefix = getPrefix()
	return cl
}

func NewClient(ctx context.Context, opts ...Option) Client {
	cl := defaultClient()

	// Apply all functional options
	for _, opt := range opts {
		cl = opt(cl)
	}

	// Initalize Logger
	lClient, err := NewLogClient(cl.projectID)
	if err != nil {
		log.Panicf("unable to create logging client: %v", err)
	}
	cl.Log = lClient.Logger(cl.loggerID)

	// Initialize user datastore
	cl = cl.initUserDatastore(ctx)

	// Initialize cached
	cl.Cache = cache.New(30*time.Minute, 10*time.Minute)

	// Initialize router
	cl.Router = gin.Default()

	// Initialize cookie store
	cl.NewStore(ctx)

	if IsProduction() {
		return cl.production(ctx).addRoutes(cl.prefix)
	}
	return cl.development(ctx).addRoutes(cl.prefix)
}

func (cl Client) production(ctx context.Context, opts ...Option) Client {
	Debugf("production")
	gin.SetMode(gin.ReleaseMode)
	cl.Router.TrustedPlatform = gin.PlatformGoogleAppEngine
	return cl
}

func (cl Client) initUserDatastore(ctx context.Context) Client {
	if IsProduction() {
		return cl.productionUserDatastore(ctx)
	}
	return cl.developmentUserDataStore(ctx)
}

func (cl Client) productionUserDatastore(ctx context.Context) Client {
	dsClient, err := datastore.NewClient(ctx, cl.userProjectID)
	if err != nil {
		panic(fmt.Errorf("unable to connect to user database: %w", err))
	}
	cl.User = dsClient
	return cl
}

func (cl Client) developmentUserDataStore(ctx context.Context) Client {
	dsClient, err := datastore.NewClient(
		ctx,
		cl.userProjectID,
		option.WithEndpoint(cl.userDSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCConnectionPool(50),
	)
	if err != nil {
		panic(fmt.Errorf("unable to connect to user database: %w", err))
	}
	cl.User = dsClient
	return cl
}

func (cl Client) development(ctx context.Context) Client {
	Debugf("development")
	gin.SetMode(gin.DebugMode)

	config := cors.DefaultConfig()
	config.AllowOrigins = cl.corsAllow
	config.AllowCredentials = true
	config.AllowWildcard = true
	cl.Router.Use(cors.New(config))
	cl.Router.SetTrustedProxies(nil)
	return cl
}

func NewGameClient[G Gamer[G], P Playerer](ctx context.Context, opts ...Option) GameClient[G, P] {
	cl := GameClient[G, P]{UserServiceClient: NewUserServiceClient(ctx, opts...)}

	var err error
	if cl.FS, err = firestore.NewClient(ctx, cl.projectID); err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}
	return cl.addRoutes(cl.prefix)
}

// AddRoutes addes routing for game.
func (cl Client) addRoutes(prefix string) Client {
	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(prefix+"/user/current", cl.cuHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	return cl
}

// AddRoutes addes routing for game.
func (cl UserServiceClient) addRoutes(prefix string) UserServiceClient {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	// New
	cl.Router.GET(prefix+"/user/new", cl.NewUserHandler)

	// Create
	cl.Router.PUT(prefix+"/user/new", cl.CreateUserHandler)

	// Update
	cl.Router.PUT(prefix+"/user/:uid/update", cl.UpdateUser("uid"))

	// Get
	cl.Router.GET(prefix+"/user/:uid/json", cl.userJSONHandler("uid"))

	cl.Router.GET(prefix+"/user/:uid/as", cl.As)

	cl.Router.GET(prefix+"/user/login", Login(prefix+"/user/auth"))

	cl.Router.GET(prefix+"/user/logout", Logout)

	cl.Router.GET(prefix+"/user/auth", cl.Auth(prefix+"/user/auth"))

	return cl
}

// AddRoutes addes routing for game.
func (cl GameClient[G, P]) addRoutes(prefix string) GameClient[G, P] {
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

func (cl GameClient[G, P]) Close() error {
	cl.FS.Close()
	return cl.Client.Close()
}

func (cl UserServiceClient) Close() error {
	return cl.Client.Close()
}

func (cl Client) Close() error {
	return cl.User.Close()
}
