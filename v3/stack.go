package sn

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/firestore"
)

// Stack provides an undo stack for a game
type Stack struct {
	Current   int
	Updated   int
	Committed int
}

// Undo updates stack to undo an action
func (s *Stack) Undo() bool {
	undo := s.Current > s.Committed
	if undo {
		s.Current--
	}
	return undo
}

// Update updates the stack for an action
func (s *Stack) Update() {
	s.Current++
	s.Updated = s.Current
}

// Reset resets the stack to the last committed action
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

// Commit commits an action to the stack
func (s *Stack) Commit() {
	s.Committed++
	s.Current, s.Updated = s.Committed, s.Committed
}

func (cl *GameClient[GT, G]) stackCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection("Stack")
}

func (cl *GameClient[GT, G]) stackDocRef(gid string, uid UID) *firestore.DocumentRef {
	return cl.stackCollectionRef().Doc(gid).Collection("For").Doc(fmt.Sprintf("%d", uid))
}

func (cl *GameClient[GT, G]) getStack(ctx context.Context, gid string, uid UID) (Stack, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	snap, err := cl.stackDocRef(gid, uid).Get(ctx)
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

	_, err := cl.stackDocRef(gid, uid).Set(ctx, &s)
	if err != nil {
		return err
	}
	return nil
}

func (cl *GameClient[GT, G]) deleteStack(ctx context.Context, gid string, uid UID) error {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	_, err := cl.stackDocRef(gid, uid).Delete(ctx)
	return err
}
