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
    "model": "claude-sonnet-4-6",
    "max_tokens": 10,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

**Important notes**:
- Uses `x-api-key` header, NOT Bearer token
- Requires `anthropic-version` header (2023-06-01 is current)
- System prompt goes in a separate `system` field, not a message role

**Documentation**: https://docs.anthropic.com/en/api/messages

---

## Google Gemini (Vertex AI)

**Endpoint**: `https://aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/publishers/google/models/MODEL:generateContent`

**Authentication**: OAuth2 via `gcloud auth application-default login` (Bearer access token)

**Requirements**:
- `gcloud` CLI installed and authenticated
- GCP project set: `gcloud config set project PROJECT_ID`

**Test command**:
```bash
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
curl -s -X POST "https://aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/global/publishers/google/models/gemini-3-pro:generateContent" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": {
      "role": "user",
      "parts": [{"text": "Hello"}]
    }
  }'
```

**Important notes**:
- Gemini 3+ requires the `global` location
- Reasoning tokens in `usageMetadata.thoughtsTokenCount`
- Model availability varies by region/project allowlist

**Documentation**: https://cloud.google.com/vertex-ai/docs

---

## Ollama Cloud (flat-rate open-weight route)

Cloud-hosted open-weight inference that **rides the local ollama daemon** — the naming
convention IS the code path. Canonical design:
[design_docs/planned/v0_34_0/m-ollama-cloud-provider.md](../../../design_docs/planned/v0_34_0/m-ollama-cloud-provider.md).

**Endpoints**:
| Purpose | Endpoint | Auth |
|---|---|---|
| Catalogue (which models exist) | `GET https://ollama.com/v1/models` | none (200 unauthenticated) |
| INFERENCE | `POST http://localhost:11434/v1/chat/completions` with model `"<tag>:cloud"` | **device key** via `ollama signin` (the daemon proxies to ollama.com; no env var involved) |
| Quota gauge | `GET https://ollama.com/api/usage` | Bearer `OLLAMA_API_KEY` (**gauge only** — the local daemon does NOT proxy this route, V24) |

**Test command** (inference through the daemon, per V21):
```bash
curl -s http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "glm-5.3-flash:cloud",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 2000
  }'
```

**Quota gauge**:
```bash
curl -s https://ollama.com/api/usage -H "Authorization: Bearer $OLLAMA_API_KEY" \
  | jq '{session: .limits.session.usage, weekly: .limits.weekly.usage,
         by_model: .limits.session.models}'
```

**Important notes**:
- `OLLAMA_API_KEY` is **only** for `/api/usage`. Direct Bearer inference against
  ollama.com is NOT wired (D3 future work) — inference goes through the local daemon.
- `/api/usage` is a **numerator with no denominator**: `limits.{session,weekly}.usage`
  is a coarse level, no limit/remaining/reset_at field is published (V26), so a
  pre-flight "refuse to start if quota is low" gate is unbuildable.
- `activity.cost` always reports `"0.00000"` — flat-rate subscription, which is exactly
  why models.yml pricing is IMPUTED from the OpenRouter twin (D1) and banks as
  `list-price-equivalent`, never `0/0` (that maps to the false `free-local` provenance).
- Model weights × (input, cached input, output) tokens meter at **usage levels 1–4**
  (`gpt-oss:20b` = 1, `deepseek-v4-pro` = 4). Session limit resets 5h; weekly 7d.
  Tiers (V9): Free = 1 concurrent model, Pro = 3, Max = 10 (Max paused for new subs).
- The response `model` field is the BASE name (suffix stripped by the proxy, V21) and
  ollama's ollama provider historically reported 0 tokens from the native API path —
  the `/v1` path returns standard OpenAI usage shape (V27, fixed).
- Reasoning models can burn the whole output budget on thinking at low `max_tokens`
  (V25) — same lesson as the GLM-5.2 truncation; probe with ≥2000.
- Concurrency: cloud rows are EXEMPT from the single-GPU serial clamp (D4, they load
  nothing on the GPU), but ANY motoko row still serializes on its fixed backend port.

**Documentation**: https://docs.ollama.com/cloud · https://ollama.com/pricing (V9)

---

## Common Errors

### 401 Unauthorized
- OpenAI/Anthropic/Google: key missing, revoked, or wrong header shape
- Ollama Cloud: not signed in (`ollama signin`) for inference; bad/expired
  `OLLAMA_API_KEY` for the gauge

### 403 Forbidden
- Key lacks access to the model or region
- Google: model not allowlisted for your project

### 404 Not Found
- Model not available yet (OpenAI/Vertex: check again in 1-2 weeks)
- Ollama Cloud: model not in `GET https://ollama.com/v1/models`
- Ollama `/api/usage` 404 on the LOCAL daemon: wrong host — the gauge lives on
  ollama.com only (V24)

### 429 Rate Limit
- Transient burst limit, distinct from account cap
- Ollama Cloud over-concurrency: requests are **queued, not rejected**

## Pricing Endpoints

- OpenAI: https://openai.com/api/pricing/
- Anthropic: https://www.anthropic.com/api#pricing
- Google: https://cloud.google.com/vertex-ai/pricing
- OpenRouter (all vendors, machine-readable): `GET https://openrouter.ai/api/v1/models` —
  used by `make verify-model-pricing` to diff models.yml rates against the live catalogue
- Ollama: subscription tiers (Free/Pro/Max) — no per-token price published; impute
  from the OpenRouter twin per D1