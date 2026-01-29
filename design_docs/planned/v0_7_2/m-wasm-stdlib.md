# M-WASM-STDLIB: Embed Standard Library in WASM Binary

**Status**: Implemented
**Target**: v0.7.2
**Priority**: P1 - Medium (follows M-WASM-REPL)
**Estimated**: 2 hours
**Dependencies**: M-WASM-REPL (completed)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact - same stdlib, just embedded |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Effects remain explicit in stdlib |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Improves AI tooling - stdlib "just works" in browser |
| A8: Minimal Syntax | +1 | No new syntax - removes need for workarounds |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | stdlib composes seamlessly in browser |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: ✅ Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine tooling in browser

## Problem Statement

**Current State:**
- WASM REPL cannot use stdlib modules like `std/list`, `std/json`, `std/string`
- Browser has no filesystem access to load `std/*.ail` files
- Users must manually `loadModule()` equivalent code for every stdlib function they need
- Makes browser demos significantly harder to build

**Impact:**
- Demo builders must duplicate stdlib functionality
- Browser experience is severely limited compared to CLI
- Reduces AILANG's appeal for interactive tutorials and playgrounds

## Goals

**Primary Goal:** Make AILANG stdlib available automatically in browser WASM builds.

**Success Metrics:**
- `repl.importModule('std/list')` works without manual loading
- All 20 stdlib modules available in browser
- Binary size increase < 100KB
- Init time increase < 200ms

## Solution Design

### Overview

Use Go's `embed` package to include stdlib source files in the WASM binary. On REPL initialization, load all stdlib modules into the ModuleRegistry automatically.

### Architecture

**Components:**
1. **Embedded FS**: Go embed directive for `std/*.ail` files
2. **Auto-loader**: Load stdlib into ModuleRegistry on init
3. **Lazy compilation option**: Compile on first import (optional optimization)

### Implementation Plan

**Phase 1: Embed and Auto-load** (~1.5 hours)
- [ ] Add `//go:embed std/*.ail` directive to cmd/wasm/main.go
- [ ] Create `loadEmbeddedStdlib()` function
- [ ] Load all stdlib modules on `NewWasmREPL()` init
- [ ] Test in browser

**Phase 2: Documentation** (~30 min)
- [ ] Update WASM integration docs
- [ ] Update limitations table
- [ ] Add examples using stdlib in browser

### Files to Modify/Create

**Modified files:**
- `cmd/wasm/main.go` - Add embed directive and auto-load, ~30 LOC

**Documentation:**
- `docs/docs/guides/wasm-integration.md` - Update limitations section

## Examples

### Example 1: Using std/list in Browser

**Before (v0.7.2 without this feature):**
```javascript
// Must manually define list functions
repl.loadModule('mylist', `
let map = \\f. \\xs. ...  -- must reimplement!
let filter = \\p. \\xs. ...
`);
```

**After:**
```javascript
// Stdlib just works!
repl.importModule('std/list');
repl.eval('map(\\x. x * 2, [1, 2, 3])');
// Returns: "[2, 4, 6] :: [Int]"
```

### Example 2: JSON Processing in Browser

```javascript
repl.importModule('std/json');
repl.eval('json_decode("[1,2,3]")');
// Returns: "Some([1, 2, 3]) :: Option[Json]"
```

## Success Criteria

- [x] `repl.importModule('std/list')` works in browser
- [x] `repl.importModule('std/json')` works in browser
- [x] `repl.importModule('std/string')` works in browser
- [x] All 20 stdlib modules importable
- [x] Binary size increase measured and documented (~120KB stdlib embedded, 33MB total WASM)
- [ ] Init time increase measured and documented (requires browser testing)
- [x] All existing tests passing
- [x] Documentation updated
- [x] Limitations table updated

## Testing Strategy

**Unit tests:**
- Test embedded FS contains expected files
- Test loadEmbeddedStdlib() loads all modules

**Integration tests:**
- Build WASM and verify stdlib available in browser
- Test typical stdlib usage patterns

**Manual testing:**
- Load playground page
- Import std/list, std/json, std/string
- Execute typical operations

## Non-Goals

**Not in this feature:**
- Lazy compilation (compile all upfront for simplicity)
- Selective stdlib inclusion (include all 20 modules)
- Minification of stdlib source (source is small enough)

## Timeline

**Day 1** (~2 hours):
- Phase 1: Embed and auto-load implementation
- Phase 2: Documentation updates
- Testing and verification

**Total: ~2 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Binary size increase | Low | Stdlib is ~50KB source, gzip helps |
| Init time slowdown | Medium | Measure first; could lazy-load if needed |
| Stdlib compilation errors in WASM | Medium | Test each module individually |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_10/m-string-conversion.md](design_docs/implemented/v0_5_10/m-string-conversion.md) (0.51)
- [design_docs/implemented/v0_6_1/m-dx21-stdlib-version-warning-once.md](design_docs/implemented/v0_6_1/m-dx21-stdlib-version-warning-once.md) (0.49)
- [design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md](design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md) (0.46)

**Planned (check for overlap):**
- [design_docs/planned/v0_7_1/m-stdlib-gaps.md](design_docs/planned/v0_7_1/m-stdlib-gaps.md) (0.46)
- [design_docs/planned/v0_7_2/m-wasm-repl-module-loading.md](design_docs/planned/v0_7_2/m-wasm-repl-module-loading.md) (0.45) - **Prerequisite**

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [M-WASM-REPL](m-wasm-repl-module-loading.md) - Module loading foundation
- [Go embed package](https://pkg.go.dev/embed) - Embedding files in Go binaries

## Future Work

- **Lazy compilation**: Only compile stdlib modules on first import (if init time is a problem)
- **Selective bundling**: Build flag to include/exclude specific stdlib modules
- **CDN stdlib**: Fetch stdlib from CDN instead of embedding (for smaller binary)

---

**Document created**: 2026-01-29
**Last updated**: 2026-01-29
