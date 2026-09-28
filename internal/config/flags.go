package config

import (
	"flag"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// ServerFlags holds the daemon command line flags.
type ServerFlags struct {
	Config             string
	Listen             string
	CAN                string
	ErrorFrames        bool
	AllowPlain         bool
	TLSCert            string
	TLSKey             string
	TLSCA              string
	TLSClientAuth      string
	MaxConnections     int
	ClientQueue        int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	MaxFramesPerSecond int
	StatsListen        string
	NoStats            bool
	LogLevel           string
	LogFormat          string
	Version            bool
}

// RegisterServerFlags registers the daemon flags on flags. The defaults
// shown in the help output are the built-in ones; the effective values are
// produced by LoadServer.
func RegisterServerFlags(flags *flag.FlagSet) *ServerFlags {
	f := &ServerFlags{}
	d := domain.DefaultConfigServer()
	flags.StringVar(&f.Config, "config", "", "path to the JSON configuration file (default "+DefaultServerConfigPath+" when it exists)")
	flags.StringVar(&f.Listen, "listen", d.Listen, "TCP listen address")
	flags.StringVar(&f.CAN, "can", d.CAN.Interface, "SocketCAN interface to serve")
	flags.BoolVar(&f.ErrorFrames, "error-frames", d.CAN.ErrorFrames, "forward CAN error frames")
	flags.BoolVar(&f.AllowPlain, "allow-plain", d.AllowPlain, "allow plain mode on a non-loopback address")
	flags.StringVar(&f.TLSCert, "tls-cert", "", "TLS server certificate file (enables TLS)")
	flags.StringVar(&f.TLSKey, "tls-key", "", "TLS server key file")
	flags.StringVar(&f.TLSCA, "tls-ca", "", "CA file used to verify client certificates")
	flags.StringVar(&f.TLSClientAuth, "tls-client-auth", string(domain.ClientAuthNone),
		"client certificate mode: none, verify_if_given, require_and_verify")
	flags.IntVar(&f.MaxConnections, "max-connections", d.Limits.MaxConnections, "maximum number of simultaneous clients")
	flags.IntVar(&f.ClientQueue, "client-queue", d.Limits.ClientQueue, "per-client frame queue length")
	flags.DurationVar(&f.ReadTimeout, "read-timeout", d.Limits.ReadTimeout, "TLS handshake timeout")
	flags.DurationVar(&f.WriteTimeout, "write-timeout", d.Limits.WriteTimeout, "per-frame write timeout")
	flags.DurationVar(&f.IdleTimeout, "idle-timeout", d.Limits.IdleTimeout, "close a client after this idle time (0 disables)")
	flags.IntVar(&f.MaxFramesPerSecond, "max-frames-per-second", d.Limits.MaxFramesPerSecond,
		"per-client frame rate limit (0 disables)")
	flags.StringVar(&f.StatsListen, "stats-listen", d.Stats.Listen, "HTTP statistics listen address")
	flags.BoolVar(&f.NoStats, "no-stats", false, "disable the HTTP statistics listener")
	flags.StringVar(&f.LogLevel, "log-level", string(d.Log.Level), "log level: debug, info, warn, error")
	flags.StringVar(&f.LogFormat, "log-format", string(d.Log.Format), "log format: text, json")
	flags.BoolVar(&f.Version, "version", false, "print the version and exit")
	return f
}

// applyServerFlags merges the explicitly set flags into cfg.
func applyServerFlags(flags *flag.FlagSet, f *ServerFlags, cfg *domain.ConfigServer) {
	set := visitedFlags(flags)
	if set["listen"] {
		cfg.Listen = f.Listen
	}
	if set["can"] {
		cfg.CAN.Interface = f.CAN
	}
	if set["error-frames"] {
		cfg.CAN.ErrorFrames = f.ErrorFrames
	}
	if set["allow-plain"] {
		cfg.AllowPlain = f.AllowPlain
	}
	if set["tls-cert"] {
		cfg.TLS.CertFile = f.TLSCert
	}
	if set["tls-key"] {
		cfg.TLS.KeyFile = f.TLSKey
	}
	if set["tls-ca"] {
		cfg.TLS.CAFile = f.TLSCA
	}
	if set["tls-client-auth"] {
		cfg.TLS.ClientAuth = domain.ClientAuth(f.TLSClientAuth)
	}
	if set["max-connections"] {
		cfg.Limits.MaxConnections = f.MaxConnections
	}
	if set["client-queue"] {
		cfg.Limits.ClientQueue = f.ClientQueue
	}
	if set["read-timeout"] {
		cfg.Limits.ReadTimeout = f.ReadTimeout
	}
	if set["write-timeout"] {
		cfg.Limits.WriteTimeout = f.WriteTimeout
	}
	if set["idle-timeout"] {
		cfg.Limits.IdleTimeout = f.IdleTimeout
	}
	if set["max-frames-per-second"] {
		cfg.Limits.MaxFramesPerSecond = f.MaxFramesPerSecond
	}
	if set["stats-listen"] {
		cfg.Stats.Listen = f.StatsListen
	}
	if set["no-stats"] {
		cfg.Stats.Enable = !f.NoStats
	}
	if set["log-level"] {
		cfg.Log.Level = domain.LogLevel(f.LogLevel)
	}
	if set["log-format"] {
		cfg.Log.Format = domain.LogFormat(f.LogFormat)
	}
}

// ClientFlags holds the client command line flags shared by all commands.
type ClientFlags struct {
	Config      string
	Server      string
	StatsServer string
	TLS         bool
	TLSCA       string
	TLSCert     string
	TLSKey      string
	TLSServer   string
	TLSInsecure bool
	LogLevel    string
	LogFormat   string
	Version     bool
}

// RegisterClientFlags registers the shared client flags on flags.
func RegisterClientFlags(flags *flag.FlagSet) *ClientFlags {
	f := &ClientFlags{}
	d := domain.DefaultConfigClient()
	flags.StringVar(&f.Config, "config", "", "path to the JSON configuration file (default "+DefaultClientConfigPath+" when it exists)")
	flags.StringVar(&f.Server, "server", d.Server, "cantcp server address")
	flags.StringVar(&f.StatsServer, "stats-server", d.StatsServer, "server statistics URL")
	flags.BoolVar(&f.TLS, "tls", d.TLS.Enable, "connect over TLS 1.3")
	flags.StringVar(&f.TLSCA, "tls-ca", "", "CA file used to verify the server certificate (default: system roots)")
	flags.StringVar(&f.TLSCert, "tls-cert", "", "client certificate file (mTLS)")
	flags.StringVar(&f.TLSKey, "tls-key", "", "client key file (mTLS)")
	flags.StringVar(&f.TLSServer, "tls-server-name", "", "server name to verify (default: the host from --server)")
	flags.BoolVar(&f.TLSInsecure, "tls-insecure", false, "skip certificate verification (debugging only)")
	flags.StringVar(&f.LogLevel, "log-level", string(d.Log.Level), "log level: debug, info, warn, error")
	flags.StringVar(&f.LogFormat, "log-format", string(d.Log.Format), "log format: text, json")
	flags.BoolVar(&f.Version, "version", false, "print the version and exit")
	return f
}

// applyClientFlags merges the explicitly set flags into cfg.
func applyClientFlags(flags *flag.FlagSet, f *ClientFlags, cfg *domain.ConfigClient) {
	set := visitedFlags(flags)
	if set["server"] {
		cfg.Server = f.Server
	}
	if set["stats-server"] {
		cfg.StatsServer = f.StatsServer
	}
	if set["tls"] {
		cfg.TLS.Enable = f.TLS
	}
	if set["tls-ca"] {
		cfg.TLS.CAFile = f.TLSCA
	}
	if set["tls-cert"] {
		cfg.TLS.CertFile = f.TLSCert
	}
	if set["tls-key"] {
		cfg.TLS.KeyFile = f.TLSKey
	}
	if set["tls-server-name"] {
		cfg.TLS.ServerName = f.TLSServer
	}
	if set["tls-insecure"] {
		cfg.TLS.Insecure = f.TLSInsecure
	}
	if set["log-level"] {
		cfg.Log.Level = domain.LogLevel(f.LogLevel)
	}
	if set["log-format"] {
		cfg.Log.Format = domain.LogFormat(f.LogFormat)
	}
}

// visitedFlags returns the names of the flags explicitly set on the command
// line.
func visitedFlags(flags *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}
