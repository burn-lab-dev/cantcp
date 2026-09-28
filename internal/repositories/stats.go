// Package repositories implements the storage side of the application: the
// in-memory statistics store of the daemon.
package repositories

import (
	"cmp"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// Stats is the in-memory statistics store: atomic counters for the aggregate
// values and a mutex-protected registry of the per-client counters. All
// methods are safe for concurrent use.
type Stats struct {
	version   string
	iface     string
	startedAt time.Time

	canFramesRead    atomic.Uint64
	canFramesWritten atomic.Uint64
	canReadErrors    atomic.Uint64
	canWriteErrors   atomic.Uint64

	connCurrent atomic.Int64
	connTotal   atomic.Uint64
	framesIn    atomic.Uint64
	framesOut   atomic.Uint64
	bytesIn     atomic.Uint64
	bytesOut    atomic.Uint64
	dropped     atomic.Uint64

	nextID  atomic.Int64
	mu      sync.Mutex
	clients map[int64]*ClientCounters
}

// ClientCounters is the per-client counter block. The daemon holds the
// pointer for the lifetime of the connection, so the hot path updates it
// without a map lookup.
type ClientCounters struct {
	socket    string
	since     time.Time
	framesIn  atomic.Uint64
	framesOut atomic.Uint64
	bytesIn   atomic.Uint64
	bytesOut  atomic.Uint64
	dropped   atomic.Uint64
}

// NewStats builds the store. version and iface are reported in the snapshot,
// startedAt is the daemon start time used for the uptime.
func NewStats(version, iface string, startedAt time.Time) *Stats {
	return &Stats{
		version:   version,
		iface:     iface,
		startedAt: startedAt,
		clients:   make(map[int64]*ClientCounters),
	}
}

// CANFrameRead records one frame read from the CAN interface.
func (s *Stats) CANFrameRead() { s.canFramesRead.Add(1) }

// CANFrameWritten records one frame written to the CAN interface.
func (s *Stats) CANFrameWritten() { s.canFramesWritten.Add(1) }

// CANReadError records one failed CAN read.
func (s *Stats) CANReadError() { s.canReadErrors.Add(1) }

// CANWriteError records one failed CAN write.
func (s *Stats) CANWriteError() { s.canWriteErrors.Add(1) }

// ClientConnected registers a client and returns its id and counter block.
func (s *Stats) ClientConnected(socket string, since time.Time) (int64, *ClientCounters) {
	id := s.nextID.Add(1)
	cc := &ClientCounters{socket: socket, since: since}
	s.mu.Lock()
	s.clients[id] = cc
	s.mu.Unlock()
	s.connCurrent.Add(1)
	s.connTotal.Add(1)
	return id, cc
}

// ClientDisconnected removes the client.
func (s *Stats) ClientDisconnected(id int64) {
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
	s.connCurrent.Add(-1)
}

// FrameIn records one frame received from a client.
func (s *Stats) FrameIn() { s.framesIn.Add(1) }

// FrameOut records one frame sent to a client.
func (s *Stats) FrameOut() { s.framesOut.Add(1) }

// BytesIn records n bytes read from clients.
func (s *Stats) BytesIn(n uint64) { s.bytesIn.Add(n) }

// BytesOut records n bytes written to clients.
func (s *Stats) BytesOut(n uint64) { s.bytesOut.Add(n) }

// Dropped records one frame dropped because a client queue was full.
func (s *Stats) Dropped() { s.dropped.Add(1) }

// FrameIn records one frame received from the client.
func (c *ClientCounters) FrameIn() { c.framesIn.Add(1) }

// FrameOut records one frame sent to the client.
func (c *ClientCounters) FrameOut() { c.framesOut.Add(1) }

// BytesIn records n bytes read from the client.
func (c *ClientCounters) BytesIn(n uint64) { c.bytesIn.Add(n) }

// BytesOut records n bytes written to the client.
func (c *ClientCounters) BytesOut(n uint64) { c.bytesOut.Add(n) }

// FrameDropped records one frame dropped for the client.
func (c *ClientCounters) FrameDropped() { c.dropped.Add(1) }

// Snapshot returns a consistent point-in-time copy of all counters. The
// client list is sorted by socket address.
func (s *Stats) Snapshot(now time.Time) domain.Stats {
	s.mu.Lock()
	clients := make([]domain.StatsClient, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, domain.StatsClient{
			Socket:    c.socket,
			Since:     c.since,
			FramesIn:  c.framesIn.Load(),
			FramesOut: c.framesOut.Load(),
			BytesIn:   c.bytesIn.Load(),
			BytesOut:  c.bytesOut.Load(),
			Dropped:   c.dropped.Load(),
		})
	}
	s.mu.Unlock()
	slices.SortFunc(clients, func(a, b domain.StatsClient) int {
		return cmp.Compare(a.Socket, b.Socket)
	})

	uptime := max(int64(now.Sub(s.startedAt).Seconds()), 0)
	return domain.Stats{
		Version:       s.version,
		UptimeSeconds: uptime,
		CAN: domain.StatsCAN{
			Interface:     s.iface,
			FramesRead:    s.canFramesRead.Load(),
			FramesWritten: s.canFramesWritten.Load(),
			ReadErrors:    s.canReadErrors.Load(),
			WriteErrors:   s.canWriteErrors.Load(),
		},
		TCP: domain.StatsTCP{
			ConnectionsCurrent: uint64(max(s.connCurrent.Load(), 0)),
			ConnectionsTotal:   s.connTotal.Load(),
			FramesIn:           s.framesIn.Load(),
			FramesOut:          s.framesOut.Load(),
			BytesIn:            s.bytesIn.Load(),
			BytesOut:           s.bytesOut.Load(),
			Dropped:            s.dropped.Load(),
		},
		Clients: clients,
	}
}
