# M-ARCH1: AI Provider Base Class

**Status**: Planned
**Target**: v0.6.5
**Priority**: P1 (Medium-High)
**Estimated**: 8-12 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism properties |
| A2: Replayability | 0 | No change to trace behavior |
| A3: Effect Legibility | 0 | Side effects remain explicit in providers |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Centralizes validation logic for easier local checks |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Reduces code duplication, smaller codebase for AI to maintain |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Common base composes better with new providers |
| A11: Structured Failure | +1 | Unified error types across all providers |
| A12: System Boundary | 0 | No change to boundary handling |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis by reducing duplication

## Problem Statement

The `internal/ai/` package contains 4 provider implementations (Anthropic, OpenAI, Gemini, Ollama) with significant code duplication.

**Current State:**
- ~300 lines of duplicated boilerplate across 4 providers
- Identical constructor pattern: `NewClient(apiKey, ...ClientOption)` (~30 lines × 4)
- Identical functional option pattern: `WithBaseURL()`, `WithHTTPClient()` (~25 lines × 4)
- Identical OTEL instrumentation setup (~15 lines × 4)
- Identical error response structs defined locally in each file (~20 lines × 4)
- Inconsistent error recording patterns

**Locations:**
- `internal/ai/anthropic/client.go:34-56` - ClientOption pattern
- `internal/ai/openai/client.go:31-53` - ClientOption pattern (duplicate)
- `internal/ai/gemini/client.go:38-60` - ClientOption pattern (duplicate)
- `internal/ai/ollama/client.go` - Similar pattern

**Impact:**
- Bug fixes must be applied to 4 files
- New providers require copying 100+ lines of boilerplate
- Inconsistent behavior when implementations drift
- Higher maintenance burden for AI assistants

## Goals

**Primary Goal:** Extract common AI provider boilerplate into shared base class, eliminating ~300 lines of duplication.

**Success Metrics:**
- Reduce duplicated code from ~300 lines to ~50 lines (common imports, etc.)
- New provider implementation requires <50 lines of provider-specific code
- Single location for OTEL instrumentation, error handling, HTTP configuration
- All existing tests continue to pass

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Use Go struct embedding (not interface) for BaseClient | Determines how providers access shared functionality; embedding exposes all base methods on provider type, interface would require delegation | human | design | high |
| Place base package at `internal/ai/base/` | Package path determines import graph and dependency direction for all 4 providers | human | design | high |
| Unified `ProviderError` replaces per-provider error structs | All callers must switch to single error type; affects every error handling path | human | design | med |
| OTEL span helpers in base (not per-provider) | Provider-specific attributes (model, token counts) must still be attachable; base spans set the attribute schema | human | design | med |
| Functional options pattern shared across providers | Locks all providers into same option signature `base.ClientOption`; provider-specific options need extension mechanism | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Confirm struct embedding over interface-based composition for BaseClient
- [ ] Define the exact `ProviderError` fields and decide if status code + request ID are sufficient
- [ ] Decide whether provider-specific `ClientOption` extensions are allowed or all options live in base
- [ ] Agree on OTEL span naming convention (`provider.operation` vs `ai.provider.operation`)
- [ ] Validate that `internal/ai/base/` creates no import cycle with existing provider packages

## Solution Design

### Overview

Create `internal/ai/base/client.go` with a `BaseClient` struct that provides common functionality. Provider implementations embed this base and override only provider-specific logic.

### Architecture

```
internal/ai/
├── base/
│   ├── client.go         # BaseClient struct + common methods
│   ├── options.go        # Shared ClientOption functions
│   ├── errors.go         # Unified error types
│   └── telemetry.go      # OTEL span helpers
├── anthropic/
│   └── client.go         # Embeds BaseClient, implements Generate()
├── openai/
│   └── client.go         # Embeds BaseClient, implements Generate()
├── gemini/
│   └── client.go         # Embeds BaseClient, implements Generate()
└── ollama/
    └── client.go         # Embeds BaseClient, implements Generate()
```

**Components:**

1. **BaseClient**: Common struct with APIKey, BaseURL, HTTPClient, timeout, and span creation
2. **ClientOption**: Unified functional options (WithBaseURL, WithHTTPClient, WithTimeout)
3. **ErrorTypes**: Shared `ProviderError` struct with provider name, status code, message
4. **TelemetryHelpers**: `StartProviderSpan()`, `RecordProviderError()` functions

### Implementation Plan

**Phase 1: Extract Base** (~4 hours)
- [ ] Create `internal/ai/base/client.go` with BaseClient struct
- [ ] Create `internal/ai/base/options.go` with shared ClientOption
- [ ] Create `internal/ai/base/errors.go` with unified error types
- [ ] Add unit tests for base package

**Phase 2: Migrate Providers** (~4 hours)
- [ ] Refactor `anthropic/client.go` to embed BaseClient
- [ ] Refactor `openai/client.go` to embed BaseClient
- [ ] Refactor `gemini/client.go` to embed BaseClient
- [ ] Refactor `ollama/client.go` to embed BaseClient
- [ ] Verify all existing tests pass

**Phase 3: OTEL Consolidation** (~2 hours)
- [ ] Create `internal/ai/base/telemetry.go` with span helpers
- [ ] Replace inline OTEL code in all 4 providers with helpers
- [ ] Add telemetry tests

### Files to Modify/Create

**New files:**
- `internal/ai/base/client.go` - BaseClient struct (~100 LOC)
- `internal/ai/base/options.go` - ClientOption functions (~50 LOC)
- `internal/ai/base/errors.go` - Unified error types (~40 LOC)
- `internal/ai/base/telemetry.go` - OTEL helpers (~60 LOC)
- `internal/ai/base/client_test.go` - Base tests (~100 LOC)

**Modified files:**
- `internal/ai/anthropic/client.go` - Embed BaseClient, remove boilerplate (~-80 LOC)
- `internal/ai/openai/client.go` - Embed BaseClient, remove boilerplate (~-80 LOC)
- `internal/ai/gemini/client.go` - Embed BaseClient, remove boilerplate (~-80 LOC)
- `internal/ai/ollama/client.go` - Embed BaseClient, remove boilerplate (~-60 LOC)

## Examples

### Example 1: Provider Implementation

**Before (anthropic/client.go ~180 lines):**
```go
type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
    return func(c *Client) { c.baseURL = url }
}

func WithHTTPClient(client *http.Client) ClientOption {
    return func(c *Client) { c.httpClient = client }
}

func NewClient(apiKey string, opts ...ClientOption) *Client {
    c := &Client{
        apiKey:     apiKey,
        baseURL:    "https://api.anthropic.com",
        httpClient: http.DefaultClient,
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}

func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
    ctx, span := anthropicTracer.Start(ctx, "anthropic.generate",
        trace.WithAttributes(
            attribute.String("ai.provider", "anthropic"),
            attribute.String("ai.model", req.Model),
        ),
    )
    defer span.End()
    // ... provider-specific logic
}
```

**After (anthropic/client.go ~60 lines):**
```go
type Client struct {
    base.BaseClient
}

func NewClient(apiKey string, opts ...base.ClientOption) *Client {
    return &Client{
        BaseClient: base.NewBaseClient("anthropic", "https://api.anthropic.com", apiKey, opts...),
    }
}

func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
    ctx, span := c.StartSpan(ctx, "generate", req)
    defer span.End()
    // ... provider-specific logic only
}
```

### Example 2: Adding New Provider

**New provider (e.g., mistral/client.go) would require only:**
```go
type Client struct {
    base.BaseClient
}

func NewClient(apiKey string, opts ...base.ClientOption) *Client {
    return &Client{
        BaseClient: base.NewBaseClient("mistral", "https://api.mistral.ai", apiKey, opts...),
    }
}

func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
    ctx, span := c.StartSpan(ctx, "generate", req)
    defer span.End()
    // Only provider-specific request/response handling
}
```

## Success Criteria

- [ ] BaseClient provides constructor, options, HTTP client
- [ ] All 4 providers embed BaseClient and compile
- [ ] All existing provider tests pass without modification
- [ ] Provider implementations reduced by 60%+ lines each
- [ ] OTEL spans work identically to before
- [ ] Error types unified across providers
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Test BaseClient constructor with various options
- Test ClientOption functions (WithBaseURL, WithHTTPClient, WithTimeout)
- Test error type creation and formatting
- Test telemetry helpers create correct spans

**Integration tests:**
- Run existing provider tests (should pass unchanged)
- Test each provider still works with real APIs (manual)

**Manual testing:**
- Run `ailang eval-suite` with refactored providers
- Verify OTEL traces appear correctly in backend

## Deferred Decisions

The following are intentionally left open for the implementer:

- Internal field naming and accessor methods on BaseClient — [agent may resolve]
- Whether `WithTimeout` uses `context.WithTimeout` or `http.Client.Timeout` — [agent may resolve]
- Order of OTEL attributes on provider spans (beyond provider name and model) — [agent may resolve]
- Whether to add a `Validate()` method on BaseClient for pre-flight checks — [agent may resolve]
- Retry logic hooks in base (placeholder interface vs omit entirely for now) — [human may resolve in future iteration]

## Non-Goals

**Not in this feature:**
- Adding new providers - Focus on refactoring existing 4
- Changing provider behavior - Purely structural refactor

## Timeline

**Day 1** (4 hours):
- Create base package with BaseClient, options, errors

**Day 2** (4 hours):
- Migrate all 4 providers to use BaseClient
- Run tests, fix any issues

**Day 3** (2 hours):
- Add OTEL helpers
- Final testing and documentation

**Total: ~10 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking provider behavior | High | Run all existing tests before/after each change |
| OTEL span attributes differ | Medium | Document expected attributes, verify in tests |
| Import cycles | Low | base/ has no dependencies on provider packages |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_10/m-unified-ai-providers.md](design_docs/implemented/v0_5_10/m-unified-ai-providers.md) (0.42)
- [design_docs/implemented/v0_3_0/m_eval_ai_benchmarking.md](design_docs/implemented/v0_3_0/m_eval_ai_benchmarking.md) (0.38)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/ai/anthropic/client.go` - Current implementation reference
- `internal/ai/openai/client.go` - Current implementation reference

## Future Work

- Add provider health checks to BaseClient
- Add automatic retry logic in base (with configurable policy)
- Add request/response logging option

---

**Document created**: 2026-01-05
**Last updated**: 2026-01-05
