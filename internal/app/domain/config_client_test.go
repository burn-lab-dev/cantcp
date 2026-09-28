package domain

import (
	"errors"
	"testing"
)

func TestDefaultConfigClient(t *testing.T) {
	cfg := DefaultConfigClient()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default configuration is invalid: %v", err)
	}
	if cfg.Server != "127.0.0.1:29536" {
		t.Fatalf("Server = %q, want 127.0.0.1:29536", cfg.Server)
	}
	if cfg.StatsServer != "http://127.0.0.1:29537" {
		t.Fatalf("StatsServer = %q, want http://127.0.0.1:29537", cfg.StatsServer)
	}
	if cfg.TLS.Enable {
		t.Fatal("TLS must be disabled by default")
	}
}

func TestConfigClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ConfigClient)
		wantErr string
	}{
		{name: "defaults"},
		{
			name:   "remote TLS server",
			mutate: func(c *ConfigClient) { c.Server = "can.example:29536"; c.TLS.Enable = true },
		},
		{
			name:    "bad server address",
			mutate:  func(c *ConfigClient) { c.Server = "can.example" },
			wantErr: `server: "can.example" is not a host:port address`,
		},
		{
			name:    "bad stats URL scheme",
			mutate:  func(c *ConfigClient) { c.StatsServer = "ftp://host/stats" },
			wantErr: `stats_server: "ftp://host/stats" is not an http:// or https:// URL`,
		},
		{
			name:    "stats URL without host",
			mutate:  func(c *ConfigClient) { c.StatsServer = "http:///stats" },
			wantErr: `stats_server: "http:///stats" is not an http:// or https:// URL`,
		},
		{
			name:    "TLS without enable",
			mutate:  func(c *ConfigClient) { c.TLS.CAFile = "ca.pem" },
			wantErr: "tls: client TLS settings require tls.enable",
		},
		{
			name:    "bad log level",
			mutate:  func(c *ConfigClient) { c.Log.Level = "loud" },
			wantErr: `log.level: unknown value "loud" (want debug, info, warn, error)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfigClient()
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
