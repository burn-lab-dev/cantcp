package domain

import "time"

// ConfigLimits bounds the daemon resources: connections, per-client queue,
// timeouts and the optional client-to-CAN frame rate limit. ReadTimeout
// bounds the TLS handshake, WriteTimeout one frame write, IdleTimeout the
// silence between two client frames.
type ConfigLimits struct {
	MaxConnections     int
	ClientQueue        int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	MaxFramesPerSecond int
}

// Limits constraints accepted by Validate.
const (
	limitsMinConnections = 1
	limitsMaxConnections = 4096
	limitsMinQueue       = 1
	limitsMaxQueue       = 1 << 20
)

// Validate checks every limit. A zero MaxFramesPerSecond disables the rate
// limit; a zero IdleTimeout disables the idle timeout.
func (c ConfigLimits) Validate() Error {
	if c.MaxConnections < limitsMinConnections || c.MaxConnections > limitsMaxConnections {
		return ErrInvalid("limits.max_connections: %d is out of range [%d, %d]",
			c.MaxConnections, limitsMinConnections, limitsMaxConnections)
	}
	if c.ClientQueue < limitsMinQueue || c.ClientQueue > limitsMaxQueue {
		return ErrInvalid("limits.client_queue: %d is out of range [%d, %d]",
			c.ClientQueue, limitsMinQueue, limitsMaxQueue)
	}
	if c.ReadTimeout <= 0 {
		return ErrInvalid("limits.read_timeout: must be positive")
	}
	if c.WriteTimeout <= 0 {
		return ErrInvalid("limits.write_timeout: must be positive")
	}
	if c.IdleTimeout < 0 {
		return ErrInvalid("limits.idle_timeout: must not be negative")
	}
	if c.MaxFramesPerSecond < 0 {
		return ErrInvalid("limits.max_frames_per_second: must not be negative")
	}
	return nil
}
