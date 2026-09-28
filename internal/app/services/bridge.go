// Package services holds the long-running daemon components: the bridge
// between the CAN bus and the TCP clients.
package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

// iBus is the CAN bus dependency of the bridge.
type iBus interface {
	ReadFrame(buf []byte) (int, error)
	WriteFrame(frame []byte) error
}

// iStats is the counter sink of the bridge.
type iStats interface {
	CANFrameRead()
	CANFrameWritten()
	CANReadError()
	CANWriteError()
	FrameOut()
	Dropped()
}

// ClientCounters is the per-client counter sink used on frame drops.
type ClientCounters interface {
	FrameDropped()
}

// Bridge reads the CAN bus and fans the raw frames out to the subscribers;
// frames written by the clients go to the bus. A full client queue drops the
// frame and bumps the counters: a slow client never blocks the bus.
type Bridge struct {
	bus   iBus
	stats iStats
	log   *slog.Logger

	mu      sync.RWMutex
	clients map[*Subscription]struct{}
}

// NewBridge builds the bridge.
func NewBridge(bus iBus, stats iStats, log *slog.Logger) *Bridge {
	return &Bridge{
		bus:     bus,
		stats:   stats,
		log:     log,
		clients: make(map[*Subscription]struct{}),
	}
}

// Run reads the CAN bus until the bus is closed or fails. The caller closes
// the bus to stop the bridge.
func (b *Bridge) Run(ctx context.Context) error {
	buf := make([]byte, maxFrameLen)
	for {
		n, err := b.bus.ReadFrame(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			b.stats.CANReadError()
			return fmt.Errorf("bridge: %w", err)
		}
		frame := bytes.Clone(buf[:n])
		b.stats.CANFrameRead()
		b.broadcast(frame)
		if ctx.Err() != nil {
			return nil
		}
	}
}

// WriteFrame sends one raw frame (16 or 72 bytes) to the CAN bus.
func (b *Bridge) WriteFrame(frame []byte) error {
	if err := b.bus.WriteFrame(frame); err != nil {
		b.stats.CANWriteError()
		return err
	}
	b.stats.CANFrameWritten()
	return nil
}

// Subscribe registers a client with a frame queue of the given size. The
// returned subscription must be closed with Close.
func (b *Bridge) Subscribe(queueSize int, counters ClientCounters) *Subscription {
	s := &Subscription{
		frames:   make(chan []byte, queueSize),
		counters: counters,
	}
	s.close = func() {
		b.mu.Lock()
		delete(b.clients, s)
		b.mu.Unlock()
	}
	b.mu.Lock()
	b.clients[s] = struct{}{}
	b.mu.Unlock()
	return s
}

// Clients returns the current number of subscribers.
func (b *Bridge) Clients() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// broadcast delivers the frame to every subscriber without blocking.
func (b *Bridge) broadcast(frame []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.clients {
		select {
		case s.frames <- frame:
		default:
			s.counters.FrameDropped()
			b.stats.Dropped()
		}
	}
}

// Subscription is one client connection as seen by the bridge.
type Subscription struct {
	frames   chan []byte
	counters ClientCounters
	close    func()
}

// Frames returns the receive side of the frame queue: raw CAN frames
// destined for the client.
func (s *Subscription) Frames() <-chan []byte { return s.frames }

// Close removes the subscription.
func (s *Subscription) Close() { s.close() }
