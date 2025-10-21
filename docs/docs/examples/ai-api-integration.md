# AI API Integration Examples

AILANG v0.3.9 introduces first-class support for calling AI APIs with HTTP headers, JSON encoding, and structured error handling.

## Overview

**Features:**
- ✅ Custom HTTP headers (Authorization, API keys, etc.)
- ✅ JSON encoding with type-safe ADT construction
- ✅ Result-based error handling
- ✅ Security features (header validation, HTTPS enforcement)
- ✅ Support for OpenAI, Anthropic, Google, and other AI APIs

## Quick Start

### 1. Claude (Anthropic) API

**Example:** `examples/claude_haiku_call.ail`

```ailang
import std/json (encode, jo, ja, kv, js, jnum)
import std/net (httpRequest, NetError, Transport, InvalidHeader)
import std/io (println)

func chatClaude(prompt: string, apiKey: string) -> string ! {Net, IO} {
  let url = "https://api.anthropic.com/v1/messages";
  let headers = [
    {name: "x-api-key", value: apiKey},
    {name: "anthropic-version", value: "2023-06-01"},
    {name: "content-type", value: "application/json"}
  ];

  let body = encode(
    jo([
      kv("model", js("claude-3-5-haiku-20241022")),
      kv("max_tokens", jnum(100.0)),
      kv("messages", ja([
        jo([
          kv("role", js("user")),
          kv("content", js(prompt))
        ])
      ]))
    ])
  );

  match httpRequest("POST", url, headers, body) {
    Ok(resp) =>
      if resp.ok then resp.body
      else concat_String("HTTP error: ", show(resp.status))
    Err(err) => match err {
      Transport(msg) => concat_String("Network error: ", msg)
      InvalidHeader(hdr) => concat_String("Invalid header: ", hdr)
      -- ... handle other errors
    }
  }
}
```

**Usage:**
```bash
# Get API key from https://console.anthropic.com/
export ANTHROPIC_API_KEY="sk-ant-..."

# Replace placeholder in the file, then run:
ailang run --caps Net,IO --entry main examples/claude_haiku_call.ail
```

**Example Output:**
```
===========================================
AILANG + Claude Haiku API Example
===========================================

Prompt: Write a haiku about functional programming

Calling Claude Haiku API...
✓ Status: 200

Response from Claude:
---
{"model":"claude-3-5-haiku-20241022","id":"msg_...",
 "content":[{"type":"text","text":"Pure functions flow by\n
Immutable data glides smooth\nCode without side paths"}],
 "usage":{"input_tokens":14,"output_tokens":98}}
---

✓ Example completed successfully!
```

### 2. OpenAI API

**Example:** `examples/ai_call.ail`

```ailang
import std/json (encode, jo, ja, kv, js, jnum)
import std/net (httpRequest)

func chatOpenAI(prompt: string, apiKey: string) -> string ! {Net, IO} {
  let url = "https://api.openai.com/v1/chat/completions";
  let headers = [
    {name: "Authorization", value: concat_String("Bearer ", apiKey)},
    {name: "Content-Type", value: "application/json"}
  ];

  let body = encode(
    jo([
      kv("model", js("gpt-4o-mini")),
      kv("messages", ja([
        jo([kv("role", js("user")), kv("content", js(prompt))])
      ]))
    ])
  );

  match httpRequest("POST", url, headers, body) {
    Ok(resp) => resp.body
    Err(err) => -- ... handle errors
  }
}
```

**Usage:**
```bash
# Get API key from https://platform.openai.com/api-keys
export OPENAI_API_KEY="sk-..."

ailang run --caps Net,IO --entry main examples/ai_call.ail
```

## JSON Encoding

AILANG provides a type-safe JSON ADT for constructing JSON payloads:

### Json ADT

```ailang
type Json =
  | JNull                                      -- null
  | JBool(bool)                                -- true/false
  | JNumber(float)                             -- 42.0, 3.14
  | JString(string)                            -- "hello"
  | JArray(List[Json])                         -- [1, 2, 3]
  | JObject(List[{key: string, value: Json}])  -- {"a": 1}
```

### Convenience Helpers

```ailang
import std/json (jn, jb, jnum, js, ja, jo, kv, encode)

-- Primitives
jn()            -- JNull
jb(true)        -- JBool(true)
jnum(42.0)      -- JNumber(42.0)
js("hello")     -- JString("hello")

-- Collections
ja([jnum(1.0), jnum(2.0)])  -- JArray([JNumber(1.0), JNumber(2.0)])
jo([kv("x", jnum(1.0))])    -- JObject([{key: "x", value: JNumber(1.0)}])

-- Encoding
encode(jo([kv("name", js("Alice"))]))  -- {"name":"Alice"}
```

### Complex Example

```ailang
-- Build a complex API request
let request = jo([
  kv("model", js("gpt-4")),
  kv("temperature", jnum(0.7)),
  kv("messages", ja([
    jo([
      kv("role", js("system")),
      kv("content", js("You are a helpful assistant"))
    ]),
    jo([
      kv("role", js("user")),
      kv("content", js("Hello!"))
    ])
  ])),
  kv("functions", ja([
    jo([
      kv("name", js("get_weather")),
      kv("parameters", jo([
        kv("type", js("object")),
        kv("properties", jo([
          kv("location", jo([kv("type", js("string"))]))
        ]))
      ]))
    ])
  ]))
]);

let json = encode(request);
-- Result: {"model":"gpt-4","temperature":0.7,"messages":[...],...}
```

## HTTP Request Function

### Signature

```ailang
httpRequest(
  method: string,
  url: string,
  headers: List[{name: string, value: string}],
  body: string
) -> Result[HttpResponse, NetError] ! {Net}
```

### Types

```ailang
type HttpResponse = {
  status: int,
  headers: List[{name: string, value: string}],
  body: string,
  ok: bool  -- true if status 200-299
}

type NetError =
  | Transport(string)      -- Network/connection error
  | DisallowedHost(string) -- Domain not in allowlist
  | InvalidHeader(string)  -- Blocked header name
  | BodyTooLarge(string)   -- Response > 5MB
```

### Error Handling

```ailang
match httpRequest("POST", url, headers, body) {
  Ok(resp) =>
    if resp.ok then {
      -- Success: status 200-299
      processResponse(resp.body)
    } else {
      -- HTTP error: 4xx or 5xx
      handleHTTPError(resp.status)
    }
  Err(err) => match err {
    Transport(msg) => {
      -- Network error: DNS, timeout, TLS, etc.
      logError(concat_String("Network: ", msg))
    }
    DisallowedHost(host) => {
      -- Security: domain not in allowlist
      logError(concat_String("Blocked: ", host))
    }
    InvalidHeader(name) => {
      -- Security: dangerous header blocked
      logError(concat_String("Invalid header: ", name))
    }
    BodyTooLarge(size) => {
      -- Response exceeded 5MB limit
      logError(concat_String("Too large: ", size))
    }
  }
}
```

## Security Features

### 1. Header Validation

**Blocked headers** (hop-by-hop, dangerous):
- `Connection`, `Proxy-Connection`, `Keep-Alive`
- `Transfer-Encoding`, `Upgrade`, `Trailer`, `TE`
- `Host` (prevents Host header injection)
- `Accept-Encoding` (prevents decompression issues)
- `Content-Length` (calculated automatically)

```ailang
-- ✅ Allowed
let headers = [
  {name: "Authorization", value: "Bearer token"},
  {name: "Content-Type", value: "application/json"},
  {name: "User-Agent", value: "MyApp/1.0"}
];

-- ❌ Blocked (returns InvalidHeader error)
let badHeaders = [
  {name: "Connection", value: "close"},  -- Blocked!
  {name: "Host", value: "evil.com"}      -- Blocked!
];
```

### 2. Cross-Origin Authorization Stripping

Authorization headers are automatically removed on cross-origin redirects:

```ailang
-- Initial request to api.example.com with Authorization header
-- Redirect to cdn.example.com
-- → Authorization header automatically stripped
```

### 3. Method Whitelist

Only `GET` and `POST` are allowed in v0.3.9:

```ailang
httpRequest("GET", url, headers, body)   -- ✅ Allowed
httpRequest("POST", url, headers, body)  -- ✅ Allowed
httpRequest("PUT", url, headers, body)   -- ❌ Returns InvalidMethod error
```

### 4. Other Security Features

- ✅ HTTPS enforced (http:// requires `--net-allow-http` flag)
- ✅ DNS rebinding prevention
- ✅ Private IP blocking (localhost, 10.x, 192.168.x, 172.16-31.x)
- ✅ Body size limit (5MB default)
- ✅ Domain allowlist support
- ✅ Redirect validation (max 5 redirects)

## Common Patterns

### 1. Retry on Failure

```ailang
func callWithRetry(prompt: string, apiKey: string, retries: int) -> string ! {Net, IO} {
  match chatClaude(prompt, apiKey) {
    Ok(resp) => resp
    Err(Transport(msg)) =>
      if retries > 0 then {
        println("Retrying...");
        callWithRetry(prompt, apiKey, retries - 1)
      } else {
        concat_String("Failed after retries: ", msg)
      }
    Err(other) => show(other)  -- Don't retry non-transient errors
  }
}
```

### 2. Response Parsing

```ailang
-- Note: JSON decoding is planned for v0.4.0
-- For now, parse the JSON string manually or use external tools

func extractContent(jsonBody: string) -> string ! {} {
  -- Manual parsing (simplified)
  -- In practice, you'd use a proper JSON parser
  jsonBody
}
```

### 3. Streaming Responses

Streaming is not yet supported in v0.3.9. The entire response is buffered:

```ailang
-- ❌ Not yet supported
-- func streamChat(prompt: string) -> Stream[string] ! {Net}

-- ✅ Current approach (full response)
func chat(prompt: string) -> string ! {Net} {
  -- Returns complete response after request finishes
}
```

## Troubleshooting

### "InvalidHeader" Error

```
Error: Err(InvalidHeader("connection"))
```

**Cause:** Attempting to use a blocked header
**Solution:** Remove hop-by-hop headers (Connection, Transfer-Encoding, etc.)

### "DisallowedHost" Error

```
Error: Err(DisallowedHost("evil.com"))
```

**Cause:** Domain not in allowlist (if configured)
**Solution:** Add domain to allowlist or remove allowlist restriction

### "Transport" Error with TLS

```
Error: Err(Transport("tls: certificate verify failed"))
```

**Cause:** Invalid TLS certificate
**Solution:** Check the API endpoint URL is correct

### HTTP 401 Unauthorized

```
HTTP error: 401
```

**Cause:** Invalid or missing API key
**Solution:** Verify your API key is correct and properly formatted

## API-Specific Examples

### Google Gemini

```ailang
func chatGemini(prompt: string, apiKey: string) -> string ! {Net, IO} {
  let url = concat_String(
    "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=",
    apiKey
  );
  let headers = [{name: "Content-Type", value: "application/json"}];
  let body = encode(
    jo([
      kv("contents", ja([
        jo([kv("parts", ja([jo([kv("text", js(prompt))])]))])
      ]))
    ])
  );

  match httpRequest("POST", url, headers, body) {
    Ok(resp) => resp.body
    Err(err) => show(err)
  }
}
```

### Generic REST API

```ailang
func callAPI(endpoint: string, payload: Json, apiKey: string) -> string ! {Net, IO} {
  let headers = [
    {name: "Authorization", value: concat_String("Bearer ", apiKey)},
    {name: "Content-Type", value: "application/json"}
  ];

  match httpRequest("POST", endpoint, headers, encode(payload)) {
    Ok(resp) => resp.body
    Err(err) => show(err)
  }
}
```

## Next Steps

- **JSON Decoding** (planned v0.4.0): Parse JSON responses into AILANG values
- **Streaming** (planned v0.4.0): Stream API responses for real-time output
- **Environment Variables** (planned): `getEnv("API_KEY")` for safer key management
- **More HTTP Methods** (planned): PUT, DELETE, PATCH support
- **Custom Timeout** (planned): Per-request timeout configuration

## Related Documentation

See the AILANG repository for more examples and documentation:
- `/examples/` - Complete example files
- `/stdlib/` - Standard library implementations
- `/docs/LIMITATIONS.md` - Known limitations and workarounds
