# TLS keys with OpenSSL

**English** | [Русский](TLS-KEYS.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

This document is the practical companion to [TLS.md](TLS.md): it generates a
private CA, a server certificate and client certificates with the `openssl`
command line tool. The canonical reference for the commands is the official
OpenSSL project:

- repository: <https://github.com/openssl/openssl>;
- documentation: <https://docs.openssl.org/> (`openssl-req`, `openssl-ca`,
  `openssl-x509`, `openssl-verify` manual pages).

The examples create an ECDSA P-256 chain, valid for the hosts `localhost` and
`127.0.0.1`; add your real names and addresses to the SAN list.

## 1. Private CA

```sh
mkdir -p /etc/cantcp/tls && cd /etc/cantcp/tls

# CA key and self-signed certificate (10 years).
openssl ecparam -name prime256v1 -genkey -noout -out ca-key.pem
chmod 600 ca-key.pem
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem \
  -subj "/CN=cantcp local CA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"
```

## 2. Server certificate

```sh
# Server key and CSR.
openssl ecparam -name prime256v1 -genkey -noout -out server-key.pem
chmod 600 server-key.pem
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=can-gateway.example"

# Sign with the CA; the SAN list is what the client verifies.
cat > server-ext.cnf <<'EOF'
subjectAltName = DNS:can-gateway.example, DNS:localhost, IP:127.0.0.1
extendedKeyUsage = serverAuth
keyUsage = digitalSignature
EOF
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 825 -out server.pem -extfile server-ext.cnf
rm -f server.csr
```

Run the daemon with `--tls-cert server.pem --tls-key server-key.pem`. A
client that connects to an IP address must have that IP in the SAN; adjust
the SAN list and re-issue instead of using `--tls-insecure`.

## 3. Client certificate (mutual TLS)

```sh
openssl ecparam -name prime256v1 -genkey -noout -out client-key.pem
chmod 600 client-key.pem
openssl req -new -key client-key.pem -out client.csr \
  -subj "/CN=cantcp-client-1"
cat > client-ext.cnf <<'EOF'
extendedKeyUsage = clientAuth
keyUsage = digitalSignature
EOF
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 825 -out client.pem -extfile client-ext.cnf
rm -f client.csr
```

Run the daemon with `--tls-ca ca.pem --tls-client-auth require_and_verify`
and the client with `--tls --tls-ca ca.pem --tls-cert client.pem
--tls-key client-key.pem`.

## 4. Verification

```sh
# The certificate chains to the CA and the key matches.
openssl verify -CAfile ca.pem server.pem client.pem
openssl pkey -in server-key.pem -pubout | diff - <(openssl x509 -in server.pem -pubkey -noout)

# What a real client will check (host name included).
openssl s_client -connect can-gateway.example:29536 -tls1_3 \
  -CAfile ca.pem -servername can-gateway.example </dev/null

# The handshake version and the peer subject.
openssl s_client -connect can-gateway.example:29536 -tls1_3 -CAfile ca.pem </dev/null 2>/dev/null \
  | grep -E "Protocol|subject="
```

## 5. Using the files

- daemon: `--tls-cert server.pem --tls-key server-key.pem` (plus `--tls-ca`
  and `--tls-client-auth` for mTLS); the paths can live in the JSON
  configuration file instead of the command line;
- CLI: `--tls --tls-ca ca.pem [--tls-cert client.pem --tls-key client-key.pem]`;
- **Go programs** with `cantcp-lib-go`: load the pair with
  `tls.LoadX509KeyPair` and the CA with `x509.NewCertPool`; a complete
  server and client example lives in the library repository
  (`examples/tls/`);
- **Python programs** with `cantcp-lib-python`: pass the paths to
  `ssl.SSLContext.load_cert_chain` and `load_verify_locations`; a complete
  example lives in the library repository (`examples/tls/`).

## 6. Rotation

Replace the server files and reload the daemon (`systemctl reload cantcpd`,
see [TLS.md](TLS.md)). The CA and the client certificates are read at client
start: distribute the new client material and restart the clients. Keep the
CA key offline: it is only needed when issuing certificates.
