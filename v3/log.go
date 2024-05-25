package sn

import (
	"context"
	"fmt"

	"log"
	"os"
	"runtime"
	"strings"

	"cloud.google.com/go/logging"
	"google.golang.org/api/option"
)

const (
	logLevel = "LOGLEVEL"

	LvlNone    = "NONE"
	LvlDebug   = "DEBUG"
	LvlInfo    = "INFO"
	LvlWarning = "WARNING"
	LvlError   = "ERROR"

	debugLabel   = "[DEBUG]"
	infoLabel    = "[INFO]"
	warningLabel = "[WARNING]"
	errorLabel   = "[ERROR]"
	panicLabel   = "[PANIC]"

	nodeEnv = "NODE_ENV"
)

var DefaultLevel = LvlDebug

func showLogFor(label string) bool {
	switch getLevel() {
	case LvlNone:
		return false
	case LvlDebug:
		return (label == debugLabel) || (label == infoLabel) || (label == warningLabel) || (label == errorLabel)
	case LvlInfo:
		return (label == infoLabel) || (label == warningLabel) || (label == errorLabel)
	case LvlWarning:
		return (label == warningLabel) || (label == errorLabel)
	case LvlError:
		return (label == errorLabel)
	default:
		return true
	}
}

func getLevel() string {
	v, found := os.LookupEnv(logLevel)
	if !found {
		return DefaultLevel
	}
	switch v {
	case LvlNone:
		return LvlNone
	case LvlDebug:
		return LvlDebug
	case LvlInfo:
		return LvlInfo
	case LvlWarning:
		return LvlWarning
	case LvlError:
		return LvlError
	default:
		return DefaultLevel
	}
}

func Debugf(tmpl string, args ...interface{}) {
	debugf(3, tmpl, args...)
}

func debugf(skip int, tmpl string, args ...interface{}) {
	if !showLogFor(debugLabel) {
		return
	}
	log.Printf(debugLabel+" "+caller(skip)+tmpl, args...)
}

func Infof(tmpl string, args ...interface{}) {
	infof(3, tmpl, args...)
}

func infof(skip int, tmpl string, args ...interface{}) {
	if !showLogFor(infoLabel) {
		return
	}
	log.Printf(infoLabel+" "+caller(skip)+tmpl, args...)
}

func Warningf(tmpl string, args ...interface{}) {
	warningf(3, tmpl, args...)
}

func warningf(skip int, tmpl string, args ...interface{}) {
	if !showLogFor(warningLabel) {
		return
	}
	log.Printf(warningLabel+" "+caller(skip)+tmpl, args...)
}

func Errorf(tmpl string, args ...interface{}) {
	errorf(3, tmpl, args...)
}

func errorf(skip int, tmpl string, args ...interface{}) {
	if !showLogFor(errorLabel) {
		return
	}
	log.Printf(errorLabel+" "+caller(skip)+tmpl, args...)
}

func Panicf(fmt string, args ...interface{}) {
	panicf(3, fmt, args...)
}

func panicf(skip int, tmpl string, args ...interface{}) {
	log.Panicf(panicLabel+" "+caller(skip)+tmpl, args...)
}

func caller(skip int) string {
	pc, file, line, _ := runtime.Caller(skip)
	files := strings.Split(file, "/")
	if lenFiles := len(files); lenFiles > 1 {
		file = files[lenFiles-1]
	}
	fun := runtime.FuncForPC(pc).Name()
	funs := strings.Split(fun, "/")
	if lenFuns := len(funs); lenFuns > 2 {
		fun = strings.Join(funs[len(funs)-2:], "/")
	}
	return fmt.Sprintf("%v#%v(L: %v)\n\t => ", file, fun, line)
}

func Printf(fmt string, args ...interface{}) {
	log.Printf(caller(3)+fmt, args...)
}

func NewLogClient(parent string, opts ...option.ClientOption) (*LogClient, error) {
	if !IsProduction() {
		return new(LogClient), nil
	}
	cl, err := logging.NewClient(context.Background(), parent, opts...)
	return &LogClient{cl}, err
}

type LogClient struct {
	*logging.Client
}

func (cl *LogClient) Close() error {
	return cl.Close()
}

type Logger struct {
	logID string
	*logging.Logger
}

func (cl *LogClient) Logger(logID string, opts ...logging.LoggerOption) *Logger {
	if !IsProduction() {
		return new(Logger)
	}
	return &Logger{logID: logID, Logger: cl.Client.Logger(logID, opts...)}
}

func (l *Logger) Debugf(tmpl string, args ...interface{}) {
	if !IsProduction() {
		debugf(3, tmpl, args...)
		return
	}

	if !showLogFor(debugLabel) {
		return
	}

	if l.Logger == nil {
		Warningf("missing logger")
	}

	l.StandardLogger(logging.Debug).Printf(debugLabel+" "+caller(3)+tmpl, args...)
}

func (l *Logger) StandardLogger(s logging.Severity) *log.Logger {
	if !IsProduction() || l == nil {
		return log.New(os.Stdout, "", 0)
	}
	return l.Logger.StandardLogger(s)
}

func (l *Logger) Infof(tmpl string, args ...interface{}) {
	if !IsProduction() {
		infof(3, tmpl, args...)
		return
	}

	if !showLogFor(infoLabel) {
		return
	}

	if l.Logger == nil {
		warningf(3, "missing logger")
		infof(3, tmpl, args...)
		return
	}

	l.StandardLogger(logging.Info).Printf(debugLabel+" "+caller(3)+tmpl, args...)
}

func (l *Logger) Warningf(tmpl string, args ...interface{}) {
	if !IsProduction() {
		warningf(3, tmpl, args...)
		return
	}

	if !showLogFor(infoLabel) {
		return
	}

	if l.Logger == nil {
		warningf(3, "missing logger")
		warningf(3, tmpl, args...)
		return
	}

	l.StandardLogger(logging.Warning).Printf(debugLabel+" "+caller(3)+tmpl, args...)
}

func (l *Logger) Errorf(tmpl string, args ...interface{}) {
	if !IsProduction() {
		errorf(3, tmpl, args...)
		return
	}

	if !showLogFor(infoLabel) {
		return
	}

	if l.Logger == nil {
		warningf(3, "missing logger")
		errorf(3, tmpl, args...)
		return
	}

	l.StandardLogger(logging.Error).Printf(debugLabel+" "+caller(3)+tmpl, args...)
}

func (l *Logger) Panicf(tmpl string, args ...interface{}) {
	if !IsProduction() {
		panicf(3, tmpl, args...)
		return
	}

	if l.Logger == nil {
		warningf(3, "missing logger")
		panicf(3, tmpl, args...)
		return
	}

	l.StandardLogger(logging.Error).Panicf(debugLabel+" "+caller(3)+tmpl, args...)
}

// IsProduction returns true if NODE_ENV environment variable is equal to "production".
// GAE sets NODE_ENV environement to "production" on deployment.
// NODE_ENV can be overridden in app.yaml configuration.
// func isProduction() bool {
// 	return os.Getenv(nodeEnv) == production
// }
