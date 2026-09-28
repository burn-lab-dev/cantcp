package tlsconfig

import (
	"crypto/tls"
	"fmt"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// NewClient builds the client configuration: TLS 1.3, an optional CA file
// replacing the system roots, an optional client certificate for mTLS, an
// optional server name override, and the debugging-only insecure mode.
func NewClient(cfg domain.ConfigTLSClient) (*tls.Config, error) {
	conf := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		ServerName: cfg.ServerName,
		// InsecureSkipVerify is an explicit, validated configuration option
		// intended for debugging only; the daemon documentation marks it as
		// dangerous.
		InsecureSkipVerify: cfg.Insecure,
	}
	if cfg.CAFile != "" {
		pool, err := loadCAPool(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		conf.RootCAs = pool
	}
	if cfg.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("tls: load client certificate: %w", err)
		}
		conf.Certificates = []tls.Certificate{cert}
	}
	return conf, nil
}
