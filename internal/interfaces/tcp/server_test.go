package tcp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"

	"github.com/burn-lab-dev/cantcp/internal/adapters/tlsconfig"
	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/app/services"
	"github.com/burn-lab-dev/cantcp/internal/repositories"
	"github.com/burn-lab-dev/cantcp/internal/testbus"
	"github.com/burn-lab-dev/cantcp/internal/testcert"
)

// testServer is a running server over an in-memory bus.
type testServer struct {
	bus    *testbus.Bus
	stats  *repositories.Stats
	bridge *services.Bridge
	server *Server
}

// freePort reserves and releases a loopback port for the server under test.
func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

// startServer starts a plain or TLS server on a loopback port and stops it
// in the test cleanup.
func startServer(t *testing.T, mutate func(*domain.ConfigServer), tlsReloader *tlsconfig.Reloader) *testServer {
	t.Helper()
	bus := testbus.New()
	stats := repositories.NewStats("test", "can0", time.Now())
	logger := slog.New(slog.DiscardHandler)
	bridge := services.NewBridge(bus, stats, logger)
	cfg := domain.DefaultConfigServer()
	cfg.Listen = freePort(t)
	if mutate != nil {
		mutate(&cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("configuration is invalid: %v", err)
	}
	srv := NewServer(cfg, bridge, stats, tlsReloader, logger)
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	bridgeDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		_ = srv.Serve(ctx)
	}()
	go func() {
		defer close(bridgeDone)
		_ = bridge.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		bus.Close()
		<-serveDone
		<-bridgeDone
	})
	return &testServer{bus: bus, stats: stats, bridge: bridge, server: srv}
}

// dial opens a plain client connection and closes it in the cleanup.
func (s *testServer) dial(t *testing.T) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", s.server.Addr().String())
	if err != nil {
		t.Fatalf("Dial() = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// waitUntil waits for cond for up to three seconds or fails the test.
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func TestServer_PlainExchange(t *testing.T) {
	s := startServer(t, nil, nil)
	conn := s.dial(t)

	in := testbus.Frame(0x123, 0xDE, 0xAD, 0xBE, 0xEF)
	if err := cantcp.NewEncoder(conn).Encode(in); err != nil {
		t.Fatalf("Encode() = %v", err)
	}
	waitUntil(t, "the frame on the bus", func() bool { return len(s.bus.Written()) == 1 })
	if got := s.bus.Written()[0]; !bytes.Equal(got, in) {
		t.Fatalf("bus frame = %x, want %x", got, in)
	}

	out := testbus.Frame(0x7FF, 1, 2)
	s.bus.Push(out)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := cantcp.NewDecoder(conn).Decode()
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if !bytes.Equal(got, out) {
		t.Fatalf("received frame = %x, want %x", got, out)
	}

	waitUntil(t, "the statistics", func() bool {
		snap := s.stats.Snapshot(time.Now())
		return snap.TCP.ConnectionsCurrent == 1 &&
			snap.TCP.ConnectionsTotal >= 1 &&
			snap.TCP.FramesIn == 1 && snap.TCP.FramesOut == 1 &&
			snap.CAN.FramesRead == 1 && snap.CAN.FramesWritten == 1 &&
			snap.TCP.BytesIn > 0 && snap.TCP.BytesOut > 0
	})
}

func TestServer_MaxConnectionsAndUpdateConfig(t *testing.T) {
	s := startServer(t, func(cfg *domain.ConfigServer) { cfg.Limits.MaxConnections = 2 }, nil)
	first := s.dial(t)
	waitUntil(t, "the first client", func() bool {
		return s.stats.Snapshot(time.Now()).TCP.ConnectionsCurrent == 1
	})

	// Reduce the limit: the next accepted connection must be rejected.
	cfg := *s.server.cfg.Load()
	cfg.Limits.MaxConnections = 1
	s.server.UpdateConfig(cfg)

	second := s.dial(t)
	_ = second.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := second.Read(make([]byte, 1)); err == nil {
		t.Fatal("the connection over the limit must be closed")
	}

	// The first client stays connected: a read returns a timeout, not data
	// and not a close.
	_ = first.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := first.Read(make([]byte, 1))
	var nerr net.Error
	if !errors.As(err, &nerr) || !nerr.Timeout() {
		t.Fatalf("first client read = %v, want a timeout", err)
	}
}

func TestServer_IdleTimeout(t *testing.T) {
	s := startServer(t, func(cfg *domain.ConfigServer) {
		cfg.Limits.IdleTimeout = 100 * time.Millisecond
	}, nil)
	conn := s.dial(t)
	waitUntil(t, "the client", func() bool {
		return s.stats.Snapshot(time.Now()).TCP.ConnectionsCurrent == 1
	})

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	start := time.Now()
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("an idle client must be disconnected")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("the idle disconnect took %v, want about 100ms", elapsed)
	}
	waitUntil(t, "the closed client", func() bool {
		return s.stats.Snapshot(time.Now()).TCP.ConnectionsCurrent == 0
	})
}

func TestServer_RateLimit(t *testing.T) {
	s := startServer(t, func(cfg *domain.ConfigServer) {
		cfg.Limits.MaxFramesPerSecond = 2
	}, nil)
	conn := s.dial(t)
	waitUntil(t, "the client", func() bool {
		return s.stats.Snapshot(time.Now()).TCP.ConnectionsCurrent == 1
	})

	enc := cantcp.NewEncoder(conn)
	for i := 0; i < 5; i++ {
		if err := enc.Encode(testbus.Frame(0x200 + uint32(i))); err != nil {
			t.Fatalf("Encode() = %v", err)
		}
	}
	waitUntil(t, "two frames on the bus", func() bool { return len(s.bus.Written()) == 2 })
	time.Sleep(300 * time.Millisecond)
	if got := len(s.bus.Written()); got != 2 {
		t.Fatalf("bus frames = %d, want 2: the rate limit must drop the rest", got)
	}
	if got := s.stats.Snapshot(time.Now()).TCP.Dropped; got != 3 {
		t.Fatalf("dropped = %d, want 3", got)
	}
}

// tlsServerFiles generates a server certificate chain for the TLS tests.
func tlsServerFiles(t *testing.T) testcert.Files {
	t.Helper()
	files, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	return files
}

// caPool builds a certificate pool from the CA PEM.
func caPool(t *testing.T, files testcert.Files) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(files.CAPEM) {
		t.Fatal("AppendCertsFromPEM() = false, want true")
	}
	return pool
}

func TestServer_TLSExchange(t *testing.T) {
	files := tlsServerFiles(t)
	reloader, err := tlsconfig.NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone,
	})
	if err != nil {
		t.Fatalf("tlsconfig.NewServer() = %v", err)
	}
	s := startServer(t, nil, reloader)
	addr := s.server.Addr().String()
	pool := caPool(t, files)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	})
	if err != nil {
		t.Fatalf("tls.Dial() = %v", err)
	}
	defer conn.Close()
	if version := conn.ConnectionState().Version; version != tls.VersionTLS13 {
		t.Fatalf("protocol version = %x, want TLS 1.3", version)
	}

	in := testbus.Frame(0x321, 1)
	if err := cantcp.NewEncoder(conn).Encode(in); err != nil {
		t.Fatalf("Encode() = %v", err)
	}
	waitUntil(t, "the frame on the bus", func() bool { return len(s.bus.Written()) == 1 })

	out := testbus.Frame(0x322, 2)
	s.bus.Push(out)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := cantcp.NewDecoder(conn).Decode()
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if !bytes.Equal(got, out) {
		t.Fatalf("received frame = %x, want %x", got, out)
	}
}

func TestServer_TLSRejectsLegacyVersion(t *testing.T) {
	files := tlsServerFiles(t)
	reloader, err := tlsconfig.NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone,
	})
	if err != nil {
		t.Fatalf("tlsconfig.NewServer() = %v", err)
	}
	s := startServer(t, nil, reloader)
	_, err = tls.Dial("tcp", s.server.Addr().String(), &tls.Config{
		RootCAs: caPool(t, files), ServerName: "localhost", MaxVersion: tls.VersionTLS12,
	})
	if err == nil {
		t.Fatal("a TLS 1.2 client connected, want a protocol version error")
	}
}

func TestServer_TLSRejectsPlainClient(t *testing.T) {
	files := tlsServerFiles(t)
	reloader, err := tlsconfig.NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone,
	})
	if err != nil {
		t.Fatalf("tlsconfig.NewServer() = %v", err)
	}
	s := startServer(t, nil, reloader)
	conn := s.dial(t)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("a plain client on a TLS server must be disconnected")
	}
}

func TestServer_RejectsUnknownTLSClient(t *testing.T) {
	files := tlsServerFiles(t)
	reloader, err := tlsconfig.NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone,
	})
	if err != nil {
		t.Fatalf("tlsconfig.NewServer() = %v", err)
	}
	s := startServer(t, nil, reloader)
	// A TLS client that does not trust the server CA: the server must close
	// the connection after the failed handshake.
	conn, err := tls.Dial("tcp", s.server.Addr().String(), &tls.Config{
		ServerName: "localhost", MinVersion: tls.VersionTLS13,
	})
	if err == nil {
		_ = conn.Close()
		t.Fatal("tls.Dial() = nil, want a certificate verification error")
	}
}
