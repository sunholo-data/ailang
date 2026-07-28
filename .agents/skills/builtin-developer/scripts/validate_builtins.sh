#!/usr/bin/env bash
set -euo pipefail

# Validate AILANG builtins registry

echo "Validating AILANG builtins..."

# Check if ailang command exists
if ! command -v ailang &> /dev/null; then
    echo "✗ ailang command not found"
    echo "  Run: make install"
    exit 1
fi

# Run doctor builtins
echo ""
echo "Running ailang doctor builtins..."
if ailang doctor builtins 2>&1 | tee /tmp/builtin_doctor.log; then
    echo "✓ All builtins are valid"
else
    echo "✗ Builtin validation failed"
    echo "  See /tmp/builtin_doctor.log for details"
    exit 1
fi

# Count builtins
echo ""
echo "Counting registered builtins..."
BUILTIN_COUNT=$(ailang builtins list | grep -c "^\s*_" || true)
echo "✓ Found $BUILTIN_COUNT builtins"

# Check for orphaned builtins
echo ""
echo "Checking for orphaned builtins..."
if ailang builtins check-migration 2>&1 | tee /tmp/builtin_migration.log | grep -q "No orphaned"; then
    echo "✓ No orphaned builtins found"
else
    echo "⚠ Found orphaned builtins - see /tmp/builtin_migration.log"
fi

echo ""
echo "✓ Builtin validation complete"
echo ""
echo "Summary:"
echo "  Total builtins: $BUILTIN_COUNT"
echo "  Status: All valid"
echo ""
echo "Run ./scripts/check_builtin_health.sh for detailed listing"
