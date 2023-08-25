package sn

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gorilla/securecookie"
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

// // CookieClient for generating secure cookie store
//
//	type CookieClient struct {
//		*Client
//	}
//
// // NewCookieClient creates a client for generating a secured cookie store
//
//	func NewCookieClient(snClient *Client) *CookieClient {
//		return &CookieClient{snClient}
//	}
func (cl Client) getSecrets(c context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	s, found := cl.mcGetSecrets()
	if found {
		return s, nil
	}

	s, err := cl.dsGetSecrets(c)
	if err != datastore.ErrNoSuchEntity {
		return s, err
	}

	cl.Log.Warningf("generated new secrets")
	return cl.updateSecrets(c)
}

// mcGet attempts to pull secret from cache
func (cl Client) mcGetSecrets() (*Secret, bool) {
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
func (cl Client) dsGetSecrets(c context.Context) (*Secret, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	s := &Secret{Key: SecretsKey()}
	err := cl.User.Get(c, s.Key, s)
	return s, err
}

func (cl Client) updateSecrets(c context.Context) (*Secret, error) {
	s, err := GenSecrets()
	if err != nil {
		return nil, err
	}

	_, err = cl.User.Put(c, s.Key, s)
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
func (cl Client) NewStore(ctx context.Context) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	s, err := cl.getSecrets(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to create cookie store: %v", err))
	}

	var store cookie.Store
	if !IsProduction() {
		store = cookie.NewStore(s.HashKey, s.BlockKey)
		opts := sessions.Options{
			Domain: "fake-slothninja.com",
			Path:   "/",
		}
		store.Options(opts)
	} else {
		store = cookie.NewStore(s.HashKey, s.BlockKey)
		opts := sessions.Options{
			Domain: "slothninja.com",
			Path:   "/",
			MaxAge: 60 * 60 * 24, // 1 Day in seconds
			Secure: true,
		}
		store.Options(opts)
	}
	cl.Router.Use(sessions.Sessions(sessionName, store))
}
