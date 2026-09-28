package domain

import "testing"

func TestLogLevel_valid(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  bool
	}{
		{name: "debug", level: LogLevelDebug, want: true},
		{name: "info", level: LogLevelInfo, want: true},
		{name: "warn", level: LogLevelWarn, want: true},
		{name: "error", level: LogLevelError, want: true},
		{name: "unknown", level: "trace", want: false},
		{name: "empty", level: "", want: false},
		{name: "case matters", level: "INFO", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.valid(); got != tt.want {
				t.Fatalf("valid(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}
