// Package sn implements services for SlothNinja Games Website
package sn

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client provides a client for a service
type Client struct {
	Cache  *cache.Cache
	Router *gin.Engine
	options
}

func defaultClient() *Client {
	cl := new(Client)
	cl.projectID = getProjectID()
	cl.url = getURL()
	cl.frontEndURL = getFrontEndURL()
	cl.backEndURL = getBackEndURL()
	cl.port = getPort()
	cl.backEndPort = getBackEndPort()
	cl.frontEndPort = getFrontEndPort()
	cl.secretsProjectID = getSecretsProjectID()
	cl.secretsDSURL = getSecretsDSURL()
	cl.prefix = getPrefix()
	cl.home = getHome()
	return cl
}

// NewClient returns a new service client
func NewClient(ctx context.Context, opts ...Option) *Client {
	cl := defaultClient()

	// Apply all functional options
	for _, opt := range opts {
		cl = opt(cl)
	}

	// Initalize
	return cl.initCache().
		initRouter().
		initSession(ctx).
		initEnvironment().
		addRoutes()
}

func (cl *Client) initCache() *Client {
	cl.Cache = cache.New(30*time.Minute, 10*time.Minute)
	return cl
}

func (cl *Client) initRouter() *Client {
	cl.Router = gin.Default()
	return cl
}

func (cl *Client) initEnvironment() *Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if IsProduction() {
		cl.Router.TrustedPlatform = gin.PlatformGoogleAppEngine
		return cl
	}

	// Is development
	cl.Router.SetTrustedProxies(nil)
	return cl
}

// Close closes client
func (cl *Client) Close() error {
	return nil
}

func (cl *GameClient[GT, G]) revCollectionRef(gid string) *firestore.CollectionRef {
	return cl.gameDocRef(gid).Collection("Rev")
}

func (cl *GameClient[GT, G]) revDocRef(gid string, rev Rev) *firestore.DocumentRef {
	return cl.revCollectionRef(gid).Doc(rev.toString())
}

func (cl *GameClient[GT, G]) gameDocRef(gid string) *firestore.DocumentRef {
	return cl.gameCollectionRef().Doc(gid)
}

func (cl *GameClient[GT, G]) gameCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection("Game")
}

func (cl *GameClient[GT, G]) cachedDocRef(id string, uid UID, rev Rev) *firestore.DocumentRef {
	return cl.cachedCollectionRef(id, uid).Doc(rev.toString())
}

func (cl *GameClient[GT, G]) cachedCollectionRef(gid string, uid UID) *firestore.CollectionRef {
	return cl.gameDocRef(gid).Collection("CacheFor").Doc(uid.toString()).Collection("Rev")
}

func (cl *GameClient[GT, G]) messageDocRef(gid string, mid string) *firestore.DocumentRef {
	return cl.messagesCollectionRef(gid).Doc(mid)
}

func (cl *GameClient[GT, G]) messagesCollectionRef(gid string) *firestore.CollectionRef {
	return cl.gameCollectionRef().Doc(gid).Collection("Messages")
}

func (cl *GameClient[GT, G]) indexDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection("Index").Doc(id)
}

func (cl *GameClient[GT, G]) viewDocRef(gid string, uid UID) *firestore.DocumentRef {
	return cl.gameDocRef(gid).Collection("For").Doc(uid.toString())
}

// Close closes the game service client
func (cl *GameClient[GT, G]) Close() error {
	cl.FS.Close()
	return cl.Client.Close()
}

func (cl *GameClient[GT, G]) commit(ctx *gin.Context, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.header().UpdatedAt = timestamppb.Now()

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txCommit(tx, g, uid)
	})
}

func (cl *GameClient[GT, G]) txCommit(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.stack().commit()

	index, err := cl.txGetIndex(tx, g.id())
	if err != nil {
		return err
	}

	if index.Rev+1 != g.stack().Committed {
		return fmt.Errorf("unexpected game change")
	}

	if err := cl.txDeleteCachedRevs(tx, g, uid); err != nil {
		return err
	}

	return cl.txSave(tx, g, uid)
}

func (cl *GameClient[GT, G]) save(ctx *gin.Context, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txSave(tx, g, uid)
	})
}

func (cl *GameClient[GT, G]) txSave(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if err := cl.txUpdateViews(tx, g, uid); err != nil {
		return err
	}

	if err := cl.txUpdateRev(tx, g); err != nil {
		return err
	}

	if err := cl.txSaveStacks(tx, g, uid); err != nil {
		return err
	}

	return cl.txUpdateIndex(tx, g)
}

func (cl *GameClient[GT, G]) txUpdateIndex(tx *firestore.Transaction, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.indexDocRef(g.id()), g.toIndex())
}

func (cl *GameClient[GT, G]) txUpdateRev(tx *firestore.Transaction, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.revDocRef(g.id(), g.stack().Current), g)
}

func (cl *GameClient[GT, G]) updateViews(ctx *gin.Context, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txUpdateViews(tx, g, uid)
	})
}

func (cl *GameClient[GT, G]) txUpdateViews(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	// By implementing Views interface, game may provide a customized view for each user.
	// Primarily used to ensure hidden game information not leaked to users via json objects
	// sent to browsers.
	uids, views := g.Views()
	if !slices.Contains(uids, uid) {
		uids = append(uids, uid)
		views = append(views, g.ViewFor(uid))
	}

	for i, v := range views {
		if err := tx.Set(cl.viewDocRef(g.id(), uids[i]), v); err != nil {
			return err
		}
	}
	return nil
}

// attempts to remove revs passed current save
func (cl *GameClient[GT, G]) txDeleteCachedRevs(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var err error
	for rev := g.stack().Current + 1; rev <= g.stack().end(); rev++ {
		err = errors.Join(err, cl.txDeleteCachedRev(tx, g, uid, rev))
	}
	if err != nil {
		return err
	}

	g.stack().trunc()
	return nil
}

func (cl *GameClient[GT, G]) txDeleteCachedRev(tx *firestore.Transaction, g G, uid UID, rev Rev) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Delete(cl.cachedDocRef(g.id(), uid, rev))
}

// By implementing Views interface, game may provide a customized view for each user.
// Primarily used to ensure hidden game information not leaked to users via json objects
// sent to browsers.
func (cl *GameClient[GT, G]) txViews(tx *firestore.Transaction, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	uids, views := g.Views()
	for i, v := range views {
		if err := tx.Set(cl.viewDocRef(g.id(), uids[i]), v); err != nil {
			return err
		}
	}
	return nil
}

func (cl *GameClient[GT, G]) txSaveStack(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.stackDocRef(g.id(), uid), g.stack())
}

func (cl *GameClient[GT, G]) txSaveStacks(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var err error
	for _, uid := range g.header().UserIDS {
		err = errors.Join(err, cl.txSaveStack(tx, g, uid))
	}

	return errors.Join(err, cl.txSaveStack(tx, g, 0))
}
