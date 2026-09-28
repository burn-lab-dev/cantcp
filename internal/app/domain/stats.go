package domain

import "time"

// Stats is a point-in-time snapshot served over HTTP and rendered as
// Prometheus metrics.
type Stats struct {
	Version       string        `json:"version"`
	UptimeSeconds int64         `json:"uptime_seconds"`
	CAN           StatsCAN      `json:"can"`
	TCP           StatsTCP      `json:"tcp"`
	Clients       []StatsClient `json:"clients"`
}

// StatsCAN aggregates the CAN bus counters of the daemon.
type StatsCAN struct {
	Interface     string `json:"interface"`
	FramesRead    uint64 `json:"frames_read"`
	FramesWritten uint64 `json:"frames_written"`
	ReadErrors    uint64 `json:"read_errors"`
	WriteErrors   uint64 `json:"write_errors"`
}

// StatsTCP aggregates the TCP listener counters.
type StatsTCP struct {
	ConnectionsCurrent uint64 `json:"connections_current"`
	ConnectionsTotal   uint64 `json:"connections_total"`
	FramesIn           uint64 `json:"frames_in"`
	FramesOut          uint64 `json:"frames_out"`
	BytesIn            uint64 `json:"bytes_in"`
	BytesOut           uint64 `json:"bytes_out"`
	Dropped            uint64 `json:"dropped"`
}

// StatsClient describes one connected client. Socket is the remote address;
// Since is the moment the client was accepted.
type StatsClient struct {
	Socket    string    `json:"socket"`
	Since     time.Time `json:"since"`
	FramesIn  uint64    `json:"frames_in"`
	FramesOut uint64    `json:"frames_out"`
	BytesIn   uint64    `json:"bytes_in"`
	BytesOut  uint64    `json:"bytes_out"`
	Dropped   uint64    `json:"dropped"`
}
