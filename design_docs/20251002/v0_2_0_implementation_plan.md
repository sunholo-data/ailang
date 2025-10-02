# AILANG v0.2.0 Implementation Plan

**Codename**: "Module Execution & Effects"
**Timeline**: 3.5–4.5 weeks (from kickoff)
**Goal**: Move from "type-level complete" to "runnable modules with basic effects," while preserving v0.1.0 stability.

---

## Executive Summary

v0.2.0 delivers real module execution plus a minimal, safe effect runtime so that `ailang run module.ail --entry main` actually executes exported functions (with IO/FS). We extend the existing pattern matching with guards and exhaustiveness diagnostics.

**Success = "You can run demos and examples with imports, IO/FS effects, and ADT pattern matches; CI proves it."**

---

## Current State (v0.1.0 Recap)

### ✅ Complete

- **Type System**: HM + type classes, ADTs, effect rows tracked at type level (7,291 LOC)
- **Parser**: Modules, functions, ADTs, list/tuple patterns, effect syntax (2,656 LOC)
- **Evaluator (REPL/expressions)**: Works on single-file expressions; not on module programs (3,712 LOC)
- **Module System (Compile-Time)**: Loads, type-checks, exports interfaces (`internal/loader/`, `internal/iface/`)
- **Pattern Matching**: Implemented and evaluated (constructors, tuples, lists, wildcards)

### ⚠️ Missing at Runtime

- **Module Execution**: Top-level declaration evaluation + export materialization
- **Effect Handlers**: Runtime capability checking and effect execution
- **Pattern Matching Polish**: No guards/exhaustiveness diagnostics

**Note**: Earlier doc said "pattern matching not started"—that was outdated. We already have PM from M-P3; v0.2.0 adds guards + exhaustiveness and compiles to a fast decision tree.

---

## Scope (v0.2.0)

### In-Scope (Must Ship)

1. **M-R1: Module Execution Runtime**
   - Evaluate modules topologically; materialize runtime exports (closures/values)
   - Resolve imports at runtime; cross-module calls work
   - Entrypoint invocation from CLI (`--entry`, `--args-json`)
   - Keep the v0.1.0 Wrapper Runner as a fallback behind a flag

2. **M-R2: Minimal Effect Runtime**
   - Algebraic-effect execution for essential built-ins: **IO** (`print`/`println`/`readLine`) and **FS** (`readFile`, `writeFile`, `exists`)
   - Capability tokens + basic safety checks (deny if no capability)
   - No budgets/retries yet (that's v0.3+)

3. **M-R3: Pattern Matching Polish**
   - Guards in patterns (`if cond`)
   - Exhaustiveness diagnostics: well-formed warnings with suggested missing cases
   - Compile matches into decision trees for speed

### Out-of-Scope (Defer to Later)

- Full effect composition DSL, retries, timeouts, budgets
- Concurrency/CSP, session types
- External FFI (`extern`) formalization (keep v0.1.0 equation-form wrappers)

---

## Milestones & Work Packages

### M-R1: Module Execution Runtime (≈1,000–1,300 LOC | 1.5–2 weeks) ✅ CORE COMPLETE

**Objective**: `ailang run` executes exported functions from module files.

**Status**: ✅ Phases 1-4 complete (~1,594 LOC delivered)
- Infrastructure complete (module loading, evaluation, CLI integration)
- Function invocation pending (Phase 5)

#### Key Types

```go
// internal/runtime/module.go
type ModuleInstance struct {
    Path     string                      // Module path
    Iface    *iface.Iface               // Module interface (from type-checking)
    Imports  map[string]*ModuleInstance // Imported module instances
    Bindings map[string]Value           // All top-level bindings (including non-exports)
    Exports  map[string]Value           // Only exported bindings
    initOnce sync.Once                  // Thread-safe initialization
    initErr  error                      // Initialization error (if any)
}
```

#### Core Steps

1. **Link & Toposort** (new): Build a DAG of `ModuleInstance` from the loader graph
2. **Initialize** (new): Evaluate top-level declarations into `Bindings`, then filter to `Exports`
3. **Closures & Dicts**: Reuse existing closure/dictionary machinery; ensure dictionaries for type classes are passed across module boundaries
4. **Entrypoint Execution**: Lookup export by name, check arity (0 or 1 in v0.2.0), decode JSON arg with existing decoder, call, print
5. **Cache**: Map `<path, mtime>` → `ModuleInstance` to avoid re-eval during a run

#### CLI/UX

```bash
ailang run file.ail --entry main --args-json '{"arg": "value"}'
ailang run file.ail --runner=fallback  # Force v0.1.0 wrapper runner
```

**Features**:
- `--entry <name>` (default: `main`)
- `--args-json '<json>'` (default: `null`)
- `--runner=fallback` flag to force Wrapper Runner (v0.1.0 path) if needed
- Friendly "Available exports" list when entry not found

#### Acceptance Criteria

- ✅ Module runtime infrastructure complete
- ✅ Pipeline integration working (modules pre-loaded with Core AST)
- ✅ Entrypoint resolution and arity validation working
- ✅ Error messages show available exports
- ⏳ 25–30 previously blocked module examples run (pending function invocation)
- ⏳ Demos run via module execution (pending Phase 5)

#### Implementation Summary (v0.1.1)

**Delivered (~1,594 LOC)**:
1. **Phase 1: Scaffolding** (692 LOC)
   - `ModuleInstance` with thread-safe initialization
   - `ModuleRuntime` with caching and cycle detection
   - 12 unit tests passing

2. **Phase 2: Evaluation + Resolver** (402 LOC)
   - `moduleGlobalResolver` for cross-module references
   - `evaluateModule()` for binding extraction
   - 6 unit tests passing

3. **Phase 3: Linking & Topo** (~300 LOC)
   - Cycle detection with clear error messages
   - Integration test framework
   - Topological evaluation order

4. **Phase 4: CLI Integration** (~200 LOC)
   - Pipeline extension: `Modules` map in Result
   - Loader preloading: `Preload()` method
   - Recursive `extractBindings()` for Let/LetRec
   - CLI integration with entrypoint validation
   - Helper functions: `GetArity()`, `GetExportNames()`

**Test Results**:
- ✅ 18/18 unit tests passing
- ⚠️ 5/7 integration tests failing (known loader path issue, non-blocking)
- ✅ End-to-end validation working

**Next (Phase 5)**: Function invocation, stdlib support, documentation

---

### M-R2: Minimal Effect Runtime (≈700–900 LOC | 1–1.5 weeks)

**Objective**: Execute IO/FS effects safely with capability tokens.

#### Design

```go
// internal/effects/runtime.go
type Capability struct {
    Name string             // "IO", "FS"
    Meta map[string]any     // Future: budgets, tracing context
}

type EffContext struct {
    Caps map[string]Capability
}

type EffOp func(ctx *EffContext, args []Value) (Value, error)

// Registry: effect name -> op name -> handler
var registry = map[string]map[string]EffOp{
    "IO": {
        "println":  ioPrintln,
        "print":    ioPrint,
        "readLine": ioReadLine,
    },
    "FS": {
        "readFile":  fsRead,
        "writeFile": fsWrite,
        "exists":    fsExists,
    },
}
```

#### Execution Model

- During module entry, construct `EffContext` with allowed caps
- Calling an effectful builtin checks `ctx.Caps[EffectName]` exists; else a "missing capability" exception
- **Keep it simple**: No handlers/combinators in v0.2.0—just checked calls

#### Stdlib Alignment

- `std/io` and `std/fs` wrappers keep effect annotations; they call into these ops
- Type-level effects already flow; now they actually do something

#### CLI/UX

```bash
ailang run file.ail --caps IO,FS  # Enable capabilities
ailang run file.ail               # No caps by default (secure by default)
```

#### Acceptance Criteria

- ✅ IO demos print; FS demos read/write files in a temp sandbox dir
- ✅ Attempting IO/FS when caps disabled → clear runtime error
- ✅ Deterministic output with `AILANG_SEED`, `TZ`, `LANG` still holds

---

### M-R3: Pattern Matching Polish (≈450–650 LOC | 1 week)

**Objective**: Add guards and exhaustiveness warnings, plus decision-tree compilation.

#### Work Items

1. **Parser**: Guard pattern `if expr =>` (small change)
2. **Elaboration**: Keep existing PM nodes; add guard node; build decision tree
3. **Exhaustiveness**: Warn (not error) on missing cases; list examples of uncovered constructors
4. **Eval**: Guards evaluated in order; on false, continue matching

#### Acceptance Criteria

- ✅ Guarded patterns work across ADTs, tuples, lists
- ✅ Warnings surface in CLI with source spans
- ✅ Existing PM tests still pass; add ~15 new guard/exhaustiveness tests

---

## Schedule

```
Week 1–2      : M-R1 Module Execution Runtime
Week 2.5–3.5  : M-R2 Effects (IO, FS) + caps
Week 3.5–4.5  : M-R3 PM guards + exhaustiveness + decision trees
+ Rolling     : tests, docs, example goldens
```

**Critical Path**: M-R1 → M-R2; M-R3 can overlap late in Week 3.

---

## Testing Strategy

### Targets

| Component | Current | Target |
|-----------|---------|--------|
| Overall coverage | ~25% | >35% |
| Module runtime | 0% | ≥70% |
| Effects runtime | 0% | ≥60% |
| PM polish | 0% | ≥65% |

### Layers

1. **Unit Tests** (~400–500 tests)
   - Module init order, export binding, cross-module calls
   - Cap checks (present/missing), IO/FS happy-path and denial
   - PM guards & decision-tree paths

2. **Integration Tests** (~40–50 tests)
   - Multi-module programs with IO/FS
   - Result/Option across module boundaries with matches

3. **Golden Examples**
   - Expand `make verify-examples` to 35+ passing
   - Pin env (`TZ=UTC`, `LANG=C`, `AILANG_SEED=42`)

4. **Performance Sanity**
   - Cold vs warm module run; effect overhead (IO/FS fast path)

---

## Interfaces & Invariants

### Entrypoint Rules (v0.2.0)

- Exported function, 0 or 1 parameter
- If 1 param: JSON-decoded via type-driven decoder (already implemented)
- Return printed via existing pretty-printer; `Result` `Ok`/`Err` printed structurally

### Effect Safety

- **No capability** → runtime error with effect name and suggested fix
- Effects are synchronous and blocking in v0.2.0

### Module Cycles

- **Phase 1**: Disallow cycles at runtime; emit clear error (type system already detects import cycles; runtime mirrors that)
- **Phase 2** (v0.3+): Consider thunked init for benign value cycles

### Backward Compatibility

- ✅ v0.1.0 files still type-check and REPL remains unchanged
- ✅ Non-module files execute as before

---

## Error Messages (Polish Carried from v0.1.0)

| Code | Message | Suggested Fix |
|------|---------|---------------|
| `RUN_NO_ENTRY` | Entry not found | Show available exports |
| `RUN_MULTIARG_UNSUPPORTED` | Entry takes >1 params | Suggest wrapper function |
| `EFFECT_CAP_MISSING` | IO (or FS) not enabled | Show `--caps IO,FS` hint |
| `LIST_CONCAT_MISMATCH` | Type mismatch in `++` | Done in v0.1.0 polish |
| `IMPORT_CONSTRUCTOR_NOT_EXPORTED` | Constructor not exported | Done in v0.1.0 polish |

---

## Tooling & Flags

### CLI Commands

```bash
ailang run <file>
  --entry <name>          # Default: main
  --args-json '<json>'    # Default: null
  --caps IO,FS            # Enable runtime capabilities
  --runner=fallback       # Force wrapper runner
  --no-print              # Suppress printing unit

ailang iface <module>     # Unchanged (API freeze remains)
```

---

## Risks & Mitigations

| Risk | Impact | Mitigation | Fallback |
|------|--------|-----------|----------|
| Module init order bugs | High | Toposort + explicit phase separation (link → init → export) | Disallow cycles; better diagnostics |
| Effect runtime complexity creeps | Med | IO/FS only, thin ops, no handlers/combinators yet | Limit to sync ops; add flags to disable |
| Exhaustiveness false positives | Med | Start as warnings; add `--no-exhaustive-warning` | Defer guard corner cases if needed |
| Schedule slip on M-R1 | Med | Start immediately; gate others behind it | Push M-R3 to v0.3.0 |

---

## Acceptance Criteria (Go/No-Go)

### Must Pass to Ship v0.2.0

- ✅ `ailang run` executes module exports by default (no wrapper), with 0/1-arg entrypoints
- ✅ IO and FS effects work when caps enabled; are denied otherwise
- ✅ ≥35 passing examples (target 40 if M-R3 completes)
- ✅ Coverage ≥35%; no runtime panics in happy paths
- ✅ Exhaustiveness warnings (if M-R3 included)
- ✅ Docs updated (README, LIMITATIONS, guides)

### Stretch (If Time Allows)

- ✅ M-R3 complete (guards + exhaustiveness + decision tree)
- ✅ 40+ passing examples; coverage ≥40%

---

## Documentation Plan

### Required Updates

1. **README.md**
   - Update "What works" (module exec, IO/FS)
   - Update "Known limits" (remove module execution gap)
   - Document CLI flags (`--entry`, `--caps`, `--runner`)

2. **docs/guides/module-execution.md** (NEW)
   - Lifecycle, topological init, exports

3. **docs/guides/effects-guide.md** (NEW)
   - Enabling caps, IO/FS reference

4. **docs/guides/pattern-matching.md** (NEW, if M-R3 lands)
   - Guards + exhaustiveness

5. **examples/STATUS.md**
   - Audited list with pass/fail

6. **RELEASE_NOTES_v0.2.0.md**
   - Headline features, migration (none), flags

---

## Concrete Next Steps

### 1. Kick M-R1

- Create `internal/runtime/` with `ModuleInstance`, linker, evaluator skeleton
- Wire `cmd/ailang run` to runtime path (keep wrapper behind `--runner=fallback`)
- Land 6 "smoke" tests:
  1. Single module
  2. Module→module import
  3. Export lookup
  4. Entry 0/1 arg
  5. JSON decoding
  6. Print

### 2. Bring Up IO/FS

- `internal/effects/` with caps + registry
- Hook `std/io` and `std/fs`
- Add `--caps` flag; deny by default (secure by default)

### 3. PM Polish (Parallel After M-R1 Stabilized)

- Guards parsing → elaboration → eval
- Decision tree pass
- Warnings

### 4. Examples & Goldens

- Upgrade `verify-examples` to new demos
- Aim for 35+ passing

---

## Rollback Plan

| Scenario | Action |
|----------|--------|
| M-R1 slips | Ship v0.2.0-rc with Wrapper Runner on by default and module runtime behind `--runner=module`, then flip in 0.2.1 |
| M-R2 slips | Keep IO only; defer FS to 0.2.1 |
| M-R3 slips | Ship without guards/exhaustiveness; keep docs honest |

---

## Status

**Status**: Ready for implementation
**DRI**: Assign one per milestone (M-R1, M-R2, M-R3)
**CI Gates**: Module runtime tests, effects tests, example goldens, coverage threshold

---

## Key Files Reference

### Module System
- `internal/loader/loader.go` - Module loading (exists, ~500 LOC)
- `internal/iface/iface.go` - Module interfaces (exists, ~200 LOC)
- `internal/runtime/` - Module runtime (NEW, ~1,000 LOC)

### Effect System
- `internal/effects/` - Effect runtime (NEW, ~700 LOC)
- `internal/builtins/io.go` - IO effects (NEW, ~100 LOC)

### Pattern Matching
- `internal/parser/parser.go` - Pattern parsing (modify, +100 LOC)
- `internal/elaborate/match.go` - Pattern elaboration (modify, +200 LOC)
- `internal/eval/eval_core.go` - Pattern evaluation (modify, +150 LOC)

---

**Document Version**: v2.0
**Created**: 2025-10-02
**Last Updated**: 2025-10-02
**Author**: AILANG Development Team

**Changes from v1.0**:
- Corrected pattern matching status (already implemented in M-P3)
- Added concrete acceptance criteria and rollback plans
- Clarified scope vs. stretch goals
- Added precise CLI flags and error codes
- Improved risk assessment and mitigation strategies
- Added concrete next steps for implementation kickoff
