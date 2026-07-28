#!/bin/bash
# suggest_improvements.sh - Analyze help text and suggest improvements
# Usage: ./suggest_improvements.sh
# Exit codes: 0 = all good, 1 = suggestions available

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
HELP_GO="$REPO_ROOT/cmd/ailang/help.go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== CLI Help Text Analysis ===${NC}"
echo ""

HAS_SUGGESTIONS=0

# Check 1: File length (should be scannable - target <250 lines)
LINE_COUNT=$(wc -l < "$HELP_GO" | tr -d ' ')
echo -e "${BLUE}1. File Length Check${NC}"
if [ $LINE_COUNT -gt 300 ]; then
    echo -e "  ${RED}✗ help.go is $LINE_COUNT lines (target: <250 for scannability)${NC}"
    echo "    Suggestion: Move detailed docs to per-command help functions"
    HAS_SUGGESTIONS=1
elif [ $LINE_COUNT -gt 250 ]; then
    echo -e "  ${YELLOW}⚠ help.go is $LINE_COUNT lines (acceptable, but approaching limit)${NC}"
    echo "    Suggestion: Consider moving some details to per-command help"
else
    echo -e "  ${GREEN}✓ help.go is $LINE_COUNT lines (good length)${NC}"
fi
echo ""

# Check 2: Examples section
echo -e "${BLUE}2. Examples Check${NC}"
EXAMPLE_COUNT=$(grep -c "# " "$HELP_GO" | tr -d ' ' || echo "0")
if [ $EXAMPLE_COUNT -lt 10 ]; then
    echo -e "  ${YELLOW}⚠ Only $EXAMPLE_COUNT examples found${NC}"
    echo "    Suggestion: Add more practical examples (aim for 15-20)"
    HAS_SUGGESTIONS=1
else
    echo -e "  ${GREEN}✓ Found $EXAMPLE_COUNT examples${NC}"
fi
echo ""

# Check 3: Category organization
echo -e "${BLUE}3. Category Organization${NC}"
CATEGORIES=$(grep -c "^[[:space:]]*fmt.Println(\"[A-Z]" "$HELP_GO" | tr -d ' ' || echo "0")
echo -e "  ${CYAN}Found $CATEGORIES category headers${NC}"
if [ $CATEGORIES -lt 6 ]; then
    echo -e "  ${YELLOW}⚠ Consider adding more categories for better organization${NC}"
    HAS_SUGGESTIONS=1
else
    echo -e "  ${GREEN}✓ Good category organization${NC}"
fi
echo ""

# Check 4: Command with subcommands listed
echo -e "${BLUE}4. Subcommand Documentation${NC}"
COMMANDS_WITH_SUBCOMMANDS=$(grep -E "(coordinator|trace|observatory|budget|messages)" "$HELP_GO" | wc -l | tr -d ' ')
SUBCOMMANDS_LISTED=$(grep -E "^\s+(coordinator|trace|observatory|budget|messages) " "$HELP_GO" | wc -l | tr -d ' ')

echo -e "  ${CYAN}Commands with subcommands: $COMMANDS_WITH_SUBCOMMANDS${NC}"
echo -e "  ${CYAN}Subcommands explicitly listed: $SUBCOMMANDS_LISTED${NC}"

if [ $SUBCOMMANDS_LISTED -eq 0 ]; then
    echo -e "  ${YELLOW}⚠ Subcommands not shown in main help${NC}"
    echo "    Suggestion: List key subcommands for coordinator, trace, etc."
    HAS_SUGGESTIONS=1
else
    echo -e "  ${GREEN}✓ Subcommands are documented${NC}"
fi
echo ""

# Check 5: Cross-references
echo -e "${BLUE}5. Cross-References Check${NC}"
CROSS_REFS=$(grep "See also" "$HELP_GO" 2>/dev/null | wc -l | tr -d ' ')
if [ "$CROSS_REFS" -lt 3 ]; then
    echo -e "  ${YELLOW}⚠ Few or no cross-references found ($CROSS_REFS)${NC}"
    echo "    Suggestion: Add 'See also:' references between related commands"
    echo "    Example: 'See also: ailang coordinator --help'"
    HAS_SUGGESTIONS=1
else
    echo -e "  ${GREEN}✓ Found $CROSS_REFS cross-references${NC}"
fi
echo ""

# Check 6: Environment variables section
echo -e "${BLUE}6. Environment Variables Section${NC}"
if grep -q "Environment Variables:" "$HELP_GO"; then
    ENV_VARS=$(grep -E "^\s+(DEBUG_|AILANG_|OTEL_|GOOGLE_)" "$HELP_GO" | wc -l | tr -d ' ')
    echo -e "  ${GREEN}✓ Environment variables section exists ($ENV_VARS variables)${NC}"
else
    echo -e "  ${RED}✗ No environment variables section found${NC}"
    echo "    Suggestion: Add 'Environment Variables:' section"
    HAS_SUGGESTIONS=1
fi
echo ""

# Check 7: Common flags reference
echo -e "${BLUE}7. Common Flags Reference${NC}"
if grep -q "Global Flags:" "$HELP_GO"; then
    echo -e "  ${GREEN}✓ Global flags section exists${NC}"
else
    echo -e "  ${YELLOW}⚠ No 'Global Flags' section found${NC}"
    echo "    Suggestion: Add common flags reference (--help, --version, --json, etc.)"
    HAS_SUGGESTIONS=1
fi
echo ""

# Summary
if [ $HAS_SUGGESTIONS -eq 0 ]; then
    echo -e "${GREEN}=== All checks passed! ===${NC}"
    echo ""
    echo "The help text is well-organized and complete."
    exit 0
else
    echo -e "${YELLOW}=== Suggestions Available ===${NC}"
    echo ""
    echo "Review the suggestions above to improve CLI discoverability."
    echo ""
    echo "Priority actions:"
    echo "  1. Ensure all commands are categorized"
    echo "  2. Add practical examples for complex commands"
    echo "  3. Document environment variables"
    echo "  4. Add cross-references between related commands"
    exit 1
fi
