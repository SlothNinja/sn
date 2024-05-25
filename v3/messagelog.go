package sn

import (
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

// var ErrNotFound = errors.New("not found")

type Message struct {
	Text             string
	CreatorID        UID
	CreatorName      string
	CreatorEmailHash string
	CreatorGravType  string
	Read             []UID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewMessage(u *User, text string) Message {
	t := time.Now()
	return Message{
		Text:             text,
		CreatorID:        u.ID,
		CreatorName:      u.Name,
		CreatorEmailHash: u.EmailHash,
		CreatorGravType:  u.GravType,
		Read:             []UID{u.ID},
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

func (cl *GameClient[GT, G]) updateReadHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		read, err := getRead(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		Debugf("read: %#v", read)

		if err := cl.updateRead(ctx, cu.ID, read); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) updateRead(ctx *gin.Context, uid UID, read []string) error {
	for _, mid := range read {
		if _, err := cl.messageDocRef(getID(ctx), mid).Update(ctx, []firestore.Update{
			{Path: "Read", Value: firestore.ArrayUnion(uid)},
		}); err != nil {
			return err
		}
	}
	return nil
	// m, err := cl.getRead(ctx)
	// sid := fmt.Sprintf("%d", u.ID())
	// if status.Code(err) == codes.NotFound {
	// 	m = make(readMap)
	// 	m[sid] = 1
	// } else {
	// 	m[sid] += 1
	// }
	// _, err = cl.readDocRef(getID(ctx)).Set(ctx, m)
	// return err
}

// func (cl Client[G, P]) getRead(ctx *gin.Context) (readMap, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	snap, err := cl.readDocRef(getID(ctx)).Get(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	m := make(readMap)
// 	err = snap.DataTo(&m)
// 	return m, err
// }

func getRead(ctx *gin.Context) ([]string, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var obj struct {
		Read []string
	}
	err := ctx.ShouldBind(&obj)
	return obj.Read, err
}

func getMessage(ctx *gin.Context) (Message, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var obj struct {
		Text    string   `json:"text"`
		Creator *User    `json:"creator"`
		Read    []string `json:"read"`
	}

	if err := ctx.ShouldBind(&obj); err != nil {
		return Message{}, err
	}
	return NewMessage(obj.Creator, obj.Text), nil
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

func (cl *GameClient[GT, G]) addMessageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		m, err := cl.validateAddMessage(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.addMessage(ctx, m); err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) validateAddMessage(ctx *gin.Context) (Message, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	cu, err := cl.RequireLogin(ctx)
	if err != nil {
		return Message{}, err
	}

	m, err := getMessage(ctx)
	if err != nil {
		return Message{}, err
	}

	if m.CreatorID != cu.ID {
		return Message{}, fmt.Errorf("invalid creator: %w", ErrValidation)
	}
	return m, nil
}

func (cl *GameClient[GT, G]) addMessage(ctx *gin.Context, m Message) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	_, err := cl.messagesCollectionRef(getID(ctx)).NewDoc().Create(ctx, m)
	return err
}

// func (cl Client[G, P]) createMLog(ctx *gin.Context) error {
// 	_, err := cl.mlogDocRef(getID(ctx)).Get(ctx)
// 	if err == nil {
// 		return nil
// 	}
// 	if status.Code(err) != codes.NotFound {
// 		return err
// 	}
// 	_, err = cl.mlogDocRef(getID(ctx)).Create(ctx, nil)
// 	return err
// }

// func readPath(uid UID) string {
// 	return fmt.Sprintf("Read.%d", uid)
// }
