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

				// // Handle custom level values.
				// level := a.Value.Any().(slog.Level)
				// switch {
				// case level < slog.LevelDebug:
				// 	a.Value = slog.StringValue("DEFAULT")
				// case level < slog.LevelInfo:
				// 	a.Value = slog.StringValue("DEBUG")
				// case level < slog.LevelWarn:
				// 	a.Value = slog.IntValue(300)
				// case level < slog.LevelError:
				// 	a.Value = slog.IntValue(400)
				// default:
				// 	a.Value = slog.IntValue(500)
				// }
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
		err := (&level).UnmarshalText([]byte(s))
		if err != nil {
			return level
		}
	}
	return slog.LevelDebug
}
