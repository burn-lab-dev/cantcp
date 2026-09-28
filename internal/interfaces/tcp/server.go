// Package tcp implements the cantcp TCP listener of the daemon and the
// client dialer.
package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"

	"github.com/burn-lab-dev/cantcp/internal/adapters/tlsconfig"
	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/app/services"
	"github.com/burn-lab-dev/cantcp/internal/repositories"
)

// iBridge is the bridge dependency of the server.
type iBridge interface {
	Subscribe(queueSize int, counters services.ClientCounters) *services.Subscription
	WriteFrame(frame []byte) error
}

// iStats is the counter sink of the server.
type iStats interface {
	ClientConnected(socket string, since time.Time) (int64, *repositories.ClientCounters)
	ClientDisconnected(id int64)
	FrameIn()
	FrameOut()
	BytesIn(n uint64)
	BytesOut(n uint64)
	Dropped()
}

// Server accepts cantcp clients and connects them to the bridge. The
// configuration is re-read on every accept, so the SIGHUP reload applies the
// limits to new connections.
type Server struct {
	cfg    atomic.Pointer[domain.ConfigServer]
	bridge iBridge
	stats  iStats
	tls    *tlsconfig.Reloader
	log    *slog.Logger

	listener net.Listener
	conns    sync.WaitGroup
	current  atomic.Int64
	closed   atomic.Bool
}

// NewServer builds the server. tlsReloader is nil in plain mode.
func NewServer(cfg domain.ConfigServer, bridge iBridge, stats iStats, tlsReloader *tlsconfig.Reloader, log *slog.Logger) *Server {
	s := &Server{bridge: bridge, stats: stats, tls: tlsReloader, log: log}
	s.cfg.Store(&cfg)
	return s
}

// Listen opens the TCP listener on the configured address.
func (s *Server) Listen() error {
	addr := s.cfg.Load().Listen
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp: listen %s: %w", addr, err)
	}
	s.listener = ln
	return nil
}

// Addr returns the actual listener address, nil before Listen.
func (s *Server) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// UpdateConfig replaces the reloadable settings (limits). The listen address
// and the TLS mode are fixed for the lifetime of the process.
func (s *Server) UpdateConfig(cfg domain.ConfigServer) { s.cfg.Store(&cfg) }

// Serve accepts connections until ctx is canceled. It must be called after
// Listen.
func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return errors.New("tcp: Serve called before Listen")
	}
	go func() {
		<-ctx.Done()
		s.closed.Store(true)
		_ = s.listener.Close()
	}()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() || errors.Is(err, net.ErrClosed) {
				break
			}
			return fmt.Errorf("tcp: accept: %w", err)
		}
		cfg := s.cfg.Load()
		if s.current.Load() >= int64(cfg.Limits.MaxConnections) {
			s.log.Warn("connection rejected: too many clients", "remote", conn.RemoteAddr().String())
			_ = conn.Close()
			continue
		}
		s.current.Add(1)
		s.conns.Add(1)
		go func() {
			defer s.conns.Done()
			defer s.current.Add(-1)
			s.handle(ctx, conn)
		}()
	}
	s.conns.Wait()
	return nil
}

// handle runs one client connection: the TLS handshake, the counters, the
// subscription and the read/write loops.
func (s *Server) handle(ctx context.Context, conn net.Conn) {
	cfg := *s.cfg.Load()
	remote := conn.RemoteAddr().String()
	log := s.log.With("client", remote)

	if s.tls != nil {
		tconn := tls.Server(conn, s.tls.Config())
		_ = tconn.SetDeadline(time.Now().Add(cfg.Limits.ReadTimeout))
		if err := tconn.HandshakeContext(ctx); err != nil {
			log.Warn("TLS handshake failed", "error", err)
			_ = conn.Close()
			return
		}
		_ = tconn.SetDeadline(time.Time{})
		conn = tconn
	}

	id, counters := s.stats.ClientConnected(remote, time.Now())
	defer s.stats.ClientDisconnected(id)

	sub := s.bridge.Subscribe(cfg.Limits.ClientQueue, counters)
	defer sub.Close()

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn = &countingConn{
		Conn:    conn,
		onRead:  func(n int) { s.stats.BytesIn(uint64(n)); counters.BytesIn(uint64(n)) },
		onWrite: func(n int) { s.stats.BytesOut(uint64(n)); counters.BytesOut(uint64(n)) },
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.writeLoop(connCtx, conn, sub, counters, cfg.Limits, log)
		cancel()
		_ = conn.Close()
	}()
	s.readLoop(conn, counters, cfg.Limits, log)
	cancel()
	_ = conn.Close()
	wg.Wait()
}

// readLoop decodes the client stream and writes the frames to the CAN bus.
func (s *Server) readLoop(conn net.Conn, counters *repositories.ClientCounters, limits domain.ConfigLimits, log *slog.Logger) {
	dec := cantcp.NewDecoder(conn)
	var limiter *rateLimiter
	if limits.MaxFramesPerSecond > 0 {
		limiter = newRateLimiter(limits.MaxFramesPerSecond)
	}
	for {
		if limits.IdleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(limits.IdleTimeout))
		}
		frame, err := dec.Decode()
		if err != nil {
			logReadEnd(log, err)
			return
		}
		if limiter != nil && !limiter.allow(time.Now()) {
			counters.FrameDropped()
			s.stats.Dropped()
			s.log.Debug("frame dropped: rate limit", "client", conn.RemoteAddr().String())
			continue
		}
		if err := s.bridge.WriteFrame(frame); err != nil {
			log.Warn("CAN write failed", "error", err)
			return
		}
		counters.FrameIn()
		s.stats.FrameIn()
	}
}

// writeLoop encodes the frames queued by the bridge and writes them to the
// client. A write error ends the connection.
func (s *Server) writeLoop(ctx context.Context, conn net.Conn, sub *services.Subscription, counters *repositories.ClientCounters, limits domain.ConfigLimits, log *slog.Logger) {
	enc := cantcp.NewEncoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-sub.Frames():
			_ = conn.SetWriteDeadline(time.Now().Add(limits.WriteTimeout))
			if err := enc.Encode(frame); err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Warn("encode failed", "error", err)
				}
				return
			}
			counters.FrameOut()
			s.stats.FrameOut()
		}
	}
}

// logReadEnd classifies the end of the client stream: a clean close, a
// shutdown, an idle timeout or a real decode error.
func logReadEnd(log *slog.Logger, err error) {
	var nerr net.Error
	switch {
	case errors.Is(err, io.EOF):
		log.Debug("client closed the connection")
	case errors.Is(err, net.ErrClosed):
	case errors.As(err, &nerr) && nerr.Timeout():
		log.Debug("client idle timeout")
	default:
		log.Warn("decode failed", "error", err)
	}
}
