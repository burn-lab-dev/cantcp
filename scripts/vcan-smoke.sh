#!/usr/bin/env bash
# End-to-end smoke test of cantcpd/cantcp-cli on a virtual CAN bus.
#
# Usage:
#   scripts/vcan-smoke.sh [--iface vcan0] [--report DIR] [--keep]
#
# The interface must exist, be up and have MTU 72 (see docs/TESTING.md):
#
#   sudo modprobe vcan
#   sudo ip link add dev vcan0 type vcan
#   sudo ip link set vcan0 mtu 72
#   sudo ip link set up vcan0
#
# Root is needed only for those commands, not for the test itself. The script
# builds the binaries, runs the daemon on loopback ports, exercises the frame
# matrix, the Python client (when the library checkout has a virtualenv),
# limits, TLS and SIGHUP, then prints a report. With --report DIR it writes
# report.json, report.md and report.html into DIR (used by the CI).
set -uo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
iface=vcan0
report_dir=
keep=0

while [ $# -gt 0 ]; do
	case $1 in
	--iface)
		iface=$2
		shift 2
		;;
	--report)
		report_dir=$2
		shift 2
		;;
	--keep)
		keep=1
		shift
		;;
	-h | --help)
		sed -n '2,20p' "$0"
		exit 0
		;;
	*)
		echo "vcan-smoke: unknown argument $1" >&2
		exit 2
		;;
	esac
done

work=$(mktemp -d)
server_pid=
results=()

cleanup() {
	if [ -n "$server_pid" ]; then
		kill "$server_pid" 2>/dev/null
	fi
	pkill -x cantcpd 2>/dev/null
	pkill -x cantcp-cli 2>/dev/null
	pkill -x candump 2>/dev/null
	if [ "$keep" = 1 ]; then
		echo "artifacts kept in $work"
	else
		rm -rf "$work"
	fi
}
trap cleanup EXIT

record() {
	# record <PASS|FAIL|SKIP> <name> <detail>
	results+=("$1|$2|$3")
	printf '%-4s %s: %s\n' "$1" "$2" "$3"
}

# start_server <extra flags...>; waits until the statistics endpoint answers.
start_server() {
	setsid "$root/bin/cantcpd" --can "$iface" --listen 127.0.0.1:29536 \
		--stats-listen 127.0.0.1:29537 "$@" </dev/null >"$work/server.log" 2>&1 &
	server_pid=$!
	for _ in $(seq 1 50); do
		if curl -fsS --max-time 1 http://127.0.0.1:29537/healthz >/dev/null 2>&1; then
			return 0
		fi
		sleep 0.1
	done
	echo "vcan-smoke: the daemon did not start:" >&2
	cat "$work/server.log" >&2
	return 1
}

stop_server() {
	if [ -n "$server_pid" ]; then
		kill "$server_pid" 2>/dev/null
		wait "$server_pid" 2>/dev/null
		server_pid=
	fi
}

stats() {
	curl -fsS --max-time 3 http://127.0.0.1:29537/api/v1/stats
}

# --- preflight -------------------------------------------------------------

if ! ip link show "$iface" >/dev/null 2>&1; then
	echo "vcan-smoke: $iface does not exist; create it first:" >&2
	echo "  sudo modprobe vcan && sudo ip link add dev $iface type vcan" >&2
	echo "  sudo ip link set $iface mtu 72 && sudo ip link set up $iface" >&2
	exit 2
fi
mtu=$(cat "/sys/class/net/$iface/mtu")
if [ "$mtu" -lt 72 ]; then
	echo "vcan-smoke: $iface MTU is $mtu; CAN FD needs MTU 72:" >&2
	echo "  sudo ip link set $iface mtu 72" >&2
	exit 2
fi

echo "== building the binaries"
if ! (cd "$root" && make build >"$work/build.log" 2>&1); then
	echo "vcan-smoke: make build failed:" >&2
	cat "$work/build.log" >&2
	exit 2
fi

python_client=
python_dir=$root/../cantcp-lib-python
if [ -x "$python_dir/.venv/bin/python3" ] && [ -f "$python_dir/examples/client.py" ]; then
	python_client="$python_dir/.venv/bin/python3 $python_dir/examples/client.py"
fi

# --- frame matrix ----------------------------------------------------------

matrix="$work/matrix.txt"
python3 - "$matrix" <<'PY'
import sys

lines = [
    "123#1122334455667788",  # classic 8
    "000#",                  # classic 0
    "7FF#AA",                # classic 1
    "7FF#R",                 # remote
    "1ABCDE#DEADBEEF",       # extended
]
ids = ["123", "1ABCDE"]
for i, n in enumerate([0, 1, 8, 12, 16, 20, 24, 32, 48, 64]):
    data = bytes((j + i + 1) & 0xFF for j in range(n)).hex().upper()
    lines.append(f"{ids[i % 2]}##{'1' if n else '0'}{data}")
with open(sys.argv[1], "w") as handle:
    handle.write("\n".join(lines) + "\n")
PY
matrix_len=$(wc -l <"$matrix")

if start_server; then
	(timeout 20 candump -n "$matrix_len" "$iface" >"$work/candump.log" 2>&1 &)
	(timeout 20 "$root/bin/cantcp-cli" listen --json --count "$matrix_len" \
		>"$work/listen.json" 2>"$work/listen.err" &)
	sleep 1
	"$root/bin/cantcp-cli" send --input "$matrix" >/dev/null 2>&1
	sleep 2
	seen=$(wc -l <"$work/listen.json" 2>/dev/null || echo 0)
	bus=$(wc -l <"$work/candump.log" 2>/dev/null || echo 0)
	if [ "$seen" = "$matrix_len" ] && [ "$bus" = "$matrix_len" ]; then
		record PASS matrix "$matrix_len frames of the matrix echoed to the client and seen on the bus"
	else
		record FAIL matrix "client got $seen/$matrix_len, bus saw $bus/$matrix_len"
	fi
	stop_server
else
	record FAIL matrix "the daemon did not start"
fi

# --- python client interop -------------------------------------------------

if [ -n "$python_client" ] && start_server; then
	# shellcheck disable=SC2086
	(timeout 20 $python_client listen --count 2 >"$work/py-listen.json" 2>"$work/py-listen.err" &)
	sleep 3
	"$root/bin/cantcp-cli" send --id 111 --data AABB >/dev/null 2>&1
	# shellcheck disable=SC2086
	$python_client send --id 222 --data CCDD >/dev/null 2>&1
	sleep 2
	if [ "$(wc -l <"$work/py-listen.json" 2>/dev/null || echo 0)" = 2 ]; then
		record PASS python-interop "the Python client received frames from the Go client and itself"
	else
		record FAIL python-interop "the Python client received $(wc -l <"$work/py-listen.json" 2>/dev/null || echo 0)/2 frames"
	fi
	stop_server
elif [ -z "$python_client" ]; then
	record SKIP python-interop "no cantcp-lib-python virtualenv next to the checkout"
fi

# --- rate limit ------------------------------------------------------------

if start_server --max-frames-per-second 10; then
	"$root/bin/cantcp-cli" send --id 100 --data 01 --count 100 --interval 2ms >/dev/null 2>&1
	sleep 0.5
	dropped=$(stats | python3 -c 'import json,sys; print(json.load(sys.stdin)["tcp"]["dropped"])')
	if [ "$dropped" -ge 80 ]; then
		record PASS rate-limit "100 frames in one connection, $dropped dropped by the 10 fps limit"
	else
		record FAIL rate-limit "expected >= 80 dropped, got $dropped"
	fi
	stop_server
else
	record FAIL rate-limit "the daemon did not start"
fi

# --- idle timeout ----------------------------------------------------------

if start_server --idle-timeout 1s; then
	start=$(date +%s)
	timeout 6 "$root/bin/cantcp-cli" listen --count 1 >/dev/null 2>&1
	code=$?
	elapsed=$(( $(date +%s) - start ))
	if [ "$code" -eq 0 ] && [ "$elapsed" -le 3 ]; then
		record PASS idle-timeout "a silent client was disconnected in ${elapsed}s"
	else
		record FAIL idle-timeout "exit=$code elapsed=${elapsed}s"
	fi
	stop_server
else
	record FAIL idle-timeout "the daemon did not start"
fi

# --- max connections -------------------------------------------------------

if start_server --max-connections 1; then
	(timeout 5 "$root/bin/cantcp-cli" listen --count 1 >/dev/null 2>&1 &)
	sleep 1
	timeout 5 "$root/bin/cantcp-cli" listen --count 1 >/dev/null 2>&1
	code=$?
	if [ "$code" -eq 0 ]; then
		record PASS max-connections "the second client was rejected and closed"
	else
		record FAIL max-connections "the second client exit=$code"
	fi
	stop_server
else
	record FAIL max-connections "the daemon did not start"
fi

# --- TLS 1.3 + mutual TLS + SIGHUP rotation --------------------------------

tls="$work/tls"
mkdir -p "$tls"
(
	cd "$tls" || exit 1
	openssl ecparam -name prime256v1 -genkey -noout -out ca-key.pem 2>/dev/null
	openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem -subj "/CN=cantcp smoke CA" \
		-addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" 2>/dev/null
	openssl ecparam -name prime256v1 -genkey -noout -out server-key.pem 2>/dev/null
	openssl req -new -key server-key.pem -out server.csr -subj "/CN=localhost" 2>/dev/null
	printf 'subjectAltName = DNS:localhost, IP:127.0.0.1\nextendedKeyUsage = serverAuth\n' >server-ext.cnf
	openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
		-days 825 -out server.pem -extfile server-ext.cnf 2>/dev/null
	openssl ecparam -name prime256v1 -genkey -noout -out client-key.pem 2>/dev/null
	openssl req -new -key client-key.pem -out client.csr -subj "/CN=cantcp smoke client" 2>/dev/null
	printf 'extendedKeyUsage = clientAuth\n' >client-ext.cnf
	openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
		-days 825 -out client.pem -extfile client-ext.cnf 2>/dev/null
	rm -f ./*.csr
)

tls_flags=(--tls-cert "$tls/server.pem" --tls-key "$tls/server-key.pem" --tls-ca "$tls/ca.pem" --tls-client-auth require_and_verify)
client_flags=(--tls --tls-ca "$tls/ca.pem" --tls-cert "$tls/client.pem" --tls-key "$tls/client-key.pem")

if start_server "${tls_flags[@]}"; then
	(timeout 20 "$root/bin/cantcp-cli" listen "${client_flags[@]}" --json --count 1 \
		>"$work/tls-listen.json" 2>"$work/tls-listen.err" &)
	sleep 1
	"$root/bin/cantcp-cli" send "${client_flags[@]}" --id 7A5 --data CAFE >/dev/null 2>&1
	sleep 1.5
	if grep -q '"data":"cafe"' "$work/tls-listen.json" 2>/dev/null; then
		record PASS tls-mtls "the mutual-TLS exchange delivered the frame"
	else
		record FAIL tls-mtls "the frame did not arrive over mutual TLS"
	fi

	legacy=$(echo | timeout 5 openssl s_client -connect 127.0.0.1:29536 -tls1_2 \
		-CAfile "$tls/ca.pem" 2>&1 | grep -c "alert protocol version")
	if [ "$legacy" -ge 1 ]; then
		record PASS tls-version "TLS 1.2 was rejected with a protocol alert"
	else
		record FAIL tls-version "TLS 1.2 was not rejected"
	fi

	serial_before=$(echo | timeout 5 openssl s_client -connect 127.0.0.1:29536 -tls1_3 \
		-CAfile "$tls/ca.pem" -cert "$tls/client.pem" -key "$tls/client-key.pem" 2>/dev/null \
		| openssl x509 -noout -serial)
	(
		cd "$tls" || exit 1
		openssl ecparam -name prime256v1 -genkey -noout -out server-key2.pem 2>/dev/null
		openssl req -new -key server-key2.pem -out server2.csr -subj "/CN=localhost" 2>/dev/null
		openssl x509 -req -in server2.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
			-days 825 -out server2.pem -extfile server-ext.cnf 2>/dev/null
		cp server2.pem server.pem && cp server-key2.pem server-key.pem
		rm -f server2.csr
	)
	kill -HUP "$server_pid"
	sleep 1
	serial_after=$(echo | timeout 5 openssl s_client -connect 127.0.0.1:29536 -tls1_3 \
		-CAfile "$tls/ca.pem" -cert "$tls/client.pem" -key "$tls/client-key.pem" 2>/dev/null \
		| openssl x509 -noout -serial)
	if [ -n "$serial_after" ] && [ "$serial_before" != "$serial_after" ]; then
		record PASS sighup "the certificate was rotated without a restart"
	else
		record FAIL sighup "the certificate did not change after SIGHUP"
	fi
	stop_server
else
	record FAIL tls-mtls "the TLS daemon did not start"
fi

# --- report ----------------------------------------------------------------

pass=0
fail=0
for line in "${results[@]}"; do
	case $line in
	PASS*) pass=$((pass + 1)) ;;
	FAIL*) fail=$((fail + 1)) ;;
	esac
done

echo
echo "vcan-smoke: $pass passed, $fail failed, on $iface (MTU $mtu)"

if [ -n "$report_dir" ]; then
	mkdir -p "$report_dir"
	python3 - "$report_dir" "$iface" "$mtu" "$pass" "$fail" "${results[@]}" <<'PY'
import html
import json
import sys
from datetime import datetime, timezone

report_dir, iface, mtu, passed, failed = sys.argv[1:6]
rows = [line.split("|", 2) for line in sys.argv[6:]]
stamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

with open(f"{report_dir}/report.json", "w", encoding="utf-8") as handle:
    json.dump(
        {
            "generated_at": stamp,
            "interface": iface,
            "mtu": mtu,
            "passed": int(passed),
            "failed": int(failed),
            "results": [{"status": s, "name": n, "detail": d} for s, n, d in rows],
        },
        handle,
        indent=2,
    )
    handle.write("\n")

lines = [
    f"# vcan smoke report — {stamp}",
    "",
    f"- interface: `{iface}` (MTU {mtu})",
    f"- result: **{passed} passed, {failed} failed**",
    "",
    "| Status | Check | Detail |",
    "|---|---|---|",
]
for status, name, detail in rows:
    lines.append(f"| {status} | `{name}` | {detail} |")
markdown = "\n".join(lines) + "\n"
with open(f"{report_dir}/report.md", "w", encoding="utf-8") as handle:
    handle.write(markdown)

with open(f"{report_dir}/report.html", "w", encoding="utf-8") as handle:
    handle.write(
        "<!DOCTYPE html>\n<html><head><meta charset='utf-8'><title>vcan smoke report</title>"
        "<style>body{font-family:sans-serif;max-width:60rem;margin:2rem auto;padding:0 1rem}"
        "pre{background:#f4f4f4;padding:1rem;overflow-x:auto}td,th{border:1px solid #ccc;padding:.3rem .6rem}"
        "table{border-collapse:collapse}</style></head><body><pre>"
        + html.escape(markdown)
        + "</pre></body></html>\n"
    )
print(f"vcan-smoke: report written to {report_dir}")
PY
fi

if [ "$fail" -gt 0 ]; then
	exit 1
fi
