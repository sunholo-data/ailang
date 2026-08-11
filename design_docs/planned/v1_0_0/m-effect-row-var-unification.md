# M-EFFECT-ROW-VAR-UNIFICATION — Discharge effect-row variables in the effect-checking pass

**Status**: Planned
**Target**: v1.0.0
**Priority**: P0 (High) — static effect soundness hole, not just DX friction (see V10/V11)
**Estimated**: 4–5 days (row algebra + explicit type-check/pipeline interface + validation + tests/docs)
**Dependencies**: None
**Planner-Lane**: opus-required (touches shared row algebra in `internal/types/` and the effect-validation pass; the contamination history in V25 makes a mechanical port risky)
**Source**: GitHub issue [#616](https://github.com/sunholo-data/ailang/issues/616), re-reproduced and extended at `origin/dev` = `af6d56144`

## Problem Statement

A lowercase name in an effect row — `! {e}` — parses, type-checks, and *reads* as effect
polymorphism. The parser accepts it **deliberately** for that purpose (V18), and the type layer
builds `RowVar` tails (V19), but does not share parameter/result occurrences during call
instantiation (V31–V33). The separate
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
calls take a different, correct code path (V14–V15).

### What issue #616 gets wrong

The issue frames a dichotomy: "reject lowercase names (fastest)" or "implement unification".
The measurements refute that framing:

- **"Implement" is not parser work, but it crosses a real interface boundary.** The parser builds
  the intended syntax (V18), while the direct-call probe shows that CoreTypeInfo does not publish
  the instantiated result effect (V31–V35). The fix therefore needs explicit type-checker output
  plus validator consumption, not a parser rejection or an incidental type-info lookup.
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

**V16 (arm j) — the let-alias route accepts the V2 program.** This observation is retained
unchanged; V31–V33 refute the former inference that it proves a direct callee's result effect is
instantiated. Aliasing routes around `declaredEffects`, and the exact program from V2 checks clean:
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

**V30a — no duplicate/covering design doc.** `design_docs/planned/` contains no row-var/effect
doc (grep for `616|row variable|rowvar` matches only unrelated ollama-streaming docs). The
closest prior work, `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md`, is a
**different bug** (TYPE-layer unification of lambda closed rows against concrete rows, e.g.
`{IO, Stream}`; its own reproducer errors in *type unification*, not effect checking). Its
status header still reads "Planned" despite sitting in `implemented/` — known stale-header
class; do not trust status headers as facts.

**V31 — direct same-module occurrence type is present but its result effect is uninstantiated.**
Controller probe at base `817bb0274`, after instrumenting the App case and rebuilding:
```
$ ITER180_PROBE=1 ./bin/ailang check tmp/eff616/eff_b.ail
[ITER180] App appID=13 funcID=11 funcExpr=*core.Var funcName=runIt usedDeclared=true declaredRow={labels=[] tail=e}
[ITER180]   typeInfo.Get(funcID) PRESENT goType=*types.TFunc2 type=() -> int -> int ! {...ρ3}
[ITER180]   TFunc2.EffectRow RAW={labels=[] tail=ρ3}
[ITER180]   extractEffectFromType=nil
[ITER180]   typeInfo.Get(appID) PRESENT goType=*types.TCon type=int
```
The occurrence is present and occurrence-shaped, but `ρ3` is unsolved rather than the closed
empty row required by V2. The earlier interpretation of V16 is therefore superseded: V16 proves
only that the let-alias route accepts, not that direct-call result effects are instantiated.

**V32 — call occurrences are distinct and argument rows are instantiated; result rows are not.**
The same controller probe on arm l (V9/V10), forced through a fresh-content check per V38:
```
$ ITER180_PROBE=1 ./bin/ailang check tmp/eff616/eff_l.ail
[ITER180] App appID=21 funcID=19 funcName=runIt usedDeclared=true declaredRow={labels=[] tail=e}
[ITER180]   typeInfo.Get(funcID) PRESENT type=() -> int -> int ! {...ρ8}
[ITER180]   TFunc2.EffectRow RAW={labels=[] tail=ρ8}
[ITER180]     param[0] *types.TFunc2 () -> int          effrow={labels=[] tail=nil}
[ITER180]     return *types.TCon int
[ITER180]   extractEffectFromType=nil
[ITER180] App appID=25 funcID=23 funcName=runIt usedDeclared=true declaredRow={labels=[] tail=e}
[ITER180]   typeInfo.Get(funcID) PRESENT type=() -> int ! {IO} -> int ! {...ρ9}
[ITER180]   TFunc2.EffectRow RAW={labels=[] tail=ρ9}
[ITER180]     param[0] *types.TFunc2 () -> int ! {IO}   effrow={labels=[IO] tail=nil}
[ITER180]     return *types.TCon int
[ITER180]   extractEffectFromType=nil
```
The two argument types correctly differ (`{}` versus `{IO}`), while result tails remain `ρ8` and
`ρ9`. Per-occurrence storage exists; the needed result is not published there.

**V33 — parameter `e` and result `e` are not the same instantiated variable.** In V32 appID 25,
the parameter row is concrete `{IO}` while the result row remains `ρ9`. If both signature positions
shared one variable, parameter unification would also solve the result. This refutes the prior
premise that the type layer needs no change.

**V34 — CONTROL: concrete shared rows resolve at the same occurrence node.**
```
$ ITER180_PROBE=1 ./bin/ailang check <fresh-temp>/run_concrete.ail
[ITER180] App appID=17 funcID=15 funcName=runC usedDeclared=true declaredRow={labels=[IO] tail=nil}
[ITER180]   typeInfo.Get(funcID) PRESENT type=() -> int ! {IO} -> int ! {IO}
[ITER180]   TFunc2.EffectRow RAW={labels=[IO] tail=nil}
[ITER180]     param[0] () -> int ! {IO}   effrow={labels=[IO] tail=nil}
[ITER180]   extractEffectFromType={labels=[IO] tail=nil}
RC=0
```
The storage/zonking path works for concrete rows; V31–V33 are row-variable-specific.

**V35 — CONTROL: a return-only row variable has no argument-derived source.**
```
$ ITER180_PROBE=1 ./bin/ailang check <fresh-temp>/ret_only.ail
[ITER180] App appID=7 funcID=5 funcName=retOnly usedDeclared=true declaredRow={labels=[] tail=e}
[ITER180]   typeInfo.Get(funcID) PRESENT type=() -> int ! {...ρ3}
[ITER180]   TFunc2.EffectRow RAW={labels=[] tail=ρ3}  extractEffectFromType=nil
RC=1
```
`retOnly() -> int ! {e} = 42` called by a pure `caller` is the boundary any argument-derived
scheme must define.

**V36 — `extractEffectFromType` destroys tails.**
```
$ sed -n '270,284p' internal/pipeline/validate_effects.go
```
Observed: both `*types.TFunc2` and `*types.TApp` branches return `nil` when `len(row.Labels) == 0`
without checking `row.Tail`; V31/V32/V35 consequently print `extractEffectFromType=nil` for
`{labels=[] tail=ρN}`. This helper cannot be reused for row-variable resolution.

**V37 — `UnionEffectRows` drops tails by construction, confirming R2.**
```
$ sed -n '511,617p' internal/types/effects.go; grep -c "Tail" internal/types/effects.go
```
Observed: the function's merge body has no tail-preserving branch; lines 606–608 return `nil` for an
empty merged label set and line 614 explicitly returns `Tail: nil`. Whole-file positive control:
`grep -c` returns `3` (lines 359, 467, 614). Fix site: `effects.go:606-616`.

**V38 — passing `ailang check` inputs are content-cached and can hide instrumentation.**
```
$ ITER180_PROBE=1 ./bin/ailang check tmp/eff616/eff_g.ail   # second unchanged pass
# 0 probe lines
$ ITER180_PROBE=1 ./bin/ailang check tmp/eff616/eff_b.ail   # failing control
# 10 probe lines
$ printf '\n' >> tmp/eff616/eff_g.ail; ITER180_PROBE=1 ./bin/ailang check tmp/eff616/eff_g.ail
# 25 probe lines
```
Passing soundness arms must use a fresh temp path/content or a documented cache bypass; an unchanged
passing rerun is uninformative.

**V39 — `UnionEffectRows` grep has six hits, including two production callers.**
```
$ grep -rn "UnionEffectRows(" internal/ cmd/ --include='*.go'
internal/pipeline/validate_effects.go:541: suggestedEffects := types.UnionEffectRows(declared, required)
internal/pipeline/validate_effects_rows.go:81: merged := types.UnionEffectRows(a, b)
internal/types/effects_budget_test.go:273: result := UnionEffectRows(rowA, rowB)
internal/types/effects.go:511:func UnionEffectRows(a, b *Row) *Row {
internal/types/effects_test.go:119:func TestUnionEffectRows(t *testing.T) {
internal/types/effects_test.go:183:result := UnionEffectRows(rowA, rowB)
```
There are six grep hits: one definition, three test occurrences, and two production callers, both
outside `internal/types`. The semantic change is therefore not confined to subsumption callers.

## Root-Cause Mechanism (one paragraph)

`ValidateEffects` substitutes the declared `{e}` row for a direct same-module callee (V24/V25).
That tail reaches consumers with incompatible semantics: subsumption treats it as non-pure,
diagnostics omit it, and `UnionEffectRows` deletes it (V21–V23/V37). The type checker stores distinct
call occurrences and concretized argument rows, but the callee result row is a fresh unsolved
metavariable at each occurrence because the parameter and result uses of `e` are not shared
(V31–V33). Therefore the validator cannot recover the call effect from `CoreTypeInfo`; the pipeline
must receive an explicit per-call instantiated effect from the type checker, while row union must
preserve any tail that legitimately remains.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Teach the shipped row-variable feature; do not reject its syntax | Rejecting breaks 13 stdlib signatures (V27), so the evidence has settled this direction | controller evidence | design | high |
| Publish per-call instantiated effects explicitly (A3) | `CoreTypeInfo` result rows are unsolved even when argument rows are concrete (V31–V35); an explicit interface makes the needed contract testable | design | design | high |
| Fail loudly only when the documented publication invariant is violated | An expected unsolved `CoreTypeInfo` row is not an error; absence/malformed data from the new post-inference per-call map is | design | compile | med |
| Preserve identical tails in `UnionEffectRows` | The current function deletes tails and launders repeated local calls (V37/R2) | reviewer requirement | compile | high |
| An effect-check failure with an empty diff becomes structurally impossible (or fails loudly as an internal invariant violation) | This is the general form of the V5/blank-message defect; without the invariant the next tail-like bug is again invisible | agent | compile | low |
| Declared row var still does NOT absorb concrete required effects | Preserves V13 semantics; changing it would silently weaken the effect system | agent (keep as-is) | design | low |

### Design Freeze

Before implementation begins, these design decisions are frozen:

- [x] Use A3: the type-checking pipeline publishes `CallEffects[appID]` after inference/zonking.
- [x] Every successfully typed App gets an entry: closed empty for pure, concrete labels for a closed
  instantiation, or an explicitly open row only when the surrounding polymorphic context owns that
  tail. Missing/malformed entries violate the interface and fail loudly with callee and App ID.
- [x] A fresh unsolved row metavariable in incidental `CoreTypeInfo` is expected (V31–V35), is never
  consulted for discharge, and cannot trigger the fail-loud path.
- [x] `UnionEffectRows` preserves an identical tail, rejects/conflict-reports distinct tails, and
  never silently converts a surviving tail to nil.
- [x] `EffectRowDiff` reports undischarged tails; an empty diff on failure is an internal invariant error.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of the new row-var diagnostic and whether it gets a structured code — **agent may choose**; if a code is added it MUST be verified unallocated first (`grep -rn "<code>" internal/ cmd/`); note the current effect-check failure text has no code at all.
- Representation of the per-call map (new field on the existing inference result versus a small
  companion result struct) — implementer may choose; its semantic contract above is fixed.
- Whether `formatRow` (the DEBUG_EFFECTS printer) learns to render tails (`[| e]`) — **agent may choose**; strongly recommended given V17.
- Test fixture file naming/organization under `examples/runnable/` — **agent may choose**.
- Whether arm-e-style accidental passes get a changelog callout — **human at review**.

## Solution Design

### Overview

Implement A3: inference explicitly publishes the instantiated effect of every App, keyed by App ID,
and validation consumes that interface only when the same-module declared row has a tail. Also:

1. **Preserve tails during union.** `UnionEffectRows` keeps an identical tail when merging rows and
   surfaces incompatible tails instead of deleting either. This closes the local `runTwice` hole
   independently of top-level App discharge (V37).
2. **Give tails defined subsumption semantics** so any tail that legitimately survives (a
   row-polymorphic function's own body, arm a/V1) is handled explicitly rather than by
   accident: `required.Tail != nil` is subsumed only by a declared row with a matching tail;
   it is *reported by name* against a closed or pure declaration. A row with empty labels and
   nil tail normalizes to pure.
3. **Make the blank message impossible**: `EffectRowDiff` gains an
   `UndischargedRowVars []string` field populated from tails; `writeEffectDiff` prints it;
   `formatEffectError` returns an internal-invariant error ("effect check failed but the diff
   is empty — this is a compiler bug, please report") if a failure ever again produces an
   empty diff; the suggested-fix line is only printed when it differs from the current
   signature.

### Architecture

**Components:**

1. **Per-call effect publication** (type-check pipeline): while checking an App, resolve its function
   effect under the same substitution used for its arguments and write the zonked row to
   `CallEffects[e.ID()]`. The contract covers V35: a return-only row variable in a concrete caller
   must be solved by contextual/default empty-row rules before publication; if inference legitimately
   leaves it generalized, publish that owned open tail rather than fabricating purity.
2. **App collector consumption** (`internal/pipeline/validate_effects.go`): for a same-module callee
   whose declared row has a tail, read `CallEffects[e.ID()]` and combine its row with declared concrete
   labels. Concrete declarations keep the current contamination-safe path (V25). Missing entry,
   wrong key/type, or a supposedly closed entry containing an unowned metavariable is a violated
   documented invariant and fails loudly. The unsolved `CoreTypeInfo` rows in V31/V35 are expected
   legacy observations, not invariant failures.
3. **Tail-preserving union and subsumption** (`internal/types/effects.go:511-617,624`,
   `internal/types/effect_subsumption.go:57` `DiffEffectRows`): normalize label-empty+tail-nil
   rows to nil at entry; teach `DiffEffectRows` to emit `UndischargedRowVars` when
   `required.Tail` is not covered by `declared.Tail`; `SubsumeEffectRows` fails when that
   field is non-empty. Concrete-label logic is unchanged (preserves V12/V13/V15 behavior).
   All callers are inside this pass (V26), so the semantic change cannot leak elsewhere.
4. **Error-format invariant** (`internal/pipeline/validate_effects.go:520-563`): print
   `Undischarged effect row variable(s): e — the callee's '{e}' could not be discharged at
   this call site` (wording deferred); empty-diff-on-failure → internal-invariant error;
   suppress the suggested-fix line when identical to the current signature.
5. **Fixtures and tests** (new `internal/pipeline/effect_rowvar_discharge_test.go`, new
   runnable example): every arm from the Verification Log that changes or must not change
   becomes a pinned test.

### Implementation Plan

**Phase 1: row algebra + diagnostics (~1 day)**
- [ ] Update `UnionEffectRows` at `internal/types/effects.go:606-616` to preserve `Tail` when
  merging rows (keeping the tail if identical); define a loud conflict for distinct non-nil tails.
- [ ] Add a unit AC and source regression for
  `func runTwice(f: () -> int ! {e}) -> int = f() + f()`: the tail survives union and forces the
  enclosing signature to declare `! {e}` (R2/V37).
- [ ] `EffectRowDiff.UndischargedRowVars` + `DiffEffectRows` tail handling + normalization helper
- [ ] `SubsumeEffectRows` consumes the new field; unit tests directly on rows (incl. the exact poisonous shape from V19: non-nil, label-empty, tail `e`)
- [ ] `writeEffectDiff` prints the new field; `formatEffectError` invariant guard + suggested-fix suppression

**Phase 2: explicit per-call effect interface (~1–2 days)**
- [ ] Publish zonked `CallEffects[appID]` during type checking and thread it into validation.
- [ ] Consume it in the App case, gated on `calleeEffects.Tail != nil`; never use
  `extractEffectFromType` for this path (V36).
- [ ] Add contract tests for pure, `{IO}`, two independent occurrences, V35 return-only, missing map
  entry, malformed/unowned tail, and concrete-row control.
- [ ] `formatRow` tail rendering for DEBUG_EFFECTS (recommended)

**Phase 3: fixtures, regression sweep, docs (~1 day)**
- [ ] Port arms a,b,d,e,g,k,l,m,n,h,i,j into pipeline tests (same-module AND cross-module); base-red assertions for b (accept), g/l (reject naming IO)
- [ ] New runnable example `examples/runnable/effect_row_var_pure_caller.ail` + manifest entry
- [ ] `make test`, `make verify-examples` (expect manifest drift class, not type regressions, if red)
- [ ] CHANGELOG.md entry; `docs/LIMITATIONS.md` update if row-var limitations are listed there; close-out comment on #616

### Files to Modify/Create

**Modified files:**
- type-check result/publication file selected during implementation (~+60 LOC) — per-App effect map
- pipeline orchestration file selected during implementation (~+20 LOC) — thread map to validator
- `internal/pipeline/validate_effects.go` (+70/−15 LOC) — consume per-call effects, invariant errors, DEBUG tails
- `internal/pipeline/validate_effects_rows.go` (+15/−0 LOC) — normalization helper (label-empty + tail-nil → nil)
- `internal/types/effect_subsumption.go` (+35/−5 LOC) — `UndischargedRowVars` in `EffectRowDiff` + `DiffEffectRows` tail logic
- `internal/types/effects.go` (+30/−10 LOC) — tail-preserving `UnionEffectRows`, subsumption, normalization

**Explicitly unchanged:** `extractEffectFromType` in `validate_effects.go:270-284`. V36 proves it
drops label-empty tails, so this design neither calls nor extends it; changing the generic helper
would broaden behavior for unrelated callers without being needed by A3.

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
| `SubsumeEffectRows` / `DiffEffectRows` | Five measured pass call sites | Tail-aware semantics; concrete-label behavior pinned | V26 |
| `UnionEffectRows` | Six grep hits: one definition, three test occurrences, two production callers (`validate_effects.go`, `validate_effects_rows.go`) | Both production callers receive tail preservation; suggested-row and required-row output need tests | V39 |
| Type-check result boundary | Existing CoreTypeInfo carries distinct occurrences but unsolved result rows | Add explicit `CallEffects` output; do not reinterpret CoreTypeInfo | V31–V35 |
| App-case callee-row selection | Same-module concrete declarations avoid CoreTypeInfo contamination | Tail-bearing declarations consume `CallEffects`; concrete path unchanged | V25, V34 |
| Lambda sub-pass (`validate_effects.go:162`) | Enforces only CLOSED declared lambda rows (`declared.Tail == nil`) — open rows deliberately skipped (#386) | Unchanged. Note: after this fix `required` reaching :164 can newly contain resolved labels where it silently carried/dropped tails before; the closed-row gate’s semantics are unaffected, but the M3 test matrix must include an inline-lambda arm | V20 |
| Ghost-effect erasure (`eraseGhostEffects`, runs on `required` before subsumption) | Label-based removal (`Debug`) | Operates on labels only; tails pass through it untouched — no interaction, but pin with one test (mixed `{Debug, e}` callee) | code read, `validate_effects.go:31-40` |
| Effect budgets/params on rows (`Budgets`/`Params`, mode subsumption) | `DiffEffectRows` param logic; `unionRequiredEffectRows` conflict-preserving param merge | Untouched by tail logic (params key off labels). Budgets on a row-var tail are meaningless today and remain out of scope | V22 code read |
| Cross-module callees (`VarGlobal` → typeInfo path) | Already correct (V14/V15) | Unchanged — the new resolution applies only when the declared-map path is taken (`*core.Var` hit) | V14/V15 as pinned ACs |
| `iface` freezing / formatter / elaboration (`internal/iface/builder.go` 25 RowVar mentions, `internal/format/types.go` 8, `internal/elaborate/file_funcs.go` 2) | Serialize/print/carry row-var signatures | Read-only consumers of the same `Row` struct; no struct field is changed (the new field is on `EffectRowDiff`, a validation-only type) | `grep -rn RowVar internal/ --include="*.go" \| grep -v _test` file census |
| Runtime capability checks | Label/capability-based, no row vars (`internal/effects/` absent from the RowVar census) | Unchanged; remains the backstop measured in V11 | same census |

### Disambiguation strategy

No grammar disambiguation changes. Semantically, declared concrete labels remain authoritative and
the new per-App publication supplies only the instantiated tail contribution. `CoreTypeInfo` is not
a fallback. A missing/malformed publication is an interface violation; an open row explicitly owned
by the surrounding generic context is valid data.

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

- [ ] **AC1** (arm b, V2/V31): from a fresh temp path, `ailang check` exits **0** and the
  publication probe records a closed empty `CallEffects[appID]`. Base: rc=1 and CoreTypeInfo tail `ρ3`.
- [ ] **AC2** (arm g, V7): fresh temp path exits **1** with `Missing effects: IO`; probe records
  `{IO}` for the App. Base: rc=0.
- [ ] **AC3** (arm l, V9/V32): fresh temp path exits **1** with `Missing effects: IO`; its two App
  entries are independently `{}` and `{IO}`. Base: rc=0 and result tails are `ρ8`/`ρ9`.
- [ ] **AC4 (R2/runTwice)**: a unit test merging the same tail twice returns that tail, and a
  fresh-path source test using `func runTwice(f: () -> int ! {e}) -> int = f() + f()` fails and
  requires `! {e}` on the enclosing signature. Base: source check rc=0 laundering IO (Round-1 C2),
  and `UnionEffectRows` returns nil for empty labels (V37).
- [ ] **AC5 (V35 boundary)**: fresh-path `retOnly() -> int ! {e} = 42` / pure caller exits 0,
  and its published call effect is closed empty. A unit variant that deliberately omits the App map
  entry fails with the named invariant error. Base: source rc=1 with unsolved `ρ3` (V35).
- [ ] **AC6 (per-occurrence publication)**: a type-check unit test asserts V9's two App IDs publish
  independently as `{}` and `{IO}`. Base: no explicit publication interface exists and CoreTypeInfo
  instead contains unsolved `ρ8`/`ρ9` (V32).
- [ ] **AC7 (both union callers)**: unit tests drive the two production callers measured in V39 and
  assert an identical tail survives both required-row collection and suggested-row construction.
  Base: `UnionEffectRows` explicitly emits `Tail:nil` (V37).
- [ ] **AC8 (interface invariant)**: missing, wrong-key, malformed, and unowned-metavariable entries
  each fail loudly naming the callee and App ID; an explicitly owned open tail succeeds. Base: the
  interface and invariant do not exist (V20/V31).
- [ ] **AC9**: `go test ./internal/pipeline/ ./internal/types/ -count=1` is green including the new
  base-red AC1–AC8/AC10 tests AND the AC11 no-regression arms. Base: suite is green without them and
  does not reach the defect (V28/V29) — so "suite green" alone is NOT this AC; the new base-red tests
  are what make it non-vacuous.
- [ ] **AC10 (blank-diff invariant)**: (a) the poisonous row shape against nil reports `e`; (b) a
  forced empty-diff failure returns the compiler-invariant error rather than a blank message. Base:
  the blank message is produced by V2/V17 and the guard is absent (V20).
- [ ] **AC11 (no-regression gate — one test per "Programs that MUST still work" entry, 1–6)**: this
  design's whole risk is OVER-rejection, so the entries in the Conflict Surface are a done-gate, not
  a description. Specifically: arm a keeps rc **0** (V1); arm d and arm e keep rc **0** (V5/V6);
  arm k keeps rc **0** (V8 — it is in the deliberately-accepts set and must not flip to a reject);
  arm m's `mixed` accepted / `mixedUndeclared` rejected with `Missing effects: IO` (V12); arm n keeps
  rc **1** with `Missing effects: IO` (V13); arms h/i keep their cross-module accept/reject with
  byte-identical `Missing effects:` text (V14/V15); `./bin/ailang check
  examples/runnable/effectful_list_t1_mapE_basic.ail` exits **0** (V30); and one dedicated
  concrete-row recursive-call test pins the 71b610d68 contamination shape (entry 6). Each arm runs
  from a fresh temp path per V38. Base: every listed exit code and message is the measured base state
  (V1/V5/V6/V8/V12/V13/V14/V15/V28/V29/V30), so this AC cannot be vacuously green — it can only be
  satisfied by the arms still behaving as the base does. Note `ailang check std/list.ail` is NOT a
  gate: red at base for an unrelated module-name reason (V30).
- [ ] **AC12 (documentation deliverable)**: `CHANGELOG.md` entry describing the accept→reject class
  change and its migration path; `docs/LIMITATIONS.md` row-variable entry updated or removed if
  present (check at implementation time); `#616` closed with a comment linking the arms and naming
  which of its two proposed options the evidence refuted. Base: none of these exist.

## Testing Strategy

**Unit tests (types):** `DiffEffectRows`/`SubsumeEffectRows` on constructed rows — poisonous
shape (label-empty + tail), tail-vs-tail same name, tail-vs-tail different name, tail vs
closed, tail vs nil, normalization (label-empty + tail-nil ≡ pure), params/budgets untouched
by tail logic.

**Integration tests (pipeline):** arms b/g/l, `runTwice`, V35, and the unchanged regression arm
matrix as source-level `check` tests, same
harness style as `check386`/`TestEffectRowVariableImportsStillValidate`
(`internal/pipeline/effect_mode_subsumption_test.go:174`). Plus: inline-lambda arm (lambda
sub-pass interplay, Conflict Surface row 3) and mixed `{Debug, e}` ghost-erasure arm.

**Cache methodology constraint (V38):** every passing `.ail` probe is copied to a newly created temp
directory and given fresh content/module identity before `check`; alternatively a future documented
cache-bypass flag may be used. Re-running an unchanged passing path is not evidence and cannot satisfy
an AC. Failing controls remain paired with passing probes when instrumentation output is asserted.

**Regression-surface tests:** one per "Programs that MUST still work" entry (fixture list
above; the stdlib entries via the existing example + import test).

**Mutation kill matrix** (each observable is DOWNSTREAM of the mechanism — check exit codes
and message content, never internal state set alongside the mutated code):

| Mutation | Killed by | Downstream observable |
|---|---|---|
| Publish one declaration-level row instead of per-App rows | AC3 two-instantiation probe | entries must be independently `{}` and `{IO}` |
| "Fix" by silently stripping tails instead of resolving them | AC2/AC3 | arm g/l must exit 1 naming IO; tail-stripping re-accepts them (this is why strip-the-tail is not a fix — it is the laundering bug with better DX) |
| Missing per-App entry falls back to CoreTypeInfo | AC8 missing-entry unit test | invariant error must name callee and App ID; V31-style `ρN` is never accepted |
| Tail subsumption matches by presence, not name | Test where the enclosing function declares `! {f}` but the required tail post-substitution is a different var — construct via a helper whose row var cannot unify with `f` (implementer sketches during M2; if the type layer makes this unrepresentable, document why and drop the name-match distinction as unreachable) | rc flips on the mismatched-name arm |
| Empty-diff guard removed | AC10(b) | the invariant error text disappears → test fails |
| Suggested-fix suppression removed | diagnostic unit asserts a fix line appears only when it differs | message content |

**Manual testing:** re-run the full arm matrix from the Verification Log; re-run the V11
runtime probe (uncapped run of arm l must now be unreachable — the file no longer passes
`check`).

## Non-Goals

- **Sharing signature row variables through ordinary unification (A2)** — A3 adds a publication
  interface without redesigning general signature instantiation.
- **Parser changes** — `! {e}` syntax, the typo guard (V4), and PAR_EFF002 for uppercase unknowns (V3) are all untouched.
- **Runtime capability semantics** — the V11 backstop behavior is unchanged.
- **Budgets/params on row-var tails** (`! {e @limit=5}`-class questions) — out of scope; today's label-keyed budget logic is preserved as-is.
- **The `m-effect-row-poly-params` residue** (lambda closed-row unification against concrete rows) — different bug, already implemented; its stale "Planned" header is a docs chore, not this sprint.
- **`formatRow`-style debug polish beyond tail rendering** — nice-to-have only.

## Timeline

**Day 1**: Phase 1 tail-preserving union, subsumption/diff, diagnostics, unit tests.
**Day 2**: publish per-App effects and thread the result through the pipeline.
**Day 3**: validator consumption, contract failures, V35 boundary, arms b/g/l.
**Day 4**: regression matrix, cache-safe manual probes, examples and docs.
**Day 5 (buffer)**: conflict-surface sweep and full gates.

Total: 4–5 days. A3 crosses the type-check/pipeline boundary and `UnionEffectRows` has two production
callers (V39), so the previous 3–4 day estimate was too narrow.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Re-importing 71b610d68 contamination | High | A3 never uses CoreTypeInfo for discharge; concrete declaration regression stays in the matrix |
| Per-App publication incomplete | High | Contract requires every typed App; missing/malformed data fails loudly; AC8 |
| Tail union change affects suggested and required rows | Medium | Test both measured production callers from V39, including `runTwice` and diagnostic suggestions |
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

**Direction: A3 — explicitly publish per-call instantiated effects, then consume that documented
interface in effect validation.** This follows R1's reviewer-authored default. V31–V33 show why:
CoreTypeInfo has occurrence granularity but not the instantiated result effect. Phase 1 also repairs
tail union so repeated calls inside a generic function cannot launder the tail (V37/R2).

**Why not A2:** sharing a single row variable across all signature positions is the deepest root fix,
but it changes general instantiation/unification semantics and has the largest conflict surface. The
measurements establish the defect, not that this broader change is safe within this bounded sprint.

**Why not B2:** V32 proves argument types can derive `runIt`'s effect, but V35 proves that rule is
incomplete for return-only variables. Encoding signature-variable matching again in validation would
duplicate type-checker semantics and still need a separate boundary rule. A3 publishes the answer at
the layer that owns inference.

No new human decision is required; V27 already refutes rejecting the shipped syntax.

## References

- **Issue**: [#616 — Effect row variables parse but never unify](https://github.com/sunholo-data/ailang/issues/616)
- **Related (DISTINCT) prior work**: `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md` — type-layer lambda/closed-row unification, shipped v0.29.0; stale "Planned" header noted in V30a
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

### Round 2 — controller pre-measurement

Both round-1 objections were measured rather than merely forwarded, and both are confirmed:

- **R1 confirmed (V31–V36):** direct callee occurrence info is present but its result row is an
  unsolved metavariable; parameter rows are independently instantiated, concrete control works, and
  `extractEffectFromType` drops tails. This revision replaces the refuted CoreTypeInfo architecture
  with A3, an explicit per-App instantiated-effect interface.
- **R2 confirmed (V37/V39 and Round-1 C2):** `UnionEffectRows` deletes tails, including the repeated
  local-call shape. Phase 1 now preserves identical tails and includes the requested `runTwice`
  regression and AC.
- **Methodology constraint recorded (V38):** all passing source probes use fresh path/content or a
  documented cache bypass, so cached silence cannot be counted as evidence.

These are controller-verified measurements at base `817bb0274`; the re-quorum can review the revised
architecture and ACs without repeating the probes.

**Controller correction to the revision itself (recorded, not hidden).** The revision replaced the
previous 14 acceptance criteria with 10, and in doing so dropped the entire NO-REGRESSION half of the
gate — old AC4/AC5/AC6/AC7/AC8/AC9 (arms k, a, d+e, n, m, h/i), AC12 (the runnable stdlib example) and
AC13 (the 71b610d68 concrete-row class) had no counterpart, and AC14 (the documentation deliverable)
had none either. The *content* survived, in the Conflict Surface's "Programs that MUST still work"
(entries 1–6) and in the Implementation Plan's docs bullet, but not as anything a sprint could fail
on. Since this design's dominant risk is OVER-rejection — it converts a class of accepts into rejects
— a done-gate with no regression arm is the wrong shape, so the controller restored them as **AC11**
(one test per Conflict-Surface entry, every exit code and message quoted from its base measurement)
and **AC12** (CHANGELOG / LIMITATIONS / `#616` close-out), and made AC9 name AC11 so "suite green"
cannot stand in for it. Nothing was removed and no reviewer objection was overridden; the ACs added
are the ones the revision's own Testing Strategy already prescribes tests for. Final: **12** ACs,
1009 lines. `V39`'s caller census was re-derived first-party by the controller and matches exactly
(6 grep hits = 1 definition + 3 test occurrences + 2 production callers at
`validate_effects.go:541` and `validate_effects_rows.go:81`; known-positive control
`SubsumeEffectRows(` = 14 hits).
