# Provider API Endpoints

Quick reference for testing AI model access across different providers.

## OpenAI

**Endpoint**: `https://api.openai.com/v1/chat/completions`

**Authentication**: Bearer token via `OPENAI_API_KEY` environment variable

**Test command**:
```bash
curl -s https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-5.1",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_completion_tokens": 10
  }'
```

**Important notes**:
- GPT-5.1+ uses `max_completion_tokens` instead of `max_tokens`
- Reasoning tokens reported separately in `completion_tokens_details.reasoning_tokens`
- Model name resolves to dated version (e.g., `gpt-5.1-2025-11-13`)

**Documentation**: https://platform.openai.com/docs/api-reference

---

## Anthropic

**Endpoint**: `https://api.anthropic.com/v1/messages`

**Authentication**: API key via `x-api-key` header (not Bearer token!)

**Test command**:
```bash
curl -s https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 10
  }'
```

**Important notes**:
- API key goes in `x-api-key` header, not `Authorization`
- Requires `anthropic-version` header
- Model names include full date suffix (e.g., `claude-sonnet-4-5-20250929`)
- Token fields: `input_tokens`, `output_tokens` (not `prompt_tokens`/`completion_tokens`)

**Documentation**: https://docs.anthropic.com/en/api/getting-started

---

## Google Gemini (Vertex AI)

**Endpoint**: `https://{REGION}-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/{REGION}/publishers/google/models/{MODEL}:generateContent`

**Authentication**: OAuth2 via `gcloud` Application Default Credentials

**Setup**:
```bash
# Install gcloud
# https://cloud.google.com/sdk/docs/install

# Authenticate
gcloud auth application-default login

# Set project
gcloud config set project YOUR_PROJECT_ID
```

**Test command**:
```bash
# Get access token
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
PROJECT_ID=$(gcloud config get-value project)
REGION="us-central1"
MODEL="gemini-2.5-pro"

curl -s -X POST \
  "https://$REGION-aiplatform.googleapis.com/v1/projects/$PROJECT_ID/locations/$REGION/publishers/google/models/$MODEL:generateContent" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Hello"}]
    }]
  }'
```

**Important notes**:
- Uses Vertex AI, not public Gemini API
- No API key needed - uses `gcloud` authentication
- Token fields: `promptTokenCount`, `candidatesTokenCount`, `totalTokenCount`
- New models may not be available immediately (typically 1-2 weeks after announcement)
- Check availability: Look for 404 errors

**Documentation**: https://cloud.google.com/vertex-ai/docs/generative-ai/model-reference/gemini

---

## Common Errors

### 401 Unauthorized
- **OpenAI**: Check `OPENAI_API_KEY` is set
- **Anthropic**: Check `ANTHROPIC_API_KEY` is set
- **Google**: Run `gcloud auth application-default login`

### 403 Forbidden
- **OpenAI**: API key may be invalid or quota exceeded
- **Anthropic**: API key invalid or account issue
- **Google**: Check GCP project permissions

### 404 Not Found
- **OpenAI**: Model name incorrect (check for typos)
- **Anthropic**: Model name incorrect (check date suffix)
- **Google**: Model not yet available in Vertex AI (check again in 1-2 weeks)

### 429 Rate Limit
- Wait and retry with exponential backoff
- Consider upgrading API tier (OpenAI)
- Check quota limits (Google Cloud Console)

---

## Pricing Endpoints

**OpenAI**: https://openai.com/api/pricing/
**Anthropic**: https://www.anthropic.com/pricing
**Google**: https://ai.google.dev/pricing

Convert pricing:
- **Per 1M tokens** → **Per 1K tokens**: Divide by 1000
- Example: $1.25 per 1M = $0.00125 per 1K
