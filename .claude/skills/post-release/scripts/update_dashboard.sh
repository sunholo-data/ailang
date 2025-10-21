#!/usr/bin/env bash
# Update website benchmark dashboard with new release data

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version>" >&2
    echo "Example: $0 0.3.14" >&2
    exit 1
fi

VERSION="$1"
# Handle both v0.3.15 and 0.3.15 formats
RESULTS_DIR="eval_results/baselines/$VERSION"
MARKDOWN_FILE="docs/docs/benchmarks/performance.md"
JSON_FILE="docs/static/benchmarks/latest.json"

# Verify results directory exists
if [[ ! -d "$RESULTS_DIR" ]]; then
    echo "Error: Results directory not found: $RESULTS_DIR" >&2
    echo "Run eval baseline first with run_eval_baseline.sh" >&2
    exit 1
fi

echo "Updating dashboard for $VERSION..."
echo

# NOTE: We do NOT regenerate performance.md - it's a static template with React components
# The React components read data from latest.json, so we only update the JSON file

# Generate JSON with history preservation (writes to docs/static/benchmarks/latest.json automatically)
echo "1/3 Generating dashboard JSON with history..."
if ailang eval-report "$RESULTS_DIR" "$VERSION" --format=json > /dev/null; then
    echo "  ✓ Written to $JSON_FILE (history preserved)"
else
    echo "  ✗ Failed to generate JSON" >&2
    exit 1
fi
echo

# Validate JSON (version in JSON matches what we passed)
echo "2/3 Validating JSON..."
if VERSION_CHECK=$(jq -r '.version' "$JSON_FILE" 2>/dev/null) && [[ "$VERSION_CHECK" == "$VERSION" ]]; then
    SUCCESS_RATE=$(jq -r '.aggregates.finalSuccess' "$JSON_FILE" 2>/dev/null)
    echo "  ✓ Version: $VERSION_CHECK"
    echo "  ✓ Success rate: $SUCCESS_RATE"
else
    echo "  ✗ JSON validation failed (expected: $VERSION, got: $VERSION_CHECK)" >&2
    exit 1
fi
echo

# Clear Docusaurus cache
echo "3/3 Clearing Docusaurus cache..."
if (cd docs && npm run clear > /dev/null 2>&1); then
    echo "  ✓ Cache cleared"
else
    echo "  ⚠ Cache clear failed (npm may not be installed)"
fi
echo

# Summary
echo "✓ Dashboard JSON updated for $VERSION"
echo "  Data file: $JSON_FILE"
echo "  Template: $MARKDOWN_FILE (static - not modified)"
echo
echo "Next steps:"
echo "  1. Test locally: cd docs && npm start"
echo "  2. Visit: http://localhost:3000/ailang/docs/benchmarks/performance"
echo "  3. Verify radar charts and timeline show $VERSION"
echo "  4. Commit: git add $JSON_FILE"
echo "  5. Commit: git commit -m 'Update benchmark dashboard for $VERSION'"
echo "  6. Push: git push"
echo
echo "Note: performance.md is a static template with React components."
echo "      The components read data from latest.json - no need to regenerate it."
