#!/usr/bin/env bash
# Run evaluation baseline for a release version

set -euo pipefail

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
            percent=$((current * 100 / expected_count))
            echo "[$phase] Progress: $current/$expected_count files ($percent%)"
        fi
    done
}

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

# Normalize version: ensure directory always has "v" prefix
# Strip any existing "v" prefix first, then add it
VERSION_NORMALIZED="${VERSION#v}"
RESULTS_DIR="eval_results/baselines/v$VERSION_NORMALIZED"

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

# Step 1: Run standard eval baseline (pass normalized version WITH v prefix)
echo "=== Step 1/2: Standard Eval (0-shot + repair) ==="
if [[ -n "$FULL_FLAG" ]]; then
    monitor_progress "$RESULTS_DIR" 480 "Standard" &
    MONITOR_PID=$!
    make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED" FULL=true
    kill $MONITOR_PID 2>/dev/null || true
else
    monitor_progress "$RESULTS_DIR" 246 "Standard" &
    MONITOR_PID=$!
    make eval-baseline EVAL_VERSION="v$VERSION_NORMALIZED"
    kill $MONITOR_PID 2>/dev/null || true
fi

# Step 2: Run agent eval on curated benchmarks
echo
echo "=== Step 2/2: Agent Eval (multi-turn) ==="
echo "Running agent eval on curated benchmarks..."

# Full agent benchmark suite (v0.4.8+) - 46 benchmarks
# Trimmed from 56: removed trivial (4) and easy (6, kept fizzbuzz)
# See: benchmark-manager skill for details
#
# Categories:
#   - 1 easy (fizzbuzz): validation benchmark
#   - ~19 medium: core language features
#   - ~17 hard: complex algorithms and effects
#   - 6 stretch goals: symbolic_diff, mini_interpreter, lambda_calc, graph_bfs, type_unify, red_black_tree
#
# Agent mode requires explicit --benchmarks list (safety feature)
AGENT_BENCHMARKS="adt_option,api_call_json,balanced_parens,binary_tree_sum,canonical_normalization,cli_args,config_file_parser,csv_to_json_converter,effect_composition,effect_pure_separation,effect_tracking_io_fs,error_handling,exhaustive_pattern_matching,explicit_state_threading,expression_evaluator,fizzbuzz,float_eq,fold_reduce,gcd_lcm,graph_bfs,higher_order_functions,immutable_data_structures,inline_tests,json_encode,json_parse,json_transform,lambda_calc,list_comprehension,log_file_analyzer,merge_sort,mini_interpreter,nested_records,no_runtime_crashes_option,numeric_modulo,pattern_matching_complex,pipeline,record_update,records_person,recursion_fibonacci,red_black_tree,run_length_encode,state_machine_traffic_light,symbolic_diff,tree_transformation_pipeline,type_safe_record_access,type_unify"

echo "Benchmarks: all 46 benchmarks"
echo

# Pre-flight validation and configuration summary
echo "=== Pre-Flight Check ==="
echo "Version: $VERSION"
echo "Mode: ${FULL_FLAG:-DEV}"
echo "Benchmarks: 46 (1 easy, ~19 medium, ~17 hard, 6 stretch)"
echo "Agent parallelism: 2"
echo "Agent timeout: default (no override)"
echo "Prompt version: latest (auto-selected)"
echo
echo "Expected results:"
if [[ -n "$FULL_FLAG" ]]; then
    echo "  Standard eval: ~552 files (46 benchmarks × 6 models × 2 langs)"
    echo "  Agent eval: ~184 files (46 benchmarks × 2 models × 2 langs)"
    echo "  Total: ~736 files"
else
    echo "  Standard eval: ~276 files (46 benchmarks × 3 models × 2 langs)"
    echo "  Agent eval: ~92 files (46 benchmarks × 1 model × 2 langs)"
    echo "  Total: ~368 files"
fi
echo
echo "Starting in 3 seconds... (Ctrl-C to abort)"
sleep 3
echo

# Agent eval uses haiku/sonnet models based on --full flag
# --benchmarks is required for agent mode (safety feature)
if [[ -n "$FULL_FLAG" ]]; then
    # Full mode: run both haiku and sonnet (via --full, auto-filtered from extended_suite)
    echo "Mode: FULL (haiku + sonnet via --full flag)"
    monitor_progress "$RESULTS_DIR" 184 "Agent" &
    MONITOR_PID=$!
    ailang eval-suite --agent --full \
        --benchmarks "$AGENT_BENCHMARKS" \
        --langs ailang,python \
        --agent-parallel 2 \
        --output "$RESULTS_DIR"
    kill $MONITOR_PID 2>/dev/null || true
else
    # Dev mode: haiku only (default dev_models, auto-filtered to Claude only)
    echo "Mode: DEV (haiku only via dev_models)"
    monitor_progress "$RESULTS_DIR" 92 "Agent" &
    MONITOR_PID=$!
    ailang eval-suite --agent \
        --benchmarks "$AGENT_BENCHMARKS" \
        --langs ailang,python \
        --agent-parallel 2 \
        --output "$RESULTS_DIR"
    kill $MONITOR_PID 2>/dev/null || true
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

# Check file counts (46 benchmarks × models × 2 langs)
if [[ -n "$FULL_FLAG" ]]; then
    EXPECTED_STANDARD=552   # 46 × 6 × 2
    EXPECTED_AGENT=184      # 46 × 2 × 2
    EXPECTED_TOTAL=736
else
    EXPECTED_STANDARD=276   # 46 × 3 × 2
    EXPECTED_AGENT=92       # 46 × 1 × 2
    EXPECTED_TOTAL=368
fi

# Allow 5% tolerance for filtering/failures
MIN_STANDARD=$((EXPECTED_STANDARD * 95 / 100))
MIN_AGENT=$((EXPECTED_AGENT * 85 / 100))  # Agent eval can have more variance
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
