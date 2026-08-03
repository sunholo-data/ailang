#!/usr/bin/env bash
# Run evaluation baseline for a release version.
#
# v0.14.0+: benchmark selection is driven by the tier system, not a hardcoded list.
# Default release scope is `core,stretch`. Pass --tier to override.
#
# Ollama local models: models with agent_model_name starting with "ollama/" are
# detected automatically. The script pins them in memory for the eval run (via a
# keepalive API call) and unloads them when done — so the laptop isn't permanently
# full. Set OLLAMA_EVAL_KEEPALIVE to override the pin duration (default: 60m).

set -euo pipefail

OLLAMA_EVAL_KEEPALIVE="${OLLAMA_EVAL_KEEPALIVE:-60m}"
OLLAMA_API="${OLLAMA_HOST:-http://localhost:11434}"

# ELO ratings DB (M-EVAL-STANDARD-CONFIDENCE-GATING). Same path
# cmd/ailang/eval_confidence.go's defaultObservatoryDB() resolves to.
OBSERVATORY_DB="${OBSERVATORY_DB:-$HOME/.ailang/state/observatory.db}"

# Default aggregate budget for --full standard-eval runs (M-EVAL-STANDARD-CONFIDENCE-GATING
# Design Freeze: ~15% headroom over the highest real combined baseline seen,
# v0.29.2's $135). Dev-mode runs get no cap unless --budget-usd is passed explicitly.
BUDGET_USD_DEFAULT=150

# True if a prior Step 1 invocation into $RESULTS_DIR hit its --budget-usd cap
# (cmd/ailang/eval_parallel.go writes this sentinel on breach). Checked between
# each call in the confidence-gated sequence so a cap hit during, say, the
# new-models call stops the rated-models calls from starting at all — and
# after Step 1 completes, to mark baseline.json as a partial run.
budget_stopped() {
    [[ -f "$RESULTS_DIR/budget_stopped.json" ]]
}

# Extract Ollama model names from the models.yml for a given comma-separated
# list of model IDs. Returns newline-separated "ollama/<model>" strings.
# Usage: get_ollama_models "opencode-gemma4-e4b,claude-sonnet-4-5"
get_ollama_models() {
    local model_ids="$1"
    local yml="internal/eval_harness/models.yml"
    [[ -f "$yml" ]] || return 0

    local IFS=','
    for id in $model_ids; do
        # Look for agent_model_name: "ollama/..." under this model's block
        local name
        name=$(awk "/^  ${id}:/{found=1} found && /agent_model_name:/{print; exit}" "$yml" \
               | grep -oE '"ollama/[^"]+"' | tr -d '"')
        [[ -n "$name" ]] && echo "$name"
    done
    # ALWAYS return 0: under `set -e`, the loop's last `[[ ... ]] && echo` returns 1
    # when the final model isn't ollama-backed (the common case for cloud/OS suites),
    # which would otherwise kill the whole script at the setup_ollama_models call and
    # silently skip the entire agent step. (Fixed 2026-06-05.)
    return 0
}

# Pin an Ollama model in memory for the duration of the eval run.
# Fires a minimal generate request with keep_alive=OLLAMA_EVAL_KEEPALIVE.
# No-ops silently if Ollama is not running.
ollama_warmup() {
    local model="$1"           # e.g. "ollama/gemma4:e4b"
    local ollama_model="${model#ollama/}"  # strip "ollama/" prefix for API
    echo "→ Warming up Ollama model: $ollama_model (keepalive=${OLLAMA_EVAL_KEEPALIVE})"
    local response
    response=$(curl -s --max-time 300 "${OLLAMA_API}/api/generate" \
        -d "{\"model\":\"${ollama_model}\",\"prompt\":\"hi\",\"stream\":false,\"keep_alive\":\"${OLLAMA_EVAL_KEEPALIVE}\"}" \
        2>&1)
    if echo "$response" | grep -q '"done":true'; then
        echo "  ✓ $ollama_model loaded and pinned"
    else
        echo "  ⚠ Could not warm up $ollama_model (Ollama running?)"
    fi
}

# Unload an Ollama model immediately after eval to free RAM.
ollama_unload() {
    local model="$1"
    local ollama_model="${model#ollama/}"
    echo "→ Unloading Ollama model: $ollama_model"
    curl -sf --max-time 10 "${OLLAMA_API}/api/generate" \
        -d "{\"model\":\"${ollama_model}\",\"prompt\":\"\",\"keep_alive\":\"0\"}" \
        > /dev/null 2>&1 \
        && echo "  ✓ $ollama_model unloaded" \
        || echo "  ⚠ Could not unload $ollama_model"
}

# Warmup all Ollama models in a model list, unload them on EXIT.
# Usage: setup_ollama_models "opencode-gemma4-e4b,opencode-gemma4-26b"
OLLAMA_MODELS_TO_UNLOAD=()
setup_ollama_models() {
    local model_ids="$1"
    local ollama_models
    ollama_models=$(get_ollama_models "$model_ids")
    [[ -z "$ollama_models" ]] && return 0

    echo ""
    echo "=== Ollama Model Setup ==="
    while IFS= read -r m; do
        ollama_warmup "$m"
        OLLAMA_MODELS_TO_UNLOAD+=("$m")
    done <<< "$ollama_models"
    echo ""

    # Register teardown — runs on EXIT (normal finish, Ctrl-C, or error)
    trap 'teardown_ollama_models' EXIT
}

teardown_ollama_models() {
    [[ ${#OLLAMA_MODELS_TO_UNLOAD[@]} -eq 0 ]] && return 0
    echo ""
    echo "=== Ollama Model Teardown ==="
    for m in "${OLLAMA_MODELS_TO_UNLOAD[@]}"; do
        ollama_unload "$m"
    done
}

# Default tier scope for release baselines:
#   - `core`     = headline metric (19 benchmarks post-v0.29.0 re-tier)
#   - `stretch`  = harder benchmarks we expect mixed results on (29 benchmarks)
#   - `frontier` = top-end discriminators, v0.29.0+ (8 benchmarks; at least one
#                  frontier model must FAIL each in standard mode or it demotes —
#                  release baselines are the only routine source of that data)
# Vision tier (research-grade) is excluded by default; smoke tier is a subset of
# CI sanity checks and also excluded by default to keep releases focused.
DEFAULT_TIER="core,stretch,frontier"

# Count benchmarks matching a tier spec (comma-separated tiers).
# Usage: count_benchmarks_in_tiers "core,stretch"
count_benchmarks_in_tiers() {
    local tier_csv="$1"
    local total=0
    local tier
    IFS=',' read -ra TIERS <<< "$tier_csv"
    for tier in "${TIERS[@]}"; do
        local c
        c=$(grep -l "^tier: ${tier}\b" benchmarks/*.yml 2>/dev/null | wc -l | tr -d ' ')
        total=$((total + c))
    done
    echo "$total"
}

# Resolve a tier spec to a comma-separated list of benchmark IDs.
# Agent mode requires explicit --benchmarks (safety guardrail); this expands
# the tier selection so we pass a vetted list to `ailang eval-suite --agent`.
# Usage: resolve_benchmarks_in_tiers "core,stretch"
resolve_benchmarks_in_tiers() {
    local tier_csv="$1"
    local ids=()
    local tier
    IFS=',' read -ra TIERS <<< "$tier_csv"
    for tier in "${TIERS[@]}"; do
        while IFS= read -r f; do
            # Benchmark id = filename without .yml extension
            local base
            base="$(basename "$f" .yml)"
            ids+=("$base")
        done < <(grep -l "^tier: ${tier}\b" benchmarks/*.yml 2>/dev/null | sort)
    done
    local IFS=','
    echo "${ids[*]}"
}

# --- Confidence gating (M-EVAL-STANDARD-CONFIDENCE-GATING) ---
#
# Extends the ELO confidence-select mechanism already shipped for the local rig
# (M-EVAL-RATING-EFFICIENCY, --benchmarks-by-confidence) to the cloud standard
# baseline. Gating is opt-in (--confidence-gate) and applies ONLY to core+stretch;
# frontier always runs in full for every model, every release — its curation
# contract (see SKILL.md) requires routine full-coverage failure data. A model in
# extended_suite with no standard-mode rating history always gets the full tier,
# regardless of gating state, since the selector has nothing to gate it against.

# Extract the extended_suite model ID list from models.yml. Returns
# newline-separated model IDs. Empty output on any parse failure (fail-open —
# caller treats "no models" the same as "everything is new").
extended_suite_models() {
    local yml="internal/eval_harness/models.yml"
    [[ -f "$yml" ]] || return 0
    awk '
        /^extended_suite:/ { insuite=1; next }
        insuite && /^[a-zA-Z]/ { exit }
        insuite && /^  - "/ { gsub(/^  - "|".*$/, ""); print }
    ' "$yml"
    return 0
}

# Query observatory.db for models with standard-mode rating coverage. Returns
# newline-separated model IDs, or nothing if sqlite3/the DB is unavailable —
# fail-open (caller treats "no coverage data" as "every model is new", which
# only matters once the confidence-gate probe below has already succeeded;
# if the probe fails the whole gated path is abandoned before this is used).
rated_standard_models() {
    command -v sqlite3 >/dev/null 2>&1 || return 0
    [[ -f "$OBSERVATORY_DB" ]] || return 0
    sqlite3 "$OBSERVATORY_DB" "SELECT model_id FROM model_ratings WHERE mode='standard';" 2>/dev/null || true
}

# Probe whether standard-mode confidence gating is usable at all, and if so,
# print the confidence-selected benchmark list intersected with core+stretch
# (frontier is excluded here — it's handled by a separate, always-full call).
# Prints nothing and returns 1 if the ratings DB has no standard-mode data
# (selectBenchmarksByConfidence's own error — this is the fail-open path);
# prints a possibly-empty benchmark list and returns 0 otherwise. An empty
# (but successful) list means core+stretch is fully saturated for the
# population — the caller should skip that call entirely, not pass "" through
# (an empty --benchmarks string would auto-discover ALL benchmarks instead).
confidence_gated_core_stretch() {
    local probe
    if ! probe=$(bin/ailang eval-suite --benchmarks-by-confidence auto --max-benchmarks 0 --dry-run 2>&1); then
        return 1
    fi
    local selected
    selected=$(echo "$probe" | grep "^Benchmarks:" | sed 's/^Benchmarks: *//')
    local core_stretch
    core_stretch="$(resolve_benchmarks_in_tiers "core,stretch")"
    # Intersect selected (comma-space-separated, per eval_suite.go's dry-run
    # printer) with core,stretch (plain comma-separated) — sorted-set
    # intersection via comm. Both sides MUST be trimmed of whitespace before
    # sorting: comma-space splitting leaves a leading space on every token
    # after the first, which silently breaks comm's exact-match comparison
    # down to just the first (unindented) entry.
    comm -12 \
        <(echo "$selected" | tr ',' '\n' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sort -u) \
        <(echo "$core_stretch" | tr ',' '\n' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sort -u) \
        | paste -sd ',' -
    return 0
}

# Runs the confidence-gated standard eval sequence: models with no standard-mode
# rating history get the full tier (nothing to gate them against yet); rated
# models get the confidence-gated core+stretch selection plus an always-full
# frontier call. Falls open to today's single full-tier call if the ratings DB
# isn't seeded/usable. Checks budget_stopped() between each call so a cap hit
# partway through the sequence stops the remaining calls from starting.
# Usage: run_confidence_gated_standard <version_normalized> <tier_flag> <budget_usd>
run_confidence_gated_standard() {
    local version_normalized="$1"
    local tier_flag="$2"
    local budget_usd="$3"

    # bin/ailang must reflect current source before the dry-run probe below —
    # normally `make eval-baseline`'s own `build` prerequisite handles this,
    # but the probe calls bin/ailang directly, ahead of any make invocation.
    make -s build

    local gated_core_stretch
    if ! gated_core_stretch="$(confidence_gated_core_stretch)"; then
        echo "confidence-gate: standard ratings unavailable — falling back to full tier"
        TIER="$tier_flag" BUDGET_USD="$budget_usd" make eval-baseline EVAL_VERSION="v$version_normalized" FULL=true
        return 0
    fi

    local ext_models rated_models new_models rated_subset
    ext_models="$(extended_suite_models)"
    rated_models="$(rated_standard_models)"
    new_models=""
    rated_subset=""
    while IFS= read -r m; do
        [[ -z "$m" ]] && continue
        if echo "$rated_models" | grep -qx "$m" 2>/dev/null; then
            rated_subset="${rated_subset:+$rated_subset,}$m"
        else
            new_models="${new_models:+$new_models,}$m"
        fi
    done <<< "$ext_models"

    local new_count rated_count
    new_count=$(echo "$new_models" | tr ',' '\n' | grep -c . || true)
    rated_count=$(echo "$rated_subset" | tr ',' '\n' | grep -c . || true)
    echo "confidence-gate: $new_count new model(s) (no standard rating history), $rated_count rated model(s)"

    if [[ -n "$new_models" ]]; then
        echo "confidence-gate: new models get the full tier — $tier_flag"
        MODELS="$new_models" TIER="$tier_flag" RESUME=true BUDGET_USD="$budget_usd" make eval-baseline EVAL_VERSION="v$version_normalized"
        if budget_stopped; then
            echo "confidence-gate: budget cap hit during the new-models call — stopping the sequence here"
            return 0
        fi
    fi

    if [[ -n "$rated_subset" ]]; then
        if [[ -n "$gated_core_stretch" ]]; then
            local gated_count
            gated_count=$(echo "$gated_core_stretch" | tr ',' '\n' | grep -c . || true)
            echo "confidence-gate: rated models — gated core+stretch ($gated_count of $(count_benchmarks_in_tiers "core,stretch") benchmarks)"
            MODELS="$rated_subset" BENCHMARKS="$gated_core_stretch" RESUME=true BUDGET_USD="$budget_usd" make eval-baseline EVAL_VERSION="v$version_normalized"
            if budget_stopped; then
                echo "confidence-gate: budget cap hit during the gated core+stretch call — stopping the sequence here (frontier skipped)"
                return 0
            fi
        else
            echo "confidence-gate: core+stretch fully saturated for rated models — skipping (max savings case, not an error)"
        fi
        echo "confidence-gate: rated models — frontier (always full, every release)"
        MODELS="$rated_subset" TIER=frontier RESUME=true BUDGET_USD="$budget_usd" make eval-baseline EVAL_VERSION="v$version_normalized"
    fi
}

# Validation mode: check configuration without running full eval
if [[ "${1:-}" == "--validate" ]]; then
    echo "Validating agent eval configuration..."

    # Verify ailang command exists
    if ! command -v ailang &> /dev/null; then
        echo "ERROR: ailang command not found"
        exit 1
    fi
    echo "  ailang command: found"

    # Report tier counts
    for tier in smoke core stretch vision; do
        count=$(grep -l "^tier: ${tier}\b" benchmarks/*.yml 2>/dev/null | wc -l | tr -d ' ')
        echo "  $tier benchmarks: $count"
    done

    default_count=$(count_benchmarks_in_tiers "$DEFAULT_TIER")
    echo "  Default tier scope: $DEFAULT_TIER ($default_count benchmarks)"
    echo "  ✓ Agent eval configuration valid"
    exit 0
fi

# Progress monitoring (runs in background)
monitor_progress() {
    local results_dir="$1"
    local expected_count="$2"
    local phase="$3"

    echo "[$phase] Monitoring progress..."
    while true; do
        sleep 120  # Report every 2 minutes

        if [[ -d "$results_dir" ]]; then
            current=$(find "$results_dir" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
            if [[ $expected_count -gt 0 ]]; then
                percent=$((current * 100 / expected_count))
                echo "[$phase] Progress: $current/$expected_count files ($percent%)"
            else
                echo "[$phase] Progress: $current files"
            fi
        fi
    done
}

# Arg parsing: <version> [--full] [--tier <spec>] [--cross-harness]
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version> [--full] [--tier <spec>] [--cross-harness]" >&2
    echo "Example: $0 0.3.14 --full" >&2
    echo "Example: $0 0.13.0 --full --tier core,stretch" >&2
    echo "Example: $0 0.15.0 --full --cross-harness" >&2
    echo "" >&2
    echo "Options:" >&2
    echo "  --full              Use all production models (default: dev models)" >&2
    echo "  --tier <spec>       Comma-separated tiers: smoke,core,stretch,frontier,vision" >&2
    echo "                      (default: $DEFAULT_TIER)" >&2
    echo "  --cross-harness     Use harness_suite (sonnet+gemini via claude/opencode)" >&2
    echo "                      to measure harness-induced benchmark deltas" >&2
    echo "  --lang-harness      Add Step 3: 4-language sweep using lang_harness_suite" >&2
    echo "                      (cheapest model per harness × ailang,python,js,go × core)" >&2
    echo "                      Feeds the Agent Harness Explorer language/harness data" >&2
    echo "  --skip-existing     Pass --skip-existing to eval-suite (resume an interrupted run)" >&2
    echo "  --agent-only        Skip Step 1 (standard); run only the agent step. Implies" >&2
    echo "                      --skip-existing. Use when the standard baseline is already done." >&2
    echo "  --confidence-gate   Standard eval only: skip re-confirming Trivial-band (saturated)" >&2
    echo "                      core/stretch benchmarks per ELO ratings in observatory.db" >&2
    echo "                      (M-EVAL-STANDARD-CONFIDENCE-GATING). frontier always stays full." >&2
    echo "                      A model with no standard-mode rating history always gets the" >&2
    echo "                      full tier. Falls open to today's full-tier behavior if the" >&2
    echo "                      ratings DB is unseeded/unavailable. Opt-in; off by default." >&2
    echo "  --budget-usd <N>    Aggregate cost ceiling for Step 1 (standard eval). Default for" >&2
    echo "                      --full runs: \$$BUDGET_USD_DEFAULT. Dev-mode runs get no cap unless" >&2
    echo "                      set explicitly. Pass 0 to disable the cap on a --full run." >&2
    exit 1
fi

VERSION="$1"
shift
FULL_FLAG=""
TIER_FLAG="$DEFAULT_TIER"
CROSS_HARNESS=""
LANG_HARNESS=""
SKIP_EXISTING=""   # when set, pass --skip-existing to eval-suite (resume runs)
AGENT_ONLY=""      # when set, skip the standard step entirely
CONFIDENCE_GATE=""  # when set, gate core/stretch by ELO confidence (M-EVAL-STANDARD-CONFIDENCE-GATING)
BUDGET_USD_FLAG=""  # explicit --budget-usd override; empty = use the --full default

while [[ $# -gt 0 ]]; do
    case "$1" in
        --full)
            FULL_FLAG="FULL=true"
            shift
            ;;
        --skip-existing)
            SKIP_EXISTING="--skip-existing"
            shift
            ;;
        --agent-only)
            AGENT_ONLY="true"
            SKIP_EXISTING="--skip-existing"
            shift
            ;;
        --confidence-gate)
            CONFIDENCE_GATE="true"
            shift
            ;;
        --budget-usd)
            if [[ $# -lt 2 ]]; then
                echo "ERROR: --budget-usd requires an argument" >&2
                exit 1
            fi
            BUDGET_USD_FLAG="$2"
            shift 2
            ;;
        --budget-usd=*)
            BUDGET_USD_FLAG="${1#--budget-usd=}"
            shift
            ;;
        --tier)
            if [[ $# -lt 2 ]]; then
                echo "ERROR: --tier requires an argument" >&2
                exit 1
            fi
            TIER_FLAG="$2"
            shift 2
            ;;
        --tier=*)
            TIER_FLAG="${1#--tier=}"
            shift
            ;;
        --cross-harness)
            CROSS_HARNESS="true"
            shift
            ;;
        --lang-harness)
            LANG_HARNESS="true"
            shift
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

# Normalize version: ensure directory always has "v" prefix
VERSION_NORMALIZED="${VERSION#v}"
RESULTS_DIR="eval_results/baselines/v$VERSION_NORMALIZED"

# Resolve the effective standard-eval budget (M-EVAL-STANDARD-CONFIDENCE-GATING
# Design Freeze): explicit --budget-usd always wins. Otherwise --full runs
# default to BUDGET_USD_DEFAULT; dev-mode runs get no cap (empty = unset,
# matching eval-suite's own "0 or empty = no cap" semantics downstream).
if [[ -n "$BUDGET_USD_FLAG" ]]; then
    EFFECTIVE_BUDGET_USD="$BUDGET_USD_FLAG"
elif [[ -n "$FULL_FLAG" ]]; then
    EFFECTIVE_BUDGET_USD="$BUDGET_USD_DEFAULT"
else
    EFFECTIVE_BUDGET_USD=""
fi

# Compute expected benchmark count for the selected tier(s)
BENCHMARK_COUNT=$(count_benchmarks_in_tiers "$TIER_FLAG")
if [[ "$BENCHMARK_COUNT" -eq 0 ]]; then
    echo "ERROR: no benchmarks match --tier $TIER_FLAG" >&2
    echo "Hint: valid tiers are smoke, core, stretch, vision" >&2
    exit 1
fi

echo "Running eval baseline for $VERSION..."
echo "Tier scope: $TIER_FLAG ($BENCHMARK_COUNT benchmarks)"
if [[ -n "$FULL_FLAG" ]]; then
    echo "Mode: FULL (extended_suite, 18 models incl. claude-fable-5 + claude-opus-5, + agent_suite)"
    echo "Expected cost: run \`ailang eval-suite --full --tier $TIER_FLAG --dry-run\` for a real"
    echo "  computed estimate (M-EVAL-STANDARD-CONFIDENCE-GATING) — recent full releases banked"
    echo "  \$98-135 combined (standard+agent); see SKILL.md's Cost & time section for the real figures."
    echo "Expected time: ~45-90 minutes"
else
    echo "Mode: DEV (3 dev models)"
    echo "Expected cost: ~\$0.10-0.20"
    echo "Expected time: ~5-10 minutes"
fi
if [[ -n "$EFFECTIVE_BUDGET_USD" ]]; then
    echo "Standard-eval budget cap: \$$EFFECTIVE_BUDGET_USD (see \`ailang eval-suite --dry-run\` for a real per-run estimate)"
fi
echo

# Step 1: Run standard eval baseline (skipped entirely with --agent-only)
if [[ -n "$AGENT_ONLY" ]]; then
    echo "=== Step 1/2: Standard Eval — SKIPPED (--agent-only) ==="
    EXPECTED_STANDARD=0
else
    echo "=== Step 1/2: Standard Eval (0-shot + repair) ==="
    # Expected file count for standard eval: benchmarks × models × langs (2)
    if [[ -n "$FULL_FLAG" ]]; then
        EXPECTED_STANDARD=$((BENCHMARK_COUNT * 6 * 2))   # ~6 extended models
    else
        EXPECTED_STANDARD=$((BENCHMARK_COUNT * 3 * 2))   # 3 dev models
    fi

    monitor_progress "$RESULTS_DIR" "$EXPECTED_STANDARD" "Standard" &
    MONITOR_PID=$!
    if [[ -n "$CONFIDENCE_GATE" && -n "$FULL_FLAG" ]]; then
        run_confidence_gated_standard "$VERSION_NORMALIZED" "$TIER_FLAG" "$EFFECTIVE_BUDGET_USD"
    elif [[ -n "$FULL_FLAG" ]]; then
        TIER="$TIER_FLAG" BUDGET_USD="$EFFECTIVE_BUDGET_USD" make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED" FULL=true
    else
        TIER="$TIER_FLAG" BUDGET_USD="$EFFECTIVE_BUDGET_USD" make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED"
    fi
    kill $MONITOR_PID 2>/dev/null || true

    # Mark baseline.json as a partial run if the budget cap fired at any point
    # during Step 1 (mirrors the .agent metadata augmentation pattern below).
    if budget_stopped; then
        echo "⚠️  Budget cap hit during Step 1 — baseline.json will be marked budget_stopped: true"
        BASELINE_METADATA_STD="$RESULTS_DIR/baseline.json"
        if [[ -f "$BASELINE_METADATA_STD" ]]; then
            SENTINEL="$RESULTS_DIR/budget_stopped.json"
            tmp=$(mktemp)
            jq --slurpfile sentinel "$SENTINEL" \
               '.budget_stopped = true | .budget_stopped_detail = $sentinel[0]' \
               "$BASELINE_METADATA_STD" > "$tmp" && mv "$tmp" "$BASELINE_METADATA_STD"
        fi
    fi

    # Self-seed standard-mode ratings from this release's own results, so the
    # NEXT release's gating reflects the most current data (closes the loop —
    # same self-refreshing pattern the local rig already uses for agent mode).
    # Only on --full runs: dev mode's 3 cheap models would otherwise pollute
    # release-grade ratings with noise from a tiny, non-representative run.
    if [[ -n "$FULL_FLAG" ]]; then
        echo "confidence-gate: re-fitting standard ratings from this release's results..."
        go run ./tools/eval-elo --mode standard --persist "$OBSERVATORY_DB" "$RESULTS_DIR" \
            || echo "confidence-gate: WARNING — ratings re-fit failed, next release will not see this data until it's re-run manually"
    fi
fi

# Step 2: Run agent eval on tier-selected benchmarks
echo
echo "=== Step 2/2: Agent Eval (multi-turn) ==="
echo "Running agent eval on tier=$TIER_FLAG benchmarks ($BENCHMARK_COUNT total)..."
echo

# Agent model selection:
#   DEV mode         → claude-sonnet-4-6 + gemini-3-flash (2 harnesses, fast)
#   FULL mode        → agent_suite (all 4 harnesses: claude + gemini + codex + opencode)
#   --cross-harness  → harness_suite (sonnet+opencode-sonnet, gemini+opencode-gemini)
#                      measures harness-induced delta for the same underlying model
#
# agent_suite requires OPENAI_API_KEY (codex) and a running opencode binary.
# harness_suite requires opencode binary only.
# Missing harnesses are skipped gracefully by the executor factory.
if [[ -n "$CROSS_HARNESS" ]]; then
    AGENT_MODELS="harness_suite"
    AGENT_HARNESS_DESC="harness_suite (claude-sonnet-4-6, opencode-sonnet-4-6, gemini-3-flash, opencode-gemini-3-flash)"
    AGENT_HARNESS_COUNT=4
elif [[ -n "$FULL_FLAG" ]]; then
    AGENT_MODELS="agent_suite"
    AGENT_HARNESS_DESC="agent_suite (sonnet + flash + gpt5-mini + opencode-haiku — sonnet kept as longitudinal anchor)"
    AGENT_HARNESS_COUNT=4
else
    AGENT_MODELS="claude-sonnet-4-6,gemini-3-flash"
    AGENT_HARNESS_DESC="claude + gemini (2 harnesses)"
    AGENT_HARNESS_COUNT=2
fi

# Pre-warm any Ollama-backed models now that AGENT_MODELS is resolved.
# Eliminates cold starts and ensures RAM is freed on EXIT/Ctrl-C.
setup_ollama_models "$AGENT_MODELS"

# Pre-flight summary
echo "=== Pre-Flight Check ==="
echo "Version: $VERSION"
echo "Tier: $TIER_FLAG ($BENCHMARK_COUNT benchmarks)"
echo "Mode: ${FULL_FLAG:+FULL}${FULL_FLAG:-DEV}"
echo "Agent models: $AGENT_HARNESS_DESC"
echo "Agent parallelism: 2"
echo

# Agent mode is AILANG-only (M-EVAL agent redesign 2026-07-11). The agent-loop uplift
# question is about AILANG; Python capability is covered by the standard run. Applies to
# both FULL and dev agent runs.
# Agent expected file counts: benchmarks × harnesses × langs(=1)
EXPECTED_AGENT=$((BENCHMARK_COUNT * AGENT_HARNESS_COUNT * 1))
AGENT_LANGS="ailang"

echo "Expected results:"
echo "  Standard eval: ~$EXPECTED_STANDARD files"
echo "  Agent eval: ~$EXPECTED_AGENT files"
echo "  Total: ~$((EXPECTED_STANDARD + EXPECTED_AGENT)) files"
echo
echo "Starting in 3 seconds... (Ctrl-C to abort)"
sleep 3
echo

# Agent mode requires explicit --benchmarks (CLI safety guardrail), so expand
# the tier spec to a vetted benchmark list.
AGENT_BENCHMARKS_CSV="$(resolve_benchmarks_in_tiers "$TIER_FLAG")"
if [[ -z "$AGENT_BENCHMARKS_CSV" ]]; then
    echo "ERROR: could not resolve --tier $TIER_FLAG to any benchmark IDs" >&2
    exit 1
fi

echo "Mode: ${CROSS_HARNESS:+CROSS-HARNESS }${FULL_FLAG:+FULL }${FULL_FLAG:-DEV} (langs=$AGENT_LANGS)"
echo "Models: $AGENT_HARNESS_DESC"
echo "Agent benchmarks: $BENCHMARK_COUNT resolved from tier=$TIER_FLAG"
monitor_progress "$RESULTS_DIR" "$EXPECTED_AGENT" "Agent" &
MONITOR_PID=$!
ailang eval-suite --agent \
    --models "$AGENT_MODELS" \
    --benchmarks "$AGENT_BENCHMARKS_CSV" \
    --langs "$AGENT_LANGS" \
    --parallel 2 \
    ${SKIP_EXISTING} \
    --output "$RESULTS_DIR"
kill $MONITOR_PID 2>/dev/null || true

# Augment baseline.json with an "agent" object capturing what the agent stage
# actually measured. Locks in the resolved benchmark list so cross-release
# agent comparisons can detect set drift (see feedback_* memory for rationale).
BASELINE_METADATA="$RESULTS_DIR/baseline.json"
if [[ -f "$BASELINE_METADATA" ]]; then
    AGENT_IDS_JSON=$(find "$RESULTS_DIR/agent" -name "*.json" -type f \
        -exec jq -r '.id' {} \; 2>/dev/null | sort -u | jq -R . | jq -sc .)
    AGENT_IDS_JSON="${AGENT_IDS_JSON:-[]}"
    AGENT_COUNT_JSON=$(echo "$AGENT_IDS_JSON" | jq 'length')
    AGENT_FILE_COUNT=$(find "$RESULTS_DIR/agent" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

    tmp=$(mktemp)
    jq --arg tier "$TIER_FLAG" \
       --arg models "$AGENT_MODELS" \
       --arg langs "$AGENT_LANGS" \
       --argjson count "$AGENT_COUNT_JSON" \
       --argjson files "$AGENT_FILE_COUNT" \
       --argjson resolved "$AGENT_IDS_JSON" \
       '.agent = {tier_spec: $tier, count: $count, files: $files, models: $models, langs: $langs, resolved: $resolved}' \
       "$BASELINE_METADATA" > "$tmp" && mv "$tmp" "$BASELINE_METADATA"
    echo "✓ baseline.json augmented with agent stage metadata ($AGENT_COUNT_JSON benchmarks, $AGENT_FILE_COUNT files)"
else
    echo "⚠️  baseline.json not found at $BASELINE_METADATA — skipping agent metadata augmentation"
fi

# Show combined results
echo
if [[ -d "$RESULTS_DIR" ]]; then
    STANDARD_COUNT=$(find "$RESULTS_DIR/standard" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
    AGENT_COUNT=$(find "$RESULTS_DIR/agent" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
    TOTAL_COUNT=$((STANDARD_COUNT + AGENT_COUNT))
    echo "✓ Baseline complete"
    echo "  Results: $RESULTS_DIR"
    echo "  Standard eval: $STANDARD_COUNT files"
    echo "  Agent eval: $AGENT_COUNT files"
    echo "  Total: $TOTAL_COUNT files"
else
    echo "✗ Results directory not found: $RESULTS_DIR" >&2
    exit 1
fi

# Validation
echo
echo "=== Validation ==="
EXPECTED_TOTAL=$((EXPECTED_STANDARD + EXPECTED_AGENT))

# Allow tolerance for filtering/failures
MIN_STANDARD=$((EXPECTED_STANDARD * 95 / 100))
MIN_AGENT=$((EXPECTED_AGENT * 85 / 100))   # Agent eval has more variance
MIN_TOTAL=$((EXPECTED_TOTAL * 90 / 100))

if [[ $STANDARD_COUNT -lt $MIN_STANDARD ]]; then
    echo "⚠️  WARNING: Standard eval produced fewer files than expected"
    echo "   Expected: ~$EXPECTED_STANDARD, Got: $STANDARD_COUNT (minimum: $MIN_STANDARD)"
fi

if [[ $AGENT_COUNT -lt $MIN_AGENT ]]; then
    echo "⚠️  WARNING: Agent eval produced fewer files than expected"
    echo "   Expected: ~$EXPECTED_AGENT, Got: $AGENT_COUNT (minimum: $MIN_AGENT)"
fi

if [[ $TOTAL_COUNT -lt $MIN_TOTAL ]]; then
    echo "⚠️  WARNING: Total file count below expected minimum"
    echo "   Expected: ~$EXPECTED_TOTAL, Got: $TOTAL_COUNT (minimum: $MIN_TOTAL)"
    echo "   This may indicate interrupted runs or configuration issues"
else
    echo "✓ File counts within expected range"
fi

# Step 3: Language × Harness sweep (only when --lang-harness was used).
# Runs lang_harness_suite (cheapest model per harness) across all 4 langs on core tier.
# This feeds the Agent Harness Explorer language/harness comparison data.
# Separate from Step 2 (ailang+python flagship models) so the two stories stay distinct.
if [[ -n "$LANG_HARNESS" ]]; then
    echo
    echo "=== Step 3: Language × Harness Sweep (4 langs × cheap models) ==="
    LANG_HARNESS_TIER="core"
    LANG_HARNESS_COUNT=$(count_benchmarks_in_tiers "$LANG_HARNESS_TIER")
    LANG_HARNESS_BENCHMARKS="$(resolve_benchmarks_in_tiers "$LANG_HARNESS_TIER")"
    LANG_HARNESS_LANGS="ailang,python,javascript,go"
    LANG_HARNESS_MODELS="lang_harness_suite"
    LANG_HARNESS_HARNESS_COUNT=4
    EXPECTED_LANG_HARNESS=$((LANG_HARNESS_COUNT * LANG_HARNESS_HARNESS_COUNT * 4))

    echo "Models: lang_harness_suite (claude-haiku-4-5, gemini-3-flash, gpt5-4-mini, opencode-haiku)"
    echo "Langs: $LANG_HARNESS_LANGS"
    echo "Tier: $LANG_HARNESS_TIER ($LANG_HARNESS_COUNT benchmarks)"
    echo "Expected results: ~$EXPECTED_LANG_HARNESS files"
    echo

    setup_ollama_models "$LANG_HARNESS_MODELS"

    monitor_progress "$RESULTS_DIR" "$EXPECTED_LANG_HARNESS" "LangHarness" &
    MONITOR_PID=$!
    ailang eval-suite --agent \
        --models "$LANG_HARNESS_MODELS" \
        --benchmarks "$LANG_HARNESS_BENCHMARKS" \
        --langs "$LANG_HARNESS_LANGS" \
        --parallel 2 \
        --output "$RESULTS_DIR"
    kill $MONITOR_PID 2>/dev/null || true

    LANG_HARNESS_COUNT_ACTUAL=$(find "$RESULTS_DIR/agent" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
    echo "✓ Language × Harness sweep complete ($LANG_HARNESS_COUNT_ACTUAL total agent files)"
fi

# Cross-harness comparison matrix (only when --cross-harness was used).
# Prints the grouped-by-family table to stdout and saves it alongside the
# baseline so the result is always reproducible without re-running the eval.
if [[ -n "$CROSS_HARNESS" ]]; then
    echo
    echo "=== Cross-Harness Comparison Matrix ==="
    MATRIX_VERSION="v$VERSION_NORMALIZED-harness"
    HARNESS_MATRIX_OUT="$RESULTS_DIR/cross_harness_matrix.md"
    ailang eval-matrix "$RESULTS_DIR" "$MATRIX_VERSION" --group-by=model-family \
        | tee "$HARNESS_MATRIX_OUT"
    echo
    echo "✓ Cross-harness matrix saved: $HARNESS_MATRIX_OUT"
fi
