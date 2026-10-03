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
# select_model also calls these (demote list + ration gate). Without them every call was a
# "command not found" that happened to return nonzero — the suite went green on a half-loaded
# seam. The billing reader _mc_load_ration stays out of scope and is stubbed below.
awk '/^_mc_canon_id\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_help.sh"
awk '/^_mc_demote\(\) /' "$DRIVER" >> "$TMP/fn_help.sh"
awk '/^_mc_is_demoted\(\) \{/,/^\}$/' "$DRIVER" >> "$TMP/fn_help.sh"
awk '/^_mc_rung_bucket\(\) \{/,/^\}$/' "$HERE/lib/lane-probe.sh" >> "$TMP/fn_help.sh"
awk '/^_mc_is_over_ration\(\) \{/,/^\}$/' "$HERE/lib/lane-probe.sh" >> "$TMP/fn_help.sh"
# Guard the extraction itself: an empty extract would make every test vacuously fail
# in confusing ways — fail loudly at the seam instead.
[ -s "$TMP/fn_set.sh" ] && [ -s "$TMP/fn_sel.sh" ] \
  && [ "$(/usr/bin/grep -c '^_mc_[a-z_]*()' "$TMP/fn_help.sh")" -eq 5 ] \
  || { echo "FAIL extraction: function boundaries not found in $DRIVER"; exit 1; }

log(){ :; }
OVERRIDE_FILE="$TMP/nonexistent-override"
PROBE_TIMEOUT=5
MISSION_MODEL=""
PREFS="claude-opus-5,claude-fable-5-1"
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

PI_PROBES=""
_mc_probe_pi(){ PI_PROBES="$PI_PROBES $1"; _mc_bounded "$PROBE_TIMEOUT" pi --model "$1"; }
# Stub of the billing reader: only sets MC_OVER_RATION from a test variable.
MC_OVER_RATION=""; MC_DEMOTED=""   # the driver initialises MC_DEMOTED at :893
_mc_load_ration(){ MC_OVER_RATION="${RATION:-}"; }

. "$TMP/fn_help.sh"
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

# --- ration gate + demotion seams (previously unexercised: the helpers were undefined) ---
PROBE_CNT=0
_mc_probe_codex(){ PROBE_CNT=$((PROBE_CNT + 1)); [ "${CODEX_OK:-0}" = "1" ]; }
# Plain assignments, not `VAR=x check ...`: a prefix on a function call is TEMPORARY in bash, so
# PI_PROBES/PROBE_CNT would read back as their pre-call values and the zero-probe asserts
# would be vacuous.
RATION="codex ollama anthropic openrouter"; CODEX_OK=1; PI_OK="ollama/glm-5.3:cloud|openrouter/z-ai/glm-5.3"
PI_PROBES=""; PROBE_CNT=0
check "all-blocked-no-controller" 1 "none"
[ -z "$PI_PROBES" ] && [ "$PROBE_CNT" -eq 0 ] \
  && echo "PASS all-blocked-no-controller-zero-probes" \
  || { echo "FAIL all-blocked-no-controller-zero-probes (pi:'$PI_PROBES' codex:$PROBE_CNT)"; fail=1; }

CONTROLLER_FALLBACK="pi:openrouter/z-ai/glm-5.3,pi:ollama/glm-5.3:cloud"; RATION="openrouter"; CODEX_OK=0
PI_PROBES=""
check "openrouter-blocked-reaches-next" 0 "pi:ollama/glm-5.3:cloud"
case "$PI_PROBES" in *openrouter*) echo "FAIL openrouter-blocked-reaches-next-no-probe (probed:$PI_PROBES)"; fail=1 ;;
  *ollama*) echo "PASS openrouter-blocked-reaches-next-no-probe (probed:$PI_PROBES)" ;;
  *) echo "FAIL openrouter-blocked-reaches-next-no-probe (ollama never probed)"; fail=1 ;; esac

RATION=""; MC_DEMOTED=" pi:openrouter/z-ai/glm-5.3"; PI_PROBES=""
check "demoted-rung-skipped" 0 "pi:ollama/glm-5.3:cloud"
case "$PI_PROBES" in *openrouter*) echo "FAIL demoted-rung-skipped-no-probe (probed:$PI_PROBES)"; fail=1 ;;
  *) echo "PASS demoted-rung-skipped-no-probe" ;; esac
MC_DEMOTED=""

# Anti-vacuity: re-run this suite and count undefined-helper errors on stderr. At the base
# commit this was 63 (21 selections x 3 missing helpers), so the suite was green on nothing.
if [ -z "${CC_NESTED:-}" ]; then
  _e="$TMP/nested.err"
  CC_NESTED=1 /bin/bash "$0" >/dev/null 2>"$_e"
  _n=$(/usr/bin/grep -c 'command not found' "$_e")
  [ "$_n" -eq 0 ] && echo "PASS no-command-not-found-on-stderr" \
    || { echo "FAIL no-command-not-found-on-stderr ($_n)"; fail=1; }
fi

exit $fail
