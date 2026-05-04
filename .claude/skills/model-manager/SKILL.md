---
name: model-manager
description: Test, validate, and add new AI models to the eval suite. Use when user asks to add new models, test model access, check pricing, or update models.yml.
---

# Model Manager

Test API access, validate configurations, and add new AI models to the AILANG eval suite.

## Quick Start

**Most common usage:**
```bash
# User says: "Can we add GPT-5.1 to the eval suite?"
# This skill will:
# 1. Test API access to GPT-5.1
# 2. Find the correct API model name
# 3. Look up pricing information
# 4. Update models.yml configuration
# 5. Run a test benchmark to verify
```

## When to Use This Skill

Invoke this skill when:
- User asks to "add a new model" to eval suite
- User mentions checking if a model is "accessible" or "available"
- User wants to "test API access" to a model
- User asks to "update models.yml" or "check pricing"
- User says "can we use [model name]?" for evaluations

## Available Scripts

### `scripts/test_model_access.sh <provider> <model-name>`
Test API access to a model and display authentication status.

**Usage:**
```bash
# Test OpenAI model
scripts/test_model_access.sh openai gpt-5.1

# Test Anthropic model
scripts/test_model_access.sh anthropic claude-sonnet-4-5-20250929

# Test Google Gemini via Vertex AI
scripts/test_model_access.sh google gemini-3-pro-preview-11-2025
```

**Output:**
```
Testing: openai/gpt-5.1
✓ OPENAI_API_KEY found
✓ API call successful
✓ Model: gpt-5.1-2025-11-13
✓ Tokens: 13 input, 10 output (10 reasoning)
Ready to add to models.yml
```

### `scripts/find_model_info.sh <model-keywords>`
Search for model information using web search and return API names + pricing.

**Usage:**
```bash
# Find GPT-5.1 info
scripts/find_model_info.sh "GPT-5.1 API model name pricing"

# Find Gemini 3 Pro info
scripts/find_model_info.sh "Gemini 3 Pro API documentation"
```

**Output:**
```
Searching for: GPT-5.1 API model name pricing
✓ Found API names:
  - gpt-5.1 (Thinking mode)
  - gpt-5.1-chat-latest (Instant mode)
✓ Pricing:
  Input: $1.25 per 1M tokens
  Output: $10.00 per 1M tokens
  Cached: $0.125 per 1M tokens
```

### `scripts/update_models_yml.sh <friendly-name> <api-name> <provider> <input-price> <output-price>`
Add a new model to models.yml configuration.

**Usage:**
```bash
# Add GPT-5.1
scripts/update_models_yml.sh \
  gpt5-1 \
  "gpt-5.1" \
  openai \
  0.00125 \
  0.01
```

**Output:**
```
Adding model to models.yml:
  Friendly name: gpt5-1
  API name: gpt-5.1
  Provider: openai
  Pricing: $0.00125 / $0.01 per 1K tokens

✓ Updated models.yml
✓ Validated YAML syntax
✓ Ready to test
```

### `scripts/verify_vertex_model.sh <model-name>`
Check if a Gemini model is available in Vertex AI.

**Usage:**
```bash
# Check if Gemini 3 Pro is available
scripts/verify_vertex_model.sh gemini-3-pro-preview-11-2025
```

**Output:**
```
Checking Vertex AI for: gemini-3-pro-preview-11-2025
✓ GCP project: multivac-internal-prod
✓ Access token obtained
✗ Model not found (404)
Recommendation: Monitor for availability, check again in 1-2 weeks
```

### `scripts/run_test_benchmark.sh <model-name>`
Run a small test benchmark to verify model works end-to-end.

**Usage:**
```bash
# Test GPT-5.1 with fizzbuzz benchmark
scripts/run_test_benchmark.sh gpt5-1
```

**Output:**
```
Running test benchmark: fizzbuzz
Model: gpt5-1
✓ Benchmark completed
✓ Result: PASS (100%)
✓ Tokens: 245 input, 89 output
✓ Cost: $0.002
Model is ready for production use
```

## Workflow

### 1. Test API Access

**First, verify you can call the model:**

```bash
# Use test_model_access.sh
scripts/test_model_access.sh openai gpt-5.1
```

**What to check:**
- API key is set (OPENAI_API_KEY, ANTHROPIC_API_KEY, or gcloud auth)
- API call succeeds (not 401/403/404)
- Model returns expected structure
- Token usage is reported

**For Gemini models:**
- Uses Vertex AI (not public API)
- Requires `gcloud auth application-default login`
- Check availability with `verify_vertex_model.sh`

### 2. Find Model Information

**Search for official documentation:**

```bash
# Find API model name and pricing
scripts/find_model_info.sh "GPT-5.1 API documentation pricing"
```

**What to gather:**
- Exact API model name (e.g., `gpt-5.1` not `GPT-5.1`)
- Provider (openai, anthropic, google)
- Input price per 1K tokens
- Output price per 1K tokens
- Context limits (if relevant)
- Special features (adaptive reasoning, caching, etc.)

**Reference:** See [resources/provider_endpoints.md](resources/provider_endpoints.md)

### 3. Update models.yml

**Add the model configuration:**

```bash
# Add to models.yml
scripts/update_models_yml.sh \
  <friendly-name> \
  <api-name> \
  <provider> \
  <input-per-1k> \
  <output-per-1k>
```

**Naming conventions:**
- Friendly name: `gpt5-1`, `claude-sonnet-4-5`, `gemini-3-pro`
- API name: Exact string for API calls
- Use hyphens, lowercase

**Also update:**
- Model suites (`benchmark_suite`, `extended_suite`, `dev_models`)
- Add notes about special features
- Document agent CLI support (if available)

### 4. Run Test Benchmark

**Verify end-to-end:**

```bash
# Test with a simple benchmark
scripts/run_test_benchmark.sh <model-name>
```

**What to verify:**
- Benchmark completes successfully
- Results are reasonable (not garbage output)
- Token usage matches expectations
- Cost calculation works
- No errors in logs

### 5. Apply the Smoke-Test Gate (HARD RULE)

**Rule of thumb (project-wide):**
> **Rule out adding a model to our eval suite if it can't pass ALL the smoke tests.**

A "smoke test" is a small set (typically 3 benchmarks) that established proprietary
frontier models (claude-sonnet-4-6, gpt5-mini) pass cleanly. If those pass and a
candidate model fails any of them, the candidate doesn't enter the eval rotation —
the failure is on the model, not the harness or the benchmark.

**Standard smoke set (May 2026):**
- `fizzbuzz` — control-flow + simple I/O
- `adt_option` — AILANG ADT pattern matching (language-specific)
- `csv_to_json_converter` — string parsing + records (medium complexity)

**Run smoke against a candidate:**
```bash
ailang eval-suite \
  --models <candidate>,claude-sonnet-4-6 \
  --benchmarks fizzbuzz,adt_option,csv_to_json_converter \
  --langs ailang \
  --output /tmp/smoke_<candidate> \
  --parallel 2

# Tabulate pass/fail
for f in /tmp/smoke_<candidate>/standard/*.json; do
  name=$(basename "$f" .json | sed 's/_[0-9]*$//')
  jq -r --arg name "$name" '"\($name)\t\(if .compile_ok and .runtime_ok and .stdout_ok then "PASS" else "FAIL" end)\t\(.err_code // "—")"' "$f"
done | column -ts $'\t'
```

**Decision tree:**
1. **claude-sonnet-4-6 fails any benchmark** — smoke set is broken; fix the
   benchmark before evaluating candidates.
2. **Candidate fails all 3** — CUT. Do not add to models.yml. Note the failure
   types in the cut commit message (WRONG_LANG, syntax, runtime, wrong-output)
   for future reference.
3. **Candidate fails 1 of 3 (2/3 pass)** — NEAR-MISS. Optionally keep with a
   "near-miss" comment block in models.yml (see precedent: `or-gemma-4-26b`,
   `or-qwen3-coder-flash`). Re-run periodically; if it starts passing all 3,
   that's a signal stdlib/prompt has improved.
4. **Candidate passes all 3** — proceed to step 6 (Document) and add to
   models.yml normally.

**Failure-mode taxonomy** (worth capturing in the cut commit message):

| Failure | Meaning | Likely cause |
|---------|---------|--------------|
| `WRONG_LANG` | Model produced Python/JS instead of AILANG | Prompt-following gap; small/MoE models lose plot at 23k-token system prompt |
| `syntax-error` (no WRONG_LANG) | Invented AILANG syntax (e.g. `let rec`, `\n.` lambda) | Model hasn't seen enough AILANG in training |
| `wrong-output` | Compiled and ran, wrong stdout | Spec-following gap, not language gap |
| `runtime-error` | Compiled, crashed at runtime | Logic bug |

**2026-05-04 finding (precedent):** Tested 6 SOTA OS models (Gemma 4 26B, Qwen3
30B-A3B, Qwen3 235B-A22B, DeepSeek V4 Flash, Kimi K2.6, Qwen3 Coder Flash)
against this smoke set. Proprietary baselines passed 3/3; **zero OS models
passed all 3**. Most common failure: WRONG_LANG (model produced Python). Even
frontier-class OS models fall back on training-corpus patterns when given
AILANG's 23k-token teaching prompt — they've seen plenty of Python but very
little AILANG. Two near-misses (`or-gemma-4-26b`, `or-qwen3-coder-flash`)
retained on the watchlist; rest cut.

**Implication for stdlib/prompt work:** the smoke test doubles as a
language-improvement metric. Re-run it after stdlib changes or prompt
revisions; if the near-miss watchlist starts passing the third benchmark, the
language has become more "trainable-feel."

**Caveat — agent mode is a separate gate:** the smoke set above runs in
**standard** (single-shot API generation) mode. Models that fail standard mode
may still perform usefully in **agent** mode (`--agent` flag, opencode/pi
harnesses) where they get multi-turn iteration. If a candidate fails standard
smoke, run `ailang eval-suite --agent --models <candidate> ...` separately
before fully cutting it. Agent mode results don't override the standard-mode
gate but can justify adding the model under a different harness entry (e.g.
`opencode-<candidate>`, `pi-<candidate>`).

**2026-05-04 agent-mode smoke finding (precedent):** Tested 9 OS-via-OR
candidates through opencode harness. Cross-mode behaviour:

| Model | Standard | Agent | Δ |
|-------|---------:|------:|--:|
| **GLM 5** (z.ai) | not tested | **3/3** ✅ | — first OS model to pass |
| Gemma 4 26B | 2/3 | 2/3 | 0 (same near-miss) |
| DeepSeek V4 Flash | 0/3 | 2/3 | **+2** (agent unlock) |
| GLM 4.7 Flash | not tested | 2/3 | — near-miss |
| Kimi K2.6 | 1/3 | 1/3 | 0 |
| Qwen3 30B-A3B | 1/3 | 1/3 | 0 |
| Qwen3 Coder Flash | 2/3 | 1/3 | **-1** (agent regressed) |
| DeepSeek V4 Pro | not tested | 1/3 | Pro under-performed Flash |
| Qwen3 235B-A22B | 0/3 | 0/3 | 0 |

Key takeaways for the model-manager workflow:

1. **Agent mode is not a universal fix.** Most models that fail standard
   smoke also fail agent smoke. Multi-turn helps when the model can read
   compile errors and adjust; it hurts when the model interprets tool-call
   setup as the answer (Qwen3 Coder Flash regression).

2. **Pro tier ≠ better.** DeepSeek V4 Pro (1/3) under-performed V4 Flash
   (2/3) on AILANG smoke. The Pro reasoning/long-output overhead can hurt
   simple-task accuracy. Test both tiers when available.

3. **csv_to_json_converter is the gating benchmark.** Of the 27 benchmark
   runs (9 models × 3), csv_to_json was the single most-failed test — only
   GLM 5 passed it among OS candidates. v0.14.2 baseline confirms:
   gpt5-4-mini and gemini-3-1-pro also fail csv_to_json, while
   claude-sonnet-4-6, claude-opus-4-7, gpt5-5, gemini-3-flash all pass.
   Use csv_to_json as the highest-signal smoke when running a quick test.

4. **GLM 5 is genuinely cost-competitive frontier OS.** $0.60/$2.08 per 1M
   tokens, ~5–7× cheaper than Claude Sonnet 4.6 on input. Worth standing
   inclusion in eval rotation alongside frontier proprietary models.

5. **Vendor-prefix wiring is forward-compat infrastructure.** When adding
   models from a new vendor (e.g. `z-ai/`, `moonshotai/`, `microsoft/`),
   add the prefix to `internal/ai/config.go::openrouterVendorPrefixes` so
   future ad-hoc `ailang run --ai vendor/model` invocations work without
   needing a models.yml entry.

### 6. Document the Model

**Update relevant documentation:**
- Add model to this skill's resource guide
- Note any special parameters (e.g., `max_completion_tokens` for GPT-5.1)
- Document authentication requirements
- Add to teaching prompts if needed

### 7. Optional: Run Full Eval

**If model looks good:**

```bash
# Run small eval suite
ailang eval-suite --models <model-name> --benchmarks fizzbuzz,recursion_factorial

# Run full suite (expensive!)
make eval-baseline EVAL_VERSION=vX.Y.Z FULL=true
```

## Resources

### Provider Endpoints
See [resources/provider_endpoints.md](resources/provider_endpoints.md) for:
- API endpoint URLs for each provider
- Authentication methods
- How to test access manually
- Common errors and fixes

### Pricing Guide
See [resources/pricing_guide.md](resources/pricing_guide.md) for:
- How to find official pricing
- Price conversion (per 1M → per 1K)
- Cost calculation verification
- Caching and discounts

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow and script descriptions)
2. **Execute as needed**: Scripts in `scripts/` (testing, updating, verification)
3. **Load on demand**: Resources (detailed endpoint docs, pricing references)

## Notes

**Important:**
- Always test API access BEFORE updating models.yml
- Vertex AI (Gemini) requires gcloud auth, not API key
- GPT-5.1+ uses `max_completion_tokens` instead of `max_tokens`
- New models may not be available in all regions immediately
- Check for preview/beta status before adding to production suites

**Prerequisites:**
- API keys set in environment (OPENAI_API_KEY, ANTHROPIC_API_KEY)
- For Gemini: `gcloud` CLI installed and authenticated
- For Gemini: GCP project set (`gcloud config set project PROJECT_ID`)
- `curl`, `python3`, and `jq` available in PATH

**Files modified by this skill:**
- `internal/eval_harness/models.yml` - Model configurations
- (Optional) `prompts/vX.Y.Z.md` - Teaching prompts
- (Optional) `.claude/skills/model-manager/resources/` - Local model database
