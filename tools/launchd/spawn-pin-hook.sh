#!/bin/bash
# PreToolUse hook: deterministic role-spawn path enforcement for mission-control.
#
# Contract: design_docs/m-spawn-pin-enforcement.md §3.2–§3.6. Reads the
# PreToolUse JSON from stdin, applies the §3.3 decision order, and emits ONE line
# of JSON on stdout. Exits 0 in EVERY branch — a non-zero exit or malformed JSON
# is an ALLOW in Claude Code semantics, i.e. a silently disabled safeguard.
# An abort-on-error flag is therefore FORBIDDEN here; only `set -u` is used.
set -u

PAYLOAD=$(cat)

LOG="${HOME}/.ailang/state/mission-${MISSION_NAME:-v1}-spawn-hook.log"
mkdir -p "$(dirname "$LOG")" 2>/dev/null || true

# Append one tab-separated decision line. A log failure never changes the verdict.
# $1=role $2=pin $3=model $4=subagent_type $5=verdict $6=reason-token
log_line() {
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$1" "$2" "$3" "$4" "$5" "$6" \
    >> "$LOG" 2>/dev/null || true
}

# Emit JSON via jq (safe against quotes in the reason). $1=verdict $2=reason $3=token
emit_jq() {
  log_line "${ROLE:--}" "${PIN:--}" "${MODEL:--}" "${SUB:--}" "$1" "$3"
  jq -nc --arg d "$1" --arg r "$2" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:$d,permissionDecisionReason:$r}}'
  exit 0
}

# Passthrough: emit NO decision at all (empty stdout, exit 0), so the platform's
# own permission flow decides exactly as it would without this hook. An explicit
# "allow" here would BYPASS the permission system for every Agent/Task call in
# every attended session that loads this repo's settings — the opposite of the
# design's "status quo untouched". Logged with verdict `passthrough`. $1=token
emit_pass() {
  log_line "${ROLE:--}" "${PIN:--}" "${MODEL:--}" "${SUB:--}" "passthrough" "$1"
  exit 0
}

# Emit JSON via printf (constant reason, no jq dependency). $1=verdict $2=reason $3=token
emit_printf() {
  log_line "${ROLE:--}" "${PIN:--}" "${MODEL:--}" "${SUB:--}" "$1" "$3"
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"%s","permissionDecisionReason":"%s"}}\n' "$1" "$2"
  exit 0
}

# --- 2. Marker gate -----------------------------------------------------------
# Attended sessions and other repos (marker absent) are untouched: allow.
if [ "${MISSION_CONTROL_ACTIVE:-}" != "1" ]; then
  emit_pass "passthrough:marker-absent"
fi

# --- 3. Parser gate -----------------------------------------------------------
# A missing parser is never an allow while a mission is running. This branch is
# checked BEFORE any jq call and emits valid deny JSON without jq.
if ! command -v jq >/dev/null 2>&1; then
  emit_printf "deny" "fail-closed:jq-missing — the spawn-pin hook cannot parse the PreToolUse payload without jq; refusing to allow an unchecked mission spawn" "fail-closed:jq-missing"
fi

# --- 3b. Payload gate ---------------------------------------------------------
# An empty or unparsable payload while a mission is running is a DENY, never an
# allow: every jq -r below would read "" from it and the hook would otherwise
# fall through to a passthrough (judge finding F2, iteration 324).
if ! printf '%s' "$PAYLOAD" | jq -e . >/dev/null 2>&1; then
  emit_jq "deny" "fail-closed:payload-unparsable — the PreToolUse payload is empty or not valid JSON; refusing to allow an unchecked mission spawn" "fail-closed:payload-unparsable"
fi

# --- 4. Parse the five fields we read; ignore everything else ----------------
TOOL_NAME=$(printf '%s' "$PAYLOAD" | jq -r '.tool_name // ""')
MODEL=$(printf '%s' "$PAYLOAD" | jq -r '.tool_input.model // ""')
SUB=$(printf '%s' "$PAYLOAD" | jq -r '.tool_input.subagent_type // ""')
PROMPT=$(printf '%s' "$PAYLOAD" | jq -r '.tool_input.prompt // ""')
DESC=$(printf '%s' "$PAYLOAD" | jq -r '.tool_input.description // ""')

if [ "$TOOL_NAME" != "Agent" ] && [ "$TOOL_NAME" != "Task" ]; then
  emit_pass "passthrough:not-a-spawn"
fi

# --- 5. Role token ------------------------------------------------------------
# Match the explicit MISSION-ROLE: token against the FIRST line of the prompt,
# else the WHOLE of the description. The hook reads nothing else — it never
# infers a role from prose (design M8 measured a false positive doing exactly that).
ROLE=""
FIRST_LINE=$(printf '%s\n' "$PROMPT" | head -1)
ROLE=$(printf '%s\n' "$FIRST_LINE" | sed -n 's/^MISSION-ROLE:[[:space:]]*\([A-Za-z-]*\)[[:space:]]*$/\1/p')
if [ -z "$ROLE" ]; then
  ROLE=$(printf '%s\n' "$DESC" | sed -n 's/^MISSION-ROLE:[[:space:]]*\([A-Za-z-]*\)[[:space:]]*$/\1/p')
fi
if [ -n "$ROLE" ]; then
  ROLE=$(printf '%s' "$ROLE" | tr 'A-Z' 'a-z')
fi

# --- 6. No token found --------------------------------------------------------
if [ -z "$ROLE" ]; then
  if [ "$SUB" = "Explore" ]; then
    emit_jq "allow" "explore-readonly — subagent_type Explore — read-only agent, no role token required" "explore-readonly"
  fi
  emit_jq "deny" "fail-closed:role-missing — MISSION_CONTROL_ACTIVE=1 and this Agent/Task call carries no role token. Add \"MISSION-ROLE: <designer|planner|executor|evaluator>\" as the FIRST line of the prompt, or use subagent_type: Explore for a read-only reality-check." "fail-closed:role-missing"
fi

# --- 7. Token not in the role set ---------------------------------------------
case "$ROLE" in
  designer|planner|executor|evaluator) ;;
  *) emit_jq "deny" "fail-closed:role-unknown — role token '$ROLE' is not one of designer, planner, executor, evaluator" "fail-closed:role-unknown" ;;
esac

# --- 8. Read the role's model pin (bash-3.2 indirect form) --------------------
ROLE_UC=$(printf '%s' "$ROLE" | tr 'a-z' 'A-Z')
_v="MISSION_${ROLE_UC}_MODEL"
PIN="${!_v:-}"
if [ -z "$PIN" ]; then
  emit_jq "deny" "fail-closed:${ROLE}-model-missing — MISSION_${ROLE_UC}_MODEL is unset; no default is applied" "fail-closed:${ROLE}-model-missing"
fi

# --- 9. Provider pin (contains ':') -------------------------------------------
# Does NOT look at tool_input.model at all, so retrying with a different alias
# is denied identically.
case "$PIN" in
  *:*)
    emit_jq "deny" "deny:provider-pin — $ROLE is pinned to $PIN; Agent-tool alias spawn refused — use the cross-provider recipe (resolve-role-spawn.sh $ROLE)" "deny:provider-pin"
    ;;
esac

# --- 10. Evaluator collision (generator != judge) ------------------------------
if [ "$ROLE" = "evaluator" ]; then
  EXEC_RESOLVED="${MISSION_EXECUTOR_RESOLVED:-${MISSION_EXECUTOR_MODEL:-}}"
  if [ "$PIN" = "$EXEC_RESOLVED" ]; then
    emit_jq "deny" "deny:generator-equals-judge — evaluator alias '$PIN' equals the executor's resolved model; generator!=judge — re-route the evaluator (resolve-role-spawn.sh evaluator)" "deny:generator-equals-judge"
  fi
fi

# --- 11. Otherwise allow ------------------------------------------------------
emit_jq "allow" "allow:alias-pin — $ROLE alias pin '$PIN' permits the Agent-tool spawn path" "allow:alias-pin"
