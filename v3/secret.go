package sn

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gorilla/securecookie"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	hashKeyLength    = 64
	blockKeyLength   = 32
	secretsProjectID = "SECRETS_PROJECT_ID"
	secretsDSHost    = "SECRETS_DS_HOST"
	sessionName      = "sng-oauth"
)

// Secret stores secrets for secure cookie
type Secret struct {
	HashKey   []byte         `json:"hashKey"`
	BlockKey  []byte         `json:"blockKey"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Key       *datastore.Key `datastore:"__key__" json:"-"`
}

type secretsClient struct {
	Client
	DS *datastore.Client
}

func (cl Client) getSessionSecrets(ctx context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	s, found := cl.mcGetSessionSecrets()
	if found {
		return s, nil
	}

	s, err := cl.dsGetSessionSecrets(ctx)
	if err != datastore.ErrNoSuchEntity {
		return s, err
	}

	cl.Log.Debugf("generated new secrets")
	return cl.updateSessionSecrets(ctx)
}

// mcGet attempts to pull secret from cache
func (cl Client) mcGetSessionSecrets() (*Secret, bool) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	k := SecretsKey().Encode()

	item, found := cl.Cache.Get(k)
	if !found {
		return nil, false
	}

	s, ok := item.(*Secret)
	if !ok {
		cl.Cache.Delete(k)
		return nil, false
	}
	return s, true
}

// dsGet attempt to pull secret from datastore
func (cl Client) dsGetSessionSecrets(ctx context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	secretsDS, err := cl.getSessionSecretsDatastore(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get secrets datastore: %v", err)
	}

	s := &Secret{Key: SecretsKey()}
	err = secretsDS.Get(ctx, s.Key, s)
	return s, err
}

func (cl Client) updateSessionSecrets(c context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	secretsDS, err := cl.getSessionSecretsDatastore(c)
	if err != nil {
		return nil, fmt.Errorf("unable to get secrets datastore: %v", err)
	}

	s, err := genSessionSecrets()
	if err != nil {
		return nil, err
	}

	_, err = secretsDS.Put(c, s.Key, s)
	return s, err
}

func (cl Client) getSessionSecretsDatastore(ctx context.Context) (*datastore.Client, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if IsProduction() {
		return cl.getProductionSessionSecretsDatastore(ctx)
	}
	return cl.getDevelopmentSessionSecretsDataStore(ctx)
}

func (cl Client) getProductionSessionSecretsDatastore(ctx context.Context) (*datastore.Client, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return datastore.NewClient(ctx, cl.secretsProjectID)
}

func (cl Client) getDevelopmentSessionSecretsDataStore(ctx context.Context) (*datastore.Client, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return datastore.NewClient(
		ctx,
		cl.secretsProjectID,
		option.WithEndpoint(cl.secretsDSURL),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCConnectionPool(50),
	)
}

func SecretsKey() *datastore.Key {
	return datastore.NameKey("Secrets", "cookie", nil)
}

func genSessionSecrets() (*Secret, error) {
	s := &Secret{
		HashKey:  securecookie.GenerateRandomKey(hashKeyLength),
		BlockKey: securecookie.GenerateRandomKey(blockKeyLength),
		Key:      SecretsKey(),
	}

	if s.HashKey == nil {
		return s, fmt.Errorf("generated hashKey was nil")
	}

	if s.BlockKey == nil {
		return s, fmt.Errorf("generated blockKey was nil")
	}

	return s, nil
}

func (s *Secret) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(s, ps)
}

func (s *Secret) Save() ([]datastore.Property, error) {
	s.UpdatedAt = time.Now()
	return datastore.SaveStruct(s)
}

func (s *Secret) LoadKey(k *datastore.Key) error {
	s.Key = k
	return nil
}

// Store represents a secure cookie store
type Store cookie.Store

// NewStore generates a new secure cookie store
func (cl Client) initSession(ctx context.Context) Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	s, err := cl.getSessionSecrets(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to create cookie store: %v", err))
	}

	store := cookie.NewStore(s.HashKey, s.BlockKey)
	opts := sessions.Options{
		Domain: "fake-slothninja.com",
		Path:   "/",
	}
	if IsProduction() {
		opts = sessions.Options{
			Domain: "slothninja.com",
			Path:   "/",
			MaxAge: 60 * 60 * 24, // 1 Day in seconds
			Secure: true,
		}
	}
	store.Options(opts)
	cl.Router.Use(sessions.Sessions(sessionName, store))
	return cl
}
