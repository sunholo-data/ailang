#!/usr/bin/env bash
# Update website benchmark dashboard with new release data

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version>" >&2
    echo "Example: $0 0.3.14" >&2
    exit 1
fi

VERSION="$1"
RESULTS_DIR="eval_results/baselines/v$VERSION"
MARKDOWN_FILE="docs/docs/benchmarks/performance.md"
JSON_FILE="docs/static/benchmarks/latest.json"

# Verify results directory exists
if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "Error: Results directory not found: $RESULTS_DIR" >&2
    echo "Run eval baseline first with run_eval_baseline.sh" >&2
    exit 1
fi

echo "Updating dashboard for v$VERSION..."
echo

# Generate Docusaurus markdown (suppress progress stderr)
echo "1/5 Generating Docusaurus markdown..."
if ailang eval-report "$RESULTS_DIR" "v$VERSION" --format=docusaurus 2>/dev/null > "$MARKDOWN_FILE"; then
    echo "  ✓ Written to $MARKDOWN_FILE"
else
    echo "  ✗ Failed to generate markdown" >&2
    exit 1
fi
echo

# Generate JSON with history preservation (writes to docs/static/benchmarks/latest.json automatically)
echo "2/5 Generating dashboard JSON with history..."
if ailang eval-report "$RESULTS_DIR" "v$VERSION" --format=json > /dev/null; then
    echo "  ✓ Written to $JSON_FILE (history preserved)"
else
    echo "  ✗ Failed to generate JSON" >&2
    exit 1
fi
echo

# Validate JSON
echo "3/5 Validating JSON..."
if VERSION_CHECK=$(jq -r '.version' "$JSON_FILE" 2>/dev/null) && [[ "$VERSION_CHECK" == "v$VERSION" ]]; then
    SUCCESS_RATE=$(jq -r '.aggregates.finalSuccess' "$JSON_FILE" 2>/dev/null)
    echo "  ✓ Version: $VERSION_CHECK"
    echo "  ✓ Success rate: $SUCCESS_RATE"
else
    echo "  ✗ JSON validation failed" >&2
    exit 1
fi
echo

# Clear Docusaurus cache
echo "4/5 Clearing Docusaurus cache..."
if (cd docs && npm run clear > /dev/null 2>&1); then
    echo "  ✓ Cache cleared"
else
    echo "  ⚠ Cache clear failed (npm may not be installed)"
fi
echo

# Summary
echo "5/5 Summary"
echo "  ✓ Dashboard updated for v$VERSION"
echo "  ✓ Markdown: $MARKDOWN_FILE"
echo "  ✓ JSON: $JSON_FILE"
echo
echo "Next steps:"
echo "  1. Test locally: cd docs && npm start"
echo "  2. Visit: http://localhost:3000/ailang/docs/benchmarks/performance"
echo "  3. Verify timeline shows v$VERSION"
echo "  4. Commit: git add $MARKDOWN_FILE $JSON_FILE"
echo "  5. Commit: git commit -m 'Update benchmark dashboard for v$VERSION'"
echo "  6. Push: git push"
