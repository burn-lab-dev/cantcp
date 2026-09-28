# vcan smoke report — 2026-09-28

**English** | [Русский](2026-09-28-vcan-smoke.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## Environment

- host: developer workstation, Linux (Ubuntu 24.04), kernel with the `vcan`
  module;
- bus: `vcan0`, MTU 72 (CAN FD enabled);
- versions: `cantcp v0.1.0`, `cantcp-lib-go v0.1.1`, `cantcp-lib-python
  0.1.1`, Go 1.24/1.25/1.26, Python 3.10–3.14 in CI;
- commands: `go test -race ./...`, `scripts/vcan-smoke.sh --report DIR`, plus
  the manual load and interop checks below.

## Automated smoke (8/8)

| Status | Check | Detail |
|---|---|---|
| PASS | `matrix` | 15 frames of the matrix echoed to the client and seen on the bus |
| PASS | `python-interop` | the Python client received frames from the Go client and itself |
| PASS | `rate-limit` | 100 frames in one connection, 90 dropped by the 10 fps limit |
| PASS | `idle-timeout` | a silent client was disconnected in 1s |
| PASS | `max-connections` | the second client was rejected and closed |
| PASS | `tls-mtls` | the mutual-TLS exchange delivered the frame |
| PASS | `tls-version` | TLS 1.2 was rejected with a protocol alert |
| PASS | `sighup` | the certificate was rotated without a restart |

The matrix covers classic frames (0 and 8 bytes), CAN FD with every valid
data length (0, 1, 8, 12, 16, 20, 24, 32, 48, 64), the extended identifier,
the remote frame and the BRS/ESI flags; the data patterns exercise the
`C3 3C` magic inside the payload.

## Manual checks

| Check | Result |
|---|---|
| 5000 numbered frames at a moderate rate (Python client → daemon → CLI listener) | 5000/5000, unique, complete, in order, no duplicates |
| 200000-frame burst via `cangen -g 0` with a slow client (`client_queue=1`) | 98832 frames dropped for that client, the daemon stayed alive and kept serving |
| Burst without a slow client (`cangen -g 0`, 200000 frames) | an independent `candump` lost frames as well: the loss is a `vcan`/SocketCAN property at extreme bursts, not a gateway fault; moderate rates are lossless |
| Three listeners (two CLI, one Python) + two sources (CLI, Python) at once | every listener received every frame, including the echo of its own |
| Reload under a live session (`SIGHUP`, rotated certificate) | the live session kept working, new connections used the new certificate |

## Findings of this run (all fixed before the release)

1. The gateway did not see (and therefore did not broadcast) frames it wrote
   itself — fixed by enabling `CAN_RAW_RECV_OWN_MSGS` (PR #8).
2. `CAN_RAW_FD_FRAMES` was missing: CAN FD writes failed with `EINVAL`
   (fixed before, PR #5 of the library).
3. The codec rejected the kernel's `CANFD_FDF` marker — fixed in
   cantcp-lib-go v0.1.1 and cantcp-lib-python 0.1.1.
4. The APT repository served only `Packages.gz` and apt did not pick the
   index up; the package lacked `Depends: adduser` — both fixed, verified
   with `apt install` in a clean container.
5. TX error frames are not delivered by `vcan`: the error-frame path must be
   tested on hardware (planned for 2026-09-29).

## Known limitations

- Extreme bursts (`cangen -g 0`) overrun `vcan` buffers; a real CAN bus at
  125k–1M cannot produce such rates, and moderate rates are lossless.
- Hardware-specific checks (bitrates, terminator/error counters, bus-off,
  FD data phase) run on 2026-09-29 with two physical adapters and will be
  appended here.
