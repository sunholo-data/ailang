#!/bin/bash
# Stubbed-channel tests over the REAL emit blocks in mission-control.sh.
# The blocks are awk-extracted from the file, never retyped: a retyped copy tests the copy.
set -uo pipefail

SP="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DRV="$REPO_ROOT/tools/launchd/mission-control.sh"
LAB="${TMPDIR:-/tmp}/emitlab.$$"; rm -rf "$LAB"; mkdir -p "$LAB"

awk '/^_mc_notify\(\) \{/,/^\}/' "$DRV"                        > "$LAB/notify.sh"
awk '/^if \[ -n "\$_pin_degraded" \]; then/,/^fi$/' "$DRV"     > "$LAB/pin_block.sh"
awk '/^if \[ -n "\$_lane_degraded" \]; then/,/^fi$/' "$DRV"    > "$LAB/lane_block.sh"

for f in notify pin_block lane_block; do
  if [ ! -s "$LAB/$f.sh" ]; then echo "FATAL: extraction of $f produced nothing"; exit 1; fi
done
echo "extracted: notify=$(wc -l < "$LAB/notify.sh") pin=$(wc -l < "$LAB/pin_block.sh") lane=$(wc -l < "$LAB/lane_block.sh") lines"

PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; echo "        trace: $2"; }
check(){ case "$2" in *"$3"*) ok "$1";; *) bad "$1" "$(printf '%s' "$2"|tr '\n' '|')";; esac; }
checkno(){ case "$2" in *"$3"*) bad "$1" "$(printf '%s' "$2"|tr '\n' '|')";; *) ok "$1";; esac; }

run() { # $1=block  $2=degraded-value  -> prints trace; env AILANG_RC/GH_RC/ISSUE tweak it
  local block="$1" val="$2"
  /bin/bash -c '
    set -uo pipefail
    TRACE=""
    log() { TRACE="$TRACE
LOG:$*"; }
    ailang() { TRACE="$TRACE
AILANG:$*"; return ${AILANG_RC:-0}; }
    gh()     { TRACE="$TRACE
GH:$*"; return ${GH_RC:-0}; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE="${ISSUE-635}"; LOG=/tmp/x.log; REPO=/tmp/repo
    MODEL=claude-opus-5; MODEL_WHY="probe ok"
    MISSION_DESIGNER_MODEL=d; MISSION_PLANNER_MODEL=p; MISSION_EXECUTOR_MODEL=e; MISSION_EVALUATOR_MODEL=v
    _pin_degraded=""; _lane_degraded=""
    . "$1"            # _mc_notify
    eval "$3=\"$4\""  # set the ledger under test
    . "$2"            # the block
    printf "%s" "$TRACE"
    echo "
RC:$?"
  ' _ "$LAB/notify.sh" "$LAB/$block.sh" \
    "$( [ "$block" = pin_block ] && echo _pin_degraded || echo _lane_degraded )" "$val" 2>&1
}

echo "== pin notice =="
T=$(run pin_block "- driver pin: FAILED")
check "fires on both channels (ailang)"     "$T" "AILANG:messages send controlplane"
check "posts to the bookkeeping issue"      "$T" "GH:issue comment 635"
check "titled as UNPINNED"                  "$T" "driver ran UNPINNED"
check "names the tracking issue"            "$T" "#558"

echo "== pin notice: SILENT when healthy (the control) =="
T=$(run pin_block "")
checkno "no ailang call"                    "$T" "AILANG:"
checkno "no gh call"                        "$T" "GH:"

echo "== failed post is LOUD, and never aborts =="
T=$(AILANG_RC=1 run pin_block "- x")
check "warns on send failure"               "$T" "LOG:WARNING: driver-pin notice FAILED to send"
check "block still exits 0"                 "$T" "RC:0"
T=$(GH_RC=1 run pin_block "- x")
check "warns on issue failure"              "$T" "LOG:WARNING: driver-pin notice FAILED to post"
check "block still exits 0"                 "$T" "RC:0"

echo "== unset issue warns rather than silently skipping =="
T=$(ISSUE= run pin_block "- x")
check "warns on unset issue"                "$T" "MISSION_GH_ISSUE is unset"
checkno "no gh call attempted"              "$T" "GH:issue"

echo "== REGRESSION: lane block still works through the extracted _mc_notify =="
T=$(run lane_block "- codex lane down")
check "lane fires on ailang"                "$T" "AILANG:messages send controlplane"
check "lane posts to the issue"             "$T" "GH:issue comment 635"
check "lane keeps its own title"            "$T" "executor/planner lane degraded"
check "lane logs its summary"               "$T" "LOG:LANE DEGRADED this fire"
T=$(run lane_block "")
checkno "lane SILENT when healthy"          "$T" "AILANG:"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
