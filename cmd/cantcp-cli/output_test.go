package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// testTime is the fixed receive timestamp used by the output tests.
var testTime = time.Date(2026, 9, 28, 12, 0, 0, 123456000, time.UTC)

// outputPrefix is the candump line prefix for testTime.
func outputPrefix() string {
	return fmt.Sprintf("(%d.%06d) can0 ", testTime.Unix(), testTime.Nanosecond()/1000)
}

func TestFormatCandump(t *testing.T) {
	tests := []struct {
		name  string
		frame cantcp.Frame
		want  string
	}{
		{
			name:  "classic",
			frame: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{0x11, 0x22}},
			want:  "123#1122",
		},
		{
			name:  "classic without data",
			frame: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{}},
			want:  "123#",
		},
		{
			name:  "extended",
			frame: cantcp.Frame{ID: 0x1ABCDE, Type: cantcp.TypeClassic, EFF: true, Data: []byte{1}},
			want:  "001ABCDE#01",
		},
		{
			name:  "remote",
			frame: cantcp.Frame{ID: 0x7FF, Type: cantcp.TypeClassic, RTR: true, Data: []byte{}},
			want:  "7FF#R",
		},
		{
			name:  "CAN FD with BRS",
			frame: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, BRS: true, Data: []byte{0xDE, 0xAD}},
			want:  "123##1DEAD",
		},
		{
			name:  "CAN FD with BRS and ESI",
			frame: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, BRS: true, ESI: true, Data: []byte{0xDE}},
			want:  "123##3DE",
		},
		{
			name:  "error frame",
			frame: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, ERR: true, Data: []byte{0, 0, 0, 4}},
			want:  "123#E00000004",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			frame := tt.frame
			formatCandump(&buf, "can0", &frame, testTime)
			want := outputPrefix() + tt.want
			if got := buf.String(); got != want {
				t.Fatalf("formatCandump() = %q, want %q", got, want)
			}
		})
	}
}

func TestWriteFrameJSON(t *testing.T) {
	frame := cantcp.Frame{ID: 0x1ABCDE, Type: cantcp.TypeFd, EFF: true, BRS: true, Data: []byte{0xDE, 0xAD}}
	var buf bytes.Buffer
	if err := writeFrameJSON(&buf, &frame, testTime); err != nil {
		t.Fatalf("writeFrameJSON() = %v, want nil", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		t.Fatalf("output = %q, want a trailing newline", buf.String())
	}
	var obj jsonFrame
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}
	if obj.ID != 0x1ABCDE || !obj.Extended || !obj.FD || !obj.BRS || obj.Remote || obj.Error {
		t.Fatalf("object = %+v, want an extended CAN FD frame with BRS", obj)
	}
	if obj.Time != testTime {
		t.Fatalf("Time = %v, want %v", obj.Time, testTime)
	}
	if obj.Data != hex.EncodeToString(frame.Data) {
		t.Fatalf("Data = %q, want %q", obj.Data, hex.EncodeToString(frame.Data))
	}
}
