# M-STREAM-DX: Streaming Demo DX Improvements

**Status**: Planned
**Target**: v0.8.1
**Priority**: P1 (Medium) - Unblocks 6 streaming demos, fixes 1 open type checker bug
**Estimated**: 2-3 days (~16 hours)
**Dependencies**: M-STREAM-PHASE2-DX (typed ADTs, shipped), M-DX1 (builtin system)

## Origin

Feedback from building 6 streaming demos against `std/stream` v0.8.0.
Full feedback document at `/tmp/skill-optimization-report.html`.

## Triage Summary

Of 15 items reported, **10 are already resolved or invalid**:

| Item | Status | Resolution |
|------|--------|------------|
| Bug 1: Result type leak Ok→Err | FIXED | iface/builder.go TypeVar case (v0.8.0) |
| Bug 3: Num[string] leak | FIXED | Same root cause as bug 1 |
| Bug 4: Record type leak across fields | FIXED | Same root cause |
| Bug 5: withStream/withSSE crash | FIXED | Same root cause — constructor schemes had corrupted type variables |
| `take` in std/list | EXISTS | Line 107 of std/list.ail |
| `drop` in std/list | EXISTS | Line 116 of std/list.ail |
| `charAt` in std/string | NOT NEEDED | `substring(s, i, 1)` works; adding charAt for one demo is over-engineering |
| `intToString` alias | REJECTED | AILANG convention is short names (`intToStr`). Aliases create confusion. Fix via docs. |
| Concurrent streams | REJECTED | Violates A1 (Determinism) and A6 (Safe Concurrency). Belongs in future task graph system. |
| Stream budget subdivision | REJECTED | Over-engineering. Current `Stream @limit=N` is sufficient. |
| Typed config records | REJECTED | JSON string config is intentional — allows Go runtime to parse arbitrary config without AILANG→Go struct plumbing. |

**5 items remain actionable** — this doc covers those.

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change — all additions are pure functions or existing effect extensions |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | `transmitBinary` uses existing `Stream` effect — binary I/O becomes explicit |
| A4: Explicit Authority | 0 | No new capabilities required |
| A5: Bounded Verification | +1 | Bug 2 fix improves local type checking for ADT-wrapped records |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `zipWith`, `bytes.slice` reduce AI token waste (agents currently reimplement these) |
| A8: Minimal Syntax | +1 | No new syntax — stdlib functions only |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | `zipWith` composes map+zip; `bytes.slice` composes with existing bytes API |
| A11: Structured Failure | 0 | No change to error types |
| A12: System Boundary | 0 | `transmitBinary` extends existing boundary crossing |

**Net Score: +5** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): All changes improve machine analysis / reduce token waste

## Problem Statement

After building 6 streaming demos, several gaps prevent clean AILANG code:

**Current State:**
- `map(\t. t.name, items)` fails when items are ADT-wrapped records — forces verbose pattern match workaround
- No `zipWith` — agents must `map` over `zip` result and destructure tuples (3x more tokens)
- No `bytes.slice` — PCM audio chunking uses `substring` on string-typed audio data (semantically wrong)
- `transmit` only accepts `string` — binary data must be base64-encoded (3x bandwidth overhead)
- AI agents consistently write `intToString` instead of `intToStr` (100% of first attempts fail)

**Impact:**
- 6 streaming demos use unnecessary workarounds
- AI agents waste turns on naming mismatches
- Binary streaming requires base64 encoding (bandwidth tripled)

## Goals

**Primary Goal:** Remove the most impactful friction points for streaming demo development.

**Success Metrics:**
1. `zipWith(\col val. col ++ ": " ++ val, schema, row)` compiles and runs
2. `bytes.slice(audioData, offset, chunkSize)` compiles and runs
3. `transmitBinary(conn, pcmAudio)` sends raw bytes over WebSocket
4. Bug 2 workaround (pattern match in lambda) no longer needed
5. Teaching prompt documents `show` as primary int/float→string conversion

## Solution Design

### Phase 1: stdlib Pure Functions (~2 hours)

#### 1a. Add `zipWith` to std/list

```ailang
-- Combine two lists element-wise with a function
-- zipWith(\a b. a + b, [1,2,3], [10,20,30]) => [11, 22, 33]
export let zipWith = \f xs ys.
  map(\pair. match pair { (a, b) => f(a, b) }, zip(xs, ys))
```

**Why not a recursive implementation?** Reusing `zip` + `map` is simpler, composes with existing tested code, and avoids duplicating list traversal logic. Performance difference is negligible for AILANG's use cases.

**Files:** `std/list.ail` (+3 LOC)

#### 1b. Add `bytes.slice` builtin + stdlib wrapper

**Builtin** (`_bytes_slice`):
```go
// _bytes_slice(b: bytes, start: int, length: int) -> Option[bytes]
// Returns None if start or length is out of bounds (no panics)
```

**Stdlib wrapper** (`std/bytes.ail`):
```ailang
export let slice = \b start len. _bytes_slice(b, start, len)
```

Returns `Option[bytes]` (not `bytes`) because out-of-bounds is a data error, not a crash — consistent with `nth` returning `Option[a]`.

**Files:**
- `internal/builtins/bytes.go` (+25 LOC) — register `_bytes_slice`
- `std/bytes.ail` (+3 LOC) — export wrapper

### Phase 2: Stream Binary Transmit (~4 hours)

#### 2a. Add `transmitBinary` to std/stream

```ailang
-- Send raw bytes over a stream connection
-- transmitBinary(conn, pcmAudio) ! {Stream}
export let transmitBinary = \conn data. _stream_transmit_binary(conn, data)
```

**Runtime implementation** (`internal/effects/stream.go`):
- New `StreamSendBinary` effect handler
- Accepts `bytes` value, sends as WebSocket binary frame
- Uses existing `gorilla/websocket` `WriteMessage(websocket.BinaryMessage, data)`
- Budget accounting same as `transmit` (counts as 1 message)

**Files:**
- `internal/builtins/stream.go` (+15 LOC) — register `_stream_transmit_binary`
- `internal/effects/stream.go` (+20 LOC) — `StreamSendBinary` handler
- `std/stream.ail` (+3 LOC) — export wrapper

**Note:** This is explicitly scoped in M-STREAM-BIDI (success metric 3: "Binary data can be sent/received via bytes type") so we are implementing a planned deliverable, not adding scope.

### Phase 3: Bug 2 Fix — ADT Record Field Access (~8 hours)

**Bug:** `map(\t. t.name, items)` where `type Item = Item({name: string})`.

**Root cause:** `inferRecordAccess` in `typechecker_data.go:59-95` creates a `TRecordOpen` constraint. The unifier has no rule for `TCon ~ TRecordOpen` because ADT constructors are nominally typed, not structurally typed.

**This is the hardest item and requires careful design.** Two approaches:

**Option A: Auto-unwrap single-constructor ADTs in field access (RECOMMENDED)**

When the type checker sees `t.name` and `t` has type `Item` where `Item = Item({name: string})`:
1. Look up `Item` in the ADT registry
2. If it has exactly ONE constructor whose ONE field is a record type → auto-unwrap
3. Create the record constraint against the unwrapped record type

**Pros:** Natural semantics — single-constructor ADTs are essentially newtypes
**Cons:** Only works for single-constructor, single-field ADTs
**Axiom check:** A8 neutral (no new syntax), A7 +1 (reduces AI verbosity)

**Option B: No fix — document pattern match workaround**

Leave as-is. `match t { Item({name, ...}) => name }` works today.

**Pros:** No type checker changes
**Cons:** Verbose, especially in lambdas passed to `map`/`filter`

**Recommendation:** Option A for single-constructor single-record-field ADTs only. This is the "newtype" pattern (Haskell precedent) and auto-unwrapping is well-understood.

**Files:**
- `internal/types/typechecker_data.go` (~+30 LOC) — check for single-constructor record ADT
- `internal/types/unification_records.go` (~+20 LOC) — add `TCon ~ TRecordOpen` rule for qualifying ADTs
- Test files (~+60 LOC)

**Risk:** Medium. Type checker changes can have cascading effects. Needs thorough testing.

### Phase 4: Documentation (~2 hours)

#### 4a. Teaching prompt update

- Document `show` as the primary way to convert any type to string
- Note `intToStr`/`floatToStr` for specific formatting needs
- Add `SSEData` pattern matching example
- Document that `withStream`/`withSSE` now work (were broken by bug 1/5, now fixed)
- Add `zipWith`, `bytes.slice`, `transmitBinary` to function reference

#### 4b. std/stream docs

Add SSEData usage example to module header:
```ailang
-- SSE event handling:
-- onEvent(conn, \event. match event {
--   SSEData(eventType, data) => handleData(eventType, data)
--   StreamError(err) => match err { ... }
--   _ => true
-- })
```

**Files:**
- `prompts/` — new prompt version (+SSEData, +show recommendation, +new functions)
- `std/stream.ail` — expanded header comments (+10 LOC)

## Non-Goals

**Not in this feature:**

- **Concurrent stream connections** — Violates A1 (Determinism) and A6 (Safe Concurrency). AILANG deliberately removed CSP. Concurrent I/O belongs in future deterministic task graph system (v0.4.0+), not stream stdlib. Sequential connection handling is the correct pattern for a deterministic language.
- **Stream budget subdivision** — Over-engineering. `Stream @limit=N` covers current needs. Per-connection budgets can be revisited when usage patterns emerge from production deployments.
- **Typed config records for std/stream** — The JSON string config is intentional. It allows the Go runtime to parse arbitrary config without needing AILANG record → Go struct conversion at the FFI boundary. The verbosity is a DX annoyance, not a correctness issue.
- **`intToString`/`floatToString` aliases** — AILANG follows short naming convention (`intToStr`, `floatToStr`). Adding aliases creates two ways to do the same thing (violates A8: Minimal Syntax). Fix via documentation instead.
- **`charAt` function** — `substring(s, i, 1)` already works. Adding a single-use function violates the "don't add features for hypothetical needs" principle.
- **Multi-constructor ADT field access** — Bug 2 fix only applies to single-constructor, single-record-field ADTs (newtype pattern). Multi-constructor ADTs require explicit pattern matching — this is by design (exhaustiveness checking).

## Examples

### Example 1: zipWith for column formatting

**Before:**
```ailang
let formatted = map(\pair. match pair {
  (col, val) => col ++ ": " ++ val
}, zip(schema, row))
```

**After:**
```ailang
let formatted = zipWith(\col val. col ++ ": " ++ val, schema, row)
```

### Example 2: Binary audio streaming

**Before (base64 workaround — 3x bandwidth):**
```ailang
let encoded = toBase64(fromString(audioChunk))
transmit(conn, encoded)
```

**After (direct binary):**
```ailang
let chunk = match slice(audioData, offset, chunkSize) {
  Some(data) => data
  None => fromString("")
}
transmitBinary(conn, chunk)
```

### Example 3: ADT record field access (Bug 2 fix)

**Before (verbose pattern match):**
```ailang
let names = map(\t. match t { ToolDecl({name, description, parameters}) => name }, tools)
```

**After (direct field access on newtype):**
```ailang
let names = map(\t. t.name, tools)
```

## Success Criteria

- [x] Bugs 1, 3, 4, 5 confirmed fixed (pre-existing)
- [ ] `zipWith` added to std/list with tests
- [ ] `bytes.slice` builtin + stdlib wrapper with bounds checking
- [ ] `transmitBinary` sends binary WebSocket frames
- [ ] Bug 2 fixed for single-constructor record ADTs
- [ ] Teaching prompt updated (show, SSEData, new functions)
- [ ] All tests passing
- [ ] `make verify-examples` passes
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- `zipWith` — empty lists, unequal lengths, various function types
- `bytes.slice` — valid ranges, out-of-bounds (start, length, both), empty bytes
- `transmitBinary` — mock stream context, binary frame verification
- Bug 2 — `map(\t. t.field, adtList)` for single-constructor newtypes

**Integration tests:**
- End-to-end pipeline test: parse → typecheck → eval for ADT record field access
- Stream binary transmit with mock WebSocket server

**Manual testing:**
- Run streaming demo files with `ailang run --caps Stream,IO`
- Verify SSEData pattern matching in REPL

## Timeline

**Day 1** (~6 hours):
- Phase 1: `zipWith` + `bytes.slice` (2h)
- Phase 2: `transmitBinary` (4h)

**Day 2** (~8 hours):
- Phase 3: Bug 2 investigation and fix (6h)
- Phase 3: Testing (2h)

**Day 3** (~4 hours):
- Phase 4: Documentation and prompt update (2h)
- Integration testing and cleanup (2h)

**Total: ~18 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Bug 2 fix causes type regression | High | Limit to single-constructor single-field ADTs; comprehensive regression tests |
| `bytes.slice` off-by-one errors | Low | Return `Option[bytes]` — bounds checked, no panics |
| `transmitBinary` WebSocket compat | Low | gorilla/websocket already supports binary frames |
| Teaching prompt version churn | Low | Bundle all DX changes in one prompt version |

## Related Documents

- [M-STREAM-PHASE2-DX](design_docs/planned/v0_8_1/m-stream-phase2-dx.md) — Typed ADT exports (shipped)
- [M-STREAM-BIDI](design_docs/planned/v0_8_1/m-stream-bidi-primitives.md) — Original streaming design (binary transmit is success metric 3)
- [M-STDLIB-GAPS](design_docs/planned/v0_7_4/m-stdlib-gaps-sprint-plan.md) — Prior stdlib gap fill (nth, last, join, etc.)
- [M-DX1](design_docs/implemented/v0_3_10/M-DX1_developer_experience.md) — Builtin development workflow

## Future Work

- **Multi-constructor ADT field access** — Would require structural subtyping or row-polymorphic ADTs. Significant type system change, defer to v0.9.0+.
- **`readFileBytes` returning `bytes`** — Currently returns `Result[string, string]`. When changed, demos should switch from `substring` to `bytes.slice` for binary data.
- **Stream config builder** — If JSON config verbosity becomes a recurring complaint, consider a builder pattern: `streamConfig().withHeader("Auth", token).build()`. Deferred until more usage data.
- **Concurrent streams via task graphs** — The correct AILANG approach to `runEventLoops([conn1, conn2])`. Requires deterministic task graph scheduler (v0.4.0+ roadmap). Not a stream library concern.

---

**Document created**: 2026-02-17
**Last updated**: 2026-02-17
