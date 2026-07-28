#!/usr/bin/env bash
# Check if a Gemini model is available in Vertex AI
# Usage: verify_vertex_model.sh <model-name>

set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <model-name>"
    echo ""
    echo "Examples:"
    echo "  $0 gemini-3-pro-preview-11-2025"
    echo "  $0 gemini-2.5-pro"
    exit 1
fi

MODEL="$1"

echo "Checking Vertex AI for: $MODEL"
echo ""

# Check gcloud
if ! command -v gcloud &> /dev/null; then
    echo "✗ gcloud CLI not found"
    echo "Install: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Get access token
ACCESS_TOKEN=$(gcloud auth application-default print-access-token 2>/dev/null || true)
if [ -z "$ACCESS_TOKEN" ]; then
    echo "✗ Not authenticated"
    echo "Run: gcloud auth application-default login"
    exit 1
fi
echo "✓ Access token obtained"

# Get project
PROJECT_ID=$(gcloud config get-value project 2>/dev/null || true)
if [ -z "$PROJECT_ID" ]; then
    echo "✗ No GCP project set"
    echo "Run: gcloud config set project PROJECT_ID"
    exit 1
fi
echo "✓ GCP project: $PROJECT_ID"

# Test API call (use global endpoint for newer models)
LOCATION="global"
URL="https://aiplatform.googleapis.com/v1/projects/$PROJECT_ID/locations/$LOCATION/publishers/google/models/$MODEL:generateContent"

RESPONSE=$(curl -s -X POST "$URL" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": {
      "role": "user",
      "parts": [
        {
          "text": "test"
        }
      ]
    }
  }')

# Check for errors
if echo "$RESPONSE" | grep -q '"error"'; then
    ERROR_CODE=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['error']['code'])" 2>/dev/null || echo "unknown")

    if [ "$ERROR_CODE" = "404" ]; then
        echo "✗ Model not found (404)"
        echo ""
        echo "Recommendation: Monitor for availability, check again in 1-2 weeks"
        echo "Note: New models are typically announced before Vertex AI rollout"
        echo "Note: Testing with global endpoint (required for Gemini 3+)"
        exit 1
    else
        echo "✗ Error: $ERROR_CODE"
        echo "$RESPONSE" | python3 -m json.tool
        exit 1
    fi
fi

echo "✓ Model is available in Vertex AI"
echo ""
echo "Ready to add to models.yml!"
