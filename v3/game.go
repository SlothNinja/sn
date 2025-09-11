package sn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
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
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Game implements a game
type Game[S, PT any, P playerer[PT]] struct {
	Header  Header
	Players Players[PT, P]
	Log     glog
	State   S
}

// Gamer interface implemented by Game
type Gamer[G any] interface {
	Viewer[G]
	Comparer

	header() *Header
	id() string
	setID(string)
	stack() *Stack
	setStack(*Stack)
	getIndex() *index
	isCurrentPlayer(*User) bool
	newEntry(string, H)
	playerStats() []*Stats
	playerUIDS() []UID
	ptr[G]
	getResults([]elo, []elo) results
	sendEndGameNotifications(results) error
	setCurrentPlayerers
	setFinishOrder(compareFunc) placesMap
	starter
	statsFor(PID) *Stats
	updateUStats([]ustat, []*Stats, []UID) []ustat
	updateStatsFor(PID)
	uidsForPidser
}

// Viewer interface provides methods used to return game states suitable for viewing by a given user.
// In short, viewer should remove private game data that should not be exposed to a particular user.
// For example, removing data for cards in hands of other players.type
type Viewer[T any] interface {
	Views() ([]UID, []*T)
	ViewFor(UID) *T
}

type compareFunc func(PID, PID) int

// Comparer provides interface for comparing players associated with the player ids
// Returns 1 if player of first pid greater, 0 if equal, and -1 if if player of first pid is less than
type Comparer interface {
	Compare(PID, PID) int
}

type starter interface {
	Start(Header) PID
}

type setCurrentPlayerers interface {
	SetCurrentPlayers(...PID) []PID
}

func getID(ctx *gin.Context) string {
	return ctx.Param("id")
}

func (cl *GameClient[GT, G]) resetHandler() gin.HandlerFunc {
	return cl.stackHandler((*Stack).reset)
}

func (cl *GameClient[GT, G]) undoHandler() gin.HandlerFunc {
	return cl.stackHandler((*Stack).undo)
}

func (cl *GameClient[GT, G]) redoHandler() gin.HandlerFunc {
	return cl.stackHandler((*Stack).redo)
}

func (cl *GameClient[GT, G]) stackHandler(update func(*Stack) bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		stack, err := cl.getStack(ctx, getID(ctx), cu.ID)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !update(stack) {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		g, err := cl.getGame(ctx, cu, stack.Current)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.setStack(stack)
		g.header().UpdatedAt = timestamppb.Now()

		if err := cl.updateViews(ctx, g); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) abandonHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

		cu, err := cl.RequireAdmin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g, err := cl.getGame(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.header().Status = Abandoned
		if err := cl.save(ctx, g); err != nil {
			JErr(ctx, err)
			return
		}

		msg := fmt.Sprintf("%s has been abandoned.", g.header().Title)
		ctx.JSON(http.StatusOK, H{"Message": msg})
	}
}

func (cl *GameClient[GT, G]) rollbackHandler() gin.HandlerFunc {
	return cl.rollHandler((*Stack).rollbackward)
}

func (cl *GameClient[GT, G]) rollforwardHandler() gin.HandlerFunc {
	return cl.rollHandler((*Stack).rollforward)
}

func (cl *GameClient[GT, G]) rollHandler(update func(*Stack, int) bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

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

		gid := getID(ctx)
		stack, err := cl.getStack(ctx, gid, 0)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !update(stack, obj.Rev) {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		g, err := cl.getGame(ctx, cu, stack.Current)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.setStack(stack)
		g.header().UpdatedAt = timestamppb.Now()

		err = cl.save(ctx, g)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

// return first item of v, if v as length 1
// otherwise, returns default value d
func firstOrDefault[T any](v []T, d T) T {
	if len(v) == 1 {
		return v[0]
	}
	return d
}

// getGame returns game for current, unless a single rev value provide.
// In which case, getGame returns the requested rev
func (cl *GameClient[GT, G]) getGame(ctx *gin.Context, cu *User, rev ...int) (G, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	gid := getID(ctx)
	stack, err := cl.getStack(ctx, gid, cu.ID)
	if status.Code(err) == codes.NotFound {
		if stack, err = cl.getStack(ctx, gid, 0); status.Code(err) == codes.NotFound {
			Warnf("stack not found")
			stack = new(Stack)
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}

	rv := firstOrDefault(rev, stack.Current)
	g, err := cl.getRev(ctx, gid, rv)
	if err != nil {
		return nil, err
	}

	stack.Current = rv
	g.setStack(stack)
	return g, nil
}

func (cl *GameClient[GT, G]) getRev(ctx *gin.Context, gid string, rev int) (G, error) {
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

func (cl *GameClient[GT, G]) cacheRev(ctx *gin.Context, g G) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if err := cl.txUpdateViews(tx, g); err != nil {
			return err
		}

		return cl.txUpdateRev(tx, g)
	})
}

// Result provides a return value associated with performing a game action
type Result struct {
	Message string
}

// ActionFunc provides a func type for game actions executed by CachedHandler or SavedHandler
type ActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, *User) (Result, error)

// CachedHandler provides a general purpose handler for performing cached game actions
func (cl *GameClient[GT, G]) CachedHandler(action ActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

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

		result, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		g.stack().update()

		g.header().UpdatedAt = timestamppb.Now()
		if err := cl.cacheRev(ctx, g); err != nil {
			JErr(ctx, err)
			return
		}

		if len(result.Message) > 0 {
			ctx.JSON(http.StatusOK, H{"Message": result.Message})
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

// CommitHandler provides a general purpose handler for performing saved game actions
func (cl *GameClient[GT, G]) CommitHandler(action ActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

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

		result, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}
		g.stack().update()

		if err := cl.commit(ctx, g); err != nil {
			JErr(ctx, err)
			return
		}

		if len(result.Message) > 0 {
			ctx.JSON(http.StatusOK, H{"Message": result.Message})
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

// FinishResult provides a return value associated with performing a finish turn action
type FinishResult struct {
	CurrentPlayerID PID
	NextPlayerIDS   []PID
	Message         string
	Token           SubToken
	Data            H
}

// FinishTurnActionFunc provides a func type for finish turn action executed by FinishTurnHandler
type FinishTurnActionFunc[GT any, G Gamer[GT]] func(G, *gin.Context, *User) (FinishResult, error)

// FinishTurnHandler provides a general purpose handler for performing finish turn actions
func (cl *GameClient[GT, G]) FinishTurnHandler(action FinishTurnActionFunc[GT, G]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

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

		result, err := action(g, ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.updateStatsFor(result.CurrentPlayerID)

		if len(result.NextPlayerIDS) == 0 {
			cl.endGame(ctx, g)
			return
		}
		// g.reset(result.NextPlayerIDS...)
		notify := g.SetCurrentPlayers(result.NextPlayerIDS...)

		err = cl.commit(ctx, g)
		if err != nil {
			JErr(ctx, err)
			return
		}

		go func() {
			if err := cl.updateSubs(ctx, g.id(), result.Token, cu.ID); err != nil {
				Warnf("attempted to update sub: %q: %v", result.Token, err)
			}

			response, err := cl.sendNotifications(ctx, g, notify)
			if err != nil {
				Warnf("attempted to send notifications to: %v: %v", result.NextPlayerIDS, err)
			}
			Warnf("batch send response: %v", response)
		}()

		if len(result.Message) > 0 {
			ctx.JSON(http.StatusOK, gin.H{
				"Message": result.Message,
				"Game":    g.ViewFor(cu.ID),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"Game": g.ViewFor(cu.ID)})
	}
}

func (g *Game[S, T, P]) updateStatsFor(pid PID) {
	stats := g.statsFor(pid)
	stats.Moves++
	stats.Think += time.Since(g.header().UpdatedAt.AsTime())
}

func (g *Game[S, T, P]) playerUIDS() []UID {
	return pie.Map(g.Players, func(p P) UID { return g.Header.UserIDS[p.PID().ToUIndex()] })
}

// UIDForPID returns the player id associated with the user id
func (g *Game[S, T, P]) UIDForPID(pid PID) UID {
	if i := int(pid.ToUIndex()); i > 0 && i < len(g.Header.UserIDS)-1 {
		return g.Header.UserIDS[i]
	}
	return noUID
}

// PlayerByPID returns the player associated with the player id
func (g *Game[S, T, P]) PlayerByPID(pid PID) P {
	const notFound = -1
	index := pie.FindFirstUsing(g.Players, func(p P) bool { return p.PID() == pid })
	if index == notFound {
		var zerop P
		return zerop
	}
	return g.Players[index]
}

// PlayerByUID returns the player associated with the user id
func (g *Game[S, T, P]) PlayerByUID(uid UID) P {
	index := UIndex(pie.FindFirstUsing(g.Header.UserIDS, func(id UID) bool { return id == uid }))
	return g.PlayerByPID(index.ToPID())
}

// IndexForPlayer returns the index for the player and bool indicating whether player found.
// if not found, returns -1
func (g *Game[S, T, P]) IndexForPlayer(p1 P) int {
	return pie.FindFirstUsing(g.Players, func(p2 P) bool { return p1.PID() == p2.PID() })
}

// PlayerByIndex returns player at associated index in Players slice
// PlayerByIndex treats Players slice as a circular buffer, thus permitting indices larger than length and indices less than 0
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

// PIDS returns the player identiers for the players slice
func (ps Players[T, P]) PIDS() []PID {
	return pie.Map(ps, func(p P) PID { return p.PID() })
}

// IndexFor returns the index for the given player in Players slice
// Also, return true if player found, and false if player not found in Players slice
func (ps Players[T, P]) IndexFor(p1 P) (int, bool) {
	const notFound = -1
	const found = true
	index := pie.FindFirstUsing(ps, func(p2 P) bool { return p1.PID() == p2.PID() })
	if index == notFound {
		return index, !found
	}
	return index, found
}

// Randomize randomizes the order of the players in the Players slice
func (ps Players[T, P]) Randomize() {
	rand.Shuffle(len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] })
}

// RandomizePlayers randomizes the order of the players in the Players slice, and
// updates the order of such players in the Header for the game.
func (g *Game[S, T, P]) RandomizePlayers() {
	g.Players.Randomize()
	g.UpdateOrder()
}

// CurrentPlayers returns the players whose turn it is.
func (g *Game[S, T, P]) CurrentPlayers() Players[T, P] {
	return pie.Map(g.Header.CPIDS, func(pid PID) P { return g.PlayerByPID(pid) })
}

// CurrentPlayer returns the player whose turn it is.
func (g *Game[S, T, P]) CurrentPlayer() P {
	return pie.First(g.CurrentPlayers())
}

// CurrentPlayerFor returns player asssociated with user if such player is current player
// CurrentPlayerFor also returns true if player asssociated with user is found.
// Otherwise, returns false.
func (g *Game[S, T, P]) CurrentPlayerFor(u *User) (P, bool) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var zerop P
	if u == nil {
		return zerop, false
	}

	ps := g.CurrentPlayers()
	pid := g.Header.PIDFor(u.ID)
	index := pie.FindFirstUsing(ps, func(p P) bool { return p.PID() == pid })

	const notFound = -1
	if index == notFound {
		Debugf("u: %#v", u)
		if u.Admin && u.GodMode && len(ps) == 1 {
			return ps[0], true
		}
		return zerop, false
	}

	return ps[index], true
}

// SetCurrentPlayers sets the current players to those associated with provided player ids.
// SetCurrentPlayers also returns player ids of player that were not already a current player.
// The returned player ids is helpful for determining to whom turn notifications should be sent
func (g *Game[S, T, P]) SetCurrentPlayers(pids ...PID) []PID {
	added, _ := pie.Diff(g.Header.CPIDS, pids)
	g.Header.CPIDS = slices.Clone(pids)
	return added
}

// RemoveFromCurrentPlayers removes from current players those associated with provided player ids.
func (g *Game[S, T, P]) RemoveFromCurrentPlayers(pids ...PID) {
	g.Header.CPIDS, _ = pie.Diff(pids, g.Header.CPIDS)
}

func (g Game[S, T, P]) isCurrentPlayer(cu *User) bool {
	_, found := g.CurrentPlayerFor(cu)
	return found
}

// ValidatePlayerAction performs basic validations for determining whether provided user
// can perform a player action. If user can perform player action, ValidatePlayerAction
// returns player associated with user. Otherwise, ValidatePlayerAction returns an error.
func (g *Game[S, T, P]) ValidatePlayerAction(cu *User) (P, error) {
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

// ValidateCurrentPlayer performs basic validations for determining whether user is associated
// with a current player (i.e., a player whose turn it is in the game).
// If user is associated with current player, ValidateCurrentPlayer returns the associated player
// Otherwise, ValidateCurrentPlayer returns an error
func (g *Game[S, T, P]) ValidateCurrentPlayer(cu *User) (P, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cp, found := g.CurrentPlayerFor(cu)
	if !found {
		return cp, ErrPlayerNotFound
	}
	return cp, nil
}

// ValidateFinishTurn performs basic validations for determining whether user is associated
// with a player (i.e., a player whose turn it is in the game) that can finish their turn.
// If user can finish a turn, ValidateFinishTurn returns the associated player and their associate SubToken
// Otherwise, ValidateFinishTurn returns their associated SubToken and error
func (g *Game[S, T, P]) ValidateFinishTurn(ctx *gin.Context, cu *User) (P, SubToken, error) {
	token, err := getToken(ctx)
	if err != nil {
		var zerop P
		return zerop, token, err
	}

	cp, err := g.ValidateCurrentPlayer(cu)
	switch {
	case err != nil:
		return nil, token, err
	case !cp.getPerformedAction():
		return nil, token, fmt.Errorf("%s has yet to perform an action: %w", cu.Name, ErrValidation)
	case !cp.getCanFinish():
		return nil, token, fmt.Errorf("%s has performed an action, but has taken partial actions preventing a turn finish: %w", cu.Name, ErrValidation)
	default:
		return cp, token, nil
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

type uidsForPidser interface {
	UIDSForPIDS([]PID) []UID
}

// UIDSForPIDS returns the user ids for the users associated with the given player ids.
func (g *Game[S, T, P]) UIDSForPIDS(pids []PID) []UID {
	return pie.Map(pids, func(pid PID) UID { return g.UIDForPID(pid) })
}

// NextPlayer returns player after cp that satisfies all tests ts
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

func (cl *GameClient[GT, G]) endGame(ctx *gin.Context, g G) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	places := g.setFinishOrder(g.Compare)
	g.header().Status = Completed
	g.header().EndedAt = timestamppb.Now()
	g.header().Phase = "Game Over"

	stats, err := cl.getUStats(ctx, g.header().UserIDS...)
	if err != nil {
		JErr(ctx, err)
		return
	}
	stats = g.updateUStats(stats, g.playerStats(), g.playerUIDS())

	oldElos, newElos, err := cl.updateElo(ctx, g.header().users(), places)
	if err != nil {
		JErr(ctx, err)
		return
	}

	rs := g.getResults(oldElos, newElos)
	g.newEntry("game-results", H{"Results": rs})

	if err := cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if err := cl.txCommit(tx, g); err != nil {
			return err
		}

		if err := cl.txSaveUStats(tx, stats); err != nil {
			return err
		}

		return cl.txSaveElos(tx, newElos)
	}); err != nil {
		JErr(ctx, err)
		return
	}

	if err := g.sendEndGameNotifications(rs); err != nil {
		// log but otherwise ignore send errors
		Warnf("%v", err.Error())
	}
	ctx.JSON(http.StatusOK, nil)

}

func (g *Game[S, T, P]) setFinishOrder(compare compareFunc) placesMap {
	// Set to no current player
	g.SetCurrentPlayers()

	// sortedByScore(g.Players)
	g.sortPlayers(compare)

	place := 1
	places := make(placesMap, len(g.Players))
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
		const equalTo = 0
		if j := i + 1; j < len(g.Players) {
			p2 := g.Players[j]
			if compare(p1.PID(), p2.PID()) != equalTo {
				place++
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

// UpdateOrder reflects current order of players to the game header
func (g *Game[S, T, P]) UpdateOrder() {
	g.Header.OrderIDS = g.Players.PIDS()
}

type results []result

func (g *Game[S, T, P]) getResults(oldElos, newElos []elo) results {
	Debugf(msgEnter)
	defer Debugf(msgExit)

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

func (g *Game[S, T, P]) sendEndGameNotifications(rs results) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

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
	Debugf("WinnerIDS: %#v", g.Header.WinnerIDS)
	return pie.Map(g.Header.WinnerIDS, func(uid UID) string { return g.Header.NameFor(g.PlayerByUID(uid).PID()) })
}

func (g *Game[S, T, P]) addNewPlayers() {
	g.Players = make(Players[T, P], g.Header.NumPlayers)
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

// Start provides initial game setup and starts play of the game.
func (g *Game[S, T, P]) Start(h Header) PID {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	g.Header = h
	g.Header.Status = Running

	g.addNewPlayers()

	cp := pie.First(g.Players)
	g.SetCurrentPlayers(cp.PID())
	return cp.PID()
}

// Views implements part of Viewer interface
func (g *Game[S, T, P]) Views() ([]UID, []*Game[S, T, P]) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	return []UID{0}, []*Game[S, T, P]{g.ViewFor(0)}
}

// ViewFor implements part of Viewer interface
func (g *Game[S, T, P]) ViewFor(_ UID) *Game[S, T, P] {
	return g.DeepCopy()
}

// DeepCopy provides a deep copy of the game
func (g *Game[S, T, P]) DeepCopy() *Game[S, T, P] {
	return deepCopy(g)
}

func deepCopy[T any](obj T) T {
	v, err := json.Marshal(obj)
	if err != nil {
		Errorf("unable to marshal object: %v", err)
		panic("unable to marshal object")
	}

	var obj2 T
	err = json.Unmarshal(v, &obj2)
	if err != nil {
		Errorf("unable to unmarshal object: %v", err)
		panic("unable to unmarshal object")
	}
	return obj2
}

func (g *Game[S, T, P]) header() *Header {
	return &(g.Header)
}

func (g *Game[S, T, P]) id() string {
	return g.header().id()
}

func (g *Game[S, T, P]) setID(id string) {
	g.header().setID(id)
}

func (g *Game[S, T, P]) stack() *Stack {
	return &(g.header().Undo)
}

func (g *Game[S, T, P]) setStack(s *Stack) {
	g.header().Undo = *s
}

func (g *Game[S, T, P]) getIndex() *index {
	return &index{
		Header: *(g.header()),
		Rev:    g.stack().Committed,
	}
}
