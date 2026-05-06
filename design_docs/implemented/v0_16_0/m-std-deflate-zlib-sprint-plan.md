# Sprint Plan: M-STD-DEFLATE-ZLIB

## Summary

Ship a new `std/deflate` stdlib module exposing four pure functions (`inflate`, `inflateZlib`, `deflate`, `deflateZlib`) backed by Go's `compress/flate` and `compress/zlib`. Unblocks the ailang-parse PDF annotation extractor (FlateDecode in PDF `/ObjStm` streams) and lays groundwork for HTTP `Content-Encoding: deflate`, PNG `IDAT`, and WebSocket `permessage-deflate` use cases.

**Duration:** 1 day (~6 hours, single session)
**Dependencies:** None — sits alongside existing `std/gzip` (v0.12.0)
**Risk Level:** Low — direct mirror of `std/gzip` shape; Go stdlib provides all four primitives natively

**Source request:** msg_20260506_074820_1f185b21 from `cli` agent (ailang-parse)
**Design doc:** [design_docs/planned/v0_16_0/m-std-deflate-zlib.md](m-std-deflate-zlib.md)

## Current Status Analysis

### Completed Recently (precedent)
- ✅ **std/gzip** (v0.12.0): ~297 LOC builtins + ~30 LOC stdlib wrapper + tests — direct template for this sprint
- ✅ **std/zip** (v0.7.3 + v0.11.0): ~260 LOC builtins for ZIP-archive APIs — established the base64 boundary contract
- ✅ **std/tar** (v0.12.0): shipped alongside std/gzip — same sprint shape

### Velocity Reference
- std/gzip total: ~500 LOC (builtins + tests + stdlib + example) shipped in a single sprint
- This sprint targets ~460 LOC, same shape, no new architecture

### Remaining from Design Doc
All three phases (builtins, stdlib wrapper, tests+example) are unimplemented.

## Proposed Milestones

### Milestone 1: Builtins (`internal/builtins/deflate.go`)

**Goal:** Register four Go-side builtins backed by `compress/flate` and `compress/zlib`. Apply the 100MB output cap (mirroring `std/gzip`).

**Estimated:** ~250 LOC implementation + ~150 LOC tests = **~400 LOC**
**Duration:** ~3 hours

**Tasks:**
- Copy `internal/builtins/gzip.go` as scaffold for `deflate.go`
- Register `_deflate_inflate`: raw deflate via `flate.NewReader`
- Register `_deflate_inflateZlib`: zlib-wrapped via `zlib.NewReader`
- Register `_deflate_deflate(level)`: raw deflate via `flate.NewWriter(level)`
- Register `_deflate_deflateZlib(level)`: zlib-wrapped via `zlib.NewWriterLevel`
- Apply 100MB decompressed-size cap (re-use the limit pattern from `gzip.go`)
- Wire registrations into `RegisterAll` (same place gzip is registered)
- Write `internal/builtins/deflate_test.go`: round-trip × 4 functions × 5 levels, cross-format negative, malformed input, 100MB cap test, level-monotonicity test

**Acceptance Criteria:**
- [ ] All four builtins registered (visible via `ailang prompt` after install)
- [ ] Round-trip tests pass: `inflate(deflate(x, level))` == x for level ∈ {0, 1, 6, 9, -1}
- [ ] Round-trip tests pass: `inflateZlib(deflateZlib(x, level))` == x
- [ ] Cross-format negative: `inflate(deflateZlib(x, 6))` returns `Err` (header bytes confuse raw deflate)
- [ ] Malformed input → typed `Err`, no panic
- [ ] Output cap test: hand-built stream that decompresses past 100MB → `Err`
- [ ] All tests passing (`make test`)
- [ ] `make lint` clean

**Risks:**
- Output-cap test fixture is fiddly to construct. **Mitigation:** copy the gzip cap-test pattern directly; same `io.LimitReader`-based guard.

---

### Milestone 2: Stdlib wrapper (`std/deflate.ail`)

**Goal:** Expose the four builtins as a clean AILANG-level module with doc-comments matching `std/gzip.ail`.

**Estimated:** ~30 LOC (mirrors `std/gzip.ail` exactly)
**Duration:** ~1 hour

**Tasks:**
- Create `std/deflate.ail` with module header, `import std/result (Result)`, four `export pure func` declarations
- Add doc-comments explaining the two pairs (raw vs zlib-wrapped) with concrete use cases (PDF, HTTP, PNG)
- Verify stdlib loader picks it up (likely zero changes if discovery is glob-based; check `internal/loader` if not)
- `make build && make install` and verify `ailang -e 'import std/deflate (inflate); ...'` resolves

**Acceptance Criteria:**
- [ ] `std/deflate.ail` exports `inflate`, `inflateZlib`, `deflate`, `deflateZlib`
- [ ] Module loads without error (`ailang run` on a one-liner that imports it)
- [ ] Doc-comments include the "use `inflateZlib` for HTTP/PDF, `inflate` for raw deflate" guidance

**Risks:**
- Loader auto-discovery might miss it. **Mitigation:** copy `std/gzip` registration verbatim; if there's an explicit list, add `deflate` next to `gzip`.

---

### Milestone 3: Example + CHANGELOG + design doc move

**Goal:** Ship a runnable example demonstrating PDF FlateDecode-style usage, update CHANGELOG, and move the design doc to `implemented/v0_16_x/`.

**Estimated:** ~30 LOC example + ~20 LOC CHANGELOG/doc updates = **~50 LOC**
**Duration:** ~2 hours

**Tasks:**
- Create `examples/std_deflate_pdf_objstm.ail`: hand-rolled FlateDecode payload (a small zlib-wrapped string), decode it via `inflateZlib`, print the result
- Run `make verify-examples` to confirm it passes
- Add CHANGELOG.md entry under v0.16.0 with API summary + use cases
- Move design doc from `design_docs/planned/v0_16_0/` to `design_docs/implemented/v0_16_x/`
- Add brief implementation report appended to the moved doc (LOC counts, file paths, test coverage)

**Acceptance Criteria:**
- [ ] `examples/std_deflate_pdf_objstm.ail` runs to completion via `make verify-examples`
- [ ] CHANGELOG.md has v0.16.0 entry referencing `std/deflate` with the four function names and primary use case (PDF FlateDecode)
- [ ] Design doc moved to `design_docs/implemented/v0_16_x/m-std-deflate-zlib.md`
- [ ] Implementation report appended (status: Implemented, LOC actual vs estimate, files)

**Risks:**
- `make verify-examples` is strict about effect annotations. **Mitigation:** the example is pure (no `! {IO}` needed for the deflate calls); only the final `println` carries `! {IO}`.

---

## Success Metrics

- **Test coverage:** All four functions covered by round-trip + boundary + error tests (~150 LOC of tests for ~250 LOC of impl = 60% test ratio, healthy)
- **Examples passing:** 1 new runnable example (`examples/std_deflate_pdf_objstm.ail`)
- **Documentation updated:** CHANGELOG.md, design doc moved to implemented/
- **All tests passing:** ✅ `make test`
- **All linting passing:** ✅ `make lint`
- **External validation:** `cli` agent (ailang-parse) confirms migration to `inflateZlib` handles `/ObjStm`-bearing PDFs (out-of-tree, post-merge)

## Dependencies

- None internal — Go's `compress/flate` and `compress/zlib` are stdlib, std/gzip established the boundary contract
- External (post-sprint): ailang-parse migrates its PDF annotation extractor to use `inflateZlib`; not blocking this sprint

## Open Questions

None — both design-freeze items are resolved (new `std/deflate` module, 100MB cap).

## Notes

- **Why a single-day sprint:** The architecture is locked, the boundary contract is established, and Go's stdlib provides all four primitives natively. This is a pure mechanical port of the `std/gzip` template with a different compress/decompress algorithm underneath.
- **Naming clarification documented in M1:** `inflate`/`deflate` = raw RFC 1951; `inflateZlib`/`deflateZlib` = RFC 1950 wrapped (header + adler32 trailer). Module doc-comment and example both call this out.
- **Out of scope:** streaming inflate, native `bytes` type, file-based variants — see design doc Non-Goals.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_16_0/m-std-deflate-zlib-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-STD-DEFLATE-ZLIB.json`
