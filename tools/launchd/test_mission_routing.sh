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
# RE-AFFIRMED 2026-09-05 (Mark, attended): astra goes IN THE CHAIN and does NOT
# take a primary. These two arms were briefly flipped to astra earlier the same
# day and are restored — sol keeps both primaries on months of in-role track
# record, against astra's single fizzbuzz round-trip and an rc=0 probe.
grep -q 'MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol' "$driver" \
  && ok "executor primary remains Codex Sol" || bad "executor primary remains Codex Sol" "missing default"
grep -q 'MISSION_PLANNER_MODEL:-codex:gpt-5.6-sol' "$driver" \
  && ok "planner primary remains Codex Sol" || bad "planner primary remains Codex Sol" "missing default"
# The fallback chains must stay `pi:*`-headed. A `codex:*` value here would run
# UNPROBED — the codex loop hands off to a value that only the *pi* loop probes —
# which is the "pin running unprobed on World" defect ailang#611 fixed. This is the
# arm that dies if someone "helpfully" inserts sol as the first fallback rung.
grep -q 'MISSION_EXECUTOR_FALLBACK:-pi:' "$driver" \
  && ok "executor fallback head is a probed pi lane" || bad "executor fallback head is a probed pi lane" "codex:* head runs unprobed (#611)"
grep -q 'MISSION_PLANNER_FALLBACK:-pi:' "$driver" \
  && ok "planner fallback head is a probed pi lane" || bad "planner fallback head is a probed pi lane" "codex:* head runs unprobed (#611)"
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

# --- DELIVERY MECHANISM, NOT JUST THE CONSUMER (docs iteration 1, 2026-08-28) ------
# THE MISS THIS GUARDS, stated plainly because it defeated 34 passing tests: every
# assertion above invokes the script as `MISSION_PLANNER_ALLOWLIST=... "$DERIVE" ...`,
# which exports the variable FOR THAT COMMAND. The production path does something
# different — the driver SOURCES a mission env file whose entries are bare
# assignments, and `derive-planner-lane.sh` runs as a CHILD PROCESS, which inherits
# only EXPORTED variables. So the allowlist was set and never delivered, every docs
# design doc failed closed to opus, and the whole widening was inert in production
# while its tests were green. Found by the docs mission itself, live
# (`env | grep MISSION_PLANNER_ALLOWLIST` empty in a real session), fixed by
# exporting it in the driver.
#
# Rule this encodes: a variable's TEST must exercise the same delivery path as its
# USE. Setting it on the command line tests the consumer and proves nothing about
# whether anything ever sets it.
grep -qE '^export MISSION_PLANNER_ALLOWLIST' "$driver" \
  && ok "driver EXPORTS the planner allowlist (bare assignment never reaches the child)" \
  || bad "driver EXPORTS the planner allowlist (bare assignment never reaches the child)" "not exported — allowlist is inert in production"

# Order is load-bearing: `export VAR="${VAR:-default}"` only picks up a mission's
# value if the env file was sourced FIRST. Asserted by line number so a reordering
# that silently reverts every mission to the default default is RED.
_src_line=$(grep -n 'config/ailang/mission-\${MISSION_NAME}.env' "$driver" | head -1 | cut -d: -f1)
_exp_line=$(grep -n '^export MISSION_PLANNER_ALLOWLIST' "$driver" | head -1 | cut -d: -f1)
if [ -n "$_src_line" ] && [ -n "$_exp_line" ] && [ "$_exp_line" -gt "$_src_line" ]; then
  ok "allowlist export comes AFTER the mission env file is sourced (line $_exp_line > $_src_line)"
else
  bad "allowlist export comes AFTER the mission env file is sourced" "src=$_src_line exp=$_exp_line"
fi

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
# CONTROLLER LADDER. Amended 2026-09-05 (Mark, attended): astra is inserted
# BETWEEN opus and fable — "ahead of each fable instance, that falls back to fable".
# The single exact-string arm this replaces bundled three separate guarantees, so
# any one edit reddened it without saying which broke. Split, because two of the
# three are negative assertions that must keep biting on their own.
grep -q 'MISSION_MODEL_PREFS:-claude-opus-5,codex:gpt-6-astra,claude-fable-5-1}' "$driver" \
  && ok "controller ladder is opus-5 -> astra -> fable-5-1" || bad "controller ladder is opus-5 -> astra -> fable-5-1" "wrong ladder"
# The ORDER is the ruling, not just the membership: astra before fable, fable kept
# as what it falls back to. An astra entry placed AFTER fable would satisfy a naive
# membership check and invert the decision.
grep -q 'MISSION_MODEL_PREFS:-[^}]*codex:gpt-6-astra,claude-fable' "$driver" \
  && ok "astra sits AHEAD of fable in the ladder" || bad "astra sits AHEAD of fable in the ladder" "astra not immediately before fable"
# opus-4.8 REMOVED from the controller ladder (Mark 2026-08-26). Negative
# assertion so a silent reintroduction is RED, not merely unasserted.
grep -q 'MISSION_MODEL_PREFS:-[^}]*opus-4-8' "$driver" \
  && bad "controller ladder excludes opus-4.8" "opus-4-8 reintroduced" \
  || ok "controller ladder excludes opus-4.8"
# Fable 5 -> 5.1 (Mark 2026-09-02). The bracket class is load-bearing: plain
# 'claude-fable-5' is a SUBSTRING of 'claude-fable-5-1', so an unanchored negative
# would fire on the correct value and this arm would be permanently red.
grep -q 'MISSION_MODEL_PREFS:-[^}]*claude-fable-5[,}]' "$driver" \
  && bad "controller ladder uses fable-5-1, not bare fable-5" "bare fable-5 in the ladder" \
  || ok "controller ladder uses fable-5-1, not bare fable-5"
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
# AMENDED 2026-09-05 (Mark, attended): astra is ADDED to the controller chain
# BEHIND sol, not ahead of it. Two arms, and they are different claims: the first
# is the pre-existing guarantee that sol still LEADS the chain (it must not be
# displaced); the second is that astra sits immediately behind it. Order is the
# whole point of this change — reversing them would make astra the effective
# codex controller, which is exactly what Mark declined.
grep -q 'MISSION_CONTROLLER_FALLBACK:-codex:gpt-5.6-sol' "$driver" \
  && ok "controller fallback still leads with Codex Sol" || bad "controller fallback still leads with Codex Sol" "sol displaced from the head"
grep -q 'MISSION_CONTROLLER_FALLBACK:-codex:gpt-5.6-sol,codex:gpt-6-astra' "$driver" \
  && ok "Codex Astra is the rung directly behind Sol" || bad "Codex Astra is the rung directly behind Sol" "astra missing or out of order"

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
  PREFS="claude-opus-5,claude-fable-5-1"
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
  PREFS="claude-opus-5,claude-fable-5-1"
  CONTROLLER_FALLBACK="codex:gpt-5.6-sol"
  OVERRIDE_FILE="$2/no-override"
  select_model || exit 1
  /usr/bin/env | grep -E "^(MC_EXPORT_CONTROL|MODEL|MODEL_WHY|CONTROLLER_ID)=" | LC_ALL=C sort | tr "\n" "|"
' _ "$lab/select.sh" "$lab")
want "controller pin is exported to child processes (#696)" "$out" \
  "CONTROLLER_ID=claude:claude-opus-5|MC_EXPORT_CONTROL=sentinel|MODEL=claude-opus-5|MODEL_WHY=probe ok|"
rm -rf "$lab"

# --- M1 RESOLVER (M-SPAWN-PIN-ENFORCEMENT, 2026-09-03) -------------------------
# resolve-role-spawn.sh maps a role's spawn pin to a recipe or agent-tool alias.
# Each arm sets its own env explicitly (env -u / explicit assignments) so no arm
# inherits another's. MISSION_EXECUTOR_RESOLVED is unset in the evaluator arms so
# they exercise the raw MISSION_EXECUTOR_MODEL path (M3 has not landed yet).
RESOLVE="$ROOT/tools/launchd/resolve-role-spawn.sh"

# R1: provider pin (contains ':') -> recipe. Kills a deleted `:`-detection branch.
out=$(MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna "$RESOLVE" executor)
want "R1 provider pin resolves to a recipe" "$out" "recipe codex:gpt-5.6-luna declared:provider-pin"

# R2: bare alias -> agent-tool. Kills a "make every pin a recipe" mutation.
out=$(MISSION_DESIGNER_MODEL=fable "$RESOLVE" designer)
want "R2 bare alias resolves to an agent-tool" "$out" "agent-tool fable declared:alias-pin"

# R3: evaluator alias != executor resolved -> no collision (R4's false-positive control).
out=$(env -u MISSION_EXECUTOR_RESOLVED -u MISSION_EVALUATOR_FALLBACK \
  MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol "$RESOLVE" evaluator)
want "R3 evaluator alias with distinct executor stays an agent-tool" "$out" "agent-tool sonnet declared:alias-pin"

# R4: evaluator alias == executor resolved -> reroute to the fallback chain head.
out=$(env -u MISSION_EXECUTOR_RESOLVED \
  MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_MODEL=sonnet \
  MISSION_EVALUATOR_FALLBACK=pi:ollama/minimax-m3:cloud,pi:openrouter/minimax/minimax-m3 "$RESOLVE" evaluator)
want "R4 evaluator collision reroutes to the fallback head" "$out" "reroute pi:ollama/minimax-m3:cloud generator-equals-judge"

# R4b: same collision but no fallback -> fail closed.
out=$(env -u MISSION_EXECUTOR_RESOLVED -u MISSION_EVALUATOR_FALLBACK \
  MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_MODEL=sonnet "$RESOLVE" evaluator)
want "R4b evaluator collision with no fallback fails closed" "$out" "refuse fail-closed:evaluator-collision-no-fallback"

# R5: planner consumes derive-planner-lane.sh; provider:model lane -> recipe, token copied through.
out=$(MISSION_PLANNER_MODEL=pi:ollama/kimi-k3:cloud "$RESOLVE" planner \
  "$ROOT/tools/launchd/testdata/planner-lane/c-clean-infra.md")
want "R5 planner provider lane maps to a recipe" "$out" "recipe pi:ollama/kimi-k3:cloud declared:codex-ok"

# R6: planner opus lane -> agent-tool opus, reason token copied through verbatim.
out=$(MISSION_PLANNER_MODEL=codex:gpt-5.6-sol "$RESOLVE" planner \
  "$ROOT/tools/launchd/testdata/planner-lane/a-unlisted-language-path.md")
want "R6 planner opus lane maps to agent-tool opus" "$out" "agent-tool opus fail-closed:path-not-in-codex-allowlist"

# R7: unknown role -> fail closed.
out=$("$RESOLVE" judge)
want "R7 unknown role fails closed" "$out" "refuse fail-closed:role-unknown"

# --- M2 SPAWN-PIN HOOK WIRING (M-SPAWN-PIN-ENFORCEMENT, 2026-09-03) -----------
# Arm W: the spawn-pin hook suite must be wired into make/test.mk, or a suite
# that exists but is never invoked is green forever while enforcing nothing —
# the same class as the "driver EXPORTS the planner allowlist" arm above.
grep -q 'test_spawn_pin_hook.sh' "$ROOT/make/test.mk" \
  && ok "spawn-pin hook suite is wired into make/test.mk" \
  || bad "spawn-pin hook suite is wired into make/test.mk" "missing from make/test.mk"

# --- M3 DRIVER EXPORTS + DOCS ALLOWLIST (M-SPAWN-PIN-ENFORCEMENT, 2026-09-03) --
# D1: Layer 3 must publish the resolved plan only after lane degradation has
# finished rewriting the role pins. Moving it beside the initial role exports
# would publish stale values while the driver silently runs different ones.
_res_line=$(grep -n 'export MISSION_CONTROL_ACTIVE=1' "$driver" | head -1 | cut -d: -f1)
_deg_line=$(grep -nF 'codex ${role_lc} lane -> falling back to' "$driver" | head -1 | cut -d: -f1)
if [ -n "$_res_line" ] && [ -n "$_deg_line" ] && [ "$_res_line" -gt "$_deg_line" ]; then
  ok "D1 resolved-role exports come AFTER lane degradation (line $_res_line > $_deg_line)"
else
  bad "D1 resolved-role exports come AFTER lane degradation" "resolved=$_res_line degradation=$_deg_line"
fi

# D2: exercise the real Layer-3 block across a child-process boundary. As in
# the #696 arm, unset ambient values and carry a known-positive export control
# through the SAME /usr/bin/env call so a broken instrument cannot look green.
lab=$(mktemp -d "${TMPDIR:-/tmp}/mission-layer3.XXXXXX") || exit 1
awk '/^# Layer 3 \(M-SPAWN-PIN-ENFORCEMENT\):/,/^unset _role _mv _rv$/' "$driver" > "$lab/layer3.sh"
out=$(/bin/bash -c '
  set -uo pipefail
  unset MISSION_DESIGNER_MODEL MISSION_PLANNER_MODEL MISSION_EXECUTOR_MODEL MISSION_EVALUATOR_MODEL
  unset MISSION_DESIGNER_RESOLVED MISSION_PLANNER_RESOLVED MISSION_EXECUTOR_RESOLVED MISSION_EVALUATOR_RESOLVED
  unset MISSION_DESIGNER_PATH MISSION_PLANNER_PATH MISSION_EXECUTOR_PATH MISSION_EVALUATOR_PATH
  export MC_EXPORT_CONTROL=sentinel
  MISSION_EXECUTOR_MODEL=codex:x
  MISSION_EVALUATOR_MODEL=sonnet
  . "$1"
  /usr/bin/env | grep -E "^(MC_EXPORT_CONTROL|MISSION_(EXECUTOR|EVALUATOR)_(RESOLVED|PATH))=" | LC_ALL=C sort | tr "\n" "|"
' _ "$lab/layer3.sh")
want "D2 resolved role plan is exported to child processes" "$out" \
  "MC_EXPORT_CONTROL=sentinel|MISSION_EVALUATOR_PATH=agent-tool|MISSION_EVALUATOR_RESOLVED=sonnet|MISSION_EXECUTOR_PATH=recipe|MISSION_EXECUTOR_RESOLVED=codex:x|"
rm -rf "$lab"

# Arm 12: the versioned docs allowlist admits top-level scripts/* while the
# exact pre-widening list remains the fail-closed control.
lab=$(mktemp -d "${TMPDIR:-/tmp}/mission-arm12.XXXXXX") || exit 1
cat > "$lab/scripts-doc.md" <<'EOF'
**Planner-Lane**: codex-ok

## Files
- `scripts/verify_examples.go`
EOF
out=$(/bin/bash -c 'unset MISSION_PLANNER_ALLOWLIST; . "$1"; export MISSION_PLANNER_ALLOWLIST; MISSION_PLANNER_MODEL=codex:gpt-5.6-luna "$2" "$3"' \
  _ "$docsenv" "$DERIVE" "$lab/scripts-doc.md")
want "arm12 docs allowlist admits top-level scripts" "$out" "codex:gpt-5.6-luna declared:codex-ok"
PRE_SCRIPTS_AL='tools/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*|docs/*|examples/*|README.md|CHANGELOG.md|.claude/skills/docs-sync/scripts/*'
out=$(MISSION_PLANNER_ALLOWLIST="$PRE_SCRIPTS_AL" MISSION_PLANNER_MODEL=codex:gpt-5.6-luna \
  "$DERIVE" "$lab/scripts-doc.md")
want "arm12 pre-widening allowlist still denies top-level scripts" "$out" "opus fail-closed:path-not-in-codex-allowlist"
rm -rf "$lab"

# --- M4 SKILL DELIVERY GUARDS (M-SPAWN-PIN-ENFORCEMENT, 2026-09-03) ------------
skill="$ROOT/.claude/skills/mission-control/SKILL.md"
if grep -q 'resolve-role-spawn.sh' "$skill" && grep -q 'MISSION-ROLE:' "$skill"; then
  ok "S1 mission-control skill invokes resolver and requires role tokens"
else
  bad "S1 mission-control skill invokes resolver and requires role tokens" "resolver call or MISSION-ROLE token missing"
fi
# The longer literal is line-wrapped and cannot be matched by line-oriented grep.
grep -q 'enum in this build lists' "$skill" \
  && ok "S2 fable capability paragraph survives the spawn-pattern edit" \
  || bad "S2 fable capability paragraph survives the spawn-pattern edit" "capability control missing"

# S3/S4/S5 — astra's placement, and the collision it creates. Rewritten 2026-09-05
# after Mark corrected the first attempt: astra is an ADDITIONAL fable-class entry
# to vary between, NOT a replacement for fable's slot.
grep -q 'now `claude:claude-fable-5-1` → `codex:gpt-6-astra` → `pi:ollama/deepseek-v4-flash:0731-cloud` → repeat' "$skill" \
  && ok "S3 designer rotation is fable -> astra -> deepseek (astra ADDED, fable kept)" \
  || bad "S3 designer rotation is fable -> astra -> deepseek (astra ADDED, fable kept)" "rotation is not the three-entry list"
# The driver seed must NOT have moved: astra is a rotation entry, so nothing pins it.
# This is the arm that dies if someone re-applies the "astra takes the fable slot"
# version, which looked identical in a role table and was not what was asked for.
grep -q 'MISSION_DESIGNER_MODEL:-claude:claude-fable-5-1' "$driver" \
  && ok "S4 designer seed is still fable (astra is an entry, not a pin)" \
  || bad "S4 designer seed is still fable (astra is an entry, not a pin)" "seed moved off fable"
# S5: astra sits in the designer rotation AND in the default quorum roster, so on
# astra's turn the author is one of its own reviewers. That is a real defect with a
# named workaround, not a footnote — this arm fails if the quorum default gains
# astra while the skill stops carrying the substitution instruction, i.e. if the
# collision ever becomes undocumented.
if grep -q 'gpt6-astra,gemini-3-1-pro,oc-glm-5-2' cmd/ailang/design_quorum.go; then
  if grep -q 'ASTRA IS ALSO A QUORUM REVIEWER' "$skill"; then
    ok "S5 astra-in-quorum collision is documented where the designer is chosen"
  else
    bad "S5 astra-in-quorum collision is documented where the designer is chosen" "quorum names astra but the rotation row does not warn"
  fi
else
  ok "S5 astra-in-quorum collision is documented where the designer is chosen"
fi

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
