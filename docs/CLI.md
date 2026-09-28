# cantcp-cli

**English** | [Русский](CLI.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

`cantcp-cli` connects to a cantcp server (see [SERVER.md](SERVER.md)),
listens to CAN frames, sends frames and reads the server statistics.

## Commands

```
cantcp-cli <command> [flags]

  listen    connect and print the CAN frames received from the server
  send      send frames built from flags or read from a candump-style file
  stats     fetch the server statistics over HTTP
  version   print the version
  help      print the top-level help
```

Run `cantcp-cli <command> -h` for the command flags.

## Connection flags (all commands)

| Flag | Default | Meaning |
|---|---|---|
| `--config` | `/etc/cantcp/cantcp-cli.json` | JSON configuration file |
| `--server` | `127.0.0.1:29536` | cantcp server address |
| `--stats-server` | `http://127.0.0.1:29537` | statistics URL (the `stats` command) |
| `--tls` | off | connect over TLS 1.3 |
| `--tls-ca` | — | CA file (default: system roots) |
| `--tls-cert`, `--tls-key` | — | client certificate for mutual TLS |
| `--tls-server-name` | — | verified name (default: host from `--server`) |
| `--tls-insecure` | off | skip verification, **debugging only** |
| `--log-level`, `--log-format` | `info`, `text` | logging |

## listen

```sh
# Print every frame in the candump format.
cantcp-cli listen

# Only 0x7A0..0x7AF, JSON lines, stop after 100 frames.
cantcp-cli listen --id 7A0 --mask 7F0 --json --count 100

# Stop after 30 seconds.
cantcp-cli listen --timeout 30s
```

| Flag | Meaning |
|---|---|
| `--id`, `--mask` | hexadecimal identifier filter: a frame passes when `frame.id & mask == id & mask`; a zero mask accepts everything |
| `--json` | one JSON object per frame |
| `--count` | stop after n frames (0: unlimited) |
| `--timeout` | stop after this time (0: unlimited) |
| `--can-name` | interface name used in the candump output (default `can0`) |

The candump output is:

```
(1696000000.123456) can0 123#11223344
(1696000000.123456) can0 12345678#1122
(1696000000.123456) can0 123#R
(1696000000.123456) can0 123#E00000004
(1696000000.123456) can0 123##1DEADBEEF
```

The timestamp is the local receive time; the flag digit of a CAN FD frame
carries BRS in bit 0 and ESI in bit 1. The JSON form is:

```json
{"time":"2026-09-28T12:00:00.123456Z","id":291,"fd":false,"data":"11223344"}
```

The frames are flushed per frame: pipes and log collectors see them
immediately. Exit code 0 means a clean end (stream closed, count or timeout
reached).

## send

```sh
# One classic frame.
cantcp-cli send --id 123 --data 11223344

# Extended identifier, 100 times with 10 ms between the sends.
cantcp-cli send --id 1ABCDE --data 01 --count 100 --interval 10ms

# One CAN FD frame with the bit rate switch flag.
cantcp-cli send --id 123 --data 0102030405060708090A0B0C --fd --brs

# Every frame from a candump-style file, or from stdin.
cantcp-cli send --input frames.txt
cantcp-cli send - < frames.txt
```

| Flag | Meaning |
|---|---|
| `--id`, `--data` | identifier and payload, hexadecimal |
| `--ext` | force the extended (29-bit) identifier |
| `--fd`, `--brs` | CAN FD frame and bit rate switch |
| `--count`, `--interval` | repeat the single frame (only with `--id`) |
| `--input` | file with candump-style lines, `-` reads stdin |

Input line syntax (the same as `listen` prints):

```
123#11223344          classic frame
12345678#1122         extended identifier (8 hex digits)
123#R                 remote frame
123##1DEADBEEF        CAN FD, flag digit: bit 0 BRS, bit 1 ESI
(1696000000.1) can0 123#1122     a full candump log line
```

Empty lines and lines starting with `#` are skipped; a broken line stops the
command with the file name and the line number. The frame is validated
(identifier range, CAN FD data length) before it is sent.

## stats

```sh
cantcp-cli stats                 # human-readable table
cantcp-cli stats --json          # raw JSON response
cantcp-cli stats --stats-server http://10.0.0.5:29537
```

The command requests `<stats-server>/api/v1/stats`; the response format is
described in [STATS.md](STATS.md).

## TLS examples

```sh
cantcp-cli listen --server can-gateway.example:29536 \
  --tls --tls-ca /etc/cantcp/tls/ca.pem

cantcp-cli send --id 123 --data 01 \
  --server can-gateway.example:29536 --tls \
  --tls-ca /etc/cantcp/tls/ca.pem \
  --tls-cert /etc/cantcp/tls/client.pem \
  --tls-key  /etc/cantcp/tls/client-key.pem
```

`--tls-insecure` disables certificate verification: it is useful for a bench
with a self-signed certificate and dangerous anywhere else. The client
configuration file is described in [CONFIG.md](CONFIG.md).

## Environment and configuration file

Every shared flag has a `CANTCP_*` environment variable
(`CANTCP_SERVER`, `CANTCP_STATS_SERVER`, `CANTCP_TLS_ENABLE`, ...) and a JSON
file counterpart; the flags win over the environment, the environment over
the file, the file over the defaults. See [CONFIG.md](CONFIG.md).
