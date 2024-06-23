package sn

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Pallinder/go-randomdata"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const invitationKind = "Invitation"
const hashKind = "Hash"

type invitation struct{ Header }

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

func (cl *GameClient[GT, G]) getInvitation(ctx *gin.Context) (invitation, error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var inv invitation

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

		var inv invitation
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

		var inv invitation
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

func FromForm(ctx *gin.Context, cu *User) (invitation, []byte, error) {
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
		return invitation{}, nil, err
	}

	var inv invitation
	inv.Title = cu.Name + "'s Game"
	if obj.Title != "" {
		inv.Title = obj.Title
	}

	inv.Type = obj.Type
	inv.NumPlayers = obj.NumPlayers
	inv.OptString = obj.OptString
	inv.Status = Recruiting

	var hash []byte
	if len(obj.Password) > 0 {
		hash, err = bcrypt.GenerateFromPassword([]byte(obj.Password), bcrypt.DefaultCost)
		if err != nil {
			return invitation{}, nil, err
		}
		inv.Private = true
	}
	inv.addCreator(cu)
	inv.addUser(cu)
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

		start, err := inv.acceptWith(cu, []byte(obj.Password), hash)
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

		ctx.JSON(http.StatusOK, gin.H{"Message": inv.startGameMessage(cpid)})
	}
}

// Returns (true, nil) if game should be started
func (inv *invitation) acceptWith(u *User, pwd, hash []byte) (bool, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	err := inv.validateAcceptWith(u, pwd, hash)
	if err != nil {
		return false, err
	}

	inv.addUser(u)
	if len(inv.UserIDS) == int(inv.NumPlayers) {
		return true, nil
	}
	return false, nil
}

func (inv *invitation) validateAcceptWith(u *User, pwd, hash []byte) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	switch {
	case len(inv.UserIDS) >= int(inv.NumPlayers):
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case inv.hasUser(u):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case len(hash) != 0:
		err := bcrypt.CompareHashAndPassword(hash, pwd)
		if err != nil {
			Debugf(err.Error())
			return fmt.Errorf("%s provided incorrect password for Game %s: %w",
				u.Name, inv.Title, ErrValidation)
		}
		return nil
	default:
		return nil
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

		err = inv.drop(cu)
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

func (inv *invitation) drop(u *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	if err := inv.validateDrop(u); err != nil {
		return err
	}

	inv.removeUser(u)
	return nil
}

func (inv *invitation) validateDrop(u *User) error {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	switch {
	case inv.Status != Recruiting:
		return fmt.Errorf("game is no longer recruiting, thus %s can't drop: %w", u.Name, ErrValidation)
	case !inv.hasUser(u):
		return fmt.Errorf("%s has not joined this game, thus %s can't drop: %w", u.Name, u.Name, ErrValidation)
	}
	return nil
}

func (inv *invitation) dropGameMessage(u *User) string {
	return fmt.Sprintf("%s dropped from game invitation: %s", u.Name, inv.Title)
}

func (h *Header) hasUser(u *User) bool {
	return pie.Contains(h.UserIDS, u.ID)
}

func (inv *invitation) removeUser(u2 *User) {
	i := inv.IndexFor(u2.ID)
	if i == UIndexNotFound {
		return
	}

	start := int(i)
	end := start + 1

	inv.UserIDS = slices.Delete(inv.UserIDS, start, end)
	inv.UserNames = slices.Delete(inv.UserNames, start, end)
	inv.UserEmails = slices.Delete(inv.UserEmails, start, end)
	inv.UserEmailHashes = slices.Delete(inv.UserEmailHashes, start, end)
	inv.UserEmailNotifications = slices.Delete(inv.UserEmailNotifications, start, end)
	inv.UserGravTypes = slices.Delete(inv.UserGravTypes, start, end)
}

func (inv *invitation) addUser(u *User) {
	inv.UserIDS = append(inv.UserIDS, u.ID)
	inv.UserNames = append(inv.UserNames, u.Name)
	inv.UserEmails = append(inv.UserEmails, u.Email)
	inv.UserEmailHashes = append(inv.UserEmailHashes, u.EmailHash)
	inv.UserEmailNotifications = append(inv.UserEmailNotifications, u.EmailNotifications)
	inv.UserGravTypes = append(inv.UserGravTypes, u.GravType)
}

func (inv *invitation) addCreator(u *User) {
	inv.CreatorID = u.ID
	inv.CreatorName = u.Name
	inv.CreatorEmail = u.Email
	inv.CreatorEmailHash = u.EmailHash
	inv.CreatorEmailNotifications = u.EmailNotifications
	inv.CreatorGravType = u.GravType
}
