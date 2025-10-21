#!/bin/bash
# Promote high-quality tests to runnable examples

set -e
cd "$(dirname "$0")/.."

PROMOTE=(
  "test_fizzbuzz.ail"
  "test_effect_io_simple.ail"
  "test_guard_bool.ail"
  "test_import_func.ail"
  "test_module_minimal.ail"
  "test_io_builtins.ail"
  "test_m_r7_comprehensive.ail"
  "micro_clock_measure.ail"
  "micro_net_fetch.ail"
  "bug_float_comparison.ail"
)

echo "Promoting high-quality tests to runnable examples..."

for f in "${PROMOTE[@]}"; do
  if [ -f "examples/tests/$f" ]; then
    echo "  Promoting: $f"
    git mv "examples/tests/$f" "examples/runnable/$f" 2>/dev/null || mv "examples/tests/$f" "examples/runnable/$f"

    # Fix module path: examples/test_name -> examples/runnable/test_name
    sed -i.bak "s|module examples/\([a-z_]*\)$|module examples/runnable/\1|" "examples/runnable/$f"
    rm -f "examples/runnable/$f.bak"
  else
    echo "  Warning: $f not found in tests/"
  fi
done

echo ""
echo "Promoted ${#PROMOTE[@]} examples to runnable/"
echo "Running verification..."
