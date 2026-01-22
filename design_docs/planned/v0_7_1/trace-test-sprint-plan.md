# Sprint Plan: M-TRACE-TEST - Trace Testing Framework

## Summary
Create a minimal testing utility that allows AILANG programs to verify traces were correctly generated, enabling automated trace verification for trace-dependent code.

**Duration:** 1 day (4-6 hours active work)
**Dependencies:** None
**Risk Level:** Low
**Design Doc:** [trace-test.md](trace-test.md)

## Current Status Analysis

### Existing Infrastructure
- **Telemetry package exists:** `internal/telemetry/` with OTEL integration
- **Observatory backend exists:** `internal/observatory/` with trace storage/querying
- **Builtin registry exists:** `internal/builtins/spec.go` with registration pattern
- **No stdlib directory:** Must be created

### What's Missing
- `stdlib/` directory structure
- `_trace_check` builtin function
- `stdlib/trace_test.ail` module
- Example/test files

### Velocity
- Recent work: v0.7.0 release, Firebase auth, observatory improvements
- Estimated capacity: 150-200 LOC/day for focused work
- This is a small, self-contained feature with clear scope

## Proposed Milestones

### Milestone 1: Core Infrastructure (~2 hours)
**Goal:** Create stdlib directory and `_trace_check` builtin

**Estimated:** 80 LOC implementation + 40 LOC tests = 120 LOC

**Tasks:**
1. Create `stdlib/` directory in project root
2. Add `_trace_check` builtin to `internal/builtins/spec.go`:
   - Type: `string -> bool`
   - Effect: None (reads trace state, doesn't modify)
   - Implementation: Query recent traces for span name match
3. Add implementation in `internal/effects/trace.go`:
   - Access telemetry/observatory to check trace existence
   - Return true if span with matching name exists in current trace context
4. Register builtin in init()

**Acceptance Criteria:**
- [ ] `stdlib/` directory exists
- [ ] `_trace_check` builtin registered and validated by `ailang doctor builtins`
- [ ] Builtin returns bool based on trace name lookup
- [ ] Unit tests for the builtin pass

**Risks:**
- Trace context propagation may need investigation - Mitigation: Use observatory's recent span query

### Milestone 2: AILANG Module (~1.5 hours)
**Goal:** Create `stdlib/trace_test.ail` with assertion helpers

**Estimated:** 50 LOC implementation + 20 LOC examples = 70 LOC

**Tasks:**
1. Create `stdlib/trace_test.ail`:
   ```ailang
   module stdlib/trace_test

   -- Check if a trace span exists by name
   let assert_trace_exists = \name. _trace_check(name)

   -- Example test function (returns 1 on success)
   let test_simple_trace = \_ . {
     assert_trace_exists("compile.parse");
     assert_trace_exists("compile.typecheck");
     1
   }
   ```
2. Ensure module compiles with `ailang check`
3. Create additional assertion helpers if useful:
   - `assert_trace_not_exists`
   - `trace_count` (returns count of matching spans)

**Acceptance Criteria:**
- [ ] `stdlib/trace_test.ail` compiles without errors
- [ ] Module exports `assert_trace_exists` function
- [ ] Example test function works with current traces
- [ ] `ailang check stdlib/trace_test.ail` passes

**Risks:**
- Module path resolution for stdlib - Mitigation: Follow existing module patterns

### Milestone 3: Examples and Documentation (~1 hour)
**Goal:** Create working examples and update documentation

**Estimated:** 40 LOC examples + docs updates = 60 LOC

**Tasks:**
1. Create `examples/trace_testing.ail` demonstrating usage
2. Update CLAUDE.md to document new stdlib module
3. Update CHANGELOG.md with feature addition
4. Move design doc to `design_docs/implemented/v0_7_1/`

**Acceptance Criteria:**
- [ ] `examples/trace_testing.ail` exists and works
- [ ] Example verified with `make verify-examples`
- [ ] CLAUDE.md updated with trace_test module info
- [ ] CHANGELOG.md entry added

**Risks:**
- None - straightforward documentation work

## Success Metrics
- `ailang doctor builtins` reports no issues with new builtin
- `stdlib/trace_test.ail` compiles successfully
- Example file works: `ailang check examples/trace_testing.ail`
- All existing tests still pass: `make test`
- All linting clean: `make lint`

## Implementation Notes

### Builtin Registration Pattern
Follow existing pattern in `internal/builtins/spec.go`:
```go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "stdlib/trace_test",
    Name:    "_trace_check",
    NumArgs: 1,
    IsPure:  true,  // Reading traces is side-effect free
    Effect:  "",
    Type:    func() types.Type {
        return types.NewTFunc2([]types.Type{types.String}, types.Bool, nil)
    },
    Impl:    traceCheckImpl,
})
```

### Trace Query Implementation
```go
func traceCheckImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    spanName := args[0].(eval.StringValue).Value
    // Query observatory for recent spans matching name
    // Return BoolValue(true) if found, BoolValue(false) otherwise
}
```

## Open Questions
1. **Scope of trace lookup:** Should `_trace_check` look at:
   - Only current trace context?
   - Recent traces within time window?
   - All stored traces?

   **Recommendation:** Start with "current session traces" for simplicity.

2. **Error handling:** Should missing trace infrastructure error or return false?

   **Recommendation:** Return false (fail safe, allow tests to run without traces).

## Axiom Compliance
From design doc - Net Score: **+5** (safe to proceed)
- A2 Replayability: +1 (makes traces verifiable)
- A3 Effect Legibility: +1 (makes trace effects explicit)
- A5 Bounded Verification: +1 (enables local trace checks)
- A7 Machines First: +1 (improves machine testing)
- A10 Composability: +1 (composes with existing tests)

## Timeline Summary
| Phase | Duration | LOC |
|-------|----------|-----|
| M1: Core Infrastructure | 2 hours | 120 |
| M2: AILANG Module | 1.5 hours | 70 |
| M3: Examples & Docs | 1 hour | 60 |
| **Total** | **4-5 hours** | **~250 LOC** |
