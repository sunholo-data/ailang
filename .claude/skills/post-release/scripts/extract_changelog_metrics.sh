#!/usr/bin/env bash
# Extract comprehensive benchmark metrics for CHANGELOG
# Extracts: 0-shot, final, repair effectiveness, and agent eval metrics

set -euo pipefail

VERSION="${1:-}"
RESULTS_DIR="${2:-eval_results/baselines/$VERSION}"
JSON_FILE="docs/static/benchmarks/latest.json"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version> [results_dir]" >&2
    echo "Example: $0 0.4.1" >&2
    exit 1
fi

if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "Error: Results directory not found: $RESULTS_DIR" >&2
    exit 1
fi

if [[ ! -f "$RESULTS_DIR/summary.jsonl" ]]; then
    echo "Error: summary.jsonl not found. Run 'ailang eval-summary $RESULTS_DIR' first" >&2
    exit 1
fi

if [[ ! -f "$JSON_FILE" ]]; then
    echo "Error: Dashboard JSON not found: $JSON_FILE" >&2
    echo "Run update_dashboard.sh first" >&2
    exit 1
fi

SUMMARY="$RESULTS_DIR/summary.jsonl"

echo "Extracting metrics from $RESULTS_DIR..."
echo

# Extract standard eval metrics (AILANG only)
AILANG_TOTAL=$(jq -s 'map(select(.lang == "ailang")) | length' "$SUMMARY")
AILANG_0SHOT=$(jq -s 'map(select(.lang == "ailang" and .first_attempt_ok == true)) | length' "$SUMMARY")
AILANG_FINAL=$(jq -s 'map(select(.lang == "ailang" and .stdout_ok == true)) | length' "$SUMMARY")

# Calculate percentages
AILANG_0SHOT_PCT=$(echo "scale=1; $AILANG_0SHOT * 100 / $AILANG_TOTAL" | bc)
AILANG_FINAL_PCT=$(echo "scale=1; $AILANG_FINAL * 100 / $AILANG_TOTAL" | bc)
AILANG_REPAIR_GAIN=$(echo "scale=1; $AILANG_FINAL_PCT - $AILANG_0SHOT_PCT" | bc)

# Python metrics
PYTHON_TOTAL=$(jq -s 'map(select(.lang == "python")) | length' "$SUMMARY")
PYTHON_FINAL=$(jq -s 'map(select(.lang == "python" and .stdout_ok == true)) | length' "$SUMMARY")
PYTHON_FINAL_PCT=$(echo "scale=1; $PYTHON_FINAL * 100 / $PYTHON_TOTAL" | bc)

# Agent eval metrics
AILANG_AGENT_TOTAL=$(jq -s 'map(select(.lang == "ailang" and .eval_mode == "agent")) | length' "$SUMMARY")
AILANG_AGENT_SUCCESS=$(jq -s 'map(select(.lang == "ailang" and .eval_mode == "agent" and .stdout_ok == true)) | length' "$SUMMARY")
AILANG_AGENT_PCT=$(echo "scale=1; $AILANG_AGENT_SUCCESS * 100 / $AILANG_AGENT_TOTAL" | bc 2>/dev/null || echo "0.0")

PYTHON_AGENT_TOTAL=$(jq -s 'map(select(.lang == "python" and .eval_mode == "agent")) | length' "$SUMMARY")
PYTHON_AGENT_SUCCESS=$(jq -s 'map(select(.lang == "python" and .eval_mode == "agent" and .stdout_ok == true)) | length' "$SUMMARY")
PYTHON_AGENT_PCT=$(echo "scale=1; $PYTHON_AGENT_SUCCESS * 100 / $PYTHON_AGENT_TOTAL" | bc 2>/dev/null || echo "0.0")

# Overall success rate
OVERALL_RATE=$(jq -r '.aggregates.finalSuccess' "$JSON_FILE")
TOTAL_RUNS=$(jq -r '.totalRuns' "$JSON_FILE")
OVERALL_PCT=$(echo "$OVERALL_RATE * 100" | bc -l | xargs printf "%.1f")

# Find previous version in history
PREV_VERSION=$(jq -r '.history | sort_by(.timestamp) | reverse | .[1].version // "none"' "$JSON_FILE" 2>/dev/null || echo "none")

# Try to get previous version's summary.jsonl if it exists
PREV_RESULTS_DIR="eval_results/baselines/$PREV_VERSION"
PREV_SUMMARY="$PREV_RESULTS_DIR/summary.jsonl"

if [[ "$PREV_VERSION" != "none" ]] && [[ "$PREV_VERSION" != "null" ]] && [[ -f "$PREV_SUMMARY" ]]; then
    # Extract previous metrics
    PREV_AILANG_0SHOT=$(jq -s 'map(select(.lang == "ailang" and .first_attempt_ok == true)) | length' "$PREV_SUMMARY")
    PREV_AILANG_FINAL=$(jq -s 'map(select(.lang == "ailang" and .stdout_ok == true)) | length' "$PREV_SUMMARY")
    PREV_AILANG_TOTAL=$(jq -s 'map(select(.lang == "ailang")) | length' "$PREV_SUMMARY")

    PREV_AILANG_0SHOT_PCT=$(echo "scale=1; $PREV_AILANG_0SHOT * 100 / $PREV_AILANG_TOTAL" | bc)
    PREV_AILANG_FINAL_PCT=$(echo "scale=1; $PREV_AILANG_FINAL * 100 / $PREV_AILANG_TOTAL" | bc)
    PREV_AILANG_REPAIR_GAIN=$(echo "scale=1; $PREV_AILANG_FINAL_PCT - $PREV_AILANG_0SHOT_PCT" | bc)

    PREV_PYTHON_TOTAL=$(jq -s 'map(select(.lang == "python")) | length' "$PREV_SUMMARY")
    PREV_PYTHON_FINAL=$(jq -s 'map(select(.lang == "python" and .stdout_ok == true)) | length' "$PREV_SUMMARY")
    PREV_PYTHON_FINAL_PCT=$(echo "scale=1; $PREV_PYTHON_FINAL * 100 / $PREV_PYTHON_TOTAL" | bc)

    # Agent eval
    PREV_AILANG_AGENT_TOTAL=$(jq -s 'map(select(.lang == "ailang" and .eval_mode == "agent")) | length' "$PREV_SUMMARY")
    PREV_AILANG_AGENT_SUCCESS=$(jq -s 'map(select(.lang == "ailang" and .eval_mode == "agent" and .stdout_ok == true)) | length' "$PREV_SUMMARY")
    PREV_AILANG_AGENT_PCT=$(echo "scale=1; $PREV_AILANG_AGENT_SUCCESS * 100 / $PREV_AILANG_AGENT_TOTAL" | bc 2>/dev/null || echo "0.0")

    PREV_PYTHON_AGENT_TOTAL=$(jq -s 'map(select(.lang == "python" and .eval_mode == "agent")) | length' "$PREV_SUMMARY")
    PREV_PYTHON_AGENT_SUCCESS=$(jq -s 'map(select(.lang == "python" and .eval_mode == "agent" and .stdout_ok == true)) | length' "$PREV_SUMMARY")
    PREV_PYTHON_AGENT_PCT=$(echo "scale=1; $PREV_PYTHON_AGENT_SUCCESS * 100 / $PREV_PYTHON_AGENT_TOTAL" | bc 2>/dev/null || echo "0.0")

    # Calculate changes
    CHANGE_0SHOT=$(echo "scale=1; $AILANG_0SHOT_PCT - $PREV_AILANG_0SHOT_PCT" | bc)
    CHANGE_FINAL=$(echo "scale=1; $AILANG_FINAL_PCT - $PREV_AILANG_FINAL_PCT" | bc)
    CHANGE_REPAIR=$(echo "scale=1; $AILANG_REPAIR_GAIN - $PREV_AILANG_REPAIR_GAIN" | bc)
    CHANGE_PYTHON=$(echo "scale=1; $PYTHON_FINAL_PCT - $PREV_PYTHON_FINAL_PCT" | bc)
    CHANGE_AGENT_AILANG=$(echo "scale=1; $AILANG_AGENT_PCT - $PREV_AILANG_AGENT_PCT" | bc)
    CHANGE_AGENT_PYTHON=$(echo "scale=1; $PYTHON_AGENT_PCT - $PREV_PYTHON_AGENT_PCT" | bc)

    # Format change strings with +/- signs
    [[ $(echo "$CHANGE_0SHOT > 0" | bc) -eq 1 ]] && CHANGE_0SHOT="+$CHANGE_0SHOT"
    [[ $(echo "$CHANGE_FINAL > 0" | bc) -eq 1 ]] && CHANGE_FINAL="+$CHANGE_FINAL"
    [[ $(echo "$CHANGE_REPAIR > 0" | bc) -eq 1 ]] && CHANGE_REPAIR="+$CHANGE_REPAIR"
    [[ $(echo "$CHANGE_PYTHON > 0" | bc) -eq 1 ]] && CHANGE_PYTHON="+$CHANGE_PYTHON"
    [[ $(echo "$CHANGE_AGENT_AILANG > 0" | bc) -eq 1 ]] && CHANGE_AGENT_AILANG="+$CHANGE_AGENT_AILANG"
    [[ $(echo "$CHANGE_AGENT_PYTHON > 0" | bc) -eq 1 ]] && CHANGE_AGENT_PYTHON="+$CHANGE_AGENT_PYTHON"
else
    PREV_VERSION="none"
    PREV_AILANG_0SHOT_PCT="N/A"
    PREV_AILANG_FINAL_PCT="N/A"
    PREV_AILANG_REPAIR_GAIN="N/A"
    PREV_PYTHON_FINAL_PCT="N/A"
    PREV_AILANG_AGENT_PCT="N/A"
    PREV_PYTHON_AGENT_PCT="N/A"
    CHANGE_0SHOT="N/A"
    CHANGE_FINAL="N/A"
    CHANGE_REPAIR="N/A"
    CHANGE_PYTHON="N/A"
    CHANGE_AGENT_AILANG="N/A"
    CHANGE_AGENT_PYTHON="N/A"
fi

# Output CHANGELOG template
echo "=== CHANGELOG.md Template ==="
echo
cat << EOF
### Benchmark Results (M-EVAL)

**Overall Performance**: ${OVERALL_PCT}% success rate ($TOTAL_RUNS total runs)

**Standard Eval (0-shot + self-repair):**

| Metric | $PREV_VERSION | $VERSION | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | ${PREV_AILANG_0SHOT_PCT}% | ${AILANG_0SHOT_PCT}% ($AILANG_0SHOT/$AILANG_TOTAL) | **${CHANGE_0SHOT}%** |
| **Final (with repair)** | ${PREV_AILANG_FINAL_PCT}% | ${AILANG_FINAL_PCT}% ($AILANG_FINAL/$AILANG_TOTAL) | **${CHANGE_FINAL}%** |
| **Repair effectiveness** | +${PREV_AILANG_REPAIR_GAIN}pp | +${AILANG_REPAIR_GAIN}pp | **${CHANGE_REPAIR}pp** |
| **Python (final)** | ${PREV_PYTHON_FINAL_PCT}% | ${PYTHON_FINAL_PCT}% ($PYTHON_FINAL/$PYTHON_TOTAL) | ${CHANGE_PYTHON}% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | $PREV_VERSION | $VERSION | Change |
|----------|--------|--------|--------|
| **AILANG** | ${PREV_AILANG_AGENT_PCT}% | ${AILANG_AGENT_PCT}% ($AILANG_AGENT_SUCCESS/$AILANG_AGENT_TOTAL) | **${CHANGE_AGENT_AILANG}%** |
| **Python** | ${PREV_PYTHON_AGENT_PCT}% | ${PYTHON_AGENT_PCT}% ($PYTHON_AGENT_SUCCESS/$PYTHON_AGENT_TOTAL) | **${CHANGE_AGENT_PYTHON}%** |

**Key Findings:**
[Add analysis based on the data above]

EOF

echo "=== End Template ==="
echo
echo "Template generated for $VERSION (compared to $PREV_VERSION)"
