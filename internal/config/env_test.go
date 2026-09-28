package config

import (
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// envFrom builds a getenvFunc over a map.
func envFrom(pairs map[string]string) getenvFunc {
	return func(name string) (string, bool) {
		v, ok := pairs[name]
		return v, ok
	}
}

func TestApplyServerEnv_Full(t *testing.T) {
	cfg := domain.DefaultConfigServer()
	err := applyServerEnv(envFrom(map[string]string{
		"CANTCP_LISTEN":                "0.0.0.0:29536",
		"CANTCP_CAN":                   "vcan0",
		"CANTCP_ERROR_FRAMES":          "true",
		"CANTCP_ALLOW_PLAIN":           "true",
		"CANTCP_TLS_CERT":              "c.pem",
		"CANTCP_TLS_KEY":               "k.pem",
		"CANTCP_TLS_CA":                "ca.pem",
		"CANTCP_TLS_CLIENT_AUTH":       "verify_if_given",
		"CANTCP_MAX_CONNECTIONS":       "8",
		"CANTCP_CLIENT_QUEUE":          "16",
		"CANTCP_READ_TIMEOUT":          "1s",
		"CANTCP_WRITE_TIMEOUT":         "2s",
		"CANTCP_IDLE_TIMEOUT":          "3s",
		"CANTCP_MAX_FRAMES_PER_SECOND": "100",
		"CANTCP_STATS_LISTEN":          "127.0.0.1:9999",
		"CANTCP_STATS_ENABLE":          "false",
		"CANTCP_LOG_LEVEL":             "debug",
		"CANTCP_LOG_FORMAT":            "json",
	}), &cfg)
	if err != nil {
		t.Fatalf("applyServerEnv() = %v, want nil", err)
	}
	want := domain.ConfigServer{
		Listen:     "0.0.0.0:29536",
		AllowPlain: true,
		CAN:        domain.ConfigCAN{Interface: "vcan0", ErrorFrames: true},
		TLS: domain.ConfigTLS{
			CertFile: "c.pem", KeyFile: "k.pem", CAFile: "ca.pem",
			ClientAuth: domain.ClientAuthVerifyIfGiven,
		},
		Limits: domain.ConfigLimits{
			MaxConnections: 8, ClientQueue: 16,
			ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 3 * time.Second,
			MaxFramesPerSecond: 100,
		},
		Stats: domain.ConfigStats{Enable: false, Listen: "127.0.0.1:9999"},
		Log:   domain.ConfigLog{Level: domain.LogLevelDebug, Format: domain.LogFormatJSON},
	}
	if cfg != want {
		t.Fatalf("applyServerEnv() = %+v, want %+v", cfg, want)
	}
}

func TestApplyServerEnv_Errors(t *testing.T) {
	tests := []struct {
		name    string
		pairs   map[string]string
		wantErr string
	}{
		{
			name:    "error frames is not a boolean",
			pairs:   map[string]string{"CANTCP_ERROR_FRAMES": "yes"},
			wantErr: `CANTCP_ERROR_FRAMES: invalid boolean "yes"`,
		},
		{
			name:    "allow plain is not a boolean",
			pairs:   map[string]string{"CANTCP_ALLOW_PLAIN": "sure"},
			wantErr: `CANTCP_ALLOW_PLAIN: invalid boolean "sure"`,
		},
		{
			name:    "max connections is not an integer",
			pairs:   map[string]string{"CANTCP_MAX_CONNECTIONS": "many"},
			wantErr: `CANTCP_MAX_CONNECTIONS: invalid integer "many"`,
		},
		{
			name:    "client queue is not an integer",
			pairs:   map[string]string{"CANTCP_CLIENT_QUEUE": "1.5"},
			wantErr: `CANTCP_CLIENT_QUEUE: invalid integer "1.5"`,
		},
		{
			name:    "read timeout is not a duration",
			pairs:   map[string]string{"CANTCP_READ_TIMEOUT": "soon"},
			wantErr: `CANTCP_READ_TIMEOUT: invalid duration "soon"`,
		},
		{
			name:    "write timeout is not a duration",
			pairs:   map[string]string{"CANTCP_WRITE_TIMEOUT": "2 seconds"},
			wantErr: `CANTCP_WRITE_TIMEOUT: invalid duration "2 seconds"`,
		},
		{
			name:    "idle timeout is not a duration",
			pairs:   map[string]string{"CANTCP_IDLE_TIMEOUT": "0x10"},
			wantErr: `CANTCP_IDLE_TIMEOUT: invalid duration "0x10"`,
		},
		{
			name:    "rate limit is not an integer",
			pairs:   map[string]string{"CANTCP_MAX_FRAMES_PER_SECOND": "fast"},
			wantErr: `CANTCP_MAX_FRAMES_PER_SECOND: invalid integer "fast"`,
		},
		{
			name:    "stats enable is not a boolean",
			pairs:   map[string]string{"CANTCP_STATS_ENABLE": "on-off"},
			wantErr: `CANTCP_STATS_ENABLE: invalid boolean "on-off"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := domain.DefaultConfigServer()
			err := applyServerEnv(envFrom(tt.pairs), &cfg)
			if err == nil {
				t.Fatalf("applyServerEnv() = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("applyServerEnv() = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestApplyClientEnv_Full(t *testing.T) {
	cfg := domain.DefaultConfigClient()
	err := applyClientEnv(envFrom(map[string]string{
		"CANTCP_SERVER":          "can.example:29536",
		"CANTCP_STATS_SERVER":    "https://can.example:29537",
		"CANTCP_TLS_ENABLE":      "true",
		"CANTCP_TLS_CA":          "ca.pem",
		"CANTCP_TLS_CERT":        "c.pem",
		"CANTCP_TLS_KEY":         "k.pem",
		"CANTCP_TLS_SERVER_NAME": "can.example",
		"CANTCP_TLS_INSECURE":    "true",
		"CANTCP_LOG_LEVEL":       "warn",
		"CANTCP_LOG_FORMAT":      "json",
	}), &cfg)
	if err != nil {
		t.Fatalf("applyClientEnv() = %v, want nil", err)
	}
	want := domain.ConfigClient{
		Server:      "can.example:29536",
		StatsServer: "https://can.example:29537",
		TLS: domain.ConfigTLSClient{
			Enable: true, CAFile: "ca.pem", CertFile: "c.pem", KeyFile: "k.pem",
			ServerName: "can.example", Insecure: true,
		},
		Log: domain.ConfigLog{Level: domain.LogLevelWarn, Format: domain.LogFormatJSON},
	}
	if cfg != want {
		t.Fatalf("applyClientEnv() = %+v, want %+v", cfg, want)
	}
}

func TestApplyClientEnv_Errors(t *testing.T) {
	tests := []struct {
		name    string
		pairs   map[string]string
		wantErr string
	}{
		{
			name:    "TLS enable is not a boolean",
			pairs:   map[string]string{"CANTCP_TLS_ENABLE": "maybe"},
			wantErr: `CANTCP_TLS_ENABLE: invalid boolean "maybe"`,
		},
		{
			name:    "TLS insecure is not a boolean",
			pairs:   map[string]string{"CANTCP_TLS_INSECURE": "sometimes"},
			wantErr: `CANTCP_TLS_INSECURE: invalid boolean "sometimes"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := domain.DefaultConfigClient()
			err := applyClientEnv(envFrom(tt.pairs), &cfg)
			if err == nil {
				t.Fatalf("applyClientEnv() = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("applyClientEnv() = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
