package sn

import (
	"fmt"
	"html/template"
	"math/rand"
	"time"
)

//var store = sessions.NewCookieStore([]byte("slothninja-games-rocks"))

type VError struct {
	msgs []string
}

func IsVError(err error) bool {
	_, ok := err.(*VError)
	return ok
}

func NewVError(format string, args ...interface{}) *VError {
	return new(VError).AddMessagef(format, args...)
}

func (e *VError) Error() string {
	var s string
	for _, msg := range e.msgs {
		s += msg + "\n"
	}
	return s
}

func (e *VError) Errors() []string {
	return e.msgs
}

func (e *VError) IsNil() bool {
	return len(e.msgs) == 0
}

func (e *VError) AddMessagef(format string, args ...interface{}) *VError {
	e.msgs = append(e.msgs, fmt.Sprintf(format, args...))
	return e
}

func (e *VError) AppendMessages(err *VError) {
	e.msgs = append(e.msgs, err.msgs...)
}

func (e *VError) HTML() []template.HTML {
	m := make([]template.HTML, len(e.msgs))
	for i, msg := range e.msgs {
		m[i] = template.HTML(msg)
	}
	return m
}

var MyRand = rand.New(rand.NewSource(time.Now().UnixNano()))

//func Encode(src interface{}) ([]byte, error) {
//	buf := new(bytes.Buffer)
//	enc := gob.NewEncoder(buf)
//
//	if err := enc.Encode(src); err != nil {
//		return nil, err
//	}
//	return buf.Bytes(), nil
//}
//
//func Decode(dst interface{}, value []byte) error {
//	buf := bytes.NewBuffer(value)
//	dec := gob.NewDecoder(buf)
//	return dec.Decode(dst)
//}
