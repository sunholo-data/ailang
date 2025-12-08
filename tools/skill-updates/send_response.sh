#!/bin/bash
# send_response.sh - Send response to external project
# Usage: send_response.sh <project> <title> <message> [priority]
set -euo pipefail

usage() {
    echo "Usage: send_response.sh <project> <title> <message> [priority]"
    echo ""
    echo "Arguments:"
    echo "  project   - Target project name (e.g., stapledons_voyage)"
    echo "  title     - Brief title for the response"
    echo "  message   - Detailed message body"
    echo "  priority  - Optional: low, medium (default), high"
    echo ""
    echo "Examples:"
    echo "  send_response.sh stapledons_voyage 'Bug acknowledged' 'Design doc created'"
    echo "  send_response.sh my_project 'Answer: intToFloat' 'Use intToFloat(n)' high"
    exit 1
}

if [ $# -lt 3 ]; then
    usage
fi

PROJECT="$1"
TITLE="$2"
MESSAGE="$3"
PRIORITY="${4:-medium}"

# Escape special characters for JSON
escape_json() {
    local str="$1"
    str="${str//\\/\\\\}"
    str="${str//\"/\\\"}"
    str="${str//$'\n'/\\n}"
    str="${str//$'\r'/\\r}"
    str="${str//$'\t'/\\t}"
    echo "$str"
}

ESCAPED_TITLE=$(escape_json "$TITLE")
ESCAPED_MESSAGE=$(escape_json "$MESSAGE")

# Build JSON payload
PAYLOAD="{\"type\":\"response\",\"title\":\"${ESCAPED_TITLE}\",\"description\":\"${ESCAPED_MESSAGE}\",\"priority\":\"${PRIORITY}\",\"from_project\":\"ailang_core\"}"

# Send the message
echo "Sending response to $PROJECT..."
ailang agent send "$PROJECT" "$PAYLOAD"

echo ""
echo "✅ Response sent successfully"
