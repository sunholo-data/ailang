#!/bin/bash
# find_language_gaps.sh - Analyze agent transcripts to identify AILANG language gaps
#
# Usage: ./find_language_gaps.sh <baseline_dir>
# Example: ./find_language_gaps.sh eval_results/baselines/v0.6.2
#          ./find_language_gaps.sh eval_results/baselines/v0.6.2/agent  # Also works

set -e

INPUT_DIR="${1:-eval_results/baselines/v0.6.2}"

# Detect if user passed the agent directory directly or the baseline directory
if [[ "$INPUT_DIR" == */agent ]]; then
    # User passed agent directory directly
    AGENT_DIR="$INPUT_DIR"
    BASELINE_DIR="${INPUT_DIR%/agent}"
elif [ -d "$INPUT_DIR/agent" ]; then
    # User passed baseline directory containing agent/
    AGENT_DIR="$INPUT_DIR/agent"
    BASELINE_DIR="$INPUT_DIR"
else
    echo "Error: No agent results found"
    echo "Tried: $INPUT_DIR/agent"
    echo ""
    echo "Usage: $0 <baseline_dir>"
    echo "Example: $0 eval_results/baselines/v0.7.0"
    echo "         $0 eval_results/baselines/v0.7.0/agent"
    exit 1
fi

# Verify agent directory has JSON files
if ! ls "$AGENT_DIR"/*_ailang_*.json &>/dev/null; then
    echo "Error: No AILANG agent results found in $AGENT_DIR"
    echo "Expected files matching: *_ailang_*.json"
    exit 1
fi

echo "========================================"
echo "AILANG Language Gap Analysis"
echo "Baseline: $BASELINE_DIR"
echo "Agent dir: $AGENT_DIR"
echo "========================================"
echo ""

# Count failures
TOTAL=$(ls "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | wc -l | tr -d ' ')
FAILURES=$(cat "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | jq -s 'map(select(.stdout_ok == false)) | length')

echo "Total AILANG agent runs: $TOTAL"
echo "Failures: $FAILURES"
echo ""

echo "========================================"
echo "1. FUNCTION SEARCH PATTERNS"
echo "   (Agent looking for functions it can't find)"
echo "========================================"
echo ""

cat "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | \
    jq -r 'select(.stdout_ok == false) | "\(.id): \(.stderr)"' | \
    grep -i "let me check\|what function\|is available\|what.*available\|Let me.*builtins" | \
    head -15 || echo "No search patterns found"

echo ""
echo "========================================"
echo "2. UNDEFINED VARIABLE ERRORS"
echo "   (Functions agents tried to use that don't exist)"
echo "========================================"
echo ""

cat "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | \
    jq -r 'select(.stdout_ok == false) | .stderr // ""' | \
    grep -oE "undefined variable: [a-zA-Z_]+" | \
    sort | uniq -c | sort -rn | head -15 || echo "No undefined variables found"

echo ""
echo "========================================"
echo "3. TYPE CONFUSION PATTERNS"
echo "   (Repeated type errors suggest unclear type system)"
echo "========================================"
echo ""

cat "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | \
    jq -r 'select(.stdout_ok == false) | .stderr // ""' | \
    grep -i "type error\|type unification\|cannot unify" | \
    head -10 || echo "No type confusion patterns found"

echo ""
echo "========================================"
echo "4. BENCHMARKS WITH HIGHEST TURNS (stuck loops)"
echo "========================================"
echo ""

cat "$AGENT_DIR"/*_ailang_*.json 2>/dev/null | \
    jq -r 'select(.stdout_ok == false) | "\(.agent_turns // 0)\t\(.id)"' | \
    sort -rn | head -10

echo ""
echo "========================================"
echo "5. HALLUCINATED FUNCTION MAPPING"
echo "   Check if builtins exist for these:"
echo "========================================"
echo ""

# Common hallucinations based on analysis
echo "Agent looks for    | Check builtin           | Check stdlib"
echo "-------------------|-------------------------|------------------"
echo "floatToInt         | ailang builtins show _float_to_int | grep floatToInt std/*.ail"
echo "intToFloat         | ailang builtins show _int_to_float | grep intToFloat std/*.ail"
echo "stringSlice        | ailang builtins show _str_slice    | grep substring std/string.ail"
echo "contains           | ailang builtins show _str_find     | grep contains std/string.ail"
echo ""

echo "========================================"
echo "6. RECOMMENDED ACTIONS"
echo "========================================"
echo ""
echo "Run these commands to check if fixes are needed:"
echo ""
echo "  # Check if numeric conversion wrappers exist"
echo "  grep -E 'floatToInt|intToFloat' std/math.ail"
echo ""
echo "  # Check what's documented in current prompt"
echo "  grep -i 'string\|math\|convert' prompts/\$(ailang prompt --version 2>/dev/null | head -1).md | head -20"
echo ""
echo "  # List all available builtins"
echo "  ailang builtins list | grep -E 'float|int|str'"
echo ""
