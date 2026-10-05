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
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_SANDBOX_NODE_MODULES MISSION_PI_SANDBOX_STAGE_PARENT MISSION_PI_CLAUDE_TMP_DIR

MISSION_PI_POLL_SECONDS=1; export MISSION_PI_POLL_SECONDS
field() { jq -r ".$2" "$1"; } # field <verdict-file> <key>

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
  { echo '#!/usr/bin/env bash'; echo 'cat >/dev/null'; echo ': > "$PI_SANDBOX_READY_FILE"'; echo "$1"; } > "$TMP/bin/pi"
  chmod +x "$TMP/bin/pi"
  PATH="$TMP/bin:$PATH"; export PATH
}

mkrepo() { # mkrepo <dir> <dirty|clean|dirty-untracked>
  rm -rf "$1"; mkdir -p "$1"; git -C "$1" init -q
  echo base > "$1/f.txt"; git -C "$1" add -A
  git -C "$1" -c user.email=t@t -c user.name=t commit -qm base
  [ "$2" = "dirty" ] && echo changed > "$1/f.txt"
  [ "$2" = "dirty-untracked" ] && echo u0 > "$1/u.txt"
  return 0
}

echo "TEST 1: happy path -> ok (rc 0)"
mkstub 'echo work > work.txt; printf "%s\n" "{\"type\":\"tool_execution_end\"}" "{\"type\":\"agent_end\"}"'
mkrepo "$TMP/wt1" dirty; echo "do the thing" > "$TMP/d1.txt"
"$SUT" --model m --directive "$TMP/d1.txt" --workdir "$TMP/wt1" --out "$TMP/o1.ndjson" \
       --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
check "happy path" 0 $? ok "$TMP/o1.ndjson.verdict.json"

echo "TEST 2: pi succeeds but changes nothing -> empty_worktree (rc 10)"
mkstub 'printf "%s\n" "{\"type\":\"tool_execution_end\"}" "{\"type\":\"agent_end\"}"'
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
SNAP_EVERY="${MISSION_PI_SNAP_EVERY:-50}"
if [ "${SNAP_LINES:-0}" -ge 1 ] && [ "${SNAP_LINES:-0}" -le "$SNAP_EVERY" ] && [ "${BANK_BYTES:-1}" -eq 0 ]; then
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

echo "TEST 4b: silent while a tool call is OPEN -> tool_hang (rc 18), not stream_dead"
# Regression pin, 2026-09-29: three executor runs sat in a hung `ailang messages list`
# and were banked as the model's stream_dead.
mkstub 'printf "{\"type\":\"tool_execution_start\",\"toolName\":\"bash\",\"args\":{\"command\":\"ailang messages list --unread\"}}\n"; sleep 300'
mkrepo "$TMP/wt4b" clean; echo d > "$TMP/d4b.txt"
"$SUT" --model m --directive "$TMP/d4b.txt" --workdir "$TMP/wt4b" --out "$TMP/o4b.ndjson" \
       --max-seconds 60 --stall-seconds 6 >/dev/null 2>&1
check "tool hang" 18 $? tool_hang "$TMP/o4b.ndjson.verdict.json"
if jq -e '.hung_tool | test("ailang messages list")' "$TMP/o4b.ndjson.verdict.json" >/dev/null 2>&1; then
  echo "  PASS: verdict names the hung command"; PASS=$((PASS+1))
else
  echo "  FAIL: hung_tool missing from verdict"; FAIL=$((FAIL+1))
fi

echo "TEST 5: progress keeps the clock alive past the stall bound (no false positive)"
# The guard must NOT fire on a slow-but-working run, or it just re-creates the old
# 300 MB ceiling in a new costume.
mkstub 'echo work > work.txt; for i in 1 2 3 4 5 6 7 8; do printf "{\"type\":\"tool_execution_end\",\"i\":%s}\n" "$i"; sleep 2; done; printf "{\"type\":\"agent_end\"}\n"'
mkrepo "$TMP/wt5" dirty; echo d > "$TMP/d5.txt"
"$SUT" --model m --directive "$TMP/d5.txt" --workdir "$TMP/wt5" --out "$TMP/o5.ndjson" \
       --max-seconds 90 --stall-seconds 6 >/dev/null 2>&1
check "slow-but-working run survives" 0 $? ok "$TMP/o5.ndjson.verdict.json"

echo "TEST 5b: pi children run as an unattended mission stage"
# The session-protocol gate waives the inbox call only when AILANG_MISSION_STAGE is set;
# without it every sandboxed role blocks on an unreachable message store (2026-09-29).
mkstub 'printf "%s" "${AILANG_MISSION_STAGE:-unset}" > stage_marker.txt; printf "{\"type\":\"agent_end\"}\n"'
mkrepo "$TMP/wt5b" clean; echo d > "$TMP/d5b.txt"
"$SUT" --model m --directive "$TMP/d5b.txt" --workdir "$TMP/wt5b" --out "$TMP/o5b.ndjson" \
       --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
if [ "$(cat "$TMP/wt5b/stage_marker.txt" 2>/dev/null)" = "1" ]; then
  echo "  PASS: pi child sees AILANG_MISSION_STAGE=1"; PASS=$((PASS+1))
else
  echo "  FAIL: pi child AILANG_MISSION_STAGE=$(cat "$TMP/wt5b/stage_marker.txt" 2>/dev/null || echo '<no marker>')"; FAIL=$((FAIL+1))
fi

echo "TEST 6: pi runs INSIDE --workdir, not the caller's cwd"
# Regression pin. Caught live 2026-08-26: a real run reported 4 tool executions and 0
# changed files because pi edited the caller's cwd while the git assertion read
# --workdir. A guard asserting on a directory the model never touched is worse than no
# guard — it reports empty_worktree for good work, and would report ok for none.
mkstub 'pwd > cwd_marker.txt; printf "{\"type\":\"agent_end\"}\n"'
mkrepo "$TMP/wt7" clean; echo d > "$TMP/d7.txt"
( cd "$TMP" && "$SUT" --model m --directive "$TMP/d7.txt" --workdir "$TMP/wt7" \
       --out "$TMP/o7.ndjson" --max-seconds 30 --stall-seconds 10 ) >/dev/null 2>&1
if [ -f "$TMP/wt7/cwd_marker.txt" ]; then
  echo "  PASS: pi ran in the worktree ($(cat "$TMP/wt7/cwd_marker.txt"))"; PASS=$((PASS+1))
else
  echo "  FAIL: pi ran outside --workdir; marker not in worktree"; FAIL=$((FAIL+1))
fi
# ...and the run it just did must therefore register as real work.
check "worktree write is seen by the verdict" 0 0 ok "$TMP/o7.ndjson.verdict.json"

echo "TEST 7: bad arguments -> launch_failed (rc 14)"
"$SUT" --model m --directive /nonexistent/nope --workdir "$TMP" --out "$TMP/o6.ndjson" >/dev/null 2>&1
[ $? -eq 14 ] && { echo "  PASS: missing directive rejected"; PASS=$((PASS+1)); } \
              || { echo "  FAIL: missing directive not rejected"; FAIL=$((FAIL+1)); }

# Pre-dirty worktree arms (ticket pi-runner:verdict-blind-to-commits-and-predirty). A tree that is
# already dirty at launch must not read ok unless pi changed it; TEST 1 and TEST 5 used to pass only
# because of that hole.
run_pd() { # run_pd <n> <repo-mode> <stub-body> -> runs the SUT, sets PDRC and PDV
  mkstub "$3"
  mkrepo "$TMP/pd$1" "$2"; echo d > "$TMP/dpd$1.txt"
  "$SUT" --model m --directive "$TMP/dpd$1.txt" --workdir "$TMP/pd$1" --out "$TMP/opd$1.ndjson" \
         --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
  PDRC=$?; PDV="$TMP/opd$1.ndjson.verdict.json"
}
expect_field() { # expect_field <name> <verdict-file> <key> <want>
  got=$(field "$2" "$3")
  if [ "$got" = "$4" ]; then echo "  PASS: $1 ($3=$got)"; PASS=$((PASS+1))
  else echo "  FAIL: $1 — $3 expected $4, got $got"; FAIL=$((FAIL+1)); fi
}

echo "TEST 8.1: pre-dirty + no-op pi -> empty_worktree (rc 10)"
run_pd 1 dirty ':'
check "pre-dirty no-op" 10 $PDRC empty_worktree "$PDV"
expect_field "pre-dirty no-op predirty_files" "$PDV" predirty_files 1
expect_field "pre-dirty no-op changed_since_start" "$PDV" worktree_changed_since_start false
V1="$PDV"

echo "TEST 8.2: pre-dirty + new untracked file -> ok"
run_pd 2 dirty 'echo n > new.txt'
check "pre-dirty new file" 0 $PDRC ok "$PDV"
expect_field "pre-dirty new file changed_since_start" "$PDV" worktree_changed_since_start true

echo "TEST 8.3: pre-dirty + further edit to the already-dirty file -> ok"
run_pd 3 dirty 'echo changed2 > f.txt'
check "pre-dirty further edit" 0 $PDRC ok "$PDV"
expect_field "J: arm 1 predirty_files" "$V1" predirty_files 1
expect_field "J: arm 3 predirty_files" "$PDV" predirty_files 1
expect_field "J: arm 3 changed_since_start" "$PDV" worktree_changed_since_start true

echo "TEST 8.4: clean + no-op pi (control) -> empty_worktree"
run_pd 4 clean ':'
check "clean no-op" 10 $PDRC empty_worktree "$PDV"
expect_field "clean no-op predirty_files" "$PDV" predirty_files 0

echo "TEST 8.5: clean + committing pi (control) -> ok via commits"
run_pd 5 clean 'echo c > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm c'
check "clean commit" 0 $PDRC ok "$PDV"
expect_field "clean commit commits_since_start" "$PDV" commits_since_start 1
expect_field "clean commit changed_since_start" "$PDV" worktree_changed_since_start false

echo "TEST 8.6: pre-dirty untracked file edited in place -> ok"
run_pd 6 dirty-untracked 'echo u1 > u.txt'
check "untracked edit" 0 $PDRC ok "$PDV"

echo "TEST 8.7: pre-dirty + pi reverts the dirty edit -> ok (decision: the tree did change)"
run_pd 7 dirty 'git checkout -- f.txt'
check "revert of pre-dirty edit" 0 $PDRC ok "$PDV"
expect_field "revert worktree_changed_files" "$PDV" worktree_changed_files 0
expect_field "revert predirty_files" "$PDV" predirty_files 1

echo
echo "passed=$PASS failed=$FAIL"
[ "$FAIL" -eq 0 ]
