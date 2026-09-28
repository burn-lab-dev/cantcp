# The cantcp protocol

**English** | [Русский](PROTOCOL.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

## Frames on the wire

`cantcp` carries raw Linux SocketCAN frames in a TCP byte stream. A packet is:

```
[magic 2 bytes][type 1 byte][can_frame 16 | canfd_frame 72][CRC-8 1 byte]
```

- magic: `0xC3 0x3C` by default;
- type: `0x01` for `struct can_frame` (16 bytes), `0x02` for
  `struct canfd_frame` (72 bytes);
- frame: the byte-for-byte Linux SocketCAN layout (little-endian `can_id`,
  length byte, padding and data);
- CRC-8 (polynomial `0x07` by default) over `magic + type + frame`.

The frames are not parsed by the transport: `cantcpd` reads a raw frame from
the socket and re-sends it as one packet, and vice versa. The parse model
(identifier, flags, payload) is used by `cantcp-cli` for display and input.

## What is not in the protocol

The protocol carries frames only:

- no handshake, no subscriptions, no keepalives: the connection is plain TCP;
- no authentication or encryption: use TLS 1.3 (see
  [TLS.md](TLS.md)) or deploy inside a trusted perimeter;
- no statistics: `cantcpd` serves them over HTTP (see [STATS.md](STATS.md));
- no timestamps: the receiver stamps frames with its local time;
- no flow control: the daemon drops frames for a full client queue and counts
  the drops.

This keeps the transport minimal and the libraries small. Features that need
application logic (statistics, health checks, subscriptions) belong to the
server API, not to the frame stream.

## Implementations

The protocol is implemented byte-for-byte identically by the companion
libraries, which share the test vectors:

- Go: [cantcp-lib-go](https://github.com/burn-lab-dev/cantcp-lib-go) —
  `NewDecoder` / `NewEncoder` wrap `Split` / `Encode` with an `io` API;
- Python: [cantcp-lib-python](https://github.com/burn-lab-dev/cantcp-lib-python)
  — the same API surface, `Parser` / `Decoder` / `Encoder`.

`cantcpd` and `cantcp-cli` use the Go library; a Python service or tool can
talk to the daemon with the Python library.

## Compatibility

- Protocol version: **v0** (the packet layout above). Breaking changes are
  possible before v1.0.0.
- Frame limits: CAN 2.0 (0..8 payload bytes) and CAN FD (0..64 bytes, lengths
  encodable in the 4-bit DLC field).
- Error frames are carried as classic frames with `CAN_ERR_FLAG` in the
  identifier; enable them in the daemon with `can.error_frames`.
- A decoder that sees garbage resynchronises on the next magic: the stream
  self-synchronises, which is a deliberate property of v0.

## Security notes

The CRC-8 is a framing sanity check, not cryptography: about one in 256
random byte sequences passes it. The protocol provides no integrity,
confidentiality or replay protection. Use TLS 1.3 for untrusted networks and
treat the plain mode as a bench/segment tool. See
[SECURITY.md](../SECURITY.md).
