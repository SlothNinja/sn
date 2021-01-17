package sn

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type Client struct {
	DS     *datastore.Client
	Log    *log.Logger
	Cache  *cache.Cache
	Router *gin.Engine
}

func NewClient(dsClient *datastore.Client, logger *log.Logger, mcache *cache.Cache, router *gin.Engine) *Client {
	return &Client{
		DS:     dsClient,
		Log:    logger,
		Cache:  mcache,
		Router: router,
	}
}
