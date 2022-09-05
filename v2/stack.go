package sn

type Stack struct {
	Current   int64 `json:"current"`
	Updated   int64 `json:"updated"`
	Committed int64 `json:"committed"`
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
