# Changelog

All notable changes to this project are documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
this project adheres to [Semantic Versioning](https://semver.org/). Until
v1.0.0 the command line, the configuration and the JSON API may change.

## [Unreleased]

### Fixed

- `cantcpd` enables the SocketCAN loopback (`CAN_RAW_RECV_OWN_MSGS`): frames
  sent by a client are broadcast to every client (including the sender), so
  gateway traffic is visible to all monitors, not only frames from other bus
  nodes. Found in the virtual-bus test run.

## [0.1.0] - 2026-09-28

### Added

- `cantcpd`: SocketCAN-to-TCP gateway with the cantcp frame stream
  (`github.com/burn-lab-dev/cantcp-lib-go`).
- Plain and TLS 1.3 modes, optional mutual TLS (`verify_if_given`,
  `require_and_verify`); plain mode outside loopback requires `--allow-plain`.
- JSON configuration, `CANTCP_*` environment variables and flags with full
  startup validation and the precedence flags > environment > file >
  defaults.
- Per-client frame queues with drop counters, connection/queue/timeout/rate
  limits, graceful shutdown and SIGHUP reload of the TLS material, the log
  level and the limits.
- HTTP statistics: `/healthz`, `/api/v1/stats` (JSON) and `/metrics`
  (Prometheus text format).
- `cantcp-cli`: `listen` (candump and JSON output, id/mask filter), `send`
  (flags and candump-style files) and `stats` commands with the same TLS
  options.
- `.deb` packages for amd64, arm64 and armhf with a hardened systemd unit,
  man pages and configuration examples; an APT repository built on GitHub
  Pages and a release workflow.
- Tests: table-driven unit tests, TLS policy tests (1.3 only, mTLS), an
  end-to-end test over `vcan`, fuzz tests for the candump parser; CI runs
  `gofmt`, `go vet`, `go test -race`, cross builds, `govulncheck` and
  CodeQL.
