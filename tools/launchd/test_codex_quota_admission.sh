#!/bin/bash
# Exercise the real probe and role preflight with no provider calls.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
log() { :; }
PROBE_TIMEOUT=1
PROBES=0
BLOCKED=1
_mc_is_over_ration() { [ "$BLOCKED" = 1 ]; }
_mc_bounded() { PROBES=$((PROBES+1)); MC_BOUNDED_OUT=ok; return 0; }
eval "$(awk '/^_mc_probe_codex\(\) \{/,/^\}$/' "$DRIVER")"
_mc_probe_codex test-model; rc=$?
[ "$rc" = 75 ] && [ "$PROBES" = 0 ] || { echo 'FAIL blocked probe spent inference'; exit 1; }
BLOCKED=0
_mc_probe_codex test-model
[ "$PROBES" = 1 ] || { echo 'FAIL positive control did not call provider'; exit 1; }
BLOCKED=1
MISSION_DESIGNER_MODEL=claude:test
MISSION_PLANNER_MODEL=codex:test-model
MISSION_EXECUTOR_MODEL=codex:test-model
MISSION_EVALUATOR_MODEL=claude:test
MISSION_PLANNER_FALLBACK=pi:ollama/test
MISSION_EXECUTOR_FALLBACK=pi:ollama/test
MISSION_PLANNER_FALLBACK_CHAIN=pi:ollama/test
MISSION_EXECUTOR_FALLBACK_CHAIN=pi:ollama/test
_cx_probed=:; _cx_failed=:; _cx_rcmap=''; _lane_degraded=''
QUOTA_SIG=quota
_chain_head() { printf '%s' "${1%%,*}"; }
_chain_tail() { printf ''; }
block=$(awk '/^for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do/{active=1;buf=""} active{buf=buf $0 "\n"} active && /^done$/{if (buf ~ /cx_model=/) printf "%s",buf;active=0}' "$DRIVER")
[ -n "$block" ] || { echo 'FAIL extraction'; exit 1; }
eval "$block"
[ "$PROBES" = 1 ] && [ "$MISSION_PLANNER_MODEL" = pi:ollama/test ] && [ "$MISSION_EXECUTOR_MODEL" = pi:ollama/test ] || { echo "FAIL role routing: $PROBES $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL"; exit 1; }
echo 'PASS Codex admission: blocked probe spends zero; planner and executor fall back; usable positive control calls provider'
