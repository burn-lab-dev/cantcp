package domain

import "testing"

func TestLogFormat_valid(t *testing.T) {
	tests := []struct {
		name   string
		format LogFormat
		want   bool
	}{
		{name: "text", format: LogFormatText, want: true},
		{name: "json", format: LogFormatJSON, want: true},
		{name: "unknown", format: "logfmt", want: false},
		{name: "empty", format: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.valid(); got != tt.want {
				t.Fatalf("valid(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}
