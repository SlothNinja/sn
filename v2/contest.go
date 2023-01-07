package sn

import (
	"errors"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/client"
	"github.com/gin-gonic/gin"
)

const (
	contestKind = "Contest"
)

var (
	ErrMissingKey   = errors.New("missing key")
	ErrNotFound     = errors.New("not found")
	ErrInvalidCache = errors.New("invalid cached value")
)

type ContestClient struct {
	*client.Client
}

func NewContestClient(snClient *client.Client) *ContestClient {
	return &ContestClient{snClient}
}

type Contest struct {
	c         *gin.Context
	Key       *datastore.Key `datastore:"__key__"`
	GameID    int64
	Type      Type
	R         float64
	RD        float64
	Outcome   float64
	Applied   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Contest) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(c, ps)
}

func (c *Contest) Save() ([]datastore.Property, error) {
	c.UpdatedAt = time.Now()
	return datastore.SaveStruct(c)
}

func (c *Contest) LoadKey(k *datastore.Key) error {
	c.Key = k
	return nil
}

type Result struct {
	GameID  int64
	Type    Type
	R       float64
	RD      float64
	Outcome float64
}

type ResultsMap map[*datastore.Key][]*Result

func New(id int64, pk *datastore.Key, gid int64, t Type, r, rd, outcome float64) *Contest {
	return &Contest{
		Key:     datastore.IDKey(contestKind, id, pk),
		GameID:  gid,
		Type:    t,
		R:       r,
		RD:      rd,
		Outcome: outcome,
	}
}

func key(id int64, pk *datastore.Key) *datastore.Key {
	return datastore.IDKey(contestKind, id, pk)
}

func (client *ContestClient) GenContests(places []ResultsMap) map[*datastore.Key][]*Contest {
	cs := make(map[*datastore.Key][]*Contest)
	for _, rmap := range places {
		for ukey, rs := range rmap {
			for _, r := range rs {
				cs[ukey] = append(cs[ukey], New(0, ukey, r.GameID, r.Type, r.R, r.RD, r.Outcome))
			}
		}
	}
	return cs
}

func GenContests(c *gin.Context, places []ResultsMap) []*Contest {
	var cs []*Contest
	for _, rmap := range places {
		for ukey, rs := range rmap {
			for _, r := range rs {
				cs = append(cs, New(0, ukey, r.GameID, r.Type, r.R, r.RD, r.Outcome))
			}
		}
	}
	return cs
}

func (client *ContestClient) UnappliedFor(c *gin.Context, ukey *datastore.Key, t Type) ([]*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	q := datastore.NewQuery(contestKind).
		Ancestor(ukey).
		Filter("Applied=", false).
		Filter("Type=", string(t)).
		KeysOnly()

	ks, err := client.DS.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	return client.getMulti(c, ks)
}

type ContestMap map[Type][]*Contest

func (client *ContestClient) Unapplied(c *gin.Context, ukey *datastore.Key) (ContestMap, error) {
	q := datastore.NewQuery(contestKind).
		Ancestor(ukey).
		Filter("Applied=", false).
		KeysOnly()

	ks, err := client.DS.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	cs, err := client.getMulti(c, ks)
	if err != nil {
		return nil, err
	}

	cm := make(ContestMap, len(types()))
	for _, c := range cs {
		c.Applied = true
		cm[c.Type] = append(cm[c.Type], c)
	}
	return cm, nil
}

func (client *ContestClient) mcGet(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if k == nil {
		return nil, ErrMissingKey
	}

	ek := k.Encode()
	item, found := client.Cache.Get(ek)
	if !found {
		return nil, ErrNotFound
	}

	contest, ok := item.(*Contest)
	if !ok {
		return nil, ErrInvalidCache
	}
	return contest, nil
}

func (client *ContestClient) dsGet(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if k == nil {
		return nil, ErrMissingKey
	}

	contest := new(Contest)
	err := client.DS.Get(c, k, contest)
	if err != nil {
		return nil, err
	}

	client.Cache.SetDefault(k.Encode(), contest)
	return contest, nil
}

func (client *ContestClient) get(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	contest, err := client.mcGet(c, k)
	if err != nil {
		return client.dsGet(c, k)
	}
	return contest, nil
}

func (client *ContestClient) getMulti(c *gin.Context, ks []*datastore.Key) ([]*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	l, isNil := len(ks), true
	contests := make([]*Contest, l)
	me := make(datastore.MultiError, l)
	for i, k := range ks {
		contests[i], me[i] = client.get(c, k)
		if me[i] != nil {
			isNil = false
		}
	}
	if isNil {
		return contests, nil
	}
	return contests, me
}
