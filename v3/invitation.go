package sn

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

const invitationKind = "Invitation"
const hashKind = "Hash"

func updateTime() (t time.Time) { return }

type Invitation interface {
	FromForm(*gin.Context, User) ([]byte, error)
	Head() *Header
	Default()
	Start() (Game, PID, error)
}

type Tester[P any] struct {
	A P
}

func (t Tester[P]) GetA() P {
	return t.A
}

type tester struct {
	Tester[int]
}

func (t tester) test() int {
	return t.GetA()
}

func (cl Client) InvitationDocRef(id string) *firestore.DocumentRef {
	return cl.InvitationCollectionRef().Doc(id)
}

func (cl Client) InvitationCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(invitationKind)
}

func (cl Client) HashDocRef(id string) *firestore.DocumentRef {
	return cl.InvitationDocRef(id).Collection(hashKind).Doc("hash")
}

func GetInvitation[T Invitation](ctx *gin.Context, cl Client, inv T) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	id := getID(ctx)
	snap, err := cl.InvitationDocRef(id).Get(ctx)
	if err != nil {
		return err
	}

	if err = snap.DataTo(inv); err != nil {
		return err
	}

	inv.Head().ID = id
	return nil
}

func (cl Client) GetHash(ctx context.Context, id string) ([]byte, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	snap, err := cl.HashDocRef(id).Get(ctx)
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

func (cl Client) DeleteInvitation(ctx context.Context, id string) error {
	return cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return cl.DeleteInvitationIn(ctx, tx, id)
	})
}

func (cl Client) DeleteInvitationIn(ctx context.Context, tx *firestore.Transaction, id string) error {
	ref := cl.InvitationDocRef(id)
	if err := tx.Delete(ref); err != nil {
		return err
	}

	if err := tx.Delete(cl.HashDocRef(id)); err != nil {
		return err
	}
	return nil
}

func NewInvitationHandler[I Invitation](cl Client, inv I) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		inv.Default()
		ctx.JSON(http.StatusOK, gin.H{"Invitation": inv})
	}
}

func CreateInvitationHandler[I Invitation](cl Client, inv I) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		hash, err := inv.FromForm(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			ref := cl.InvitationCollectionRef().NewDoc()
			if err := tx.Create(ref, inv); err != nil {
				return err
			}

			if err := tx.Create(cl.HashDocRef(ref.ID), gin.H{"Hash": hash}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			JErr(ctx, err)
			return
		}

		// capture title before resetting to defaults
		title := inv.Head().Title
		inv.Default()
		ctx.JSON(http.StatusOK, gin.H{
			"Invitation": inv,
			"Message":    fmt.Sprintf("%s created game %q", cu.Name, title),
		})
	}
}

func AcceptHandler(cl Client, inv Invitation) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := GetInvitation(ctx, cl, inv); err != nil {
			JErr(ctx, err)
			return
		}

		var hash []byte
		if inv.Head().Private {
			hash, err = cl.GetHash(ctx, inv.Head().ID)
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

		start, err := inv.Head().AcceptWith(cu, []byte(obj.Password), hash)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if !start {
			inv.Head().UpdatedAt = updateTime()
			_, err = cl.InvitationDocRef(inv.Head().ID).Set(ctx, inv)
			if err != nil {
				JErr(ctx, err)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"Message": inv.Head().acceptGameMessage(cu)})
			return
		}

		cl.Log.Debugf("inv: %#v", inv)
		g, cpid, err := inv.Start()
		if err != nil {
			JErr(ctx, err)
			return
		}
		cl.Log.Debugf("g: %#v", g)

		if err := cl.FS.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			if err := SaveGameIn(ctx, cl, tx, g, cu); err != nil {
				return err
			}
			return cl.DeleteInvitationIn(ctx, tx, inv.Head().ID)
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
		ctx.JSON(http.StatusOK, gin.H{"Message": inv.Head().StartGameMessage(cpid)})
	}
}

func (h Header) acceptGameMessage(u User) string {
	return fmt.Sprintf("%s accepted game invitation: %s", u.Name, h.Title)
}

func (h Header) StartGameMessage(pid PID) string {
	return fmt.Sprintf("<div>Game: %s has started.</div><div></div><div><strong>%s</strong> is start player.</div>",
		h.Title, h.NameFor(pid))
}

func DropHandler(cl Client, inv Invitation) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		if err := GetInvitation(ctx, cl, inv); err != nil {
			JErr(ctx, err)
			return
		}

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		err = inv.Head().Drop(cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if len(inv.Head().UserIDS) != 0 {
			inv.Head().UpdatedAt = updateTime()
			_, err = cl.InvitationDocRef(inv.Head().ID).Set(ctx, inv)
		} else {
			err = cl.DeleteInvitation(ctx, inv.Head().ID)
		}
		if err != nil {
			JErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"Message": inv.Head().dropGameMessage(cu)})
	}
}

func (h Header) dropGameMessage(u User) string {
	return fmt.Sprintf("%s dropped from game invitation: %s", u.Name, h.Title)
}

func Commit[G Game](ctx context.Context, cl Client, g G, cu User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g.Head().Undo.Commit()
	return Save(ctx, cl, g, cu)
}

func Save[G Game](ctx context.Context, cl Client, g G, u User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	return cl.FS.RunTransaction(ctx, func(c context.Context, tx *firestore.Transaction) error {
		return SaveGameIn(ctx, cl, tx, g, u)
	})
}

func SaveGameIn[G Game](ctx context.Context, cl Client, tx *firestore.Transaction, g G, cu User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	g.Head().UpdatedAt = updateTime()

	if err := tx.Set(cl.GameDocRef(g.Head().ID, g.Head().Rev()), g); err != nil {
		return err
	}

	if err := tx.Set(cl.CommittedDocRef(g.Head().ID), g); err != nil {
		return err
	}

	uids, views := g.Views()
	for i, v := range views {
		if err := tx.Set(cl.ViewDocRef(g.Head().ID, uids[i]), v); err != nil {
			return err
		}
	}
	// for _, p := range g.Players {
	// 	if err := tx.Set(cl.ViewDocRef(g.id, g.uidForPID(p.ID)), g.viewFor(p)); err != nil {
	// 		return err
	// 	}
	// }
	return ClearCached[G](ctx, cl, g, cu)
}

func ClearCached[G Game](ctx context.Context, cl Client, g G, cu User) error {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	refs := cl.CachedCollectionRef(g.Head().ID).DocumentRefs(ctx)
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
		if cu.Admin || docRefFor(ref, cu.ID()) {
			_, err = ref.Delete(ctx)
			if err != nil {
				return err
			}
		}
	}

	_, err := cl.StackDocRef(g.Head().ID, cu.ID()).Delete(ctx)

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

func DetailsHandler(cl Client, inv Invitation) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cl.Log.Debugf(msgEnter)
		defer cl.Log.Debugf(msgExit)

		if err := GetInvitation(ctx, cl, inv); err != nil {
			JErr(ctx, err)
			return
		}

		cu, err := cl.RequireLogin(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		uids := make([]UID, len(inv.Head().UserIDS))
		copy(uids, inv.Head().UserIDS)

		if hasUID := pie.Any(inv.Head().UserIDS, func(id UID) bool { return id == cu.ID() }); !hasUID {
			uids = append(uids, cu.ID())
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
