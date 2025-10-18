# v0.4.0: Net Effect Enhancements

**Status**: 📋 Planned
**Priority**: P0 (Critical for AI API calling)
**Estimated**: ~800 LOC
**Duration**: 1-2 weeks
**Dependencies**: v0.3.0-alpha4 (Net effect foundation)
**Blocks**: OpenAI, Claude, and other AI APIs requiring custom headers

## Problem Statement

**Current State** (v0.3.0-alpha4):
- Net effect supports HTTP GET/POST
- Cannot set custom HTTP headers
- Cannot read environment variables
- Cannot parse JSON responses

**Impact**:
```ailang
-- ❌ BLOCKED in v0.3.0-alpha4
-- Cannot call OpenAI API (requires Authorization header)
let response = httpPost("https://api.openai.com/v1/chat/completions", body)
-- Error: OpenAI returns 401 Unauthorized (missing Authorization header)

-- ❌ BLOCKED in v0.3.0-alpha4
-- Cannot extract text from JSON response
let text = parseJSON(response).candidates[0].content.parts[0].text
-- Error: No JSON parsing support
```

**Business Impact**:
- Cannot integrate with 90% of modern AI APIs (OpenAI, Claude, Cohere, etc.)
- Cannot extract structured data from API responses
- Must hardcode secrets in source files (security risk)
- Eval harness works but Net effect doesn't (feature gap)

## Goals

### Primary Goals (Must Achieve)
1. **Custom HTTP Headers**: Set arbitrary headers on requests
2. **JSON Parsing**: Parse JSON strings to structured values
3. **Environment Variables**: Read env vars securely

### Secondary Goals (Nice to Have)
4. **Response Status/Headers**: Access HTTP status code and response headers
5. **Request Methods**: Support PUT, PATCH, DELETE

### Non-Goals (Deferred to v0.5.0+)
- Streaming responses
- WebSocket support
- OAuth flows
- Binary request/response bodies

## Design

### Feature 1: Custom HTTP Headers

**API Design**:
```ailang
import std/net (httpPostWithHeaders)

func callOpenAI() -> string ! {Net, Env} {
  let apiKey = getEnv("OPENAI_API_KEY");
  let headers = [
    ("Authorization", "Bearer " ++ apiKey),
    ("Content-Type", "application/json")
  ];

  let body = "{\"model\":\"gpt-4\",\"messages\":[...]}";
  httpPostWithHeaders("https://api.openai.com/v1/chat/completions", headers, body)
}
```

**Type Signature**:
```ailang
httpPostWithHeaders : (url: String, headers: [(String, String)], body: String) -> String ! {Net}
httpGetWithHeaders  : (url: String, headers: [(String, String)]) -> String ! {Net}
```

**Security Considerations**:
- **Block sensitive headers**: Prevent setting `Host`, `Content-Length`, `Connection`
- **Validate header names**: Only allow alphanumeric + dash (`-`)
- **Size limits**: Max 50 headers, max 8KB total header size
- **No header injection**: Validate no `\r\n` in header values

**Implementation**:
```go
// internal/effects/net.go
func netHttpPostWithHeaders(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Capability check
    if !ctx.HasCap("Net") {
        return nil, NewCapabilityError("Net")
    }

    // 2. Type checking: url (String), headers (List[(String, String)]), body (String)
    urlStr := args[0].(*eval.StringValue)
    headersList := args[1].(*eval.ListValue)  // List of tuples
    bodyStr := args[2].(*eval.StringValue)

    // 3. Validate headers
    headers := make(map[string]string)
    for _, hdr := range headersList.Elements {
        tuple := hdr.(*eval.TupleValue)
        name := tuple.Elements[0].(*eval.StringValue).Value
        value := tuple.Elements[1].(*eval.StringValue).Value

        // Security: Block sensitive headers
        if isBlockedHeader(name) {
            return nil, fmt.Errorf("E_NET_HEADER_BLOCKED: cannot set header: %s", name)
        }

        // Security: Validate header name/value
        if !isValidHeaderName(name) {
            return nil, fmt.Errorf("E_NET_HEADER_INVALID: invalid header name: %s", name)
        }
        if strings.ContainsAny(value, "\r\n") {
            return nil, fmt.Errorf("E_NET_HEADER_INJECTION: header value contains CRLF")
        }

        headers[name] = value
    }

    // 4. Build HTTP request
    req, err := http.NewRequest("POST", urlStr.Value, strings.NewReader(bodyStr.Value))
    for name, value := range headers {
        req.Header.Set(name, value)
    }

    // 5. Execute (reuse existing security validation)
    return executeRequest(ctx, req)
}

// Helper: Check if header is blocked
func isBlockedHeader(name string) bool {
    blocked := []string{"host", "content-length", "connection", "transfer-encoding"}
    lower := strings.ToLower(name)
    for _, b := range blocked {
        if lower == b {
            return true
        }
    }
    return false
}
```

**Stdlib Wrapper**:
```ailang
-- stdlib/std/net.ail
export func httpPostWithHeaders(url: String, headers: [(String, String)], body: String) -> String ! {Net} {
  _net_httpPostWithHeaders(url, headers, body)
}

export func httpGetWithHeaders(url: String, headers: [(String, String)]) -> String ! {Net} {
  _net_httpGetWithHeaders(url, headers)
}
```

**Tests**:
```go
func TestNetCustomHeaders(t *testing.T) {
    ctx := NewEffContext()
    ctx.Grant(NewCapability("Net"))
    ctx.Net = NewNetContext()

    t.Run("custom headers work", func(t *testing.T) {
        url := &eval.StringValue{Value: "https://httpbin.org/headers"}
        headers := &eval.ListValue{
            Elements: []eval.Value{
                &eval.TupleValue{
                    Elements: []eval.Value{
                        &eval.StringValue{Value: "X-Custom-Header"},
                        &eval.StringValue{Value: "test-value"},
                    },
                },
            },
        }

        result, err := netHttpGetWithHeaders(ctx, []eval.Value{url, headers})
        if err != nil {
            t.Fatalf("Expected success, got: %v", err)
        }
        // httpbin.org echoes headers back
        if !strings.Contains(result.(*eval.StringValue).Value, "X-Custom-Header") {
            t.Error("Custom header not sent")
        }
    })

    t.Run("blocked headers rejected", func(t *testing.T) {
        url := &eval.StringValue{Value: "https://example.com"}
        headers := &eval.ListValue{
            Elements: []eval.Value{
                &eval.TupleValue{
                    Elements: []eval.Value{
                        &eval.StringValue{Value: "Host"},
                        &eval.StringValue{Value: "evil.com"},
                    },
                },
            },
        }

        _, err := netHttpGetWithHeaders(ctx, []eval.Value{url, headers})
        if err == nil || !strings.Contains(err.Error(), "E_NET_HEADER_BLOCKED") {
            t.Error("Should block Host header")
        }
    })

    t.Run("header injection prevented", func(t *testing.T) {
        url := &eval.StringValue{Value: "https://example.com"}
        headers := &eval.ListValue{
            Elements: []eval.Value{
                &eval.TupleValue{
                    Elements: []eval.Value{
                        &eval.StringValue{Value: "X-Custom"},
                        &eval.StringValue{Value: "value\r\nInjected-Header: evil"},
                    },
                },
            },
        }

        _, err := netHttpGetWithHeaders(ctx, []eval.Value{url, headers})
        if err == nil || !strings.Contains(err.Error(), "E_NET_HEADER_INJECTION") {
            t.Error("Should prevent CRLF injection")
        }
    })
}
```

### Feature 2: Environment Variable Reading

**API Design**:
```ailang
import std/env (getEnv, hasEnv)

func loadAPIKey() -> String ! {Env} {
  if hasEnv("OPENAI_API_KEY") then {
    getEnv("OPENAI_API_KEY")
  } else {
    ""
  }
}
```

**Type Signatures**:
```ailang
getEnv : (name: String) -> String ! {Env}
hasEnv : (name: String) -> Bool ! {Env}
```

**Capability**: Requires `--caps Env` (new capability)

**Security Considerations**:
- Requires explicit `--caps Env` grant
- Logs environment variable names accessed (for audit)
- No way to list all env vars (only read specific keys)

**Implementation**:
```go
// internal/effects/env.go (NEW)
package effects

import (
    "fmt"
    "os"
    "github.com/sunholo/ailang/internal/eval"
)

func init() {
    RegisterOp("Env", "getEnv", envGetEnv)
    RegisterOp("Env", "hasEnv", envHasEnv)
}

func envGetEnv(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if !ctx.HasCap("Env") {
        return nil, NewCapabilityError("Env")
    }

    if len(args) != 1 {
        return nil, fmt.Errorf("E_ENV_TYPE_ERROR: getEnv: expected 1 argument, got %d", len(args))
    }

    nameStr, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("E_ENV_TYPE_ERROR: getEnv: expected String, got %T", args[0])
    }

    value := os.Getenv(nameStr.Value)
    return &eval.StringValue{Value: value}, nil
}

func envHasEnv(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if !ctx.HasCap("Env") {
        return nil, NewCapabilityError("Env")
    }

    if len(args) != 1 {
        return nil, fmt.Errorf("E_ENV_TYPE_ERROR: hasEnv: expected 1 argument, got %d", len(args))
    }

    nameStr, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("E_ENV_TYPE_ERROR: hasEnv: expected String, got %T", args[0])
    }

    _, exists := os.LookupEnv(nameStr.Value)
    return &eval.BoolValue{Value: exists}, nil
}
```

**Stdlib Wrapper**:
```ailang
-- stdlib/std/env.ail (NEW)
module std/env

-- Get environment variable value
-- Returns empty string if not set
export func getEnv(name: String) -> String ! {Env} {
  _env_getEnv(name)
}

-- Check if environment variable exists
export func hasEnv(name: String) -> Bool ! {Env} {
  _env_hasEnv(name)
}
```

**Tests**:
```go
func TestEnvOperations(t *testing.T) {
    ctx := NewEffContext()
    ctx.Grant(NewCapability("Env"))

    t.Run("getEnv existing var", func(t *testing.T) {
        os.Setenv("TEST_VAR", "test-value")
        defer os.Unsetenv("TEST_VAR")

        name := &eval.StringValue{Value: "TEST_VAR"}
        result, err := envGetEnv(ctx, []eval.Value{name})

        if err != nil {
            t.Fatalf("Expected success, got: %v", err)
        }
        if result.(*eval.StringValue).Value != "test-value" {
            t.Errorf("Expected 'test-value', got: %s", result.(*eval.StringValue).Value)
        }
    })

    t.Run("getEnv missing var returns empty string", func(t *testing.T) {
        name := &eval.StringValue{Value: "NONEXISTENT_VAR"}
        result, err := envGetEnv(ctx, []eval.Value{name})

        if err != nil {
            t.Fatalf("Expected success, got: %v", err)
        }
        if result.(*eval.StringValue).Value != "" {
            t.Errorf("Expected empty string, got: %s", result.(*eval.StringValue).Value)
        }
    })

    t.Run("hasEnv checks existence", func(t *testing.T) {
        os.Setenv("TEST_VAR", "value")
        defer os.Unsetenv("TEST_VAR")

        name1 := &eval.StringValue{Value: "TEST_VAR"}
        result1, _ := envHasEnv(ctx, []eval.Value{name1})
        if !result1.(*eval.BoolValue).Value {
            t.Error("Expected true for existing var")
        }

        name2 := &eval.StringValue{Value: "NONEXISTENT"}
        result2, _ := envHasEnv(ctx, []eval.Value{name2})
        if result2.(*eval.BoolValue).Value {
            t.Error("Expected false for non-existent var")
        }
    })

    t.Run("requires Env capability", func(t *testing.T) {
        ctxNoCap := NewEffContext()  // No capabilities

        name := &eval.StringValue{Value: "TEST"}
        _, err := envGetEnv(ctxNoCap, []eval.Value{name})

        if err == nil {
            t.Fatal("Should require Env capability")
        }
        capErr, ok := err.(*CapabilityError)
        if !ok || capErr.Effect != "Env" {
            t.Errorf("Expected CapabilityError for Env, got: %v", err)
        }
    })
}
```

### Feature 3: JSON Parsing

**API Design**:
```ailang
import std/json (parseJSON, getValue)

func extractGeminiText(response: String) -> String {
  let json = parseJSON(response);
  let candidates = getValue(json, "candidates");
  let first = listGet(candidates, 0);
  let content = getValue(first, "content");
  let parts = getValue(content, "parts");
  let firstPart = listGet(parts, 0);
  getValue(firstPart, "text")
}
```

**Type Signatures**:
```ailang
type JSONValue =
  | JSONString(String)
  | JSONNumber(Float)
  | JSONBool(Bool)
  | JSONNull
  | JSONArray([JSONValue])
  | JSONObject([(String, JSONValue)])

parseJSON : String -> JSONValue
getValue  : (JSONValue, String) -> JSONValue
toJSONString : JSONValue -> String
```

**Implementation**:
- Use Go's `encoding/json` package
- Map JSON types to AILANG ADT values
- Pure function (no effects required)

**Stdlib Wrapper**:
```ailang
-- stdlib/std/json.ail (NEW)
module std/json

type JSONValue =
  | JSONString(String)
  | JSONNumber(Float)
  | JSONBool(Bool)
  | JSONNull
  | JSONArray([JSONValue])
  | JSONObject([(String, JSONValue)])

-- Parse JSON string to structured value
export func parseJSON(json: String) -> JSONValue {
  _json_parse(json)
}

-- Get field value from JSON object
export func getValue(obj: JSONValue, key: String) -> JSONValue {
  _json_getValue(obj, key)
}

-- Convert JSON value back to string
export func toJSONString(val: JSONValue) -> String {
  _json_toString(val)
}
```

### Feature 4: Response Status/Headers (Secondary Goal)

**API Design**:
```ailang
import std/net (httpGetFull)

type HTTPResponse = {
  status: Int,
  headers: [(String, String)],
  body: String
}

func checkStatus(url: String) -> Bool ! {Net} {
  let response = httpGetFull(url);
  response.status == 200
}
```

**Deferred**: This is lower priority. Start with custom headers + JSON parsing.

## Implementation Plan

### Phase 1: Custom Headers (Week 1, Days 1-3)
1. **Day 1**: Implement `_net_httpPostWithHeaders()` and `_net_httpGetWithHeaders()`
2. **Day 2**: Add header validation and security checks
3. **Day 3**: Write comprehensive tests, stdlib wrappers

### Phase 2: Environment Variables (Week 1, Days 4-5)
1. **Day 4**: Implement `_env_getEnv()` and `_env_hasEnv()`
2. **Day 5**: Write tests, stdlib wrapper, integrate with CLI (`--caps Env`)

### Phase 3: JSON Parsing (Week 2, Days 1-3)
1. **Day 1**: Design JSONValue ADT, implement parser
2. **Day 2**: Implement `getValue()`, `toJSONString()` helpers
3. **Day 3**: Write comprehensive tests, stdlib wrapper

### Phase 4: Integration & Examples (Week 2, Days 4-5)
1. **Day 4**: Update OpenAI and Gemini examples to use new features
2. **Day 5**: Documentation, CHANGELOG update, release v0.4.0

## Testing Strategy

**Unit Tests**:
- Header validation (block sensitive headers, prevent injection)
- Environment variable reading (existing, missing, capability check)
- JSON parsing (all JSON types, nested objects, arrays)

**Integration Tests**:
- Call OpenAI API with Authorization header
- Call Gemini API and parse JSON response
- Read API key from environment variable

**Security Tests**:
- Header injection attempts
- Blocked header attempts
- Env capability enforcement

## Breaking Changes

None. All new features are additive.

## Migration Guide

**For OpenAI API (before v0.4.0)**:
```ailang
-- Use eval harness
ailang eval --model gpt-4 --provider openai benchmarks/simple_math.md
```

**For OpenAI API (v0.4.0+)**:
```ailang
import std/net (httpPostWithHeaders)
import std/env (getEnv)
import std/json (parseJSON, getValue)

func callOpenAI(prompt: String) -> String ! {Net, Env} {
  let apiKey = getEnv("OPENAI_API_KEY");
  let headers = [("Authorization", "Bearer " ++ apiKey)];
  let body = "{\"model\":\"gpt-4\",\"messages\":[{\"role\":\"user\",\"content\":\"" ++ prompt ++ "\"}]}";

  let response = httpPostWithHeaders("https://api.openai.com/v1/chat/completions", headers, body);
  let json = parseJSON(response);
  let choices = getValue(json, "choices");
  let first = listGet(choices, 0);
  let message = getValue(first, "message");
  getValue(message, "content")
}
```

## Error Codes

**Custom Headers**:
- `E_NET_HEADER_BLOCKED` - Attempted to set blocked header (Host, Content-Length, etc.)
- `E_NET_HEADER_INVALID` - Invalid header name format
- `E_NET_HEADER_INJECTION` - Header value contains CRLF

**Environment Variables**:
- `E_ENV_CAP_MISSING` - Env capability not granted
- `E_ENV_TYPE_ERROR` - Wrong argument type

**JSON Parsing**:
- `E_JSON_PARSE_ERROR` - Invalid JSON syntax
- `E_JSON_TYPE_ERROR` - Type mismatch (e.g., accessing string as object)
- `E_JSON_KEY_NOT_FOUND` - Object key doesn't exist

## Future Work (v0.5.0+)

**Streaming Responses**:
```ailang
httpStream : (String, (String -> ())) -> () ! {Net, IO}
```

**Binary Bodies**:
```ailang
httpPostBytes : (String, Bytes) -> Bytes ! {Net}
```

**OAuth Support**:
```ailang
oauth2Token : (String, String, [String]) -> String ! {Net, Env}
```

## References

- Net effect foundation: `internal/effects/net.go`
- Eval harness headers: `internal/eval_harness/api_google.go:70`
- Go JSON package: https://pkg.go.dev/encoding/json
- HTTP header security: https://owasp.org/www-community/attacks/HTTP_Response_Splitting

## Bug: If-Then-Else Function Calls

**Issue Reported**: "The parser doesn't like function calls in if-then-else"

**Example**:
```ailang
if condition then doSomething() else doOther()
-- Parse error: expected next token to be else, got () instead
```

**Root Cause**: Parser expects simple expressions in if-then-else, not function application

**Workaround**:
```ailang
-- Use blocks
if condition then { doSomething() } else { doOther() }

-- Or assign to variable
let action = if condition then 1 else 0;
if action == 1 then doSomething() else doOther()
```

**Fix in v0.4.0**: Update parser to allow function calls in if-then-else branches
- File: `internal/parser/parser.go`
- Function: `parseIfExpression()`
- Allow function application in then/else branches
