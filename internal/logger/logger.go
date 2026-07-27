package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type ctxKey struct{}

func Into(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func From(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}
