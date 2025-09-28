#!/bin/bash
# Test REPL with single-line let expressions

echo "Testing single-line let expressions..."
echo ""

# Test 1: Simple let expression on one line
echo 'let x = 5 in x * 2' | ./bin/ailang repl 2>&1 | grep -A2 "λ>"

echo ""
echo "---"
echo ""

# Test 2: Let with record on one line
echo 'let user = {name: "Alice", age: 30} in user' | ./bin/ailang repl 2>&1 | grep -A2 "λ>"

echo ""
echo "---"
echo ""

# Test 3: Let with lambda on one line
echo 'let double = \x. x * 2 in double(21)' | ./bin/ailang repl 2>&1 | grep -A2 "λ>"