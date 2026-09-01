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

state_block=$(tmpdir)/state-dir.sh
awk '/^# --- HEARTBEAT STATE DIR START ---/,/^# --- HEARTBEAT STATE DIR END ---/' "$DRIVER" > "$state_block"
if [ ! -s "$state_block" ]; then echo "FATAL: extraction of heartbeat state dir produced nothing" >&2; exit 2; fi
t=$(tmpdir); fallback=$(tmpdir)
AILANG_STATE_DIR="$t"; STATE_DIR="$fallback"; MISSION_NAME=v1; MISSION_ATTEMPT=1
. "$state_block"
printf '0\tnow\tfired\t1\t\n' > "$_mc_slot_state/mission-v1-heartbeat"
MISSION_NAME=v1 MISSION_ATTEMPT=1 AILANG_STATE_DIR="$AILANG_STATE_DIR" "$HELPER" stamp gate-0 >/dev/null
check "S-1 driver and helper resolve AILANG_STATE_DIR identically" "[ \"\$(wc -l < '$t/mission-v1-heartbeat' | tr -d ' ')\" -eq 2 ] && [ ! -e '$fallback/mission-v1-heartbeat' ]"
rm -rf "$t" "$fallback"

verdict_block=$(tmpdir)/slot-verdict.sh
awk '/^# --- SLOT VERDICT START ---/,/^# --- SLOT VERDICT END ---/' "$DRIVER" > "$verdict_block"
if [ ! -s "$verdict_block" ]; then echo "FATAL: extraction of slot_verdict produced nothing" >&2; exit 2; fi

run_verdict() {
  state="$1"; rc="$2"; run_attempt="${3:-1}"
  (
    set -u
    AILANG_STATE_DIR="$state"; STATE_DIR="$state"; _mc_slot_state="$state"; MISSION_NAME=v1; MISSION_ATTEMPT="$run_attempt"
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

t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp abort "operator requested stop" >/dev/null; out=$(run_verdict "$t" 0)
check "rc=0 last=abort => ABORTED" "printf '%s' \"$out\" | grep -q 'slot-verdict: ABORTED rc=0'"
rm -rf "$t"

t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-3 >/dev/null; out=$(run_verdict "$t" 143)
check "rc=143 last=gate-3 => KILLED" "printf '%s' \"$out\" | grep -q 'slot-verdict: KILLED at=gate-3 rc=143'"
rm -rf "$t"

t=$(tmpdir); MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-2 >/dev/null; out=$(run_verdict "$t" 1)
check "rc=1 last=gate-2 => CRASHED" "printf '%s' \"$out\" | grep -q 'slot-verdict: CRASHED at=gate-2 rc=1'"
rm -rf "$t"

t=$(tmpdir); i=1; while [ "$i" -le 250 ]; do MISSION_NAME=v1 AILANG_STATE_DIR="$t" "$HELPER" stamp complete >/dev/null; run_verdict "$t" 0 >/dev/null; i=$((i + 1)); done
check "250 appends leave exactly 200 lines" "[ \"\$(wc -l < '$t/mission-v1-slot-verdicts.log' | tr -d ' ')\" -eq 200 ]"
rm -rf "$t"

retry_block=$(tmpdir)/retry-history.sh
awk '/^    # --- RETRY HISTORY START ---/,/^    # --- RETRY HISTORY END ---/' "$DRIVER" > "$retry_block"
if [ ! -s "$retry_block" ]; then echo "FATAL: extraction of retry history produced nothing" >&2; exit 2; fi
t=$(tmpdir); _mc_slot_state="$t"; MISSION_NAME=v1; RC=75; attempt=1; TRANSIENT_RETRIES=3; CONTROLLER_ID=test:retry; START_EPOCH=$(( $(date +%s) - 4 ))
printf '0\tnow\tgate-2\t1\t\n' > "$t/mission-v1-heartbeat"
. "$retry_block"
check "superseded attempt appends runtime RETRIED history row" "awk '/verdict=RETRIED/ && /at=gate-2/ && /rc=75/ && /attempt=1\/3/ && /elapsed_s=[1-9][0-9]*/ && /stamps= *1/ && /controller=test:retry/ { found=1 } END { exit !found }' '$t/mission-v1-slot-verdicts.log'"
rm -rf "$t"
attempt_block=$(tmpdir)/attempt-heartbeat.sh
awk '/^  # --- ATTEMPT HEARTBEAT START ---/,/^  # --- ATTEMPT HEARTBEAT END ---/' "$DRIVER" > "$attempt_block"
if [ ! -s "$attempt_block" ]; then echo "FATAL: extraction of attempt heartbeat produced nothing" >&2; exit 2; fi
t=$(tmpdir); AILANG_STATE_DIR="$t"; STATE_DIR="$t"; _mc_slot_state="$t"; MISSION_NAME=v1; attempt=1
. "$attempt_block"
MISSION_NAME=v1 MISSION_ATTEMPT=1 AILANG_STATE_DIR="$t" "$HELPER" stamp gate-4 >/dev/null
attempt=2; . "$attempt_block"; out=$(run_verdict "$t" 0 2)
check "attempt 2 pre-gate-0 => DIED-PRE-GATE-0 attempt=2" "printf '%s' \"$out\" | grep -q 'DIED-PRE-GATE-0.*attempt=2/3'"
rm -rf "$t"

end_line=$(grep -n '^# --- SLOT VERDICT END ---' "$DRIVER" | cut -d: -f1)
release_line=$(grep -n 'rm -f "$PIDFILE"   # this instance owns' "$DRIVER" | cut -d: -f1)
start_line=$(grep -n '^# --- SLOT VERDICT START ---' "$DRIVER" | cut -d: -f1)
run_line=$(grep -n '^  _mc_run_once; RC=\$?' "$DRIVER" | cut -d: -f1)
check "slot verdict completes before PIDFILE guard release" "case '$end_line:$release_line' in *[!0-9:]*|:*|*:) false ;; *) [ '$end_line' -lt '$release_line' ] ;; esac"
check "slot verdict starts after _mc_run_once retry call" "case '$start_line:$run_line' in *[!0-9:]*|:*|*:) false ;; *) [ '$start_line' -gt '$run_line' ] ;; esac"

notify_block=$(tmpdir)/slot-notify.sh
awk '/^# --- SLOT NOTIFY START ---/,/^# --- SLOT NOTIFY END ---/' "$DRIVER" > "$notify_block"
if [ ! -s "$notify_block" ]; then echo "FATAL: extraction of slot notify produced nothing" >&2; exit 2; fi
t=$(tmpdir); _mc_slot_state="$t"; MISSION_NAME=v1; RC=0; MISSION_ATTEMPT=1; TRANSIENT_RETRIES=3; LOG="$t/driver.log"; MSG_FROM=test; MISSION_GH_ISSUE=; calls="$t/calls"
_mc_bounded() { printf '%s\n' "$*" >> "$calls"; }
_mc_slot_verdict='REAPED at=gate-2'; . "$notify_block"; . "$notify_block"
_mc_slot_verdict='REAPED at=gate-3'; . "$notify_block"
_mc_slot_verdict=COMPLETED; . "$notify_block"
_mc_slot_verdict='REAPED at=gate-3'; . "$notify_block"
check "phase-2 notices use _mc_bounded and dedupe verdict episodes" "[ \"\$(wc -l < '$calls' | tr -d ' ')\" -eq 3 ] && [ \"\$(grep -c '^30 ailang messages send' '$calls')\" -eq 3 ] && [ -f '$t/mission-v1-reaped.episode' ]"
rm -rf "$t"

skill="$ROOT/.claude/skills/mission-control/SKILL.md"
span_ok=0
for pair in 'Gate 0:gate-0' 'Gate 1:gate-1' 'Gate 2:gate-2' 'Gate 3 —:gate-3' 'Gate 3b:gate-3b' 'Gate 4:gate-4' 'Gate 5:gate-5'; do
  heading=${pair%%:*}; label=${pair#*:}
  if awk -v h="$heading" -v l="stamp $label" 'index($0,"## " h)==1 {inspan=1; next} inspan && /^## Gate/ {exit} inspan && index($0,l) {found=1} END {exit !found}' "$skill"; then span_ok=$((span_ok + 1)); fi
done
if awk 'index($0,"## Gate 5")==1 {inspan=1; next} inspan && /^## Gate/ {exit} inspan && /stamp complete/ {found=1} END {exit !found}' "$skill"; then span_ok=$((span_ok + 1)); fi
check "every gate section carries its own stamp instruction (8/8)" "[ '$span_ok' -eq 8 ] && grep -q 'stamp abort <reason>' '$skill'"

if [ "$fail" -eq 0 ]; then
  echo "PASS: $pass heartbeat arms ran"
  exit 0
fi
echo "FAIL: $fail heartbeat arms failed"
exit 1
