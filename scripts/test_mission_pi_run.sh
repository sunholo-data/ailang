#!/usr/bin/env bash
# test_mission_pi_run.sh — prove each verdict of scripts/mission_pi_run.sh actually fires.
#
# Every case stubs `pi` on PATH so no model is called and no dollars are spent. The
# suite deliberately includes a case for EACH exit code: a guard that has never fired
# on a positive is not evidence the guard works, which is the exact trap the pi lane's
# previous `stopReason` assertion fell into (it passed on 0 of 4 real failures).

set -u
HERE=$(cd "$(dirname "$0")" && pwd)
SUT="$HERE/mission_pi_run.sh"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

PASS=0; FAIL=0
check() { # check <name> <expected-rc> <actual-rc> <expected-verdict> <verdict-file>
  got_v=$(sed -n 's/.*"verdict": "\([^"]*\)".*/\1/p' "$5" 2>/dev/null)
  if [ "$2" = "$3" ] && [ "$4" = "$got_v" ]; then
    echo "  PASS: $1 (rc=$3 verdict=$got_v)"; PASS=$((PASS+1))
  else
    echo "  FAIL: $1 — expected rc=$2/$4, got rc=$3/${got_v:-<none>}"; FAIL=$((FAIL+1))
  fi
}

mkstub() { # mkstub <script-body> -> writes a `pi` stub and prepends it to PATH
  mkdir -p "$TMP/bin"
  { echo '#!/usr/bin/env bash'; echo 'cat >/dev/null'; echo "$1"; } > "$TMP/bin/pi"
  chmod +x "$TMP/bin/pi"
  PATH="$TMP/bin:$PATH"; export PATH
}

mkrepo() { # mkrepo <dir> <dirty|clean>
  rm -rf "$1"; mkdir -p "$1"; git -C "$1" init -q
  echo base > "$1/f.txt"; git -C "$1" add -A
  git -C "$1" -c user.email=t@t -c user.name=t commit -qm base
  [ "$2" = "dirty" ] && echo changed > "$1/f.txt"
  return 0
}

echo "TEST 1: happy path -> ok (rc 0)"
mkstub 'printf "%s\n" "{\"type\":\"tool_execution\"}" "{\"type\":\"agent_end\"}"'
mkrepo "$TMP/wt1" dirty; echo "do the thing" > "$TMP/d1.txt"
"$SUT" --model m --directive "$TMP/d1.txt" --workdir "$TMP/wt1" --out "$TMP/o1.ndjson" \
       --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
check "happy path" 0 $? ok "$TMP/o1.ndjson.verdict.json"

echo "TEST 2: pi succeeds but changes nothing -> empty_worktree (rc 10)"
mkrepo "$TMP/wt2" clean; echo d > "$TMP/d2.txt"
"$SUT" --model m --directive "$TMP/d2.txt" --workdir "$TMP/wt2" --out "$TMP/o2.ndjson" \
       --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
check "empty worktree" 10 $? empty_worktree "$TMP/o2.ndjson.verdict.json"

echo "TEST 3: reasoning-only stream -> reasoning_stall (rc 11)"
# The measured failure: message_update flows continuously, nothing else ever does.
mkstub 'while :; do printf "{\"type\":\"message_update\",\"n\":%s}\n" "$RANDOM"; sleep 0.05; done'
mkrepo "$TMP/wt3" clean; echo d > "$TMP/d3.txt"
"$SUT" --model m --directive "$TMP/d3.txt" --workdir "$TMP/wt3" --out "$TMP/o3.ndjson" \
       --max-seconds 60 --stall-seconds 6 --verdict "$TMP/v3.json" >/dev/null 2>&1
check "reasoning stall" 11 $? reasoning_stall "$TMP/v3.json"
SNAP_LINES=$(wc -l < "$TMP/o3.ndjson.snapshot.ndjson" 2>/dev/null | tr -d ' ')
BANK_BYTES=$(wc -c < "$TMP/o3.ndjson" 2>/dev/null | tr -d ' ')
if [ "${SNAP_LINES:-0}" -le 1 ] && [ "${BANK_BYTES:-1}" -eq 0 ]; then
  echo "  PASS: message_update filtered (banked=${BANK_BYTES}B) and snapshot bounded (${SNAP_LINES} line)"; PASS=$((PASS+1))
else
  echo "  FAIL: filter leaked — banked=${BANK_BYTES}B snapshot=${SNAP_LINES} lines"; FAIL=$((FAIL+1))
fi

echo "TEST 4: silent stream -> stream_dead (rc 12)"
mkstub 'sleep 300'
mkrepo "$TMP/wt4" clean; echo d > "$TMP/d4.txt"
"$SUT" --model m --directive "$TMP/d4.txt" --workdir "$TMP/wt4" --out "$TMP/o4.ndjson" \
       --max-seconds 60 --stall-seconds 6 >/dev/null 2>&1
check "stream dead" 12 $? stream_dead "$TMP/o4.ndjson.verdict.json"

echo "TEST 5: progress keeps the clock alive past the stall bound (no false positive)"
# The guard must NOT fire on a slow-but-working run, or it just re-creates the old
# 300 MB ceiling in a new costume.
mkstub 'for i in 1 2 3 4 5 6 7 8; do printf "{\"type\":\"tool_execution\",\"i\":%s}\n" "$i"; sleep 2; done; printf "{\"type\":\"agent_end\"}\n"'
mkrepo "$TMP/wt5" dirty; echo d > "$TMP/d5.txt"
"$SUT" --model m --directive "$TMP/d5.txt" --workdir "$TMP/wt5" --out "$TMP/o5.ndjson" \
       --max-seconds 90 --stall-seconds 6 >/dev/null 2>&1
check "slow-but-working run survives" 0 $? ok "$TMP/o5.ndjson.verdict.json"

echo "TEST 6: bad arguments -> launch_failed (rc 14)"
"$SUT" --model m --directive /nonexistent/nope --workdir "$TMP" --out "$TMP/o6.ndjson" >/dev/null 2>&1
[ $? -eq 14 ] && { echo "  PASS: missing directive rejected"; PASS=$((PASS+1)); } \
              || { echo "  FAIL: missing directive not rejected"; FAIL=$((FAIL+1)); }

echo
echo "passed=$PASS failed=$FAIL"
[ "$FAIL" -eq 0 ]
