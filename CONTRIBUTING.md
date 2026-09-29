# Contributing

**English** | [Русский](CONTRIBUTING.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

Thanks for taking the time to contribute. Small, focused changes are the
easiest to review; for a larger feature open an issue first so we can agree
on the shape.

## Requirements

- Go 1.24 or newer; the CI also runs 1.26.
- `gofmt`, `go vet`, `go test -race` must pass.
- Only the standard library: the project has no third-party dependencies and
  we intend to keep it that way.
- Conventional Commits for the commit messages (`feat:`, `fix:`, `docs:`,
  `test:`, `chore:`, ...).

## Development loop

```sh
make build     # bin/cantcpd, bin/cantcp-cli
make test      # go test -race ./...
make vet
make fmt       # gofmt check
make cross     # linux/amd64, linux/arm64, linux/armv7
make deb       # .deb packages (needs dpkg-deb)
```

The end-to-end tests need a virtual CAN interface:

```sh
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set vcan0 mtu 72     # the CAN FD test needs MTU 72
sudo ip link set up vcan0
go test ./internal/e2e/
```

Without `vcan0` the socketcan and e2e tests skip themselves, so `go test
./...` stays green on a machine without root.

## Code conventions

- one object per file; constants and interfaces live in the file of their
  object;
- table-driven tests, even for a single case: check everything that is
  stored, returned and reaches the mocks, including error texts and types;
- domain errors carry a type (`domain.Error`, `errors.Is`-checked sentinels
  from the libraries);
- comments and documentation in English; the Russian documentation mirrors
  the English one file by file;
- fuzz tests for parsers (the candump parser already has one);
- no frame payloads in logs, ever.

## Confidentiality

Everything in this repository is public: documentation, test reports, issues
and pull request texts. Keep them generic — do not name employers, customers,
their products, devices, people or locations. Describe environments as, for
example, "an aarch64 board running Ubuntu 22.04" instead of a concrete
product, and redact addresses and keys before pasting logs.

## Pull requests

1. Fork the repository and create a branch (`feat/tls-reload`).
2. Keep the change focused; update the tests and both language versions of
   the affected documentation.
3. Make sure `make fmt vet test` is green; `make cross` and `make deb` are
   welcome for packaging changes.
4. Open the PR with a short description: what changed, why, how it was
   tested.

`main` is protected: changes land through pull requests with a green CI
only.

## Security

Do not open a public issue for a vulnerability: see [SECURITY.md](SECURITY.md).

## License

By contributing you agree that your contribution is licensed under the MIT
license of the project.
