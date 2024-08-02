package sn

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/securecookie"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	hashKeyLength  = 64
	blockKeyLength = 32
)

// sessionSecret stores secrets for secure cookie
type sessionSecret struct {
	HashKey   []byte    `json:"hashKey"`
	BlockKey  []byte    `json:"blockKey"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (cl *Client) getSessionSecrets(ctx context.Context) (*sessionSecret, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	s, found := cl.mcGetSessionSecrets()
	if found {
		return s, nil
	}

	s, err := cl.dsGetSessionSecrets(ctx)
	if err != datastore.ErrNoSuchEntity {
		return s, err
	}

	slog.Debug("generated new secrets")
	return cl.updateSessionSecrets(ctx)
}

// mcGet attempts to pull secret from cache
func (cl *Client) mcGetSessionSecrets() (*sessionSecret, bool) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	k := secretsKey().Encode()

	item, found := cl.Cache.Get(k)
	if !found {
		return nil, false
	}

	s, ok := item.(*sessionSecret)
	if !ok {
		cl.Cache.Delete(k)
		return nil, false
	}
	return s, true
}

// dsGet attempt to pull secret from datastore
func (cl *Client) dsGetSessionSecrets(ctx context.Context) (*sessionSecret, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	secretsDS, err := cl.getSessionSecretsDatastore(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get secrets datastore: %v", err)
	}

	s := new(sessionSecret)
	err = secretsDS.Get(ctx, secretsKey(), s)
	if err != nil {
		return nil, err
	}

	k := secretsKey().Encode()
	cl.Cache.Set(k, s, 0)
	return s, err
}

func (cl *Client) updateSessionSecrets(c context.Context) (*sessionSecret, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	secretsDS, err := cl.getSessionSecretsDatastore(c)
	if err != nil {
		return nil, fmt.Errorf("unable to get secrets datastore: %v", err)
	}

	s, err := genSessionSecrets()
	if err != nil {
		return nil, err
	}

	_, err = secretsDS.Put(c, secretsKey(), s)
	return s, err
}

func (cl *Client) getSessionSecretsDatastore(ctx context.Context) (*datastore.Client, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if IsProduction() {
		return cl.getProductionSessionSecretsDatastore(ctx)
	}
	return cl.getDevelopmentSessionSecretsDataStore(ctx)
}

func (cl *Client) getProductionSessionSecretsDatastore(ctx context.Context) (*datastore.Client, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return datastore.NewClient(ctx, cl.secretsProjectID)
}

func (cl *Client) getDevelopmentSessionSecretsDataStore(ctx context.Context) (*datastore.Client, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return datastore.NewClient(
		ctx,
		cl.secretsProjectID,
		option.WithEndpoint(cl.secretsDSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCConnectionPool(50),
	)
}

func secretsKey() *datastore.Key {
	return datastore.NameKey("Secrets", "cookie", nil)
}

func genSessionSecrets() (*sessionSecret, error) {
	s := &sessionSecret{
		HashKey:   securecookie.GenerateRandomKey(hashKeyLength),
		BlockKey:  securecookie.GenerateRandomKey(blockKeyLength),
		UpdatedAt: time.Now(),
	}

	if s.HashKey == nil {
		return s, fmt.Errorf("generated hashKey was nil")
	}

	if s.BlockKey == nil {
		return s, fmt.Errorf("generated blockKey was nil")
	}

	return s, nil
}
