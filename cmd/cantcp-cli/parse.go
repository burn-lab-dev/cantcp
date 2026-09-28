package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// errCommentLine marks a comment or an empty input line: the send command
// skips such lines.
var errCommentLine = errors.New("comment line")

// parseFrameLine parses one candump-style line into a frame. Accepted forms:
//
//	123#11223344
//	12345678#1122           (extended identifier)
//	123#R                   (remote frame)
//	123##1DEADBEEF          (CAN FD; the flag digit: bit 0 BRS, bit 1 ESI)
//	(1696000000.123456) can0 123#1122
//
// The extra tokens of the candump log format (timestamp, interface) are
// ignored. Empty lines and lines starting with '#' return errCommentLine.
func parseFrameLine(line string) (cantcp.Frame, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return cantcp.Frame{}, errCommentLine
	}
	token := line
	if fields := strings.Fields(line); len(fields) > 0 {
		token = fields[len(fields)-1]
	}
	parts := strings.SplitN(token, "#", 3)
	if len(parts) < 2 {
		return cantcp.Frame{}, fmt.Errorf("frame %q: missing '#': want id#data", line)
	}
	id, err := strconv.ParseUint(parts[0], 16, 32)
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("frame %q: invalid identifier %q", line, parts[0])
	}
	// The candump log format marks an extended identifier with eight hex
	// digits and a standard one with three.
	eff := len(parts[0]) > 3
	if len(parts) == 2 {
		return classicFrame(id, eff, parts[1], line)
	}
	if parts[1] != "" {
		return cantcp.Frame{}, fmt.Errorf("frame %q: invalid separator: want id##flags", line)
	}
	return fdFrame(id, eff, parts[2], line)
}

// classicFrame builds a classic CAN frame: an empty payload, the "R" remote
// marker or hex data.
func classicFrame(id uint64, eff bool, data, line string) (cantcp.Frame, error) {
	f := cantcp.Frame{ID: uint32(id), Type: cantcp.TypeClassic, EFF: eff}
	switch data {
	case "":
	case "R", "r":
		f.RTR = true
	default:
		payload, err := hex.DecodeString(data)
		if err != nil {
			return cantcp.Frame{}, fmt.Errorf("frame %q: invalid payload: %w", line, err)
		}
		f.Data = payload
	}
	return validateFrame(f, line)
}

// fdFrame builds a CAN FD frame from "flags+data": the first hex digit
// carries the flags (bit 0 BRS, bit 1 ESI), the rest is the payload.
func fdFrame(id uint64, eff bool, rest, line string) (cantcp.Frame, error) {
	if rest == "" {
		return cantcp.Frame{}, fmt.Errorf("frame %q: missing CAN FD flags digit", line)
	}
	flags, err := strconv.ParseUint(rest[:1], 16, 8)
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("frame %q: invalid CAN FD flags %q", line, rest[:1])
	}
	payload, err := hex.DecodeString(rest[1:])
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("frame %q: invalid payload: %w", line, err)
	}
	f := cantcp.Frame{
		ID:   uint32(id),
		Type: cantcp.TypeFd,
		EFF:  eff,
		BRS:  flags&1 != 0,
		ESI:  flags&2 != 0,
		Data: payload,
	}
	return validateFrame(f, line)
}

// buildFrame builds one frame from the send flags.
func buildFrame(idStr, dataStr string, extended, fd, brs bool) (cantcp.Frame, error) {
	id, err := strconv.ParseUint(idStr, 16, 32)
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("invalid identifier %q: want hexadecimal, for example 123 or 1ABCDE", idStr)
	}
	payload, err := hex.DecodeString(dataStr)
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("invalid payload %q: want hexadecimal bytes, for example 11223344: %w", dataStr, err)
	}
	f := cantcp.Frame{
		ID:   uint32(id),
		Type: cantcp.TypeClassic,
		EFF:  extended || id > 0x7FF,
		Data: payload,
	}
	if fd {
		f.Type = cantcp.TypeFd
		f.BRS = brs
	}
	return validateFrame(f, idStr+"#"+dataStr)
}

// validateFrame round-trips the frame through the strict codec so that
// invalid identifiers and data lengths are rejected before sending.
func validateFrame(f cantcp.Frame, line string) (cantcp.Frame, error) {
	raw, err := f.MarshalBinary()
	if err != nil {
		return cantcp.Frame{}, fmt.Errorf("frame %q: %w", line, err)
	}
	var out cantcp.Frame
	if err := out.UnmarshalBinary(raw); err != nil {
		return cantcp.Frame{}, fmt.Errorf("frame %q: %w", line, err)
	}
	return out, nil
}

// parseID parses an optional hexadecimal identifier; an empty string is zero.
func parseID(value, name string) (uint32, error) {
	if value == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid hexadecimal identifier %q", name, value)
	}
	return uint32(v), nil
}

// matchFilter reports whether the frame passes the id/mask filter. A zero
// mask accepts every frame.
func matchFilter(f *cantcp.Frame, id, mask uint32) bool {
	if mask == 0 {
		return true
	}
	return f.ID&mask == id&mask
}
