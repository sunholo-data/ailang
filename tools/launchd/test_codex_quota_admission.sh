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
eval "$(awk '/^_mc_probe_codex\(\) \{/,/^\}$/' "$DRIVER" "$HERE/lib/lane-probe.sh")"
# The codex notice line names the ration reason when the probe was ration-blocked (rc=75).
eval "$(awk '/^_mc_ration_reason\(\) \{/,/^\}$/' "$DRIVER" "$HERE/lib/lane-probe.sh")"
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
# Anthropic is dry in this case, so the executor's Anthropic rung (2026-09-29; haiku-5-5 since 2026-10-08) is probed,
# refused, and the role walks to pi — the path this case has always asserted.
_mc_probe() { return 1; }
eval "$block"
[ "$PROBES" = 1 ] && [ "$MISSION_PLANNER_MODEL" = pi:ollama/test ] && [ "$MISSION_EXECUTOR_MODEL" = pi:ollama/test ] || { echo "FAIL role routing: $PROBES $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL"; exit 1; }
echo 'PASS Codex admission: blocked probe spends zero; planner and executor fall back; usable positive control calls provider'

# Reset credits in reserve (2026-09-24): `--over` appends the clause to a blocked bucket's
# reason; _mc_reset_hint must lift exactly that clause, and nothing when none is held.
body=$(awk '$0 == "_mc_reset_hint() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER" "$HERE/lib/lane-probe.sh")
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

# OPUS BEFORE PI (2026-09-26): with the knob on, a dry codex hands planner/executor to opus
# when the Anthropic probe passes, and to the pi chain when it does not. Knob off = unchanged.
# EXECUTOR RUNG 2 (2026-09-29; model claude-haiku-5-5 since 2026-10-08): the EXECUTOR takes claude:claude-haiku-5-5 ahead of both
# opus and pi whenever its probe passes, knob or no knob; MISSION_EXECUTOR_ANTHROPIC_RUNG=''
# turns it off. The planner is untouched by the rung.
_opb_run() {  # $1 = knob, $2 = anthropic probe rc, $3 = executor rung (omit = driver default)
  OPB_RC="$2"; AN_PROBES=0
  _mc_probe() { AN_PROBES=$((AN_PROBES+1)); return "$OPB_RC"; }
  MISSION_OPUS_BEFORE_PI="$1"
  if [ $# -lt 3 ]; then unset MISSION_EXECUTOR_ANTHROPIC_RUNG; else MISSION_EXECUTOR_ANTHROPIC_RUNG="$3"; fi
  MISSION_PLANNER_MODEL=codex:test-model; MISSION_EXECUTOR_MODEL=codex:test-model
  unset MISSION_PLANNER_CHAIN_REMAINING MISSION_EXECUTOR_CHAIN_REMAINING
  _cx_probed=:; _cx_failed=:; _cx_rcmap=''; _lane_degraded=''; _an_probed=:; _an_failed=:
  eval "$block"
}
_opb_run 1 0
[ "$MISSION_PLANNER_MODEL" = opus ] && [ "$MISSION_EXECUTOR_MODEL" = claude:claude-haiku-5-5 ] && [ "$AN_PROBES" = 2 ] \
  || { echo "FAIL opus-before-pi positive: $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL probes=$AN_PROBES"; exit 1; }
case "$_lane_degraded" in *'handed to `opus`'*) ;; *) echo "FAIL ledger does not name opus"; exit 1 ;; esac
case "$_lane_degraded" in *'handed to `claude:claude-haiku-5-5`'*) ;; *) echo "FAIL ledger does not name the haiku rung"; exit 1 ;; esac
[ "$MISSION_EXECUTOR_CHAIN_REMAINING" = "pi:ollama/test" ] \
  || { echo "FAIL haiku rung dropped the pi head from the remaining chain: [$MISSION_EXECUTOR_CHAIN_REMAINING]"; exit 1; }
_opb_run 1 75
[ "$MISSION_PLANNER_MODEL" = pi:ollama/test ] && [ "$MISSION_EXECUTOR_MODEL" = pi:ollama/test ] && [ "$AN_PROBES" = 2 ] \
  || { echo "FAIL opus-before-pi drought: $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL probes=$AN_PROBES"; exit 1; }
_opb_run 0 0
[ "$MISSION_PLANNER_MODEL" = pi:ollama/test ] && [ "$MISSION_EXECUTOR_MODEL" = claude:claude-haiku-5-5 ] && [ "$AN_PROBES" = 1 ] \
  || { echo "FAIL knob off: planner must walk pi, executor still takes the rung: $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL probes=$AN_PROBES"; exit 1; }
_opb_run 0 0 ''
[ "$MISSION_PLANNER_MODEL" = pi:ollama/test ] && [ "$MISSION_EXECUTOR_MODEL" = pi:ollama/test ] && [ "$AN_PROBES" = 0 ] \
  || { echo "FAIL rung disabled + knob off changed routing: $MISSION_PLANNER_MODEL $MISSION_EXECUTOR_MODEL probes=$AN_PROBES"; exit 1; }
echo 'PASS opus-before-pi + executor rung 2: haiku-5-5 takes a dry executor, opus the planner; Anthropic dry walks pi; rung and knob off unchanged'

# log() must be defined before its first caller, or bash resolves macOS /usr/bin/log.
first_def=$(grep -n '^log() {' "$DRIVER" | head -1 | cut -d: -f1)
first_call=$(grep -n '^[[:space:]]*log "' "$DRIVER" | head -1 | cut -d: -f1)
[ -n "$first_def" ] && [ -n "$first_call" ] && [ "$first_def" -lt "$first_call" ] \
  || { echo "FAIL log() defined at line $first_def but first called at line $first_call"; exit 1; }
echo 'PASS log() is defined before its first caller'
