package sn

import (
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Player stats for a single game
type Stats struct {
	// Number of points scored by player
	Score int64
	// Number of games played at player count
	GamesPlayed int64
	// Number of games won at player count
	Won int64
	// Number of moves made by player
	Moves int64
	// Amount of time passed between player moves by player
	Think time.Duration
	// Position player finished (e.g., 1st, 2nd, etc.)
	Finish int
}

const ustatsKind = "UStats"

type UStat struct {
	// Below slices
	// Index 0: Total at all player counts
	// Index 1: Reserved
	// Index 2: Total 2P games
	// Index 3: Total 3P games
	// Index 4: Total 4P games
	// Index 5: Total 5P games
	// Index 6: Total 6P games

	ID UID
	// Number of games played
	Played []int64

	// Number of games won
	Won []int64
	// Number of points scored
	Scored []int64
	// Number of moves made by player
	Moves []int64
	// Amount of time passed between player moves by player
	Think []time.Duration
	// Average amount of time passed between player moves by player
	ThinkAvg []time.Duration
	// Sum of position finishes
	Finish []int64
	// Average finishing position
	FinishAvg []float32
	// Average Score
	ScoreAvg []float32
	// Win percentage
	WinPercentage []float32
	// Win percentage
	ExpectedWinPercentage []float32

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newUStat(uid UID, maxPlayers int) UStat {
	return UStat{
		// Key:                   newUStatsKey(uid),
		ID:                    uid,
		Played:                make([]int64, maxPlayers+1),
		Won:                   make([]int64, maxPlayers+1),
		Scored:                make([]int64, maxPlayers+1),
		Moves:                 make([]int64, maxPlayers+1),
		Think:                 make([]time.Duration, maxPlayers+1),
		ThinkAvg:              make([]time.Duration, maxPlayers+1),
		Finish:                make([]int64, maxPlayers+1),
		FinishAvg:             make([]float32, maxPlayers+1),
		ScoreAvg:              make([]float32, maxPlayers+1),
		WinPercentage:         make([]float32, maxPlayers+1),
		ExpectedWinPercentage: make([]float32, maxPlayers+1),
	}
}

func (cl Client[G, P]) ustatDocRef(uid UID) *firestore.DocumentRef {
	return cl.FS.Collection(ustatsKind).Doc(fmt.Sprintf("%d", uid))
}

func (g Game[P]) updateUStats(stats []UStat, pstats []*Stats, uids []UID) []UStat {
	var ustats = make([]UStat, len(stats))
	for i := range stats {
		ustats[i] = g.updateUStat(stats[i], pstats[i], uids[i])
	}
	return ustats
}

func (g Game[P]) updateUStat(stat UStat, pstats *Stats, uid UID) UStat {
	stat.Played[0]++
	stat.Played[g.NumPlayers]++
	for _, id := range g.WinnerIDS {
		if id == uid {
			stat.Won[0]++
			stat.Won[g.NumPlayers]++
			break
		}
	}

	stat.Moves[0] += pstats.Moves
	stat.Moves[g.NumPlayers] += pstats.Moves

	stat.Think[0] += pstats.Think
	stat.Think[g.NumPlayers] += pstats.Think

	stat.Scored[0] += int64(pstats.Score)
	stat.Scored[g.NumPlayers] += int64(pstats.Score)

	stat.Finish[0] += int64(pstats.Finish)
	stat.Finish[g.NumPlayers] += int64(pstats.Finish)

	if stat.Played[0] != 0 {
		stat.WinPercentage[0] = float32(stat.Won[0]) / float32(stat.Played[0])
		stat.ExpectedWinPercentage[0] = (float32(stat.Played[3])/3.0 + float32(stat.Played[4])/4.0 +
			float32(stat.Played[5])/5.0) / float32(stat.Played[0])
		stat.FinishAvg[0] = float32(stat.Finish[0]) / float32(stat.Played[0])
		stat.ScoreAvg[0] = float32(stat.Scored[0]) / float32(stat.Played[0])
	}

	if stat.Moves[0] != 0 {
		stat.ThinkAvg[0] = stat.Think[0] / time.Duration(stat.Moves[0])
	}

	if stat.Played[g.NumPlayers] > 0 {
		stat.WinPercentage[g.NumPlayers] = float32(stat.Won[g.NumPlayers]) / float32(stat.Played[g.NumPlayers])
		stat.FinishAvg[g.NumPlayers] = float32(stat.Finish[g.NumPlayers]) / float32(stat.Played[g.NumPlayers])
		stat.ScoreAvg[g.NumPlayers] = float32(stat.Scored[g.NumPlayers]) / float32(stat.Played[g.NumPlayers])
	}

	if g.NumPlayers != 0 {
		stat.ExpectedWinPercentage[g.NumPlayers] = 1.0 / float32(g.NumPlayers)
	}

	if stat.Moves[g.NumPlayers] != 0 {
		stat.ThinkAvg[g.NumPlayers] = stat.Think[g.NumPlayers] / time.Duration(stat.Moves[g.NumPlayers])
	}
	return stat

}

func (cl Client[G, P]) GetUStats(ctx *gin.Context, maxPlayers int, uids ...UID) ([]UStat, error) {
	Debugf(msgEnter)
	Debugf(msgExit)

	l := len(uids)
	ustats := make([]UStat, l)
	for i, uid := range uids {
		snap, err := cl.ustatDocRef(uid).Get(ctx)
		if status.Code(err) == codes.NotFound {
			ustats[i] = newUStat(uid, maxPlayers)
			ustats[i].CreatedAt = time.Now()
			continue
		}

		var stat UStat
		err = snap.DataTo(&stat)
		if err != nil {
			return nil, err
		}
		ustats[i] = stat
	}

	return ustats, nil
}

func (cl Client[G, P]) SaveUStatsIn(tx *firestore.Transaction, ustats []UStat) error {
	t := time.Now()
	for _, ustat := range ustats {
		ustat.UpdatedAt = t
		if err := tx.Set(cl.ustatDocRef(ustat.ID), ustat); err != nil {
			return err
		}
	}
	return nil
}
