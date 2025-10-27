#!/usr/bin/env bash
# View AILANG agent conversation transcripts for analysis
# Usage: ./agent_transcripts.sh <results_dir> [benchmark_id] [--failed-only]

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
BENCHMARK_FILTER="${2:-}"
FAILED_ONLY=false

# Parse flags
for arg in "$@"; do
    if [[ "$arg" == "--failed-only" ]]; then
        FAILED_ONLY=true
    fi
done

if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "❌ Error: Results directory not found: $RESULTS_DIR"
    echo "Usage: $0 <results_dir> [benchmark_id] [--failed-only]"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 AILANG Agent Conversation Transcripts"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📁 Results Directory: $RESULTS_DIR"
if [[ -n "$BENCHMARK_FILTER" ]]; then
    echo "🔍 Filter: $BENCHMARK_FILTER"
fi
if [[ "$FAILED_ONLY" == "true" ]]; then
    echo "⚠️  Showing only FAILED benchmarks"
fi
echo ""

# Find AILANG result files
ailang_files=$(find "$RESULTS_DIR" -name "*_ailang_*.json" -type f | sort)

if [[ -z "$ailang_files" ]]; then
    echo "⚠️  No AILANG agent results found in $RESULTS_DIR"
    exit 0
fi

count=0
for file in $ailang_files; do
    benchmark_id=$(jq -r '.id' "$file")

    # Apply benchmark filter
    if [[ -n "$BENCHMARK_FILTER" && "$benchmark_id" != *"$BENCHMARK_FILTER"* ]]; then
        continue
    fi

    # Apply failed-only filter
    success=$(jq -r '(.compile_ok and .runtime_ok and .stdout_ok)' "$file")
    if [[ "$FAILED_ONLY" == "true" && "$success" == "true" ]]; then
        continue
    fi

    # Get metrics
    turns=$(jq -r '.agent_turns // 0' "$file")
    input_tokens=$(jq -r '.input_tokens // 0' "$file")
    output_tokens=$(jq -r '.output_tokens // 0' "$file")
    duration_ms=$(jq -r '.duration_ms // 0' "$file")
    duration_sec=$(echo "scale=1; $duration_ms / 1000" | bc -l)
    error_category=$(jq -r '.error_category // "unknown"' "$file")

    # Status indicator
    if [[ "$success" == "true" ]]; then
        status="✅ SUCCESS"
        status_color=""
    else
        status="❌ FAILED ($error_category)"
        status_color="⚠️ "
    fi

    count=$((count + 1))

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔷 AILANG: $benchmark_id - $status"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📊 Metrics:"
    echo "   Turns:         $turns"
    echo "   Input Tokens:  $input_tokens"
    echo "   Output Tokens: $output_tokens"
    echo "   Duration:      ${duration_sec}s"
    echo ""

    # Show transcript
    transcript=$(jq -r '.agent_transcript // "No transcript available"' "$file")

    if [[ "$transcript" != "No transcript available" ]]; then
        echo "📝 Conversation Transcript:"
        echo "────────────────────────────────────────────────────────────────────────"

        # Extract just the turn-by-turn conversation (skip task prompt)
        echo "$transcript" | awk '/^Transcript:$/,0' | tail -n +2 | head -100

        # Count lines
        total_lines=$(echo "$transcript" | wc -l | xargs)
        if [[ $total_lines -gt 102 ]]; then
            echo ""
            echo "... (transcript truncated, $total_lines total lines)"
            echo ""
            echo "💡 View full transcript with: jq -r '.agent_transcript' $file"
        fi
    else
        echo "⚠️  No transcript available"
    fi

    echo ""
    echo "📄 Full JSON: $file"
    echo ""
done

if [[ $count -eq 0 ]]; then
    echo "⚠️  No matching transcripts found"
    if [[ -n "$BENCHMARK_FILTER" ]]; then
        echo "    Try without the filter: $0 $RESULTS_DIR"
    fi
    if [[ "$FAILED_ONLY" == "true" ]]; then
        echo "    Try without --failed-only flag"
    fi
else
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Displayed $count transcript(s)"
    echo ""
    echo "💡 Tips:"
    echo "   • View specific benchmark: $0 $RESULTS_DIR <benchmark_id>"
    echo "   • View only failures: $0 $RESULTS_DIR --failed-only"
    echo "   • See KPIs summary: ./agent_kpis.sh $RESULTS_DIR"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
fi
