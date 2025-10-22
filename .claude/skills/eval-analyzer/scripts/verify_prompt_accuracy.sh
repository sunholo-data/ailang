#!/usr/bin/env bash
# Verify that prompt documentation matches actual implementation
#
# Usage:
#   verify_prompt_accuracy.sh <prompt_version>
#
# Example:
#   verify_prompt_accuracy.sh v0.3.16

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <prompt_version>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 v0.3.16" >&2
    exit 1
fi

PROMPT_VERSION="$1"
PROMPT_FILE=$(jq -r ".versions[\"$PROMPT_VERSION\"].file" prompts/versions.json)

if [ "$PROMPT_FILE" == "null" ]; then
    echo "Error: Prompt version not found: $PROMPT_VERSION" >&2
    exit 1
fi

echo "=== Prompt Accuracy Check for $PROMPT_VERSION ==="
echo "Prompt file: $PROMPT_FILE"
echo ""

# Check for false limitations
echo "--- Checking for False Limitations ---"
echo ""

# Check HTTP headers limitation
if grep -q "NO custom HTTP headers" "$PROMPT_FILE"; then
    echo "⚠️  ISSUE: Prompt says 'NO custom HTTP headers'"

    # Check if httpRequest actually exists
    if grep -q "httpRequest" stdlib/std/net.ail; then
        echo "❌  MISMATCH: httpRequest() with headers EXISTS in stdlib/std/net.ail!"
        echo "    Line in prompt:"
        grep -n "NO custom HTTP headers" "$PROMPT_FILE"
        echo ""
    else
        echo "✅  OK: httpRequest() not found in stdlib (limitation is accurate)"
    fi
else
    echo "✅  No false HTTP headers limitation found"
fi

echo ""
echo "--- Checking for Undocumented Features ---"
echo ""

# Check if httpRequest exists but isn't documented
if grep -q "func httpRequest" stdlib/std/net.ail; then
    echo "✅  httpRequest() exists in stdlib"

    if grep -q "httpRequest" "$PROMPT_FILE"; then
        echo "✅  httpRequest() documented in prompt"
    else
        echo "❌  MISSING: httpRequest() NOT documented in prompt!"
        echo "    Feature exists but models won't know about it"
    fi
else
    echo "ℹ️   httpRequest() not implemented yet"
fi

echo ""
echo "--- Summary ---"
echo ""

# Count issues
ISSUES=0

if grep -q "NO custom HTTP headers" "$PROMPT_FILE" && grep -q "httpRequest" stdlib/std/net.ail; then
    ISSUES=$((ISSUES + 1))
    echo "❌  False limitation: HTTP headers"
fi

if grep -q "func httpRequest" stdlib/std/net.ail && ! grep -q "httpRequest" "$PROMPT_FILE"; then
    ISSUES=$((ISSUES + 1))
    echo "❌  Undocumented feature: httpRequest()"
fi

if [ $ISSUES -eq 0 ]; then
    echo "✅  No prompt accuracy issues found!"
else
    echo ""
    echo "Found $ISSUES prompt accuracy issue(s)"
    echo "These issues will cause models to generate wrong code!"
fi
