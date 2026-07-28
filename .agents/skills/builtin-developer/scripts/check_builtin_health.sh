#!/usr/bin/env bash
set -euo pipefail

# Check AILANG builtin health and list all builtins

echo "AILANG Builtin Health Check"
echo "============================"
echo ""

# Check if ailang command exists
if ! command -v ailang &> /dev/null; then
    echo "✗ ailang command not found"
    echo "  Run: make install"
    exit 1
fi

# Run doctor
echo "1. Running ailang doctor builtins..."
echo "-----------------------------------"
if ailang doctor builtins; then
    echo ""
    echo "✓ All builtins are valid"
else
    echo ""
    echo "✗ Builtin validation failed"
    exit 1
fi

echo ""
echo "2. Listing builtins by module..."
echo "--------------------------------"
ailang builtins list --by-module

echo ""
echo "3. Builtin statistics..."
echo "------------------------"
TOTAL=$(ailang builtins list | grep -c "^\s*_" || true)
PURE=$(ailang builtins list | grep -c "\[pure\]" || true)
EFFECTS=$((TOTAL - PURE))

echo "Total builtins: $TOTAL"
echo "Pure functions: $PURE"
echo "Effect functions: $EFFECTS"

echo ""
echo "✓ Health check complete"
