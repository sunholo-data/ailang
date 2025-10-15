#!/bin/bash
# Audit all examples and categorize by status

AILANG="./bin/ailang"
RESULTS_FILE="examples/AUDIT_RESULTS.tmp"

echo "=== AILANG Example Audit ===" > "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Ensure ailang is built
if [ ! -f "$AILANG" ]; then
    echo "Building ailang..."
    make build > /dev/null 2>&1
fi

# Categories
WORKING=()
TYPE_CHECK_ONLY=()
BROKEN=()

# Test each example
while IFS= read -r file; do
    echo "Testing: $file"

    # Detect required capabilities from effect annotations
    caps=""
    if grep -q '! {IO}' "$file" || grep -q '_io_' "$file"; then
        caps="IO"
    fi
    if grep -q '! {FS}' "$file" || grep -q '_fs_' "$file"; then
        if [ -n "$caps" ]; then
            caps="$caps,FS"
        else
            caps="FS"
        fi
    fi

    # Try to run it with appropriate capabilities
    if [ -n "$caps" ]; then
        output=$("$AILANG" --caps "$caps" run "$file" 2>&1)
    else
        output=$("$AILANG" run "$file" 2>&1)
    fi
    exit_code=$?

    # Check if it's a module (has 'module' declaration)
    is_module=$(grep -q "^module " "$file" && echo "yes" || echo "no")

    # Categorize based on output
    if [ $exit_code -eq 0 ]; then
        if echo "$output" | grep -q "Module evaluation not yet supported"; then
            # Module that type-checks but can't execute
            TYPE_CHECK_ONLY+=("$file")
        else
            # Actually works!
            WORKING+=("$file")
        fi
    else
        # Parse or type error
        BROKEN+=("$file")
    fi
done < <(find examples/ -name "*.ail" -type f | sort)

# Write results
echo "## ✅ Working (${#WORKING[@]} files)" >> "$RESULTS_FILE"
echo "These examples parse, type-check, and execute successfully:" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
for file in "${WORKING[@]}"; do
    echo "- $file" >> "$RESULTS_FILE"
done

echo "" >> "$RESULTS_FILE"
echo "## ⚠️ Type-Checks Only (${#TYPE_CHECK_ONLY[@]} files)" >> "$RESULTS_FILE"
echo "These examples parse and type-check but cannot execute (module limitation):" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
for file in "${TYPE_CHECK_ONLY[@]}"; do
    echo "- $file" >> "$RESULTS_FILE"
done

echo "" >> "$RESULTS_FILE"
echo "## ❌ Broken (${#BROKEN[@]} files)" >> "$RESULTS_FILE"
echo "These examples have parse errors or type errors:" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"
for file in "${BROKEN[@]}"; do
    echo "- $file" >> "$RESULTS_FILE"
done

echo "" >> "$RESULTS_FILE"
echo "### Summary" >> "$RESULTS_FILE"
echo "- Total examples: $((${#WORKING[@]} + ${#TYPE_CHECK_ONLY[@]} + ${#BROKEN[@]}))" >> "$RESULTS_FILE"
echo "- Working: ${#WORKING[@]}" >> "$RESULTS_FILE"
echo "- Type-checks only: ${#TYPE_CHECK_ONLY[@]}" >> "$RESULTS_FILE"
echo "- Broken: ${#BROKEN[@]}" >> "$RESULTS_FILE"

cat "$RESULTS_FILE"
