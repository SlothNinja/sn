package sn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
)

type MLog struct {
	Key        *datastore.Key `datastore:"__key__" json:"key"`
	Messages   []*Message     `datastore:"-" json:"-"`
	SavedState string         `datastore:",noindex" json:"messages"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

func (ml *MLog) Load(ps []datastore.Property) error {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	err := datastore.LoadStruct(ml, ps)
	if err != nil {
		return err
	}

	var ms []*Message
	err = json.Unmarshal([]byte(ml.SavedState), &ms)
	if err != nil {
		return err
	}
	ml.Messages = ms
	return nil
}

func (ml *MLog) Save() ([]datastore.Property, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	bs, err := json.Marshal(ml.Messages)
	if err != nil {
		return nil, err
	}
	ml.SavedState = string(bs)
	return datastore.SaveStruct(ml)
}

func (ml *MLog) LoadKey(k *datastore.Key) error {
	ml.Key = k
	return nil
}

func NewMLog(id int64) *MLog {
	return &MLog{Key: datastore.IDKey(mlogKind, id, nil)}
}

func (cl Client) AddMessage(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	id, err := getID(c)
	if err != nil {
		JErr(c, err)
		return
	}

	ml, err := cl.getMLog(c, id)
	if err != nil {
		JErr(c, fmt.Errorf("unable to get message log with ID: %v", id))
		return
	}

	obj := struct {
		Creator User   `json:"creator"`
		Message string `json:"message"`
	}{}

	err = c.ShouldBind(&obj)
	if err != nil {
		JErr(c, err)
		return
	}

	m := ml.NewMessage(c)
	m.Text = obj.Message
	m.Creator = obj.Creator
	_, err = cl.DS.Put(c, ml.Key, ml)
	if err != nil {
		JErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": m})
}

func getID(c *gin.Context) (int64, error) {
	sid := c.Param("id")
	return strconv.ParseInt(sid, 10, 64)
}

func (cl Client) getMLog(c *gin.Context, id int64) (*MLog, error) {
	ml := NewMLog(id)
	err := cl.DS.Get(c, ml.Key, ml)
	return ml, err
}

func (cl Client) GetMLog(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	id, err := getID(c)
	if err != nil {
		JErr(c, err)
		return
	}

	ml, err := cl.getMLog(c, id)
	if err != nil {
		JErr(c, fmt.Errorf("unable to get message log with ID: %v: %w", id, err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": ml.Messages})
}

func From(c *gin.Context) (ml *MLog) {
	ml, _ = c.Value(mlogKey).(*MLog)
	return
}

func with(c *gin.Context, ml *MLog) *gin.Context {
	c.Set(mlogKey, ml)
	return c
}
