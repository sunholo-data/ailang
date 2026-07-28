#!/usr/bin/env bash
set -euo pipefail

# Run headless Claude with automatic retry on failure

PROMPT="$1"
MAX_RETRIES="${2:-3}"
OUTPUT_FILE="${3:-/tmp/claude_retry_output.json}"

echo "Running headless Claude with retry..."
echo "Prompt: $PROMPT"
echo "Max retries: $MAX_RETRIES"
echo

RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    echo "Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES..."

    # Try to run claude
    if claude -p "$PROMPT" --output-format json > "$OUTPUT_FILE" 2>/tmp/claude_retry_error.log; then
        # Check if output is valid JSON
        if jq -e '.sessionId' "$OUTPUT_FILE" &> /dev/null; then
            echo "✓ Success!"
            cat "$OUTPUT_FILE"
            exit 0
        else
            echo "⚠ Output not valid JSON, retrying..."
        fi
    else
        EXIT_CODE=$?
        echo "⚠ Command failed with exit code $EXIT_CODE"

        # Check for permanent failures
        if [ $EXIT_CODE -eq 2 ]; then
            echo "✗ Permanent failure (invalid arguments)"
            cat /tmp/claude_retry_error.log
            exit 1
        fi
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))

    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        SLEEP_TIME=$((2 ** RETRY_COUNT))
        echo "  Waiting ${SLEEP_TIME}s before retry..."
        sleep $SLEEP_TIME
    fi
done

echo "✗ Failed after $MAX_RETRIES attempts"
echo
echo "Last error:"
cat /tmp/claude_retry_error.log
exit 2
