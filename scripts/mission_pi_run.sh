#!/usr/bin/env bash
# mission_pi_run.sh — run the pi executor lane under guards that can actually see
# the two failure modes we have measured, and emit a TYPED verdict instead of rc=0.
#
# WHY THIS EXISTS (measured 2026-08-26 from OpenRouter Broadcast traces, which are
# the provider's own side of the wire — see docs/docs/guides/debugging.md):
#
#   Every pi executor "silent failure" on record has the same shape on the wire:
#   the model streams ONLY reasoning tokens and never emits content or a tool call.
#   In the whole 08-18..08-22 broadcast corpus, 3 of 173 generations had no
#   finish_reason (cancelled); ALL THREE had `completion: ""` with output_tokens ==
#   reasoning_tokens. The other 170 all carried content or tool_calls. The signature
#   is clean, and it is NOT deepseek-specific — it also fired on z-ai/glm-5.2 under
#   OpenCode, on a different provider host.
#
#   The runs did not fail on their own. WE killed them:
#     * 2026-08-18: 7,130 reasoning tokens / 111s, killed by the old 300 MB NDJSON
#       size ceiling.
#     * 2026-08-19: 1,827 reasoning tokens / 73s, cancelled.
#
#   And the size ceiling was measuring the WRONG THING. pi's `message_update` event
#   carries the WHOLE accumulated message, not a delta (verified first-party in
#   pi 0.73.1, dist/core/agent-session.js:421-427), so NDJSON bytes grow QUADRATICALLY
#   in emitted tokens. 7,130 tokens produced 330 MB. Extrapolated to the model's
#   declared 65,536-token budget that is ~28 GB — i.e. the old ceiling silently
#   capped the lane at roughly 7,000 reasoning tokens, an accidental limit that no
#   prompt change could ever have fixed.
#
# WHAT THIS SCRIPT DOES ABOUT IT
#
#   1. FILTERS `message_update` out of the banked NDJSON. Nothing is lost: `message_end`
#      carries the complete final message including reasoning. Size becomes LINEAR, so
#      the disk-exhaustion hazard goes away without a ceiling that truncates real work.
#   2. Keeps the newest message_update in a separate single-record SNAPSHOT file, so a
#      run killed mid-turn still has full forensics at bounded cost.
#   3. Uses the banked file's mtime as a PROGRESS CLOCK. Because updates are filtered,
#      a content-free reasoning turn writes nothing to it — the clock freezes exactly
#      when the failure mode is occurring. This costs nothing and needs no parsing.
#   4. Distinguishes the two stalls, which need different responses:
#        banked frozen + snapshot advancing -> `reasoning_stall` (model thinking, no output)
#        banked frozen + snapshot frozen    -> `stream_dead`     (upstream host hung)
#      `stream_dead` is real and transient: a bare-id deepseek call was measured
#      hanging 90s with HTTP 200 and an empty body on 2026-08-26, while 14/14 retries
#      immediately afterwards succeeded across 6 different provider hosts.
#   5. Asserts the worktree diff is NON-EMPTY. Of the three assertions the old recipe
#      mandated, this is the only load-bearing one: `stopReason` is now known evadable
#      in BOTH directions — `length` pre-2026-08-13, and a clean `stop` at 625 tokens
#      post-fix — so it can neither confirm nor deny that work happened.
#
# EXIT CODES (the verdict is also written as JSON to --verdict)
#   0  ok               — pi finished and the worktree changed
#   10 empty_worktree   — pi finished, changed nothing. The false-green in its pure form.
#   11 reasoning_stall  — killed: reasoning with no content/tool-call past the stall bound
#   12 stream_dead      — killed: no bytes at all past the stall bound
#   13 wall_timeout     — killed: exceeded --max-seconds
#   14 launch_failed    — pi could not start / bad arguments
#
# Bash 3.2 (rig default). No `declare -A`, no `${v,,}`, no `timeout(1)`.

set -u

MODEL=""
DIRECTIVE=""
WORKDIR=""
OUT=""
VERDICT=""
MAX_SECONDS="${MISSION_PI_MAX_SECONDS:-1800}"
STALL_SECONDS="${MISSION_PI_STALL_SECONDS:-600}"
POLL_SECONDS="${MISSION_PI_POLL_SECONDS:-10}"
SNAP_EVERY="${MISSION_PI_SNAP_EVERY:-50}"

usage() {
  cat >&2 <<'EOF'
usage: mission_pi_run.sh --model M --directive FILE --workdir DIR --out NDJSON
                         [--verdict JSON] [--max-seconds N] [--stall-seconds N]

  --model         pi model id, e.g. openrouter/deepseek/deepseek-v4-flash-0731
  --directive     file whose contents are delivered to pi on stdin
  --workdir       worktree pi runs in; its git diff is the load-bearing assertion
  --out           path for the FILTERED ndjson (message_update removed)
  --verdict       path for the verdict JSON (default: <out>.verdict.json)
  --max-seconds   wall-clock cap        (default 1800, env MISSION_PI_MAX_SECONDS)
  --stall-seconds no-progress cap       (default 600,  env MISSION_PI_STALL_SECONDS)

Deliberately generous stall default: a legitimate thinking turn on a hard sprint
runs for minutes. This bound exists to catch a turn that never ENDS, not one that
takes a while. The measured failures froze at 111s and 73s.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --model)         MODEL="$2"; shift 2 ;;
    --directive)     DIRECTIVE="$2"; shift 2 ;;
    --workdir)       WORKDIR="$2"; shift 2 ;;
    --out)           OUT="$2"; shift 2 ;;
    --verdict)       VERDICT="$2"; shift 2 ;;
    --max-seconds)   MAX_SECONDS="$2"; shift 2 ;;
    --stall-seconds) STALL_SECONDS="$2"; shift 2 ;;
    -h|--help)       usage; exit 0 ;;
    *) echo "mission_pi_run.sh: unknown argument '$1'" >&2; usage; exit 14 ;;
  esac
done

[ -n "$MODEL" ] && [ -n "$DIRECTIVE" ] && [ -n "$WORKDIR" ] && [ -n "$OUT" ] || {
  echo "mission_pi_run.sh: --model, --directive, --workdir and --out are all required" >&2
  usage; exit 14
}
[ -f "$DIRECTIVE" ] || { echo "mission_pi_run.sh: directive file not found: $DIRECTIVE" >&2; exit 14; }
[ -d "$WORKDIR" ]   || { echo "mission_pi_run.sh: workdir not found: $WORKDIR" >&2; exit 14; }
[ -n "$VERDICT" ]   || VERDICT="${OUT}.verdict.json"

# The run cds into $WORKDIR, so every path we hand the filter must be absolute or the
# output lands somewhere the caller is not looking. Resolve rather than require.
case "$OUT" in /*) ;; *) OUT="$(pwd)/$OUT" ;; esac
case "$VERDICT" in /*) ;; *) VERDICT="$(pwd)/$VERDICT" ;; esac
case "$DIRECTIVE" in /*) ;; *) DIRECTIVE="$(pwd)/$DIRECTIVE" ;; esac
WORKDIR=$(cd "$WORKDIR" && pwd)

SNAP="${OUT}.snapshot.ndjson"
ERR="${OUT}.stderr"
: > "$OUT"; : > "$SNAP"; : > "$ERR"

# mtime in epoch seconds. BSD stat (darwin) first, GNU stat second — the rig is
# darwin but CI legs are not, and a silently-empty stat would freeze the clock and
# make every run look stalled.
mtime_of() {
  stat -f %m "$1" 2>/dev/null || stat -c %Y "$1" 2>/dev/null || echo 0
}
now() { date +%s; }

# Deliver the directive on stdin and CLOSE it — pi waits forever on an open stdin,
# which is a wedge the mission loop has hit before.
#
# The awk filter is the whole trick. It must be awk and not a bash read-loop: pi
# emits message_update at ~3 MB/s during a long turn, and a shell loop becomes the
# bottleneck and backpressures the model.
#
# `close(snap)` before each snapshot write reopens with `>`, which truncates — so the
# snapshot stays exactly one record instead of growing.
#
# `set -m` matters and is not cosmetic: without job control a background job shares the
# script's process group, so the `kill -- -PID` below would either fail or — if the pid
# collided with our own pgid — kill this script. With it, the job leads its own group and
# the negative-pid kill reaches pi's children, which is the whole point of killing at all.
set -m
(
  # RUN IN THE WORKTREE. pi edits files relative to its CWD, and --workdir is what we
  # later assert the git diff of. Without this cd the two are different directories:
  # the model does real work somewhere else and the verdict reads empty_worktree — or,
  # worse, reads `ok` off a worktree that was dirty for unrelated reasons. Caught live
  # 2026-08-26 when a run reported 4 tool executions and 0 changed files, and the
  # model's own closing message said it could not find the file and created it.
  cd "$WORKDIR" || exit 14
  pi --mode json --no-session --model "$MODEL" < "$DIRECTIVE" 2>"$ERR" |
    awk -v out="$OUT" -v snap="$SNAP" -v every="$SNAP_EVERY" '
      /"type":"message_update"/ {
        n++
        if (n % every == 1) { close(snap); print $0 > snap }
        next
      }
      { print $0 >> out; fflush(out) }
    '
) &
RUNNER_PID=$!

START=$(now)
LAST_OUT_M=$(mtime_of "$OUT")
PROGRESS_AT=$START
# Snapshot mtime AS OF the last progress event. The stall verdict compares against
# this, not against the previous poll: the question is "has the model emitted anything
# at all during THIS stall window", and a per-poll comparison answers a different
# (and much noisier) question — it misses any snapshot cadence slower than the poll.
SNAP_AT_PROGRESS=$(mtime_of "$SNAP")
OUTCOME=""

# Never poll slower than a third of the stall window, or the first observation can
# land past the bound and report a stall that never happened.
if [ "$POLL_SECONDS" -gt $((STALL_SECONDS / 3)) ] && [ $((STALL_SECONDS / 3)) -gt 0 ]; then
  POLL_SECONDS=$((STALL_SECONDS / 3))
fi

while :; do
  sleep "$POLL_SECONDS"

  # Liveness is checked AFTER the sleep and BEFORE the stall arithmetic. The reverse
  # order reports a stall for any run that completes inside one poll interval.
  if ! kill -0 "$RUNNER_PID" 2>/dev/null; then
    OUTCOME="finished"
    break
  fi

  T=$(now)
  OUT_M=$(mtime_of "$OUT")
  SNAP_M=$(mtime_of "$SNAP")

  # Any write to the FILTERED file is progress: a tool call, a turn boundary, a
  # completed message. Reasoning deltas are excluded by construction, which is
  # precisely why this clock detects the failure and a raw byte counter cannot.
  if [ "$OUT_M" != "$LAST_OUT_M" ]; then
    LAST_OUT_M="$OUT_M"
    PROGRESS_AT="$T"
    SNAP_AT_PROGRESS="$SNAP_M"
  fi

  if [ $((T - PROGRESS_AT)) -ge "$STALL_SECONDS" ]; then
    if [ "$SNAP_M" != "$SNAP_AT_PROGRESS" ]; then
      OUTCOME="reasoning_stall"   # thinking hard, emitting nothing usable
    else
      OUTCOME="stream_dead"       # nothing at all is arriving
    fi
    break
  fi

  if [ $((T - START)) -ge "$MAX_SECONDS" ]; then
    OUTCOME="wall_timeout"
    break
  fi
done

if [ "$OUTCOME" != "finished" ]; then
  # Kill the process GROUP: pi spawns children, and killing only the shell leaves
  # the model streaming into a filter nobody reads.
  kill -TERM -"$RUNNER_PID" 2>/dev/null || kill -TERM "$RUNNER_PID" 2>/dev/null
  sleep 2
  kill -KILL -"$RUNNER_PID" 2>/dev/null || kill -KILL "$RUNNER_PID" 2>/dev/null
fi
wait "$RUNNER_PID" 2>/dev/null
PI_RC=$?

ELAPSED=$(( $(now) - START ))
DIFF_LINES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
AGENT_END=$(grep -c '"type":"agent_end"' "$OUT" 2>/dev/null | tr -d ' ')
# pi emits tool_execution_START/_UPDATE/_END, never a bare "tool_execution" — an
# exact-match grep on the bare name silently reports 0 on a run that used tools.
# Counting _end (completed calls) is the number a reader actually wants.
TOOL_CALLS=$(grep -c '"type":"tool_execution_end"' "$OUT" 2>/dev/null | tr -d ' ')
OUT_BYTES=$(wc -c < "$OUT" 2>/dev/null | tr -d ' ')

case "$OUTCOME" in
  finished)
    if [ "${DIFF_LINES:-0}" -gt 0 ]; then VERDICT_NAME="ok"; RC=0
    else VERDICT_NAME="empty_worktree"; RC=10; fi ;;
  reasoning_stall) VERDICT_NAME="reasoning_stall"; RC=11 ;;
  stream_dead)     VERDICT_NAME="stream_dead";     RC=12 ;;
  wall_timeout)    VERDICT_NAME="wall_timeout";    RC=13 ;;
  *)               VERDICT_NAME="launch_failed";   RC=14 ;;
esac

cat > "$VERDICT" <<EOF
{
  "verdict": "$VERDICT_NAME",
  "rc": $RC,
  "model": "$MODEL",
  "pi_rc": $PI_RC,
  "elapsed_seconds": $ELAPSED,
  "worktree_changed_files": ${DIFF_LINES:-0},
  "tool_executions": ${TOOL_CALLS:-0},
  "agent_end_events": ${AGENT_END:-0},
  "ndjson_bytes_filtered": ${OUT_BYTES:-0},
  "ndjson": "$OUT",
  "snapshot": "$SNAP",
  "stderr": "$ERR"
}
EOF

echo "pi lane verdict: $VERDICT_NAME (rc=$RC) after ${ELAPSED}s — ${DIFF_LINES:-0} changed files, ${TOOL_CALLS:-0} tool executions" >&2
exit "$RC"
