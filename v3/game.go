package sn

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"math/rand"
	"net/http"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"github.com/mailjet/mailjet-apiv3-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	gameKind       = "Game"
	revKind        = "Rev"
	cacheKind      = "CacheFor"
	fullCachedKind = "FullCacheFor"
	viewKind       = "ViewFor"
	committedKind  = "Committed"
	indexKind      = "Index"
	readKind       = "Read"
	mlogKind       = "MLog"
	messagesKind   = "Messages"
)

type Game[S, PT any, P Playerer[PT]] struct {
	Header  Header
	Players Players[PT, P]
	Log     glog
	State   S
}

type Gamer[G any] interface {
	Viewer[G]
	Comparer

	getHeader() *Header
	isCurrentPlayer(*User) bool
	newEntry(string, H, time.Time)
	playerStats() []*Stats
	playerUIDS() []UID
	ptr[G]
	reset(...PID)
	getResults([]Elo, []Elo) results
	sendEndGameNotifications(*gin.Context, results) error
	setCurrentPlayerers
	setFinishOrder(compareFunc) PlacesMap
	starter
	statsFor(PID) *Stats
	updateUStats([]UStat, []*Stats, []UID) []UStat
}

type Viewer[T any] interface {
	Views() ([]UID, []*T)
	ViewFor(UID) *T
}

type compareFunc func(PID, PID) int

type Comparer interface {
	Compare(PID, PID) int
}

type starter interface {
	Start(Header) PID
}

type setCurrentPlayerers interface {
	SetCurrentPlayers(...PID)
}

const (
	gameKey = "Game"
	jsonKey = "JSON"
	hParam  = "hid"
)

func (cl *GameClient[GT, G]) resetHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		h, err := cl.getHeader(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
			return cl.clearCached(c, h.ID, h.Undo.Committed, cu.ID)
		})
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (cl *GameClient[GT, G]) undoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			slog.Debug(err.Error())
		}

		gid := getID(ctx)
		stack, err := cl.getStack(ctx, gid, cu.ID)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !stack.Undo() {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		err = cl.setStack(ctx, gid, cu.ID, stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (cl *GameClient[GT, G]) redoHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		gid := getID(ctx)
		stack, err := cl.getStack(ctx, gid, cu.ID)
		if err != nil {
			JErr(ctx, err)
		}

		// do nothing if stack does not change
		if !stack.Redo() {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		err = cl.setStack(ctx, gid, cu.ID, stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (cl *GameClient[GT, G]) abandonHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

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
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireAdmin(ctx)
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

		if obj.Rev <= 0 {
			ctx.JSON(http.StatusOK, nil)
		}

		g, err := cl.getRev(ctx, obj.Rev-1)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = cl.save(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (cl *GameClient[GT, G]) rollforwardHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireAdmin(ctx)
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

		err = cl.save(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (cl *GameClient[GT, G]) getRev(ctx *gin.Context, rev int) (G, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	id := getID(ctx)
	snap, err := cl.revDocRef(id, rev).Get(ctx)
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

// func (cl *GameClient[GT, G]) getGame(ctx *gin.Context, cu User, action ...stackFunc) (G, error) {
func (cl *GameClient[GT, G]) getGame(ctx *gin.Context, cu *User) (G, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	g, err := cl.getCommitted(ctx)
	if err != nil {
		return g, err
	}

	// if current user not current player, then simply return g which is committed game
	if !g.isCurrentPlayer(cu) {
		return g, nil
	}

	gid := getID(ctx)
	undo, err := cl.getStack(ctx, gid, cu.ID)

	// if undo stack not found, then simply return g which is committed game
	if status.Code(err) == codes.NotFound {
		return g, nil
	}
	if err != nil {
		return g, err
	}

	// cached state same as committed, though undo stack may differ.
	// thus, replace undo stack and return
	if undo.Current == undo.Committed {
		g.getHeader().Undo = undo
		return g, nil
	}

	// cached state differs from committed.
	// thus, get cached state, replace undo stack and return
	g, err = cl.getCached(ctx, g.getHeader().Undo.Current, cu.ID, undo.Current)
	if err != nil {
		return g, err
	}
	g.getHeader().Undo = undo
	return g, nil
}

func (cl *GameClient[GT, G]) getCached(ctx *gin.Context, rev int, uid UID, crev int) (G, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	id := getID(ctx)
	snap, err := cl.fullyCachedDocRef(id, rev, uid, crev).Get(ctx)
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
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	h, err := cl.getHeader(ctx)
	if err != nil {
		return nil, err
	}

	snap, err := cl.revDocRef(h.ID, h.Undo.Current).Get(ctx)
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.getHeader().ID = h.ID
	return g, nil
}

func (cl *GameClient[GT, G]) txGetCommitted(tx *firestore.Transaction, gid string) (G, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	h, err := cl.txGetHeader(tx, gid)
	if err != nil {
		return nil, err
	}

	snap, err := tx.Get(cl.revDocRef(h.ID, h.Undo.Current))
	if err != nil {
		return nil, err
	}

	g := G(new(GT))
	if err := snap.DataTo(g); err != nil {
		return nil, err
	}

	g.getHeader().ID = h.ID
	return g, nil
}

func (cl *GameClient[GT, G]) txGetHeader(tx *firestore.Transaction, hid string) (Header, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	snap, err := tx.Get(cl.indexDocRef(hid))
	if err != nil {
		return Header{}, err
	}

	var h Header
	if err := snap.DataTo(&h); err != nil {
		return Header{}, err
	}

	h.ID = hid
	return h, nil
}

func (cl *GameClient[GT, G]) getHeader(ctx *gin.Context) (*Header, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	id := getID(ctx)
	snap, err := cl.indexDocRef(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	h := new(Header)
	if err := snap.DataTo(h); err != nil {
		return nil, err
	}

	h.ID = id
	return h, nil
}

func (cl *GameClient[GT, G]) putCached(ctx *gin.Context, g G, u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(cl.fullyCachedDocRef(g.getHeader().ID, g.getHeader().Undo.Committed, u.ID, g.getHeader().Undo.Current), g); err != nil {
			return err
		}

		if err := tx.Set(cl.cachedDocRef(g.getHeader().ID, g.getHeader().Undo.Committed, u.ID, g.getHeader().Undo.Current), g.ViewFor(u.ID)); err != nil {
			return err
		}

		return tx.Set(cl.StackDocRef(g.getHeader().ID, u.ID), g.getHeader().Undo)
	})
}

type CachedActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, *User) (string, error)

func (cl *GameClient[GT, G]) CachedHandler(action CachedActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			slog.Debug(err.Error())
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		msg, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		g.getHeader().Undo.Update()

		if err := cl.putCached(ctx, g, cu); err != nil {
			JErr(ctx, err)
			return
		}

		if len(msg) > 0 {
			ctx.JSON(http.StatusOK, gin.H{
				"Message": msg,
				"Game":    g.ViewFor(cu.ID),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

type FinishTurnActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, *User) (PID, []PID, string, error)

func (cl *GameClient[GT, G]) FinishTurnHandler(action FinishTurnActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cpid, npids, msg, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cpStats := g.statsFor(cpid)
		cpStats.Moves++
		cpStats.Think += time.Since(g.getHeader().UpdatedAt)

		if len(npids) == 0 {
			cl.endGame(ctx, g, cu)
			return
		}
		g.reset(npids...)
		g.SetCurrentPlayers(npids...)

		err = cl.commit(ctx, g, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if len(msg) > 0 {
			ctx.JSON(http.StatusOK, gin.H{
				"Message": msg,
				"Game":    g.ViewFor(cu.ID),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (g *Game[S, T, P]) playerUIDS() []UID {
	return pie.Map(g.Players, func(p P) UID { return g.Header.UserIDS[p.PID().ToUIndex()] })
}

func (g *Game[S, T, P]) UIDForPID(pid PID) UID {
	return g.Header.UserIDS[pid.ToUIndex()]
}

func (g *Game[S, T, P]) PlayerByPID(pid PID) P {
	const notFound = -1
	index := pie.FindFirstUsing(g.Players, func(p P) bool { return p.PID() == pid })
	if index == notFound {
		var zerop P
		return zerop
	}
	return g.Players[index]
}

func (g *Game[S, T, P]) PlayerByUID(uid UID) P {
	index := UIndex(pie.FindFirstUsing(g.Header.UserIDS, func(id UID) bool { return id == uid }))
	return g.PlayerByPID(index.ToPID())
}

// IndexForPlayer returns the index for the player and bool indicating whether player found.
// if not found, returns -1
func (g *Game[S, T, P]) IndexForPlayer(p1 P) int {
	return pie.FindFirstUsing(g.Players, func(p2 P) bool { return p1.PID() == p2.PID() })
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
	return pie.Map(ps, func(p P) PID { return p.PID() })
}

func (ps Players[T, P]) IndexFor(p1 P) (int, bool) {
	const notFound = -1
	const found = true
	index := pie.FindFirstUsing(ps, func(p2 P) bool { return p1.PID() == p2.PID() })
	if index == notFound {
		return index, !found
	}
	return index, found
}

func (ps Players[T, P]) Randomize() {
	rand.Shuffle(len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] })
}

func (g *Game[S, T, P]) RandomizePlayers() {
	g.Players.Randomize()
	g.updateOrder()
}

// CurrentPlayers returns the players whose turn it is.
func (g *Game[S, T, P]) CurrentPlayers() Players[T, P] {
	return pie.Map(g.Header.CPIDS, func(pid PID) P { return g.PlayerByPID(pid) })
}

// currentPlayer returns the player whose turn it is.
func (g *Game[S, T, P]) CurrentPlayer() P {
	return pie.First(g.CurrentPlayers())
}

// Returns player asssociated with user if such player is current player
// Otherwise, return nil
func (g *Game[S, T, P]) CurrentPlayerFor(u *User) P {
	i := g.Header.IndexFor(u.ID)
	if i == -1 {
		var zerop P
		return zerop
	}

	return g.PlayerByPID(i.ToPID())
}

func (g *Game[S, T, P]) SetCurrentPlayers(ps ...PID) {
	g.Header.CPIDS = slices.Clone(ps)
}

func (g *Game[S, T, P]) RemoveCurrentPlayer(pid PID) {
	for i, pid2 := range g.Header.CPIDS {
		if pid2 == pid {
			g.Header.CPIDS = append(g.Header.CPIDS[:i], g.Header.CPIDS[i+1:]...)
			return
		}
	}
}

func (g *Game[S, T, P]) isCurrentPlayer(cu *User) bool {
	return g.CurrentPlayerFor(cu).PID() != NoPID
}

func (g *Game[S, T, P]) copyPlayers() Players[T, P] {
	return deepCopy(g.Players)
}

func (g *Game[S, T, P]) ValidatePlayerAction(cu *User) (P, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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

func (g *Game[S, T, P]) ValidateCurrentPlayer(cu *User) (P, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	cp := g.CurrentPlayerFor(cu)
	if cp.PID() == NoPID {
		return cp, ErrPlayerNotFound
	}
	return cp, nil
}

func (g *Game[S, T, P]) ValidateFinishTurn(cu *User) (P, error) {
	cp, err := g.ValidateCurrentPlayer(cu)
	switch {
	case err != nil:
		return nil, err
	case !cp.getPerformedAction():
		return nil, fmt.Errorf("%s has yet to perform an action: %w", cu.Name, ErrValidation)
	default:
		return cp, nil
	}
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

func (g *Game[S, T, P]) reset(pids ...PID) {
	var zerop P
	for _, pid := range pids {
		p := g.PlayerByPID(pid)
		if p != zerop {
			p.reset()
		}
	}
}

func (g *Game[S, T, P]) UIDSForPIDS(pids []PID) []UID {
	return pie.Map(pids, func(pid PID) UID { return g.UIDForPID(pid) })
}

// NextPlayer returns player after cp that satisfies all tests ts
// if tests ts is empty, return player after cp
func (g *Game[S, T, P]) NextPlayer(cp P, ts ...func(P) bool) P {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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

const gameOver Phase = "Game Over"

func (cl *GameClient[GT, G]) endGame(ctx *gin.Context, g G, cu *User) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	places := g.setFinishOrder(g.Compare)
	g.getHeader().Status = Completed
	g.getHeader().EndedAt = time.Now()
	g.getHeader().Phase = gameOver

	stats, err := cl.GetUStats(ctx, maxPlayers, g.getHeader().UserIDS...)
	if err != nil {
		JErr(ctx, err)
		return
	}
	stats = g.updateUStats(stats, g.playerStats(), g.playerUIDS())

	oldElos, newElos, err := cl.updateElo(ctx, g.getHeader().UserIDS, places)
	if err != nil {
		JErr(ctx, err)
		return
	}

	rs := g.getResults(oldElos, newElos)
	g.newEntry("game-results", H{"Results": rs}, g.getHeader().EndedAt)

	err = cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		if err := cl.txCommit(c, tx, g, cu); err != nil {
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

	err = g.sendEndGameNotifications(ctx, rs)
	if err != nil {
		// log but otherwise ignore send errors
		slog.Warn(err.Error())
	}
	ctx.JSON(http.StatusOK, nil)

}

func (g *Game[S, T, P]) setFinishOrder(compare compareFunc) PlacesMap {
	// Set to no current player
	g.SetCurrentPlayers()

	// sortedByScore(g.Players)
	g.sortPlayers(compare)

	place := 1
	places := make(PlacesMap, len(g.Players))
	for i, p1 := range g.Players {
		// Update player stats
		p1.getStats().Finish = place
		uid1 := g.UIDForPID(p1.PID())
		places[uid1] = place

		// Update Winners
		if place == 1 {
			g.Header.WinnerIDS = append(g.Header.WinnerIDS, uid1)
		}

		// If next player in order is tied with current player, place is not changed.
		// Otherwise, increment place
		if j := i + 1; j < len(g.Players) {
			p2 := g.Players[j]
			if compare(p1.PID(), p2.PID()) != EqualTo {
				place += 1
			}
		}

	}

	return places
}

type result struct {
	PID    PID
	Place  int
	Rating int
	Score  int64
	Inc    string
}

// reflect player order game state to header
func (g *Game[S, T, P]) updateOrder() {
	g.Header.OrderIDS = g.Players.PIDS()
}

type results []result

func (g *Game[S, T, P]) getResults(oldElos, newElos []Elo) results {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	rs := make(results, g.Header.NumPlayers)

	for _, p := range g.Players {
		i := int(p.PID().ToUIndex())
		rs[i] = result{
			PID:    p.PID(),
			Place:  p.getStats().Finish,
			Rating: newElos[i].Rating,
			Score:  p.getStats().Score,
			Inc:    fmt.Sprintf("%+d", newElos[i].Rating-oldElos[i].Rating),
		}
	}
	return rs
}

func (g *Game[S, T, P]) sendEndGameNotifications(ctx *gin.Context, rs results) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	buf := new(bytes.Buffer)
	tmpl, err := endGameTemplate()
	if err != nil {
		return err
	}

	err = tmpl.Execute(buf, gin.H{
		"Results": rs,
		"Winners": toSentence(g.winnerNames()),
	})
	if err != nil {
		return err
	}

	ms := make([]mailjet.InfoMessagesV31, len(g.Players))
	subject := fmt.Sprintf("SlothNinja Games: (%s) Has Ended", g.Header.ID)
	body := buf.String()
	for i, p := range g.Players {
		ms[i] = mailjet.InfoMessagesV31{
			From: &mailjet.RecipientV31{
				Email: "webmaster@slothninja.com",
				Name:  "Webmaster",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: g.Header.EmailFor(p.PID()),
					Name:  g.Header.NameFor(p.PID()),
				},
			},
			Subject:  subject,
			HTMLPart: body,
		}
	}
	_, err = SendMessages(ms...)
	return err
}

func endGameTemplate() (*template.Template, error) {
	return template.New("end_game_notification").Parse(`
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
}

func (g *Game[S, T, P]) winnerNames() []string {
	return pie.Map(g.Header.WinnerIDS, func(uid UID) string { return g.Header.NameFor(g.PlayerByUID(uid).PID()) })
}

func (g *Game[S, T, P]) addNewPlayers() {
	g.Players = make(Players[T, P], g.Header.NumPlayers)
	for i := range g.Players {
		g.Players[i] = g.newPlayer(i)
	}
}

func (g *Game[S, T, P]) newPlayer(i int) P {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	p2 := P(new(T))
	p2.setPID(PID(i + 1))
	return p2
}

func (g *Game[S, T, P]) Start(h Header) PID {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	g.Header = h
	g.Header.Status = Running

	g.addNewPlayers()

	cp := pie.First(g.Players)
	g.SetCurrentPlayers(cp.PID())
	return cp.PID()
}

// Views implements part of Viewer interface
func (g *Game[S, T, P]) Views() ([]UID, []*Game[S, T, P]) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	return []UID{0}, []*Game[S, T, P]{g.ViewFor(0)}
}

// ViewFor implements part of Viewer interface
func (g *Game[S, T, P]) ViewFor(uid UID) *Game[S, T, P] {
	return g.DeepCopy()
}

func (g *Game[S, T, P]) getHeader() *Header {
	return &(g.Header)
}
