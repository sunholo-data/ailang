# Deep Dive: api_call_json Benchmark Failures

**Date**: 2025-11-16
**Version**: v0.4.5
**Benchmark**: `api_call_json` (6/6 failures, 0% success)
**Status**: HTTP IS IMPLEMENTED, but models don't use it correctly

## TL;DR

**HTTP/Net is fully implemented in AILANG**, but the `api_call_json` benchmark has **0% success** because:

1. **Teaching prompt has examples, but models ignore them** (generate Python/shell syntax instead)
2. **Benchmark difficulty is "hard"** (requires JSON encoding + HTTP + custom headers)
3. **Models don't follow AILANG syntax** even when examples are provided
4. **Not an implementation gap** - this is a **model behavior issue**

## What's Actually Implemented

### std/net Module (v0.3.8+)

**File**: `std/net.ail`

**Functions**:
```ailang
-- Simple GET
httpGet(url: string) -> string ! {Net}

-- Simple POST
httpPost(url: string, body: string) -> string ! {Net}

-- Advanced with custom headers
httpRequest(
  method: string,
  url: string,
  headers: List[{name: string, value: string}],
  body: string
) -> Result[HttpResponse, NetError] ! {Net}
```

**Types**:
```ailang
type HttpResponse = {
  status: int,
  headers: List[{name: string, value: string}],
  body: string,
  ok: bool
}

type NetError =
  | Transport(string)
  | DisallowedHost(string)
  | InvalidHeader(string)
  | BodyTooLarge(string)
```

**Security features**:
- HTTPS enforced by default
- IP blocking (localhost, private IPs)
- Domain allowlist support
- DNS rebinding prevention
- Body size limits (5MB)
- Header validation
- Redirect limits (max 5)

### Examples in Teaching Prompt

The v0.4.5 teaching prompt **DOES include HTTP examples**:

**Example 1: HTTP POST with JSON**
```ailang
import std/net (httpRequest)
import std/json (encode, jo, kv, js, jnum)

export func main() -> () ! {Net, IO} {
  let jsonBody = encode(jo([
    kv("message", js("Hello from AILANG")),
    kv("count", jnum(42.0))
  ]));

  let headers = [
    {name: "X-Test-Header", value: "value123"},
    {name: "Content-Type", value: "application/json"}
  ];

  match httpRequest("POST", "https://httpbin.org/post", headers, jsonBody) {
    Ok(resp) => println(show(resp.status)),
    Err(Transport(msg)) => println("Error: " ++ msg),
    ...
  }
}
```

**Example 2: HTTP + JSON decode**
```ailang
import std/net (httpRequest)
import std/json (Json, JObject, decode)

export func main() -> () ! {Net, IO} {
  match httpRequest("GET", url, [], "") {
    Ok(resp) => match decode(resp.body) {
      Ok(JObject(kvs)) => ...,
      Err(msg) => ...
    },
    Err(...) => ...
  }
}
```

## What Models Actually Generated

### Claude Sonnet 4.5 Output
**Error**: PAR_001 (parse error)

**Generated code**:
```
http-post "https://httpbin.org/post" {
  "X-Test-Header": "value123",
  "Content-Type": "application/json"
} '{"message":"Hello from AILANG","count":42}' response

get response "status" status
print status
```

**Analysis**:
- Invented shell-like DSL syntax
- No module declaration
- No function syntax
- Completely ignored AILANG examples in prompt

### GPT-5 Output
**Error**: IMPERATIVE (wrong language style)

**Generated code**:
```python
response = http.post(
  url: "https://httpbin.org/post",
  headers: {
    "X-Test-Header": "value123",
    "Content-Type": "application/json"
  },
  json: {"message":"Hello from AILANG","count":42}
)
print(response.status_code)
```

**Analysis**:
- Python-style syntax
- No module declaration
- Imperative assignment (`response = ...`)
- Named parameters (`url:`, `headers:`, `json:`)
- Ignored AILANG functional style

### Gemini Output
**Error**: IMPERATIVE

**Generated code**:
```python
import requests

response = requests.post(
    "https://httpbin.org/post",
    headers={
        "X-Test-Header": "value123",
        "Content-Type": "application/json"
    },
    json={"message": "Hello from AILANG", "count": 42}
)

print(response.status_code)
```

**Analysis**:
- Pure Python with requests library
- Completely ignored AILANG
- Didn't even try to follow functional style

## Why Did Models Fail?

### Theory 1: Prompt Example Not Close Enough
The prompt example uses `match httpRequest(...)` with Result handling.
The benchmark asks for "Prints ONLY the response status code".

Models might struggle to connect:
1. Example: Full match/error handling
2. Task: Just print status code

**Verdict**: Partially true - example could be simpler

### Theory 2: Benchmark Complexity
The task requires:
1. JSON encoding (`encode(jo(...))`)
2. Custom headers (`[{name: ..., value: ...}]`)
3. HTTP POST (`httpRequest("POST", ...)`)
4. Error handling (`match ... Ok(resp) => ...`)
5. Status extraction (`resp.status`)

**Verdict**: True - this is a "hard" benchmark (marked as such)

### Theory 3: Models Default to Python for HTTP
Models have strong priors:
- "HTTP request" → `requests.post()`
- "JSON payload" → Python dict syntax

These priors override the teaching prompt.

**Verdict**: TRUE - This is the main issue

### Theory 4: Teaching Prompt Position
HTTP examples are ~200-300 lines into the prompt.
Models might prioritize early examples.

**Verdict**: Partially true - examples could be earlier

## Comparison: Why Some Benchmarks Succeed

### recursion_factorial (83% success)
```ailang
export func factorial(n: int) -> int =
  if n <= 1 then 1 else n * factorial(n - 1)
```

**Why it works**:
- Simple, single concept (recursion)
- No external libraries needed
- Matches functional programming intuition

### json_parse (0% success)
```python
# Models generate:
let people = parse_json(json_str)  # ❌ Function doesn't exist
```

**Why it fails**:
- Models expect `parse_json()` or `json.parse()`
- AILANG requires: `decode()` returning `Result[Json, string]`
- Must pattern match on JSON ADT
- Not obvious from other languages

### Shared pattern: External dependencies are hard
Benchmarks requiring `std/net` or `std/json` have **lower success rates**.

## Root Cause Analysis

**The real issue is NOT implementation - it's MODEL BEHAVIOR**:

1. **Strong priors for HTTP/JSON** → Models default to Python-style
2. **Complexity** → Multiple concepts (JSON + HTTP + headers) in one task
3. **ADT pattern matching** → Unfamiliar to models trained on imperative code
4. **Teaching prompt not enough** → Examples present but not persuasive

## Proposed Solutions

### Solution 1: Simplify Teaching Prompt Example (HIGH PRIORITY)

**Current** (complex):
```ailang
match httpRequest("POST", url, headers, jsonBody) {
  Ok(resp) => match decode(resp.body) {
    Ok(JObject(kvs)) => ...,
    Err(msg) => ...
  },
  Err(Transport(msg)) => ...
}
```

**Proposed** (simple):
```ailang
-- Simple HTTP GET (no error handling for clarity)
import std/net (httpRequest)
import std/io (println)

export func main() -> () ! {Net, IO} {
  match httpRequest("GET", "https://example.com", [], "") {
    Ok(resp) => println(show(resp.status)),  -- Just print status!
    Err(_) => println("Error")
  }
}

-- HTTP POST with custom headers (minimal JSON)
import std/net (httpRequest)

export func main() -> () ! {Net, IO} {
  let headers = [{name: "Content-Type", value: "application/json"}];
  let body = "{\"message\":\"Hello\"}";  -- Plain string JSON for simplicity

  match httpRequest("POST", "https://httpbin.org/post", headers, body) {
    Ok(resp) => println(show(resp.status)),
    Err(_) => println("Error")
  }
}
```

**Key changes**:
- Remove nested JSON decoding (focus on HTTP only)
- Show simple status printing (matches benchmark task)
- Use string literals for JSON body (simpler than `encode(jo(...))`)
- Minimal error handling (just `Err(_)`)

### Solution 2: Add "Common Mistakes" Section

Add to teaching prompt:
```markdown
## HTTP Requests - Common Mistakes

❌ **WRONG (Python style)**:
```python
response = http.post(url, headers={...}, json={...})
print(response.status_code)
```

✅ **CORRECT (AILANG)**:
```ailang
import std/net (httpRequest)
import std/io (println)

export func main() -> () ! {Net, IO} {
  let headers = [{name: "Content-Type", value: "application/json"}];
  let body = "{\"key\": \"value\"}";

  match httpRequest("POST", url, headers, body) {
    Ok(resp) => println(show(resp.status)),
    Err(_) => println("Error")
  }
}
```

**Key differences**:
- ✅ `import std/net (httpRequest)` NOT `import requests`
- ✅ `httpRequest("POST", ...)` NOT `http.post(...)`
- ✅ Headers as list: `[{name: ..., value: ...}]` NOT dict `{...}`
- ✅ Pattern matching: `match ... { Ok(resp) => ... }` NOT `response.status_code`
```

### Solution 3: Move HTTP Examples Earlier in Prompt

Currently HTTP examples are ~200-300 lines into prompt.
Move to **"Common Tasks"** section near the top.

### Solution 4: Update Benchmark Prompt (ALTERNATIVE)

Instead of:
```
Write a program that:
1. Makes an HTTP POST request...
```

Try:
```
Write a program using AILANG's std/net.httpRequest function that:
1. Makes an HTTP POST request...
```

**Rationale**: Explicitly mention the function name to guide models.

### Solution 5: Accept This Failure (DEFER)

**Acceptance criteria**:
- HTTP IS implemented ✅
- Examples ARE in prompt ✅
- Only 2.1% of failures (6/284)
- Models have strong Python priors for HTTP

**Recommendation**: Defer improvements to v0.5.0
- Focus on prompt improvements for higher-impact benchmarks
- 66% of failures are simpler issues (for loops, modulo, etc.)

## Impact Assessment

### If we fix api_call_json (optimistic):
- Simplified examples → 0% → 50% success? (3/6 models)
- **Impact**: 68.6% → 69.6% (+1 percentage point)

### If we fix teaching prompt (all priorities):
- 66% of failures → 75%+ success
- **Impact**: 68.6% → 75%+ (+6+ percentage points)

**Recommendation**: Focus on Priority 1 (teaching prompt) first.
api_call_json is lower ROI given implementation time.

## Conclusion

**HTTP/Net is fully implemented and working**. The api_call_json benchmark failures are due to:

1. **Model behavior** (strong Python priors for HTTP)
2. **Benchmark complexity** (JSON + HTTP + headers)
3. **Teaching prompt could be simpler** (but examples exist)

**This is NOT an implementation gap** - it's a model training/prompt engineering issue.

**Recommendation for v0.4.6**:
1. ✅ Simplify HTTP examples in teaching prompt
2. ✅ Add "Common Mistakes" section
3. ✅ Move HTTP examples earlier in prompt
4. ⏭️ Defer further work to v0.5.0 if still failing

**Expected improvement**: 0% → 30-50% success (realistic)
**Better ROI**: Focus on for loops, modulo, multi-module (66% of failures)

## Correction to implementation-gaps-analysis.md

**Update needed**: Remove HTTP/Net from "Missing Features" section.
Move to "Implemented But Models Struggle" section.

**Add new section**:
```markdown
## Implemented But Models Struggle

### HTTP/Net (std/net)
**Status**: FULLY IMPLEMENTED (v0.3.8+)
**Benchmark**: api_call_json (0% success)
**Issue**: Models generate Python-style HTTP requests despite examples

**Functions available**:
- `httpGet(url)` - Simple GET
- `httpPost(url, body)` - Simple POST
- `httpRequest(method, url, headers, body)` - Advanced

**Problem**: Strong Python priors override teaching prompt
**Solution**: Simplify examples, add "Common Mistakes" section
**Expected improvement**: 0% → 30-50% (v0.4.6)
```
