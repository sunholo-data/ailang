# M-EFFECT-HANDLERS: User-Definable Effect Handlers

**Status**: Planned
**Target**: v0.21.0 (Phase 1) → v0.22.0 / v1.0.0 for full surface
**Priority**: P1 — Medium (strategic language feature, unblocks deterministic testing story)
**Estimated**: ~80–120 hours across 2–3 sprints (Phase 1 alone: ~30–40h)
**Dependencies**:
  - [M-R2 Effect Runtime](../../implemented/v0_2_0/m_r2_effect_system.md) (v0.2.0 ✅) — capability runtime
  - [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) (v0.6.2 ✅) — budget plumbing
  - [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) — parameterised effects (related, not blocking)

**Commissioning context**: Comparison with [Koka](https://koka-lang.github.io/koka/doc/index.html) (May 2026). AILANG's effect system already cites Leijen 2014 (`docs/docs/reference/effects.md:13`), but ships only **built-in** effects (`IO`, `FS`, `Net`, `AI`, `Clock`, etc.). Koka's killer feature — and the one piece we don't have — is **user-definable handlers** that turn effects from "tracked" into "programmable."

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Handler-based mocking replaces non-deterministic real effects in tests |
| A2: Replayability | +1 | Handlers can record/replay effect operations as typed values, not opaque traces |
| A3: Effect Legibility | +1 | Handlers are explicit syntactic constructs (`handle e with H`), not hidden middleware |
| A4: Explicit Authority | 0 | Handlers run inside the existing capability sandbox; no new authority surface |
| A5: Bounded Verification | +1 | Each handler is locally type-checked against the effect signature it discharges |
| A6: Safe Concurrency | 0 | One-shot handlers only (Phase 1); multi-shot deferred to align with Fork/Call model |
| A7: Machines First | +1 | LLM agents can write deterministic test doubles in AILANG instead of mocking in Go |
| A8: Minimal Syntax | −0 | Adds `effect` declaration + `handle ... with ...` form — confined, but new surface |
| A9: Cost Visibility | +1 | Handler discharge is visible in the type (effect leaves the row when handled) |
| A10: Composability | +1 | Handler stacks compose naturally; same algebra as Koka/Eff |
| A11: Structured Failure | +1 | Unhandled effects become typed errors at elaboration time |
| A12: System Boundary | +1 | Makes "where the system boundary is" syntactic — handlers ARE the boundary |

**Net Score: +9** → **Decision: Proceed to implementation (Phase 1)**

### Hard Violation Check

- [x] A1: Handlers strengthen determinism (mockable effects); do not introduce ambient state.
- [x] A3: Effect handling is syntactic and visible at the type level.
- [x] A4: Handlers cannot grant capabilities they don't already hold — discharge ≠ authority.
- [x] A7: Designed primarily for machine-written test doubles, not human ergonomics.

---

## Problem Statement

### What AILANG has today (v0.20.0)

AILANG's effect system is **closed at the language level**:

```ailang
-- Built-in: works
func greet() -> () ! {IO} = println("hi")

-- User-defined: NOT possible today
effect Yield[a] {
  yield: a -> ()
}
```

The runtime maps each effect name (`IO`, `FS`, `Net`, `AI`, `Clock`, `Brain`, `AbsPath`) to a hard-coded Go handler in `internal/effects/`. There is no way for `.ail` code to introduce a new effect, and no way to handle an existing effect in user code.

### Consequences

**1. Testing is awkward.** To mock `AI` for a benchmark, you must either:
   - Set `AILANG_AI_PROVIDER=mock` (env-var coupling, not type-checked)
   - Edit Go code in `internal/effects/ai.go`
   - Write a fake provider plugin

   None of these are reachable from inside an `.ail` test file.

**2. No domain effects.** Patterns that fit algebraic effects naturally — generators (`yield`), parsers (`commit`/`backtrack`), state machines, structured logging contexts, dependency injection — must be encoded as either:
   - Result/Option-returning functions (loses the "tracked side effect" property)
   - Built-in effect requests to the AILANG core team

**3. Eval harness can't ship deterministic test doubles.** The benchmark suite would benefit enormously from `handle AI with recorded_responses` — record a real run once, replay deterministically forever. Today this lives in Go, gated by env vars.

**4. We cite Leijen 2014 for the type system but stop at the row algebra.** The 2014 paper is half about handlers. We've shipped half the theory.

### Impact

- **Affects:** All `.ail` authors who want testable code; the eval harness; anyone trying to express control-flow patterns (iterators, parsers) idiomatically.
- **Severity:** Medium — there are workarounds for each individual case, but the cumulative ergonomic + architectural tax is significant and grows with each new built-in effect.

---

## Goals

**Primary Goal:** Let AILANG programs declare new effects and define handlers that discharge them, with full type-system integration (effect row subtraction on handle).

**Success Metrics:**
- A user can write `effect E { op : T -> U }` and `handle expr with H` in pure AILANG.
- The eval harness can replace `AI` with a user-defined `handle AI with recorded` and pass all benchmarks deterministically — without touching Go.
- Handler discharge is reflected in the type: `(() -> a ! {E, IO}) → (() -> a ! {IO})` after handling `E`.
- Phase 1 supports at least three reference effects in stdlib: `Yield[a]`, `State[s]`, `Reader[r]`.
- Zero regressions on existing examples; `make verify-examples` and the v0.20.0 benchmark suite stay green.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Handler semantics: one-shot vs multi-shot (deep) vs shallow | Determines whether handlers can implement non-determinism / backtracking. Multi-shot interacts with concurrency model and trace replay. | human (design committee) | design | high |
| Syntax: `handle e with H` vs `with H handle e` vs `match e with effect ...` | Affects parser conflict surface, especially with existing `match` and `with` (record update) | human | design | high |
| Named handlers (`handle e with H@name`) in Phase 1 or deferred | Koka introduced named handlers later for good reason; affects how multiple instances of the same effect compose | human | design | med |
| Where handlers live in the type: row subtraction vs explicit discharge token | Determines how `effect E` interacts with existing built-in effects in the row algebra | compiler (type system) | compile | high |
| Discharge of built-in effects (can a user `handle IO` to mock println?) | Big A4 question: does discharging `IO` bypass the capability check, or is the cap still required to *install* the handler? | human (security review) | design | high |
| Resume keyword: `resume` (Koka) vs `continue` vs explicit continuation value | Affects readability and how the elaborator generates continuations | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] One-shot vs multi-shot handlers (Phase 1 default: **one-shot, deep** — pending committee approval)
- [ ] Syntax form for `handle` (proposed: `handle e with H` — pending parser conflict analysis below)
- [ ] User-defined effects: can they shadow built-in effect names? (proposed: **no**, reserve `IO`/`FS`/`Net`/`AI`/`Clock`/`Brain`/`AbsPath`)
- [ ] Can users handle built-in effects? (proposed: **yes, but capability still required to install the handler** — handlers don't grant authority)

---

## Solution Design

### Overview

Three new language constructs, layered:

1. **`effect E[α] { op : T -> U, ... }`** — declares a new effect with one or more operations.
2. **`handler H : E[α] { op(x) = ...; return(v) = ... }`** — defines a handler that implements those operations.
3. **`handle expr with H`** — runs `expr`, intercepting calls to `E`'s operations through `H`. The effect `E` is removed from the result's effect row.

This mirrors Koka §3.4 almost exactly. Where AILANG differs from Koka:

- **Capability model preserved.** Handling `IO` does not bypass `--caps IO`; the handler runs inside the capability sandbox. The cap check moves from "every call to `println`" to "installing a handler that discharges `IO`."
- **One-shot deep by default.** Phase 1 ships only the form that interacts cleanly with the existing tree-walking evaluator. Multi-shot waits for the bytecode VM.
- **No `fip`/`fbip`/`mask`/`override` decorators.** Out of scope.
- **No row polymorphism over effect *operations*.** You can still be polymorphic over the row (`! e`), but operations are nominal.

### Architecture

```
                  ┌─────────────────────────────┐
                  │ Surface syntax (.ail)       │
                  │   effect E { op : T -> U }  │
                  │   handler H : E { ... }     │
                  │   handle e with H           │
                  └────────────┬────────────────┘
                               │ lexer + parser
                               ▼
                  ┌─────────────────────────────┐
                  │ AST: EffectDecl, Handler,   │
                  │      HandleExpr             │
                  └────────────┬────────────────┘
                               │ elaborator
                               ▼
                  ┌─────────────────────────────┐
                  │ Core: handle{op_i ↦ h_i}    │
                  │       with explicit         │
                  │       continuations         │
                  └────────────┬────────────────┘
                               │ type checker
                               ▼
                  ┌─────────────────────────────┐
                  │ Effect row: subtract E      │
                  │ Verify ops match signature  │
                  └────────────┬────────────────┘
                               │ evaluator
                               ▼
                  ┌─────────────────────────────┐
                  │ Runtime: per-frame handler  │
                  │ stack; op calls walk stack  │
                  │ to find nearest matching H  │
                  └─────────────────────────────┘
```

**Components:**

1. **Effect Declaration** (`internal/parser/`, `internal/ast/`): new top-level form `effect Name[type-params] { op : Type, ... }`. Registers the effect in the module interface.
2. **Handler Definition** (`internal/parser/`, `internal/ast/`, `internal/elaborate/`): a handler is sugar for a record of functions plus a `return` clause. Elaborates to a struct with one closure per operation.
3. **`handle` Expression** (`internal/parser/`, `internal/types/`, `internal/eval/`): typing rule removes the discharged effect from the row; evaluator pushes the handler onto a frame-local stack before evaluating the body.
4. **Effect Operation Call** (`internal/eval/`): when an op of effect `E` is called, the evaluator walks the handler stack to find the innermost `H` discharging `E`, captures the continuation, and invokes `H.op(arg, k)`.
5. **Row Algebra Extension** (`internal/types/effects.go`): the row algebra must support nominal user-defined effect names in addition to built-ins. No change to row-poly substitution.

### Implementation Plan

**Phase 1: Effect declarations + one-shot deep handlers (~30–40h)** — v0.21.0

- [ ] Lexer: reserve `effect`, `handler`, `handle`, `with` (already reserved?), `resume`.
- [ ] Parser: `EffectDecl`, `HandlerDecl`, `HandleExpr` AST nodes.
- [ ] Elaborator: lower `handler` to a record of closures; lower `handle e with H` to a runtime primitive.
- [ ] Type checker: effect row subtraction on `handle`; nominal effect registry.
- [ ] Evaluator: handler stack on the per-fiber frame; `resume` captures continuation (one-shot, deep).
- [ ] Stdlib: `std/effect/yield`, `std/effect/state`, `std/effect/reader`.
- [ ] Tests: 30+ unit tests in `internal/eval/handlers_test.go`, 5+ integration examples.
- [ ] Docs: new `docs/docs/reference/effect-handlers.md`.

**Phase 2: Handlers over built-in effects (~25–30h)** — v0.22.0

- [ ] Allow `handle e with mockAI` where `mockAI : Handler[AI]`.
- [ ] Cap check stays: installing a built-in handler still requires the matching `--caps`.
- [ ] Stdlib: `std/test/mock_ai`, `std/test/mock_fs`, `std/test/mock_clock`.
- [ ] Eval harness: convert at least one benchmark to handler-mocked AI; measure determinism delta.

**Phase 3: Named handlers (~25–30h)** — v1.0.0 or later

- [ ] `handle e with H@name` syntax; effect rows track names.
- [ ] Multiple instances of same effect (`State[Int]@counter` + `State[String]@log`).
- [ ] This is the form Koka added in §3.4.13 after experience showed unnamed handlers were too coarse.

### Files to Modify/Create

**New files:**
- `internal/eval/handlers.go` — handler stack, op dispatch, `resume` implementation (~400 LOC)
- `internal/types/effect_decl.go` — nominal effect registry (~200 LOC)
- `stdlib/effect/yield.ail`, `stdlib/effect/state.ail`, `stdlib/effect/reader.ail` (~50 LOC each)
- `docs/docs/reference/effect-handlers.md` — user-facing docs
- `examples/handlers_yield.ail`, `examples/handlers_state.ail`, `examples/handlers_mock_ai.ail`

**Modified files:**
- `internal/lexer/lexer.go` — add `effect`, `handler`, `resume` keywords (~30 LOC)
- `internal/parser/parser_decl.go` — `EffectDecl`, `HandlerDecl` (~200 LOC)
- `internal/parser/parser_expr.go` — `HandleExpr`, conflict with `match`/record-update `with` (~150 LOC)
- `internal/ast/ast.go` — new AST nodes (~100 LOC)
- `internal/elaborate/effects.go` — lower handler to closure record (~200 LOC)
- `internal/types/effects.go` — row subtraction on handle (~150 LOC)
- `internal/types/typechecker.go` — handle-expression typing rule (~100 LOC)
- `internal/eval/eval.go` — handler frame, op call dispatch (~150 LOC)
- `internal/iface/builder.go` — export user-defined effects in module interface (~80 LOC; **must handle all 8 ast.Type variants per memory entry**)

---

## Conflict Surface

**This section is required because the change touches `internal/lexer/`, `internal/parser/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/effects/`, `internal/eval/`.**

### 1. New keywords vs existing identifiers

`effect`, `handler`, `handle`, `resume`, `with` — what already uses these?

| Token | Current use | Conflict? | Disambiguation |
|-------|-------------|-----------|----------------|
| `effect` | Used in **type annotations** as effect row marker (`! {IO, FS}`) but NEVER as a leading keyword | None at expression start; only ambiguous if a user has a value named `effect` | Reserve as keyword (breaking for code that names a variable `effect`) |
| `handler` | Not used | None | Free to reserve |
| `handle` | Not used (verified via `grep -E "\"handle\"" internal/lexer/`) | None | Free to reserve |
| `resume` | Not used | None | Free to reserve, but valid AILANG programs may use it as an identifier today |
| `with` | Currently used in **record update** (`{ r with field: val }`) | **YES — major** | See below |

### 2. The `with` ambiguity (highest risk)

`expr with H` collides syntactically with record update `{ r with field: v }`. Concrete parser scenarios:

```ailang
-- Existing record update
let updated = { user with name: "alice" }

-- Proposed handler form
let result = handle expr with myHandler
```

Distinct because `with` is currently only valid **inside braces after a record expression**, not after an arbitrary expression. But `handle` is a new prefix that creates a new context for `with`. Lookahead from `handle` token should suffice: `handle EXPR with HANDLER_EXPR`. The parser must commit to handler-mode on seeing `handle`.

**Risk:** If a future change allows `with` to appear in other expression positions (e.g., trait dictionary passing, capability annotations), the conflict surface grows. **Mitigation:** consider `handle e using H` or `handle e by H` as alternatives — explicitly cited as a design committee question.

### 3. Effect row syntax already exists

```ailang
func f() -> Int ! {IO, FS, MyEffect}  -- MyEffect is just a name in the row today
```

The row algebra already accepts arbitrary identifiers; what's missing is the **registry** that says which identifiers are real effects. Currently the type checker has a fixed set. After this change, the registry becomes dynamic per-module. **Conflict:** an existing program that uses `MyEffect` as an effect name without declaring it must continue to error (or now, after declaration, succeed) — behavior must not silently change.

### 4. Existing programs that must still work (regression fixtures)

- `examples/effect_io.ail` — built-in `IO` effect, unchanged.
- `examples/effect_fs.ail` — built-in `FS` effect, unchanged.
- `examples/effect_ai.ail` — built-in `AI` effect, unchanged.
- `examples/record_update.ail` — `{ r with f: v }` syntax, unchanged.
- All 23 benchmarks in `benchmarks/` — none use user-defined effects today, all must pass.
- `motoko-agent` integration suite — uses `handle` as an identifier in 0 places (verified before sprint start).

### 5. Deliberate breaking changes

- Code that names a variable `effect`, `handler`, `handle`, or `resume` will fail to parse. These are uncommon enough to accept; flag in changelog under "breaking."
- User-defined effect named `IO`, `FS`, `Net`, `AI`, `Clock`, `Brain`, `AbsPath`: **rejected at declaration time** with a clear error pointing to the built-in.

### 6. ast.Type switch exhaustiveness (per memory entry)

The 3 converter functions (`iface/builder.go`, `types/typechecker.go`, `elaborate/file.go`) each handle 8 ast.Type variants. Adding effect declarations does NOT add a new ast.Type variant — effects ride the existing row machinery — but the iface builder MUST emit user-defined effect declarations correctly. **Audit step:** verify the iface schema serializes nominal user effects round-trip.

### 7. The honest answer is *not* "no conflicts"

This change rewires four pipeline stages and reserves five keywords. The biggest unknown is whether the `with` ambiguity will surface in error messages confusingly when users forget which form they're writing. Plan: write 10 deliberately-wrong examples and measure error-message quality before declaring Phase 1 done.

---

## Examples

### Example 1: User-defined `Yield` effect (generators)

**Before** (today — must encode as a list, eager):

```ailang
func count_up(n: Int) -> [Int] = [1..n]
let total = sum(count_up(1000000))   -- builds 8MB list
```

**After** (Phase 1):

```ailang
effect Yield[a] {
  yield: a -> ()
}

func count_up(n: Int) -> () ! {Yield[Int]} =
  for i in 1..n { yield(i) }

handler sum_handler : Yield[Int] {
  return(_) = 0
  yield(x) = x + resume(())
}

let total = handle count_up(1000000) with sum_handler
-- Streams; never builds the list.
```

### Example 2: Mock `AI` effect for deterministic tests

**Before** (today):

```ailang
-- Test file sets env vars; AI call hits real provider unless mocked in Go
let answer = ai_call("what is 2+2?")  -- non-deterministic, costs $$
```

**After** (Phase 2):

```ailang
handler recorded_ai : AI {
  return(v) = v
  call(prompt) =
    let response = lookup(prompt, recorded_pairs)
    in resume(response)
}

let answer = handle ai_call("what is 2+2?") with recorded_ai
-- Deterministic, free, type-checked.
```

### Example 3: `State` effect (sketched)

```ailang
effect State[s] {
  get: () -> s
  put: s -> ()
}

func counter() -> Int ! {State[Int]} =
  let x = get();
  put(x + 1);
  get()

handler with_state[s](initial: s) : State[s] {
  return(v) = (v, s)         -- thread final state out
  get()    = \s -> resume(s)(s)
  put(x)   = \_ -> resume(())(x)
}

let (final, count) = handle counter() with with_state(0)
-- final = 1, count = 1
```

---

## Success Criteria

- [ ] `effect` / `handler` / `handle` parse and type-check on the three reference examples above.
- [ ] Handler discharge correctly subtracts the effect from the row (typed test).
- [ ] `resume` captures and invokes the continuation exactly once (Phase 1 one-shot semantics).
- [ ] Built-in effect handling (Phase 2) preserves capability checks — verified by negative test (no caps → still errors).
- [ ] Eval harness ships at least one benchmark mocked via `handle AI with recorded_ai` and runs deterministically across 10 successive invocations.
- [ ] All 23 v0.20.0 benchmarks pass unchanged.
- [ ] `make verify-examples` green.
- [ ] Documentation: `docs/docs/reference/effect-handlers.md` + 3 examples in `examples/`.
- [ ] Error messages for unhandled effects point at the call site and name the missing handler.

---

## Testing Strategy

**Unit tests:**
- Effect declaration parsing (positive/negative).
- Handler well-formedness (every op declared, `return` present).
- Row subtraction on `handle` typing.
- `resume` one-shot semantics; double-resume errors with a clear message.
- Handler stack walk: innermost match wins.

**Integration tests:**
- Three reference handlers (`Yield`, `State`, `Reader`) end-to-end.
- Phase 2: `mock_ai` against the eval harness.
- Negative: declare effect named `IO` → rejected.
- Negative: handler missing an op → elaboration error.

**Manual testing:**
- LSP hover on `handle` shows the discharged effect.
- Error messages for the 10 deliberately-wrong examples (Conflict Surface §7).

**Regression:**
- Full benchmark suite (`make eval-suite-core`).
- All `examples/*.ail` (`make verify-examples`).
- `motoko-agent` integration smoke test.

---

## Deferred Decisions

- **Multi-shot continuations** — agent should NOT implement multi-shot in Phase 1. Defer until bytecode VM lands; needs explicit committee approval.
- **`mask`/`override` modifiers** — Koka uses these to control handler search. Useful but out of scope for Phase 1; agent may sketch in stdlib comments.
- **Effect aliases** (`effect Logging = Yield[String]`) — agent may include if cheap; else defer.
- **`resume` argument arity** for ops returning unit — agent may choose `resume()` or `resume(())`; document the choice.
- **Polymorphic recursion through handlers** — agent should reject and document; full HM doesn't support this anyway (see existing limitation on Y-combinator).

## Non-Goals

- ❌ **Multi-shot handlers** — non-determinism / backtracking. Defer to v1.0.0+.
- ❌ **First-class continuations as values** — `resume` is the only continuation-introducing form.
- ❌ **Effect inference for handlers** — handler must declare which effect it discharges.
- ❌ **Modular effects** (Koka's `mod`) — handlers are flat in Phase 1.
- ❌ **Perceus-style refcounting / FBIP** — runtime is unchanged; tree-walking eval stays.
- ❌ **Handler subtyping / structural matching** — nominal handlers only.

---

## Timeline (Phase 1 only)

**Week 1** (~12h):
- Lexer + parser for `effect`, `handler`, `handle` (committee-approved syntax).
- AST nodes; conflict-surface fixtures (the 10 deliberately-wrong examples written FIRST).

**Week 2** (~14h):
- Elaborator: handler → closure record lowering.
- Type checker: row subtraction; nominal effect registry.
- Iface builder: round-trip user-defined effects.

**Week 3** (~12h):
- Evaluator: handler stack, op dispatch, `resume`.
- Stdlib: `std/effect/yield`, `std/effect/state`, `std/effect/reader`.
- Tests + docs.

**Total Phase 1: ~38h across 3 weeks.**

Phase 2 (~28h) and Phase 3 (~25h) scheduled when Phase 1 ships green.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `with` parser ambiguity surfaces in confusing error messages | Med | 10 deliberately-wrong examples gated on Phase 1 sign-off; consider `using` keyword if quality is poor |
| One-shot semantics insufficient for real use cases | Med | Phase 1 stdlib (`Yield`, `State`, `Reader`) covers the 80% case; multi-shot waits for VM |
| Handler stack walk slow in tree-walking eval | Low | Profile on the eval harness; acceptable if benchmark suite stays within 10% of v0.20.0 |
| Capability bypass via user-defined handler discharging `IO` | **High** | Phase 2 only; require explicit cap-to-install rule; security review before merge |
| Existing programs break due to new keywords | Low | `make verify-examples` + benchmark suite + motoko integration as gates |
| Iface schema regresses on user-defined effects (per ast.Type switch memory) | Med | Round-trip serialization test in `internal/iface/`; verify all 8 ast.Type variants still handled |

---

## Related Documents

**Implemented (foundational):**
- [design_docs/implemented/v0_2_0/m_r2_effect_system.md](../../implemented/v0_2_0/m_r2_effect_system.md) — capability runtime (0.45)
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — budget plumbing
- [design_docs/implemented/v0_6_2/m-bug-effect-checker-sprint-plan.md](../../implemented/v0_6_2/m-bug-effect-checker-sprint-plan.md) — effect-checker fixes (0.39)

**Planned (check for overlap):**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — parameterised effects `!{E[mode=...]}` (0.35). **Coordinate:** handlers + parameterised effects together complete the Leijen 2014 picture.
- [design_docs/planned/v0_13_0/m-exec-hierarchy-refactor.md](../v0_13_0/m-exec-hierarchy-refactor.md) — executor refactor (0.35)
- [design_docs/planned/v0_13_0/m-dx-expected-fail-fixes.md](../v0_13_0/m-dx-expected-fail-fixes.md) — DX (0.37)

## References

- [Design Axioms](/docs/references/axioms)
- [Koka Language Manual §3.4](https://koka-lang.github.io/koka/doc/book.html) — effect handlers, named handlers, scoped handlers.
- Plotkin & Pretnar (2009), [Handlers of Algebraic Effects](/references/plotkin-pretnar-2009.pdf) — already cited.
- Leijen (2014), [Koka: Row Polymorphic Effect Types](/references/leijen-koka-2014-arxiv.pdf) — already cited; handler chapter is what we're newly drawing on.
- Leijen (2017), *Type Directed Compilation of Row-Typed Algebraic Effects* — POPL'17, named handlers motivation.
- [Effekt language](https://effekt-lang.org/) — a research successor that explored capability-based effect handlers (relevant if Phase 3 reconsiders the cap interaction).

## Future Work

- **Multi-shot handlers** for non-determinism (post-VM).
- **Named handlers** for multiple effect instances (Phase 3).
- **Handler-defined `AI` test fixtures** as the default eval harness mock strategy.
- **Algebraic effect inference** — if HM extensions ever land, infer the effect a handler discharges.
- **Coordination with M-EFFECT-REFINEMENT** — `effect E[mode=...]` parameterised handlers.

---

**Document created**: 2026-05-16
**Last updated**: 2026-05-16
