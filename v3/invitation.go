package sn

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Pallinder/go-randomdata"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

const invitationKind = "Invitation"
const hashKind = "Hash"

func updateTime() (t time.Time) { return }

// type Invitation[I any] interface {
// 	FromForm(*gin.Context, User) (I, []byte, error)
// 	Head() *Header
// 	Default() I
// }

type Invitation struct{ Header }

func (cl *GameClient[P, S]) invitationDocRef(id string) *firestore.DocumentRef {
	return cl.invitationCollectionRef().Doc(id)
}

func (cl *GameClient[P, S]) invitationCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(invitationKind)
}

func (cl *GameClient[P, S]) hashDocRef(id string) *firestore.DocumentRef {
	return cl.invitationDocRef(id).Collection(hashKind).Doc("hash")
}

func (cl *GameClient[P, S]) abortHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		if _, err := cl.RequireAdmin(ctx); err != nil {
			JErr(ctx, err)
			return
		}

		inv, err := cl.getInvitation(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		inv.Status = Aborted
		inv.UpdatedAt = updateTime()
		if _, err := cl.invitationDocRef(inv.ID).Set(ctx, inv); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[P, S]) getInvitation(ctx *gin.Context) (Invitation, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var inv Invitation

	id := getID(ctx)
	snap, err := cl.invitationDocRef(id).Get(ctx)
	if err != nil {
		return inv, err
	}

	if err = snap.DataTo(&inv); err != nil {
		return inv, err
	}

	inv.ID = id
	return inv, nil
}

func (cl *GameClient[P, S]) getHash(ctx context.Context, id string) ([]byte, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	snap, err := cl.hashDocRef(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	hashInf, err := snap.DataAt("Hash")
	if err != nil {
		return nil, err
	}

	hash, ok := hashInf.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for stored hash")
	}
	return hash, nil
}

func (cl *GameClient[P, S]) deleteInvitation(ctx context.Context, id string) error {
	return cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return cl.deleteInvitationIn(ctx, tx, id)
	})
}

func (cl *GameClient[P, S]) deleteInvitationIn(ctx context.Context, tx *firestore.Transaction, id string) error {
	ref := cl.invitationDocRef(id)
	if err := tx.Delete(ref); err != nil {
		return err
	}

	if err := tx.Delete(cl.hashDocRef(id)); err != nil {
		return err
	}
	return nil
}

func (cl *GameClient[P, S]) newInvitationHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		var inv Invitation
		inv.Title = randomdata.SillyName()

		ctx.JSON(http.StatusOK, gin.H{"Invitation": inv})
	}
}

func (cl *GameClient[P, S]) createInvitationHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.getCU(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		var inv Invitation
		inv, hash, err := FromForm(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			ref := cl.invitationCollectionRef().NewDoc()
			if err := tx.Create(ref, inv); err != nil {
				return err
			}

			if err := tx.Create(cl.hashDocRef(ref.ID), gin.H{"Hash": hash}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			JErr(ctx, err)
			return
		}

		var inv2 Invitation
		inv2.Title = randomdata.SillyName()
		ctx.JSON(http.StatusOK, gin.H{
			"Invitation": inv2,
			"Message":    fmt.Sprintf("%s created game %q", cu.Name, inv.Title),
		})
	}
}

func FromForm(ctx *gin.Context, cu User) (Invitation, []byte, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	obj := struct {
		Type       Type
		Title      string
		NumPlayers int
		OptString  string
		Password   string
	}{}

	err := ctx.ShouldBind(&obj)
	if err != nil {
		return Invitation{}, nil, err
	}

	var inv Invitation
	inv.Title = cu.Name + "'s Game"
	if obj.Title != "" {
		inv.Title = obj.Title
	}

	inv.Type = obj.Type
	inv.NumPlayers = obj.NumPlayers
	inv.OptString = obj.OptString
	inv.Status = Recruiting

	// nv.NumPlayers = defaultPlayers
	// f obj.NumPlayers >= minPlayers && obj.NumPlayers <= maxPlayers {
	//        inv.NumPlayers = obj.NumPlayers
	//

	// ounds := defaultRounds
	// f obj.RoundsPerPlayer >= minRounds && obj.RoundsPerPlayer <= maxRounds {
	//        rounds = obj.RoundsPerPlayer
	//
	// nv.OptString, err = encodeOptions(rounds)
	// f err != nil {
	//        return nil, nil, err
	//

	var hash []byte
	if len(obj.Password) > 0 {
		hash, err = bcrypt.GenerateFromPassword([]byte(obj.Password), bcrypt.DefaultCost)
		if err != nil {
			return Invitation{}, nil, err
		}
		inv.Private = true
	}
	inv.AddCreator(cu)
	inv.AddUser(cu)
	return inv, hash, nil
}

func (cl *GameClient[P, S]) acceptHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		inv, err := cl.getInvitation(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		var hash []byte
		if inv.Private {
			hash, err = cl.getHash(ctx, inv.ID)
			if err != nil {
				JErr(ctx, err)
				return
			}
		}

		obj := struct {
			Password string
		}{}

		err = ctx.ShouldBind(&obj)
		if err != nil {
			JErr(ctx, err)
			return
		}

		start, err := inv.AcceptWith(cu, []byte(obj.Password), hash)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if !start {
			inv.UpdatedAt = updateTime()
			_, err = cl.invitationDocRef(inv.ID).Set(ctx, inv)
			if err != nil {
				JErr(ctx, err)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"Message": inv.acceptGameMessage(cu)})
			return
		}

		g := new(Game[P, S])
		cp := g.Start(&(inv.Header))
		if err != nil {
			JErr(ctx, err)
			return
		}
		cl.Log.Debugf("g: %#v", g)

		if err := cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			if err := cl.txSave(ctx, tx, g, cu); err != nil {
				return err
			}
			return cl.deleteInvitationIn(ctx, tx, inv.ID)
		}); err != nil {
			JErr(ctx, err)
			return
		}

		// cl.sendTurnNotificationsTo(c, g, cp)
		// 	err = cl.sendNotifications(c, g)
		// 	if err != nil {
		// 		cl.Log.Warningf(err.Error())
		// 	}
		//
		ctx.JSON(http.StatusOK, gin.H{"Message": inv.startGameMessage(cp.getPID())})
	}
}

func (h Header) acceptGameMessage(u User) string {
	return fmt.Sprintf("%s accepted game invitation: %s", u.Name, h.Title)
}

func (h Header) startGameMessage(pid PID) string {
	return fmt.Sprintf("<div>Game: %s has started.</div><div></div><div><strong>%s</strong> is start player.</div>",
		h.Title, h.NameFor(pid))
}

func (cl *GameClient[P, S]) dropHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		inv, err := cl.getInvitation(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = inv.Drop(cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		Debugf("ID: %#v", inv.ID)
		if len(inv.UserIDS) != 0 {
			inv.UpdatedAt = updateTime()
			_, err = cl.invitationDocRef(inv.ID).Set(ctx, inv)
		} else {
			err = cl.deleteInvitation(ctx, inv.ID)
		}
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"Message": inv.dropGameMessage(cu)})
	}
}

func (h Header) dropGameMessage(u User) string {
	return fmt.Sprintf("%s dropped from game invitation: %s", u.Name, h.Title)
}

func (cl *GameClient[P, S]) commit(ctx context.Context, g *Game[P, S], cu User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g.Header.Undo.Commit()
	return cl.save(ctx, g, cu)
}

func (cl *GameClient[P, S]) save(ctx context.Context, g *Game[P, S], u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		return cl.txSave(ctx, tx, g, u)
	})
}

func (cl *GameClient[P, S]) txSave(ctx context.Context, tx *firestore.Transaction, g *Game[P, S], u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g.Header.UpdatedAt = updateTime()

	if err := tx.Set(cl.gameDocRef(g.Header.ID, g.Header.Rev()), g); err != nil {
		return err
	}

	if err := tx.Set(cl.committedDocRef(g.Header.ID), g); err != nil {
		return err
	}

	if viewer, ok := any(g).(Viewer[P, S]); ok {
		uids, views := viewer.Views()
		for i, v := range views {
			if err := tx.Set(cl.viewDocRef(g.Header.ID, uids[i]), v); err != nil {
				return err
			}
		}
	}
	return cl.clearCached(ctx, g, u)
}

func (cl *GameClient[P, S]) clearCached(ctx context.Context, g *Game[P, S], cu User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	refs := cl.cachedCollectionRef(g.Header.ID).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		// if current user is admin, clear all cached docs
		// otherwise clear only if cached doc is for current user
		if cu.Admin || docRefFor(ref, cu.ID) {
			_, err = ref.Delete(ctx)
			if err != nil {
				return err
			}
		}
	}

	_, err := cl.StackDocRef(g.Header.ID, cu.ID).Delete(ctx)

	return err
}

func docRefFor(ref *firestore.DocumentRef, uid UID) bool {
	ss := pie.Reverse(strings.Split(ref.ID, "-"))
	s := pie.Pop(&ss)
	if *s == "0" {
		s = pie.Pop(&ss)
	}
	return *s == fmt.Sprintf("%d", uid)
}

type detail struct {
	ID     int64
	ELO    int
	Played int64
	Won    int64
	WP     float32
}

func (cl *GameClient[P, S]) detailsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		inv, err := cl.getInvitation(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		uids := make([]UID, len(inv.getHeader().UserIDS))
		copy(uids, inv.getHeader().UserIDS)

		if hasUID := pie.Any(inv.getHeader().UserIDS, func(id UID) bool { return id == cu.ID }); !hasUID {
			uids = append(uids, cu.ID)
		}

		// elos, err := cl.Elo.GetMulti(c, uids)
		// if err != nil {
		// 	JErr(c, err)
		// 	return
		// }

		// ustats, err := cl.getUStats(c, uids...)
		// if err != nil {
		// 	JErr(c, err)
		// 	return
		// }

		// details := make([]detail, len(elos))
		// for i := range elos {
		// 	played, won, wp := ustats[i].Played, ustats[i].Won, ustats[i].WinPercentage
		// 	details[i] = detail{
		// 		ID:     elos[i].ID,
		// 		ELO:    elos[i].Rating,
		// 		Played: played[0],
		// 		Won:    won[0],
		// 		WP:     wp[0],
		// 	}
		// }

		// c.JSON(http.StatusOK, gin.H{"details": details})
	}
}
