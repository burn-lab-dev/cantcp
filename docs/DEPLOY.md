# Deployment

**English** | [Русский](DEPLOY.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## Packages

Releases ship two `.deb` packages per architecture:

| Package | Contents |
|---|---|
| `cantcpd` | `/usr/bin/cantcpd`, `/etc/cantcp/cantcpd.json` (conffile), the systemd unit, the man page, configuration examples |
| `cantcp-cli` | `/usr/bin/cantcp-cli`, the man page, configuration examples |

Architectures: `amd64`, `arm64`, `armhf` (ARMv7). Install from a file:

```sh
sudo apt install ./cantcpd_0.1.0_amd64.deb ./cantcp-cli_0.1.0_amd64.deb
```

or from the APT repository:

```sh
curl -fsSL https://burn-lab-dev.github.io/cantcp/cantcp.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/cantcp.gpg
echo "deb [signed-by=/usr/share/keyrings/cantcp.gpg] https://burn-lab-dev.github.io/cantcp stable main" \
  | sudo tee /etc/apt/sources.list.d/cantcp.list
sudo apt update
sudo apt install cantcpd cantcp-cli
```

The repository is rebuilt from the release `.deb` files by the release
workflow and signed with the project APT key; the public key is
`cantcp.gpg`. Its fingerprint is:

```
7321 58E5 361A 8288 7F6E  E1A7 126B 909D ED55 B3AD
```

Check the downloaded key before trusting it:

```sh
curl -fsSL https://burn-lab-dev.github.io/cantcp/cantcp.gpg | gpg --show-keys
```

## systemd

The `cantcpd` package installs `/lib/systemd/system/cantcpd.service`. The
unit:

- runs as the system user `cantcp` with the single capability `CAP_NET_RAW`;
- is sandboxed: `ProtectSystem=strict`, `ProtectHome=yes`, `PrivateTmp=yes`,
  `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX AF_NETLINK AF_CAN`
  (netlink is needed to resolve the interface index),
  `NoNewPrivileges=yes`, `SystemCallFilter=@system-service @network-io`;
- restarts on failure (`Restart=always`, `RestartSec=2s`);
- reloads on `systemctl reload cantcpd` (`ExecReload` sends `SIGHUP`).

```sh
sudo systemctl status cantcpd
sudo journalctl -u cantcpd -f
sudo systemctl reload cantcpd      # after rotating the TLS material
```

The package enables and starts the unit after installation. If `can0` does
not exist yet, the daemon exits with a clear error and systemd restarts it
until the interface appears — bringing a real CAN interface up is an
administrator task (`ip link set can0 up type can bitrate 500000`).

### One bus per instance

A daemon instance serves one CAN interface. For several buses, run several
instances with their own configuration and ports:

```ini
# /etc/systemd/system/cantcpd@.service
[Unit]
Description=cantcp gateway on %i
After=network.target

[Service]
ExecStart=/usr/bin/cantcpd --config /etc/cantcp/%i.json
# ... the hardening options from the packaged unit ...
User=cantcp
Group=cantcp
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/cantcp/can1.json
{"listen": "127.0.0.1:29546", "can": {"interface": "can1"}}
```

```sh
sudo systemctl enable --now cantcpd@can1
```

Each instance has its own statistics port (`stats.listen`), limits and log
stream: a failure on one bus stays on one bus.

## Cross-compilation

The binaries are pure Go, no cgo and no system libraries. Build for any Go
target:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath ./cmd/cantcpd
```

Release targets and their Debian equivalents:

| Target | GOOS | GOARCH | GOARM | Debian |
|---|---|---|---|---|
| x86-64 | linux | amd64 | — | `amd64` |
| ARM64 | linux | arm64 | — | `arm64` |
| ARMv7 | linux | arm | 7 | `armhf` |

`make cross` builds the three targets into `bin/`, `make deb` wraps them
into the `.deb` packages under `dist/`. The version is embedded from the git
tag (`git describe`), so tag the tree before building a release.

## Building the packages yourself

```sh
git tag v0.1.0            # if not tagged yet
make cross                # binaries
make deb VERSION=v0.1.0   # dist/*.deb
```

`scripts/build-deb.sh` is a plain POSIX shell script around `go build` and
`dpkg-deb`; `scripts/apt-repo.sh` turns a directory of `.deb` files into a
signed APT repository (`APT_GPG_KEY` holds the ASCII-armored private key,
`dpkg-scanpackages` and `apt-ftparchive` do the layout).

## USB CAN adapters and virtual buses

```sh
# Real hardware (SocketCAN drivers are in the kernel).
sudo ip link set can0 up type can bitrate 500000

# Virtual bus: tests without a device.
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set vcan0 mtu 72     # CAN FD (classic CAN uses MTU 16)
sudo ip link set up vcan0
```

Quick smoke test after installation:

```sh
cantcpd --can vcan0 &                 # or: sudo systemctl start cantcpd
cantcp-cli send --id 123 --data 11223344
candump vcan0                          # can-utils; the frame is on the bus
cantcp-cli stats
kill %1
```

## Monitoring

- `/healthz` — liveness for systemd or a load balancer;
- `/api/v1/stats` — a full JSON snapshot for scripts;
- `/metrics` — Prometheus: scrape `cantcp_*` metrics and alert on
  `rate(cantcp_dropped_frames_total[5m]) > 0` and
  `increase(cantcp_can_errors_total[1h]) > 0`.
