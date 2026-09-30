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

# Ration gate (2026-09-29). The driver ration-gates the lanes it RESOLVES, but a role
# spawned inside the iteration went wherever this script pointed — and the planner arm
# below emits derive-planner-lane.sh's codex "anthropic-fallback" whatever the ration
# says. World iter-208 ran its planner AND executor on codex after the driver had
# refused codex as over ration for that same fire. The driver exports the over buckets
# as MISSION_OVER_RATION; a recipe on one of them is rerouted to the driver's own
# resolved lane for the role when that lane is in budget, and refused otherwise.
#
# _rs_bucket MUST agree with the driver's _mc_rung_bucket — test_mission_routing.sh
# compares the two on every rung shape.
_rs_bucket() {
  case "$1" in
    codex:*) printf 'codex' ;;
    pi:openrouter/*) printf 'openrouter' ;;
    pi:ollama/*:cloud|pi:ollama/*-cloud) printf 'ollama' ;;
    pi:ollama/*) printf '' ;;
    claude:*) printf 'anthropic' ;;
    pi:*) printf '' ;;
    *) printf 'anthropic' ;;
  esac
}
_rs_over() {
  local b; b=$(_rs_bucket "$1")
  [ -n "$b" ] || return 1
  case " ${MISSION_OVER_RATION:-} " in *" $b "*) return 0 ;; esac
  return 1
}
# emit_recipe <provider:model> <reason-token> — uses $ROLE_UC for the reroute target.
emit_recipe() {
  local pm="$1" reason="$2" rv alt
  if _rs_over "$pm"; then
    rv="MISSION_${ROLE_UC}_RESOLVED"; alt="${!rv:-}"
    case "$alt" in
      *:*) if [ "$alt" != "$pm" ] && ! _rs_over "$alt"; then
             emit "recipe $alt over-ration-reroute:$(_rs_bucket "$pm")"
           fi ;;
    esac
    emit "refuse over-ration:$(_rs_bucket "$pm")"
  fi
  emit "recipe $pm $reason"
}

# Anthropic model FAMILY of a role value, so the generator != judge check compares
# models rather than spellings: the Agent-tool alias `sonnet` and the CLI pin
# `claude:claude-sonnet-5-5` are the same model. Without this, a codex-dry fire that
# hands the executor to claude:claude-sonnet-5-5 (rung 2, 2026-09-29) would be judged
# by the `sonnet` evaluator unnoticed.
family() {
  v=${1#claude:}
  case "$v" in
    sonnet|claude-sonnet-*) printf 'sonnet' ;;
    opus|claude-opus-*) printf 'opus' ;;
    fable|claude-fable-*) printf 'fable' ;;
    haiku|claude-haiku-*) printf 'haiku' ;;
    *) printf '%s' "$v" ;;
  esac
}

ROLE=${1:-}
if [ -z "$ROLE" ]; then
  emit "refuse fail-closed:role-missing"
fi

case "$ROLE" in
  designer|planner|executor|evaluator) ;;
  *) emit "refuse fail-closed:role-unknown" ;;
esac
# The env vars are UPPERCASE (MISSION_EXECUTOR_MODEL); bash-3.2 has no ${v^^}.
ROLE_UC=$(printf '%s' "$ROLE" | tr 'a-z' 'A-Z')

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
      emit_recipe "$provider_model" "$reason"
      ;;
    *)
      # Defensive: derive always emits a well-formed line; if it somehow does
      # not, fail closed rather than emit garbage.
      emit "refuse fail-closed:derive-unparsable"
      ;;
  esac
fi

# Non-planner roles: read the role's model pin (bash-3.2 indirect form).
_v="MISSION_${ROLE_UC}_MODEL"
PIN="${!_v:-}"
if [ -z "$PIN" ]; then
  emit "refuse fail-closed:${ROLE}-model-missing"
fi

case "$PIN" in
  *:*)
    emit_recipe "$PIN" "declared:provider-pin"
    ;;
  *)
    # Bare alias. Evaluator collision check: generator != judge.
    if [ "$ROLE" = "evaluator" ]; then
      EXEC_RESOLVED="${MISSION_EXECUTOR_RESOLVED:-${MISSION_EXECUTOR_MODEL:-}}"
      if [ "$(family "$PIN")" = "$(family "$EXEC_RESOLVED")" ]; then
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
