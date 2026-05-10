# M-AI-OPENROUTER-APP-ATTRIBUTION: per-request app attribution overrides for OpenRouter

**Status**: Implemented (v0.18.9)
**Target**: v0.18.9 (one-day patch)
**Priority**: P2 (Medium — enables marketplace visibility + per-pipeline cost tracking)
**Dependencies**: ✅ M-AI-OPENROUTER-PROVIDER (v0.16.0)
**Author**: Kimi-K2.6 + Mark
**Created**: 2026-05-10
**Source**: https://openrouter.ai/docs/app-attribution

## Problem

AILANG's OpenRouter integration was sending **static, hardcoded** app-attribution headers:

```go
const (
    defaultHTTPReferer = "https://ailang.sunholo.com"
    defaultXTitle      = "AILANG"
    defaultCategories  = "cli-agent,cloud-agent"
)
```

This meant:
1. **No per-run customization**: Eval suite, coordinator, REPL, and user pipelines all appeared as the same app in OpenRouter analytics
2. **No marketplace category flexibility**: Always `cli-agent,cloud-agent`, even for creative-writing or gaming use cases
3. **Wrong header name**: Used old `X-Title` instead of new canonical `X-OpenRouter-Title` (breaks future OpenRouter middleware)
4. **No category validation**: Unrecognized categories were silently sent and silently dropped by OpenRouter

## Goals

Allow per-request (per-run) app-attribution overrides via CLI flags, with three-layer precedence:

1. **CLI flags** (`--openrouter-referer`, `--openrouter-title`, `--openrouter-categories`) — per-run override
2. **Env vars** (`OPENROUTER_HTTP_REFERER`, `OPENROUTER_X_TITLE`, `OPENROUTER_CATEGORIES`) — per-shell override
3. **Built-in defaults** — fallback

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Attribution is metadata; does not affect program semantics |
| A2: Replayability | 0 | Attribution is not part of the execution trace |
| A3: Effect Legibility | 0 | No new effects; purely metadata |
| A4: Explicit Authority | +1 | CLI flags make attribution explicit, not ambient |
| A7: Machines First | +1 | Enables AI-driven pipeline cost tracking by category |
| A9: Cost Visibility | +2 | Per-app/per-category analytics in OpenRouter dashboard |
| A12: System Boundary | 0 | HTTP headers are already a declared boundary |

**Net Score: +4** → **Decision: ✅ Proceed**

## Solution Design

### Three-Layer Precedence

```
Per-request (highest)  →  Env vars  →  Built-in defaults (lowest)
   --openrouter-*           OPENROUTER_*      https://ailang.sunholo.com
```

The `Attribution` struct carries per-request overrides:

```go
// internal/ai/provider.go
type Attribution struct {
    HTTPReferer string // e.g. "https://ailang.sunholo.com/eval"
    Title       string // e.g. "AILANG Eval Suite"
    Categories  string // e.g. "cli-agent,programming-app"
}
```

Embedded in `ai.Request`:

```go
type Request struct {
    Attribution *Attribution // When set, overrides env vars and defaults
    // ... existing fields
}
```

### OpenRouter Client Changes

`setAttributionHeaders` now accepts `*ai.Attribution` and applies precedence:

```go
func setAttributionHeaders(r *http.Request, attr *ai.Attribution) {
    referer := defaultHTTPReferer
    title := defaultXTitle
    categories := defaultCategories

    // Layer 2: env vars
    if v := os.Getenv("OPENROUTER_HTTP_REFERER"); v != "" { referer = v }
    if v := os.Getenv("OPENROUTER_X_TITLE"); v != "" { title = v }
    if v := os.Getenv("OPENROUTER_CATEGORIES"); v != "" { categories = v }

    // Layer 1: per-request overrides
    if attr != nil {
        if attr.HTTPReferer != "" { referer = attr.HTTPReferer }
        if attr.Title != "" { title = attr.Title }
        if attr.Categories != "" { categories = attr.Categories }
    }

    r.Header.Set("HTTP-Referer", referer)
    r.Header.Set("X-OpenRouter-Title", title)  // Canonical (v0.16.0+)
    r.Header.Set("X-Title", title)             // Backwards compat
    r.Header.Set("X-OpenRouter-Categories", categories)
}
```

### Handler Option

```go
// internal/ai/handler.go
func WithAttribution(a *Attribution) HandlerOption {
    return func(h *Handler) {
        h.attribution = a
    }
}
```

Used by `cmd/ailang/ai_handlers.go` when setting up AI handlers from CLI flags.

### Eval Harness Integration

```go
// internal/eval_harness/ai_agent.go
type AIAgent struct {
    // ...
    attribution *ai.Attribution
}

func (a *AIAgent) WithAttribution(attr *ai.Attribution) *AIAgent {
    a.attribution = attr
    return a
}
```

The `providerAdapter` threads `attribution` through to every `ai.Request`:

```go
req := &ai.Request{
    Model:        p.model,
    SystemPrompt: systemPrompt,
    UserPrompt:   prompt,
    MaxTokens:    4096,
    Attribution:  p.attribution,
}
```

### CLI Flags

Added to `ailang eval`, `ailang run`, `ailang exec`:

```
--openrouter-referer      "Override HTTP-Referer"
--openrouter-title        "Override X-OpenRouter-Title"
--openrouter-categories   "Override X-OpenRouter-Categories"
```

Usage:

```bash
# Eval suite with custom attribution
ailang eval --benchmark fizzbuzz \
  --openrouter-referer "https://ailang.sunholo.com/eval" \
  --openrouter-title "AILANG Eval Suite" \
  --openrouter-categories "cli-agent,programming-app"

# Run with company attribution
ailang run --caps IO,AI --ai gemini-2.5-flash \
  --openrouter-referer "https://mycompany.com/ailang-agent" \
  --openrouter-title "MyCompany AILANG" \
  --openrouter-categories "cloud-agent"
```

## Files Changed

| File | Lines | Nature |
|------|-------|--------|
| `internal/ai/provider.go` | +13 | New `Attribution` struct + field on `Request` |
| `internal/ai/handler.go` | +13 | `WithAttribution()` handler option |
| `internal/ai/openrouter/client.go` | +24/-14 | 3-layer precedence, canonical header |
| `internal/ai/openrouter/chat.go` | +1/-1 | Pass `req.Attribution` |
| `internal/eval_harness/ai_agent.go` | +14/-2 | `WithAttribution()` fluent API |
| `internal/eval_harness/ai_provider.go` | +16/-3 | Thread through `providerAdapter` |
| `cmd/ailang/eval.go` | +15 | New `--openrouter-*` flags |
| `cmd/ailang/main_run.go` | +27/-4 | Wire attribution into batch loop |
| `cmd/ailang/run_helpers.go` | +4/-2 | Updated `executeBatchItem` signature |
| `cmd/ailang/serve_api.go` | +2/-1 | Fixed call site |
| `cmd/ailang/ai_handlers.go` | +6/-5 | Thread `attr` through setupAIHandler |

**Total: +136/-28 across 11 files**

## Acceptance

- [x] `go vet` passes on changed packages
- [x] `gofmt` clean on all modified files
- [x] Existing OpenRouter tests still pass (no regression)
- [x] Commit `cc726339` on `dev` branch
- [x] CLI flags appear in `ailang eval --help`

## Deferred Work

1. **Category validation** — Check against OpenRouter's allowed list (`cli-agent`, `ide-extension`, `cloud-agent`, `programming-app`, `native-app-builder`, `creative-writing`, `video-gen`, `image-gen`, `writing-assistant`, `general-chat`, `personal-agent`, `roleplay`, `game`). Currently silently dropped by OpenRouter if invalid.

2. **`ailang.toml` config section** — Persistent per-project defaults:
   ```toml
   [openrouter]
   referer = "https://mycompany.com/ailang-pipeline"
   title = "MyCompany AILANG Agent"
   categories = ["cli-agent", "cloud-agent"]
   ```

3. **Same flags on `coordinator` command** — Currently only `eval`, `run`, `exec`, `serve-api` have them. Coordinator tasks should inherit project defaults from `ailang.toml`.

## Related Documentation

- [OpenRouter App Attribution](https://openrouter.ai/docs/app-attribution) — External canonical reference
- [M-AI-OPENROUTER-PROVIDER](implemented/v0_16_0/m-ai-openrouter-provider.md) — Parent milestone for OpenRouter integration
- [M-AI-OPENROUTER-REASONING-FIELD](implemented/v0_18_9/m-ai-openrouter-reasoning-field.md) — Sibling patch (v0.18.9)
