#!/usr/bin/env bash
# Test API access to a model
# Usage: test_model_access.sh <provider> <model-name>

set -euo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <provider> <model-name>"
    echo ""
    echo "Providers: openai, anthropic, google"
    echo ""
    echo "Examples:"
    echo "  $0 openai gpt-5.1"
    echo "  $0 anthropic claude-sonnet-4-5-20250929"
    echo "  $0 google gemini-3-pro-preview-11-2025"
    exit 1
fi

PROVIDER="$1"
MODEL="$2"

echo "Testing: $PROVIDER/$MODEL"
echo ""

# Test based on provider
case "$PROVIDER" in
    openai)
        # Check API key
        if [ -z "${OPENAI_API_KEY:-}" ]; then
            echo "✗ OPENAI_API_KEY not set"
            exit 1
        fi
        echo "✓ OPENAI_API_KEY found"

        # Test API call
        RESPONSE=$(curl -s https://api.openai.com/v1/chat/completions \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $OPENAI_API_KEY" \
          -d "{
            \"model\": \"$MODEL\",
            \"messages\": [{\"role\": \"user\", \"content\": \"Say hello in exactly 3 words\"}],
            \"max_completion_tokens\": 10
          }")

        # Check for errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            echo "✗ API call failed"
            echo "$RESPONSE" | python3 -m json.tool
            exit 1
        fi

        echo "✓ API call successful"

        # Extract info
        ACTUAL_MODEL=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['model'])")
        INPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usage']['prompt_tokens'])")
        OUTPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usage']['completion_tokens'])")

        echo "✓ Model: $ACTUAL_MODEL"
        echo "✓ Tokens: $INPUT_TOKENS input, $OUTPUT_TOKENS output"

        # Check for reasoning tokens (GPT-5.1+)
        REASONING_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usage'].get('completion_tokens_details', {}).get('reasoning_tokens', 0))" 2>/dev/null || echo "0")
        if [ "$REASONING_TOKENS" != "0" ]; then
            echo "✓ Reasoning tokens: $REASONING_TOKENS (adaptive reasoning enabled)"
        fi

        echo ""
        echo "✓ Ready to add to models.yml"
        ;;

    anthropic)
        # Check API key
        if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
            echo "✗ ANTHROPIC_API_KEY not set"
            exit 1
        fi
        echo "✓ ANTHROPIC_API_KEY found"

        # Test API call
        RESPONSE=$(curl -s https://api.anthropic.com/v1/messages \
          -H "Content-Type: application/json" \
          -H "x-api-key: $ANTHROPIC_API_KEY" \
          -H "anthropic-version: 2023-06-01" \
          -d "{
            \"model\": \"$MODEL\",
            \"messages\": [{\"role\": \"user\", \"content\": \"Say hello in exactly 3 words\"}],
            \"max_tokens\": 10
          }")

        # Check for errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            echo "✗ API call failed"
            echo "$RESPONSE" | python3 -m json.tool
            exit 1
        fi

        echo "✓ API call successful"

        # Extract info
        ACTUAL_MODEL=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['model'])")
        INPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usage']['input_tokens'])")
        OUTPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usage']['output_tokens'])")

        echo "✓ Model: $ACTUAL_MODEL"
        echo "✓ Tokens: $INPUT_TOKENS input, $OUTPUT_TOKENS output"
        echo ""
        echo "✓ Ready to add to models.yml"
        ;;

    google)
        # Check gcloud auth
        if ! command -v gcloud &> /dev/null; then
            echo "✗ gcloud CLI not found"
            echo "Install: https://cloud.google.com/sdk/docs/install"
            exit 1
        fi
        echo "✓ gcloud CLI found"

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

        # Test API call (Vertex AI)
        # Try global endpoint first (required for newer models like Gemini 3)
        LOCATION="global"
        URL="https://aiplatform.googleapis.com/v1/projects/$PROJECT_ID/locations/$LOCATION/publishers/google/models/$MODEL:generateContent"

        echo "✓ Testing with global endpoint"

        RESPONSE=$(curl -s -X POST "$URL" \
          -H "Authorization: Bearer $ACCESS_TOKEN" \
          -H "Content-Type: application/json" \
          -d '{
            "contents": {
              "role": "user",
              "parts": [
                {
                  "text": "Say hello in exactly 3 words"
                }
              ]
            }
          }')

        # Check for errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            ERROR_CODE=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['error']['code'])" 2>/dev/null || echo "unknown")
            echo "✗ API call failed (error code: $ERROR_CODE)"

            if [ "$ERROR_CODE" = "404" ]; then
                echo ""
                echo "Model not available in Vertex AI yet."
                echo "Recommendation: Check again in 1-2 weeks"
                echo "Note: Newer models (Gemini 3+) require global endpoint"
            else
                echo "$RESPONSE" | python3 -m json.tool
            fi
            exit 1
        fi

        echo "✓ API call successful"

        # Extract info
        INPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usageMetadata']['promptTokenCount'])")
        OUTPUT_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usageMetadata']['candidatesTokenCount'])")

        echo "✓ Model: $MODEL"
        echo "✓ Tokens: $INPUT_TOKENS input, $OUTPUT_TOKENS output"

        # Check for reasoning tokens (Gemini 3+)
        REASONING_TOKENS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['usageMetadata'].get('thoughtsTokenCount', 0))" 2>/dev/null || echo "0")
        if [ "$REASONING_TOKENS" != "0" ]; then
            echo "✓ Reasoning tokens: $REASONING_TOKENS (adaptive reasoning enabled)"
        fi

        echo ""
        echo "✓ Ready to add to models.yml"
        ;;

    *)
        echo "✗ Unknown provider: $PROVIDER"
        echo "Supported providers: openai, anthropic, google"
        exit 1
        ;;
esac
