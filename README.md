# cantcp

[![CI](https://github.com/burn-lab-dev/cantcp/actions/workflows/ci.yml/badge.svg)](https://github.com/burn-lab-dev/cantcp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/burn-lab-dev/cantcp/actions/workflows/codeql.yml/badge.svg)](https://github.com/burn-lab-dev/cantcp/actions/workflows/codeql.yml)
[![Go version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**English** | [Русский](README.ru.md)

The gateway pair for the **cantcp** protocol (Linux SocketCAN over TCP):

- **`cantcpd`** — the daemon: bridges one SocketCAN interface and TCP clients,
  in plain mode (trusted perimeter) or over TLS 1.3;
- **`cantcp-cli`** — the client: listens to CAN frames, sends them, reads
  server statistics.

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

> **Status: v0.** The protocol (`v0`) and the command line may change before
> v1.0.0. Frames are carried by the companion libraries:
> [cantcp-lib-go](https://github.com/burn-lab-dev/cantcp-lib-go) and
> [cantcp-lib-python](https://github.com/burn-lab-dev/cantcp-lib-python).

## Features

- one static binary per component, no cgo, no runtime dependencies;
- plain TCP or TLS 1.3 with optional mutual TLS; the two modes are exclusive;
- safe defaults: plain mode listens on the loopback interface only, exposing
  it to the network requires `--allow-plain`;
- settings from the JSON file, the `CANTCP_*` environment variables and the
  command line flags, every value validated on startup;
- per-client queues with a drop counter: a slow client never blocks the bus;
- connection, timeout and frame rate limits;
- HTTP statistics (`/api/v1/stats`, `/metrics`, `/healthz`);
- SIGHUP reload of the TLS material, the log level and the limits;
- `.deb` packages for amd64, arm64 and armhf with a hardened systemd unit,
  plus an APT repository on GitHub Pages.

## Installation

From the APT repository (amd64, arm64, armhf):

```sh
curl -fsSL https://burn-lab-dev.github.io/cantcp/cantcp.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/cantcp.gpg
echo "deb [signed-by=/usr/share/keyrings/cantcp.gpg] https://burn-lab-dev.github.io/cantcp stable main" \
  | sudo tee /etc/apt/sources.list.d/cantcp.list
sudo apt update
sudo apt install cantcpd cantcp-cli
```

From a `.deb` file (download it from the releases page):

```sh
sudo apt install ./cantcpd_0.1.0_amd64.deb ./cantcp-cli_0.1.0_amd64.deb
```

With the Go toolchain (Go 1.24 or newer):

```sh
go install github.com/burn-lab-dev/cantcp/cmd/cantcp-cli@latest
```

## Quick start (plain mode)

```sh
# On the gateway machine: a real can0 interface, or a virtual one for tests.
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set vcan0 mtu 72      # optional: enables CAN FD frames
sudo ip link set up vcan0

# Start the daemon on the loopback interface.
cantcpd --can vcan0

# In another terminal: watch the frames and send one.
cantcp-cli listen
cantcp-cli send --id 123 --data 11223344

# Read the server statistics.
cantcp-cli stats
```

Minimal end-to-end check without a CAN device — the CLI talks to the daemon
over TCP, the daemon talks to `vcan0`; use `candump vcan0`, `cansend` or a
second `cantcp-cli` to observe the frames on the bus.

## TLS mode

```sh
cantcpd --listen 0.0.0.0:29536 \
  --tls-cert /etc/cantcp/tls/server.pem \
  --tls-key  /etc/cantcp/tls/server-key.pem \
  --tls-ca   /etc/cantcp/tls/ca.pem \
  --tls-client-auth require_and_verify

cantcp-cli listen --server can-gateway.example:29536 \
  --tls --tls-ca /etc/cantcp/tls/ca.pem \
  --tls-cert /etc/cantcp/tls/client.pem \
  --tls-key  /etc/cantcp/tls/client-key.pem
```

Key and certificate generation: [docs/TLS-KEYS.md](docs/TLS-KEYS.md).
TLS configuration in depth: [docs/TLS.md](docs/TLS.md).

## Cross-compilation

The binaries are pure Go (no cgo), so every Go target works. Releases cover:

| Target | GOOS | GOARCH | GOARM | Debian architecture |
|---|---|---|---|---|
| x86-64 | linux | amd64 | — | `amd64` |
| ARM64 | linux | arm64 | — | `arm64` |
| ARMv7 | linux | arm | 7 | `armhf` |

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build ./cmd/cantcpd
```

Any other target works the same way (`mipsle`, `riscv64`, ...). See
[docs/DEPLOY.md](docs/DEPLOY.md).

## Documentation

| Document | Content |
|---|---|
| [docs/PROTOCOL.md](docs/PROTOCOL.md) | The cantcp stream and its libraries |
| [docs/SERVER.md](docs/SERVER.md) | Daemon behaviour, modes, limits, SIGHUP |
| [docs/CLI.md](docs/CLI.md) | Commands, filters, frame syntax |
| [docs/CONFIG.md](docs/CONFIG.md) | JSON configuration, sources, validation |
| [docs/STATS.md](docs/STATS.md) | Statistics API and Prometheus metrics |
| [docs/TLS.md](docs/TLS.md) | TLS 1.3 and mutual TLS |
| [docs/TLS-KEYS.md](docs/TLS-KEYS.md) | Generating keys with OpenSSL |
| [docs/DEPLOY.md](docs/DEPLOY.md) | Packages, systemd, APT, cross-builds |
| [SECURITY.md](SECURITY.md) | Threat model and deployment checklist |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Development and pull requests |

## Building from source

```sh
make build    # bin/cantcpd, bin/cantcp-cli for the host
make test     # go test -race ./...
make cross    # linux/amd64, linux/arm64, linux/armv7
make deb      # .deb packages into dist/ (requires dpkg-deb)
```

## License

MIT — see [LICENSE](LICENSE).
