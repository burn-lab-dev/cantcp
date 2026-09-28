package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// netPipe returns a connected pair with a deadline for the pipe tests.
func netPipe(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	deadline := time.Now().Add(5 * time.Second)
	_ = server.SetDeadline(deadline)
	_ = client.SetDeadline(deadline)
	return server, client
}

// rawFrame builds a raw classic frame for the pipe tests.
func rawFrame(id uint32, data ...byte) []byte {
	f := cantcp.Frame{ID: id, Type: cantcp.TypeClassic, Data: data}
	raw, err := f.MarshalBinary()
	if err != nil {
		panic("rawFrame: " + err.Error())
	}
	return raw
}

// runListenPipe writes the frames into a pipe and runs listenFrames on the other
// side.
func runListenPipe(t *testing.T, frames [][]byte, filterID, filterMask uint32, asJSON bool, count int) string {
	t.Helper()
	server, client := netPipe(t)
	defer server.Close()
	defer client.Close()
	go func() {
		enc := cantcp.NewEncoder(server)
		for _, frame := range frames {
			if err := enc.Encode(frame); err != nil {
				return
			}
		}
		_ = server.Close()
	}()
	var buf bytes.Buffer
	if err := listenFrames(client, filterID, filterMask, asJSON, "can0", count, &buf); err != nil {
		t.Fatalf("listenFrames() = %v, want nil", err)
	}
	return buf.String()
}

func TestListenFrames_Candump(t *testing.T) {
	out := runListenPipe(t, [][]byte{rawFrame(0x123, 0x11), rawFrame(0x456, 0x22)}, 0, 0, false, 0)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "can0 123#11") {
		t.Fatalf("line 0 = %q, want the 0x123 frame", lines[0])
	}
	if !strings.Contains(lines[1], "can0 456#22") {
		t.Fatalf("line 1 = %q, want the 0x456 frame", lines[1])
	}
}

func TestListenFrames_Filter(t *testing.T) {
	out := runListenPipe(t, [][]byte{rawFrame(0x123, 0x11), rawFrame(0x456, 0x22)}, 0x456, 0x7FF, false, 0)
	if strings.Contains(out, "123#") {
		t.Fatalf("output = %q, the filtered frame must not be printed", out)
	}
	if !strings.Contains(out, "456#") {
		t.Fatalf("output = %q, want the 0x456 frame", out)
	}
}

func TestListenFrames_Count(t *testing.T) {
	out := runListenPipe(t, [][]byte{rawFrame(0x123, 0x11), rawFrame(0x456, 0x22)}, 0, 0, false, 1)
	if lines := strings.Count(strings.TrimSpace(out), "\n"); lines != 0 {
		t.Fatalf("output = %q, want exactly one line", out)
	}
	if !strings.Contains(out, "123#") {
		t.Fatalf("output = %q, want the first frame", out)
	}
}

func TestListenFrames_JSON(t *testing.T) {
	out := runListenPipe(t, [][]byte{rawFrame(0x123, 0x11)}, 0, 0, true, 0)
	var obj jsonFrame
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &obj); err != nil {
		t.Fatalf("Unmarshal(%q) = %v", out, err)
	}
	if obj.ID != 0x123 || obj.Data != "11" || obj.FD {
		t.Fatalf("object = %+v, want the 0x123 classic frame", obj)
	}
}

func TestListenFrames_DecodeError(t *testing.T) {
	server, client := netPipe(t)
	defer server.Close()
	defer client.Close()
	go func() {
		// A packet cut off in the middle: the decoder reports ErrTruncated.
		_, _ = server.Write([]byte{0xC3, 0x3C, 0x01, 0x00})
		_ = server.Close()
	}()
	var buf bytes.Buffer
	err := listenFrames(client, 0, 0, false, "can0", 0, &buf)
	if err == nil {
		t.Fatal("listenFrames() = nil, want a decode error")
	}
	if !strings.Contains(err.Error(), "decode:") {
		t.Fatalf("listenFrames() = %q, want a decode error", err)
	}
}

func TestSendFrames(t *testing.T) {
	server, client := netPipe(t)
	defer server.Close()
	defer client.Close()
	frames := []cantcp.Frame{
		{ID: 0x123, Type: cantcp.TypeClassic, Data: []byte{1}},
		{ID: 0x456, Type: cantcp.TypeClassic, Data: []byte{2}},
	}

	var out bytes.Buffer
	sendErr := make(chan error, 1)
	go func() { sendErr <- sendFrames(client, frames, 2, 0, &out) }()

	dec := cantcp.NewDecoder(server)
	got := make([]cantcp.Frame, 0, 4)
	for len(got) < 4 {
		frame, err := dec.DecodeFrame()
		if err != nil {
			t.Fatalf("DecodeFrame() = %v", err)
		}
		got = append(got, frame)
	}
	if err := <-sendErr; err != nil {
		t.Fatalf("sendFrames() = %v, want nil", err)
	}
	for i, want := range []uint32{0x123, 0x456, 0x123, 0x456} {
		if got[i].ID != want {
			t.Fatalf("frame %d id = %03X, want %03X", i, got[i].ID, want)
		}
	}
	if !strings.Contains(out.String(), "sent 4 frame(s)") {
		t.Fatalf("output = %q, want the sent counter", out.String())
	}
}

func TestSendFrames_Interval(t *testing.T) {
	server, client := netPipe(t)
	defer server.Close()
	defer client.Close()
	frames := []cantcp.Frame{{ID: 0x1, Type: cantcp.TypeClassic, Data: []byte{}}}

	var out bytes.Buffer
	sendErr := make(chan error, 1)
	start := time.Now()
	go func() { sendErr <- sendFrames(client, frames, 3, 20*time.Millisecond, &out) }()

	dec := cantcp.NewDecoder(server)
	for i := 0; i < 3; i++ {
		if _, err := dec.DecodeFrame(); err != nil {
			t.Fatalf("DecodeFrame() = %v", err)
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatalf("sendFrames() = %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("sending took %v, want at least two intervals (40ms)", elapsed)
	}
}

func TestReadFrameFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frames.txt")
	content := "# a comment\n\n123#1122\n(1696000000.123456) can0 7FF#R\n1ABCDE##1DE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	frames, err := readFrameFile(path)
	if err != nil {
		t.Fatalf("readFrameFile() = %v, want nil", err)
	}
	if len(frames) != 3 {
		t.Fatalf("frames = %d, want 3", len(frames))
	}
	if frames[0].ID != 0x123 || frames[0].Type != cantcp.TypeClassic || !bytes.Equal(frames[0].Data, []byte{0x11, 0x22}) {
		t.Fatalf("frame 0 = %+v, want 123#1122", frames[0])
	}
	if frames[1].ID != 0x7FF || !frames[1].RTR {
		t.Fatalf("frame 1 = %+v, want 7FF#R", frames[1])
	}
	if frames[2].ID != 0x1ABCDE || frames[2].Type != cantcp.TypeFd || !frames[2].EFF {
		t.Fatalf("frame 2 = %+v, want the extended CAN FD frame", frames[2])
	}
}

func TestReadFrameFile_Errors(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.txt")
	if err := os.WriteFile(broken, []byte("123#11\nnot a frame\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "missing file",
			path:    filepath.Join(dir, "missing.txt"),
			wantErr: "no such file or directory",
		},
		{
			name:    "broken line with the line number",
			path:    broken,
			wantErr: "broken.txt:2:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readFrameFile(tt.path)
			if err == nil {
				t.Fatalf("readFrameFile(%s) = nil, want an error", tt.path)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("readFrameFile(%s) = %q, want a substring %q", tt.path, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestPrintStatsTable(t *testing.T) {
	snap := domain.Stats{
		Version:       "v0.1.0",
		UptimeSeconds: 90,
		CAN:           domain.StatsCAN{Interface: "can0", FramesRead: 2, FramesWritten: 1},
		TCP:           domain.StatsTCP{ConnectionsCurrent: 1, ConnectionsTotal: 2, FramesIn: 3, FramesOut: 4, BytesIn: 5, BytesOut: 6, Dropped: 7},
		Clients: []domain.StatsClient{
			{Socket: "10.0.0.1:5", Since: testTime, FramesIn: 3, FramesOut: 4, BytesIn: 5, BytesOut: 6, Dropped: 7},
		},
	}
	var buf bytes.Buffer
	printStatsTable(&buf, &snap)
	out := buf.String()
	for _, want := range []string{
		"version:  v0.1.0",
		"uptime:   1m30s",
		"CAN can0: frames read 2, written 1",
		"TCP:      clients 1",
		"SOCKET",
		"10.0.0.1:5",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("table does not contain %q:\n%s", want, out)
		}
	}

	empty := domain.Stats{Version: "dev"}
	buf.Reset()
	printStatsTable(&buf, &empty)
	if strings.Contains(buf.String(), "SOCKET") {
		t.Fatalf("table with no clients must not print the header:\n%s", buf.String())
	}
}
