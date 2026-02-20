# Sprint Plan: M-STREAM-DX

**Sprint ID**: M-STREAM-DX
**Duration**: 1.5 days (~12 hours)
**Risk Level**: Low (M1-M3b), Medium (M4)
**Design Doc**: [m-stream-dx-improvements.md](m-stream-dx-improvements.md)

---

## Sprint Summary

**Goal**: Remove the top 6 friction points discovered while building streaming demos. Add `zipWith` to std/list, `bytes.slice` builtin, `transmitBinary` + `ssePost` to std/stream, fix ADT-wrapped record field access (Bug 2), and update teaching prompt.

**Key Discoveries**:
1. The `StreamSend` Go handler already supports `Bin(bytes)` binary frames (line 358 of `stream.go`). So `transmitBinary` only needs a builtin + stdlib wrapper — no new Go effect code.
2. `sseConnect` is hardcoded to GET (line 82 of `stream_sse.go`). Claude/OpenAI/Gemini all use POST+SSE for streaming AI responses. Adding `ssePost` unblocks `claude_chat` demo.

**Revised estimate**: 12 hours (down from 18) due to existing binary support.

---

## Current Status

| Item | Status |
|------|--------|
| `std/list.ail` | Has 29 functions including `take`, `drop`, `zip`. Missing: `zipWith` |
| `std/bytes.ail` | Has 5 functions: `fromString`, `toString`, `toBase64`, `fromBase64`, `length`. Missing: `slice` |
| `std/stream.ail` | Has `transmit(conn, string)`. Binary support exists in Go runtime (`Bin` case) but no AILANG-level `transmitBinary` function. `sseConnect` is GET-only — POST+SSE needed for AI APIs |
| Bug 2 | OPEN — `map(\t. t.name, items)` fails for ADT-wrapped records. `inferRecordAccess` creates `TRecordOpen` which can't unify with `TCon` |
| Teaching prompt | v0.8.1 active. Missing: `show` recommendation, SSEData docs, new functions |

---

## Milestones

### M1: Add `zipWith` to std/list (~15 min)

**Tasks**:
1. Add `zipWith` function to `std/list.ail` (reuse existing `zip` + `map`)
2. Add `export` declaration
3. Verify with `ailang check std/list.ail`

**Implementation**:
```ailang
export let zipWith = \f xs ys.
  map(\pair. match pair { (a, b) => f(a, b) }, zip(xs, ys))
```

**Acceptance Criteria**:
- [ ] `zipWith(\a b. a + b, [1,2,3], [10,20,30])` returns `[11, 22, 33]`
- [ ] `zipWith(\a b. a ++ ": " ++ b, ["col"], ["val"])` returns `["col: val"]`
- [ ] `zipWith(\a b. a, [], [])` returns `[]`
- [ ] `make test` passes
- [ ] `make lint` clean

**Files**: `std/list.ail` (+3 LOC)

---

### M2: Add `bytes.slice` builtin (~1.5 hours)

**Tasks**:
1. Register `_bytes_slice` builtin in `internal/builtins/bytes.go`
   - Type: `bytes -> int -> int -> Option[bytes]`
   - Returns `None` for out-of-bounds (start < 0, start+len > length)
   - Follow existing `_bytes_from_base64` pattern for Option return
2. Add `slice` wrapper in `std/bytes.ail`
3. Add unit test in `internal/builtins/bytes_test.go`

**Acceptance Criteria**:
- [ ] `slice(fromString("hello"), 0, 3)` returns `Some(bytes for "hel")`
- [ ] `slice(fromString("hello"), 1, 2)` returns `Some(bytes for "el")`
- [ ] `slice(fromString("hello"), 5, 1)` returns `None` (out of bounds)
- [ ] `slice(fromString("hello"), -1, 2)` returns `None` (negative start)
- [ ] `slice(fromString(""), 0, 0)` returns `Some(empty bytes)`
- [ ] `ailang doctor builtins` validates (total count increments by 1)
- [ ] `make test` passes
- [ ] `make lint` clean

**Files**:
- `internal/builtins/bytes.go` (+60 LOC)
- `std/bytes.ail` (+4 LOC)
- `internal/builtins/bytes_test.go` (+40 LOC)

---

### M3: Add `transmitBinary` to std/stream (~1 hour)

**Key insight**: The Go `StreamSend` handler (line 338-370 of `internal/effects/stream.go`) already handles `Bin(bytes)` variant via pattern match on `adt.CtorName`. So we only need:
1. A new builtin `_stream_transmit_binary` that wraps bytes in `Bin(bytes)` ADT and delegates to existing `StreamSend`
2. A stdlib wrapper in `std/stream.ail`

**Tasks**:
1. Register `_stream_transmit_binary` builtin in `internal/builtins/stream.go`
   - Type: `StreamConn -> bytes -> Result[unit, StreamErrorKind] ! {Stream}`
   - Implementation: wrap bytes in `TaggedValue{CtorName: "Bin"}`, call `StreamSend`
2. Add `transmitBinary` wrapper in `std/stream.ail`
3. Add test verifying binary frame sent correctly

**Acceptance Criteria**:
- [ ] `transmitBinary(conn, fromString("hello"))` sends binary WebSocket frame
- [ ] Budget accounting works (counts as 1 message)
- [ ] SSE connection returns `Err(ProtocolError(...))` (read-only)
- [ ] `ailang doctor builtins` validates
- [ ] `make test` passes

**Files**:
- `internal/builtins/stream.go` (+45 LOC)
- `std/stream.ail` (+5 LOC)
- `internal/effects/stream_test.go` (+30 LOC)

---

### M3b: Add `ssePost` to std/stream (~2 hours)

**Problem**: `sseConnect` (stream_sse.go:82) is hardcoded to HTTP GET. Claude, OpenAI, and Gemini streaming APIs all use POST+SSE: the request body contains the prompt/config, and the response is an SSE stream. The `claude_chat` demo is blocked by this.

**Tasks**:
1. Add `StreamSSEPost` function in `internal/effects/stream_sse.go`
   - Signature: `(url: string, body: string, config: string) -> Result[StreamConn, StreamErrorKind] ! {Stream}`
   - Copy `StreamSSEConnect`, change method to POST, add body parameter from args[1]
   - Set `Content-Type: application/json` by default (configurable via config headers)
2. Register `_stream_sse_post` builtin in `internal/builtins/stream.go`
3. Add `ssePost` wrapper in `std/stream.ail`
4. Register op: `RegisterOp("Stream", "sse_post", StreamSSEPost)`
5. Add test with mock HTTP POST+SSE server

**Acceptance Criteria**:
- [ ] `ssePost(url, body, config)` sends HTTP POST with body, reads SSE response
- [ ] Headers from config are applied (Authorization: Bearer)
- [ ] Content-Type defaults to application/json
- [ ] SSE events dispatched same as sseConnect (SSEData, Closed, etc.)
- [ ] Budget accounting works (counts as 1 connection)
- [ ] `ailang doctor builtins` validates
- [ ] `make test` passes

**Files**:
- `internal/effects/stream_sse.go` (+60 LOC)
- `internal/builtins/stream.go` (+40 LOC)
- `std/stream.ail` (+5 LOC)
- `internal/effects/stream_test.go` (+40 LOC)

---

### M4: Fix Bug 2 — ADT record field access (~5 hours)

**Root cause**: `inferRecordAccess` (typechecker_data.go:59-95) creates `TRecordOpen` constraint. The unifier has no rule for `TCon ~ TRecordOpen` when the TCon is a single-constructor ADT wrapping a record.

**Approach**: Auto-unwrap single-constructor, single-record-field ADTs in `inferRecordAccess`.

**Tasks**:
1. Add ADT lookup helper in `typechecker_data.go`:
   - Given a type that resolves to a `TCon`, check if it's a single-constructor ADT
   - If the single constructor has exactly one field that is a record type, return that record type
   - Otherwise return nil (not a newtype pattern)
2. In `inferRecordAccess`, before creating the `TRecordOpen` constraint:
   - Check if `getType(recordNode)` is a TCon that qualifies as a newtype
   - If so, use the unwrapped record type as the constraint target instead of the raw TCon
3. Add regression tests
4. Test with streaming demo pattern: `map(\t. t.name, tools)`

**Acceptance Criteria**:
- [ ] `type Item = Item({name: string, value: int})` then `map(\t. t.name, [Item({name: "x", value: 1})])` compiles and returns `["x"]`
- [ ] Multi-constructor ADTs still require pattern match (no regression)
- [ ] Multi-field constructors still require pattern match
- [ ] Plain record field access unchanged (no regression)
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks**:
- Type checker changes can cascade — run FULL test suite after each change
- May need to handle the case where the ADT type is not yet resolved (type variable)
- Mitigation: Only unwrap when the type is a concrete `TCon`, not a type variable

**Files**:
- `internal/types/typechecker_data.go` (+40 LOC)
- `internal/types/typechecker_data_test.go` (+80 LOC)

---

### M5: Teaching prompt + docs update (~1.5 hours)

**Tasks**:
1. Create `prompts/v0.8.2.md` (copy from v0.8.1, add changes)
2. Add `show` recommendation for int/float→string conversion
3. Add `SSEData` pattern matching example
4. Document `zipWith`, `bytes.slice`, `transmitBinary`
5. Note `withStream`/`withSSE` now work
6. Update `std/stream.ail` header with SSEData usage example
7. Update `prompts/versions.json` to point to new version

**Acceptance Criteria**:
- [ ] `ailang prompt` displays new version
- [ ] `show` documented as primary string conversion
- [ ] SSEData example in prompt and std/stream.ail header
- [ ] New functions documented

**Files**:
- `prompts/v0.8.2.md` (new, ~same size as v0.8.1)
- `prompts/versions.json` (+1 entry)
- `std/stream.ail` (+10 LOC header comments)

---

## Execution Order

```
M1 (zipWith, 15min) ───┐
                        ├── M4 (Bug 2 fix, 5h) ── M5 (docs, 1.5h)
M2 (bytes.slice, 1.5h) ┤
M3 (transmitBinary, 1h)┤
M3b (ssePost, 2h) ─────┘
```

M1, M2, M3, M3b are independent and can run in parallel. M4 is the critical path. M5 depends on all others completing.

## Success Metrics

- Test coverage: maintained or improved
- `ailang doctor builtins` validates (count +3: `_bytes_slice`, `_stream_transmit_binary`, `_stream_sse_post`)
- `make test` passes
- `make lint` clean
- `make verify-examples` passes
- Teaching prompt updated

## Open Questions

- **M4 scope boundary**: Should we also handle `type Wrapper = Wrapper(InnerADT)` where `InnerADT` is itself an ADT (not a record)? **Recommendation: No** — limit to single-constructor wrapping a record type. Multi-level unwrapping is a future consideration.

---

**Created**: 2026-02-17
