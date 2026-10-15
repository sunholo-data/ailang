#!/bin/bash
# Execute real Anthropic quota admission with offline fixtures.
#
# The gap this pins: _mc_set_controller has consulted the ration for the CONTROLLER since
# the ration landed, but the role pre-flight calls _mc_probe directly, so the designer and
# evaluator kept probing and spending Anthropic while the controller was yielding for
# exactly that reason. A half-walked ration reads as a working one in every log line.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
log(){ :; }
PROBE_TIMEOUT=1
PROBES=0
QUOTA_SIG='usage limit'
MC_OVER_RATION=anthropic
_mc_load_ration(){ :; }
_mc_bounded(){ PROBES=$((PROBES+1));MC_BOUNDED_OUT=ok;return 0; }
# _mc_probe gained a call to _mc_ration_unreadable (which calls _mc_ration_reason) when the
# Anthropic CLI-usage fallback landed, so both must be extracted or _mc_probe runs with an
# undefined function and this suite reports a degradation-ledger defect that does not exist.
for fn in _mc_rung_bucket _mc_is_over_ration _mc_ration_reason _mc_ration_unreadable _mc_probe; do
 body=$(awk -v f="$fn" '$0 == f "() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER" "$HERE/lib/lane-probe.sh")
 [ -n "$body" ] || { echo "FAIL extraction $fn";exit 1; }
 eval "$body"
done

# Both spellings of an Anthropic rung: the explicit claude: pin and the bare Agent alias.
for model in claude-fable-5-1 sonnet opus claude-opus-5-5; do
 _mc_probe "$model";rc=$?
 [ "$rc" = 75 ] || { echo "FAIL $model probed at rc=$rc, want 75 (ration)";exit 1; }
done
[ "$PROBES" = 0 ] || { echo "FAIL ration-blocked probes spent $PROBES inference calls";exit 1; }

# Positive control: the SAME function must still probe when the bucket is not over. An
# empty search is a claim, so prove the instrument can return 0 before trusting the 75s.
MC_OVER_RATION=codex
_mc_probe sonnet;rc=$?
[ "$rc" = 0 ] && [ "$PROBES" = 1 ] || { echo "FAIL positive control rc=$rc probes=$PROBES";exit 1; }
echo 'PASS Anthropic admission spends zero when over ration, and still probes when not'

# Protect the ROLE-loop wiring, not just the probe function: this is the path that was
# bypassing the gate, and a test of the function alone would have passed all along.
MC_OVER_RATION=anthropic
PROBES=0
role=$(awk '/^for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do/{on=1;buf=""} on{buf=buf $0 "\n"} on && /^done$/{if(buf ~ /an_model=/)print buf;on=0}' "$DRIVER")
[ -n "$role" ] || { echo 'FAIL anthropic role loop extraction';exit 1; }
# The extracted block is itself `for role in ... EVALUATOR; do`, so eval'ing it overwrites
# $role with "EVALUATOR" — a second eval would then run that as a command. Hold the source
# in a name the loop cannot clobber.
_role_src="$role"
printf '%s' "$role" | grep -q '_mc_probe "$an_model"' || { echo 'FAIL role loop bypasses shared gate';exit 1; }

MISSION_DESIGNER_MODEL=claude:claude-fable-5-1
MISSION_PLANNER_MODEL=codex:gpt-5.6-sol
MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol
MISSION_EVALUATOR_MODEL=sonnet
MISSION_DESIGNER_FALLBACK=codex:gpt-6-astra
MISSION_EVALUATOR_FALLBACK=pi:ollama/minimax-m3:cloud
# _mc_ration_reason reads MC_RATION_REASONS, which _mc_load_ration populates in production
# and the stub above does not. Supply it, or the arm below asserts on a state the fleet cannot
# reach (over ration with no reason line) and reads as a defect that is really a missing input.
MC_RATION_REASONS='anthropic over 12.0% used / 11.4% allowed'
_an_probed=:;_an_failed=:;_an_rcmap='';_lane_degraded=''
_chain_head(){ printf '%s' "${1%%,*}"; }
_chain_tail(){ printf ''; }
eval "$_role_src"
[ "$PROBES" = 0 ] || { echo "FAIL role loop spent $PROBES Anthropic calls while over ration";exit 1; }
[ "$MISSION_DESIGNER_MODEL" = codex:gpt-6-astra ] || { echo "FAIL designer stayed on $MISSION_DESIGNER_MODEL";exit 1; }
[ "$MISSION_EVALUATOR_MODEL" = pi:ollama/minimax-m3:cloud ] || { echo "FAIL evaluator stayed on $MISSION_EVALUATOR_MODEL";exit 1; }
# codex:* roles are the other loop's business and must be untouched by this one.
[ "$MISSION_PLANNER_MODEL" = codex:gpt-5.6-sol ] || { echo 'FAIL anthropic loop rewrote a codex rung';exit 1; }
# The ledger must name the ration, not report a broken lane: they resume differently.
printf '%s' "$_lane_degraded" | grep -q 'over daily ration' || { echo "FAIL degradation ledger hid the cause: $_lane_degraded";exit 1; }
echo 'PASS role loop yields Anthropic lanes to their chains and reports the ration as the cause'

# THE SPLIT IS THE POINT, and it decides whether to SPEND, not just what to print. "Over" is
# a measurement: probing anyway spends against a limit we know we have passed. "Unknown" is
# the ABSENCE of a measurement, and refusing on it idles a lane that may be perfectly healthy
# — which is exactly what happened for a week in September 2026, when an expired Anthropic
# credential read as an exhausted quota and three World iterations were routed away from a
# bucket sitting at ~89% free.
#
# So under an UNREADABLE quota the role loop must PROBE and leave the lane alone, where under
# an OVER one (the arm above) it must refuse without spending a call.
PROBES=0
MC_RATION_REASONS='anthropic unknown credential unreadable'
MISSION_DESIGNER_MODEL=claude:claude-fable-5-1
MISSION_EVALUATOR_MODEL=sonnet
MISSION_DESIGNER_FALLBACK=codex:gpt-6-astra
MISSION_EVALUATOR_FALLBACK=pi:ollama/minimax-m3:cloud
_an_probed=:;_an_failed=:;_an_rcmap='';_lane_degraded=''
eval "$_role_src"
[ "$PROBES" -gt 0 ] \
  || { echo "FAIL an UNREADABLE quota refused without probing — that is the September 2026 defect";exit 1; }
[ "$MISSION_DESIGNER_MODEL" = claude:claude-fable-5-1 ] \
  || { echo "FAIL an UNREADABLE quota displaced the designer to $MISSION_DESIGNER_MODEL";exit 1; }
printf '%s' "$_lane_degraded" | grep -q 'over daily ration' \
  && { echo "FAIL an unreadable quota was reported as OVER — they resume differently: $_lane_degraded";exit 1; }
echo "PASS an UNREADABLE Anthropic quota PROBES the lane (${PROBES} call(s)) instead of refusing it"

# CONTROLLER SELECTION gets the same split (2026-10-15, triage's first armed fire: anthropic
# `unknown` on a 429, haiku executor probed, sonnet controller refused unprobed).
body=$(awk '$0 == "_mc_ration_refuses() {" {on=1} on {print} on && /^}$/ {exit}' "$HERE/lib/lane-probe.sh")
[ -n "$body" ] || { echo 'FAIL extraction _mc_ration_refuses';exit 1; }
eval "$body"
MC_OVER_RATION='codex anthropic openrouter'
MC_RATION_REASONS='anthropic unknown: usage endpoint returned HTTP 429'
for model in claude-sonnet-5-5 sonnet claude:claude-opus-5-5; do
 _mc_ration_refuses "$model" && { echo "FAIL controller refuses $model on an UNREADABLE quota";exit 1; }
done
# Only Anthropic is excused: a codex rung over ration, and a metered rung, still refuse.
_mc_ration_refuses codex:gpt-6.1-sol || { echo 'FAIL over-ration codex rung admitted';exit 1; }
_mc_ration_refuses pi:openrouter/z-ai/glm-5.3 || { echo 'FAIL over-ration openrouter rung admitted';exit 1; }
MC_RATION_REASONS='anthropic over 12.0% used / 11.4% allowed'
_mc_ration_refuses sonnet || { echo 'FAIL controller admitted a measurably OVER anthropic rung';exit 1; }
# Positive control: a rung whose bucket is not blocked at all is never refused.
MC_OVER_RATION='anthropic'
_mc_ration_refuses codex:gpt-6.1-sol && { echo 'FAIL controller refused a rung in an unblocked bucket';exit 1; }
# Wiring: every controller-selection gate in select_model uses the shared verdict.
sel=$(awk '/^select_model\(\) \{/{on=1} on{print} on && /^}$/{exit}' "$DRIVER")
[ -n "$sel" ] || { echo 'FAIL select_model extraction';exit 1; }
printf '%s' "$sel" | grep -q '_mc_is_over_ration' && { echo 'FAIL select_model still gates on _mc_is_over_ration';exit 1; }
[ "$(printf '%s' "$sel" | grep -c '_mc_ration_refuses')" -ge 4 ] || { echo 'FAIL select_model lost a _mc_ration_refuses gate';exit 1; }
echo 'PASS controller selection probes an UNREADABLE Anthropic quota and still refuses an OVER one'

# SPAWN EXPORT gets the same split (2026-10-15: haiku executor admitted by the driver, then
# refused by resolve-role-spawn as over-ration:anthropic because the export still listed it).
body=$(awk '$0 == "_mc_spawn_over_ration() {" {on=1} on {print} on && /^}$/ {exit}' "$HERE/lib/lane-probe.sh")
[ -n "$body" ] || { echo 'FAIL extraction _mc_spawn_over_ration';exit 1; }
eval "$body"
MC_OVER_RATION='codex anthropic openrouter'
MC_RATION_REASONS='anthropic unknown: usage endpoint returned HTTP 429'
[ "$(_mc_spawn_over_ration)" = 'codex openrouter' ] || { echo "FAIL spawn export kept an UNREADABLE anthropic: '$(_mc_spawn_over_ration)'";exit 1; }
MC_RATION_REASONS='anthropic over 12.0% used / 11.4% allowed'
[ "$(_mc_spawn_over_ration)" = 'codex anthropic openrouter' ] || { echo "FAIL spawn export dropped an OVER anthropic: '$(_mc_spawn_over_ration)'";exit 1; }
grep -q '^MISSION_OVER_RATION=$(_mc_spawn_over_ration); export MISSION_OVER_RATION$' "$DRIVER" \
  || { echo 'FAIL driver exports MISSION_OVER_RATION without _mc_spawn_over_ration';exit 1; }
# End to end through the resolver: a claude: executor recipe is emitted, not refused.
out=$(MISSION_OVER_RATION='codex openrouter' MISSION_EXECUTOR_MODEL=claude:claude-haiku-5-5 /bin/bash "$HERE/resolve-role-spawn.sh" executor 2>/dev/null)
case "$out" in "recipe claude:claude-haiku-5-5"*) : ;; *) echo "FAIL resolver on the fixed export: $out";exit 1 ;; esac
echo 'PASS spawn export drops an UNREADABLE Anthropic bucket, keeps an OVER one, and the resolver emits the recipe'

# ORDERING: the kill switch must be reached BEFORE any inference probe.
#
# It is numbered gate 1 and was gate 1, but the role-probe block was inserted above it, so a
# disabled mission went on probing every provider before discovering it was off — 19 minutes
# and four Anthropic probes on the docs fire of 2026-09-08. A line-order assertion is crude,
# but the property is a line-order property: there is no behaviour to observe once the two
# are the wrong way round except wasted spend.
ks=$(grep -n '^# 1\. Kill switch' "$DRIVER" | head -1 | cut -d: -f1)
probes=$(grep -n '^fi # legacy role probes' "$DRIVER" | head -1 | cut -d: -f1)
[ -n "$ks" ] && [ -n "$probes" ] || { echo 'FAIL could not locate kill switch or probe block';exit 1; }
[ "$ks" -lt "$probes" ] || { echo "FAIL kill switch (line $ks) is BELOW the probe block (line $probes) — a disabled mission would probe first";exit 1; }
echo "PASS kill switch (line $ks) precedes the role probes (line $probes)"
