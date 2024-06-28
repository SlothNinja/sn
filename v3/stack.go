package sn

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

type Stack struct {
	Current   int
	Updated   int
	Committed int
}

func (s *Stack) Undo() bool {
	undo := s.Current > s.Committed
	if undo {
		s.Current--
	}
	return undo
}

func (s *Stack) Update() {
	s.Current++
	s.Updated = s.Current
}

func (s *Stack) Reset() bool {
	reset := s.Current != s.Committed || s.Updated != s.Current
	if reset {
		s.Current, s.Updated = s.Committed, s.Committed
	}
	return reset
}

// Redo moves the undo stack forward
func (s *Stack) Redo() bool {
	redo := s.Updated > s.Committed && s.Current < s.Updated
	if redo {
		s.Current++
	}
	return redo
}

func (s *Stack) Commit() {
	s.Committed++
	s.Current, s.Updated = s.Committed, s.Committed
}

const stackKind = "Stack"

// func stackKey(id int64, uid sn.UID) *datastore.Key {
// 	return datastore.NameKey(stackKind, "stack", cachedRootKey(id, uid))
// }

// func stackDocRef(cl *firestore.Client, id string, uid sn.UID) *firestore.DocumentRef {
// 	return cl.Collection(stackKind).Doc(fmt.Sprintf("%s-%d", id, uid))
// }

func getID(ctx *gin.Context) string {
	return ctx.Param("id")
}

func (cl *GameClient[GT, G]) StackCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(stackKind)
}

func (cl *GameClient[GT, G]) StackDocRef(gid string, uid UID) *firestore.DocumentRef {
	return cl.StackCollectionRef().Doc(gid).Collection("For").Doc(fmt.Sprintf("%d", uid))
}

func (cl *GameClient[GT, G]) getStack(ctx context.Context, gid string, uid UID) (Stack, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	snap, err := cl.StackDocRef(gid, uid).Get(ctx)
	if err != nil {
		return Stack{}, err
	}

	var stack Stack
	err = snap.DataTo(&stack)
	if err != nil {
		return Stack{}, err
	}

	return stack, err
}

func (cl *GameClient[GT, G]) setStack(ctx context.Context, gid string, uid UID, s Stack) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.StackDocRef(gid, uid).Set(ctx, &s)
	if err != nil {
		return err
	}
	return nil
}

func (cl *GameClient[GT, G]) deleteStack(ctx context.Context, gid string, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.StackDocRef(gid, uid).Delete(ctx)
	return err
}
