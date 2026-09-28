package domain

import (
	"errors"
	"testing"
	"time"
)

// validLimits returns a valid base configuration for the tests.
func validLimits() ConfigLimits {
	return ConfigLimits{
		MaxConnections: 16,
		ClientQueue:    1024,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
	}
}

func TestConfigLimits_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ConfigLimits)
		wantErr string
	}{
		{name: "valid base"},
		{
			name:   "single connection and queue",
			mutate: func(c *ConfigLimits) { c.MaxConnections = 1; c.ClientQueue = 1 },
		},
		{
			name:   "max frames limit and idle timeout",
			mutate: func(c *ConfigLimits) { c.MaxFramesPerSecond = 1000; c.IdleTimeout = time.Minute },
		},
		{
			name:    "zero connections",
			mutate:  func(c *ConfigLimits) { c.MaxConnections = 0 },
			wantErr: "limits.max_connections: 0 is out of range [1, 4096]",
		},
		{
			name:    "too many connections",
			mutate:  func(c *ConfigLimits) { c.MaxConnections = 4097 },
			wantErr: "limits.max_connections: 4097 is out of range [1, 4096]",
		},
		{
			name:    "zero queue",
			mutate:  func(c *ConfigLimits) { c.ClientQueue = 0 },
			wantErr: "limits.client_queue: 0 is out of range [1, 1048576]",
		},
		{
			name:    "huge queue",
			mutate:  func(c *ConfigLimits) { c.ClientQueue = 1<<20 + 1 },
			wantErr: "limits.client_queue: 1048577 is out of range [1, 1048576]",
		},
		{
			name:    "zero read timeout",
			mutate:  func(c *ConfigLimits) { c.ReadTimeout = 0 },
			wantErr: "limits.read_timeout: must be positive",
		},
		{
			name:    "negative read timeout",
			mutate:  func(c *ConfigLimits) { c.ReadTimeout = -time.Second },
			wantErr: "limits.read_timeout: must be positive",
		},
		{
			name:    "zero write timeout",
			mutate:  func(c *ConfigLimits) { c.WriteTimeout = 0 },
			wantErr: "limits.write_timeout: must be positive",
		},
		{
			name:    "negative idle timeout",
			mutate:  func(c *ConfigLimits) { c.IdleTimeout = -time.Second },
			wantErr: "limits.idle_timeout: must not be negative",
		},
		{
			name:    "negative rate limit",
			mutate:  func(c *ConfigLimits) { c.MaxFramesPerSecond = -1 },
			wantErr: "limits.max_frames_per_second: must not be negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validLimits()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			err := cfg.Validate()
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
