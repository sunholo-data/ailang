#!/bin/bash
# Sync the active AILANG prompt to docs for website build
# This script is run automatically during docs build

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCS_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$DOCS_ROOT")"

PROMPTS_DIR="$PROJECT_ROOT/prompts"
VERSIONS_JSON="$PROMPTS_DIR/versions.json"
OUTPUT_DIR="$DOCS_ROOT/docs/prompts"

# Get active version from versions.json
ACTIVE_VERSION=$(jq -r '.active' "$VERSIONS_JSON")
if [ -z "$ACTIVE_VERSION" ] || [ "$ACTIVE_VERSION" = "null" ]; then
    echo "Error: Could not determine active prompt version from $VERSIONS_JSON"
    exit 1
fi

# Get the source file for active version
SOURCE_FILE=$(jq -r ".versions[\"$ACTIVE_VERSION\"].file" "$VERSIONS_JSON")
if [ -z "$SOURCE_FILE" ] || [ "$SOURCE_FILE" = "null" ]; then
    echo "Error: Could not find file path for version $ACTIVE_VERSION"
    exit 1
fi

SOURCE_PATH="$PROJECT_ROOT/$SOURCE_FILE"
if [ ! -f "$SOURCE_PATH" ]; then
    echo "Error: Source file not found: $SOURCE_PATH"
    exit 1
fi

# Create output file with frontmatter
OUTPUT_FILE="$OUTPUT_DIR/current.md"

cat > "$OUTPUT_FILE" << EOF
---
title: Current Teaching Prompt
sidebar_position: 1
description: The active AILANG teaching prompt ($ACTIVE_VERSION) - auto-synced from source
---

<!-- AUTO-GENERATED: This file is synced from prompts/$ACTIVE_VERSION.md during build -->
<!-- DO NOT EDIT DIRECTLY - changes will be overwritten -->
<!-- Source: $SOURCE_FILE -->
<!-- Active Version: $ACTIVE_VERSION -->

EOF

# Append the prompt content (skip any existing frontmatter)
if head -1 "$SOURCE_PATH" | grep -q "^---"; then
    # Has frontmatter - skip it
    awk '/^---$/{c++;next}c<2' "$SOURCE_PATH" | tail -n +2 >> "$OUTPUT_FILE"
else
    # No frontmatter - copy as-is
    cat "$SOURCE_PATH" >> "$OUTPUT_FILE"
fi

echo "Synced prompt $ACTIVE_VERSION to $OUTPUT_FILE"
