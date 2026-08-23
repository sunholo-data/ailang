#!/bin/bash
# Stubbed-channel tests over the REAL emit blocks in mission-control.sh.
# The blocks are awk-extracted from the file, never retyped: a retyped copy tests the copy.
set -uo pipefail

SP="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DRV="$REPO_ROOT/tools/launchd/mission-control.sh"
LAB="${TMPDIR:-/tmp}/emitlab.$$"; rm -rf "$LAB"; mkdir -p "$LAB"

awk '/^_mc_notify\(\) \{/,/^\}/' "$DRV"                        > "$LAB/notify.sh"
awk '/^# --- DRIVER PIN DECISION START ---/,/^# --- DRIVER PIN DECISION END ---/' "$DRV" > "$LAB/pin_decision.sh"
awk '/^if \[ -n "\$_pin_degraded" \]; then/,/^fi$/' "$DRV"     > "$LAB/pin_block.sh"
awk '/^if \[ -n "\$_pin_drift_degraded" \]; then/,/^fi$/' "$DRV" > "$LAB/pin_drift_block.sh"
awk '/^if \[ -n "\$_lane_degraded" \]; then/,/^fi$/' "$DRV"    > "$LAB/lane_block.sh"

for f in notify pin_decision pin_block pin_drift_block lane_block; do
  if [ ! -s "$LAB/$f.sh" ]; then echo "FATAL: extraction of $f produced nothing"; exit 1; fi
done
echo "extracted: notify=$(wc -l < "$LAB/notify.sh") pin-decision=$(wc -l < "$LAB/pin_decision.sh") pin=$(wc -l < "$LAB/pin_block.sh") pin-drift=$(wc -l < "$LAB/pin_drift_block.sh") lane=$(wc -l < "$LAB/lane_block.sh") lines"

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

run_drift() { # $1=status $2=drift $3=threshold $4=state-value (absent for no file)
  local state_dir="$LAB/state.$$.${RUN_SEQ:-0}"
  RUN_SEQ=$((${RUN_SEQ:-0}+1))
  rm -rf "$state_dir"; mkdir -p "$state_dir"
  if [ "$4" != absent ]; then printf '%s\n' "$4" > "$state_dir/pin-drift"; fi
  /bin/bash -c '
    set -uo pipefail
    TRACE=""
    log() { TRACE="$TRACE
LOG:$*"; }
    ailang() { TRACE="$TRACE
AILANG:$*"; return 0; }
    gh()     { TRACE="$TRACE
GH:$*"; return 0; }
    MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
    AILANG_DRIVER_SRC=/source/ailang-motoko
    PIN_STATUS="$1"; PIN_DRIFT="$2"; AILANG_DRIVER_DRIFT_WARN="$3"
    PIN_DRIFT_FILE="$4/pin-drift"; PIN_NOTE="pinned test note"
    _pin_degraded=""; _pin_drift_degraded=""
    . "$5"
    . "$6"
    echo "DECISION_RC:$?"
    . "$7"
    . "$8"
    if [ -f "$PIN_DRIFT_FILE" ]; then
      echo "STATE:$(cat "$PIN_DRIFT_FILE")"
    else
      echo "STATE:absent"
    fi
    printf "%s" "$TRACE"
  ' _ "$1" "$2" "$3" "$state_dir" "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_drift_block.sh" "$LAB/pin_block.sh" 2>&1
  rm -rf "$state_dir"
}

arm_ok() { # $1=name $2=trace $3=required substring; remaining args are forbidden substrings
  local name="$1" trace="$2" required="$3" forbidden
  shift 3
  case "$trace" in *"DECISION_RC:0"*) ;; *) bad "$name" "$(printf '%s' "$trace"|tr '\n' '|')"; return;; esac
  case "$trace" in *"$required"*) ;; *) bad "$name" "$(printf '%s' "$trace"|tr '\n' '|')"; return;; esac
  for forbidden in "$@"; do
    case "$trace" in *"$forbidden"*) bad "$name" "$(printf '%s' "$trace"|tr '\n' '|')"; return;; esac
  done
  ok "$name"
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

echo "== held pin source-clone drift =="
T=$(run_drift pinned 170 25 absent)
case "$T" in *"DECISION_RC:0"*"AILANG:messages send controlplane"*"GH:issue comment 635"*"/source/ailang-motoko"*"170"*) ok "drift-a: first threshold notice reaches both channels with path/count";; *) bad "drift-a: first threshold notice reaches both channels with path/count" "$(printf '%s' "$T"|tr '\n' '|')";; esac
T=$(run_drift pinned 170 25 170)
arm_ok "drift-b: equal state dedupes" "$T" "STATE:170" "AILANG:" "GH:"
T=$(run_drift pinned 340 25 170)
arm_ok "drift-c: doubling notifies" "$T" "AILANG:messages send controlplane" "STATE:170"
T=$(run_drift pinned 3 25 170)
arm_ok "drift-d: below threshold is silent and re-arms" "$T" "STATE:absent" "AILANG:" "GH:"
T=$(run_drift disabled 170 25 absent)
arm_ok "drift-e: disabled pin is silent" "$T" "STATE:absent" "AILANG:" "GH:"
T=$(run_drift pinned '?' 25 absent)
arm_ok "drift-f: unknown drift is log-only" "$T" "LOG:driver pin drift: unknown" "AILANG:" "GH:"
T=$(run_drift STALE 170 25 absent)
case "$T" in *"DECISION_RC:0"*"driver ran UNPINNED"*) case "$T" in *"source clone drifted"*) bad "drift-g: STALE keeps original notice only" "$(printf '%s' "$T"|tr '\n' '|')";; *) ok "drift-g: STALE keeps original notice only";; esac;; *) bad "drift-g: STALE keeps original notice only" "$(printf '%s' "$T"|tr '\n' '|')";; esac

# drift-h: the body must name the SOURCE CLONE, never $REPO. On the pinned pass pin-root.sh has
# already exported MISSION_WORKDIR=<pin worktree>, and REPO is derived from it — so a body built
# from $REPO tells the human to reconcile a detached throwaway whose drift is 0 by construction.
# Measured live 2026-08-23: MISSION_WORKDIR=/Users/.../.ailang-driver-pin/motoko while
# AILANG_DRIVER_SRC=/Users/.../dev/sunholo-data/ailang-motoko at 170 behind.
T=$(run_drift pinned 170 25 absent)
arm_ok "drift-h: notice names the source clone, not the pin worktree" "$T" "/source/ailang-motoko" "/pinned/driver-worktree"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
