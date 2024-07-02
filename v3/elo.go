package sn

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	elogo "github.com/kortemy/elo-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const eloKind = "Elo"
const historyKind = "History"
const defaultRating = 1500

type Elo struct {
	ID        UID
	Rating    int
	UpdatedAt time.Time
}

func newEloDefault(uid UID) Elo {
	return Elo{ID: uid, Rating: defaultRating}
}

func (cl *GameClient[GT, G]) EloDocRef(uid UID) *firestore.DocumentRef {
	return cl.eloCollectionRef().Doc(fmt.Sprintf("%d", uid))
}

func (cl *GameClient[GT, G]) eloCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(eloKind)
}

func (cl *GameClient[GT, G]) EloHistoryRef(uid UID) *firestore.CollectionRef {
	return cl.EloDocRef(uid).Collection(historyKind)
}

// func (e Elo) IncompleteKey() *datastore.Key {
// 	if e.Key == nil || e.Key.Parent == nil {
// 		return nil
// 	}
// 	return datastore.IncompleteKey(eloKind, e.Key.Parent)
// }

func eloCopy(elo Elo) *Elo {
	return &elo
}

// type EloClient struct {
// 	*Client
// }

// func NewEloClient(snClient *Client, prefix string) *EloClient {
// 	client := &EloClient{
// 		Client: snClient,
// 	}
// 	return client
// 	// return client.addRoutes(prefix)
// }
//
// func (cl *EloClient) GetMulti(c *gin.Context, uids []UID) ([]*Elo, error) {
// 	ks := pie.Map(uids, func(uid UID) *datastore.Key { return newCurrentEloKey(uid) })
// 	elos := make([]*Elo, len(uids))
// 	err := cl.DS.GetMulti(c, ks, elos)
// 	return filterNoSuchEntity(elos, uids, err)
// }
//
// func (cl *EloClient) Get(c *gin.Context, uid UID) (*Elo, error) {
// 	elos, err := cl.GetMulti(c, []UID{uid})
// 	elos, err = filterNoSuchEntity(elos, []UID{uid}, err)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(elos) != 1 {
// 		return nil, fmt.Errorf("incorrect number of ratings returned")
// 	}
// 	return pie.First(elos), nil
// }

type Results map[int][]UID

// range through may has not guaranteed order
// places provides ordered list of map keys so map may be tranversed in order
func (rs Results) places() []int {
	var places []int
	for place := range rs {
		places = append(places, place)
	}
	sort.Slice(places, func(p1, p2 int) bool { return p1 < p2 })
	return places
}

func merrFilter(err error, ignore ...error) error {
	if err == nil {
		return err
	}

	merr, ok := err.(datastore.MultiError)
	if !ok {
		return err
	}

	for _, err1 := range merr {
		if !pie.Any(ignore, func(err2 error) bool { return err1 == err2 }) {
			return err
		}
	}
	return nil
}

type eloMap map[UID]Elo
type PlacesMap map[UID]int

func updateEloFor(uid1 UID, elos eloMap, places PlacesMap) int {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	var delta int
	elo := elogo.NewElo()
	for uid2 := range elos {
		if uid1 == uid2 {
			continue
		}
		score := 0.0
		if places[uid1] == places[uid2] {
			score = 0.5
		}
		// places are essentially first, second, third, etc.
		// thus, a lower place indicates better performance
		if places[uid1] < places[uid2] {
			score = 1
		}
		delta += elo.RatingDelta(elos[uid1].Rating, elos[uid2].Rating, score)
	}
	return elos[uid1].Rating + delta
}

func (cl *GameClient[GT, G]) SaveElosIn(tx *firestore.Transaction, elos []Elo) error {
	for _, elo := range elos {
		if err := tx.Set(cl.EloDocRef(elo.ID), elo); err != nil {
			return err
		}
		if err := tx.Create(cl.EloHistoryRef(elo.ID).NewDoc(), elo); err != nil {
			return err
		}
	}
	return nil
}

// // if merely missing from datastore, provide default elo entity
// func filterNoSuchEntity(elos []*Elo, uids []UID, err error) ([]*Elo, error) {
// 	if err == nil {
// 		return elos, err
// 	}
//
// 	// if no Elo entity associated with user, create initial Elo entity for user
//
// 	if l1, l2 := len(elos), len(uids); l1 != l2 {
// 		return elos, fmt.Errorf("len(elos):%d len(uids):%d: must be same length", l1, l2)
// 	}
//
// 	me, ok := err.(datastore.MultiError)
// 	if !ok {
// 		return nil, err
// 	}
//
// 	for i, e := range me {
//
// 		if e == datastore.ErrNoSuchEntity {
// 			elos[i] = newEloDefault(uids[i])
// 		} else if e != nil {
// 			return nil, err
// 		}
// 	}
// 	return elos, nil
// }

// // Update pulls current Elo from db and provides rating updates and deltas per results for users associated with uids.
// // Returns ratings, updates, and current Elo (not updated) in same order as supplied uids
// func (cl *EloClient) Update(c *gin.Context, uids []UID, places PlacesMap) ([]*Elo, []*Elo, error) {
//
// 	oldElos, err := cl.GetMulti(c, uids)
//
// 	oldElos, err = filterNoSuchEntity(oldElos, uids, err)
//
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	newElos := make([]*Elo, len(uids))
// 	eloMap := make(eloMap, len(oldElos))
// 	for i, elo := range oldElos {
// 		eloMap[uids[i]] = elo
// 		newElos[i] = eloCopy(*elo)
// 	}
//
// 	for i, uid := range uids {
// 		newElos[i].Rating = updateEloFor(uid, eloMap, places)
// 	}
//
// 	return oldElos, newElos, nil
// }

// Update pulls current Elo from db and provides rating updates and deltas per results for users associated with uids.
// Returns ratings, updates, and current Elo (not updated) in same order as supplied uids
func (cl *GameClient[GT, G]) updateElo(ctx *gin.Context, uids []UID, places PlacesMap) ([]Elo, []Elo, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	oldElos := make([]Elo, len(uids))
	for i, uid := range uids {
		snap, err := cl.EloDocRef(uid).Get(ctx)
		if status.Code(err) == codes.NotFound {
			oldElos[i] = newEloDefault(uid)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		var elo Elo
		if err := snap.DataTo(&elo); err != nil {
			return nil, nil, err
		}
		oldElos[i] = elo
	}

	eloMap := make(eloMap, len(oldElos))
	for i, elo := range oldElos {
		eloMap[uids[i]] = elo
	}

	t := time.Now()
	newElos := make([]Elo, len(uids))
	for i, uid := range uids {
		newElos[i] = Elo{
			ID:        uid,
			Rating:    updateEloFor(uid, eloMap, places),
			UpdatedAt: t,
		}
	}

	return oldElos, newElos, nil
}
