# New AI Models Status Report
**Date**: November 18, 2025
**Models Tested**: OpenAI GPT-5.1, Google Gemini 3 Pro

---

## Executive Summary

| Model | API Access | Ready for Eval Suite | Notes |
|-------|------------|----------------------|-------|
| **GPT-5.1** | ✅ **YES** | ✅ **YES** | Fully accessible, ready to add |
| **GPT-5.1 Instant** | ✅ **YES** | ✅ **YES** | Fully accessible, ready to add |
| **Gemini 3 Pro** | ❌ **NO** | ❌ **NOT YET** | Not available in Vertex AI yet |

---

## OpenAI GPT-5.1 - ✅ READY

### API Access Confirmed
- ✅ Both `gpt-5.1` and `gpt-5.1-chat-latest` are accessible
- ✅ API calls successful with proper authentication
- ✅ Released: Mid-November 2025
- ✅ Currently running in production

### API Model Names
```
gpt-5.1                  → resolves to gpt-5.1-2025-11-13 (Thinking mode with adaptive reasoning)
gpt-5.1-chat-latest      → GPT-5.1 Instant (faster, general purpose)
gpt-5.1-codex            → Extended for coding workloads
gpt-5.1-codex-mini       → Smaller codex variant
```

### Pricing (Same as GPT-5)
```
Input:  $1.25 per 1M tokens  ($0.00125 per 1K)
Output: $10.00 per 1M tokens ($0.01000 per 1K)
Cached: $0.125 per 1M tokens (90% discount)
```

### Key Features
1. **Adaptive Reasoning**: Dynamically adjusts thinking time based on task complexity
2. **Reasoning Tokens**: Shows separate `reasoning_tokens` count in usage stats (10 tokens in our test)
3. **New Parameters**:
   - Use `max_completion_tokens` instead of `max_tokens`
   - New `reasoning_effort` parameter (`none`, `low`, `medium`, `high`)
4. **Extended Caching**: Up to 24 hour cache retention
5. **New Developer Tools**: `apply_patch` and `shell` tools for code editing and command execution

### Recommended for Eval Suite
- ✅ **gpt-5.1** (main reasoning model) - Add to `extended_suite`
- ✅ **gpt-5.1-chat-latest** (instant mode) - Add to `dev_models` for faster iteration

---

## Google Gemini 3 Pro - ❌ NOT READY YET

### Current Status
- ❌ **Not available in Vertex AI** (404 error)
- ✅ Announced TODAY (November 18, 2025)
- ⏳ Preview access via Google AI Studio (web interface)
- ⏳ Vertex AI rollout expected soon (typically 1-2 weeks after announcement)

### Tested Model Names (All returned 404)
```
gemini-3-pro-preview-11-2025
gemini-3-pro
gemini-3
```

### Pricing (When available)
```
Context ≤200k tokens:
  Input:  $2.00 per 1M tokens  ($0.00200 per 1K)
  Output: $12.00 per 1M tokens ($0.01200 per 1K)

Context >200k tokens:
  Input:  $4.00 per 1M tokens  ($0.00400 per 1K)
  Output: $18.00 per 1M tokens ($0.01800 per 1K)

Context caching: Not yet announced
```

### Key Features (When available)
1. **State-of-the-art Reasoning**: 23.4% on MathArena Apex (new SOTA)
2. **Multimodal**: 81% MMMU-Pro, 87.6% Video-MMMU
3. **Coding**: 1487 Elo on WebDev Arena leaderboard
4. **SWE-bench**: 76.2% (code agents)
5. **Context Window**: 1M tokens (same as 2.5 Pro)
6. **Gemini 3 Deep Think Mode**: Coming soon for AI Ultra subscribers

### Recommendation
- ⏳ **Monitor for Vertex AI availability** (check weekly)
- ⏳ Add to `extended_suite` once available
- ⏳ Consider for `benchmark_suite` if performance justifies the higher cost

---

## Current Eval Harness Authentication

### How Gemini Access Works (No API Key Needed)
Your eval harness uses **Vertex AI** via Google Cloud authentication:

```bash
# Authentication method
gcloud auth application-default login

# Project configuration
gcloud config set project multivac-internal-prod

# Credentials obtained via
gcloud auth application-default print-access-token
```

**Files**: `internal/eval_harness/api_google.go` lines 122-152

This is why you don't need `GOOGLE_API_KEY` - you use Application Default Credentials (ADC) instead!

---

## Recommended Action Plan

### Immediate (Today)
1. ✅ Add GPT-5.1 models to `models.yml`
2. ✅ Update eval suite configurations
3. ✅ Test with a small benchmark run
4. ⏳ Document new model capabilities in teaching prompts

### Short-term (1-2 weeks)
1. ⏳ Monitor Vertex AI for Gemini 3 Pro availability
2. ⏳ Once available, add to `models.yml`
3. ⏳ Run comparative benchmarks (GPT-5.1 vs Gemini 3 Pro vs Claude Sonnet 4.5)

### Medium-term (1 month)
1. ⏳ Evaluate if GPT-5.1 should replace GPT-5 in default suites
2. ⏳ Assess cost/performance tradeoffs with adaptive reasoning
3. ⏳ Update dashboard with new model results

---

## Draft models.yml Updates

See `NEW_MODELS_YAML_DRAFT.yml` for ready-to-merge configuration.

---

## Test Results

### GPT-5.1 Test (Successful)
```json
{
  "id": "chatcmpl-CdM3PLjMECXcBrJKVy7hLSWsUD9mO",
  "model": "gpt-5.1-2025-11-13",
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 10,
    "total_tokens": 23,
    "completion_tokens_details": {
      "reasoning_tokens": 10  // <-- Adaptive reasoning!
    }
  }
}
```

### Gemini 3 Pro Test (Not Found)
```json
{
  "error": {
    "code": 404,
    "message": "Publisher Model `projects/.../models/gemini-3-pro-preview-11-2025` not found."
  }
}
```

### Gemini 2.5 Pro Test (Working)
```json
{
  "candidates": [{"content": {"text": "Hello there! How can I help you today?"}}],
  "usageMetadata": {
    "promptTokenCount": 2,
    "candidatesTokenCount": 10,
    "totalTokenCount": 704
  }
}
```

---

## Questions to Consider

1. **Should we replace GPT-5 with GPT-5.1 in default suites?**
   - GPT-5.1 has same pricing but better performance
   - Adaptive reasoning may reduce costs on simple tasks

2. **Which GPT-5.1 variant for dev work?**
   - `gpt-5.1-chat-latest` (Instant) for faster iteration?
   - `gpt-5.1` (Thinking) for higher quality?

3. **When Gemini 3 Pro arrives, how to integrate?**
   - Add to extended_suite immediately?
   - Run cost/benefit analysis first?
   - 60% more expensive than Gemini 2.5 Pro - is it worth it?

---

**Generated**: 2025-11-18
**Tested by**: Claude Code
**Test scripts**: `test_new_models.sh`, `/tmp/test_gpt51.sh`, `/tmp/test_gemini3_vertex.sh`
