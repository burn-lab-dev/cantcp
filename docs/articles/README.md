# Article drafts

**English** | [Русский](README.ru.md)

> **Status (2026-09-28):** the plan is approved; the drafts are written on
> the next session (2026-09-29). The priority topics are 1 (one config, three
> sources) and 3 (gateway security).

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

Working notes for a series of articles built on this project: each draft
starts from a decision that was actually made in the code, with links to the
files. The drafts live here first; the published versions appear on Habr and
link back to the repository.

## Series outline

1. **One config, three sources, one validator.**
   Flags, `CANTCP_*` environment variables and a strict JSON file with a
   documented precedence; why the validation lives in the domain
   (`internal/app/domain/config_*.go`) and what the error messages look
   like. Code: `internal/config/`, `internal/app/domain/`.

2. **Hexagonal architecture in a small Go binary.**
   Layers, ports and adapters in a project with no frameworks: domain →
   services → interfaces → adapters, manual DI in `main`, table-driven
   tests. Code: the whole `internal/` tree.

3. **Security of a CAN gateway.**
   Why CRC-8 is not integrity, what TLS 1.3 and mTLS add, why plain mode
   listens on the loopback, how `--allow-plain` and the threat model table
   were designed. Code: `internal/adapters/tlsconfig/`,
   `internal/interfaces/tcp/`, `SECURITY.md`.

4. **Packaging Go binaries: .deb without helpers.**
   `CGO_ENABLED=0`, cross-compilation for amd64/arm64/armhf, a POSIX shell
   script around `dpkg-deb`, a hardened systemd unit, and an APT repository
   from `dpkg-scanpackages` + `apt-ftparchive` + GPG on GitHub Pages.
   Code: `scripts/`, `deb/`, `.github/workflows/`.

5. **vcan in CI: end-to-end tests without hardware.**
   How the socketcan adapter, the bridge, the TCP listener and the client
   meet on a virtual bus in a GitHub runner, and why the tests skip
   themselves without `vcan0` locally. Code: `internal/e2e/`.

## Conventions

- a draft is a separate file named after the topic;
- every claim about behaviour links to the repository file that implements
  it;
- the Russian version of a published article lives on Habr; the repository
  keeps the outline and the code references.

The session-by-session decisions behind these topics are recorded in the
project journal of the BURN-LAB workspace (`reflections/`), which the
articles are distilled from.
