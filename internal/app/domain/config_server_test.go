package domain

import (
	"errors"
	"testing"
)

func TestDefaultConfigServer(t *testing.T) {
	cfg := DefaultConfigServer()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default configuration is invalid: %v", err)
	}
	if cfg.Listen != "127.0.0.1:29536" {
		t.Fatalf("Listen = %q, want 127.0.0.1:29536", cfg.Listen)
	}
	if cfg.CAN.Interface != "can0" {
		t.Fatalf("CAN.Interface = %q, want can0", cfg.CAN.Interface)
	}
	if cfg.AllowPlain {
		t.Fatal("AllowPlain must be false by default")
	}
	if !cfg.Stats.Enable {
		t.Fatal("statistics must be enabled by default")
	}
	if cfg.TLS.ClientAuth != ClientAuthNone {
		t.Fatalf("TLS.ClientAuth = %q, want none", cfg.TLS.ClientAuth)
	}
}

func TestConfigServer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ConfigServer)
		wantErr string
	}{
		{name: "defaults"},
		{
			name: "plain on a loopback IP",
			mutate: func(c *ConfigServer) {
				c.Listen = "127.0.0.2:29536"
			},
		},
		{
			name: "plain on IPv6 loopback",
			mutate: func(c *ConfigServer) {
				c.Listen = "[::1]:29536"
			},
		},
		{
			name: "plain on a host name",
			mutate: func(c *ConfigServer) {
				c.Listen = "localhost:29536"
			},
		},
		{
			name: "plain outside with allow_plain",
			mutate: func(c *ConfigServer) {
				c.Listen = "0.0.0.0:29536"
				c.AllowPlain = true
			},
		},
		{
			name: "TLS on a public address",
			mutate: func(c *ConfigServer) {
				c.Listen = "0.0.0.0:29536"
				c.TLS = ConfigTLS{CertFile: "c.pem", KeyFile: "k.pem", ClientAuth: ClientAuthNone}
			},
		},
		{
			name:    "plain outside without allow_plain",
			mutate:  func(c *ConfigServer) { c.Listen = "0.0.0.0:29536" },
			wantErr: `listen: plain mode on "0.0.0.0:29536" is not loopback; set allow_plain or configure TLS`,
		},
		{
			name:    "plain on a public IPv6 address",
			mutate:  func(c *ConfigServer) { c.Listen = "[2001:db8::1]:29536" },
			wantErr: `listen: plain mode on "[2001:db8::1]:29536" is not loopback; set allow_plain or configure TLS`,
		},
		{
			name:    "bad listen address",
			mutate:  func(c *ConfigServer) { c.Listen = "can0" },
			wantErr: `listen: "can0" is not a host:port address`,
		},
		{
			name:    "bad port",
			mutate:  func(c *ConfigServer) { c.Listen = "127.0.0.1:70000" },
			wantErr: `listen: "127.0.0.1:70000" has an invalid port (want 1..65535)`,
		},
		{
			name:    "bad CAN interface",
			mutate:  func(c *ConfigServer) { c.CAN.Interface = "" },
			wantErr: "can.interface: must not be empty",
		},
		{
			name:    "bad TLS pair",
			mutate:  func(c *ConfigServer) { c.TLS = ConfigTLS{CertFile: "c.pem", ClientAuth: ClientAuthNone} },
			wantErr: "tls.key_file: must be set together with tls.cert_file",
		},
		{
			name:    "bad limits",
			mutate:  func(c *ConfigServer) { c.Limits.MaxConnections = 0 },
			wantErr: "limits.max_connections: 0 is out of range [1, 4096]",
		},
		{
			name:    "bad stats address",
			mutate:  func(c *ConfigServer) { c.Stats.Listen = "127.0.0.1" },
			wantErr: `stats.listen: "127.0.0.1" is not a host:port address`,
		},
		{
			name:    "bad log format",
			mutate:  func(c *ConfigServer) { c.Log.Format = "logfmt" },
			wantErr: `log.format: unknown value "logfmt" (want text, json)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfigServer()
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
