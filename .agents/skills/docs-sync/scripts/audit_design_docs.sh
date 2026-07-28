#!/bin/bash
# audit_design_docs.sh - Compare planned vs implemented design docs
# Usage: ./audit_design_docs.sh [--json]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

JSON_OUTPUT=false
if [[ "$1" == "--json" ]]; then
    JSON_OUTPUT=true
fi

echo "=== Design Docs Audit ==="
echo "Project root: $PROJECT_ROOT"
echo ""

# Count design docs
PLANNED_COUNT=$(find "$PROJECT_ROOT/design_docs/planned" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
IMPLEMENTED_COUNT=$(find "$PROJECT_ROOT/design_docs/implemented" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')

echo "Design Docs Summary:"
echo "  Planned: $PLANNED_COUNT documents"
echo "  Implemented: $IMPLEMENTED_COUNT documents"
echo ""

# List planned version folders
echo "Planned Features by Version:"
for dir in "$PROJECT_ROOT/design_docs/planned"/v0_*; do
    if [[ -d "$dir" ]]; then
        version=$(basename "$dir")
        count=$(find "$dir" -name "*.md" | wc -l | tr -d ' ')
        echo "  $version: $count docs"
    fi
done
echo ""

# List implemented version folders
echo "Implemented Features by Version:"
for dir in "$PROJECT_ROOT/design_docs/implemented"/v0_*; do
    if [[ -d "$dir" ]]; then
        version=$(basename "$dir")
        count=$(find "$dir" -name "*.md" | wc -l | tr -d ' ')
        echo "  $version: $count docs"
    fi
done
echo ""

# Check for website pages that reference planned features
echo "Website Pages Referencing Planned Features:"
ARCHITECTURE_DIR="$PROJECT_ROOT/docs/docs/architecture"
if [[ -d "$ARCHITECTURE_DIR" ]]; then
    for file in "$ARCHITECTURE_DIR"/*.mdx "$ARCHITECTURE_DIR"/*.md; do
        if [[ -f "$file" ]]; then
            basename "$file"
        fi
    done
fi
echo ""

# Check for design docs mentioned in website but not implemented
echo "=== Feature Status Check ==="

# Key features to check (bash 3 compatible - no associative arrays)
check_feature() {
    local keyword="$1"
    local feature="$2"

    # Check if mentioned in website
    website_mentions=$(grep -r "$keyword" "$PROJECT_ROOT/docs/docs" 2>/dev/null | wc -l | tr -d ' ')

    # Check if implemented (in Go code)
    code_mentions=$(grep -r "$keyword" "$PROJECT_ROOT/internal" "$PROJECT_ROOT/cmd" 2>/dev/null | wc -l | tr -d ' ')

    if [[ "$website_mentions" -gt 0 && "$code_mentions" -eq 0 ]]; then
        echo "  WARNING: '$feature' mentioned $website_mentions times in docs but NOT in code"
    elif [[ "$website_mentions" -gt 0 && "$code_mentions" -gt 0 ]]; then
        echo "  OK: '$feature' documented and implemented"
    fi
}

check_feature "SharedMem" "Shared Semantic State"
check_feature "--profile" "Execution Profiles"
check_feature "normalize" "Deterministic Tooling"
check_feature "suggest-imports" "Import Suggestion"
check_feature "reflectType" "Runtime Reflection"

echo ""

# Check website roadmap version consistency with design doc folders
echo "=== Version Consistency Check ==="
echo "Design doc folder = source of truth for target version"
echo ""

ROADMAP_DIR="$PROJECT_ROOT/docs/docs/roadmap"
if [[ -d "$ROADMAP_DIR" ]]; then
    for file in "$ROADMAP_DIR"/*.md "$ROADMAP_DIR"/*.mdx; do
        if [[ -f "$file" ]]; then
            filename=$(basename "$file")

            # Extract version from PLANNED FOR banner
            claimed_version=$(grep -o "PLANNED FOR v[0-9.]*" "$file" 2>/dev/null | head -1 | sed 's/PLANNED FOR //')

            # Extract design doc link
            design_doc_link=$(grep -o "design_docs/planned/v[0-9_]*/[^)]*" "$file" 2>/dev/null | head -1)

            if [[ -n "$design_doc_link" ]]; then
                # Extract version from design doc path (v0_6_0 -> v0.6.0)
                doc_version=$(echo "$design_doc_link" | grep -o "v[0-9_]*" | head -1 | sed 's/_/./g')

                if [[ -n "$claimed_version" && -n "$doc_version" ]]; then
                    if [[ "$claimed_version" == "$doc_version" ]]; then
                        echo "  OK: $filename - version $claimed_version matches design doc folder"
                    else
                        echo "  MISMATCH: $filename claims $claimed_version but design doc is in $doc_version"
                    fi
                fi
            fi
        fi
    done
fi

echo ""
echo "=== Audit Complete ==="
