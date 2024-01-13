package sn

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/barkimedes/go-deepcopy"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"github.com/mailjet/mailjet-apiv3-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	gameKind      = "Game"
	cachedKind    = "Cached"
	viewKind      = "View"
	committedKind = "Committed"
	indexKind     = "Index"
	readKind      = "Read"
	mlogKind      = "MLog"
	messagesKind  = "Messages"
)

type Game[S, PT any, P Playerer[PT]] struct {
	Header  Header
	Players Players[PT, P]
	Log     glog
	State   S
}

type Gamer[G any] interface {
	SetCurrentPlayerers
	Viewer[G]
	Starter
	Ptr[G]

	getHeader() *Header
	isCurrentPlayer(User) bool
	playerStats() []*Stats
	statsFor(PID) *Stats
	playerUIDS() []UID
	sendEndGameNotifications(*gin.Context, []Elo, []Elo) error
	setFinishOrder() PlacesMap
	updateUStats([]UStat, []*Stats, []UID) []UStat
	reset(PID)
}

type Viewer[T any] interface {
	Views() ([]UID, []*T)
}

type Starter interface {
	Start(Header) PID
}

type SetCurrentPlayerers interface {
	SetCurrentPlayers(...PID)
}

func (g *Game[S, T, P]) getHeader() *Header {
	return &(g.Header)
}

// type Gamer[G any, P Playerer] interface {
// 	Views() ([]UID, []G)
// 	SetCurrentPlayers(...P)
// 	Start(*Header) P
//
// 	setFinishOrder() PlacesMap
// 	getHeader() *Header
// 	playerStats() []*Stats
// 	playerUIDS() []UID
// 	sendEndGameNotifications(*gin.Context, []Elo, []Elo) error
// 	updateUStats([]UStat, []*Stats, []UID) []UStat
// 	isCurrentPlayer(User) bool
// }

// type Viewer[S, T any, P Playerer[T]] interface {
// 	Views() ([]UID, []*Game[S, T, P])
// }

func (cl *GameClient[GT, G]) gameDocRef(id string, rev int) *firestore.DocumentRef {
	return cl.gameCollectionRef().Doc(fmt.Sprintf("%s-%d", id, rev))
}

func (cl *GameClient[GT, G]) gameCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(gameKind)
}

func (cl *GameClient[GT, G]) cachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	// return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d", rev, uid))
	return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d", uid)).Collection("Current").Doc(fmt.Sprintf("%d", rev))
}

func (cl *GameClient[GT, G]) fullyCachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	// return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d-0", rev, uid))
	return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d", uid)).Collection("Current").Doc(fmt.Sprintf("%d", rev)).Collection("Full").Doc("0")
}

func (cl *GameClient[GT, G]) cachedCollectionRef(id string) *firestore.CollectionRef {
	return cl.committedDocRef(id).Collection(cachedKind)
}

func (cl *GameClient[GT, G]) messageDocRef(gid, mid string) *firestore.DocumentRef {
	return cl.messagesCollectionRef(gid).Doc(mid)
}

func (cl *GameClient[GT, G]) messagesCollectionRef(id string) *firestore.CollectionRef {
	return cl.committedDocRef(id).Collection(messagesKind)
}

func (cl *GameClient[GT, G]) committedDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection(committedKind).Doc(id)
}

func (cl *GameClient[GT, G]) indexDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection(indexKind).Doc(id)
}

func (cl *GameClient[GT, G]) viewDocRef(id string, uid UID) *firestore.DocumentRef {
	return cl.committedDocRef(id).Collection(viewKind).Doc(fmt.Sprintf("%d", uid))
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
// 	Debug(msgEnter)
// 	defer Debug(msgExit)
//
// 	err = h.validateAccept(c, u)
// 	if err != nil {
// 		return false, err
// 	}
//
// 	h.AddUser(u)
// 	Debug("h: %#v", h)
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
	Debugf(msgEnter)
	defer Debugf(msgExit)

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
			Debugf(err.Error())
			return fmt.Errorf("%s provided incorrect password for Game %s: %w",
				u.Name, h.Title, ErrValidation)
		}
		return nil
	default:
		return nil
	}
}

func (h *Header) Drop(u User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if err := h.validateDrop(u); err != nil {
		return err
	}

	h.RemoveUser(u)
	return nil
}

func (h Header) validateDrop(u User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

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

func (cl *GameClient[GT, G]) resetHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getCommitted(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.clearCached(ctx, g, cu)
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

func (cl *GameClient[GT, G]) undoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		id := getID(ctx)
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		ref := cl.StackDocRef(id, cu.ID)
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

func (cl *GameClient[GT, G]) redoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		id := getID(ctx)
		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		ref := cl.StackDocRef(id, cu.ID)
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

func (cl *GameClient[GT, G]) abandonHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireAdmin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getCommitted(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.getHeader().Status = Abandoned
		if err := cl.save(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) rollbackHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cuid, err := cl.RequireAdmin(ctx)
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

		g, err := cl.getRev(ctx, obj.Rev-1)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.save(ctx, g, cuid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) rollforwardHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cuid, err := cl.RequireAdmin(ctx)
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

		g, err := cl.getRev(ctx, obj.Rev+1)
		if status.Code(err) == codes.NotFound {
			JErr(ctx, fmt.Errorf("cannot roll forward any further: %w", ErrValidation))
			return
		}
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.save(ctx, g, cuid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) getRev(ctx *gin.Context, rev int) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	id := getID(ctx)
	snap, err := cl.gameDocRef(id, rev).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}
	g.getHeader().ID = id
	return g, nil
}

func (cl *GameClient[GT, G]) getGame(ctx *gin.Context, cu User, action ...stackFunc) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g, err := cl.getCommitted(ctx)
	if err != nil {
		return g, err
	}

	// if current user not current player, then simply return g which is committed game
	if !g.isCurrentPlayer(cu) {
		return g, nil
	}

	undo, err := cl.getStack(ctx, cu.ID)
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
		g.getHeader().Undo = undo
		return g, nil
	}

	g, err = cl.getCached(ctx, undo.Current, cu.ID)
	if err != nil {
		return g, err
	}
	g.getHeader().Undo = undo
	return g, nil
}

func (cl *GameClient[GT, G]) getCached(ctx *gin.Context, rev int, uid UID) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	id := getID(ctx)
	snap, err := cl.fullyCachedDocRef(id, rev, uid).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.getHeader().ID = id
	return g, nil
}

const (
	cachedRootKind = "CachedRoot"
)

func (cl *GameClient[GT, G]) getCommitted(ctx *gin.Context) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	id := getID(ctx)
	snap, err := cl.committedDocRef(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.getHeader().ID = id
	return g, nil
}

// func newGame[G any]() G {
// 	var g G
// 	return reflect.New(reflect.TypeOf(g).Elem()).Interface().(G)
// }

func (cl *GameClient[GT, G]) putCached(ctx *gin.Context, g G, u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(cl.fullyCachedDocRef(g.getHeader().ID, g.getHeader().Rev(), u.ID), g); err != nil {
			return err
		}

		if err := tx.Set(cl.cachedDocRef(g.getHeader().ID, g.getHeader().Rev(), u.ID), viewFor(g, u.ID)); err != nil {
			return err
		}

		return tx.Set(cl.StackDocRef(g.getHeader().ID, u.ID), g.getHeader().Undo)
	})
}

func viewFor[T any](g Viewer[T], uid1 UID) *T {
	uids, views := g.Views()
	return views[pie.FindFirstUsing(uids, func(uid2 UID) bool { return uid1 == uid2 })]
}

type CachedActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, User) error

func (cl *GameClient[GT, G]) CachedHandler(action CachedActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err = action(g, ctx, cu); err != nil {
			JErr(ctx, err)
			return
		}
		g.getHeader().Undo.Update()

		if err := cl.putCached(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

type FinishTurnActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, User) (PID, PID, error)

func (cl *GameClient[GT, G]) FinishTurnHandler(action FinishTurnActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			cl.Log.Warningf(err.Error())
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cpid, npid, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cl.Log.Debugf("cpid: %v npid: %v, err: %v", cpid, npid, err)

		cpStats := g.statsFor(cpid)
		cpStats.Moves++
		cpStats.Think += time.Since(g.getHeader().UpdatedAt)

		if npid == NoPID {
			cl.endGame(ctx, g, cu)
			return
		}
		g.reset(npid)
		g.SetCurrentPlayers(npid)

		if err := cl.commit(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (g *Game[S, T, P]) playerUIDS() []UID {
	return pie.Map(g.Players, func(p P) UID { return g.Header.UserIDS[p.getPID().toUIndex()] })
}

func (g *Game[S, T, P]) UIDForPID(pid PID) UID {
	Debugf("pid: %v", pid)
	return g.Header.UserIDS[pid.toUIndex()]
}

func (g *Game[S, T, P]) PlayerByPID(pid PID) P {
	const notFound = -1
	index := pie.FindFirstUsing(g.Players, func(p P) bool { return p.getPID() == pid })
	if index == notFound {
		var zerop P
		return zerop
	}
	return g.Players[index]
}

func (g *Game[S, T, P]) PlayerByUID(uid UID) P {
	index := UIndex(pie.FindFirstUsing(g.Header.UserIDS, func(id UID) bool { return id == uid }))
	return g.PlayerByPID(index.toPID())
}

// IndexForPlayer returns the index for the player and bool indicating whether player found.
// if not found, returns -1
func (g *Game[S, T, P]) IndexForPlayer(p1 P) int {
	return pie.FindFirstUsing(g.Players, func(p2 P) bool { return p1.getPID() == p2.getPID() })
}

// treats Players as a circular buffer, thus permitting indices larger than length and indices less than 0
func (g *Game[S, T, P]) PlayerByIndex(i int) P {
	l := len(g.Players)
	if l < 1 {
		var zerop P
		return zerop
	}

	r := i % l
	if r < 0 {
		return g.Players[l+r]
	}
	return g.Players[r]
}

func (ps Players[T, P]) PIDS() []PID {
	return pie.Map(ps, func(p P) PID { return p.getPID() })
}

func (ps Players[T, P]) Randomize() Players[T, P] {
	return pie.Shuffle(ps, myRandomSource)
}

func (g *Game[S, T, P]) RandomizePlayers() {
	// g.Players = pie.Shuffle(g.Players, myRandomSource)
	g.Players = g.Players.Randomize()
	g.UpdateOrder()
}

// CurrentPlayers returns the players whose turn it is.
func (g *Game[S, T, P]) CurrentPlayers() []P {
	return pie.Map(g.Header.CPIDS, func(pid PID) P { return g.PlayerByPID(pid) })
}

// currentPlayer returns the player whose turn it is.
func (g *Game[S, T, P]) CurrentPlayer() P {
	return pie.First(g.CurrentPlayers())
}

// Returns player asssociated with user if such player is current player
// Otherwise, return nil
func (g *Game[S, T, P]) CurrentPlayerFor(u User) P {
	i := g.Header.indexFor(u.ID)
	if i == -1 {
		var zerop P
		return zerop
	}

	return g.PlayerByPID(i.toPID())
}

func (g *Game[S, T, P]) SetCurrentPlayers(ps ...PID) {
	if len(ps) == 0 {
		g.Header.CPIDS = nil
	}
	g.Header.CPIDS = make([]PID, len(ps), len(ps))
	copy(g.Header.CPIDS, ps)
}

func (g *Game[S, T, P]) isCurrentPlayer(cu User) bool {
	return g.CurrentPlayerFor(cu).getPID() != NoPID
}

func (g *Game[S, T, P]) copyPlayers() []P {
	return deepcopy.MustAnything(g.Players).([]P)
	// return pie.Map(g.Players, func(p P) P { return p.Copy().(P) })
}

func (g *Game[S, T, P]) ValidatePlayerAction(cu User) (P, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cp, err := g.ValidateCurrentPlayer(cu)
	switch {
	case err != nil:
		return cp, err
	case cp.getPerformedAction():
		return cp, fmt.Errorf("current player already performed action: %w", ErrValidation)
	default:
		return cp, nil
	}
}

func (g *Game[S, T, P]) ValidateCurrentPlayer(cu User) (P, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cp := g.CurrentPlayerFor(cu)
	if cp.getPID() == NoPID {
		return cp, ErrPlayerNotFound
	}
	return cp, nil
}

func (g *Game[S, T, P]) playerStats() []*Stats {
	return pie.Map(g.Players, func(p P) *Stats { return p.getStats() })
}

func (g *Game[S, T, P]) statsFor(pid PID) *Stats {
	var zerop P
	p := g.PlayerByPID(pid)
	if p == zerop {
		return nil
	}
	return p.getStats()
}

func (g *Game[S, T, P]) reset(pid PID) {
	var zerop P
	p := g.PlayerByPID(pid)
	if p != zerop {
		p.reset()
	}
}

func (g *Game[S, T, P]) UIDSForPIDS(pids []PID) []UID {
	return pie.Map(pids, func(pid PID) UID { return g.UIDForPID(pid) })
}

// func (g Game[P]) UserKeyFor(pid PID) *datastore.Key {
// 	return NewUser(g.UIDForPID(pid)).Key
// }

// cp specifies the current
// return player after cp that satisfies all tests ts
// if tests ts is empty, return player after cp
func (g *Game[S, T, P]) NextPlayer(cp P, ts ...func(P) bool) P {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	start := g.IndexForPlayer(cp) + 1
	stop := start + g.Header.NumPlayers

	// g.playerByIndex uses the players slice as-if it were circular buffer
	// start is one index after cp
	// stop is num players later, thus one pass through circular buffer
	for i := start; i <= stop; i++ {
		np := g.PlayerByIndex(i)
		if pie.All(ts, func(t func(P) bool) bool { return t(np) }) {
			return np
		}
	}
	var zerop P
	return zerop
}

const maxPlayers = 6

func (cl *GameClient[GT, G]) endGame(ctx *gin.Context, g G, cu User) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	places := g.setFinishOrder()
	g.getHeader().Status = Completed
	g.getHeader().EndedAt = updateTime()

	stats, err := cl.GetUStats(ctx, maxPlayers, g.getHeader().UserIDS...)
	if err != nil {
		JErr(ctx, err)
		return
	}
	stats = g.updateUStats(stats, g.playerStats(), g.playerUIDS())

	oldElos, newElos, err := cl.UpdateElo(ctx, g.getHeader().UserIDS, places)
	if err != nil {
		JErr(ctx, err)
		return
	}

	g.getHeader().Undo.Commit()
	err = cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := cl.txSave(ctx, tx, g, cu); err != nil {
			return err
		}

		if err := cl.SaveUStatsIn(tx, stats); err != nil {
			return err
		}

		return cl.SaveElosIn(tx, newElos)
	})
	if err != nil {
		JErr(ctx, err)
		return
	}

	err = g.sendEndGameNotifications(ctx, oldElos, newElos)
	if err != nil {
		// log but otherwise ignore send errors
		cl.Log.Warningf(err.Error())
	}
	ctx.JSON(http.StatusOK, nil)

}

const announceWinners Phase = "announce winners"

func (g *Game[S, T, P]) setFinishOrder() PlacesMap {
	g.Header.Phase = announceWinners
	g.Header.Status = Completed

	// Set to no current player
	g.SetCurrentPlayers()

	sortedByScore(g.Players, Descending)

	place := 1
	places := make(PlacesMap, len(g.Players))
	for i, p1 := range g.Players {
		// Update player stats
		p1.getStats().Finish = place
		uid1 := g.UIDForPID(p1.getPID())
		places[uid1] = place

		// Update Winners
		if place == 1 {
			g.Header.WinnerIDS = append(g.Header.WinnerIDS, uid1)
		}

		// if next player in order is tied with current player
		// place is not changed.
		// otherwise, place is set to index plus two (one to account for zero index and one to increment place)
		if j := i + 1; j < len(g.Players) {
			p2 := g.Players[j]
			if compareByScore(p1, p2) != EqualTo {
				place = i + 2
			}
		}

	}

	// g.newEntry(message{
	// 	"template": "announce-winners",
	// 	"winners":  g.WinnerIDS,
	// })
	return places
}

type result struct {
	Place  int
	Rating int
	Score  int64
	Name   string
	Inc    string
}

// reflect player order game state to header
func (g *Game[S, T, P]) UpdateOrder() {
	g.Header.OrderIDS = pie.Map(g.Players, func(p P) PID { return p.getPID() })
}

type results []result

func (g *Game[S, T, P]) sendEndGameNotifications(ctx *gin.Context, oldElos, newElos []Elo) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.Header.Status = Completed
	rs := make(results, g.Header.NumPlayers)

	for i, p := range g.Players {
		rs[i] = result{
			Place:  p.getStats().Finish,
			Rating: newElos[i].Rating,
			Score:  p.getStats().Score,
			Name:   g.Header.NameFor(p.getPID()),
			Inc:    fmt.Sprintf("%+d", newElos[i].Rating-oldElos[i].Rating),
		}
	}

	buf := new(bytes.Buffer)
	tmpl := template.New("end_game_notification")
	tmpl, err := tmpl.Parse(`
<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">
<html>
        <head>
                <meta http-equiv="content-type" content="text/html; charset=ISO-8859-1">
        </head>
        <body bgcolor="#ffffff" text="#000000">
                {{range $i, $r := $.Results}}
                <div style="height:3em">
                        <div style="height:3em;float:left;padding-right:1em">{{$r.Place}}.</div>
                        <div style="height:1em">{{$r.Name}} scored {{$r.Score}} points.</div>
                        <div style="height:1em">Elo {{$r.Inc}} (-> {{$r.Rating}})</div>
                </div>
                {{end}}
                <p></p>
                <p>Congratulations: {{$.Winners}}.</p>
        </body>
</html>`)
	if err != nil {
		return err
	}

	err = tmpl.Execute(buf, gin.H{
		"Results": rs,
		"Winners": ToSentence(g.winnerNames()),
	})
	if err != nil {
		return err
	}

	ms := make([]mailjet.InfoMessagesV31, len(g.Players))
	subject := fmt.Sprintf("SlothNinja Games: Tammany Hall (%s) Has Ended", g.Header.ID)
	body := buf.String()
	for i, p := range g.Players {
		ms[i] = mailjet.InfoMessagesV31{
			From: &mailjet.RecipientV31{
				Email: "webmaster@slothninja.com",
				Name:  "Webmaster",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: g.Header.EmailFor(p.getPID()),
					Name:  g.Header.NameFor(p.getPID()),
				},
			},
			Subject:  subject,
			HTMLPart: body,
		}
	}
	_, err = SendMessages(ctx, ms...)
	return err
}

func (g *Game[S, T, P]) winnerNames() []string {
	return pie.Map(g.Header.WinnerIDS, func(uid UID) string { return g.Header.NameFor(g.PlayerByUID(uid).getPID()) })
}

func (g *Game[S, T, P]) addNewPlayers() {
	g.Players = make([]P, g.Header.NumPlayers)
	for i := range g.Players {
		g.Players[i] = g.newPlayer(i)
	}
}

func (g *Game[S, T, P]) newPlayer(i int) P {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	p2 := P(new(T))
	p2.setPID(PID(i + 1))
	return p2
}

func (g *Game[S, T, P]) Start(h Header) PID {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.Header = h
	g.Header.Status = Running
	g.Header.StartedAt = updateTime()

	g.addNewPlayers()

	cp := pie.First(g.Players)
	g.SetCurrentPlayers(cp.getPID())
	g.NewEntry("start-game", nil, Line{"PIDS": g.Players.PIDS()})
	Warningf("g.Log: %#v", g.Log)
	return cp.getPID()
}
