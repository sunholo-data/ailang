# Scope Declassify authority and enforce positive IFC call labels

Closes #752
Refs #1134

Previously, positive parameter labels accepted unrelated argument labels and
Declassify granted whole-body authority. Add opt-in single-label authority via
`Declassify[label=email]` and reject uncovered local-call arguments with
`param_label_cover`. Bare Declassify keeps its existing behavior.

Scoped results preserve unauthorized full argument labels, including labels in
records and closures, even through unlabelled formals. Scoped unlabelled returns
undergo return coverage checks, and lexical bindings override module signatures.
Effect declaration coverage is directional; function-value rows remain invariant.

Validation: types/parser tests, core suite, example gate (234 passed, 9 existing
skips, zero failures), formatting, architecture boundaries and file-size gates
passed. Types statement coverage increased from 52.7% to 52.9%. Contract and
secret compatibility verdicts match the sprint plan. Lint passed with zero issues. Independent review round 2 passed (100/100).
The full suite is deferred to CI as specified by the approved plan.

Cross-module enforcement remains #1134 and must build on this authority summary
and result policy: none=[], bare=["*"], scoped=[single label]. No runtime IFC
change is included.

Base: dev
Branch: coordinator/task-8ffaca1d
