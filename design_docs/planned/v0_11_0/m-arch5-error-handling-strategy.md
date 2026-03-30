# M-ARCH5: Error Handling Strategy

**Status**: Planned
**Target**: v0.6.5
**Priority**: P2 (Medium)
**Estimated**: 10-14 hours
**Dependencies**: M-ARCH1 (AI Provider Base Class)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism |
| A2: Replayability | +1 | Consistent error structure enables replay analysis |
| A3: Effect Legibility | +1 | Errors are explicit and typed |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Single error pattern to verify |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured errors for machine parsing |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Errors compose across packages |
| A11: Structured Failure | +2 | Core improvement to error handling |
| A12: System Boundary | +1 | Clear error propagation across boundaries |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Structured errors improve machine analysis

## Problem Statement

Error handling is inconsistent across packages, making debugging difficult and error propagation unpredictable.

**Current State - 3 Different Patterns:**

1. **Pattern 1: Wrapped Provider Error** (ai/anthropic)
```go
return nil, ai.NewProviderError("anthropic", 0, "failed to marshal request", err)
```

2. **Pattern 2: OTEL + Bare Error** (ai/openai)
```go
span.RecordError(err)
span.SetStatus(codes.Error, err.Error())
return nil, err  // Bare error, no context
```

3. **Pattern 3: Success=false in Result** (coordinator providers)
```go
if err != nil {
    result.Success = false
    result.Error = fmt.Sprintf("Claude Code execution error: %v", err)
    return result, nil  // Returns nil error!
}
```

**Inconsistent Error Types:**
- `internal/ai/anthropic/client.go:107` - `errorResponse` struct
- `internal/ai/openai/client.go:49` - Different `errorResponse` struct
- `internal/ai/gemini/client.go:56` - Another `errorResponse` struct

**Impact:**
- Callers must handle 3 different error patterns
- OTEL spans sometimes have errors, sometimes don't
- `result.Error` vs `error` return causes confusion
- Hard to build unified error handling middleware

## Goals

**Primary Goal:** Establish single error handling pattern across `ai/`, `executor/`, and `coordinator/` packages.

**Success Metrics:**
- One error type for provider errors (`ProviderError`)
- One error type for execution errors (`ExecutionError`)
- Consistent OTEL error recording
- All errors wrapped with context
- No `return result, nil` when result.Success is false

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Three distinct error types (ProviderError, ExecutionError, CoordinatorError) instead of one generic | Callers use type assertions to distinguish error sources; adding/removing a type changes all switch sites | human | design | high |
| Eliminate `return result, nil` pattern when `result.Success == false` | Every coordinator caller that checks `err == nil` before inspecting `result.Success` must be updated | human | design | high |
| Centralized OTEL `RecordSpanError()` helper | All providers must switch to single recording pattern; provider-specific span attributes depend on error type | human | design | med |
| Fluent builder API (`WithStatusCode().WithRequestID()`) for error construction | Locks the error enrichment pattern; alternative is struct literal construction | human | design | low |
| Error types implement `Unwrap()` for `errors.Is()`/`errors.As()` compatibility | Required for Go idiomatic error handling; determines whether callers can inspect wrapped causes | compiler | compile | med |
| `internal/errors/` package path (not per-package error types) | Centralizes errors but creates a dependency from ai/, executor/, and coordinator/ to one package | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Define exact fields for each error type (ProviderError, ExecutionError, CoordinatorError)
- [ ] Audit all callers of coordinator provider methods that rely on `result.Error` + `nil` error return
- [ ] Decide if `internal/errors/` conflicts with stdlib `errors` package (naming/import alias strategy)
- [ ] Confirm OTEL span attribute schema for error recording (attribute keys and value types)
- [ ] Coordinate with M-ARCH1 (AI Provider Base Class) on shared ProviderError type ownership
- [ ] Determine whether CoordinatorError needs a TaskID field or if that context comes from wrapping

## Solution Design

### Overview

Create `internal/errors/` package with standardized error types. All packages adopt these types and consistent error handling patterns.

### Architecture

```
internal/errors/
├── provider.go      # ProviderError for AI providers (~80 LOC)
├── execution.go     # ExecutionError for executors (~80 LOC)
├── coordinator.go   # CoordinatorError for daemon (~80 LOC)
├── telemetry.go     # OTEL error recording helpers (~60 LOC)
└── errors_test.go   # Tests (~200 LOC)
```

**Error Types:**

1. **ProviderError**: AI provider failures (API errors, auth, rate limits)
2. **ExecutionError**: CLI execution failures (process, timeout, output parsing)
3. **CoordinatorError**: Task coordination failures (routing, approval, worktree)

### The Pattern

```go
// 1. Always wrap errors with context
return nil, errors.NewProviderError("anthropic", "generate", err).
    WithStatusCode(resp.StatusCode).
    WithRequestID(requestID)

// 2. Always record OTEL errors the same way
errors.RecordSpanError(span, err, "operation failed")

// 3. Never return nil error when operation failed
// Before: return result, nil  // with result.Success = false
// After:  return result, errors.NewExecutionError("claude", "execute", err)
```

### Implementation Plan

**Phase 1: Create Error Package** (~4 hours)
- [ ] Create `internal/errors/provider.go` with ProviderError
- [ ] Create `internal/errors/execution.go` with ExecutionError
- [ ] Create `internal/errors/coordinator.go` with CoordinatorError
- [ ] Create `internal/errors/telemetry.go` with OTEL helpers
- [ ] Add comprehensive tests

**Phase 2: Migrate AI Providers** (~3 hours)
- [ ] Refactor `anthropic/client.go` to use ProviderError
- [ ] Refactor `openai/client.go` to use ProviderError
- [ ] Refactor `gemini/client.go` to use ProviderError
- [ ] Refactor `ollama/client.go` to use ProviderError
- [ ] Use `errors.RecordSpanError()` consistently

**Phase 3: Migrate Executors** (~2 hours)
- [ ] Refactor `executor/claude/` to use ExecutionError
- [ ] Refactor `executor/gemini/` to use ExecutionError
- [ ] Remove `return result, nil` pattern

**Phase 4: Migrate Coordinator** (~3 hours)
- [ ] Refactor coordinator providers to use CoordinatorError
- [ ] Remove `result.Error` field in favor of error return
- [ ] Update callers to handle new error pattern

### Files to Modify/Create

**New files:**
- `internal/errors/provider.go` (~80 LOC)
- `internal/errors/execution.go` (~80 LOC)
- `internal/errors/coordinator.go` (~80 LOC)
- `internal/errors/telemetry.go` (~60 LOC)
- `internal/errors/errors_test.go` (~200 LOC)

**Modified files:**
- `internal/ai/anthropic/client.go` - Use ProviderError (~-20 LOC)
- `internal/ai/openai/client.go` - Use ProviderError (~-15 LOC)
- `internal/ai/gemini/client.go` - Use ProviderError (~-20 LOC)
- `internal/ai/ollama/client.go` - Use ProviderError (~-15 LOC)
- `internal/executor/claude/claude.go` - Use ExecutionError (~-10 LOC)
- `internal/executor/gemini/gemini.go` - Use ExecutionError (~-10 LOC)
- `internal/coordinator/provider_claude.go` - Use CoordinatorError (~-15 LOC)
- `internal/coordinator/provider_gemini.go` - Use CoordinatorError (~-15 LOC)

## Examples

### Example 1: ProviderError Type

```go
package errors

// ProviderError represents an error from an AI provider
type ProviderError struct {
    Provider   string // "anthropic", "openai", "gemini", "ollama"
    Operation  string // "generate", "embed", "stream"
    StatusCode int    // HTTP status code (0 if not applicable)
    RequestID  string // Provider request ID for debugging
    Message    string // Human-readable message
    Cause      error  // Underlying error
}

func NewProviderError(provider, operation string, cause error) *ProviderError {
    return &ProviderError{
        Provider:  provider,
        Operation: operation,
        Cause:     cause,
    }
}

func (e *ProviderError) WithStatusCode(code int) *ProviderError {
    e.StatusCode = code
    return e
}

func (e *ProviderError) WithRequestID(id string) *ProviderError {
    e.RequestID = id
    return e
}

func (e *ProviderError) WithMessage(msg string) *ProviderError {
    e.Message = msg
    return e
}

func (e *ProviderError) Error() string {
    if e.Message != "" {
        return fmt.Sprintf("%s %s failed: %s", e.Provider, e.Operation, e.Message)
    }
    if e.Cause != nil {
        return fmt.Sprintf("%s %s failed: %v", e.Provider, e.Operation, e.Cause)
    }
    return fmt.Sprintf("%s %s failed", e.Provider, e.Operation)
}

func (e *ProviderError) Unwrap() error {
    return e.Cause
}
```

### Example 2: OTEL Helper

```go
package errors

import (
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// RecordSpanError consistently records an error on a span
func RecordSpanError(span trace.Span, err error, message string) {
    if err == nil {
        return
    }
    span.RecordError(err)
    span.SetStatus(codes.Error, message)

    // Add error attributes for filtering
    if pe, ok := err.(*ProviderError); ok {
        span.SetAttributes(
            attribute.String("error.provider", pe.Provider),
            attribute.String("error.operation", pe.Operation),
            attribute.Int("error.status_code", pe.StatusCode),
        )
    }
}
```

### Example 3: Consistent Error Handling

**Before (anthropic/client.go):**
```go
if resp.StatusCode != http.StatusOK {
    var errResp errorResponse
    json.NewDecoder(resp.Body).Decode(&errResp)
    return nil, fmt.Errorf("anthropic API error (%d): %s",
        resp.StatusCode, errResp.Error.Message)
}
```

**After:**
```go
if resp.StatusCode != http.StatusOK {
    var errResp apiErrorResponse  // internal struct
    json.NewDecoder(resp.Body).Decode(&errResp)
    return nil, errors.NewProviderError("anthropic", "generate", nil).
        WithStatusCode(resp.StatusCode).
        WithMessage(errResp.Error.Message).
        WithRequestID(resp.Header.Get("X-Request-ID"))
}
```

### Example 4: Coordinator Error Pattern

**Before (provider_claude.go):**
```go
if err != nil {
    result.Success = false
    result.Error = fmt.Sprintf("Claude Code execution error: %v", err)
    return result, nil  // Confusing: returns nil error!
}
```

**After:**
```go
if err != nil {
    return nil, errors.NewCoordinatorError("claude_provider", "execute", err).
        WithTaskID(task.ID)
}
```

## Success Criteria

- [ ] All AI providers use ProviderError
- [ ] All executors use ExecutionError
- [ ] All coordinator providers use CoordinatorError
- [ ] No `return result, nil` when operation failed
- [ ] All OTEL error recording uses RecordSpanError helper
- [ ] Error messages include context (provider, operation, IDs)
- [ ] All existing tests pass
- [ ] Documentation explains error patterns

## Testing Strategy

**Unit tests:**
- Test error type creation and formatting
- Test Unwrap() returns cause
- Test WithX() builder methods
- Test RecordSpanError adds correct attributes

**Integration tests:**
- Provider errors propagate correctly
- Executor errors include execution context
- OTEL spans have error information

**Manual testing:**
- Trigger API errors, verify error messages
- Check OTEL traces for error attributes

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact error message format strings (as long as they include provider, operation, and cause) — [agent may resolve]
- Whether to use a package alias for `internal/errors` to avoid conflict with stdlib `errors` — [agent may resolve]
- Whether `RecordSpanError` should accept variadic attributes or only extract from typed errors — [agent may resolve]
- Error categorization (`IsRetryable()`, `IsFatal()` methods) — will be added once error types stabilize — [human may resolve]
- Error metrics and alerting integration (count by type, provider) — [human may resolve]
- Structured error logging format (JSON vs text) — [agent may resolve]

## Non-Goals

**Not in this feature:**
- Custom error responses to API clients - API layer concern
- Error recovery/retry logic - Handled by callers, not error types

## Timeline

**Day 1** (4 hours):
- Create error package with types and tests

**Day 2** (4 hours):
- Migrate AI providers and executors

**Day 3** (4 hours):
- Migrate coordinator
- Final testing and documentation

**Total: ~12 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking error handling in callers | High | Update callers incrementally, test each change |
| Loss of error information | Medium | Ensure all current context preserved in new types |
| OTEL attribute changes | Low | Document expected attributes |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_16/M-DX2-M4-COMPLETE.md](design_docs/implemented/v0_3_16/M-DX2-M4-COMPLETE.md) (0.41)

**Planned (depends on):**
- [design_docs/planned/v0_6_5/m-arch1-ai-provider-base-class.md](design_docs/planned/v0_6_5/m-arch1-ai-provider-base-class.md) - Extract error types together

**Planned (related):**
- [design_docs/planned/v0_7_0/m-error-propagation.md](design_docs/planned/v0_7_0/m-error-propagation.md) (0.37)

## References

- [Design Axioms](/docs/references/axioms)
- Effective Go: Errors - https://go.dev/doc/effective_go#errors
- Go 1.13 Error Wrapping - https://go.dev/blog/go1.13-errors

## Future Work

- Error categorization (IsRetryable, IsFatal)
- Error metrics (count by type, provider)
- Error alerting integration
- Structured error logging

---

**Document created**: 2026-01-05
**Last updated**: 2026-01-05
