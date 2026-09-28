package config

import (
	"flag"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// parseFlags registers the flags, parses args and returns the flag set and
// the flag values.
func parseFlags(t *testing.T, args []string, register func(*flag.FlagSet)) *flag.FlagSet {
	t.Helper()
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	register(flags)
	if err := flags.Parse(args); err != nil {
		t.Fatalf("Parse(%v) = %v, want nil", args, err)
	}
	return flags
}

func TestRegisterServerFlags_Defaults(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterServerFlags(flags)
	d := domain.DefaultConfigServer()
	if f.Config != "" {
		t.Fatalf("Config = %q, want empty", f.Config)
	}
	if f.Listen != d.Listen {
		t.Fatalf("Listen = %q, want %q", f.Listen, d.Listen)
	}
	if f.CAN != d.CAN.Interface {
		t.Fatalf("CAN = %q, want %q", f.CAN, d.CAN.Interface)
	}
	if f.MaxConnections != d.Limits.MaxConnections {
		t.Fatalf("MaxConnections = %d, want %d", f.MaxConnections, d.Limits.MaxConnections)
	}
	if f.ReadTimeout != d.Limits.ReadTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", f.ReadTimeout, d.Limits.ReadTimeout)
	}
	if f.StatsListen != d.Stats.Listen {
		t.Fatalf("StatsListen = %q, want %q", f.StatsListen, d.Stats.Listen)
	}
	if f.LogLevel != string(d.Log.Level) {
		t.Fatalf("LogLevel = %q, want %q", f.LogLevel, d.Log.Level)
	}
	if f.Version {
		t.Fatal("Version must be false by default")
	}
}

func TestApplyServerFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want func() domain.ConfigServer
	}{
		{
			name: "no flags keeps the defaults",
			want: domain.DefaultConfigServer,
		},
		{
			name: "listen and allow plain",
			args: []string{"--listen", "0.0.0.0:29536", "--allow-plain"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Listen = "0.0.0.0:29536"
				cfg.AllowPlain = true
				return cfg
			},
		},
		{
			name: "can and error frames",
			args: []string{"--can", "vcan0", "--error-frames"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.CAN.Interface = "vcan0"
				cfg.CAN.ErrorFrames = true
				return cfg
			},
		},
		{
			name: "TLS material",
			args: []string{
				"--tls-cert", "c.pem", "--tls-key", "k.pem",
				"--tls-ca", "ca.pem", "--tls-client-auth", "require_and_verify",
			},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.TLS = domain.ConfigTLS{
					CertFile: "c.pem", KeyFile: "k.pem", CAFile: "ca.pem",
					ClientAuth: domain.ClientAuthRequireAndVerify,
				}
				return cfg
			},
		},
		{
			name: "limits",
			args: []string{
				"--max-connections", "3", "--client-queue", "5",
				"--read-timeout", "1s", "--write-timeout", "2s", "--idle-timeout", "3s",
				"--max-frames-per-second", "9",
			},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Limits = domain.ConfigLimits{
					MaxConnections: 3, ClientQueue: 5,
					ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 3 * time.Second,
					MaxFramesPerSecond: 9,
				}
				return cfg
			},
		},
		{
			name: "no stats",
			args: []string{"--no-stats"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Stats.Enable = false
				return cfg
			},
		},
		{
			name: "no stats explicitly false",
			args: []string{"--no-stats=false"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Stats.Enable = true
				return cfg
			},
		},
		{
			name: "stats listen",
			args: []string{"--stats-listen", "0.0.0.0:9999"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Stats.Listen = "0.0.0.0:9999"
				return cfg
			},
		},
		{
			name: "log",
			args: []string{"--log-level", "error", "--log-format", "json"},
			want: func() domain.ConfigServer {
				cfg := domain.DefaultConfigServer()
				cfg.Log = domain.ConfigLog{Level: domain.LogLevelError, Format: domain.LogFormatJSON}
				return cfg
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			f := RegisterServerFlags(flags)
			if err := flags.Parse(tt.args); err != nil {
				t.Fatalf("Parse(%v) = %v, want nil", tt.args, err)
			}
			cfg := domain.DefaultConfigServer()
			applyServerFlags(flags, f, &cfg)
			if want := tt.want(); cfg != want {
				t.Fatalf("applyServerFlags() = %+v, want %+v", cfg, want)
			}
		})
	}
}

func TestRegisterClientFlags_Defaults(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterClientFlags(flags)
	d := domain.DefaultConfigClient()
	if f.Server != d.Server {
		t.Fatalf("Server = %q, want %q", f.Server, d.Server)
	}
	if f.StatsServer != d.StatsServer {
		t.Fatalf("StatsServer = %q, want %q", f.StatsServer, d.StatsServer)
	}
	if f.TLS || f.TLSInsecure {
		t.Fatal("TLS must be disabled by default")
	}
}

func TestApplyClientFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want func() domain.ConfigClient
	}{
		{
			name: "no flags keeps the defaults",
			want: domain.DefaultConfigClient,
		},
		{
			name: "connection addresses",
			args: []string{"--server", "can.example:29536", "--stats-server", "http://can.example:29537"},
			want: func() domain.ConfigClient {
				cfg := domain.DefaultConfigClient()
				cfg.Server = "can.example:29536"
				cfg.StatsServer = "http://can.example:29537"
				return cfg
			},
		},
		{
			name: "TLS",
			args: []string{
				"--tls", "--tls-ca", "ca.pem", "--tls-cert", "c.pem", "--tls-key", "k.pem",
				"--tls-server-name", "can.example", "--tls-insecure",
			},
			want: func() domain.ConfigClient {
				cfg := domain.DefaultConfigClient()
				cfg.TLS = domain.ConfigTLSClient{
					Enable: true, CAFile: "ca.pem", CertFile: "c.pem", KeyFile: "k.pem",
					ServerName: "can.example", Insecure: true,
				}
				return cfg
			},
		},
		{
			name: "log",
			args: []string{"--log-level", "debug", "--log-format", "json"},
			want: func() domain.ConfigClient {
				cfg := domain.DefaultConfigClient()
				cfg.Log = domain.ConfigLog{Level: domain.LogLevelDebug, Format: domain.LogFormatJSON}
				return cfg
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			f := RegisterClientFlags(flags)
			if err := flags.Parse(tt.args); err != nil {
				t.Fatalf("Parse(%v) = %v, want nil", tt.args, err)
			}
			cfg := domain.DefaultConfigClient()
			applyClientFlags(flags, f, &cfg)
			if want := tt.want(); cfg != want {
				t.Fatalf("applyClientFlags() = %+v, want %+v", cfg, want)
			}
		})
	}
}
