# Test reports

**English** | [Русский](README.ru.md)

Developed by **[BURN-LAB](https://burn-lab.ru)** — embedded software
development: Linux, drivers, CAN and industrial telemetry.

Reports of test runs against the protocol canon
([cantcp-spec](https://github.com/burn-lab-dev/cantcp-spec)) and the
daemon/client pair:

- automatic runs: `scripts/vcan-smoke.sh` in CI uploads `report.json` and
  `report.md` as artifacts and into the job summary;
- committed reports: one Markdown file per run, named
  `YYYY-MM-DD-<environment>.md` (and `.ru.md`), written after a manual run
  (`scripts/vcan-smoke.sh --report DIR`, `scripts/hw-smoke.sh --report DIR`);
- the release workflow mirrors this directory to the GitHub Pages site
  (`/test-reports/`).

A report states the environment (machine, kernel, Go, library versions), the
interface and MTU or bitrate, the check table with pass/fail and the
deviations found. Keep the raw `report.json` next to the Markdown when it is
long.

| Report | Environment |
|---|---|
| [2026-09-28-vcan-smoke.md](2026-09-28-vcan-smoke.md) | virtual bus (vcan0, MTU 72), first full run |
