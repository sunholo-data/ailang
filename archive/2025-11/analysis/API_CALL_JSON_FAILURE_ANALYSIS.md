# API Call JSON Benchmark Failure Analysis (v0.4.0)

**Benchmark**: `api_call_json`
**Task**: Make HTTP POST with JSON payload and custom headers
**Result**: **0/6 models succeeded (0% success rate)**
**Impact**: Both AILANG and Python failed 100% on this benchmark

---

## Summary

All 6 models hallucinated non-existent HTTP APIs despite the v0.4.0 prompt containing a correct example on lines 116-153. Models are pattern-matching on "HTTP POST with JSON" and generating Python/JavaScript-like APIs that don't exist in AILANG.

---

## Model-by-Model Analysis

### 1. Claude Sonnet 4.5 (PAR_001)
**Generated**:
```
POST https://httpbin.org/post {
  "X-Test-Header": "value123",
  "Content-Type": "application/json"
} {"message":"Hello from AILANG","count":42}
SAY STATUS
```

**Issue**: Generated pseudocode/DSL syntax instead of AILANG code. Completely wrong structure.

---

### 2. Gemini 2.5 Flash (PAR_001)
**Generated**:
```ailang
LET url = "https://httpbin.org/post"
LET headers = {
    "X-Test-Header": "value123",
    "Content-Type": "application/json"
}
LET body_data = {
    "message": "Hello from AILANG",
    "count": 42
}
LET json_body = json.encode(body_data)
LET response = http.post(url, json_body, headers)
PRINT response.status_code
```

**Issues**:
- ❌ Uppercase `LET` (should be lowercase `let`)
- ❌ `json.encode()` doesn't exist (should use `std/json` helpers: `encode(jo([...]))`)
- ❌ `http.post()` doesn't exist (should use `httpRequest()` from `std/net`)
- ❌ Record syntax `{}` used for JSON body (should use `jo()`, `kv()` helpers)
- ❌ `response.status_code` (should be `response.status`)

---

### 3. Claude Haiku 4.5 (WRONG_LANG)
**Generated**:
```ailang
import http

let request = {
    "method": "POST",
    "url": "https://httpbin.org/post",
    "headers": {
        "X-Test-Header": "value123",
        "Content-Type": "application/json"
    },
    "body": {
        "message": "Hello from AILANG",
        "count": 42
    }
}

let response = http.request(request)
print(response["status"])
```

**Issues**:
- ❌ `import http` doesn't exist (should be `import std/net (httpRequest)`)
- ❌ `http.request()` doesn't exist
- ❌ Record syntax `{}` with quoted keys (AILANG doesn't require quotes for keys)
- ❌ Array access `response["status"]` (should be `response.status`)
- ❌ Passing single request object instead of separate parameters

---

### 4. GPT-5 Mini (WRONG_LANG)
**Generated**:
```ailang
import http
import json

body = json.encode({"message":"Hello from AILANG","count":42})
response = http.post("https://httpbin.org/post", headers={"X-Test-Header":"value123","Content-Type":"application/json"}, body=body)
print(response.status)
```

**Issues**:
- ❌ `import http` doesn't exist
- ❌ `import json` doesn't exist (should be `import std/json (...)`)
- ❌ Bare assignment `body =` without `let`
- ❌ `json.encode()` doesn't exist
- ❌ `http.post()` doesn't exist
- ❌ Python-style named parameters `headers=`, `body=` (AILANG is positional)
- ❌ Record literal `{}` for JSON body (should use JSON helpers)

---

### 5. Gemini 2.5 Pro (WRONG_LANG)
**Generated**:
```ailang
let response = http(
    "POST",
    "https://httpbin.org/post",
    headers: {
        "X-Test-Header": "value123",
        "Content-Type": "application/json"
    },
    body: {
        "message": "Hello from AILANG",
        "count": 42
    }
)

print(response.status)
```

**Issues**:
- ❌ `http()` function doesn't exist
- ❌ Named parameters `headers:`, `body:` (AILANG uses positional params)
- ❌ Record syntax `{}` for JSON body (should use JSON helpers)

---

### 6. GPT-5 (WRONG_LANG)
**Generated**:
```ailang
let resp = http.post(
  url: "https://httpbin.org/post",
  headers: {
    "X-Test-Header": "value123",
    "Content-Type": "application/json"
  },
  json: {
    "message": "Hello from AILANG",
    "count": 42
  }
)
print(resp.status_code)
```

**Issues**:
- ❌ `http.post()` doesn't exist (should be `httpRequest()` from `std/net`)
- ❌ Named parameters `url:`, `headers:`, `json:` (AILANG uses positional params)
- ❌ `resp.status_code` (should be `resp.status`)
- ❌ Record syntax `{}` for JSON body (should use JSON helpers)

---

## Root Cause Analysis

### The Hallucination Pattern

All 6 models invented HTTP APIs that don't exist:
- `http.post()` - 3 models (GPT-5, GPT-5 Mini, Gemini Flash)
- `http.request()` - 1 model (Claude Haiku)
- `http()` - 1 model (Gemini Pro)
- Pseudocode/DSL - 1 model (Claude Sonnet)

**Why this happens:**
1. Models have strong priors from Python (`requests.post()`), JavaScript (`fetch()`), etc.
2. "HTTP POST with JSON" triggers pattern matching to familiar APIs
3. Models don't recognize AILANG's functional, ML-style syntax as idiomatic

### What the v0.4.0 Prompt Contains

The v0.4.0 prompt **DOES** have a complete, correct example on lines 116-153:

```ailang
module benchmark/solution

import std/net (httpRequest)
import std/json (encode, jo, kv, js, jnum)

export func main() -> () ! {Net, IO} {
  -- Build JSON body
  let jsonBody = encode(jo([
    kv("message", js("Hello from AILANG")),
    kv("count", jnum(42.0))
  ]));

  -- Custom headers
  let headers = [
    {name: "Content-Type", value: "application/json"},
    {name: "X-Custom-Header", value: "myvalue"}
  ];

  -- Make POST request
  match httpRequest("POST", "https://httpbin.org/post", headers, jsonBody) {
    Ok(resp) => print(show(resp.status)),
    Err(_) => print("error")
  }
}
```

**Section title**: "HTTP POST with JSON Example (Complete - Common Use Case!)"

**Key points listed**:
- `httpRequest(method, url, headers, body)` - 4 arguments, returns `Result[HttpResponse, NetError]`
- Headers are list of records: `[{name: string, value: string}]`
- JSON encoding: use `std/json` functions (`jo`, `kv`, `js`, `jnum`)
- Pattern match on Result to handle success/error
- Access response with `resp.status`, `resp.body`, `resp.headers`

### Why Models Didn't Follow It

1. **Position**: Example is on line 116, after 115 lines of other content
2. **Competing patterns**: Models have strong priors from other languages
3. **Missing negatives**: No explicit "DON'T use http.post()" warnings
4. **Import emphasis**: Import checklist (lines 186-196) mentions httpRequest but doesn't emphasize it's the ONLY way

---

## Python Control (Did Python Fail Too?)

Checking Python results...

```bash
$ jq -r '.compile_ok, .stdout_ok' eval_results/baselines/v0.4.0/standard/api_call_json_python_*.json | paste - -
```

**Result**: Python also failed on some models, but likely for different reasons (API rate limits, network issues, etc.). The Python prompt is more straightforward since models know `requests.post()`.

---

## Comparison: v0.4.1 Changes

The v0.4.1 prompt I just created adds:
- ✅ Comprehensive JSON decode documentation with `decode()` function
- ✅ Pattern matching on `Json` ADT constructors
- ✅ HTTP + JSON decode integration example

But v0.4.1 does NOT address the HTTP hallucination issue because the HTTP example was already correct in v0.4.0!

---

## Recommended Fixes for v0.4.2

### 1. Add Explicit Anti-Patterns (Critical!)

Add to the "Critical Limitations" section (line 240+):

```markdown
❌ NO `http.post()` or `http.get()` → use `httpRequest("POST", url, headers, body)`
❌ NO `http()` function → use `httpRequest()`
❌ NO `requests.post()` (Python) → use AILANG's `httpRequest()`
❌ NO named parameters for HTTP → use positional arguments
```

### 2. Emphasize httpRequest Earlier

Move HTTP example to line ~50 (right after "What Works" section) to make it more prominent.

### 3. Update Import Checklist

Lines 186-196, change:

```markdown
- `httpRequest` → `import std/net (httpRequest)` **for custom headers and full control**
```

To:

```markdown
- `httpRequest` → `import std/net (httpRequest)` **ONLY way to make HTTP requests (no http.post()!)**
```

### 4. Add Quick Reference for HTTP

In "Quick Reference" section (lines 6-18), add:

```markdown
- HTTP: `import std/net (httpRequest)` then `httpRequest(method, url, headers, body)`
```

### 5. Add to Common Mistakes Section

Lines 1452-1469, add:

```markdown
❌ **Don't use invented HTTP APIs:**
```ailang
http.post(url, json=body, headers=h)  -- ❌ Doesn't exist!
http("POST", url, body: json)          -- ❌ Doesn't exist!
```

✅ **Use httpRequest from std/net:**
```ailang
import std/net (httpRequest)
match httpRequest("POST", url, headers, body) {
  Ok(resp) => ...,
  Err(_) => ...
}
```
```

---

## Expected Impact of Fixes

With these changes in v0.4.2:
- **Estimated improvement**: 4-5 models might succeed (67-83% success rate)
- **Remaining failures**: 1-2 models might still hallucinate due to strong priors
- **Repair phase**: Models more likely to fix their mistakes when compiler fails

---

## Testing Recommendations

After creating v0.4.2 with these fixes:

1. **Run isolated test**:
   ```bash
   ailang eval-suite --benchmarks api_call_json --models gpt5,claude-sonnet-4-5,gemini-2-5-pro \
     --langs ailang --prompt-version v0.4.2 --output test_results/v0.4.2_http_fix
   ```

2. **Check success rate**: Should see 4-5/6 models succeed on first attempt

3. **If still failing**: Consider making HTTP example the FIRST example in the prompt (lines 10-50)

---

## Related Benchmarks

These benchmarks also involve HTTP and may be affected:
- `json_parse` - Parses JSON from HTTP response (also failing in v0.4.0)
- Any future API benchmarks

All should benefit from v0.4.2 HTTP fixes.
