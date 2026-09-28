# Configuration

**English** | [Русский](CONFIG.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## Sources and precedence

Both binaries build their configuration from four sources; a later source
overrides an earlier one:

1. built-in defaults;
2. the JSON file (`--config`, default `/etc/cantcp/cantcpd.json` for the
   daemon and `/etc/cantcp/cantcp-cli.json` for the client);
3. the `CANTCP_*` environment variables;
4. the command line flags (only the flags actually given on the command line
   count, so `--listen 127.0.0.1:29536` also overrides the file).

A file passed with `--config` must exist; the default path may be absent. The
JSON parser is strict: unknown keys, wrong types and trailing data are
errors — a typo stops the start instead of being silently ignored.

Durations are Go strings: `5s`, `250ms`, `1m30s`.

## Daemon (`cantcpd`)

| JSON key | Flag | Environment | Default | Validation |
|---|---|---|---|---|
| `listen` | `--listen` | `CANTCP_LISTEN` | `127.0.0.1:29536` | `host:port`, port 1..65535 |
| `allow_plain` | `--allow-plain` | `CANTCP_ALLOW_PLAIN` | `false` | plain on a non-loopback address requires it |
| `can.interface` | `--can` | `CANTCP_CAN` | `can0` | non-empty, at most 15 characters |
| `can.error_frames` | `--error-frames` | `CANTCP_ERROR_FRAMES` | `false` | boolean |
| `tls.cert_file` | `--tls-cert` | `CANTCP_TLS_CERT` | — | set together with the key |
| `tls.key_file` | `--tls-key` | `CANTCP_TLS_KEY` | — | set together with the certificate |
| `tls.ca_file` | `--tls-ca` | `CANTCP_TLS_CA` | — | required for client verification |
| `tls.client_auth` | `--tls-client-auth` | `CANTCP_TLS_CLIENT_AUTH` | `none` | `none`, `verify_if_given`, `require_and_verify` |
| `limits.max_connections` | `--max-connections` | `CANTCP_MAX_CONNECTIONS` | `16` | 1..4096 |
| `limits.client_queue` | `--client-queue` | `CANTCP_CLIENT_QUEUE` | `1024` | 1..1048576 |
| `limits.read_timeout` | `--read-timeout` | `CANTCP_READ_TIMEOUT` | `5s` | > 0 |
| `limits.write_timeout` | `--write-timeout` | `CANTCP_WRITE_TIMEOUT` | `5s` | > 0 |
| `limits.idle_timeout` | `--idle-timeout` | `CANTCP_IDLE_TIMEOUT` | `0s` | ≥ 0 |
| `limits.max_frames_per_second` | `--max-frames-per-second` | `CANTCP_MAX_FRAMES_PER_SECOND` | `0` | ≥ 0 |
| `stats.enable` | `--no-stats` | `CANTCP_STATS_ENABLE` | `true` | boolean |
| `stats.listen` | `--stats-listen` | `CANTCP_STATS_LISTEN` | `127.0.0.1:29537` | `host:port` when enabled |
| `log.level` | `--log-level` | `CANTCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `log.format` | `--log-format` | `CANTCP_LOG_FORMAT` | `text` | `text`, `json` |

Example: [../examples/cantcpd-plain.json](../examples/cantcpd-plain.json) and
[../examples/cantcpd-tls.json](../examples/cantcpd-tls.json).

## Client (`cantcp-cli`)

| JSON key | Flag | Environment | Default | Validation |
|---|---|---|---|---|
| `server` | `--server` | `CANTCP_SERVER` | `127.0.0.1:29536` | `host:port`, port 1..65535 |
| `stats_server` | `--stats-server` | `CANTCP_STATS_SERVER` | `http://127.0.0.1:29537` | `http://` or `https://` URL |
| `tls.enable` | `--tls` | `CANTCP_TLS_ENABLE` | `false` | boolean; other TLS keys require it |
| `tls.ca_file` | `--tls-ca` | `CANTCP_TLS_CA` | — | file readable at connect time |
| `tls.cert_file` | `--tls-cert` | `CANTCP_TLS_CERT` | — | set together with the key |
| `tls.key_file` | `--tls-key` | `CANTCP_TLS_KEY` | — | set together with the certificate |
| `tls.server_name` | `--tls-server-name` | `CANTCP_TLS_SERVER_NAME` | — | any DNS name |
| `tls.insecure` | `--tls-insecure` | `CANTCP_TLS_INSECURE` | `false` | debugging only |
| `log.level` | `--log-level` | `CANTCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `log.format` | `--log-format` | `CANTCP_LOG_FORMAT` | `text` | `text`, `json` |

Example: [../examples/cantcp-cli.json](../examples/cantcp-cli.json).

## Validation rules

Every value is validated before the process starts or a reload is applied:

- **addresses**: `host:port` with a port in 1..65535; an empty host means all
  interfaces;
- **plain mode**: a non-loopback listen address without TLS requires
  `allow_plain`, otherwise the configuration is rejected. `localhost`,
  `127.0.0.0/8` and `::1` count as loopback;
- **TLS**: the certificate and the key are set together; `client_auth` other
  than `none` requires the certificate pair and a CA file; the CA file
  without TLS is rejected; TLS is 1.3 only;
- **CAN interface**: non-empty, at most 15 characters, no spaces, `/` or `:`;
  the interface is checked (and must be a CAN interface) when the daemon
  starts;
- **limits**: connections 1..4096, queue 1..1048576, read/write timeouts > 0,
  idle timeout and rate limit ≥ 0;
- **log**: the level and the format are from the known sets.

Error messages name the JSON key (or the flag-derived field) and the accepted
values, for example:

```
can.interface: must not be empty
listen: plain mode on "0.0.0.0:29536" is not loopback; set allow_plain or configure TLS
limits.max_connections: 0 is out of range [1, 4096]
```

## Reload

`SIGHUP` re-reads the same command line arguments, the environment and the
file, validates the result and applies the TLS material, the log level and
the limits. The listen address, the CAN interface, the TLS switch and the
statistics address are fixed for the process lifetime: change them with a
restart.
