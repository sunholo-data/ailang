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
DRIVER="${DRIVER:-$HERE/mission-control.sh}"

TMP="${TMPDIR:-/tmp}/mc-stall-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

awk '/^_mc_progress_bytes\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_prog.sh"
awk '/^_mc_stalled\(\) \{/,/^\}$/'        "$DRIVER" > "$TMP/fn_stalled.sh"
# _mc_etime_secs is extracted, not stubbed: the age arm is only as good as the
# driver's own "[[DD-]HH:]MM:SS" parser, so the suite must run the real one.
awk '/^_mc_etime_secs\(\) \{/,/^\}$/'     "$DRIVER" > "$TMP/fn_etime.sh"
awk '/^_mc_tree_write_bytes\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_treew.sh"
awk '/^_mc_rusage_snapshot\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_rusage.sh"
awk '/^_mc_run_once\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_run_once.sh"
# Guard the extraction: an empty extract would make every arm vacuously pass.
[ -s "$TMP/fn_prog.sh" ] && [ -s "$TMP/fn_stalled.sh" ] && [ -s "$TMP/fn_etime.sh" ] && [ -s "$TMP/fn_treew.sh" ] && [ -s "$TMP/fn_rusage.sh" ] \
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
# shellcheck source=/dev/null
. "$TMP/fn_treew.sh"
. "$TMP/fn_rusage.sh"

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
STUB_RUSAGE=0
STUB_RUSAGE_ROWS=""
STUB_PIDS="1000 1001"
STUB_BIRTH="Mon Sep 28 12:00:00 2026"
STUB_VANISHED_PID=""
MC_DRIVER_ROOT="$(cd "$HERE/../.." && pwd)"
LOGGED=""

_mc_descendants() { printf '%s\n' $STUB_PIDS; }
ps() {
  case "$*" in
    *etime*) printf '%s\n' "$STUB_ETIME" ;;
    *%cpu*)  printf '%s\n' "$STUB_CPU" ;;
    *lstart*)
      if [ -n "$STUB_VANISHED_PID" ]; then
        case "$*" in *"-p $STUB_VANISHED_PID") return 1 ;; esac
      fi
      printf '%s\n' "$STUB_BIRTH" ;;
  esac
}
_mc_rusage_read() {
  if [ -n "$STUB_RUSAGE_ROWS" ]; then printf '%s\n' "$STUB_RUSAGE_ROWS"; return; fi
  printf '1000\t0\t0\n1001\t%s\t0\n' "$STUB_RUSAGE"
}
log() { LOGGED="$LOGGED$*"$'\n'; }
# lsof -F records for the files the tree holds open. Empty by default, so the
# wedge arms below see no open-for-write file unless a case sets one.
STUB_OPEN_W=""
lsof() {
  [ -n "$STUB_OPEN_W" ] || return 1
  printf 'p1001\nf1\naw\ntREG\nn%s\nf2\naw\ntCHR\nn/dev/null\nf3\nar\ntREG\nn%s\n' \
    "$STUB_OPEN_W" "$STUB_OPEN_W.readonly"
}

STALL_CHILD_AGE=2400
STALL_CPU_PCT=2
CONTROLLER_PROVIDER=claude

# Own-session fixtures under a fake HOME; inherited state roots cannot escape TMP.
export HOME="$TMP/home"
export AILANG_STATE_DIR="$TMP/state" PI_CODING_AGENT_DIR="$TMP/home/.pi/agent"
MC_CLAUDE_SESSION_ID="F0EF90F8-18DF-459C-8561-06D7B1040255"
MC_PI_SESSION_ID="01a0d1e3"
WORK="$TMP/work/.pin-v1"
mkdir -p "$WORK"
cd "$WORK" || exit 1
SLUG=$(printf '%s' "$PWD" | tr '/.' '--')
PROJ="$HOME/.claude/projects/$SLUG"
mkdir -p "$PROJ"
TRANSCRIPT="$PROJ/$MC_CLAUDE_SESSION_ID.jsonl"
: > "$TRANSCRIPT"

LOG="$TMP/driver.log"; : > "$LOG"
_mc_heartbeat="$TMP/heartbeat"; : > "$_mc_heartbeat"

reset_state() { unset _MC_PROG_PREV _MC_HB_PREV _MC_STALL_NOPROG_LOGGED _MC_RUSAGE_PREV _MC_RUSAGE_IDENT_PREV; LOGGED=""; STUB_RUSAGE=0; STUB_RUSAGE_ROWS=""; STUB_PIDS="1000 1001"; STUB_BIRTH="Mon Sep 28 12:00:00 2026"; STUB_VANISHED_PID=""; }
grow() { printf '%s\n' "$1" >> "$2"; }        # append one line to a file
# _mc_stalled carries state between samples (_MC_PROG_PREV) and writes through
# log(). Both die in a subshell, so the verdict is taken in THIS shell and left
# in $V — a `$(verdict)` capture would re-seed on every call and pass vacuously.
V=""
verdict() { if _mc_stalled 1000; then V=stalled; else V=live; fi; }

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
# 8. Own-session discovery must find the controller's transcript and its growth.
# ---------------------------------------------------------------------------
reset_state
before=$(_mc_progress_bytes)
grow "another turn" "$TRANSCRIPT"
after=$(_mc_progress_bytes)
if [ -n "$before" ] && [ -n "$after" ] && [ "$after" -gt "$before" ]; then
  ok "_mc_progress_bytes finds the own-session transcript and sees it grow"
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

# ---------------------------------------------------------------------------
# 10. THE 2026-09-24 REGRESSION: a controller blocked on a child executor. Its
#     own transcript, the driver log and the heartbeat are all flat and the tree
#     reads 0% CPU, but the child is streaming into a file it holds open. That is
#     live work — the executor finished M1-M3 24 minutes after it was killed.
# ---------------------------------------------------------------------------
reset_state
STUB_CPU="0.0"
EXEC_OUT="$TMP/pi_exec.ndjson"; : > "$EXEC_OUT"; : > "$EXEC_OUT.readonly"
STUB_OPEN_W="$EXEC_OUT"
verdict                             # seed
grow '{"type":"message_end"}' "$EXEC_OUT"
verdict
check "a child streaming into a file it holds open for write is live" "$V" "live"

# ...and the arm must not make the wedge unkillable: an open-for-write file that
# does NOT grow is no progress. (A read-only file growing must not count either.)
verdict
grow "someone else's append" "$EXEC_OUT.readonly"
verdict
check "an open-for-write file that stays flat is still stalled" "$V" "stalled"
STUB_OPEN_W=""

# ---------------------------------------------------------------------------
# 11. A pi controller's own transcript. `pi -p` buffers to the end, so the
#     driver log is flat for the whole run; without this arm, every pi slot past
#     STALL_CHILD_AGE died on the first 10 quiet minutes (pi is itself a
#     descendant of the watched subshell, so the long-child arm is always true).
#     The path rule is pi's safePath: leading / dropped, [/\:] -> -, dots KEPT.
# ---------------------------------------------------------------------------
reset_state
CONTROLLER_PROVIDER=pi
PI_SLUG="--$(printf '%s' "${PWD#/}" | tr '/' '-')--"
case "$PI_SLUG" in *.pin-v1--) ok "pi slug keeps dots ($PI_SLUG)" ;; *) bad "pi slug should keep dots, got $PI_SLUG" ;; esac
PI_DIR="$PI_CODING_AGENT_DIR/sessions/$PI_SLUG"; mkdir -p "$PI_DIR"
PI_T="$PI_DIR/2026-09-24T05-29-00-416Z_${MC_PI_SESSION_ID}.jsonl"; : > "$PI_T"
verdict
grow '{"type":"message","message":{"role":"toolResult"}}' "$PI_T"
verdict
check "a pi controller whose session transcript grows is live" "$V" "live"
verdict
verdict
check "a pi controller with a flat transcript and nothing else moving is stalled" "$V" "stalled"
CONTROLLER_PROVIDER=claude

# a1 wrong-transcript. Kills mutation: revert Claude to `ls -t | head -1`.
reset_state
DECOY="$PROJ/other-session.jsonl"; : > "$DECOY"
touch -t 202001010000 "$TRANSCRIPT"; touch -t 203001010000 "$DECOY"
verdict
grow "unrelated parent" "$DECOY"; touch -t 203001010000 "$DECOY"
verdict
check "a1 wrong-transcript: newer unrelated parent cannot keep own session live" "$V" "stalled"
rm "$DECOY"

# a2 subagent progress. Kills mutation: drop the own-session subagent fold.
reset_state
SUBDIR="$PROJ/$MC_CLAUDE_SESSION_ID/subagents"; mkdir -p "$SUBDIR"
SUBAGENT="$SUBDIR/agent-x.jsonl"; : > "$SUBAGENT"
verdict
grow "foreground Agent tool result" "$SUBAGENT"
verdict
check "a2 subagent progress: own parent flat but own subagent grows" "$V" "live"

# a3 other session's subagent. Kills mutation: unanchored `*/subagents` glob.
reset_state
OTHER_SUB="$PROJ/other-session/subagents"; mkdir -p "$OTHER_SUB"
: > "$OTHER_SUB/agent-y.jsonl"
verdict
grow "unrelated subagent" "$OTHER_SUB/agent-y.jsonl"
grow "metadata is not transcript progress" "$SUBDIR/agent-x.meta.json"
verdict
check "a3 other subagent: unrelated session and metadata cannot count" "$V" "stalled"
rm -r "$PROJ/other-session" "$PROJ/$MC_CLAUDE_SESSION_ID"

# a4 physical-path slug. Kills mutation: re-derive Claude directory from PWD.
reset_state
PHYSICAL_PROJ="$HOME/.claude/projects/-private-different-physical-slug"
mkdir -p "$PHYSICAL_PROJ"
mv "$TRANSCRIPT" "$PHYSICAL_PROJ/"
verdict
grow "own physical-path turn" "$PHYSICAL_PROJ/$MC_CLAUDE_SESSION_ID.jsonl"
verdict
check "a4 physical-path slug: exact case session id across project slugs" "$V" "live"
mv "$PHYSICAL_PROJ/$MC_CLAUDE_SESSION_ID.jsonl" "$TRANSCRIPT"
rmdir "$PHYSICAL_PROJ"

# a5 unset/empty Claude id. Kills mutations: newest fallback OR LOG-alone rc0.
SAVED_CLAUDE_ID="$MC_CLAUDE_SESSION_ID"
for mode in unset empty; do
  reset_state
  if [ "$mode" = unset ]; then unset MC_CLAUDE_SESSION_ID; else MC_CLAUDE_SESSION_ID=""; fi
  instrument_rc=0; ( _mc_progress_bytes ) > "$TMP/no-id-bytes" || instrument_rc=$?
  verdict; verdict
  warnings=$(printf '%s' "$LOGGED" | grep -c 'NO progress instrument')
  if [ "$instrument_rc" -ne 0 ] && [ ! -s "$TMP/no-id-bytes" ] &&
     [ "$V" = live ] && [ "$_MC_STALL_WHY" = no-progress-instrument ] && [ "$warnings" -eq 1 ]; then
    ok "a5 $mode Claude id: existing LOG cannot arm kill; warns once"
  else
    bad "a5 $mode Claude id: rc=$instrument_rc verdict=$V why=$_MC_STALL_WHY warnings=$warnings"
  fi
done
MC_CLAUDE_SESSION_ID="$SAVED_CLAUDE_ID"

# a6 pi wrong-transcript. Kills mutation: restore pi `ls -t | head -1`.
reset_state
CONTROLLER_PROVIDER=pi
DECOY="$PI_DIR/2030-01-01T00-00-00_other-pi-session.jsonl"; : > "$DECOY"
touch -t 202001010000 "$PI_T"; touch -t 203001010000 "$DECOY"
before=$(_mc_progress_bytes)
grow "unrelated pi turn" "$DECOY"; touch -t 203001010000 "$DECOY"
after=$(_mc_progress_bytes)
check "a6 pi wrong-transcript: newer unrelated file adds no bytes" "$after" "$before"
SAVED_PI_ID="$MC_PI_SESSION_ID"; SAVED_LOG="$LOG"
for mode in unset empty; do
  reset_state
  if [ "$mode" = unset ]; then unset MC_PI_SESSION_ID; else MC_PI_SESSION_ID=""; fi
  instrument_rc=0; bytes=$(_mc_progress_bytes) || instrument_rc=$?
  log_bytes=$(wc -c < "$LOG" | tr -d ' ')
  LOG=""; no_log_rc=0; _mc_progress_bytes > "$TMP/no-pi-id-bytes" || no_log_rc=$?
  LOG="$SAVED_LOG"
  if [ "$instrument_rc" -eq 0 ] && [ "$bytes" = "$log_bytes" ] &&
     [ "$no_log_rc" -ne 0 ] && [ ! -s "$TMP/no-pi-id-bytes" ]; then
    ok "a6 $mode pi id: only LOG bytes count with existing rc semantics"
  else
    bad "a6 $mode pi id: rc=$instrument_rc bytes=$bytes log=$log_bytes no_log_rc=$no_log_rc"
  fi
done
MC_PI_SESSION_ID="$SAVED_PI_ID"; CONTROLLER_PROVIDER=claude
rm "$DECOY"

# a7 launch wiring. Kills mutations: remove Claude flag OR per-attempt assignment.
# Limit the launch extract to Claude's continued command, so pi's flag cannot pass.
sed -n '/^    claude -p /,/^  fi/p' "$TMP/fn_run_once.sh" > "$TMP/claude_launch.sh"
assignment_line=$(grep -n 'MC_CLAUDE_SESSION_ID=' "$TMP/fn_run_once.sh" | head -1 | cut -d: -f1)
launch_line=$(grep -n '^    claude -p ' "$TMP/fn_run_once.sh" | head -1 | cut -d: -f1)
if [ -s "$TMP/fn_run_once.sh" ] && [ -n "$assignment_line" ] && [ -n "$launch_line" ] &&
   [ "$assignment_line" -lt "$launch_line" ] &&
   grep -q -- '--session-id "$MC_CLAUDE_SESSION_ID"' "$TMP/fn_run_once.sh" &&
   grep -q 'MC_CLAUDE_SESSION_ARGS' "$TMP/claude_launch.sh"; then
  ok "a7 launch wiring: per-attempt id and Claude session flag"
else
  bad "a7 launch wiring: missing per-attempt assignment or Claude session flag"
fi

# 12. D-FLEET-7: the 04:45 mutation drill was measured at >=62.54 CPU-s
# per 120-s window. The transcript, heartbeat, open-file bytes, and sampled
# instantaneous CPU were flat; the approved rusage delta must keep it live.
reset_state
verdict
STUB_RUSAGE=6254
verdict
check "04:45 positive: measured drill descendant CPU keeps flat signals live" "$V" "live"

# A W1 poll starts new short children but accrues only <=1.72 CPU-s/window.
# The 10-second threshold must leave it killable even when PID churns.
reset_state
verdict
STUB_PIDS="1000 1001 1002"
STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t0\t172\n1002\t0\t0')
verdict
check "W1 negative: short failed polls below 10 CPU-s are stalled" "$V" "stalled"

reset_state
verdict
verdict
check "W2 negative: blocked read with zero descendant CPU is stalled" "$V" "stalled"

# The controller root's own CPU is excluded even when it alone crosses the
# threshold. Its reaped-child counter is descendant work and must be included.
reset_state
verdict
STUB_RUSAGE_ROWS=$(printf '1000\t2000\t0\n1001\t0\t0')
verdict
check "root-own exclusion: busy controller alone leaves wedge stalled" "$V" "stalled"
reset_state
verdict
STUB_RUSAGE_ROWS=$(printf '1000\t0\t1200\n1001\t0\t0')
verdict
check "root-reaped inclusion: vanished direct child keeps drill live" "$V" "live"

reset_state
verdict
STUB_RUSAGE_ROWS=$(printf '1000\tNA\tNA\n1001\t0\t0')
verdict
check "unavailable rusage fails open instead of fabricating zero CPU" "$V" "live"
check "unavailable rusage identifies the instrument failure" "$_MC_STALL_WHY" "rusage-unavailable"

# A short-lived, non-root PID may disappear during the bounded read. A stable
# root and long child still provide a valid comparison and must not reset it.
reset_state
verdict
STUB_PIDS="1000 1001 1002"
STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t0\t0\n1002\tNA\tNA')
verdict
check "vanished non-root rusage row still permits a stalled verdict" "$V" "stalled"
reset_state
verdict
STUB_PIDS="1000 1001 1002"
STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t0\t0\n1002\t0\t0')
STUB_VANISHED_PID=1002
verdict
check "vanished non-root birth time still permits a stalled verdict" "$V" "stalled"
reset_state
verdict
STUB_VANISHED_PID=1000
verdict
check "unavailable root birth time still fails open" "$V" "live"

reset_state
verdict
STUB_BIRTH="Tue Sep 29 12:00:00 2026"
verdict
check "PID identity change invalidates the sample pair" "$V" "live"

# A vanished non-root child transfers its CPU into a persistent descendant's
# reaped-child counter. This fixture has no race: the two PID sets are explicit.
reset_state
STUB_PIDS="1000 1001 1002"
STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t100\t0\n1002\t0\t0')
verdict
STUB_PIDS="1000 1001"
STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t100\t1200')
verdict
check "vanished non-root child: reaped CPU keeps drill live" "$V" "live"

# Kill three one-line mutants of the extracted production function. A mutation
# is useful only when its named behavioral arm changes verdict.
sed 's/-ge 1000/-ge 999999/' "$TMP/fn_stalled.sh" > "$TMP/mut_no_rusage.sh"
if ( . "$TMP/mut_no_rusage.sh"; reset_state; verdict; STUB_RUSAGE=6254; verdict; [ "$V" = stalled ] ); then
  ok "mutation: disabling rusage turns 04:45 positive red"
else
  bad "mutation: disabling rusage did not turn 04:45 positive red"
fi
sed 's/-ge 1000/-gt 0/' "$TMP/fn_stalled.sh" > "$TMP/mut_any_cpu.sh"
if ( . "$TMP/mut_any_cpu.sh"; reset_state; verdict; STUB_RUSAGE=172; verdict; [ "$V" = live ] ); then
  ok "mutation: any CPU growth turns W1 negative red"
else
  bad "mutation: any CPU growth did not turn W1 negative red"
fi
sed 's/total + own + child/total + own/' "$TMP/fn_rusage.sh" > "$TMP/mut_no_reaped.sh"
if ( . "$TMP/mut_no_reaped.sh"; reset_state
     STUB_PIDS="1000 1001 1002"
     STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t100\t0\n1002\t0\t0'); verdict
     STUB_PIDS="1000 1001"
     STUB_RUSAGE_ROWS=$(printf '1000\t0\t0\n1001\t100\t1200'); verdict
     [ "$V" = stalled ] ); then
  ok "mutation: dropping reaped CPU turns vanished-child positive red"
else
  bad "mutation: dropping reaped CPU did not turn vanished-child positive red"
fi
sed 's/if \[ "$p" != "$root" \]; then total=/if true; then total=/' "$TMP/fn_rusage.sh" > "$TMP/mut_root_own.sh"
if ( . "$TMP/mut_root_own.sh"; reset_state; verdict
     STUB_RUSAGE_ROWS=$(printf '1000\t2000\t0\n1001\t0\t0'); verdict
     [ "$V" = live ] ); then
  ok "mutation: adding root-own CPU turns root-own exclusion red"
else
  bad "mutation: adding root-own CPU did not turn root-own exclusion red"
fi
sed 's/total=$((total + child)); root_seen=1/root_seen=1/' "$TMP/fn_rusage.sh" > "$TMP/mut_root_reaped.sh"
if ( . "$TMP/mut_root_reaped.sh"; reset_state; verdict
     STUB_RUSAGE_ROWS=$(printf '1000\t0\t1200\n1001\t0\t0'); verdict
     [ "$V" = stalled ] ); then
  ok "mutation: dropping root-reaped CPU turns root-reaped inclusion red"
else
  bad "mutation: dropping root-reaped CPU did not turn root-reaped inclusion red"
fi

# Live kernel control for the helper, including a trivial child reaped before
# a busy child (the evaluator's previously unreproduced ordering concern).
if [ "$(uname -s)" = Darwin ]; then
  command python3 - "$MC_DRIVER_ROOT/tools/launchd/lib/proc_rusage.py" <<'PY' && ok "live rusage sees a reaped busy child after a trivial child" || bad "live rusage missed reaped child CPU"
import subprocess, sys
subprocess.run([sys.executable, "-c", "pass"], check=True)
subprocess.run([sys.executable, "-c", "x=0\nfor i in range(4000000): x+=i"], check=True)
result = subprocess.run([sys.executable, sys.argv[1], str(__import__('os').getpid())], text=True, capture_output=True, check=True)
assert int(result.stdout.strip().split('\t')[2]) > 0, result.stdout
PY
  if (
    ps() { command ps "$@"; }
    _mc_rusage_read() { command python3 "$MC_DRIVER_ROOT/tools/launchd/lib/proc_rusage.py" "$@"; }
    mkfifo "$TMP/live-start"
    sh -c 'read x < "$1"; python3 -c "x=0
for i in range(4000000): x+=i"; touch "$2"; sleep 5' sh "$TMP/live-start" "$TMP/live-done" &
    worker=$!
    trap 'kill "$worker" 2>/dev/null; wait "$worker" 2>/dev/null' EXIT
    _mc_rusage_snapshot "$$" "$$ $worker" || exit 1
    before=$_MC_RUSAGE_TOTAL
    printf 'go\n' > "$TMP/live-start"
    i=0
    while [ ! -f "$TMP/live-done" ] && [ "$i" -lt 100 ]; do sleep 0.05; i=$((i + 1)); done
    [ -f "$TMP/live-done" ] || exit 1
    _mc_rusage_snapshot "$$" "$$ $worker" || exit 1
    [ "$_MC_RUSAGE_TOTAL" -gt "$before" ]
  ); then
    ok "live extracted snapshot sees reaped descendant CPU grow"
  else
    bad "live extracted snapshot missed descendant CPU growth"
  fi
fi

echo "---"
echo "stall watchdog: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
