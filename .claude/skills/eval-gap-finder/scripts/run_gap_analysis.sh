#!/usr/bin/env bash
# run_gap_analysis.sh - Full gap analysis workflow
#
# Usage: ./run_gap_analysis.sh [eval_dir]
# Example: ./run_gap_analysis.sh eval_results/baselines/v0.6.5
#
# If no eval_dir provided, runs fresh evals with dev models first.
#
# Output: Complete gap analysis with actionable recommendations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EVAL_DIR="${1:-}"

echo "=============================================="
echo "    AILANG Eval Gap Analysis"
echo "=============================================="
echo ""

# Step 1: Get or run eval results
if [[ -z "$EVAL_DIR" ]]; then
    echo "No eval directory provided. Running fresh evals..."
    echo ""

    # Create output directory
    EVAL_DIR="eval_results/gap-analysis-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$EVAL_DIR"

    echo "Running evals with dev models (gemini-3-flash, claude-haiku-4-5)..."
    ailang eval-suite --models gemini-3-flash,claude-haiku-4-5 --output "$EVAL_DIR"
    echo ""
fi

if [[ ! -d "$EVAL_DIR" ]]; then
    echo "Error: Directory not found: $EVAL_DIR"
    exit 1
fi

echo "Analyzing: $EVAL_DIR"
echo ""

# Step 2: Generate summary if needed
SUMMARY_FILE="$EVAL_DIR/summary.jsonl"
if [[ ! -f "$SUMMARY_FILE" ]]; then
    echo "Generating summary.jsonl..."
    ailang eval-summary "$EVAL_DIR"
    echo ""
fi

# Step 3: Show overall stats
echo "=============================================="
echo "    Step 1: Overall Statistics"
echo "=============================================="
echo ""

AILANG_TOTAL=$(jq -rs 'map(select(.lang == "ailang")) | length' "$SUMMARY_FILE")
AILANG_PASS=$(jq -rs 'map(select(.lang == "ailang" and .stdout_ok == true)) | length' "$SUMMARY_FILE")
PYTHON_TOTAL=$(jq -rs 'map(select(.lang == "python")) | length' "$SUMMARY_FILE")
PYTHON_PASS=$(jq -rs 'map(select(.lang == "python" and .stdout_ok == true)) | length' "$SUMMARY_FILE")

if [[ "$AILANG_TOTAL" -gt 0 ]]; then
    AILANG_RATE=$(echo "scale=1; $AILANG_PASS * 100 / $AILANG_TOTAL" | bc)
else
    AILANG_RATE="0"
fi

if [[ "$PYTHON_TOTAL" -gt 0 ]]; then
    PYTHON_RATE=$(echo "scale=1; $PYTHON_PASS * 100 / $PYTHON_TOTAL" | bc)
else
    PYTHON_RATE="0"
fi

echo "AILANG: $AILANG_PASS / $AILANG_TOTAL = ${AILANG_RATE}%"
echo "Python: $PYTHON_PASS / $PYTHON_TOTAL = ${PYTHON_RATE}%"
echo ""

# Step 4: Identify Python-only gaps
echo "=============================================="
echo "    Step 2: Python-Only Gaps"
echo "=============================================="
echo ""

"$SCRIPT_DIR/identify_python_only.sh" "$EVAL_DIR"
echo ""

# Step 5: Categorize errors
echo "=============================================="
echo "    Step 3: Error Categories"
echo "=============================================="
echo ""

"$SCRIPT_DIR/categorize_errors.sh" "$EVAL_DIR"
echo ""

# Step 6: Summary and recommendations
echo "=============================================="
echo "    Step 4: Action Items"
echo "=============================================="
echo ""

echo "Based on the analysis, here are recommended actions:"
echo ""

# Count gaps by category
WRONG_LANG=$(jq -rs 'map(select(.lang == "ailang" and .error_category == "WRONG_LANG")) | length' "$SUMMARY_FILE")
PARSE_ERR=$(jq -rs 'map(select(.lang == "ailang" and (.error_category == "PAR_001" or .error_category == "compile_error"))) | length' "$SUMMARY_FILE")
TYPE_ERR=$(jq -rs 'map(select(.lang == "ailang" and .error_category == "type_error")) | length' "$SUMMARY_FILE")
LOGIC_ERR=$(jq -rs 'map(select(.lang == "ailang" and .error_category == "logic_error")) | length' "$SUMMARY_FILE")

if [[ "$WRONG_LANG" -gt 0 ]]; then
    echo "1. WRONG_LANG ($WRONG_LANG errors)"
    echo "   Action: Strengthen 'NOT Python' emphasis in prompt"
    echo "   Add more contrast examples (Python vs AILANG)"
    echo ""
fi

if [[ "$PARSE_ERR" -gt 0 ]]; then
    echo "2. Parse/Syntax Errors ($PARSE_ERR errors)"
    echo "   Action: Add more syntax examples to prompt"
    echo "   Check if patterns are documented"
    echo ""
fi

if [[ "$TYPE_ERR" -gt 0 ]]; then
    echo "3. Type Errors ($TYPE_ERR errors)"
    echo "   Action: Check for language gaps (polymorphic types, etc.)"
    echo "   Add type annotation examples to prompt"
    echo "   Create design docs for language limitations"
    echo ""
fi

if [[ "$LOGIC_ERR" -gt 0 ]]; then
    echo "4. Logic Errors ($LOGIC_ERR errors)"
    echo "   Action: Add algorithm examples to prompt"
    echo "   May need benchmark-specific fixes"
    echo ""
fi

echo "=============================================="
echo "    Next Steps"
echo "=============================================="
echo ""
echo "1. For each Python-only gap, examine the AILANG code:"
echo "   jq -r 'select(.benchmark == \"BENCHMARK\" and .lang == \"ailang\")' $SUMMARY_FILE"
echo ""
echo "2. Test any proposed examples BEFORE adding to prompt:"
echo "   $SCRIPT_DIR/test_example.sh /tmp/test.ail"
echo ""
echo "3. If example fails, check if it's a language limitation:"
echo "   - Create design doc: design_docs/planned/vX_Y_Z/m-<feature>.md"
echo "   - Add workaround to prompt with note"
echo ""
echo "4. After prompt updates, re-run evals to verify improvement:"
echo "   ailang eval-suite --models gemini-3-flash,claude-haiku-4-5"
echo ""
echo "Done! Gap analysis complete."
