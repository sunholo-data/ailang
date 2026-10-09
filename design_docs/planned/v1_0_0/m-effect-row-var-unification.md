# M-EFFECT-ROW-VAR-UNIFICATION — Discharge effect-row variables in the effect-checking pass

**Status**: Planned
**Target**: v1.0.0
**Priority**: P0 (High) — static effect soundness hole, not just DX friction (see V10/V11)
**Estimated**: 5 days (minimal App constraint repair + shared publication + row validation + corpus evidence)
**Dependencies**: M-EFFECT-LATENT-FUNCTION-VALUES, landed in PR #1708 (`3d4f49720`) — the minimum required dependency, not the sweep base (see the sprint plan's sweep protocol)
**Ordering**: runs BEFORE M-PURE-ROW-AND-IFACE-PURITY (#1443), which re-measures its pins after this lands. The two sprints must not run in parallel.
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
`std/ai/streaming.ail`, `std/smoke.ail` — V27); V14–V15 establish only the declared row-polymorphic import controls. Inferred rows
need separate coverage: #1091 demonstrated pure-row over-generalization across imports.

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

## Current Design — maintainer ruling, 2026-10-08

**Unparked for re-planning. Refs #616.** Mark ruled D-10 = **option A** on
2026-10-08: a **minimal constraint repair at the App constraint**, where
`internal/types/inference.go` mints an independent `freshEffectRow`. This supersedes
PR #1678 and the earlier A3-only proposal; publication alone cannot repair an
incorrectly constrained call row. Review findings:
[PR #1678 review comment](https://github.com/sunholo-data/ailang/pull/1678#issuecomment-6067485482).
The V1–V39 and quorum measurements below remain historical evidence at their named
bases, not claims about the post-#1708 base.

The minimum implementation base is dev at `3d4f49720` (PR #1708), which shipped
M-EFFECT-LATENT-FUNCTION-VALUES. The executor branch starts from current dev, and the
base/fixed corpus sweep uses the branch's actual merge-base with dev (recorded by SHA),
not `3d4f49720`: #1707 has since changed `effects.go`, `effect_subsumption.go`,
`validate_effects*.go` and `pipeline_single.go`, so a `3d4f49720` base would misreport flips. Its per-App publication is the required authority:
**extend LatentParamMask with the resolved call row in one publication record**.
Do not introduce a standalone `CallEffects[appID]` map or parallel lookup authority.
Do not execute against the pre-#1708 base or concurrently with its implementation.

### Fix site and bounded systemic audit

The AST App/FuncCall path in `inference.go` constructs a fresh result effect row
independently of the argument rows. The Core App path in
`typechecker_functions.go:inferApp` likewise constructs an expected function type
with a fresh effect row and solves application equalities. Audit both paths,
shared scheme instantiation/substitution and row equality so the same signature
row variable relates callback and call result at that constraint. Repair only the
necessary equality/sharing; do not rebuild global inference, add a row join, or
reconstruct inference by matching signature names inside validation.

Retain #386's local equality replay and enclosing-scope protections: eagerly
closing a callee row has previously broken recursive multi-effect functions.
Prove the repair on repeated calls, independent pure/IO instantiations, recursive
calls and mixed concrete/open rows before broadening any solver change.

### One publication, two lifecycle stages

Extend the backing publication in `internal/types/typechecker_latent_mask.go`
with a record containing the pre-application `LatentParamMask` and the post-solve,
zonked **callee call row**. The mask is captured before argument unification;
the row is finalized after the relevant substitutions/defaulting. The call row
excludes argument evaluation effects: the collector combines those separately.
Existing `LatentParamMask` callers can project from the same record during API
migration; there must be one backing store and one authoritative presence bit.

Every successfully typed function App publishes a record, including pure calls
(closed empty row) and all-false masks. Absence differs from purity. Clone masks
and rows on publication/lookup so consumers cannot mutate inference state.
Unowned unsolved metavariables, missing records and malformed records fail loudly
with App ID/span; an open row is valid only when owned by the surrounding generic
context. Return-only row variables require a tested minimal/default-empty
instantiation in concrete callers; never silently discard an unresolved tail.

Thread the unified lookup through both `pipeline_single.go` and
`pipeline_module_compile.go` into `ValidateEffectsWithCalls` and the effect
collector. Declared concrete rows retain contamination-safe authority (71b610d68);
row-polymorphic calls use the solved publication, retaining declared concrete
labels. Cover local and imported, declared and inferred polymorphic callees.
`CoreTypeInfo`/`EffectValueType` retain their established structural/value uses;
incidental callee occurrence types are not a fallback for the published call row.

### Row algebra and diagnostics

Normalize closed empty rows to pure. Required-row and suggested-row union both
preserve an identical tail and a sole open tail; distinct unresolved tails must
produce a deterministic explicit conflict, never a silent choice or purity.
Use the existing no-join constraint model to resolve equal tails before union.
Do not equate tails by printed spelling when binder identities differ.

Subsumption/diff must report an undischarged tail against a pure/closed or
incompatible declaration; matching owned tails pass. A declared tail does not
absorb arbitrary concrete IO. Keep parameter/budget checks intact. Diagnostics
name missing labels, parameter mismatches, or unresolved tails. An empty diff on
failure is an internal invariant error. Suppress an identical suggested signature;
render tail information so any printed migration is actionable.

### #1091 re-scope, folded into this design

V14/V15 prove correctness only for **declared** row-polymorphic imports, not all
cross-module calls. #1091 / M-EFFECT-PURE-ROW-OVERGENERALIZATION showed a declared
pure recursive helper exported with a generalized row and an importer acquiring
spurious FS. Its source fix shipped in v0.35.3; preserve closure before
generalization and test standalone importer versus explicit dependency checking.
Also cover inferred effect-transparent option/result/list combinators without
`! {e}` annotations. Leaving VarGlobal untouched solely because V14/V15 passed
is no longer an acceptable justification.

#1718 (callbacks nested in list/tuple/ADT arguments) remains outside this sprint.
Only a trivially shared correction with no new structural traversal belongs here;
otherwise record the remaining gap without weakening #1708's existing mask checks.

### File and execution constraints

`internal/types/typechecker_core.go` is **exactly 800 lines** at this base, the CI
limit. Put new logic in companion files (including publication helpers), and use
replacement declarations without net growth where the checker needs a field.
Check the sizes of other touched files too; do not accumulate overflow elsewhere.

Execution gates: focused `go test ./internal/types/... ./internal/pipeline/...
./internal/elaborate/...`, `make test-core`, `make lint`, `make check-boundaries`,
`make check-file-sizes`, `make check-changelog`. Do not run full `make test` in the executor: RAM-backed
`/tmp` has caused SIGBUS. The executor commits locally and cannot push.
Documentation uses `Refs #616`; no issue-closing directive belongs in these docs.

## Acceptance Criteria

- [ ] **AC1 — constraint repair**: #616's same-module `runIt(pureFn)` checks and
  publishes a closed pure call row; IO instantiation publishes IO independently.
  Pin AST and Core App constraints, not merely a validator workaround.
- [ ] **AC2 — diagnostics**: regression covers the original blank diff and
  identical Suggested fix. Pure caller accepts; a wrong FS wrapper around IO
  rejects naming IO with a changed suggestion. No empty user-facing failure.
- [ ] **AC3 — repeated-call soundness**: the historical pure
  `runTwice(f: () -> int ! {e}) -> int = f() + f()` plus `runTwice(noisy)`
  is rejected before execution. Record the old printing-twice behavior when it
  still reproduces on the new base with IO granted. Corrected row-polymorphic
  helper and IO caller run and print twice; capability enforcement is unchanged.
- [ ] **AC4 — one authority**: mask and zonked call row share one per-App record;
  pre-instantiation mask survives solving; missing/malformed records fail loudly;
  pure/all-false differs from absent; lookup snapshots cannot mutate the store.
- [ ] **AC5 — open-row boundary**: same-tail union/subsumption passes, distinct
  tails conflict explicitly, unowned tails fail, return-only pure call resolves
  without hidden fallback. Exercise both required and suggested-row union callers.
- [ ] **AC6 — non-regression**: arms a/d/e/k/m/n/h/i/j, inferred combinators,
  declared-pure imports (#1091), recursive concrete contamination controls,
  #386 no-join tests and #1708 latent/storage-only controls keep their contracts.
  Historical base behavior must be re-measured rather than assumed.
- [ ] **AC7 — corpus sweep**: compare base (the executor branch's merge-base with dev,
  recorded by SHA) and fixed `ailang check` for every
  `examples/**/*.ail` and `std/**/*.ail`, with cold caches and matching stdlib.
  Record inventory, command, status and diagnostics per file, including existing
  failures; investigate std/stream and std/ai/streaming consumers explicitly.
- [ ] **AC8 — flip rule**: each changed status is listed by path in the fragment
  `changelogs/unreleased/<YYYY-MM-DD>-effect-row-var-unification.md` (opening with a
  `### Fixed — …` heading; `make check-changelog` passes) with before/after,
  reason and migration. Newly accepted pure programs need no migration; newly
  rejected unsound programs require real effects or the shared generic row.
  An unexplained flip or sound-program rejection blocks completion. Distinct-tail
  conflicts in valid streaming code require repair, not blanket reclassification.
- [ ] **AC9 — examples/docs**: add and verify a pure caller example and a corrected
  noisy callback example (twice output). Update the example manifest and applicable
  effect limitations. Fragment explains the soundness change and remaining #1718.
- [ ] **AC10 — completion**: all focused/core/lint/boundary/size/changelog gates pass;
  evidence records hashes, exact commands and mutation checks. Local commits only.

## Sprint and scope

The companion sprint plan decomposes this work into constraint repair, shared
publication/validation, regression matrix, and corpus/migration verification.
No parser syntax changes, global inference redesign, runtime capability changes,
new row joins, arbitrary tail budget semantics, or nested callback traversal.
The earlier plan in PR #1678 is superseded, not an executor input.

## References

- [#616](https://github.com/sunholo-data/ailang/issues/616)
- [#1091](https://github.com/sunholo-data/ailang/issues/1091)
- [#1708](https://github.com/sunholo-data/ailang/pull/1708), `3d4f49720`
- [#1718](https://github.com/sunholo-data/ailang/issues/1718)
- `design_docs/planned/v0_48_0/m-effect-latent-function-values.md`
- `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md`

## Historical quorum verification log (superseded architecture)

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
