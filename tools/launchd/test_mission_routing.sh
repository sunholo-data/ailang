#!/bin/bash
# Offline regression checks for Anthropic→Codex mission routing and decision state.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DERIVE="$ROOT/tools/launchd/derive-planner-lane.sh"
FIXTURE="$ROOT/tools/launchd/testdata/planner-lane/n-no-backtick-bullet.md"
PASS=0; FAIL=0
ok() { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL: $1 (got: $2)"; }
want() { [ "$2" = "$3" ] && ok "$1" || bad "$1" "$2"; }

out=$(MISSION_PLANNER_MODEL=codex:gpt-5.6-sol MISSION_ANTHROPIC_AVAILABLE=1 "$DERIVE" "$FIXTURE" 2>/dev/null)
want "planner remains fail-closed to Opus while Anthropic is available" "$out" "opus fail-closed:unparsable-path-entry"

out=$(MISSION_PLANNER_MODEL=codex:gpt-5.6-sol MISSION_ANTHROPIC_AVAILABLE=0 "$DERIVE" "$FIXTURE" 2>/dev/null)
want "planner falls back to Codex Sol when Anthropic is unavailable" "$out" "codex:gpt-5.6-sol anthropic-fallback:fail-closed:unparsable-path-entry"

out=$(MISSION_PLANNER_MODEL=codex:gpt-5.6-sol MISSION_ANTHROPIC_AVAILABLE=0 \
  MISSION_PLANNER_ANTHROPIC_FALLBACK=codex:test-sol "$DERIVE" "$FIXTURE" 2>/dev/null)
want "planner Anthropic fallback is configurable" "$out" "codex:test-sol anthropic-fallback:fail-closed:unparsable-path-entry"

driver="$ROOT/tools/launchd/mission-control.sh"
# FLEET CHANGE (Mark 2026-08-26, attended): executor primary moved codex -> the
# flat-rate ollama route for deepseek-v4-flash. Same weights the fallback below
# already reaches via OpenRouter, so this is a route change, not a capability
# change. The old assertion encoded the superseded policy and is replaced, not
# kept alongside (no backward compatibility for stale tests).
grep -q 'MISSION_EXECUTOR_MODEL:-pi:ollama/deepseek-v4-flash:0731-cloud}' "$driver" \
  && ok "executor primary is the flat-rate ollama deepseek lane" || bad "executor primary is the flat-rate ollama deepseek lane" "missing default"
# The planner carries the strongest open-weight model, per "better models do
# planning only" — one run per iteration, so an 18x quota draw is affordable.
grep -q 'MISSION_PLANNER_MODEL:-pi:ollama/kimi-k3:cloud}' "$driver" \
  && ok "planner primary is the ollama kimi-k3 lane" || bad "planner primary is the ollama kimi-k3 lane" "missing default"
# A pi: planner pin must actually REACH the lane. Before derive-planner-lane.sh
# Step 0 accepted pi:, every non-codex value emitted "opus fail-closed:env-pin",
# so the pin read as applied in the driver log while opus actually ran.
out=$(MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "pi planner pin reaches its lane, not a silent opus" "$out" "pi:ollama/kimi-k3:cloud declared:codex-ok"
# ...but the allowlist must still bind it exactly as it binds codex.
out=$(MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/a-unlisted-language-path.md")
want "pi planner still fails closed outside the allowlist" "$out" "opus fail-closed:path-not-in-codex-allowlist"
# Anchored on the closing brace DELIBERATELY. The previous form pinned the `:floor`
# suffix, but a suffix-free grep would prefix-match `...-0731:floor` too and pass on
# both values — so the brace is what keeps this assertion discriminating and makes a
# silent return of the price-pin RED. `:floor` was dropped 2026-08-18 (routed the
# executor to the least-healthy endpoint; see the driver's own note).
grep -q 'MISSION_EXECUTOR_FALLBACK:-pi:openrouter/deepseek/deepseek-v4-flash-0731}' "$driver" \
  && ok "executor DeepSeek fallback is provider-unpinned (no :floor)" || bad "executor DeepSeek fallback is provider-unpinned (no :floor)" "missing or price-pinned fallback"
grep -q 'MISSION_CONTROLLER_FALLBACK:-codex:gpt-5.6-sol' "$driver" \
  && ok "controller has Codex Sol fallback" || bad "controller has Codex Sol fallback" "missing fallback"

"$ROOT/scripts/mission_decisions.sh" --check --file "$ROOT/design_docs/v1-mission.md" >/dev/null \
  && ok "decision ledger validates" || bad "decision ledger validates" "invalid"
open=$("$ROOT/scripts/mission_decisions.sh" --open --file "$ROOT/design_docs/v1-mission.md")
case "$open" in *$'D-ROUTE-1\t'*) bad "resolved routing decision is not re-asked" "D-ROUTE-1 appeared OPEN" ;; *) ok "resolved routing decision is not re-asked" ;; esac

# Exercise the real controller selector with probes stubbed: all Anthropic
# candidates fail, then the configured Codex subscription fallback succeeds.
lab=$(mktemp -d "${TMPDIR:-/tmp}/mission-routing.XXXXXX") || exit 1
awk '/^_mc_probe_codex\(\) \{/,/^\}/' "$driver" > "$lab/select.sh"
awk '/^_mc_set_controller\(\) \{/,/^\}/' "$driver" >> "$lab/select.sh"
awk '/^select_model\(\) \{/,/^\}/' "$driver" >> "$lab/select.sh"
out=$(/bin/bash -c '
  set -uo pipefail
  . "$1"
  _mc_probe() { return 1; }
  _mc_probe_codex() { [ "$1" = gpt-5.6-sol ]; }
  log() { :; }
  PREFS="claude-opus-5,claude-fable-5"
  CONTROLLER_FALLBACK="codex:gpt-5.6-sol"
  OVERRIDE_FILE="$2/no-override"
  select_model || exit 1
  printf "%s|%s|%s" "$CONTROLLER_ID" "$MISSION_ANTHROPIC_AVAILABLE" "$MODEL_WHY"
' _ "$lab/select.sh" "$lab")
want "controller selector traverses Anthropic to Codex" "$out" "codex:gpt-5.6-sol|0|Anthropic unavailable; subscription fallback"

# #696 regression guard. `_mc_set_controller` must EXPORT the controller pin, not
# merely assign it: the skill's routing contract reads `$MODEL`/`$MODEL_WHY` from
# inside the controller session — a CHILD of this driver — and every role's
# end-of-chain fallback (#611) terminates at `$MODEL`. The assertion above reads
# those variables in the SAME shell, so it passes whether or not `export` is
# present; the observable has to cross a process boundary, and `/usr/bin/env` is
# that boundary. The pre-`unset` matters: this suite is run by a controller
# session that already has MODEL exported, so without it the child would inherit
# the ambient value and the arm would pass for the wrong reason.
# MC_EXPORT_CONTROL is the known-positive control, read from the SAME `env` call:
# if it is missing the instrument is broken, not the driver.
out=$(/bin/bash -c '
  set -uo pipefail
  unset MISSION_MODEL MODEL MODEL_WHY CONTROLLER_ID CONTROLLER_PROVIDER MISSION_ANTHROPIC_AVAILABLE
  export MC_EXPORT_CONTROL=sentinel
  . "$1"
  _mc_probe() { [ "$1" = claude-opus-5 ]; }
  _mc_probe_codex() { return 1; }
  log() { :; }
  PREFS="claude-opus-5,claude-fable-5"
  CONTROLLER_FALLBACK="codex:gpt-5.6-sol"
  OVERRIDE_FILE="$2/no-override"
  select_model || exit 1
  /usr/bin/env | grep -E "^(MC_EXPORT_CONTROL|MODEL|MODEL_WHY|CONTROLLER_ID)=" | LC_ALL=C sort | tr "\n" "|"
' _ "$lab/select.sh" "$lab")
want "controller pin is exported to child processes (#696)" "$out" \
  "CONTROLLER_ID=claude:claude-opus-5|MC_EXPORT_CONTROL=sentinel|MODEL=claude-opus-5|MODEL_WHY=probe ok|"
rm -rf "$lab"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
