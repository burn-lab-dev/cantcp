# TLS 1.3

**English** | [Русский](TLS.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

The plain cantcp stream has no encryption or authentication (see
[PROTOCOL.md](PROTOCOL.md)). Whenever the network is not fully trusted, run
the daemon over TLS.

## Version and ciphers

- **TLS 1.3 only.** `MinVersion` and `MaxVersion` are pinned to 1.3 on both
  sides; a TLS 1.2 client is rejected during the handshake. There is no
  option to lower the version.
- The cipher suites are the Go defaults for TLS 1.3 (AES-GCM and ChaCha20-
  Poly1305 with SHA-256/384). They are not exposed as configuration: the
  defaults are strong, and a configuration knob only creates a way to make
  them worse.

## Server modes

The TLS mode is selected by setting the certificate pair:

```sh
cantcpd --listen 0.0.0.0:29536 \
  --tls-cert /etc/cantcp/tls/server.pem \
  --tls-key  /etc/cantcp/tls/server-key.pem
```

`tls.client_auth` controls the client certificates (`tls.client_auth` in the
configuration file):

| Mode | Behaviour |
|---|---|
| `none` (default) | clients are not asked for a certificate; the server certificate still proves the server to the client |
| `verify_if_given` | a presented client certificate is verified against the CA; a client without one is accepted |
| `require_and_verify` | a client certificate signed by the CA is mandatory |

Both verifying modes require `tls.ca_file` — the CA that signed the client
certificates:

```sh
cantcpd --listen 0.0.0.0:29536 \
  --tls-cert /etc/cantcp/tls/server.pem \
  --tls-key  /etc/cantcp/tls/server-key.pem \
  --tls-ca   /etc/cantcp/tls/ca.pem \
  --tls-client-auth require_and_verify
```

`require_and_verify` is the recommended mode for production: it
authenticates both ends. The daemon refuses to start if the certificate, the
key or the CA cannot be read or the key does not match the certificate.

## Client modes

```sh
cantcp-cli listen --server can-gateway.example:29536 \
  --tls --tls-ca /etc/cantcp/tls/ca.pem

cantcp-cli send --id 123 --data 01 \
  --server can-gateway.example:29536 --tls \
  --tls-ca /etc/cantcp/tls/ca.pem \
  --tls-cert /etc/cantcp/tls/client.pem \
  --tls-key  /etc/cantcp/tls/client-key.pem
```

- `--tls` switches the connection to TLS 1.3;
- `--tls-ca` replaces the system trust store (use it for a private CA);
- `--tls-cert` / `--tls-key` add the client certificate for mTLS;
- `--tls-server-name` overrides the name verified against the certificate
  (default: the host part of `--server`);
- `--tls-insecure` disables verification. It exists for benches with
  self-signed certificates; in any other use the connection can be
  intercepted. The client logs a warning when it is used.

The server name must match a subject alternative name (SAN) of the server
certificate; the generation commands in
[TLS-KEYS.md](TLS-KEYS.md) put `localhost` and `127.0.0.1` into the SAN and
show how to add more.

## Certificate rotation

Send `SIGHUP` (or `systemctl reload cantcpd`) after replacing the files: the
daemon re-reads the certificate, the key and the CA and starts using them for
new connections. Established connections keep their session; if the new
material is broken, the current configuration stays in place and the error is
logged:

```sh
sudo systemctl reload cantcpd
journalctl -u cantcpd -n 5
```

The client re-reads its certificate files on every start only; restart the
client to pick up a rotated client certificate.

## Threat model notes

- TLS 1.3 gives confidentiality, integrity and peer authentication (with
  mTLS). It does not add application-level replay protection: an attacker
  who has the session keys or sits inside an authenticated session can
  replay frames. Restrict who can connect and treat the CAN bus itself as
  the security boundary it is.
- The private keys are read by the daemon process; protect the files
  (`chmod 600`, root or the `cantcp` user) and the host.
- Revocation is out of scope: with `require_and_verify`, remove the client
  certificate from the CA and restart the daemon, or use a CA with short
  certificate lifetimes.
- Keep `--tls-insecure` out of production configurations and code comments;
  the option is intentionally loud in the documentation.
