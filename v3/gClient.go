package sn

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GameClient provides a client for a game service
type GameClient[GT any, G Gamer[GT]] struct {
	*Client
	Auth *auth.Client
	FS   *firestore.Client
	FCM  *messaging.Client
}

// NewGameClient returns a new game service client
func NewGameClient[GT any, G Gamer[GT]](ctx context.Context, opts ...Option) (*GameClient[GT, G], error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)
	cl := &GameClient[GT, G]{Client: NewClient(ctx, opts...)}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cl.projectID})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to create firebase app: %w", err)
	}

	cl.Auth, err = app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %w", err)
	}

	if cl.FS, err = app.Firestore(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to firestore database: %w", err)
	}

	if cl.FCM, err = app.Messaging(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to firebase messaging: %w", err)
	}
	return cl.addRoutes(cl.prefix), nil
}

func (cl *GameClient[GT, G]) getGame(ctx *gin.Context, u *User) (G, UID, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	uid := u.ID
	if u.GodMode {
		var err error
		if uid, err = getUID(ctx); err != nil {
			return nil, 0, err
		}
	}

	gid := getID(ctx)
	stack, err := cl.getStack(ctx, gid, uid)
	if err != nil {
		return nil, 0, err
	}

	g, err := cl.getGameWithStack(ctx, gid, uid, stack)
	return g, uid, err
}

func (cl *GameClient[GT, G]) getGameWithStack(ctx *gin.Context, gid string, uid UID, stack *Stack) (g G, err error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if stack.currentIsCached() {
		g, err = cl.getCached(ctx, gid, uid, stack.Current)
	} else {
		g, err = cl.getRev(ctx, gid, stack.Current)
	}

	if err != nil {
		return nil, err
	}
	g.setStack(stack)
	return g, nil
}

func (cl *GameClient[GT, G]) getRev(ctx *gin.Context, gid string, rev Rev) (G, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	snap, err := cl.revDocRef(gid, rev).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.setID(gid)
	return g, nil
}

func (cl *GameClient[GT, G]) getCached(ctx *gin.Context, gid string, uid UID, rev Rev) (G, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	snap, err := cl.cachedDocRef(gid, uid, rev).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.setID(gid)
	return g, nil
}

func (cl *GameClient[GT, G]) getIndex(ctx *gin.Context, id string) (*index, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	snap, err := cl.indexDocRef(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	i := new(index)
	if err := snap.DataTo(i); err != nil {
		return nil, err
	}

	i.setID(id)
	return i, nil
}

func (cl *GameClient[GT, G]) txGetIndex(tx *firestore.Transaction, id string) (*index, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	snap, err := tx.Get(cl.indexDocRef(id))
	if err != nil {
		return nil, err
	}

	i := new(index)
	if err := snap.DataTo(i); err != nil {
		return nil, err
	}

	i.setID(id)
	return i, nil
}

func (cl *GameClient[GT, G]) cacheRev(ctx *gin.Context, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if err := cl.txUpdateViews(tx, g, uid); err != nil {
			return err
		}

		if err := cl.txSaveStack(tx, g, uid); err != nil {
			return err
		}

		if err := cl.txSaveStack(tx, g, uid); err != nil {
			return err
		}

		return cl.txCacheRev(tx, g, uid)
	})
}

func (cl *GameClient[GT, G]) txCacheRev(tx *firestore.Transaction, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return tx.Set(cl.cachedDocRef(g.id(), uid, g.stack().Current), g)
}

func (cl *GameClient[GT, G]) endGame(ctx *gin.Context, g G, uid UID) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	places, placesSMap := g.setFinishOrder(g.Compare)
	g.header().Status = Completed
	g.header().EndedAt = timestamppb.Now()
	g.header().Phase = "Game Over"
	g.header().Places = placesSMap

	stats, err := cl.getUStats(ctx, g.header().UserIDS...)
	if err != nil {
		return err
	}
	stats = g.updateUStats(stats, g.playerStats(), g.playerUIDS())

	oldElos, newElos, err := cl.updateElo(ctx, g.header().users(), places)
	if err != nil {
		return err
	}

	rs := g.getResults(oldElos, newElos)
	g.newEntry("game-results", H{"Results": rs})

	if err := cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if err := cl.txCommit(tx, g, uid); err != nil {
			return err
		}

		if err := cl.txSaveUStats(tx, stats); err != nil {
			return err
		}

		return cl.txSaveElos(tx, newElos)
	}); err != nil {
		return err
	}

	if err := g.sendEndGameNotifications(rs); err != nil {
		// log but otherwise ignore send errors
		Warnf("%v", err.Error())
	}
	return nil
}
