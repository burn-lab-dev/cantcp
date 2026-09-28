package tcp

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestCountingConn(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	var read, written atomic.Int64
	conn := &countingConn{
		Conn:    left,
		onRead:  func(n int) { read.Add(int64(n)) },
		onWrite: func(n int) { written.Add(int64(n)) },
	}

	done := make(chan error, 1)
	go func() {
		_, err := right.Write([]byte("12345"))
		done <- err
	}()
	buf := make([]byte, 5)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("ReadFull() = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("peer Write() = %v", err)
	}
	if got := read.Load(); got != 5 {
		t.Fatalf("onRead total = %d, want 5", got)
	}

	go func() {
		buf := make([]byte, 2)
		_, err := io.ReadFull(right, buf)
		done <- err
	}()
	if _, err := conn.Write([]byte("ab")); err != nil {
		t.Fatalf("Write() = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("peer ReadFull() = %v", err)
	}
	if got := written.Load(); got != 2 {
		t.Fatalf("onWrite total = %d, want 2", got)
	}
}
