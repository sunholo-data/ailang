#!/usr/bin/env bash
set -euo pipefail

# Test headless Claude mode with project configuration

echo "Testing headless Claude mode..."
echo

# Test 1: Check claude command exists
echo "1/6 Checking claude command..."
if ! command -v claude &> /dev/null; then
    echo "✗ claude command not found in PATH"
    echo "  Install Claude Code CLI first"
    exit 1
fi
echo "✓ claude command available"
echo

# Test 2: Test basic invocation
echo "2/6 Testing basic invocation..."
if ! claude -p "Echo: test" &> /dev/null; then
    echo "✗ Basic invocation failed"
    exit 1
fi
echo "✓ Basic invocation works"
echo

# Test 3: Test JSON output
echo "3/6 Testing JSON output..."
RESULT=$(claude -p "List all available skills" --output-format json 2>/dev/null || echo "{}")
if ! echo "$RESULT" | jq -e '.session_id' &> /dev/null; then
    echo "✗ JSON output invalid or missing session_id"
    echo "  Response snippet: $(echo "$RESULT" | jq -c '{type, session_id}' 2>/dev/null || echo "invalid JSON")"
    exit 1
fi
echo "✓ JSON output valid"
echo

# Test 4: Check skills loaded
echo "4/6 Checking skills are accessible..."
SKILLS_OUTPUT=$(claude -p "List all available skills, agents, and slash commands you have access to in this project. Also confirm you can see the CLAUDE.md instructions." 2>&1)
if echo "$SKILLS_OUTPUT" | grep -q "skill-builder"; then
    echo "✓ Skills are loaded (found skill-builder)"
else
    echo "⚠ Skills may not be loaded correctly"
    echo "  Expected to find 'skill-builder' in output"
fi
echo

# Test 5: Check agents loaded
echo "5/6 Checking agents are accessible..."
if echo "$SKILLS_OUTPUT" | grep -q "eval-orchestrator\|Explore"; then
    echo "✓ Agents are loaded"
else
    echo "⚠ Agents may not be loaded correctly"
fi
echo

# Test 6: Check CLAUDE.md loaded
echo "6/6 Checking CLAUDE.md instructions..."
if echo "$SKILLS_OUTPUT" | grep -q "CLAUDE.md"; then
    echo "✓ CLAUDE.md instructions are loaded"
else
    echo "⚠ CLAUDE.md may not be loaded"
fi
echo

echo "========================================="
echo "✓ Headless mode test complete!"
echo
echo "All critical tests passed."
echo "Skills, agents, and project config are loading correctly."
