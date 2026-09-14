#!/bin/bash
# Stubbed-channel tests over the REAL emit blocks in mission-control.sh.
# The blocks are awk-extracted from the file, never retyped: a retyped copy tests the copy.
set -uo pipefail

SP="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DRV="$REPO_ROOT/tools/launchd/mission-control.sh"
LAB="${TMPDIR:-/tmp}/emitlab.$$"; rm -rf "$LAB"; mkdir -p "$LAB" "$LAB/bin"

# --- M1: suite-owned PATH notifier stubs ---------------------------------------
# _mc_bounded runs `( exec "$@" )`; exec resolves EXTERNAL binaries only — a shell
# function is invisible to it (measured: `( exec f )` on a function -> rc=127). So
# the notifier stubs MUST be PATH executables, or the bounded production sends would
# exec the REAL ailang/gh and hit prod Firestore + GitHub from CI (forbidden). Each
# stub appends one ordered record to $MC_TRACE_FILE (per-arm, unified medium G1),
# honours AILANG_RC/GH_RC, and the ailang one prints the store/project env it saw so
# the child's env is observable (V4). STUB_HANG_AILANG / STUB_HANG_GH=1 select a
# never-returning variant for the bounded-cutoff arms; STUB_HANG_AILANG still counts an
# attempt first so the retry-3x/spool arm can assert attempts via the file counter even
# though the send never returns.
cat > "$LAB/bin/ailang" <<'STUB'
#!/bin/bash
if [ "${STUB_HANG:-0}" = "1" ] || [ "${STUB_HANG_AILANG:-0}" = "1" ]; then
  if [ -n "${MC_ATTEMPT_FILE:-}" ]; then
    n=0
    [ -f "$MC_ATTEMPT_FILE" ] && n=$(cat "$MC_ATTEMPT_FILE" 2>/dev/null || echo 0)
    n=$(( 10#$n + 1 ))
    printf '%s\n' "$n" > "$MC_ATTEMPT_FILE"
  fi
  while :; do sleep 1; done
fi
if [ -n "${MC_TRACE_FILE:-}" ]; then
  printf 'AILANG:%s\n' "$*" >> "$MC_TRACE_FILE"
fi
if [ -n "${MC_ATTEMPT_FILE:-}" ]; then
  n=0
  [ -f "$MC_ATTEMPT_FILE" ] && n=$(cat "$MC_ATTEMPT_FILE" 2>/dev/null || echo 0)
  n=$(( 10#$n + 1 ))
  printf '%s\n' "$n" > "$MC_ATTEMPT_FILE"
fi
printf 'store=<%s> proj=<%s>\n' "${AILANG_MESSAGES_STORE:-}" "${AILANG_MESSAGES_PROJECT:-}"
exit "${AILANG_RC:-0}"
STUB
chmod +x "$LAB/bin/ailang"

cat > "$LAB/bin/gh" <<'STUB'
#!/bin/bash
if [ "${STUB_HANG_GH:-0}" = "1" ]; then
  while :; do sleep 1; done
fi
if [ -n "${MC_TRACE_FILE:-}" ]; then
  printf 'GH:%s\n' "$*" >> "$MC_TRACE_FILE"
fi
exit "${GH_RC:-0}"
STUB
chmod +x "$LAB/bin/gh"

export PATH="$LAB/bin${PATH:+:$PATH}"

# HERMETICITY (evaluator B2): `_mc_bounded` runs `( exec "$@" )`, and a subshell inherits
# whatever the CALLING shell exported — independent of any `env VAR=val` prefix. CLAUDE.md
# tells every machine doing AILANG work to export AILANG_MESSAGES_STORE/PROJECT at session
# start, so in the documented operating environment the store/project guard below would stay
# green with the production `env` prefix DELETED. Measured: mutation green 37/0 with the vars
# ambient, red 36/1 without. Unset them here so the prefix is the only possible source.
unset AILANG_MESSAGES_STORE AILANG_MESSAGES_PROJECT

awk '/^_mc_notify\(\) \{/,/^\}/' "$DRV"                        > "$LAB/notify.sh"
awk '/^# --- DRIVER PIN DECISION START ---/,/^# --- DRIVER PIN DECISION END ---/' "$DRV" > "$LAB/pin_decision.sh"
awk '/^if \[ -n "\$_pin_degraded" \]; then/,/^fi$/' "$DRV"     > "$LAB/pin_block.sh"
awk '/^if \[ -n "\$_pin_drift_degraded" \]; then/,/^fi$/' "$DRV" > "$LAB/pin_drift_block.sh"
awk '/^if \[ -n "\$_pin_age_degraded" \]; then/,/^fi$/' "$DRV"    > "$LAB/pin_age_block.sh"
awk '/^if \[ -n "\$_lane_degraded" \]; then/,/^fi$/' "$DRV"    > "$LAB/lane_block.sh"
# M2: the age consumer (state paths + the age-degrade notice) must be awk-extracted by
# anchor too, so a retyped copy is never what the arms test (extraction guard below).
awk '/^# --- DRIVER PIN STATE PATHS START ---/,/^# --- DRIVER PIN STATE PATHS END ---/' "$DRV" > "$LAB/state_paths.sh"
# M2 fixture: bound the three D-60 paths with the REAL helpers the production wraps
# use, so the hang-cutoff and drain arms exercise the production code, not a retype.
awk '/^_mc_bounded\(\) \{/,/^\}/' "$DRV"                    > "$LAB/bounded.sh"
awk '/^_mc_drain_notices\(\) \{/,/^\}/' "$DRV"              > "$LAB/drain.sh"

export MC_BND="$LAB/bounded.sh"

for f in notify pin_decision pin_block pin_drift_block pin_age_block lane_block state_paths bounded drain; do
  if [ ! -s "$LAB/$f.sh" ]; then echo "FATAL: extraction of $f produced nothing"; exit 1; fi
done
echo "extracted: notify=$(wc -l < "$LAB/notify.sh") pin-decision=$(wc -l < "$LAB/pin_decision.sh") pin=$(wc -l < "$LAB/pin_block.sh") pin-drift=$(wc -l < "$LAB/pin_drift_block.sh") pin-age=$(wc -l < "$LAB/pin_age_block.sh") lane=$(wc -l < "$LAB/lane_block.sh") state-paths=$(wc -l < "$LAB/state_paths.sh") bounded=$(wc -l < "$LAB/bounded.sh") drain=$(wc -l < "$LAB/drain.sh") lines"

PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; echo "        trace: $2"; }
check(){ case "$2" in *"$3"*) ok "$1";; *) bad "$1" "$(printf '%s' "$2"|tr '\n' '|')";; esac; }
checkno(){ case "$2" in *"$3"*) bad "$1" "$(printf '%s' "$2"|tr '\n' '|')";; *) ok "$1";; esac; }

run() { # $1=block  $2=degraded-value  -> prints trace; env AILANG_RC/GH_RC/ISSUE tweak it
  local block="$1" val="$2"
  # A FRESH state dir per arm, not a shared one: the blocks now episode-GATE on files under
  # STATE_DIR, so two arms sharing it would let the first arm's episode marker silently
  # suppress the second's notice — an arm passing because it was deduped, not because the
  # code is right.
  local state_dir; state_dir=$(mktemp -d)
  local mc_trace; mc_trace=$(mktemp)
  local attempt_file; attempt_file=$(mktemp)
  MC_TRACE_FILE="$mc_trace" MC_ATTEMPT_FILE="$attempt_file" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"   # _mc_bounded (M2 production wraps the D-60 sends)
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    # log() stays a function (called directly, never through _mc_bounded) but appends to
    # the SAME unified medium the PATH stubs write, so positive AND negative checks both
    # read one trace (G1). The old split-trace medium lost AILANG records inside the
    # command-substitution subshell, so checkno assertions could pass vacuously.
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE="${ISSUE-635}"; LOG=/tmp/x.log; REPO=/tmp/repo
    MODEL=claude-opus-5; MODEL_WHY="probe ok"
    MISSION_DESIGNER_MODEL=d; MISSION_PLANNER_MODEL=p; MISSION_EXECUTOR_MODEL=e; MISSION_EVALUATOR_MODEL=v
    STATE_DIR="$5"   # episode gating (_lane_ep/_pin_ep) reads this; unbound => set -u abort
    _pin_degraded=""; _lane_degraded=""
    . "$1"            # _mc_notify
    eval "$3=\"$4\""  # set the ledger under test
    if [ "${MC_RUN_UNSET_STATE:-0}" = "1" ]; then
      # B1 guard (evaluator finding): every other arm runs a block that exits 0, so the
      # OWN capture position of run() was untested and the G2 defect could come back silently.
      # Unsetting STATE_DIR makes the block abort under `set -u`; the subshell keeps that
      # abort from killing this script, so a non-zero rc must reach _blk_rc.
      ( unset STATE_DIR; . "$2" )
    else
      . "$2"          # the block
    fi
    _blk_rc=$?        # capture rc BEFORE any printf (G2: the old `$?` after `printf`
                      # read the rc that printf left behind, masking the block failure)
    TRACE="$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    printf "%s" "$TRACE"
    printf "\nATTEMPTS:%s" "$(cat "$MC_ATTEMPT_FILE" 2>/dev/null || printf 0)"
    printf "\nRC:%s\n" "$_blk_rc"
  ' _ "$LAB/notify.sh" "$LAB/$block.sh" \
    "$( [ "$block" = pin_block ] && echo _pin_degraded || echo _lane_degraded )" "$val" "$state_dir" 2>&1
  rm -rf "$state_dir" "$mc_trace" "$attempt_file"
}

run_drift() { # $1=status $2=drift $3=threshold $4=state-value (absent for no file)
  local state_dir="$LAB/state.$$.${RUN_SEQ:-0}"
  RUN_SEQ=$((${RUN_SEQ:-0}+1))
  rm -rf "$state_dir"; mkdir -p "$state_dir"
  if [ "$4" != absent ]; then printf '%s\n' "$4" > "$state_dir/pin-drift"; fi
  local mc_trace; mc_trace=$(mktemp)
  local attempt_file; attempt_file=$(mktemp)
  MC_TRACE_FILE="$mc_trace" MC_ATTEMPT_FILE="$attempt_file" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"   # _mc_bounded
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
    AILANG_DRIVER_SRC=/source/ailang-motoko
    PIN_STATUS="$1"; PIN_DRIFT="$2"; AILANG_DRIVER_DRIFT_WARN="$3"
    PIN_DRIFT_FILE="$4/pin-drift"; PIN_NOTE="pinned test note"
    # M2 neutral: the age consumer is sourced on EVERY drift arm too (its inputs are
    # supplied to the same $4 state dir) but PIN_AGE=0 keeps it below any warn threshold,
    # so no existing drift arm changes its inputs, its sends, or its PASS name (D-3).
    PIN_AGE=0; PIN_AGE_FILE="$4/pin-age"
    # The extracted blocks read STATE_DIR (episode gating: _lane_ep, _pin_ep). The lab
    # supplies every other driver variable but never this one, so under `set -u` the block
    # aborts on an unbound variable before any assertion runs — 9 arms here failing for the
    # harness rather than for the code. Point it at the same per-arm temp dir as $4.
    STATE_DIR="$4"
    _pin_degraded=""; _pin_drift_degraded=""
    . "$5"
    . "$6"
    echo "DECISION_RC:$?"
    . "$7"            # pin_drift_block
    . "$8"            # pin_age_block (sourced between drift and pin; PIN_AGE=0 neutral)
    . "$9"            # pin_block
    TRACE="$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    if [ -f "$PIN_DRIFT_FILE" ]; then
      echo "STATE:$(cat "$PIN_DRIFT_FILE")"
    else
      echo "STATE:absent"
    fi
    printf "%s" "$TRACE"
  ' _ "$1" "$2" "$3" "$state_dir" "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_drift_block.sh" "$LAB/pin_age_block.sh" "$LAB/pin_block.sh" 2>&1
  rm -rf "$state_dir" "$mc_trace" "$attempt_file"
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

# --- M2: PIN_AGE consumer harnesses ---------------------------------------------
# run_age: modeled on run_drift with the same discipline — per-arm mktemp -d state,
# stubbed channels, the extracted blocks sourced from $LAB, DECISION_RC surfaced, and
# the AGE state file echoed. $1=status $2=age $3=warn ("" leaves AILANG_DRIVER_AGE_WARN
# genuinely unset) $4=age-state-value (absent = no pre-seeded state file). Identity
# fixtures come from AGE_REF/AGE_SHA/AGE_BASE (defaults match the drift harness), so the
# age-q arm can rebind the pinned ref to origin/feature by setting AGE_REF.
run_age() {
  local state_dir="$LAB/age.$$.${RUN_SEQ:-0}"
  RUN_SEQ=$((${RUN_SEQ:-0}+1))
  rm -rf "$state_dir"; mkdir -p "$state_dir"
  if [ "$4" != absent ]; then printf '%s\n' "$4" > "$state_dir/pin-age"; fi
  local mc_trace; mc_trace=$(mktemp)
  MC_TRACE_FILE="$mc_trace" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"   # _mc_bounded
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
    AILANG_DRIVER_SRC=/source/ailang-motoko
    PIN_STATUS="$1"; PIN_AGE="$2"
    if [ -n "$3" ]; then AILANG_DRIVER_AGE_WARN="$3"; else unset AILANG_DRIVER_AGE_WARN; fi
    PIN_AGE_FILE="$4/pin-age"; PIN_DRIFT=0; PIN_DRIFT_FILE="$4/pin-drift"; PIN_NOTE="pinned test note"
    STATE_DIR="$4"
    AILANG_DRIVER_REF="${AGE_REF:-origin/dev}"
    AILANG_DRIVER_PINNED="${AGE_SHA:-deadbee}"
    AILANG_DRIVER_AGE_BASE_SHA="${AGE_BASE:-<40-hex-fixture>}"
    _pin_degraded=""; _pin_drift_degraded=""; _pin_age_degraded=""
    . "$5"   # notify
    . "$6"   # pin_decision
    echo "DECISION_RC:$?"
    . "$7"   # pin_age_block
    . "$8"   # pin_block (so the STALE arm can prove the ORIGINAL pin-failure notice survives)
    TRACE="$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    if [ -f "$PIN_AGE_FILE" ]; then
      echo "AGE_STATE:$(cat "$PIN_AGE_FILE")"
    else
      echo "AGE_STATE:absent"
    fi
    printf "%s" "$TRACE"
  ' _ "$1" "$2" "$3" "$state_dir" "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_age_block.sh" "$LAB/pin_block.sh" 2>&1
  rm -rf "$state_dir" "$mc_trace"
}

# run_both: AC-L (age-independent) — ONE shared mktemp -d STATE_DIR across four sequential
# fires (drift,age), each sourcing notify+decision+drift-block+age-block into its OWN fresh
# MC_TRACE_FILE (so sends are attributable to a single fire), echoed as fire-delimited
# blocks so the four assertions can scope state files AND per-type sends per step.
run_both() { # $1..$8 = four (drift,age) pairs
  local state_dir; state_dir=$(mktemp -d)
  local out; out=$(mktemp)
  local i=1
  while [ $i -le 4 ]; do
    eval "local _d=\"\$$((i*2-1))\""
    eval "local _a=\"\$$((i*2))\""
    local t; t=$(mktemp)
    printf "===FIRE:%s===\n" "$i" >> "$out"
    MC_TRACE_FILE="$t" /bin/bash -c '
      set -uo pipefail
      . "$MC_BND"
      NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
      log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
      MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
      MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
      AILANG_DRIVER_SRC=/source/ailang-motoko
      PIN_STATUS=pinned; PIN_DRIFT="$1"; PIN_AGE="$2"
      AILANG_DRIVER_DRIFT_WARN=25; AILANG_DRIVER_AGE_WARN=25
      PIN_DRIFT_FILE="$3/pin-drift"; PIN_AGE_FILE="$3/pin-age"; PIN_NOTE="pinned test note"
      STATE_DIR="$3"
      AILANG_DRIVER_REF=origin/dev; AILANG_DRIVER_PINNED=deadbee; AILANG_DRIVER_AGE_BASE_SHA=fixture
      _pin_degraded=""; _pin_drift_degraded=""; _pin_age_degraded=""
      . "$4"   # notify
      . "$5"   # pin_decision
      . "$6"   # pin_drift_block
      . "$7"   # pin_age_block
      printf "DRIFT_STATE:%s\n" "$( [ -f "$3/pin-drift" ] && head -1 "$3/pin-drift" || printf absent )"
      printf "AGE_STATE:%s\n" "$( [ -f "$3/pin-age" ] && head -1 "$3/pin-age" || printf absent )"
      cat "$MC_TRACE_FILE" 2>/dev/null || true
    ' _ "$_d" "$_a" "$state_dir" "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_drift_block.sh" "$LAB/pin_age_block.sh" >> "$out"
    i=$((i+1))
  done
  printf "===END===\n" >> "$out"
  cat "$out"
  rm -rf "$state_dir" "$out"
}

# _fireblock $output $fire# — print the block for a single fire, delimited by the next
# FIRE marker (or ==END== for the last).
_fireblock() {
  local out="$1" f="$2" start end
  start="===FIRE:$f==="
  if [ "$f" -ge 4 ]; then end="===END==="; else end="===FIRE:$((f+1))==="; fi
  awk -v s="$start" -v e="$end" 'index($0,s){p=1;next} p&&index($0,e){exit} p' <<<"$out"
}

echo "== pin notice =="
T=$(run pin_block "- driver pin: FAILED")
check "fires on both channels (ailang)"     "$T" "AILANG:messages send controlplane"
check "posts to the bookkeeping issue"      "$T" "GH:issue comment 635"
check "titled as UNPINNED"                  "$T" "driver ran UNPINNED"
check "names the tracking issue"            "$T" "#558"
# M1 G1 tripwire: this fires genuinely, so the SAME "no ailang call" checkno assertions
# below MUST be able to see AILANG: here — proves they read the unified medium.
check "positive control: unified medium shows genuine AILANG fire (checkno would FAIL)" "$T" "AILANG:"

echo "== pin notice: SILENT when healthy (the control) =="
T=$(run pin_block "")
checkno "no ailang call"                    "$T" "AILANG:"
checkno "no gh call"                        "$T" "GH:"

echo "== failed post is LOUD, and never aborts =="
T=$(AILANG_RC=1 run pin_block "- x")
check "warns on send failure"               "$T" "LOG:WARNING: driver-pin notice FAILED to send"
check "block still exits 0"                 "$T" "RC:0"
check "retry: attempts recorded across subshells (file counter)" "$T" "ATTEMPTS:3"
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

echo "== EPISODE GATING: suppression + recovery reset (#978) =="
# #978: both dedupe guards (_lane_ep/_pin_ep fingerprints over a digit-stripped
# ledger) had NO test at any level — both guards could be deleted with the suite
# still green. These arms pin, against the EXTRACTED production blocks:
#   (a) the second identical notice is suppressed (sends stay at 1) and the
#       suppression is logged,
#   (b) a CHANGED reason re-announces,
#   (c) healing clears the marker so the NEXT episode announces again,
#   (d) the fingerprint strips digits, so a moving counter cannot defeat the gate.
run_episode_seq() { # $1=block; remaining args = ledger values, in fire order ("" = healed)
  local block="$1"; shift
  local state_dir="$LAB/epstate.$$.$block"
  rm -rf "$state_dir"; mkdir -p "$state_dir"
  local mc_trace; mc_trace=$(mktemp)
  local attempt_file; attempt_file=$(mktemp)
  MC_TRACE_FILE="$mc_trace" MC_ATTEMPT_FILE="$attempt_file" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/tmp/repo
    MODEL=claude-opus-5; MODEL_WHY="probe ok"
    MISSION_DESIGNER_MODEL=d; MISSION_PLANNER_MODEL=p; MISSION_EXECUTOR_MODEL=e; MISSION_EVALUATOR_MODEL=v
    STATE_DIR="$1"   # episode gating (_lane_ep/_pin_ep) reads this
    _pin_degraded=""; _lane_degraded=""
    . "$2"            # _mc_notify
    ledger_var=$( [ "$3" = pin_block ] && echo _pin_degraded || echo _lane_degraded )
    ep_file="$STATE_DIR/mission-v1-$( [ "$3" = pin_block ] && echo driver-pin || echo lane-degraded ).episode"
    # positional args: $1=STATE_DIR $2=notify.sh $3=block name $4=block path
    # $5.. = the ledger values, one per fire
    phase=0
    for val in "${@:5}"; do
      phase=$((phase+1))
      : > "$MC_TRACE_FILE"   # per-phase medium: each phase gets its own send count
      eval "$ledger_var=\"\$val\""
      . "$4"                 # the REAL extracted block
      n=$(grep -c "^AILANG:messages send controlplane" "$MC_TRACE_FILE" 2>/dev/null || true); n=${n:-0}
      if [ -f "$ep_file" ]; then m=present; else m=absent; fi
      suppressed=no
      case "$(cat "$MC_TRACE_FILE" 2>/dev/null)" in *"unchanged this episode"*|*"notice suppressed"*) suppressed=yes;; esac
      printf "PHASE%s:sends=%s marker=%s suppressed=%s\n" "$phase" "$n" "$m" "$suppressed"
    done
  ' _ "$state_dir" "$LAB/notify.sh" "$block" "$LAB/$block.sh" "$@" 2>&1
  rm -rf "$state_dir" "$mc_trace" "$attempt_file"
}

T=$(run_episode_seq lane_block "- codex lane down" "- codex lane down" "- planner lane down" "")
check "lane ep: first fire announces"           "$T" "PHASE1:sends=1 marker=present"
check "lane ep: second identical fire suppressed" "$T" "PHASE2:sends=0 marker=present suppressed=yes"
check "lane ep: changed reason re-announces"    "$T" "PHASE3:sends=1 marker=present suppressed=no"
check "lane ep: heal clears the marker"         "$T" "PHASE4:sends=0 marker=absent"
checkno "lane ep: heal announces nothing"       "$T" "PHASE4:sends=1"

T=$(run_episode_seq lane_block "- codex lane down rc=17" "- codex lane down rc=99")
check "lane ep: moving rc (digit strip) does not re-announce" "$T" "PHASE2:sends=0 marker=present suppressed=yes"

T=$(run_episode_seq pin_block "- driver pin: FAILED" "- driver pin: FAILED" "- driver pin: FAILED (other clone)" "")
check "pin ep: first fire announces"            "$T" "PHASE1:sends=1 marker=present"
check "pin ep: second identical fire suppressed" "$T" "PHASE2:sends=0 marker=present suppressed=yes"
check "pin ep: changed reason re-announces"     "$T" "PHASE3:sends=1 marker=present suppressed=no"
check "pin ep: heal clears the marker"          "$T" "PHASE4:sends=0 marker=absent"

echo "== RC capture: not masked by printf (G2) =="
# Drop the STATE_DIR assignment; the sourced pin_block reads $STATE_DIR for spool/episode
# gating, so under `set -u` it aborts rc=1. With the old code the `$?`-after-`printf`
# masked this as RC:0; the capture-before-printf fix must surface RC:1.
_rc_trace=$(mktemp)
T=$(AILANG_RC=1 MC_TRACE_FILE="$_rc_trace" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"   # _mc_bounded
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/tmp/repo
    MODEL=claude-opus-5; MODEL_WHY="probe ok"
    # STATE_DIR deliberately NOT set => the block aborts under `set -u`. Source it in a
    # subshell so the abort returns rc=1 to us instead of killing this whole shell
    # before we can capture it.
    _pin_degraded="- x"; _lane_degraded="- x"
    . "$1"
    ( . "$2" )
    _rc=$?
    printf "RC:%s\n" "$_rc"
  ' _ "$LAB/notify.sh" "$LAB/pin_block.sh" 2>&1)
rm -f "$_rc_trace"
check "RC capture: block failure visible, not masked by printf" "$T" "RC:1"

# The check above exercises a standalone reimplementation. This one exercises run() ITSELF —
# the harness behind 20+ of this suite's assertions, and the only place the G2 fix actually
# ships. Without it, moving run()'s `_blk_rc=$?` back after the printfs leaves the suite
# fully green (measured 37/0 before this arm existed).
export MC_RUN_UNSET_STATE=1
_rc_run=$(run pin_block "- x")
unset MC_RUN_UNSET_STATE
check "RC capture: run() itself surfaces a failing block, not printf's status" "$_rc_run" "RC:1"

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

# drift-i: a threshold of 0 persists a previous of 0, after which `-ge $((0 * 2))` is TRUE on every
# fire — the post-every-90-minutes outcome the doubling rule exists to prevent, reached through the
# block's own config knob. The floor coerces it to 25, so a drift of 3 is BELOW threshold and stays
# silent. Remove the floor and this arm reds, because 3 -ge 0 notifies.
T=$(run_drift pinned 3 0 absent)
arm_ok "drift-i: a non-positive threshold is floored, not obeyed" "$T" "using 25" "AILANG:" "GH:"

# drift-j: PIN_DRIFT unset under `set -u` must be handled, not abort. Unreachable through
# pin-root.sh today (it sets PIN_STATUS and PIN_DRIFT on consecutive lines) — pinned here so the
# invariant is enforced rather than assumed, per the evaluator's finding 4.
_drj_trace=$(mktemp)
T=$(MC_TRACE_FILE="$_drj_trace" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"   # _mc_bounded
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
    AILANG_DRIVER_SRC=/source/ailang-motoko
    PIN_STATUS=pinned; PIN_DRIFT_FILE=/tmp/nonexistent.$$/pin-drift; PIN_NOTE=n
    _pin_degraded=""; _pin_drift_degraded=""
    . "$1"
    . "$2"
    echo "DECISION_RC:$?"
    . "$3"
    TRACE="$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    printf "%s" "$TRACE"
  ' _ "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_drift_block.sh" 2>&1)
rm -f "$_drj_trace"
arm_ok "drift-j: unset PIN_DRIFT is log-only, not a set -u abort" "$T" "LOG:driver pin drift: unknown" "AILANG:" "GH:"

# --- M2: PIN_AGE consumer arms (AC-E..M, AC-O, AC-Q notice) ---------------
echo "== age state paths =="
_apv=$(mktemp -d); _apm=$(mktemp -d)
STATE_DIR="$_apv" MISSION_NAME=v1 . "$LAB/state_paths.sh"
if [ "$PIN_AGE_FILE" = "$_apv/mission-control.pin-age" ]; then
  ok "age-paths-v1: v1 resolves mission-control.pin-age"
else
  bad "age-paths-v1: v1 resolves mission-control.pin-age" "got $PIN_AGE_FILE"
fi
if [ "$PIN_AGE_FILE" != "$PIN_DRIFT_FILE" ]; then
  ok "age-paths-v1: age path is distinct from the drift path"
else
  bad "age-paths-v1: age path is distinct from the drift path" "age=$PIN_AGE_FILE drift=$PIN_DRIFT_FILE"
fi
STATE_DIR="$_apm" MISSION_NAME=motoko . "$LAB/state_paths.sh"
if [ "$PIN_AGE_FILE" = "$_apm/mission-motoko.pin-age" ]; then
  ok "age-paths-mission: motoko resolves mission-motoko.pin-age"
else
  bad "age-paths-mission: motoko resolves mission-motoko.pin-age" "got $PIN_AGE_FILE"
fi
if [ "$PIN_AGE_FILE" != "$PIN_DRIFT_FILE" ]; then
  ok "age-paths-mission: age path is distinct from the drift path"
else
  bad "age-paths-mission: age path is distinct from the drift path" "age=$PIN_AGE_FILE drift=$PIN_DRIFT_FILE"
fi
rm -rf "$_apv" "$_apm"

echo "== age notification arms =="
T=$(run_age pinned 25 25 absent)
case "$T" in
  *"AILANG:messages send controlplane"*"driver pin is stale (25 behind)"*"GH:issue comment 635"*)
    ok "age-a: first threshold age notice reaches both channels with count";;
  *) bad "age-a: first threshold age notice reaches both channels with count" "$(printf '%s' "$T"|tr '\n' '|')";; esac
case "$T" in
  *"Pinned ref: origin/dev"*"Target SHA: deadbee"*"Baseline origin/dev SHA:"*)
    ok "age-a: notice carries pinned ref, target SHA and baseline SHA";;
  *) bad "age-a: notice carries pinned ref, target SHA and baseline SHA" "$(printf '%s' "$T"|tr '\n' '|')";; esac
case "$T" in
  *"executing code 25 commits behind origin/dev"*)
    ok "age-a: notice carries the executing-old-code wording";;
  *) bad "age-a: notice carries the executing-old-code wording" "$(printf '%s' "$T"|tr '\n' '|')";; esac
case "$T" in
  *"AGE_STATE:25"*) ok "age-a: age state stores 25";;
  *) bad "age-a: age state stores 25" "$(printf '%s' "$T"|tr '\n' '|')";; esac

T=$(run_age pinned 170 25 170)
arm_ok "age-b: equal age dedupes, state unchanged, no sends" "$T" "AGE_STATE:170" "AILANG:" "GH:"
case "$T" in *"deduped until doubling from 170"*) ok "age-b: dedupe is logged positively with the previous count";; *) bad "age-b: dedupe is logged positively with the previous count" "$(printf '%s' "$T"|tr '\n' '|')";; esac

T=$(run_age pinned 340 25 170)
case "$T" in *"AGE_STATE:340"*"AILANG:messages send controlplane"*"driver pin is stale (340 behind)"*"GH:issue comment 635"*) ok "age-c: doubling notifies both channels and stores 340";; *) bad "age-c: doubling notifies both channels and stores 340" "$(printf '%s' "$T"|tr '\n' '|')";; esac
T=$(run_age pinned 339 25 170)
arm_ok "age-c2: one below doubling (339) dedupes" "$T" "AGE_STATE:170" "AILANG:" "GH:"

T=$(run_age pinned 3 25 170)
arm_ok "age-d: below threshold re-arms and removes age state" "$T" "AGE_STATE:absent" "AILANG:" "GH:"
T=$(run_age pinned 25 25 absent)
case "$T" in *"AGE_STATE:25"*"AILANG:messages send controlplane"*"driver pin is stale (25 behind)"*"GH:issue comment 635"*) ok "age-d-followup: threshold age emits again after re-arm";; *) bad "age-d-followup: threshold age emits again after re-arm" "$(printf '%s' "$T"|tr '\n' '|')";; esac

T=$(run_age pinned '?' 25 170)
case "$T" in *"AGE_STATE:170"*) arm_ok "age-f: unknown age is log-only and preserves state" "$T" "LOG:driver pin age: unknown" "AILANG:" "GH:";; *) bad "age-f: unknown age is log-only and preserves state" "$(printf '%s' "$T"|tr '\n' '|')";; esac
T=$(run_age pinned abc 25 170)
case "$T" in *"AGE_STATE:170"*) arm_ok "age-f2: malformed age is log-only and preserves state" "$T" "LOG:driver pin age: unknown" "AILANG:" "GH:";; *) bad "age-f2: malformed age is log-only and preserves state" "$(printf '%s' "$T"|tr '\n' '|')";; esac

T=$(run_age pinned 3 0 absent)
check "age-i1: zero warn is floored loudly to 25" "$T" "AILANG_DRIVER_AGE_WARN='0' is not a positive integer; using 25"
T=$(run_age pinned 3 -1 absent)
check "age-i2: negative warn is floored loudly to 25" "$T" "AILANG_DRIVER_AGE_WARN='-1' is not a positive integer; using 25"
T=$(run_age pinned 3 abc absent)
check "age-i3: malformed warn is floored loudly to 25" "$T" "AILANG_DRIVER_AGE_WARN='abc' is not a positive integer; using 25"
T=$(run_age pinned 3 '' absent)
checkno "age-i4: unset warn defaults quietly (no floor log)" "$T" "is not a positive integer"
T=$(run_age pinned 3 2 absent)
case "$T" in *"AILANG:messages send controlplane"*"driver pin is stale (3 behind)"*) ok "age-i5: positive override below 25 stays valid (2 emits at age 3)";; *) bad "age-i5: positive override below 25 stays valid (2 emits at age 3)" "$(printf '%s' "$T"|tr '\n' '|')";; esac

# age-j: PIN_AGE genuinely unset under `set -u` (drift-j's standalone heredoc shape).
_agj_trace=$(mktemp)
T=$(MC_TRACE_FILE="$_agj_trace" /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=motoko; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-motoko
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/pinned/driver-worktree
    AILANG_DRIVER_SRC=/source/ailang-motoko
    PIN_STATUS=pinned; PIN_AGE_FILE=/tmp/nonexistent.$$/pin-age; PIN_NOTE=n
    _pin_degraded=""; _pin_drift_degraded=""; _pin_age_degraded=""
    . "$1"
    . "$2"
    echo "DECISION_RC:$?"
    . "$3"
    TRACE="$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    printf "%s" "$TRACE"
  ' _ "$LAB/notify.sh" "$LAB/pin_decision.sh" "$LAB/pin_age_block.sh" 2>&1)
rm -f "$_agj_trace"
arm_ok "age-j: unset PIN_AGE under set -u is log-only, not an abort" "$T" "LOG:driver pin age: unknown" "AILANG:" "GH:"

# previous state ABSENT on purpose: with prev=170 the doubling dedupe hid a deleted status guard
# (MUT-R survived the drill at iter-354); absent state makes the guard the only suppressor.
T=$(run_age STALE 30 25 absent)
case "$T" in *"AGE_STATE:absent"*"driver ran UNPINNED"*)
  case "$T" in *"driver pin is stale"*) bad "age-m1: STALE skips age with no pin-age send, state untouched" "$(printf '%s' "$T"|tr '\n' '|')";; *) ok "age-m1: STALE skips age with no pin-age send, state untouched";; esac;;
  *) bad "age-m1: STALE skips age with no pin-age send, state untouched" "$(printf '%s' "$T"|tr '\n' '|')";; esac
case "$T" in *"driver ran UNPINNED"*) ok "age-m2: the original pin-failure notice still works";; *) bad "age-m2: the original pin-failure notice still works" "$(printf '%s' "$T"|tr '\n' '|')";; esac
T=$(run_age disabled 30 25 absent)
arm_ok "age-m3: disabled skips age silently with state untouched" "$T" "AGE_STATE:absent" "AILANG:" "GH:"

# AC-L: four sequential fires over ONE shared STATE_DIR; both state files AND the per-type
# sends are asserted at EACH step (titles: `source clone drifted` vs `driver pin is stale`).
echo "== age-independent (AC-L) =="
T=$(run_both 170 30 170 60 340 60 340 3)
_fb=$(_fireblock "$T" 1)
case "$_fb" in *"DRIFT_STATE:170"*"AGE_STATE:30"*"pinned source clone drifted (170 behind)"*"driver pin is stale (30 behind)"*) ok "age-independent-1: drift 170 + age 30 emits both notices";; *) bad "age-independent-1: drift 170 + age 30 emits both notices" "$(printf '%s' "$_fb"|tr '\n' '|')";; esac
_fb=$(_fireblock "$T" 2)
case "$_fb" in *"DRIFT_STATE:170"*"AGE_STATE:60"*"driver pin is stale (60 behind)"*)
  case "$_fb" in *"source clone drifted (170 behind)"*) bad "age-independent-2: age 60 emits age only, drift state stays 170" "$(printf '%s' "$_fb"|tr '\n' '|')";; *) ok "age-independent-2: age 60 emits age only, drift state stays 170";; esac;;
  *) bad "age-independent-2: age 60 emits age only, drift state stays 170" "$(printf '%s' "$_fb"|tr '\n' '|')";; esac
_fb=$(_fireblock "$T" 3)
case "$_fb" in *"DRIFT_STATE:340"*"AGE_STATE:60"*"pinned source clone drifted (340 behind)"*)
  case "$_fb" in *"driver pin is stale (60 behind)"*) bad "age-independent-3: drift 340 emits drift only, age state stays 60" "$(printf '%s' "$_fb"|tr '\n' '|')";; *) ok "age-independent-3: drift 340 emits drift only, age state stays 60";; esac;;
  *) bad "age-independent-3: drift 340 emits drift only, age state stays 60" "$(printf '%s' "$_fb"|tr '\n' '|')";; esac
_fb=$(_fireblock "$T" 4)
case "$_fb" in *"DRIFT_STATE:340"*"AGE_STATE:absent"*)
  case "$_fb" in *"driver pin is stale"*|*"source clone drifted"*) bad "age-independent-4: age 3 removes age state only, drift state stays 340" "$(printf '%s' "$_fb"|tr '\n' '|')";; *) ok "age-independent-4: age 3 removes age state only, drift state stays 340";; esac;;
  *) bad "age-independent-4: age 3 removes age state only, drift state stays 340" "$(printf '%s' "$_fb"|tr '\n' '|')";; esac

echo "== age-q: ref binding =="
T=$(AGE_REF=origin/feature AGE_SHA=ffffffffffffffffffffffffffffffffffffffff AGE_BASE=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb run_age pinned 26 25 absent)
case "$T" in *"AILANG:messages send controlplane"*"GH:issue comment 635"*"origin/feature"*)
  case "$T" in *"Pinned ref: origin/dev"*|*"Pinned ref: \`origin/dev\`"*) bad "age-q: notice sends on both channels and names origin/feature, never a hardcoded origin/dev ref" "$(printf '%s' "$T"|tr '\n' '|')";; *) ok "age-q: notice sends on both channels and names origin/feature, never a hardcoded origin/dev ref";; esac;;
  *) bad "age-q: notice sends on both channels and names origin/feature, never a hardcoded origin/dev ref" "$(printf '%s' "$T"|tr '\n' '|')";; esac
case "$T" in *"AGE_STATE:26"*"Baseline origin/dev SHA: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"*) ok "age-q: baseline SHA is printed and age state stores 26";; *) bad "age-q: baseline SHA is printed and age state stores 26" "$(printf '%s' "$T"|tr '\n' '|')";; esac

# --- M2: bounded production D-60 paths -------------------------------------------
# run_bounded DEADLINE OUTPUTFILE CMD... — run CMD in a background subshell with an
# outer test-level deadline; SIGKILL the whole tree at the deadline and return 124.
# Bounded-wait discipline: a removed production bound must FAIL this arm cleanly
# rather than wedge CI (rule 7).
_run_bounded_killtree() {
  local p="$1" c
  for c in $(pgrep -P "$p" 2>/dev/null); do _run_bounded_killtree "$c"; done
  kill -9 "$p" 2>/dev/null
}
run_bounded() {
  local dl="$1" out="$2"; shift 2
  ( "$@" >"$out" 2>&1 ) &
  local pid=$!
  local outer=$(( $(date +%s) + dl ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$outer" ]; then _run_bounded_killtree "$pid"; return 124; fi
    sleep 1
  done
  wait "$pid" 2>/dev/null; return $?
}

# (1) single-shot _mc_bounded against a never-returning send stub: rc=124, <=7s.
# Uses the REAL _mc_bounded helper directly (production-independent), outer 15s.
_ss_trace=$(mktemp); _ss_outf=$(mktemp); _ss_beg=$(date +%s)
run_bounded 15 "$_ss_outf" env MC_TRACE_FILE="$_ss_trace" STUB_HANG_AILANG=1 \
  /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    _mc_bounded 2 env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
      ailang messages send controlplane "hang" --from mission-control
    echo "HANG_RC:$?"
  '
_ss_rc=$?; _ss_elapsed=$(( $(date +%s) - _ss_beg )); _ss_out="$(cat "$_ss_outf")"
if [ "$_ss_rc" = "124" ]; then
  bad "production: hanging direct send is cut off at the bound" "outer 15s watchdog tripped (bound removed)"
else
  case "$_ss_out" in
    *"HANG_RC:124"*) [ "$_ss_elapsed" -le 7 ] && ok "production: hanging direct send is cut off at the bound" \
      || bad "production: hanging direct send is cut off at the bound" "rc=124 but elapsed=${_ss_elapsed}s > 7s ($_ss_out)";;
    *) bad "production: hanging direct send is cut off at the bound" "expected HANG_RC:124, got: rc=$_ss_rc ($_ss_out)";; esac
fi
rm -f "$_ss_trace" "$_ss_outf"

# (2) full _mc_notify against a hanging direct send: 3 retries -> spool 1 row, GH still
# called once. Outer 60s. Also (3) carries the synthesised rc=124 diagnostic (G5).
_ms_spool=$(mktemp -d); _ms_att=$(mktemp); _ms_tr=$(mktemp); _ms_outf=$(mktemp)
run_bounded 60 "$_ms_outf" env MC_TRACE_FILE="$_ms_tr" MC_ATTEMPT_FILE="$_ms_att" \
  STUB_HANG_AILANG=1 MISSION_NOTIFY_TIMEOUT=2 \
  /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/tmp/repo
    STATE_DIR="$3"
    _pin_degraded="- driver pin: hang"
    . "$1"
    . "$2"
    _rc=$?
    echo "BLOCK_RC:$_rc"
    printf "%s" "$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    echo ""
    echo "ATTEMPTS:$(cat "$MC_ATTEMPT_FILE" 2>/dev/null || echo 0)"
  ' _ "$LAB/notify.sh" "$LAB/pin_block.sh" "$_ms_spool"
_ms_rc=$?; _ms_out="$(cat "$_ms_outf")"
_ms_rows=0; [ -f "$_ms_spool/mission-v1-notice-spool.tsv" ] && _ms_rows=$(wc -l < "$_ms_spool/mission-v1-notice-spool.tsv" | tr -d ' ')
_ms_good=0
case "$_ms_out" in *"BLOCK_RC:0"*) _ms_good=1;; esac
case "$_ms_out" in *"ATTEMPTS:3"*) : ;; *) _ms_good=0;; esac
case "$_ms_out" in *"GH:issue comment 635"*) : ;; *) _ms_good=0;; esac
[ "$_ms_rows" -eq 1 ] || _ms_good=0
if [ "$_ms_good" -eq 1 ]; then
  ok "production: direct send retries 3x then spools on timeout"
else
  bad "production: direct send retries 3x then spools on timeout" "rc=$_ms_rc rows=$_ms_rows got: ($_ms_out)"
fi
case "$_ms_out" in *"timed out after 2s (no output)"*) ok "synthesised timeout diagnostic";; *) bad "synthesised timeout diagnostic" "no synthesised tail, got: rc=$_ms_rc ($_ms_out)";; esac
rm -rf "$_ms_spool" "$_ms_att" "$_ms_tr" "$_ms_outf"

# (4) hanging drain send: one-row spool + hanging ailang -> cut off, row retained. 15s.
_dn_spool=$(mktemp -d)
printf 'TS\tTitle\tBody line one\n' > "$_dn_spool/mission-v1-notice-spool.tsv"
_dn_tr=$(mktemp); _dn_outf=$(mktemp)
run_bounded 15 "$_dn_outf" env MC_TRACE_FILE="$_dn_tr" STUB_HANG_AILANG=1 MISSION_NOTIFY_TIMEOUT=2 \
  /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    STATE_DIR="$1"
    . "$2"
    _mc_drain_notices
    echo "DRAIN_RC:$?"
  ' _ "$_dn_spool" "$LAB/drain.sh"
_dn_rc=$?; _dn_out="$(cat "$_dn_outf")"
_dn_rows=0; [ -f "$_dn_spool/mission-v1-notice-spool.tsv" ] && _dn_rows=$(wc -l < "$_dn_spool/mission-v1-notice-spool.tsv" | tr -d ' ')
if [ "$_dn_rc" = "124" ]; then
  bad "production: hanging drain send is cut off and row retained" "outer 15s watchdog tripped - drain send unbounded"
else
  case "$_dn_out" in
    *"DRAIN_RC:0"*) [ "$_dn_rows" -eq 1 ] && ok "production: hanging drain send is cut off and row retained" \
      || bad "production: hanging drain send is cut off and row retained" "drain rc0 but spool rows=$_dn_rows ($_dn_out)";;
    *) bad "production: hanging drain send is cut off and row retained" "no DRAIN_RC:0, rc=$_dn_rc ($_dn_out)";; esac
fi
rm -rf "$_dn_spool" "$_dn_tr" "$_dn_outf"

# (4b) large-backlog drain: 5-row spool + hanging ailang + aggregate budget -> whole
# drain bounded, 1 attempt, 4 deferred. Outer 30s. Instrument hygiene: the driver log
# (MC_TRACE_FILE) is the ONLY thing asserted; the bounded helper's job-control stderr
# lands in the run_bounded output file, never the trace file.
_dl_spool=$(mktemp -d)
for _i in 1 2 3 4 5; do printf 'TS\tTitle%s\tBody%s\n' "$_i" "$_i" >> "$_dl_spool/mission-v1-notice-spool.tsv"; done
_dl_tr=$(mktemp); _dl_att=$(mktemp); _dl_outf=$(mktemp); _dl_beg=$(date +%s)
run_bounded 30 "$_dl_outf" env MC_TRACE_FILE="$_dl_tr" MC_ATTEMPT_FILE="$_dl_att" \
  STUB_HANG_AILANG=1 MISSION_NOTIFY_TIMEOUT=2 MISSION_DRAIN_BUDGET=8 \
  /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    STATE_DIR="$1"
    . "$2"
    _mc_drain_notices
    echo "DRAIN_RC:$?"
  ' _ "$_dl_spool" "$LAB/drain.sh"
_dl_rc=$?; _dl_elapsed=$(( $(date +%s) - _dl_beg ))
_dl_rows=0; [ -f "$_dl_spool/mission-v1-notice-spool.tsv" ] && _dl_rows=$(wc -l < "$_dl_spool/mission-v1-notice-spool.tsv" | tr -d ' ')
_dl_attn=$(cat "$_dl_att" 2>/dev/null || echo 0)
_dl_log="$(cat "$_dl_tr" 2>/dev/null || true)"
if [ "$_dl_rc" = "124" ]; then
  bad "drain: whole drain returns within aggregate budget and preserves unattempted rows" "outer 30s watchdog tripped - aggregate budget removed"
else
  _dl_good=1
  [ "$_dl_elapsed" -le 15 ] || _dl_good=0
  [ "$_dl_rows" -eq 5 ] || _dl_good=0
  [ "$_dl_attn" -eq 1 ] || _dl_good=0
  if [ "$_dl_good" -eq 1 ]; then
    ok "drain: whole drain returns within aggregate budget and preserves unattempted rows"
  else
    bad "drain: whole drain returns within aggregate budget and preserves unattempted rows" "elapsed=${_dl_elapsed}s rows=$_dl_rows attempts=$_dl_attn rc=$_dl_rc"
  fi
fi
case "$_dl_log" in
  *"deferred 4 row(s), aggregate budget 8s exhausted"*) ok "drain: deferred-drain diagnostic names deferred row count";;
  *) bad "drain: deferred-drain diagnostic names deferred row count" "$(printf '%s' "$_dl_log"|tr '\n' '|')";;
esac
rm -rf "$_dl_spool" "$_dl_tr" "$_dl_att" "$_dl_outf"

# (5) hanging gh comment: ailang healthy, gh hangs -> bounded cutoff, WARNING, exit 0.
_gh_tr=$(mktemp); _gh_outf=$(mktemp); _gh_sp=$(mktemp -d); _gh_beg=$(date +%s)
run_bounded 15 "$_gh_outf" env MC_TRACE_FILE="$_gh_tr" STUB_HANG_GH=1 MISSION_NOTIFY_TIMEOUT=2 \
  /bin/bash -c '
    set -uo pipefail
    . "$MC_BND"
    NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"
    log() { printf "LOG:%s\n" "$*" >> "$MC_TRACE_FILE"; }
    MISSION_NAME=v1; MISSION_REPO=sunholo-data/ailang; MSG_FROM=mission-control
    MISSION_GH_ISSUE=635; LOG=/tmp/x.log; REPO=/tmp/repo
    STATE_DIR="$3"
    _pin_degraded="- driver pin: gh"
    . "$1"
    . "$2"
    _gh_rc2=$?
    echo "BLOCK_RC:$_gh_rc2"
    printf "%s" "$(cat "$MC_TRACE_FILE" 2>/dev/null || true)"
    echo ""
  ' _ "$LAB/notify.sh" "$LAB/pin_block.sh" "$_gh_sp"
_gh_rc=$?; _gh_out="$(cat "$_gh_outf")"; _gh_elapsed=$(( $(date +%s) - _gh_beg ))
if [ "$_gh_rc" = "124" ]; then
  bad "production: hanging gh comment is cut off and warns" "outer 15s watchdog tripped - gh bound removed"
else
  _gh_good=0
  case "$_gh_out" in *"BLOCK_RC:0"*) _gh_good=1;; esac
  case "$_gh_out" in *"LOG:WARNING: driver-pin notice FAILED to post to issue #635"*) : ;; *) _gh_good=0;; esac
  [ "$_gh_elapsed" -le 7 ] || _gh_good=0
  if [ "$_gh_good" -eq 1 ]; then
    ok "production: hanging gh comment is cut off and warns"
  else
    bad "production: hanging gh comment is cut off and warns" "rc=$_gh_rc elapsed=${_gh_elapsed}s got: ($_gh_out)"
  fi
fi
rm -rf "$_gh_tr" "$_gh_outf" "$_gh_sp"

# (6) guard (green/green): the env prefix delivers store/project to the child, visible in
# the failure-arm WARNING tail. Exists so the env-prefix mutation has a named victim.
_ge_t=$(AILANG_RC=1 run pin_block "- x")
check "direct send reaches child with store=gcp" "$_ge_t" "store=<gcp> proj=<ailang-multivac>"

# (7) guard (green/green): exactly one preflight drain invocation after the pin decision.
_wiring=$(awk '/^# --- DRIVER PIN DECISION END ---/,0' "$DRV" | grep -c '^_mc_drain_notices$')
[ "$_wiring" -eq 1 ] && ok "wiring: exactly one preflight drain call" || bad "wiring: exactly one preflight drain call" "got $_wiring"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
