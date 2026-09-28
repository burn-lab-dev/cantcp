package domain

import (
	"errors"
	"testing"
)

func TestConfigLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigLog
		wantErr string
	}{
		{
			name: "text info",
			cfg:  ConfigLog{Level: LogLevelInfo, Format: LogFormatText},
		},
		{
			name: "json debug",
			cfg:  ConfigLog{Level: LogLevelDebug, Format: LogFormatJSON},
		},
		{
			name:    "unknown level",
			cfg:     ConfigLog{Level: "trace", Format: LogFormatText},
			wantErr: `log.level: unknown value "trace" (want debug, info, warn, error)`,
		},
		{
			name:    "unknown format",
			cfg:     ConfigLog{Level: LogLevelInfo, Format: "logfmt"},
			wantErr: `log.format: unknown value "logfmt" (want text, json)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Validate() = %q, want %q", err.Error(), tt.wantErr)
			}
			var de domainError
			if !errors.As(err, &de) || de.Type() != ErrorInvalid {
				t.Fatalf("Validate() type = %v, want ErrorInvalid", err.Type())
			}
		})
	}
}
