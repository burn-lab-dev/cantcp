package main

import (
	"testing"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"
)

// FuzzParseFrameLine checks that the candump-style parser never panics and
// that every accepted line produces a frame that passes the strict codec.
func FuzzParseFrameLine(f *testing.F) {
	seeds := []string{
		"123#11223344",
		"7FF#",
		"123#R",
		"1ABCDE#01",
		"123##1DEADBEEF",
		"123##3DEADBEEF",
		"(1696000000.123456) can0 123#1122",
		"# comment",
		"",
		"XYZ#11",
		"123#112",
		"123##",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, line string) {
		frame, err := parseFrameLine(line)
		if err != nil {
			return
		}
		raw, err := frame.MarshalBinary()
		if err != nil {
			t.Fatalf("parseFrameLine(%q) returned a frame that does not marshal: %v", line, err)
		}
		var round cantcp.Frame
		if err := round.UnmarshalBinary(raw); err != nil {
			t.Fatalf("frame from %q does not unmarshal: %v", line, err)
		}
		if round.ID != frame.ID || round.Type != frame.Type || round.EFF != frame.EFF ||
			round.RTR != frame.RTR || round.BRS != frame.BRS || round.ESI != frame.ESI {
			t.Fatalf("round trip of %q changed the frame: %+v -> %+v", line, frame, round)
		}
	})
}
