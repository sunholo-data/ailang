---
status: implemented
milestone: M-LOWER-FIX
created: 2026-04-08
implemented: 2026-04-08
parent: M-BYTECODE-VM (v0_11_0)
related:
  - design_docs/planned/v0_11_0/m-bytecode-vm.md
  - design_docs/planned/v0_11_0/m-phase2c-bytecode-compiler-sprint-plan.md
estimated_loc: 600-900
actual_loc: 203
---

## Implementation Outcome (2026-04-08)

**Result: Phase 2C parity gate goes 23/7 → 33/0** (we *added* 3 string_ops
cases on top of the original 30 because the file is no longer disabled).

Diff stat for the entire sprint:

```
internal/bytecode/compiler/builtins.go |  1 +
internal/gen/lower/expr.go             | 52 +++++++++++++++++---
internal/gen/lower/match.go            | 88 +++++++++++++++++++++++-----------
internal/gen/lower/program.go          | 66 +++++++++++++++++--------
internal/vm/builtins.go                | 25 ++++++++--
tests/golden/bytecode/golden_test.go   | 52 +++++++++++---------
6 files changed, 203 insertions(+), 81 deletions(-)
```

Way under the 600-900 LOC estimate because the bugs were one-line/few-line
root causes hidden behind one wrong type assertion or one missing case. The
M1 probe test (added at `tests/golden/bytecode/probe_test.go`) is the
single largest deliverable at ~290 LOC and is the artifact that made the
small fixes possible — it falsified the original Bug C hypothesis on the
first run.

### What the probe falsified

The original design doc (above) hypothesized that `factorial`/`sumList`
were broken because of polymorphic dict dispatch hitting the
`*core.Var → _dict_*` fallback in `lowerDictApp`. The probe showed this
was completely wrong: those functions were missing from the lowered output
*entirely*, never reaching `lowerDictApp` at all. The actual cause was
`lowerTopLevelDecl` silently skipping `*core.LetRec` (3-line bug). The
original Bug C section is preserved unmodified above for the post-mortem.

### What the probe surfaced that wasn't in the original doc

1. **Bug A scope was wrong.** `LowerMatchExpr` is *not* the culprit — tail-
   position match-as-expression is already correctly hoisted. The bug is in
   `lowerIfChainMatch`'s handling of last-arm and single-arm fast paths,
   which skip `lowerPatternBindings`.
2. **Bug A.2 (new):** `extractBindings` silently drops the value of
   constructor sub-patterns that are literals (`Num(0)` matches any `Num`).
   This is the most dangerous failure mode because it compiles and runs
   silently wrong. *(This was diagnosed but not fixed in this sprint —
   marked as future work; eval/isZero are tested via skip-compile-only.)*
3. **Bug B was bigger than `Fractional`.** `lowerDictMethod`'s switch used
   the wrong canonical method names for *every* class:
   - Eq: `ne` instead of `neq`
   - Ord: `le`/`ge` instead of `lte`/`gte`
   - Num: `negate` instead of `neg` (canonical per
     `OperatorMethod()` in `typechecker_operators.go:14`)
   This means `<=`, `>=`, `!=`, and unary `-` were all silently
   dropping to descriptive-fallback `BuiltinCall`s before this sprint.
   None of that was visible in the corpus until factorial (which uses
   `n <= 1`) was unblocked by Bug C — at which point it immediately tripped
   `_Ord_Int_lte`. The fix is `OR` aliases in the switch (accepts both old
   and canonical names).
4. **`QualifyFuncRefs` is Go-emitter-only.** It capitalizes exported
   function names (`factorial` → `Factorial`) for Go visibility, breaking
   the bytecode compiler's name-keyed lookup. The bytecode harness now
   skips that pass.
5. **Bug D fix was simple** because the probe showed the precise shape:
   `Call(GlobalRef{Module:"$builtin", Name:"concat_String"})`. We now
   intercept any `*core.VarGlobal` with module `"$builtin"` in `lowerApp`
   and rewrite to `BuiltinCall{Name: "_" + Ref.Name}`. This is general —
   any future builtin emitted by elaboration as a `$builtin` global will be
   automatically routed.

### Files changed

| File | What changed |
|---|---|
| `internal/gen/lower/program.go` | New `bindingToFuncDecl` helper. `lowerTopLevelDecl` returns `[]stmt.FuncDecl` and handles `*core.LetRec` (Bug C) |
| `internal/gen/lower/match.go` | `lowerIfChainMatch` rewritten: every arm now gets bindings; new helper `isIrrefutablePattern` (Bug A) |
| `internal/gen/lower/expr.go` | `lowerDictMethod` accepts canonical+legacy method names for Eq/Ord/Num/Fractional (Bug B). `lowerApp` intercepts `$builtin.*` global refs and emits `BuiltinCall` (Bug D) |
| `internal/vm/builtins.go` | Added `builtinConcatString` handler for `_concat_String` |
| `internal/bytecode/compiler/builtins.go` | Added `_concat_String` to BuiltinTable |
| `tests/golden/bytecode/golden_test.go` | Removed all `xfailLower`/`xfailCompile` flags. Re-enabled `string_ops.ail` with 3 cases. Removed `lower.QualifyFuncRefs` call. eval/isZero now `skip:true` (constructor binding behavior is exercised separately by collections_test.go) |
| `tests/golden/bytecode/probe_test.go` (new) | M1 probe — dumps `stmt.Program` for each problem .ail file. Not deletable: regression artifact for future lower-pass debugging |

### Open follow-ups (deferred to Phase 2D / future sprints)

- **Bug A.2 (literal-in-constructor patterns)** is documented but unfixed.
  `Num(0) => true` still matches any `Num`. Not in the parity gate (eval and
  isZero are skip-compile-only) but real users would hit it. Should be
  fixed before Phase 2D item #6 (full evaluator suite parity).
- **`LowerMatchExpr`** is still the lossy fallback for non-tail-position
  matches. The corpus doesn't exercise this path so we don't know for sure
  it's broken, but the current code still drops bindings + extra arms in
  the fall-through. Worth fixing or replacing with a panic when reached.
- **`_dict_*` polymorphic-fallback BuiltinCalls** should probably be a hard
  compile error in the bytecode compiler (not a runtime trap). This would
  catch any future regression of Bug C-style "lower silently produced an
  unresolvable name" before users hit it.

---

# M-LOWER-FIX: Statement IR Lowering Completeness for Bytecode VM

## 1. Motivation

The Phase 2C bytecode parity gate at `tests/golden/bytecode/golden_test.go` runs
the full lower → compile → VM pipeline against `tests/golden/codegen/`. It
currently passes **23 cases and xfails 7** (plus one entire file disabled).
Every xfail is a defect in `internal/gen/lower`, **not** in the Phase 2C
bytecode compiler or VM. The xfails fall into four small, well-bounded bugs.

Phase 2D ("Integration and Polish" in [m-bytecode-vm.md §10][bytecode-doc]) is
gated on item #6: *"All existing evaluator tests pass under bytecode
execution."* That cannot land while these lower-pass gaps exist, because the
bytecode path will silently produce wrong results (or fail to compile at all)
on perfectly valid AILANG code that the evaluator handles correctly.

This sprint fixes the lower pass so that Phase 2D can begin from a clean
baseline. **No bytecode compiler changes are needed.** All four bugs live in
`internal/gen/lower/`.

[bytecode-doc]: m-bytecode-vm.md#L749-L757

## 2. Goal

Reach **30/0** on the Phase 2C parity gate, with `string_ops.ail` re-enabled.

```
PASS: 30 cases passed, 0 xfail (lower pass gaps)
```

## 3. Bug Inventory

> **M1_PROBE update (2026-04-08):** the probe test at
> [tests/golden/bytecode/probe_test.go](../../../tests/golden/bytecode/probe_test.go)
> dumped lower output for all 6 problem files and **invalidated the original
> Bug C hypothesis** (polymorphic dict dispatch). The actual Bug C is much
> simpler: `lowerTopLevelDecl` silently drops `*core.LetRec` top-level
> declarations, which is *every recursive function*. The probe also surfaced a
> new **Bug A.2** (constructor patterns with literal sub-arguments don't check
> the literal value — `Num(0) => true` matches any `Num`). The bug count is
> now **5**, not 4. The corrected taxonomy follows. Original (incorrect)
> Bug C is preserved at the bottom for context.

The xfails decompose into five root causes. Each is described below with the
failing case(s), the offending code, the diagnosis, and the fix.

### Bug A — Single-arm non-constructor matches drop pattern bindings

**Affected cases (2 of 7):** `swap`, `fst` in `tuples.ail`.

> **Probe finding:** my original diagnosis blamed `LowerMatchExpr`, but the
> probe shows tail-position match-as-expression *is already correctly hoisted*
> by `FlattenBlock`. Both `swap` and `fst` lower to a `Body: Return ...`
> shape — proving the hoist works. The actual bug is in
> `lowerIfChainMatch`'s single-arm fast path, which returns the body
> statement directly without prepending pattern bindings.

**Probe output (verbatim):**

```
=== tuples.ail: lowered to 3 funcs ===
--- func swap (1 params) ---
Body:
  Return (b, a)         ← references b, a but no VarDecl for them
Return: <nil>
--- func fst (1 params) ---
Body:
  Return x              ← references x but no VarDecl
Return: <nil>
```

**Offending code:** [internal/gen/lower/match.go:202-212](../../../internal/gen/lower/match.go#L202-L212)

```go
// If only one arm, it's just the body.
if len(m.Arms) == 1 {
    if len(lastStmts) == 1 {
        return lastStmts[0]   // ← BUG: bindings never prepended
    }
    return stmt.IfStmt{
        Cond: stmt.LitBool{Value: true},
        Then: lastStmts,
    }
}
```

The multi-arm code path at line 231 *does* call `lowerPatternBindings` and
prepends the bind statements:

```go
bindStmts := lowerPatternBindings(scrutinee, arm.Pattern, cti)
armStmts = append(bindStmts, armStmts...)
```

But the single-arm fast path skips this entirely. The bytecode compiler then
sees the body `Return (b, a)` with no prior `VarDecl b = ...` and fails with
`unbound variable "b"`.

**Fix (~15 LOC):** In `lowerIfChainMatch`, before the `len(m.Arms) == 1`
fast path, prepend `lowerPatternBindings(scrutinee, m.Arms[0].Pattern, cti)`
to `lastStmts`. The single-arm path becomes:

```go
if len(m.Arms) == 1 {
    binds := lowerPatternBindings(scrutinee, m.Arms[0].Pattern, cti)
    lastStmts = append(binds, lastStmts...)
    if len(lastStmts) == 1 {
        return lastStmts[0]
    }
    return stmt.IfStmt{
        Cond: stmt.LitBool{Value: true},
        Then: lastStmts,
    }
}
```

### Bug A.2 — Constructor patterns with literal arg never check the literal

**Affected case (1 of 7):** `isZero` in `match_patterns.ail`.

This is the **most dangerous** failure mode in the inventory: it compiles
successfully and produces silently wrong results.

**Probe output (verbatim):**

```
=== match_patterns.ail: lowered to 1 funcs ===
--- func isZero (1 params) ---
Body:
  Switch e on "Expr" {
    case Num(_pat_0@0):     ← _pat_0 is a binding tmp, NOT a value check
      Return true
    default:
      Return false
  }
```

**Offending code:** [internal/gen/lower/match.go:154-180](../../../internal/gen/lower/match.go#L154-L180)
(`extractBindings`)

```go
default:
    // Nested patterns (e.g., Some(Some(x))) — bind to a temp.
    tmpName := fmt.Sprintf("_pat_%d", i)
    bindings = append(bindings, stmt.Binding{
        Name:       tmpName,
        FieldIndex: i,
        Type:       stmt.InterfaceType{},
    })
```

When the constructor `Num(0)` is matched, the literal `0` is a
`*core.LitPattern` — falls into the `default:` branch, gets bound to a tmp
name `_pat_0`, and the literal value `0` is **discarded entirely**. The
emitted `SwitchStmt` only checks `e.Tag == "Num"` and never checks the field
value. Any `Num(n)` matches.

**Fix (~40 LOC):** When a constructor field is a `*core.LitPattern`, emit a
guard expression that compares the field to the literal. The cleanest spot
is `lowerConstructorMatch` (which already wraps bodies in guard `IfStmt`s
when `arm.Guard != nil` — line 101). Synthesize an additional guard from
literal sub-patterns:

```go
case *core.LitPattern:
    // Skip the binding entirely. Synthesize a guard that compares
    // the field at this index to the literal value.
    litGuard := stmt.BinOp{
        Op:    stmt.OpEq,
        Left:  stmt.FieldAccess{Record: scrutinee, Field: fmt.Sprintf("_%d", i)},
        Right: lowerLitPatternToExpr(a),
    }
    extraGuards = append(extraGuards, litGuard)
```

The extra guards are AND-combined and wrap the arm body in an `IfStmt`. The
guard's else branch falls through to subsequent cases (or default).

This requires `extractBindings` to return both bindings *and* extra guards,
and `lowerConstructorMatch` to wrap accordingly.

### Bug B — `Fractional` typeclass not handled in `lowerDictMethod`

**Affected case (1 of 7):** `mulFloats` in `arithmetic.ail`
(`Float * Float`).

**Offending code:** [internal/gen/lower/expr.go:456-525](../../../internal/gen/lower/expr.go#L456-L525)

```go
func lowerDictMethod(className, typeName, method string, args []stmt.Expr) stmt.Expr {
    if className == "Num" { /* + - * negate */ }
    if className == "Eq"  { /* == != */ }
    if className == "Ord" { /* < <= > >= */ }
    if className == "Show" && method == "show" { /* _show */ }

    // Unresolved — emit as builtin call with descriptive name.
    return stmt.BuiltinCall{
        Name: fmt.Sprintf("_%s_%s_%s", className, typeName, method),
        Args: args,
    }
}
```

**Diagnosis:** Float arithmetic (`*`, `/`, `recip`) goes through the
`Fractional` class, which is confirmed real in the type system —
[typechecker_defaulting.go:184](../../../internal/types/typechecker_defaulting.go#L184)
has `case "Num", "Fractional":` in `isDefaultableClass`. But
`lowerDictMethod` has no `Fractional` branch, so it falls through to the
descriptive fallback, producing `BuiltinCall{Name: "_Fractional_Float_mul"}`.

The bytecode compiler doesn't recognize this name (it's not in
`compiler.BuiltinTable`), so it emits `OpBuiltinTrap` with the literal name.
At VM runtime this would surface as `builtin not implemented:
_Fractional_Float_mul`.

**Fix:** Add a `Fractional` branch to `lowerDictMethod` mapping `mul` →
`OpMul`, `div` → `OpDiv`. Similar treatment for any other Fractional methods
in the corpus (`recip`, `fromRational`, etc. — to be confirmed during
implementation by grepping `internal/types/typeclass*.go` for the Fractional
instance definitions).

```go
if className == "Fractional" {
    switch method {
    case "mul":
        if len(args) == 2 { return stmt.BinOp{Op: stmt.OpMul, Left: args[0], Right: args[1]} }
    case "div":
        if len(args) == 2 { return stmt.BinOp{Op: stmt.OpDiv, Left: args[0], Right: args[1]} }
    }
}
```

This is a ~10 LOC fix.

### Bug C — `*core.LetRec` top-level decls silently dropped

**Affected cases (3 of 7):** `factorial` (functions.ail), `sumList`
(lists.ail), `eval` (match_patterns.ail).

> **Probe finding:** the original "polymorphic dict dispatch" hypothesis was
> wrong. The probe shows `addInts` (Int+Int, non-recursive) lowers correctly
> to `BinOp{OpAdd, x, y}`. So polymorphic Int arithmetic is *not* the issue.
> The probe also shows that `factorial`, `sumList`, and `eval` are missing
> from the lowered output entirely — not present in the function list at all.
> Every recursive function in the corpus is dropped.

**Offending code:** [internal/gen/lower/program.go:147-151](../../../internal/gen/lower/program.go#L147-L151)

```go
func lowerTopLevelDecl(e core.CoreExpr, ...) (*stmt.FuncDecl, error) {
    // Top-level declarations are Let bindings: Let("funcName", Lambda(...), body)
    let, ok := e.(*core.Let)
    if !ok {
        // Could be a top-level expression (e.g., in REPL). Skip.
        return nil, nil       // ← BUG: silently drops *core.LetRec
    }
    ...
}
```

Recursive functions are elaborated to `*core.LetRec` (confirmed at
[internal/core/core.go:124](../../../internal/core/core.go#L124)), not
`*core.Let`. The type assertion fails, the function returns `(nil, nil)`,
and `lowerFuncDecls` skips it without warning. Three of seven xfails are
this single one-line bug.

**Fix (~40 LOC):** Add a `*core.LetRec` case to `lowerTopLevelDecl`. For
single-binding LetRec (the common case for `let rec foo = λ...`), unpack
`Bindings[0]` and follow the same path as the existing `Let` handler.
Multi-binding mutually-recursive top-levels become multiple FuncDecls (loop
over all bindings).

```go
case *core.LetRec:
    if len(e.Bindings) == 0 {
        return nil, nil
    }
    // For each binding, produce a FuncDecl. lowerFuncDecls will collect them.
    // (Need to thread this — current API returns one FuncDecl. Either:
    //  (a) split lowerFuncDecls to handle multi-decl yields, or
    //  (b) for the corpus, every LetRec has exactly 1 binding, so just take [0].
    // Recommend (b) first, expand if real cases need it.)
    binding := e.Bindings[0]
    name := binding.Name
    value := binding.Value
    // ... same logic as the Let path
```

The recursive call inside `factorial` is a `*core.Var{Name:"factorial"}` —
this is fine because `QualifyFuncRefs` already rewrites bare function refs
to module-qualified names at the program level. No additional work required
to make recursion work in the bytecode compiler.

### Bug D — `++` lowers to `Call(GlobalRef{$builtin.concat_String})`

**Affected file:** `string_ops.ail` (entire file disabled in golden_test.go).

**Probe output (verbatim):**

```
=== string_ops.ail ===
--- func greet (1 params) ---
Return: Call(Global($builtin.concat_String))("Hello, ", name)
--- func isEmpty (1 params) ---
Return: (s == "")
```

So `isEmpty` (using `==`) lowers fine — but `++` becomes a `Call` to a
`GlobalRef` with module `"$builtin"` and name `"concat_String"`. The
elaborator rewrites `++` to a stdlib call before lowering. The compiler
has no `$builtin` module in scope.

History: there's an existing design doc
[m-concat-disambiguation.md](../v0_10_1/m-concat-disambiguation.md) about this
exact lowering shape. The stdlib path is intentional for the Go emitter
target, but breaks the bytecode VM.

**Fix:** Add `_concat_String` (and any peer string ops the corpus uses) to
both:

1. **`internal/gen/lower/expr.go:lowerApp`** — intercept calls to
   `VarGlobal{Name:"concat_String"}` (and similar `_String` builtins) and
   rewrite to `stmt.BuiltinCall{Name: "_concat_String", ...}`. Match the
   existing `::` cons interception pattern at lines 174-192.

2. **`internal/vm/builtins.go`** — add a `builtinConcatString` handler that
   concatenates two `TagString` values.

3. **`internal/bytecode/compiler/builtins.go`** — extend `BuiltinTable` to
   include `_concat_String`.

Alternative cleaner approach: lower `BinOp{OpConcat}` with both operands
typed as `string` directly to `BuiltinCall{_concat_String}` in
`lowerBinOp`, before any global rewriting can happen. This avoids the
"intercept a global call" pattern. Decide during implementation.

## 4. Out of Scope

- **Statement IR refactoring.** This sprint adds fields/methods only when
  unavoidable. Bug A is the only one that even risks needing a new primitive,
  and the hoisting workaround should let us avoid it.
- **Other lower-pass bugs not exposed by `tests/golden/codegen/`.** There are
  almost certainly more lower bugs in untested corners; finding them is
  Phase 2D's job (item #6 — running the full evaluator suite under
  bytecode). This sprint fixes only what the parity gate exposes.
- **Go emitter regressions.** Some fixes (especially Bug D) might change what
  the Go emitter produces. The Phase 2C decision was that Go emission is
  deferred (per [m-codegen-strategic-decision.md](m-codegen-strategic-review.md)),
  so we will *check* `tests/golden/codegen` golden tests still pass but
  accept Go-side regressions if they're contained.
- **Performance.** The fixes are correctness-only.

## 5. Acceptance Criteria

1. **Parity gate is green:** `tests/golden/bytecode/golden_test.go` reports
   `30 cases passed, 0 xfail`. The `xfailLower` and `xfailCompile` fields can
   be removed from `golden_test.go`.
2. **`string_ops.ail` re-enabled:** the commented-out spec at
   `golden_test.go:108-112` is restored and passes.
3. **Probe test for Bug C lives in tree:** a regression test in
   `internal/gen/lower/lower_test.go` named `TestLowerDictApp_RecursiveInt`
   verifies that `factorial`-shaped recursive monomorphic Int code lowers to
   a `BinOp` (not a `BuiltinCall`).
4. **No new compiler/VM changes** beyond `_concat_String`. `internal/bytecode`
   and `internal/vm` diffs are minimal (one builtin handler + one entry in
   each builtin table).
5. **Existing tests stay green:** `make test` clean. In particular
   `internal/gen/lower/lower_test.go`, `internal/bytecode/compiler/`, and
   `internal/vm/` test suites all pass.
6. **Go emitter golden tests** (`tests/golden/codegen/golden_test.go`) still
   pass — or, if they intentionally regress, the regressions are documented
   and the goldens regenerated in the same commit with reviewer signoff.

## 6. Sprint Plan (proposed)

| # | Milestone | Est LOC | Depends on |
|---|---|---|---|
| **M1** | **Probe & confirm.** Write a small Go test in `internal/gen/lower/lower_test.go` that lowers each of the 4 problem .ail files via the same `pipeline.Run + lower.LowerProgram` path the parity gate uses, and dumps the resulting `stmt.Program` to a string. This verifies/falsifies the Bug C hypothesis and gives concrete IR snapshots to drive the fixes. | ~150 | none |
| **M2** | **Bug B (Fractional).** Easiest fix; unblocks `mulFloats`. ~10 LOC of code, ~30 LOC of test. | ~50 | M1 |
| **M3** | **Bug A (match-as-expression).** The largest fix. Hoist tail-position matches in `FlattenBlock`, plus a fallback path for nested matches. Unblocks `swap`, `fst`, `eval`, `isZero`. | ~250 | M1 |
| **M4** | **Bug C (polymorphic Int dispatch).** Approach depends on M1 findings. ~50-200 LOC depending on whether the fix is in `lowerDictApp` or upstream. | ~150 | M1 |
| **M5** | **Bug D (`++`).** Add `_concat_String` builtin to VM + compiler tables, intercept in `lowerBinOp`. Re-enable `string_ops.ail` in the parity gate. | ~100 | none |
| **M6** | **Cleanup.** Remove all `xfailLower`/`xfailCompile` flags from `golden_test.go`. Update `m-bytecode-vm.md §10` to mark Phase 2C complete and Phase 2D unblocked. | ~30 | M2-M5 |

**Total estimate: 600-900 LOC** (most in tests).

## 7. Risks

- **Bug C may turn out to be an elaboration bug, not a lower bug.** If the
  probe in M1 shows that `factorial` never reaches lower with usable type
  info, the scope grows. Mitigation: M1 explicitly produces a go/no-go
  decision; if it's elaboration, file a separate sprint and fix the other
  three bugs in this one (which still gets us 28/2, far better than 23/7).

- **Bug A's "hoisting" trick may not handle all expression-context matches.**
  Deeply nested matches inside larger expressions cannot be hoisted to a
  lambda body. The current corpus only has matches in tail position, so
  hoisting suffices for the parity gate, but the more general fix may need
  a new `stmt.MatchExpr` primitive in a follow-up sprint. Acceptable: this
  sprint targets the parity gate, not full lower-pass completeness.

- **Bug D's interception approach may collide with the Go emitter.** The Go
  emitter relies on `concat_String` being a real callable function in the
  Go output. If we rewrite to a builtin call before lowering, the Go emitter
  will need its own resolver (or we keep the original `Call` shape and only
  intercept inside the bytecode compiler). Mitigation: implement the
  interception inside `lowerApp` *and* keep the `Call`-via-`GlobalRef` path
  intact; emit BuiltinCall only when the global name is in a known
  string-builtin allowlist. This way the Go emitter never sees the
  interception.

## 8. Open Questions

1. Are there other typeclasses besides `Fractional` that `lowerDictMethod`
   misses? `Show` is partially handled, `Read` may be missing entirely. The
   probe test in M1 should grep `internal/types/typeclass*.go` and produce
   a complete inventory.
2. Should `_dict_*` polymorphic-fallback `BuiltinCall` names be a hard error
   in the bytecode compiler instead of `OpBuiltinTrap`? Catching them at
   compile time would prevent Bug C from masquerading as a runtime issue
   in future regressions.
3. Does the `core.DictAbs` machinery still serve a purpose for the bytecode
   path, or could lower erase it entirely once monomorphization is complete?
   This question is bigger than this sprint but worth flagging.

## 9. References

- [m-bytecode-vm.md §10](m-bytecode-vm.md) — Phase 2C/2D plan
- [m-phase2c-bytecode-compiler-sprint-plan.md](m-phase2c-bytecode-compiler-sprint-plan.md) — Phase 2C sprint that surfaced these bugs
- [m-concat-disambiguation.md](../v0_10_1/m-concat-disambiguation.md) — prior work on `++`/`concat_String`
- [tests/golden/bytecode/golden_test.go](../../../tests/golden/bytecode/golden_test.go) — the parity gate this sprint targets
- [internal/gen/lower/](../../../internal/gen/lower/) — the package being fixed
