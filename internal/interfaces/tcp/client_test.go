package tcp

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/testcert"
)

// dialTestConfig returns a client configuration pointing at addr.
func dialTestConfig(addr string) domain.ConfigClient {
	cfg := domain.DefaultConfigClient()
	cfg.Server = addr
	return cfg
}

func TestDial_Plain(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	conn, err := Dial(context.Background(), dialTestConfig(listener.Addr().String()), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("Dial() = %v, want nil", err)
	}
	defer conn.Close()
	if _, ok := conn.(*tls.Conn); ok {
		t.Fatal("Dial() returned a TLS connection in plain mode")
	}

	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatalf("Write() = %v", err)
	}
	buf := make([]byte, 5)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("ReadFull() = %v", err)
	}
	if string(buf) != "hello" {
		t.Fatalf("echo = %q, want hello", buf)
	}
}

func TestDial_TLS(t *testing.T) {
	files, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	cert, err := tls.LoadX509KeyPair(files.CertFile, files.KeyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair() = %v", err)
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	})
	if err != nil {
		t.Fatalf("tls.Listen() = %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if tconn, ok := conn.(*tls.Conn); ok {
			_ = tconn.Handshake()
		}
		_, _ = io.Copy(io.Discard, conn)
	}()

	cfg := dialTestConfig(listener.Addr().String())
	cfg.TLS = domain.ConfigTLSClient{Enable: true, CAFile: files.CAFile, ServerName: "localhost"}
	conn, err := Dial(context.Background(), cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("Dial() = %v, want nil", err)
	}
	defer conn.Close()
	tconn, ok := conn.(*tls.Conn)
	if !ok {
		t.Fatal("Dial() returned a plain connection in TLS mode")
	}
	if version := tconn.ConnectionState().Version; version != tls.VersionTLS13 {
		t.Fatalf("protocol version = %x, want TLS 1.3", version)
	}
}

func TestDial_Insecure(t *testing.T) {
	files, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	cert, err := tls.LoadX509KeyPair(files.CertFile, files.KeyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair() = %v", err)
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	})
	if err != nil {
		t.Fatalf("tls.Listen() = %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if tconn, ok := conn.(*tls.Conn); ok {
			_ = tconn.Handshake()
		}
		_, _ = io.Copy(io.Discard, conn)
	}()

	cfg := dialTestConfig(listener.Addr().String())
	cfg.TLS = domain.ConfigTLSClient{Enable: true, Insecure: true}
	conn, err := Dial(context.Background(), cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("Dial() = %v, want nil with the insecure option", err)
	}
	_ = conn.Close()
}

func TestDial_Errors(t *testing.T) {
	// A listener that never speaks TLS: the TLS handshake must fail.
	plainListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	defer plainListener.Close()
	go func() {
		for {
			conn, err := plainListener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	// A closed port: the dial itself must fail.
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	closedAddr := closed.Addr().String()
	_ = closed.Close()

	tests := []struct {
		name string
		cfg  func() domain.ConfigClient
	}{
		{
			name: "connection refused",
			cfg:  func() domain.ConfigClient { return dialTestConfig(closedAddr) },
		},
		{
			name: "TLS handshake against a plain listener",
			cfg: func() domain.ConfigClient {
				cfg := dialTestConfig(plainListener.Addr().String())
				cfg.TLS = domain.ConfigTLSClient{Enable: true, Insecure: true}
				return cfg
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := Dial(context.Background(), tt.cfg(), slog.New(slog.DiscardHandler))
			if err == nil {
				_ = conn.Close()
				t.Fatal("Dial() = nil, want an error")
			}
		})
	}
}
