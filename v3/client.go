// Package sn implements services for SlothNinja Games Website
package sn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
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

// GameClient provides a client for a game service
type GameClient[GT any, G Gamer[GT]] struct {
	*Client
	FS  *firestore.Client
	FCM *messaging.Client
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

// NewGameClient returns a new game service client
func NewGameClient[GT any, G Gamer[GT]](ctx context.Context, opts ...Option) *GameClient[GT, G] {
	cl := &GameClient[GT, G]{Client: NewClient(ctx, opts...)}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cl.projectID})
	if err != nil {
		panic(fmt.Errorf("unable to connect to create firebase app: %w", err))
	}
	if cl.FS, err = app.Firestore(ctx); err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}

	if cl.FCM, err = app.Messaging(ctx); err != nil {
		panic(fmt.Errorf("unable to connect to firebase messaging: %w", err))
	}
	return cl.addRoutes(cl.prefix)
}

// AddRoutes adds routing for game.
func (cl *Client) addRoutes() *Client {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/current", cl.cuHandler())

	/////////////////////////////////////////////
	// Update God Mode
	cl.Router.PUT(cl.prefix+"/user/update-god-mode", cl.updateGodModeHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	return cl
}

// AddRoutes adds routing for game.
func (cl *GameClient[GT, G]) addRoutes(prefix string) *GameClient[GT, G] {
	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", cl.newInvitationHandler())

	// Create
	iGroup.PUT("/new", cl.createInvitationHandler())

	// Drop
	iGroup.PUT("/drop/:id", cl.dropHandler())

	// Accept
	iGroup.PUT("/accept/:id", cl.acceptHandler())

	// Details
	iGroup.GET("/details/:id", cl.detailsHandler())

	// Abort
	iGroup.PUT("abort/:id", cl.abortHandler())

	/////////////////////////////////////////////
	// Game Group
	gGroup := cl.Router.Group(prefix + "/game")

	// Reset
	gGroup.PUT("reset/:id", cl.resetHandler())

	// Undo
	gGroup.PUT("undo/:id", cl.undoHandler())

	// Redo
	gGroup.PUT("redo/:id", cl.redoHandler())

	// Rollback
	gGroup.PUT("rollback/:id", cl.rollbackHandler())

	// Rollforward
	gGroup.PUT("rollforward/:id", cl.rollforwardHandler())

	// Abandon
	gGroup.PUT("abandon/:id", cl.abandonHandler())

	/////////////////////////////////////////////
	// Message Log
	msg := cl.Router.Group(prefix + "/mlog")

	// Update Read
	msg.PUT("/updateRead/:id", cl.updateReadHandler())

	// Add
	msg.PUT("/add/:id", cl.addMessageHandler())

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

func (cl *GameClient[GT, G]) commit(ctx *gin.Context, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.header().UpdatedAt = timestamppb.Now()

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txCommit(tx, g, cu)
	})
}

func (cl *GameClient[GT, G]) txCommit(tx *firestore.Transaction, g G, cu *User) error {
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

	if err := cl.txDeleteCachedRevs(tx, g, cu); err != nil {
		return err
	}

	return cl.txSave(tx, g, cu)
}

func (cl *GameClient[GT, G]) save(ctx *gin.Context, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txSave(tx, g, cu)
	})
}

func (cl *GameClient[GT, G]) txSave(tx *firestore.Transaction, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if err := cl.txUpdateViews(tx, g, cu); err != nil {
		return err
	}

	if err := cl.txUpdateRev(tx, g); err != nil {
		return err
	}

	if err := cl.txSaveStacks(tx, g, cu); err != nil {
		return err
	}

	return cl.txUpdateIndex(tx, g)
}

func (cl *GameClient[GT, G]) txUpdateIndex(tx *firestore.Transaction, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.indexDocRef(g.id()), g.getIndex())
}

func (cl *GameClient[GT, G]) txUpdateRev(tx *firestore.Transaction, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.revDocRef(g.id(), g.stack().Current), g)
}

func (cl *GameClient[GT, G]) updateViews(ctx *gin.Context, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txUpdateViews(tx, g, cu)
	})
}

func (cl *GameClient[GT, G]) txUpdateViews(tx *firestore.Transaction, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if cu == nil {
		return ErrUserNil
	}

	// By implementing Views interface, game may provide a customized view for each user.
	// Primarily used to ensure hidden game information not leaked to users via json objects
	// sent to browsers.
	uids, views := g.Views()
	if cu.GodMode && !g.isPlayer(cu) {
		uids = append(uids, cu.ID)
		views = append(views, g.ViewFor(cu.ID))
	}

	for i, v := range views {
		if err := tx.Set(cl.viewDocRef(g.id(), uids[i]), v); err != nil {
			return err
		}
	}
	return nil
}

// attempts to remove revs passed current save
func (cl *GameClient[GT, G]) txDeleteCachedRevs(tx *firestore.Transaction, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var err error
	for rev := g.stack().Current + 1; rev <= g.stack().end(); rev++ {
		err = errors.Join(err, cl.txDeleteCachedRev(tx, g, cu, rev))
	}
	if err != nil {
		return err
	}

	g.stack().trunc()
	return nil
}

func (cl *GameClient[GT, G]) txDeleteCachedRev(tx *firestore.Transaction, g G, cu *User, rev Rev) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if cu == nil {
		Warnf("user nil, cached document not deleted")
		return nil
	}
	return tx.Delete(cl.cachedDocRef(g.id(), cu.ID, rev))
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

func (cl *GameClient[GT, G]) txSaveStacks(tx *firestore.Transaction, g G, cu *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if cu == nil {
		return ErrUserNil
	}

	var err error
	for _, uid := range g.header().UserIDS {
		err = errors.Join(err, cl.txSaveStack(tx, g, uid))
	}

	if cu.GodMode && !g.isPlayer(cu) {
		err = errors.Join(err, cl.txSaveStack(tx, g, cu.ID))
	}
	return err
}
