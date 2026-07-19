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

### 2a. Reasoning-Model Check (REQUIRED before gating any verdict)

**The GLM-5.2 lesson (v0.30.0, 2026-07-19):** GLM-5.2 was rejected in June as "worse
than 5.1" — but the regression was OUR truncation: its always-on thinking phase
(28–32K tokens) shared a 32,768 `max_output_tokens` budget with content, and the
harness recorded neither `reason_tokens` nor `finish_reason`, so the guillotine was
invisible. With 64K headroom, 5.2 beats 5.1 on every axis. Kimi K3 nearly repeated
this (thinks by default, same 32K cap, unmeasured).

**Before writing any gate verdict for a new model:**

1. **Probe default thinking** — one small OpenRouter/API call, no reasoning params:
   ```bash
   # reasoning_tokens > 0 → the model thinks by default
   curl -s https://openrouter.ai/api/v1/chat/completions -H "Authorization: Bearer $OPENROUTER_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"model":"<api_name>","messages":[{"role":"user","content":"Prove there are infinitely many primes, then state the 10th prime."}],"max_tokens":3000}' \
     | jq '{reasoning: .usage.completion_tokens_details.reasoning_tokens, provider}'
   ```
2. **If it thinks: size `max_output_tokens` for thinking + content**, not content
   alone — floor 65536 for always-thinkers (check the real ceiling via
   `/api/v1/models` → `top_provider.max_completion_tokens`). Do NOT leave the
   fleet-default 32768; thinking shares that budget.
3. **Leave default thinking ON.** A thinking-tuned model with thinking suppressed
   is not the model you're gating. The knob that is actually enforced is
   `max_tokens` headroom — OpenRouter third-party upstreams (Baidu, StreamLake)
   ignore `reasoning: {max_tokens: N}` (probed 2026-07-19); `reasoning_max_tokens`
   in models.yml is best-effort only.
4. **Read `finish_reason` + `reason_tokens` in every failure before concluding
   capability** (recorded on standard results since v0.30.0). A `finish=length`
   failure with huge `reason_tokens` is truncation, not weakness. Results banked
   before v0.30.0 cannot show this — never re-derive a verdict from them alone.
5. **Cost note:** thinking bills as output tokens. Budget/cost projections for a
   reasoning model must use `output + reasoning`, and expect ~2-10x the completion
   tokens of a non-thinking peer.

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

The smoke test is the **canonical `smoke` tier** — benchmarks tagged `tier: smoke`
in their YAML spec, selected with `--tier smoke` (NOT a hand-typed `--benchmarks`
list). These are the fundamental "can it speak AILANG at all" tests that the
established frontier tier passes cleanly. If a candidate fails them, the failure is
on the model, not the harness or the benchmark.

**The smoke tier is the source of truth — do NOT hardcode a benchmark list.**
Run `ailang eval-suite --tier smoke --dry-run` to see the current set (23 as of
2026-06-16, up from 17 — it grows, so always derive it, never trust this number).
It includes fizzbuzz, adt_option, gcd_lcm, nested_records, record_update,
recursion_fibonacci, type_safe_record_access, balanced_parens, etc. — all fundamental.

> **⚠️ csv_to_json_converter is `tier: core`, NOT `tier: smoke`.** An earlier
> version of this skill hardcoded `fizzbuzz,adt_option,csv_to_json_converter` as
> "the smoke set." That was wrong: csv_to_json is a **core-tier discriminator** that
> the *majority of frontier models fail* in standard mode (gpt5 base, gemini-3-pro,
> gemini-3-flash, sonnet-4-5, gpt5-mini all FAIL it; only the top tier —
> opus-4-6/4-7, sonnet-4-6, gemini-3-1-pro, gpt5-2-codex/gpt5-4 — pass). Gating OS
> models on csv_to_json means "be top-3-tier or be cut," which unfairly excludes
> viable models. Keep csv_to_json in `--tier core` runs for **ranking**, never as
> an include/exclude gate. (Empirically verified against eval baselines 2026-06-02.)

**Run smoke against a candidate (canonical tier):**

Standard mode accepts `--tier smoke` directly. **Agent mode requires an explicit
`--benchmarks` list** (deliberate cost guardrail), so derive it from the `tier:`
tags — never hardcode (the list drifts):

```bash
# Derive the smoke set from the tier tags (works for both modes, stays in sync)
SMOKE=$(grep -l 'tier: smoke' benchmarks/*.yml | xargs -n1 basename | sed 's/\.yml$//' | paste -sd, -)

# Standard mode:
ailang eval-suite --models <candidate>,claude-sonnet-4-6 --tier smoke \
  --langs ailang --output /tmp/smoke_<candidate> --parallel 2

# Agent mode (must pass the derived list explicitly):
ailang eval-suite --agent --models <candidate>,claude-sonnet-4-6 \
  --benchmarks "$SMOKE" --langs ailang --output /tmp/smoke_<candidate> --parallel 2

# Tabulate pass/fail (agent mode → results land under /agent, standard → /standard)
for f in /tmp/smoke_<candidate>/*/*.json; do
  name=$(basename "$f" .json | sed 's/_[0-9]*$//')
  jq -r --arg name "$name" '"\($name)\t\(if .compile_ok and .runtime_ok and .stdout_ok then "PASS" else "FAIL" end)\t\(.err_code // .error_category // "—")"' "$f"
done | column -ts $'\t'
```

**Decision tree** (N = number of benchmarks in the smoke tier, 23 as of 2026-06-16 — derive it, don't assume):
1. **claude-sonnet-4-6 fails any smoke-tier benchmark** — smoke tier is broken;
   fix the benchmark (or its `tier:` tag) before evaluating candidates.
2. **Candidate fails most of the tier** — CUT. Do not add to models.yml. Note the
   failure types in the cut commit message (WRONG_LANG, syntax, runtime,
   wrong-output) for future reference.
3. **Candidate fails 1–2 (near-clean)** — NEAR-MISS. Optionally keep with a
   "near-miss" comment block in models.yml (precedent: `motoko-or-gemma-4-26b`,
   `motoko-or-qwen3-5-35b-a3b`). Re-run periodically; if it starts passing the
   tier clean, that's a signal stdlib/prompt has improved. Note: agent-mode
   failures with `error_category: api_error` + "step budget exhausted" are a
   **harness step-budget cap, not a model gap** — don't count them as capability
   failures (bump the motoko v2 step budget instead).
4. **Candidate passes the tier clean** — it has cleared the FLOOR, nothing more.
   Smoke qualifies a model; it does NOT rank it (see step 5.5). Add the opt-in
   models.yml entry now (gate PASSED, promotion PENDING), then go to step 5.5 to
   decide whether it actually earns a suite slot / replaces an incumbent.

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

3. **csv_to_json_converter is a `core`-tier DISCRIMINATOR, not a smoke gate.**
   Of the 27 benchmark runs (9 models × 3), csv_to_json was the single most-failed
   test — only GLM 5 passed it among OS candidates. ⚠️ **CORRECTION (2026-06-02):**
   this is exactly why it must NOT gate inclusion — it's failed by the *majority of
   frontier models* (gpt5 base, gemini-3-pro, gemini-3-flash, sonnet-4-5, gpt5-mini
   all FAIL; only opus-4-6/4-7, sonnet-4-6, gemini-3-1-pro, gpt5-2-codex/gpt5-4
   pass). It lives in `tier: core`, not `tier: smoke`. Use it as a high-signal
   **ranking/discriminator** metric in `--tier core` runs and as a language-
   improvement tracker — never as an OS-model include/exclude gate. The gate is
   `--tier smoke`.

4. **GLM 5 is genuinely cost-competitive frontier OS.** $0.60/$2.08 per 1M
   tokens, ~5–7× cheaper than Claude Sonnet 4.6 on input. Worth standing
   inclusion in eval rotation alongside frontier proprietary models.

5. **Vendor-prefix wiring is forward-compat infrastructure.** When adding
   models from a new vendor (e.g. `z-ai/`, `moonshotai/`, `microsoft/`,
   `minimax/`), add the prefix to
   `internal/ai/config.go::openrouterVendorPrefixes` so future ad-hoc
   `ailang run --ai vendor/model` invocations work without needing a
   models.yml entry.

6. **Per-benchmark timeouts can be tighter than agent-mode needs.** The
   `csv_to_json_converter.yml` spec has `timeout: 90s` baked in (set to
   match Claude Sonnet 4.6's ~43s typical solve time). OS models in agent
   mode routinely need 90–180s of iteration on csv_to_json — they CAN
   solve it but get killed by the timeout. Two follow-up models that
   demonstrated this on 2026-05-04:
     - **Kimi K2.6** (Moonshot): fizzbuzz✅ 119s, adt_option✅ 47s,
       csv_to_json❌ (timeout — initial run also had api_errors)
     - **MiniMax M2.7**: fizzbuzz✅ 46s, adt_option✅ 42s,
       csv_to_json❌ (timeout, not capability)
   Both are effectively 2/3 near-misses pending a benchmark timeout bump.
   When investigating a model that fails only csv_to_json with
   `error_category=api_error` and stderr saying "exceeded hard timeout
   (1m30s)", the failure is the benchmark spec, not the model.

7. **api_error vs syntax-error vs WRONG_LANG matters.** When tabulating
   smoke results, always check `error_category`:
   - `api_error` — infrastructure issue (rate limit, timeout, network).
     Re-run before counting against the model.
   - `compile_error` (no err_code) — syntax-error: model produced AILANG
     that doesn't parse. Genuine model gap.
   - `WRONG_LANG` — model produced Python/JS/etc. instead of AILANG.
     Genuine prompt-following gap.
   - `runtime_error` — compiled but crashed. Logic bug in generation.

### 5.5 Smoke is a FLOOR, not a RANKING — use `--tier core` to decide add/replace (HARD RULE)

> **Passing smoke is necessary but NOT sufficient. Smoke says "this model can
> speak AILANG at all"; it does NOT say "this model is good enough to add" or
> "this model beats the incumbent." Those are RANKING questions, and smoke is
> saturated — every frontier-class model scores ~the same on it. Never make an
> add/keep/replace decision on smoke numbers. Make it on `--tier core`.**

Why: the smoke tier is deliberately fundamental ("can it speak AILANG"), so any
viable model passes ~all of it. A smoke **tie is the expected outcome**, not a
signal — it carries zero ranking information. The discriminator is the **core
tier** (`--tier core`, ~26 benchmarks incl. `csv_to_json_converter`, the
contract/state-machine tests) where frontier models genuinely spread.

**Decision flow once a candidate PASSES smoke:**
1. **New vendor/family, no incumbent** — run `--tier core` head-to-head vs the
   `claude-sonnet-4-6` anchor to size where it lands. Add to suites if it earns it.
2. **Replacing or competing with an incumbent** (e.g. GLM-5.2 vs GLM-5.1) — run
   `--tier core` for **candidate + incumbent + anchor in ONE command**, `--langs
   ailang`. **Only promote/replace if the candidate matches-or-beats the incumbent
   on core.** A core tie at higher cost → keep the incumbent. A clear core win →
   the cost bump may be justified.
3. **Close call on N=1** — core is ~26 single-shot runs; OS-model variance is real.
   If candidate and incumbent are within 1–2 benchmarks, escalate to N≥3 trials
   before deciding (don't flip an incumbent on a 1-benchmark N=1 delta).

```bash
# The discriminating run — candidate vs incumbent vs anchor, core tier, one command:
ailang eval-suite --models <candidate>,<incumbent>,claude-sonnet-4-6 \
  --tier core --langs ailang --output /tmp/core_<candidate> --parallel 4
```

> **⚠️ Anti-pattern (2026-06-16, GLM-5.2 vs GLM-5.1):** GLM-5.2 (newest z-ai,
> reasoning model, 1M ctx, +43% price) cleared standard smoke at **22/23 — an
> exact tie with GLM-5.1** (both failed only `dense_operator_program`, which the
> `claude-sonnet-4-6` anchor ALSO failed → a benchmark/harness issue, not a model
> gap). The first-pass conclusion was *"tie at +43% cost → keep GLM-5.1."* **That
> was WRONG.** A smoke tie is meaningless because smoke is saturated — it proves
> only that GLM-5.2 cleared the floor and QUALIFIES. The replacement decision had
> to be made on `--tier core`, where the two versions can actually separate. Rule:
> when a candidate ties the incumbent on smoke, that's your cue to run core, NOT
> your answer.

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
