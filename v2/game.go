package sn

import (
	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
)

type Client struct {
	DS *datastore.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{DS: dsClient}
}

func GamesRoot(c *gin.Context) *datastore.Key {
	return datastore.NameKey("Games", "root", nil)
}
