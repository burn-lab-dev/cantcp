//go:build linux

// Package socketcan implements a raw SocketCAN bus on top of the Linux AF_CAN
// socket API using only the standard library. The bus serves exactly one CAN
// interface; run one bus per interface.
package socketcan

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Linux CAN socket constants from linux/can.h and linux/can/raw.h.
const (
	canRAW          = 1          // CAN_RAW
	solCANRaw       = 101        // SOL_CAN_RAW
	canRawErrFilter = 2          // CAN_RAW_ERR_FILTER
	canRawFDFrames  = 5          // CAN_RAW_FD_FRAMES
	canErrMask      = 0x1FFFFFFF // CAN_ERR_MASK
)

// pollWait is the timeout of one select(2) wait: shutdown and readiness are
// noticed with this granularity.
const pollWait = 100 * time.Millisecond

// Bus is an open SocketCAN raw socket bound to one interface.
type Bus struct {
	iface  string
	fd     int
	closed atomic.Bool
}

// sockaddrCAN is struct sockaddr_can from linux/can.h. The field offsets
// follow the C layout: can_family (2 bytes), padding (2), can_ifindex (4),
// rx_id (4), tx_id (4) — 16 bytes in total.
type sockaddrCAN struct {
	Family  uint16
	_       uint16
	Ifindex uint32
	RxID    uint32
	TxID    uint32
}

// Open creates a raw CAN socket bound to the interface. errorFrames enables
// delivery of CAN error frames.
func Open(iface string, errorFrames bool) (*Bus, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("socketcan: %w", err)
	}
	fd, err := syscall.Socket(syscall.AF_CAN, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, canRAW)
	if err != nil {
		return nil, fmt.Errorf("socketcan: open %s: %w", iface, err)
	}
	if errorFrames {
		if err := syscall.SetsockoptInt(fd, solCANRaw, canRawErrFilter, canErrMask); err != nil {
			_ = syscall.Close(fd)
			return nil, fmt.Errorf("socketcan: enable error frames on %s: %w", iface, err)
		}
	}
	// A raw socket rejects 72-byte CAN FD frames with EINVAL until
	// CAN_RAW_FD_FRAMES is set. Classic frames keep working on both classic
	// and FD interfaces.
	if err := syscall.SetsockoptInt(fd, solCANRaw, canRawFDFrames, 1); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("socketcan: enable CAN FD on %s: %w", iface, err)
	}
	if err := bindCAN(fd, ifi.Index); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("socketcan: bind %s: %w", iface, err)
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("socketcan: set non-blocking on %s: %w", iface, err)
	}
	return &Bus{iface: iface, fd: fd}, nil
}

// bindCAN binds the socket to the interface index via bind(2). The
// syscall.Sockaddr interface cannot be implemented outside the syscall
// package, so the call is issued directly with the raw sockaddr.
func bindCAN(fd, ifindex int) error {
	sa := sockaddrCAN{Family: syscall.AF_CAN, Ifindex: uint32(ifindex)}
	_, _, errno := syscall.Syscall(syscall.SYS_BIND, uintptr(fd), uintptr(unsafe.Pointer(&sa)), unsafe.Sizeof(sa))
	if errno != 0 {
		return errno
	}
	return nil
}

// Interface returns the interface name.
func (b *Bus) Interface() string { return b.iface }

// ReadFrame reads one raw CAN frame (16 or 72 bytes) into buf and returns its
// length. It blocks until a frame arrives; the wait is interrupted every
// pollWait to notice Close.
func (b *Bus) ReadFrame(buf []byte) (int, error) {
	for {
		if b.closed.Load() {
			return 0, net.ErrClosed
		}
		n, err := syscall.Read(b.fd, buf)
		switch {
		case err == nil:
			return n, nil
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK):
			if err := b.wait(true); err != nil {
				return 0, err
			}
		default:
			if b.closed.Load() {
				return 0, net.ErrClosed
			}
			return 0, fmt.Errorf("socketcan: read %s: %w", b.iface, err)
		}
	}
}

// WriteFrame writes one raw CAN frame (16 or 72 bytes).
func (b *Bus) WriteFrame(frame []byte) error {
	for {
		if b.closed.Load() {
			return net.ErrClosed
		}
		n, err := syscall.Write(b.fd, frame)
		switch {
		case err == nil && n == len(frame):
			return nil
		case err == nil:
			return fmt.Errorf("socketcan: write %s: short write %d/%d", b.iface, n, len(frame))
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK):
			if err := b.wait(false); err != nil {
				return err
			}
		default:
			if b.closed.Load() {
				return net.ErrClosed
			}
			return fmt.Errorf("socketcan: write %s: %w", b.iface, err)
		}
	}
}

// Close closes the socket. Concurrent ReadFrame and WriteFrame calls return
// net.ErrClosed.
func (b *Bus) Close() error {
	if b.closed.Swap(true) {
		return nil
	}
	return syscall.Close(b.fd)
}

// wait blocks until the socket becomes readable (readable is true) or
// writable, or pollWait expires. It returns net.ErrClosed when the bus is
// closed while waiting.
func (b *Bus) wait(readable bool) error {
	for {
		if b.closed.Load() {
			return net.ErrClosed
		}
		var fds syscall.FdSet
		if err := fdSet(&fds, b.fd); err != nil {
			return err
		}
		tv := syscall.NsecToTimeval(pollWait.Nanoseconds())
		var (
			n   int
			err error
		)
		if readable {
			n, err = syscall.Select(b.fd+1, &fds, nil, nil, &tv)
		} else {
			n, err = syscall.Select(b.fd+1, nil, &fds, nil, &tv)
		}
		switch {
		case errors.Is(err, syscall.EINTR):
			continue
		case err != nil:
			if b.closed.Load() {
				return net.ErrClosed
			}
			return fmt.Errorf("socketcan: select %s: %w", b.iface, err)
		case n > 0:
			return nil
		}
	}
}

// fdSet sets the file descriptor bit in the set. The element width is taken
// from the type so the code works on both 32-bit and 64-bit platforms.
func fdSet(set *syscall.FdSet, fd int) error {
	bits := int(unsafe.Sizeof(set.Bits[0]) * 8)
	if fd < 0 || fd >= len(set.Bits)*bits {
		return fmt.Errorf("socketcan: file descriptor %d does not fit the select set", fd)
	}
	set.Bits[fd/bits] |= 1 << (uint(fd) % uint(bits))
	return nil
}
