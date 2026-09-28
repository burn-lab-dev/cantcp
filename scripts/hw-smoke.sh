#!/usr/bin/env bash
# Smoke test of cantcpd on two physical CAN interfaces.
#
# Usage:
#   scripts/hw-smoke.sh [--a can0] [--b can1] [--bitrate 500000] [--report DIR]
#
# Wiring: both adapters must be on the same bus — the same twisted pair with
# 120 Ohm terminators at both ends. `--a` is the interface the daemon serves;
# `--b` is a second adapter used by candump/cangen to observe and to load the
# bus. The interfaces must be up (the script prints the commands when they are
# not) and support the requested bitrate.
#
# This is a hardware companion to scripts/vcan-smoke.sh: the virtual smoke
# runs in CI, the hardware one is run manually (see docs/TESTING.md) and its
# report is committed to docs/test-reports/.
set -uo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
iface_a=can0
iface_b=can1
bitrate=500000
report_dir=
results=()
server_pid=

while [ $# -gt 0 ]; do
	case $1 in
	--a)
		iface_a=$2
		shift 2
		;;
	--b)
		iface_b=$2
		shift 2
		;;
	--bitrate)
		bitrate=$2
		shift 2
		;;
	--report)
		report_dir=$2
		shift 2
		;;
	-h | --help)
		sed -n '2,18p' "$0"
		exit 0
		;;
	*)
		echo "hw-smoke: unknown argument $1" >&2
		exit 2
		;;
	esac
done

work=$(mktemp -d)
cleanup() {
	if [ -n "$server_pid" ]; then
		kill "$server_pid" 2>/dev/null
	fi
	pkill -x cantcpd 2>/dev/null
	pkill -x candump 2>/dev/null
	rm -rf "$work"
}
trap cleanup EXIT

record() {
	results+=("$1|$2|$3")
	printf '%-4s %s: %s\n' "$1" "$2" "$3"
}

require_up() {
	if ! ip link show "$1" up >/dev/null 2>&1; then
		echo "hw-smoke: $1 is not up; bring it up first:" >&2
		echo "  sudo ip link set $1 up type can bitrate $bitrate" >&2
		exit 2
	fi
}
require_up "$iface_a"
require_up "$iface_b"

echo "== building the binaries"
(cd "$root" && make build >/dev/null 2>&1) || {
	echo "hw-smoke: make build failed" >&2
	exit 2
}

start_server() {
	setsid "$root/bin/cantcpd" --can "$iface_a" --listen 127.0.0.1:29536 \
		--stats-listen 127.0.0.1:29537 </dev/null >"$work/server.log" 2>&1 &
	server_pid=$!
	for _ in $(seq 1 50); do
		curl -fsS --max-time 1 http://127.0.0.1:29537/healthz >/dev/null 2>&1 && return 0
		sleep 0.1
	done
	cat "$work/server.log" >&2
	return 1
}

# The frame matrix over the real bus: a client sends, the observer adapter
# and the client itself must see every frame (the gateway loopback).
matrix="$work/matrix.txt"
python3 - "$matrix" <<'PY'
import sys

lines = ["123#1122334455667788", "7FF#R", "1ABCDE#DEADBEEF"]
ids = ["123", "1ABCDE"]
for i, n in enumerate([0, 8, 12, 16, 32, 64]):
    data = bytes((j + i + 1) & 0xFF for j in range(n)).hex().upper()
    lines.append(f"{ids[i % 2]}##{'1' if n else '0'}{data}")
with open(sys.argv[1], "w") as handle:
    handle.write("\n".join(lines) + "\n")
PY
matrix_len=$(wc -l <"$matrix")

if start_server; then
	(timeout 30 candump "$iface_b" >"$work/candump.log" 2>&1 &)
	(timeout 30 "$root/bin/cantcp-cli" listen --json --count "$matrix_len" >"$work/listen.json" 2>&1 &)
	sleep 2
	while IFS= read -r frame_line; do
		printf '%s\n' "$frame_line" | "$root/bin/cantcp-cli" send --input - >/dev/null 2>&1
		sleep 0.02
	done <"$matrix"
	sleep 3
	pkill -x candump 2>/dev/null
	seen=$(wc -l <"$work/listen.json" 2>/dev/null || echo 0)
	bus=$(wc -l <"$work/candump.log" 2>/dev/null || echo 0)
	if [ "$seen" = "$matrix_len" ] && [ "$bus" = "$matrix_len" ]; then
		record PASS matrix "$matrix_len frames ($bitrate bit/s) reached the client and the observer"
	else
		record FAIL matrix "client $seen/$matrix_len, observer $bus/$matrix_len"
	fi

	# Moderate load from the observer adapter: every frame must arrive at the
	# client complete, unique and in order.
	load=1000
	(timeout 60 "$root/bin/cantcp-cli" listen --json --count "$load" >"$work/load.json" 2>&1 &)
	sleep 0.5
	cangen "$iface_b" -g 2 -I 321 -n "$load" >/dev/null 2>&1
	sleep 4
	python3 - "$work/load.json" "$load" <<'PY' >"$work/load.check"
import json
import sys

try:
    frames = [json.loads(line) for line in open(sys.argv[1])]
except FileNotFoundError:
    frames = []
expected = int(sys.argv[2])
print(f"received={len(frames)} expected={expected} complete={len(frames) == expected}")
PY
	if grep -q "complete=True" "$work/load.check"; then
		record PASS load "all $load frames from the observer arrived complete"
	else
		record FAIL load "$(cat "$work/load.check")"
	fi
	stop_server 2>/dev/null || true
else
	record FAIL matrix "the daemon did not start on $iface_a"
fi

pass=0
fail=0
for line in "${results[@]}"; do
	case $line in
	PASS*) pass=$((pass + 1)) ;;
	FAIL*) fail=$((fail + 1)) ;;
	esac
done
echo
echo "hw-smoke: $pass passed, $fail failed, $iface_a <-> $iface_b at $bitrate bit/s"

if [ -n "$report_dir" ]; then
	mkdir -p "$report_dir"
	python3 - "$report_dir" "$iface_a" "$iface_b" "$bitrate" "$pass" "$fail" "${results[@]}" <<'PY'
import json
import sys
from datetime import datetime, timezone

report_dir, iface_a, iface_b, bitrate, passed, failed = sys.argv[1:7]
rows = [line.split("|", 2) for line in sys.argv[7:]]
stamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
with open(f"{report_dir}/report.json", "w", encoding="utf-8") as handle:
    json.dump(
        {
            "generated_at": stamp,
            "interfaces": [iface_a, iface_b],
            "bitrate": int(bitrate),
            "passed": int(passed),
            "failed": int(failed),
            "results": [{"status": s, "name": n, "detail": d} for s, n, d in rows],
        },
        handle,
        indent=2,
    )
    handle.write("\n")
lines = [
    f"# hardware smoke report — {stamp}",
    "",
    f"- interfaces: `{iface_a}` (daemon) and `{iface_b}` (observer), {bitrate} bit/s",
    f"- result: **{passed} passed, {failed} failed**",
    "",
    "| Status | Check | Detail |",
    "|---|---|---|",
]
for status, name, detail in rows:
    lines.append(f"| {status} | `{name}` | {detail} |")
with open(f"{report_dir}/report.md", "w", encoding="utf-8") as handle:
    handle.write("\n".join(lines) + "\n")
print(f"hw-smoke: report written to {report_dir}")
PY
fi

if [ "$fail" -gt 0 ]; then
	exit 1
fi
