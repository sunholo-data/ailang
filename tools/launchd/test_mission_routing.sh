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
# FLEET (Mark 2026-08-26, attended): codex KEEPS both primary roles; the Ollama
# Cloud lanes sit at the FIRST FALLBACK. Asserting the primaries explicitly so a
# future promotion has to be deliberate rather than accidental.
grep -q 'MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol' "$driver" \
  && ok "executor primary remains Codex Sol" || bad "executor primary remains Codex Sol" "missing default"
grep -q 'MISSION_PLANNER_MODEL:-codex:gpt-5.6-sol' "$driver" \
  && ok "planner primary remains Codex Sol" || bad "planner primary remains Codex Sol" "missing default"
# Executor fallback: SAME deepseek-v4-flash weights, flat-rate ollama route
# instead of metered OpenRouter. The ratified codex->deepseek->opus chain is
# preserved; only the route changed. Brace-anchored so a prefix cannot pass.
# Each ollama rung must be BACKED BY ITS OPENROUTER TWIN, so exhausting the
# Ollama Cloud quota (unpublished denominator, so unpredictable) degrades the
# ROUTE and not the model. Asserted as a full chain, brace-anchored.
grep -q 'MISSION_EXECUTOR_FALLBACK:-pi:ollama/deepseek-v4-flash:0731-cloud,pi:openrouter/deepseek/deepseek-v4-flash-0731}' "$driver" \
  && ok "executor chain is ollama -> openrouter twin" || bad "executor chain is ollama -> openrouter twin" "missing or unchained"
# Planner fallback: kimi-k3 sits BETWEEN codex and opus, so opus stays last resort.
grep -q 'MISSION_PLANNER_FALLBACK:-pi:ollama/kimi-k3:cloud,pi:openrouter/moonshotai/kimi-k3}' "$driver" \
  && ok "planner chain is ollama -> openrouter twin" || bad "planner chain is ollama -> openrouter twin" "missing or unchained"
# A pi: pin must actually REACH its lane. Before derive-planner-lane.sh Step 0
# accepted pi:, every non-codex value emitted "opus fail-closed:env-pin", so a pi
# lane would read as pinned in the driver log while opus actually ran. This stays
# load-bearing now that the planner FALLBACK is a pi: lane.
out=$(MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "pi planner pin reaches its lane, not a silent opus" "$out" "pi:ollama/kimi-k3:cloud declared:codex-ok"
# ...but the allowlist must still bind it exactly as it binds codex.
out=$(MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/a-unlisted-language-path.md")
want "pi planner still fails closed outside the allowlist" "$out" "opus fail-closed:path-not-in-codex-allowlist"

# THE EMITTED LANE MUST CARRY ITS MODEL (2026-08-28). Step 5 used to emit a bare
# `codex` for any codex:* pin, dropping the model. Invisible on V1 by coincidence —
# its pin is codex:gpt-5.6-sol and the consumer default is also sol, so the dropped
# value equalled the fallback. NOT invisible on a mission pinned to a cheaper tier:
# the docs mission pins codex:gpt-5.6-luna ($0.20/$1.20 per M) and would have
# planned on gpt-5.6-sol ($2/$10) every iteration. Asserted with TWO different
# codex models so a re-hardcoded literal cannot satisfy both.
out=$(MISSION_PLANNER_MODEL='codex:gpt-5.6-luna' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "codex planner lane keeps its model (luna)" "$out" "codex:gpt-5.6-luna declared:codex-ok"
out=$(MISSION_PLANNER_MODEL='codex:gpt-5.6-sol' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "codex planner lane keeps its model (sol)" "$out" "codex:gpt-5.6-sol declared:codex-ok"

# --- PER-MISSION ALLOWLIST (M-DOCS-MISSION, 2026-08-28) --------------------------
# The allowlist is infra-only by default, which SILENTLY defeats any mission whose
# subject matter is docs/: the cheap planner pin reads as configured in the driver
# log while opus actually runs, every iteration. Four assertions, because the risk
# is symmetric — widening must work, and it must not become a hole.
DOCS_AL='tools/launchd/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*|docs/*|examples/*|README.md|CHANGELOG.md'

# 1. DEFAULT allowlist still denies docs paths => v1/world/motoko are unaffected.
out=$(MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/o-docs-mission-paths.md")
want "docs paths fail closed under the DEFAULT allowlist" "$out" "opus fail-closed:path-not-in-codex-allowlist"

# 2. WIDENED allowlist reaches the pinned cheap lane (the whole point).
out=$(MISSION_PLANNER_ALLOWLIST="$DOCS_AL" MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/o-docs-mission-paths.md")
want "docs paths reach the cheap lane under a widened allowlist" "$out" "pi:ollama/kimi-k3:cloud declared:codex-ok"

# 3. Widening for docs must NOT admit compiler paths — still an allowlist, not an off switch.
out=$(MISSION_PLANNER_ALLOWLIST="$DOCS_AL" MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/a-unlisted-language-path.md")
want "widened allowlist still denies compiler paths" "$out" "opus fail-closed:path-not-in-codex-allowlist"

# 4. `docs/../internal/...` must not ride a `docs/*` prefix out of the sandbox.
out=$(MISSION_PLANNER_ALLOWLIST="$DOCS_AL" MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/p-docs-traversal-escape.md")
want "traversal cannot escape an allowlisted prefix" "$out" "opus fail-closed:path-not-in-codex-allowlist"

# 5. `set -f` in the matcher is load-bearing: this script runs with cwd = the repo,
# so an unquoted `tools/launchd/*` in the matcher loop would PATHNAME-EXPAND into the
# real file list and then match none of the DECLARED paths. Asserted behaviourally,
# from the repo root, using the default allowlist's own literal prefix.
out=$(cd "$ROOT" && MISSION_PLANNER_MODEL='pi:ollama/kimi-k3:cloud' "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "allowlist globs are patterns, not repo-cwd pathname expansions" "$out" "pi:ollama/kimi-k3:cloud declared:codex-ok"

# --- DOCS MISSION LADDER (2026-08-28) -------------------------------------------
# The docs mission runs a subscription-first cost ladder: subscription -> flat-rate
# -> metered. Asserted on the VERSIONED env copy, because the live file in
# ~/.config/ailang is not reviewable and drifts silently.
docsenv="$ROOT/tools/launchd/mission-env/mission-docs.env"
[ -r "$docsenv" ] && ok "docs mission env profile exists" || bad "docs mission env profile exists" "missing"

# THE TRAP THIS GUARDS: derive-planner-lane.sh Step 0 accepts only codex:* or pi:*.
# A bare Anthropic alias as the PLANNER pin emits "opus fail-closed:env-pin" and
# silently runs OPUS — the most expensive model in the fleet, on the mission built to
# avoid it. Negative assertion, so a well-meaning "put sonnet first everywhere" edit
# is RED rather than silently expensive.
grep -qE '^MISSION_PLANNER_MODEL="\$\{MISSION_PLANNER_MODEL:-(codex|pi):' "$docsenv" \
  && ok "docs planner pin is a vetted non-opus lane (not a bare alias)" \
  || bad "docs planner pin is a vetted non-opus lane (not a bare alias)" "bare alias fails closed to opus"

# The allowlist widening is what makes that pin reach its lane at all.
grep -q 'MISSION_PLANNER_ALLOWLIST' "$docsenv" \
  && ok "docs mission widens the planner allowlist" \
  || bad "docs mission widens the planner allowlist" "missing — planner would fail closed to opus every fire"

# Rung 2 -> rung 3 must be the SAME WEIGHTS on two routes, so exhausting the
# flat-rate quota costs money and never capability.
grep -q 'MISSION_EXECUTOR_FALLBACK:-pi:ollama/glm-5.3-flash:cloud,pi:openrouter/z-ai/glm-5.3-flash}' "$docsenv" \
  && ok "docs executor chain is flat-rate -> metered twin (same weights)" \
  || bad "docs executor chain is flat-rate -> metered twin (same weights)" "missing or unchained"
grep -q 'MISSION_PLANNER_FALLBACK:-pi:ollama/glm-5.3-flash:cloud,pi:openrouter/z-ai/glm-5.3-flash}' "$docsenv" \
  && ok "docs planner chain is flat-rate -> metered twin (same weights)" \
  || bad "docs planner chain is flat-rate -> metered twin (same weights)" "missing or unchained"

# GENERATOR != JUDGE, asserted as VENDOR DISJOINTNESS ACROSS THE WHOLE CHAIN, not just
# the primary. The executor walks OpenAI -> Z-AI -> Z-AI; an evaluator on the same
# ladder would share a vendor at every rung, which is exactly the shared blind spot the
# rule exists to prevent.
grep -qE '^MISSION_EVALUATOR_FALLBACK=.*minimax' "$docsenv" \
  && ok "docs evaluator chain is vendor-disjoint from the executor" \
  || bad "docs evaluator chain is vendor-disjoint from the executor" "evaluator shares the executor vendor"
grep -qE '^MISSION_EVALUATOR_FALLBACK=.*(glm|deepseek|gpt-)' "$docsenv" \
  && bad "docs evaluator never shares an executor-chain vendor" "evaluator chain contains an executor vendor" \
  || ok "docs evaluator never shares an executor-chain vendor"
# The `:floor` guard, generalised. It previously pinned the OpenRouter route, which
# broke when the fallback moved to the flat-rate ollama route (same deepseek-v4-flash
# weights). The route is now asserted above; what must survive ANY route change is
# the rule itself: never price-pin the executor fallback. `:floor` is OpenRouter
# provider.sort=price, so it routes to the CHEAPEST endpoint — and the two cheapest
# for this model carry NEGATIVE health (StreamLake -2, Decart -5 of 28, measured
# 2026-08-18). World iteration 91 then returned rc=0 with ZERO BYTES CHANGED twice at
# stopReason=stop, so the guard's own stopReason assertion passed on a total failure.
# Written as a NEGATIVE assertion so it keeps biting if the fallback ever returns to
# OpenRouter, instead of silently passing because the string no longer matches.
grep -q 'MISSION_EXECUTOR_FALLBACK:-[^}]*:floor' "$driver" \
  && bad "executor fallback is never price-pinned (:floor)" "fallback is :floor-pinned" \
  || ok "executor fallback is never price-pinned (:floor)"
# opus-4.8 REMOVED from the controller ladder (Mark 2026-08-26). Negative
# assertion so a silent reintroduction is RED, not merely unasserted.
grep -q 'MISSION_MODEL_PREFS:-claude-opus-5,claude-fable-5}' "$driver" \
  && ok "controller ladder is opus-5 -> fable-5 (no opus-4.8)" || bad "controller ladder is opus-5 -> fable-5 (no opus-4.8)" "wrong ladder"
# ACTIVE lines only: a comment recording the removal must not trip the guard,
# but a real reintroduction must. (This distinction is why the first version of
# this assertion went red on its own changelog note.)
grep -v '^[[:space:]]*#' "$driver" | grep -q 'claude-opus-4-8' \
  && bad "opus-4.8 stays removed" "opus-4.8 reappeared in active driver code" \
  || ok "opus-4.8 stays removed"
# NO-SINGLE-PROVIDER-ROLE: the evaluator was the only role with no fallback at
# all. Every role must now name at least two providers across its chain.
grep -q 'MISSION_EVALUATOR_FALLBACK:-pi:ollama/minimax-m3:cloud,pi:openrouter/minimax/minimax-m3}' "$driver" \
  && ok "evaluator chain is ollama -> openrouter twin" || bad "evaluator chain is ollama -> openrouter twin" "missing evaluator chain"
# The chain walker must exist, or a comma value would be passed to pi as ONE
# model name and every fallback would 404.
grep -q '_chain_head()' "$driver" && grep -q 'CHAIN_REMAINING' "$driver" \
  && ok "driver has the chain walker" || bad "driver has the chain walker" "helpers or remaining-var missing"
# The evaluator fallback must not collide with planner or executor lanes, or the
# judge would share a model with a generator.
# VENDOR-level, not model-level. deepseek-v4-pro vs deepseek-v4-flash are
# different models but the same vendor and generation, and a shared systematic
# blind spot is exactly what generator!=judge is for. This guard caught that.
grep -q 'MISSION_EVALUATOR_FALLBACK:-pi:ollama/\(kimi\|deepseek\)' "$driver" \
  && bad "evaluator vendor is distinct from planner/executor" "evaluator shares a VENDOR with a generator" \
  || ok "evaluator vendor is distinct from planner/executor"
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
