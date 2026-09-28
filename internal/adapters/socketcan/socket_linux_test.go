//go:build linux

package socketcan

import (
	"bytes"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// testInterface returns the CAN interface to test with: vcan0 when it
// exists. The tests are skipped otherwise, so the suite runs without root
// and without kernel modules.
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

// canFDInterface returns the interface for the CAN FD test. A virtual bus is
// created with MTU 16 by default; CAN FD needs MTU 72:
//
//	sudo ip link set vcan0 mtu 72
func canFDInterface(t *testing.T) string {
	t.Helper()
	iface := testInterface(t)
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		t.Skipf("%s: %v", iface, err)
	}
	if ifi.MTU < 72 {
		t.Skipf("%s MTU is %d: CAN FD needs MTU 72 (ip link set %s mtu 72)", iface, ifi.MTU, iface)
	}
	return iface
}

// canFDF is the CANFD_FDF bit the Linux kernel may set in the flags byte of
// a delivered CAN FD frame; a raw pass-through must keep it and the tests
// must not treat it as corruption.
const canFDF = 0x04

// rawClassic builds a raw classic frame.
func rawClassic(t *testing.T, id uint32, data ...byte) []byte {
	t.Helper()
	f := cantcp.Frame{ID: id, Type: cantcp.TypeClassic, Data: data}
	raw, err := f.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() = %v", err)
	}
	return raw
}

// readWithTimeout reads one frame with a deadline.
func readWithTimeout(t *testing.T, bus *Bus) []byte {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, maxFrameLen)
		n, err := bus.ReadFrame(buf)
		if err != nil {
			ch <- result{err: err}
			return
		}
		ch <- result{data: buf[:n]}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("ReadFrame() = %v", res.err)
		}
		return res.data
	case <-time.After(3 * time.Second):
		t.Fatal("no frame received within the deadline")
		return nil
	}
}

func TestOpen_RoundTripClassic(t *testing.T) {
	iface := testInterface(t)
	reader, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open(reader) = %v", err)
	}
	defer reader.Close()
	writer, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open(writer) = %v", err)
	}
	defer writer.Close()
	if reader.Interface() != iface {
		t.Fatalf("Interface() = %q, want %q", reader.Interface(), iface)
	}

	// The writer and the reader are separate sockets: the test does not
	// depend on the vcan loopback option.
	want := rawClassic(t, 0x123, 0x11, 0x22, 0x33)
	if err := writer.WriteFrame(want); err != nil {
		t.Fatalf("WriteFrame() = %v", err)
	}
	got := readWithTimeout(t, reader)
	if !bytes.Equal(got, want) {
		t.Fatalf("frame = %x, want %x", got, want)
	}
}

func TestOpen_RoundTripFD(t *testing.T) {
	iface := canFDInterface(t)
	reader, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open(reader) = %v", err)
	}
	defer reader.Close()
	writer, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open(writer) = %v", err)
	}
	defer writer.Close()

	f := cantcp.Frame{ID: 0x1ABCDE, Type: cantcp.TypeFd, EFF: true, BRS: true, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	want, err := f.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() = %v", err)
	}
	if err := writer.WriteFrame(want); err != nil {
		t.Fatalf("WriteFrame() = %v", err)
	}
	got := readWithTimeout(t, reader)
	// Mask the kernel's CANFD_FDF bit before the comparison: the adapter
	// passes the frame through as it came from the bus.
	if len(got) == len(want) {
		got = bytes.Clone(got)
		got[5] &^= canFDF
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("frame = %x, want %x", got, want)
	}
}

func TestBus_SeesOwnFrames(t *testing.T) {
	// CAN_RAW_RECV_OWN_MSGS: a frame written by the bus is readable through
	// the same bus, so the gateway can broadcast traffic it wrote itself.
	iface := testInterface(t)
	bus, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	defer bus.Close()

	want := rawClassic(t, 0x321, 0x0A, 0x0B)
	if err := bus.WriteFrame(want); err != nil {
		t.Fatalf("WriteFrame() = %v", err)
	}
	got := readWithTimeout(t, bus)
	if !bytes.Equal(got, want) {
		t.Fatalf("frame = %x, want %x", got, want)
	}
}

func TestOpen_MissingInterface(t *testing.T) {
	if _, err := Open("cantcp-does-not-exist", false); err == nil {
		t.Fatal("Open() = nil, want an error")
	}
}

func TestOpen_ErrorFrames(t *testing.T) {
	iface := testInterface(t)
	bus, err := Open(iface, true)
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	defer bus.Close()
}

func TestBus_CloseUnblocksRead(t *testing.T) {
	iface := testInterface(t)
	bus, err := Open(iface, false)
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := bus.ReadFrame(make([]byte, maxFrameLen))
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if err := bus.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("ReadFrame() = %v, want net.ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not unblock ReadFrame()")
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("second Close() = %v, want nil", err)
	}
	if err := bus.WriteFrame(rawClassic(t, 0x1)); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("WriteFrame() after Close = %v, want net.ErrClosed", err)
	}
	if _, err := bus.ReadFrame(make([]byte, maxFrameLen)); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("ReadFrame() after Close = %v, want net.ErrClosed", err)
	}
}

func TestFDSet(t *testing.T) {
	var set syscall.FdSet
	if err := fdSet(&set, 0); err != nil {
		t.Fatalf("fdSet(0) = %v", err)
	}
	if err := fdSet(&set, 3); err != nil {
		t.Fatalf("fdSet(3) = %v", err)
	}
	if err := fdSet(&set, -1); err == nil {
		t.Fatal("fdSet(-1) = nil, want an error")
	}
	if err := fdSet(&set, len(set.Bits)*64); err == nil {
		t.Fatal("fdSet(out of range) = nil, want an error")
	}
}
