# M-EFFECT-PURE-ROW-OVERGENERALIZATION — a `pure func` must not export an effect-polymorphic row

**Status**: Planned
**Target**: v0.35.3
**Priority**: P1 (Medium) — blocking false rejection of valid code; NOT a soundness hole (V9)
**Estimated**: 1.5 days
**Dependencies**: None. Adjacent to (and NOT blocked by) the parked
[M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) — see Related Documents.
**Source**: GitHub issue [#1091](https://github.com/sunholo-data/ailang/issues/1091), reproduced
and root-caused in this session at `origin/dev` = `3ee5bb177`.

## Problem Statement

A function declared `pure` exports an **effect-polymorphic** signature — an open effect row with a
quantified row variable — instead of the closed empty row that `pure` promises. Importers then
instantiate that row fresh, it unifies with whatever effect the surrounding inference supplies, and
the effect checker reports the *caller* as using an effect nothing in the program performs.

The trigger is narrow and entirely invisible to the author: **the `pure` function's body calls a
recursive function**. A recursive call shares the enclosing function's effect-row variable
(`typechecker_functions.go` `inferApp`, "a recursive self-call shares the enclosing function's
effect-row variable"), so the row never resolves to a concrete set. With nothing concrete to unify
against, the unresolved row var survives to generalization, where
`generalizeWithConstraints` quantifies **every** free effect-row var of the type
(`typechecker_functions.go:478-495`) — including this one. `pure` is never consulted.

### Reproduction (issue #1091, reconstructed and confirmed)

`sunholo-data/ailang-parse` @ `160c7f1`. Three reference-doc loaders each carried private copies of
the same XML-surgery helpers; the refactor extracts them to `docparse/services/pkg_template`.
The helpers are `export pure func`, do no I/O, and import only `std/string`. Their bodies are
byte-identical to the module-local versions that checked clean immediately before the move.

```
$ ailang check docparse/services/docx_template.ail
Error: effect checking failed in docparse/services/docx_template:
  Effect checking failed for function 'docxTplCleanForSectPr'
  Missing effects: FS
  Current signature: func docxTplCleanForSectPr(...) -> T
  Suggested fix:     func docxTplCleanForSectPr(...) -> T ! {FS}
```

The named function is:

```ailang
pure func docxTplCleanForSectPr(docXml: string) -> string {
  let bodyEnd = pkgLastIndexOf(docXml, "</w:body>");
  let inner = if bodyEnd >= 0 then substring(docXml, 0, bodyEnd) else docXml;
  pkgDropSpans(inner, "<w:sectPrChange", "</w:sectPrChange>")
}
```

Nothing about FS is involved anywhere on this path. `ailang run` on the real entry point fails
identically (V4), so the program genuinely does not run — this is not a checking-mode artifact.

### Why the two imported helpers behave differently

Both are `export pure func` with contracts, called from the same function body. Only one is
contaminated:

```
[DEBUG_EFFECTS] VarGlobal(docparse/services/pkg_template.pkgLastIndexOf) -> [FS]
[DEBUG_EFFECTS] VarGlobal(docparse/services/pkg_template.pkgDropSpans)   -> []
```

`pkgLastIndexOf` delegates to a private **recursive** scan; `pkgDropSpans`'s exported row happened
to close. Their exported schemes differ accordingly (V6/V7):

| helper body | exported scheme | importer sees |
|---|---|---|
| delegates to a **recursive** private func | `(string, string) -> int ! {...ρ2}`, `RowVars=[ρ2]` | **open → absorbs `FS`** |
| delegates to a non-recursive private func | `(string, string) -> int`, `RowVars=[]` | pure, correct |

### Why it only appears after moving code between modules

`validate_effects.go:344-353` already carries a fix for this symptom class, added by
[M-BUG-EFFECT-CHECKER-CONFLATION](../../implemented/v0_6_2/m-bug-effect-checker-conflation.md)
(v0.6.2) and commented *"Use declared effects instead of CoreTypeInfo to avoid contamination"*.
That bypass consults the declared-effects map **only when the callee is a `*core.Var`** — a
module-local binding. A cross-module callee is a `*core.VarGlobal` and falls through to the
`CoreTypeInfo` path the comment is warning about. So the same helper is safe in its own module and
unsafe one import away, which is exactly the observed before/after of the refactor.

### Scope: false rejection, not a soundness hole

The over-generalization is asymmetric. It fires only when the declared row is **empty**: with any
concrete declared effect the recursive call unifies against it and the row closes. An `! {IO}`
function that recurses exports the closed `int -> int ! {IO}`, and a `pure` caller importing it is
correctly **rejected** (V9). So the defect rejects valid programs; it does not admit invalid ones.
This is why the priority is P1 and not P0.

### Impact

- Blocks any cross-module extraction of a `pure` helper whose body recurses — i.e. exactly the
  de-duplication refactor this was found in. The only workaround is keeping N copies of the helper,
  which is what the refactor exists to remove.
- Recursion is the *idiomatic* way to write a scan in AILANG (no loops — A8), and shared string/XML
  helpers are precisely the code worth extracting, so the trigger sits on a common path.
- The error is maximally misleading: it names an effect (`FS`) that appears nowhere on the path,
  blames the caller rather than the import, and its "Suggested fix" tells the author to declare an
  effect the function does not have. Following that suggestion propagates a false `! {FS}` outward
  through every transitive caller.

## Verification Log

Base: worktree at `origin/dev` = `3ee5bb177`. Repro tree: a scratchpad copy of `ailang-parse` @
`160c7f1` with the extraction patch reconstructed (the issue's patch was never pushed; the upstream
checkout is clean at that SHA — V2). `/tmp/ailang-probe` is this tree's `cmd/ailang` plus a
temporary `DEBUG_XMOD` print in `inferVarGlobal` dumping `scheme.TypeVars`/`RowVars`/`Type` and the
instantiated type; the probe was reverted after measurement and is **not** part of this design.

| # | Claim | Command / evidence | Result |
|---|---|---|---|
| V1 | Not already in progress | `gh issue view 1091`; `gh pr list --search 1091`; issue timeline; grep of `design_docs/` | 0 comments, no labels, no assignee, no PR, no branch, no cross-refs — **confirmed unstarted** |
| V2 | `ailang-parse` @ `160c7f1` is clean; patch unpushed | `git status --short` (empty), `ls docparse/services/pkg_template.ail` → No such file | Confirmed — repro had to be reconstructed |
| V3 | Extraction reproduces the failure at HEAD | `ailang check docparse/services/docx_template.ail` on the patched copy | `Missing effects: FS` on `docxTplCleanForSectPr` — reproduced on **v0.35.2** (issue filed against v0.35.1-95) |
| V4 | It is not a checking-mode artifact | `ailang run docparse/main.ail --entry main` | Same error — the program does not run |
| V5 | Baseline (pre-patch) is clean | `ailang check` on the unpatched copy | `✓ No errors found!` |
| V6 | A recursive delegate opens the exported row | `DEBUG_XMOD=1` probe, helper delegating to a recursive private scan | `scheme.RowVars=[ρ2]`, `scheme.Type=(string, string) -> int ! {...ρ2}` |
| V7 | A non-recursive delegate closes it | same probe, private delegate made non-recursive | `scheme.RowVars=[]`, `scheme.Type=(string, string) -> int`; `ailang check` → `✓ No errors found!` |
| V8 | The contaminated row is what the effect pass reads | `DEBUG_EFFECTS=1 ailang check` | `Callee type effects (from CoreTypeInfo): [FS]` then `VarGlobal(...pkgLastIndexOf) -> [FS]`, while `...pkgDropSpans -> []` |
| V9 | **Not** a soundness hole; asymmetric to empty rows | helper `! {IO}` recursing, imported by a `pure` caller | exports **closed** `int -> int ! {IO}`; caller correctly **rejected** |
| V10 | The declared-effects bypass is `*core.Var`-only | read `internal/pipeline/validate_effects.go:344-353` | `if funcVar, ok := e.Func.(*core.Var); ok` — cross-module `*core.VarGlobal` falls through to `CoreTypeInfo` |
| V11 | Generalization quantifies every free effect-row var, unconditionally | read `internal/types/typechecker_functions.go:478-495` | `freeEffectRowVarsInType(typ)` minus env-free vars → `RowVars`; the declared row is never consulted |
| V12 | Recursion is why the row stays unresolved | read `inferApp` comment, `internal/types/typechecker_functions.go:530-540` | *"a recursive self-call shares the enclosing function's effect-row variable"* — the local solve deliberately will not bind it |
| V13 | The LetRec path runs a defaulting pass immediately before generalizing | read `internal/types/typechecker_functions.go:330-377` | type-class defaulting only; **no effect-row defaulting** — the natural insertion point |
| V14 | Deliberate row polymorphism must be preserved and is distinguishable | `DEBUG_XMOD` probe on a module importing `std/list.mapE` | `RowVars=[e]`, `(a -> α202 ! {...e}, list[a]) -> [α202] ! {...e}` — `e` is a **user-declared type param** (`std/list.ail:220` `mapE[a, b, e](f: a -> b ! {e}, ...) -> [b] ! {e}`), unlike the compiler-fresh `ρ2` |
| V15 | The surface AST is already available where interfaces are built | read `internal/iface/builder.go:331,337,350,443-445` | `Build(prog, constructors, astFile)`, already downcast `astFile.(*ast.File)` — declared effects reachable without a new plumbing path |
| V16 | `ailang iface` is NOT a valid instrument for this question | `ailang iface std/list` | `mapE` prints `"effects": [], "pure": true` — the JSON view flattens the row var away; the Scheme-level probe is required |
| V17 | The issue's "revealing part" is a CLI artifact, not a checker signal | `ailang check pkg_template.ail docx_template.ail` and the reverse order | Only the **first** file is checked (`→ Type checking <first>...` only). Passing the dependency did not fix anything — trailing args are ignored |
| V18 | `--- ` no existing doc covers this defect | neural/SimHash `ailang docs search`; read of both top hits | v0.6.2 conflation doc = same symptom, **intra-module**, different mechanism, already fixed for `*core.Var`. Row-var doc = declared `! {e}`, and is **parked**. Neither covers an inferred row on a `pure` declaration |

**Correction to be filed on #1091**: V17 refutes the issue's "Narrowing" claim that supplying the
dependency on the command line makes the failure go away. It does not; `ailang check` ignores
arguments after the first. Anyone chasing that as a clue is chasing a CLI bug, not this one.

## Goals

**Primary Goal:** A function whose declaration specifies a closed effect row exports exactly that
row. `pure` means `{}` at the module boundary, in every body shape, including recursive ones.

**Success Metrics:**
- The #1091 reproduction (`ailang check` and `ailang run` on the extracted `ailang-parse` tree)
  passes without altering any `.ail` source.
- `std/list.mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE` and the other declared row-polymorphic
  stdlib signatures keep `RowVars=[e]` — deliberate effect polymorphism is untouched (V14).
- No `pure`-declared export anywhere in `std/` or `examples/` carries a quantified effect row.
- `make test` and `make verify-examples` green; no change to any existing effect-checking test's
  expected text.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at **inference** (default the unresolved row to the declared row before generalization) vs at the **iface boundary** (rewrite the exported scheme's row from the surface AST) | Inference fixes the type everywhere (CoreTypeInfo, REPL, iface) and matches where the defect is (V11); the iface patch is narrower but leaves the wrong row inside the defining module's own type info | agent | design | high |
| Discriminate declared-closed from declared-polymorphic by **the declaration**, not by row-var naming (`ρN` vs `e`) | Keying on the compiler's fresh-var prefix would be a silent trap the first time a user names a row `rho`; keying on the declaration is the actual semantics (V14) | compiler | design | high |
| Whether to **also** extend the `validate_effects.go` declared-effects bypass to `*core.VarGlobal` | It is defence-in-depth for the same symptom class, but it papers over a wrong *type* rather than fixing it, and the cross-module declared row is not in that map today | agent | design | med |
| Leave the `ailang check` multi-file argument bug (V17) out of scope | It is a real, separate CLI defect that misled the reporter; fixing it here would blur the sprint's acceptance surface | agent | design | low |

**Recommendation on row 1: fix at inference.** V11 places the defect squarely in
`generalizeWithConstraints`, V13 shows the LetRec path already runs a defaulting pass one statement
earlier with the right data in scope, and a fix there corrects `CoreTypeInfo` too — which is what
`validate_effects` actually reads (V8). The iface-boundary variant would leave `DEBUG_EFFECTS`
still reporting `[FS]` inside the defining module and would need re-doing when the parked row-var
work resumes.

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Fix site**: inference — default an unresolved effect row to the declared row before
      generalization, in the LetRec/Let binding paths (`internal/types/typechecker_functions.go`).
- [x] **Discriminator**: the function's declared effect row. A declared row that is closed
      (`pure`, or no `!` annotation, or an explicit closed `! {…}`) closes the inferred row; a
      declared row carrying a user-written row variable (`! {e}`, `e` bound in the signature's type
      params) is left polymorphic.
- [x] **`validate_effects` bypass**: not extended in this sprint. The type is fixed at the source;
      extending the bypass would hide a regression in that fix rather than catch it. Recorded as
      Future Work.
- [x] **`ailang check` multi-file bug**: out of scope, filed separately (see Non-Goals).

## Solution Design

### Overview

Close the loop that `pure` already promises. When a top-level binding's declared effect row is
closed, its inferred effect row must be **defaulted to that declared row** before the scheme is
generalized, so no compiler-fresh row variable is ever quantified into a `pure` function's exported
type. Declared row-polymorphic signatures (`! {e}`) are unaffected because their row variable comes
from the declaration, not from inference.

### Architecture

Today, for a recursive `pure` binding:

```
infer body ──► effect row = ρ2 (unresolved: recursive self-call shares it, V12)
             │
             ├─ defaulting pass (type classes only — V13)
             │
             └─ generalizeWithConstraints
                  freeEffectRowVarsInType(typ) = {ρ2}, not free in env
                  ⇒ RowVars = [ρ2]                                     ◄── DEFECT (V11)
                  ⇒ exported: (string, string) -> int ! {...ρ2}
                                    │
                       importer instantiates ρ2 ↦ ε90, unifies with FS
                                    │
                       validate_effects reads CoreTypeInfo ⇒ "Missing effects: FS"  (V8, V10)
```

After:

```
infer body ──► effect row = ρ2
             │
             ├─ defaulting pass (type classes)
             ├─ NEW: effect-row defaulting
             │        declared row closed?  ⇒ bind ρ2 := declared row (for pure: {})
             │        declared row has a user row var? ⇒ leave ρ2 alone
             │
             └─ generalizeWithConstraints
                  freeEffectRowVarsInType(typ) = {} ⇒ RowVars = []
                  ⇒ exported: (string, string) -> int
```

The new step is a *defaulting* rule, not a new check: it does not reject anything that is accepted
today, and it cannot mask a genuine effect. A genuine effect appears as a **concrete label** in the
row, and the recursive-call unification already closes such rows against the declared set (V9) —
where the declared row is closed and the inferred row carries a label the declaration omits, the
existing effect-checking pass still rejects it. Only an *unresolved variable* is defaulted.

### Implementation Plan

**Phase 1 — Regression tests first (~3h)**
- [ ] Go test in `internal/types/`: a recursive `pure` binding generalizes with `RowVars == nil`
      and a closed effect row (unit-level, asserts on the Scheme — the level `ailang iface` cannot
      see, V16).
- [ ] Go test: a declared `! {e}` binding still generalizes with `RowVars == ["e"]` (pins V14).
- [ ] Two-module `.ail` fixture reproducing #1091 minimally (`pure` export delegating to a private
      recursive scan; importer calls it from a `pure` function reachable from an `! {FS}` one),
      asserted to check clean.
- [ ] Negative fixture: an importer that genuinely needs the effect is still rejected (pins V9).

**Phase 2 — The fix (~4h)**
- [ ] Add effect-row defaulting alongside the existing defaulting pass in the LetRec path
      (`typechecker_functions.go:330-377`) and the corresponding Let/top-level binding path.
- [ ] Resolve the declared row for the binding; bind unresolved effect-row vars to it when it is
      closed.
- [ ] Confirm `DEBUG_EFFECTS` now reports `VarGlobal(...pkgLastIndexOf) -> []`.

**Phase 3 — Validation (~3h)**
- [ ] `make test`, `make lint`, `make verify-examples`.
- [ ] Re-run the reconstructed `ailang-parse` extraction: `ailang check` and `ailang run` both green.
- [ ] Sweep `std/` for `pure` exports carrying a quantified row (should be none post-fix).
- [ ] CHANGELOG entry.

### Files to Modify/Create

| File | Change | LOC |
|------|--------|-----|
| `internal/types/typechecker_functions.go` | Effect-row defaulting before generalization, in the LetRec and Let binding paths | ~50 |
| `internal/types/effect_row_defaulting_test.go` | New — Scheme-level unit tests (closed stays closed, `! {e}` stays polymorphic) | ~120 |
| `internal/pipeline/validate_effects_xmod_test.go` | New — two-module fixtures: #1091 shape accepted, genuine-effect shape still rejected | ~110 |
| `CHANGELOG.md` | Entry under v0.35.3 | ~6 |

## Examples

### Example 1: the #1091 shape (currently rejected, must be accepted)

```ailang
-- pkg_template.ail
module docparse/services/pkg_template
import std/string (length, find, substring)

export pure func pkgLastIndexOf(hay: string, needle: string) -> int =
  pkgLastIdxScan(hay, needle, 0, -1)          -- private, RECURSIVE

pure func pkgLastIdxScan(hay: string, needle: string, offset: int, best: int) -> int {
  let i = if offset >= length(hay) then -1
          else find(substring(hay, offset, length(hay)), needle);
  if i < 0 then best else pkgLastIdxScan(hay, needle, offset + i + 1, offset + i)
}
```

```
before:  scheme.RowVars=[ρ2]   (string, string) -> int ! {...ρ2}   ⇒ importer sees [FS]
after:   scheme.RowVars=[]     (string, string) -> int             ⇒ importer sees []
```

### Example 2: deliberate effect polymorphism (must be unchanged)

```ailang
-- std/list.ail:220
export func mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e} { ... }
```

```
before and after:  scheme.RowVars=[e]   (a -> b ! {...e}, list[a]) -> [b] ! {...e}
```

## Success Criteria

- [ ] Reconstructed `ailang-parse` extraction: `ailang check docparse/services/docx_template.ail`
      and `ailang run docparse/main.ail --entry main` both succeed, with no `.ail` source edits
- [ ] `DEBUG_EFFECTS=1` reports `VarGlobal(docparse/services/pkg_template.pkgLastIndexOf) -> []`
- [ ] `std/list.mapE` keeps `RowVars=[e]` (V14 pinned as a test)
- [ ] Genuine cross-module effect requirement still rejected (V9 pinned as a test)
- [ ] No `pure`-declared export in `std/` carries a quantified effect row
- [ ] `make test` green; no existing effect-test expected-text changes
- [ ] `make verify-examples` green
- [ ] CHANGELOG.md updated

## Testing Strategy

Three levels, because the defect is invisible at two of them:

1. **Scheme level** (`internal/types/`) — the only level where `RowVars` is observable. `ailang iface`
   flattens it away (V16) and `ailang check` only reports the downstream symptom.
2. **Cross-module pipeline** (`internal/pipeline/`) — two-module fixtures, since the defect cannot
   reproduce inside one module (V10: the local `*core.Var` bypass masks it).
3. **End-to-end** — the reconstructed `ailang-parse` extraction, which is the only artifact that
   exercises the full tangle that defeated minimization (see Risks).

Both directions are pinned: the accept case (#1091) and the reject case (V9), so the fix cannot
pass by simply making the effect checker more permissive.

## Deferred Decisions

Latitude granted to the implementer:
- The exact representation of "declared row is closed" (re-elaborate from the surface AST vs. read a
  row already carried on the core binding), provided the discriminator is the declaration and not a
  row-var name prefix.
- Whether the Let and LetRec paths share one helper or carry two call sites.
- Test file names and fixture module names.

## Non-Goals

- **Not** fixing `ailang check`'s multi-file argument handling (V17) — real, separate, and filed
  on its own; conflating it would blur this sprint's acceptance surface.
- **Not** implementing row-variable resolution in `validate_effects.go` — that is the parked
  M-EFFECT-ROW-VAR-UNIFICATION, a larger P0 about *declared* `! {e}` signatures.
- **Not** extending the `validate_effects.go` declared-effects bypass to `*core.VarGlobal`.
- **Not** changing the effect-checker error message text.

## Timeline

| Day | Work |
|-----|------|
| Day 1 (am) | Phase 1 — regression tests, red |
| Day 1 (pm) | Phase 2 — the fix, tests green |
| Day 2 (am) | Phase 3 — full suite, examples, `ailang-parse` end-to-end, CHANGELOG |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Closing rows too eagerly silently swallows a real effect | Low | High | Only *unresolved variables* are defaulted, never concrete labels; V9's reject case is a pinned test in both directions |
| Breaks deliberate row polymorphism in `std/` | Low | High | V14 pinned as a Scheme-level test on `mapE`; `make verify-examples` covers the consumers |
| The defect resists minimization, so a fix looks green on a toy case and fails on the real one | **Measured** | Med | The reporter failed to minimize across four attempts; an independent dependency-aware delta-debugger converged at **37/37 decls retained** (every decl is transitively reachable from an `! {FS}` function, so removals surface a type error that masks the FS error). Therefore the reconstructed `ailang-parse` tree is a **required** acceptance artifact, not an optional one |
| Interaction with the parked row-var work | Med | Low | This fix removes rows that should never have existed; it does not touch declared-`! {e}` handling, which is that doc's entire subject. It also **retires** one of that doc's pinned ACs — see below |

## Related Documents

- [M-BUG-EFFECT-CHECKER-CONFLATION](../../implemented/v0_6_2/m-bug-effect-checker-conflation.md)
  (implemented v0.6.2) — **same symptom, different mechanism.** Spurious `IO` on a pure function,
  fixed by the declared-effects bypass at `validate_effects.go:344-353`. That fix is `*core.Var`-only
  and so does not reach cross-module callees (V10); this doc addresses the type that bypass was
  compensating for.
- [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) (planned v1.0.0, P0,
  **parked** — `eabab0611` "the fix site moved a fourth time"). Adjacent, distinct, and this work
  **refutes one of its pinned acceptance criteria**: its Conflict Surface records
  *"Cross-module callees (`VarGlobal` → typeInfo path) | Already correct (V14/V15) | Unchanged"*,
  and its Problem Statement rests on the stdlib row-var signatures surviving "only because
  cross-module calls take a different, correct code path (V14–V15)". V6/V8 here show that path is
  **not** already correct. That doc's V14/V15 arms test *declared* `! {e}` functions, which do work;
  they do not cover an *inferred* row on a `pure` declaration. **Action:** when that doc is
  unparked, its V14/V15 pinned ACs must be re-scoped to "declared row-polymorphic callees only".
- [M-EFFECT-ROW-SHOW-INTERP](../../implemented/v0_31_0/m-effect-row-show-interp.md) (#386,
  implemented v0.31.0) — added the row-var generalization in `generalizeWithConstraints` that this
  doc constrains. #386 was right that free row vars must be quantified so separate imported uses do
  not share one identity; it did not exclude rows belonging to a *declared-closed* function.

## References

- Issue [#1091](https://github.com/sunholo-data/ailang/issues/1091)
- `internal/types/typechecker_functions.go:444` — `generalizeWithConstraints`
- `internal/types/typechecker_functions.go:330-377` — LetRec defaulting + generalization
- `internal/pipeline/validate_effects.go:344-353` — the `*core.Var`-only declared-effects bypass
- `internal/iface/builder.go:337` — `Build(prog, constructors, astFile)`
- `std/list.ail:220` — `mapE[a, b, e]`

## Future Work

- Extend the `validate_effects.go` declared-effects bypass to `*core.VarGlobal` as defence in depth,
  once the cross-module declared row is available to that pass.
- Fix `ailang check`'s silent dropping of every argument after the first (V17).
- Revisit `ailang iface`'s effect projection, which reports a row-polymorphic export as
  `"effects": [], "pure": true` (V16) — indistinguishable from a genuinely pure one.

## Conflict Surface

Required: this change touches `internal/types/`.

**1. What does this change extend?** The generalization step for top-level bindings — specifically
which effect-row variables get quantified into an exported `Scheme`.

**2. What else lives in that position?**

| Construct | Today | Post-change | Evidence |
|---|---|---|---|
| Declared row-polymorphic exports (`! {e}`, `e` a signature type param) — 13 shipped signatures across `std/list`, `std/stream`, `std/ai/streaming`, `std/smoke` | `RowVars=[e]`, row var originates in the declaration | **Unchanged** — declared row is not closed, so no defaulting applies | V14 |
| `pure` / unannotated exports with a **non-recursive** body | Row already closes during inference; `RowVars=[]` | **Unchanged** — nothing to default | V7 |
| `pure` / unannotated exports with a **recursive** body | `RowVars=[ρN]` (defect) | Row defaulted to `{}`, `RowVars=[]` | V6 |
| Exports with a concrete declared row (`! {IO}`, `! {FS}`) and a recursive body | Recursive call unifies against the concrete row; exports closed | **Unchanged** — already closed, defaulting is a no-op | V9 |
| Type-class defaulting in the same pass | Runs immediately before generalization | **Unchanged** — the new rule operates on effect rows only, and runs alongside it | V13 |
| `validate_effects.go` `*core.Var` bypass | Consults declared effects for module-local callees | **Unchanged** — deliberately not extended (Design Freeze) | V10 |
| `internal/iface` serialization / `internal/format` printing | Read-only consumers of `Scheme.RowVars` | Unchanged code; observed output changes only where a row was wrongly quantified | V15, V16 |
| Runtime capability checks | Label-based, no row vars | Unchanged — remains the backstop | V9 |

**3. How is the ambiguity resolved?** By the *declaration*, never by the row variable's name.
A row var reaching generalization is quantified iff the binding's declared effect row is itself
row-polymorphic. The tempting shortcut — treating compiler-fresh `ρN` differently from source-named
`e` — is explicitly rejected in the Design Freeze: it would break the first time a user names a row
variable `rho`.

**4. Programs that MUST still work** (all present in-tree; each becomes a pinned test):
- `std/list.ail` `mapE`/`filterE`/`foldlE` and their consumers — `make verify-examples`
- Any `pure` recursive function used **within** its own module (the pre-refactor `docx_template.ail`
  shape, V5)
- The V9 reject case: a `pure` caller of a genuinely effectful cross-module function
- `internal/pipeline/effect_mode_subsumption_test.go` — existing expected text unchanged

**5. What deliberately changes?** Exactly one thing: a `pure`-declared export whose body recurses
stops exporting an effect-polymorphic row. Any code that *depended* on that row absorbing an effect
was relying on the defect — such code is currently rejected, not accepted, so there is no program
that works today and stops working.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism either way |
| A2: Replayability | 0 | Traces unaffected |
| A3: Effect Legibility | **+1** | The core of the fix: `pure` at a module boundary now *means* pure. Today a signature reading `pure` exports "any effects", and the checker reports effects no code performs |
| A4: Explicit Authority | 0 | No capability/authority change; runtime checks untouched (V9) |
| A5: Bounded Verification | **+1** | An import's effect contract becomes decidable from its own declaration rather than from the importer's unification context |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | **+1** | Removes a false, actively misleading diagnostic — its "Suggested fix" instructs the author to declare an effect the function does not have, propagating the error outward through every transitive caller |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | **+1** | Restores cross-module extraction of pure helpers — the refactor the issue was blocked on; recursion is the idiomatic scan under A8, so the trigger sits on a common path |
| A11: Structured Failure | 0 | Error shapes unchanged; one false failure removed |
| A12: System Boundary | 0 | No boundary crossing change |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects — strictly *more* effect legibility; only unresolved row
      variables are defaulted, never concrete labels, and the V9 reject case is pinned in both directions
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): removes a false diagnostic; does not trade machine analysis for human convenience

### Quorum Trigger Check

Attended session. The four mechanical triggers:

| # | Trigger | Fired? |
|---|---------|--------|
| 1 | Design-freeze items needing human ratification | **No** — all four freeze decisions are agent-resolvable and resolved above |
| 2 | Overrides shared machinery | **No** — every Conflict Surface row is "unchanged" or "reuse"; the one behavioural change removes rows that should never have been created |
| 3 | Touches cost/KPI semantics or banked-data schema | **No** |
| 4 | Load-bearing premises about external systems | **No** — every premise is in-repo and re-checkable (V1–V18) |

**No trigger fired → quorum skipped**, per the skill's "skip when ALL are false" rule. Noted here
rather than left implicit. The one cross-doc claim (the V14/V15 refutation above) is in-repo and
carries its own evidence rows.
