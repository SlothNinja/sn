package sn

import (
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	elogo "github.com/kortemy/elo-go"
)

type elo struct {
	UID       UID
	Name      string
	EmailHash string
	GravType  string
	Rating    int
	UpdatedAt time.Time
}

func newEloDefault(u User) elo {
	const defaultRating = 1500
	return elo{
		UID:       u.ID,
		Name:      u.Name,
		EmailHash: u.EmailHash,
		GravType:  u.GravType,
		Rating:    defaultRating,
		UpdatedAt: time.Now(),
	}
}

func (cl *GameClient[GT, G]) eloDocRef(uid UID) *firestore.DocumentRef {
	return cl.eloCollectionRef().Doc(fmt.Sprintf("%d", uid))
}

func (cl *GameClient[GT, G]) eloCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection("Elo")
}

func (cl *GameClient[GT, G]) eloHistoryRef(uid UID) *firestore.CollectionRef {
	return cl.eloDocRef(uid).Collection("History")
}

type eloMap map[UID]elo
type placesMap map[UID]int

func updateEloFor(uid1 UID, elos eloMap, places placesMap) int {
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

func (cl *GameClient[GT, G]) txSaveElos(tx *firestore.Transaction, elos []elo) error {
	for i, elo := range elos {
		if err := tx.Set(cl.eloDocRef(elos[i].UID), elo); err != nil {
			return err
		}
		if err := tx.Create(cl.eloHistoryRef(elos[i].UID).NewDoc(), elo); err != nil {
			return err
		}
	}
	return nil
}

func (cl *GameClient[GT, G]) getElos(ctx *gin.Context, users ...User) ([]elo, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	refs := pie.Map(users, func(user User) *firestore.DocumentRef { return cl.eloDocRef(user.ID) })
	snaps, err := cl.FS.GetAll(ctx, refs)
	if err != nil {
		return nil, err
	}

	elos := make([]elo, len(snaps))
	for i, snap := range snaps {
		if !snap.Exists() {
			elos[i] = newEloDefault(users[i])
			continue
		}

		var elo elo
		if err := snap.DataTo(&elo); err != nil {
			return nil, err
		}
		elos[i] = elo
	}
	return elos, nil
}

// Update pulls current Elo from db and provides rating updates and deltas per results for users associated with uids.
// Returns ratings, updates, and current Elo (not updated) in same order as supplied uids
func (cl *GameClient[GT, G]) updateElo(ctx *gin.Context, us []User, places placesMap) ([]elo, []elo, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	oldElos, err := cl.getElos(ctx, us...)
	if err != nil {
		return nil, nil, err
	}

	eloMap := make(eloMap, len(oldElos))
	for i, elo := range oldElos {
		eloMap[us[i].ID] = elo
	}

	t := time.Now()
	newElos := make([]elo, len(us))
	for i, u := range us {
		newElos[i] = elo{
			UID:       u.ID,
			Name:      u.Name,
			EmailHash: u.EmailHash,
			GravType:  u.GravType,
			Rating:    updateEloFor(u.ID, eloMap, places),
			UpdatedAt: t,
		}
	}

	return oldElos, newElos, nil
}
