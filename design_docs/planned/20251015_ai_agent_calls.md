# AI Agent Calls: HTTP Headers + JSON Support

**Status**: 📋 PLANNED
**Target Version**: v0.3.8
**Priority**: P0 (Fast MVP)
**Effort**: ~2 days
**Created**: 2025-10-15

---

## Problem Statement

AILANG currently has all the infrastructure for safe outbound HTTP calls (Net effect, domain allowlist, timeouts), but lacks two critical pieces needed for real-world AI API integration:

1. **HTTP headers** - Cannot send `Authorization`, `Content-Type`, etc.
2. **JSON encode/decode** - No structured payload support

This blocks a compelling use case: calling AI APIs (OpenAI, Anthropic, Google) directly from AILANG code.

---

## Current State (v0.3.7)

### ✅ Already Working

- **Net effect**: Outbound HTTP/HTTPS with timeout + domain allowlist
- **Caps model**: `--caps Net,IO` to enable calls safely
- **Allowlist**: `--net-allow=api.openai.com` per domain
- **Security**: Deny-by-default, local/private IP blocked
- **Timeouts**: Enforced in client
- **REPL stability**: Good enough to demo

### ❌ Missing

- HTTP headers on requests (only httpGet/httpPost with no headers)
- JSON encode/decode helpers
- Environment variable access for secrets (optional)

---

## Proposed Solution

### A. HTTP Headers Support

**New Types**:
```ailang
type HttpHeader = {key: string, value: string}

type HttpResponse = {
  status: int,
  headers: List<HttpHeader>,
  body: string,
  ok: bool           -- true iff 200-299
}
```

**New API**:
```ailang
httpRequest(
  method: string,
  url: string,
  headers: List<HttpHeader>,
  body: string
) -> Result<HttpResponse, string> ! {Net}
```

**Key Design Decisions**:
- **Result type**: Transport errors (DNS, timeout, TLS) vs HTTP errors (4xx/5xx) are distinguishable
- **Response headers**: Exposed for reading auth tokens, rate limits, etc.
- **ok field**: Quick check for 2xx status without pattern matching
- **Header keys**: Use `key` not `name` (conventional terminology)

**Implementation**:
- Add to `internal/effects/net.go`
- Keep `httpGet`/`httpPost` as sugar that calls `httpRequest` with empty headers/body
- Both return same `Result<HttpResponse, string>` type (no shape divergence)
- **LOC**: ~150 (net.go + tests)

**Header Validation (Security)**:
- Treat keys **case-insensitively** (per HTTP spec)
- **Block hop-by-hop headers**: Connection, Proxy-Connection, Keep-Alive, Transfer-Encoding, Upgrade, Trailer
- **Block Host override** (SSRF prevention)
- Coalesce duplicate headers with comma (except Set-Cookie, allow multiples)
- Auto-compute Content-Length (ignore user-provided values)

**Redirect Handling**:
- Follow up to **10 redirects** for GET/HEAD only
- **Don't auto-redirect POST** (requires re-asking caller)
- Enforce **allowlist after redirects** (final host must be allowed)
- Clear error if redirect leads to disallowed domain

**Response Handling**:
- Set `Accept-Encoding: gzip` by default
- Transparently decompress gzip responses
- **Hard cap**: 5 MB response body (configurable via `--net-max-bytes`)
- Clear error if response exceeds limit: "response exceeds 5MB (set --net-max-bytes to increase)"

**Example**:
```ailang
let headers = [
  {key: "Authorization", value: "Bearer sk-..."},
  {key: "Content-Type", value: "application/json"}
];
match httpRequest("POST", url, headers, body) {
  Ok(resp) ->
    if resp.ok then resp.body
    else "Error: " ++ intToString(resp.status)
  Err(msg) -> "Transport error: " ++ msg
}
```

### B. Minimal JSON Support

**New module**: `std/json.ail`

**ADT**:
```ailang
type Json =
  | JNull
  | JBool(bool)
  | JNumber(float)  -- use float for all numbers (not for exact money/integers > 2^53)
  | JString(string)
  | JArray(List<Json>)
  | JObject(List<{key: string, value: Json}>)  -- list-of-pairs preserves order
```

**Functions**:
```ailang
-- Core encode/decode
encode(obj: Json) -> string
decode(str: string) -> Result<Json, string>  -- defer to v0.3.9

-- Convenience constructors
jb(b: bool) -> Json           -- JBool(b)
jn(n: float) -> Json          -- JNumber(n)
js(s: string) -> Json         -- JString(s)
ja(xs: List<Json>) -> Json    -- JArray(xs)
jo(pairs: List<{key: string, value: Json}>) -> Json  -- JObject(pairs)

-- Helper for building objects
kv(k: string, v: Json) -> {key: string, value: Json}
```

**Phase 1 (MVP)**: Encode only (most AI calls only need to build request, read response as raw string initially)

**Key Design Decisions**:
- **JNumber(float)**: Single numeric type for simplicity. Document: "Not for exact money/integers > 2^53—decode will round"
- **JObject as list-of-pairs**: Preserves insertion order, allows duplicate keys (some APIs tolerate them)
- **Convenience constructors**: Make examples readable without verbose constructor calls
- **String escaping**: Handle `\n \r \t \b \f \" \\ \uXXXX` (test with surrogates, control chars)

**Implementation**:
- Add to `stdlib/std/json.ail`
- Builtin Go backing: `internal/builtins/json.go`
- **LOC**: ~220 (ADT + encoder + helpers + tests)

**Example (Readable)**:
```ailang
import std/json (encode, jo, ja, kv, js, jn)

let body = encode(
  jo([
    kv("model", js("gpt-4o-mini")),
    kv("messages", ja([
      jo([
        kv("role", js("user")),
        kv("content", js(prompt))
      ])
    ])),
    kv("temperature", jn(0.7))
  ])
);
```

**Example (Verbose, also valid)**:
```ailang
import std/json (encode, JObject, JString, JArray, JNumber)

let body = encode(
  JObject([
    {key: "model", value: JString("gpt-4o-mini")},
    {key: "messages", value: JArray([
      JObject([
        {key: "role", value: JString("user")},
        {key: "content", value: JString(prompt)}
      ])
    ])},
    {key: "temperature", value: JNumber(0.7)}
  ])
);
```

### C. MVP Example (v0.3.8)

**File**: `examples/ai_call.ail`

```ailang
import std/net (httpRequest)
import std/io (println)
import std/json (encode, jo, ja, kv, js)

export func chatOpenAI(prompt: string, key: string) -> string ! {Net} {
  let url = "https://api.openai.com/v1/chat/completions";
  let headers = [
    {key: "Authorization", value: "Bearer " ++ key},
    {key: "Content-Type", value: "application/json"}
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
    Ok(resp) ->
      if resp.ok then resp.body
      else {
        println("HTTP Error " ++ intToString(resp.status));
        -- Print first 512 chars of error body (prevent console spam)
        println(substring(resp.body, 0, 512));
        ""
      }
    Err(msg) -> {
      println("Transport error: " ++ msg);
      ""
    }
  }
}

export func main() -> () ! {IO, Net} {
  let reply = chatOpenAI("Say hi!", "<API_KEY>");
  if reply != "" then println(reply) else ()
}
```

**CLI**:
```bash
ailang run --caps IO,Net \
  --net-allow=api.openai.com \
  --net-timeout=30s \
  examples/ai_call.ail
```

**Key Improvements**:
- Uses Result matching to handle transport vs HTTP errors separately
- Checks `resp.ok` before printing response
- Truncates error bodies to 512 chars (prevents console spam)
- Uses readable JSON helpers (`jo`, `kv`, `js`)

---

## Optional Enhancements

### 1. Environment Variable Access (P1)

**API**:
```ailang
env(key: string) -> Option<string> ! {Env}
```

**Usage**:
```ailang
let key = env("OPENAI_API_KEY") |> getOrElse("");
let reply = chatOpenAI("Say hi!", key);
```

**Security**: New `Env` capability required (`--caps Env`)

**LOC**: ~60 (new effect + runtime)

### 2. Retry Policy (P2)

**Module**: `std/ai/retry.ail`

```ailang
retryWithBackoff(
  f: () -> a ! {Net},
  maxRetries: int,
  initialDelay: float
) -> Result<a, string> ! {Net}
```

Handles 429/5xx with exponential backoff.

**LOC**: ~80

### 3. Convenience Wrapper (P2)

**Module**: `std/ai/openai.ail`

```ailang
export func chat(
  model: string,
  messages: List<{role: string, content: string}>,
  apiKey: string
) -> string ! {Net} {
  -- wraps httpRequest + JSON encode/decode
}
```

**LOC**: ~100

---

## Security & DX

### Existing Guardrails ✅
- Deny-by-default networking
- Per-domain allowlist (`--net-allow`)
- Enforced timeouts (30s default)
- Local/Private IP blocked
- Caps required (`--caps Net`)
- Same URL parser, SSRF guards for all Net operations

### New Guardrails (v0.3.8)
- **Headers**: No-headers by default; only via `httpRequest` (explicit)
- **Header validation**: Block hop-by-hop headers (Connection, Proxy-Connection, Keep-Alive, Transfer-Encoding, Upgrade, Trailer)
- **Host override blocked**: Prevent SSRF via Host header manipulation
- **Response size cap**: 5 MB default (configurable via `--net-max-bytes`)
- **Redirect validation**: Enforce allowlist after redirects (final host must be allowed)
- **Secret redaction**: Redact `Authorization` and `api-key` header values from logs/errors
- **REPL banner**: Show `λ[Net,IO]>` when Net capability granted, remind to pass `--net-allow`

### DX Improvements
- **Eq/Ord auto-imported**: Examples don't need to import prelude noise
- **Clear error messages**:
  - "response exceeds 5MB (set --net-max-bytes to increase)"
  - "redirect to disallowed host: api.evil.com (add to --net-allow)"
  - "blocked hop-by-hop header: Connection"
  - "transport error: DNS lookup failed for api.example.com"
- **REPL guidance**: "To use Net in REPL: start with --caps Net --net-allow=api.openai.com"

---

## Test Plan

### Unit Tests (Table-Driven)

**HTTP Request Validation**:
- ✅ Reject disallowed domains
- ✅ Reject hop-by-hop headers (Connection, Proxy-Connection, Keep-Alive, Transfer-Encoding, Upgrade, Trailer)
- ✅ Reject Host override
- ✅ Case-insensitive header matching (content-type vs Content-Type)
- ✅ Timeout cancels request (use fake RoundTripper)
- ✅ Redirect to disallowed host fails with clear error
- ✅ Response over 5 MB fails with clear error
- ✅ Gzip decompression works transparently
- ✅ Follow up to 10 redirects (GET/HEAD only)
- ✅ POST redirects fail with clear error

**JSON Encoding**:
- ✅ String escapes: `" \n \r \t \b \f \\ \uXXXX`
- ✅ Surrogate pairs and control characters
- ✅ Arrays and objects nesting (10 levels deep)
- ✅ Deterministic output (golden files)
- ✅ Empty arrays/objects
- ✅ Unicode handling

### Integration Tests (Hermetic)
**Preferred**: Use mock `http.Client` with custom RoundTripper (no external calls)

**Test scenarios**:
- Transport errors (DNS, timeout, TLS)
- HTTP errors (4xx, 5xx) with error bodies
- Successful 2xx responses with headers
- Redirects (allowed and disallowed hosts)
- Oversized responses
- Compressed responses

**One smoke test** (local only, not CI):
- Run against httpbin.org behind `--net-allow=httpbin.org`
- `httpbin.org/post` echoes back request headers/body

### Golden Tests
- JSON encode output matches expected strings (deterministic)
- HTTP request formatting matches curl equivalent (headers, body)

### Example Tests
- `examples/ai_call.ail` runs without errors (with mock API key)
- Error handling paths (bad API key, rate limit, timeout)

---

## Timeline

| Task | Estimate | LOC | Notes |
|------|----------|-----|-------|
| HTTP headers on Net | 0.5 days | ~150 | Result type, header validation, redirects, size cap |
| JSON encode (minimal) | 1.0 days | ~220 | ADT + encode + convenience helpers + tests |
| Example + docs + REPL | 0.5 days | ~50 | ai_call.ail + README + CHANGELOG + prompt updates |
| **Total** | **2.0 days** | **~420** | Fits 2-day budget with hardening |

---

## Implementation Phases

### Phase 1: MVP (v0.3.8) - 2 days
- ✅ `httpRequest` with headers
- ✅ `std/json` with encode only
- ✅ Example: `examples/ai_call.ail` (OpenAI)
- ✅ Tests: unit + smoke
- ✅ Docs: Update README, CHANGELOG

### Phase 2: Polish (v0.3.9) - 1 day
- ✅ JSON decode
- ✅ `env()` for secrets (Env capability)
- ✅ `std/ai/openai.ail` wrapper
- ✅ More examples (Anthropic, Google Gemini)

### Phase 3: Production (v0.4.0)
- ✅ Retry with backoff (`std/ai/retry.ail`)
- ✅ Streaming responses (if needed)
- ✅ Rate limiting helper
- ✅ M-EVAL benchmark: AI-to-AI calls

---

## Success Criteria

### v0.3.8 (MVP)
- [ ] User can call OpenAI API from AILANG with custom headers
- [ ] User can build JSON payloads with `std/json`
- [ ] Security guardrails enforced (allowlist, caps, timeout)
- [ ] Example works end-to-end: `ailang run examples/ai_call.ail`
- [ ] Tests pass in CI

### v0.3.9 (Polish)
- [ ] User can parse JSON responses
- [ ] User can read API keys from environment
- [ ] Convenience wrappers reduce boilerplate

### v0.4.0 (Production)
- [ ] Retry logic handles transient failures
- [ ] Multiple AI vendors supported (OpenAI, Anthropic, Google)
- [ ] M-EVAL benchmark includes AI-to-AI scenarios

---

## Related Work

- **Net effect** (v0.3.0): [design_docs/implemented/v0_3/20250928_net_effect.md](../implemented/v0_3/20250928_net_effect.md)
- **Effects system** (v0.2.0): [design_docs/implemented/v0_2/effects_system.md](../implemented/v0_2/effects_system.md)
- **v0.4.0 Roadmap**: [design_docs/roadmap_v0_4_0.md](../roadmap_v0_4_0.md)

---

## Resolved Design Questions

1. **JSON ADT vs Map type**: Should `JObject` use `Map<string, Json>` or wait for row polymorphism?
   - **Decision**: ✅ Use `List<{key: string, value: Json}>` for order preservation and duplicate key support

2. **Numeric precision**: Use `float` for all JSON numbers, or separate `int` and `float`?
   - **Decision**: ✅ Single `JNumber(float)` for MVP. Document limitations for exact money/large integers.

3. **Response shape**: How to distinguish transport errors from HTTP errors?
   - **Decision**: ✅ Return `Result<HttpResponse, string>`. Transport errors in Err, HTTP errors in Ok with status code.

4. **Header naming**: Use `name` or `key` for header field?
   - **Decision**: ✅ Use `key` (conventional HTTP terminology)

5. **Response headers**: Should response include headers?
   - **Decision**: ✅ Yes, needed for rate limits, auth tokens, pagination

6. **Redirects**: Auto-follow or fail?
   - **Decision**: ✅ Follow up to 10 redirects for GET/HEAD only; fail POST redirects with clear error

7. **Response size limits**: How to prevent runaway costs?
   - **Decision**: ✅ Hard cap at 5 MB (configurable via `--net-max-bytes`)

8. **Env capability scope**: Should `env()` be privileged (like FS paths), or just another capability?
   - **Decision**: ✅ Treat as normal capability; user must pass `--caps Env`

9. **Streaming responses**: Do we need streaming for AI APIs in v0.3.8?
   - **Decision**: ✅ No, defer to v0.4.0 (most APIs support non-streaming)

10. **Convenience helpers**: Verbose constructors or terse helpers?
    - **Decision**: ✅ Provide both: `JString("x")` and `js("x")` for different use cases

## Open Questions (Future Work)

1. **Per-request timeouts**: Should `httpRequest` accept optional timeout parameter?
   - Current: Global 30s timeout via CLI flag
   - Future: Add optional `timeoutMs: int?` parameter in v0.3.9

2. **Structured error types**: Use ADT instead of string for errors?
   - Current: `Result<HttpResponse, string>`
   - Future: `Result<HttpResponse, HttpError>` where `HttpError = {kind: "timeout" | "dns" | "tls" | "status", ...}`

3. **Response streaming**: Callback-based streaming for large responses?
   - Defer to v0.4.0 with new effect op: `httpRequestStream(... , callback: (chunk: string) -> ()) ! {Net}`

---

## References

- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Anthropic API Reference](https://docs.anthropic.com/claude/reference)
- [Google Gemini API Reference](https://ai.google.dev/api)
- JSON RFC: [RFC 8259](https://tools.ietf.org/html/rfc8259)

---

**Implementation Checklist**:

### Runtime (internal/effects/net.go)
- [ ] Add `HttpHeader` and `HttpResponse` types
- [ ] Implement `httpRequest` with Result return type
- [ ] Header validation (hop-by-hop, Host, case-insensitive matching)
- [ ] Redirect handling (10 max, GET/HEAD only, allowlist enforcement)
- [ ] Response size cap (5 MB, configurable via `--net-max-bytes`)
- [ ] Gzip decompression (transparent)
- [ ] Secret redaction (Authorization, api-key in logs/errors)
- [ ] Update `httpGet`/`httpPost` to delegate to `httpRequest`
- [ ] Add unit tests (table-driven, mock RoundTripper)

### Stdlib (stdlib/std/json.ail)
- [ ] Define `Json` ADT (JNull, JBool, JNumber, JString, JArray, JObject)
- [ ] Implement `encode(obj: Json) -> string` (Go backing)
- [ ] Add convenience constructors (`jb`, `jn`, `js`, `ja`, `jo`)
- [ ] Add `kv(k: string, v: Json)` helper
- [ ] String escaping tests (`\n \r \t \b \f \" \\ \uXXXX`, surrogates)
- [ ] Golden tests (deterministic output)
- [ ] Nesting tests (10 levels deep)

### Examples
- [ ] Create `examples/ai_call.ail` (OpenAI chat completion)
- [ ] Error handling (transport errors, HTTP errors, truncated bodies)
- [ ] Verify example runs with mock API key

### Documentation
- [ ] Update README.md (AI agent calls capability)
- [ ] Update CHANGELOG.md (v0.3.8 entry)
- [ ] Update prompts/v0.3.8.md (Net + JSON examples)
- [ ] Add REPL guidance (Net capability usage)

### Testing
- [ ] Run full test suite (`make test`)
- [ ] Smoke test against httpbin.org (local only)
- [ ] Verify CI passes

---

**Generated**: 2025-10-15
**Author**: Claude Code
**Version**: AILANG v0.3.7
**Updated**: 2025-10-15 (incorporated implementation feedback)
