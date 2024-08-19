package sn

import (
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
)

type ustat struct {
	// UID of user
	ID UID

	// Number of games played
	Played int64

	// Number of games won
	Won int64

	// Number of points scored
	Scored int64

	// Number of moves made by player
	Moves int64

	// Amount of time passed between player moves by player
	Think time.Duration

	// Average amount of time passed between player moves by player
	ThinkAvg time.Duration

	// Sum of position finishes
	Finish int64

	// Average finishing position
	FinishAvg float32

	// Average Score
	ScoreAvg float32

	// Win percentage
	WinPercentage float32

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newUStat(uid UID) ustat {
	return ustat{ID: uid}
}

func (cl *GameClient[GT, G]) ustatDocRef(uid UID) *firestore.DocumentRef {
	return cl.FS.Collection("UStat").Doc(fmt.Sprintf("%d", uid))
}

func (g *Game[S, T, P]) updateUStats(stats []ustat, pstats []*Stats, uids []UID) []ustat {
	var ustats = make([]ustat, len(stats))
	for i := range stats {
		ustats[i] = g.updateUStat(stats[i], pstats[i], uids[i])
	}
	return ustats
}

func (g *Game[S, T, P]) updateUStat(stat ustat, pstats *Stats, uid UID) ustat {
	stat.Played++
	for _, id := range g.Header.WinnerIDS {
		if id == uid {
			stat.Won++
			break
		}
	}

	stat.Moves += pstats.Moves
	stat.Think += pstats.Think
	stat.Scored += int64(pstats.Score)
	stat.Finish += int64(pstats.Finish)
	if stat.Played != 0 {
		stat.WinPercentage = float32(stat.Won) / float32(stat.Played)
		stat.FinishAvg = float32(stat.Finish) / float32(stat.Played)
		stat.ScoreAvg = float32(stat.Scored) / float32(stat.Played)
	}

	if stat.Moves != 0 {
		stat.ThinkAvg = stat.Think / time.Duration(stat.Moves)
	}

	return stat
}

func (cl *GameClient[GT, G]) getUStats(ctx *gin.Context, uids ...UID) ([]ustat, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	refs := pie.Map(uids, func(uid UID) *firestore.DocumentRef { return cl.ustatDocRef(uid) })
	snaps, err := cl.FS.GetAll(ctx, refs)
	if err != nil {
		return nil, err
	}

	ustats := make([]ustat, len(snaps))
	for i, snap := range snaps {
		if !snap.Exists() {
			ustats[i] = newUStat(uids[i])
			continue
		}

		var ustat ustat
		if err := snap.DataTo(&ustat); err != nil {
			return nil, err
		}
		ustats[i] = ustat
	}
	return ustats, nil
}

func (cl *GameClient[GT, G]) txSaveUStats(tx *firestore.Transaction, ustats []ustat) error {
	t := time.Now()
	for _, ustat := range ustats {
		ustat.UpdatedAt = t
		if err := tx.Set(cl.ustatDocRef(ustat.ID), ustat); err != nil {
			return err
		}
	}
	return nil
}
