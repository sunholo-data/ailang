#!/usr/bin/env bash
# Profile an AILANG file with phase timing breakdown
#
# Usage: ./profile_ailang.sh <file.ail>

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <file.ail>"
    exit 1
fi

FILE="$1"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"

if [[ -x "$PROJECT_ROOT/bin/ailang" ]]; then
    AILANG="$PROJECT_ROOT/bin/ailang"
else
    AILANG="ailang"
fi

echo "Profiling: $FILE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Run with debug-compile to get phase timing
"$AILANG" check --debug-compile "$FILE" 2>&1

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Phase interpretation:"
echo "  - Lexing/Parsing: Time to tokenize and parse source"
echo "  - Elaboration: Surface AST to Core AST transformation"
echo "  - Type checking: Hindley-Milner inference"
echo "  - Validation: CoreTypeInfo and effect checking"
echo ""
echo "If a phase is slow:"
echo "  - Parsing: Simplify syntax, reduce nesting"
echo "  - Type checking: Add type annotations, reduce polymorphism"
echo "  - Elaboration: Simplify pattern matching"
