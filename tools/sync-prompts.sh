#!/usr/bin/env bash
# Sync prompts/ directory to docs/prompts/ for website

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PROMPTS_SRC="$REPO_ROOT/prompts"
PROMPTS_DEST="$REPO_ROOT/docs/prompts"

echo "=== Syncing prompts/ to docs/prompts/ ==="

# Create destination directory if it doesn't exist
mkdir -p "$PROMPTS_DEST"

# Copy all .md files from prompts/ to docs/prompts/ and add Jekyll front matter
for file in "$PROMPTS_SRC"/*.md; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        basename_no_ext=$(basename "$file" .md)
        dest_file="$PROMPTS_DEST/$filename"

        echo "  Processing $filename"

        # Check if file already has front matter
        if head -n 1 "$file" | grep -q "^---"; then
            # Already has front matter, just copy
            cp "$file" "$dest_file"
        else
            # Add front matter for Jekyll
            {
                echo "---"
                echo "layout: page"
                echo "title: AI Prompt - $basename_no_ext"
                echo "parent: AI Prompts"
                echo "nav_order: 1"
                echo "---"
                echo ""
                cat "$file"
            } > "$dest_file"
        fi
    fi
done

# Create parent index if it doesn't exist
index_file="$PROMPTS_DEST/index.md"
if [ ! -f "$index_file" ]; then
    echo "  Creating prompts index"
    cat > "$index_file" << 'EOF'
---
layout: page
title: AI Prompts
nav_order: 5
has_children: true
---

# AI Prompts for AILANG

These prompts teach AI models how to write correct AILANG code.

## Available Prompts

- **v0.3.0** - Current version (recommended)
- **v0.2.0** - Previous version
- **python** - Python comparison guide
EOF
fi

echo "✓ Prompts synced successfully"
echo ""
echo "Files in docs/prompts/:"
ls -1 "$PROMPTS_DEST"
