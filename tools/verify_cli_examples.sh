#!/bin/bash
# Verify CLI examples from documented patterns
# Usage: ./tools/verify_cli_examples.sh [--verbose]
#
# This script reads CLI examples from examples/cli_examples.txt and verifies
# each one runs successfully. Examples should have format:
#
#   # Comment explaining the example
#   $ ailang run --entry main examples/runnable/hello.ail
#   Hello, AILANG!
#
# Lines starting with $ are commands, following lines (without $) are expected output.

set -e

VERBOSE=0
if [[ "$1" == "--verbose" ]]; then
    VERBOSE=1
fi

EXAMPLES_FILE="examples/cli_examples.txt"
PASSED=0
FAILED=0
FAILURES=""

if [[ ! -f "$EXAMPLES_FILE" ]]; then
    echo "Error: $EXAMPLES_FILE not found"
    echo "Create it with documented CLI examples to verify"
    exit 1
fi

echo "Verifying CLI examples from $EXAMPLES_FILE..."
echo ""

current_cmd=""
expected_output=""

while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and comments
    if [[ -z "$line" ]] || [[ "$line" =~ ^# ]]; then
        continue
    fi

    # Command line (starts with $)
    if [[ "$line" =~ ^\$ ]]; then
        # Execute previous command if any
        if [[ -n "$current_cmd" ]]; then
            if [[ $VERBOSE -eq 1 ]]; then
                echo "Running: $current_cmd"
            fi

            actual_output=$(eval "$current_cmd" 2>&1) || true

            # Trim whitespace for comparison
            expected_trimmed=$(echo "$expected_output" | xargs)
            actual_trimmed=$(echo "$actual_output" | xargs)

            # Check if expected output is contained in actual output
            # (allows for progress messages and warnings in output)
            if [[ -z "$expected_trimmed" ]] || [[ "$actual_output" == *"$expected_trimmed"* ]]; then
                echo "✓ $current_cmd"
                ((PASSED++))
            else
                echo "✗ $current_cmd"
                echo "  Expected (substring): $expected_trimmed"
                echo "  Actual: $actual_trimmed"
                ((FAILED++))
                FAILURES="$FAILURES\n  - $current_cmd"
            fi
        fi

        # Start new command (remove leading $ and space)
        current_cmd="${line:2}"
        expected_output=""
    else
        # Expected output line
        if [[ -z "$expected_output" ]]; then
            expected_output="$line"
        else
            expected_output="$expected_output\n$line"
        fi
    fi
done < "$EXAMPLES_FILE"

# Execute last command
if [[ -n "$current_cmd" ]]; then
    if [[ $VERBOSE -eq 1 ]]; then
        echo "Running: $current_cmd"
    fi

    actual_output=$(eval "$current_cmd" 2>&1) || true
    expected_trimmed=$(echo -e "$expected_output" | xargs)
    actual_trimmed=$(echo "$actual_output" | xargs)

    # Check if expected output is contained in actual output
    if [[ -z "$expected_trimmed" ]] || [[ "$actual_output" == *"$expected_trimmed"* ]]; then
        echo "✓ $current_cmd"
        ((PASSED++))
    else
        echo "✗ $current_cmd"
        echo "  Expected (substring): $expected_trimmed"
        echo "  Actual: $actual_trimmed"
        ((FAILED++))
        FAILURES="$FAILURES\n  - $current_cmd"
    fi
fi

echo ""
echo "================================"
echo "Results: $PASSED passed, $FAILED failed"

if [[ $FAILED -gt 0 ]]; then
    echo -e "Failed commands:$FAILURES"
    exit 1
else
    echo "All CLI examples verified!"
    exit 0
fi
