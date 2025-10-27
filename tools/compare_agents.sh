#!/usr/bin/env bash
# Compare Python vs AILANG agent evaluation results
# Usage: ./tools/compare_agents.sh <results_dir>
# Example: ./tools/compare_agents.sh eval_results/python_vs_ailang

set -euo pipefail

RESULTS_DIR="${1:-eval_results/python_vs_ailang}"

if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "❌ Error: Results directory not found: $RESULTS_DIR"
    echo "Usage: $0 <results_dir>"
    exit 1
fi

# Find Python and AILANG result files
PYTHON_FILES=("$RESULTS_DIR"/*_python_*.json)
AILANG_FILES=("$RESULTS_DIR"/*_ailang_*.json)

if [[ ! -e "${PYTHON_FILES[0]}" ]]; then
    echo "❌ Error: No Python results found in $RESULTS_DIR"
    exit 1
fi

if [[ ! -e "${AILANG_FILES[0]}" ]]; then
    echo "❌ Error: No AILANG results found in $RESULTS_DIR"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔬 Python vs AILANG Agent Comparison"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📁 Results Directory: $RESULTS_DIR"
echo ""

# Compare each benchmark
for python_file in "${PYTHON_FILES[@]}"; do
    # Extract benchmark ID from filename (format: benchmarkID_lang_model_timestamp.json)
    benchmark_id=$(basename "$python_file" | cut -d_ -f1)

    # Find corresponding AILANG file
    ailang_file=$(find "$RESULTS_DIR" -name "${benchmark_id}_ailang_*.json" | head -1)

    if [[ ! -f "$ailang_file" ]]; then
        echo "⚠️  Skipping $benchmark_id: No AILANG result found"
        continue
    fi

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 Benchmark: $benchmark_id"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Extract metrics using jq
    py_success=$(jq -r '.compile_ok and .runtime_ok and .stdout_ok' "$python_file")
    ai_success=$(jq -r '.compile_ok and .runtime_ok and .stdout_ok' "$ailang_file")

    py_turns=$(jq -r '.agent_turns // 0' "$python_file")
    ai_turns=$(jq -r '.agent_turns // 0' "$ailang_file")

    py_input_tokens=$(jq -r '.input_tokens // 0' "$python_file")
    ai_input_tokens=$(jq -r '.input_tokens // 0' "$ailang_file")

    py_output_tokens=$(jq -r '.output_tokens // 0' "$python_file")
    ai_output_tokens=$(jq -r '.output_tokens // 0' "$ailang_file")

    py_total_tokens=$(jq -r '.total_tokens // 0' "$python_file")
    ai_total_tokens=$(jq -r '.total_tokens // 0' "$ailang_file")

    py_cost=$(jq -r '.cost_usd // 0' "$python_file")
    ai_cost=$(jq -r '.cost_usd // 0' "$ailang_file")

    py_duration=$(jq -r '.duration_ms // 0' "$python_file")
    ai_duration=$(jq -r '.duration_ms // 0' "$ailang_file")

    py_prompt_version=$(jq -r '.prompt_version // "unknown"' "$python_file")
    ai_prompt_version=$(jq -r '.prompt_version // "unknown"' "$ailang_file")

    # Display comparison table
    printf "┌─────────────────────────┬──────────────────────┬──────────────────────┬──────────┐\n"
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Metric" "Python" "AILANG" "Winner"
    printf "├─────────────────────────┼──────────────────────┼──────────────────────┼──────────┤\n"

    # Success
    py_status=$([[ "$py_success" == "true" ]] && echo "✅ PASS" || echo "❌ FAIL")
    ai_status=$([[ "$ai_success" == "true" ]] && echo "✅ PASS" || echo "❌ FAIL")
    if [[ "$py_success" == "true" && "$ai_success" != "true" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_success" == "true" && "$py_success" != "true" ]]; then
        winner="🔷 AILANG"
    elif [[ "$py_success" == "true" && "$ai_success" == "true" ]]; then
        winner="🤝 Tie"
    else
        winner="❌ Both"
    fi
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Success" "$py_status" "$ai_status" "$winner"

    # Turns (fewer is better)
    if [[ "$py_turns" -lt "$ai_turns" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_turns" -lt "$py_turns" ]]; then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Conversation Turns" "$py_turns" "$ai_turns" "$winner"

    # Input tokens (fewer is better for efficiency)
    if [[ "$py_input_tokens" -lt "$ai_input_tokens" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_input_tokens" -lt "$py_input_tokens" ]]; then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Input Tokens" "$py_input_tokens" "$ai_input_tokens" "$winner"

    # Output tokens (fewer is better for conciseness)
    if [[ "$py_output_tokens" -lt "$ai_output_tokens" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_output_tokens" -lt "$py_output_tokens" ]]; then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Output Tokens" "$py_output_tokens" "$ai_output_tokens" "$winner"

    # Total tokens
    if [[ "$py_total_tokens" -lt "$ai_total_tokens" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_total_tokens" -lt "$py_total_tokens" ]]; then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Total Tokens" "$py_total_tokens" "$ai_total_tokens" "$winner"

    # Cost (lower is better)
    if (( $(echo "$py_cost < $ai_cost" | bc -l) )); then
        winner="🐍 Python"
    elif (( $(echo "$ai_cost < $py_cost" | bc -l) )); then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    printf "│ %-23s │ \$%-19.4f │ \$%-19.4f │ %-8s │\n" "Cost (USD)" "$py_cost" "$ai_cost" "$winner"

    # Duration (faster is better)
    if [[ "$py_duration" -lt "$ai_duration" ]]; then
        winner="🐍 Python"
    elif [[ "$ai_duration" -lt "$py_duration" ]]; then
        winner="🔷 AILANG"
    else
        winner="🤝 Tie"
    fi
    py_duration_sec=$(echo "scale=2; $py_duration / 1000" | bc)
    ai_duration_sec=$(echo "scale=2; $ai_duration / 1000" | bc)
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Duration (sec)" "${py_duration_sec}s" "${ai_duration_sec}s" "$winner"

    # Prompt version
    printf "│ %-23s │ %-20s │ %-20s │ %-8s │\n" "Prompt Version" "$py_prompt_version" "$ai_prompt_version" "N/A"

    printf "└─────────────────────────┴──────────────────────┴──────────────────────┴──────────┘\n"
    echo ""

    # Display file paths
    echo "📄 Result Files:"
    echo "   🐍 Python: $python_file"
    echo "   🔷 AILANG: $ailang_file"
    echo ""

    # Display solution code
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "💾 Solution Code"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Python solution
    echo "🐍 Python Solution:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    py_code=$(jq -r '.code // "No code captured"' "$python_file")
    if [[ "$py_code" != "No code captured" ]]; then
        echo "$py_code"
    else
        echo "(No solution code available)"
    fi
    echo ""

    # AILANG solution
    echo "🔷 AILANG Solution:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    ai_code=$(jq -r '.code // "No code captured"' "$ailang_file")
    if [[ "$ai_code" != "No code captured" ]]; then
        echo "$ai_code"
    else
        echo "(No solution code available)"
    fi
    echo ""

    # Check for transcripts
    if jq -e '.agent_transcript' "$python_file" > /dev/null 2>&1; then
        echo "📝 Full transcripts available in JSON .agent_transcript field"
    fi

    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Comparison complete!"
echo ""
echo "💡 Tip: To inspect solution code and transcripts, use jq:"
echo "   jq '.code' $RESULTS_DIR/*_python_*.json"
echo "   jq '.agent_transcript' $RESULTS_DIR/*_ailang_*.json"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
