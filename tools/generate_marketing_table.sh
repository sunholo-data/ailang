#!/usr/bin/env bash
#
# generate_marketing_table.sh - Generate marketing-ready comparison table
#
# Usage:
#   ./tools/generate_marketing_table.sh INPUT_DIR [OUTPUT_FILE]
#
# Example:
#   ./tools/generate_marketing_table.sh eval_results/full_comparison docs/comparison_table.md
#
# Output: Markdown table showing AILANG vs Python for each benchmark across all models

set -uo pipefail

if [ $# -lt 1 ]; then
  echo "Error: Missing input directory"
  echo "Usage: $0 INPUT_DIR [OUTPUT_FILE]"
  exit 1
fi

INPUT_DIR="$1"
OUTPUT_FILE="${2:-comparison_table.md}"

if [ ! -d "$INPUT_DIR" ]; then
  echo "Error: Directory not found: $INPUT_DIR"
  exit 1
fi

echo "Generating marketing comparison table from $INPUT_DIR..."

# Get all unique benchmarks and models
BENCHMARKS=$(cat "$INPUT_DIR"/*.json | jq -r '.id' | sort -u | grep -v "^null$")
MODELS=$(cat "$INPUT_DIR"/*.json | jq -r '.model' | sort -u)

# Generate markdown table
cat > "$OUTPUT_FILE" << 'EOF'
# AILANG vs Python: AI Code Generation Benchmark Comparison

> Generated from automated evaluation across multiple AI models

## Summary

EOF

# Add summary stats
TOTAL_RUNS=$(cat "$INPUT_DIR"/*.json | jq -s 'length')
AILANG_SUCCESS=$(cat "$INPUT_DIR"/*_ailang_*.json | jq -s 'map(select(.stdout_ok == true)) | length')
PYTHON_SUCCESS=$(cat "$INPUT_DIR"/*_python_*.json | jq -s 'map(select(.stdout_ok == true)) | length')
AILANG_TOTAL=$(cat "$INPUT_DIR"/*_ailang_*.json | jq -s 'length')
PYTHON_TOTAL=$(cat "$INPUT_DIR"/*_python_*.json | jq -s 'length')
AILANG_AVG_COST=$(cat "$INPUT_DIR"/*_ailang_*.json | jq -s 'if length > 0 then (map(.cost_usd) | (add / length * 10000 | round) / 10000) else 0 end')
PYTHON_AVG_COST=$(cat "$INPUT_DIR"/*_python_*.json | jq -s 'if length > 0 then (map(.cost_usd) | (add / length * 10000 | round) / 10000) else 0 end')

AILANG_PCT=$(( AILANG_SUCCESS * 100 / AILANG_TOTAL ))
PYTHON_PCT=$(( PYTHON_SUCCESS * 100 / PYTHON_TOTAL ))
DELTA=$(( AILANG_PCT - PYTHON_PCT ))

if [ $DELTA -gt 0 ]; then
  DELTA_STR="**+${DELTA}%** 🏆 AILANG advantage"
elif [ $DELTA -lt 0 ]; then
  DELTA_STR="**${DELTA}%** ⚠️ Python advantage"
else
  DELTA_STR="**Tie**"
fi

cat >> "$OUTPUT_FILE" << EOF
### Key Performance Indicator

**AILANG vs Python Delta: $DELTA_STR**

### Detailed Metrics

| Metric | AILANG | Python | Delta |
|--------|--------|--------|-------|
| Success Rate | ${AILANG_PCT}% ($AILANG_SUCCESS/$AILANG_TOTAL) | ${PYTHON_PCT}% ($PYTHON_SUCCESS/$PYTHON_TOTAL) | $DELTA% |
| Avg Cost per Benchmark | \$$AILANG_AVG_COST | \$$PYTHON_AVG_COST | $(echo "$AILANG_AVG_COST $PYTHON_AVG_COST" | awk '{printf "%.2f", $1-$2}') |
| Models Tested | $(echo "$MODELS" | wc -l | tr -d ' ') | $(echo "$MODELS" | wc -l | tr -d ' ') | - |
| Benchmarks | $(echo "$BENCHMARKS" | wc -l | tr -d ' ') | $(echo "$BENCHMARKS" | wc -l | tr -d ' ') | - |

## Detailed Comparison

| Benchmark | Description | AILANG Success | Python Success | AILANG Tokens (in/out) | Python Tokens (in/out) | AILANG Speed (ms) | Python Speed (ms) | AILANG Cost | Python Cost | Status |
|-----------|-------------|----------------|----------------|------------------------|------------------------|-------------------|-------------------|-------------|-------------|--------|
EOF

# For each benchmark, compare AILANG vs Python
for BENCH in $BENCHMARKS; do
  # Get description from first file
  DESC=$(grep -h "^description:" "benchmarks/${BENCH}.yml" 2>/dev/null | sed 's/description: "\(.*\)"/\1/' | tr -d '"' || echo "N/A")

  # Get AILANG metrics (average across models)
  AILANG_DATA=$(cat "$INPUT_DIR"/${BENCH}_ailang_*.json 2>/dev/null | jq -s '
    if length > 0 then
      {
        pass: (map(select(.stdout_ok == true)) | length),
        total: length,
        avg_input: (map(.input_tokens) | add / length | round),
        avg_output: (map(.output_tokens) | add / length | round),
        avg_speed: (map(.duration_ms) | add / length | round),
        avg_cost: (map(.cost_usd) | (add / length * 10000 | round) / 10000),
        errors: (map(select(.stdout_ok == false) | .error_category) | unique | join(", "))
      }
    else
      {pass: 0, total: 0, avg_input: 0, avg_output: 0, avg_speed: 0, avg_cost: 0, errors: "N/A"}
    end
  ')

  # Get Python metrics (average across models)
  PYTHON_DATA=$(cat "$INPUT_DIR"/${BENCH}_python_*.json 2>/dev/null | jq -s '
    if length > 0 then
      {
        pass: (map(select(.stdout_ok == true)) | length),
        total: length,
        avg_input: (map(.input_tokens) | add / length | round),
        avg_output: (map(.output_tokens) | add / length | round),
        avg_speed: (map(.duration_ms) | add / length | round),
        avg_cost: (map(.cost_usd) | (add / length * 10000 | round) / 10000),
        errors: (map(select(.stdout_ok == false) | .error_category) | unique | join(", "))
      }
    else
      {pass: 0, total: 0, avg_input: 0, avg_output: 0, avg_speed: 0, avg_cost: 0, errors: "N/A"}
    end
  ')

  AILANG_PASS=$(echo "$AILANG_DATA" | jq -r '.pass')
  AILANG_TOTAL=$(echo "$AILANG_DATA" | jq -r '.total')
  AILANG_IN=$(echo "$AILANG_DATA" | jq -r '.avg_input')
  AILANG_OUT=$(echo "$AILANG_DATA" | jq -r '.avg_output')
  AILANG_SPEED=$(echo "$AILANG_DATA" | jq -r '.avg_speed')
  AILANG_COST=$(echo "$AILANG_DATA" | jq -r '.avg_cost')
  AILANG_ERRORS=$(echo "$AILANG_DATA" | jq -r '.errors')

  PYTHON_PASS=$(echo "$PYTHON_DATA" | jq -r '.pass')
  PYTHON_TOTAL=$(echo "$PYTHON_DATA" | jq -r '.total')
  PYTHON_IN=$(echo "$PYTHON_DATA" | jq -r '.avg_input')
  PYTHON_OUT=$(echo "$PYTHON_DATA" | jq -r '.avg_output')
  PYTHON_SPEED=$(echo "$PYTHON_DATA" | jq -r '.avg_speed')
  PYTHON_COST=$(echo "$PYTHON_DATA" | jq -r '.avg_cost')
  PYTHON_ERRORS=$(echo "$PYTHON_DATA" | jq -r '.errors')

  # Determine status (with safe numeric comparison)
  AILANG_PASS_NUM="${AILANG_PASS:-0}"
  PYTHON_PASS_NUM="${PYTHON_PASS:-0}"

  if [ "$AILANG_PASS_NUM" -gt 0 ] 2>/dev/null && [ "$PYTHON_PASS_NUM" -gt 0 ] 2>/dev/null; then
    STATUS="✅ Both passing"
  elif [ "$AILANG_PASS_NUM" -gt 0 ] 2>/dev/null; then
    STATUS="🏆 AILANG only (Python: $PYTHON_ERRORS)"
  elif [ "$PYTHON_PASS_NUM" -gt 0 ] 2>/dev/null; then
    STATUS="⚠️ Python only (AILANG: $AILANG_ERRORS)"
  else
    STATUS="❌ Both failing"
  fi

  echo "| $BENCH | $DESC | $AILANG_PASS/$AILANG_TOTAL | $PYTHON_PASS/$PYTHON_TOTAL | $AILANG_IN/$AILANG_OUT | $PYTHON_IN/$PYTHON_OUT | ${AILANG_SPEED}ms | ${PYTHON_SPEED}ms | \$$AILANG_COST | \$$PYTHON_COST | $STATUS |" >> "$OUTPUT_FILE"
done

# Add model breakdown
cat >> "$OUTPUT_FILE" << EOF

## Model-by-Model Results

| Model | AILANG Pass Rate | Python Pass Rate | Advantage |
|-------|------------------|------------------|-----------|
EOF

for MODEL in $MODELS; do
  MODEL_AILANG_PASS=$(cat "$INPUT_DIR"/*_ailang_*${MODEL}*.json 2>/dev/null | jq -s 'map(select(.stdout_ok == true)) | length' || echo 0)
  MODEL_AILANG_TOTAL=$(cat "$INPUT_DIR"/*_ailang_*${MODEL}*.json 2>/dev/null | jq -s 'length' || echo 1)
  MODEL_PYTHON_PASS=$(cat "$INPUT_DIR"/*_python_*${MODEL}*.json 2>/dev/null | jq -s 'map(select(.stdout_ok == true)) | length' || echo 0)
  MODEL_PYTHON_TOTAL=$(cat "$INPUT_DIR"/*_python_*${MODEL}*.json 2>/dev/null | jq -s 'length' || echo 1)

  AILANG_PCT=$(( MODEL_AILANG_PASS * 100 / MODEL_AILANG_TOTAL ))
  PYTHON_PCT=$(( MODEL_PYTHON_PASS * 100 / MODEL_PYTHON_TOTAL ))

  if [ "$AILANG_PCT" -gt "$PYTHON_PCT" ]; then
    ADV="+$(( AILANG_PCT - PYTHON_PCT ))% for AILANG"
  elif [ "$PYTHON_PCT" -gt "$AILANG_PCT" ]; then
    ADV="+$(( PYTHON_PCT - AILANG_PCT ))% for Python"
  else
    ADV="Tie"
  fi

  echo "| $MODEL | $AILANG_PCT% ($MODEL_AILANG_PASS/$MODEL_AILANG_TOTAL) | $PYTHON_PCT% ($MODEL_PYTHON_PASS/$MODEL_PYTHON_TOTAL) | $ADV |" >> "$OUTPUT_FILE"
done

cat >> "$OUTPUT_FILE" << EOF

---

*Generated: $(date)*
*Source: Automated AI code generation benchmarks*
EOF

echo "✓ Marketing table generated: $OUTPUT_FILE"
