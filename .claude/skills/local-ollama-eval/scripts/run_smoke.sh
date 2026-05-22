#!/usr/bin/env bash
# run_smoke.sh — Run the canonical smoke tier (or a subset) on a local Ollama model.
#
# Usage:
#   run_smoke.sh <model>                # full smoke tier (17 benchmarks)
#   run_smoke.sh <model> <benchmark>    # single benchmark only
#   run_smoke.sh <model> bench1,bench2  # explicit list
#
# Examples:
#   run_smoke.sh opencode-gemma4-26b
#   run_smoke.sh opencode-gemma4-26b fizzbuzz
#   run_smoke.sh opencode-gemma4-26b fizzbuzz,adt_option

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <model> [benchmark|benchmark-list]"
  echo "Example: $0 opencode-gemma4-26b"
  echo "         $0 opencode-gemma4-26b fizzbuzz"
  exit 1
fi

MODEL="$1"
shift

# Default smoke set if no explicit benchmarks given.
DEFAULT_SMOKE="fizzbuzz,adt_option,balanced_parens,binary_tree_sum,canonical_convergence,canonical_normalization,dense_operator_program,explicit_state_threading,gcd_lcm,immutable_data_structures,inline_tests,nested_records,numeric_modulo,record_update,records_book,recursion_fibonacci,type_safe_record_access"

if [[ $# -ge 1 && -n "$1" ]]; then
  BENCHMARKS="$1"
else
  BENCHMARKS="$DEFAULT_SMOKE"
fi

# Sanity: warn if OTLP env not set (live monitoring won't work).
if [[ -z "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" ]]; then
  echo "⚠ OTEL_EXPORTER_OTLP_ENDPOINT not set — spans won't land in observatory.db."
  echo "  For live monitoring with 'ailang chains live':"
  echo "    export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957"
  echo "  Continuing without observability..."
fi

# Sanity: ensure the AILANG server is running if env var is set.
if [[ -n "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" ]]; then
  if ! curl -s -o /dev/null -w "%{http_code}" "${OTEL_EXPORTER_OTLP_ENDPOINT}/health" 2>/dev/null | grep -q 200; then
    echo "⚠ OTEL endpoint ${OTEL_EXPORTER_OTLP_ENDPOINT} not responding. Run: make services-start"
    echo "  Continuing — spans will be lost but eval will still run."
  fi
fi

OUTPUT="eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_${MODEL}_smoke"
mkdir -p "$(dirname "$OUTPUT")"

echo "──────────────────────────────────────────────────────────────"
echo " Local Ollama Smoke Tier"
echo "──────────────────────────────────────────────────────────────"
echo " Model:     ${MODEL}"
echo " Bench(s):  $(echo "$BENCHMARKS" | tr , ' ' | wc -w | tr -d ' ') benchmark(s)"
echo " Parallel:  2  (recommended for gemma4:26b)"
echo " Timeout:   2400s per benchmark"
echo " Output:    ${OUTPUT}"
echo " OTLP:      ${OTEL_EXPORTER_OTLP_ENDPOINT:-(unset, no live spans)}"
echo "──────────────────────────────────────────────────────────────"
echo ""

time make eval-smoke MODELS="${MODEL}" EXTRA="-agent -langs ailang -benchmarks ${BENCHMARKS} -output ${OUTPUT} -parallel 2 -agent-timeout 2400"

echo ""
echo "──────────────────────────────────────────────────────────────"
echo " Results: ${OUTPUT}"
echo ""
echo " Tabulate:"
echo "   for f in ${OUTPUT}/agent/*.json; do"
echo "     jq -r '\"\\(.benchmark_id)\\t\\(if .compile_ok and .runtime_ok and .stdout_ok then \"PASS\" else \"FAIL\" end)\\tdur=\\((.duration_ms/1000)|floor)s\\tcat=\\(.error_category // \"—\")\"' \"\$f\""
echo "   done | column -ts \$'\\t'"
echo ""
echo " Or use:"
echo "   ailang eval-summary ${OUTPUT}"
echo "──────────────────────────────────────────────────────────────"
