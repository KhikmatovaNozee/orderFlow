package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"  warn  ", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"бред какой-то", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLevel(tt.input); got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	log := New("info")
	if log == nil {
		t.Fatal("New вернул nil")
	}
}

func TestFrom(t *testing.T) {
	t.Run("возвращает логгер из контекста", func(t *testing.T) {
		buf := &bytes.Buffer{}
		want := slog.New(slog.NewJSONHandler(buf, nil))

		ctx := Into(context.Background(), want)
		From(ctx).Info("привет")

		if !strings.Contains(buf.String(), "привет") {
			t.Error("сообщение ушло не в тот логгер")
		}
	})

	t.Run("без логгера в контексте отдаёт дефолтный", func(t *testing.T) {
		if From(context.Background()) == nil {
			t.Error("From вернул nil вместо дефолтного логгера")
		}
	})
}
