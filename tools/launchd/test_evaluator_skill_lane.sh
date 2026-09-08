#!/bin/bash
# The evaluator's chain must never degrade onto a harness that cannot load its methodology.
#
# "Act as independent evaluator under sprint-evaluator methodology" names a SKILL — a
# 100-point rubric plus three scripts under .agents/skills/sprint-evaluator/. Measured
# 2026-09-08: pi, run inside a workspace containing that skill, reported none loaded, and the
# docs canary's evaluator failed 3/3 there. The chain was pi -> pi -> codex: every rung
# skill-less, so any Anthropic outage silently swapped a judge with a method for one without.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
fail=0

chain=$(grep -oE 'MISSION_EVALUATOR_FALLBACK:-[^}"]*' "$DRIVER" | sed 's/.*:-//')
[ -n "$chain" ] || { echo "INSTRUMENT BROKEN: cannot read MISSION_EVALUATOR_FALLBACK"; exit 2; }

# Positive control: the string must actually contain rungs, or "no bad rung" is vacuous.
case "$chain" in *:*|opus) : ;; *) echo "INSTRUMENT BROKEN: chain '$chain' has no rungs"; exit 2;; esac

for rung in $(printf '%s' "$chain" | tr ',' ' '); do
  case "$rung" in
    pi:*|opencode:*|codex:*)
      echo "FAIL: evaluator may degrade to '$rung', which does not load .agents/skills/"; fail=1 ;;
    claude:*|opus|sonnet|haiku) echo "ok: $rung" ;;
    *) echo "FAIL: unclassified evaluator rung '$rung' — classify it before shipping"; fail=1 ;;
  esac
done

# And the primary itself.
primary=$(grep -oE 'MISSION_EVALUATOR_MODEL:-[^}"]*' "$DRIVER" | sed 's/.*:-//')
case "$primary" in
  pi:*|opencode:*|codex:*) echo "FAIL: evaluator PRIMARY '$primary' cannot load skills"; fail=1 ;;
  *) echo "ok: primary $primary" ;;
esac

[ $fail -eq 0 ] && echo "PASS every evaluator rung loads the sprint-evaluator skill"
exit $fail
