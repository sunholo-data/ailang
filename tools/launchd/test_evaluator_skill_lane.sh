#!/bin/bash
# The evaluator chain must always retain a lane that can load its own methodology.
#
# "Act as independent evaluator under sprint-evaluator methodology" names a SKILL — a 100-point
# rubric with a 70 threshold plus three executable scripts under .agents/skills/. A judge that
# cannot load it gets the NAME of the procedure and none of its content, applies a rubric it
# never sees, and does not stop. The docs canary failed 3/3 exactly that way.
#
# WHY THIS IS NOT "no pi rungs" (measured 2026-09-08): pi CAN load skills now — in a fresh
# worktree under an untrusted path it reported the rubric and threshold without reading a file.
# But that depends on a MACHINE PRECONDITION no chain can verify: workspace-trust.ts installed
# globally in ~/.pi/agent/extensions/. Where that is unmet, pi silently loads no skills.
#
# So the invariant is not "claude only" — it is: the chain must END on a lane that works with
# no precondition at all. A claude rung is skill-capable by construction.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
fail=0

chain=$(grep -oE 'MISSION_EVALUATOR_FALLBACK:-[^}"]*' "$DRIVER" | sed 's/.*:-//')
[ -n "$chain" ] || { echo "INSTRUMENT BROKEN: cannot read MISSION_EVALUATOR_FALLBACK"; exit 2; }
case "$chain" in *:*|*opus*) : ;; *) echo "INSTRUMENT BROKEN: chain '$chain' has no rungs"; exit 2;; esac

last=""; claude_seen=0
for rung in $(printf '%s' "$chain" | tr ',' ' '); do
  last="$rung"
  case "$rung" in
    claude:*|opus|sonnet|haiku) claude_seen=1; echo "ok (no precondition): $rung" ;;
    pi:*)                       echo "ok (needs global workspace-trust.ts): $rung" ;;
    opencode:*|codex:*)         echo "FAIL: '$rung' has no measured skill support"; fail=1 ;;
    *)                          echo "FAIL: unclassified rung '$rung' — classify it before shipping"; fail=1 ;;
  esac
done

# The TAIL must be precondition-free: it is what runs when everything else has been skipped.
case "$last" in
  claude:*|opus|sonnet|haiku) echo "ok: tail '$last' needs no machine precondition" ;;
  *) echo "FAIL: tail '$last' depends on a precondition; the last resort must not"; fail=1 ;;
esac
[ "$claude_seen" -eq 1 ] || { echo "FAIL: no precondition-free rung anywhere in the chain"; fail=1; }

primary=$(grep -oE 'MISSION_EVALUATOR_MODEL:-[^}"]*' "$DRIVER" | sed 's/.*:-//')
case "$primary" in
  opencode:*|codex:*) echo "FAIL: primary '$primary' has no measured skill support"; fail=1 ;;
  *) echo "ok: primary $primary" ;;
esac

[ $fail -eq 0 ] && echo "PASS evaluator chain keeps a precondition-free skill-capable lane"
exit $fail
