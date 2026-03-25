# M-DEBUG-ERASURE: Implement Debug Ghost Effect Erasure in Release Mode

**Status**: Planned
**Target**: v0.10.0
**Priority**: P0 (Debug is marketed as ghost/erasable but erasure is unimplemented)
**Estimated**: 1-2 days (~6 hours)
**Dependencies**: None (Debug effect already works at runtime)
**Origin**: sunholo/logging v0.2.0 rewrite depends on ghost erasure; discovery via agent message audit (2026-03-24)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No resolution semantics changed |
| A2: Replayability | +1 | Release builds have deterministic erasure |
| A3: Effect Legibility | +2 | Ghost effects actually erase as documented |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Release mode is a verifiable build flag |
| A7: Machines First | +1 | Agents can rely on ghost erasure semantics |
| A9: Cost Visibility | +2 | Debug truly becomes zero-cost in release |
| A11: Structured Failure | 0 | No failure changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Erasure is deterministic (all Debug removed)
- [x] A3 (Effects): Makes effect declarations match reality
- [x] A4 (Authority): No ambient access changes
- [x] A7 (Machines First): Agents can reason about ghost effects correctly

## Problem Statement

The Debug effect was introduced as a "ghost effect" in v0.4.10 (M-GAME-E1). The design specifies:
- In release mode (`--release`), Debug is erased from effect rows
- Debug calls (`Debug.log`, `Debug.check`) are rewritten to `()`
- Functions with only `! {Debug}` become pure in release builds
- Zero runtime cost in production

**What actually exists:**
- `cmd/ailang/compile.go:24` — `--release` flag is defined
- `cmd/ailang/compile.go:101-105` — Prints "RELEASE MODE - Debug erased"
- **Nothing else.** The flag is never passed to `pipeline.Config`. No erasure pass exists.

**Impact:**
- Documentation/prompts claim Debug is erasable — it isn't
- sunholo/logging v0.2.0 was rewritten to use Debug specifically for ghost semantics
- `! {Debug}` functions cannot become pure in release builds
- Effect-polymorphic code that requires purity rejects Debug-only functions

## Goals

1. `--release` flag actually erases Debug from Core AST
2. Effect rows with only Debug become pure (nil)
3. Debug.log/Debug.check calls become `()` (unit literal)
4. Existing tests unaffected (erasure only in release mode)
5. `ailang run --release` also works (not just compile)

## Solution Design

### New Pipeline Pass: `DebugEraser`

New file: `internal/pipeline/debug_erasure.go`

```go
type DebugEraser struct{}

func (e *DebugEraser) Erase(prog *core.Program) *core.Program
func (e *DebugEraser) eraseExpr(expr core.CoreExpr) core.CoreExpr
func (e *DebugEraser) eraseEffectRow(row *types.Row) *types.Row
```

**eraseEffectRow**: Remove "Debug" label from effect row. If row becomes empty, return nil (pure).

**eraseExpr**: Recursive expression walker (same pattern as `OpLowerer.lowerExpr`):
- `App` where callee is `_debug_log` or `_debug_check` → return `Lit{Kind: UnitLit}`
- `Lambda` → erase effect row in type annotation
- `Let`/`LetRec`/`If`/`Match` → recurse into children
- All other nodes → recurse, preserve structure

**Erase**: Iterate all declarations, apply `eraseExpr` to each body, update function types.

### Pipeline Integration

**Insertion point**: After `OpLowerer.Lower()`, before evaluation/codegen.

At this point Debug calls are already lowered to canonical `App` form, making pattern matching straightforward.

### Config Changes

```go
// internal/pipeline/pipeline.go — Config struct
ReleaseMode bool  // M-DEBUG-ERASURE: Erase Debug ghost effect
```

### Wiring

1. `cmd/ailang/compile.go` — Pass `ReleaseMode: *releaseFlag` to `pipeline.Config`
2. `cmd/ailang/main_run.go` — Add `--release` flag to `ailang run`, pass to Config
3. `internal/pipeline/pipeline_single.go` — After lowering, if `cfg.ReleaseMode`, run `DebugEraser.Erase()`
4. `internal/pipeline/pipeline_module.go` — Same insertion for module pipeline

### Effect Validation Interaction

Effect validation (`ValidateEffects`) runs **before** lowering in the pipeline. Two options:

**Option A (recommended)**: Leave validation as-is. Debug is valid in source declarations. Erasure happens after validation, before eval/codegen. Source code still declares `! {Debug}` — the erasure is a post-validation optimization.

**Option B**: Skip Debug in validation during release mode. Unnecessary — Debug declarations are correct, they just get erased later.

### Runtime Path (`ailang run --release`)

For interpreted execution, two approaches:

**Option A (recommended)**: Erase in Core AST before eval. Same pass as compile, just applied to the eval path.

**Option B**: Make DebugContext a no-op. Simpler but less principled — Debug calls still execute, just do nothing.

Choose Option A for consistency: erasure happens at the same pipeline stage regardless of execution mode.

## Implementation Plan

### Phase 1: Core erasure pass (~2 hours)
- [ ] Add `ReleaseMode bool` to `pipeline.Config`
- [ ] Create `internal/pipeline/debug_erasure.go` with `DebugEraser`
- [ ] Implement `eraseEffectRow` (remove Debug from Row labels)
- [ ] Implement `eraseExpr` (rewrite Debug calls to unit, recurse)
- [ ] Implement `Erase` (iterate declarations)
- [ ] Unit tests: `internal/pipeline/debug_erasure_test.go`

### Phase 2: Pipeline integration (~1.5 hours)
- [ ] Insert erasure after lowering in `pipeline_single.go`
- [ ] Insert erasure after lowering in `pipeline_module.go`
- [ ] Wire `--release` in `compile.go` to `Config.ReleaseMode`
- [ ] Add `--release` flag to `ailang run` in `main_run.go`
- [ ] Integration test: compile with `--release`, verify Debug absent from output

### Phase 3: Codegen support (~1.5 hours)
- [ ] Verify Go codegen omits Debug-related code in release mode
- [ ] Generate build-tagged debug stubs (if not already handled)
- [ ] Test: `ailang compile --release --emit-go` produces Debug-free Go

### Phase 4: Docs + edge cases (~1 hour)
- [ ] Update CHANGELOG.md
- [ ] Move original M-GAME-E1 design doc note about erasure to "implemented"
- [ ] Edge case: function whose ONLY effect is Debug → becomes pure
- [ ] Edge case: nested Debug calls in let bindings
- [ ] Edge case: Debug in match arms

## Files to Modify

| File | Change |
|------|--------|
| `internal/pipeline/pipeline.go` | Add `ReleaseMode` to Config |
| `internal/pipeline/debug_erasure.go` | **NEW** — DebugEraser pass |
| `internal/pipeline/debug_erasure_test.go` | **NEW** — Unit tests |
| `internal/pipeline/pipeline_single.go` | Insert erasure after lowering |
| `internal/pipeline/pipeline_module.go` | Insert erasure after lowering |
| `cmd/ailang/compile.go` | Wire `releaseFlag` to Config |
| `cmd/ailang/main_run.go` | Add `--release` flag |

## Effort Estimate

| Phase | LOC | Time |
|-------|-----|------|
| P1: Core erasure pass | ~150 | 2 hours |
| P2: Pipeline integration | ~50 | 1.5 hours |
| P3: Codegen support | ~30 | 1.5 hours |
| P4: Docs + edge cases | ~30 | 1 hour |
| **Total** | **~260** | **~6 hours (1 day)** |

**Risk level:** Low — additive pass, no existing semantics changed, only active when `--release` flag is passed.

## Verification

1. `make test` — all existing tests pass (erasure not active by default)
2. `ailang run --release examples/runnable/debug_effect.ail` — Debug calls produce no output
3. `ailang compile --release --emit-go examples/runnable/debug_effect.ail` — generated Go has no Debug references
4. New test: function with `! {Debug}` only → effect row is nil after erasure
5. New test: `! {IO, Debug}` → becomes `! {IO}` after erasure

## Non-Goals

- Changing Debug runtime behavior in non-release mode
- Adding other ghost effects (future work)
- Build-tagged Go files for debug vs release (already partially exists in codegen)
- Workspace-mode erasure policies

## Related Documents

- [M-GAME-E1-debug-effect.md](../../implemented/v0_5_0/M-GAME-E1-debug-effect.md) — Original Debug design (specifies erasure)
- sunholo/logging v0.2.0 AGENT.md — Claims zero-cost via ghost erasure

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
