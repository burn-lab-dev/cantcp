// Package tlsconfig builds TLS 1.3 server and client configurations and
// reloads the server certificate material without restarting the process.
package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"sync/atomic"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// Reloader owns the server tls.Config and swaps it atomically on Reload.
// Connections handshaken with the previous configuration keep their session.
type Reloader struct {
	current atomic.Pointer[tls.Config]
}

// NewServer loads the certificate, the key and the CA and builds the server
// configuration. TLS 1.3 is the only accepted version.
func NewServer(cfg domain.ConfigTLS) (*Reloader, error) {
	r := &Reloader{}
	if err := r.Reload(cfg); err != nil {
		return nil, err
	}
	return r, nil
}

// Reload builds the server configuration from cfg, re-reads the certificate
// files and atomically replaces the current configuration. New connections
// use the new material; on error the current configuration stays in place.
func (r *Reloader) Reload(cfg domain.ConfigTLS) error {
	conf, err := buildServer(cfg)
	if err != nil {
		return err
	}
	r.current.Store(conf)
	return nil
}

// Config returns the current server configuration. The pointer is never nil.
func (r *Reloader) Config() *tls.Config { return r.current.Load() }

// buildServer loads the key pair and assembles the TLS 1.3 server
// configuration.
func buildServer(cfg domain.ConfigTLS) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: load certificate: %w", err)
	}
	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}
	switch cfg.ClientAuth {
	case domain.ClientAuthVerifyIfGiven:
		conf.ClientAuth = tls.VerifyClientCertIfGiven
	case domain.ClientAuthRequireAndVerify:
		conf.ClientAuth = tls.RequireAndVerifyClientCert
	default:
		conf.ClientAuth = tls.NoClientCert
	}
	if cfg.CAFile != "" {
		pool, err := loadCAPool(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		conf.ClientCAs = pool
	}
	return conf, nil
}
