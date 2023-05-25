package sn

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	gameKind      = "Game"
	cachedKind    = "Cached"
	viewKind      = "View"
	committedKind = "Committed"
)

type Game[G any] interface {
	Head() *Header
	IsCurrentPlayer(User) bool
	Views() ([]UID, []G)
	New() G
}

func (cl Client[G, I]) GameDocRef(id string, rev int) *firestore.DocumentRef {
	return cl.GameCollectionRef().Doc(fmt.Sprintf("%s-%d", id, rev))
}

func (cl Client[G, I]) GameCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(gameKind)
}

func (cl Client[G, I]) CachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	return cl.CachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d", rev, uid))
}

func (cl Client[G, I]) FullyCachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	return cl.CachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d-0", rev, uid))
}

func (cl Client[G, I]) CachedCollectionRef(id string) *firestore.CollectionRef {
	return cl.CommittedDocRef(id).Collection(cachedKind)
}

func (cl Client[G, I]) CommittedDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection(committedKind).Doc(id)
}

func (cl Client[G, I]) ViewDocRef(id string, uid UID) *firestore.DocumentRef {
	return cl.CommittedDocRef(id).Collection(viewKind).Doc(fmt.Sprintf("%d", uid))
}

// type Gamers []Gamer
//
//	type Gamer interface {
//		PhaseName() string
//		FromParams(*gin.Context, User, Type) error
//		ColorMapFor(User) ColorMap
//		headerer
//	}
//
//	type GetPlayerers interface {
//		GetPlayerers() Playerers
//	}
//
//	func GamesRoot(c *gin.Context) *datastore.Key {
//		return datastore.NameKey("Games", "root", nil)
//	}
//
//	func (h *Header) GetAcceptDialog() bool {
//		return h.Private()
//	}
//
//	func (h *Header) RandomTurnOrder() {
//		ps := h.gamer.(GetPlayerers).GetPlayerers()
//		for i := 0; i < h.NumPlayers; i++ {
//			ri := MyRand.Intn(h.NumPlayers)
//			ps[i], ps[ri] = ps[ri], ps[i]
//		}
//		h.SetCurrentPlayerers(ps[0])
//
//		h.OrderIDS = make([]PID, len(ps))
//		for i, p := range ps {
//			h.OrderIDS[i] = p.ID()
//		}
//	}
//
// Returns (true, nil) if game should be started
// func (h *Header) Accept(c *gin.Context, u User) (start bool, err error) {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	err = h.validateAccept(c, u)
// 	if err != nil {
// 		return false, err
// 	}
//
// 	h.AddUser(u)
// 	Debugf("h: %#v", h)
// 	if len(h.UserIDS) == h.NumPlayers {
// 		return true, nil
// 	}
// 	return false, nil
// }

// func (h *Header) validateAccept(c *gin.Context, u User) error {
// 	switch {
// 	case len(h.UserIDS) >= h.NumPlayers:
// 		return NewVError("Game already has the maximum number of players.")
// 	case h.HasUser(u):
// 		return NewVError("%s has already accepted this invitation.", u.Name)
// 	case h.Password != "" && c.PostForm("password") != h.Password:
// 		return NewVError("%s provided incorrect password for Game #%d: %s.", u.Name, h.ID, h.Title)
// 	}
// 	return nil
// }

// Returns (true, nil) if game should be started
func (h *Header) AcceptWith(u User, pwd, hash []byte) (bool, error) {
	err := h.validateAcceptWith(u, pwd, hash)
	if err != nil {
		return false, err
	}

	h.AddUser(u)
	if len(h.UserIDS) == int(h.NumPlayers) {
		return true, nil
	}
	return false, nil
}

func (h *Header) validateAcceptWith(u User, pwd, hash []byte) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)
	switch {
	case len(h.UserIDS) >= int(h.NumPlayers):
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case h.HasUser(u):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case len(hash) != 0:
		err := bcrypt.CompareHashAndPassword(hash, pwd)
		if err != nil {
			Warningf(err.Error())
			return fmt.Errorf("%s provided incorrect password for Game %s: %w",
				u.Name, h.Title, ErrValidation)
		}
		return nil
	default:
		return nil
	}
}

func (h *Header) Drop(u User) error {
	if err := h.validateDrop(u); err != nil {
		return err
	}

	h.RemoveUser(u)
	return nil
}

func (h Header) validateDrop(u User) error {
	Debugf("h: %#v", h)
	switch {
	case h.Status != Recruiting:
		return fmt.Errorf("game is no longer recruiting, thus %s can't drop: %w", u.Name, ErrValidation)
	case !h.HasUser(u):
		return fmt.Errorf("%s has not joined this game, thus %s can't drop: %w", u.Name, u.Name, ErrValidation)
	}
	return nil
}

const (
	gameKey = "Game"
	jsonKey = "JSON"
	hParam  = "hid"
)

func (cl Client[G, I]) ResetHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.GetCommitted(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.ClearCached(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

type stackFunc func(*Stack) bool

var NoUndo stackFunc = func(s *Stack) bool { return false }
var Undo stackFunc = (*Stack).Undo
var Redo stackFunc = (*Stack).Redo

func (cl Client[G, I]) UndoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		id := getID(ctx)
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		ref := cl.StackDocRef(id, cu.ID())
		snap, err := ref.Get(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		var stack Stack
		err = snap.DataTo(&stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !stack.Undo() {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		_, err = ref.Set(ctx, &stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, I]) RedoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		id := getID(ctx)
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		ref := cl.StackDocRef(id, cu.ID())
		snap, err := ref.Get(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		var stack Stack
		err = snap.DataTo(&stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !stack.Redo() {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		_, err = ref.Set(ctx, &stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, I]) RollbackHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		err = ValidateAdmin(cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		obj := struct {
			Rev int
		}{}

		err = ctx.ShouldBind(&obj)
		if err != nil {
			JErr(ctx, err)
			return
		}

		rev := obj.Rev - 1
		g, err := cl.getRev(ctx, rev)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.Save(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, I]) RollforwardHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		err = ValidateAdmin(cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		obj := struct {
			Rev int
		}{}

		err = ctx.ShouldBind(&obj)
		if err != nil {
			JErr(ctx, err)
			return
		}

		rev := obj.Rev + 1
		g, err := cl.getRev(ctx, rev)
		if status.Code(err) == codes.NotFound {
			JErr(ctx, fmt.Errorf("cannot roll forward any further: %w", ErrValidation))
			return
		}
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.Save(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func ValidateAdmin(cu User) error {
	if cu.IsAdmin() {
		return nil
	}
	return errors.New("not admin")
}

func (cl Client[G, I]) getRev(ctx *gin.Context, rev int) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var g G
	g = g.New()

	id := getID(ctx)
	snap, err := cl.GameDocRef(id, rev).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}
	g.Head().ID = id
	return g, nil
}

func (cl Client[G, I]) GetGame(ctx *gin.Context, cu User, action ...stackFunc) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g, err := cl.GetCommitted(ctx)
	if err != nil {
		return g, err
	}

	// if current user not current player, then simply return g which is committed game
	if !g.IsCurrentPlayer(cu) {
		return g, nil
	}

	undo, err := cl.GetStack(ctx, cu.ID())
	// if undo stack not found, then simply return g which is committed game
	if status.Code(err) == codes.NotFound {
		return g, nil
	}
	if err != nil {
		return g, err
	}

	if len(action) == 1 {
		action[0](&undo)
	}
	if undo.Current == undo.Committed {
		g.Head().Undo = undo
		return g, nil
	}

	g, err = cl.getCached(ctx, undo.Current, cu.ID())
	if err != nil {
		return g, err
	}
	g.Head().Undo = undo
	return g, nil
}

func (cl Client[G, I]) getCached(ctx *gin.Context, rev int, uid UID) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var g G
	g = g.New()
	id := getID(ctx)
	snap, err := cl.FullyCachedDocRef(id, rev, uid).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}

	g.Head().ID = id
	return g, nil
}

const (
	cachedRootKind = "CachedRoot"
)

func (cl Client[G, I]) GetCommitted(ctx *gin.Context) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var g G
	g = g.New()
	id := getID(ctx)
	snap, err := cl.CommittedDocRef(id).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}

	g.Head().ID = id
	return g, nil
}

func (cl Client[G, I]) PutCached(ctx *gin.Context, g G, u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(cl.FullyCachedDocRef(g.Head().ID, g.Head().Rev(), u.ID()), g); err != nil {
			return err
		}

		// uids, views := g.Views()
		// for i, v := range views {
		// 	if err := tx.Set(cl.ViewDocRef(g.Head().ID, uids[i]), v); err != nil {
		// 		return err
		// 	}
		// }

		if err := tx.Set(cl.CachedDocRef(g.Head().ID, g.Head().Rev(), u.ID()), viewFor(g, u.ID())); err != nil {
			return err
		}

		return tx.Set(cl.StackDocRef(g.Head().ID, u.ID()), g.Head().Undo)
	})
}

func viewFor[G Game[G]](g G, uid1 UID) G {
	uids, views := g.Views()
	return views[pie.FindFirstUsing(uids, func(uid2 UID) bool { return uid1 == uid2 })]
}

func (cl Client[G, I]) CachedHandler(action func(G, *gin.Context, User) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.Current(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		g, err := cl.GetGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := action(g, ctx, cu); err != nil {
			JErr(ctx, err)
			return
		}
		g.Head().Undo.Update()

		if err := cl.PutCached(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}
