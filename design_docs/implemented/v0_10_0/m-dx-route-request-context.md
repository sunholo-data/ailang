# M-DX-ROUTE-CTX: HTTP Request Context for @route Handlers

**Status**: Planned
**Target**: v0.9.5
**Priority**: P1 (High — blocks production webhook processing)
**Estimated**: 2-3 days
**Dependencies**: None
**Milestone ID**: M-DX-ROUTE-CTX
**Created**: 2026-03-25
**Source**: DocParse agent message `db4bcd53`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change — headers are inherently request-scoped |
| A2: Replayability | +1 | Structured request context enables deterministic replay of webhook calls |
| A3: Effect Legibility | +1 | `{IO}` effect already declared; making inputs explicit improves legibility |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | AI agents need header access for webhook verification — currently impossible |
| A8: Minimal Syntax | +1 | Reuses existing record type, no new syntax beyond `@raw` annotation |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Route handlers become composable with external service protocols |
| A11: Structured Failure | +1 | Replaces silent 500 errors with proper argument passing |
| A12: System Boundary | +2 | HTTP is a system boundary — exposing headers is essential for boundary validation |

**Net Score: +9** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No violation — request context is inherently non-deterministic (IO effect already required)
- [x] A12 (System Boundary): This IMPROVES system boundary handling by giving handlers access to boundary data

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"protocol-opaque handler" class of issues**. The serve-api layer currently strips HTTP protocol details before dispatching to AILANG functions, making handlers protocol-agnostic. This is good for simple APIs but breaks when handlers need protocol-level information:

1. **Webhook signature verification** (this issue) — Stripe, GitHub, Slack all send signature headers
2. **Content negotiation** — `Accept` header determines response format
3. **Authentication forwarding** — `Authorization` header for downstream calls
4. **Rate limiting** — `X-RateLimit-*` headers for client awareness
5. **Idempotency** — `Idempotency-Key` header for safe retries

The design should solve the general case, not just Stripe webhooks.

**Related work:**
- `_body`/`_status`/`_headers` response records already exist in `routes.go:206-280` — this is the REQUEST equivalent
- `parseArgs()` in `handler.go:108-125` — the `{"args": [...]}` parsing that fails for external JSON formats

---

## Problem Statement

### The Bug

```ailang
@route("POST", "/billing/webhooks/stripe")
export func handleStripeWebhook(signatureHeader: string, rawBody: string, receivedAt: string) -> Result ! {IO} {
  -- Stripe sends: {"type": "checkout.session.completed", "data": {...}}
  -- serve-api receives non-{"args": [...]} JSON → passes entire body as 1 arg
  -- ERROR: "function expects 3 arguments, got 1"
  verifySignature(signatureHeader, rawBody)
}
```

Two distinct issues:

1. **No header access**: `@route` handlers receive only parsed body arguments. HTTP headers (like `Stripe-Signature`) are inaccessible from AILANG code.

2. **External JSON format**: When external services POST their own JSON (not wrapped in `{"args": [...]}`), serve-api passes the entire body as a single string argument. There is no way for the handler to declare "give me the raw body as-is."

### Current Argument Flow

```
HTTP Request → parseArgs() → AILANG function(arg1, arg2, ...)
                  ↓
        {"args": [a, b]}  →  function(a, b)     ✓ works
        "plain string"    →  function("plain")   ✓ works
        {"type": "evt"}   →  function('{"type":"evt"}')  ← stringified, no headers
```

### Desired Flow

```
HTTP Request → buildRequestContext() → AILANG function(request)
                     ↓
        {
          body: '{"type": "checkout.session.completed", ...}',
          headers: {"Stripe-Signature": "t=...,v1=..."},
          method: "POST",
          path: "/billing/webhooks/stripe"
        }
```

---

## Design Options

### Option A: `@raw` Annotation (Recommended)

Add a `@raw` annotation that changes argument passing to provide a `HttpRequest` record instead of parsed args.

```ailang
@route("POST", "/billing/webhooks/stripe")
@raw
export func handleStripeWebhook(request: HttpRequest) -> Result ! {IO} {
  let sig = get(request.headers, "Stripe-Signature")
  let body = request.body
  verifySignature(sig, body)
}
```

**HttpRequest record type:**
```ailang
-- Provided by serve-api runtime, not stdlib
type HttpRequest = {
  body: string,
  headers: Map[string, string],
  method: string,
  path: string,
  query: Map[string, string]
}
```

**Pros:**
- Explicit opt-in — existing handlers unchanged
- Minimal syntax (reuses annotation pattern)
- Single argument keeps function signatures simple
- No ambiguity about what the function receives

**Cons:**
- New annotation to parse
- Functions annotated `@raw` lose automatic arg parsing and schema generation

### Option B: `@header` Parameter Binding

Allow individual parameters to be bound to headers via annotations.

```ailang
@route("POST", "/billing/webhooks/stripe")
export func handleStripeWebhook(
  @header("Stripe-Signature") sig: string,
  @body rawBody: string,
  @query("ts") timestamp: string
) -> Result ! {IO} {
  verifySignature(sig, rawBody)
}
```

**Pros:**
- Fine-grained control
- Self-documenting parameter sources
- Could generate better OpenAPI schemas

**Cons:**
- Significant parser complexity (parameter-level annotations)
- Violates A8 (Minimal Syntax) — new annotation positions
- Over-engineering for current needs

### Option C: Implicit `request` Parameter

If a `@route` handler has a parameter named `request` of type `HttpRequest`, automatically populate it.

```ailang
@route("POST", "/billing/webhooks/stripe")
export func handleStripeWebhook(request: HttpRequest) -> Result ! {IO} {
  let sig = get(request.headers, "Stripe-Signature")
  verifySignature(sig, request.body)
}
```

**Pros:**
- No new annotations
- Convention-based

**Cons:**
- Magic naming convention — violates A3 (Effect Legibility)
- Ambiguous: does `request` come from the caller or the runtime?
- Harder for AI agents to discover

---

## Proposed Design: Option A (`@raw` Annotation)

### Parser Changes

**File: `internal/parser/parser_decl.go`**

Extend `parseAnnotation()` to recognize `@raw` as a parameterless annotation (like `@route` but with no args).

```go
// parseAnnotation handles @route("METHOD", "/path") and @raw
func (p *Parser) parseAnnotation() *ast.Annotation {
    // ... existing @route parsing ...
    if name == "raw" {
        return &ast.Annotation{Name: "raw", Args: nil}
    }
}
```

### AST Changes

No AST changes needed — `ast.Annotation` already supports parameterless annotations via `Args: nil`.

### Server Changes

**File: `internal/apiserver/routes.go`**

1. Add `IsRaw bool` to `ExportInfo` struct
2. In `extractRouteAnnotations()`, detect `@raw` annotation
3. In the route handler, when `IsRaw` is true:
   - Build an `HttpRequest` record from `*http.Request`
   - Pass it as the single argument instead of parsed args

```go
// routes.go — in the custom route handler
if info.IsRaw {
    reqRecord := buildHttpRequestRecord(r)
    args = []interface{}{reqRecord}
} else {
    args = parseArgs(r) // existing behavior
}

func buildHttpRequestRecord(r *http.Request) map[string]interface{} {
    bodyBytes, _ := io.ReadAll(r.Body)
    headers := make(map[string]interface{})
    for k, v := range r.Header {
        headers[k] = v[0] // first value for each header
    }
    query := make(map[string]interface{})
    for k, v := range r.URL.Query() {
        query[k] = v[0]
    }
    return map[string]interface{}{
        "body":    string(bodyBytes),
        "headers": headers,
        "method":  r.Method,
        "path":    r.URL.Path,
        "query":   query,
    }
}
```

### Evaluation Changes

The `HttpRequest` record is passed as a plain AILANG record value. No special runtime type needed — field access via `request.body`, `request.headers`, etc. works with existing record evaluation.

Header lookup uses `get(request.headers, "Stripe-Signature")` which returns `Option[string]` — consistent with existing Map access patterns.

### Schema Changes

**File: `internal/apiserver/schema/schema.go`**

For `@raw` routes, the OpenAPI request schema should be:
```json
{
  "type": "object",
  "description": "Raw HTTP request — body is passed as-is"
}
```

No `{"args": [...]}` wrapper enforced.

---

## Test Plan

1. **Unit test**: `@raw` annotation parsing in `parser_decl_test.go`
2. **Unit test**: `buildHttpRequestRecord()` constructs correct record shape
3. **Integration test**: `@raw @route("POST", "/webhook")` handler receives headers and raw body
4. **Integration test**: Non-`@raw` routes unchanged (backward compatibility)
5. **Integration test**: Stripe-like webhook flow — POST with custom JSON and signature header
6. **Regression**: `make test` — all existing tests pass
7. **Regression**: `make verify-examples` — existing serve-api examples unchanged
8. **Example file**: `examples/serve_api_webhook.ail` demonstrating webhook handler

---

## Example Usage

```ailang
module webhooks

import std/json (parse, get, getString)
import std/crypto (hmacSha256, constantTimeEqual)

@route("POST", "/billing/webhooks/stripe")
@raw
export func handleStripe(request: HttpRequest) -> Result ! {IO} {
  let sig = match get(request.headers, "Stripe-Signature") {
    Some(s) => s,
    None => return Err("Missing Stripe-Signature header")
  }

  let payload = request.body
  let expected = hmacSha256(webhookSecret, payload)

  match constantTimeEqual(sig, expected) {
    true => {
      let event = parse(payload)
      processEvent(event)
    },
    false => Err("Invalid webhook signature")
  }
}

@route("GET", "/health")
export func health() -> string ! {IO} {
  "ok"
}
```

---

## Key Files

| File | Purpose |
|------|---------|
| `internal/parser/parser_decl.go:100-163` | @route annotation parsing — extend for @raw |
| `internal/ast/ast_decl.go:45-80` | Annotation AST (no changes needed) |
| `internal/apiserver/routes.go:30-106` | Route extraction — add IsRaw detection |
| `internal/apiserver/routes.go:110-232` | Route handler dispatch — add @raw branch |
| `internal/apiserver/handler.go:103-189` | Argument parsing (unchanged for non-@raw) |
| `internal/apiserver/schema/schema.go:287` | Request schema — skip args validation for @raw |
| `internal/apiserver/routes_test.go` | Add @raw route tests |
| `internal/apiserver/server_test.go` | Add webhook integration test |

---

## Migration Path

This is purely additive — no breaking changes. Existing `@route` handlers continue to work exactly as before. The `@raw` annotation is opt-in.

Future work could include:
- `@header("Name")` parameter-level annotations (Option B) if fine-grained binding is needed
- A `HttpResponse` record type for structured response building (complement to existing `_body`/`_status`/`_headers`)
- Middleware-style request/response pipelines
