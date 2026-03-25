# Sprint Plan: M-DEBUG-ERASURE — Debug Ghost Effect Erasure

## Summary

Implement the missing Debug effect erasure pass so `--release` actually erases Debug calls and effect rows. This completes the ghost effect promise from M-GAME-E1 (v0.5.0) and unblocks sunholo/logging v0.2.0's zero-cost claim.

**Duration:** 1 day (4 milestones, ~6 hours)
**Dependencies:** None (Debug runtime already works)
**Risk Level:** Low — additive pass, only active with `--release` flag
**Design Doc:** `design_docs/planned/v0_10_0/m-debug-erasure-release-mode.md`

## Current Status Analysis

### Completed Recently
- `693e03fa` Add `--net-allow-metadata` flag (~80 LOC)
- `fe0a51dd` M-PKG-METADATA-URLS: repo/homepage/license_url fields (~200 LOC)
- `1cd08456` Version conflict detection in resolver (~150 LOC)
- `8a4c92cc` Package explorer components (~400 LOC)

### Velocity
- Recent average: ~150-200 LOC/day (package/tooling work, some design-doc-heavy days)
- This sprint is pure pipeline work — faster pace expected (~250 LOC/day)

### What Exists (Partial)
- `--release` flag defined in `compile.go:24` ✅
- "RELEASE MODE" message printed ✅
- `ReleaseMode` in `pipeline.Config` ❌ **MISSING**
- Debug erasure pass ❌ **MISSING**
- Wiring to `ailang run` ❌ **MISSING**

## Proposed Milestones

### M1: Core Erasure Pass
**Goal:** Create `DebugEraser` that removes Debug from effect rows and rewrites Debug calls to unit.
**Estimated:** 120 LOC implementation + 80 LOC tests = 200 LOC
**Duration:** ~2 hours

**Tasks:**
1. Add `ReleaseMode bool` to `pipeline.Config` struct
2. Create `internal/pipeline/debug_erasure.go`:
   - `DebugEraser` struct (follows `OpLowerer` pattern)
   - `eraseEffectRow(*types.Row) *types.Row` — remove "Debug" label, nil if empty
   - `eraseExpr(core.CoreExpr) core.CoreExpr` — recursive walker, `_debug_log`/`_debug_check` → unit
   - `Erase(*core.Program) *core.Program` — iterate all decls
3. Create `internal/pipeline/debug_erasure_test.go`:
   - Test: `! {Debug}` only → nil (pure)
   - Test: `! {IO, Debug}` → `! {IO}`
   - Test: Debug call App node → unit Lit
   - Test: nested Debug in let bindings
   - Test: nil effect row stays nil

**Key files to reference:**
- `internal/pipeline/op_lowering.go` — expr walker pattern (lines 122-300)
- `internal/core/core.go` — `App`, `Lit`, `UnitLit`, `VarGlobal` types
- `internal/types/effects.go` — `Row` type, `Labels` map

**Acceptance Criteria:**
- [ ] Unit tests pass for effect row erasure
- [ ] Unit tests pass for expression rewriting
- [ ] `_debug_log` and `_debug_check` calls become `Lit{Kind: UnitLit}`
- [ ] Effect rows with only Debug become nil (pure)
- [ ] Mixed effect rows lose Debug but keep others

### M2: Pipeline Integration
**Goal:** Wire erasure into both single-file and module pipelines, after lowering.
**Estimated:** 40 LOC implementation + 20 LOC tests = 60 LOC
**Duration:** ~1 hour

**Tasks:**
1. Insert erasure in `pipeline_single.go` after `lowerer.Lower()` (~line 471):
   ```go
   if cfg.ReleaseMode {
       eraser := &DebugEraser{}
       loweredProg = eraser.Erase(loweredProg)
   }
   ```
2. Insert erasure in `pipeline_module_compile.go` after `lowerer.Lower()` (~line 438):
   ```go
   if cfg.ReleaseMode {
       eraser := &DebugEraser{}
       unit.Core = eraser.Erase(unit.Core)
   }
   ```
3. Integration test: compile Debug-using program with ReleaseMode=true, verify no Debug in output

**Key files:**
- `internal/pipeline/pipeline_single.go:464-471` — single-file lowering
- `internal/pipeline/pipeline_module_compile.go:428-438` — module lowering

**Acceptance Criteria:**
- [ ] Single-file pipeline erases Debug when `ReleaseMode=true`
- [ ] Module pipeline erases Debug when `ReleaseMode=true`
- [ ] Non-release mode is completely unaffected
- [ ] `make test` passes (no regressions)

### M3: CLI Wiring
**Goal:** Wire `--release` flag to pipeline Config in both `compile` and `run` commands.
**Estimated:** 30 LOC
**Duration:** ~30 minutes

**Tasks:**
1. `cmd/ailang/compile.go:108` — add `ReleaseMode: *releaseFlag` to Config
2. `cmd/ailang/main_run.go` — add `--release` flag, pass to runFile/pipeline
3. Verify: `ailang run --release examples/runnable/debug_effect.ail` produces no Debug output
4. Verify: `ailang compile --release --emit-go` generates Debug-free Go

**Key files:**
- `cmd/ailang/compile.go:108-111` — Config construction
- `cmd/ailang/main_run.go` — runFile function and flag parsing

**Acceptance Criteria:**
- [ ] `ailang compile --release` passes ReleaseMode to pipeline
- [ ] `ailang run --release` passes ReleaseMode to pipeline
- [ ] Debug calls produce no output in release mode
- [ ] Non-release behavior unchanged

### M4: Docs and Edge Cases
**Goal:** Update CHANGELOG, verify edge cases, update design doc status.
**Estimated:** 30 LOC docs + edge case tests
**Duration:** ~30 minutes

**Tasks:**
1. CHANGELOG.md entry for Debug erasure
2. Edge case: function whose ONLY effect is Debug → becomes pure (verify works)
3. Edge case: Debug in match arms (verify erasure recurses)
4. Update design doc `m-debug-erasure-release-mode.md` status → Implemented
5. Move to `design_docs/implemented/v0_10_0/`

**Acceptance Criteria:**
- [ ] CHANGELOG updated
- [ ] All edge cases covered by tests
- [ ] Design doc moved to implemented
- [ ] `make test` && `make lint` clean

## Success Metrics

- All existing tests pass (erasure only active with `--release`)
- New tests: ≥8 test cases for erasure pass
- `ailang run --release examples/runnable/debug_effect.ail` — zero Debug output
- `ailang compile --release --emit-go` — generated Go has no Debug references
- Linting clean

## Dependencies
- None — Debug runtime already works, this is purely additive

## Open Questions
- None — design is straightforward, follows existing `OpLowerer` pattern
