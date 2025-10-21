#!/bin/bash
# Categorize examples into runnable/, snippets/, and tests/

set -e

cd "$(dirname "$0")/../examples"

echo "Categorizing examples..."

# RUNNABLE: Full programs with proper module structure and export func main()
RUNNABLE=(
  "adt_option.ail"
  "adt_simple.ail"
  "block_recursion.ail"
  "effects_basic.ail"
  "effects_pure.ail"
  "guards_basic.ail"
  "json_basic_decode.ail"
  "letrec_recursion.ail"
  "micro_block_if.ail"
  "micro_block_seq.ail"
  "micro_io_echo.ail"
  "micro_option_map.ail"
  "micro_record_person.ail"
  "patterns.ail"
  "recursion_factorial.ail"
  "recursion_fibonacci.ail"
  "recursion_mutual.ail"
  "recursion_quicksort.ail"
  "simple.ail"
  "demos/adt_pipeline.ail"
  "demos/hello_io.ail"
)

# SNIPPETS: Documentation examples (no module structure, just expressions)
SNIPPETS=(
  "arithmetic.ail"
  "func_expressions.ail"
  "hello.ail"
  "lambda_expressions.ail"
  "list_patterns.ail"
  "numeric_conversion.ail"
  "records.ail"
  "type_classes_working_reference.ail"
  "typeclasses.ail"
  "showcase/01_type_inference.ail"
  "showcase/02_lambdas.ail"
  "showcase/03_lists.ail"
  "showcase/03_type_classes.ail"
  "showcase/04_closures.ail"
  "v3_3/math/gcd.ail"
  "v3_3/imports.ail"
  "v3_3/imports_basic.ail"
)

# TESTS: Test cases (may intentionally fail or test edge cases)
TESTS=(
  "bug_float_comparison.ail"
  "bug_modulo_operator.ail"
  "recursion_error.ail"
  "test_effect_annotation.ail"
  "test_effect_capability.ail"
  "test_effect_fs.ail"
  "test_effect_io.ail"
  "test_effect_io_simple.ail"
  "test_exhaustive_bool_complete.ail"
  "test_exhaustive_bool_incomplete.ail"
  "test_exhaustive_wildcard.ail"
  "test_fizzbuzz.ail"
  "test_float_comparison.ail"
  "test_float_eq_works.ail"
  "test_float_modulo.ail"
  "test_guard_bool.ail"
  "test_guard_debug.ail"
  "test_guard_false.ail"
  "test_import_ctor.ail"
  "test_import_func.ail"
  "test_integral.ail"
  "test_invocation.ail"
  "test_io_builtins.ail"
  "test_m_r7_comprehensive.ail"
  "test_module_minimal.ail"
  "test_modulo_works.ail"
  "test_net_file_protocol.ail"
  "test_net_localhost.ail"
  "test_net_security.ail"
  "test_no_import.ail"
  "test_record_subsumption.ail"
  "test_single_guard.ail"
  "test_use_constructor.ail"
  "test_with_import.ail"
  "micro_clock_measure.ail"
  "micro_net_fetch.ail"
  "demos/effects_pure.ail"
)

# AI/Experimental examples (use unimplemented features)
# These stay in experimental/ directory (already there)
EXPERIMENTAL=(
  "ai_call.ail"
  "claude_haiku_call.ail"
  "demo_ai_api.ail"
  "demo_openai_api.ail"
)

# DEMO files (documentation, skip verify)
DEMO=(
  "block_demo.ail"
  "option_demo.ail"
  "stdlib_demo.ail"
  "stdlib_demo_simple.ail"
)

echo "Moving files to appropriate directories..."

# Move runnable examples
for file in "${RUNNABLE[@]}"; do
  if [ -f "$file" ]; then
    mkdir -p "runnable/$(dirname "$file")"
    git mv "$file" "runnable/$file" 2>/dev/null || mv "$file" "runnable/$file"
    echo "  ✓ runnable/$file"
  fi
done

# Move snippets
for file in "${SNIPPETS[@]}"; do
  if [ -f "$file" ]; then
    mkdir -p "snippets/$(dirname "$file")"
    git mv "$file" "snippets/$file" 2>/dev/null || mv "$file" "snippets/$file"
    echo "  ✓ snippets/$file"
  fi
done

# Move tests
for file in "${TESTS[@]}"; do
  if [ -f "$file" ]; then
    mkdir -p "tests/$(dirname "$file")"
    git mv "$file" "tests/$file" 2>/dev/null || mv "$file" "tests/$file"
    echo "  ✓ tests/$file"
  fi
done

# Move experimental AI examples
for file in "${EXPERIMENTAL[@]}"; do
  if [ -f "$file" ]; then
    git mv "$file" "experimental/$file" 2>/dev/null || mv "$file" "experimental/$file"
    echo "  ✓ experimental/$file"
  fi
done

# Move demo files to snippets
for file in "${DEMO[@]}"; do
  if [ -f "$file" ]; then
    git mv "$file" "snippets/$file" 2>/dev/null || mv "$file" "snippets/$file"
    echo "  ✓ snippets/$file (demo)"
  fi
done

echo ""
echo "Summary:"
echo "  Runnable examples: ${#RUNNABLE[@]}"
echo "  Documentation snippets: ${#SNIPPETS[@]}"
echo "  Test cases: ${#TESTS[@]}"
echo "  Experimental: ${#EXPERIMENTAL[@]}"
echo "  Demos: ${#DEMO[@]}"
echo ""
echo "Total organized: $((${#RUNNABLE[@]} + ${#SNIPPETS[@]} + ${#TESTS[@]} + ${#EXPERIMENTAL[@]} + ${#DEMO[@]}))"
