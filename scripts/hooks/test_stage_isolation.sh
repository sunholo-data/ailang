#!/bin/bash
# The repo's context-injecting hooks must emit NOTHING inside a frozen mission stage.
#
# Both are gated on AILANG_MISSION_STAGE, which internal/mission/dispatch sets on every stage
# task's ExtraEnv. This suite asserts the SHELL half; TestTaskFor_EveryStageIsMarkedFrozenForHooks
# asserts the Go half. Each needs a POSITIVE CONTROL, because a hook that is broken, or whose
# dependency is missing, is also silent — and silence alone would make this suite pass while
# proving nothing.
#
# THE CONTROL MUST ISOLATE THE VARIABLE UNDER TEST (fixed 2026-09-08). brain_on_prompt.sh only
# emits when its top semantic hit clears AILANG_BRAIN_MIN_SCORE (default 0.70), so the original
# control depended on a fixed prompt string continuing to score above a similarity floor against
# a corpus that is reindexed on every release. It stopped doing so — the prompt below now scores
# 0.52 — and the suite reported INSTRUMENT BROKEN for EIGHT consecutive CI runs, including the
# v0.35.3 release, while the hook was working perfectly (863 bytes at MIN_SCORE=0, 0 bytes inside
# a stage). The control now pins MIN_SCORE=0 so the ONLY variable is the mission-stage marker.
# The 0.70 floor is a scoring-policy question, calibrated and tested elsewhere; it is not this
# suite's subject, and coupling to it made a live gate unreadable.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
fail=0
PROMPT_TEXT='how does the mission iteration runtime freeze a work item spec'
PROMPT="{\"prompt\":\"$PROMPT_TEXT\"}"

check() { # $1 = script, $2 = stdin, $3.. = env pinned for BOTH arms of the control
  local name="$1"; local s="$HERE/$1"; shift
  local input="$1"; shift
  local on off
  off=$(printf '%s' "$input" | env "$@" bash "$s" 2>/dev/null | wc -c | tr -d ' ')
  on=$(printf '%s' "$input" | env "$@" AILANG_MISSION_STAGE=1 bash "$s" 2>/dev/null | wc -c | tr -d ' ')
  if [ "$off" -eq 0 ]; then
    echo "INSTRUMENT BROKEN: $name is silent WITHOUT the marker — control failed, gate proves nothing"; fail=1; return
  fi
  if [ "$on" -ne 0 ]; then
    echo "FAIL: $name emitted $on bytes inside a mission stage"; fail=1; return
  fi
  echo "PASS: $name — $off bytes normally, 0 inside a stage"
}

# brain_on_prompt.sh cannot emit at ANY threshold without an indexed brain corpus, which a fresh
# CI clone does not have. That is "cannot evaluate here", not "broken" — so probe for the corpus
# and skip loudly, the same distinction TestDriverCopiesDoNotMultiply draws for sibling checkouts.
# A reachable corpus that still yields a silent hook IS a real failure and still fails below.
brain_corpus_reachable() {
  command -v ailang >/dev/null 2>&1 || return 1
  command -v jq     >/dev/null 2>&1 || return 1
  local n
  n=$(ailang cache search -json -limit 1 -scope both "$PROMPT_TEXT" 2>/dev/null \
        | jq -r '(.results // []) | length' 2>/dev/null)
  [ "${n:-0}" -ge 1 ]
}

if brain_corpus_reachable; then
  check brain_on_prompt.sh "$PROMPT" AILANG_BRAIN_MIN_SCORE=0
else
  echo "SKIP: brain_on_prompt.sh — no brain corpus reachable, so the positive control cannot be established here (expected on a fresh clone/CI runner)"
fi
check session_start.sh ""
exit $fail
