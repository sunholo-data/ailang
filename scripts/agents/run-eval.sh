#!/bin/bash
# Eval Runner Script Agent
# Runs ailang eval-suite with parameters from message payload
#
# Environment variables (from JSON payload):
#   BENCHMARKS - comma-separated benchmark names (default: all)
#   MODELS - comma-separated model names (default: dev models)
#   LANGS - languages to test (default: ailang)
#   OUTPUT - output directory (default: eval_results/agent-runs)
#   AGENT_MODE - if "true", use agent-based evaluation

set -e

# Defaults
BENCHMARKS="${BENCHMARKS:-}"
MODELS="${MODELS:-claude-haiku-4-5}"
LANGS="${LANGS:-ailang}"
OUTPUT="${OUTPUT:-eval_results/agent-runs/$(date +%Y%m%d_%H%M%S)}"
AGENT_MODE="${AGENT_MODE:-false}"

echo "EVAL_RUNNER: Starting eval suite"
echo "  Benchmarks: ${BENCHMARKS:-all}"
echo "  Models: $MODELS"
echo "  Languages: $LANGS"
echo "  Output: $OUTPUT"
echo "  Agent mode: $AGENT_MODE"
echo ""

# Build command
CMD="ailang eval-suite"
CMD="$CMD -models $MODELS"
CMD="$CMD -langs $LANGS"
CMD="$CMD -output $OUTPUT"

if [ -n "$BENCHMARKS" ]; then
    CMD="$CMD -benchmarks $BENCHMARKS"
fi

if [ "$AGENT_MODE" = "true" ]; then
    CMD="$CMD -agent -agent-timeout 120"
fi

echo "Running: $CMD"
echo ""

# Run eval suite
$CMD

# Parse results
TOTAL=$(find "$OUTPUT" -name "*.json" | wc -l | tr -d ' ')
PASSED=$(find "$OUTPUT" -name "*.json" -exec grep -l '"stdout_ok": true' {} \; 2>/dev/null | wc -l | tr -d ' ')

if [ "$TOTAL" -gt 0 ]; then
    PASS_RATE=$(echo "scale=1; $PASSED * 100 / $TOTAL" | bc)
else
    PASS_RATE="0"
fi

echo ""
echo "EVAL_RESULT: Complete"
echo "TOTAL_RUNS: $TOTAL"
echo "PASSED: $PASSED"
echo "PASS_RATE: ${PASS_RATE}%"
echo "OUTPUT_DIR: $OUTPUT"
