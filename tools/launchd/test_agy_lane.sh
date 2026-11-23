#!/bin/bash
# The agy: role lane (internal/executor/agy via `ailang exec agy`), offline.
# Executes the REAL functions and role loop extracted from the driver and lane-probe.sh,
# with a stub `ailang` on PATH, so a probe verdict is the executor's JSON and nothing
# here spends a token. bash 3.2.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
DRIVER="$HERE/mission-control.sh"
LIB="$HERE/lib/lane-probe.sh"
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
fail() { echo "FAIL $*"; exit 1; }
log() { printf '%s\n' "$*" >> "$T/log"; }

extract() { awk -v f="$1" '$0 == f "() {" {on=1} on {print} on && /^}$/ {exit}' "$DRIVER" "$LIB"; }
for fn in _mc_bounded _mc_rung_bucket _mc_is_over_ration _mc_ration_reason _mc_ration_unreadable _mc_probe_agy; do
  body=$(extract "$fn"); [ -n "$body" ] || fail "extraction $fn"; eval "$body"
done
_mc_load_ration() { :; }

# 1. Buckets: an agy pin must never fall through to the anthropic bucket.
[ "$(_mc_rung_bucket agy:gemini-4-neon-high)" = google ] || fail "subscription pin bucket"
[ "$(_mc_rung_bucket agy:api/gemini-4-neon-high)" = gemini-api ] || fail "api-key pin bucket"
echo "PASS agy pins classify as google / gemini-api, not anthropic"

# 2. The probe's verdict is the executor's JSON success, not an exit code.
mkdir -p "$T/bin" "$T/home/.config/ailang"
: > "$T/home/.config/ailang/secrets.env"
cat > "$T/bin/ailang" <<EOF
#!/bin/sh
printf '%s\n' "\$@" > "$T/argv"
cat "$T/reply"
exit "\$(cat "$T/rc")"
EOF
chmod +x "$T/bin/ailang"
PATH="$T/bin:$PATH"; HOME="$T/home"; PROBE_TIMEOUT=20; MC_OVER_RATION=""; MC_DRIVER_ROOT="$ROOT"
export PATH HOME

printf '{"success": true, "error": ""}\n' > "$T/reply"; echo 0 > "$T/rc"
_mc_probe_agy gemini-4-neon-high; rc=$?
[ "$rc" = 0 ] || fail "success JSON read as rc=$rc ($MC_BOUNDED_OUT)"
grep -qx exec "$T/argv" && grep -qx agy "$T/argv" && grep -qx gemini-4-neon-high "$T/argv" \
  || fail "probe did not run 'ailang exec agy --model gemini-4-neon-high': $(tr '\n' ' ' < "$T/argv")"
grep -qx -- --json "$T/argv" || fail "probe must ask for --json"

# exit 0 with success:false (a soft denial, a missing sign-in) must be unusable.
printf '{"success": false, "error": "agy is not signed in for the subscription route: run make agy-signin"}\n' > "$T/reply"; echo 0 > "$T/rc"
_mc_probe_agy gemini-4-neon-high; rc=$?
[ "$rc" = 2 ] || fail "rc-0 success:false read as rc=$rc"
grep -q "make agy-signin" "$T/log" || fail "probe failure did not log the executor's error"

printf 'Error: unknown provider "agy"\n' > "$T/reply"; echo 1 > "$T/rc"
_mc_probe_agy gemini-4-neon-high; rc=$?
[ "$rc" = 2 ] || fail "an ailang without the agy executor read as rc=$rc"
echo "PASS probe verdict = executor JSON success (rc-0 failure and old binary both unusable)"

# 3. A blocked bucket refuses without spending a probe.
: > "$T/argv"; MC_OVER_RATION="google"
_mc_probe_agy gemini-4-neon-high; rc=$?
[ "$rc" = 75 ] && [ ! -s "$T/argv" ] || fail "over-ration google bucket still probed (rc=$rc)"
MC_OVER_RATION=""
echo "PASS over-ration agy rung refuses without a probe"

# 3b. The REAL ration loader: `ailang mission quota --over` printing google blocks the
#     subscription rung, and only that rung — agy:api/* is the metered key (gemini-api).
eval "$(awk '/^_mc_load_ration\(\) \{/,/^\}$/' "$LIB")"
_mc_bounded_real=$(declare -f _mc_bounded)
_mc_bounded() { MC_BOUNDED_OUT=$(printf 'google\nquota: google over: Google weekly usage 3.0%% exceeds the 15%%/day pace line 0.3%%\n'); return 0; }
MC_OVER_RATION_READ=0; MC_OVER_RATION=""
_mc_load_ration
case " $MC_OVER_RATION " in *" google "*) : ;; *) fail "loader dropped the google bucket: '$MC_OVER_RATION'" ;; esac
printf '%s' "$MC_RATION_REASONS" | grep -q '^google over' || fail "loader dropped the google reason line"
_mc_is_over_ration agy:gemini-4-neon-high || fail "over-ration google did not refuse the subscription rung"
_mc_is_over_ration agy:api/gemini-4-neon-high && fail "google ration wrongly refused the metered api-key rung"
# A failed quota command fails CLOSED on google, like codex and ollama.
_mc_bounded() { MC_BOUNDED_OUT="boom"; return 3; }
MC_OVER_RATION_READ=0; MC_OVER_RATION=""
_mc_load_ration
case " $MC_OVER_RATION " in *" google "*) : ;; *) fail "quota outage left google open: '$MC_OVER_RATION'" ;; esac
eval "$_mc_bounded_real"
_mc_load_ration() { :; }; MC_OVER_RATION=""
echo "PASS ration loader: google line blocks the subscription rung only; quota outage fails closed"

# 4. The REAL role loop: an agy executor pin is probed by _mc_probe_agy, and a failure
#    advances along MISSION_<ROLE>_CHAIN_REMAINING, never to the anthropic pre-flight.
ROLE_LOOP=$(awk '/^for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do/{on=1;buf=""} on{buf=buf $0 "\n"} on && /^done$/{if(buf ~ /pi_model=/)print buf;on=0}' "$DRIVER")
[ -n "$ROLE_LOOP" ] || fail "role loop extraction"
printf '%s' "$ROLE_LOOP" | grep -q '_mc_probe_agy "$pi_model"' || fail "role loop does not probe agy pins"
an_loop=$(awk '/^for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do/{on=1;buf=""} on{buf=buf $0 "\n"} on && /^done$/{if(buf ~ /an_model=/)print buf;on=0}' "$DRIVER")
printf '%s' "$an_loop" | grep -q '""|codex:\*|pi:\*|agy:\*) continue' || fail "anthropic pre-flight would probe an agy pin as a claude model"

AGY_PROBES=0
_mc_probe_agy() { AGY_PROBES=$((AGY_PROBES+1)); MC_BOUNDED_OUT=probe; return "$AGY_RC"; }
_mc_probe_pi() { MC_BOUNDED_OUT=ok; return 0; }
_chain_head() { printf '%s' "${1%%,*}"; }
_chain_tail() { case "$1" in *,*) printf '%s' "${1#*,}" ;; esac; }
run_loop() {
  MISSION_DESIGNER_MODEL=claude:test; MISSION_PLANNER_MODEL=claude:test; MISSION_EVALUATOR_MODEL=claude:test
  MISSION_EXECUTOR_MODEL=agy:gemini-4-neon-high
  MISSION_EXECUTOR_CHAIN_REMAINING=pi:openrouter/deepseek/deepseek-v4.1-flash
  _pi_probed=:; _pi_failed=:; _pi_rcmap=''; _lane_degraded=''
  eval "$ROLE_LOOP"
}
AGY_RC=0; run_loop
[ "$AGY_PROBES" = 1 ] && [ "$MISSION_EXECUTOR_MODEL" = agy:gemini-4-neon-high ] || fail "healthy agy pin changed ($MISSION_EXECUTOR_MODEL, probes=$AGY_PROBES)"
AGY_PROBES=0; AGY_RC=2; run_loop
[ "$AGY_PROBES" = 1 ] && [ "$MISSION_EXECUTOR_MODEL" = pi:openrouter/deepseek/deepseek-v4.1-flash ] \
  || fail "dead agy pin did not advance along its chain ($MISSION_EXECUTOR_MODEL)"
printf '%s' "$_lane_degraded" | grep -q '\*\*agy\*\* lane `gemini-4-neon-high` unusable (probe rc=`2`)' \
  || fail "degradation ledger does not name the agy lane and rc: $_lane_degraded"
echo "PASS role loop: agy pin probed, kept when healthy, advanced along its chain when dead"

# 5. Controller selection never runs agy as the controller.
grep -q 'agy:\*)' "$DRIVER" && grep -q "is an agy role lane, not a controller" "$DRIVER" \
  || fail "controller selection has no explicit agy refusal"
echo "PASS controller selection refuses agy explicitly"
echo "ALL PASS test_agy_lane.sh"
