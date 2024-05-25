package sn

import (
	"encoding/json"
	"os"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
)

const (
	msgEnter    = "Entering"
	msgExit     = "Exiting"
	GAE_VERSION = "GAE_VERSION"
	NODE_ENV    = "NODE_ENV"
	production  = "production"
	idParam     = "id"
	rootKind    = "Root"
)

func RootKey(id int64) *datastore.Key {
	return datastore.IDKey(rootKind, id, nil)
}

// IsProduction returns true if NODE_ENV environment variable is equal to "production".
// GAE sets NODE_ENV environement to "production" on deployment.
// NODE_ENV can be overridden in app.yaml configuration.
func IsProduction() bool {
	return os.Getenv(NODE_ENV) == production
}

func VersionID() string {
	return os.Getenv(GAE_VERSION)
}

func (cl *Client) RequireLogin(ctx *gin.Context) (*User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.getCU(ctx)
	if err != nil {
		return nil, ErrNotLoggedIn
	}
	return cu, nil
}

func (cl *Client) RequireAdmin(ctx *gin.Context) (*User, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	admin, err := cl.getAdmin(ctx)
	if !admin || err != nil {
		return nil, ErrNotAdmin
	}

	return cl.RequireLogin(ctx)
}

const (
	gamerKey  = "Game"
	gamersKey = "Games"
	homePath  = "/"
	adminKey  = "Admin"
)

type IndexEntry struct {
	Key        *datastore.Key `datastore:"__key__"`
	Properties []datastore.Property
}

func (e *IndexEntry) Load(ps []datastore.Property) error {
	e.Properties = ps
	return nil
}

func (e *IndexEntry) Save() ([]datastore.Property, error) {
	return e.Properties, nil
}

func (e *IndexEntry) LoadKey(k *datastore.Key) error {
	e.Key = k
	return nil
}

func (e *IndexEntry) id() int64 {
	if e.Key != nil {
		return e.Key.ID
	}
	return 0
}

// MarshalJSON implements json.Marshaler interface
func (e IndexEntry) MarshalJSON() ([]byte, error) {

	data := make(map[string]interface{})
	for _, p := range e.Properties {
		switch p.Name {
		case "CreatorID":
			data["creatorId"] = p.Value
		case "CreatorKey":
			data["creatorKey"] = p.Value
		case "CreatorName":
			data["creatorName"] = p.Value
		case "CreatorEmailHash":
			data["creatorEmailHash"] = p.Value
		case "CreatorGravType":
			data["creatorGravType"] = p.Value
		case "Type":
			data["type"] = p.Value
		case "Title":
			data["title"] = p.Value
		case "UserIDS":
			data["userIds"] = p.Value
		case "UserNames":
			data["userNames"] = p.Value
		case "UserEmailHashes":
			data["userEmailHashes"] = p.Value
		case "UserGravTypes":
			data["userGravTypes"] = p.Value
		case "UserKeys":
			data["userKeys"] = p.Value
		case "Password":
			data["password"] = p.Value
		case "PasswordHash":
			data["passwordHash"] = p.Value
		case "UpdatedAt":
			data["updatedAt"] = p.Value
		case "CPUserIndices":
			data["cpUserIndices"] = p.Value
		case "CPIDS":
			data["cpids"] = p.Value
		case "WinnerIDS":
			data["winnerIndices"] = p.Value
		case "WinnerKeys":
			data["winnerKeys"] = p.Value
		}
	}

	data["key"] = e.Key
	data["id"] = e.id()

	password, ok := data["password"].(string)
	if ok {
		passwordHash, ok := data["passwordHash"].([]byte)
		if ok {
			data["public"] = (len(password) == 0) && (len(passwordHash) == 0)
		}
	}
	delete(data, "password")
	delete(data, "passwordHash")

	updatedAt, ok := data["updatedAt"].(time.Time)
	if ok {
		data["lastUpdated"] = LastUpdated(updatedAt)
	}

	return json.Marshal(data)
}
