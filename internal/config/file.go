// Package config loads the daemon and client settings from the JSON file,
// the CANTCP_* environment variables and the command line flags, in that
// order of precedence (flags win over the environment, the environment over
// the file, the file over the built-in defaults).
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// Paths of the default configuration files, used when --config is not set.
const (
	DefaultServerConfigPath = "/etc/cantcp/cantcpd.json"
	DefaultClientConfigPath = "/etc/cantcp/cantcp-cli.json"
)

// FileServer is the JSON schema of the daemon configuration file. Pointer
// fields distinguish an absent key from a zero value. Unknown keys are
// rejected to catch typos.
type FileServer struct {
	Listen     *string     `json:"listen"`
	AllowPlain *bool       `json:"allow_plain"`
	CAN        *FileCAN    `json:"can"`
	TLS        *FileTLS    `json:"tls"`
	Limits     *FileLimits `json:"limits"`
	Stats      *FileStats  `json:"stats"`
	Log        *FileLog    `json:"log"`
}

// FileCAN is the "can" object of the daemon configuration file.
type FileCAN struct {
	Interface   *string `json:"interface"`
	ErrorFrames *bool   `json:"error_frames"`
}

// FileTLS is the server "tls" object of the configuration file.
type FileTLS struct {
	CertFile   *string `json:"cert_file"`
	KeyFile    *string `json:"key_file"`
	CAFile     *string `json:"ca_file"`
	ClientAuth *string `json:"client_auth"`
}

// FileLimits is the "limits" object; timeouts are Go duration strings
// ("5s", "1m").
type FileLimits struct {
	MaxConnections     *int    `json:"max_connections"`
	ClientQueue        *int    `json:"client_queue"`
	ReadTimeout        *string `json:"read_timeout"`
	WriteTimeout       *string `json:"write_timeout"`
	IdleTimeout        *string `json:"idle_timeout"`
	MaxFramesPerSecond *int    `json:"max_frames_per_second"`
}

// FileStats is the "stats" object of the daemon configuration file.
type FileStats struct {
	Enable *bool   `json:"enable"`
	Listen *string `json:"listen"`
}

// FileLog is the "log" object shared by the daemon and the client.
type FileLog struct {
	Level  *string `json:"level"`
	Format *string `json:"format"`
}

// FileClient is the JSON schema of the client configuration file.
type FileClient struct {
	Server      *string        `json:"server"`
	StatsServer *string        `json:"stats_server"`
	TLS         *FileTLSClient `json:"tls"`
	Log         *FileLog       `json:"log"`
}

// FileTLSClient is the client "tls" object of the configuration file.
type FileTLSClient struct {
	Enable     *bool   `json:"enable"`
	CAFile     *string `json:"ca_file"`
	CertFile   *string `json:"cert_file"`
	KeyFile    *string `json:"key_file"`
	ServerName *string `json:"server_name"`
	Insecure   *bool   `json:"insecure"`
}

// readFileFunc reads a configuration file; it is injected for tests.
type readFileFunc func(string) ([]byte, error)

// loadFileServer reads and parses the daemon configuration file strictly.
func loadFileServer(path string, read readFileFunc) (FileServer, error) {
	var f FileServer
	if err := loadFile(path, read, &f); err != nil {
		return FileServer{}, err
	}
	return f, nil
}

// loadFileClient reads and parses the client configuration file strictly.
func loadFileClient(path string, read readFileFunc) (FileClient, error) {
	var f FileClient
	if err := loadFile(path, read, &f); err != nil {
		return FileClient{}, err
	}
	return f, nil
}

// loadFile reads a JSON document: unknown fields and trailing data are
// errors, the path is included in the message.
func loadFile(path string, read readFileFunc, v any) error {
	b, err := read(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: unexpected data after the JSON value", path)
	}
	return nil
}

// applyFileServer merges the file values into cfg.
func applyFileServer(f FileServer, cfg *domain.ConfigServer) error {
	if f.Listen != nil {
		cfg.Listen = *f.Listen
	}
	if f.AllowPlain != nil {
		cfg.AllowPlain = *f.AllowPlain
	}
	if f.CAN != nil {
		if f.CAN.Interface != nil {
			cfg.CAN.Interface = *f.CAN.Interface
		}
		if f.CAN.ErrorFrames != nil {
			cfg.CAN.ErrorFrames = *f.CAN.ErrorFrames
		}
	}
	if f.TLS != nil {
		if f.TLS.CertFile != nil {
			cfg.TLS.CertFile = *f.TLS.CertFile
		}
		if f.TLS.KeyFile != nil {
			cfg.TLS.KeyFile = *f.TLS.KeyFile
		}
		if f.TLS.CAFile != nil {
			cfg.TLS.CAFile = *f.TLS.CAFile
		}
		if f.TLS.ClientAuth != nil {
			cfg.TLS.ClientAuth = domain.ClientAuth(*f.TLS.ClientAuth)
		}
	}
	if f.Limits != nil {
		if f.Limits.MaxConnections != nil {
			cfg.Limits.MaxConnections = *f.Limits.MaxConnections
		}
		if f.Limits.ClientQueue != nil {
			cfg.Limits.ClientQueue = *f.Limits.ClientQueue
		}
		if err := applyFileDuration("limits.read_timeout", f.Limits.ReadTimeout, &cfg.Limits.ReadTimeout); err != nil {
			return err
		}
		if err := applyFileDuration("limits.write_timeout", f.Limits.WriteTimeout, &cfg.Limits.WriteTimeout); err != nil {
			return err
		}
		if err := applyFileDuration("limits.idle_timeout", f.Limits.IdleTimeout, &cfg.Limits.IdleTimeout); err != nil {
			return err
		}
		if f.Limits.MaxFramesPerSecond != nil {
			cfg.Limits.MaxFramesPerSecond = *f.Limits.MaxFramesPerSecond
		}
	}
	if f.Stats != nil {
		if f.Stats.Enable != nil {
			cfg.Stats.Enable = *f.Stats.Enable
		}
		if f.Stats.Listen != nil {
			cfg.Stats.Listen = *f.Stats.Listen
		}
	}
	return applyFileLog(f.Log, &cfg.Log)
}

// applyFileClient merges the file values into cfg.
func applyFileClient(f FileClient, cfg *domain.ConfigClient) error {
	if f.Server != nil {
		cfg.Server = *f.Server
	}
	if f.StatsServer != nil {
		cfg.StatsServer = *f.StatsServer
	}
	if f.TLS != nil {
		if f.TLS.Enable != nil {
			cfg.TLS.Enable = *f.TLS.Enable
		}
		if f.TLS.CAFile != nil {
			cfg.TLS.CAFile = *f.TLS.CAFile
		}
		if f.TLS.CertFile != nil {
			cfg.TLS.CertFile = *f.TLS.CertFile
		}
		if f.TLS.KeyFile != nil {
			cfg.TLS.KeyFile = *f.TLS.KeyFile
		}
		if f.TLS.ServerName != nil {
			cfg.TLS.ServerName = *f.TLS.ServerName
		}
		if f.TLS.Insecure != nil {
			cfg.TLS.Insecure = *f.TLS.Insecure
		}
	}
	return applyFileLog(f.Log, &cfg.Log)
}

// applyFileLog merges the "log" object into cfg.
func applyFileLog(f *FileLog, cfg *domain.ConfigLog) error {
	if f == nil {
		return nil
	}
	if f.Level != nil {
		cfg.Level = domain.LogLevel(*f.Level)
	}
	if f.Format != nil {
		cfg.Format = domain.LogFormat(*f.Format)
	}
	return nil
}

// applyFileDuration parses an optional Go duration string and stores it.
func applyFileDuration(field string, raw *string, dst *time.Duration) error {
	if raw == nil {
		return nil
	}
	d, err := time.ParseDuration(*raw)
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q", field, *raw)
	}
	*dst = d
	return nil
}
