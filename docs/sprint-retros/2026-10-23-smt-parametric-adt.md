# M-SMT-PARAMETRIC-ADT retrospective — 2026-10-23

The upstream ResourceLimit report reproduced on the completed audio sprint.
The missing declaration was one symptom: Result's generic arguments had also
been erased and both payloads encoded as Int. Adding only AudioError would
therefore have left the contract ill-sorted. Mark approved a separate design
and implementation for typed parametric declarations.

## Execution

Backend work owns internal/smt, a separate worker owns CLI regressions, and root
owns checked-type plumbing, the runnable example and documentation. Independent
review owns the proof/refutation audit. All work stays in the existing checkout,
with unrelated benchmark and planning edits preserved.

The release check pairs each supported proof with a deliberately false contract
through the same encoding. The corpus covers imported ResourceLimit, nested
Option/Result, two differently typed instances, contract-only dependencies and
unsupported unit/open/higher-order boundaries. Native parametric declarations
reuse the existing deterministic declaration grouping.

## Lessons

- Read the emitted SMT script: fixing an unknown constructor requires preserving
  its actual payload sort as well as finding its declaration.
- Use checked types at each constructor node. The outer return type cannot
  describe all nested or intermediate generic values.
- Keep module-specific type information when encoding imported helper bodies.
- Check nested patterns for valid SMT syntax and first-match semantics.
- Scope parameter renaming around local binders. Independent review caught a
  nested-pattern shadowing case that incorrectly proved an outer parameter's
  condition instead of checking the bound payload; that became a blocking
  counterexample regression before the final source freeze.
- Verify the broken case reaches a genuine SAT result rather than accepting
  either an unsupported skip or a solver error as a negative control.
- Inline examples use primitive expected values through a result predicate;
  imported constructors are not currently bound in the expected-value evaluator.

## Gate record

Final source snapshot, test counts and independent verdict are recorded in the
sprint JSON and evaluation report. This sprint prepares the v0.54.18 candidate;
it does not publish a release or decide pending coordinator approvals.

Final exact source ef7bdc8e0 passed the full clean suite: 163 tested packages,
11,049 top-level PASS, 105 SKIP and zero failures. Lint, sizes, examples/manifest,
boundaries, changelog, WASM and Windows test compilation passed. Independent
review passed 20 adversarial fixtures with 27 function outcomes. The initial
checkpoint's lint failure identified four obsolete helpers; removing them and
rerunning focused tests/lint resolved it. Full clean evaluation then covered
the final source. The evaluator script's pre-existing malformed TODO-count
JSON trailer was bypassed using its human summaries and preserved raw test log;
the independent evaluation is strict JSON.

Independent final verdict: PASS 95/100, with no unresolved source or artifact blockers.
