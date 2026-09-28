# Security policy and threat model

**English** | [Русский](SECURITY.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## Reporting a vulnerability

Use GitHub security advisories (the "Report a vulnerability" button in the
Security tab) or e-mail <stepanov.mikhail.y@gmail.com>. Please do not open a
public issue for a vulnerability before it is fixed. We answer within a few
days and credit reporters in the release notes unless asked otherwise.

## What cantcp protects and what it does not

`cantcp` is a CAN transport. The plain stream gives neither confidentiality
nor integrity nor authentication; the CRC-8 is a framing check, not
cryptography (about one in 256 random byte sequences passes it). The daemon
therefore provides two deployment models:

### Trusted perimeter (plain)

- The daemon listens on `127.0.0.1` by default. Exposing it to a network
  requires the explicit `--allow-plain` flag.
- Anyone who can reach the port can read all bus traffic, inject frames and
  (within the rate limits) flood the bus. The only access control is the
  network: firewall, VLAN, VPN, physical separation.
- Suitable for: a bench, a car, an isolated industrial segment with no other
  tenants.

### TLS 1.3 (optionally mutual)

- TLS 1.3 only, no downgrade path; both directions are encrypted and
  integrity-protected.
- With `--tls-client-auth require_and_verify` both ends authenticate:
  only holders of a client certificate signed by the configured CA connect.
- Suitable for: any network that is not fully trusted, remote access over
  the internet, shared infrastructure.

## Threat model

| Threat | Plain | TLS | mTLS |
|---|---|---|---|
| Passive eavesdropping (bus telemetry) | exposed | protected | protected |
| Frame injection by an on-path attacker | possible | impossible without session keys | impossible without a client certificate |
| Server impersonation | — | client verifies the server certificate | same |
| Client impersonation | — | client is not authenticated | prevented by the certificate |
| Replay of captured frames | possible | possible inside a session; a new session key is derived per connection | same |
| DoS by connection flood | bounded by `max_connections` and timeouts | same | same |
| DoS by frame flood | bounded by `max_frames_per_second` and the queues | same | same |
| DoS by slow readers | per-client queues drop frames, the bus is not blocked | same | same |
| DoS by slow handshakes | `read_timeout` | `read_timeout` | `read_timeout` |
| CPU amplification in the stream parser | bounded by the frame rate limit; see cantcp-lib-go/SECURITY.md | same | same |
| Leakage through logs | frame payloads are never logged | same | same |

Replay deserves a note: TLS 1.3 does not add application-level replay
protection, and the cantcp protocol has no sequence numbers or nonces. An
attacker with the session keys, or a party already inside an authenticated
session, can replay frames. For buses where a replayed command is dangerous,
add application-level protection (sequence numbers, freshness checks) or
restrict physical/logical access to the bus.

## Hardening checklist

- [ ] Plain mode: keep the listener on the loopback or an isolated segment;
      firewall the port; never publish it.
- [ ] TLS: use `--tls-client-auth require_and_verify` for production; keep
      the CA key offline.
- [ ] Files: certificate keys `chmod 600`, owned by root or `cantcp`;
      `/etc/cantcp` readable by root and `cantcp` only.
- [ ] systemd: keep the packaged hardening options (the unit runs with
      `CAP_NET_RAW` only, `ProtectSystem=strict`, `SystemCallFilter`).
- [ ] Limits: bound `max_connections`, `client_queue` and, when untrusted
      clients exist, `max_frames_per_second`; set `idle_timeout` where
      appropriate.
- [ ] Statistics: keep `stats.listen` on the loopback or inside the
      monitoring network; it has no authentication.
- [ ] Logs: do not enable the trace level of the cantcp library in
      production; the daemon never uses it, and the library warns about it in
      its own SECURITY.md.
- [ ] Updates: watch the releases and security advisories; the CI runs
      govulncheck and CodeQL on every change.

## Known limitations

- No per-frame authentication: TLS protects the channel, not the bus. A
  compromised client host can send anything its CAN access allows.
- No revocation inside a running session: rotate certificates and reload
  (server) or restart clients.
- The statistics endpoint is unauthenticated by design; deploy it
  accordingly.
- `--tls-insecure` and `--allow-plain` are deliberate escape hatches: they
  exist for benches and are dangerous in production.
