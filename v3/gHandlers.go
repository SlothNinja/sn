package sn

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
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

		g, uid, err := cl.getGame(ctx, cu)
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
		if err := cl.cacheRev(ctx, g, uid); err != nil {
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

		g, uid, err := cl.getGame(ctx, cu)
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

		if err := cl.commit(ctx, g, uid); err != nil {
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

		g, uid, err := cl.getGame(ctx, cu)
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
			if err := cl.endGame(ctx, g, uid); err != nil {
				JErr(ctx, err)
				return
			}
			ctx.JSON(http.StatusOK, nil)
			return
		}
		notify := g.SetCurrentPlayers(result.NextPlayerIDS...)

		err = cl.commit(ctx, g, uid)
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

		gid := getID(ctx)

		uid := cu.ID
		if cu.GodMode {
			var err error
			if uid, err = getUID(ctx); err != nil {
				JErr(ctx, err)
				return
			}
		}

		stack, err := cl.getStack(ctx, gid, uid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		// do nothing if stack does not change
		if !update(stack) {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		g, err := cl.getGameWithStack(ctx, gid, uid, stack)
		if err != nil {
			JErr(ctx, err)
			return
		}
		g.header().UpdatedAt = timestamppb.Now()

		if err := cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
			if err := cl.txUpdateViews(tx, g, uid); err != nil {
				return err
			}
			return cl.txSaveStack(tx, g, uid)
		}); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) abandonHandler(ctx *gin.Context) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cu, err := cl.RequireAdmin(ctx)
	if err != nil {
		JErr(ctx, err)
		return
	}

	gid := getID(ctx)
	index, err := cl.getIndex(ctx, gid)
	if err != nil {
		JErr(ctx, err)
		return
	}

	g, err := cl.getRev(ctx, gid, index.Rev)
	if err != nil {
		JErr(ctx, err)
		return
	}

	g.header().Status = Abandoned
	if err := cl.save(ctx, g, cu.ID); err != nil {
		JErr(ctx, err)
		return
	}

	msg := fmt.Sprintf("%s has been abandoned.", g.header().Title)
	ctx.JSON(http.StatusOK, H{"Message": msg})
}

func (cl *GameClient[GT, G]) reviveHandler(ctx *gin.Context) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	cu, err := cl.RequireAdmin(ctx)
	if err != nil {
		JErr(ctx, err)
		return
	}

	gid := getID(ctx)
	index, err := cl.getIndex(ctx, gid)
	if err != nil {
		JErr(ctx, err)
		return
	}

	g, err := cl.getRev(ctx, gid, index.Rev)
	if err != nil {
		JErr(ctx, err)
		return
	}

	g.header().Status = Running
	if err := cl.save(ctx, g, cu.ID); err != nil {
		JErr(ctx, err)
		return
	}

	msg := fmt.Sprintf("%s has been revived.", g.header().Title)
	ctx.JSON(http.StatusOK, H{"Message": msg})
}

func (cl *GameClient[GT, G]) rollbackHandler() gin.HandlerFunc {
	return cl.rollHandler((*Stack).rollbackward)
}

func (cl *GameClient[GT, G]) rollforwardHandler() gin.HandlerFunc {
	return cl.rollHandler((*Stack).rollforward)
}

func (cl *GameClient[GT, G]) rollHandler(update func(*Stack, Rev) bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		Debugf(msgEnter)
		defer Debugf(msgExit)

		cu, err := cl.RequireAdmin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		uid, err := getUID(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		obj := struct{ Rev }{}

		err = ctx.ShouldBindBodyWithJSON(&obj)
		if err != nil {
			JErr(ctx, err)
			return
		}

		gid := getID(ctx)
		stack, err := cl.getStack(ctx, gid, uid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		Debugf("rev: %v", obj.Rev)
		Debugf("stack: %#v", stack)

		// do nothing if stack does not change
		if !update(stack, obj.Rev) {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		g, err := cl.getGameWithStack(ctx, gid, cu.ID, stack)
		if err != nil {
			JErr(ctx, err)
			return
		}

		g.header().UpdatedAt = timestamppb.Now()

		err = cl.save(ctx, g, uid)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}
