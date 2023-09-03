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

type GameClient[G Gamer[G], P Playerer] struct {
	Client
	FS *firestore.Client
}

type options struct {
	projectID        string
	url              string
	frontEndURL      string
	backEndURL       string
	dsURL            string
	port             string
	frontEndPort     string
	backEndPort      string
	dsPort           string
	userProjectID    string
	userDSURL        string
	userFrontURL     string
	secretsProjectID string
	secretsDSURL     string
	loggerID         string
	corsAllow        []string
	prefix           string
	home             string
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

func (cl Client) GetProjectID() string {
	return cl.projectID
}

func WithURL(url string) Option {
	return func(cl Client) Client {
		cl.url = url
		return cl
	}
}

func getURL() string {
	return os.Getenv("URL")
}

func (cl Client) GetURL() string {
	return cl.url
}

func WithFrontEndURL(url string) Option {
	return func(cl Client) Client {
		cl.frontEndURL = url
		return cl
	}
}

func getFrontEndURL() string {
	if url, found := os.LookupEnv("FE_URL"); found {
		return url
	}
	return getURL()
}

func (cl Client) GetFrontEndURL() string {
	return cl.frontEndURL
}

func WithBackEndURL(url string) Option {
	return func(cl Client) Client {
		cl.backEndURL = url
		return cl
	}
}

func getBackEndURL() string {
	if url, found := os.LookupEnv("BE_URL"); found {
		return url
	}
	return getURL()
}

func (cl Client) GetBackEndURL() string {
	return cl.backEndURL
}

func WithDSURL(url string) Option {
	return func(cl Client) Client {
		cl.dsURL = url
		return cl
	}
}

func getDSURL() string {
	if url, found := os.LookupEnv("DS_URL"); found {
		return url
	}
	return getURL()
}

func (cl Client) GetDSURL() string {
	return cl.dsURL
}

func WithPort(port string) Option {
	return func(cl Client) Client {
		cl.port = port
		return cl
	}
}

func getPort() string {
	return os.Getenv("PORT")
}

func (cl Client) GetPort() string {
	return cl.port
}

func WithFrontEndPort(port string) Option {
	return func(cl Client) Client {
		cl.frontEndPort = port
		return cl
	}
}

func getFrontEndPort() string {
	if port, found := os.LookupEnv("FE_PORT"); found {
		return port
	}
	return getPort()
}

func (cl Client) GetFrontEndPort() string {
	return cl.frontEndPort
}

func WithBackEndPort(port string) Option {
	return func(cl Client) Client {
		cl.backEndPort = port
		return cl
	}
}

func getBackEndPort() string {
	if port, found := os.LookupEnv("BE_PORT"); found {
		return port
	}
	return getPort()
}

func (cl Client) GetBackEndPort() string {
	return cl.backEndPort
}

func WithDSPort(port string) Option {
	return func(cl Client) Client {
		cl.dsPort = port
		return cl
	}
}

func getDSPort() string {
	if port, found := os.LookupEnv("DS_PORT"); found {
		return port
	}
	return getPort()
}

func (cl Client) GetDSPort() string {
	return cl.dsPort
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

func (cl Client) GetUserProjectID() string {
	return cl.userProjectID
}

func WithSecretsProjectID(id string) Option {
	return func(cl Client) Client {
		cl.secretsProjectID = id
		return cl
	}
}

func getSecretsProjectID() string {
	if id, found := os.LookupEnv("SECRETS_PROJECT_ID"); found {
		return id
	}
	return "user-slothninja-games"
}

func (cl Client) GetSecretsProjectID() string {
	return cl.secretsProjectID
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
		return "user.slothninja.com"
	}
	return "user.fake-slothninja.com:8086"
}

func (cl Client) GetUserDSURL() string {
	return cl.userDSURL
}

func WithUserFrontURL(url string) Option {
	return func(cl Client) Client {
		cl.userFrontURL = url
		return cl
	}
}

func getUserFrontURL() string {
	if url, found := os.LookupEnv("USER_FRONT_URL"); found {
		return url
	}
	if IsProduction() {
		return "user.slothninja.com"
	}
	return "user.fake-slothninja.com:8088"
}

func (cl Client) GetUserFrontURL() string {
	return cl.userFrontURL
}

func WithSecretsDSURL(url string) Option {
	return func(cl Client) Client {
		cl.secretsDSURL = url
		return cl
	}
}

func getSecretsDSURL() string {
	if url, found := os.LookupEnv("SECRETS_DS_URL"); found {
		return url
	}
	if IsProduction() {
		return "user.slothninja.com"
	}
	return "user.fake-slothninja.com:8086"
}

func (cl Client) GetSecretsDSURL() string {
	return cl.secretsDSURL
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

func (cl Client) GetLoggerID() string {
	return cl.loggerID
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

func (cl Client) GetCORSAllow() []string {
	return cl.corsAllow
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
	return "/sn"
}

func (cl Client) GetPrefix() string {
	return cl.prefix
}

func WithHome(path string) Option {
	return func(cl Client) Client {
		cl.home = path
		return cl
	}
}

func getHome() string {
	if prefix, found := os.LookupEnv("HOME"); found {
		return prefix
	}
	return "/"
}

func (cl Client) GetHome() string {
	return cl.home
}

type Option func(Client) Client

func defaultClient() Client {
	var cl Client
	cl.projectID = getProjectID()
	cl.url = getURL()
	cl.frontEndURL = getFrontEndURL()
	cl.backEndURL = getBackEndURL()
	cl.dsURL = getDSURL()
	cl.port = getPort()
	cl.backEndPort = getBackEndPort()
	cl.frontEndPort = getFrontEndPort()
	cl.dsPort = getDSPort()
	cl.userProjectID = getUserProjectID()
	cl.userDSURL = getUserDSURL()
	cl.userFrontURL = getUserFrontURL()
	cl.secretsProjectID = getSecretsProjectID()
	cl.secretsDSURL = getSecretsDSURL()
	cl.loggerID = getLoggerID()
	cl.corsAllow = getCORSAllow()
	cl.prefix = getPrefix()
	cl.home = getHome()
	return cl
}

func NewClient(ctx context.Context, opts ...Option) Client {
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
		initMode().
		addRoutes()
}

func (cl Client) initLogger() Client {
	lClient, err := NewLogClient(cl.projectID)
	if err != nil {
		log.Panicf("unable to create logging client: %v", err)
	}
	cl.Log = lClient.Logger(cl.loggerID)
	return cl
}

func (cl Client) initCache() Client {
	cl.Cache = cache.New(30*time.Minute, 10*time.Minute)
	return cl
}

func (cl Client) initRouter() Client {
	cl.Router = gin.Default()
	return cl
}

func (cl Client) initMode() Client {
	if IsProduction() {
		gin.SetMode(gin.ReleaseMode)
		cl.Router.TrustedPlatform = gin.PlatformGoogleAppEngine
		return cl
	}

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
	cl := GameClient[G, P]{Client: NewClient(ctx, opts...)}

	var err error
	if cl.FS, err = firestore.NewClient(ctx, cl.projectID); err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}
	return cl.addRoutes(cl.prefix)
}

// AddRoutes addes routing for game.
func (cl Client) addRoutes() Client {
	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/current", cl.cuHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

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

func (cl Client) Close() error {
	return nil
}
