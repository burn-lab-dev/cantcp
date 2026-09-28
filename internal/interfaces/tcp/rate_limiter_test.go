package tcp

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	r := newRateLimiter(5)
	if r.limit != 5 {
		t.Fatalf("limit = %d, want 5", r.limit)
	}
	if r.start.IsZero() {
		t.Fatal("the window start must be initialised")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	start := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		when time.Duration
		want bool
	}{
		{name: "first frame", when: 0, want: true},
		{name: "second frame", when: 10 * time.Millisecond, want: true},
		{name: "third frame is over the limit", when: 20 * time.Millisecond, want: false},
		{name: "still over the limit", when: 999 * time.Millisecond, want: false},
		{name: "a new window opens", when: time.Second, want: true},
		{name: "second frame of the new window", when: time.Second + time.Millisecond, want: true},
		{name: "third frame of the new window is over the limit", when: time.Second + 2*time.Millisecond, want: false},
	}
	r := newRateLimiter(2)
	r.start = start
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.allow(start.Add(tt.when)); got != tt.want {
				t.Fatalf("allow(%v) = %v, want %v", tt.when, got, tt.want)
			}
		})
	}
}
