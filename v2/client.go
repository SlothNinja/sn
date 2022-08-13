package sn

import (
	"context"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const (
	GAE_VERSION = "GAE_VERSION"
	NODE_ENV    = "NODE_ENV"
	production  = "production"
)

// IsProduction returns true if NODE_ENV environment variable is equal to "production".
// GAE sets NODE_ENV environement to "production" on deployment.
// NODE_ENV can be overridden in app.yaml configuration.
func IsProduction() bool {
	return os.Getenv(NODE_ENV) == production
}

type Client struct {
	DS     *datastore.Client
	Log    *log.Logger
	Cache  *cache.Cache
	Router *gin.Engine
}

type Options struct {
	Prefix    string
	ProjectID string
	DSURL     string
	Logger    *log.Logger
	Cache     *cache.Cache
	Router    *gin.Engine
}

const (
	msgEnter = "Entering"
	msgExit  = "Exiting"
)

func NewClient(ctx context.Context, opt Options) *Client {
	opt.Logger.Debugf(msgEnter)
	defer opt.Logger.Debugf(msgExit)

	if IsProduction() {
		opt.Logger.Debugf("production")
		dsClient, err := datastore.NewClient(ctx, opt.ProjectID)
		if err != nil {
			opt.Logger.Panicf("unable to connect to user database: %w", err)
		}
		return &Client{DS: dsClient, Log: opt.Logger, Cache: opt.Cache, Router: opt.Router}
	}
	opt.Logger.Debugf("development")
	dsClient, err := datastore.NewClient(
		ctx,
		opt.ProjectID,
		option.WithEndpoint(opt.DSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
		option.WithGRPCConnectionPool(50),
	)
	if err != nil {
		opt.Logger.Panicf("unable to connect to user database: %w", err)
	}
	return &Client{DS: dsClient, Log: opt.Logger, Cache: opt.Cache, Router: opt.Router}
}

func (cl *Client) Close() error {
	return cl.DS.Close()
}
