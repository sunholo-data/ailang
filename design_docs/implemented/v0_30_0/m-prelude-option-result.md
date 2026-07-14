# M-PRELUDE-OPTION-RESULT: Add Option/Result to the prelude (structural AI-DX fix)

**Status**: Implemented (2026-07-14, mission iteration 27)
**Target**: v0.30.0 (doc originally said v0.24.0; retargeted — see sprint plan)
**Priority**: P1 (High — structural fix; permanent, no re-teaching per model)
**Estimated**: 1.5 days
**Dependencies**: None

> **AS-BUILT NOTE (2026-07-14).** The sprint plan
> (`m-prelude-option-result-sprint-plan.md`) CORRECTED this doc's mechanism, and
> the implementation follows the plan, not the original "Architecture" /
> "Implementation Plan" sections below:
>
> - **NOT `internal/pipeline/prelude.go` + `InjectPrelude` / `InjectPreludeValues`.**
>   `InjectPreludeValues` never existed (it was a TODO comment), and the prelude
>   only ever injected the `println` type — there was no type/constructor
>   injection path there to extend.
> - **As built:** entry modules **implicitly import `std/option` + `std/result`**,
>   injected at the **loader** (`internal/loader/prelude_imports.go`, wired into
>   `ModuleLoader.Load`). That is the single call-site both the compile pipeline
>   (`mod.File.Imports`) and the runtime (`LoadedModule.Imports`) consume, so
>   compile and runtime stay in lockstep with **zero** changes to the type
>   checker, elaborator, or runtime resolver core. The injection reuses the
>   existing constructor-import machinery (selective import → `M-CTOR-AUTO` →
>   `resolveConstructorImport` → `findConstructorMatches`).
> - **Shadowing** is handled by (a) prepending implicit imports = lowest
>   precedence, (b) deduping against the user's explicit imports, and (c) skipping
>   a prelude module entirely when the entry file locally declares a colliding
>   type or constructor name (`type Option` / `type Result` / a local `Some`/`Ok`).
> - **No `cacheKeyVersion` bump** (stays `v3`): no persisted `Iface`/gob struct
>   field changed; entry-module cache keys shift naturally via their dep set.
> - Example: `examples/prelude_option_result.ail`. Tests:
>   `internal/loader/prelude_imports_test.go`,
>   `internal/pipeline/prelude_option_result_test.go`,
>   `internal/runtime/prelude_option_result_test.go`.
>
> The "Solution Design", "Architecture", and "Implementation Plan" sections below
> are preserved as the ORIGINAL proposal for historical context; trust the plan +
> this note for what actually shipped.

> **📊 RECENT-VERIFIED: 6% of recent (Apr-Jun 2026) compile failures are `None`/`Some`
> used without `import std/option`. This is a STRUCTURAL fix — the model's code is
> *correct* AILANG, it just lacks an import. Making Option/Result prelude-available
> removes the friction permanently rather than teaching every new model to import.**

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact (pure types) |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | Option/Result are pure — no effect hiding; effects stay in signatures |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification change; types resolve as before |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Removes the most common 'forgot the import' failure for AI codegen |
| A8: Minimal Syntax | +1 | Less boilerplate for the most-used types; no new syntax |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | Option/Result compose with all existing stdlib that returns them |
| A11: Structured Failure | +1 | Makes Result (the structured-failure type) frictionless to use |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Proceed** (no −1 on A1/A3/A4/A7)

---

## Prior Art (verified 2026-06-03, design-doc-creator workflow)

**The prelude mechanism is deliberate and its philosophy explicitly endorses this change.**
`internal/pipeline/prelude.go` documents:

> *"AI-First Design Philosophy: Minimize syntactic entropy — teach the compiler to carry
> context so the AI doesn't have to. The prelude removes boilerplate (e.g.
> `import std/io (println)`) while preserving AILANG's core principle: effects remain
> explicit in type signatures."*

So extending the prelude to remove the `import std/option` boilerplate is a **direct
continuation of the stated design intent**, not a contradiction of a "keep it minimal"
decision. Git history confirms it's an intentional 2-phase mechanism (commit 8ded3a60
"Phase 2: Entry-module prelude implementation"), not a hack.

**⚠️ The important nuance (caught by reading the prelude source):** the prelude has only
ever injected `println` — a **function** binding. `Option`/`Result` are **types +
data constructors**, a category the prelude has never carried. This doc therefore proposes
the FIRST type/constructor injection into the prelude. That is a genuine new capability,
not a copy of the println path — the implementation plan below reflects that (constructor
registry + match-pattern resolution, not just a type-env binding). This raised the estimate
and is the main implementation risk.

**Verified claim (`ailang check`, 2026-06-03):** `Option`/`Some`/`None` and `Result`/`Ok`/`Err`
DO require explicit import today — `pure func f(x:int) -> Option[int] = if x>0 then Some(x) else None`
fails with `undefined variable: Some` without `import std/option`. The problem is real.

## Problem Statement

Models write correct AILANG that uses `Option`, `Some`, `None`, `Result`, `Ok`, `Err`
— but forget `import std/option` / `import std/result`. The result:
```
Error: type error in benchmark/solution (decl 0): undefined variable: Some
       at benchmark/solution.ail:2:52
```

This is **6% of recent compile failures** — and unlike syntax the model gets *wrong*,
here the model's code is **idiomatically correct**. In Haskell, Rust, Swift, OCaml,
F#, the equivalents (`Maybe`/`Option`/`Result`) are in the **standard prelude** —
always in scope, no import. Models reasonably assume the same for AILANG.

**This is a structural gap, not a model error.** The right fix is to make AILANG match
the universal expectation: Option and Result available without import.

### Why structural beats prompt here

| Approach | Cost | Durability |
|---|---|---|
| Prompt fix (`m-prompt-option-none-idiom`) | Cheap | Must re-teach every new model; μRAG must surface it; still fails on cold prompts |
| **Structural (prelude)** | 1.5 days once | **Permanent.** Every model, every prompt version, forever. Also helps humans. |

The prompt fix is a band-aid; the prelude fix removes the wound.

---

## Goals

**Primary goal:** `Option`, `Some`, `None`, `Result`, `Ok`, `Err` resolve without import in entry modules.

**Success metrics:**
- `option_no_import` compile failures drop from 6% to ~0%
- `import std/option`/`import std/result` become optional (still allowed, just not required)
- No regression: explicit imports still work; user shadowing still works
- CPR improvement ~+3pts for mid-tier models

---

## Solution Design

### Current state

`internal/pipeline/prelude.go::InjectPrelude` already injects `println` (type-level)
into entry modules. It currently injects only ONE function type. Value bindings are
handled separately. The mechanism is proven — this extends it.

### What needs injecting (harder than println — these are TYPES + CONSTRUCTORS)

`Option`/`Result` are ADTs, not functions. Three things must be in scope:
1. **The type constructors** `Option[a]`, `Result[a, e]` — for type annotations
2. **The data constructors** `Some`, `None`, `Ok`, `Err` — for construction
3. **Pattern-match arms** using those constructors — must resolve in `match`

### Architecture

1. **Type env** (`InjectPrelude`): add `Option`/`Result` type constructors + the data
   constructor schemes (`Some : a -> Option[a]`, `None : Option[a]`, `Ok : a -> Result[a,e]`,
   `Err : e -> Result[a,e]`).
2. **Constructor registry**: the ADT constructor resolution path (used by both
   construction and match patterns) must know these constructors for entry modules.
   Check how `std/option`'s `export type Option` registers — replicate that for the prelude.
3. **Evaluator** (`InjectPreludeValues`): inject the runtime constructor values.
4. **Shadowing**: if the user writes `import std/option (Option, Some, None)` OR defines
   their own `type Option`, that must shadow the prelude cleanly (existing prelude
   shadowing semantics already handle the function case — verify for types).

### Implementation Plan

**Phase 1: Type-level injection** (~0.5 day)
- [ ] Extend `InjectPrelude` to add Option/Result type cons + data cons schemes
- [ ] Verify `pure func f(x:int) -> Option[int] = if x>0 then Some(x) else None` type-checks with no import

**Phase 2: Constructor + match resolution** (~0.5 day)
- [ ] Ensure `Some`/`None`/`Ok`/`Err` resolve in match patterns without import
- [ ] Wire the constructor registry for entry modules

**Phase 3: Evaluator + tests** (~0.5 day)
- [ ] `InjectPreludeValues` for the constructors (runtime)
- [ ] Tests: construct, match, shadow-by-import, shadow-by-local-def, non-entry module (NOT injected)

### Files to Modify

| File | Change | LOC |
|---|---|---|
| `internal/pipeline/prelude.go` | Inject Option/Result types + constructors | +60 |
| `internal/eval/*.go` (InjectPreludeValues) | Runtime constructor bindings | +40 |
| `internal/pipeline/prelude_test.go` | Construct/match/shadow tests | +80 |
| `prompts/v0.17.0.md` | Update: "Option/Result are prelude — no import needed" | +4 |

---

## Conflict Surface

**This touches `internal/pipeline/` + `internal/eval/` — conflict surface required.**

1. **What positions does this extend?** Identifier resolution in entry modules — bare
   `Some`/`None`/`Ok`/`Err`/`Option`/`Result` now resolve to prelude constructors.
2. **What else lives there?** User code that already imports std/option, AND user code
   that defines its OWN `type Option` or `type Result` (some benchmarks/programs do).
3. **Disambiguation:** Prelude is the LOWEST precedence. User imports and user local
   definitions both shadow it (existing prelude shadowing semantics — `println` already
   works this way). MUST verify shadowing works for TYPES, not just functions.
4. **Programs that MUST still work (fixtures):**
   - `import std/option (Option, Some, None)` then use them — explicit import unchanged
   - User defines `type Option[a] = Some(a) | None` locally — local shadows prelude, no conflict
   - User defines `type Result = Pending | Done` (DIFFERENT Result!) — local shadows, no clash
   - A non-entry (library) module using Option WITHOUT import — should it inject? Decide: prelude is **entry-module only** (matches current println behaviour), so library modules still need explicit imports. This keeps library code self-documenting.
   - `match opt { Some(x) => x, None => 0 }` with no import — must resolve
5. **Intentional change:** entry modules no longer require `import std/option`/`std/result`.

**Key risk:** a benchmark/program that defines a *custom* `Result` with different
constructors. The prelude must lose to local definitions. This is the same shadowing
the prelude already guarantees for `println`, but TYPE shadowing must be explicitly tested.

---

## Non-Goals

- Auto-importing arbitrary stdlib modules (only Option/Result — the near-universal ones)
- Injecting Option/Result into LIBRARY modules (entry-only, like println — keeps libs explicit)
- Adding Option/Result *helper functions* (map, filter, etc.) to the prelude — those still
  need `import std/option (map)`. Only the type + constructors are prelude.

---

## Related Documents

- `design_docs/planned/v0_24_0/m-prompt-option-none-idiom.md` — the PROMPT band-aid this
  structural fix supersedes (keep the prompt fix as a stopgap until this ships)
- `internal/pipeline/prelude.go` — existing prelude mechanism (injects println) this extends
- Future companion: `m-prelude-expansion` could later add other near-universal items
