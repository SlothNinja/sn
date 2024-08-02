package sn

import (
	"log/slog"
	"os"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     getLogLevel(),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})))
}

const (
	msgEnter = "Entering"
	msgExit  = "Exiting"
)

func getLogLevel() slog.Level {
	s, found := os.LookupEnv("LOGLEVEL")
	if found {
		var level slog.Level
		err := (&level).UnmarshalJSON([]byte(s))
		if err != nil {
			return level
		}
	}
	return slog.LevelDebug
}
