#!/bin/bash
# audit_commands.sh - Compare commands in main.go against help.go
# Usage: ./audit_commands.sh
# Exit codes: 0 = synchronized, 1 = discrepancies found

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
MAIN_GO="$REPO_ROOT/cmd/ailang/main.go"
HELP_GO="$REPO_ROOT/cmd/ailang/help.go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== CLI Commands Audit ===${NC}"
echo ""

# Extract commands from main.go (case statements)
echo "Extracting commands from main.go..."
MAIN_COMMANDS=$(grep -E '^\s*case "' "$MAIN_GO" | sed 's/.*case "\([^"]*\)".*/\1/' | grep -v '^$' | sort -u)

# Extract commands from help.go (cyan() calls with command names)
echo "Extracting commands from help.go..."
HELP_COMMANDS=$(grep -oE 'cyan\("[^"]+"\)' "$HELP_GO" | sed 's/cyan("\([^"]*\)")/\1/' | grep -v '^-' | grep -v '^\[' | sort -u)

# Count
MAIN_COUNT=$(echo "$MAIN_COMMANDS" | wc -l | tr -d ' ')
HELP_COUNT=$(echo "$HELP_COMMANDS" | wc -l | tr -d ' ')

echo ""
echo -e "${CYAN}Commands in main.go:${NC} $MAIN_COUNT"
echo -e "${CYAN}Commands in help.go:${NC} $HELP_COUNT"
echo ""

# Find commands in main.go but not in help.go
MISSING_IN_HELP=()
while IFS= read -r cmd; do
    if ! echo "$HELP_COMMANDS" | grep -qw "$cmd"; then
        MISSING_IN_HELP+=("$cmd")
    fi
done <<< "$MAIN_COMMANDS"

# Find commands in help.go but not in main.go (stale)
STALE_IN_HELP=()
while IFS= read -r cmd; do
    # Skip common non-command entries (flags, placeholders)
    if [[ "$cmd" =~ ^(run|repl|test|check|watch|iface|export-training|eval|eval-suite|eval-analyze|eval-report|eval-compare|eval-matrix|eval-summary|doctor|builtins|docs|debug|messages|msg|prompt|server|serve|serve-api|init|access-control|compile|editor|axioms|trace|observatory|dashboard|budget|coordinator|workspaces|exec|examples)$ ]]; then
        if ! echo "$MAIN_COMMANDS" | grep -qw "$cmd"; then
            STALE_IN_HELP+=("$cmd")
        fi
    fi
done <<< "$HELP_COMMANDS"

# Report
HAS_ISSUES=0

if [ ${#MISSING_IN_HELP[@]} -gt 0 ]; then
    echo -e "${RED}✗ Commands in main.go but NOT documented in help.go:${NC}"
    for cmd in "${MISSING_IN_HELP[@]}"; do
        echo -e "  ${RED}- $cmd${NC}"
    done
    echo ""
    HAS_ISSUES=1
fi

if [ ${#STALE_IN_HELP[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠ Commands documented in help.go but NOT in main.go (stale):${NC}"
    for cmd in "${STALE_IN_HELP[@]}"; do
        echo -e "  ${YELLOW}- $cmd${NC}"
    done
    echo ""
    HAS_ISSUES=1
fi

if [ $HAS_ISSUES -eq 0 ]; then
    echo -e "${GREEN}✓ All commands are synchronized!${NC}"
    echo ""
    echo "Commands implemented and documented:"
    echo "$MAIN_COMMANDS" | head -20
    if [ $MAIN_COUNT -gt 20 ]; then
        echo "... and $((MAIN_COUNT - 20)) more"
    fi
    exit 0
else
    echo -e "${RED}Action required:${NC}"
    if [ ${#MISSING_IN_HELP[@]} -gt 0 ]; then
        echo "  1. Add missing commands to cmd/ailang/help.go"
        echo "  2. Categorize appropriately (Core, Development, Agent Coordination, etc.)"
        echo "  3. Include brief description and example"
    fi
    if [ ${#STALE_IN_HELP[@]} -gt 0 ]; then
        echo "  1. Remove stale commands from cmd/ailang/help.go"
        echo "  2. Or verify they exist as aliases/subcommands"
    fi
    exit 1
fi
