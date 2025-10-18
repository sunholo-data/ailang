#!/usr/bin/env bash
# Generate llms.txt file from all documentation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DOCS_OUTPUT="$REPO_ROOT/docs/llms.txt"
ROOT_OUTPUT="$REPO_ROOT/llms.txt"
OUTPUT="$DOCS_OUTPUT"

echo "=== Generating llms.txt from documentation ==="

# Start with header
cat > "$OUTPUT" << 'EOF'
# AILANG Documentation for LLMs

This file contains all AILANG documentation in a single file for LLM consumption.

Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

---

EOF

# Add main documentation files
echo "## Adding core documentation..."

# README.md
echo "" >> "$OUTPUT"
echo "# README" >> "$OUTPUT"
echo "" >> "$OUTPUT"
cat "$REPO_ROOT/README.md" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "---" >> "$OUTPUT"
echo "" >> "$OUTPUT"

# CHANGELOG.md - EXCLUDED (internal version history, not needed for language learning)
# CLAUDE.md - EXCLUDED (internal dev instructions, not needed for language learning)

# Add all docs/ files
echo "## Adding docs/ directory..."

# docs/index.md
if [ -f "$REPO_ROOT/docs/index.md" ]; then
    echo "# Documentation Index" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    cat "$REPO_ROOT/docs/index.md" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "---" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# docs/docs/guides/ - KEEP only language learning guides, exclude internal dev docs
if [ -d "$REPO_ROOT/docs/docs/guides" ]; then
    # Language learning guides to keep
    for guide in "getting-started.md" "ai-prompt-guide.md" "module_execution.md" "wasm-integration.md"; do
        file="$REPO_ROOT/docs/docs/guides/$guide"
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "# Guide: $filename" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            cat "$file" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            echo "---" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
        fi
    done
    # EXCLUDED: development.md, agent-integration.md, benchmarking.md (internal dev docs)
    # EXCLUDED: evaluation/ subdirectory (internal M-EVAL-LOOP docs)
fi

# docs/reference/
if [ -d "$REPO_ROOT/docs/reference" ]; then
    for file in "$REPO_ROOT/docs/reference"/*.md; do
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "# Reference: $filename" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            cat "$file" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            echo "---" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
        fi
    done
fi

# Add prompts/ (latest version only)
echo "## Adding prompts/ directory..."

if [ -f "$REPO_ROOT/prompts/versions.json" ]; then
    # Extract active version from versions.json
    ACTIVE_VERSION=$(jq -r '.active' "$REPO_ROOT/prompts/versions.json")
    PROMPT_FILE=$(jq -r ".versions.\"$ACTIVE_VERSION\".file" "$REPO_ROOT/prompts/versions.json")

    if [ -f "$REPO_ROOT/$PROMPT_FILE" ]; then
        filename=$(basename "$PROMPT_FILE")
        echo "# AI Teaching Prompt (Latest: $ACTIVE_VERSION)" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
        cat "$REPO_ROOT/$PROMPT_FILE" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
        echo "---" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
    fi
fi

# Add implementation status
if [ -f "$REPO_ROOT/docs/reference/implementation-status.md" ]; then
    echo "# Implementation Status" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    cat "$REPO_ROOT/docs/reference/implementation-status.md" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "---" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# Add examples/STATUS.md
if [ -f "$REPO_ROOT/examples/STATUS.md" ]; then
    echo "# Examples Status" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    cat "$REPO_ROOT/examples/STATUS.md" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "---" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# Calculate size
SIZE=$(wc -c < "$OUTPUT" | tr -d ' ')
LINES=$(wc -l < "$OUTPUT" | tr -d ' ')


# Copy to root for easy local access  
cp "$DOCS_OUTPUT" "$ROOT_OUTPUT"

echo "✓ llms.txt generated successfully"
echo "  Size: $SIZE bytes"
echo "  Lines: $LINES"
echo "  Website: $DOCS_OUTPUT"
echo "  Root: $ROOT_OUTPUT"
echo "  URL: https://sunholo-data.github.io/ailang/llms.txt"
