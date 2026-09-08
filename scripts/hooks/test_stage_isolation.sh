#!/bin/bash
# The repo's context-injecting hooks must emit NOTHING inside a frozen mission stage.
#
# Both are gated on AILANG_MISSION_STAGE, which internal/mission/dispatch sets on every stage
# task's ExtraEnv. This suite asserts the SHELL half; TestTaskFor_EveryStageIsMarkedFrozenForHooks
# asserts the Go half. Each needs a POSITIVE CONTROL, because a hook that is broken, or whose
# dependency is missing, is also silent — and silence alone would make this suite pass while
# proving nothing.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
fail=0
PROMPT='{"prompt":"how does the mission iteration runtime freeze a work item spec"}'

check() { # $1 = script, $2 = stdin
  local s="$HERE/$1" on off
  off=$(printf '%s' "$2" | bash "$s" 2>/dev/null | wc -c | tr -d ' ')
  on=$(printf '%s' "$2" | AILANG_MISSION_STAGE=1 bash "$s" 2>/dev/null | wc -c | tr -d ' ')
  if [ "$off" -eq 0 ]; then
    echo "INSTRUMENT BROKEN: $1 is silent WITHOUT the marker — control failed, gate proves nothing"; fail=1; return
  fi
  if [ "$on" -ne 0 ]; then
    echo "FAIL: $1 emitted $on bytes inside a mission stage"; fail=1; return
  fi
  echo "PASS: $1 — $off bytes normally, 0 inside a stage"
}

check brain_on_prompt.sh "$PROMPT"
check session_start.sh ""
exit $fail
