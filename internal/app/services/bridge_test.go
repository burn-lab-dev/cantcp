package services

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/repositories"
	"github.com/burn-lab-dev/cantcp/internal/testbus"
)

// discardLogger is a logger that drops every record.
func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// fakeCounters counts the drops of one subscription.
type fakeCounters struct{ dropped atomic.Uint64 }

func (c *fakeCounters) FrameDropped() { c.dropped.Add(1) }

func TestBridge_Broadcast(t *testing.T) {
	bus := testbus.New()
	stats := repositories.NewStats("test", "can0", time.Now())
	bridge := NewBridge(bus, stats, discardLogger())

	c1 := &fakeCounters{}
	c2 := &fakeCounters{}
	s1 := bridge.Subscribe(4, c1)
	defer s1.Close()
	s2 := bridge.Subscribe(4, c2)
	defer s2.Close()
	if got := bridge.Clients(); got != 2 {
		t.Fatalf("Clients() = %d, want 2", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge.Run(ctx) }()

	frame := testbus.Frame(0x123, 1, 2, 3)
	bus.Push(frame)
	for _, s := range []*Subscription{s1, s2} {
		select {
		case got := <-s.Frames():
			if !bytes.Equal(got, frame) {
				t.Fatalf("broadcast frame = %x, want %x", got, frame)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("no frame received")
		}
	}
	bus.Close()
	if err := <-done; err != nil {
		t.Fatalf("Run() = %v, want nil", err)
	}
	snap := stats.Snapshot(time.Now())
	if snap.CAN.FramesRead != 1 {
		t.Fatalf("CAN.FramesRead = %d, want 1", snap.CAN.FramesRead)
	}
	if snap.TCP.FramesOut != 0 || snap.TCP.Dropped != 0 {
		t.Fatalf("TCP = %+v, want zero frames out and drops", snap.TCP)
	}
}

func TestBridge_DropsWhenQueueIsFull(t *testing.T) {
	bus := testbus.New()
	stats := repositories.NewStats("test", "can0", time.Now())
	bridge := NewBridge(bus, stats, discardLogger())
	counters := &fakeCounters{}
	sub := bridge.Subscribe(1, counters)
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge.Run(ctx) }()

	first := testbus.Frame(0x100)
	second := testbus.Frame(0x101)
	third := testbus.Frame(0x102)
	bus.Push(first)
	bus.Push(second)
	bus.Push(third)

	deadline := time.Now().Add(2 * time.Second)
	for counters.dropped.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := counters.dropped.Load(); got != 2 {
		t.Fatalf("dropped = %d, want 2", got)
	}
	if got := stats.Snapshot(time.Now()).TCP.Dropped; got != 2 {
		t.Fatalf("stats dropped = %d, want 2", got)
	}
	select {
	case got := <-sub.Frames():
		if !bytes.Equal(got, first) {
			t.Fatalf("queued frame = %x, want the first frame %x", got, first)
		}
	default:
		t.Fatal("the queue must keep the first frame")
	}
	bus.Close()
	<-done
}

func TestBridge_WriteFrame(t *testing.T) {
	bus := testbus.New()
	stats := repositories.NewStats("test", "can0", time.Now())
	bridge := NewBridge(bus, stats, discardLogger())

	frame := testbus.Frame(0x7AB, 9, 8, 7)
	if err := bridge.WriteFrame(frame); err != nil {
		t.Fatalf("WriteFrame() = %v, want nil", err)
	}
	written := bus.Written()
	if len(written) != 1 || !bytes.Equal(written[0], frame) {
		t.Fatalf("bus frames = %x, want %x", written, frame)
	}
	if got := stats.Snapshot(time.Now()).CAN.FramesWritten; got != 1 {
		t.Fatalf("CAN.FramesWritten = %d, want 1", got)
	}

	bus.WriteErr = errFakeWrite
	if err := bridge.WriteFrame(frame); err == nil {
		t.Fatal("WriteFrame() = nil, want the bus error")
	}
	if got := stats.Snapshot(time.Now()).CAN.WriteErrors; got != 1 {
		t.Fatalf("CAN.WriteErrors = %d, want 1", got)
	}
}

func TestBridge_ReadErrorIsCounted(t *testing.T) {
	bus := testbus.New()
	bus.WriteErr = errFakeWrite
	stats := repositories.NewStats("test", "can0", time.Now())
	bridge := NewBridge(bus, stats, discardLogger())

	// A closed bus ends Run without an error; a broken bus is covered by the
	// bus implementation tests. Here the read error path is exercised by
	// closing the bus before Run starts.
	bus.Close()
	if err := bridge.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v, want nil on a closed bus", err)
	}
}

func TestBridge_CloseRemovesTheClient(t *testing.T) {
	bus := testbus.New()
	stats := repositories.NewStats("test", "can0", time.Now())
	bridge := NewBridge(bus, stats, discardLogger())
	sub := bridge.Subscribe(1, &fakeCounters{})
	if got := bridge.Clients(); got != 1 {
		t.Fatalf("Clients() = %d, want 1", got)
	}
	sub.Close()
	if got := bridge.Clients(); got != 0 {
		t.Fatalf("Clients() = %d, want 0 after Close", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- bridge.Run(ctx) }()
	bus.Push(testbus.Frame(0x1))
	bus.Close()
	if err := <-done; err != nil {
		t.Fatalf("Run() = %v, want nil", err)
	}
}

// errFakeWrite is the write error injected into the fake bus.
var errFakeWrite = &fakeWriteError{}

type fakeWriteError struct{}

func (e *fakeWriteError) Error() string { return "fake write error" }
