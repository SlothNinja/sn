package sn

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// var ErrNotFound = errors.New("not found")

type Message struct {
	Text             string
	CreatorID        UID
	CreatorName      string
	CreatorEmailHash string
	CreatorGravType  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewMessage(u User, text string) Message {
	t := time.Now()
	return Message{
		Text:             text,
		CreatorID:        u.ID(),
		CreatorName:      u.Name,
		CreatorEmailHash: u.EmailHash,
		CreatorGravType:  u.GravType,
		CreatedAt:        t,
		UpdatedAt:        t,
	}
}

// type MLog struct {
// 	Key        *datastore.Key `datastore:"__key__"`
// 	Messages   []Message      `datastore:"-"`
// 	Read       readMap        `datastore:"-" json:"read"`
// 	SavedState string         `datastore:",noindex"`
// 	CreatedAt  time.Time
// 	UpdatedAt  time.Time
// }
//
// var (
// 	ErrMissingID = errors.New("missing identifier")
// )
//
// func (cl Client[G, I, P]) mcGetMLog(c *gin.Context, id string) (MLog, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	k := mlKey(id).Encode()
// 	item, found := cl.Cache.Get(k)
// 	if !found {
// 		return MLog{}, ErrNotFound
// 	}
//
// 	ml, ok := item.(MLog)
// 	if !ok {
// 		// delete the invaide cached value
// 		cl.Cache.Delete(k)
// 		return MLog{}, ErrInvalidCache
// 	}
// 	return ml, nil
// }
//
// func (cl MLogClient) dsGet(c *gin.Context, id int64) (MLog, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	ml := NewMLog(id)
// 	err := cl.DS.Get(c, ml.Key, &ml)
// 	return ml, err
// }

// func (cl MLogClient) Get(c *gin.Context, id int64) (MLog, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	if id == 0 {
// 		return MLog{}, ErrMissingID
// 	}
//
// 	ml, err := cl.mcGet(c, id)
// 	if err == nil {
// 		return ml, err
// 	}
//
// 	return cl.dsGet(c, id)
// }

// func (cl Client[G, I, P]) getMLog(ctx *gin.Context) gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		cl.Log.Debugf(msgEnter)
// 		defer cl.Log.Debugf(msgExit)
//
// 		var mlog MLog
// 		id := getID(ctx)
// 		snap, err := cl.mlogDocRef(id).Get(ctx)
// 		if err != nil {
// 			JErr(ctx, err)
// 			return
// 		}
//
// 		// update read message for current user
// 		// ignore any associated errors as inability to update is not worth "stopping the world"
// 		cu, err := cl.Current(ctx)
// 		if err != nil {
// 			cl.Log.Warningf(err.Error())
// 		} else if err := cl.updateRead(ctx, mlog, cu); err != nil {
// 			cl.Log.Warningf(err.Error())
// 		}
//
// 		if err := snap.DataTo(&mlog); err != nil {
// 			JErr(ctx, err)
// 			return
// 		}
//
// 		ctx.JSON(http.StatusOK, gin.H{"mlog": mlog})
// 	}
// }

func (cl Client[G, P]) updateReadHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cu, err := cl.Current(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		read, err := getRead(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.updateRead(ctx, cu, read); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, P]) updateRead(ctx *gin.Context, u User, read int) error {
	m, err := cl.getRead(ctx)
	sid := fmt.Sprintf("%d", u.ID())
	if status.Code(err) == codes.NotFound {
		m = make(readMap)
		m[sid] = 1
	} else {
		m[sid] += 1
	}
	_, err = cl.messageCollectionRef(getID(ctx)).Doc("Read").Set(ctx, m)
	return err
}

func (cl Client[G, P]) getRead(ctx *gin.Context) (readMap, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	snap, err := cl.messageCollectionRef(getID(ctx)).Doc("Read").Get(ctx)
	if err != nil {
		return nil, err
	}

	m := make(readMap)
	err = snap.DataTo(&m)
	return m, err
}

func getRead(ctx *gin.Context) (int, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var obj struct {
		Read int `json:"message"`
	}
	err := ctx.ShouldBind(&obj)
	return obj.Read, err
}

func getMessage(ctx *gin.Context) (m Message, r int, err error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var obj struct {
		Text    string `json:"text"`
		Creator User   `json:"creator"`
		Read    int    `json:"read"`
	}

	if err = ctx.ShouldBind(&obj); err != nil {
		return m, r, err
	}
	return NewMessage(obj.Creator, obj.Text), obj.Read, nil
}

// func (cl MLogClient) Unread(c *gin.Context, id int64, u User) (int, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	if id == 0 {
// 		return -1, ErrMissingID
// 	}
//
// 	ml, err := cl.mcGet(c, id)
// 	if err == nil {
// 		cl.Log.Debugf("mcGet ml.Read: %#v", ml.Read)
// 		cl.Log.Debugf("mcGet len(ml.Messages): %v", len(ml.Messages))
// 		return len(ml.Messages) - ml.Read[u.ID()], nil
// 	}
//
// 	ml, err = cl.dsGet(c, id)
// 	if err != nil {
// 		return -1, err
// 	}
// 	cl.Log.Debugf("dsGet ml.Read: %#v", ml.Read)
// 	cl.Log.Debugf("dsGet len(ml.Messages): %v", len(ml.Messages))
// 	return len(ml.Messages) - ml.Read[u.ID()], nil
// }

// func (cl MLogClient) Put(c *gin.Context, id int64, ml MLog) (*datastore.Key, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	k, err := cl.DS.Put(c, mlKey(id), ml)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return k, cl.mcPut(c, k.ID, ml)
// }

// func (cl MLogClient) mcPut(c *gin.Context, id int64, ml MLog) error {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	if id == 0 {
// 		return ErrMissingID
// 	}
//
// 	cl.Cache.SetDefault(mlKey(id).Encode(), ml)
// 	return nil
// }
//
// func (cl MLogClient) Handler(c *gin.Context) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	id, err := GetID(c)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	ml, err := cl.Get(c, id)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	cu, err := cl.User.Current(c)
// 	if err == nil {
// 		ml, err = cl.UpdateRead(c, ml, cu)
// 		if err != nil {
// 			JErr(c, err)
// 			return
// 		}
// 	}
//
// 	c.JSON(http.StatusOK, gin.H{
// 		"messages": ml.Messages,
// 		"unread":   0,
// 	})
// }

func (cl Client[G, P]) addMessageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		m, read, err := cl.validateAddMessage(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.addMessage(ctx, m, read); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, P]) validateAddMessage(ctx *gin.Context) (Message, int, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.Current(ctx)
	if err != nil {
		return Message{}, -1, err
	}

	m, read, err := getMessage(ctx)
	if err != nil {
		return Message{}, -1, err
	}

	if m.CreatorID != cu.ID() {
		return Message{}, -1, fmt.Errorf("invalid creator: %w", ErrValidation)
	}
	return m, read, nil
}

func (cl Client[G, P]) addMessage(ctx *gin.Context, m Message, read int) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	gid := getID(ctx)
	_, err := cl.messageCollectionRef(gid).NewDoc().Create(ctx, m)

	r, err := cl.getRead(ctx)
	sid := fmt.Sprintf("%d", m.CreatorID)
	if status.Code(err) == codes.NotFound {
		r = make(readMap)
		r[sid] = 1
	} else {
		r[sid] += 1
	}
	_, err = cl.messageCollectionRef(gid).Doc("Read").Set(ctx, r)
	return err
}

func readPath(uid UID) string {
	return fmt.Sprintf("Read.%d", uid)
}
