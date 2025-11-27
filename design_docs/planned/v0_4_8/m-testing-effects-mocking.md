# M-TESTING-EFFECTS: Effect Mocking for Inline Tests

**Status**: Planned
**Target**: v0.4.8+
**Priority**: P2 (Low)
**Estimated**: 3-4 days
**Dependencies**: M-TESTING-DEPS (v0.4.8)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Inline mocks vs external test fixtures |
| Preserve Semantic Clarity | + | +1 | Mock specifications explicit in test syntax |
| Increase Determinism | + | +1 | Tests are hermetic, no real IO/FS/Net calls |
| Lower Token Cost | + | +1 | AI can test effectful functions without boilerplate |
| **Net Score** | | **+4** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

Inline tests currently only work for **pure functions**. Functions with effects (`! IO`, `! FS`, `! Net`, `! Clock`) cannot be tested inline because:

1. The test harness runs in a pure context (no effect handlers)
2. There's no syntax for specifying mock effect responses
3. Effect execution requires capability tokens that tests don't have

**Current State:**
- Only pure functions can be tested inline (22 functions in M-TESTING-INLINE-CORE)
- Functions with effects require manual testing via `main` function
- No way to specify deterministic responses for IO/FS/Net/Clock operations
- Existing `MockEffContext` in Go tests not exposed to AILANG test syntax

**Impact:**
- AI code generators cannot test effectful functions inline
- Reduces test coverage to pure functions only
- Real-world programs heavily use effects (file I/O, HTTP, time)

### Error Example

```typescript
-- This FAILS: function has IO effect
func greet(name: string) -> string ! IO
  tests [("Alice", "Hello, Alice!")]  -- Error: unhandled effect IO
{
  _io_print("Greeting: " ++ name);
  "Hello, " ++ name ++ "!"
}
```

## Goals

**Primary Goal:** Enable inline tests for effectful functions by providing a mock specification syntax.

**Success Metrics:**
- IO effects can be mocked in inline tests
- FS effects can be mocked in inline tests
- Clock effects can be mocked in inline tests
- Net effects can be mocked in inline tests
- Unmocked effect calls fail with clear error messages
- Tests remain hermetic (no real side effects)

## Solution Design

### Overview

Add a `mocks` clause to test case syntax that specifies expected effect calls and their return values. The test harness intercepts effect calls and returns mocked values.

### Syntax Design

**Option A: Per-test-case mocks (Preferred)**
```typescript
func readConfig(path: string) -> string ! FS
  tests [
    ("config.json", "{\"key\": \"value\"}")
      mocks { _fs_readFile("config.json") => "{\"key\": \"value\"}" }
  ]
{
  _fs_readFile(path)
}
```

**Option B: Shared mocks block**
```typescript
func readConfig(path: string) -> string ! FS
  tests [
    ("config.json", "{\"key\": \"value\"}"),
    ("other.json", "{}")
  ]
  mocks {
    _fs_readFile("config.json") => "{\"key\": \"value\"}",
    _fs_readFile("other.json") => "{}"
  }
{
  _fs_readFile(path)
}
```

### Architecture

**Components:**
1. **Mock Registry** (`internal/testing/mocks.go`): Maps builtin calls to mock responses
2. **Test Effect Context** (`internal/testing/testctx.go`): Intercepts effect calls during test execution
3. **Parser Extensions**: Parse `mocks` clause in test syntax
4. **Harness Integration**: Inject mock context into evaluator

### Implementation Plan

**Phase 1: Mock Registry** (~8 hours)
- [ ] Create `internal/testing/mocks.go`
- [ ] Implement `MockRegistry` with builtin name → response mapping
- [ ] Implement `MockSpec` with argument matching (exact, wildcard)
- [ ] Add unit tests for registry matching logic

**Phase 2: Test Effect Context** (~4 hours)
- [ ] Create `TestEffectContext` implementing `effects.EffectContext`
- [ ] Route effect calls through mock registry
- [ ] Return error for unmocked effect calls (fail loudly)
- [ ] Integration tests with mock context

**Phase 3: Parser Extensions** (~8 hours)
- [ ] Add `MOCKS` keyword to lexer
- [ ] Parse `mocks { ... }` block in test cases
- [ ] Create `ast.MockSpec` node type
- [ ] Parser tests for mock syntax

**Phase 4: Harness Integration** (~8 hours)
- [ ] Modify `TestExecutor` to build mock registry from test case
- [ ] Pass `TestEffectContext` to evaluator
- [ ] End-to-end tests with IO, FS, Clock, Net mocks
- [ ] Documentation and examples

### Files to Modify/Create

**New files:**
- `internal/testing/mocks.go` - Mock registry (~150 LOC)
- `internal/testing/mocks_test.go` - Unit tests (~200 LOC)
- `internal/testing/testctx.go` - Test effect context (~100 LOC)

**Modified files:**
- `internal/lexer/token.go` - Add MOCKS keyword (~5 LOC)
- `internal/parser/parser.go` - Parse mocks clause (~100 LOC)
- `internal/ast/ast.go` - Add MockSpec node (~30 LOC)
- `internal/testing/executor.go` - Integrate mock context (~50 LOC)
- `internal/testing/harness.go` - Pass mocks to harness (~30 LOC)

## Examples

### Example 1: IO Mocking

**Before:**
```typescript
func prompt(message: string) -> string ! IO
  -- Cannot test: requires IO effect
{
  _io_print(message);
  _io_readLine()
}
```

**After:**
```typescript
func prompt(message: string) -> string ! IO
  tests [
    ("Name: ", "Alice")
      mocks {
        _io_print("Name: ") => (),
        _io_readLine() => "Alice"
      }
  ]
{
  _io_print(message);
  _io_readLine()
}
```

### Example 2: FS Mocking

```typescript
func loadJson(path: string) -> string ! FS
  tests [
    ("data.json", "{\"key\": \"value\"}")
      mocks { _fs_readFile("data.json") => "{\"key\": \"value\"}" }
  ]
{
  _fs_readFile(path)
}
```

### Example 3: Clock Mocking

```typescript
func timestamp() -> int ! Clock
  tests [
    ((), 1699900000)
      mocks { _clock_now() => 1699900000 }
  ]
{
  _clock_now()
}
```

### Example 4: Error Simulation

```typescript
func safeRead(path: string) -> string ! FS
  tests [
    ("missing.txt", "default")
      mocks { _fs_readFile("missing.txt") => error "File not found" }
  ]
{
  -- Would need error handling syntax
  _fs_readFile(path)
}
```

### Example 5: Wildcard Matching

```typescript
func logger(msg: string) -> () ! IO
  tests [
    ("hello", ())
      mocks { _io_print(*) => () }  -- Match any argument
  ]
{
  _io_print("[LOG] " ++ msg)
}
```

## Success Criteria

- [ ] IO effects can be mocked (`_io_print`, `_io_readLine`)
- [ ] FS effects can be mocked (`_fs_readFile`, `_fs_writeFile`)
- [ ] Clock effects can be mocked (`_clock_now`)
- [ ] Net effects can be mocked (`_net_httpGet`)
- [ ] Unmocked effect calls fail with clear error messages
- [ ] Wildcard argument matching works
- [ ] Error simulation works (`=> error "message"`)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Mock registry argument matching (exact, wildcard)
- Mock registry response lookup
- TestEffectContext routing

**Integration tests:**
- IO mock end-to-end
- FS mock end-to-end
- Clock mock end-to-end
- Net mock end-to-end
- Unmocked call error handling

**Manual testing:**
- Create example files with mocked effects
- Verify `ailang test` runs successfully

## Non-Goals

**Not in this feature:**
- State-dependent mocks - Mocks that change based on previous calls (future)
- Partial mocking - Some effects mocked, others real (complex, risky)
- Mock generators - Procedural mock value generation (future)
- Property-based mocking - Integration with property-based testing (see M-TESTING-PROPERTY)
- Call verification - Assert mock was called N times (future enhancement)

## Timeline

**Day 1** (8 hours):
- Phase 1: Mock Registry

**Day 2** (8 hours):
- Phase 2: Test Effect Context
- Phase 3: Parser Extensions (start)

**Day 3** (8 hours):
- Phase 3: Parser Extensions (complete)
- Phase 4: Harness Integration (start)

**Day 4** (8 hours):
- Phase 4: Harness Integration (complete)
- Documentation and examples

**Total: ~32 hours across 4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Syntax complexity | Medium | Start with simple mock block, add features incrementally |
| Type safety of mocks | High | Leverage existing type checker for mock argument types |
| Performance overhead | Low | Mock lookup is O(n) per call; use hashmap if needed |
| Eval integration | Medium | Use existing MockEffContext pattern from Go tests |

## References

- [M-TESTING-INLINE-CORE design doc](../v0_4_7/m-testing-inline-core-evaluation.md)
- [M-TESTING-DEPS design doc](m-testing-deps-cross-function-dependencies.md)
- [MockEffContext pattern](../../../internal/effects/testctx/) - Existing Go test infrastructure
- [Effect system](../../../internal/effects/effects.go)

## Future Work

- **Call verification**: Assert that mocks were called expected number of times
- **Sequence mocks**: Return different values on successive calls
- **State-dependent mocks**: Mock responses based on accumulated state
- **Mock generators**: Generate mock values procedurally

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
