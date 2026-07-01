## M-DERIVING-EQ-RUNTIME: Auto-derived `Eq` dictionary missing at runtime + module-eval errors exit 0

**Status**: PLANNED (two related bugs)
**Target**: v0.27.x
**Priority**: P1 (the exit-0 bug is high — it lets runtime failures pass naïve gates); P2 (the Eq-dict bug)
**Estimated**: 1–2 days (two independent fixes)
**Dependencies**: None.

**Found during**: M-SNAKE-FEEDBACK (building `verify-examples-toplevel`). Verified on v0.26.2.
**Independently confirmed**: Felix / Snake Showdown team via `submit_feedback` (ticket `fb_cef305f96ccc24ae`, v0.27.0) — reports deriving `(Eq)` on user ADTs "fails at runtime inside **lambda closures**", which is a useful narrowing of the repro (the missing-dictionary error surfaces when the derived instance is used inside a closure). Contact: kevin.faurholt@gmail.com.

---

## Bug 1 — Auto-derived `Eq` dictionary is not resolved at runtime

`examples/deriving_eq.ail` type-checks cleanly but **fails at evaluation**:

```
$ ailang run --caps IO --entry main examples/deriving_eq.ail
✓ Running examples/deriving_eq.ail
Error: module evaluation failed: failed to evaluate let colorTest in module
examples/deriving_eq: missing dictionary method: prelude::Eq::Color::eq
```

`Color` is an ADT with auto-derived `Eq` (M-DX19, [commit bb3b3a83](../../examples/deriving_eq.ail)).
The type checker accepts `colorTest = (Red == Red)` and friends, but the runtime has no
`prelude::Eq::Color::eq` dictionary entry — the derived instance is type-checked but never
materialized (or not registered under the name the evaluator looks up). So `deriving (Eq)`
is, in at least this case, **type-safe but not runnable**.

This is the only example using auto-derived `Eq`, so the gap shipped uncaught. Scope check:
audit auto-derived `Eq`/`Ord`/`Show` for other ADT shapes (nullary constructors vs.
constructors with fields; nested ADTs) — the dictionary may be missing for a whole class, not
just `Color`.

### Pointers
- Derived-Eq design: M-DX19 ("Auto-derive Eq for ADT types").
- Error string `missing dictionary method` — grep the evaluator's dictionary lookup
  (`internal/eval`) and the deriving/elaboration path that should register
  `prelude::Eq::<Type>::eq`.

## Bug 2 — Module evaluation errors exit with status 0

The run above prints `Error: module evaluation failed: …` but **exits 0**:

```
$ ailang run --caps IO --entry main examples/deriving_eq.ail; echo "exit: $?"
... Error: module evaluation failed ...
exit: 0
```

A non-zero exit is the contract every CI gate, shell pipeline, and `&&` chain relies on. Exit
0 on a hard evaluation error means:
- Naïve example gates (and the prior CI) report success on a failing program — this is part of
  why the rotted examples went unnoticed; output-grep was the only signal that worked.
- `verify-examples-toplevel` had to grep stdout for `Error:` rather than trust `$?`.

This is likely separate from Bug 1 (it would mask *any* eval error, not just deriving). It may
share a root with the budget-exhaustion path, which also surfaces as a printed error.

### Pointers
- `executeModuleEntrypoint` error handling — [cmd/ailang/main_run_exec.go](../../cmd/ailang/main_run_exec.go)
  (the `execErr != nil` branch flushes + `os.Exit(1)` for *some* paths; module-eval errors
  appear to print and fall through to a 0 exit). Trace which error path prints
  `module evaluation failed` and why it doesn't reach `os.Exit(1)`.

## Acceptance Criteria

- [ ] **Bug 2 first** (higher leverage): any module-evaluation error exits non-zero. Add a test
      that runs a deliberately-failing program as a subprocess and asserts `exit != 0`. Once
      this lands, `verify-examples-toplevel` can trust the exit code (simplify the `Error:` grep).
- [ ] **Bug 1**: `examples/deriving_eq.ail` runs and prints its three boolean tests; the
      derived `prelude::Eq::<Type>::eq` dictionary is registered for nullary-constructor ADTs
      (and the audit covers field-bearing/nested ADTs).
- [ ] `deriving_eq` removed from the `verify-examples-toplevel` run-skip list.
- [ ] Regression test asserting `Red == Red` evaluates to `true` for a `deriving (Eq)` ADT.
