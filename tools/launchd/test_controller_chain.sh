#!/bin/bash
# test_controller_chain.sh — the CONTROLLER_FALLBACK chain walk in select_model.
#
# WHY THIS EXISTS. Mark's 2026-08-31 directive extended the controller fallback from a
# single `codex:<model>` slot to an ordered chain ending in two pi/GLM rungs
# (flat-rate Ollama Cloud, then the same-weights OpenRouter metered twin), after the
# 08-29..31 weekend showed a joint Anthropic+codex dry-out refuses every fire of a
# 13h-cadence mission for half a day at a time. The chain walk is pure bash-3.2 logic
# with probe seams, so it is testable without spending a probe or an iteration: the
# functions are extracted from the driver and run against stubbed probes.
#
# Extraction, not duplication: the functions under test are awk'd out of
# mission-control.sh itself, so this suite cannot drift green against an edited driver.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"

TMP="${TMPDIR:-/tmp}/ctl-chain-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

awk '/^_mc_set_controller\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_set.sh"
awk '/^select_model\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_sel.sh"
# Guard the extraction itself: an empty extract would make every test vacuously fail
# in confusing ways — fail loudly at the seam instead.
[ -s "$TMP/fn_set.sh" ] && [ -s "$TMP/fn_sel.sh" ] \
  || { echo "FAIL extraction: function boundaries not found in $DRIVER"; exit 1; }

log(){ :; }
OVERRIDE_FILE="$TMP/nonexistent-override"
PROBE_TIMEOUT=5
MISSION_MODEL=""
PREFS="claude-opus-5,claude-fable-5"
CONTROLLER_FALLBACK="codex:gpt-5.6-sol,pi:ollama/glm-5.3:cloud,pi:openrouter/z-ai/glm-5.3"

_mc_probe(){ return 1; }                        # every Anthropic rung quota-limited
_mc_probe_codex(){ [ "${CODEX_OK:-0}" = "1" ]; }
_mc_bounded(){                                  # succeed iff --model's value is in PI_OK (| delim)
  local a prev=""
  for a in "$@"; do
    if [ "$prev" = "--model" ]; then
      case "|${PI_OK:-}|" in *"|$a|"*) return 0 ;; *) return 7 ;; esac
    fi
    prev="$a"
  done
  return 7
}

. "$TMP/fn_set.sh"
. "$TMP/fn_sel.sh"

fail=0
check(){ # name expected_rc expected_id
  local name="$1" want_rc="$2" want_id="$3" rc got
  CONTROLLER_ID=""; CONTROLLER_PROVIDER=""
  select_model; rc=$?
  got="${CONTROLLER_ID:-none}"
  if [ "$rc" = "$want_rc" ] && [ "$got" = "$want_id" ]; then
    echo "PASS $name -> rc=$rc id=$got"
  else
    echo "FAIL $name -> rc=$rc id=$got (wanted rc=$want_rc id=$want_id)"; fail=1
  fi
}

CODEX_OK=1 PI_OK="" check "codex-first-when-usable"        0 "codex:gpt-5.6-sol"
CODEX_OK=0 PI_OK="ollama/glm-5.3:cloud" \
                  check "flat-rate-rung-when-codex-dry"    0 "pi:ollama/glm-5.3:cloud"
CODEX_OK=0 PI_OK="openrouter/z-ai/glm-5.3" \
                  check "openrouter-final-rung"            0 "pi:openrouter/z-ai/glm-5.3"
CODEX_OK=0 PI_OK="" check "all-rungs-dry-refuses"          1 "none"
CONTROLLER_FALLBACK="foo:bar,pi:ollama/glm-5.3:cloud" CODEX_OK=0 PI_OK="ollama/glm-5.3:cloud" \
                  check "unsupported-entry-skipped"        0 "pi:ollama/glm-5.3:cloud"
# The provider tag must reach _mc_run_once's branch: a pi rung selects provider=pi.
CODEX_OK=0 PI_OK="ollama/glm-5.3:cloud" CONTROLLER_FALLBACK="pi:ollama/glm-5.3:cloud" \
  select_model >/dev/null 2>&1
[ "${CONTROLLER_PROVIDER:-}" = "pi" ] \
  && echo "PASS pi-rung-sets-provider-pi" \
  || { echo "FAIL pi-rung-sets-provider-pi (got '${CONTROLLER_PROVIDER:-}')"; fail=1; }

exit $fail
