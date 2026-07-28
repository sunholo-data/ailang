# Model Pricing Guide

How to find, verify, and convert pricing information for AI models.

## Finding Official Pricing

### OpenAI
**Pricing page**: https://openai.com/api/pricing/

**How to find**:
1. Go to OpenAI Platform pricing page
2. Find model in the list (e.g., "GPT-5.1")
3. Note input and output prices per 1M tokens
4. Check for cached token discounts

**Example** (GPT-5.1):
```
Input:  $1.25 per 1M tokens
Output: $10.00 per 1M tokens
Cached: $0.125 per 1M tokens (90% discount)
```

### Anthropic
**Pricing page**: https://www.anthropic.com/pricing

**How to find**:
1. Go to Anthropic pricing page
2. Find model family (e.g., "Claude Sonnet 4.5")
3. Note input and output prices per 1M tokens
4. Check for prompt caching discounts

**Example** (Claude Sonnet 4.5):
```
Input:  $3.00 per 1M tokens
Output: $15.00 per 1M tokens
```

### Google Gemini
**Pricing page**: https://ai.google.dev/pricing

**How to find**:
1. Go to Google AI pricing page
2. Find model (e.g., "Gemini 2.5 Pro")
3. Note context-dependent pricing (≤200k vs >200k)
4. Check for caching discounts

**Example** (Gemini 2.5 Pro):
```
Context ≤200k:
  Input:  $1.25 per 1M tokens
  Output: $10.00 per 1M tokens

Context >200k:
  Input:  $2.50 per 1M tokens
  Output: $15.00 per 1M tokens
```

---

## Price Conversion

**models.yml uses price per 1K tokens**, not per 1M.

**Conversion formula**:
```
Price per 1K = (Price per 1M) / 1000
```

**Examples**:

| Per 1M | Per 1K (models.yml) |
|--------|---------------------|
| $1.25  | 0.00125             |
| $10.00 | 0.01                |
| $0.25  | 0.00025             |
| $2.00  | 0.002               |

**Quick reference**:
```bash
# Calculate per 1K from per 1M
$ python3 -c "print(1.25 / 1000)"  # Input price
0.00125

$ python3 -c "print(10.0 / 1000)"  # Output price
0.01
```

---

## Cost Calculation Verification

After adding a model, verify cost calculation works:

**Test script**:
```bash
# Run test benchmark
.claude/skills/model-manager/scripts/run_test_benchmark.sh <model-name>

# Check output for cost
# Should show: "✓ Cost: $0.XXX"
```

**Manual verification**:
```python
# Example for GPT-5.1
input_tokens = 245
output_tokens = 89

input_cost = (245 / 1000) * 0.00125   # $0.00031
output_cost = (89 / 1000) * 0.01      # $0.00089
total_cost = input_cost + output_cost  # $0.00120
```

**Check against AILANG cost tracking**:
```bash
# Look for cost in benchmark result JSON
cat eval_results/.../fizzbuzz_ailang_<model>_*.json | jq '.cost'
```

---

## Context Caching

Some models offer reduced pricing for cached tokens.

### OpenAI (Prompt Caching)
**Applies to**: GPT-5, GPT-5.1
**Discount**: 90% (e.g., $1.25 → $0.125 per 1M)
**Retention**: Up to 24 hours (GPT-5.1), 5 minutes (GPT-5)
**Notes**: Automatic when prompt prefix is reused

### Anthropic (Prompt Caching)
**Applies to**: Claude Sonnet/Opus models
**Discount**: ~90% (check current pricing)
**Retention**: 5 minutes
**Notes**: Requires explicit cache control headers

### Google (Context Caching)
**Applies to**: Gemini 2.5 Pro, Gemini 2.5 Flash
**Discount**: 90% (e.g., $1.25 → $0.125 per 1M for ≤200k)
**Retention**: Configurable (up to 1 hour default)
**Notes**: Minimum 2048 tokens to enable caching

**Important**: AILANG eval harness does not yet track cached token costs separately. All costs use standard pricing.

---

## Special Pricing Features

### GPT-5.1 Adaptive Reasoning
- Reasoning tokens charged at output token rate
- Reported separately in `completion_tokens_details.reasoning_tokens`
- Can vary based on task complexity (adaptive)

### Gemini Context-Dependent Pricing
- Different rates for ≤200k vs >200k context
- models.yml uses ≤200k pricing by default
- Note >200k pricing in model notes section

### Anthropic Message Batching
- Discounts available for batch API (not used in eval harness)
- Standard pricing applies for streaming/synchronous calls

---

## Verifying Published Pricing

When adding a new model, cross-reference multiple sources:

1. **Official pricing page** (authoritative)
2. **API documentation** (may include pricing)
3. **Release announcement** (often mentions pricing)
4. **Community resources** (verify against official)

**Red flags**:
- ⚠️ Third-party sites may have outdated pricing
- ⚠️ Beta/preview pricing may change at GA
- ⚠️ Regional pricing variations (models.yml assumes US pricing)
- ⚠️ Volume discounts not reflected in published rates

**When in doubt**:
- Test with a small API call
- Check token usage and compare expected cost
- Monitor actual billing in provider dashboard

---

## Example: Adding GPT-5.1

**Step 1**: Find pricing
- Go to https://openai.com/api/pricing/
- Find "GPT-5.1": $1.25 input, $10.00 output per 1M

**Step 2**: Convert to per 1K
- Input: 1.25 / 1000 = 0.00125
- Output: 10.00 / 1000 = 0.01

**Step 3**: Add to models.yml
```bash
.claude/skills/model-manager/scripts/update_models_yml.sh \
  gpt5-1 \
  "gpt-5.1" \
  openai \
  0.00125 \
  0.01 \
  "GPT-5.1 with adaptive reasoning"
```

**Step 4**: Verify cost calculation
```bash
.claude/skills/model-manager/scripts/run_test_benchmark.sh gpt5-1

# Expected output:
# ✓ Cost: $0.002  (approximately, depends on token usage)
```

---

## Cost Optimization Tips

**For development** (use cheap models):
- `dev_models` suite: gpt5-mini, claude-haiku-4-5, gemini-2-5-flash
- ~5x cheaper than flagship models
- Good enough for quick iteration

**For benchmarks** (use balanced models):
- `benchmark_suite`: gpt5-1, claude-sonnet-4-5, gemini-2-5-pro
- Flagship models for quality
- Moderate cost

**For full evals** (be prepared for cost):
- `extended_suite`: All 6+ models
- Comprehensive coverage
- ~2-3x cost of benchmark suite

**Example costs** (approximate, for full eval suite):
```
dev_models (3 models, 50 benchmarks):    ~$5-10
benchmark_suite (3 models, 50 benchmarks): ~$15-25
extended_suite (6 models, 50 benchmarks):  ~$30-50
```

---

## Troubleshooting

### Cost shows $0.00
**Possible causes**:
- Model not in models.yml pricing config
- Token usage not reported by API
- Cost calculation bug

**Fix**:
1. Check model exists in `internal/eval_harness/models.yml`
2. Check pricing fields are set (`input_per_1k`, `output_per_1k`)
3. Verify token usage in API response

### Cost seems too high/low
**Possible causes**:
- Pricing conversion error (per 1M vs per 1K)
- Wrong model selected
- Context caching applied (reduces cost)

**Fix**:
1. Manually calculate expected cost
2. Compare with API response token usage
3. Check provider billing dashboard for actual charges
