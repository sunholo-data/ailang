# M-AI-PREFERRED-LANGUAGE: Closing the Remaining Gaps for AI-Native Programming

**Status**: Phase 1 + M5a Implemented (M4 `aa930e153`, M1 `2a432678e`, M2 `c8022abce`, M5a `|>` — 2026-11-25); Remaining: M3 (typed holes `_?`), M5b (`?` v1.1.0)
**Target**: v0.62.0 / v1.1.0 (phased across pre-1.0 quick wins and post-1.0 language extensions)
**Priority**: P1 (High — directly targets AI single-shot accuracy and 1-turn agent self-repair)
**Estimated**: 5 independent milestones (M1–M5), executable in any order
**Dependencies**:
- `M-SYNTAX-ERGONOMICS-V1` (`de314a613`) & `V2` (`39cfcae29`, `c6b634d55`)
- `M-AGENT-DIAGNOSTICS-ERGONOMICS` (`cf3140a33`)
- `M-AILANG-LSP-FOR-AI` (`design_docs/implemented/v0_20_0/m-ailang-lsp-for-ai.md`)
**Related Planned / Iceboxed Docs**:
- `design_docs/planned/v1_1_0/m-pipe-operator.md` (reopen condition triggered by `cp_60149` on 2026-11-24!)
- `design_docs/planned/v1_1_0/m-error-propagation.md` (evidence-gated icebox — Rev-1 ANF-bind design + 2 open architecture questions preserved there)
- `design_docs/deferred/m-arch5-error-handling-strategy.md` (historical error-handling strategy)
**Author**: Antigravity (attended session with Mark)
**Created**: 2026-11-25

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All 5 milestones are purely deterministic compiler/stdlib/diagnostic transforms |
| A2: Replayability | 0 | No runtime nondeterminism |
| A3: Effect Legibility | +1 | `ailang fix` + typed holes surface exact inferred effect rows instead of hiding them |
| A4: Explicit Authority | 0 | No capability or authority changes |
| A5: Bounded Verification | +2 | AILANG-syntax Z3 counterexamples turn `ailang verify` into an instant self-healing loop for agents |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Structured quick-fix edits, typed holes, and readable counterexamples eliminate prose-parsing guesswork |
| A8: Minimal Syntax | 0 | M1, M2, M4 add zero syntax; M3 (`_?`) is 1 expression token; M5 (`|>` / `?`) links existing v1.1.0 specs |
| A9: Cost Visibility | 0 | Zero hidden runtime overhead |
| A10: Composability | +2 | `std/list` + `std/result` combinators and `|>` eliminate hand-rolled recursive boilerplate |
| A11: Structured Failure | +2 | Machine-applicable edits (`codeAction` / `ailang fix`) + structured diagnostics |
| A12: System Boundary | 0 | No host/FFI boundary changes |

**Net Score: +10** (≥ +2, no −1 on A1/A3/A4/A7).

---

## Problem Statement & Prior Art Audit

After shipping `M-SYNTAX-ERGONOMICS-V1/V2` and `M-AGENT-DIAGNOSTICS-ERGONOMICS` (+12.9 pp on `stretch + frontier` 1st-attempt and final pass rates), five high-leverage gaps remain between AILANG and an AI's ideal programming substrate:

| # | Capability | Prior Art in Repo | What's Missing / Reopen Evidence |
|---|---|---|---|
| **M1** | **AILANG-syntax Z3 counterexamples** | `internal/smt/` (`ailang verify`) | Counterexamples still print raw SMT-LIB (e.g. `(seq.extract $p_xs 1 ...)` on `hof_verify.ail`). Rendering them as valid AILANG values (`xs = [9223372036854775807]`) lets agents read and write regression tests directly. |
| **M2** | **Structured quick-fix edits & `ailang fix`** | `m-ailang-lsp-for-ai.md` (shipped hover/def/refs/diagnostics) | Diagnostics still return prose suggestions only; no machine-applicable `TextEdit` patches (`ailang check --json` / LSP `textDocument/codeAction` / `ailang fix`) for missing stdlib imports, `IMP010` renames, or missing effect annotations. |
| **M3** | **Typed holes (`_?`)** | `m-ailang-lsp-for-ai.md` (`hover` works on existing bindings) | No way to ask the typechecker *"what type + effect row + in-scope bindings fit at this incomplete position?"* mid-synthesis. |
| **M4** | **Missing `std/list` & `std/result` combinators** | `std/list.ail` already has `flatMap`, `zipWith`, `takeWhile`, `dropWhile`, `findIndex`, `dedup`; `std/option.ail` has `getOrElse` | Still missing: `std/list` `filterMap`, `all`, `partition`, `enumerate`, `groupBy`; `std/result` `getOrElse` (currently only panicking `unwrap`). |
| **M5** | **Left-to-right data flow (`|>`) & error propagation (`?`)** | `design_docs/planned/v1_1_0/m-pipe-operator.md` & `m-error-propagation.md` (both iceboxed 2026-07-23) | **`m-pipe-operator.md` reopen condition triggered** on 2026-11-24 (`cp_60149` wrote `\|>` unprompted in `stretch + frontier`!). `m-error-propagation.md` remains gated on its 2 open v1.1 architecture questions (ANF-bind audit + compiler↔stdlib `Result` constructor linkage). |

---

## Proposed Milestones (Phased)

### Phase 1 — Pre-1.0 Zero-Syntax-Risk Wins (v0.62.0)

#### M1: AILANG-Syntax Z3 Counterexamples (`internal/smt/`)
- Decode Z3 `get-value` models for `Int`, `Real`, `Bool`, `String`, `(Seq T)` (`[e1, e2, ...]`), declared records (`{ field: val, ... }`), and ADTs (`Some(v)`, `None`, user constructors) back into canonical AILANG syntax.
- Fall back to raw SMT-LIB only for unencodable/symbolic expressions, clearly labeled.
- **Payoff:** Agents can paste counterexamples directly into `test "..." { assert ... }` blocks and repair contracts in 1 turn.

#### M2: Machine-Applicable Quick-Fix Edits & `ailang fix` (`internal/diag/`, `internal/lsp/`, `cmd/ailang/`)
- Attach optional `Edits []TextEdit` (`{File, StartLine, StartCol, EndLine, EndCol, NewText}`) to structured diagnostics where the fix is 100% deterministic:
  1. `IMP010` exact/close stdlib symbol rename or wrong-module move (powered by `internal/importhint`)
  2. Missing stdlib import (`undefined variable: filter` → insert `import std/list (filter)`)
  3. Missing effect annotation on a function signature (`! {IO}`)
  4. `PAR018` (`let rec` → `letrec`) and `PAR_HASH_COMMENT` (`#` → `--`)
- Expose via `ailang check --json`, LSP `textDocument/codeAction`, and `ailang fix <file>`.

#### M4: Missing `std/list` & `std/result` Combinators (`std/list.ail`, `std/result.ail`)
- Add iterative $O(n)$ (0 VM fallbacks under `--strict-bytecode`):
  - `std/list`: `all`, `filterMap`, `partition`, `enumerate`, `groupBy`
  - `std/result`: `getOrElse[a, e](r: Result[a, e], default: a) -> a` (parity with `std/option.getOrElse`)
- Freeze `.stdlib-golden/{list,result}.{json,sha256}` and document in the next teaching prompt.

### Phase 2 — Synthesis & Post-1.0 Ergonomics (v0.63.0 / v1.1.0)

#### M3: Typed Holes (`_?`)
- Lex/parse `_?` (or `?name`) as an `ast.Hole` expression that always unifies with the expected type and emits a structured diagnostic `TC_HOLE`:
  - Expected type + required effect row at the hole
  - In-scope local bindings & imported functions whose return type unifies with the expected type
- Zero runtime semantics (compilation stops with `TC_HOLE` until filled).

#### M5: Reopen `m-pipe-operator.md` (`|>`) & Evaluate `m-error-propagation.md` (`?`)
- **M5a (`|>`):** Unpark [m-pipe-operator.md](../v1_1_0/m-pipe-operator.md) — its reopen condition (*"`|>` appearing in eval banks"*) is now empirically met (`cp_60149`), and its Rev-1 design is already technically complete (6–8h).
- **M5b (`?`):** Keep [m-error-propagation.md](../v1_1_0/m-error-propagation.md) in v1.1.0 until its 2 quorum-flagged architecture questions (ANF-bind characterization tests + compiler↔stdlib `Result` constructor registry) are resolved.

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Owner | Change Cost |
|---|----------|---------|----------------|-------|-------------|
| **D1** | Sequencing | **(a)** Phase 1 (M4 → M1 → M2) pre-1.0, Phase 2 (M3, M5) post-1.0 vs **(b)** Unpark M5a (`\|>`) into Phase 1 now | **(a)** Ship M4 + M1 + M2 first (zero syntax risk before 1.0), then decide on M5a (`\|>`) | Mark | Low |
| **D2** | `ailang fix` safety boundary | **(a)** Only compiler-verified mechanical edits (imports, `IMP010` renames, `let rec`→`letrec`, effect rows) vs **(b)** Also rewrite structural syntax (`PAR_IF_BRANCH_BLOCK`) | **(a)** Mechanical edits only — never risk mis-bracing user logic | Mark | Low |
| **D3** | Hole syntax (M3) | **(a)** `_?` / `?name` vs **(b)** `???` | **(a)** `_?` (avoids clashing with future `?` propagation operator in M5b) | Mark | Low |

### Design Freeze
- [x] **D1** ratified — **(a)** Phase 1 = M4 → M1 → M2 before 1.0 (Mark, 2026-11-25)
- [x] **D2** ratified — **(a)** Mechanical compiler-verified edits only for `ailang fix` (Mark, 2026-11-25)
- [x] **D3** ratified — **(a)** `_?` for typed holes (Mark, 2026-11-25)

---

## Success Criteria
- **M1:** `ailang verify examples/runnable/contracts/hof_verify.ail` renders its counterexample as valid AILANG syntax (`xs = [9223372036854775807]`) rather than raw SMT-LIB.
- **M2:** `ailang check --json` emits structured `edits` for `IMP010`, missing imports, and missing effect rows; `ailang fix` repairs a fixture suite in 1 pass.
- **M4:** All new `std/list` and `std/result` combinators pass evaluator + `--strict-bytecode` (0 VM fallbacks) + `make verify-stdlib`.
- **M3 / M5:** Each ships with parser/typechecker/formatter round-trip tests and 0 regressions in `make ci-quick`.
