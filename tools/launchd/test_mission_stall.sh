#!/bin/bash
# test_mission_stall.sh — the stall watchdog's no-progress predicate.
#
# WHY THIS EXISTS. On 2026-09-02 the stall watchdog killed 4 V1 and 3 world
# iterations in one day, including a V1 slot that was at Gate 5 committing its own
# iteration-321 record (which is why 321 has no record). The predicate was a single
# arm — instantaneous `ps %cpu` summed over the tree, "idle" below 2% — and the
# comment above it asserted the arm errs safe: "a session doing real work reads
# non-idle and is NOT flagged ... we miss late stalls, never kill live work".
# Measured against a LIVE controller whose transcript grew 15-45KB per 30s, that
# same expression read 0.10 / 0.30 / 0.80 / 1.40. An agent's wall-clock is spent
# BLOCKED ON THE MODEL API, not on CPU, so the safety claim was exactly inverted.
#
# The claim was in a comment, and a comment cannot red. Every arm below is a
# property the driver now has to keep, and arm 1 is the one that dies if anyone
# reintroduces a CPU-only test.
#
# Extraction, not duplication: the functions under test are awk'd out of
# mission-control.sh itself, so this suite cannot drift green against an edited
# driver.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"

TMP="${TMPDIR:-/tmp}/mc-stall-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

awk '/^_mc_progress_bytes\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_prog.sh"
awk '/^_mc_stalled\(\) \{/,/^\}$/'        "$DRIVER" > "$TMP/fn_stalled.sh"
# _mc_etime_secs is extracted, not stubbed: the age arm is only as good as the
# driver's own "[[DD-]HH:]MM:SS" parser, so the suite must run the real one.
awk '/^_mc_etime_secs\(\) \{/,/^\}$/'     "$DRIVER" > "$TMP/fn_etime.sh"
# Guard the extraction: an empty extract would make every arm vacuously pass.
[ -s "$TMP/fn_prog.sh" ] && [ -s "$TMP/fn_stalled.sh" ] && [ -s "$TMP/fn_etime.sh" ] \
  || { echo "FAIL extraction: function boundaries not found in $DRIVER"; exit 1; }
# And guard that we extracted the REAL predicate, not a stub with the same name.
grep -q '_MC_PROG_PREV' "$TMP/fn_stalled.sh" \
  || { echo "FAIL extraction: _mc_stalled carries no progress state — wrong function?"; exit 1; }

# shellcheck source=/dev/null
. "$TMP/fn_etime.sh"
# shellcheck source=/dev/null
. "$TMP/fn_prog.sh"
# shellcheck source=/dev/null
. "$TMP/fn_stalled.sh"

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "ok - $1"; }
bad()  { FAIL=$((FAIL+1)); echo "not ok - $1"; }
check(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1 (want=$3 got=$2)"; fi; }

# ---------------------------------------------------------------------------
# Harness. The driver's own helpers are stubbed at their seams: _mc_descendants
# (process tree), ps (etime + %cpu), log (driver log).
# ---------------------------------------------------------------------------
STUB_ETIME="90:00"     # 90m — past STALL_CHILD_AGE by default
STUB_CPU="0.0"
LOGGED=""

_mc_descendants() { echo "1000"; echo "1001"; }
ps() {
  case "$*" in
    *etime*) printf '%s\n' "$STUB_ETIME" ;;
    *%cpu*)  printf '%s\n' "$STUB_CPU" ;;
  esac
}
log() { LOGGED="$LOGGED$*"$'\n'; }

STALL_CHILD_AGE=2400
STALL_CPU_PCT=2
CONTROLLER_PROVIDER=claude

# A fake HOME + cwd so _mc_progress_bytes resolves a transcript the way the real
# driver does: ~/.claude/projects/<cwd with / and . mapped to ->/<uuid>.jsonl
export HOME="$TMP/home"
WORK="$TMP/work/.pin-v1"
mkdir -p "$WORK"
cd "$WORK" || exit 1
SLUG=$(printf '%s' "$PWD" | tr '/.' '--')
PROJ="$HOME/.claude/projects/$SLUG"
mkdir -p "$PROJ"
TRANSCRIPT="$PROJ/f0ef90f8-18df-459c-8561-06d7b1040255.jsonl"
: > "$TRANSCRIPT"

LOG="$TMP/driver.log"; : > "$LOG"
_mc_heartbeat="$TMP/heartbeat"; : > "$_mc_heartbeat"

reset_state() { unset _MC_PROG_PREV _MC_HB_PREV _MC_STALL_NOPROG_LOGGED; LOGGED=""; }
grow() { printf '%s\n' "$1" >> "$2"; }        # append one line to a file
# _mc_stalled carries state between samples (_MC_PROG_PREV) and writes through
# log(). Both die in a subshell, so the verdict is taken in THIS shell and left
# in $V — a `$(verdict)` capture would re-seed on every call and pass vacuously.
V=""
verdict() { if _mc_stalled 999; then V=stalled; else V=live; fi; }

# ---------------------------------------------------------------------------
# 1. THE REGRESSION ARM. A controller blocked on the model API reads under the
#    CPU floor while its transcript grows. That is a LIVE session, and the
#    2026-09-02 predicate called it stalled. Numbers are the measured ones.
# ---------------------------------------------------------------------------
reset_state
STUB_CPU="0.80"
verdict                             # seed the baseline
for pct in 1.40 0.30 0.10 0.50; do
  STUB_CPU="$pct"
  grow "assistant turn" "$TRANSCRIPT"          # 15-45KB/30s in the real thing
done
verdict
check "a session under the CPU floor whose transcript GROWS is live" "$V" "live"

# ---------------------------------------------------------------------------
# 2. The wedge it exists to catch: long-lived child, nothing moving anywhere.
# ---------------------------------------------------------------------------
reset_state
STUB_CPU="0.0"
verdict                             # seed
verdict
check "long child + flat transcript + flat heartbeat + idle CPU is stalled" "$V" "stalled"

# ---------------------------------------------------------------------------
# 3. The heartbeat is an independent arm: a gate stamp is progress even when the
#    transcript happens to be flat between two samples (a long tool call).
# ---------------------------------------------------------------------------
reset_state
verdict
grow "$(date +%s)	iso	gate-3	1	" "$_mc_heartbeat"
verdict
check "a gate heartbeat stamp counts as progress" "$V" "live"

# ---------------------------------------------------------------------------
# 4. CPU above the floor still vetoes, so the old arm is kept, not discarded.
# ---------------------------------------------------------------------------
reset_state
verdict
STUB_CPU="7.0"
verdict
check "CPU above the floor is live even with flat progress" "$V" "live"

# ---------------------------------------------------------------------------
# 5. No long-lived descendant → not the wedge fingerprint, whatever else is true.
# ---------------------------------------------------------------------------
reset_state
STUB_CPU="0.0"; STUB_ETIME="00:30"
verdict
verdict
check "no descendant past STALL_CHILD_AGE is never stalled" "$V" "live"
STUB_ETIME="90:00"

# ---------------------------------------------------------------------------
# 6. A first sample can prove nothing — it seeds, it never kills.
# ---------------------------------------------------------------------------
reset_state
STUB_CPU="0.0"
verdict
check "the first sample seeds the baseline and reports live" "$V" "live"

# ---------------------------------------------------------------------------
# 7. Fail OPEN and LOUD: with no progress instrument we cannot tell a wedge from
#    live work, so we must not guess. HARD_TIMEOUT still bounds the slot.
# ---------------------------------------------------------------------------
reset_state
SAVED_LOG="$LOG"; SAVED_PROV="$CONTROLLER_PROVIDER"
LOG=""; CONTROLLER_PROVIDER="someday-provider"   # no transcript rule, no driver log
verdict
verdict
check "no progress instrument → never killed" "$V" "live"
case "$LOGGED" in
  *"NO progress instrument"*) ok "no progress instrument → warns loudly" ;;
  *) bad "no progress instrument → warns loudly (log was: ${LOGGED:-<empty>})" ;;
esac
LOG="$SAVED_LOG"; CONTROLLER_PROVIDER="$SAVED_PROV"

# ---------------------------------------------------------------------------
# 8. The transcript path rule itself. The slug maps BOTH '/' and '.' to '-'
#    (/Users/u/.pin/v1 → -Users-u--pin-v1); getting it wrong silently disarms
#    arm 1, which is the failure that would put us back where we started.
# ---------------------------------------------------------------------------
reset_state
before=$(_mc_progress_bytes)
grow "another turn" "$TRANSCRIPT"
after=$(_mc_progress_bytes)
if [ -n "$before" ] && [ -n "$after" ] && [ "$after" -gt "$before" ]; then
  ok "_mc_progress_bytes finds the transcript by cwd-slug and sees it grow"
else
  bad "_mc_progress_bytes did not track the transcript (before=$before after=$after slug=$SLUG)"
fi

# The driver log is the arm for codex/pi, which stream into it.
reset_state
CONTROLLER_PROVIDER=codex
before=$(_mc_progress_bytes)
grow "codex streamed a line" "$LOG"
after=$(_mc_progress_bytes)
if [ -n "$before" ] && [ -n "$after" ] && [ "$after" -gt "$before" ]; then
  ok "_mc_progress_bytes tracks the driver log for a streaming provider"
else
  bad "_mc_progress_bytes did not track the driver log (before=$before after=$after)"
fi
CONTROLLER_PROVIDER=claude

# ---------------------------------------------------------------------------
# 9. The heartbeat path is DERIVED when the driver sets no _mc_heartbeat — the
#    world loop stamps gates but its driver has no such variable, and an arm that
#    is silently inert in one of three loops is not an arm.
# ---------------------------------------------------------------------------
reset_state
SAVED_HB="$_mc_heartbeat"
unset _mc_heartbeat
export AILANG_STATE_DIR="$TMP/state"; mkdir -p "$AILANG_STATE_DIR"
MISSION_NAME="world"
DERIVED="$AILANG_STATE_DIR/mission-world-heartbeat"; : > "$DERIVED"
verdict
grow "$(date +%s)	iso	gate-4	1	" "$DERIVED"
verdict
check "the heartbeat path is derived from MISSION_NAME when unset" "$V" "live"
_mc_heartbeat="$SAVED_HB"

echo "---"
echo "stall watchdog: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
