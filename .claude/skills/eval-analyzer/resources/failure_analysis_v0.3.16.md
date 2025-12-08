# AILANG v0.3.16 Eval Failure Analysis

## Executive Summary

**Overall Success Rate**: 31.0% (63/204 runs)
**Failure Rate**: 69.0% (141/204 runs)

**Failure Breakdown:**
- **compile_error**: 62% (90 failures) - Models generating invalid AILANG syntax
- **logic_error**: 31% (46 failures) - Valid code but wrong output
- **runtime_error**: 6% (9 failures) - Crashes during execution

**Top Error Codes:**
- PAR_001 (68 occurrences) - Parse errors, syntax mistakes
- WRONG_LANG (10 occurrences) - Models generating Python/other languages instead of AILANG
- IMPERATIVE (7 occurrences) - Imperative patterns (e.g., `x = y` instead of `let x = y`)
- CAP_001 (3 occurrences) - Missing capability grants

## Root Cause Analysis

### Problem 1: Prompt-Benchmark Mismatch (CRITICAL BUG)

**The Bug**: v0.3.16 prompt contains a **FALSE limitation** that contradicts the actual implementation!

**What the prompt says**:
```markdown
⚠️ NO custom HTTP headers (OpenAI/Claude APIs blocked until v0.4.0)
```

**Reality**: `httpRequest()` with custom headers has been **working since v0.3.9**!
```ailang
import std/net (httpRequest)

let headers = [{name: "X-Test-Header", value: "value123"}]
let response = httpRequest("POST", url, headers, body)  -- ✅ WORKS!
```

**What happened**:
1. v0.3.9 added `httpRequest()` with headers but kept old "NO headers" limitation (contradictory)
2. v0.3.16 **removed httpRequest documentation** but **kept the false limitation**
3. Models read "NO custom HTTP headers" and gave up or hallucinated syntax
4. Result: All 6 models failed `api_call_json` benchmark (0% success rate)

**Impact**: Any benchmark requiring HTTP headers will fail at 0% because models think it's impossible

**Generated Code Patterns** (models trying to work around "impossible" requirement):
- **gemini-2-5-flash**: Used uppercase `LET` keyword (doesn't exist in AILANG)
- **claude-haiku-4-5**: Used `import HTTP` (doesn't exist) and Python-style dict syntax
- **claude-sonnet-4-5**: Invented shell-like syntax `http-post "url" {...} '...' response`
- **gpt5**: Used Python-style keyword arguments `http.post(url: "...", headers: {...}, json: {...})`

**Root Cause**: Documentation regression - feature exists but prompt says it doesn't!

**Fix**: Update v0.3.17 prompt to:
1. **REMOVE** false "NO custom HTTP headers" limitation
2. **ADD** httpRequest() documentation with examples
3. **ADD** to import checklist: `httpRequest` → `import std/net (httpRequest)`

### Problem 2: Syntax Confusion (PAR_001 errors)

**Models are generating invalid AILANG syntax patterns:**

1. **Python-style keyword arguments**:
   ```python
   # ❌ WRONG (GPT-5)
   http.post(url: "...", headers: {...}, json: {...})
   ```
   AILANG uses positional arguments only

2. **Uppercase keywords**:
   ```
   # ❌ WRONG (Gemini)
   LET url = "..."
   ```
   AILANG keywords are lowercase: `let`, `func`, `type`, etc.

3. **Module-qualified calls without imports**:
   ```
   # ❌ WRONG (Gemini)
   http.post(...)  # No import statement!
   ```
   AILANG requires explicit imports: `import std/net (httpPost)`

4. **Invented syntax**:
   ```
   # ❌ WRONG (Claude Sonnet)
   http-post "url" {...} '...' response
   ```
   No such syntax exists in AILANG

**Root Cause**: Models are trained on Python/JavaScript/Bash and default to familiar syntax patterns.

**Recommendation**: Enhance prompt with more negative examples (anti-patterns).

### Problem 3: Prompt Version Tracking

**How to find prompt version**:
1. Check `eval_results/baselines/{version}/baseline.json` for `"version": "0.3.16"`
2. Cross-reference with `prompts/versions.json` active version
3. For v0.3.16 baseline → used `prompts/v0.3.16.md` prompt

**Enhancement idea**: Add `prompt_version` and `prompt_hash` fields to individual result JSON files for easier tracking (currently only in baseline.json)

## Detailed Failure Patterns

### API Call JSON Benchmark (0% success, 6/6 models failed)

**Expected AILANG code** (what models SHOULD have generated):
```ailang
module benchmark/solution

import std/net (httpRequest)
import std/json (encode, jo, kv, js, jnum)

export func main() -> () ! {Net, IO} {
  let headers = [
    {name: "X-Test-Header", value: "value123"},
    {name: "Content-Type", value: "application/json"}
  ];
  let body = encode(jo([
    kv("message", js("Hello from AILANG")),
    kv("count", jnum(42.0))
  ]));
  match httpRequest("POST", "https://httpbin.org/post", headers, body) {
    Ok(resp) => print(show(resp.status)),
    Err(e) => print("error")
  }
}
```

**Why this is correct:**
- Uses `httpRequest()` with custom headers (supported since v0.3.9)
- Pattern matches on Result type for error handling
- Accesses `resp.status` field from HttpResponse record
- Uses JSON encoding functions from std/json

**What models generated instead**:

1. **Gemini**: Uppercase `LET`, no imports, wrong module qualifier
2. **Claude Haiku**: Invented `import HTTP`, Python dict syntax
3. **Claude Sonnet**: Shell-like command syntax
4. **GPT-5**: Python keyword argument syntax

**Why all failed**: Benchmark requires HTTP headers, but prompt says they're not supported.

### Common Error Categories

#### 1. Compile Error (62% of failures)

**Subcategories:**
- Parse errors (PAR_001): Wrong syntax, invalid tokens
- Wrong language (WRONG_LANG): Python/JS generated instead of AILANG
- Imperative style (IMPERATIVE): Using `=` instead of `let`
- Missing imports: Using functions without importing them

**Pattern**: Models default to familiar syntax from training data

#### 2. Logic Error (31% of failures)

**Patterns:**
- Correct syntax, but wrong algorithm
- Off-by-one errors in recursion
- Incorrect pattern matching logic
- Wrong output format (e.g., printing list instead of just status code)

**Pattern**: Models understand the task but implement wrong logic

#### 3. Runtime Error (6% of failures)

**Patterns:**
- Stack overflow (infinite recursion)
- Missing capability grants (CAP_001)
- Type errors not caught by type checker
- Record field access on wrong type

**Pattern**: Rare - type system catches most errors at compile time

## Benchmarks with 0% Success Rate (All Models Failed)

1. **api_call_json** - Requires unsupported HTTP headers
2. (Need to examine 16 more to categorize)

## Model Performance Comparison

(Data from summary.jsonl - need to run analyze_failures.sh to populate)

## Recommendations

### Immediate Actions (v0.3.17)

1. **Fix prompt-benchmark mismatch**:
   - Remove `api_call_json` benchmark OR implement HTTP headers
   - Audit all benchmarks for unsupported features

2. **Add prompt version to results**:
   - Modify eval harness to store `prompt_version` field
   - Enables correlation analysis between prompt changes and success rates

3. **Enhance syntax documentation**:
   - Add "Common Mistakes" section with anti-patterns
   - Show ❌ WRONG / ✅ CORRECT examples for:
     - Function calls (positional vs keyword args)
     - Keywords (lowercase only)
     - Import syntax
     - Module qualification

### Medium-term Actions (v0.4.0)

1. **Implement HTTP headers support**:
   - Design: How should headers be passed? List of records? Map?
   - Security: Whitelist allowed headers
   - Testing: Update benchmarks to use new API

2. **Improve error messages**:
   - When parse fails, suggest common fixes
   - "Did you mean `let` instead of `LET`?"
   - "AILANG doesn't support keyword arguments - use positional arguments"

3. **Expand eval analysis**:
   - Create repair suggestion system
   - Identify which syntax errors can be auto-fixed
   - Test if auto-repair improves success rates

### Long-term Actions (v0.5.0+)

1. **Systematic prompt engineering**:
   - A/B test prompt variations
   - Measure: Which wording reduces parse errors most?
   - Track prompt effectiveness over time

2. **Model-specific prompts**:
   - GPT-5 makes different mistakes than Claude
   - Custom prompts per model family?
   - Trade-off: Maintenance burden vs. performance gain

3. **Synthetic training data**:
   - Generate AILANG code examples from successful runs
   - Fine-tune models on AILANG syntax
   - Reduce reliance on prompt engineering

## Next Steps

1. Run `.claude/skills/eval-analyzer/scripts/analyze_failures.sh` for detailed stats
2. Examine remaining 16 benchmarks with 0% success rate
3. Categorize all failure patterns systematically
4. Create design doc for HTTP headers support
5. Update eval harness to store prompt version
