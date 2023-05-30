package sn

import (
	"fmt"

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

func (cl Client[G, I, P]) StackDocRef(id string, uid UID) *firestore.DocumentRef {
	return cl.StackCollectionRef().Doc(fmt.Sprintf("%s-%d", id, uid))
}

func (cl Client[G, I, P]) StackCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(stackKind)
}

func getID(ctx *gin.Context) string {
	return ctx.Param("id")
}

func (cl Client[G, I, P]) GetStack(ctx *gin.Context, uid UID) (s Stack, err error) {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	var snap *firestore.DocumentSnapshot
	if snap, err = cl.StackDocRef(getID(ctx), uid).Get(ctx); err != nil {
		return Stack{}, err
	}

	if err = snap.DataTo(&s); err != nil {
		return Stack{}, err
	}

	return s, err
}
