#!/usr/bin/env bash
# Show agent KPIs (tokens, turns, cost) for easy analysis
# Focus: Minimize tokens and turns to improve language/prompts
# Usage: ./agent_kpis.sh <results_dir>

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"

if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "❌ Error: Results directory not found: $RESULTS_DIR"
    echo "Usage: $0 <results_dir>"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🤖 Agent KPIs: Tokens & Turns Analysis"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📁 Results Directory: $RESULTS_DIR"
echo ""

# Count total agent results
total_files=$(find "$RESULTS_DIR" -name "*.json" -type f | wc -l | xargs)
echo "📊 Total benchmarks: $total_files"
echo ""

# Group by language for Python vs AILANG comparison
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📈 KPIs by Language (Goal: Minimize tokens & turns)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

for lang in python ailang; do
    lang_files=$(find "$RESULTS_DIR" -name "*_${lang}_*.json" -type f)

    if [[ -z "$lang_files" ]]; then
        continue
    fi

    # Calculate aggregate stats
    total_turns=0
    total_input_tokens=0
    total_output_tokens=0
    total_cost=0
    success_count=0
    fail_count=0
    count=0

    for file in $lang_files; do
        turns=$(jq -r '.agent_turns // 0' "$file")
        input_tokens=$(jq -r '.input_tokens // 0' "$file")
        output_tokens=$(jq -r '.output_tokens // 0' "$file")
        cost=$(jq -r '.cost_usd // 0' "$file")
        success=$(jq -r '(.compile_ok and .runtime_ok and .stdout_ok)' "$file")

        total_turns=$((total_turns + turns))
        total_input_tokens=$((total_input_tokens + input_tokens))
        total_output_tokens=$((total_output_tokens + output_tokens))
        total_cost=$(echo "$total_cost + $cost" | bc -l)
        count=$((count + 1))

        if [[ "$success" == "true" ]]; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
    done

    # Calculate averages
    avg_turns=$(echo "scale=1; $total_turns / $count" | bc -l)
    avg_input_tokens=$(echo "scale=0; $total_input_tokens / $count" | bc -l)
    avg_output_tokens=$(echo "scale=0; $total_output_tokens / $count" | bc -l)
    avg_cost=$(echo "scale=4; $total_cost / $count" | bc -l)
    success_rate=$(echo "scale=1; $success_count * 100 / $count" | bc -l)

    # Display language stats
    if [[ "$lang" == "python" ]]; then
        icon="🐍"
        name="Python"
    else
        icon="🔷"
        name="AILANG"
    fi

    echo "$icon $name ($count benchmarks, ${success_rate}% success)"
    echo "   Avg Turns:        $avg_turns (lower is better)"
    echo "   Avg Input Tokens: $avg_input_tokens (lower is better)"
    echo "   Avg Output Tokens: $avg_output_tokens"
    echo "   Avg Cost:         \$$avg_cost"
    echo "   Total Cost:       \$$total_cost"
    echo ""
done

# Top 5 most expensive benchmarks (by turns)
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⏱️  Most Expensive (by turns - candidates for optimization)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

find "$RESULTS_DIR" -name "*.json" -type f -exec jq -r '{id, lang, turns: .agent_turns, tokens: .total_tokens, cost: .cost_usd, success: (.compile_ok and .runtime_ok and .stdout_ok)} | "\(.turns)|\(.id)|\(.lang)|\(.tokens)|\(.cost)|\(.success)"' {} \; | \
    sort -t'|' -k1 -rn | \
    head -5 | \
    while IFS='|' read -r turns id lang tokens cost success; do
        status=$([ "$success" = "true" ] && echo "✅" || echo "❌")
        printf "%-3s %s %-25s %-8s %6s turns, %8s tokens, \$%.4f\n" "$status" "$([ "$lang" = "python" ] && echo "🐍" || echo "🔷")" "$id" "$lang" "$turns" "$tokens" "$cost"
    done

echo ""

# Top 5 most efficient benchmarks (lowest turns among successes)
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🏆 Most Efficient (lowest turns among successes - learn from these)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

find "$RESULTS_DIR" -name "*.json" -type f -exec jq -r 'select(.compile_ok and .runtime_ok and .stdout_ok) | {id, lang, turns: .agent_turns, tokens: .total_tokens, cost: .cost_usd} | "\(.turns)|\(.id)|\(.lang)|\(.tokens)|\(.cost)"' {} \; | \
    sort -t'|' -k1 -n | \
    head -5 | \
    while IFS='|' read -r turns id lang tokens cost; do
        printf "✅ %s %-25s %-8s %6s turns, %8s tokens, \$%.4f\n" "$([ "$lang" = "python" ] && echo "🐍" || echo "🔷")" "$id" "$lang" "$turns" "$tokens" "$cost"
    done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💡 Optimization Tips:"
echo ""
echo "  • High turns → Check if prompt is unclear or language has rough edges"
echo "  • High input tokens → Prompt might be too verbose"
echo "  • Failed benchmarks → View transcript with: ./agent_transcripts.sh"
echo "  • Compare languages → Use: ./compare_agents_py_vs_ailang.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
