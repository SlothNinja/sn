package sn

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubToken represents a firebase subscription string used for sending WebPush notifications
type SubToken string

type subscription struct {
	Token SubToken
	Time  time.Time
}

type subscriptions struct {
	Subs []subscription
	Time time.Time
}

func (subs *subscriptions) toTokenStrings() []string {
	return pie.Map(subs.Subs, func(sub subscription) string { return string(sub.Token) })
}

func (cl *GameClient[GT, G]) subCollectionRef(gid string) *firestore.CollectionRef {
	return cl.gameDocRef(gid).Collection("Sub")
}

func (cl *GameClient[GT, G]) subDocRef(gid string, uid UID) *firestore.DocumentRef {
	return cl.subCollectionRef(gid).Doc(strconv.Itoa(int(uid)))
}

func (cl *GameClient[GT, G]) subInvCollectionRef(id string) *firestore.CollectionRef {
	return cl.invitationDocRef(id).Collection("Sub")
}

func (cl *GameClient[GT, G]) subInvDocRef(id string, uid UID) *firestore.DocumentRef {
	return cl.subInvCollectionRef(id).Doc(strconv.Itoa(int(uid)))
}

func (cl *GameClient[GT, G]) addSub(ctx *gin.Context, gid string, token SubToken, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if token == "" {
		return nil
	}

	t := time.Now()
	newSubs := &subscriptions{
		Subs: []subscription{subscription{Token: token, Time: t}},
		Time: t,
	}
	_, err := cl.subInvDocRef(gid, uid).Set(ctx, newSubs)
	return err
}

func (cl *GameClient[GT, G]) removeSubs(ctx *gin.Context, gid string, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	k := subscriptionKey(gid, uid)
	cl.Cache.Delete(k)
	_, err := cl.subDocRef(gid, uid).Delete(ctx)
	return err
}

func (cl *GameClient[GT, G]) updateSubs(ctx *gin.Context, gid string, token SubToken, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	if token == "" {
		return nil
	}

	subs, err := cl.getSubscriptions(ctx, gid, uid)
	if (err != nil) && (status.Code(err) != codes.NotFound) {
		return err
	}

	if subs == nil {
		subs = new(subscriptions)
	}

	found, changed := false, false
	newSubs := new(subscriptions)
	newSubs.Time = time.Now()
	const month = time.Hour * 24 * 30
	for _, sub := range subs.Subs {
		switch {
		case sub.Token == token:
			sub.Time = newSubs.Time
			found, changed = true, true
			newSubs.Subs = append(newSubs.Subs, sub)
		case time.Since(sub.Time).Hours() < month.Hours():
			newSubs.Subs = append(newSubs.Subs, sub)
		default:
			changed = true
		}
	}

	if !found {
		newSub := subscription{Token: token, Time: time.Now()}
		newSubs.Subs = append(newSubs.Subs, newSub)
		changed = true
	}

	if !changed {
		return nil
	}

	if _, err := cl.subDocRef(gid, uid).Set(ctx, newSubs); err != nil {
		return err
	}

	k := subscriptionKey(gid, uid)
	cl.Cache.Delete(k)
	return nil
}

// mcGetSubscription attempts to pull subscriptoin tokens from cache
func (cl *GameClient[GT, G]) mcGetSubscriptions(gid string, uid UID) (*subscriptions, bool) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	k := subscriptionKey(gid, uid)

	item, found := cl.Cache.Get(k)
	if !found {
		return nil, false
	}

	subs, ok := item.(*subscriptions)
	if !ok {
		cl.Cache.Delete(k)
		return nil, false
	}
	return subs, true
}

// dsGetSubscriptions attempts to pull datastore
func (cl *GameClient[GT, G]) dsGetSubscriptions(ctx context.Context, gid string, uid UID) (*subscriptions, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	snap, err := cl.subDocRef(gid, uid).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get subscriptions for: %v: %w", uid, err)
	}

	subs := new(subscriptions)
	if err := snap.DataTo(subs); err != nil {
		return nil, fmt.Errorf("unable to get data from snapshot: %w", err)
	}

	k := subscriptionKey(gid, uid)
	cl.Cache.Set(k, subs, 0)

	return subs, nil
}

func (cl *GameClient[GT, G]) getSubscriptions(ctx context.Context, gid string, uid UID) (*subscriptions, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	subs, found := cl.mcGetSubscriptions(gid, uid)
	if found {
		return subs, nil
	}

	return cl.dsGetSubscriptions(ctx, gid, uid)
}

func (cl *GameClient[GT, G]) getTokenStrings(ctx context.Context, gid string, uids []UID) ([]string, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	var tokens []string
	for _, uid := range uids {
		subs, err := cl.getSubscriptions(ctx, gid, uid)
		slog.Debug(fmt.Sprintf("uid: %v subs: %v", uid, subs))
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		tokens = slices.Concat(tokens, subs.toTokenStrings())
	}
	return tokens, nil
}

func (cl *GameClient[GT, G]) putSubscriptions(ctx context.Context, gid string, uid UID, subs *subscriptions) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.subDocRef(gid, uid).Set(ctx, subs)
	if err != nil {
		return err
	}

	k := subscriptionKey(gid, uid)
	cl.Cache.Delete(k)
	return nil
}

func subscriptionKey(gid string, uid UID) string {
	return fmt.Sprintf("subkey-%s-%d", gid, uid)
}

type tokenInput struct {
	Token SubToken
}

func getToken(ctx *gin.Context) (SubToken, error) {
	input := new(tokenInput)
	err := ctx.ShouldBind(input)
	return input.Token, err
}
