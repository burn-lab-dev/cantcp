package config

import (
	"flag"
	"io/fs"
	"testing"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// loadServerFrom loads the server configuration for the given arguments,
// environment and file content. The file is served for both "cfg.json" and
// the default path; an empty content means the file does not exist.
func loadServerFrom(t *testing.T, args []string, env map[string]string, content string) (domain.ConfigServer, error) {
	t.Helper()
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterServerFlags(flags)
	if err := flags.Parse(args); err != nil {
		t.Fatalf("Parse(%v) = %v, want nil", args, err)
	}
	read := func(path string) ([]byte, error) {
		if content == "" {
			return nil, fs.ErrNotExist
		}
		return []byte(content), nil
	}
	return LoadServer(flags, f, envFrom(env), read)
}

func TestLoadServer(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		content string
		want    func() domain.ConfigServer
		wantErr string
	}{
		{
			name: "defaults when nothing is set",
			want: domain.DefaultConfigServer,
		},
		{
			name:    "file values",
			content: `{"listen": "10.0.0.1:29536", "allow_plain": true, "log": {"level": "warn"}}`,
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Listen = "10.0.0.1:29536"
				cfg.AllowPlain = true
				cfg.Log.Level = domain.LogLevelWarn
				return cfg
			},
		},
		{
			name:    "environment wins over the file",
			content: `{"listen": "10.0.0.1:29536", "allow_plain": true}`,
			env:     map[string]string{"CANTCP_LISTEN": "10.0.0.2:29536"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Listen = "10.0.0.2:29536"
				cfg.AllowPlain = true
				return cfg
			},
		},
		{
			name:    "flag wins over the environment and the file",
			content: `{"listen": "10.0.0.1:29536"}`,
			env:     map[string]string{"CANTCP_LISTEN": "10.0.0.2:29536"},
			args:    []string{"--listen", "10.0.0.3:29536", "--config", "cfg.json", "--allow-plain"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Listen = "10.0.0.3:29536"
				cfg.AllowPlain = true
				return cfg
			},
		},
		{
			name:    "explicit missing file",
			args:    []string{"--config", "cfg.json"},
			wantErr: "cfg.json: file does not exist",
		},
		{
			name:    "invalid merged configuration",
			content: `{"can": {"interface": ""}}`,
			wantErr: "can.interface: must not be empty",
		},
		{
			name:    "unknown file field",
			args:    []string{"--config", "cfg.json"},
			content: `{"can": {"interface": "can0"}, "extra": 1}`,
			wantErr: `cfg.json: json: unknown field "extra"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadServerFrom(t, tt.args, tt.env, tt.content)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadServer() = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("LoadServer() = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadServer() = %v, want nil", err)
			}
			if want := tt.want(); cfg != want {
				t.Fatalf("LoadServer() = %+v, want %+v", cfg, want)
			}
		})
	}
}

// loadClientFrom loads the client configuration for the given arguments,
// environment and file content.
func loadClientFrom(t *testing.T, args []string, env map[string]string, content string) (domain.ConfigClient, error) {
	t.Helper()
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterClientFlags(flags)
	if err := flags.Parse(args); err != nil {
		t.Fatalf("Parse(%v) = %v, want nil", args, err)
	}
	read := func(path string) ([]byte, error) {
		if content == "" {
			return nil, fs.ErrNotExist
		}
		return []byte(content), nil
	}
	return LoadClient(flags, f, envFrom(env), read)
}

func TestLoadClient(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		content string
		want    func() domain.ConfigClient
		wantErr string
	}{
		{
			name: "defaults when nothing is set",
			want: domain.DefaultConfigClient,
		},
		{
			name:    "file values",
			content: `{"server": "can.example:29536"}`,
			want: func() domain.ConfigClient {
				cfg := domain.DefaultConfigClient()
				cfg.Server = "can.example:29536"
				return cfg
			},
		},
		{
			name: "flag wins over the environment",
			env:  map[string]string{"CANTCP_SERVER": "can.example:1"},
			args: []string{"--server", "can.example:2"},
			want: func() domain.ConfigClient {
				cfg := domain.DefaultConfigClient()
				cfg.Server = "can.example:2"
				return cfg
			},
		},
		{
			name:    "explicit missing file",
			args:    []string{"--config", "cfg.json"},
			wantErr: "cfg.json: file does not exist",
		},
		{
			name:    "invalid merged configuration",
			content: `{"server": "no-port"}`,
			wantErr: `server: "no-port" is not a host:port address`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadClientFrom(t, tt.args, tt.env, tt.content)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadClient() = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("LoadClient() = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadClient() = %v, want nil", err)
			}
			if want := tt.want(); cfg != want {
				t.Fatalf("LoadClient() = %+v, want %+v", cfg, want)
			}
		})
	}
}
