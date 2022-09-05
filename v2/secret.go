package sn

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/client"
	"github.com/SlothNinja/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gorilla/securecookie"
)

const (
	hashKeyLength    = 64
	blockKeyLength   = 32
	secretsProjectID = "SECRETS_PROJECT_ID"
	secretsDSHost    = "SECRETS_DS_HOST"
)

// Secret stores secrets for secure cookie
type Secret struct {
	HashKey   []byte         `json:"hashKey"`
	BlockKey  []byte         `json:"blockKey"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Key       *datastore.Key `datastore:"__key__" json:"-"`
}

// CookieClient for generating secure cookie store
type CookieClient struct {
	*client.Client
}

// NewCookieClient creates a client for generating a secured cookie store
func NewCookieClient(snClient *client.Client) *CookieClient {
	return &CookieClient{snClient}
}

func (cl *CookieClient) get(c context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	s, found := cl.mcGet()
	if found {
		return s, nil
	}

	s, err := cl.dsGet(c)
	if err != datastore.ErrNoSuchEntity {
		return s, err
	}

	cl.Log.Warningf("generated new secrets")
	return cl.update(c)
}

// mcGet attempts to pull secret from cache
func (cl *CookieClient) mcGet() (*Secret, bool) {
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
func (cl *CookieClient) dsGet(c context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	s := &Secret{Key: SecretsKey()}
	err := cl.DS.Get(c, s.Key, s)
	return s, err
}

func (cl *CookieClient) update(c context.Context) (*Secret, error) {
	s, err := GenSecrets()
	if err != nil {
		return nil, err
	}

	_, err = cl.DS.Put(c, s.Key, s)
	return s, err
}

func SecretsKey() *datastore.Key {
	return datastore.NameKey("Secrets", "cookie", nil)
}

func GenSecrets() (*Secret, error) {
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
func (cl *CookieClient) NewStore(ctx context.Context) (Store, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	s, err := cl.get(ctx)
	if err != nil {
		return nil, err
	}

	if !client.IsProduction() {
		cl.Log.Debugf("hashKey: %s\nblockKey: %s",
			base64.StdEncoding.EncodeToString(s.HashKey),
			base64.StdEncoding.EncodeToString(s.BlockKey),
		)
		store := cookie.NewStore(s.HashKey, s.BlockKey)
		opts := sessions.Options{
			Domain: "fake-slothninja.com",
			Path:   "/",
		}
		store.Options(opts)
		return store, nil
	}
	store := cookie.NewStore(s.HashKey, s.BlockKey)
	opts := sessions.Options{
		Domain: "slothninja.com",
		Path:   "/",
		Secure: true,
	}
	store.Options(opts)
	return store, nil
}
