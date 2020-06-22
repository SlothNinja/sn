package sn

import (
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"time"
)

//var store = sessions.NewCookieStore([]byte("slothninja-games-rocks"))

const (
	GAE_VERSION = "GAE_VERSION"
	NODE_ENV    = "NODE_ENV"
	production  = "production"
)

// IsProduction returns true if NODE_ENV environment variable is equal to "production".
// GAE sets NODE_ENV environement to "production" on deployment.
// NODE_ENV can be overridden in app.yaml configuration.
func IsProduction() bool {
	return os.Getenv(NODE_ENV) == production
}

func VersionID() string {
	return os.Getenv(GAE_VERSION)
}

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

const day = 24 * time.Hour
const month = 31 * day
const year = 365 * day

func LastUpdated(v time.Time) string {
	duration := time.Since(v)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%.0f sec", duration.Seconds())
	case duration < time.Hour:
		return fmt.Sprintf("%.0f min", duration.Minutes())
	case duration < day:
		return fmt.Sprintf("%.0f hour", duration.Hours())
	case duration < month:
		return fmt.Sprintf("%.0f day", duration.Hours()/24)
	case duration < year:
		return fmt.Sprintf("%0.f month", duration.Hours()/(24*31))
	default:
		return fmt.Sprintf("%d year", duration.Hours()/(24*31*365))
	}
}

func ToSentence(strings []string) (sentence string) {
	switch length := len(strings); length {
	case 0:
	case 1:
		sentence = strings[0]
	case 2:
		sentence = strings[0] + " and " + strings[1]
	default:
		for i, s := range strings {
			switch i {
			case 0:
				sentence += s
			case length - 1:
				sentence += ", and " + s
			default:
				sentence += ", " + s
			}
		}
	}
	return sentence
}
