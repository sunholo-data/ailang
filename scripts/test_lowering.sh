#!/bin/bash
# Test script for operator lowering

set -e

echo "=== Testing Operator Lowering ==="

# Build the compiler
echo "Building ailang..."
go build ./cmd/ailang

# Test integer operations
echo -n "Testing integer ops... "
result=$(./ailang run tests/binops_int.ail 2>&1 | tail -n1)
if [ "$result" = "14" ]; then
    echo "✓ PASS"
else
    echo "✗ FAIL: expected 14, got $result"
    exit 1
fi

# Test float operations
echo -n "Testing float ops... "
result=$(./ailang run tests/binops_float.ail 2>&1 | tail -n1)
if [ "$result" = "1.5" ]; then
    echo "✓ PASS"
else
    echo "✗ FAIL: expected 1.5, got $result"
    exit 1
fi

# Test precedence
echo -n "Testing precedence... "
result=$(./ailang run tests/precedence_lowering.ail 2>&1 | tail -n1)
if [ "$result" = "14" ]; then
    echo "✓ PASS"
else
    echo "✗ FAIL: expected 14, got $result"
    exit 1
fi

# Test short-circuit
echo -n "Testing short-circuit... "
result=$(./ailang run tests/short_circuit.ail 2>&1 | tail -n1)
if [ "$result" = "false" ]; then
    echo "✓ PASS"
else
    echo "✗ FAIL: expected false, got $result"
    exit 1
fi

echo "=== All Lowering Tests Passed ==="