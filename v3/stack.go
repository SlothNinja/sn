package sn

import (
	"context"
	"strconv"

	"cloud.google.com/go/firestore"
)

// Stack provides an undo stack for a game
type Stack struct {
	Current   int
	Updated   int
	Committed int
	UpdateEnd int
	CommitEnd int
}

// undo updates stack to undo an action
func (s *Stack) undo() bool {
	undo := s.Current > s.Committed
	if undo {
		s.Current--
	}
	return undo
}

// update updates the stack for an action
func (s *Stack) update() {
	s.Current++
	s.Updated = s.Current
	s.UpdateEnd = max(s.UpdateEnd, s.Updated)
}

// reset resets the stack to the last committed action
func (s *Stack) reset() bool {
	reset := s.Current != s.Committed || s.Updated != s.Current
	if reset {
		s.Current, s.Updated = s.Committed, s.Committed
	}
	return reset
}

// redo moves the undo stack forward
func (s *Stack) redo() bool {
	redo := s.Updated > s.Committed && s.Current < s.Updated
	if redo {
		s.Current++
	}
	return redo
}

// commit commits an action to the stack
func (s *Stack) commit() {
	s.Committed++
	s.Current, s.Updated = s.Committed, s.Committed
	s.UpdateEnd = max(s.UpdateEnd, s.Committed)
	s.CommitEnd = max(s.CommitEnd, s.Committed)
}

// end returns the larger of UpdateEnd and CommitEnd
func (s *Stack) end() int {
	return max(s.UpdateEnd, s.CommitEnd)
}

// trunc sets UpdateEnd and CommitEnd to current Committed
func (s *Stack) trunc() {
	s.UpdateEnd, s.CommitEnd = s.Committed, s.Committed
}

// rollbackward rolls the stack backward to rev
// returns true if rolling back to rev was possible
// otherwise returns false and does not update stack
func (s *Stack) rollbackward(rev int) bool {
	rollbackward := s.Current == s.Committed && s.Committed > 0 && rev >= 0 && rev < s.Current
	if rollbackward {
		s.Current, s.Updated, s.Committed = rev, rev, rev
	}
	return rollbackward
}

// Rollbackward rolls the stack forward to rev
// returns true if rolling forward to rev was possible
// otherwise returns false and does not update stack
func (s *Stack) rollforward(rev int) bool {
	rollforward := s.Current == s.Committed && s.Committed < s.CommitEnd && rev <= s.CommitEnd && rev > s.Current
	if rollforward {
		s.Current, s.Updated, s.Committed = rev, rev, rev
	}
	return rollforward
}

func (cl *GameClient[GT, G]) stackCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection("Stack")
}

func (cl *GameClient[GT, G]) stackDocRef(gid string, uid UID) *firestore.DocumentRef {
	return cl.stackCollectionRef().Doc(gid).Collection("For").Doc(strconv.Itoa(int(uid)))
}

func (cl *GameClient[GT, G]) getStack(ctx context.Context, gid string, uid UID) (*Stack, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	snap, err := cl.stackDocRef(gid, uid).Get(ctx)
	if err != nil {
		return nil, err
	}

	stack := new(Stack)
	err = snap.DataTo(stack)
	if err != nil {
		return nil, err
	}

	return stack, nil
}
