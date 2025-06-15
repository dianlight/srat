package tlog

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const (
	LevelTrace  slog.Level = -8
	LevelDebug  slog.Level = slog.LevelDebug
	LevelInfo   slog.Level = slog.LevelInfo
	LevelNotice slog.Level = 2
	LevelWarn   slog.Level = slog.LevelWarn
	LevelError  slog.Level = slog.LevelError
	LevelFatal  slog.Level = 12
)

var programLevel = new(slog.LevelVar) // Info by default

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	isTerminal := isatty.IsTerminal(os.Stderr.Fd())

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      programLevel,
			TimeFormat: time.RFC3339,
			NoColor:    !isTerminal,
			AddSource:  true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					switch level {
					case LevelTrace:
						a.Value = slog.StringValue("TRACE")
					case LevelNotice:
						a.Value = slog.StringValue("NOTICE")
					case LevelFatal:
						a.Value = slog.StringValue("FATAL")
					}
				}
				return a
			},
		}),
	))

}

func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Log(context.Background(), slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...any) {
	slog.Log(context.Background(), slog.LevelInfo, msg, args...)
}

func Notice(msg string, args ...any) {
	slog.Log(context.Background(), LevelNotice, msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Log(context.Background(), slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	slog.Log(context.Background(), slog.LevelError, msg, args...)
}

func Fatal(msg string, args ...any) {
	slog.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

func SetLevel(level slog.Level) {
	programLevel.Set(level)
}

func GetLevel() slog.Level {
	return programLevel.Level()
}

func SetLevelFromString(level string) error {
	switch level {
	case "trace":
		programLevel.Set(LevelTrace)
	case "debug":
		programLevel.Set(LevelDebug)
	case "info":
		programLevel.Set(LevelInfo)
	case "notice":
		programLevel.Set(LevelNotice)
	case "warn", "warning":
		programLevel.Set(LevelWarn)
	case "error":
		programLevel.Set(LevelError)
	case "fatal":
		programLevel.Set(LevelFatal)
	default:
		return errors.New("invalid log level")
	}
	return nil
}
