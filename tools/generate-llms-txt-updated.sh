#!/usr/bin/env bash
# Generate llms.txt file from all documentation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Generate to docs/ for website serving
DOCS_OUTPUT="$REPO_ROOT/docs/llms.txt"
# Also copy to root for easy local access
ROOT_OUTPUT="$REPO_ROOT/llms.txt"

echo "=== Generating llms.txt from documentation ==="

# Generate the file (use docs location as primary)
OUTPUT="$DOCS_OUTPUT"

# Start with header
cat > "$OUTPUT" << 'EOF'
# AILANG Documentation for LLMs

This file contains all AILANG documentation in a single file for LLM consumption.

Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

---

EOF

# [Rest of the script content - all the cat >> "$OUTPUT" commands remain the same]

# At the end, copy to root as well
cp "$DOCS_OUTPUT" "$ROOT_OUTPUT"

# Calculate size
SIZE=$(wc -c < "$DOCS_OUTPUT" | tr -d ' ')
LINES=$(wc -l < "$DOCS_OUTPUT" | tr -d ' ')

echo "✓ llms.txt generated successfully"
echo "  Size: $SIZE bytes"
echo "  Lines: $LINES"
echo "  Website location: $DOCS_OUTPUT"
echo "  Root location: $ROOT_OUTPUT"
echo "  Will be available at: https://sunholo-data.github.io/ailang/llms.txt"
