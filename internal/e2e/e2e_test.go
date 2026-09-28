//go:build linux

// Package e2e runs the complete daemon stack over a virtual CAN interface:
// the real socketcan adapter, the bridge, the TCP listener, the statistics
// endpoint and a cantcp client.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	nethttp "net/http"
	"testing"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"

	"github.com/burn-lab-dev/cantcp/internal/adapters/socketcan"
	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/app/services"
	statshttp "github.com/burn-lab-dev/cantcp/internal/interfaces/http"
	"github.com/burn-lab-dev/cantcp/internal/interfaces/tcp"
	"github.com/burn-lab-dev/cantcp/internal/repositories"
)

// testInterface returns vcan0 when it is available.
func testInterface(t *testing.T) string {
	t.Helper()
	ifi, err := net.InterfaceByName("vcan0")
	if err != nil {
		t.Skip("vcan0 is not available: run modprobe vcan and ip link add dev vcan0 type vcan")
	}
	if ifi.Flags&net.FlagUp == 0 {
		t.Skip("vcan0 is down: run ip link set up vcan0")
	}
	return "vcan0"
}

// freePort reserves and releases a loopback port.
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

// rawFrame builds a raw classic frame.
func rawFrame(id uint32, data ...byte) []byte {
	f := cantcp.Frame{ID: id, Type: cantcp.TypeClassic, Data: data}
	raw, err := f.MarshalBinary()
	if err != nil {
		panic("e2e: " + err.Error())
	}
	return raw
}

// readDeadline reads one frame from the connection with a deadline.
func readDeadline(t *testing.T, dec interface{ Decode() ([]byte, error) }) []byte {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := dec.Decode()
		ch <- result{data: data, err: err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("Decode() = %v", res.err)
		}
		return res.data
	case <-time.After(3 * time.Second):
		t.Fatal("no frame received within the deadline")
		return nil
	}
}

func TestEndToEnd_Vcan(t *testing.T) {
	iface := testInterface(t)

	// The daemon side.
	bus, err := socketcan.Open(iface, false)
	if err != nil {
		t.Fatalf("socketcan.Open() = %v", err)
	}
	defer bus.Close()
	stats := repositories.NewStats("e2e", iface, time.Now())
	logger := slog.New(slog.DiscardHandler)
	bridge := services.NewBridge(bus, stats, logger)

	cfg := domain.DefaultConfigServer()
	cfg.Listen = freePort(t)
	cfg.Stats.Listen = freePort(t)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("configuration is invalid: %v", err)
	}
	tcpServer := tcp.NewServer(cfg, bridge, stats, nil, logger)
	if err := tcpServer.Listen(); err != nil {
		t.Fatalf("tcp.Listen() = %v", err)
	}
	statsServer := statshttp.NewServer(cfg.Stats, stats, logger)
	if err := statsServer.Listen(); err != nil {
		t.Fatalf("stats Listen() = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-ctx.Done(); _ = bus.Close() }()
	components := make(chan struct{}, 3)
	go func() { defer func() { components <- struct{}{} }(); _ = bridge.Run(ctx) }()
	go func() { defer func() { components <- struct{}{} }(); _ = tcpServer.Serve(ctx) }()
	go func() { defer func() { components <- struct{}{} }(); _ = statsServer.Run(ctx) }()
	defer func() {
		cancel()
		bus.Close()
		for i := 0; i < 3; i++ {
			<-components
		}
	}()

	// The peer bus emulates the remote CAN node on the same vcan interface.
	peer, err := socketcan.Open(iface, false)
	if err != nil {
		t.Fatalf("socketcan.Open(peer) = %v", err)
	}
	defer peer.Close()

	conn, err := net.Dial("tcp", tcpServer.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	enc := cantcp.NewEncoder(conn)
	dec := cantcp.NewDecoder(conn)

	// client -> daemon -> bus
	fromClient := rawFrame(0x123, 1, 2, 3)
	if err := enc.Encode(fromClient); err != nil {
		t.Fatalf("Encode() = %v", err)
	}
	peerBuf := make([]byte, 72)
	peerRead := make(chan []byte, 1)
	go func() {
		n, err := peer.ReadFrame(peerBuf)
		if err == nil {
			peerRead <- bytes.Clone(peerBuf[:n])
		} else {
			peerRead <- nil
		}
	}()
	select {
	case got := <-peerRead:
		if !bytes.Equal(got, fromClient) {
			t.Fatalf("peer frame = %x, want %x", got, fromClient)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the daemon did not write the frame to the bus")
	}

	// The gateway echoes the frames it wrote itself (CAN_RAW_RECV_OWN_MSGS):
	// the sender sees its own frame as if it appeared on the bus.
	if got := readDeadline(t, dec); !bytes.Equal(got, fromClient) {
		t.Fatalf("echo frame = %x, want %x", got, fromClient)
	}

	// bus -> daemon -> client
	fromBus := rawFrame(0x456, 4, 5)
	if err := peer.WriteFrame(fromBus); err != nil {
		t.Fatalf("peer.WriteFrame() = %v", err)
	}
	if got := readDeadline(t, dec); !bytes.Equal(got, fromBus) {
		t.Fatalf("client frame = %x, want %x", got, fromBus)
	}

	// statistics endpoint
	client := &nethttp.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + statsServer.Addr().String() + "/api/v1/stats")
	if err != nil {
		t.Fatalf("GET stats = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("stats status = %d, want 200", resp.StatusCode)
	}
	var snap domain.Stats
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if snap.CAN.Interface != iface {
		t.Fatalf("CAN.Interface = %q, want %q", snap.CAN.Interface, iface)
	}
	if snap.CAN.FramesRead < 1 || snap.CAN.FramesWritten < 1 {
		t.Fatalf("CAN = %+v, want at least one frame in both directions", snap.CAN)
	}
	if snap.TCP.FramesIn < 1 || snap.TCP.FramesOut < 1 {
		t.Fatalf("TCP = %+v, want at least one frame in both directions", snap.TCP)
	}
	if len(snap.Clients) != 1 {
		t.Fatalf("Clients = %+v, want exactly one client", snap.Clients)
	}
}
