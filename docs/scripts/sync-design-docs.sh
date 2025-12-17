#!/bin/bash
# sync-design-docs.sh - Generate design docs pages from source at build time
# This ensures documentation always reflects the actual design docs in the repo

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCS_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$DOCS_DIR")"
DESIGN_DOCS_DIR="$PROJECT_ROOT/design_docs"

GITHUB_BASE="https://github.com/sunholo-data/ailang/blob/main/design_docs"

# Output files
IMPLEMENTED_OUTPUT="$DOCS_DIR/docs/design-docs.md"
ROADMAP_OUTPUT="$DOCS_DIR/docs/roadmap/index.md"

# Extract title from markdown file (first H1 heading)
extract_title() {
    local file="$1"
    local title
    title=$(grep -m1 "^# " "$file" 2>/dev/null | sed 's/^# //')
    if [ -z "$title" ]; then
        # Fallback to filename
        title=$(basename "$file" .md | tr '_-' ' ')
    fi
    echo "$title"
}

# Convert version folder name to display version (v0_3_10 -> v0.3.10)
format_version() {
    echo "$1" | sed 's/_/./g'
}

# Sort versions in reverse semantic order
sort_versions() {
    while read -r version; do
        echo "$version"
    done | sort -t'.' -k1,1Vr -k2,2nr -k3,3nr
}

# Generate implemented design docs page
generate_implemented() {
    cat > "$IMPLEMENTED_OUTPUT" << 'EOF'
---
title: Design Documents
sidebar_position: 100
---

# Design Documents

This page is automatically generated from the [design_docs/implemented](https://github.com/sunholo-data/ailang/tree/main/design_docs/implemented) directory. Each design document describes a feature, bug fix, or architectural decision that has been implemented.

EOF

    # Get all version directories sorted
    local versions
    versions=$(ls -1 "$DESIGN_DOCS_DIR/implemented" 2>/dev/null | sort_versions)

    local total_docs=0
    local version_count=0

    for version_dir in $versions; do
        local version_path="$DESIGN_DOCS_DIR/implemented/$version_dir"
        [ -d "$version_path" ] || continue

        local display_version
        display_version=$(format_version "$version_dir")

        # Get all markdown files in this version
        local docs
        docs=$(ls -1 "$version_path"/*.md 2>/dev/null || true)
        [ -z "$docs" ] && continue

        version_count=$((version_count + 1))

        echo "" >> "$IMPLEMENTED_OUTPUT"
        echo "## $display_version" >> "$IMPLEMENTED_OUTPUT"
        echo "" >> "$IMPLEMENTED_OUTPUT"

        for doc in $docs; do
            local filename
            filename=$(basename "$doc")
            local title
            title=$(extract_title "$doc")
            local github_link="$GITHUB_BASE/implemented/$version_dir/$filename"

            echo "- [$title]($github_link)" >> "$IMPLEMENTED_OUTPUT"
            total_docs=$((total_docs + 1))
        done
    done

    # Add summary at the end
    cat >> "$IMPLEMENTED_OUTPUT" << EOF

---

*Generated at build time. $total_docs design documents across $version_count versions.*
EOF

    echo "Generated $IMPLEMENTED_OUTPUT ($total_docs docs, $version_count versions)"
}

# Generate roadmap page from planned design docs
generate_roadmap() {
    cat > "$ROADMAP_OUTPUT" << 'EOF'
---
title: Roadmap
sidebar_position: 1
---

# AILANG Roadmap

This page is automatically generated from the [design_docs/planned](https://github.com/sunholo-data/ailang/tree/main/design_docs/planned) directory. Each item represents a planned feature or improvement with a detailed design document.

For completed features, see [Design Documents](/docs/design-docs).

EOF

    # Get all planned version directories sorted
    local versions
    versions=$(ls -1 "$DESIGN_DOCS_DIR/planned" 2>/dev/null | sort_versions)

    local total_docs=0
    local version_count=0

    for version_dir in $versions; do
        local version_path="$DESIGN_DOCS_DIR/planned/$version_dir"
        [ -d "$version_path" ] || continue

        local display_version
        display_version=$(format_version "$version_dir")

        # Get all markdown files in this version
        local docs
        docs=$(ls -1 "$version_path"/*.md 2>/dev/null || true)
        [ -z "$docs" ] && continue

        version_count=$((version_count + 1))

        echo "" >> "$ROADMAP_OUTPUT"
        echo "## Planned for $display_version" >> "$ROADMAP_OUTPUT"
        echo "" >> "$ROADMAP_OUTPUT"

        for doc in $docs; do
            local filename
            filename=$(basename "$doc")
            local title
            title=$(extract_title "$doc")
            local github_link="$GITHUB_BASE/planned/$version_dir/$filename"

            echo "- [$title]($github_link)" >> "$ROADMAP_OUTPUT"
            total_docs=$((total_docs + 1))
        done
    done

    # Add vision section
    cat >> "$ROADMAP_OUTPUT" << 'EOF'

## Long-term Vision

AILANG is designed as a deterministic language for autonomous AI code synthesis. The long-term roadmap includes:

- **Structural Reflection** - Typed quasiquotes and AST manipulation
- **Schema Registry** - Machine-readable type and effect definitions
- **Capability Budgets** - Resource-bounded effects
- **Training Data Export** - Execution traces for AI self-training

For the complete vision, see [Why AILANG](/docs/why-ailang) and [Vision](/docs/vision).

EOF

    # Add footer
    cat >> "$ROADMAP_OUTPUT" << EOF
---

*Generated at build time. $total_docs planned features across $version_count upcoming versions.*
EOF

    echo "Generated $ROADMAP_OUTPUT ($total_docs docs, $version_count versions)"
}

# Main
echo "Syncing design docs..."
generate_implemented
generate_roadmap
echo "Done!"
