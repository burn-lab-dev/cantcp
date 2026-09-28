package main

import (
	"bytes"
	"errors"
	"testing"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

func TestParseFrameLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    cantcp.Frame
		wantErr string
	}{
		{
			name: "classic frame",
			line: "123#11223344",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{0x11, 0x22, 0x33, 0x44}},
		},
		{
			name: "classic frame without data",
			line: "7FF#",
			want: cantcp.Frame{ID: 0x7FF, Type: cantcp.TypeClassic, Data: []byte{}},
		},
		{
			name: "extended identifier by value",
			line: "1ABCDE#01",
			want: cantcp.Frame{ID: 0x1ABCDE, Type: cantcp.TypeClassic, EFF: true, Data: []byte{0x01}},
		},
		{
			name: "extended identifier with eight digits",
			line: "00000123#0A",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, EFF: true, Data: []byte{0x0A}},
		},
		{
			name: "remote frame",
			line: "123#R",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, RTR: true, Data: []byte{}},
		},
		{
			name: "remote frame lowercase",
			line: "123#r",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, RTR: true, Data: []byte{}},
		},
		{
			name: "CAN FD with BRS",
			line: "123##1DEADBEEF",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, BRS: true, Data: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		},
		{
			name: "CAN FD with BRS and ESI",
			line: "123##3DEADBEEF",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, BRS: true, ESI: true, Data: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		},
		{
			name: "CAN FD without flags",
			line: "123##012",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, Data: []byte{0x12}},
		},
		{
			name: "candump log format",
			line: "(1696000000.123456) can0 123#1122",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{0x11, 0x22}},
		},
		{
			name: "surrounding spaces",
			line: "  123#11  ",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{0x11}},
		},
		{
			name:    "comment line",
			line:    "# a comment",
			wantErr: errCommentLine.Error(),
		},
		{
			name:    "empty line",
			line:    "   ",
			wantErr: errCommentLine.Error(),
		},
		{
			name:    "missing separator",
			line:    "123 1122",
			wantErr: `frame "123 1122": missing '#': want id#data`,
		},
		{
			name:    "invalid identifier",
			line:    "XY#11",
			wantErr: `frame "XY#11": invalid identifier "XY"`,
		},
		{
			name:    "identifier too large for the frame",
			line:    "FFFFFFFF#11",
			wantErr: `frame "FFFFFFFF#11": `,
		},
		{
			name:    "odd payload",
			line:    "123#112",
			wantErr: `frame "123#112": invalid payload: `,
		},
		{
			name:    "CAN FD without flags digit",
			line:    "123##",
			wantErr: `frame "123##": missing CAN FD flags digit`,
		},
		{
			name:    "CAN FD invalid flags",
			line:    "123##Z12",
			wantErr: `frame "123##Z12": invalid CAN FD flags "Z"`,
		},
		{
			name:    "wrong double hash",
			line:    "123#x#12",
			wantErr: `frame "123#x#12": invalid separator: want id##flags`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFrameLine(tt.line)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseFrameLine(%q) = nil, want %q", tt.line, tt.wantErr)
				}
				if !errors.Is(err, errCommentLine) && tt.wantErr == errCommentLine.Error() {
					t.Fatalf("parseFrameLine() = %v, want errCommentLine", err)
				}
				if tt.wantErr != errCommentLine.Error() && err.Error()[:len(tt.wantErr)] != tt.wantErr {
					t.Fatalf("parseFrameLine() = %q, want a prefix %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFrameLine(%q) = %v, want nil", tt.line, err)
			}
			if got.ID != tt.want.ID || got.Type != tt.want.Type || got.EFF != tt.want.EFF ||
				got.RTR != tt.want.RTR || got.BRS != tt.want.BRS || got.ESI != tt.want.ESI {
				t.Fatalf("frame = %+v, want %+v", got, tt.want)
			}
			if !bytes.Equal(got.Data, tt.want.Data) {
				t.Fatalf("data = %x, want %x", got.Data, tt.want.Data)
			}
			if raw := got.GetRaw(); len(raw) != 16 && len(raw) != 72 {
				t.Fatalf("GetRaw() length = %d, want 16 or 72", len(raw))
			}
		})
	}
}

func TestBuildFrame(t *testing.T) {
	tests := []struct {
		name       string
		id, data   string
		ext, fd    bool
		brs        bool
		want       cantcp.Frame
		wantErrSub string
	}{
		{
			name: "classic",
			id:   "123", data: "1122",
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{0x11, 0x22}},
		},
		{
			name: "forced extended",
			id:   "123", data: "", ext: true,
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeClassic, EFF: true, Data: []byte{}},
		},
		{
			name: "extended by value",
			id:   "1ABCDE", data: "00",
			want: cantcp.Frame{ID: 0x1ABCDE, Type: cantcp.TypeClassic, EFF: true, Data: []byte{0x00}},
		},
		{
			name: "CAN FD with BRS",
			id:   "123", data: "0102030405060708090A0B0C", fd: true, brs: true,
			want: cantcp.Frame{ID: 0x123, Type: cantcp.TypeFd, BRS: true, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		},
		{
			name: "invalid identifier",
			id:   "ZZ", data: "",
			wantErrSub: `invalid identifier "ZZ"`,
		},
		{
			name: "invalid payload",
			id:   "123", data: "0",
			wantErrSub: `invalid payload "0"`,
		},
		{
			name: "identifier too large",
			id:   "FFFFFFFF", data: "",
			wantErrSub: `frame "FFFFFFFF#": `,
		},
		{
			name: "classic payload too long",
			id:   "123", data: "010203040506070809",
			wantErrSub: `frame "123#010203040506070809": `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFrame(tt.id, tt.data, tt.ext, tt.fd, tt.brs)
			if tt.wantErrSub != "" {
				if err == nil {
					t.Fatalf("buildFrame() = nil, want an error containing %q", tt.wantErrSub)
				}
				if !bytes.Contains([]byte(err.Error()), []byte(tt.wantErrSub)) {
					t.Fatalf("buildFrame() = %q, want a substring %q", err.Error(), tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildFrame() = %v, want nil", err)
			}
			if got.ID != tt.want.ID || got.Type != tt.want.Type || got.EFF != tt.want.EFF || got.BRS != tt.want.BRS {
				t.Fatalf("frame = %+v, want %+v", got, tt.want)
			}
			if !bytes.Equal(got.Data, tt.want.Data) {
				t.Fatalf("data = %x, want %x", got.Data, tt.want.Data)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    uint32
		wantErr string
	}{
		{name: "empty", value: "", want: 0},
		{name: "hex", value: "1ABC", want: 0x1ABC},
		{name: "zero", value: "0", want: 0},
		{
			name:    "invalid",
			value:   "xyz",
			wantErr: `id: invalid hexadecimal identifier "xyz"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseID(tt.value, "id")
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseID(%q) = %v, want %q", tt.value, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseID(%q) = %v, want nil", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("parseID(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchFilter(t *testing.T) {
	tests := []struct {
		name    string
		id      uint32
		frameID uint32
		mask    uint32
		want    bool
	}{
		{name: "zero mask matches everything", id: 0, mask: 0, frameID: 0x123, want: true},
		{name: "exact match", id: 0x123, mask: 0x7FF, frameID: 0x123, want: true},
		{name: "exact mismatch", id: 0x123, mask: 0x7FF, frameID: 0x124, want: false},
		{name: "masked match", id: 0x100, mask: 0x700, frameID: 0x1FF, want: true},
		{name: "masked mismatch", id: 0x100, mask: 0x700, frameID: 0x200, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &cantcp.Frame{ID: tt.frameID, Type: cantcp.TypeClassic}
			if got := matchFilter(frame, tt.id, tt.mask); got != tt.want {
				t.Fatalf("matchFilter(%03X, %03X, %03X) = %v, want %v", tt.frameID, tt.id, tt.mask, got, tt.want)
			}
		})
	}
}
