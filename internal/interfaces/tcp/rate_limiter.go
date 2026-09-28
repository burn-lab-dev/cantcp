package tcp

import "time"

// rateLimiter is a fixed one-second window frame limiter per client.
type rateLimiter struct {
	limit   int
	start   time.Time
	counter int
}

// newRateLimiter builds a limiter that allows limit frames per second.
func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{limit: limit, start: time.Now()}
}

// allow reports whether one more frame fits the current window. The limiter
// is used from the connection goroutine only.
func (r *rateLimiter) allow(now time.Time) bool {
	if now.Sub(r.start) >= time.Second {
		r.start = now
		r.counter = 0
	}
	if r.counter >= r.limit {
		return false
	}
	r.counter++
	return true
}
