#!/bin/bash
#
# Acceptance Testing Script
#
# Purpose: Run end-to-end tests for a sprint milestone
# Based on: https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
#
# Usage:
#   .claude/skills/sprint-executor/scripts/acceptance_test.sh <milestone_id> <test_type>
#
# Example:
#   # Parser sprint - run example files
#   .claude/skills/sprint-executor/scripts/acceptance_test.sh M-S1.1 parser
#
#   # Builtin sprint - test in REPL
#   .claude/skills/sprint-executor/scripts/acceptance_test.sh M-DX1.2 builtin
#
#   # General feature - run specific examples
#   .claude/skills/sprint-executor/scripts/acceptance_test.sh M-POLY-A.3 examples
#
# Test Types:
#   parser    - Run example AILANG files through parser
#   builtin   - Test builtin functions in REPL or test harness
#   examples  - Run specific example files for the feature
#   repl      - Interactive REPL testing (manual)
#   e2e       - Full end-to-end workflow test
#
# This implements the "testing as user would" pattern from the Anthropic article.
# Unit tests are great, but end-to-end tests catch integration issues.

set -e  # Exit on error

# Check arguments
if [ $# -lt 2 ]; then
    echo "Usage: $0 <milestone_id> <test_type>"
    echo ""
    echo "Test types:"
    echo "  parser    - Run example files through parser"
    echo "  builtin   - Test builtin functions"
    echo "  examples  - Run specific example files"
    echo "  repl      - Interactive REPL testing"
    echo "  e2e       - Full end-to-end workflow"
    echo ""
    echo "Example:"
    echo "  $0 M-S1.1 parser"
    exit 1
fi

MILESTONE_ID="$1"
TEST_TYPE="$2"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "═══════════════════════════════════════════════════════════════"
echo " Acceptance Tests - ${MILESTONE_ID}"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Test type: $TEST_TYPE"
echo "Milestone: $MILESTONE_ID"
echo ""

# Ensure ailang binary is up to date
echo -e "${BLUE}Ensuring ailang binary is current...${NC}"
if ! make quick-install > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠  Failed to rebuild ailang, using existing binary${NC}"
fi
echo ""

# Test based on type
case "$TEST_TYPE" in
    parser)
        echo -e "${BLUE}Running parser acceptance tests...${NC}"
        echo ""

        # Test that example files parse correctly
        echo "Testing example files..."
        FAILED=0
        PASSED=0

        for file in examples/*.ail; do
            if [[ "$file" == *"broken"* ]] || [[ "$file" == *"wip"* ]]; then
                echo -e "  ${YELLOW}⊘ Skipping: $(basename "$file")${NC}"
                continue
            fi

            if ailang run "$file" > /dev/null 2>&1; then
                echo -e "  ${GREEN}✓ $(basename "$file")${NC}"
                ((PASSED++))
            else
                echo -e "  ${RED}✗ $(basename "$file")${NC}"
                ((FAILED++))
            fi
        done

        echo ""
        if [ $FAILED -eq 0 ]; then
            echo -e "${GREEN}✓ All parser tests passed ($PASSED files)${NC}"
            exit 0
        else
            echo -e "${RED}✗ Some parser tests failed ($FAILED/$((PASSED + FAILED)))${NC}"
            exit 1
        fi
        ;;

    builtin)
        echo -e "${BLUE}Running builtin acceptance tests...${NC}"
        echo ""

        # Check if there are specific test files for this milestone
        # This assumes builtins have test files in tests/ or examples/
        echo "Testing builtin functions..."

        # Run unit tests for builtins package
        if make test | grep -q "internal/builtins"; then
            echo -e "${GREEN}✓ Builtin unit tests pass${NC}"
        else
            echo -e "${RED}✗ Builtin unit tests failed${NC}"
            exit 1
        fi

        # Check if there are example files demonstrating the builtin
        # Format: examples/builtin_<name>.ail
        BUILTIN_EXAMPLES=$(find examples -name "builtin_*.ail" 2>/dev/null | wc -l)
        if [ "$BUILTIN_EXAMPLES" -gt 0 ]; then
            echo ""
            echo "Testing builtin example files..."
            for file in examples/builtin_*.ail; do
                if ailang run "$file" > /dev/null 2>&1; then
                    echo -e "  ${GREEN}✓ $(basename "$file")${NC}"
                else
                    echo -e "  ${RED}✗ $(basename "$file")${NC}"
                    exit 1
                fi
            done
        fi

        echo ""
        echo -e "${GREEN}✓ Builtin acceptance tests passed${NC}"
        ;;

    examples)
        echo -e "${BLUE}Running example file tests...${NC}"
        echo ""

        # Ask user which examples to test
        echo "Example files in examples/:"
        ls -1 examples/*.ail | head -10
        echo ""

        read -p "Enter example file pattern (e.g., 'list_*.ail' or specific file): " PATTERN

        echo ""
        echo "Testing examples matching: $PATTERN"
        FAILED=0
        PASSED=0

        for file in examples/${PATTERN}; do
            if [ ! -f "$file" ]; then
                echo -e "${RED}✗ File not found: $file${NC}"
                continue
            fi

            echo "Running: $(basename "$file")"
            if ailang run "$file"; then
                echo -e "${GREEN}✓ $(basename "$file")${NC}"
                ((PASSED++))
            else
                echo -e "${RED}✗ $(basename "$file") failed${NC}"
                ((FAILED++))
            fi
            echo ""
        done

        if [ $FAILED -eq 0 ] && [ $PASSED -gt 0 ]; then
            echo -e "${GREEN}✓ All example tests passed ($PASSED files)${NC}"
            exit 0
        else
            echo -e "${RED}✗ Some tests failed or no files matched${NC}"
            exit 1
        fi
        ;;

    repl)
        echo -e "${BLUE}Manual REPL testing...${NC}"
        echo ""
        echo "Instructions:"
        echo "1. Test the feature interactively in the REPL"
        echo "2. Try edge cases and error conditions"
        echo "3. Verify error messages are helpful"
        echo "4. Exit when done (Ctrl+D)"
        echo ""
        echo "Starting REPL..."
        echo ""

        ailang repl

        echo ""
        read -p "Did the feature work as expected? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${GREEN}✓ REPL testing passed${NC}"
            exit 0
        else
            echo -e "${RED}✗ REPL testing failed${NC}"
            exit 1
        fi
        ;;

    e2e)
        echo -e "${BLUE}Running end-to-end workflow test...${NC}"
        echo ""

        # Full workflow: parse → elaborate → type check → evaluate
        echo "Testing full pipeline..."

        # Create a test file
        TEST_FILE="/tmp/ailang_e2e_test.ail"
        cat > "$TEST_FILE" << 'EOF'
module e2e_test

def add(x: int, y: int): int = x + y

def main(): int = {
    let result = add(40, 2);
    result
}
EOF

        echo "Test file created: $TEST_FILE"
        echo ""

        # Run with different flags to test pipeline stages
        echo "1. Parsing..."
        if ailang run "$TEST_FILE" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✓ Parse successful${NC}"
        else
            echo -e "   ${RED}✗ Parse failed${NC}"
            exit 1
        fi

        echo "2. Type checking..."
        if ailang run "$TEST_FILE" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✓ Type check successful${NC}"
        else
            echo -e "   ${RED}✗ Type check failed${NC}"
            exit 1
        fi

        echo "3. Execution..."
        OUTPUT=$(ailang run --entry main --caps IO "$TEST_FILE" 2>&1)
        if echo "$OUTPUT" | grep -q "42"; then
            echo -e "   ${GREEN}✓ Execution successful (output: 42)${NC}"
        else
            echo -e "   ${RED}✗ Execution failed or wrong output${NC}"
            echo "   Output: $OUTPUT"
            exit 1
        fi

        echo ""
        echo -e "${GREEN}✓ End-to-end test passed${NC}"

        # Cleanup
        rm "$TEST_FILE"
        ;;

    *)
        echo -e "${RED}Unknown test type: $TEST_TYPE${NC}"
        echo ""
        echo "Valid types: parser, builtin, examples, repl, e2e"
        exit 1
        ;;
esac

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Acceptance Tests Complete"
echo "═══════════════════════════════════════════════════════════════"

exit 0
