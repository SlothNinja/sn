package sn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/messaging"
	"github.com/Pallinder/go-randomdata"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type invitation struct{ Header }

func (cl *GameClient[GT, G]) invitationDocRef(id string) *firestore.DocumentRef {
	return cl.invitationCollectionRef().Doc(id)
}

func (cl *GameClient[GT, G]) invitationCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection("Invitation")
}

func (cl *GameClient[GT, G]) hashDocRef(id string) *firestore.DocumentRef {
	return cl.invitationDocRef(id).Collection("Hash").Doc("hash")
}

func (cl *GameClient[GT, G]) abortHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

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
		now := timestamppb.Now()
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
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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
	return cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return cl.txDeleteInvitation(tx, id)
	})
}

func (cl *GameClient[GT, G]) txDeleteInvitation(tx *firestore.Transaction, id string) error {
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
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		var inv invitation
		inv.Title = randomdata.SillyName()

		ctx.JSON(http.StatusOK, gin.H{"Invitation": inv})
	}
}

func (cl *GameClient[GT, G]) createInvitationHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

		cu, err := cl.getCU(ctx)
		if err != nil {
			JErr(ctx, err)
			return
		}

		var inv invitation
		inv, hash, token, err := fromForm(ctx, cu)
		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
			t := timestamppb.Now()
			inv.CreatedAt, inv.UpdatedAt = t, t
			ref := cl.invitationCollectionRef().NewDoc()
			if err := tx.Create(ref, inv); err != nil {
				return err
			}
			inv.ID = ref.ID

			if len(hash) > 0 {
				return tx.Create(cl.hashDocRef(inv.ID), gin.H{"Hash": hash})
			}
			return nil
		}); err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.updateSubs(ctx, inv.ID, token, cu.ID); err != nil {
			slog.Warn(fmt.Sprintf("attempted to update sub: %q: %v", token, err))
		}

		inv2 := inv
		inv2.Title = randomdata.SillyName()
		ctx.JSON(http.StatusOK, gin.H{
			"Invitation": inv2,
			"Message":    fmt.Sprintf("%s created game %q", cu.Name, inv.Title),
		})
	}
}

func fromForm(ctx *gin.Context, cu *User) (invitation, []byte, SubToken, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	obj := struct {
		Type       Type
		Title      string
		NumPlayers int
		OptString  string
		Password   string
		Token      SubToken
	}{}

	err := ctx.ShouldBind(&obj)
	if err != nil {
		return invitation{}, nil, "", err
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
			return invitation{}, nil, "", err
		}
		inv.Private = true
	}
	inv.addCreator(cu)
	inv.addUser(cu)
	return inv, hash, obj.Token, nil
}

func (cl *GameClient[GT, G]) acceptHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

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
			Token    SubToken
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

		if _, err := cl.FCM.SubscribeToTopic(ctx, []string{string(obj.Token)}, strconv.Itoa(int(cu.ID))); err != nil {
			Warnf("attempted to update sub: %q: %v", obj.Token, err)
		}

		// if err := cl.updateSubs(ctx, inv.ID, obj.Token, cu.ID); err != nil {
		// 	slog.Warn(fmt.Sprintf("attempted to update sub: %q: %v", obj.Token, err))
		// }

		go func() {
			message := &messaging.Message{
				Topic: strconv.Itoa(int(cu.ID)),
				Notification: &messaging.Notification{
					Title:    "You joined game",
					Body:     "Thanks for joining game.",
					ImageURL: "https://tammany.slothninja.com/logo.png",
				},
				Webpush: &messaging.WebpushConfig{
					FCMOptions: &messaging.WebpushFCMOptions{
						Link: "https://www.slothninja.com",
					},
				},
			}
			name, err := cl.FCM.Send(ctx, message)
			if err != nil {
				Warnf("attempted to send join notifications to: %v: %v", obj.Token, err)
			}
			Warnf("batch send name: %s", name)
		}()

		if !start {
			inv.UpdatedAt = timestamppb.Now()
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

		if err := cl.FS.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
			if err := cl.txSave(tx, g, cu); err != nil {
				return err
			}
			return cl.txDeleteInvitation(tx, inv.ID)
		}); err != nil {
			JErr(ctx, err)
			return
		}

		go func() {
			responses, err := cl.sendNotifications(ctx, g, g.getHeader().CPIDS)
			if err != nil {
				slog.Warn(fmt.Sprintf("attempted to send notifications to: %v: %v", g.getHeader().CPIDS, err))
			}
			slog.Warn(fmt.Sprintf("batch send response: %v", responses))
		}()

		ctx.JSON(http.StatusOK, gin.H{"Message": inv.startGameMessage(cpid)})
	}
}

// Returns (true, nil) if game should be started
func (inv *invitation) acceptWith(u *User, pwd, hash []byte) (bool, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	switch {
	case len(inv.UserIDS) >= int(inv.NumPlayers):
		return fmt.Errorf("game already has the maximum number of players: %w", ErrValidation)
	case inv.hasUser(u):
		return fmt.Errorf("%s has already accepted this invitation: %w", u.Name, ErrValidation)
	case len(hash) != 0:
		err := bcrypt.CompareHashAndPassword(hash, pwd)
		if err != nil {
			slog.Debug(err.Error())
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
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

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
			inv.UpdatedAt = timestamppb.Now()
			_, err = cl.invitationDocRef(inv.ID).Set(ctx, inv)
		} else {
			err = cl.deleteInvitation(ctx, inv.ID)
		}

		if err != nil {
			JErr(ctx, err)
			return
		}

		if err := cl.removeSubs(ctx, inv.ID, cu.ID); err != nil {
			slog.Warn(fmt.Sprintf("error removing subs for %v: %v", cu.ID, err))
		}

		ctx.JSON(http.StatusOK, gin.H{"Message": inv.dropGameMessage(cu)})
	}
}

func (inv *invitation) drop(u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if err := inv.validateDrop(u); err != nil {
		return err
	}

	inv.removeUser(u)
	return nil
}

func (inv *invitation) validateDrop(u *User) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

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
	i, found := inv.IndexFor(u2.ID)
	if !found {
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

type detail struct {
	ID     UID
	ELO    int
	Played int64
	Won    int64
	WP     float32
}

func (cl *GameClient[GT, G]) detailsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug(msgEnter)
		defer slog.Debug(msgExit)

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

		uids := make([]UID, len(inv.UserIDS))
		copy(uids, inv.UserIDS)

		us := make([]*User, len(inv.Users()))
		copy(us, inv.Users())

		if hasUID := pie.Any(inv.UserIDS, func(id UID) bool { return id == cu.ID }); !hasUID {
			uids = append(uids, cu.ID)
			us = append(us, cu)
		}

		elos, err := cl.getElos(ctx, us...)
		if err != nil {
			JErr(ctx, err)
			return
		}

		ustats, err := cl.getUStats(ctx, uids...)
		if err != nil {
			JErr(ctx, err)
			return
		}

		details := make([]detail, len(elos))
		for i := range elos {
			played, won, wp := ustats[i].Played, ustats[i].Won, ustats[i].WinPercentage
			details[i] = detail{
				ID:     uids[i],
				ELO:    elos[i].Rating,
				Played: played,
				Won:    won,
				WP:     wp,
			}
		}

		ctx.JSON(http.StatusOK, gin.H{"Details": details})
	}
}
