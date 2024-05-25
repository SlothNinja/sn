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

// func updateTime() (t time.Time) { return }

// type Invitation[I any] interface {
// 	FromForm(*gin.Context, User) (I, []byte, error)
// 	Head() *Header
// 	Default() I
// }

type Invitation struct{ Header }

func (cl *GameClient[GT, G]) invitationDocRef(id string) *firestore.DocumentRef {
	return cl.invitationCollectionRef().Doc(id)
}

func (cl *GameClient[GT, G]) invitationCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(invitationKind)
}

func (cl *GameClient[GT, G]) hashDocRef(id string) *firestore.DocumentRef {
	return cl.invitationDocRef(id).Collection(hashKind).Doc("hash")
}

func (cl *GameClient[GT, G]) abortHandler() gin.HandlerFunc {
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
		now := time.Now()
		inv.UpdatedAt = now
		inv.EndedAt = now
		if _, err := cl.invitationDocRef(inv.ID).Set(ctx, inv); err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, nil)
	}
}

func (cl *GameClient[GT, G]) getInvitation(ctx *gin.Context) (Invitation, error) {
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

func (cl *GameClient[GT, G]) getHash(ctx context.Context, id string) ([]byte, error) {
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

func (cl *GameClient[GT, G]) deleteInvitation(ctx context.Context, id string) error {
	return cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return cl.deleteInvitationIn(ctx, tx, id)
	})
}

func (cl *GameClient[GT, G]) deleteInvitationIn(ctx context.Context, tx *firestore.Transaction, id string) error {
	ref := cl.invitationDocRef(id)
	if err := tx.Delete(ref); err != nil {
		return err
	}

	if err := tx.Delete(cl.hashDocRef(id)); err != nil {
		return err
	}
	return nil
}

func (cl *GameClient[GT, G]) newInvitationHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		var inv Invitation
		inv.Title = randomdata.SillyName()

		ctx.JSON(http.StatusOK, gin.H{"Invitation": inv})
	}
}

func (cl *GameClient[GT, G]) createInvitationHandler() gin.HandlerFunc {
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

		if err := cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
			t := time.Now()
			inv.CreatedAt, inv.UpdatedAt = t, t
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

		inv2 := inv
		inv2.Title = randomdata.SillyName()
		cl.Log.Debugf("inv2: %#v", inv2)
		ctx.JSON(http.StatusOK, gin.H{
			"Invitation": inv2,
			"Message":    fmt.Sprintf("%s created game %q", cu.Name, inv.Title),
		})
	}
}

func FromForm(ctx *gin.Context, cu *User) (Invitation, []byte, error) {
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

func (cl *GameClient[GT, G]) acceptHandler() gin.HandlerFunc {
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
			inv.UpdatedAt = time.Now()
			_, err = cl.invitationDocRef(inv.ID).Set(ctx, inv)
			if err != nil {
				JErr(ctx, err)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"Message": inv.acceptGameMessage(cu)})
			return
		}

		g := G(new(GT))
		cpid := g.Start(inv.Header)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
			if err := cl.txSave(c, tx, g, cu); err != nil {
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
		ctx.JSON(http.StatusOK, gin.H{"Message": inv.startGameMessage(cpid)})
	}
}

func (h Header) acceptGameMessage(u *User) string {
	return fmt.Sprintf("%s accepted game invitation: %s", u.Name, h.Title)
}

func (h Header) startGameMessage(pid PID) string {
	return fmt.Sprintf("<div>Game: %s has started.</div><div></div><div><strong>%s</strong> is start player.</div>",
		h.Title, h.NameFor(pid))
}

func (cl *GameClient[GT, G]) dropHandler() gin.HandlerFunc {
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

		if len(inv.UserIDS) != 0 {
			inv.UpdatedAt = time.Now()
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

func (h Header) dropGameMessage(u *User) string {
	return fmt.Sprintf("%s dropped from game invitation: %s", u.Name, h.Title)
}

func (cl *GameClient[GT, G]) commit(ctx *gin.Context, g G, u *User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		return cl.txCommit(c, tx, g, u)
	})
}

func (cl *GameClient[GT, G]) txCommit(ctx context.Context, tx *firestore.Transaction, g G, u *User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	h := g.getHeader()

	gc, err := cl.txGetCommitted(tx, h.ID)
	if err != nil {
		return err
	}

	if h.UpdatedAt != gc.getHeader().UpdatedAt {
		return fmt.Errorf("unexpected game change")
	}

	if err := cl.clearCached(ctx, h.ID, h.Undo.Committed, u.ID); err != nil {
		return err
	}

	h.Undo.Commit()
	return cl.txSave(ctx, tx, g, u)
}

func (cl *GameClient[GT, G]) save(ctx *gin.Context, g G, u *User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		//h := g.getHeader()
		//hid := h.ID
		//committed := h.Undo.Committed

		// if err := cl.txSave(tx, g, u); err != nil {
		// 	return err
		// }
		// return cl.clearCached(tx, hid, committed, u.ID)
		return cl.txSave(c, tx, g, u)
	})
}

func (cl *GameClient[GT, G]) txSave(ctx context.Context, tx *firestore.Transaction, g G, u *User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	h := g.getHeader()
	h.UpdatedAt = time.Now()

	if err := tx.Set(cl.revDocRef(h.ID, h.Rev()), g); err != nil {
		return err
	}

	if err := tx.Set(cl.indexDocRef(h.ID), h); err != nil {
		return err
	}

	// By implementing Views interface, game may provide a customized view for each user.
	// Primarily used to ensure hidden game information not leaked to users via json objects
	// sent to browsers.
	uids, views := g.Views()
	for i, v := range views {
		if err := tx.Set(cl.viewDocRef(h.ID, h.Rev(), uids[i]), v); err != nil {
			return err
		}
	}
	return cl.clearCached(ctx, h.ID, h.Undo.Committed, u.ID)
}

func (cl *GameClient[GT, G]) clearCached(ctx context.Context, gid string, rev int, uid UID) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	refs := cl.cachedCollectionRef(gid, rev, uid).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		_, err = ref.Delete(ctx)
		if err != nil {
			return err
		}
	}

	refs = cl.fullyCachedCollectionRef(gid, rev, uid).DocumentRefs(ctx)
	for {
		ref, err := refs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		_, err = ref.Delete(ctx)
		if err != nil {
			return err
		}
	}

	return cl.deleteStack(ctx, gid, uid)
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

func (cl *GameClient[GT, G]) detailsHandler() gin.HandlerFunc {
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
