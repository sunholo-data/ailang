#!/bin/bash
# Resolve a mission role's spawn pin to a concrete recipe or agent-tool alias.
#
# Contract: design_docs/m-spawn-pin-enforcement.md §3.1. Always exits 0 and emits
# exactly ONE line on stdout in the derive-planner-lane.sh convention
# `<value...> <reason-token>`. The planner role is special: it CONSUMES
# derive-planner-lane.sh verbatim and maps that script's single output line,
# copying the reason token through unchanged (never re-derives).
set -u

emit() {
  printf '%s\n' "$1"
  exit 0
}

ROLE=${1:-}
if [ -z "$ROLE" ]; then
  emit "refuse fail-closed:role-missing"
fi

case "$ROLE" in
  designer|planner|executor|evaluator) ;;
  *) emit "refuse fail-closed:role-unknown" ;;
esac

# Planner role is special and consumes derive-planner-lane.sh verbatim.
if [ "$ROLE" = "planner" ]; then
  DERIVE="$(cd "$(dirname "$0")" && pwd)/derive-planner-lane.sh"
  if [ ! -x "$DERIVE" ]; then
    emit "refuse fail-closed:derive-script-missing"
  fi
  doc=${2:-}
  lane=$("$DERIVE" "$doc" 2>/dev/null)
  case "$lane" in
    opus\ *)
      emit "agent-tool opus ${lane#opus }"
      ;;
    *:*\ *)
      provider_model=${lane%% *}
      reason=${lane#* }
      emit "recipe $provider_model $reason"
      ;;
    *)
      # Defensive: derive always emits a well-formed line; if it somehow does
      # not, fail closed rather than emit garbage.
      emit "refuse fail-closed:derive-unparsable"
      ;;
  esac
fi

# Non-planner roles: read the role's model pin (bash-3.2 indirect form). The
# env var is UPPERCASE (MISSION_EXECUTOR_MODEL), so uppercase the role first.
ROLE_UC=$(printf '%s' "$ROLE" | tr 'a-z' 'A-Z')
_v="MISSION_${ROLE_UC}_MODEL"
PIN="${!_v:-}"
if [ -z "$PIN" ]; then
  emit "refuse fail-closed:${ROLE}-model-missing"
fi

case "$PIN" in
  *:*)
    emit "recipe $PIN declared:provider-pin"
    ;;
  *)
    # Bare alias. Evaluator collision check: generator != judge.
    if [ "$ROLE" = "evaluator" ]; then
      EXEC_RESOLVED="${MISSION_EXECUTOR_RESOLVED:-${MISSION_EXECUTOR_MODEL:-}}"
      if [ "$PIN" = "$EXEC_RESOLVED" ]; then
        FALLBACK="${MISSION_EVALUATOR_FALLBACK:-}"
        if [ -z "$FALLBACK" ]; then
          emit "refuse fail-closed:evaluator-collision-no-fallback"
        fi
        head=${FALLBACK%%,*}
        emit "reroute $head generator-equals-judge"
      fi
    fi
    emit "agent-tool $PIN declared:alias-pin"
    ;;
esac
