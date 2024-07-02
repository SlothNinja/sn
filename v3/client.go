// SN implements services for SlothNinja Games Website
package sn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iterator"
)

type Client struct {
	Cache  *cache.Cache
	Router *gin.Engine
	options
}

type GameClient[GT any, G Gamer[GT]] struct {
	*Client
	FS *firestore.Client
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
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if IsProduction() {
		cl.Router.TrustedPlatform = gin.PlatformGoogleAppEngine
		return cl
	}

	// Is development
	cl.Router.SetTrustedProxies(nil)
	return cl
}

func NewGameClient[GT any, G Gamer[GT]](ctx context.Context, opts ...Option) *GameClient[GT, G] {
	cl := &GameClient[GT, G]{Client: NewClient(ctx, opts...)}

	var err error
	if cl.FS, err = firestore.NewClient(ctx, cl.projectID); err != nil {
		panic(fmt.Errorf("unable to connect to firestore database: %w", err))
	}
	return cl.addRoutes(cl.prefix)
}

// AddRoutes addes routing for game.
func (cl *Client) addRoutes() *Client {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/current", cl.cuHandler())

	// warmup
	cl.Router.GET("_ah/warmup", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	return cl
}

// AddRoutes addes routing for game.
func (cl *GameClient[GT, G]) addRoutes(prefix string) *GameClient[GT, G] {
	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", cl.newInvitationHandler())

	// // Create
	iGroup.PUT("/new", cl.createInvitationHandler())

	// // Drop
	iGroup.PUT("/drop/:id", cl.dropHandler())

	// // Accept
	iGroup.PUT("/accept/:id", cl.acceptHandler())

	// // Details
	// iGroup.GET("/details/:id", cl.detailsHandler())

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

func (cl *Client) Close() error {
	return nil
}

func (cl *GameClient[GT, G]) revCollectionRef(gid string) *firestore.CollectionRef {
	return cl.gameDocRef(gid).Collection(revKind)
}

func (cl *GameClient[GT, G]) revDocRef(gid string, rev int) *firestore.DocumentRef {
	return cl.revCollectionRef(gid).Doc(strconv.Itoa(rev))
}

func (cl *GameClient[GT, G]) gameDocRef(gid string) *firestore.DocumentRef {
	return cl.gameCollectionRef().Doc(gid)
}

func (cl *GameClient[GT, G]) gameCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(gameKind)
}

func (cl *GameClient[GT, G]) cachedDocRef(id string, rev int, uid UID, crev int) *firestore.DocumentRef {
	return cl.cachedCollectionRef(id, rev, uid).Doc(strconv.Itoa(crev))
}

func (cl *GameClient[GT, G]) cachedCollectionRef(id string, rev int, uid UID) *firestore.CollectionRef {
	return cl.revDocRef(id, rev).Collection(cacheKind).Doc(strconv.Itoa(int(uid))).Collection(revKind)
}

func (cl *GameClient[GT, G]) fullyCachedDocRef(id string, rev int, uid UID, crev int) *firestore.DocumentRef {
	// return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d-0", rev, uid))
	return cl.fullyCachedCollectionRef(id, rev, uid).Doc(strconv.Itoa(crev))
}

func (cl *GameClient[GT, G]) fullyCachedCollectionRef(id string, rev int, uid UID) *firestore.CollectionRef {
	return cl.revDocRef(id, rev).Collection(fullCachedKind).Doc(strconv.Itoa(int(uid))).Collection(revKind)
}

func (cl *GameClient[GT, G]) messageDocRef(gid string, mid string) *firestore.DocumentRef {
	return cl.messagesCollectionRef(gid).Doc(mid)
}

func (cl *GameClient[GT, G]) messagesCollectionRef(gid string) *firestore.CollectionRef {
	return cl.committedCollectionRef().Doc(gid).Collection(messagesKind)
}

func (cl *GameClient[GT, G]) committedDocRef(gid string, rev int) *firestore.DocumentRef {
	return cl.committedCollectionRef().Doc(fmt.Sprintf("%s-%d", gid, rev))
}

func (cl *GameClient[GT, G]) committedCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(gameKind)
}

func (cl *GameClient[GT, G]) indexDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection(indexKind).Doc(id)
}

func (cl *GameClient[GT, G]) viewCollectionRef(id string, rev int) *firestore.CollectionRef {
	return cl.revDocRef(id, rev).Collection(viewKind)
}

func (cl *GameClient[GT, G]) viewDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	return cl.viewCollectionRef(id, rev).Doc(strconv.Itoa(int(uid)))
}

func (cl *GameClient[GT, G]) Close() error {
	cl.FS.Close()
	return cl.Client.Close()
}

func (cl *GameClient[GT, G]) commit(ctx *gin.Context, g G, u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		return cl.txCommit(c, tx, g, u)
	})
}

func (cl *GameClient[GT, G]) txCommit(ctx context.Context, tx *firestore.Transaction, g G, u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	gc, err := cl.txGetCommitted(tx, g.getHeader().ID)
	if err != nil {
		return err
	}

	if g.getHeader().UpdatedAt != gc.getHeader().UpdatedAt {
		return fmt.Errorf("unexpected game change")
	}

	if err := cl.clearCached(ctx, g.getHeader().ID, g.getHeader().Undo.Committed, u.ID); err != nil {
		return err
	}

	g.getHeader().Undo.Commit()
	return cl.txSave(ctx, tx, g, u)
}

func (cl *GameClient[GT, G]) save(ctx *gin.Context, g G, u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		return cl.txSave(c, tx, g, u)
	})
}

func (cl *GameClient[GT, G]) txSave(ctx context.Context, tx *firestore.Transaction, g G, u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	g.getHeader().UpdatedAt = time.Now()

	if err := tx.Set(cl.revDocRef(g.getHeader().ID, g.getHeader().Undo.Current), g); err != nil {
		return err
	}

	if err := tx.Set(cl.indexDocRef(g.getHeader().ID), g.getHeader()); err != nil {
		return err
	}

	// By implementing Views interface, game may provide a customized view for each user.
	// Primarily used to ensure hidden game information not leaked to users via json objects
	// sent to browsers.
	uids, views := g.Views()
	for i, v := range views {
		if err := tx.Set(cl.viewDocRef(g.getHeader().ID, g.getHeader().Undo.Current, uids[i]), v); err != nil {
			return err
		}
	}
	return cl.clearCached(ctx, g.getHeader().ID, g.getHeader().Undo.Committed, u.ID)
}

func (cl *GameClient[GT, G]) clearCached(ctx context.Context, gid string, rev int, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	refs := cl.cachedCollectionRef(gid, rev, uid).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		_, err = ref.Delete(ctx)
		if err != nil {
			return err
		}
	}

	refs = cl.fullyCachedCollectionRef(gid, rev, uid).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		_, err = ref.Delete(ctx)
		if err != nil {
			return err
		}
	}

	return cl.deleteStack(ctx, gid, uid)
}

func docRefFor(ref *firestore.DocumentRef, uid UID) bool {
	ss := pie.Reverse(strings.Split(ref.ID, "-"))
	s := pie.Pop(&ss)
	if *s == "0" {
		s = pie.Pop(&ss)
	}
	return *s == fmt.Sprintf("%d", uid)
}

type detail struct {
	ID     int64
	ELO    int
	Played int64
	Won    int64
	WP     float32
}

// func (cl *GameClient[GT, G]) detailsHandler() gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		cl.Log.Debugf(msgEnter)
// 		defer cl.Log.Debugf(msgExit)
//
// 		inv, err := cl.getInvitation(ctx)
// 		if err != nil {
// 			JErr(ctx, err)
// 			return
// 		}
//
// 		cu, err := cl.RequireLogin(ctx)
// 		if err != nil {
// 			JErr(ctx, err)
// 			return
// 		}
//
// 		uids := make([]UID, len(inv.getHeader().UserIDS))
// 		copy(uids, inv.getHeader().UserIDS)
//
// 		if hasUID := pie.Any(inv.getHeader().UserIDS, func(id UID) bool { return id == cu.ID }); !hasUID {
// 			uids = append(uids, cu.ID)
// 		}
//
// 	}
// }
