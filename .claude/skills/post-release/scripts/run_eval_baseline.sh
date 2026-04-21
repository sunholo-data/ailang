#!/usr/bin/env bash
# Run evaluation baseline for a release version.
#
# v0.14.0+: benchmark selection is driven by the tier system, not a hardcoded list.
# Default release scope is `core,stretch`. Pass --tier to override.

set -euo pipefail

# Default tier scope for release baselines:
#   - `core`    = headline metric (22 benchmarks as of v0.14.0)
#   - `stretch` = harder benchmarks we expect mixed results on (11 benchmarks)
# Vision tier (research-grade) is excluded by default; smoke tier is a subset of
# CI sanity checks and also excluded by default to keep releases focused.
DEFAULT_TIER="core,stretch"

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

# Arg parsing: <version> [--full] [--tier <spec>]
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version> [--full] [--tier <spec>]" >&2
    echo "Example: $0 0.3.14 --full" >&2
    echo "Example: $0 0.13.0 --full --tier core,stretch" >&2
    echo "" >&2
    echo "Options:" >&2
    echo "  --full              Use all production models (default: dev models)" >&2
    echo "  --tier <spec>       Comma-separated tiers: smoke,core,stretch,vision" >&2
    echo "                      (default: $DEFAULT_TIER)" >&2
    exit 1
fi

VERSION="$1"
shift
FULL_FLAG=""
TIER_FLAG="$DEFAULT_TIER"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --full)
            FULL_FLAG="FULL=true"
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
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

# Normalize version: ensure directory always has "v" prefix
VERSION_NORMALIZED="${VERSION#v}"
RESULTS_DIR="eval_results/baselines/v$VERSION_NORMALIZED"

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
    echo "Mode: FULL (extended model suite)"
    echo "Expected cost: ~\$0.50-1.00"
    echo "Expected time: ~15-20 minutes"
else
    echo "Mode: DEV (3 dev models)"
    echo "Expected cost: ~\$0.10-0.20"
    echo "Expected time: ~5-10 minutes"
fi
echo

# Step 1: Run standard eval baseline
echo "=== Step 1/2: Standard Eval (0-shot + repair) ==="
# Expected file count for standard eval: benchmarks × models × langs (2)
if [[ -n "$FULL_FLAG" ]]; then
    EXPECTED_STANDARD=$((BENCHMARK_COUNT * 6 * 2))   # ~6 extended models
else
    EXPECTED_STANDARD=$((BENCHMARK_COUNT * 3 * 2))   # 3 dev models
fi

monitor_progress "$RESULTS_DIR" "$EXPECTED_STANDARD" "Standard" &
MONITOR_PID=$!
if [[ -n "$FULL_FLAG" ]]; then
    TIER="$TIER_FLAG" make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED" FULL=true
else
    TIER="$TIER_FLAG" make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED"
fi
kill $MONITOR_PID 2>/dev/null || true

# Step 2: Run agent eval on tier-selected benchmarks
echo
echo "=== Step 2/2: Agent Eval (multi-turn) ==="
echo "Running agent eval on tier=$TIER_FLAG benchmarks ($BENCHMARK_COUNT total)..."
echo

# Pre-flight summary
echo "=== Pre-Flight Check ==="
echo "Version: $VERSION"
echo "Tier: $TIER_FLAG ($BENCHMARK_COUNT benchmarks)"
echo "Mode: ${FULL_FLAG:-DEV}"
echo "Agent models: claude-sonnet-4-5 (Claude executor), gemini-3-flash (Gemini executor)"
echo "Agent parallelism: 2"
echo

# Agent expected file counts: benchmarks × 2 executors × langs
if [[ -n "$FULL_FLAG" ]]; then
    EXPECTED_AGENT=$((BENCHMARK_COUNT * 2 * 2))  # both langs
    AGENT_LANGS="ailang,python"
else
    EXPECTED_AGENT=$((BENCHMARK_COUNT * 2 * 1))  # AILANG only (faster)
    AGENT_LANGS="ailang"
fi

echo "Expected results:"
echo "  Standard eval: ~$EXPECTED_STANDARD files"
echo "  Agent eval: ~$EXPECTED_AGENT files"
echo "  Total: ~$((EXPECTED_STANDARD + EXPECTED_AGENT)) files"
echo
echo "Starting in 3 seconds... (Ctrl-C to abort)"
sleep 3
echo

AGENT_MODELS="claude-sonnet-4-5,gemini-3-flash"

# Agent mode requires explicit --benchmarks (CLI safety guardrail), so expand
# the tier spec to a vetted benchmark list.
AGENT_BENCHMARKS_CSV="$(resolve_benchmarks_in_tiers "$TIER_FLAG")"
if [[ -z "$AGENT_BENCHMARKS_CSV" ]]; then
    echo "ERROR: could not resolve --tier $TIER_FLAG to any benchmark IDs" >&2
    exit 1
fi

echo "Mode: ${FULL_FLAG:+FULL }(Claude Sonnet + Gemini 3 Flash, langs=$AGENT_LANGS)"
echo "Executors: claude, gemini"
echo "Agent benchmarks: $BENCHMARK_COUNT resolved from tier=$TIER_FLAG"
monitor_progress "$RESULTS_DIR" "$EXPECTED_AGENT" "Agent" &
MONITOR_PID=$!
ailang eval-suite --agent \
    --models "$AGENT_MODELS" \
    --benchmarks "$AGENT_BENCHMARKS_CSV" \
    --langs "$AGENT_LANGS" \
    --agent-parallel 2 \
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
