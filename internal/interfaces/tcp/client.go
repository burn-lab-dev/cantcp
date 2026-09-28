package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/adapters/tlsconfig"
	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// dialTimeout bounds the plain and TLS connection setup.
const dialTimeout = 10 * time.Second

// Dial opens a cantcp connection to the server described by cfg: plain TCP
// or TLS 1.3. The caller closes the returned connection.
func Dial(ctx context.Context, cfg domain.ConfigClient, log *slog.Logger) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	if !cfg.TLS.Enable {
		conn, err := dialer.DialContext(ctx, "tcp", cfg.Server)
		if err != nil {
			return nil, fmt.Errorf("connect to %s: %w", cfg.Server, err)
		}
		return conn, nil
	}
	tlsCfg, err := tlsconfig.NewClient(cfg.TLS)
	if err != nil {
		return nil, err
	}
	if cfg.TLS.Insecure {
		log.Warn("TLS certificate verification is disabled: debugging only")
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", cfg.Server, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to %s over TLS: %w", cfg.Server, err)
	}
	log.Debug("TLS connection established",
		"server", cfg.Server, "version", tls.VersionName(conn.ConnectionState().Version))
	return conn, nil
}
