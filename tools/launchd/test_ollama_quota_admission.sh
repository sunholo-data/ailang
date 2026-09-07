#!/bin/bash
# Execute real Pi quota admission with offline fixtures, including local-model control.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
log(){ :; }
PROBE_TIMEOUT=1
PROBES=0
MC_OVER_RATION=ollama
_mc_load_ration(){ :; }
_mc_bounded(){ PROBES=$((PROBES+1));MC_BOUNDED_OUT=ok;return 0; }
for fn in _mc_rung_bucket _mc_is_over_ration _mc_probe_pi; do
 body=$(awk -v f="$fn" '$0 == f "() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER")
 [ -n "$body" ] || { echo "FAIL extraction $fn";exit 1; }
 eval "$body"
done
for model in ollama/kimi-k3:cloud ollama/deepseek-v4-flash:0731-cloud; do
 _mc_probe_pi "$model";rc=$?
 [ "$rc" = 75 ] && [ "$PROBES" = 0 ] || { echo 'FAIL cloud probe spent inference';exit 1; }
done
_mc_probe_pi ollama/gemma4:27b
_mc_probe_pi openrouter/z-ai/glm-5.3
[ "$PROBES" = 2 ] || { echo 'FAIL local/fallback positive controls';exit 1; }
# Protect the actual role-loop wiring as well as the probe function.
role=$(awk '/^for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do/{on=1;buf=""} on{buf=buf $0 "\n"} on && /^done$/{if(buf ~ /pi_model=/)print buf;on=0}' "$DRIVER")
[ -n "$role" ] && printf '%s' "$role" | grep -q '_mc_probe_pi "$pi_model"' || { echo 'FAIL role loop bypass';exit 1; }
echo 'PASS Ollama cloud admission spends zero; local Ollama and OpenRouter remain reachable; role loop uses shared gate'
# Run the extracted role loop: cloud planner and executor must advance to metered twins.
MISSION_DESIGNER_MODEL=claude:test
MISSION_PLANNER_MODEL=pi:ollama/kimi-k3:cloud
MISSION_EXECUTOR_MODEL=pi:ollama/deepseek-v4-flash:0731-cloud
MISSION_EVALUATOR_MODEL=claude:test
MISSION_PLANNER_CHAIN_REMAINING=pi:openrouter/moonshot/kimi-k3
MISSION_EXECUTOR_CHAIN_REMAINING=pi:openrouter/deepseek/v4-flash
_pi_probed=:;_pi_failed=:;_pi_rcmap='';_lane_degraded=''
_chain_head(){ printf '%s' "${1%%,*}"; }
_chain_tail(){ printf ''; }
eval "$role"
[ "$PROBES" = 4 ] && [ "$MISSION_PLANNER_MODEL" = pi:openrouter/moonshot/kimi-k3 ] && [ "$MISSION_EXECUTOR_MODEL" = pi:openrouter/deepseek/v4-flash ] || { echo 'FAIL actual role fallback';exit 1; }
# A failed quota command blocks both protected subscriptions, without an API call.
eval "$(awk '/^_mc_load_ration\(\) \{/,/^\}$/' "$DRIVER")"
_mc_bounded(){ MC_BOUNDED_OUT='unavailable';return 124; }
MC_OVER_RATION_READ=0
_mc_load_ration
[ "$MC_OVER_RATION" = 'codex ollama' ] || { echo 'FAIL quota command outage opened admission';exit 1; }
echo 'PASS actual planner/executor fallbacks and quota-command outage admission'
# Absolute and file controller pins cannot bypass cloud admission.
_mc_load_ration(){ :; }
MC_OVER_RATION=ollama
_mc_is_demoted(){ return 1; }
_mc_canon_id(){ printf '%s' "$1"; }
_mc_probe(){ return 0; }
for fn in _mc_set_controller select_model; do
 eval "$(awk -v f="$fn" '$0 == f "() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER")"
done
TEMP=$(mktemp -d)
trap 'rm -rf "$TEMP"' EXIT
OVERRIDE_FILE="$TEMP/controller"
PREFS=claude:healthy
CONTROLLER_FALLBACK=''
MISSION_MODEL=pi:ollama/kimi-k3:cloud
select_model
[ "$CONTROLLER_ID" = claude:healthy ] || { echo 'FAIL absolute pin bypass';exit 1; }
MISSION_MODEL=''
printf '%s\n' pi:ollama/kimi-k3:cloud > "$OVERRIDE_FILE"
select_model
[ "$CONTROLLER_ID" = claude:healthy ] || { echo 'FAIL file pin bypass';exit 1; }
EXEC_ONCE_FILE="$TEMP/executor"
printf '%s\n' pi:ollama/kimi-k3:cloud > "$EXEC_ONCE_FILE"
MISSION_EXECUTOR_MODEL=pi:openrouter/safe
once_block=$(awk '/^if \[ -f "\$EXEC_ONCE_FILE" \]; then/{on=1} on{print} on && /^fi$/{exit}' "$DRIVER")
[ -n "$once_block" ] || { echo 'FAIL one-shot extraction';exit 1; }
eval "$once_block"
[ "$MISSION_EXECUTOR_MODEL" = pi:openrouter/safe ] && [ -f "$EXEC_ONCE_FILE" ] || { echo 'FAIL one-shot bypass/lost override';exit 1; }
echo 'PASS controller pins and deferred one-shot override respect admission'
