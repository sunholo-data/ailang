#!/usr/bin/env bash
set -euo pipefail

# Fix module path mismatches (MOD010 errors)
# Makes module declarations match canonical file paths

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
    echo -e "${YELLOW}🔍 DRY RUN MODE - No files will be modified${NC}"
    echo ""
fi

# Counters
FIXED=0
UNCHANGED=0
SKIPPED=0

echo "Fixing module path mismatches..."
echo ""

# Fix examples/tests/ files (23 files)
# Pattern: module examples/test_* -> module examples/tests/test_*
for file in "$PROJECT_ROOT"/examples/tests/*.ail; do
    if [ ! -f "$file" ]; then
        continue
    fi

    filename=$(basename "$file" .ail)

    # Check if module declaration exists and needs fixing
    if grep -q "^module examples/$filename" "$file"; then
        echo -e "${BLUE}📝 $file${NC}"
        echo "   Current: module examples/$filename"
        echo "   Fixed:   module examples/tests/$filename"

        if [ "$DRY_RUN" = false ]; then
            # Idempotent fix: only replace if not already correct
            sed -i.bak "s|^module examples/$filename|module examples/tests/$filename|g" "$file"
            rm -f "${file}.bak"
            ((FIXED++))
        else
            ((FIXED++))
        fi
    elif grep -q "^module examples/tests/$filename" "$file"; then
        # Already correct
        ((UNCHANGED++))
    else
        echo -e "${YELLOW}⚠️  Skipped: $file (no matching pattern)${NC}"
        ((SKIPPED++))
    fi
done

# Fix examples/tests/demos/ files
for file in "$PROJECT_ROOT"/examples/tests/demos/*.ail; do
    if [ ! -f "$file" ]; then
        continue
    fi

    filename=$(basename "$file" .ail)

    if grep -q "^module examples/demos/$filename" "$file"; then
        echo -e "${BLUE}📝 $file${NC}"
        echo "   Current: module examples/demos/$filename"
        echo "   Fixed:   module examples/tests/demos/$filename"

        if [ "$DRY_RUN" = false ]; then
            sed -i.bak "s|^module examples/demos/$filename|module examples/tests/demos/$filename|g" "$file"
            rm -f "${file}.bak"
            ((FIXED++))
        else
            ((FIXED++))
        fi
    elif grep -q "^module examples/tests/demos/$filename" "$file"; then
        ((UNCHANGED++))
    else
        echo -e "${YELLOW}⚠️  Skipped: $file (no matching pattern)${NC}"
        ((SKIPPED++))
    fi
done

# Fix examples/runnable/ files that declare wrong path (2 files: micro_clock_measure, micro_net_fetch)
for file in "$PROJECT_ROOT"/examples/tests/micro_clock_measure.ail "$PROJECT_ROOT"/examples/tests/micro_net_fetch.ail; do
    if [ ! -f "$file" ]; then
        continue
    fi

    filename=$(basename "$file" .ail)

    if grep -q "^module examples/runnable/$filename" "$file"; then
        echo -e "${BLUE}📝 $file${NC}"
        echo "   Current: module examples/runnable/$filename"
        echo "   Fixed:   module examples/tests/$filename"

        if [ "$DRY_RUN" = false ]; then
            sed -i.bak "s|^module examples/runnable/$filename|module examples/tests/$filename|g" "$file"
            rm -f "${file}.bak"
            ((FIXED++))
        else
            ((FIXED++))
        fi
    elif grep -q "^module examples/tests/$filename" "$file"; then
        ((UNCHANGED++))
    fi
done

# Fix examples/snippets/ files (5 files)
for file in "$PROJECT_ROOT"/examples/snippets/{numeric_conversion,option_demo,stdlib_demo,stdlib_demo_simple}.ail; do
    if [ ! -f "$file" ]; then
        continue
    fi

    filename=$(basename "$file" .ail)

    if grep -q "^module examples/$filename" "$file"; then
        echo -e "${BLUE}📝 $file${NC}"
        echo "   Current: module examples/$filename"
        echo "   Fixed:   module examples/snippets/$filename"

        if [ "$DRY_RUN" = false ]; then
            sed -i.bak "s|^module examples/$filename|module examples/snippets/$filename|g" "$file"
            rm -f "${file}.bak"
            ((FIXED++))
        else
            ((FIXED++))
        fi
    elif grep -q "^module examples/snippets/$filename" "$file"; then
        ((UNCHANGED++))
    else
        echo -e "${YELLOW}⚠️  Skipped: $file (no matching pattern)${NC}"
        ((SKIPPED++))
    fi
done

# Fix examples/snippets/v3_3/ files (2 files)
for file in "$PROJECT_ROOT"/examples/snippets/v3_3/{imports,imports_basic}.ail; do
    if [ ! -f "$file" ]; then
        continue
    fi

    filename=$(basename "$file" .ail)

    if grep -q "^module examples/v3_3/$filename" "$file"; then
        echo -e "${BLUE}📝 $file${NC}"
        echo "   Current: module examples/v3_3/$filename"
        echo "   Fixed:   module examples/snippets/v3_3/$filename"

        if [ "$DRY_RUN" = false ]; then
            sed -i.bak "s|^module examples/v3_3/$filename|module examples/snippets/v3_3/$filename|g" "$file"
            rm -f "${file}.bak"
            ((FIXED++))
        else
            ((FIXED++))
        fi
    elif grep -q "^module examples/snippets/v3_3/$filename" "$file"; then
        ((UNCHANGED++))
    else
        echo -e "${YELLOW}⚠️  Skipped: $file (no matching pattern)${NC}"
        ((SKIPPED++))
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}Summary:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${GREEN}Fixed:${NC}     $FIXED files"
echo -e "  ${BLUE}Unchanged:${NC} $UNCHANGED files"
echo -e "  ${YELLOW}Skipped:${NC}   $SKIPPED files"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}🔍 This was a dry run. Run without --dry-run to apply changes.${NC}"
else
    echo -e "${GREEN}✓ Module paths fixed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Run: make verify-examples"
    echo "  2. Check: git diff examples/"
    echo "  3. Commit: git add examples/ && git commit -m 'Fix module path mismatches (MOD010)'"
fi
