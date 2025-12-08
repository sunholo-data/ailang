#!/usr/bin/env bash
# Test AILANG prompt effectiveness with quick eval (AILANG only, dev models)
#
# Usage:
#   test_prompt.sh <version>
#
# Example:
#   test_prompt.sh v0.3.18

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <version>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 v0.3.18" >&2
    exit 1
fi

VERSION="$1"

# Check if version exists
PROMPT_FILE=$(jq -r ".versions[\"$VERSION\"].file" prompts/versions.json)
if [ "$PROMPT_FILE" == "null" ]; then
    echo "Error: Version $VERSION not found in prompts/versions.json" >&2
    exit 1
fi

if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: Prompt file not found: $PROMPT_FILE" >&2
    exit 1
fi

echo "Testing prompt: $VERSION"
echo "  File: $PROMPT_FILE"
echo "  Models: dev models (claude-haiku-4-5)"
echo "  Languages: AILANG only (not Python)"
echo "  Benchmarks: All AILANG-compatible benchmarks"
echo ""

OUTPUT_DIR="eval_results/prompt_test_${VERSION}"

echo "→ Running eval suite (AILANG only)..."
echo "  This tests prompt effectiveness without Python baseline comparison"
echo "  Expected time: ~3-5 minutes"
echo ""

# Run eval with AILANG only
ailang eval-suite \
  --models claude-haiku-4-5 \
  --langs ailang \
  --output "$OUTPUT_DIR"

EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo "✗ Eval suite failed with exit code $EXIT_CODE" >&2
    exit $EXIT_CODE
fi

echo ""
echo "✓ Eval complete"
echo ""
echo "→ Analyzing results..."

# Generate summary
.claude/skills/eval-analyzer/scripts/quick_summary.sh "$OUTPUT_DIR"

echo ""
echo "→ Detailed failure analysis..."
.claude/skills/eval-analyzer/scripts/analyze_failures.sh "$OUTPUT_DIR" ailang

echo ""
echo "───────────────────────────────────────────────"
echo "Test complete for $VERSION"
echo ""
echo "Results: $OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "  1. Check success rate (target: >40% for AILANG)"
echo "  2. If <40%, investigate failure patterns"
echo "  3. Compare with previous version: ailang eval-compare <old_dir> $OUTPUT_DIR"
echo "  4. Examine specific failures: .claude/skills/eval-analyzer/scripts/examine_code.sh $OUTPUT_DIR <benchmark>"
