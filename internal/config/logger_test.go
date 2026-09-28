package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		cfg      domain.ConfigLog
		wantSub  string
		notSub   string
		wantDbg  bool
		wantInfo bool
	}{
		{
			name:     "text info",
			cfg:      domain.ConfigLog{Level: domain.LogLevelInfo, Format: domain.LogFormatText},
			wantSub:  "level=INFO msg=hello",
			notSub:   "level=DEBUG msg=debug",
			wantInfo: true,
		},
		{
			name:     "json debug",
			cfg:      domain.ConfigLog{Level: domain.LogLevelDebug, Format: domain.LogFormatJSON},
			wantSub:  `"level":"INFO","msg":"hello"`,
			wantDbg:  true,
			wantInfo: true,
		},
		{
			name:     "error level filters info",
			cfg:      domain.ConfigLog{Level: domain.LogLevelError, Format: domain.LogFormatText},
			notSub:   "msg=hello",
			wantInfo: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger, levelVar := NewLogger(tt.cfg, &buf)
			if levelVar == nil {
				t.Fatal("NewLogger() returned a nil level variable")
			}
			logger.Debug("debug")
			logger.Info("hello")
			out := buf.String()
			if tt.wantSub != "" && !strings.Contains(out, tt.wantSub) {
				t.Fatalf("log output = %q, want a substring %q", out, tt.wantSub)
			}
			if tt.notSub != "" && strings.Contains(out, tt.notSub) {
				t.Fatalf("log output = %q, must not contain %q", out, tt.notSub)
			}
			if tt.wantDbg != strings.Contains(out, "debug") {
				t.Fatalf("debug visibility mismatch in %q", out)
			}
			if tt.wantInfo != strings.Contains(out, "hello") {
				t.Fatalf("info visibility mismatch in %q", out)
			}
		})
	}
}

func TestNewLogger_LevelCanBeRaised(t *testing.T) {
	var buf bytes.Buffer
	logger, levelVar := NewLogger(domain.ConfigLog{Level: domain.LogLevelError, Format: domain.LogFormatText}, &buf)
	logger.Debug("hidden")
	levelVar.Set(slog.LevelDebug)
	logger.Debug("visible")
	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatalf("log output = %q, the debug record must be filtered", out)
	}
	if !strings.Contains(out, "visible") {
		t.Fatalf("log output = %q, the debug record must pass after the level change", out)
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level domain.LogLevel
		want  slog.Level
	}{
		{name: "debug", level: domain.LogLevelDebug, want: slog.LevelDebug},
		{name: "info", level: domain.LogLevelInfo, want: slog.LevelInfo},
		{name: "warn", level: domain.LogLevelWarn, want: slog.LevelWarn},
		{name: "error", level: domain.LogLevelError, want: slog.LevelError},
		{name: "unknown falls back to info", level: "loud", want: slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlogLevel(tt.level); got != tt.want {
				t.Fatalf("SlogLevel(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}
