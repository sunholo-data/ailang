#!/usr/bin/env bash
# run_benchmark_suite.sh - Run all benchmarks with the recommended model suite

set -euo pipefail

# Configuration
MODELS=("gpt5" "claude-sonnet-4-5" "gemini-2-5-pro")
BENCHMARKS=("fizzbuzz" "json_parse" "pipeline" "cli_args" "adt_option")
SEED=42
LANGS="python,ailang"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}🚀 AILANG Benchmark Suite${NC}"
echo "=========================="
echo ""
echo "Models: ${MODELS[@]}"
echo "Benchmarks: ${BENCHMARKS[@]}"
echo "Seed: ${SEED}"
echo "Languages: ${LANGS}"
echo ""

# Check API keys
missing_keys=0

if [ -z "${OPENAI_API_KEY:-}" ]; then
    echo -e "${YELLOW}⚠️  OPENAI_API_KEY not set (skipping GPT-5)${NC}"
    missing_keys=1
fi

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo -e "${YELLOW}⚠️  ANTHROPIC_API_KEY not set (skipping Claude)${NC}"
    missing_keys=1
fi

# Check for gcloud authentication (Gemini uses Vertex AI)
if ! command -v gcloud &> /dev/null; then
    echo -e "${YELLOW}⚠️  gcloud CLI not found (skipping Gemini)${NC}"
    missing_keys=1
elif ! gcloud auth application-default print-access-token &> /dev/null; then
    echo -e "${YELLOW}⚠️  gcloud not authenticated (skipping Gemini)${NC}"
    echo "    Run: gcloud auth application-default login"
    missing_keys=1
fi

if [ $missing_keys -eq 1 ]; then
    echo ""
    echo "Set API keys to run with real models:"
    echo "  export OPENAI_API_KEY='sk-...'"
    echo "  export ANTHROPIC_API_KEY='sk-ant-...'"
    echo "  gcloud auth application-default login  # For Gemini"
    echo ""
    read -p "Continue with available models? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Clean previous results
echo -e "${CYAN}→${NC} Cleaning previous results..."
make eval-clean || rm -rf eval_results/*.json eval_results/*.csv eval_results/*.md

# Run benchmarks
total_runs=$((${#MODELS[@]} * ${#BENCHMARKS[@]}))
current_run=0

for model in "${MODELS[@]}"; do
    # Check if we have the API key for this model
    case $model in
        gpt5|gpt5-mini)
            if [ -z "${OPENAI_API_KEY:-}" ]; then
                echo -e "${YELLOW}⚠️  Skipping $model (no API key)${NC}"
                continue
            fi
            ;;
        claude-sonnet-4-5)
            if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
                echo -e "${YELLOW}⚠️  Skipping $model (no API key)${NC}"
                continue
            fi
            ;;
        gemini-2-5-pro)
            if ! gcloud auth application-default print-access-token &> /dev/null; then
                echo -e "${YELLOW}⚠️  Skipping $model (gcloud not authenticated)${NC}"
                continue
            fi
            ;;
    esac

    for benchmark in "${BENCHMARKS[@]}"; do
        current_run=$((current_run + 1))
        echo ""
        echo -e "${CYAN}[$current_run/$total_runs]${NC} Running ${GREEN}$benchmark${NC} with ${GREEN}$model${NC}"

        if ailang eval \
            --benchmark "$benchmark" \
            --model "$model" \
            --seed "$SEED" \
            --langs "$LANGS"; then
            echo -e "${GREEN}✓${NC} Completed"
        else
            echo -e "${RED}✗${NC} Failed (continuing...)"
        fi

        # Rate limiting: wait between runs
        if [ $current_run -lt $total_runs ]; then
            echo -e "${YELLOW}⏱${NC}  Waiting 5 seconds (rate limiting)..."
            sleep 5
        fi
    done
done

echo ""
echo -e "${CYAN}→${NC} Generating report..."
make eval-report

echo ""
echo -e "${GREEN}✓ Benchmark suite complete!${NC}"
echo ""
echo "Results:"
echo "  - JSON: eval_results/*.json"
echo "  - CSV:  eval_results/summary.csv"
echo "  - MD:   eval_results/leaderboard.md"
echo ""
echo "View results:"
echo "  cat eval_results/leaderboard.md"
