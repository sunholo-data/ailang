#!/bin/bash
# Offline regression checks for the spawn-pin PreToolUse hook.
# Contract: design_docs/m-spawn-pin-enforcement.md §3.2–§3.6 and the M2 test plan.
# Each arm builds a payload with jq -nc, pipes it into the hook with the arm's
# env, and asserts the decision plus the reason token. Every arm uses a fresh
# $tmp as HOME so the decision log lands in a temp dir, never the real state dir.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HOOK="$ROOT/tools/launchd/spawn-pin-hook.sh"
PASS=0; FAIL=0
ok() { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL: $1 (got: $2)"; }
want() { [ "$2" = "$3" ] && ok "$1" || bad "$1" "$2"; }
contains() { case "$1" in *"$2"*) return 0;; *) return 1;; esac; }

# Run the hook with the given payload and env assignments; sets DEC, REASON, RC.
# env is used (not bare VAR=value words) so the assignments reach the child.
run_hook() {
  local payload="$1"; shift
  tmp=$(mktemp -d)
  out=$(printf '%s' "$payload" | env HOME="$tmp" MISSION_NAME=test "$@" bash "$HOOK")
  RC=$?
  OUT="$out"
  LOGLINE=$(tail -1 "$tmp/.ailang/state/mission-test-spawn-hook.log" 2>/dev/null)
  DEC=$(printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecision')
  REASON=$(printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecisionReason')
  rm -rf "$tmp"
}

# --- Arm 1: provider pin (contains ':') -> deny, regardless of which alias -----
payload1=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"general-purpose",prompt:"MISSION-ROLE: executor\nInvoke the sprint-executor skill",description:"Execute sprint M1"}}')
run_hook "$payload1" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && contains "$REASON" "Agent-tool alias spawn refused" && contains "$REASON" "codex:gpt-5.6-luna"; then
  ok "arm1 deny provider-pin"
else
  bad "arm1 deny provider-pin" "dec=$DEC reason=$REASON"
fi
ARM1_REASON="$REASON"

# --- Arm 2: identical to arm 1 but model=opus -> byte-identical deny reason ----
payload2=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"opus",subagent_type:"general-purpose",prompt:"MISSION-ROLE: executor\nInvoke the sprint-executor skill",description:"Execute sprint M1"}}')
run_hook "$payload2" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && [ "$REASON" = "$ARM1_REASON" ]; then
  ok "arm2 deny reason byte-identical to arm1 (no silent retry through another alias)"
else
  bad "arm2 deny reason byte-identical to arm1 (no silent retry through another alias)" "dec=$DEC reason=$REASON"
fi

# --- Arm 3: pin unset -> deny fail-closed:executor-model-missing ---------------
run_hook "$payload1" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:executor-model-missing"; then
  ok "arm3 deny executor-model-missing"
else
  bad "arm3 deny executor-model-missing" "dec=$DEC reason=$REASON"
fi

# --- Arm 4: evaluator alias == executor resolved -> deny generator!=judge -------
payload_eval=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"general-purpose",prompt:"MISSION-ROLE: evaluator\nEvaluate the sprint",description:"Evaluate sprint"}}')
run_hook "$payload_eval" MISSION_CONTROL_ACTIVE=1 MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_RESOLVED=sonnet
if [ "$DEC" = "deny" ] && contains "$REASON" "generator!=judge"; then
  ok "arm4 deny generator-equals-judge"
else
  bad "arm4 deny generator-equals-judge" "dec=$DEC reason=$REASON"
fi

# --- Arm 5: evaluator alias != executor resolved -> allow alias-pin ------------
run_hook "$payload_eval" MISSION_CONTROL_ACTIVE=1 MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_RESOLVED=codex:gpt-5.6-sol
if [ "$DEC" = "allow" ] && contains "$REASON" "allow:alias-pin"; then
  ok "arm5 allow alias-pin"
else
  bad "arm5 allow alias-pin" "dec=$DEC reason=$REASON"
fi

# --- Arm 6: two role skills named in prose, no token -> deny role-missing ------
payload6=$(jq -nc --arg p "Evaluate whether the sprint-planner's plan was followed by the sprint-executor" '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"general-purpose",prompt:$p,description:"evaluate"}}')
run_hook "$payload6" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:role-missing"; then
  ok "arm6 deny role-missing (no prose-based role inference)"
else
  bad "arm6 deny role-missing (no prose-based role inference)" "dec=$DEC reason=$REASON"
fi

# --- Arm 7: no token, general-purpose, marker present -> deny role-missing -----
payload7=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"general-purpose",prompt:"do something",description:"x"}}')
run_hook "$payload7" MISSION_CONTROL_ACTIVE=1
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:role-missing"; then
  ok "arm7 deny role-missing"
else
  bad "arm7 deny role-missing" "dec=$DEC reason=$REASON"
fi

# --- Arm 7ctl: byte-identical payload to arm 7, marker absent -> NO DECISION ---
# Empty stdout + exit 0: the platform's own permission flow decides, exactly as
# without the hook. An explicit "allow" here would bypass permissions for every
# Agent/Task call in every attended session (judge finding F1, iteration 324).
run_hook "$payload7" MISSION_CONTROL_ACTIVE=
lv=$(printf '%s' "$LOGLINE" | awk -F'\t' '{print $6"/"$7}')
if [ -z "$OUT" ] && [ "$RC" -eq 0 ] && [ "$lv" = "passthrough/passthrough:marker-absent" ]; then
  ok "arm7ctl marker-absent emits NO decision (attended-session status quo), logged as passthrough"
else
  bad "arm7ctl marker-absent emits NO decision (attended-session status quo), logged as passthrough" "out=$OUT rc=$RC log=$lv"
fi

# --- Arm NS: marker present but tool_name is not Agent/Task -> NO DECISION -----
payloadNS=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Bash",tool_input:{command:"echo hi"}}')
run_hook "$payloadNS" MISSION_CONTROL_ACTIVE=1
lv=$(printf '%s' "$LOGLINE" | awk -F'\t' '{print $6"/"$7}')
if [ -z "$OUT" ] && [ "$RC" -eq 0 ] && [ "$lv" = "passthrough/passthrough:not-a-spawn" ]; then
  ok "armNS not-a-spawn emits NO decision, logged as passthrough"
else
  bad "armNS not-a-spawn emits NO decision, logged as passthrough" "out=$OUT rc=$RC log=$lv"
fi

# --- Arm 8p / 8e: malformed or EMPTY payload while marker present -> deny -------
run_hook "not json" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:payload-unparsable" && [ "$RC" -eq 0 ]; then
  ok "arm8p deny payload-unparsable (malformed JSON), exit 0"
else
  bad "arm8p deny payload-unparsable (malformed JSON), exit 0" "dec=$DEC reason=$REASON rc=$RC"
fi
run_hook "" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:payload-unparsable" && [ "$RC" -eq 0 ]; then
  ok "arm8e deny payload-unparsable (empty stdin), exit 0"
else
  bad "arm8e deny payload-unparsable (empty stdin), exit 0" "dec=$DEC reason=$REASON rc=$RC"
fi
# Arm 8n: a bare JSON `null` literal is valid JSON but carries no fields; the
# payload gate must treat it as unparsable (jq -e exits 1 on null/false). The
# round-2 judge found this surface correct but untested (iteration 324).
run_hook "null" MISSION_CONTROL_ACTIVE=1 MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:payload-unparsable" && [ "$RC" -eq 0 ]; then
  ok "arm8n deny payload-unparsable (bare null literal), exit 0"
else
  bad "arm8n deny payload-unparsable (bare null literal), exit 0" "dec=$DEC reason=$REASON rc=$RC"
fi

# --- Arm 7a: no token, subagent_type=Explore, marker present -> allow ----------
payload7a=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"Explore",prompt:"check something",description:"x"}}')
run_hook "$payload7a" MISSION_CONTROL_ACTIVE=1
if [ "$DEC" = "allow" ] && contains "$REASON" "explore-readonly"; then
  ok "arm7a allow explore-readonly"
else
  bad "arm7a allow explore-readonly" "dec=$DEC reason=$REASON"
fi

# --- Arm 7b: token value not in the role set -> deny role-unknown --------------
payload7b=$(jq -nc '{hook_event_name:"PreToolUse",tool_name:"Agent",tool_input:{model:"sonnet",subagent_type:"general-purpose",prompt:"MISSION-ROLE: judge\njudge the work",description:"x"}}')
run_hook "$payload7b" MISSION_CONTROL_ACTIVE=1
if [ "$DEC" = "deny" ] && contains "$REASON" "fail-closed:role-unknown"; then
  ok "arm7b deny role-unknown"
else
  bad "arm7b deny role-unknown" "dec=$DEC reason=$REASON"
fi

# --- Arm 8j: jq missing -> deny fail-closed:jq-missing, exit 0 -----------------
tmp=$(mktemp -d)
mkdir -p "$tmp/nobin"
out=$(printf '%s' "$payload1" | env HOME="$tmp" MISSION_NAME=test MISSION_CONTROL_ACTIVE=1 \
  MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna PATH="$tmp/nobin" /bin/bash "$HOOK")
rc=$?
dec=$(printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecision')
reason=$(printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecisionReason')
if [ "$dec" = "deny" ] && contains "$reason" "fail-closed:jq-missing" && [ "$rc" -eq 0 ]; then
  ok "arm8j deny jq-missing, exit 0"
else
  bad "arm8j deny jq-missing, exit 0" "dec=$dec reason=$reason rc=$rc"
fi
rm -rf "$tmp"

# --- Arm L: decision log format (2 lines, 7 tab-separated fields) -------------
tmp=$(mktemp -d)
printf '%s' "$payload1" | env HOME="$tmp" MISSION_NAME=test MISSION_CONTROL_ACTIVE=1 \
  MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna bash "$HOOK" >/dev/null
printf '%s' "$payload7a" | env HOME="$tmp" MISSION_NAME=test MISSION_CONTROL_ACTIVE=1 bash "$HOOK" >/dev/null
LOG="$tmp/.ailang/state/mission-test-spawn-hook.log"
# All SEVEN fields of both lines are asserted (judge finding, iteration 324: checking
# only the last two let a silent role/pin field swap in log_line() go unnoticed).
if [ -f "$LOG" ]; then
  lines=$(wc -l < "$LOG" | tr -d ' ')
  nf1=$(awk -F'\t' 'NR==1{print NF}' "$LOG"); nf2=$(awk -F'\t' 'NR==2{print NF}' "$LOG")
  ts_ok=$(awk -F'\t' '$1 ~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]Z$/ {c++} END{print c+0}' "$LOG")
  rest1=$(awk -F'\t' 'NR==1{print $2"|"$3"|"$4"|"$5"|"$6"|"$7}' "$LOG")
  rest2=$(awk -F'\t' 'NR==2{print $2"|"$3"|"$4"|"$5"|"$6"|"$7}' "$LOG")
  if [ "$lines" = "2" ] && [ "$nf1" = "7" ] && [ "$nf2" = "7" ] && [ "$ts_ok" = "2" ] \
    && [ "$rest1" = "executor|codex:gpt-5.6-luna|sonnet|general-purpose|deny|deny:provider-pin" ] \
    && [ "$rest2" = "-|-|sonnet|Explore|allow|explore-readonly" ]; then
    ok "armL decision log: 2 lines x 7 fields, every field asserted"
  else
    bad "armL decision log: 2 lines x 7 fields, every field asserted" "lines=$lines nf=$nf1/$nf2 ts_ok=$ts_ok l1=$rest1 l2=$rest2"
  fi
else
  bad "armL decision log: 2 lines x 7 fields, every field asserted" "log missing"
fi
rm -rf "$tmp"

# --- Arm 7c: settings.json wiring (two overlapping PreToolUse hooks) -----------
SETTINGS="$ROOT/.claude/settings.json"
n=$(jq '.hooks.PreToolUse | length' "$SETTINGS")
c0=$(jq -r '.hooks.PreToolUse[0].hooks[0].command' "$SETTINGS")
m1=$(jq -r '.hooks.PreToolUse[1].matcher' "$SETTINGS")
c1=$(jq -r '.hooks.PreToolUse[1].hooks[0].command' "$SETTINGS")
if [ "$n" -ge 2 ] && contains "$c0" "coordinator_hook.sh" \
  && [ "$m1" = "Agent|Task" ] && contains "$c1" "tools/launchd/spawn-pin-hook.sh"; then
  ok "arm7c settings.json wires coordinator first + spawn-pin hook beside it"
else
  bad "arm7c settings.json wires coordinator first + spawn-pin hook beside it" "n=$n c0=$c0 m1=$m1 c1=$c1"
fi

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
