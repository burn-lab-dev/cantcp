# Testing

**English** | [Русский](TESTING.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

Two environments, one matrix:

- **virtual bus (vcan)** — runs everywhere, including CI, no hardware;
- **hardware bus** — two CAN adapters on one twisted pair, run manually.

## Unit tests

```sh
make test    # go test -race ./...
make fmt vet
make cross   # the release targets must build
make deb VERSION=v0.0.0
```

The socketcan and end-to-end tests skip themselves when the virtual interface
is absent, so `make test` stays green on a machine without root.

## Virtual bus

```sh
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set vcan0 mtu 72      # CAN FD; without MTU 72 FD writes fail
sudo ip link set up vcan0

scripts/vcan-smoke.sh              # full smoke, ~15 s
scripts/vcan-smoke.sh --report /tmp/cantcp-report
```

The smoke script builds the binaries, runs the daemon on loopback ports and
checks:

| Check | What it proves |
|---|---|
| `matrix` | classic/FD/EFF/RTR frames, every CAN FD data length (0..64), echoed to the client and seen by a `candump` observer |
| `python-interop` | a client of `cantcp-lib-python` (examples/client.py) exchanges frames with the Go `cantcp-cli` through the same daemon |
| `rate-limit` | 100 frames in one connection with a 10 fps limit: the excess is dropped and counted |
| `idle-timeout` | a silent client is disconnected |
| `max-connections` | the connection over the limit is closed |
| `tls-mtls` | a mutual-TLS exchange delivers frames |
| `tls-version` | TLS 1.2 is rejected |
| `sighup` | the certificate is rotated without a restart |

The script exits non-zero when a check fails and writes `report.json`,
`report.md` and `report.html` with `--report`.

More manual checks:

```sh
# Full-duplex load and integrity (5000 numbered frames, moderate rate)
scripts/vcan-smoke.sh --keep && ls /tmp   # artifacts of the run

# Burst check: vcan itself may drop frames at extreme bursts (g=0);
# an independent candump loses frames too — it is a bus property, not a
# gateway bug. Moderate rates (what a real CAN bus does) must be lossless.
cangen vcan0 -g 1 -I 321 -n 5000
```

## Hardware bus

Wire two adapters to the same bus with 120 Ohm terminators, bring them up:

```sh
sudo ip link set can0 up type can bitrate 500000
sudo ip link set can1 up type can bitrate 500000
scripts/hw-smoke.sh --a can0 --b can1 --bitrate 500000 --report /tmp/hw-report
```

`--a` is served by `cantcpd`, `--b` is the observer/load adapter. The script
repeats the matrix and a moderate load check on the real bus; classic frames
only by default (`--fd` adds the FD set and needs MTU-72 adapters). Use
`--bin /usr/bin` to test the installed packages instead of `bin/`, and
`--listen`/`--stats` to avoid a running service on the default ports. Repeat
at 125k, 250k, 500k and 1M; for CAN FD-capable adapters also with FD data
phase (500k/1M/2M/4M).

Hardware results are committed to [docs/test-reports](docs/test-reports/) and
mirrored to the GitHub Pages site.

## What cannot be tested virtually

- TX error frames: `vcan` does not deliver them (found in the first smoke
  run); error-frame handling is a hardware test;
- bus-off and error counters: a real controller is needed (disconnect a
  terminator or short the pair on a bench);
- real bitrates and FD timings.

## CI

- `test` — gofmt, vet, `go test -race` on Go 1.24/1.25/1.26;
- `e2e` — loads vcan, creates `vcan0` with MTU 72, runs `go test -race ./...`
  and `scripts/vcan-smoke.sh`, uploads the report as an artifact and into the
  job summary;
- `canon` — the codec vectors must match `cantcp-spec`;
- `cross` — the release builds and the `.deb` packages;
- `vuln`, CodeQL, fuzz on schedule.

## Adding a check

Add the scenario to `scripts/vcan-smoke.sh` as a `record PASS|FAIL` block,
keep it deterministic (bounded timeouts, no absolute timings) and, when it
needs hardware, put it into `scripts/hw-smoke.sh` and this document instead.
