# Consumer Contract Gap Analysis

**Analysis Date**: 2025-12-02
**Current AILANG Version**: v0.4.10 (preparing v0.5.x)
**Contract Document**: `consumer-contract-v0.5.md`

This document analyzes the gaps between the consumer contract requirements and current implementation.

---

## Summary

| Requirement | Contract Version | Status | Notes |
|-------------|------------------|--------|-------|
| Go codegen CLI | v0.5.0 | ✅ **DONE** | M-GAME-C |
| Record type codegen | v0.5.0 | ✅ **DONE** | M-GAME-A |
| Sum type discriminator codegen | v0.5.0 | ✅ **DONE** | Verified 2025-12-02 |
| `export func` → Go callable | v0.5.0 | ⚠️ **EXPERIMENTAL** | Marked experimental |
| RNG effect | v0.5.1 | ⚠️ **INTERPRETER ONLY** | Not in Go codegen |
| Debug effect | v0.5.1 | ❌ **NOT DONE** | New requirement |
| AI effect | v0.5.1 | ❌ **NOT DONE** | New requirement |
| `extern func` | v0.5.2 | ✅ **DONE** | M-GAME-C |
| CLI flags | v0.5.2 | ✅ **DONE** | M-GAME-C |
| ABI stability docs | v0.5.3 | ✅ **DONE** | M-GAME-D |
| `examples/sim_stub/` | v0.5.0 | ✅ **DONE** | M-GAME-D |
| CI integration | v0.5.0 | ✅ **DONE** | M-GAME-D |

**Overall: 8/12 complete, 2 partial, 2 not started**

---

## Detailed Gap Analysis

### 1. Go Codegen Available ✅ DONE

**Contract requires:**
```bash
ailang compile --emit-go --package-name <name> --out <dir> <file.ail>
```

**Current implementation:**
- ✅ `--emit-go` flag works
- ✅ `--package-name` flag works
- ✅ `--out` flag works
- ✅ Generates valid Go code
- ✅ Deterministic output

**No gaps.**

---

### 2. ADT → Discriminator Structs ✅ DONE

**Contract requires:**
```go
// Sum type generates discriminator-based struct
type DrawCmdKind int
const (
    DrawCmdKindSprite DrawCmdKind = iota
    DrawCmdKindRect
    DrawCmdKindText
)
type DrawCmd struct {
    Kind   DrawCmdKind
    Sprite *DrawCmdSprite
    // ...
}
```

**Current implementation (verified 2025-12-02):**
- ✅ Discriminator-based structs (NOT interface-based)
- ✅ `XxxKind` enum with iota constants
- ✅ Separate struct per variant (`DrawCmdSprite`, etc.)
- ✅ Main struct with `Kind` + nullable variant pointers
- ✅ Constructor functions `NewDrawCmdSprite(...)`, etc.
- ✅ Helper methods `IsSprite()`, `IsRect()`, `IsText()`

**Minor limitation (not blocking):**
- Contract shows named fields: `| Sprite(x: int, y: int, id: int)`
- AILANG syntax: `| Sprite(int, int, int)` (positional only)
- Generated fields: `Value0`, `Value1` instead of `X`, `Y`

**Future enhancement:** Support named constructor fields for better Go field names.

**No gaps for v0.5.x.**

---

### 3. Exported Functions Callable from Go ⚠️ EXPERIMENTAL

**Contract requires:**
```go
func InitWorld(seed int64) World { ... }
func Step(world World, input FrameInput) (World, FrameOutput, error) { ... }
```

**Current implementation:**
- ⚠️ `funcs.go` generation marked "experimental"
- ⚠️ May fail for complex functions
- ⚠️ Effects don't propagate as `error` return

**Gaps:**
1. Function codegen is incomplete/experimental
2. Effect-to-error propagation not implemented
3. Pure functions vs effectful functions handling

**Action needed:**
1. Improve function codegen reliability
2. Add effect → error propagation
3. Handle pure functions (no error return)

---

### 4. RNG Effect with Determinism ⚠️ INTERPRETER ONLY

**Contract requires:**
```ailang
func generate_map(seed: int) -> Map ! {RNG} {
    let width = RNG.rand_int(100)
    ...
}
```

With guarantees:
- `AILANG_SEED=N` produces identical sequences
- Capability check enforced at runtime

**Current implementation:**
- ✅ `std/rand` module exists with builtins
- ✅ `_rand_seed`, `_rand_int`, `_rand_float`, `_rand_bool` implemented
- ⚠️ Works in interpreter, **NOT in Go codegen**
- ⚠️ No `AILANG_SEED` environment variable support

**Gaps:**
1. RNG effect not available in generated Go code
2. No environment variable seeding

**Action needed:**
1. Generate RNG handler code for Go
2. Add `AILANG_SEED` env var support
3. Ensure determinism in generated code

---

### 5. Debug Effect with Structured Output ❌ NOT DONE

**Contract requires:**
```ailang
effect Debug {
    assert(cond: bool, msg: string) -> unit
    log(msg: string) -> unit
}
-- Note: collect() is HOST-ONLY, not callable from AILANG
```

With guarantees:
- Assertions collected, not thrown
- Host calls `DebugContext.Collect()` after step returns
- `--release` mode erases Debug from effect rows AND calls
- Debug is a "ghost effect" - erasable at type level

**Current implementation:**
- ❌ No `Debug` effect defined
- ❌ No `Debug.assert`, `Debug.log`
- ❌ No `--release` mode

**Gaps:**
1. Entire Debug effect needs implementation
2. Need structured DebugOutput type (shared schema)
3. Need release mode with effect row erasure
4. Need location injection (hidden arguments)

**Design doc updated:** See M-GAME-E1-debug-effect.md
- Redesigned as foundational tracing substrate (not game-specific)
- Write-only effect (no collect in language)
- Ghost effect semantics (erasable in release)
- Backend-agnostic host contract

---

### 6. AI Effect with Pluggable Handler ❌ NOT DONE

**Contract requires:**
```ailang
effect AI {
    call(input: string) -> string  -- Opaque string→string, JSON by convention
}
```

With guarantees:
- Generic string→string interface (JSON by convention, not enforced)
- Operation named `call` (neutral, not game-flavored)
- Handler pluggable at Go runtime
- Nil handler = **error** (no silent fallback)
- Explicit stub handler for tests

**Current implementation:**
- ❌ No `AI` effect defined
- ❌ No string→string oracle interface

**Gaps:**
1. Entire AI effect needs implementation
2. Need pluggable handler system for Go
3. Need explicit stub mode (not default)
4. Need nil handler error

**Design doc updated:** See M-GAME-E2-ai-effect.md
- Redesigned as general-purpose AI oracle (not game-specific)
- Operation renamed from `decide` to `call`
- Nil handler = loud error, not silent stub
- Record/replay handler pattern for debugging
- Integration with Debug effect for telemetry

---

### 7. Extern Functions ✅ DONE

**Contract requires:**
```ailang
extern func find_path(world: World, from: Coord, to: Coord) -> Path
```

**Current implementation:**
- ✅ `extern func` syntax parsed
- ✅ Type checking works
- ✅ Stubs generated in `extern_stubs.go`
- ✅ Error codes EXT001-003
- ✅ Documentation in go-interop.md

**No gaps.**

---

### 8. CLI Flags ✅ DONE

**Contract requires:**
- `--emit-go`
- `--package-name <name>`
- `--out <dir>`

**Current implementation:**
- ✅ All flags implemented in `cmd/ailang/compile.go`

**No gaps.**

---

### 9. ABI Stability Documentation ✅ DONE

**Contract requires:**
- Stable preview promise
- Breaking change policy
- Migration guide

**Current implementation:**
- ✅ `docs/docs/guides/go-interop.md` has ABI stability section
- ✅ README has Go Interop section
- ✅ CHANGELOG has ABI freeze announcement

**No gaps.**

---

### 10. `examples/sim_stub/` ✅ DONE

**Contract requires:**
- Minimal `world.ail` with types
- Minimal `main.go` driver that runs 10 ticks
- CI job that validates output

**Current implementation:**
- ✅ `examples/sim_stub/world.ail`
- ✅ `examples/sim_stub/main.go`
- ✅ `examples/sim_stub/impl.go`
- ✅ `examples/sim_stub/Makefile`
- ✅ `examples/sim_stub/expected_output.txt`
- ✅ `make test-sim-stub` target
- ✅ `.github/workflows/test-game-codegen.yml`

**No gaps.**

---

## Priority Roadmap

### Phase 1: Critical (v0.5.0 release blockers)

1. **Verify sum type codegen** (1 day)
   - Test with DrawCmd example
   - Document actual format
   - Update if needed

2. **Stabilize function codegen** (3-5 days)
   - Remove "experimental" designation
   - Fix common failure cases
   - Add comprehensive tests

### Phase 2: Effects (v0.5.1)

3. **Debug effect** (2-3 days)
   - Define effect and builtins
   - Implement collector
   - Add to Go codegen

4. **AI effect** (2-3 days)
   - Define effect
   - Create handler interface
   - Implement stub

5. **RNG in codegen** (1-2 days)
   - Generate RNG handler
   - Add AILANG_SEED support

### Phase 3: Polish (v0.5.2-0.5.3)

6. **Effect → error propagation** (2 days)
   - Pure functions: no error return
   - Effectful functions: return error

7. **Release mode** (1 day)
   - `--release` flag
   - Debug code elimination

---

## Recommended Next Sprint

**M-GAME-E: Effect System for Go Codegen**

Focus: Implement Debug and AI effects with Go code generation support.

Milestones:
- E1: Debug effect (assert, log, collect)
- E2: AI effect (decide with JSON interface)
- E3: RNG handler for Go codegen
- E4: Effect → error propagation

Estimated: 10-12 days

---

*This gap analysis should be reviewed before each release to track progress against the consumer contract.*
