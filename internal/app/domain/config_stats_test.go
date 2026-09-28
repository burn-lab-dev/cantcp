package domain

import (
	"errors"
	"testing"
)

func TestConfigStats_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigStats
		wantErr string
	}{
		{name: "disabled with a bad address", cfg: ConfigStats{Enable: false, Listen: "bad"}},
		{name: "enabled", cfg: ConfigStats{Enable: true, Listen: "127.0.0.1:29537"}},
		{name: "enabled on all interfaces", cfg: ConfigStats{Enable: true, Listen: ":29537"}},
		{
			name:    "enabled without port",
			cfg:     ConfigStats{Enable: true, Listen: "127.0.0.1"},
			wantErr: `stats.listen: "127.0.0.1" is not a host:port address`,
		},
		{
			name:    "enabled with a zero port",
			cfg:     ConfigStats{Enable: true, Listen: "127.0.0.1:0"},
			wantErr: `stats.listen: "127.0.0.1:0" has an invalid port (want 1..65535)`,
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
