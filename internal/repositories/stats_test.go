package repositories

import (
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// startTime is the fixed daemon start used by the tests.
var startTime = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func TestStats_EmptySnapshot(t *testing.T) {
	stats := NewStats("v0.1.0", "can0", startTime)
	snap := stats.Snapshot(startTime.Add(90 * time.Second))

	if snap.Version != "v0.1.0" {
		t.Fatalf("Version = %q, want v0.1.0", snap.Version)
	}
	if snap.UptimeSeconds != 90 {
		t.Fatalf("UptimeSeconds = %d, want 90", snap.UptimeSeconds)
	}
	if snap.CAN.Interface != "can0" {
		t.Fatalf("CAN.Interface = %q, want can0", snap.CAN.Interface)
	}
	if len(snap.Clients) != 0 {
		t.Fatalf("Clients = %v, want empty", snap.Clients)
	}
	if (snap.TCP != domain.StatsTCP{}) {
		t.Fatalf("TCP = %+v, want zero", snap.TCP)
	}
	if (snap.CAN != domain.StatsCAN{Interface: "can0"}) {
		t.Fatalf("CAN = %+v, want only the interface", snap.CAN)
	}
}

func TestStats_Counters(t *testing.T) {
	stats := NewStats("v0.1.0", "can0", startTime)
	stats.CANFrameRead()
	stats.CANFrameRead()
	stats.CANFrameWritten()
	stats.CANReadError()
	stats.CANWriteError()
	stats.FrameIn()
	stats.FrameOut()
	stats.FrameOut()
	stats.BytesIn(100)
	stats.BytesOut(250)
	stats.Dropped()

	id1, c1 := stats.ClientConnected("10.0.0.2:5000", startTime.Add(time.Second))
	id2, c2 := stats.ClientConnected("10.0.0.1:4000", startTime.Add(2*time.Second))
	c1.FrameIn()
	c1.FrameOut()
	c1.BytesIn(10)
	c1.BytesOut(20)
	c1.FrameDropped()
	c2.FrameIn()

	snap := stats.Snapshot(startTime.Add(time.Minute))
	if got := snap.TCP.ConnectionsCurrent; got != 2 {
		t.Fatalf("ConnectionsCurrent = %d, want 2", got)
	}
	wantCAN := domain.StatsCAN{
		Interface:     "can0",
		FramesRead:    2,
		FramesWritten: 1,
		ReadErrors:    1,
		WriteErrors:   1,
	}
	if snap.CAN != wantCAN {
		t.Fatalf("CAN = %+v, want %+v", snap.CAN, wantCAN)
	}
	wantTCP := domain.StatsTCP{
		ConnectionsCurrent: 2,
		ConnectionsTotal:   2,
		FramesIn:           1,
		FramesOut:          2,
		BytesIn:            100,
		BytesOut:           250,
		Dropped:            1,
	}
	if snap.TCP != wantTCP {
		t.Fatalf("TCP = %+v, want %+v", snap.TCP, wantTCP)
	}
	wantClients := []domain.StatsClient{
		{
			Socket:    "10.0.0.1:4000",
			Since:     startTime.Add(2 * time.Second),
			FramesIn:  1,
			BytesIn:   0,
			BytesOut:  0,
			FramesOut: 0,
			Dropped:   0,
		},
		{
			Socket:    "10.0.0.2:5000",
			Since:     startTime.Add(time.Second),
			FramesIn:  1,
			FramesOut: 1,
			BytesIn:   10,
			BytesOut:  20,
			Dropped:   1,
		},
	}
	if len(snap.Clients) != len(wantClients) {
		t.Fatalf("Clients = %+v, want %+v", snap.Clients, wantClients)
	}
	for i := range wantClients {
		if snap.Clients[i] != wantClients[i] {
			t.Fatalf("Clients[%d] = %+v, want %+v", i, snap.Clients[i], wantClients[i])
		}
	}

	stats.ClientDisconnected(id1)
	stats.ClientDisconnected(id2)
	snap = stats.Snapshot(startTime)
	if snap.TCP.ConnectionsCurrent != 0 {
		t.Fatalf("ConnectionsCurrent = %d, want 0 after disconnects", snap.TCP.ConnectionsCurrent)
	}
	if snap.TCP.ConnectionsTotal != 2 {
		t.Fatalf("ConnectionsTotal = %d, want 2", snap.TCP.ConnectionsTotal)
	}
	if len(snap.Clients) != 0 {
		t.Fatalf("Clients = %v, want empty after disconnects", snap.Clients)
	}
}

func TestStats_SnapshotDoesNotGoBackInTime(t *testing.T) {
	stats := NewStats("dev", "can0", startTime)
	snap := stats.Snapshot(startTime.Add(-time.Minute))
	if snap.UptimeSeconds != 0 {
		t.Fatalf("UptimeSeconds = %d, want 0 for a snapshot before the start", snap.UptimeSeconds)
	}
}
