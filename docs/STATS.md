# Statistics API

**English** | [Русский](STATS.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

The daemon serves its statistics on a dedicated HTTP listener
(`127.0.0.1:29537` by default, `stats.listen`, disabled with `--no-stats` or
`stats.enable: false`). The listener is plain HTTP: keep it on the loopback
or an internal network, it carries no authentication.

## Endpoints

| Method and path | Content |
|---|---|
| `GET /healthz` | `200` with `{"status":"ok"}` while the process is alive |
| `GET /api/v1/stats` | JSON snapshot (see below) |
| `GET /metrics` | Prometheus text exposition format, version 0.0.4 |

`cantcp-cli stats` wraps `/api/v1/stats` and prints a table or the raw JSON.

## JSON snapshot

```json
{
  "version": "v0.1.0",
  "uptime_seconds": 3600,
  "can": {
    "interface": "can0",
    "frames_read": 120345,
    "frames_written": 42,
    "read_errors": 0,
    "write_errors": 1
  },
  "tcp": {
    "connections_current": 2,
    "connections_total": 17,
    "frames_in": 42,
    "frames_out": 120000,
    "bytes_in": 2100,
    "bytes_out": 6100000,
    "dropped": 345
  },
  "clients": [
    {
      "socket": "10.0.0.5:51234",
      "since": "2026-09-28T11:00:00Z",
      "frames_in": 39,
      "frames_out": 120000,
      "bytes_in": 1950,
      "bytes_out": 6100000,
      "dropped": 345
    }
  ]
}
```

- `can.frames_read` / `frames_written` — frames exchanged with the bus;
- `can.read_errors` / `write_errors` — failed bus operations;
- `tcp.frames_in` / `frames_out` — frames received from clients and sent to
  them (that is, written to the bus and read from it);
- `tcp.bytes_*` — TCP bytes including TLS overhead;
- `tcp.dropped` — frames dropped for full client queues or by the frame rate
  limit; the per-client counters are in `clients[]`;
- `clients[]` is sorted by the socket address; a client disappears from the
  list on disconnect, the aggregate counters keep its totals.

Types: all counters are unsigned 64-bit; `since` is RFC 3339; `uptime_seconds`
is monotonic and never negative.

## Prometheus metrics

```sh
curl -s http://127.0.0.1:29537/metrics
```

```
# HELP cantcp_build_info Build information; the version label carries the release.
# TYPE cantcp_build_info gauge
cantcp_build_info{version="v0.1.0"} 1
# HELP cantcp_uptime_seconds Seconds since the daemon started.
# TYPE cantcp_uptime_seconds gauge
cantcp_uptime_seconds 3600
# HELP cantcp_can_frames_total CAN frames exchanged with the bus; direction="in" is read from the bus, direction="out" is written to it.
# TYPE cantcp_can_frames_total counter
cantcp_can_frames_total{direction="in"} 120345
cantcp_can_frames_total{direction="out"} 42
# HELP cantcp_can_errors_total Failed CAN operations; direction is read or write.
# TYPE cantcp_can_errors_total counter
cantcp_can_errors_total{direction="read"} 0
cantcp_can_errors_total{direction="write"} 1
# HELP cantcp_tcp_connections_current Clients connected right now.
# TYPE cantcp_tcp_connections_current gauge
cantcp_tcp_connections_current 2
# HELP cantcp_tcp_connections_total Connections accepted since the daemon started.
# TYPE cantcp_tcp_connections_total counter
cantcp_tcp_connections_total 17
# HELP cantcp_tcp_frames_total Frames exchanged with the clients; ...
# TYPE cantcp_tcp_frames_total counter
cantcp_tcp_frames_total{direction="in"} 42
cantcp_tcp_frames_total{direction="out"} 120000
# HELP cantcp_tcp_bytes_total TCP bytes exchanged with the clients; ...
# TYPE cantcp_tcp_bytes_total counter
cantcp_tcp_bytes_total{direction="in"} 2100
cantcp_tcp_bytes_total{direction="out"} 6100000
# HELP cantcp_dropped_frames_total Frames dropped because a client queue was full or the client exceeded the rate limit.
# TYPE cantcp_dropped_frames_total counter
cantcp_dropped_frames_total 345
```

## Operating notes

- The statistics listener has its own read, write and header timeouts, so a
  stuck scraper cannot exhaust the daemon.
- The counters are in-memory only: a restart resets them. For history use
  Prometheus or another scraper and `rate()`/`increase()`.
- Watch `cantcp_dropped_frames_total` and `cantcp_can_errors_total`: they are
  the two signals that the daemon is under pressure or the bus is unhealthy.
- If the client list grows with unexpected peers, review `listen`,
  `allow_plain` and the firewall — anyone who can reach the port can connect.
