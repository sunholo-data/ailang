### Fixed — monomorphisation captured the free names of a copied lambda body, on every engine (#2401) (2026-11-18)

**This changes program output on all engines (evaluator, `--bytecode`, `--strict-bytecode`) for
the programs it affects; they printed a wrong answer or failed before.** Monomorphisation copies a
polymorphic let-bound lambda's body to its call site, so the body's free names were resolved
again THERE: `let k = 10; let f = \x. k; let g = \k. f(3); g(99)` printed 99 (correct 10), and so
did a call of `f` inside `{ let k = 99; … }` or a match arm `Some(k)`; a local `let r = \y. r(3)`
next to a top-level `r` called itself through the copy; a copied `length(ys) + n` met a list
pattern variable `n` and failed with a type mismatch. An alias followed a later rebinding by
name: `let g = \x. x; let f = g; let g = \x. 7; f(3)` printed 7 (correct 3).

The pipeline now makes local names unique in every declaration right after type checking
(`core.UniqueProgramLocals`, moved from the VM lowering to `internal/core`): a binder is renamed
only when it would share a name with an earlier binder of the declaration or with a free
reference. A copied body or an alias then refers to binders that enclose the call site and that
nothing in between rebinds, so monomorphisation and the name-keyed passes after it are
capture-free by construction (`core.CheckNoShadowing` states the invariant; tests check it on the
pipeline's output). Type checking and its diagnostics run before the renaming, so error
messages, warnings, interfaces and LSP results keep the source names. Every specialisation that
happened before still happens (172 on the tracked corpus and the banked solutions, before and
after).
