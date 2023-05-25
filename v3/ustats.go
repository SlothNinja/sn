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

func ustatDocRef(cl *firestore.Client, uid UID) *firestore.DocumentRef {
	return cl.Collection(ustatsKind).Doc(fmt.Sprintf("%d", uid))
}

func (h Header) UpdateUStats(stats []UStat, pstats []Stats, uids []UID) {
	for i := range stats {
		stats[i] = h.UpdateUStat(stats[i], pstats[i], uids[i])
	}
}

func (h Header) UpdateUStat(stat UStat, pstats Stats, uid UID) UStat {
	Debugf("stat: %#v", stat)
	Debugf("pstats: %#v", pstats)
	Debugf("uid: %#v", uid)
	stat.Played[0]++
	stat.Played[h.NumPlayers]++
	for _, id := range h.WinnerIDS {
		if id == uid {
			stat.Won[0]++
			stat.Won[h.NumPlayers]++
			break
		}
	}

	stat.Moves[0] += pstats.Moves
	stat.Moves[h.NumPlayers] += pstats.Moves

	stat.Think[0] += pstats.Think
	stat.Think[h.NumPlayers] += pstats.Think

	stat.Scored[0] += int64(pstats.Score)
	stat.Scored[h.NumPlayers] += int64(pstats.Score)

	stat.Finish[0] += int64(pstats.Finish)
	stat.Finish[h.NumPlayers] += int64(pstats.Finish)

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

	if stat.Played[h.NumPlayers] > 0 {
		stat.WinPercentage[h.NumPlayers] = float32(stat.Won[h.NumPlayers]) / float32(stat.Played[h.NumPlayers])
		stat.FinishAvg[h.NumPlayers] = float32(stat.Finish[h.NumPlayers]) / float32(stat.Played[h.NumPlayers])
		stat.ScoreAvg[h.NumPlayers] = float32(stat.Scored[h.NumPlayers]) / float32(stat.Played[h.NumPlayers])
	}

	if h.NumPlayers != 0 {
		stat.ExpectedWinPercentage[h.NumPlayers] = 1.0 / float32(h.NumPlayers)
	}

	if stat.Moves[h.NumPlayers] != 0 {
		stat.ThinkAvg[h.NumPlayers] = stat.Think[h.NumPlayers] / time.Duration(stat.Moves[h.NumPlayers])
	}
	return stat

}

func GetUStats(ctx *gin.Context, cl *firestore.Client, maxPlayers int, uids ...UID) ([]UStat, error) {
	Debugf(msgEnter)
	Debugf(msgExit)

	l := len(uids)
	ustats := make([]UStat, l)
	for i, uid := range uids {
		snap, err := ustatDocRef(cl, uid).Get(ctx)
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

func (cl Client[G, I]) SaveUStatsIn(tx *firestore.Transaction, ustats []UStat) error {
	t := time.Now()
	for _, ustat := range ustats {
		ustat.UpdatedAt = t
		if err := tx.Set(ustatDocRef(cl.FS, ustat.ID), ustat); err != nil {
			return err
		}
	}
	return nil
}
