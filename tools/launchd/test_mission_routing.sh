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
grep -q 'MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol' "$driver" \
  && ok "executor remains Codex Sol primary" || bad "executor remains Codex Sol primary" "missing default"
grep -q 'MISSION_EXECUTOR_FALLBACK:-pi:openrouter/deepseek/deepseek-v4-flash-0731:floor' "$driver" \
  && ok "executor remains DeepSeek v4 Flash fallback" || bad "executor remains DeepSeek v4 Flash fallback" "missing fallback"
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
rm -rf "$lab"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
