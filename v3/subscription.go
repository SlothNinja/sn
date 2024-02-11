package sn

import (
	"context"
	// "log"
	"os"
	"time"

	"cloud.google.com/go/datastore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

const subscriptionKind = "Subscription"

// type SubscriptionClient struct {
// 	*Client
// 	Messaging *messaging.Client
// }
//
// func NewSubscriptionClient(ctx context.Context, snClient *Client) *SubscriptionClient {
// 	return &SubscriptionClient{
// 		Client:    snClient,
// 		Messaging: newMsgClient(ctx),
// 	}
// }

type Subscription struct {
	Key       *datastore.Key `datastore:"__key__"`
	Tokens    []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Subscription) Load(ps []datastore.Property) error {
	var ps2 []datastore.Property
	for _, p := range ps {
		if p.Name != "Key" {
			ps2 = append(ps2, p)
		}
	}
	return datastore.LoadStruct(s, ps2)
}

func (s *Subscription) Save() ([]datastore.Property, error) {
	t := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = t
	}

	s.UpdatedAt = t
	return datastore.SaveStruct(s)
}

func (s *Subscription) LoadKey(k *datastore.Key) error {
	s.Key = k
	return nil
}

func (s *Subscription) Subscribe(token string) bool {
	if token == "" {
		return false
	}

	_, found := s.find(token)
	if found {
		return false
	}

	s.Tokens = append(s.Tokens, token)
	return true
}

func (s *Subscription) Unsubscribe(token string) bool {
	if token == "" {
		return false
	}

	i, found := s.find(token)
	if !found {
		return false
	}

	s.Tokens = append(s.Tokens[:i], s.Tokens[i+1:]...)
	return true
}

func (s *Subscription) find(token string) (int, bool) {
	if token == "" {
		return -1, false
	}
	for i, t := range s.Tokens {
		if t == token {
			return i, true
		}
	}
	return -1, false
}

func (s *Subscription) other(token string) []string {
	if token == "" {
		return s.Tokens
	}
	i, found := s.find(token)
	if !found {
		return s.Tokens
	}

	return append(s.Tokens[:i], s.Tokens[i+1:]...)
}

// func (cl *SubscriptionClient) Get(c *gin.Context) (*Subscription, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	id, err := GetID(c)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	s := newSubscription(id)
// 	err = cl.DS.Get(c, s.Key, s)
// 	if err != nil && err != datastore.ErrNoSuchEntity {
// 		return nil, err
// 	}
// 	return s, nil
// }
//
// func (cl *SubscriptionClient) Put(c *gin.Context, s *Subscription) (*datastore.Key, error) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	return cl.DS.Put(c, s.Key, s)
// }

func newSubscription(id int64) *Subscription {
	return &Subscription{Key: newSubscriptionKey(id)}
}

func newSubscriptionKey(id int64) *datastore.Key {
	return datastore.IDKey(subscriptionKind, id, RootKey(id))
}

// func GetSubscriptionToken(c *gin.Context) (string, error) {
//
// 	obj := struct {
// 		Token string `json:"token"`
// 	}{}
//
// 	err := c.ShouldBind(&obj)
// 	return obj.Token, err
// }

// func (cl *SubscriptionClient) SendRefreshMessages(c *gin.Context) error {
// 	cl.Log.Debugf("entering sendRefreshMessages")
// 	defer cl.Log.Debugf("exiting sendRefreshMessages")
//
// 	s, err := cl.Get(c)
// 	if err != nil {
// 		return err
// 	}
// 	cl.Log.Debugf("subscription: %#v", s)
//
// 	token, err := GetSubscriptionToken(c)
// 	if err != nil {
// 		return err
// 	}
// 	cl.Log.Debugf("token: %#v", token)
//
// 	tokens := s.other(token)
// 	cl.Log.Debugf("tokens: %#v", tokens)
// 	if len(tokens) > 0 {
// 		resp, err := cl.Messaging.SendMulticast(c, &messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Data:   map[string]string{"action": "refresh"},
// 		})
// 		if resp != nil {
// 			cl.Log.Debugf("batch response: %+v", resp)
// 			for _, r := range resp.Responses {
// 				cl.Log.Debugf("response: %+v", r)
// 			}
// 		}
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }
//
// func (cl *SubscriptionClient) SubscribeHandler(c *gin.Context) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	token, err := GetSubscriptionToken(c)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	s, err := cl.Get(c)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	changed := s.Subscribe(token)
// 	if changed {
// 		_, err := cl.Put(c, s)
// 		if err != nil {
// 			JErr(c, err)
// 			return
// 		}
// 	}
//
// 	c.JSON(http.StatusOK, gin.H{"subscribed": s.Tokens})
// }
//
// func (cl *SubscriptionClient) UnsubscribeHandler(c *gin.Context) {
// 	cl.Log.Debugf(msgEnter)
// 	defer cl.Log.Debugf(msgExit)
//
// 	token, err := GetSubscriptionToken(c)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	s, err := cl.Get(c)
// 	if err != nil {
// 		JErr(c, err)
// 		return
// 	}
//
// 	Debugf("original s: %+v", s)
// 	changed := s.Unsubscribe(token)
// 	if changed {
// 		Debugf("changed s: %+v", s)
// 		_, err := cl.Put(c, s)
// 		if err != nil {
// 			JErr(c, err)
// 			return
// 		}
// 	}
//
// 	c.JSON(http.StatusOK, gin.H{"subscribed": s.Tokens})
// }

const fbCreds = "FB_CREDS"

func newMsgClient(ctx context.Context) *messaging.Client {
	if IsProduction() {
		Debugf("production")
		app, err := firebase.NewApp(ctx, nil)
		if err != nil {
			Panicf("unable to create messaging client: %v", err)
			return nil
		}
		cl, err := app.Messaging(ctx)
		if err != nil {
			Panicf("unable to create messaging client: %v", err)
			return nil
		}
		return cl
	}
	Debugf("development")
	app, err := firebase.NewApp(
		ctx,
		nil,
		option.WithGRPCConnectionPool(50),
		option.WithCredentialsFile(os.Getenv(fbCreds)),
	)
	if err != nil {
		Panicf("unable to create messaging client: %v", err)
		return nil
	}
	cl, err := app.Messaging(ctx)
	if err != nil {
		Panicf("unable to create messaging client: %v", err)
		return nil
	}
	return cl
}
