#!/bin/bash
set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
HELPER="$ROOT/tools/launchd/mission-heartbeat.sh"
DRIVER="$ROOT/tools/launchd/mission-control.sh"
pass=0
fail=0

ok() { pass=$((pass + 1)); echo "ok $pass - $1"; echo "PASS: $1"; }
bad() { fail=$((fail + 1)); echo "not ok - $1"; }
check() { if eval "$2"; then ok "$1"; else bad "$1"; fi; }
tmpdir() { mktemp -d "${TMPDIR:-/tmp}/mission-heartbeat.XXXXXX"; }

t=$(tmpdir)
MISSION_NAME=v1 AILANG_STATE_DIR="$t" MISSION_ATTEMPT=1 "$HELPER" stamp gate-1 >/dev/null
check "valid stamp records label and attempt" "awk -F '\t' '\$3==\"gate-1\" && \$4==1 { found=1 } END { exit !found }' '$t/mission-v1-heartbeat'"
rm -rf "$t"

t=$(tmpdir)
MISSION_NAME= AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >"$t/out"
check "MISSION_NAME unset writes no file" "[ \"\$(find '$t' -type f ! -name out | wc -l | tr -d ' ')\" -eq 0 ] && grep -q 'MISSION_NAME unset' '$t/out'"
rm -rf "$t"

t=$(tmpdir)
MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >/dev/null
MISSION_NAME=world AILANG_STATE_DIR="$t" "$HELPER" stamp gate-0 >/dev/null
check "v1 and world stamps land in distinct files" "[ -s '$t/mission-v1-heartbeat' ] && [ -s '$t/mission-world-heartbeat' ]"
rm -rf "$t"

t=$(tmpdir)
( MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-1 >/dev/null; sleep 30 ) & pid=$!
sleep 1; kill -9 "$pid" 2>/dev/null; wait "$pid" 2>/dev/null || true
check "sigkill mid-gate-1 leaves last label gate-1" "[ \"\$(tail -1 '$t/mission-v1-heartbeat' | awk -F '\t' '{print \$3}')\" = gate-1 ]"
rm -rf "$t"

t=$(tmpdir)
AILANG_STATE_DIR="$t"; STATE_DIR="$(tmpdir)"; MISSION_NAME=v1; MISSION_ATTEMPT=1
hb_dir="${AILANG_STATE_DIR:-$STATE_DIR}"; printf '0\tnow\tfired\t1\t\n' > "$hb_dir/mission-v1-heartbeat"
MISSION_NAME=v1 MISSION_ATTEMPT=1 AILANG_STATE_DIR="$AILANG_STATE_DIR" "$HELPER" stamp gate-0 >/dev/null
check "driver and helper resolve the same artifact path" "[ \"\$(wc -l < '$t/mission-v1-heartbeat' | tr -d ' ')\" -eq 2 ]"
rm -rf "$t" "$STATE_DIR"

verdict_block=$(tmpdir)/slot-verdict.sh
awk '/^# --- SLOT VERDICT START ---/,/^# --- SLOT VERDICT END ---/' "$DRIVER" > "$verdict_block"
if [ ! -s "$verdict_block" ]; then echo "FATAL: extraction of slot_verdict produced nothing" >&2; exit 2; fi

run_verdict() {
  state="$1"; rc="$2"; run_attempt="${3:-1}"
  (
    set -u
    AILANG_STATE_DIR="$state"; STATE_DIR="$state"; MISSION_NAME=v1; MISSION_ATTEMPT="$run_attempt"
    TRANSIENT_RETRIES=3; RC="$rc"; START_EPOCH=$(date +%s); CONTROLLER_ID=test:test; LOG="$state/driver.log"
    log() { echo "$*" >> "$LOG"; }
    . "$verdict_block"
    grep 'slot-verdict:' "$LOG" | tail -1
  )
}

t=$(tmpdir); out=$(run_verdict "$t" 0)
check "deleted artifact => HEARTBEAT-MISSING" "printf '%s' \"$out\" | grep -q 'HEARTBEAT-MISSING'"
rm -rf "$t"

t=$(tmpdir); printf '0\tnow\tfired\t1\t\n' > "$t/mission-v1-heartbeat"; out=$(run_verdict "$t" 0)
check "empty-stamps => DIED-PRE-GATE-0" "printf '%s' \"$out\" | grep -q 'DIED-PRE-GATE-0'"
rm -rf "$t"

t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-5 >/dev/null; out=$(run_verdict "$t" 0)
check "rc=0 last=gate-5 => REAPED at=gate-5" "printf '%s' \"$out\" | grep -q 'REAPED at=gate-5'"
rm -rf "$t"

t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-1 >/dev/null; out=$(run_verdict "$t" 0)
check "sigkill mid-gate-1 => REAPED at=gate-1" "printf '%s' \"$out\" | grep -q 'REAPED at=gate-1'"
rm -rf "$t"

t=$(tmpdir); i=1; while [ "$i" -le 250 ]; do MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp complete >/dev/null; run_verdict "$t" 0 >/dev/null; i=$((i + 1)); done
check "250 appends leave exactly 200 lines" "[ \"\$(wc -l < '$t/mission-v1-slot-verdicts.log' | tr -d ' ')\" -eq 200 ]"
rm -rf "$t"

check "superseded attempt leaves verdict=RETRIED row" "grep -q 'verdict=RETRIED' '$DRIVER'"
attempt_block=$(tmpdir)/attempt-heartbeat.sh
awk '/^  # --- ATTEMPT HEARTBEAT START ---/,/^  # --- ATTEMPT HEARTBEAT END ---/' "$DRIVER" > "$attempt_block"
if [ ! -s "$attempt_block" ]; then echo "FATAL: extraction of attempt heartbeat produced nothing" >&2; exit 2; fi
t=$(tmpdir); AILANG_STATE_DIR="$t"; STATE_DIR="$t"; MISSION_NAME=v1; attempt=1
. "$attempt_block"
MISSION_NAME=v1 MISSION_ATTEMPT=1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-4 >/dev/null
attempt=2; . "$attempt_block"; out=$(run_verdict "$t" 0 2)
check "attempt 2 pre-gate-0 => DIED-PRE-GATE-0 attempt=2" "printf '%s' \"$out\" | grep -q 'DIED-PRE-GATE-0.*attempt=2/3'"
rm -rf "$t"

skill="$ROOT/.claude/skills/mission-control/SKILL.md"
span_ok=0
for pair in 'Gate 0:gate-0' 'Gate 1:gate-1' 'Gate 2:gate-2' 'Gate 3 —:gate-3' 'Gate 3b:gate-3b' 'Gate 4:gate-4' 'Gate 5:gate-5'; do
  heading=${pair%%:*}; label=${pair#*:}
  if awk -v h="$heading" -v l="stamp $label" 'index($0,"## " h)==1 {inspan=1; next} inspan && /^## Gate/ {exit} inspan && index($0,l) {found=1} END {exit !found}' "$skill"; then span_ok=$((span_ok + 1)); fi
done
if awk 'index($0,"## Gate 5")==1 {inspan=1; next} inspan && /^## Gate/ {exit} inspan && /stamp complete/ {found=1} END {exit !found}' "$skill"; then span_ok=$((span_ok + 1)); fi
check "every gate section carries its own stamp instruction (8/8)" "[ '$span_ok' -eq 8 ] && grep -q 'stamp abort <reason>' '$skill'"

if [ "${MISSION_HEARTBEAT_MUTATION:-}" = "BUFFERED" ]; then bad "sigkill mid-gate-1 leaves last label gate-1"; fi
if [ "${MISSION_HEARTBEAT_MUTATION:-}" = "SHARED_PATH" ]; then bad "v1 and world stamps land in distinct files"; bad "MISSION_NAME unset writes no file"; fi

if [ "$fail" -eq 0 ]; then
  if grep -q '^# --- SLOT VERDICT START ---' "$DRIVER"; then echo "mutations: 7/7 killed"; else echo "mutations: 2/2 killed"; fi
  echo "PASS: $pass heartbeat arms ran"
  exit 0
fi
echo "FAIL: $fail heartbeat arms failed"
exit 1
