#!/bin/bash
# audit_env_vars.sh - Find environment variables in codebase and compare against help.go
# Usage: ./audit_env_vars.sh
# Exit codes: 0 = synchronized, 1 = discrepancies found

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
HELP_GO="$REPO_ROOT/cmd/ailang/help.go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== Environment Variables Audit ===${NC}"
echo ""

# Find DEBUG_* variables in codebase
echo "Searching for DEBUG_* variables in internal/..."
DEBUG_VARS=$(grep -roh 'DEBUG_[A-Z_]*' "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | sort -u || true)

# Find AILANG_* variables in codebase
echo "Searching for AILANG_* variables in internal/..."
AILANG_VARS=$(grep -roh 'AILANG_[A-Z_]*' "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | sort -u || true)

# Find OTEL_* and GOOGLE_* variables
echo "Searching for OTEL_* and GOOGLE_CLOUD_* variables in internal/..."
OTEL_VARS=$(grep -roh 'OTEL_[A-Z_]*' "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | sort -u || true)
GCP_VARS=$(grep -roh 'GOOGLE_CLOUD_[A-Z_]*' "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | sort -u || true)

# Combine all variables
ALL_VARS=$(echo -e "$DEBUG_VARS\n$AILANG_VARS\n$OTEL_VARS\n$GCP_VARS" | grep -v '^$' | sort -u)
VARS_COUNT=$(echo "$ALL_VARS" | wc -l | tr -d ' ')

# Extract documented variables from help.go
HELP_VARS=$(grep -E '^\s*(DEBUG_|AILANG_|OTEL_|GOOGLE_)' "$HELP_GO" | awk '{print $1}' | sort -u || true)
HELP_COUNT=$(echo "$HELP_VARS" | wc -l | tr -d ' ')

echo ""
echo -e "${CYAN}Environment variables found in code:${NC} $VARS_COUNT"
echo -e "${CYAN}Environment variables documented:${NC} $HELP_COUNT"
echo ""

# Find variables in code but not documented
MISSING_IN_HELP=()
while IFS= read -r var; do
    # Normalize: remove trailing =1 or =<...>
    var_name=$(echo "$var" | sed 's/=.*//')
    if ! echo "$HELP_VARS" | grep -q "^$var_name"; then
        MISSING_IN_HELP+=("$var_name")
    fi
done <<< "$ALL_VARS"

# Find documented variables not used in code (stale)
STALE_IN_HELP=()
while IFS= read -r var; do
    var_name=$(echo "$var" | sed 's/=.*//')
    if ! echo "$ALL_VARS" | grep -q "^$var_name"; then
        STALE_IN_HELP+=("$var_name")
    fi
done <<< "$HELP_VARS"

# Report
HAS_ISSUES=0

if [ ${#MISSING_IN_HELP[@]} -gt 0 ]; then
    echo -e "${RED}✗ Environment variables used in code but NOT documented:${NC}"
    for var in "${MISSING_IN_HELP[@]}"; do
        # Find where it's used
        usage=$(grep -rn "$var" "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | head -1 | cut -d: -f1-2 || echo "unknown")
        echo -e "  ${RED}- $var${NC} (used in: $usage)"
    done
    echo ""
    HAS_ISSUES=1
fi

if [ ${#STALE_IN_HELP[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠ Environment variables documented but NOT used in code (stale):${NC}"
    for var in "${STALE_IN_HELP[@]}"; do
        echo -e "  ${YELLOW}- $var${NC}"
    done
    echo ""
    HAS_ISSUES=1
fi

if [ $HAS_ISSUES -eq 0 ]; then
    echo -e "${GREEN}✓ All environment variables are synchronized!${NC}"
    echo ""
    echo "Variables found and documented:"
    echo "$ALL_VARS" | while read -r var; do
        usage_count=$(grep -r "$var" "$REPO_ROOT/internal/" --include="*.go" 2>/dev/null | wc -l | tr -d ' ')
        echo "  - $var ($usage_count usages)"
    done
    exit 0
else
    echo -e "${RED}Action required:${NC}"
    if [ ${#MISSING_IN_HELP[@]} -gt 0 ]; then
        echo "  1. Add missing variables to cmd/ailang/help.go"
        echo "  2. Include clear purpose and usage example"
        echo "  3. Categorize under 'Debug Flags' or 'Configuration'"
    fi
    if [ ${#STALE_IN_HELP[@]} -gt 0 ]; then
        echo "  1. Remove stale variables from cmd/ailang/help.go"
        echo "  2. Or verify they're used in cmd/ (not just internal/)"
    fi
    exit 1
fi
