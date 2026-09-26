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
# The codex notice line names the ration reason when the probe was ration-blocked (rc=75).
eval "$(awk '/^_mc_ration_reason\(\) \{/,/^\}$/' "$DRIVER")"
type _mc_ration_reason >/dev/null 2>&1 || { echo 'FAIL extraction _mc_ration_reason'; exit 1; }
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

# Reset credits in reserve (2026-09-24): `--over` appends the clause to a blocked bucket's
# reason; _mc_reset_hint must lift exactly that clause, and nothing when none is held.
body=$(awk '$0 == "_mc_reset_hint() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER")
[ -n "$body" ] || { echo "FAIL extraction _mc_reset_hint"; exit 1; }
eval "$body"
MC_RATION_REASONS='codex over: provider-reported Codex account usage exceeds ration or exhausts a window; 1 Codex reset credit(s) in reserve (next expires 2026-10-22) — attended only: `ailang mission quota --codex-reset --yes`
ollama over: Ollama Cloud weekly gauge consumed 14.6pp in the last 23h, over the 10pp/day ration; new cloud routing blocked'
hint=$(_mc_reset_hint)
case "$hint" in
  "1 Codex reset credit(s) in reserve (next expires 2026-10-22) — attended only: \`ailang mission quota --codex-reset --yes\`") ;;
  *) echo "FAIL reset hint: got [$hint]"; exit 1 ;;
esac
MC_RATION_REASONS='codex over: provider-reported Codex account usage exceeds ration or exhausts a window'
[ -z "$(_mc_reset_hint)" ] || { echo "FAIL reset hint printed with no credit held"; exit 1; }
echo 'PASS reset credits in reserve are lifted into the notice, and absent when none are held'
