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
for fn in _mc_rung_bucket _mc_is_over_ration _mc_probe; do
 body=$(awk -v f="$fn" '$0 == f "() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER")
 [ -n "$body" ] || { echo "FAIL extraction $fn";exit 1; }
 eval "$body"
done

# Both spellings of an Anthropic rung: the explicit claude: pin and the bare Agent alias.
for model in claude-fable-5-1 sonnet opus claude-opus-5; do
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
printf '%s' "$role" | grep -q '_mc_probe "$an_model"' || { echo 'FAIL role loop bypasses shared gate';exit 1; }

MISSION_DESIGNER_MODEL=claude:claude-fable-5-1
MISSION_PLANNER_MODEL=codex:gpt-5.6-sol
MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol
MISSION_EVALUATOR_MODEL=sonnet
MISSION_DESIGNER_FALLBACK=codex:gpt-6-astra
MISSION_EVALUATOR_FALLBACK=pi:ollama/minimax-m3:cloud
_an_probed=:;_an_failed=:;_an_rcmap='';_lane_degraded=''
_chain_head(){ printf '%s' "${1%%,*}"; }
_chain_tail(){ printf ''; }
eval "$role"
[ "$PROBES" = 0 ] || { echo "FAIL role loop spent $PROBES Anthropic calls while over ration";exit 1; }
[ "$MISSION_DESIGNER_MODEL" = codex:gpt-6-astra ] || { echo "FAIL designer stayed on $MISSION_DESIGNER_MODEL";exit 1; }
[ "$MISSION_EVALUATOR_MODEL" = pi:ollama/minimax-m3:cloud ] || { echo "FAIL evaluator stayed on $MISSION_EVALUATOR_MODEL";exit 1; }
# codex:* roles are the other loop's business and must be untouched by this one.
[ "$MISSION_PLANNER_MODEL" = codex:gpt-5.6-sol ] || { echo 'FAIL anthropic loop rewrote a codex rung';exit 1; }
# The ledger must name the ration, not report a broken lane: they resume differently.
printf '%s' "$_lane_degraded" | grep -q 'over daily ration' || { echo "FAIL degradation ledger hid the cause: $_lane_degraded";exit 1; }
echo 'PASS role loop yields Anthropic lanes to their chains and reports the ration as the cause'

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
