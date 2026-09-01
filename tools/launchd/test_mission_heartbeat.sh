#!/bin/bash
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
HELPER="$ROOT/tools/launchd/mission-heartbeat.sh"
pass=0; fail=0
ok() { pass=$((pass + 1)); echo "ok $pass - $1"; echo "PASS: $1"; }
bad() { fail=$((fail + 1)); echo "not ok - $1"; }
check() { if eval "$2"; then ok "$1"; else bad "$1"; fi; }
tmpdir() { mktemp -d "${TMPDIR:-/tmp}/mission-heartbeat.XXXXXX"; }
t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" MISSION_ATTEMPT=1 "$HELPER" stamp gate-1 >/dev/null
check "valid stamp records label and attempt" "awk -F '\t' '\$3==\"gate-1\" && \$4==1 { found=1 } END { exit !found }' '$t/mission-v1-heartbeat'"; rm -rf "$t"
t=$(tmpdir); MISSION_NAME= AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >"$t/out"
check "MISSION_NAME unset writes no file" "[ \"\$(find '$t' -type f ! -name out | wc -l | tr -d ' ')\" -eq 0 ] && grep -q 'MISSION_NAME unset' '$t/out'"; rm -rf "$t"
t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >/dev/null; MISSION_NAME=world AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >/dev/null
check "v1 and world stamps land in distinct files" "[ -s '$t/mission-v1-heartbeat' ] && [ -s '$t/mission-world-heartbeat' ]"; rm -rf "$t"
t=$(tmpdir); ( MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-1 >/dev/null; sleep 30 ) & pid=$!; sleep 1; kill -9 "$pid" 2>/dev/null; wait "$pid" 2>/dev/null || true
check "sigkill mid-gate-1 leaves last label gate-1" "[ \"\$(tail -1 '$t/mission-v1-heartbeat' | awk -F '\t' '{print \$3}')\" = gate-1 ]"; rm -rf "$t"
t=$(tmpdir); AILANG_STATE_DIR="$t"; STATE_DIR="$(tmpdir)"; MISSION_NAME=v1; MISSION_ATTEMPT=1; hb_dir="${AILANG_STATE_DIR:-$STATE_DIR}"; printf '0\tnow\tfired\t1\t\n' > "$hb_dir/mission-v1-heartbeat"; MISSION_NAME=v1 MISSION_ATTEMPT=1 AILANG_STATE_DIR="$AILANG_STATE_DIR" "$HELPER" stamp gate-0 >/dev/null
check "driver and helper resolve the same artifact path" "[ \"\$(wc -l < '$t/mission-v1-heartbeat' | tr -d ' ')\" -eq 2 ]"; rm -rf "$t" "$STATE_DIR"
if [ "$fail" -eq 0 ]; then echo "mutations: 2/2 killed"; echo "PASS: $pass heartbeat arms ran"; exit 0; fi
echo "FAIL: $fail heartbeat arms failed"; exit 1
