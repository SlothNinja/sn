package sn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/firestore"
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
	readKind      = "Read"
	mlogKind      = "MLog"
	messagesKind  = "Messages"
)

type Game[P Playerer] struct {
	Header
	Players Players[P]
}

type Gamer[G any] interface {
	Views() ([]UID, []G)
	SetCurrentPlayers(...Playerer)
	Start(*Header) Playerer

	setFinishOrder() PlacesMap
	getHeader() *Header
	playerStats() []*Stats
	playerUIDS() []UID
	sendEndGameNotifications(*gin.Context, []Elo, []Elo) error
	updateUStats([]UStat, []*Stats, []UID) []UStat
	isCurrentPlayer(User) bool
}

func (cl Client[G, P]) gameDocRef(id string, rev int) *firestore.DocumentRef {
	return cl.gameCollectionRef().Doc(fmt.Sprintf("%s-%d", id, rev))
}

func (cl Client[G, P]) gameCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(gameKind)
}

func (cl Client[G, P]) cachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d", rev, uid))
}

func (cl Client[G, P]) fullyCachedDocRef(id string, rev int, uid UID) *firestore.DocumentRef {
	return cl.cachedCollectionRef(id).Doc(fmt.Sprintf("%d-%d-0", rev, uid))
}

func (cl Client[G, P]) cachedCollectionRef(id string) *firestore.CollectionRef {
	return cl.committedDocRef(id).Collection(cachedKind)
}

func (cl Client[G, P]) messageDocRef(gid, mid string) *firestore.DocumentRef {
	return cl.messagesCollectionRef(gid).Doc(mid)
}

func (cl Client[G, P]) messagesCollectionRef(id string) *firestore.CollectionRef {
	return cl.committedDocRef(id).Collection(messagesKind)
}

func (cl Client[G, P]) committedDocRef(id string) *firestore.DocumentRef {
	return cl.FS.Collection(committedKind).Doc(id)
}

func (cl Client[G, P]) viewDocRef(id string, uid UID) *firestore.DocumentRef {
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

func (cl Client[G, P]) ResetHandler() gin.HandlerFunc {
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

func (cl Client[G, P]) UndoHandler() gin.HandlerFunc {
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

func (cl Client[G, P]) RedoHandler() gin.HandlerFunc {
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

func (cl Client[G, P]) RollbackHandler() gin.HandlerFunc {
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

		g, err := cl.getRev(ctx, obj.Rev-1)
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

func (cl Client[G, P]) RollforwardHandler() gin.HandlerFunc {
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

		g, err := cl.getRev(ctx, obj.Rev+1)
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

func (cl Client[G, P]) getRev(ctx *gin.Context, rev int) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g := newGame[G]()
	id := getID(ctx)
	snap, err := cl.gameDocRef(id, rev).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}
	g.getHeader().ID = id
	return g, nil
}

func (cl Client[G, P]) GetGame(ctx *gin.Context, cu User, action ...stackFunc) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g, err := cl.GetCommitted(ctx)
	if err != nil {
		return g, err
	}

	// if current user not current player, then simply return g which is committed game
	if !g.isCurrentPlayer(cu) {
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
		g.getHeader().Undo = undo
		return g, nil
	}

	g, err = cl.getCached(ctx, undo.Current, cu.ID())
	if err != nil {
		return g, err
	}
	g.getHeader().Undo = undo
	return g, nil
}

func (cl Client[G, P]) getCached(ctx *gin.Context, rev int, uid UID) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g := newGame[G]()
	id := getID(ctx)
	snap, err := cl.fullyCachedDocRef(id, rev, uid).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}

	g.getHeader().ID = id
	return g, nil
}

const (
	cachedRootKind = "CachedRoot"
)

func (cl Client[G, P]) GetCommitted(ctx *gin.Context) (G, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g := newGame[G]()
	id := getID(ctx)
	snap, err := cl.committedDocRef(id).Get(ctx)
	if err != nil {
		return g, err
	}

	if err := snap.DataTo(g); err != nil {
		return g, err
	}

	g.getHeader().ID = id
	return g, nil
}

func newGame[G Gamer[G]]() G {
	var g G
	return reflect.New(reflect.TypeOf(g).Elem()).Interface().(G)
}

func (cl Client[G, P]) putCached(ctx *gin.Context, g G, u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(cl.fullyCachedDocRef(g.getHeader().ID, g.getHeader().Rev(), u.ID()), g); err != nil {
			return err
		}

		if err := tx.Set(cl.cachedDocRef(g.getHeader().ID, g.getHeader().Rev(), u.ID()), viewFor(g, u.ID())); err != nil {
			return err
		}

		return tx.Set(cl.StackDocRef(g.getHeader().ID, u.ID()), g.getHeader().Undo)
	})
}

func viewFor[G Gamer[G]](g G, uid1 UID) G {
	uids, views := g.Views()
	return views[pie.FindFirstUsing(uids, func(uid2 UID) bool { return uid1 == uid2 })]
}

func (cl Client[G, P]) CachedHandler(action func(G, *gin.Context, User) error) gin.HandlerFunc {
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
		g.getHeader().Undo.Update()

		if err := cl.putCached(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl Client[G, P]) FinishTurnHandler(action func(G, *gin.Context, User) (P, P, error)) gin.HandlerFunc {
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

		cp, np, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		cl.Log.Debugf("cp: %#v\nnp: %#v", cp, np)

		cp.GetStats().Moves++
		cp.GetStats().Think += time.Since(g.getHeader().UpdatedAt)

		if np.GetPID() == NoPID {
			cl.endGame(ctx, g, cu)
			return
		}
		np.Reset()
		cl.Log.Debugf("cp.ID: %#v\nnp.ID: %#v", cp.GetPID(), np.GetPID())
		cl.Log.Debugf("CPID: %#v", g.getHeader().CPIDS)
		g.SetCurrentPlayers(np)
		cl.Log.Debugf("cp.ID: %#v\nnp.ID: %#v", cp.GetPID(), np.GetPID())
		cl.Log.Debugf("CPID: %#v", g.getHeader().CPIDS)

		if err := cl.Commit(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (g Game[P]) playerUIDS() []UID {
	return pie.Map(g.Players, func(p P) UID { return g.UserIDS[p.GetPID().ToIndex()] })
}

func (g Game[P]) UIDForPID(pid PID) UID {
	return g.UserIDS[pid.ToIndex()]
}

func (g Game[P]) PlayerByPID(pid PID) P {
	const notFound = -1
	index := pie.FindFirstUsing(g.Players, func(p P) bool { return p.GetPID() == pid })
	if index == notFound {
		var zerop P
		return zerop
	}
	return g.Players[index]
}

func (g Game[P]) PlayerByUID(uid UID) P {
	index := UIndex(pie.FindFirstUsing(g.UserIDS, func(id UID) bool { return id == uid }))
	return g.PlayerByPID(index.ToPID())
}

// IndexForPlayer returns the index for the player and bool indicating whether player found.
// if not found, returns -1
func (g Game[P]) IndexForPlayer(p1 P) int {
	return pie.FindFirstUsing(g.Players, func(p2 P) bool { return p1.GetPID() == p2.GetPID() })
}

// treats Players as a circular buffer, thus permitting indices larger than length and indices less than 0
func (g Game[P]) PlayerByIndex(i int) P {
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

func (ps Players[P]) PIDS() []PID {
	return pie.Map(ps, func(p P) PID { return p.GetPID() })
}

func (g Game[P]) RandomizePlayers() {
	g.Players = pie.Shuffle(g.Players, myRandomSource)
	g.UpdateOrder()
}

// CurrentPlayers returns the players whose turn it is.
func (g Game[P]) CurrentPlayers() []P {
	return pie.Map(g.CPIDS, func(pid PID) P { return g.PlayerByPID(pid) })
}

// currentPlayer returns the player whose turn it is.
func (g Game[P]) CurrentPlayer() P {
	return pie.First(g.CurrentPlayers())
}

// Returns player asssociated with user if such player is current player
// Otherwise, return nil
func (g Game[P]) CurrentPlayerFor(u User) P {
	i := g.IndexFor(u.ID())
	if i == -1 {
		var zerop P
		return zerop
	}

	return g.PlayerByPID(i.ToPID())
}

func (g *Game[P]) SetCurrentPlayers(ps ...Playerer) {
	g.CPIDS = pie.Map(ps, func(p Playerer) PID { return p.GetPID() })
}

func (g Game[P]) isCurrentPlayer(cu User) bool {
	return g.CurrentPlayerFor(cu).GetPID() != NoPID
}

func (g Game[P]) copyPlayers() []P {
	return pie.Map(g.Players, func(p P) P { return p.Copy().(P) })
}

func (g Game[P]) ValidatePlayerAction(cu User) (P, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cp, err := g.ValidateCurrentPlayer(cu)
	switch {
	case err != nil:
		return cp, err
	case cp.GetPerformedAction():
		return cp, fmt.Errorf("current player already performed action: %w", ErrValidation)
	default:
		return cp, nil
	}
}

func (g Game[P]) ValidateCurrentPlayer(cu User) (P, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cp := g.CurrentPlayerFor(cu)
	if cp.GetPID() == NoPID {
		return cp, ErrPlayerNotFound
	}
	return cp, nil
}

func (g Game[P]) playerStats() []*Stats {
	return pie.Map(g.Players, func(p P) *Stats { return p.GetStats() })
}

func (g Game[P]) UIDSForPIDS(pids []PID) []UID {
	return pie.Map(pids, func(pid PID) UID { return g.UIDForPID(pid) })
}

func (g Game[P]) UserKeyFor(pid PID) *datastore.Key {
	return NewUser(g.UIDForPID(pid)).Key
}

// cp specifies the current
// return player after cp that satisfies all tests ts
// if tests ts is empty, return player after cp
func (g Game[P]) NextPlayer(cp P, ts ...func(P) bool) P {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	start := g.IndexForPlayer(cp) + 1
	stop := start + g.NumPlayers

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

func (cl Client[G, P]) endGame(ctx *gin.Context, g G, cu User) {
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
		if err := cl.SaveGameIn(ctx, tx, g, cu); err != nil {
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

func (g Game[P]) setFinishOrder() PlacesMap {
	g.getHeader().Phase = announceWinners
	g.getHeader().Status = Completed

	// Set to no current player
	g.SetCurrentPlayers()

	sortedByScore(g.Players, Descending)

	place := 1
	places := make(PlacesMap, len(g.Players))
	for i, p1 := range g.Players {
		// Update player stats
		p1.GetStats().Finish = place
		uid1 := g.UIDForPID(p1.GetPID())
		places[uid1] = place

		// Update Winners
		if place == 1 {
			g.WinnerIDS = append(g.WinnerIDS, uid1)
		}

		// if next player in order is tied with current player
		// place is not changed.
		// otherwise, place is set to index plus two (one to account for zero index and one to increment place)
		if j := i + 1; j < len(g.Players) {
			p2 := g.Players[j]
			if p1.compareByScore(p2) != EqualTo {
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
func (g Game[P]) UpdateOrder() {
	g.OrderIDS = pie.Map(g.Players, func(p P) PID { return p.GetPID() })
}

type results []result

func (g Game[P]) sendEndGameNotifications(ctx *gin.Context, oldElos, newElos []Elo) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.Status = Completed
	rs := make(results, g.NumPlayers)

	for i, p := range g.Players {
		rs[i] = result{
			Place:  p.GetStats().Finish,
			Rating: newElos[i].Rating,
			Score:  p.GetStats().Score,
			Name:   g.NameFor(p.GetPID()),
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
	subject := fmt.Sprintf("SlothNinja Games: Tammany Hall (%s) Has Ended", g.ID)
	body := buf.String()
	for i, p := range g.Players {
		ms[i] = mailjet.InfoMessagesV31{
			From: &mailjet.RecipientV31{
				Email: "webmaster@slothninja.com",
				Name:  "Webmaster",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: g.EmailFor(p.GetPID()),
					Name:  g.NameFor(p.GetPID()),
				},
			},
			Subject:  subject,
			HTMLPart: body,
		}
	}
	_, err = SendMessages(ctx, ms...)
	return err
}

func (g Game[P]) winnerNames() []string {
	return pie.Map(g.WinnerIDS, func(uid UID) string { return g.NameFor(g.PlayerByUID(uid).GetPID()) })
}

func (g *Game[P]) addNewPlayers() {
	g.Players = make([]P, g.NumPlayers)
	for i := range g.Players {
		g.Players[i] = g.newPlayer(i)
	}
}

func (g Game[P]) newPlayer(i int) P {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var p P
	Debugf("p: %#v", p)
	p2 := p.New()
	Debugf("p2: %#v", p2)
	p2.setPID(PID(i + 1))
	return p2.(P)
}

func (g *Game[P]) Start(h *Header) Playerer {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.Header = *h
	g.Status = Running
	g.StartedAt = updateTime()

	g.addNewPlayers()

	cp := pie.First(g.Players)
	g.SetCurrentPlayers(cp)
	// g.newEntry(message{"template": "start-game"})
	return cp
}
