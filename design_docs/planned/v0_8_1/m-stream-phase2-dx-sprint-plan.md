# Sprint Plan: M-STREAM-PHASE2-DX — Stream Typed ADTs & DX Improvements

## Summary
Export typed ADT types from `std/stream` (StreamConn, StreamEvent, StreamErrorKind, StreamMessage, StreamStatus) following the proven `std/option`/`std/net` patterns, update all 7 builtin type signatures to use `T.Con()`/`T.App()`, and rename `eventToADT` constructors to match. This unblocks 6 demos (12 files) currently failing IMP010.

**Duration:** 1 day (4-6 hours implementation)
**Dependencies:** Phase 1 bug fixes (DONE — extractConnID, transmit auto-wrap, show(unit))
**Risk Level:** Low — infrastructure proven working in std/option, std/result, std/net
**Design Doc:** `design_docs/planned/v0_8_1/m-stream-phase2-dx.md`

## Current Status Analysis

### Completed Recently (Phase 1)
- ✅ extractConnID 3-layer unwrap: Ok(StreamConn(id)) auto-unwrap (~40 LOC)
- ✅ StreamSend auto-wrap: plain string → Text(string) (~10 LOC)
- ✅ show(unit) rendering: `*eval.UnitValue` → `"()"` (~3 LOC)
- ✅ std/stream.ail documentation comments updated
- ✅ 7/8 Phase 1 tests passing, integration test added

### Velocity
- Recent stream work: ~300 LOC impl + ~200 LOC tests in 2 days
- M-STREAM-BIDI initial implementation: ~1500 LOC in 3 days
- Average recent: ~250 LOC/day for stream-related work
- This sprint: ~200 LOC impl + ~80 LOC tests = ~280 LOC total

### Remaining from Design Doc
- ⏳ **Phase 2: Export typed ADTs** — ~280 LOC total (THIS SPRINT)
- 📋 Phase 3: foldEvents combinator — ~160 LOC (future)
- 📋 Phase 3: Typed config records — deferred

## Proposed Milestones

### Milestone 1: Add type declarations to std/stream.ail
**Goal:** Define and export StreamConn, StreamEvent, StreamErrorKind, StreamMessage, StreamStatus ADTs
**Estimated:** ~30 LOC (type declarations + import)
**Duration:** 30 minutes

**Tasks:**
1. Add `import std/result (Result, Ok, Err)` to std/stream.ail
2. Add `export type StreamConn = StreamConn(int)`
3. Add `export type StreamEvent = Message(string) | Binary(string) | Opened(string) | Closed(int, string) | StreamError(StreamErrorKind) | Ping(string) | SSEData(string, string)`
4. Add `export type StreamErrorKind = ConnectionFailed(string) | Timeout(string) | BudgetExhausted(string) | ProtocolError(string) | MessageTooLarge(string)`
5. Add `export type StreamMessage = Text(string) | Bin(string)`
6. Add `export type StreamStatus = Connecting | Open | Closing | Closed`
7. Update function signatures to reference new types

**Acceptance Criteria:**
- [ ] `ailang check std/stream.ail` passes
- [ ] Types are importable: `import std/stream (StreamEvent, Message, Closed)`
- [ ] Pattern matching works: `match event { Message(m) => ..., Closed(c, r) => ... }`

**Risks:**
- Type checker may struggle with ADTs in function signatures — Mitigation: Phase 1 signatures used `string`, we can keep runtime auto-wrapping for backward compat

### Milestone 2: Update builtin type signatures in stream.go
**Goal:** Change all 7 make*Type() functions from `T.String()` to proper ADT references using `T.Con()`/`T.App()`
**Estimated:** ~100 LOC changes across 7 functions
**Duration:** 1 hour

**Tasks:**
1. `makeStreamConnectType`: Returns `T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind"))`, effects "Stream"
2. `makeStreamSSEConnectType`: Same return type as connect
3. `makeStreamSendType`: Param `T.Con("StreamConn")` + `T.String()` (msg stays string — auto-wrap in Go), returns `T.App("Result", T.Unit(), T.Con("StreamErrorKind"))`
4. `makeStreamOnEventType`: Param `T.Con("StreamConn")` + handler type `T.Func(T.Con("StreamEvent")).Returns(T.Bool()).Build()`
5. `makeStreamRunEventLoopType`: Param `T.Con("StreamConn")`, returns `T.Unit()`
6. `makeStreamCloseType`: Param `T.Con("StreamConn")`, returns `T.Unit()`
7. `makeStreamGetStatusType`: Param `T.Con("StreamConn")`, returns `T.Con("StreamStatus")`
8. Update comments on each function to reflect new types

**Acceptance Criteria:**
- [ ] All 7 type signatures use `T.Con()`/`T.App()` instead of `T.String()`
- [ ] `go build ./internal/builtins/` compiles
- [ ] `go test ./internal/builtins/ -run TestStream` passes
- [ ] Type checker resolves ADT names correctly when checking examples

**Risks:**
- Type unification between builtin-declared types and .ail-declared types — Mitigation: This is the exact pattern used by std/net's `T.Con("NetError")` which works

### Milestone 3: Rename eventToADT constructors + update tests
**Goal:** Rename `"Error"` → `"StreamError"` in eventToADT to match the AILANG type declaration
**Estimated:** ~20 LOC changes + ~30 LOC test updates
**Duration:** 30 minutes

**Tasks:**
1. In `eventToADT()` (stream.go): Change both `"Error"` → `"StreamError"` (lines 677, 702)
2. In `stream_test.go`: Update `TestEventToADT` assertions from `"Error"` → `"StreamError"`
3. In `stream_test.go`: Update `TestStreamRunEventLoop_IdleTimeout` assertion from `"Error"` → `"StreamError"`

**Acceptance Criteria:**
- [ ] `go test ./internal/effects/ -run TestEventToADT -v` passes
- [ ] `go test ./internal/effects/ -run TestStream -v` passes (all stream tests)
- [ ] Constructor name matches `StreamEvent` type declaration in std/stream.ail

**Risks:**
- None — straightforward rename with clear test coverage

### Milestone 4: Update examples + integration verification
**Goal:** Update example files to use typed imports and verify end-to-end
**Estimated:** ~30 LOC changes per example + verification
**Duration:** 1 hour

**Tasks:**
1. Update `examples/runnable/stream_websocket.ail`:
   - Add `import std/stream (StreamEvent, Message, Closed, StreamError)`
   - Change handler signature from `(event: string)` to `(event: StreamEvent)`
   - Add pattern match on event types (if desired for demo)
2. Update `examples/runnable/stream_sse.ail`:
   - Same import/signature updates
3. Run verification:
   - `ailang check examples/runnable/stream_websocket.ail`
   - `ailang check examples/runnable/stream_sse.ail`
4. Run full test suite: `make test`
5. Run lint: `make lint`

**Acceptance Criteria:**
- [ ] Both example files type-check successfully
- [ ] `make test` passes
- [ ] `make lint` passes (no new warnings)
- [ ] Handler functions accept `StreamEvent` parameter instead of `string`

**Risks:**
- Examples may use patterns not yet supported in type checker — Mitigation: Keep examples simple, use basic match patterns

## Success Metrics
- Test coverage: All existing stream tests still passing
- Examples passing: Both stream examples type-check with typed ADTs
- Documentation: std/stream.ail function signatures updated
- All tests passing: `make test` green
- All linting passing: `make lint` clean
- **Blocker resolved:** 6 demos can now import StreamConn/StreamEvent/Message types

## Dependencies
- Phase 1 bug fixes (DONE)
- std/result module (exists — provides Result, Ok, Err)
- Type builder DSL (exists — T.Con(), T.App() proven working in std/net)

## Open Questions
- None — all patterns are proven working in std/option, std/net, std/result

## Notes
- Phase 2 type simplifications: `Binary(string)` not `Binary(bytes)`, `Opened(string)` not `Opened({record})` — can refine in Phase 3
- Backward compatibility: extractConnID auto-unwrap remains for both typed and untyped code paths
- transmit auto-wrap: Go still accepts plain strings alongside Text/Bin ADTs
- Total LOC: ~280 (implementation + test updates)
