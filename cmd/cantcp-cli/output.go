package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// formatCandump writes one frame in the candump log format:
//
//	(1696000000.123456) can0 123#11223344
//	(1696000000.123456) can0 12345678#1122
//	(1696000000.123456) can0 123#R
//	(1696000000.123456) can0 123#E00000004
//	(1696000000.123456) can0 123##1DEADBEEF
//
// The flag digit of a CAN FD frame carries BRS in bit 0 and ESI in bit 1.
// The timestamp is the local receive time: the cantcp stream does not carry
// timestamps.
func formatCandump(w io.Writer, iface string, f *cantcp.Frame, ts time.Time) {
	if f.EFF {
		fmt.Fprintf(w, "(%d.%06d) %s %08X", ts.Unix(), ts.Nanosecond()/1000, iface, f.ID)
	} else {
		fmt.Fprintf(w, "(%d.%06d) %s %03X", ts.Unix(), ts.Nanosecond()/1000, iface, f.ID)
	}
	switch {
	case f.ERR:
		fmt.Fprintf(w, "#E%X", f.Data)
	case f.RTR:
		fmt.Fprint(w, "#R")
	case f.Type == cantcp.TypeFd:
		fmt.Fprintf(w, "##%X%X", fdFlags(f), f.Data)
	default:
		fmt.Fprintf(w, "#%X", f.Data)
	}
}

// fdFlags returns the CAN FD flags digit of the candump format.
func fdFlags(f *cantcp.Frame) int {
	var v int
	if f.BRS {
		v |= 1
	}
	if f.ESI {
		v |= 2
	}
	return v
}

// jsonFrame is the JSON form of one received frame.
type jsonFrame struct {
	Time     time.Time `json:"time"`
	ID       uint32    `json:"id"`
	Extended bool      `json:"extended,omitempty"`
	Remote   bool      `json:"remote,omitempty"`
	Error    bool      `json:"error,omitempty"`
	FD       bool      `json:"fd"`
	BRS      bool      `json:"brs,omitempty"`
	ESI      bool      `json:"esi,omitempty"`
	Data     string    `json:"data"`
}

// writeFrameJSON writes one JSON object followed by a newline.
func writeFrameJSON(w io.Writer, f *cantcp.Frame, ts time.Time) error {
	obj := jsonFrame{
		Time:     ts,
		ID:       f.ID,
		Extended: f.EFF,
		Remote:   f.RTR,
		Error:    f.ERR,
		FD:       f.Type == cantcp.TypeFd,
		BRS:      f.BRS,
		ESI:      f.ESI,
		Data:     hex.EncodeToString(f.Data),
	}
	return json.NewEncoder(w).Encode(obj)
}
