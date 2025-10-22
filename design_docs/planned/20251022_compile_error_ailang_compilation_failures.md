# AILANG: Compilation Failures

**Discovered**: AI Eval Analysis - 2025-10-22
**Frequency**: 3 failures across 1 benchmark(s)
**Priority**: P0 (Critical - Must Ship)
**Estimated**:  LOC, 
**Category**: compile_error
**Impact**: critical

## Problem Statement

AIs are generating Python/JavaScript-style syntax when attempting HTTP/JSON operations in AILANG. All three affected models (claude-haiku-4-5, gemini-2-5-flash, gpt5-mini) defaulted to familiar patterns from other languages instead of using AILANG's actual syntax.

**Current State:**
- api_call_json benchmark has 75% failure rate (3/4 runs)
- All failures are parse errors, not runtime/logic errors
- Generated code shows clear Python/JS influence (namespace imports, const keyword, Python-style assignment)

**Impact:**
- **Who**: All AI models attempting HTTP/JSON operations
- **Significance**: P0 - Blocks real-world use cases (API calls, JSON parsing)
- **Workaround**: None - parse errors prevent execution entirely

## Evidence from AI Eval

**Affected Benchmarks**: api_call_json

**Models Affected**: claude-haiku-4-5, gemini-2-5-flash, gpt5-mini

**Failure Rate**: 3/4 (75.0%)

### Example Failures


**Error 1:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [IMP012_UNSUPPORTED_NAMESPACE at benchmark/solution.ail:1:8: namespace imports not yet supported PAR_NO_PREFIX_PARSE at benchmark/solution.ail:3:2: unexpected token in expression: / PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:4:28: expected next token to be ), got IDENT instead PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:32: u...
```

**Generated Code:**
```ailang
import http from 'http'

// Make HTTP POST request to httpbin.org
let response = http.post('https://httpbin.org/post', {
  headers: {
    'X-Test-Header': 'value123',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    message: 'Hello from AILANG',
    count: 42
  })
})

// Print the status code
print(response.statusCode)
```

---

**Error 2:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [PAR_NO_PREFIX_PARSE at benchmark/solution.ail:1:11: unexpected token in expression: = PAR_NO_PREFIX_PARSE at benchmark/solution.ail:1:39: unexpected token in expression: ; PAR_NO_PREFIX_PARSE at benchmark/solution.ail:2:15: unexpected token in expression: = expected ; or }, got : PAR_NO_PREFIX_PARSE at benchmark/solution.ail:3:20: u...
```

**Generated Code:**
```ailang
const URL = "https://httpbin.org/post";
const HEADERS = {
    "X-Test-Header": "value123",
    "Content-Type": "application/json"
};
const BODY_DATA = {
    "message": "Hello from AILANG",
    "count": 42
};

const BODY_STRING = JSON.stringify(BODY_DATA);
const response = HTTP.POST(URL, HEADERS, BODY_STRING);

PRINT(response.status);
```

---

**Error 3:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [IMP012_UNSUPPORTED_NAMESPACE at benchmark/solution.ail:1:8: namespace imports not yet supported IMP012_UNSUPPORTED_NAMESPACE at benchmark/solution.ail:2:8: namespace imports not yet supported PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:5: unexpected token in expression: = PAR_NO_PREFIX_PARSE at benchmark/solution.ail:5:9: unexpe...
```

**Generated Code:**
```ailang
import http
import json

url = "https://httpbin.org/post"
headers = {"X-Test-Header": "value123", "Content-Type": "application/json"}
payload = {"message": "Hello from AILANG", "count": 42}
response = http.post(url, headers=headers, body=json.encode(payload))
print(response.status_code)
```

---


## Root Cause Analysis

### Pattern 1: Namespace Imports (Error 1, Error 3)
```javascript
// AIs generate:
import http from 'http'  // JavaScript ES6 style
import http              // Python style

// AILANG actually uses:
import std/net (httpRequest)
```

**Root Cause:** Teaching prompt lacks HTTP examples. AIs default to JavaScript/Python import patterns they know.

### Pattern 2: const Keyword (Error 2)
```javascript
// AIs generate:
const URL = "..."   // JavaScript style

// AILANG actually uses:
let url = "..."     // or just inline the value
```

**Root Cause:** `const` keyword doesn't exist in AILANG. AIs assume immutability means `const` keyword.

### Pattern 3: Python-Style Assignment (Error 3)
```python
# AIs generate:
url = "..."                                      # Python style
response = http.post(url, headers=headers, ...)  # Named arguments

# AILANG actually uses:
let url = "..." in                               # Let binding
httpRequest("POST", url, headers, ...)           # Positional arguments
```

**Root Cause:** Teaching prompt doesn't show let bindings for HTTP calls. AIs default to Python's bare assignment.

### Pattern 4: Method Call Syntax (All errors)
```javascript
// AIs generate:
http.post(url, ...)      // Object.method() style
HTTP.POST(URL, ...)      // Uppercase variant

// AILANG actually uses:
httpRequest("POST", ...) // Function call with method as string
```

**Root Cause:** HTTP library doesn't exist. AIs assume OOP-style HTTP library like Python requests or JavaScript fetch.

## Proposed Solution



**Option A: Enhanced Error Messages (Quick Win, 2-4 hours)**

When AIs generate common Python/JS patterns, provide suggestions:

```bash
# Current error:
Error: namespace imports not yet supported
  at benchmark/solution.ail:1:8

# Enhanced error:
Error: namespace imports not yet supported
  at benchmark/solution.ail:1:8

  You wrote: import http from 'http'

  Did you mean one of these?
    import std/net (httpRequest)     -- For HTTP requests
    import std/json (encode, decode) -- For JSON parsing

  See: https://docs.ailang.io/std/net
```

**Pros:**
- Fast to implement (modify parser error messages)
- No language changes needed
- Directly guides AIs to correct syntax
- Useful for all users (not just AIs)

**Cons:**
- AIs still need to regenerate code (costs time/money)
- Doesn't prevent initial failure

---

**Option B: Update Teaching Prompt (Quick Win, 2 hours)**

Add HTTP/JSON examples to prompts/v0.3.8.md (or v0.3.15.md):

```ailang
-- Example: HTTP POST with JSON
module example/http_post

import std/net (httpRequest)
import std/json (encode, decode)
import std/io (println)

export func main() -> () ! {Net, IO} {
  let headers = [
    {name: "Content-Type", value: "application/json"},
    {name: "X-Custom-Header", value: "value123"}
  ] in

  let payload = encode({message: "Hello", count: 42}) in
  let response = httpRequest("POST", "https://httpbin.org/post", headers, payload) in

  println(show(response.status))
}
```

**Pros:**
- Prevents failures upfront (AIs see correct pattern)
- No code changes needed
- Establishes canonical HTTP pattern

**Cons:**
- Increases prompt size (~100 tokens)
- Requires regenerating teaching prompt
- Doesn't help if AIs ignore example

---

**Option C: Implement Glob Imports (Medium, deferred to v0.4.0)**

```ailang
-- Allow:
import std/net (*)  -- Import all exports
import std/json (*) -- Import all exports
```

**Pros:**
- More concise (AIs don't need to know specific function names)
- Aligns with Python `from module import *`

**Cons:**
- Language feature (3-4 days to implement)
- Doesn't address root cause (AIs still use wrong syntax)
- Deferred to v0.4.0 per roadmap

---

**Recommended Solution: Option A + Option B (4-6 hours total)**

1. **Phase 1** (2h): Enhanced error messages
   - Modify parser to detect common patterns (`import X from`, `const`, `=` without `let`)
   - Provide actionable suggestions in error messages
   - Test with examples from eval failures

2. **Phase 2** (2h): Update teaching prompt
   - Add HTTP POST example to prompts/v0.3.15.md
   - Add JSON encode/decode example
   - Document std/net and std/json modules

3. **Phase 3** (2h): Re-run eval baseline
   - Test api_call_json with updated prompt
   - Measure improvement: Target 75% → <10% failure
   - Document findings

### Implementation Approach

**Step 1: Enhanced Parser Errors** (2 hours)

Modify `internal/parser/parser.go` and `internal/errors/parser_errors.go`:

```go
// Detect common patterns and suggest fixes
func (p *Parser) parseImport() (*ast.ImportDecl, error) {
    // ... existing import parsing ...

    // Check for "import X from 'Y'" pattern
    if p.peekTokenIs(lexer.FROM) {
        return nil, &errors.SuggestionError{
            Code:     "IMP012_UNSUPPORTED_NAMESPACE",
            Message:  "namespace imports not yet supported",
            Location: p.curToken.Position,
            UserCode: p.extractLine(),
            Suggestions: []string{
                "import std/net (httpRequest)     -- For HTTP requests",
                "import std/json (encode, decode) -- For JSON parsing",
            },
            HelpURL: "https://docs.ailang.io/imports",
        }
    }

    // ... rest of parsing ...
}

// Detect "const" keyword
func (p *Parser) parseStatement() (ast.Stmt, error) {
    if p.curTokenIs(lexer.CONST) {
        return nil, &errors.SuggestionError{
            Code:     "PAR_CONST_NOT_SUPPORTED",
            Message:  "'const' keyword doesn't exist in AILANG",
            Location: p.curToken.Position,
            UserCode: p.extractLine(),
            Suggestions: []string{
                "Use: let name = value in ...",
                "Note: All bindings in AILANG are immutable by default",
            },
            HelpURL: "https://docs.ailang.io/syntax",
        }
    }

    // ... existing statement parsing ...
}
```

**Step 2: Update Teaching Prompt** (2 hours)

Add to `prompts/v0.3.15.md` (or create new version):

```markdown
## HTTP Requests & JSON

AILANG provides HTTP and JSON functionality through standard library modules.

### HTTP POST with JSON Payload

```ailang
module example/http_post

import std/net (httpRequest)
import std/json (encode, decode)
import std/io (println)

export func main() -> () ! {Net, IO} {
  -- Prepare headers
  let headers = [
    {name: "Content-Type", value: "application/json"},
    {name: "X-Custom-Header", value: "value123"}
  ] in

  -- Encode JSON payload
  let payload = encode({
    message: "Hello from AILANG",
    count: 42
  }) in

  -- Make HTTP POST request
  let response = httpRequest(
    "POST",                        -- HTTP method
    "https://httpbin.org/post",    -- URL
    headers,                       -- Headers list
    payload                        -- Body string
  ) in

  -- Print status code
  println(show(response.status))
}
```

### Important Notes
- No `import http from 'http'` - use `import std/net (httpRequest)`
- No `const` keyword - use `let` (immutable by default)
- No bare assignment (`x = y`) - use `let x = y in ...`
- HTTP methods are strings: `"GET"`, `"POST"`, `"PUT"`, `"DELETE"`
```

**Step 3: Re-run Eval Baseline** (2 hours)

```bash
# Update teaching prompt version
cp prompts/v0.3.8.md prompts/v0.3.15.md
# Add HTTP examples (step 2 above)

# Re-run api_call_json benchmark
ailang eval-suite --benchmarks api_call_json --models claude-haiku-4-5,gemini-2-5-flash,gpt5-mini

# Compare results
ailang eval-compare eval_results/baselines/v0.3.14 eval_results/baselines/v0.3.15

# Expected improvement:
# - Before: 3/4 failures (75%)
# - After: <1/4 failures (<25%), ideally 0/4 (0%)
```

## Technical Design

### Files to Modify

**Parser error handling:**
- `internal/parser/parser.go` - Add pattern detection (~50 LOC)
- `internal/errors/parser_errors.go` - Add suggestion error type (~30 LOC)

**Teaching prompt:**
- `prompts/v0.3.15.md` - Add HTTP/JSON examples (~150 LOC)

**Total new code:** ~230 LOC

## Implementation Plan

**Phase 1: Enhanced Error Messages** (~2 hours)
- [ ] Add suggestion error type to internal/errors/
- [ ] Detect "import X from 'Y'" pattern → suggest std/net or std/json
- [ ] Detect "const" keyword → suggest "let"
- [ ] Detect bare assignment → suggest "let ... in"
- [ ] Test with examples from eval failures

**Phase 2: Update Teaching Prompt** (~2 hours)
- [ ] Create prompts/v0.3.15.md (copy from v0.3.8.md)
- [ ] Add HTTP POST example with httpRequest
- [ ] Add JSON encode/decode example
- [ ] Document std/net module (httpRequest function)
- [ ] Document std/json module (encode, decode functions)
- [ ] Update CHANGELOG.md

**Phase 3: Validate with Eval Suite** (~2 hours)
- [ ] Re-run api_call_json benchmark with all 3 models
- [ ] Compare before/after success rates
- [ ] Document failure cases (if any remain)
- [ ] Update teaching prompt if needed

## Testing Strategy

### Parser Error Tests

```go
// internal/parser/parser_errors_test.go
func TestSuggestImportFix(t *testing.T) {
    input := `import http from 'http'`
    _, err := ParseString(input)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "namespace imports not yet supported")
    assert.Contains(t, err.Error(), "import std/net (httpRequest)")
}

func TestSuggestConstFix(t *testing.T) {
    input := `const URL = "https://example.com"`
    _, err := ParseString(input)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "'const' keyword doesn't exist")
    assert.Contains(t, err.Error(), "Use: let name = value in")
}
```

### Integration Tests

```bash
# Test enhanced errors show up correctly
echo 'import http from "http"' | ailang check -
# Expected: Suggestion error with std/net import

# Test teaching prompt examples compile
ailang check prompts/v0.3.15.md --extract-examples
# Expected: All examples type-check successfully
```

### Eval Baseline Tests

```bash
# Full eval suite on api_call_json
ailang eval-suite --benchmarks api_call_json --full

# Expected:
# - claude-haiku-4-5: failure → success
# - gemini-2-5-flash: failure → success
# - gpt5-mini: failure → success
```

## Success Criteria

- [ ] Enhanced errors detect all 3 patterns (namespace imports, const, bare assignment)
- [ ] Suggestions include correct AILANG syntax for each case
- [ ] Teaching prompt includes working HTTP POST example
- [ ] api_call_json success rate: 25% → >75% (target: 100%)
- [ ] No regressions in other benchmarks
- [ ] All unit tests passing
- [ ] Documentation updated (CHANGELOG.md, prompts/v0.3.15.md)

## References

- **Similar Features**: See design_docs/implemented/ for reference implementations
- **Design Docs**: CLAUDE.md, README.md, design_docs/planned/v0_4_0_net_enhancements.md
- **AILANG Architecture**: See CLAUDE.md, README.md

## Estimated Impact

**Before Fix**:
- api_call_json success rate: 25% (1/4 passing)
- All 3 failures are parse errors (75% failure rate)
- AIs generate wrong syntax (Python/JS influence)

**After Fix** (projected):
- api_call_json success rate: >75% (3/4 passing), target 100% (4/4)
- Parse errors reduced to <10% (improved error messages guide AIs)
- Teaching prompt examples prevent initial syntax errors

**Broader Impact**:
- Establishes canonical HTTP/JSON patterns for AILANG
- Reduces eval costs (fewer repair iterations)
- Improves AI confidence in generating HTTP/JSON code

**ROI**: 4-6 hours effort → +50pp success rate improvement on api_call_json

---

*Generated by ailang eval-analyze on 2025-10-22 11:29:43*
*Model: gpt5*
