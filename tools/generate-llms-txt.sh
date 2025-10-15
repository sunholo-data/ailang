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

# CHANGELOG.md
echo "# CHANGELOG" >> "$OUTPUT"
echo "" >> "$OUTPUT"
cat "$REPO_ROOT/CHANGELOG.md" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "---" >> "$OUTPUT"
echo "" >> "$OUTPUT"

# CLAUDE.md (project instructions)
if [ -f "$REPO_ROOT/CLAUDE.md" ]; then
    echo "# CLAUDE.md (Project Instructions)" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    cat "$REPO_ROOT/CLAUDE.md" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "---" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

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

# docs/guides/ (including subdirectories)
if [ -d "$REPO_ROOT/docs/guides" ]; then
    for file in "$REPO_ROOT/docs/guides"/*.md; do
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
fi

# docs/docs/guides/ (Docusaurus structure)
if [ -d "$REPO_ROOT/docs/docs/guides" ]; then
    for file in "$REPO_ROOT/docs/docs/guides"/*.md; do
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
    # Include subdirectories (evaluation, etc.)
    for subdir in "$REPO_ROOT/docs/docs/guides"/*; do
        if [ -d "$subdir" ]; then
            subdir_name=$(basename "$subdir")
            for file in "$subdir"/*.md; do
                if [ -f "$file" ]; then
                    filename=$(basename "$file")
                    echo "# Guide/$subdir_name: $filename" >> "$OUTPUT"
                    echo "" >> "$OUTPUT"
                    cat "$file" >> "$OUTPUT"
                    echo "" >> "$OUTPUT"
                    echo "---" >> "$OUTPUT"
                    echo "" >> "$OUTPUT"
                fi
            done
        fi
    done
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

# Add prompts/
echo "## Adding prompts/ directory..."

if [ -d "$REPO_ROOT/prompts" ]; then
    for file in "$REPO_ROOT/prompts"/*.md; do
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "# AI Prompt: $filename" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            cat "$file" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            echo "---" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
        fi
    done
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
