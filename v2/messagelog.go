package sn

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/client"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

type Message struct {
	Text             string    `json:"text"`
	CreatorID        int64     `json:"creatorId"`
	CreatorName      string    `json:"creatorName"`
	CreatorEmailHash string    `json:"creatorEmailHash"`
	CreatorGravType  string    `json:"creatorGravType"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func NewMessage(u *user.User, text string) *Message {
	t := time.Now()
	return &Message{
		Text:             text,
		CreatorID:        u.ID(),
		CreatorName:      u.Name,
		CreatorEmailHash: u.EmailHash,
		CreatorGravType:  u.GravType,
		CreatedAt:        t,
		UpdatedAt:        t,
	}
}

func (m *Message) Message() template.HTML {
	return template.HTML(template.HTMLEscapeString(m.Text))
}

type MLog struct {
	Key        *datastore.Key `datastore:"__key__"`
	Messages   []*Message     `datastore:"-"`
	Read       map[int64]int  `datastore:"-" json:"read"`
	SavedState string         `datastore:",noindex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ml *MLog) Load(ps []datastore.Property) error {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	err := datastore.LoadStruct(ml, ps)
	if err != nil {
		return err
	}

	obj := struct {
		Messages []*Message    `json:"messages"`
		Read     map[int64]int `json:"read"`
	}{}

	err = json.Unmarshal([]byte(ml.SavedState), &obj)
	if err != nil {
		var ms []*Message
		err = Decode(&ms, []byte(ml.SavedState))
		if err != nil {
			return err
		}
		ml.Messages = ms
		ml.Read = make(map[int64]int)
		return nil
	}
	ml.Messages = obj.Messages
	ml.Read = obj.Read
	if ml.Read == nil {
		ml.Read = make(map[int64]int)
	}
	return nil
}

func (ml *MLog) Save() ([]datastore.Property, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	obj := struct {
		Messages []*Message    `json:"messages"`
		Read     map[int64]int `json:"read"`
	}{Messages: ml.Messages, Read: ml.Read}

	v, err := json.Marshal(&obj)
	if err != nil {
		return nil, err
	}
	ml.SavedState = string(v)
	return datastore.SaveStruct(ml)
}

func (ml *MLog) LoadKey(k *datastore.Key) error {
	ml.Key = k
	return nil
}

type MLogClient struct {
	*client.Client
	User *user.Client
}

func NewMLogClient(snClient *client.Client, userClient *user.Client) *MLogClient {
	return &MLogClient{
		Client: snClient,
		User:   userClient,
	}
}

func NewMLog(id int64) *MLog {
	return &MLog{Key: mlKey(id)}
}

func mlKey(id int64) *datastore.Key {
	return datastore.IDKey(mlKind, id, nil)
}

const (
	mlKind = "MessageLog"
	// mlKey    = "MessageLog"
)

func (ml *MLog) AddMessage(u *user.User, text string) *Message {
	m := NewMessage(u, text)
	ml.Messages = append(ml.Messages, m)
	if ml.Read == nil {
		ml.Read = make(map[int64]int)
	}
	ml.Read[u.ID()] = len(ml.Messages)
	return m
}

var (
	ErrMissingID = errors.New("missing identifier")
)

func (cl *MLogClient) mcGet(c *gin.Context, id int64) (*MLog, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	k := mlKey(id).Encode()
	item, found := cl.Cache.Get(k)
	if !found {
		return nil, ErrNotFound
	}

	ml, ok := item.(*MLog)
	if !ok {
		// delete the invaide cached value
		cl.Cache.Delete(k)
		return nil, ErrInvalidCache
	}
	return ml, nil
}

func (cl *MLogClient) dsGet(c *gin.Context, id int64) (*MLog, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	ml := NewMLog(id)
	err := cl.DS.Get(c, ml.Key, ml)
	return ml, err
}

func (cl *MLogClient) Get(c *gin.Context, id int64) (*MLog, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if id == 0 {
		return nil, ErrMissingID
	}

	ml, err := cl.mcGet(c, id)
	if err == nil {
		return ml, err
	}

	return cl.dsGet(c, id)
}

func (cl *MLogClient) UpdateRead(c *gin.Context, ml *MLog, u *user.User) (*MLog, error) {
	ml.Read[u.ID()] = len(ml.Messages)
	_, err := cl.Put(c, ml.Key.ID, ml)
	if err != nil {
		return nil, err
	}
	return ml, nil
}

func (cl *MLogClient) Unread(c *gin.Context, id int64, u *user.User) (int, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if id == 0 {
		return -1, ErrMissingID
	}

	ml, err := cl.mcGet(c, id)
	if err == nil {
		cl.Log.Debugf("mcGet ml.Read: %#v", ml.Read)
		cl.Log.Debugf("mcGet len(ml.Messages): %v", len(ml.Messages))
		return len(ml.Messages) - ml.Read[u.ID()], nil
	}

	ml, err = cl.dsGet(c, id)
	if err != nil {
		return -1, err
	}
	cl.Log.Debugf("dsGet ml.Read: %#v", ml.Read)
	cl.Log.Debugf("dsGet len(ml.Messages): %v", len(ml.Messages))
	return len(ml.Messages) - ml.Read[u.ID()], nil
}

func (cl *MLogClient) Put(c *gin.Context, id int64, ml *MLog) (*datastore.Key, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	k, err := cl.DS.Put(c, mlKey(id), ml)
	if err != nil {
		return nil, err
	}

	return k, cl.mcPut(c, k.ID, ml)
}

func (cl *MLogClient) mcPut(c *gin.Context, id int64, ml *MLog) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	if id == 0 {
		return ErrMissingID
	}

	cl.Cache.SetDefault(mlKey(id).Encode(), ml)
	return nil
}

func (cl *MLogClient) Handler(c *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	id, err := GetID(c)
	if err != nil {
		JErr(c, err)
		return
	}

	ml, err := cl.Get(c, id)
	if err != nil {
		JErr(c, err)
		return
	}

	cu, err := cl.User.Current(c)
	if err == nil {
		ml, err = cl.UpdateRead(c, ml, cu)
		if err != nil {
			JErr(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": ml.Messages,
		"unread":   0,
	})
}

func (cl *MLogClient) AddMessageHandler(c *gin.Context) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.User.Current(c)
	if err != nil {
		JErr(c, err)
		return
	}

	id, err := GetID(c)
	if err != nil {
		JErr(c, err)
		return
	}

	obj := struct {
		Message string     `json:"message"`
		Creator *user.User `json:"creator"`
	}{}

	err = c.ShouldBind(&obj)
	if err != nil {
		JErr(c, err)
		return
	}

	if obj.Creator.ID() != cu.ID() {
		JErr(c, fmt.Errorf("invalid creator: %w", ErrValidation))
		return
	}

	ml, err := cl.Get(c, id)
	if err != nil {
		JErr(c, err)
		return
	}

	m := ml.AddMessage(cu, obj.Message)
	_, err = cl.Put(c, id, ml)
	if err != nil {
		JErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": m,
		"unread":  0,
	})
}
