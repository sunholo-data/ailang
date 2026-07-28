#!/usr/bin/env bash
set -euo pipefail

# Move a design document from planned/ to implemented/
#
# Usage: move_to_implemented.sh <doc-name> <version>
#   doc-name: Name of the doc in planned/ (without .md extension)
#   version:  Target version folder (e.g., v0_3_14)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
DESIGN_DOCS_DIR="$PROJECT_ROOT/design_docs"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

if [ $# -lt 2 ]; then
    echo -e "${RED}✗ Error: Missing required arguments${NC}"
    echo ""
    echo "Usage: move_to_implemented.sh <doc-name> <version>"
    echo ""
    echo "Arguments:"
    echo "  doc-name   Document name in planned/ (without .md)"
    echo "  version    Version folder (e.g., v0_3_14)"
    echo ""
    echo "Examples:"
    echo "  move_to_implemented.sh m-dx1-developer-experience v0_3_10"
    echo "  move_to_implemented.sh reflection-system v0_4_0"
    exit 1
fi

DOC_NAME="$1"
VERSION="$2"

# Find source document (check both planned/ root and version folders)
SOURCE_PATH=""
if [ -f "$DESIGN_DOCS_DIR/planned/${DOC_NAME}.md" ]; then
    SOURCE_PATH="$DESIGN_DOCS_DIR/planned/${DOC_NAME}.md"
elif [ -f "$DESIGN_DOCS_DIR/planned/v0_4_0/${DOC_NAME}.md" ]; then
    SOURCE_PATH="$DESIGN_DOCS_DIR/planned/v0_4_0/${DOC_NAME}.md"
else
    # Search for it
    FOUND=$(find "$DESIGN_DOCS_DIR/planned" -name "${DOC_NAME}.md" 2>/dev/null | head -1)
    if [ -n "$FOUND" ]; then
        SOURCE_PATH="$FOUND"
    else
        echo -e "${RED}✗ Error: Document not found in planned/: ${DOC_NAME}.md${NC}"
        echo ""
        echo "Available docs in planned/:"
        find "$DESIGN_DOCS_DIR/planned" -name "*.md" -type f | sed 's|.*/||' | sort
        exit 1
    fi
fi

# Create target directory
TARGET_DIR="$DESIGN_DOCS_DIR/implemented/$VERSION"
mkdir -p "$TARGET_DIR"

TARGET_PATH="$TARGET_DIR/${DOC_NAME}.md"

# Check if already exists in implemented
if [ -f "$TARGET_PATH" ]; then
    echo -e "${YELLOW}⚠ Warning: Document already exists in implemented/$VERSION/${NC}"
    echo ""
    read -p "Overwrite? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 1
    fi
fi

# Get current date for metadata update
CURRENT_DATE=$(date +%Y-%m-%d)

# Copy file (don't delete from planned yet - let user review first)
cp "$SOURCE_PATH" "$TARGET_PATH"

# Update status in the document
if grep -q "^\*\*Status\*\*:" "$TARGET_PATH"; then
    # Update existing status line
    sed -i.bak "s/^\*\*Status\*\*:.*/\*\*Status\*\*: Implemented/" "$TARGET_PATH"
    rm "${TARGET_PATH}.bak" 2>/dev/null || true
fi

# Update last updated date
if grep -q "^\*\*Last updated\*\*:" "$TARGET_PATH"; then
    sed -i.bak "s/^\*\*Last updated\*\*:.*/\*\*Last updated\*\*: $CURRENT_DATE/" "$TARGET_PATH"
    rm "${TARGET_PATH}.bak" 2>/dev/null || true
fi

# Success message
echo -e "${GREEN}✓ Copied design document to implemented:${NC}"
echo "  From: $SOURCE_PATH"
echo "  To:   $TARGET_PATH"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Review the document at $TARGET_PATH"
echo "  2. Add implementation report section:"
echo "     - What was actually built"
echo "     - Code locations (internal/...)"
echo "     - Test coverage metrics"
echo "     - Known limitations"
echo "  3. Update design_docs/README.md with version history"
echo "  4. Commit changes: git add $TARGET_PATH design_docs/README.md"
echo "  5. AFTER committing, delete original:"
echo "     git rm $SOURCE_PATH"
echo ""
echo -e "${GREEN}Template for implementation report:${NC}"
echo ""
cat <<'REPORT'
## Implementation Report

**Completed**: [Date]
**Version**: [Version number from CHANGELOG]

### What Was Built

[Summary of what was actually implemented vs planned]

### Code Locations

**New files:**
- `internal/path/file.go` (XXX LOC) - [Purpose]

**Modified files:**
- `internal/path/existing.go` (+XX/-YY LOC) - [Changes]

### Test Coverage

- Unit tests: XX passing
- Integration tests: YY passing
- Coverage: ZZ%
- Test files: `internal/path/*_test.go`

### Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| [Metric 1] | X | Y | +Z% |
| [Metric 2] | X | Y | +Z% |

### Known Limitations

- [Limitation 1] - [Why/when to address]
- [Limitation 2] - [Why/when to address]

### Future Work

[Features deferred to later versions]

REPORT
