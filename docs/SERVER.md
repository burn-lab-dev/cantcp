# The cantcpd daemon

**English** | [Русский](SERVER.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## What the daemon does

`cantcpd` bridges one Linux SocketCAN interface and TCP clients:

- frames read from the CAN bus are broadcast to every connected client as
  cantcp packets (see [PROTOCOL.md](PROTOCOL.md));
- cantcp packets received from a client are written to the CAN bus.

Everything else — statistics, health checks, monitoring — is served over a
separate HTTP listener. The frame stream itself stays minimal.

## One interface per instance

One daemon process serves exactly one CAN interface. For several buses run
several instances, each with its own unit and configuration:

```sh
cantcpd --config /etc/cantcp/can0.json --can can0 --listen 127.0.0.1:29536
cantcpd --config /etc/cantcp/can1.json --can can1 --listen 127.0.0.1:29546
```

With systemd use instance units (`cantcpd@can0.service`) or separate units.
Process isolation gives a clean failure domain: a problem on one bus does not
affect the others, and each instance has its own limits and ports.

## Connection modes

The two modes are exclusive; the mode is selected by the TLS configuration:

- **plain** — when no certificate is set. It carries frames unencrypted and
  unauthenticated: trusted perimeter only. Plain mode listens on
  `127.0.0.1:29536` by default; an external address requires the explicit
  `--allow-plain` flag, otherwise the daemon refuses to start:
  `listen: plain mode on "0.0.0.0:29536" is not loopback; set allow_plain or
  configure TLS`;
- **TLS 1.3** — when `tls.cert_file` and `tls.key_file` are set. TLS 1.2 and
  older are rejected. Client certificates are optional, verified or required
  according to `tls.client_auth`. See [TLS.md](TLS.md).

## Flow control and limits

- Each client has a bounded frame queue (`limits.client_queue`, default
  1024). When it is full, the frame is dropped for that client only and the
  `cantcp_dropped_frames_total` counter grows: a slow client never blocks the
  bus or other clients.
- `limits.max_connections` bounds the simultaneous clients; the connection
  over the limit is accepted and closed immediately.
- `limits.idle_timeout` closes a silent client (0 disables it, default).
- `limits.read_timeout` bounds the TLS handshake; `limits.write_timeout`
  bounds one frame write.
- `limits.max_frames_per_second` limits frames flowing from one client to the
  CAN bus; excess frames are dropped and counted (0 disables the limit).

## Signals

- `SIGINT`, `SIGTERM` — graceful shutdown: the listeners stop, connections
  close, the CAN socket is released.
- `SIGHUP` — reload. The daemon re-reads the same configuration sources and
  applies:
  - the TLS certificate, key and CA files (rotation without a restart);
  - the log level;
  - the limits (they apply to new traffic and the next accepts).
  The listen address, the CAN interface, the TLS switch and the statistics
  address are fixed for the lifetime of the process. A reload that fails
  validation keeps the current settings and logs the error.

## Startup checks

The daemon validates the whole configuration before doing anything and then:

1. opens and binds the SocketCAN interface, enabling error frames when
   configured;
2. loads the TLS key pair and CA (when TLS is selected);
3. opens the TCP listener and the HTTP statistics listener;
4. starts the bridge (CAN reader) and the accept loops.

A failure at any step is reported and the process exits with a non-zero code.
There is no silent degradation: an unavailable CAN interface is a startup
error.

## Statistics

The HTTP listener serves `/healthz`, `/api/v1/stats` and `/metrics`; see
[STATS.md](STATS.md). It is enabled by default on `127.0.0.1:29537` and can be
disabled with `--no-stats` or moved with `--stats-listen`.

## Logging

Logs go to stderr via `log/slog`: text (default) or JSON. Frame payloads are
never logged; the debug level adds connection events, the trace level of the
library is not used by the daemon. For journald keep the text format, for
collectors use JSON.

## Deployment

The package installs `/usr/bin/cantcpd`, the configuration
`/etc/cantcp/cantcpd.json`, a hardened systemd unit and the man page. See
[DEPLOY.md](DEPLOY.md).
