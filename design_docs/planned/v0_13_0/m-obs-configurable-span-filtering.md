# M-OBS-CONFIGURABLE-SPAN-FILTERING: Configurable Observatory Span Filtering

**Status**: Planned
**Target**: v0.9.2
**Priority**: P1 (Medium — blocks cloud dashboard observability)
**Estimated**: 4 hours
**Dependencies**: None (all code is in `internal/observatory/otlp_receiver.go`)
**Triggered by**: ailang-multivac report — coordinator OTEL spans filtered as noise with no override

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | +1 | More spans retained → richer traces for replay |
| A3: Effect Legibility | 0 | No new side effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification impact |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Config-driven filtering enables programmatic tuning |
| A8: Minimal Syntax | 0 | No new AILANG syntax |
| A9: Cost Visibility | +1 | Controllable storage cost — operators can tune noise vs retention |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | 0 | Errors remain typed |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis (more data retained when needed)

---

## Problem Statement

The Observatory's `shouldFilterSpan()` function in `internal/observatory/otlp_receiver.go` (lines 585–675) uses **hard-coded filtering lists** with no configuration mechanism. Operators cannot adjust what spans are kept or dropped without recompiling the binary.

**Current State:**
- 6 hard-coded filter categories: GCP Trace ops, OTEL SDK ops, health checks, static assets, polling endpoints (11 paths), service-specific ops (6 operations across 2 services)
- Added in commit `bde6a217` (Jan 5, 2026) as a response to 10K+ noise spans from polling
- No env vars, config file, or CLI flags to adjust filtering
- Filtered spans logged as `"observatory: filtered span name=%s (internal/noise)"` with no way to override

**Impact:**
- **Cloud operators** cannot retain coordinator spans like `coordinator.dispatch` and `executor.run` needed for debugging cloud pipelines
- **Control plane endpoints** are blanket-filtered (`/api/controlplane/*`) — cannot selectively keep important ones
- **New services** added to the platform require code changes to filter their noisy spans
- **Debugging sessions** cannot temporarily disable filtering to see all traffic

---

## Goals

**Primary Goal:** Make span filtering configurable via environment variables and optional config, while preserving current defaults as sensible out-of-box behavior.

**Success Metrics:**
1. Operators can override filtering rules without recompiling
2. `AILANG_SPAN_FILTER_ALLOW` env var can force-keep specific span names
3. `AILANG_SPAN_FILTER_DENY` env var can add new deny patterns
4. Current filtering behavior is unchanged when no config is provided (zero-config default)
5. Filter config is logged at startup for debuggability

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Allow takes priority over deny | Determines operator mental model for all filter configurations; reversing priority later would break every existing deployment's filtering behavior | human | design | high |
| Environment variables as sole config mechanism (no config file) | Operators must encode all rules in env vars; adding config file later requires migration path and dual-source resolution | human | design | med |
| Immutable config (read once at startup, no reload) | Operators must restart the binary to change filtering; adding hot-reload later requires concurrency-safe config access | human | design | med |
| Pattern syntax: prefix/exact/suffix only (no regex) | Limits expressiveness but keeps implementation simple; adding regex later is backward-compatible but changes matching semantics for edge cases | human | design | low |
| `service:pattern` scoping syntax | Baked into env var format; changing delimiter or format later breaks existing deployments | human | design | med |
| Default deny list compiled into binary | New AILANG services require code changes to add default deny rules unless operators use env vars | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Allow-before-deny priority is confirmed (not deny-before-allow or longest-match-wins)
- [ ] `service:pattern` delimiter is colon (not `/` or `@`) — must not conflict with span name characters
- [ ] Whether `AILANG_SPAN_FILTER_ALLOW` patterns use the same prefix/exact/suffix syntax as deny patterns
- [ ] Whether `AILANG_SPAN_FILTER_DISABLE=true` also suppresses the startup config log line
- [ ] Default deny list is finalized (current 25+ patterns are the correct baseline)

---

## Solution Design

### Overview

Extract hard-coded filter lists into a `SpanFilterConfig` struct loaded from environment variables at `OTLPReceiver` creation time. The struct is read-only after initialization (no runtime mutation, no config watch). Existing hard-coded rules become defaults that can be overridden.

### Architecture

```
Startup:
  NewOTLPReceiver()
    → loadSpanFilterConfig()
      → Parse AILANG_SPAN_FILTER_ALLOW (comma-separated patterns)
      → Parse AILANG_SPAN_FILTER_DENY (comma-separated patterns)
      → Parse AILANG_SPAN_FILTER_DISABLE (bool: disable all filtering)
      → Merge with hardcoded defaults
    → Store as r.filterConfig (immutable)

Per-span:
  shouldFilterSpan(name, resourceAttrs)
    → Check allow-list FIRST (allow takes priority)
    → Check deny-list (includes defaults + env overrides)
    → Default: keep
```

**Key Design Decision: Allow takes priority over deny.** This means operators can force-keep specific spans even if they match a default deny rule. This is safer for debugging — you can always add spans back.

### Components

1. **`SpanFilterConfig` struct** — Holds compiled filter rules (prefix matches, exact matches, suffix matches)
2. **`loadSpanFilterConfig()`** — Parses env vars at startup, merges with defaults
3. **Refactored `shouldFilterSpan()`** — Uses config instead of hard-coded lists

### Implementation Plan

**Phase 1: Extract Config Struct** (~1.5 hours)
- [ ] Create `SpanFilterConfig` struct with `AllowPatterns`, `DenyPatterns`, `DisableAll` fields
- [ ] Create `DefaultSpanFilterConfig()` that returns current hard-coded rules as a config
- [ ] Create `loadSpanFilterConfig()` that parses env vars and merges with defaults
- [ ] Add `filterConfig` field to `OTLPReceiver`

**Phase 2: Refactor shouldFilterSpan** (~1 hour)
- [ ] Change `shouldFilterSpan` to method on `OTLPReceiver` (access to config)
- [ ] Replace hard-coded checks with config-driven matching
- [ ] Implement allow-before-deny priority logic
- [ ] Log active filter config at startup

**Phase 3: Testing** (~1.5 hours)
- [ ] Unit tests for `loadSpanFilterConfig()` with various env var combinations
- [ ] Unit tests for filter matching (prefix, exact, suffix, service-scoped)
- [ ] Test allow-overrides-deny behavior
- [ ] Test backward compatibility (no env vars → same behavior)
- [ ] Test `AILANG_SPAN_FILTER_DISABLE=true` passes all spans

### Files to Modify

**Modified files:**
- `internal/observatory/otlp_receiver.go` — Extract config, refactor `shouldFilterSpan` (~80 LOC changed)
- `internal/observatory/otlp_receiver_test.go` — Add filter config tests (~100 LOC new)

**No new files needed** — config is small enough to live in `otlp_receiver.go`.

---

## Detailed Design

### SpanFilterConfig

```go
type SpanFilterConfig struct {
    // Allow patterns take priority — matching spans are ALWAYS kept
    AllowPatterns []FilterPattern

    // Deny patterns — matching spans are dropped (unless allow-listed)
    DenyPatterns []FilterPattern

    // DisableAll bypasses all filtering (debug mode)
    DisableAll bool
}

type FilterPattern struct {
    // Match type: "prefix", "exact", "suffix"
    Type string

    // Pattern to match against span name
    Pattern string

    // Optional: only apply to spans from this service (service.name attr)
    Service string
}
```

### Environment Variables

| Variable | Format | Default | Example |
|----------|--------|---------|---------|
| `AILANG_SPAN_FILTER_ALLOW` | Comma-separated patterns | (empty) | `coordinator.dispatch,executor.run,/api/controlplane/exec-hierarchy` |
| `AILANG_SPAN_FILTER_DENY` | Comma-separated patterns | (empty) | `my-service.noisy-op,/api/custom-poll` |
| `AILANG_SPAN_FILTER_DISABLE` | `true`/`false` | `false` | `true` |

**Pattern format:**
- `name` — exact match: `name == pattern`
- `name*` — prefix match: `strings.HasPrefix(name, "name")`
- `*name` — suffix match: `strings.HasSuffix(name, "name")`
- `service:name` — service-scoped: only match when `service.name == service`

### Default Deny List (current behavior)

These remain as compiled defaults when no env vars are set:

```
google.devtools.cloudtrace*    # GCP Trace exporter internals
opentelemetry.*                # OTEL SDK internals
/health                        # Health checks
health.check
/api/health
/assets/*                      # Static assets
*.js
*.css
*.png
*.ico
*.svg
/api/approvals*                # UI polling endpoints
/api/hierarchy*
/api/statistics*
/api/version*
/api/monitor*
/api/telemetry/config*
/api/metrics*
/api/observatory/traces*
/api/observatory/metrics*
/api/inbox*
/api/budget/*
/api/controlplane/*
/api/coordinator/events
ailang-coordinator:messages.list    # Service-scoped
ailang-coordinator:messages.count
ailang-coordinator:inbox.poll
ailang-coordinator:agent.heartbeat
ailang-server:messages.list
ailang-server:messages.count
```

---

## Examples

### Example 1: Keep coordinator dispatch spans (the reported bug)

**Before:**
```bash
# No way to override — coordinator.dispatch is filtered as noise
# Dashboard shows empty control plane for cloud tasks
```

**After:**
```bash
# In Cloud Run env or docker-compose:
AILANG_SPAN_FILTER_ALLOW=coordinator.dispatch,executor.run

# Now these spans pass through even though service-level filtering exists
```

### Example 2: Debug mode — see all spans

```bash
# Temporarily disable all filtering for debugging:
AILANG_SPAN_FILTER_DISABLE=true ailang serve

# All spans stored — useful for diagnosing what's arriving at the OTLP endpoint
```

### Example 3: Add custom deny patterns for a new service

```bash
# A new service "ailang-builder" generates noisy heartbeat spans:
AILANG_SPAN_FILTER_DENY=ailang-builder:heartbeat,ailang-builder:status.poll
```

---

## Success Criteria

- [ ] `shouldFilterSpan` uses config-driven matching instead of hard-coded lists
- [ ] `AILANG_SPAN_FILTER_ALLOW=coordinator.dispatch` causes those spans to be stored
- [ ] `AILANG_SPAN_FILTER_DISABLE=true` stores all spans
- [ ] No env vars → identical behavior to current hard-coded filtering
- [ ] Filter config logged at startup (pattern count, allow count, deny count)
- [ ] All existing tests pass (`make test`)
- [ ] New unit tests cover allow/deny/disable combinations

---

## Testing Strategy

**Unit tests:**
- `TestDefaultSpanFilterConfig` — verify defaults match current behavior
- `TestSpanFilterAllow` — allow-listed patterns bypass deny
- `TestSpanFilterDeny` — custom deny patterns block spans
- `TestSpanFilterDisable` — disable mode passes everything
- `TestSpanFilterServiceScoped` — `service:pattern` syntax works
- `TestSpanFilterBackwardCompat` — no config → same filtering as today

**Integration tests:**
- Deploy with `AILANG_SPAN_FILTER_ALLOW=coordinator.dispatch` → verify spans appear in dashboard

**Manual testing:**
- Cloud Run: set env var → check dashboard shows coordinator spans

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Config file format (`~/.ailang/span-filters.yaml`) — will be added if env var lists become unwieldy; format TBD — [human may resolve in future version]
- Observatory dashboard UI for filter management — planned for future but mechanism (settings page vs inline toggle) is open — [human may resolve]
- Whether `FilterPattern` uses compiled regex internally or plain string comparison — [agent may resolve]
- Log format for startup filter config summary (structured JSON vs human-readable) — [agent may resolve]
- Whether to expose filter metrics (filtered vs stored counts per pattern) — useful for tuning but adds overhead — [agent may resolve]

## Non-Goals

**Not in this feature:**
- **Runtime config reload** — would require file watching or API endpoint; out of scope
- **Regex patterns** — prefix/exact/suffix covers all current use cases without regex complexity
- **Per-span sampling rates** — this is allow/deny filtering, not probabilistic sampling

---

## Timeline

**Single session** (~4 hours):
- Phase 1: Extract config struct (1.5h)
- Phase 2: Refactor shouldFilterSpan (1h)
- Phase 3: Testing (1.5h)

**Total: ~4 hours in one session**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Misconfigured allow pattern floods storage | Medium | Log warning when >1000 spans/minute stored; document defaults |
| Pattern format confusion | Low | Simple format (exact/prefix/suffix); document with examples |
| Breaking existing behavior | High | Default config exactly reproduces current hard-coded rules; regression tests |

---

## Related Documents

<!-- Auto-populated by Ollama neural search on "obs configurable span filtering" -->

**Implemented (informed design):**
- [design_docs/implemented/v0_9_0/m-perf-observatory-tiered-loading.md](design_docs/implemented/v0_9_0/m-perf-observatory-tiered-loading.md) — Tiered span loading (related perf concern)
- [design_docs/implemented/v0_7_0/observatory-architecture.md](design_docs/implemented/v0_7_0/observatory-architecture.md) — Observatory entity model and backend interface
- [design_docs/implemented/v0_7_0/m-control-plane-v4-integration.md](design_docs/implemented/v0_7_0/m-control-plane-v4-integration.md) — Control plane endpoints (affected by filtering)

**Planned (check for overlap):**
- [design_docs/planned/v0_10_0/m-task-graph-spans-unification.md](design_docs/planned/v0_10_0/m-task-graph-spans-unification.md) — Task-graph span unification (may need filtering adjustments)
- [design_docs/planned/v0_10_0/m-cloud-observatory.md](design_docs/planned/v0_10_0/m-cloud-observatory.md) — Firestore backend (Tier 1 503 fix, complementary scope)

---

## Future Work

- **Config file support**: If env var lists get unwieldy, add `~/.ailang/span-filters.yaml`
- **Dashboard UI**: Add filter management page in Observatory settings
- **Sampling rates**: Per-pattern probabilistic sampling (e.g., keep 10% of polling spans)
- **Filter metrics**: Count filtered vs stored spans per pattern for tuning

---

**Document created**: 2026-03-13
**Last updated**: 2026-03-13
