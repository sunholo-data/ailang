#!/usr/bin/env bash
set -euo pipefail

# Create a new design document in design_docs/planned/
#
# Usage: create_planned_doc.sh <doc-name> [version]
#   doc-name: Lowercase with hyphens (e.g., m-dx2-feature-name)
#   version:  Optional version folder (e.g., v0_4_0)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
DESIGN_DOCS_DIR="$PROJECT_ROOT/design_docs"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get current version from CHANGELOG.md
get_current_version() {
    local CHANGELOG="$PROJECT_ROOT/CHANGELOG.md"
    if [ -f "$CHANGELOG" ]; then
        grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' "$CHANGELOG" | head -1
    else
        echo "unknown"
    fi
}

# Compute next patch version (e.g., v0.5.6 -> v0_5_7)
get_next_version_folder() {
    local current="$1"
    # Extract major.minor.patch
    local version="${current#v}"  # Remove 'v' prefix
    local major=$(echo "$version" | cut -d. -f1)
    local minor=$(echo "$version" | cut -d. -f2)
    local patch=$(echo "$version" | cut -d. -f3)
    # Increment patch
    local next_patch=$((patch + 1))
    # Return folder format (v0_5_7)
    echo "v${major}_${minor}_${next_patch}"
}

CURRENT_VERSION=$(get_current_version)
NEXT_VERSION_FOLDER=$(get_next_version_folder "$CURRENT_VERSION")

if [ $# -lt 1 ]; then
    echo -e "${RED}✗ Error: Missing required argument${NC}"
    echo ""
    echo -e "${CYAN}Current AILANG version: $CURRENT_VERSION${NC}"
    echo -e "${CYAN}Suggested next version: $NEXT_VERSION_FOLDER${NC}"
    echo ""
    echo "Usage: create_planned_doc.sh <doc-name> [version]"
    echo ""
    echo "Arguments:"
    echo "  doc-name   Document name (lowercase-with-hyphens, e.g., m-dx2-better-errors)"
    echo "  version    Optional version folder (e.g., $NEXT_VERSION_FOLDER)"
    echo ""
    echo "Examples:"
    echo "  create_planned_doc.sh m-dx2-better-errors"
    echo "  create_planned_doc.sh reflection-system $NEXT_VERSION_FOLDER"
    exit 1
fi

DOC_NAME="$1"
VERSION="${2:-}"

# Convert doc-name to search query (replace hyphens with spaces, remove m-prefix)
search_query() {
    local name="$1"
    # Remove common prefixes like m-dx1, m-perf2, etc.
    local cleaned="${name#m-}"
    cleaned="${cleaned#[a-z]*[0-9]-}"
    # Replace hyphens with spaces
    echo "${cleaned//-/ }"
}

SEARCH_QUERY=$(search_query "$DOC_NAME")

# Check for related design docs before creating
# Run both SimHash (instant) and Neural (better quality), combine unique results
echo -e "${CYAN}🔍 Searching for related design docs...${NC}"
echo ""

# Check if ailang is available
if command -v ailang &> /dev/null || [ -x "$PROJECT_ROOT/bin/ailang" ]; then
    AILANG_CMD="${PROJECT_ROOT}/bin/ailang"
    if ! [ -x "$AILANG_CMD" ]; then
        AILANG_CMD="ailang"
    fi

    # Function to merge and deduplicate results (neural results take priority)
    merge_results() {
        local simhash="$1"
        local neural="$2"
        # Combine, dedupe by path. Neural listed first so its scores take priority.
        # Format: "1. path/to/file.md (0.85)" - $2 is the path
        { echo "$neural"; echo "$simhash"; } | grep -E "^[0-9]+\." | \
            awk '{ path = $2; if (path && !seen[path]++) print }' | head -5
    }

    # --- IMPLEMENTED DOCS ---
    echo -e "${YELLOW}Implemented docs matching \"$SEARCH_QUERY\":${NC}"

    # SimHash first (instant feedback)
    echo -e "  ${CYAN}[SimHash - instant]${NC}"
    IMPL_SIMHASH=$("$AILANG_CMD" docs search --stream implemented --limit 5 "$SEARCH_QUERY" 2>/dev/null | grep -E "^\d+\." || echo "")
    if [ -n "$IMPL_SIMHASH" ]; then
        echo "$IMPL_SIMHASH" | head -3 | sed 's/^/  /'
    else
        echo "    (none found)"
    fi

    # Neural search (better quality)
    echo -e "  ${CYAN}[Neural - semantic matching]${NC}"
    IMPL_NEURAL=$("$AILANG_CMD" docs search --stream implemented --neural --limit 5 "$SEARCH_QUERY" 2>/dev/null | grep -E "^\d+\." || echo "")
    if [ -n "$IMPL_NEURAL" ]; then
        echo "$IMPL_NEURAL" | head -3 | sed 's/^/  /'
    else
        echo "    (none found)"
    fi

    # Merge for template (neural preferred)
    IMPLEMENTED=$(merge_results "$IMPL_SIMHASH" "$IMPL_NEURAL")
    [ -z "$IMPLEMENTED" ] && IMPLEMENTED="  (none found)"
    echo ""

    # --- PLANNED DOCS ---
    echo -e "${YELLOW}Planned docs matching \"$SEARCH_QUERY\":${NC}"

    # SimHash first (instant feedback)
    echo -e "  ${CYAN}[SimHash - instant]${NC}"
    PLAN_SIMHASH=$("$AILANG_CMD" docs search --stream planned --limit 5 "$SEARCH_QUERY" 2>/dev/null | grep -E "^\d+\." || echo "")
    if [ -n "$PLAN_SIMHASH" ]; then
        echo "$PLAN_SIMHASH" | head -3 | sed 's/^/  /'
    else
        echo "    (none found)"
    fi

    # Neural search (better quality)
    echo -e "  ${CYAN}[Neural - semantic matching]${NC}"
    PLAN_NEURAL=$("$AILANG_CMD" docs search --stream planned --neural --limit 5 "$SEARCH_QUERY" 2>/dev/null | grep -E "^\d+\." || echo "")
    if [ -n "$PLAN_NEURAL" ]; then
        echo "$PLAN_NEURAL" | head -3 | sed 's/^/  /'
    else
        echo "    (none found)"
    fi

    # Merge for template (neural preferred)
    PLANNED=$(merge_results "$PLAN_SIMHASH" "$PLAN_NEURAL")
    [ -z "$PLANNED" ] && PLANNED="  (none found)"
    echo ""

    # Show info if matches found (no confirmation required - proceed automatically)
    if [ "$IMPLEMENTED" != "  (none found)" ] || [ "$PLANNED" != "  (none found)" ]; then
        echo -e "${CYAN}ℹ Related docs found above - review them after creation if needed.${NC}"
        echo ""
    fi
else
    echo -e "${YELLOW}⚠ ailang not found - skipping related doc search${NC}"
    echo "  Build with: make build"
    echo ""
fi

# Determine target directory
if [ -n "$VERSION" ]; then
    TARGET_DIR="$DESIGN_DOCS_DIR/planned/$VERSION"
    mkdir -p "$TARGET_DIR"
else
    TARGET_DIR="$DESIGN_DOCS_DIR/planned"
fi

DOC_PATH="$TARGET_DIR/${DOC_NAME}.md"

# Check if document already exists
if [ -f "$DOC_PATH" ]; then
    echo -e "${YELLOW}⚠ Warning: Document already exists at $DOC_PATH${NC}"
    echo ""
    read -p "Overwrite? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 1
    fi
fi

# Get current date
CURRENT_DATE=$(date +%Y-%m-%d)

# Create document from template
cat > "$DOC_PATH" <<'EOF'
# [Feature Name]

**Status**: Planned
**Target**: [Version, e.g., v0.4.0]
**Priority**: [P0/P1/P2 - High/Medium/Low]
**Estimated**: [Time estimate, e.g., 2 days]
**Dependencies**: [None or list other features]

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | [+1/0/−1] | [e.g., "Enables reproducible traces"] |
| A2: Replayability | [+1/0/−1] | [e.g., "No impact on traces"] |
| A3: Effect Legibility | [+1/0/−1] | [e.g., "Makes IO effects explicit"] |
| A4: Explicit Authority | [+1/0/−1] | [e.g., "Enforces capability constraints"] |
| A5: Bounded Verification | [+1/0/−1] | [e.g., "Enables local type checks"] |
| A6: Safe Concurrency | [+1/0/−1] | [e.g., "No concurrency changes"] |
| A7: Machines First | [+1/0/−1] | [e.g., "Reduces AI token cost"] |
| A8: Minimal Syntax | [+1/0/−1] | [e.g., "No new syntax required"] |
| A9: Cost Visibility | [+1/0/−1] | [e.g., "Resource costs remain visible"] |
| A10: Composability | [+1/0/−1] | [e.g., "Composes with existing effects"] |
| A11: Structured Failure | [+1/0/−1] | [e.g., "Errors remain typed"] |
| A12: System Boundary | [+1/0/−1] | [e.g., "Boundary crossings explicit"] |

**Net Score: [Total]** → **Decision: [Move forward / Reject / Redesign]**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [ ] A1 (Determinism): No implicit nondeterminism introduced
- [ ] A3 (Effects): No hidden side effects
- [ ] A4 (Authority): No ambient access granted
- [ ] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

[What problem does this solve? Why is it needed?]

**Current State:**
- [Describe current pain points]
- [Include metrics if available]

**Impact:**
- [Who is affected?]
- [How significant is the problem?]

## Goals

**Primary Goal:** [Main objective in one sentence]

**Success Metrics:**
- [Measurable outcome 1]
- [Measurable outcome 2]
- [Measurable outcome 3]

## Solution Design

### Overview

[High-level description of the solution]

### Architecture

[Describe the technical approach]

**Components:**
1. **Component 1**: [Description]
2. **Component 2**: [Description]
3. **Component 3**: [Description]

### Implementation Plan

**Phase 1: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Phase 2: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Phase 3: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

### Files to Modify/Create

**New files:**
- `path/to/new_file.go` - [Purpose, ~XXX LOC]

**Modified files:**
- `path/to/existing_file.go` - [Changes needed, ~XXX LOC]

## Examples

### Example 1: [Use Case]

**Before:**
```
[Code or workflow before the change]
```

**After:**
```
[Code or workflow after the change]
```

### Example 2: [Use Case]

[Additional examples as needed]

## Success Criteria

- [ ] Criterion 1 (with acceptance test)
- [ ] Criterion 2 (with acceptance test)
- [ ] Criterion 3 (with acceptance test)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- [What to test]

**Integration tests:**
- [What to test]

**Manual testing:**
- [What to verify manually]

## Non-Goals

**Not in this feature:**
- [Thing 1] - [Why deferred]
- [Thing 2] - [Why out of scope]

## Timeline

**Week 1** (X hours):
- Phase 1 implementation

**Week 2** (X hours):
- Phase 2 implementation
- Testing

**Week 3** (X hours):
- Phase 3 implementation
- Documentation
- Release

**Total: ~X hours across Y weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| [Risk 1] | [High/Med/Low] | [How to address] |
| [Risk 2] | [High/Med/Low] | [How to address] |

## Related Documents

<!-- Auto-populated by Ollama neural search on "SEARCH_QUERY_PLACEHOLDER" -->

**Implemented (may inform design):**
RELATED_IMPLEMENTED_PLACEHOLDER

**Planned (check for overlap):**
RELATED_PLANNED_PLACEHOLDER

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [Link to related design docs]
- [Link to issues or discussions]

## Future Work

[Features that build on this but are out of scope for now]

---

**Document created**: CURRENT_DATE
**Last updated**: CURRENT_DATE
EOF

# Replace CURRENT_DATE with actual date
sed -i.bak "s/CURRENT_DATE/$CURRENT_DATE/g" "$DOC_PATH"
rm "${DOC_PATH}.bak"

# Replace search query placeholder
sed -i.bak "s/SEARCH_QUERY_PLACEHOLDER/$SEARCH_QUERY/g" "$DOC_PATH"
rm "${DOC_PATH}.bak"

# Format search results for markdown
format_results() {
    local results="$1"
    if [ "$results" = "  (none found)" ] || [ -z "$results" ]; then
        echo "- (none found)"
    else
        # Convert numbered list to markdown links
        echo "$results" | head -3 | while read -r line; do
            # Extract path and score from format "1. path/to/doc.md (0.85)"
            local path=$(echo "$line" | sed 's/^[0-9]*\. //' | sed 's/ ([0-9.]*)//')
            local score=$(echo "$line" | grep -oE '\([0-9.]+\)' || echo "")
            if [ -n "$path" ]; then
                echo "- [$path]($path) $score"
            fi
        done
    fi
}

# Replace related docs placeholders
IMPL_FORMATTED=$(format_results "$IMPLEMENTED")
PLAN_FORMATTED=$(format_results "$PLANNED")

# Use perl for multi-line replacement (more portable than sed)
perl -i -pe "s|RELATED_IMPLEMENTED_PLACEHOLDER|$IMPL_FORMATTED|g" "$DOC_PATH" 2>/dev/null || \
    sed -i.bak "s|RELATED_IMPLEMENTED_PLACEHOLDER|$IMPL_FORMATTED|g" "$DOC_PATH" && rm -f "${DOC_PATH}.bak"

perl -i -pe "s|RELATED_PLANNED_PLACEHOLDER|$PLAN_FORMATTED|g" "$DOC_PATH" 2>/dev/null || \
    sed -i.bak "s|RELATED_PLANNED_PLACEHOLDER|$PLAN_FORMATTED|g" "$DOC_PATH" && rm -f "${DOC_PATH}.bak"

# Convert to relative path for coordinator marker
RELATIVE_PATH="${DOC_PATH#$PROJECT_ROOT/}"

# Success message
echo -e "${GREEN}✓ Created design document:${NC}"
echo "  $DOC_PATH"
echo ""
echo -e "${CYAN}Version context:${NC}"
echo "  Current AILANG: $CURRENT_VERSION"
echo "  Next version:   $NEXT_VERSION_FOLDER"
if [ -n "$VERSION" ]; then
    echo "  Doc target:     $VERSION"
fi
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "  1. Edit $DOC_PATH to fill in the template"
echo "  2. Replace [placeholders] with actual content"
echo "  3. Commit when ready: git add $DOC_PATH"
echo ""
echo -e "${YELLOW}Pro tips:${NC}"
echo "  - Complete the Axiom Compliance section (score all 12 axioms)"
echo "  - Hard violations on A1/A3/A4/A7 = automatic rejection"
echo "  - Net axiom score must be ≥ +2 to proceed"
echo "  - Use M-XXX naming for milestone features"
echo "  - Include concrete examples and metrics"
echo "  - Keep estimates realistic (2x your initial guess)"
echo ""
# Output coordinator markers (deterministic - script knows exactly what was created)
echo "---"
echo "DESIGN_DOC_PATH: $RELATIVE_PATH"
