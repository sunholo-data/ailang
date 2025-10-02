#!/usr/bin/env bash
# report_eval.sh - Aggregate evaluation results into CSV and Markdown

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
OUTPUT_CSV="${RESULTS_DIR}/summary.csv"
OUTPUT_MD="${RESULTS_DIR}/leaderboard.md"

echo "📊 Aggregating evaluation results from ${RESULTS_DIR}..."

# Check if results directory exists
if [ ! -d "${RESULTS_DIR}" ]; then
    echo "Error: Results directory ${RESULTS_DIR} not found"
    exit 1
fi

# Count JSON files
json_count=$(find "${RESULTS_DIR}" -name "*.json" -type f | wc -l | tr -d ' ')
if [ "${json_count}" -eq 0 ]; then
    echo "Error: No JSON result files found in ${RESULTS_DIR}"
    exit 1
fi

echo "Found ${json_count} result file(s)"

# Generate CSV header
echo "benchmark,lang,model,seed,tokens,cost_usd,compile,runtime,stdout,duration_ms,error_type" > "${OUTPUT_CSV}"

# Extract data from JSON files and append to CSV
for json_file in "${RESULTS_DIR}"/*.json; do
    # Use jq to extract fields (if available), otherwise use simple grep/sed
    if command -v jq &> /dev/null; then
        id=$(jq -r '.id' "${json_file}")
        lang=$(jq -r '.lang' "${json_file}")
        model=$(jq -r '.model' "${json_file}")
        seed=$(jq -r '.seed' "${json_file}")
        tokens=$(jq -r '.tokens' "${json_file}")
        cost=$(jq -r '.cost_usd' "${json_file}")
        compile=$(jq -r '.compile_ok' "${json_file}")
        runtime=$(jq -r '.runtime_ok' "${json_file}")
        stdout=$(jq -r '.stdout_ok' "${json_file}")
        duration=$(jq -r '.duration_ms' "${json_file}")
        error=$(jq -r '.error_category' "${json_file}")

        echo "${id},${lang},${model},${seed},${tokens},${cost},${compile},${runtime},${stdout},${duration},${error}" >> "${OUTPUT_CSV}"
    else
        echo "Warning: jq not found, skipping ${json_file}"
    fi
done

echo "✓ CSV report saved to ${OUTPUT_CSV}"

# Generate Markdown leaderboard
if command -v jq &> /dev/null; then
    # Extract model and seed from first file
    first_file=$(find "${RESULTS_DIR}" -name "*.json" -type f | head -1)
    model=$(jq -r '.model' "${first_file}")
    seed=$(jq -r '.seed' "${first_file}")
    date=$(date +%Y-%m-%d)

    cat > "${OUTPUT_MD}" <<EOF
# AILANG vs Python Benchmark Results

**Model:** ${model} | **Seed:** ${seed} | **Date:** ${date}

| Benchmark | Lang | Tokens | Cost | Compile | Run | Pass | Duration |
|-----------|------|--------|------|---------|-----|------|----------|
EOF

    # Add rows (sorted by benchmark, then lang)
    while IFS=',' read -r id lang model seed tokens cost compile runtime stdout duration error; do
        # Skip header
        if [ "${id}" == "benchmark" ]; then
            continue
        fi

        # Format boolean values as checkmarks
        compile_icon="✅"
        [ "${compile}" == "false" ] && compile_icon="❌"

        runtime_icon="✅"
        [ "${runtime}" == "false" ] && runtime_icon="❌"

        stdout_icon="✅"
        [ "${stdout}" == "false" ] && stdout_icon="❌"

        # Format duration
        duration_sec=$(echo "scale=2; ${duration} / 1000" | bc -l 2>/dev/null || echo "0")

        # Format cost with 4 decimal places
        cost_fmt=$(printf "%.4f" "${cost}" 2>/dev/null || echo "${cost}")

        echo "| ${id} | ${lang} | ${tokens} | \$${cost_fmt} | ${compile_icon} | ${runtime_icon} | ${stdout_icon} | ${duration_sec}s |" >> "${OUTPUT_MD}"
    done < "${OUTPUT_CSV}"

    # Calculate summary statistics
    echo "" >> "${OUTPUT_MD}"
    echo "## Summary" >> "${OUTPUT_MD}"
    echo "" >> "${OUTPUT_MD}"

    # Calculate avg token reduction (AILANG vs Python)
    python_tokens=$(awk -F',' '$2=="python" {sum+=$5; count++} END {if(count>0) print sum/count; else print 0}' "${OUTPUT_CSV}")
    ailang_tokens=$(awk -F',' '$2=="ailang" {sum+=$5; count++} END {if(count>0) print sum/count; else print 0}' "${OUTPUT_CSV}")

    if [ "${python_tokens}" != "0" ] && [ "${ailang_tokens}" != "0" ]; then
        reduction=$(echo "scale=1; (1 - ${ailang_tokens} / ${python_tokens}) * 100" | bc -l 2>/dev/null || echo "0")
        echo "- **Avg Token Reduction:** ${reduction}%" >> "${OUTPUT_MD}"
    fi

    # Calculate success rates
    ailang_success=$(awk -F',' '$2=="ailang" && $9=="true" {count++} END {print count+0}' "${OUTPUT_CSV}")
    ailang_total=$(awk -F',' '$2=="ailang" {count++} END {print count+0}' "${OUTPUT_CSV}")
    python_success=$(awk -F',' '$2=="python" && $9=="true" {count++} END {print count+0}' "${OUTPUT_CSV}")
    python_total=$(awk -F',' '$2=="python" {count++} END {print count+0}' "${OUTPUT_CSV}")

    if [ "${ailang_total}" -gt 0 ]; then
        ailang_pct=$(echo "scale=0; ${ailang_success} * 100 / ${ailang_total}" | bc -l 2>/dev/null || echo "0")
        echo "- **AILANG Success Rate:** ${ailang_pct}% (${ailang_success}/${ailang_total})" >> "${OUTPUT_MD}"
    fi

    if [ "${python_total}" -gt 0 ]; then
        python_pct=$(echo "scale=0; ${python_success} * 100 / ${python_total}" | bc -l 2>/dev/null || echo "0")
        echo "- **Python Success Rate:** ${python_pct}% (${python_success}/${python_total})" >> "${OUTPUT_MD}"
    fi

    echo "" >> "${OUTPUT_MD}"
    echo "---" >> "${OUTPUT_MD}"
    echo "" >> "${OUTPUT_MD}"
    echo "*Generated by \`tools/report_eval.sh\` on ${date}*" >> "${OUTPUT_MD}"

    echo "✓ Markdown report saved to ${OUTPUT_MD}"
    echo ""
    echo "Preview:"
    cat "${OUTPUT_MD}"
else
    echo "Warning: jq not found, Markdown report not generated"
fi
