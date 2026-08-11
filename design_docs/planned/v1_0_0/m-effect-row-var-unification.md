# M-EFFECT-ROW-VAR-UNIFICATION — Discharge effect-row variables in the effect-checking pass

**Status**: Planned
**Target**: v1.0.0
**Priority**: P0 (High) — static effect soundness hole, not just DX friction (see V10/V11)
**Estimated**: 3–4 days (1d pass changes + 1d subsumption/diff semantics + 1–2d tests/fixtures/docs)
**Dependencies**: None (the type-layer machinery this relies on shipped in v0.29.0)
**Planner-Lane**: opus-required (touches shared row algebra in `internal/types/` and the effect-validation pass; the contamination history in V25 makes a mechanical port risky)
**Source**: GitHub issue [#616](https://github.com/sunholo-data/ailang/issues/616), re-reproduced and extended at `origin/dev` = `af6d56144`

## Problem Statement

A lowercase name in an effect row — `! {e}` — parses, type-checks, and *reads* as effect
polymorphism. The parser accepts it **deliberately** for that purpose (V18), and the type layer
already builds and unifies proper `RowVar` tails for it (V19, V16). But the separate
effect-checking pass (`internal/pipeline/validate_effects.go`) has no concept of a row-variable
tail (V20), producing four distinct wrong behaviors — all measured at base (`af6d56144`):

| # | Program shape (same module) | Expected | Actual at base | Row |
|---|---|---|---|---|
| 1 | Pure caller of a `! {e}` function, one call | accept | **reject**, with a blank diff and a "suggested fix" identical to the current signature | V3 |
| 2 | Caller declares any *wrong* non-empty row (`! {FS}` around an `{IO}` instantiation) | reject naming IO | **accept** | V8 |
| 3 | Pure caller of the *same* `! {e}` function, **two** calls | accept | accept — the rejection from row 1 **vanishes** when you call twice | V9 |
| 4 | Undeclared caller laundering IO through two row-var calls (`func f() -> int` performing IO) | reject naming IO | **accept**; with caps granted at the host boundary it executes the IO | V10, V11 |

Rows 2 and 4 make this a **soundness bug**: a function whose signature says *pure* can perform
IO, and `ailang check` says "No errors found". The runtime capability layer backstops it only
when the host does not grant caps (V11) — the static contract "the signature tells you the
effects" (Axiom A3) is broken. Row 1 is the DX face of the same defect that issue #616 reports.

Additionally, the row-1 error message is itself broken (the **blank-diff defect**, in scope
here): no `Missing effects:` line prints, and the `Suggested fix` is byte-identical to
`Current signature`. The code declares a failure while its own diff says nothing is missing —
an internal contradiction (V17, V21–V23).

**Impact**: any AI or human author who writes a same-module higher-order helper with `! {e}`
(exactly what the parser's own doc-comment recommends, V18) gets either a spurious,
unactionable rejection or a silent soundness hole. The stdlib already ships 13 row-variable
signatures (`std/list.ail` `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`, `std/stream.ail`,
`std/ai/streaming.ail`, `std/smoke.ail` — V27); they survive today only because cross-module
calls take a different, correct code path (V14–V16).

### What issue #616 gets wrong

The issue frames a dichotomy: "reject lowercase names (fastest)" or "implement unification".
The measurements refute that framing:

- **"Implement" is much cheaper than the issue assumes.** The parser (V18) and the type layer
  (V19) already implement row variables end-to-end; type inference already computes the correct
  instantiation at every call-site occurrence (proven by the let-alias probe V16, where routing
  the *same call* through the type-info path makes it check correctly). Exactly one pass — the
  effect checker — drops the information. This is "teach one pass", not "build a feature".
- **"Reject" is a much bigger decision than the issue presents.** It would delete a deliberate,
  partially-shipped capability, contradict the parser's own stated intent, and break the
  current stdlib: 13 shipped signatures in `std/` use row variables (V27). Rejecting lowercase
  at parse means rewriting `std/list`, `std/stream`, `std/ai/streaming`, `std/smoke` and every
  consumer.

## Verification Log

Base for every row: worktree at `origin/dev` = `af6d56144` (`git rev-parse HEAD` →
`af6d56144fff517a307d31473cd218a29e19ea8f`). Binary: `./bin/ailang` in the worktree
(`./bin/ailang --version` → "AILANG dev"); provenance verified *behaviorally* — its error text
and `DEBUG_EFFECTS` output match this tree's source line-for-line (V17 vs V23) — per the
known go-build-in-worktree version-stamp caveat.

All rows below were re-derived in this session. The controller's original V1–V5 rows were
re-run and matched, with two refinements found and recorded: (a) the effect pass's sibling
file `validate_effects_rows.go` was outside the controller's V4 grep scope (re-measured, also
zero row-var mentions — V20); (b) the controller's framing "every caller must also declare
`{e}`" is incomplete — *any* non-empty declared row passes, and two calls need no declaration
at all (V8–V10).

Test modules were written under `tmp/eff616/` (temp-path MOD010 auto-relax warnings elided
from outputs below; they are unrelated to effects).

**V1 — base + repro arm (a): row-var-only signature is accepted.**
```
$ cat > tmp/eff616/eff_a.ail <<'EOF'
module eff616/eff_a

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()
EOF
$ ./bin/ailang check tmp/eff616/eff_a.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V2 (arm b) — pure same-module caller rejected with blank diff.**
```
$ cat > tmp/eff616/eff_b.ail <<'EOF'
module eff616/eff_b

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()

func pureFn() -> int = 42

export func purePath() -> int = runIt(pureFn)
EOF
$ ./bin/ailang check tmp/eff616/eff_b.ail; echo RC=$?
Error: effect checking failed in tmp/eff616/eff_b: Effect checking failed for function 'purePath'
  Function uses effects not declared in signature


  Current signature: func purePath(...) -> T
  Suggested fix:     func purePath(...) -> T
RC=1
```
No `Missing effects:` line; suggested fix == current signature.

**V3 (arm c) — CONTROL: unknown uppercase effects are rejected at parse.** Proves the
instrument sees a positive and lowercase is genuinely special-cased, not falling through the
unknown-effect path.
```
$ # module eff616/eff_c with: export func runIt(f: () -> int ! {Bogus, Nonsense}) -> int ! {Bogus, Nonsense} = f()
$ ./bin/ailang check tmp/eff616/eff_c.ail; echo RC=$?
PAR_EFF002_UNKNOWN at tmp/eff616/eff_c.ail:3:35: unknown effect 'Bogus'
Suggestion: Did you mean 'IO'?
...(4 diagnostics total)...
RC=1
```

**V4 (arm f) — the lowercase near-miss typo guard works**: `! {io}` is rejected as a typo for
`IO`, not treated as a row variable.
```
$ # module eff616/eff_f with: export func f(g: () -> int ! {io}) -> int ! {io} = g()
$ ./bin/ailang check tmp/eff616/eff_f.ail; echo RC=$?
PAR_EFF002_UNKNOWN ... unknown effect 'io'  / Suggestion: Did you mean 'IO'?
RC=1
```

**V5 (arm d) — caller declaring `! {e}` is accepted** (the issue's "declare the phantom
effect" workaround).
```
$ # eff_b plus: export func wrapped() -> int ! {e} = runIt(pureFn)   [replacing purePath]
$ ./bin/ailang check tmp/eff616/eff_d.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V6 (arm e) — caller declaring `! {IO}` and passing an IO function is accepted — but by
accident.** (See V8: the acceptance does not depend on `IO` being the *right* label.)
```
$ cat > tmp/eff616/eff_e.ail <<'EOF'
module eff616/eff_e

import std/io (println)

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()

func ioFn() -> int ! {IO} {
  println("hi");
  42
}

export func ioPath() -> int ! {IO} = runIt(ioFn)
EOF
$ ./bin/ailang check tmp/eff616/eff_e.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V7 — SOUNDNESS (arm g): a *wrong* declared row also passes.** Same module as V6 but the
caller declares `! {FS}` around the `{IO}` instantiation:
```
$ # eff_e with:  export func laundered() -> int ! {FS} = runIt(ioFn)
$ ./bin/ailang check tmp/eff616/eff_g.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V8 (arm k) — calling twice makes the false rejection vanish.**
```
$ # eff_b with:  export func purePathTwice() -> int = runIt(pureFn) + runIt(pureFn)
$ ./bin/ailang check tmp/eff616/eff_k.ail; echo RC=$?
✓ No errors found!
RC=0
```
(One call: rejected, V2. Two calls: accepted. Mechanism: `UnionEffectRows` of two non-nil rows
drops tails and returns `nil` for label-empty results — V23.)

**V9/V10 (arm l) — full laundering: IO behind a *pure* signature passes `check`.**
```
$ cat > tmp/eff616/eff_l.ail <<'EOF'
module eff616/eff_l

import std/io (println)

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()

func pureFn() -> int = 42

func ioFn() -> int ! {IO} {
  println("hi");
  42
}

export func bothUndeclared() -> int = runIt(pureFn) + runIt(ioFn)
EOF
$ ./bin/ailang check tmp/eff616/eff_l.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V11 — runtime backstop probe on arm l.** Without caps the capability layer catches it; with
caps granted the "pure" function performs IO:
```
$ ./bin/ailang run -entry bothUndeclared tmp/eff616/eff_l.ail >/dev/null 2>&1; echo RC=$?
RC=1     # "Error: execution failed: effect 'IO' requires capability, but none provided"
$ ./bin/ailang run -entry bothUndeclared -caps IO tmp/eff616/eff_l.ail
hi
84
```

**V12 (arm m) — mixed row `! {IO, e}`: concrete labels still propagate.** A caller declaring
`{IO}` passes; an undeclared caller fails *with a correct* `Missing effects: IO` line (the
concrete half of the row works; only the tail is mishandled).
```
$ cat > tmp/eff616/eff_m.ail <<'EOF'
module eff616/eff_m

import std/io (println)

export func withLog(f: () -> int ! {e}) -> int ! {IO, e} {
  println("calling");
  f()
}

func pureFn() -> int = 42

export func mixed() -> int ! {IO} = withLog(pureFn)

export func mixedUndeclared() -> int = withLog(pureFn)
EOF
$ ./bin/ailang check tmp/eff616/eff_m.ail; echo RC=$?
Error: ... Effect checking failed for function 'mixedUndeclared'
  Missing effects: IO
  Current signature: func mixedUndeclared(...) -> T
  Suggested fix:     func mixedUndeclared(...) -> T ! {IO}
RC=1
```

**V13 (arm n) — a declared row variable does NOT absorb concrete effects** (correct today;
must be preserved):
```
$ # module eff616/eff_n:  export func leaky() -> int ! {e} { println("hi"); 42 }
$ ./bin/ailang check tmp/eff616/eff_n.ail; echo RC=$?
Error: ... Missing effects: IO
RC=1
```

**V14 (arm h) — cross-module pure caller of stdlib `mapE` is accepted (correct path).**
```
$ cat > tmp/eff616/eff_h.ail <<'EOF'
module eff616/eff_h

import std/list (mapE)

export func doubled() -> [int] = mapE(\x. x * 2, [1, 2, 3])
EOF
$ ./bin/ailang check tmp/eff616/eff_h.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V15 (arm i) — cross-module laundering IS caught (correct path).**
```
$ cat > tmp/eff616/eff_i.ail <<'EOF'
module eff616/eff_i

import std/io (println)
import std/list (forEachE)

func printOne(x: int) -> () ! {IO} = println(show(x))

export func laundered(xs: [int]) -> () ! {FS} = forEachE(printOne, xs)
EOF
$ ./bin/ailang check tmp/eff616/eff_i.ail; echo RC=$?
Error: ... Effect checking failed for function 'laundered'
  Missing effects: IO
  Current signature: func laundered(...) -> T ! {FS}
  Suggested fix:     func laundered(...) -> T ! {FS, IO}
RC=1
```

**V16 (arm j) — the let-alias probe: occurrence-level type info is correctly instantiated,
even same-module.** This is the keystone for the proposed fix: aliasing the callee routes the
call around the `declaredEffects` map into the `typeInfo` path, and the exact program from V2
then checks clean:
```
$ cat > tmp/eff616/eff_j.ail <<'EOF'
module eff616/eff_j

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()

func pureFn() -> int = 42

export func purePath() -> int = {
  let g = runIt;
  g(pureFn)
}
EOF
$ ./bin/ailang check tmp/eff616/eff_j.ail; echo RC=$?
✓ No errors found!
RC=0
```

**V17 — `DEBUG_EFFECTS` trace of arm b: every row prints `[]`, and the check still fails.**
The pass's own debug instrumentation (`formatRow`, labels-only) cannot see the poisonous tail:
```
$ DEBUG_EFFECTS=1 ./bin/ailang check tmp/eff616/eff_b.ail 2>&1 | grep -A12 purePath | head -14
[DEBUG_EFFECTS] === Validating Let binding: purePath ===
[DEBUG_EFFECTS]   Declared effects: []
[DEBUG_EFFECTS]     App (function application)
[DEBUG_EFFECTS]       Callee declared effects (from signature): []
[DEBUG_EFFECTS]     Var(pureFn) -> []
[DEBUG_EFFECTS]       App total effects: []
[DEBUG_EFFECTS]   Required effects: []
...then: Effect checking failed for function 'purePath'
```

**V18 — the parser accepts lowercase deliberately; its comment states the intent is
polymorphism, with a typo guard.** Read `internal/parser/parser_effect.go:64-77`:
```
$ grep -n "Row variables enable\|isRowVar :=" internal/parser/parser_effect.go
66:  // Row variables enable effect polymorphism: func mapE[a, b, e](f: (a) -> b ! {e}, ...) -> [b] ! {e}
69:  isRowVar := len(effectName) > 0 && effectName[0] >= 'a' && effectName[0] <= 'z'
```
Lines 70–77 downgrade `isRowVar` when the name case-insensitively matches a known effect
(measured behavior: V4).

**V19 — the type layer is row-variable aware.** Non-test call sites of `isEffectRowVar`:
```
$ grep -rn "isEffectRowVar" internal/ cmd/ --include="*.go" | grep -v _test.go
internal/types/effects.go:11/12: (definition)
internal/types/effects.go:311:   (ElaborateEffectRow — separates row vars, builds &RowVar tail)
internal/types/effects.go:385:   (ElaborateEffectRowWithBudgets — same)
internal/types/typechecker.go:226: (astTypeToType FuncType effects — builds &RowVar tail)
```
Read confirmation: `ElaborateEffectRow` (`effects.go:302`) and `...WithBudgets`
(`effects.go:369`) return, for `{e}`, a **non-nil** `&Row{Labels: <empty map>, Tail:
&RowVar{Name: "e", Kind: EffectRow}}` — nil is returned only for a fully absent annotation.
This non-nil-but-label-empty row is the poison the effect checker cannot see.

**V20 — the effect-checking pass has zero row-variable handling (with control and one
nuance).**
```
$ grep -c "RowVar\|isEffectRowVar\|rowVar" internal/pipeline/validate_effects.go
0
$ grep -c "RowVar\|isEffectRowVar\|rowVar" internal/pipeline/validate_effects_rows.go
0
$ grep -c "Effect" internal/pipeline/validate_effects.go          # control: the grep sees positives
122
```
Nuance (found in re-derivation, refining the controller's V4): `validate_effects.go:162` does
read `declared.Tail == nil` — the **lambda** sub-pass deliberately skips open-row annotations
(M-EFFECT-ROW-SHOW-INTERP, #386). That is the file's only tail touch; the top-level
declaration path (`validateDecl`, lines 203–255, and the whole collector) has none.

**V21 — `SubsumeEffectRows` nil-semantics are the false-reject site.** Read
`internal/types/effects.go:624-634`:
```go
func SubsumeEffectRows(a, b *Row) bool {
    if a == nil { return true }
    if b == nil { return a == nil }   // <-- non-nil required vs pure declared: false, tail or not
    diff := DiffEffectRows(a, b)
    return len(diff.Missing) == 0 && len(diff.ParamMismatches) == 0
}
```
A required row with empty labels and a tail is non-nil → declared-pure caller fails (V2). The
same call with any non-nil declared row diffs **labels only** → passes (V7).

**V22 — `DiffEffectRows` is labels-only.** Read `internal/types/effect_subsumption.go:57-…`:
the loop iterates `required.Labels`; `Tail` is never consulted. For arm b, `required.Labels`
is empty → `Missing` is empty.

**V23 — the blank-message mechanism is fully pinned.** Read
`internal/pipeline/validate_effects.go:520-563` and `internal/types/effects.go:511-617`:
`formatEffectError` calls `DiffEffectRows` (empty per V22, so `writeEffectDiff` prints
nothing — it only prints when `len(diff.Missing) > 0`), then `UnionEffectRows(declared=nil,
required)` returns `required` unchanged, and `FormatEffectRow` returns `""` for a label-empty
row → `Suggested fix` == `Current signature`. Also read: `UnionEffectRows` with two non-nil
inputs builds a fresh row with `Tail: nil` and returns `nil` when merged labels are empty —
the arm-k "error vanishes on the second call" mechanism (V8).

**V24 — the collector preserves the tail into `required`.** Read
`internal/pipeline/validate_effects_rows.go:13-30` (`cloneEffectRow` copies `Tail`) and
`:74-96` (`unionRequiredEffectRows(a, nil)` = `cloneEffectRow(a)`, tail preserved — the
single-call arm-b path). The App case (`validate_effects.go:337-360`) prefers
`declaredEffects[funcVar.Name]` (cloned) for same-module `*core.Var` callees; `typeInfo` is
consulted only when the name is absent from the map.

**V25 — why the pass prefers declared rows over typeInfo (contamination history).**
```
$ git log --format="%h %ad %s" --date=short -S "usedDeclared" -- internal/pipeline/validate_effects.go
71b610d68 2025-12-24 Fix effect checker bug: pure functions incorrectly required IO
```
The commit message: CoreTypeInfo could hold "contaminated types for locally-defined functions"
(recursive-call extraction); the fix routed same-module callees through declared signatures.
Any fix here must not simply revert that.

**V26 — blast radius of changing the shared row algebra**: every non-test caller of
`SubsumeEffectRows`/`DiffEffectRows` is inside the validation pass itself:
```
$ grep -rn "SubsumeEffectRows\|DiffEffectRows" internal/ cmd/ --include="*.go" | grep -v _test.go
internal/pipeline/validate_effects.go:164, :221, :244, :521, :552
internal/types/effects.go:620/624/632 (definition), internal/types/effect_subsumption.go:56/57 (definition)
```

**V27 — the stdlib ships row-variable signatures today** (so parse-time rejection would break
shipped code):
```
$ grep -rn '! {e}\|, e}' std/ --include="*.ail" | wc -l
13
```
Hits include `std/list.ail:217,228,239,250,261` (`mapE`, `filterE`, `foldlE`, `flatMapE`,
`forEachE`), `std/stream.ail:100,146,178,237` (mixed rows `! {Stream, e}`),
`std/ai/streaming.ail:174`, `std/smoke.ail:42,61`.

**V28 — existing test coverage reaches only the cross-module path** (negative claim with
control):
```
$ grep -rn '! {e' internal/pipeline/*_test.go | wc -l
0
$ grep -c '! {IO}' internal/pipeline/effect_mode_subsumption_test.go     # control
2
```
The one row-var pipeline test, `TestEffectRowVariableImportsStillValidate`
(`internal/pipeline/effect_mode_subsumption_test.go:174`), imports `std/list (mapE)` —
cross-module, i.e. the already-correct path. No test constructs a same-module caller of a
row-var function. This is why "the suite is green" is vacuous for this defect at base.

**V29 — baseline gates are green at base** (so suite-green ACs measure the change only in
combination with the new fixtures that fail at base):
```
$ go test ./internal/pipeline/ -count=1   → ok ... 4.812s
$ go test ./internal/types/ -count=1      → ok ... 0.300s
```

**V30 — instrument limitation found while selecting fixtures**: `./bin/ailang check
std/list.ail` exits 1 at base with `module name contains invalid characters ... list.ail` —
a module-path resolution artifact of checking a std file directly, unrelated to effects. It is
therefore NOT a usable AC gate; the runnable example is:
```
$ ./bin/ailang check examples/runnable/effectful_list_t1_mapE_basic.ail >/dev/null 2>&1; echo RC=$?
RC=0
```

**V31 — no duplicate/covering design doc.** `design_docs/planned/` contains no row-var/effect
doc (grep for `616|row variable|rowvar` matches only unrelated ollama-streaming docs). The
closest prior work, `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md`, is a
**different bug** (TYPE-layer unification of lambda closed rows against concrete rows, e.g.
`{IO, Stream}`; its own reproducer errors in *type unification*, not effect checking). Its
status header still reads "Planned" despite sitting in `implemented/` — known stale-header
class; do not trust status headers as facts.

## Root-Cause Mechanism (one paragraph)

`ValidateEffects` builds `declaredEffects[name]` from surface annotations via
`ElaborateEffectRowWithBudgets`, which represents `{e}` as a **non-nil row with empty labels
and a `RowVar` tail** (V19). The App collector substitutes that *declared* row for same-module
callees (V24, the 71b610d68 contamination fix, V25) — it never consults the type checker's
correctly-instantiated occurrence row (which V16 proves exists). The tail then rides into
`required`, where every downstream consumer disagrees about what it means: `SubsumeEffectRows`
treats the non-nil row as effectful (false reject vs a pure declaration, V2) while
`DiffEffectRows`/`FormatEffectRow`/`formatRow` see only labels (false accept vs any non-empty
declaration V7, blank error message V17/V23, invisible in debug traces V17), and
`UnionEffectRows` silently deletes the tail and collapses to `nil` (error vanishes on the
second call, V8; full laundering, V9–V11).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Teach the effect checker (Option A) vs reject lowercase at parse (Option B) | B deletes a shipped capability and breaks 13 stdlib signatures (V27); A is the only option compatible with current std | human (one word — default is A, see "Proposed direction") | design | high |
| Tail resolution source: type checker's instantiated row at the callee **occurrence node** | Wrong node (decl node instead of occurrence) re-imports the 71b610d68 contamination class (V25); occurrence-level correctness is proven (V16) | agent (constrained: occurrence node only) | design | med |
| Missing occurrence type-info = loud internal error, not silent tail-drop or tail-keep | A silent fallback is exactly how this bug survived — and "just strip tails" is *unsound* (it re-opens V7/V9 laundering); repo rule: no silent fallbacks | human (ratify the fail-loud) | design | med |
| An effect-check failure with an empty diff becomes structurally impossible (or fails loudly as an internal invariant violation) | This is the general form of the V5/blank-message defect; without the invariant the next tail-like bug is again invisible | agent | compile | low |
| Declared row var still does NOT absorb concrete required effects | Preserves V13 semantics; changing it would silently weaken the effect system | agent (keep as-is) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Direction ratified: **teach** (Option A), not **reject** (Option B). One word from a human settles it; everything below assumes A.
- [ ] Tail resolution reads the **occurrence node's** type info (`typeInfo.Get(e.Func.ID())` inside the App case), never the callee's declaration node.
- [ ] Missing/ill-typed occurrence info fails loudly (named function + call site in the message), never silently drops or keeps the tail.
- [ ] `EffectRowDiff` gains an explicit undischarged-row-var field; `writeEffectDiff` always prints *something* on failure; a genuinely empty diff on a failed check returns an "internal error — report this bug" error.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of the new row-var diagnostic and whether it gets a structured code — **agent may choose**; if a code is added it MUST be verified unallocated first (`grep -rn "<code>" internal/ cmd/`); note the current effect-check failure text has no code at all.
- Whether tail-vs-tail subsumption matches by name or by mere presence — **agent may choose after writing the M2 test**; name-match is the safe default since post-substitution both tails live in the same signature's scope (see Testing Strategy, mutation 4).
- Whether `formatRow` (the DEBUG_EFFECTS printer) learns to render tails (`[| e]`) — **agent may choose**; strongly recommended given V17.
- Test fixture file naming/organization under `examples/runnable/` — **agent may choose**.
- Whether arm-e-style accidental passes get a changelog callout — **human at review**.

## Solution Design

### Overview

Make the effect-validation pass row-variable aware in three places, without touching the
parser or the type layer (which are already correct):

1. **Discharge tails at call sites** (the actual fix). When the App collector substitutes a
   same-module callee's *declared* row and that row has a `Tail`, resolve the tail from the
   type checker's already-computed instantiation at that occurrence and use the resolved
   concrete labels (union of the declared row's concrete labels and the instantiated labels).
   V16 proves the instantiated row exists and is correct at occurrence granularity. Fully
   concrete declared rows take the existing path untouched — that keeps the 71b610d68 fix
   intact (V25).
2. **Give tails defined subsumption semantics** so any tail that legitimately survives (a
   row-polymorphic function's own body, arm a/V1) is handled explicitly rather than by
   accident: `required.Tail != nil` is subsumed only by a declared row with a matching tail;
   it is *reported by name* against a closed or pure declaration. A row with empty labels and
   nil tail normalizes to pure.
3. **Make the blank message impossible** (the V5 defect, in scope): `EffectRowDiff` gains an
   `UndischargedRowVars []string` field populated from tails; `writeEffectDiff` prints it;
   `formatEffectError` returns an internal-invariant error ("effect check failed but the diff
   is empty — this is a compiler bug, please report") if a failure ever again produces an
   empty diff; the suggested-fix line is only printed when it differs from the current
   signature.

### Architecture

**Components:**

1. **Tail discharge in the App collector** (`internal/pipeline/validate_effects.go`, App case
   at lines 337–360): after `calleeEffects = cloneEffectRow(declaredEff)`, if
   `calleeEffects.Tail != nil`, fetch `typeInfo.Get(e.Func.ID())`; on a `*types.TFunc2` (or
   `TApp` wrapping one — reuse `extractEffectFromType`'s shape handling), replace
   `calleeEffects` with `union(concreteLabels(declaredEff), instantiatedRow)`; the result's
   tail is whatever the instantiation left (nil when closed; the enclosing function's row var
   when the call site is itself row-polymorphic). If the lookup fails or yields no function
   type: return a loud error naming the callee and call site (no silent fallback — Critical
   Principle 2).
2. **Tail-aware subsumption** (`internal/types/effects.go:624` `SubsumeEffectRows`,
   `internal/types/effect_subsumption.go:57` `DiffEffectRows`): normalize label-empty+tail-nil
   rows to nil at entry; teach `DiffEffectRows` to emit `UndischargedRowVars` when
   `required.Tail` is not covered by `declared.Tail`; `SubsumeEffectRows` fails when that
   field is non-empty. Concrete-label logic is unchanged (preserves V12/V13/V15 behavior).
   All callers are inside this pass (V26), so the semantic change cannot leak elsewhere.
3. **Error-format invariant** (`internal/pipeline/validate_effects.go:520-563`): print
   `Undischarged effect row variable(s): e — the callee's '{e}' could not be discharged at
   this call site` (wording deferred); empty-diff-on-failure → internal-invariant error;
   suppress the suggested-fix line when identical to the current signature.
4. **Fixtures and tests** (new `internal/pipeline/effect_rowvar_discharge_test.go`, new
   runnable example): every arm from the Verification Log that changes or must not change
   becomes a pinned test.

### Implementation Plan

**Phase 1: Semantics + plumbing (~1 day)**
- [ ] `EffectRowDiff.UndischargedRowVars` + `DiffEffectRows` tail handling + normalization helper
- [ ] `SubsumeEffectRows` consumes the new field; unit tests directly on rows (incl. the exact poisonous shape from V19: non-nil, label-empty, tail `e`)
- [ ] `writeEffectDiff` prints the new field; `formatEffectError` invariant guard + suggested-fix suppression

**Phase 2: Tail discharge at App sites (~1 day)**
- [ ] Occurrence-node resolution in the App case, gated on `calleeEffects.Tail != nil`
- [ ] Loud-failure path for missing occurrence info
- [ ] `formatRow` tail rendering for DEBUG_EFFECTS (recommended)

**Phase 3: Fixtures, regression sweep, docs (~1–2 days)**
- [ ] Port arms a,b,d,e,g,k,l,m,n,h,i,j into pipeline tests (same-module AND cross-module); base-red assertions for b (accept), g/l (reject naming IO)
- [ ] New runnable example `examples/runnable/effect_row_var_pure_caller.ail` + manifest entry
- [ ] `make test`, `make verify-examples` (expect manifest drift class, not type regressions, if red)
- [ ] CHANGELOG.md entry; `docs/LIMITATIONS.md` update if row-var limitations are listed there; close-out comment on #616

### Files to Modify/Create

**Modified files:**
- `internal/pipeline/validate_effects.go` (+70/−15 LOC) — App-case tail discharge, error-format invariant, DEBUG tail rendering
- `internal/pipeline/validate_effects_rows.go` (+15/−0 LOC) — normalization helper (label-empty + tail-nil → nil)
- `internal/types/effect_subsumption.go` (+35/−5 LOC) — `UndischargedRowVars` in `EffectRowDiff` + `DiffEffectRows` tail logic
- `internal/types/effects.go` (+15/−5 LOC) — `SubsumeEffectRows` tail semantics + entry normalization

**New files:**
- `internal/pipeline/effect_rowvar_discharge_test.go` (~250 LOC) — the arm matrix
- `examples/runnable/effect_row_var_pure_caller.ail` (~15 LOC) + `examples/manifest.json` entry

## Conflict Surface

This design touches `internal/types/` (row algebra) and `internal/pipeline/` (validation
pass). No parser, lexer, AST, elaboration, codegen, eval, or runtime-effects change.

### Syntactic positions touched

None. The grammar is untouched; `! {e}` already parses (V18) and continues to parse
identically. This design changes *semantic* positions only.

### Semantic positions touched, and what else lives there

| Position | Existing occupant | Interaction | Measured by |
|---|---|---|---|
| `SubsumeEffectRows` / `DiffEffectRows` semantics | 5 call sites, all inside `validate_effects.go` (decl path :221/:244, lambda path :164, formatting :521/:552) | Tail-aware change reaches exactly these; no other package consumes them | V26 |
| App-case callee-row selection (`validate_effects.go:337-360`) | The 71b610d68 contamination fix: same-module callees read *declared* rows precisely to avoid CoreTypeInfo poisoning of recursive/local calls | Changed ONLY when the declared row has a tail; fully concrete rows keep the exact current path, so that commit's regression class stays fixed | V25 + AC12/AC13 |
| Lambda sub-pass (`validate_effects.go:162`) | Enforces only CLOSED declared lambda rows (`declared.Tail == nil`) — open rows deliberately skipped (#386) | Unchanged. Note: after this fix `required` reaching :164 can newly contain resolved labels where it silently carried/dropped tails before; the closed-row gate’s semantics are unaffected, but the M3 test matrix must include an inline-lambda arm | V20 |
| Ghost-effect erasure (`eraseGhostEffects`, runs on `required` before subsumption) | Label-based removal (`Debug`) | Operates on labels only; tails pass through it untouched — no interaction, but pin with one test (mixed `{Debug, e}` callee) | code read, `validate_effects.go:31-40` |
| Effect budgets/params on rows (`Budgets`/`Params`, mode subsumption) | `DiffEffectRows` param logic; `unionRequiredEffectRows` conflict-preserving param merge | Untouched by tail logic (params key off labels). Budgets on a row-var tail are meaningless today and remain out of scope | V22 code read |
| Cross-module callees (`VarGlobal` → typeInfo path) | Already correct (V14/V15) | Unchanged — the new resolution applies only when the declared-map path is taken (`*core.Var` hit) | V14/V15 as pinned ACs |
| `iface` freezing / formatter / elaboration (`internal/iface/builder.go` 25 RowVar mentions, `internal/format/types.go` 8, `internal/elaborate/file_funcs.go` 2) | Serialize/print/carry row-var signatures | Read-only consumers of the same `Row` struct; no struct field is changed (the new field is on `EffectRowDiff`, a validation-only type) | `grep -rn RowVar internal/ --include="*.go" \| grep -v _test` file census |
| Runtime capability checks | Label/capability-based, no row vars (`internal/effects/` absent from the RowVar census) | Unchanged; remains the backstop measured in V11 | same census |

### Disambiguation strategy

Not applicable at the token level (no grammar change). At the semantic level the "which row do
we trust" rule becomes: *declared concrete labels* (unchanged, per 71b610d68) **plus** the
type checker's occurrence-level instantiation *only for the tail part*, with a loud error when
the latter is unavailable. The rule is decidable locally at each App node.

### Programs that MUST still work

Regression fixtures (all measured at base in the Verification Log):

1. `std/list.ail:217-261` — `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE` row-var signatures, exercised via `examples/runnable/effectful_list_t1_mapE_basic.ail` (base RC=0, V30) and `TestEffectRowVariableImportsStillValidate` (base green, V28/V29)
2. `std/stream.ail:100,146,178,237` + `std/ai/streaming.ail:174` — mixed rows `! {Stream, e}` (the arm-m shape, V12)
3. Arm a (`runIt`'s own body: tail-for-tail, V1) — must keep passing under explicit tail subsumption
4. Arm d (caller declares `! {e}`, V5) and arm e (caller declares the correct `! {IO}`, V6) — keep passing, now for the right reason
5. Arms h/i (cross-module accept/reject, V14/V15) and arm n (`{e}` doesn't absorb IO, V13) — byte-compatible `Missing effects:` lines
6. The 71b610d68 regression shape (pure same-module functions called after `println` chains) — pinned by the existing suite (`internal/pipeline` green at base, V29) plus one dedicated concrete-row recursive-call test

### What deliberately changes

- **Arm b/k class (false rejects) become accepts** — the #616 headline fix.
- **Arm g/l class (laundering accepts) become rejects** naming the concrete effect. This is an
  intentional breaking change for previously-"valid" programs — but every such program is
  unsound (its signature lies about its effects), and `grep -rn '! {e' examples/ std/` shows
  the shipped corpus contains no same-module row-var caller that would newly fail (stdlib
  row-var functions are only called cross-module from examples). Migration path: declare the
  real effects, exactly as the new message instructs.
- **The blank error message becomes impossible**; failures always name at least one missing
  effect, param mismatch, or undischarged row variable.
- Anything else that breaks is a regression, not an intentional change.

## Examples

### Before (base, measured) → After (this design)

```ailang
module eff616/eff_b

export func runIt(f: () -> int ! {e}) -> int ! {e} = f()

func pureFn() -> int = 42

export func purePath() -> int = runIt(pureFn)
```
- Before: `Effect checking failed for function 'purePath'` with empty diff and
  `Suggested fix` == `Current signature` (V2).
- After: `✓ No errors found!` — `e` is instantiated to the empty row at the call site.

```ailang
export func laundered() -> int ! {FS} = runIt(ioFn)   -- ioFn: () -> int ! {IO}
```
- Before: `✓ No errors found!` (V7) — and with two calls, even a fully pure signature passes
  and executes IO under granted caps (V9–V11).
- After: rejected with `Missing effects: IO` and suggested fix `! {FS, IO}` — identical in
  shape to the cross-module message that already works today (V15).

## Acceptance Criteria

Every AC names its base-state measurement (rule: a gate red/green at base measures the repo,
not the change). "Suite green" appears only in combination with new fixtures that FAIL at
base, so it cannot be vacuously satisfied (V28 proves the current suite does not reach this
defect).

- [ ] **AC1** (arm b file, V2): `ailang check` exits **0**. Base: exits 1 with blank diff.
- [ ] **AC2** (arm g file, V7): exits **1**, output contains `Missing effects: IO`. Base: exits 0.
- [ ] **AC3** (arm l file, V9): exits **1**, output contains `Missing effects: IO`. Base: exits 0.
- [ ] **AC4** (arm k file, V8): still exits **0** (guards against over-rejection). Base: exits 0.
- [ ] **AC5** (arm a file, V1): still exits **0** (tail-for-tail in the row-poly body). Base: exits 0.
- [ ] **AC6** (arm d + arm e files, V5/V6): still exit **0**. Base: exit 0.
- [ ] **AC7** (arm n file, V13): still exits **1** with `Missing effects: IO` (declared `{e}` must not absorb concrete effects). Base: exits 1 with that line.
- [ ] **AC8** (arm m file, V12): `mixed` accepted, `mixedUndeclared` rejected with `Missing effects: IO`. Base: same (pins the mixed-row shape).
- [ ] **AC9** (arms h/i files, V14/V15): cross-module behavior unchanged — h exits 0; i exits 1 with `Missing effects: IO`. Base: same.
- [ ] **AC10 (blank-diff invariant, the V5 defect)**: (a) a unit test feeds the exact poisonous shape (`required = &Row{Labels: empty, Tail: &RowVar{"e"}}`, `declared = nil`) through the pass's error path and asserts the message names `e`; (b) a unit test asserts that a forced empty-diff failure returns the internal-invariant error, not a blank message. Base: the blank message is producible (V2/V17) and no such guard exists (V20).
- [ ] **AC11**: `go test ./internal/pipeline/ ./internal/types/ -count=1` green, INCLUDING the new tests from AC1–AC3/AC10 that fail at base. Base: green without them (V29 — 4.812s / 0.300s).
- [ ] **AC12**: `./bin/ailang check examples/runnable/effectful_list_t1_mapE_basic.ail` exits 0. Base: exits 0 (V30). (Note: `ailang check std/list.ail` is NOT a gate — red at base for an unrelated module-name reason, V30.)
- [ ] **AC13**: the 71b610d68 concrete-row regression class stays fixed — dedicated test: same-module recursive/pure functions with concrete declared rows called after `println` chains still validate. Base: green (V29 subsumes it).
- [ ] **AC14**: documentation updated — CHANGELOG.md entry; #616 closed with a comment linking the arms; `docs/LIMITATIONS.md` row-var entry updated/removed if present (check at implementation time).

## Testing Strategy

**Unit tests (types):** `DiffEffectRows`/`SubsumeEffectRows` on constructed rows — poisonous
shape (label-empty + tail), tail-vs-tail same name, tail-vs-tail different name, tail vs
closed, tail vs nil, normalization (label-empty + tail-nil ≡ pure), params/budgets untouched
by tail logic.

**Integration tests (pipeline):** the arm matrix (AC1–AC9) as source-level `check` tests, same
harness style as `check386`/`TestEffectRowVariableImportsStillValidate`
(`internal/pipeline/effect_mode_subsumption_test.go:174`). Plus: inline-lambda arm (lambda
sub-pass interplay, Conflict Surface row 3) and mixed `{Debug, e}` ghost-erasure arm.

**Regression-surface tests:** one per "Programs that MUST still work" entry (fixture list
above; the stdlib entries via the existing example + import test).

**Mutation kill matrix** (each observable is DOWNSTREAM of the mechanism — check exit codes
and message content, never internal state set alongside the mutated code):

| Mutation | Killed by | Downstream observable |
|---|---|---|
| Resolve the tail from the callee's *declaration* node instead of the occurrence node | Two-instantiation test: `func both() -> int ! {IO} = runIt(pureFn) + runIt(ioFn)` must pass AND `func bothPure() -> int = runIt(pureFn) + runIt(ioFn)` must fail naming exactly `IO` | `ailang check` rc + `Missing effects:` line content; a shared-node mutation collapses the two instantiations and flips one of the two verdicts |
| "Fix" by silently stripping tails instead of resolving them | AC2/AC3 | arm g/l must exit 1 naming IO; tail-stripping re-accepts them (this is why strip-the-tail is not a fix — it is the laundering bug with better DX) |
| Resolution falls back silently when occurrence info is missing | Loud-failure unit test with a stubbed typeInfo that returns no entry | error text names the callee and call site; silent fallback yields rc=0 |
| Tail subsumption matches by presence, not name | Test where the enclosing function declares `! {f}` but the required tail post-substitution is a different var — construct via a helper whose row var cannot unify with `f` (implementer sketches during M2; if the type layer makes this unrepresentable, document why and drop the name-match distinction as unreachable) | rc flips on the mismatched-name arm |
| Empty-diff guard removed | AC10(b) | the invariant error text disappears → test fails |
| Suggested-fix suppression removed | AC10(a) asserts fix line ≠ signature line on the row-var error | message content |

**Manual testing:** re-run the full arm matrix from the Verification Log; re-run the V11
runtime probe (uncapped run of arm l must now be unreachable — the file no longer passes
`check`).

## Non-Goals

- **Type-layer changes** — inference/unification of effect rows already works (V16, V19) and shipped in v0.29.0 (`m-effect-row-poly-params`); nothing there changes.
- **Parser changes** — `! {e}` syntax, the typo guard (V4), and PAR_EFF002 for uppercase unknowns (V3) are all untouched.
- **Runtime capability semantics** — the V11 backstop behavior is unchanged.
- **Budgets/params on row-var tails** (`! {e @limit=5}`-class questions) — out of scope; today's label-keyed budget logic is preserved as-is.
- **The `m-effect-row-poly-params` residue** (lambda closed-row unification against concrete rows) — different bug, already implemented; its stale "Planned" header is a docs chore, not this sprint.
- **`formatRow`-style debug polish beyond tail rendering** — nice-to-have only.

## Timeline

**Day 1**: Phase 1 (diff/subsumption semantics + error invariant, unit tests first).
**Day 2**: Phase 2 (App-case tail discharge + loud-failure path), arm matrix goes green.
**Day 3**: Phase 3 (fixtures, example + manifest, mutation matrix, `make test` + `make verify-examples`, docs).
**Day 4 (buffer)**: conflict-surface sweep (inline-lambda + ghost-effect arms), CHANGELOG, #616 close-out.

Total: 3–4 days — sprint-sized. Evidence this is not bigger: the fix is confined to two files
of row algebra + one pass (V26), the resolution source already exists and is correct (V16),
and cross-module behavior needs zero change (V14/V15).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Re-importing the 71b610d68 contamination class by consulting typeInfo | High | Consult typeInfo ONLY when the declared row has a tail; concrete-row path byte-identical; AC13 pins the original regression shape |
| Occurrence typeInfo missing/odd shape for some App forms (e.g. `TApp`-wrapped callees) | Medium | Reuse `extractEffectFromType`'s shape handling; loud error (never silent) surfaces any gap immediately in CI via the arm matrix |
| Newly-rejected unsound programs in the wild (arm g/l class) | Medium | Intentional (see "What deliberately changes"); message names the exact missing effect + fix; shipped corpus measured clean |
| Tail name-matching subtleties across nested row-poly functions | Medium | Explicit tail-vs-tail tests in M2; the mutation-4 arm probes name capture; deferred decision documents the fallback |
| Perf: one extra typeInfo lookup per row-var call site | Low | Lookup is a map get; gated on `Tail != nil`, which is rare (13 stdlib sites) |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism introduced; validation stays deterministic |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | **+2** | Closes a measured hole where signatures lie about effects (V9–V11); restores "the signature tells you the effects" |
| A4: Explicit Authority | +1 | Static layer again matches the capability layer instead of deferring to the runtime backstop |
| A5: Bounded Verification | +1 | Effect check becomes locally decidable at each call site (occurrence-level discharge) |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Removes a blank, self-contradictory diagnostic that no agent can act on (V2); errors become mechanically actionable |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No impact |
| A10: Composability | +1 | Row-polymorphic stdlib combinators become usable same-module, matching their cross-module behavior |
| A11: Structured Failure | +1 | New structured diff field; empty-diff failures impossible by construction |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +7** ✅ Proceed.

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): removes hidden side effects; introduces none
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): optimizes for machine-actionable diagnostics

## Proposed Direction (and what would make it wrong)

**Direction: Option A — teach the effect-checking pass to discharge row-variable tails using
the type checker's occurrence-level instantiation, give tails explicit subsumption semantics,
and make empty-diff failures structurally impossible.**

Defense, from the evidence: the parser wants polymorphism (V18), the type layer delivers it
(V19), the instantiation is provably correct at exactly the granularity the pass needs (V16),
the correct behavior already exists on the cross-module path to copy from (V14/V15), and the
alternative — rejection at parse — breaks 13 shipped stdlib signatures (V27). The blast radius
of the semantic change is five call sites in one file (V26).

**What would have to be true for this to be wrong:**
- If occurrence-level typeInfo were *not* reliably instantiated for same-module `*core.Var`
  callees, M1's resolution source would not exist. V16 (let-alias passes) is strong but
  indirect evidence; a targeted probe in M1 (log the fetched row for arm b's `runIt`
  occurrence) confirms or refutes it on day 1, and the loud-failure path means being wrong
  here is noisy, not silent.
- If the shipped ecosystem (beyond this repo) contained load-bearing programs in the arm-g/l
  class, the deliberate breakage would need a migration window. The in-repo corpus is
  measured clean; external corpora (motoko fork, demos) should be spot-checked in Phase 3.
- If Mark wants the language to *not* have effect-row variables at all, Option B is a product
  decision above this doc's pay grade — but then the stdlib rewrite must be scoped in the same
  breath, and #616's "fastest" label on it is wrong.

**Does this need a human?** Only for the one-word ratification in Design Freeze: **"teach"**
(proceed with this doc) or **"reject"** (kill row vars; requires a stdlib-rewrite companion
doc). Everything else is settled by the measurements above. Default on silence: teach.

## References

- **Issue**: [#616 — Effect row variables parse but never unify](https://github.com/sunholo-data/ailang/issues/616)
- **Related (DISTINCT) prior work**: `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md` — type-layer lambda/closed-row unification, shipped v0.29.0; stale "Planned" header noted in V31
- **Contamination history**: commit `71b610d68` (2025-12-24) — why same-module callees read declared rows
- **Lambda sub-pass provenance**: M-EFFECT-ROW-SHOW-INTERP (#386) — the `declared.Tail == nil` gate at `validate_effects.go:162`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Render row-var tails in `FormatEffectRow`/`formatRow` everywhere (signatures currently print without their tail in some diagnostics, e.g. arm n's "Current signature" omits `! {e}`)
- Budget/param semantics for row-var tails (currently undefined and out of scope)
- Docs page for effect-row polymorphism (the feature is shipped but undocumented outside code comments)

## Quorum verification log

### Round 1 — 2026-08-11T20:49:35Z — **BLOCKED** (artifact `m-effect-row-var-unification-2026-08-11T20-49-35Z.json`, metered $0.1103)

Both external reviewers present (no N−1 hole). Both rejected. Per mission-control rule 3f the
controller **measured** each objection rather than forwarding it; all four measurements below were
taken first-party at `af6d56144` with the worktree binary, each negative paired with a firing
control.

| # | Claim | Command | Observed | Verdict |
|---|---|---|---|---|
| C1 | Designer's headline "IO behind a fully pure signature" | single-call arm: `runIt(doIO)` under `export func main() -> unit` | `check` **rc=1** — REJECTED, not accepted | designer's arm-3 as I first built it **did NOT reproduce**; the effect needs the double call (C2) |
| C2 | **R2 (gemini-3-1-pro) is CONFIRMED and UNDERSTATED** | `func runTwice(f: () -> int ! {e}) -> int = f() + f()` + `main()` with **no** annotation calling it with an `{IO}` function | `check` **rc=0** "No errors found"; `run --caps IO` prints `leak` **twice** | laundering happens **inside** the row-poly function, which Phase 2's App-site discharge does not reach. Blocking objection stands |
| C3 | Wrong-row laundering | `main() -> unit ! {FS}` wrapping an `{IO}` instantiation | `check` **rc=0**; runs and prints `hi` | confirmed |
| C4 | **Severity boundary** — is this a runtime capability escape? | same program, `--caps FS` and no-caps arms | **rc=1** `effect 'IO' requires capability, but none provided` | **NO.** The capability layer backstops. This is a **static** soundness hole that defeats capability *planning from signatures* (the reporter's MCP-embedder use case), not a runtime escape. Severity must be stated this precisely |
| C5 | **R1 (gpt5-6-sol) is CONFIRMED** — is `typeInfo` the right source at a direct same-module App? | `DEBUG_EFFECTS=1 ailang check` on the arm-b repro | for `purePath`: `Callee declared effects (from signature): []` — the **declared** path is taken and `typeInfo` is **never consulted**. Control (concrete case): `Callee type effects (from CoreTypeInfo): [IO]` and `[IO]` totals, so the instrument fires | the App branch prefers `declaredEffects` and only *falls back* to `CoreTypeInfo`; for the exact defect case it takes the declared path. A Phase-2 design that discharges tails via `typeInfo` at App sites must first change **which source is consulted**. Blocking objection stands |
| C6 | Mechanism of the blank message (doc V5) | same `DEBUG_EFFECTS=1` run | `Declared effects: []` **and** `Required effects: []` — **both empty** — yet the check fails | the failure decision is **tail-level** while `DiffEffectRows`/`writeEffectDiff` are **labels-only**, so the differ correctly reports nothing missing. Confirms V5 and gives it a mechanism |
| C7 | "Just reject lowercase" is refuted | `grep -rEc '! *\{ *[a-z][A-Za-z0-9_]* *\}' std/ --include='*.ail'` | **13** row-variable signatures ship: `std/list` **5** (`mapE`,`filterE`,`foldlE`,`flatMapE`,`forEachE`), `std/stream` **4**, `std/smoke` **2**, `std/ai/streaming` **2`. Control: **4** files use `! {IO` | rejecting at parse would break shipped stdlib API. Direction is settled by evidence, not opinion |

**Framing correction this produced, and it supersedes the issue's own words twice.** `e` is **not**
a phantom *concrete effect* (issue #616's framing) and **not** a label at all — it is a row **tail**.
C6 shows the effect checker's subsumption sees the tail while every diagnostic it owns renders only
labels. So both the false rejection and the blank message are the same defect wearing two faces.

**Disposition.** Not force-passed. Both objections are TRUE, both are now MEASURED, and R1's goes to
the architecture rather than to completeness — so the narrow-refinement carve-out does **not** apply.
The doc requires ONE revision addressing C2 (bring `UnionEffectRows` tail deletion into the Solution
Design; cover the intra-function double-call case) and C5 (re-plan Phase 2 around which effect source
the App branch consults), then ONE re-quorum. No human decision is required to proceed — C7 settles
the direction the doc asked to have ratified.
