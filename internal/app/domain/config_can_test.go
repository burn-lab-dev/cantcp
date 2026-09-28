package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigCAN_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigCAN
		wantErr string
	}{
		{name: "can0", cfg: ConfigCAN{Interface: "can0"}},
		{name: "vcan with error frames", cfg: ConfigCAN{Interface: "vcan0", ErrorFrames: true}},
		{name: "dashes and dots", cfg: ConfigCAN{Interface: "can-if.0"}},
		{name: "max length", cfg: ConfigCAN{Interface: strings.Repeat("a", maxIfaceLen)}},
		{
			name:    "empty",
			cfg:     ConfigCAN{},
			wantErr: "can.interface: must not be empty",
		},
		{
			name:    "too long",
			cfg:     ConfigCAN{Interface: strings.Repeat("a", maxIfaceLen+1)},
			wantErr: `can.interface: "aaaaaaaaaaaaaaaa" is longer than 15 characters`,
		},
		{
			name:    "space",
			cfg:     ConfigCAN{Interface: "can 0"},
			wantErr: `can.interface: "can 0" contains an invalid character ' '`,
		},
		{
			name:    "slash",
			cfg:     ConfigCAN{Interface: "can/0"},
			wantErr: `can.interface: "can/0" contains an invalid character '/'`,
		},
		{
			name:    "colon",
			cfg:     ConfigCAN{Interface: "can:0"},
			wantErr: `can.interface: "can:0" contains an invalid character ':'`,
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
