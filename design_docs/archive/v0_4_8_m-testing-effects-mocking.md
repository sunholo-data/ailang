# M-TESTING-EFFECTS v1: IO Transcript Testing

**Status**: Planned
**Target**: v0.4.9+ (after M-TESTING-DEPS and M-TESTING-PROPERTY)
**Priority**: P2 (Low)
**Estimated**: 2 days
**Dependencies**: M-TESTING-DEPS (v0.4.8), M-TESTING-PROPERTY (v0.4.8+)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Inline transcript assertion vs external test setup |
| Preserve Semantic Clarity | + | +1 | `prints [...]` makes expected output explicit |
| Increase Determinism | + | +1 | Tests are hermetic, recording context instead of real stdout |
| Lower Token Cost | 0 | 0 | Adds new syntax, but simple and focused |
| **Net Score** | | **+3** | **Decision: Move forward (narrow scope)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

After M-TESTING-DEPS lands, inline tests will cover all **pure functions** in a module. However, many real programs have functions that perform IO - logging, printing status, writing output.

**Current State (post-DEPS):**
- ✅ Pure functions with dependencies can be tested inline
- ❌ Functions with any effects still cannot be tested inline
- Workaround: test via `main` function or Go tests

**Impact:**
- Common logging/output functions untestable inline
- AI code generators cannot verify output behavior

## Scope: Extremely Narrow v1

> **v1 is embarrassingly constrained by design.**

We do NOT build a full effect mocking system in v1. Instead:

**v1 Scope:**
- IO write-only transcripts (logging-like effects)
- Functions `f : A -> B ! {IO}` that only call `println`/`_io_print`
- Harness records IO transcript instead of writing to stdout
- Test asserts on transcript (lines printed)

**Explicitly NOT in v1:**
- ❌ FS (file system) - temp directories, cleanup, determinism issues
- ❌ Net (network) - external dependencies, flakiness
- ❌ Clock - seed handling, time zones
- ❌ IO read (`_io_readLine`) - requires mock input sequences
- ❌ Arbitrary mock specifications - too complex for v1

## Goals

**Primary Goal:** Enable inline tests for IO-printing functions by asserting on output transcripts.

**Success Metrics:**
- Functions that only call `_io_print`/`println` can be tested
- Test syntax is simple: `prints ["line1", "line2"]`
- Tests are hermetic (no real stdout writes)
- Clear error messages for unsupported effect calls

## Solution Design

### Target UX (Mental Model)

```typescript
func greet(name: string) -> () ! IO
  prints ["Hello, " ++ name]
{
  println("Hello, " ++ name)
}
```

Test harness runs `greet("Alice")` with a recording IO context, compares transcript to `["Hello, Alice"]`.

### Syntax

**New `prints` clause** (instead of `tests`):

```typescript
-- For functions that print and return unit
func greet(name: string) -> () ! IO
  prints ["Hello, " ++ name]
{
  println("Hello, " ++ name)
}

-- Multiple test cases
func greetMany(names: List[string]) -> () ! IO
  prints {
    (["Alice"], ["Hello, Alice"]),
    (["Bob", "Carol"], ["Hello, Bob", "Hello, Carol"])
  }
{
  forEach(names, \n. println("Hello, " ++ n))
}
```

**Key design decisions:**
- `prints` is a new keyword (not overloading `tests`)
- Single argument: expected transcript (list of lines)
- Multiple cases: `prints { (input, transcript), ... }`
- Return value is implicitly `()` and not asserted

### Architecture

**Components:**
1. **Recording IO Context**: Implements IO effect by appending to buffer
2. **Transcript Comparator**: Compares expected vs actual lines
3. **Parser Extension**: `prints` clause parsing
4. **Effect Gate**: Reject functions with non-IO effects or IO reads

**Recording IO Context:**
```go
type RecordingIOContext struct {
    transcript []string
}

func (r *RecordingIOContext) Print(s string) {
    r.transcript = append(r.transcript, s)
}

func (r *RecordingIOContext) ReadLine() (string, error) {
    return "", fmt.Errorf("_io_readLine not supported in prints tests")
}
```

### Implementation Plan

**Phase 1: Recording IO Context** (~4 hours)
- [ ] Create `internal/testing/recordio.go`
- [ ] Implement `RecordingIOContext` with transcript buffer
- [ ] Reject `_io_readLine` calls with clear error
- [ ] Unit tests for recording context

**Phase 2: Parser & AST** (~4 hours)
- [ ] Add `PRINTS` keyword to lexer
- [ ] Parse `prints [...]` and `prints { ... }` clauses
- [ ] Create `ast.PrintsSpec` node type
- [ ] Parser tests

**Phase 3: Harness Integration** (~6 hours)
- [ ] Build harness that uses `RecordingIOContext`
- [ ] Extract transcript after execution
- [ ] Compare expected vs actual transcripts
- [ ] Error reporting for mismatches

**Phase 4: Effect Gating** (~2 hours)
- [ ] Reject functions with FS, Net, Clock effects
- [ ] Reject functions with IO read effects
- [ ] Clear error messages explaining limitations

### Files to Modify/Create

**New files:**
- `internal/testing/recordio.go` - Recording IO context (~80 LOC)
- `internal/testing/recordio_test.go` - Unit tests (~100 LOC)
- `internal/testing/transcript.go` - Transcript comparison (~50 LOC)

**Modified files:**
- `internal/lexer/token.go` - Add PRINTS keyword (~3 LOC)
- `internal/parser/parser.go` - Parse prints clause (~50 LOC)
- `internal/ast/ast.go` - Add PrintsSpec node (~20 LOC)
- `internal/testing/executor.go` - Execute prints tests (~100 LOC)

## Examples

### Example 1: Simple Greeting

```typescript
func greet(name: string) -> () ! IO
  prints ["Hello, " ++ name]
{
  println("Hello, " ++ name)
}
-- greet("Alice") prints ["Hello, Alice"] ✓
```

### Example 2: Multiple Lines

```typescript
func banner(title: string) -> () ! IO
  prints ["========", title, "========"]
{
  println("========");
  println(title);
  println("========")
}
-- banner("Welcome") prints ["========", "Welcome", "========"] ✓
```

### Example 3: Multiple Test Cases

```typescript
func logLevel(level: string, msg: string) -> () ! IO
  prints {
    (("INFO", "started"), ["[INFO] started"]),
    (("ERROR", "failed"), ["[ERROR] failed"])
  }
{
  println("[" ++ level ++ "] " ++ msg)
}
```

### Example 4: What's NOT Supported

```typescript
-- ❌ NOT SUPPORTED: IO read
func prompt(msg: string) -> string ! IO
  -- Cannot use prints: function reads input
{
  println(msg);
  _io_readLine()
}

-- ❌ NOT SUPPORTED: FS effect
func saveConfig(path: string) -> () ! FS
  -- Cannot use prints: function has FS effect
{
  _fs_writeFile(path, "{}")
}

-- ❌ NOT SUPPORTED: Net effect
func fetchData(url: string) -> string ! Net
  -- Cannot use prints: function has Net effect
{
  _net_httpGet(url)
}
```

## Success Criteria

- [ ] `prints` syntax parses correctly
- [ ] Functions calling only `println`/`_io_print` can be tested
- [ ] Transcript comparison works (exact match)
- [ ] Clear error for `_io_readLine` calls
- [ ] Clear error for FS/Net/Clock effects
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- `RecordingIOContext` captures prints correctly
- Transcript comparison: exact match, mismatch detection
- Parser handles `prints` syntax

**Integration tests:**
- Simple `greet` function end-to-end
- Multi-line output
- Multiple test cases
- Error cases: IO read, FS, Net

**Manual testing:**
- Create example files with `prints` tests
- Verify `ailang test` runs successfully

## Non-Goals (Explicitly Deferred)

| Feature | Why Deferred |
|---------|--------------|
| FS mocking | Temp dirs, cleanup, determinism issues |
| Net mocking | External deps, flakiness, auth |
| Clock mocking | Seeds, time zones, complexity |
| IO read mocking | Requires mock input sequences |
| Arbitrary mock syntax | v2 - full `mocks { ... }` system |
| Transcript patterns | v2 - regex/glob matching |

## Timeline

**Day 1** (8 hours):
- Phase 1: Recording IO Context (4 hours)
- Phase 2: Parser & AST (4 hours)

**Day 2** (8 hours):
- Phase 3: Harness Integration (6 hours)
- Phase 4: Effect Gating (2 hours)

**Total: ~16 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Scope creep (adding more effects) | High | Hard gate: only IO print in v1 |
| User confusion (why no FS?) | Medium | Clear docs, error messages explaining v1 scope |
| Transcript comparison fragility | Low | Start with exact match, add patterns in v2 |

## References

- [M-TESTING-DEPS design doc](m-testing-deps-cross-function-dependencies.md)
- [MockEffContext pattern](../../../internal/effects/testctx/) - Existing Go test infrastructure
- [Effect system](../../../internal/effects/effects.go)

## Future Work (v2+)

**M-TESTING-EFFECTS v2:**
- Full `mocks { builtin(args) => result }` syntax
- IO read mocking with input sequences
- FS mocking with virtual filesystem
- Clock mocking with deterministic timestamps

**M-TESTING-EFFECTS v3:**
- Net mocking with recorded responses
- Transcript patterns (regex/glob)
- Call verification (assert mock called N times)

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
