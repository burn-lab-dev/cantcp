package http

import (
	"fmt"
	"io"
	nethttp "net/http"
	"strings"
	"time"
)

// metricsContentType is the Prometheus text exposition format version 0.0.4.
const metricsContentType = "text/plain; version=0.0.4; charset=utf-8"

// handleMetrics renders the statistics snapshot in the Prometheus text
// exposition format.
func (s *Server) handleMetrics(w nethttp.ResponseWriter, _ *nethttp.Request) {
	snap := s.stats.Snapshot(time.Now())
	var b strings.Builder
	metricInfo(&b, "cantcp_build_info", "Build information; the version label carries the release.", snap.Version)
	metricGauge(&b, "cantcp_uptime_seconds", "Seconds since the daemon started.", uint64(snap.UptimeSeconds))
	metricCounter(&b, "cantcp_can_frames_total",
		`CAN frames exchanged with the bus; direction="in" is read from the bus, direction="out" is written to it.`,
		`{direction="in"}`, snap.CAN.FramesRead)
	metricCounter(&b, "cantcp_can_frames_total",
		`CAN frames exchanged with the bus; direction="in" is read from the bus, direction="out" is written to it.`,
		`{direction="out"}`, snap.CAN.FramesWritten)
	metricCounter(&b, "cantcp_can_errors_total",
		"Failed CAN operations; direction is read or write.",
		`{direction="read"}`, snap.CAN.ReadErrors)
	metricCounter(&b, "cantcp_can_errors_total",
		"Failed CAN operations; direction is read or write.",
		`{direction="write"}`, snap.CAN.WriteErrors)
	metricGauge(&b, "cantcp_tcp_connections_current", "Clients connected right now.", snap.TCP.ConnectionsCurrent)
	metricCounter(&b, "cantcp_tcp_connections_total", "Connections accepted since the daemon started.", "", snap.TCP.ConnectionsTotal)
	metricCounter(&b, "cantcp_tcp_frames_total",
		"Frames exchanged with the clients; direction=\"in\" is received from clients, direction=\"out\" is sent to them.",
		`{direction="in"}`, snap.TCP.FramesIn)
	metricCounter(&b, "cantcp_tcp_frames_total",
		"Frames exchanged with the clients; direction=\"in\" is received from clients, direction=\"out\" is sent to them.",
		`{direction="out"}`, snap.TCP.FramesOut)
	metricCounter(&b, "cantcp_tcp_bytes_total",
		"TCP bytes exchanged with the clients; direction is in or out.",
		`{direction="in"}`, snap.TCP.BytesIn)
	metricCounter(&b, "cantcp_tcp_bytes_total",
		"TCP bytes exchanged with the clients; direction is in or out.",
		`{direction="out"}`, snap.TCP.BytesOut)
	metricCounter(&b, "cantcp_dropped_frames_total",
		"Frames dropped because a client queue was full or the client exceeded the rate limit.",
		"", snap.TCP.Dropped)
	w.Header().Set("Content-Type", metricsContentType)
	_, _ = io.WriteString(w, b.String())
}

// metricCounter writes one counter metric with the standard HELP and TYPE
// preamble.
func metricCounter(b *strings.Builder, name, help, labels string, value uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s counter\n%s%s %d\n", name, help, name, name, labels, value)
}

// metricGauge writes one gauge metric with the standard HELP and TYPE
// preamble.
func metricGauge(b *strings.Builder, name, help string, value uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, value)
}

// metricInfo writes one info metric: a gauge with a version label set to 1.
func metricInfo(b *strings.Builder, name, help, version string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s gauge\n%s{version=%q} 1\n", name, help, name, name, version)
}
