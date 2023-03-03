package sn

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	DS     *datastore.Client
	Log    *Logger
	Cache  *cache.Cache
	Router *gin.Engine
}

type Options struct {
	Prefix    string
	ProjectID string
	DSURL     string
	Logger    *Logger
	Cache     *cache.Cache
	Router    *gin.Engine
}

func NewClient(ctx context.Context, opt Options) *Client {
	opt.Logger.Debugf(msgEnter)
	defer opt.Logger.Debugf(msgEnter)

	if IsProduction() {
		opt.Logger.Debugf("production")
		dsClient, err := datastore.NewClient(ctx, opt.ProjectID)
		if err != nil {
			panic(fmt.Errorf("unable to connect to user database: %w", err))
		}
		return &Client{
			DS:     dsClient,
			Log:    opt.Logger,
			Cache:  opt.Cache,
			Router: opt.Router,
		}
	}
	opt.Logger.Debugf("development")
	dsClient, err := datastore.NewClient(
		ctx,
		opt.ProjectID,
		option.WithEndpoint(opt.DSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCConnectionPool(50),
	)
	if err != nil {
		panic(fmt.Errorf("unable to connect to user database: %w", err))
	}
	return &Client{
		DS:     dsClient,
		Log:    opt.Logger,
		Cache:  opt.Cache,
		Router: opt.Router,
	}
}

func (cl *Client) Close() error {
	return cl.DS.Close()
}
