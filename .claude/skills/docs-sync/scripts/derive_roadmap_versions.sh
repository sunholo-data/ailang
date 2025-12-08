#!/bin/bash
# derive_roadmap_versions.sh - Derive target versions from design doc folder structure
# Design doc folder = source of truth for target version
# Usage: ./derive_roadmap_versions.sh [--json] [--check] [--full]
#
# --json: Output as JSON for programmatic use
# --check: Check website pages match derived versions (exit 1 if mismatches)
# --full: Include implemented features (complete feature lifecycle)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

JSON_OUTPUT=false
CHECK_MODE=false
FULL_MODE=false

for arg in "$@"; do
    case $arg in
        --json) JSON_OUTPUT=true ;;
        --check) CHECK_MODE=true ;;
        --full) FULL_MODE=true ;;
    esac
done

# Get current version from git tag
CURRENT_VERSION=$(cd "$PROJECT_ROOT" && git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

echo "=== Roadmap Version Derivation ===" >&2
echo "Current release: $CURRENT_VERSION" >&2
echo "Design doc folders → Target versions" >&2
echo "" >&2

# Function to convert folder name to version (v0_6_0 -> v0.6.0)
folder_to_version() {
    echo "$1" | sed 's/_/./g'
}

# Function to compare versions (returns 0 if v1 <= v2)
version_lte() {
    [ "$1" = "$(echo -e "$1\n$2" | sort -V | head -n1)" ]
}

# Collect all features
MISMATCHES=0

if $JSON_OUTPUT; then
    echo "{"
    echo '  "current_version": "'$CURRENT_VERSION'",'
    echo '  "planned": ['
fi

# ============================================
# PLANNED FEATURES (design_docs/planned/)
# ============================================
echo "--- Planned Features ---" >&2

FIRST=true
for version_dir in "$PROJECT_ROOT/design_docs/planned"/v0_*; do
    if [[ -d "$version_dir" ]]; then
        folder_name=$(basename "$version_dir")
        target_version=$(folder_to_version "$folder_name")

        # Check if this version is implemented (folder version <= current version)
        if version_lte "$target_version" "$CURRENT_VERSION"; then
            status="OVERDUE"
            status_emoji="⚠️"
        else
            status="PLANNED"
            status_emoji="📋"
        fi

        # List design docs in this folder
        for doc in "$version_dir"/*.md; do
            if [[ -f "$doc" ]]; then
                doc_name=$(basename "$doc" .md)

                # Skip sprint plans and non-feature docs
                if [[ "$doc_name" == *"sprint"* ]] || [[ "$doc_name" == *"SPRINT"* ]]; then
                    continue
                fi

                # GitHub URL for the design doc
                rel_path="${doc#$PROJECT_ROOT/}"
                github_url="https://github.com/sunholo-data/ailang/blob/main/$rel_path"

                if $JSON_OUTPUT; then
                    if ! $FIRST; then echo ","; fi
                    FIRST=false
                    echo -n '    {"name": "'$doc_name'", "target": "'$target_version'", "status": "'$status'", "github_url": "'$github_url'"}'
                else
                    echo "$status_emoji $doc_name → $target_version ($status)" >&2
                fi
            fi
        done
    fi
done

if $JSON_OUTPUT; then
    echo ""
    echo "  ],"
fi

echo "" >&2

# ============================================
# IMPLEMENTED FEATURES (design_docs/implemented/)
# ============================================
if $FULL_MODE; then
    echo "--- Implemented Features ---" >&2

    if $JSON_OUTPUT; then
        echo '  "implemented": ['
    fi

    FIRST=true
    for version_dir in "$PROJECT_ROOT/design_docs/implemented"/v0_*; do
        if [[ -d "$version_dir" ]]; then
            folder_name=$(basename "$version_dir")
            impl_version=$(folder_to_version "$folder_name")

            # List design docs in this folder
            for doc in "$version_dir"/*.md; do
                if [[ -f "$doc" ]]; then
                    doc_name=$(basename "$doc" .md)

                    # Skip sprint plans
                    if [[ "$doc_name" == *"sprint"* ]] || [[ "$doc_name" == *"SPRINT"* ]]; then
                        continue
                    fi

                    # GitHub URL for the design doc
                    rel_path="${doc#$PROJECT_ROOT/}"
                    github_url="https://github.com/sunholo-data/ailang/blob/main/$rel_path"

                    if $JSON_OUTPUT; then
                        if ! $FIRST; then echo ","; fi
                        FIRST=false
                        echo -n '    {"name": "'$doc_name'", "version": "'$impl_version'", "github_url": "'$github_url'"}'
                    else
                        echo "✅ $doc_name → $impl_version (IMPLEMENTED)" >&2
                    fi
                fi
            done
        fi
    done

    if $JSON_OUTPUT; then
        echo ""
        echo "  ],"
    fi

    echo "" >&2
fi

# ============================================
# SUMMARY STATISTICS
# ============================================
if $JSON_OUTPUT; then
    # Count planned vs implemented
    planned_count=$(find "$PROJECT_ROOT/design_docs/planned" -name "*.md" 2>/dev/null | grep -v sprint | wc -l | tr -d ' ')
    impl_count=$(find "$PROJECT_ROOT/design_docs/implemented" -name "*.md" 2>/dev/null | grep -v sprint | wc -l | tr -d ' ')

    echo '  "stats": {'
    echo '    "planned_docs": '$planned_count','
    echo '    "implemented_docs": '$impl_count
    echo '  }'
    echo "}"
else
    # Print summary
    planned_count=$(find "$PROJECT_ROOT/design_docs/planned" -name "*.md" 2>/dev/null | grep -v -i sprint | wc -l | tr -d ' ')
    impl_count=$(find "$PROJECT_ROOT/design_docs/implemented" -name "*.md" 2>/dev/null | grep -v -i sprint | wc -l | tr -d ' ')
    echo "=== Summary ===" >&2
    echo "Planned design docs: $planned_count" >&2
    echo "Implemented design docs: $impl_count" >&2
fi

echo "" >&2

# ============================================
# CHECK MODE: Verify website consistency
# ============================================
if $CHECK_MODE; then
    echo "=== Checking Website Consistency ===" >&2
    echo "" >&2

    # Check roadmap pages
    echo "Checking roadmap pages..." >&2
    ROADMAP_DIR="$PROJECT_ROOT/docs/docs/roadmap"
    if [[ -d "$ROADMAP_DIR" ]]; then
        for file in "$ROADMAP_DIR"/*.md "$ROADMAP_DIR"/*.mdx; do
            if [[ -f "$file" ]]; then
                filename=$(basename "$file")

                # Skip index files
                if [[ "$filename" == "index.md" ]]; then continue; fi

                # Extract claimed version from PLANNED FOR banner
                claimed_version=$(grep -o "PLANNED FOR v[0-9.]*" "$file" 2>/dev/null | head -1 | sed 's/PLANNED FOR //')

                # Extract design doc link to get derived version
                design_doc_link=$(grep -o "design_docs/planned/v[0-9_]*/[^)]*" "$file" 2>/dev/null | head -1)

                if [[ -n "$design_doc_link" ]]; then
                    # Extract version from design doc path
                    doc_folder=$(echo "$design_doc_link" | grep -o "v[0-9_]*" | head -1)
                    derived_version=$(folder_to_version "$doc_folder")

                    if [[ -n "$claimed_version" && -n "$derived_version" ]]; then
                        if [[ "$claimed_version" == "$derived_version" ]]; then
                            echo "✅ $filename: $claimed_version (matches design doc folder)" >&2
                        else
                            echo "❌ $filename: claims $claimed_version but design doc is in $derived_version" >&2
                            MISMATCHES=$((MISMATCHES + 1))
                        fi
                    fi

                    # Check if GitHub link exists
                    github_link=$(grep -o "github.com/sunholo-data/ailang/blob/main/design_docs" "$file" 2>/dev/null | head -1)
                    if [[ -z "$github_link" ]]; then
                        echo "⚠️  $filename: missing GitHub link to design doc" >&2
                    else
                        echo "✅ $filename: has GitHub link to design doc" >&2
                    fi
                else
                    echo "⚠️  $filename: no design doc link found" >&2
                fi
            fi
        done
    fi

    echo "" >&2

    # Check reference pages for implemented features
    echo "Checking reference pages for GitHub links..." >&2
    REFERENCE_DIR="$PROJECT_ROOT/docs/docs/reference"
    if [[ -d "$REFERENCE_DIR" ]]; then
        for file in "$REFERENCE_DIR"/*.md "$REFERENCE_DIR"/*.mdx; do
            if [[ -f "$file" ]]; then
                filename=$(basename "$file")

                # Check if page links to any design docs
                design_link=$(grep -o "design_docs/implemented/v[0-9_]*/[^)]*" "$file" 2>/dev/null | head -1)
                if [[ -n "$design_link" ]]; then
                    echo "✅ $filename: links to implemented design doc" >&2
                fi
            fi
        done
    fi

    echo "" >&2
    if [[ $MISMATCHES -gt 0 ]]; then
        echo "❌ Found $MISMATCHES version mismatches!" >&2
        echo "Fix: Update 'PLANNED FOR vX.X.X' banners to match design doc folder versions" >&2
        exit 1
    else
        echo "✅ All roadmap pages consistent with design doc folders" >&2
    fi
fi

echo "=== Done ===" >&2
