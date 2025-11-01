#!/usr/bin/env bash
# Run evaluation baseline for a release version

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version> [--full]" >&2
    echo "Example: $0 0.3.14 --full" >&2
    echo "" >&2
    echo "Options:" >&2
    echo "  --full    Run with all production models (default: dev models only)" >&2
    exit 1
fi

VERSION="$1"
FULL_FLAG=""

if [[ $# -gt 1 ]] && [[ "$2" == "--full" ]]; then
    FULL_FLAG="FULL=true"
fi

echo "Running eval baseline for $VERSION..."
if [[ -n "$FULL_FLAG" ]]; then
    echo "Mode: FULL (all 6 production models)"
    echo "Expected cost: ~\$0.50-1.00"
    echo "Expected time: ~15-20 minutes"
else
    echo "Mode: DEV (3 cheap models only)"
    echo "Expected cost: ~\$0.10-0.20"
    echo "Expected time: ~5-10 minutes"
fi
echo

# Step 1: Run standard eval baseline (pass version exactly as given)
echo "=== Step 1/2: Standard Eval (0-shot + repair) ==="
if [[ -n "$FULL_FLAG" ]]; then
    make eval-baseline EVAL_VERSION="$VERSION" FULL=true
else
    make eval-baseline EVAL_VERSION="$VERSION"
fi

# Step 2: Run agent eval on curated benchmarks
echo
echo "=== Step 2/2: Agent Eval (multi-turn) ==="
echo "Running agent eval on curated benchmarks..."

# Full agent benchmark suite (19 benchmarks) - v0.4.0+
# See: BENCHMARK_AUDIT_ANALYSIS.md for detailed rationale
#
# Tier 1 - Smoke Tests (8): Fast sanity checks, 95-100% expected success
#   fizzbuzz, recursion_factorial, recursion_fibonacci, simple_print
#   records_person, list_operations, string_manipulation, nested_records
#
# Tier 2 - Differentiators (11): Agent should outperform 0-shot (60-80% vs 30-50%)
#   higher_order_functions, pattern_matching_complex, record_update
#   effect_composition, effect_tracking_io_fs, effect_pure_separation
#   exhaustive_pattern_matching, type_safe_record_access
#   explicit_state_threading, deterministic_list_transform, referential_transparency
AGENT_BENCHMARKS="fizzbuzz,recursion_factorial,recursion_fibonacci,simple_print,records_person,list_operations,string_manipulation,nested_records,higher_order_functions,pattern_matching_complex,record_update,effect_composition,effect_tracking_io_fs,effect_pure_separation,exhaustive_pattern_matching,type_safe_record_access,explicit_state_threading,deterministic_list_transform,referential_transparency"

echo "Benchmarks: $AGENT_BENCHMARKS"
echo

RESULTS_DIR="eval_results/baselines/$VERSION"

# Agent eval uses haiku/sonnet models based on --full flag
if [[ -n "$FULL_FLAG" ]]; then
    # Full mode: run both haiku and sonnet (via --full, auto-filtered from extended_suite)
    echo "Mode: FULL (haiku + sonnet via --full flag)"
    ailang eval-suite --agent --full \
        --benchmarks "$AGENT_BENCHMARKS" \
        --langs ailang,python \
        --agent-parallel 2 \
        --output "$RESULTS_DIR"
else
    # Dev mode: haiku only (default dev_models, auto-filtered to Claude only)
    echo "Mode: DEV (haiku only via dev_models)"
    ailang eval-suite --agent \
        --benchmarks "$AGENT_BENCHMARKS" \
        --langs ailang,python \
        --agent-parallel 2 \
        --output "$RESULTS_DIR"
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
