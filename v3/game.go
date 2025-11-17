package sn

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"html/template"
	"slices"
	"strconv"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"github.com/mailjet/mailjet-apiv3-go"
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
	toIndex() *index
	newEntry(string, H)
	playerStats() []*Stats
	playerUIDS() []UID
	ptr[G]
	getResults([]elo, []elo) results
	sendEndGameNotifications(results) error
	setCurrentPlayerers
	setFinishOrder(compareFunc) (placesMap, placesSMap)
	starter
	statsFor(PID) *Stats
	updateUStats([]ustat, []*Stats, []UID) []ustat
	updateStatsFor(PID)
	uidsForPidser
}

// Viewer interface provides methods used to return game states suitable for viewing by a given user.
// In short, viewer should remove private game data that should not be exposed to a particular user.
// For example, removing data for cards in hands of other players.
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
	Start(Header) (PID, error)
}

type setCurrentPlayerers interface {
	SetCurrentPlayers(...PID) []PID
}

func getID(ctx *gin.Context) string {
	return ctx.Param("id")
}

func getUID(ctx *gin.Context) (UID, error) {
	obj := struct{ UID UID }{}
	if err := ctx.ShouldBindBodyWithJSON(&obj); err != nil {
		return 0, err
	}
	return obj.UID, nil
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
	if i := int(pid.ToUIndex()); i >= 0 && i < len(g.Header.UserIDS) {
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

// Compare implements Comparer interface.
// Essentially, provides a fallback/default compare which ranks players
// in descending order of score. In other words, the more a player scores
// the earlier they are in finish order.
func (g *Game[S, T, P]) Compare(pid1, pid2 PID) int {
	return cmp.Compare(g.PlayerByPID(pid2).getScore(), g.PlayerByPID(pid1).getScore())
}

func (g *Game[S, T, P]) sortPlayers(compare func(PID, PID) int) {
	slices.SortFunc(g.Players, func(p1, p2 P) int { return compare(p1.PID(), p2.PID()) })
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
		if u.GodMode && len(ps) == 1 {
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
func (g *Game[S, T, P]) ValidateFinishTurn(ctx *gin.Context, p P) (SubToken, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	switch token, err := getToken(ctx); {
	case err != nil:
		return "", err
	case !p.getPerformedAction():
		return "", fmt.Errorf("you have yet to perform an action: %w", ErrValidation)
	case !p.getCanFinish():
		return "", fmt.Errorf("you have performed an action, but have taken partial actions preventing a turn finish: %w", ErrValidation)
	default:
		return token, nil
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

func (g *Game[S, T, P]) setFinishOrder(compare compareFunc) (placesMap, placesSMap) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

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

	places2 := make(placesSMap, len(places))
	for id := range places {
		places2[strconv.Itoa(int(id))] = places[id]
	}

	return places, places2
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
	Debugf(msgEnter)
	defer Debugf(msgExit)

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
	return DeepCopy(g)
}

// DeepCopy returns a deep copy of obj
// obj must be suitable for marshalling via json.Marshal
func DeepCopy[T any](obj T) T {
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

func (g *Game[S, T, P]) toIndex() *index {
	return &index{
		Header: *(g.header()),
		Rev:    g.stack().Committed,
	}
}
