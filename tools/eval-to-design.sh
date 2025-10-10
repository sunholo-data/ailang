#!/usr/bin/env bash
# eval-to-design.sh - Run full eval suite and generate design docs from failures

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}🔬 AILANG Eval → Design Doc Workflow${NC}"
echo "========================================"
echo ""

# Configuration
RESULTS_DIR="${1:-eval_results}"
OUTPUT_DIR="${2:-design_docs/planned}"
MODEL="${3:-gpt5}"
MIN_FREQUENCY="${4:-2}"

echo "Configuration:"
echo "  Results directory: $RESULTS_DIR"
echo "  Output directory: $OUTPUT_DIR"
echo "  Model: $MODEL"
echo "  Min frequency: $MIN_FREQUENCY"
echo ""

# Step 1: Check if eval results exist
if [ ! -d "$RESULTS_DIR" ] || [ -z "$(ls -A $RESULTS_DIR/*.json 2>/dev/null)" ]; then
    echo -e "${YELLOW}⚠️  No eval results found in $RESULTS_DIR${NC}"
    echo ""
    echo "Would you like to run the eval suite now? (y/n)"
    read -p "> " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${CYAN}→${NC} Running eval suite (parallel)..."
        bin/ailang eval-suite --output "$RESULTS_DIR" || {
            echo -e "${RED}✗ Eval suite failed${NC}"
            exit 1
        }
    else
        echo "Exiting. Run 'ailang eval-suite' first."
        exit 1
    fi
fi

# Count result files
result_count=$(ls -1 "$RESULTS_DIR"/*.json 2>/dev/null | wc -l | tr -d ' ')
echo -e "${GREEN}✓${NC} Found $result_count eval result file(s)"
echo ""

# Step 2: Analyze results
echo -e "${CYAN}→${NC} Analyzing results..."
echo ""

if ! command -v ailang &> /dev/null; then
    echo -e "${YELLOW}⚠️  ailang not found in PATH, building...${NC}"
    make build
    AILANG_CMD="./bin/ailang"
else
    AILANG_CMD="ailang"
fi

# Run analysis with dry-run first to show issues
echo "Issues discovered:"
echo ""
$AILANG_CMD eval-analyze \
    --results "$RESULTS_DIR" \
    --output "$OUTPUT_DIR" \
    --model "$MODEL" \
    --min-frequency "$MIN_FREQUENCY" \
    --dry-run

echo ""
echo -e "${CYAN}→${NC} Generate design documents for these issues? (y/n)"
read -p "> " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Skipping design doc generation."
    exit 0
fi

# Check API key
case $MODEL in
    gpt5|gpt5-mini|gpt-4*)
        if [ -z "${OPENAI_API_KEY:-}" ]; then
            echo -e "${RED}✗ OPENAI_API_KEY not set${NC}"
            echo ""
            echo "Set your API key:"
            echo "  export OPENAI_API_KEY='sk-...'"
            exit 1
        fi
        ;;
    claude-*)
        if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
            echo -e "${RED}✗ ANTHROPIC_API_KEY not set${NC}"
            echo ""
            echo "Set your API key:"
            echo "  export ANTHROPIC_API_KEY='sk-ant-...'"
            exit 1
        fi
        ;;
    gemini-*)
        if ! command -v gcloud &> /dev/null || ! gcloud auth application-default print-access-token &> /dev/null; then
            echo -e "${RED}✗ gcloud not authenticated${NC}"
            echo ""
            echo "Authenticate with:"
            echo "  gcloud auth application-default login"
            exit 1
        fi
        ;;
esac

# Step 3: Generate design docs
echo ""
echo -e "${CYAN}→${NC} Generating design documents with $MODEL..."
echo ""

$AILANG_CMD eval-analyze \
    --results "$RESULTS_DIR" \
    --output "$OUTPUT_DIR" \
    --model "$MODEL" \
    --min-frequency "$MIN_FREQUENCY" \
    --generate=true

echo ""
echo -e "${GREEN}✓ Workflow complete!${NC}"
echo ""
echo "Generated files:"
echo "  Design docs: $OUTPUT_DIR/"
echo "  Summary: $OUTPUT_DIR/EVAL_ANALYSIS_*.md"
echo "  Analysis data: $RESULTS_DIR/analysis_*.json"
echo ""
echo "Next steps:"
echo "  1. Review generated design documents:"
echo "     ls -lh $OUTPUT_DIR/"
echo ""
echo "  2. Adjust priorities and estimates as needed"
echo ""
echo "  3. Move approved designs to milestone tracking:"
echo "     mv $OUTPUT_DIR/approved_design.md design_docs/planned/"
echo ""
echo "  4. After implementing fixes, re-run evals:"
echo "     ailang eval-suite"
echo "     ailang eval-report eval_results/ v0.3.1"
echo ""
