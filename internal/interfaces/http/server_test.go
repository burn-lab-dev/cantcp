package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	nethttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/repositories"
)

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

// startStats fills the statistics store, starts the server and returns its
// base URL.
func startStats(t *testing.T) (string, *repositories.Stats) {
	t.Helper()
	stats := repositories.NewStats("v0.1.0", "can0", time.Now().Add(-90*time.Second))
	stats.CANFrameRead()
	stats.CANFrameRead()
	stats.CANFrameWritten()
	stats.CANReadError()
	stats.FrameIn()
	stats.FrameOut()
	stats.Dropped()
	id, counters := stats.ClientConnected("10.0.0.9:5000", time.Now())
	counters.FrameOut()

	cfg := domain.ConfigStats{Enable: true, Listen: freePort(t)}
	srv := NewServer(cfg, stats, slog.New(slog.DiscardHandler))
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		stats.ClientDisconnected(id)
	})
	return "http://" + srv.Addr().String(), stats
}

// get performs a GET request or fails the test.
func get(t *testing.T, url string) *nethttp.Response {
	t.Helper()
	client := &nethttp.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s = %v", url, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestServer_Healthz(t *testing.T) {
	base, _ := startStats(t)
	resp := get(t, base+pathHealth)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v, want status ok", body)
	}
}

func TestServer_Stats(t *testing.T) {
	base, _ := startStats(t)
	resp := get(t, base+pathStats)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var snap domain.Stats
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if snap.Version != "v0.1.0" {
		t.Fatalf("Version = %q, want v0.1.0", snap.Version)
	}
	if snap.UptimeSeconds < 90 {
		t.Fatalf("UptimeSeconds = %d, want at least 90", snap.UptimeSeconds)
	}
	if snap.CAN.FramesRead != 2 || snap.CAN.FramesWritten != 1 || snap.CAN.ReadErrors != 1 {
		t.Fatalf("CAN = %+v, want 2 read, 1 written, 1 error", snap.CAN)
	}
	if snap.TCP.ConnectionsCurrent != 1 || snap.TCP.FramesIn != 1 || snap.TCP.FramesOut != 1 || snap.TCP.Dropped != 1 {
		t.Fatalf("TCP = %+v, want one client, one frame in, one out, one drop", snap.TCP)
	}
	if len(snap.Clients) != 1 || snap.Clients[0].Socket != "10.0.0.9:5000" || snap.Clients[0].FramesOut != 1 {
		t.Fatalf("Clients = %+v, want one client 10.0.0.9:5000 with one frame out", snap.Clients)
	}
}

func TestServer_Metrics(t *testing.T) {
	base, _ := startStats(t)
	resp := get(t, base+pathMetrics)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, metricsContentType) {
		t.Fatalf("Content-Type = %q, want %q", got, metricsContentType)
	}
	var body strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		body.Write(buf[:n])
		if err != nil {
			break
		}
	}
	out := body.String()
	for _, want := range []string{
		`cantcp_build_info{version="v0.1.0"} 1`,
		`cantcp_can_frames_total{direction="in"} 2`,
		`cantcp_can_frames_total{direction="out"} 1`,
		`cantcp_can_errors_total{direction="read"} 1`,
		`cantcp_can_errors_total{direction="write"} 0`,
		`cantcp_tcp_connections_current 1`,
		`cantcp_tcp_frames_total{direction="in"} 1`,
		`cantcp_tcp_frames_total{direction="out"} 1`,
		`cantcp_dropped_frames_total 1`,
		"# TYPE cantcp_can_frames_total counter",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("metrics output does not contain %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "cantcp_uptime_seconds") {
		t.Fatalf("metrics output does not contain the uptime:\n%s", out)
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	base, _ := startStats(t)
	client := &nethttp.Client{Timeout: 3 * time.Second}
	req, err := nethttp.NewRequest(nethttp.MethodPost, base+pathHealth, nil)
	if err != nil {
		t.Fatalf("NewRequest() = %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestServer_RunBeforeListen(t *testing.T) {
	srv := NewServer(domain.ConfigStats{Enable: true, Listen: "127.0.0.1:29537"}, nil, slog.New(slog.DiscardHandler))
	if err := srv.Run(context.Background()); err == nil {
		t.Fatal("Run() = nil, want an error before Listen")
	}
	if srv.Addr() != nil {
		t.Fatal("Addr() must be nil before Listen")
	}
}

func TestServer_ListenErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}
	defer listener.Close()

	srv := NewServer(domain.ConfigStats{Enable: true, Listen: listener.Addr().String()}, nil, slog.New(slog.DiscardHandler))
	if err := srv.Listen(); err == nil {
		t.Fatal("Listen() = nil, want a bind error")
	}
}
