# Running Your First Baseline Tests

**Goal**: Generate empirical data on AI code generation efficiency for AILANG vs Python.

This will reveal where the AI struggles with AILANG syntax and help guide documentation improvements.

---

## Prerequisites

### 1. Build ailang

```bash
make build
# or
make install  # to make it globally available
```

### 2. Set Up API Access

The benchmark suite uses three models by default. You'll need at least one:

**Option A: Anthropic Claude (Recommended)**

1. Go to https://console.anthropic.com/
2. Create API key
3. Export it:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Option B: OpenAI GPT**

1. Go to https://platform.openai.com/api-keys
2. Create new key
3. Export it:

```bash
export OPENAI_API_KEY="sk-proj-..."
```

**Option C: Google Gemini (via Vertex AI)**

Requires Google Cloud SDK (`gcloud`):

```bash
# Install gcloud SDK if needed
# See: https://cloud.google.com/sdk/docs/install

# Authenticate
gcloud auth application-default login

# Set your GCP project
gcloud config set project YOUR_PROJECT_ID

# Enable Vertex AI API
gcloud services enable aiplatform.googleapis.com
```

**✅ All Three Working**: As of October 2025, all three API implementations are tested and working:
- ✅ Claude Sonnet 4.5 (Anthropic) - 230 tokens baseline
- ✅ GPT-5 (OpenAI) - 319 tokens baseline
- ✅ Gemini 2.5 Pro (Google Vertex AI) - 278 tokens baseline

### 3. Verify Setup

```bash
# Test with mock mode (no API key needed)
ailang eval --benchmark fizzbuzz --mock

# Should see:
# → Running benchmark: Classic FizzBuzz...
# ✓ Benchmark complete. Results saved to eval_results/
```

---

## Quick Baseline Run (5 minutes, ~$0.10)

Run all 5 benchmarks with both Python and AILANG:

### Option 1: Automated Script (Recommended)

```bash
# Run complete benchmark suite with all 3 models
./tools/run_benchmark_suite.sh
```

This will run all benchmarks with:
- GPT-5 (OpenAI)
- Claude Sonnet 4.5 (Anthropic)
- Gemini 2.5 Pro (Google)

### Option 2: Manual (Single Model)

```bash
# Clean previous results
make eval-clean

# Run all benchmarks with Claude Sonnet 4.5 (recommended)
for bench in fizzbuzz json_parse pipeline cli_args adt_option; do
    echo "Running $bench..."
    ailang eval --benchmark $bench --model claude-sonnet-4-5 --seed 42 --langs python,ailang
    sleep 5  # Rate limiting
done

# Generate report
make eval-report

# View results
cat eval_results/leaderboard.md
```

### Available Models

Check available models:
```bash
ailang eval --list-models
```

**Default models** (October 2025):
- `gpt5` - OpenAI GPT-5 (released August 2025)
- `claude-sonnet-4-5` - Anthropic Claude Sonnet 4.5 (released September 2025, **recommended**)
- `gemini-2-5-pro` - Google Gemini 2.5 Pro (released March 2025)

**Expected time**: ~3-5 minutes per model
**Expected cost**: ~$0.05-0.15 per model (full 5-benchmark suite)

---

## What to Look For

### 1. Token Efficiency

**Hypothesis**: AILANG should use fewer tokens per attempt (concise syntax)

```bash
# Check summary section of report
cat eval_results/leaderboard.md | grep "Avg Token Reduction"

# Expected: 20-40% reduction
```

### 2. Success Rate

**Hypothesis**: Python will have higher success rate (familiar to AI)

```bash
# Check success rates
cat eval_results/leaderboard.md | grep "Success Rate"

# Expected:
# - Python: 60-80% (AI knows Python well)
# - AILANG: 20-40% (unfamiliar syntax)
```

### 3. Error Patterns

**Key Question**: What AILANG syntax confuses the AI?

```bash
# Look at error categories
cat eval_results/summary.csv | grep ailang | grep -v "none"

# Common errors we expect:
# - compile_error: Missing "in" after "let"
# - compile_error: Wrong module import syntax
# - runtime_error: Unknown builtin functions
```

### 4. Per-Benchmark Analysis

```bash
# Check which benchmarks favor AILANG
cat eval_results/leaderboard.md

# Expected observations:
# - fizzbuzz: Similar tokens, Python succeeds more
# - adt_option: AILANG much fewer tokens, but may fail
# - json_parse: Python easier (stdlib familiarity)
```

---

## Detailed Analysis

### Extract AILANG Error Messages

```bash
# Look at the actual generated code
for file in eval_results/fizzbuzz_ailang_*.json; do
    echo "=== $file ==="
    jq -r '.stderr' "$file" | head -5
    echo ""
done
```

**What to look for:**
- Syntax errors → document correct syntax
- Type errors → provide examples
- Import errors → clarify module system
- Runtime errors → identify missing builtins

### Compare Token Usage

```bash
# Python avg
cat eval_results/summary.csv | grep python | awk -F, '{sum+=$5; count++} END {print "Python avg:", sum/count}'

# AILANG avg
cat eval_results/summary.csv | grep ailang | awk -F, '{sum+=$5; count++} END {print "AILANG avg:", sum/count}'
```

### Success by Category

```bash
# Python success rate
echo "Python success:"
cat eval_results/summary.csv | grep python | awk -F, '$9=="true" {s++} END {print s " / " NR " = " (s/NR*100) "%"}'

# AILANG success rate
echo "AILANG success:"
cat eval_results/summary.csv | grep ailang | awk -F, '$9=="true" {s++} END {print s " / " NR " = " (s/NR*100) "%"}'
```

---

## Using Results to Improve AILANG

### Step 1: Identify Top 3 Errors

```bash
# Extract error messages from failed AILANG runs
grep ailang eval_results/*.json | grep compile_ok | grep false -B5
```

**Example findings:**
```
Error 1: "Expected 'in' after let binding"
  → Frequency: 3/5 benchmarks
  → Fix: Add to docs, improve error message

Error 2: "Module 'option' not found"
  → Frequency: 2/5 benchmarks
  → Fix: Document stdlib imports

Error 3: "Unexpected token 'func'"
  → Frequency: 1/5 benchmarks
  → Fix: Clarify module vs non-module syntax
```

### Step 2: Update Documentation

Create or improve:
- `docs/syntax_guide.md` - Common syntax patterns
- `docs/stdlib_reference.md` - How to import/use stdlib
- `examples/ai_friendly.ail` - Examples for AI context

### Step 3: Improve Benchmark Prompts

Add AILANG-specific hints to prompts:

```yaml
# benchmarks/fizzbuzz_v2.yml
prompt: |
  Write a program in <LANG> that implements FizzBuzz.

  <LANG=AILANG> Syntax notes:
  - Let expressions: `let x = 5 in x * 2`
  - Print function: `print("text")`
  - Modulo operator: `%`
```

### Step 4: Re-run Benchmarks

```bash
# Run again with improved prompts
make eval-clean
# ... repeat baseline run ...

# Compare results
diff eval_results_v1/summary.csv eval_results_v2/summary.csv
```

**Expected improvement:**
- AILANG success rate: 20% → 40% (+100%)
- Token increase (due to hints): +10-20%
- Net benefit: Higher success rate justifies slightly longer prompts

---

## Cost Management

### Minimize API Costs

**During development:**
```bash
# Use GPT-3.5 (10x cheaper)
ailang eval --benchmark fizzbuzz --model gpt-3.5-turbo --seed 42

# Or use mock mode (free)
ailang eval --benchmark fizzbuzz --mock
```

**For final results:**
```bash
# Use GPT-4 for quality
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 42
```

### Track Spending

```bash
# Sum total cost
cat eval_results/summary.csv | awk -F, '{sum+=$6} END {print "Total cost: $" sum}'
```

---

## Sharing Results

### Generate Clean Report

```bash
# Create shareable markdown
make eval-report

# Copy to project root for visibility
cp eval_results/leaderboard.md BASELINE_RESULTS.md

# Add to git
git add BASELINE_RESULTS.md
git commit -m "Add baseline evaluation results"
```

### Publish Findings

Include in your next release notes:

```markdown
## Baseline Evaluation Results (v0.2.0)

We measured AI code generation efficiency for AILANG vs Python across 5 benchmarks:

**Key Findings:**
- ✅ AILANG uses 35% fewer tokens per attempt
- ⚠️ AILANG has 50% lower first-attempt success rate
- 📊 Most common error: Missing "in" after "let" (60% of failures)

**Next Steps:**
- Improve syntax documentation based on error patterns
- Add AILANG-specific prompts for Phase 2 (multi-turn evaluation)
- Target 80% success rate with enhanced context

Full results: [BASELINE_RESULTS.md](BASELINE_RESULTS.md)
```

---

## API Implementation Status

All three target models have been tested and verified working as of October 2, 2025:

### ✅ Anthropic Claude Sonnet 4.5

```bash
./bin/ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --langs python --seed 42
```

**Result**:
- ✅ API call successful
- Generated: 230 tokens
- Code compiled and ran successfully
- API endpoint: `https://api.anthropic.com/v1/messages`

### ✅ OpenAI GPT-5

```bash
./bin/ailang eval --benchmark fizzbuzz --model gpt5 --langs python --seed 42
```

**Result**:
- ✅ API call successful
- Generated: 319 tokens
- Code compiled and ran successfully
- API endpoint: `https://api.openai.com/v1/chat/completions`

### ✅ Google Gemini 2.5 Pro (Vertex AI)

```bash
./bin/ailang eval --benchmark fizzbuzz --model gemini-2-5-pro --langs python --seed 42
```

**Result**:
- ✅ API call successful (via Vertex AI)
- Generated: 278 tokens
- Code compiled and ran successfully
- API endpoint: `https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent`
- Authentication: OAuth via `gcloud auth application-default print-access-token`

**Implementation Notes**:
- All three use direct HTTP calls (no SDKs) for minimal dependencies
- Anthropic & OpenAI use API keys from environment variables
- Google uses Vertex AI with gcloud Application Default Credentials (ADC)
- Token counting works correctly for all three providers
- All responses successfully extract code from markdown fences

---

## Troubleshooting

### "rate limit exceeded"

OpenAI has rate limits. Wait a minute between runs:

```bash
for bench in fizzbuzz json_parse pipeline cli_args adt_option; do
    ailang eval --benchmark $bench --model gpt-4 --seed 42
    sleep 60  # Wait 1 minute
done
```

### "Model not found"

Check available models:

```bash
# OpenAI models
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY" | jq '.data[].id'

# Use full model ID
ailang eval --benchmark fizzbuzz --model gpt-4-turbo-2024-04-09
```

### Results look wrong

Verify with mock mode first:

```bash
# Run mock mode
ailang eval --benchmark fizzbuzz --mock --langs python,ailang

# Check that harness works
cat eval_results/*.json | jq .
```

---

## Next Steps

After baseline tests:

1. ✅ Analyze error patterns
2. ✅ Document common issues
3. ✅ Improve AILANG documentation
4. ✅ Update benchmark prompts
5. ✅ Re-run with improved context
6. ✅ Compare v1 vs v2 results
7. 🔮 Prepare for M-EVAL2 (multi-turn evaluation)

---

**Happy testing!** 🚀

Your baseline results will directly inform AILANG's development priorities and documentation improvements.
