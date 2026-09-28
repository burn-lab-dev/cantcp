package config

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// testRead serves content for the path "cfg.json" and reports fs.ErrNotExist
// for everything else.
func testRead(content string) readFileFunc {
	return func(path string) ([]byte, error) {
		if path != "cfg.json" {
			return nil, fs.ErrNotExist
		}
		return []byte(content), nil
	}
}

func TestLoadFileServer(t *testing.T) {
	full := `{
		"listen": "0.0.0.0:1234",
		"allow_plain": true,
		"can": {"interface": "vcan0", "error_frames": true},
		"tls": {"cert_file": "c.pem", "key_file": "k.pem", "ca_file": "ca.pem", "client_auth": "require_and_verify"},
		"limits": {
			"max_connections": 5,
			"client_queue": 10,
			"read_timeout": "1s",
			"write_timeout": "2s",
			"idle_timeout": "3s",
			"max_frames_per_second": 100
		},
		"stats": {"enable": false, "listen": "127.0.0.1:1"},
		"log": {"level": "debug", "format": "json"}
	}`
	tests := []struct {
		name    string
		content string
		check   func(*testing.T, FileServer)
		wantErr string
	}{
		{
			name:    "full document",
			content: full,
			check: func(t *testing.T, f FileServer) {
				t.Helper()
				if f.Listen == nil || *f.Listen != "0.0.0.0:1234" {
					t.Fatalf("Listen = %v, want 0.0.0.0:1234", f.Listen)
				}
				if f.AllowPlain == nil || !*f.AllowPlain {
					t.Fatal("AllowPlain must be true")
				}
				if f.CAN == nil || f.CAN.Interface == nil || *f.CAN.Interface != "vcan0" || f.CAN.ErrorFrames == nil || !*f.CAN.ErrorFrames {
					t.Fatalf("CAN = %+v, want vcan0 with error frames", f.CAN)
				}
				if f.TLS == nil || f.TLS.ClientAuth == nil || *f.TLS.ClientAuth != "require_and_verify" {
					t.Fatalf("TLS = %+v, want client_auth require_and_verify", f.TLS)
				}
				if f.Limits == nil || f.Limits.ReadTimeout == nil || *f.Limits.ReadTimeout != "1s" {
					t.Fatalf("Limits = %+v, want read_timeout 1s", f.Limits)
				}
				if f.Stats == nil || f.Stats.Enable == nil || *f.Stats.Enable {
					t.Fatalf("Stats = %+v, want enable false", f.Stats)
				}
				if f.Log == nil || f.Log.Level == nil || *f.Log.Level != "debug" {
					t.Fatalf("Log = %+v, want debug", f.Log)
				}
			},
		},
		{
			name:    "empty document is valid",
			content: `{}`,
			check: func(t *testing.T, f FileServer) {
				t.Helper()
				if f.Listen != nil || f.CAN != nil {
					t.Fatalf("empty document = %+v, want all nil", f)
				}
			},
		},
		{
			name:    "unknown field",
			content: `{"listen": "127.0.0.1:1", "lisen": "typo"}`,
			wantErr: `cfg.json: json: unknown field "lisen"`,
		},
		{
			name:    "trailing data",
			content: `{"listen": "127.0.0.1:1"} {}`,
			wantErr: "cfg.json: unexpected data after the JSON value",
		},
		{
			name:    "broken JSON",
			content: `{"listen":`,
			wantErr: "cfg.json: unexpected EOF",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := loadFileServer("cfg.json", testRead(tt.content))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("loadFileServer() = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("loadFileServer() = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadFileServer() = %v, want nil", err)
			}
			tt.check(t, file)
		})
	}
}

func TestLoadFileServer_NotExist(t *testing.T) {
	_, err := loadFileServer("missing.json", testRead(`{}`))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("loadFileServer() = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadFileClient(t *testing.T) {
	content := `{
		"server": "can.example:29536",
		"stats_server": "https://can.example:29537",
		"tls": {"enable": true, "ca_file": "ca.pem", "cert_file": "c.pem", "key_file": "k.pem", "server_name": "can.example", "insecure": true},
		"log": {"level": "warn", "format": "text"}
	}`
	file, err := loadFileClient("cfg.json", testRead(content))
	if err != nil {
		t.Fatalf("loadFileClient() = %v, want nil", err)
	}
	if file.Server == nil || *file.Server != "can.example:29536" {
		t.Fatalf("Server = %v, want can.example:29536", file.Server)
	}
	if file.TLS == nil || file.TLS.Enable == nil || !*file.TLS.Enable || file.TLS.ServerName == nil || *file.TLS.ServerName != "can.example" {
		t.Fatalf("TLS = %+v, want enabled with server_name", file.TLS)
	}
	if file.TLS.Insecure == nil || !*file.TLS.Insecure {
		t.Fatal("TLS.Insecure must be true")
	}
	if file.Log == nil || file.Log.Level == nil || *file.Log.Level != "warn" {
		t.Fatalf("Log = %+v, want warn", file.Log)
	}
}

func TestApplyFileServer(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(*testing.T, domain.ConfigServer)
		wantErr string
	}{
		{
			name: "all fields",
			content: `{
				"listen": "10.0.0.1:29536",
				"allow_plain": true,
				"can": {"interface": "can1", "error_frames": true},
				"tls": {"cert_file": "c.pem", "key_file": "k.pem", "ca_file": "ca.pem", "client_auth": "verify_if_given"},
				"limits": {
					"max_connections": 3,
					"client_queue": 7,
					"read_timeout": "1s",
					"write_timeout": "2s",
					"idle_timeout": "4s",
					"max_frames_per_second": 500
				},
				"stats": {"enable": false, "listen": "127.0.0.1:1"},
				"log": {"level": "error", "format": "json"}
			}`,
			check: func(t *testing.T, cfg domain.ConfigServer) {
				t.Helper()
				want := domain.ConfigServer{
					Listen:     "10.0.0.1:29536",
					AllowPlain: true,
					CAN:        domain.ConfigCAN{Interface: "can1", ErrorFrames: true},
					TLS: domain.ConfigTLS{
						CertFile: "c.pem", KeyFile: "k.pem", CAFile: "ca.pem",
						ClientAuth: domain.ClientAuthVerifyIfGiven,
					},
					Limits: domain.ConfigLimits{
						MaxConnections: 3, ClientQueue: 7,
						ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 4 * time.Second,
						MaxFramesPerSecond: 500,
					},
					Stats: domain.ConfigStats{Enable: false, Listen: "127.0.0.1:1"},
					Log:   domain.ConfigLog{Level: domain.LogLevelError, Format: domain.LogFormatJSON},
				}
				if cfg != want {
					t.Fatalf("applyFileServer() = %+v, want %+v", cfg, want)
				}
			},
		},
		{
			name:    "empty document keeps defaults",
			content: `{}`,
			check: func(t *testing.T, cfg domain.ConfigServer) {
				t.Helper()
				if cfg != domain.DefaultConfigServer() {
					t.Fatalf("applyFileServer() = %+v, want defaults", cfg)
				}
			},
		},
		{
			name:    "broken read timeout",
			content: `{"limits": {"read_timeout": "1x"}}`,
			wantErr: `limits.read_timeout: invalid duration "1x"`,
		},
		{
			name:    "broken write timeout",
			content: `{"limits": {"write_timeout": "5"}}`,
			wantErr: `limits.write_timeout: invalid duration "5"`,
		},
		{
			name:    "broken idle timeout",
			content: `{"limits": {"idle_timeout": "soon"}}`,
			wantErr: `limits.idle_timeout: invalid duration "soon"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := loadFileServer("cfg.json", testRead(tt.content))
			if err != nil {
				t.Fatalf("loadFileServer() = %v, want nil", err)
			}
			cfg := domain.DefaultConfigServer()
			err = applyFileServer(file, &cfg)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("applyFileServer() = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("applyFileServer() = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("applyFileServer() = %v, want nil", err)
			}
			tt.check(t, cfg)
		})
	}
}

func TestApplyFileClient(t *testing.T) {
	content := `{
		"server": "can.example:29536",
		"stats_server": "https://can.example:29537",
		"tls": {"enable": true, "ca_file": "ca.pem", "cert_file": "c.pem", "key_file": "k.pem", "server_name": "can.example", "insecure": true},
		"log": {"level": "warn", "format": "json"}
	}`
	file, err := loadFileClient("cfg.json", testRead(content))
	if err != nil {
		t.Fatalf("loadFileClient() = %v, want nil", err)
	}
	cfg := domain.DefaultConfigClient()
	if err := applyFileClient(file, &cfg); err != nil {
		t.Fatalf("applyFileClient() = %v, want nil", err)
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
		t.Fatalf("applyFileClient() = %+v, want %+v", cfg, want)
	}

	empty := domain.DefaultConfigClient()
	if err := applyFileClient(FileClient{}, &empty); err != nil {
		t.Fatalf("applyFileClient(empty) = %v, want nil", err)
	}
	if empty != domain.DefaultConfigClient() {
		t.Fatalf("applyFileClient(empty) = %+v, want defaults", empty)
	}
}

func TestApplyFileLog(t *testing.T) {
	tests := []struct {
		name string
		file *FileLog
		want domain.ConfigLog
	}{
		{
			name: "nil leaves the configuration untouched",
			file: nil,
			want: domain.ConfigLog{Level: domain.LogLevelInfo, Format: domain.LogFormatText},
		},
		{
			name: "level only",
			file: &FileLog{Level: ptr("error")},
			want: domain.ConfigLog{Level: domain.LogLevelError, Format: domain.LogFormatText},
		},
		{
			name: "format only",
			file: &FileLog{Format: ptr("json")},
			want: domain.ConfigLog{Level: domain.LogLevelInfo, Format: domain.LogFormatJSON},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := domain.ConfigLog{Level: domain.LogLevelInfo, Format: domain.LogFormatText}
			if err := applyFileLog(tt.file, &cfg); err != nil {
				t.Fatalf("applyFileLog() = %v, want nil", err)
			}
			if cfg != tt.want {
				t.Fatalf("applyFileLog() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

func TestLoadFileServer_IsStrictAboutTypes(t *testing.T) {
	_, err := loadFileServer("cfg.json", testRead(`{"limits": {"max_connections": "many"}}`))
	if err == nil {
		t.Fatal("loadFileServer() = nil, want a type error")
	}
	if !strings.Contains(err.Error(), "max_connections") {
		t.Fatalf("loadFileServer() = %q, want the field name in the message", err.Error())
	}
}
