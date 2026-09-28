// Package testbus provides an in-memory CAN bus for tests: frames pushed to
// the read side are returned by ReadFrame, frames passed to WriteFrame are
// stored for inspection.
package testbus

import (
	"net"
	"sync"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// Bus is an in-memory CAN bus.
type Bus struct {
	// ReadFrames feeds ReadFrame; Push puts a frame here.
	ReadFrames chan []byte
	// WriteErr, when set, is returned by WriteFrame.
	WriteErr error

	closed chan struct{}
	once   sync.Once

	mu     sync.Mutex
	frames [][]byte
}

// New builds an empty bus.
func New() *Bus {
	return &Bus{
		ReadFrames: make(chan []byte, 16),
		closed:     make(chan struct{}),
	}
}

// ReadFrame returns the next pushed frame or net.ErrClosed after Close.
func (b *Bus) ReadFrame(buf []byte) (int, error) {
	select {
	case <-b.closed:
		return 0, net.ErrClosed
	case frame := <-b.ReadFrames:
		return copy(buf, frame), nil
	}
}

// WriteFrame stores a copy of the frame.
func (b *Bus) WriteFrame(frame []byte) error {
	if b.WriteErr != nil {
		return b.WriteErr
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.frames = append(b.frames, append([]byte(nil), frame...))
	return nil
}

// Close unblocks ReadFrame with net.ErrClosed. It is idempotent.
func (b *Bus) Close() {
	b.once.Do(func() { close(b.closed) })
}

// Written returns a copy of the frames written to the bus.
func (b *Bus) Written() [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([][]byte, len(b.frames))
	for i, frame := range b.frames {
		out[i] = append([]byte(nil), frame...)
	}
	return out
}

// Push puts a frame on the read side. It blocks when the buffer is full.
func (b *Bus) Push(frame []byte) {
	b.ReadFrames <- append([]byte(nil), frame...)
}

// Frame builds a raw classic CAN frame with the given identifier and
// payload. It panics on an invalid frame: the helper is used in tests only.
func Frame(id uint32, data ...byte) []byte {
	f := cantcp.Frame{ID: id, Type: cantcp.TypeClassic, Data: data}
	raw, err := f.MarshalBinary()
	if err != nil {
		panic("testbus: invalid frame: " + err.Error())
	}
	return raw
}
