#!/usr/bin/env bash
# Extract benchmark metrics from dashboard JSON for CHANGELOG

set -euo pipefail

JSON_FILE="${1:-docs/static/benchmarks/latest.json}"

if [[ ! -f "$JSON_FILE" ]]; then
    echo "Error: JSON file not found: $JSON_FILE" >&2
    echo "Run update_dashboard.sh first" >&2
    exit 1
fi

echo "Extracting metrics from $JSON_FILE..."
echo

# Extract metrics
VERSION=$(jq -r '.version' "$JSON_FILE")
OVERALL_RATE=$(jq -r '.aggregates.finalSuccess' "$JSON_FILE")
TOTAL_RUNS=$(jq -r '.totalRuns' "$JSON_FILE")
AILANG_RATE=$(jq -r '.languages.ailang.success_rate // "N/A"' "$JSON_FILE")
PYTHON_RATE=$(jq -r '.languages.python.success_rate // "N/A"' "$JSON_FILE")

# Convert to percentages
OVERALL_PCT=$(echo "$OVERALL_RATE * 100" | bc -l | xargs printf "%.1f")
if [[ "$AILANG_RATE" != "N/A" ]]; then
    AILANG_PCT=$(echo "$AILANG_RATE * 100" | bc -l | xargs printf "%.1f")
else
    AILANG_PCT="N/A"
fi
if [[ "$PYTHON_RATE" != "N/A" ]]; then
    PYTHON_PCT=$(echo "$PYTHON_RATE * 100" | bc -l | xargs printf "%.1f")
else
    PYTHON_PCT="N/A"
fi

# Calculate gap if both languages present
if [[ "$AILANG_RATE" != "N/A" ]] && [[ "$PYTHON_RATE" != "N/A" ]]; then
    GAP=$(echo "($PYTHON_RATE - $AILANG_RATE) * 100" | bc -l | xargs printf "%.1f")
    GAP_TEXT="Gap: $GAP percentage points (expected for new language)"
else
    GAP_TEXT=""
fi

# Find previous version in history and calculate comparison
PREV_VERSION=$(jq -r '.history | sort_by(.timestamp) | reverse | .[1].version // "none"' "$JSON_FILE" 2>/dev/null || echo "none")
COMPARISON_TEXT="[Add comparison to previous version, e.g., \"+3.5% AILANG improvement from v0.3.X\"]"

if [[ "$PREV_VERSION" != "none" ]] && [[ "$PREV_VERSION" != "null" ]] && [[ "$AILANG_RATE" != "N/A" ]]; then
    # Try to get previous AILANG success rate (handle both old and new JSON formats)
    # Note: In history, .languages is a string ("ailang,python"), data is in .languageStats
    PREV_AILANG_RATE=$(jq -r --arg v "$PREV_VERSION" '
        .history[] | select(.version == $v) |
        .languageStats.ailang.success_rate // null
    ' "$JSON_FILE" 2>/dev/null || echo "null")

    if [[ "$PREV_AILANG_RATE" != "null" ]] && [[ "$PREV_AILANG_RATE" != "N/A" ]]; then
        PREV_AILANG_PCT=$(echo "$PREV_AILANG_RATE * 100" | bc -l | xargs printf "%.1f")
        DIFF=$(echo "($AILANG_RATE - $PREV_AILANG_RATE) * 100" | bc -l | xargs printf "%.1f")

        # Format comparison text
        if (( $(echo "$DIFF > 0" | bc -l) )); then
            COMPARISON_TEXT="+${DIFF}% AILANG improvement from $PREV_VERSION (${PREV_AILANG_PCT}% → ${AILANG_PCT}%)"
        elif (( $(echo "$DIFF < 0" | bc -l) )); then
            COMPARISON_TEXT="${DIFF}% AILANG regression from $PREV_VERSION (${PREV_AILANG_PCT}% → ${AILANG_PCT}%)"
        else
            COMPARISON_TEXT="No change from $PREV_VERSION (${AILANG_PCT}%)"
        fi
    fi
fi

# Output CHANGELOG template
echo "=== CHANGELOG.md Template ==="
echo
cat << EOF
### Benchmark Results (M-EVAL)

**Overall Performance**: ${OVERALL_PCT}% success rate ($TOTAL_RUNS total runs)

**By Language:**
- **AILANG**: ${AILANG_PCT}% - New language, learning curve
- **Python**: ${PYTHON_PCT}% - Baseline for comparison
${GAP_TEXT:+- **$GAP_TEXT}

**Comparison**: $COMPARISON_TEXT

EOF

echo "=== End Template ==="
echo
echo "Use this template in CHANGELOG.md for $VERSION"
