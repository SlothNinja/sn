package sn

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

		g, err := cl.getGameFor(ctx, cu.ID)
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
		if err := cl.cacheRev(ctx, g, cu); err != nil {
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

		g, err := cl.getGameFor(ctx, cu.ID)
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

		if err := cl.commit(ctx, g, cu); err != nil {
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

		g, err := cl.getGameFor(ctx, cu.ID)
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
			cl.endGame(ctx, g, cu)
			return
		}
		notify := g.SetCurrentPlayers(result.NextPlayerIDS...)

		err = cl.commit(ctx, g, cu)
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
