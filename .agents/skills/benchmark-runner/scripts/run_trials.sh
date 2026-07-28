#!/bin/bash
# Run benchmark trials for an AILANG prompt variant
# Usage: run_trials.sh <variant> <count> [start]
# Example: run_trials.sh ailang/spec 5 1

set -e

BENCH_DIR="/Users/mark/dev/sunholo/ai-coding-lang-bench"
VARIANT="${1:-ailang/spec}"
COUNT="${2:-5}"
START="${3:-1}"

if [ ! -d "$BENCH_DIR" ]; then
  echo "Error: Benchmark repo not found at $BENCH_DIR"
  echo "Clone it: gh repo clone sunholo-data/ai-coding-lang-bench $BENCH_DIR"
  exit 1
fi

# Regenerate AILANG-SYNTAX.md if ailang binary is newer
SYNTAX_FILE="$BENCH_DIR/AILANG-SYNTAX.md"
if command -v ailang &>/dev/null; then
  echo "Regenerating AILANG-SYNTAX.md..."
  ailang prompt > "$SYNTAX_FILE" 2>/dev/null
  echo "  $(wc -l < "$SYNTAX_FILE") lines"
fi

echo "Running $COUNT trials of $VARIANT (starting at $START)"
echo "Working dir: $BENCH_DIR"
echo ""

cd "$BENCH_DIR"
ruby benchmark.rb --lang "$VARIANT" --trials "$COUNT" --start "$START"
