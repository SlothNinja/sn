package sn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/dusted-go/logging/prettylog"
	"github.com/dusted-go/logging/stackdriver"
)

func init() {
	if IsProduction() {
		slog.SetDefault(stackLogger())
		return
	}
	slog.SetDefault(prettyLogger())
}

const (
	msgEnter = "Entering"
	msgExit  = "Exiting"
)

func prettyLogger() *slog.Logger {
	return slog.New(prettylog.NewHandler(&slog.HandlerOptions{
		Level:       getPrettyLogLevel(),
		AddSource:   true,
		ReplaceAttr: nil,
	}))
}

func stackLogger() *slog.Logger {
	return slog.New(stackdriver.NewHandler(&stackdriver.HandlerOptions{
		MinLevel:  getStackLogLevel(),
		AddSource: true,
	}))
}

func getPrettyLogLevel() slog.Level {
	s, found := os.LookupEnv("LOGLEVEL")
	if found {
		var level slog.Level
		err := (&level).UnmarshalText([]byte(s))
		if err != nil {
			return level
		}
	}
	return slog.LevelDebug
}

func getStackLogLevel() slog.Level {
	s, found := os.LookupEnv("LOGLEVEL")
	if found {
		return stackdriver.ParseLogLevel(s).Level()
	}
	return slog.LevelDebug
}

// Debugf provides formatted debug messages
func Debugf(ctx context.Context, format string, args ...any) {
	logf(ctx, slog.LevelDebug, format, args...)
}

// Infof provides formatted debug messages
func Infof(ctx context.Context, format string, args ...any) {
	logf(ctx, slog.LevelInfo, format, args...)
}

// Warnf provides formatted debug messages
func Warnf(ctx context.Context, format string, args ...any) {
	logf(ctx, slog.LevelWarn, format, args...)
}

// Errorf provides formatted debug messages
func Errorf(ctx context.Context, format string, args ...any) {
	logf(ctx, slog.LevelError, format, args...)
}

func logf(ctx context.Context, level slog.Level, format string, args ...any) {
	logger := slog.Default()
	if !logger.Enabled(ctx, slog.LevelDebug) {
		return
	}

	pc, _, _, _ := runtime.Caller(2)
	r := slog.NewRecord(time.Now(), level, fmt.Sprintf(format, args...), pc)
	_ = logger.Handler().Handle(ctx, r)
}
