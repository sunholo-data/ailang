#!/bin/bash
# PostToolUse hook for Edit/Write on .ail files: canonical-format the edited file.
# Non-blocking (always exits 0), NON-SILENT (contract clause 4), and BOUNDED
# (portable date-deadline + SIGTERM->grace->SIGKILL kill guard — no GNU `timeout`
# dependency):
#   exit 0 -> optional confirmation
#   exit 3 -> input does not parse yet (expected mid-edit) -> defer silently (clause 5)
#   timeout -> kill fmt (atomic --write wrote nothing; file byte-identical),
#     surface an advisory note via additionalContext, then exit 0
#   anything else (2 = operational error / panic; 127 = ailang missing; ...) ->
#     surface captured output to the agent via additionalContext, then exit 0
set +e

# jq is load-bearing for parsing hook input; a missing jq must SURFACE, never
# silently no-op via an empty path (quorum R2 finding).
if ! command -v jq >/dev/null 2>&1; then
  echo "format_ail hook: jq not found on PATH — .ail formatting skipped (install jq)" >&2
  exit 0
fi

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
file_path=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""')  # no 2>/dev/null: jq errors surface
case "$file_path" in
  *.ail) ;;
  *) exit 0 ;;
esac

# Bounded run (contract clause 4: never wedge a turn, even on a hung fmt).
FMT_TIMEOUT_SECS=10
FMT_TMP=$(mktemp)
ailang fmt --write "$file_path" >"$FMT_TMP" 2>&1 &   # capture stderr — NEVER 2>/dev/null
FMT_PID=$!
FMT_DEADLINE=$(( $(date +%s) + FMT_TIMEOUT_SECS ))
FMT_TIMED_OUT=0
while kill -0 "$FMT_PID" 2>/dev/null; do            # 2>/dev/null here = process-probe noise only
  if [ "$(date +%s)" -ge "$FMT_DEADLINE" ]; then
    # SIGTERM -> grace -> SIGKILL, then reap. A SIGTERM-ignoring process (deadlock,
    # trapped signal) would otherwise wedge the turn on an unbounded `wait`. This is
    # the same escalation the mission's executor wrappers use.
    kill "$FMT_PID" 2>/dev/null
    sleep 1
    kill -9 "$FMT_PID" 2>/dev/null
    FMT_TIMED_OUT=1
    break
  fi
  sleep 0.2
done
wait "$FMT_PID" 2>/dev/null
FMT_RC=$?
FMT_OUT=$(cat "$FMT_TMP")
rm -f "$FMT_TMP"
if [ "$FMT_TIMED_OUT" -eq 1 ]; then
  FMT_RC=124   # synthetic timeout code -> routed to the surface-it branch below
  FMT_OUT="ailang fmt timed out after ${FMT_TIMEOUT_SECS}s and was killed; file left as written (atomic --write wrote nothing).
$FMT_OUT"
fi

case "$FMT_RC" in
  0) echo "✓ Formatted $file_path" >&2 ;;
  3) : ;;  # not parseable yet — the ONLY silent case (contract clause 5)
  *)
    jq -n --arg ctx "ailang fmt failed (exit $FMT_RC) on $file_path — file left as written:
$FMT_OUT" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: $ctx
      }
    }' 2>/dev/null || echo "ailang fmt failed (exit $FMT_RC) on $file_path: $FMT_OUT" >&2
    ;;
esac
exit 0
