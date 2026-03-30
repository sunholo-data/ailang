# M-PKG-INTERREF: Package Inter-Function References Fail When Loaded as Dependency

**Status**: Planned
**Target**: v0.9.13
**Priority**: P0 (High — blocks all packages with helper functions)
**Estimated**: 1 day
**Dependencies**: None
**Milestone ID**: M-PKG-INTERREF
**Created**: 2026-03-26
**Source**: Agent message `01822e8e` (sunholo/demos — gemini_live package)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes nondeterministic behavior: same function works standalone, fails as dependency |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | `check --package` should catch what fails at consumer load time |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Eliminates a class of silent failures that confuse AI agents writing packages |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +2 | Fundamental — exported functions must work when composed cross-package |
| A11: Structured Failure | +1 | Currently fails with opaque "undefined variable" at consumer site; fix makes packages just work |
| A12: System Boundary | 0 | No change |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): This is a determinism FIX — same function should behave identically regardless of call site
- [x] A3 (Effects): No effect changes
- [x] A4 (Authority): No capability changes
- [x] A7 (Machines First): This fix directly improves machine reliability

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **third instance of the "resolver context leak" bug class**:

| Instance | Level | Fixed? | Design Doc |
|----------|-------|--------|------------|
| Cross-package VarGlobal resolution | Value (global refs) | ✅ v0.9.5 | `m-dx-cross-package-stdlib-resolution.md` |
| Cross-package type alias unification | Type (TCon vs TRecord) | ⏳ Planned | `m-type-cross-package-alias-unification.md` |
| **Same-module core.Var in on-demand eval** | **Value (local refs)** | **❌ This bug** | **This doc** |

The M-DX-XPKG-RESOLVE fix (v0.9.5) added `FallbackResolver` to handle `VarGlobal` lookups that cross module boundaries. But the current bug is about `core.Var` — local variable references within the same module that break when the resolver evaluates Let bindings independently.

---

## Problem Statement

### The Bug

When a package module exports functions that reference other functions in the same file, `ailang check --package .` passes but consumers importing via `pkg/` get "undefined variable" errors.

```ailang
-- In package: sunholo/gemini_live/voices.ail
module sunholo/gemini_live/voices

-- Helper function (not exported)
pure func v(name: string, lang: string) -> Voice
  Voice(name, lang)

-- Exported function calls helper
export pure func voiceCatalog() -> List[Voice]
  [v("Puck", "en"), v("Charon", "en"), v("Kore", "en")]
```

**`ailang check --package .`** → ✅ passes (syntax valid, types check out)
**Consumer: `import pkg/sunholo/gemini_live/voices (voiceCatalog)`** → ❌ `undefined variable: v`

### Four Reported Cases (from sunholo/gemini_live)

1. **voices.ail**: `voiceCatalog()` → `v()` — helper function
2. **endpoints.ail**: `aiStudioUrl()` → `googleAiUrl()` — helper function
3. **parsers.ail**: `parseMessage()` → `parseToolCall()`/`parseServerContent()`/etc. — 6 inter-referencing functions
4. **messages.ail**: functions using `concat` from `std/list` — stdlib import

Cases 1-3 are the same root cause (local Var not in evaluator env). Case 4 may be different (VarGlobal resolution of stdlib within on-demand-evaluated module).

### Root Cause

In `internal/link/resolver.go:112-132`, the resolver lazily evaluates a dependency module's Core program when first referenced:

```go
for _, decl := range coreProgram.Decls {
    switch d := decl.(type) {
    case *core.LetRec:
        bindings, err := evaluator.EvalLetRecBindings(d)
        for name, val := range bindings {
            r.memo[ref.Module][name] = val  // ✅ LetRec handles mutual refs
        }
    case *core.Let:
        val, err := evaluator.Eval(d.Value)
        r.memo[ref.Module][d.Name] = val    // ❌ NOT bound in evaluator env
    }
}
```

Each `Let` value is evaluated and stored in `r.memo`, but **never bound in the evaluator's environment**. When function B's lambda is created, it captures the evaluator's current env. If function A was defined earlier as a `Let` but not bound in the env, B's closure body will fail with "undefined variable: A" when called.

**LetRec is fine** — `EvalLetRecBindings` uses a shared recursive environment with IndirectValue cells. But non-mutually-recursive functions (the common case) are compiled as sequential `Let` bindings, and those break.

### Why `check --package` Misses It

`check --package` (in `cmd/ailang/check_package.go`) runs each file independently through `pipeline.Run()`. This performs elaboration and type checking, which both succeed because:
- The elaborator's SCC analysis correctly groups functions
- Type checking sees all module-level bindings in the symbol table
- **No evaluation happens** — the resolver bug is only triggered at eval time

The check never simulates the on-demand evaluation path that consumers trigger.

---

## Goals

**Primary Goal:** Package-exported functions that reference same-module helpers must work when loaded as dependencies.

**Success Metrics:**
- All 4 reported cases from sunholo/gemini_live work without workarounds
- Existing cross-package tests continue passing
- `check --package` catches the bug class (or at minimum, doesn't give false confidence)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Accumulate Let bindings in evaluator env during on-demand module eval | Core fix — changes how all dependency modules are evaluated | agent | design | low |
| Whether to also accumulate LetRec bindings in evaluator env | LetRec bindings may need to be visible to subsequent Let decls | agent | compile | low |
| Whether to enhance `check --package` to catch this class | DX improvement — changes validation scope | human | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Accumulate Let bindings in evaluator env (confirmed — this is the fix)
- [ ] Whether `check --package` enhancement is in-scope or deferred

## Solution Design

### Overview

Two-part fix:
1. **resolver.go**: After evaluating each Let/LetRec decl, bind results in the evaluator's local environment so subsequent declarations can reference them
2. **check_package.go** (optional): Add a "link simulation" phase that evaluates module declarations to catch this class of bug

### Architecture

**The fix is surgical — 3 lines in resolver.go.**

Currently, the resolver evaluates `d.Value` and stores the result in `r.memo` but not in the evaluator's environment. The evaluator uses `e.env.Get(name)` for `core.Var` lookups. Since `r.memo` and `e.env` are separate, same-module references fail.

**Fix**: After each Let/LetRec evaluation, also bind the results in the evaluator's env:

```go
case *core.Let:
    val, err := evaluator.Eval(d.Value)
    if err != nil { ... }
    r.memo[ref.Module][d.Name] = val
    evaluator.Env().Set(d.Name, val)  // FIX: accumulate in evaluator env

case *core.LetRec:
    bindings, err := evaluator.EvalLetRecBindings(d)
    if err != nil { ... }
    for name, val := range bindings {
        r.memo[ref.Module][name] = val
        evaluator.Env().Set(name, val)  // FIX: accumulate LetRec bindings too
    }
```

This ensures that when function N is being evaluated, functions 1..N-1 are already in scope.

**Components:**
1. **resolver.go env accumulation** — The core fix (~5 LOC)
2. **Env() accessor on CoreEvaluator** — May already exist; if not, trivial to add (~3 LOC)
3. **Regression test** — Package with sequential non-recursive helper functions (~40 LOC)

### Implementation Plan

**Phase 1: Core Fix** (~1 hour)
- [ ] Add `Env()` accessor to CoreEvaluator if not present
- [ ] Add `evaluator.Env().Set(d.Name, val)` after Let evaluation in resolver.go
- [ ] Add `evaluator.Env().Set(name, val)` after LetRec bindings in resolver.go
- [ ] Run `make test`

**Phase 2: Regression Tests** (~1 hour)
- [ ] Create test package with inter-function references (helper calling helper)
- [ ] Create test package with stdlib imports (e.g., std/list concat)
- [ ] Create test consumer that imports from test package
- [ ] Add integration test that runs the consumer and verifies output
- [ ] Run `make test` and `make verify-examples`

**Phase 3: DX Enhancement (Optional)** (~2 hours)
- [ ] Add link-simulation phase to `check --package` that evaluates declarations
- [ ] Verify it catches the "undefined variable" case
- [ ] Run `make test`

### Files to Modify/Create

**Modified files:**
- `internal/link/resolver.go` — Add env accumulation after Let/LetRec eval (~5 LOC)
- `internal/eval/eval_evaluator.go` — Add `Env()` accessor if missing (~3 LOC)

**New files:**
- `tests/runtime_integration/cross_pkg_interref/main.ail` — Consumer test (~15 LOC)
- `tests/runtime_integration/cross_pkg_interref/pkg/testhelpers/*.ail` — Test package (~20 LOC)

## Examples

### Example 1: Helper Function (Cases 1-2)

**Before (workaround — inline everything):**
```ailang
-- voices.ail — forced to inline helper
export pure func voiceCatalog() -> List[Voice]
  [Voice("Puck", "en"), Voice("Charon", "en"), Voice("Kore", "en")]
```

**After (natural style works):**
```ailang
-- voices.ail — helper functions work as expected
pure func v(name: string, lang: string) -> Voice
  Voice(name, lang)

export pure func voiceCatalog() -> List[Voice]
  [v("Puck", "en"), v("Charon", "en"), v("Kore", "en")]
```

### Example 2: Multiple Inter-Referencing Functions (Case 3)

**Before:** No clean workaround — 6 functions calling each other can't all be inlined.

**After:**
```ailang
-- parsers.ail — all inter-references resolve correctly
pure func parseToolCall(msg: Json) -> ParsedMessage { ... }
pure func parseServerContent(msg: Json) -> ParsedMessage { ... }
pure func checkOutputTranscript(content: Json) -> ParsedMessage { ... }
pure func checkModelTurn(content: Json) -> ParsedMessage { ... }

export pure func parseMessage(msg: Json) -> ParsedMessage
  -- Can freely call all helpers above
  match getType(msg) {
    "toolCall" => parseToolCall(msg),
    "serverContent" => parseServerContent(msg),
    _ => UnknownMessage
  }
```

### Example 3: Stdlib Imports (Case 4)

**Before:** `undefined variable: concat` when consumer loads the package.

**After:**
```ailang
-- messages.ail
import std/list (concat)

export pure func buildPrompt(parts: List[List[string]]) -> List[string]
  concat(parts)  -- stdlib reference works cross-package
```

## Success Criteria

- [ ] Package with sequential Let-bound helpers works when loaded as dependency
- [ ] Package with LetRec-bound (mutually recursive) helpers works
- [ ] Package with stdlib imports works when loaded as dependency
- [ ] Existing cross-package tests pass (`make test`)
- [ ] `make verify-examples` passes
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated
- [ ] Example file added demonstrating the fix

## Testing Strategy

**Unit tests:**
- Test resolver evaluates module with 3 sequential Let bindings where f3 calls f2 calls f1
- Test resolver evaluates module with LetRec followed by Let that references a LetRec binding

**Integration tests:**
- Test package with helper functions imported by consumer
- Test package with stdlib imports imported by consumer
- Reuse existing `cross_pkg_*` test pattern from `tests/runtime_integration/`

**Manual testing:**
- If sunholo/gemini_live package is available, verify all 4 reported cases

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Test package structure** — agent may choose directory layout and function names
- **Whether to use `NewChildEnvironment()` or direct `Set()` on evaluator env** — agent may choose based on evaluator internals (direct Set is simpler if env is mutable)
- **Error message improvement** — if "undefined variable" can be made more specific (e.g., "undefined variable: v (hint: same-module references require...)"), agent may add but not required

## Non-Goals

**Not attempted in this feature:**
- **`check --package` full link simulation** — May be deferred to a separate PR if the core fix is sufficient. The DX suggestion from the reporter is valid but the priority is fixing the runtime failure first.
- **Resolver architecture overhaul** — The FallbackResolver pattern from M-DX-XPKG-RESOLVE is working; this fix is additive, not a redesign.
- **Cross-module (multi-file) inter-references within a package** — This doc covers same-file inter-function references. Cross-file references within a package may have separate issues.

## Timeline

**Day 1** (~4 hours):
- Phase 1: Core fix (1h)
- Phase 2: Regression tests (1h)
- Phase 3: DX enhancement if time permits (2h)
- CI verification

**Total: ~4 hours, 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Accumulating env bindings changes eval semantics for all modules | Med | Only affects on-demand module eval in resolver, not root module eval. Existing tests validate. |
| LetRec bindings accumulated in env may shadow/conflict with subsequent Let bindings | Low | LetRec and Let names are unique within a module (parser enforces). No conflict possible. |
| `Env()` accessor doesn't exist on CoreEvaluator | Low | Trivial to add; or use existing method if available. |
| Fix may mask other resolver bugs by making more things "just work" | Low | Good — that's the intended behavior. Additional tests will catch regressions. |

## Related Documents

<!-- Auto-populated by Ollama neural search on "pkg interref fix" -->

**Directly related (same bug class):**
- [design_docs/planned/v0_9_5/m-dx-cross-package-stdlib-resolution.md](design_docs/planned/v0_9_5/m-dx-cross-package-stdlib-resolution.md) — VarGlobal resolution fix (FallbackResolver). This doc fixes the Var (local ref) counterpart.
- [design_docs/planned/m-type-cross-package-alias-unification.md](design_docs/planned/m-type-cross-package-alias-unification.md) — Type-level manifestation of resolver context leak.

**Package system context:**
- [design_docs/planned/v0_10_0/m-pkg-resolver-direct-wins.md](design_docs/planned/v0_10_0/m-pkg-resolver-direct-wins.md) — Package version resolution (different resolver, same area)
- [design_docs/implemented/v0_9_11/m-dx-app-package-adoption.md](design_docs/implemented/v0_9_11/m-dx-app-package-adoption.md) — Package adoption DX
- [design_docs/implemented/v0_9_9/m-pkg-msg-sprint-plan.md](design_docs/implemented/v0_9_9/m-pkg-msg-sprint-plan.md) — Package messaging system

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Agent message `01822e8e` — Original bug report from sunholo/demos
- `internal/link/resolver.go:112-132` — Bug location
- `internal/eval/eval_expressions.go:86-89` — "undefined variable" error source
- `internal/elaborate/file.go:133-260` — How SCC analysis compiles functions to Let/LetRec

## Future Work

- **Unified resolver context model** — All three instances of this bug class (VarGlobal, Var, Type) suggest the resolver needs a first-class "module evaluation context" that carries both global refs AND local bindings. Currently these are split across FallbackResolver (global) and evaluator env (local).
- **`check --package` link simulation** — If deferred from this fix, should be tracked as a separate issue to prevent false confidence from `check --package` passing.

---

**Document created**: 2026-03-26
**Last updated**: 2026-03-26
