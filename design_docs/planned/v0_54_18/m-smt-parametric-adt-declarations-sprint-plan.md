# M-SMT-PARAMETRIC-ADT: typed error payload verification

**Status**: In progress — Mark approved design and implementation on 2026-10-23
**Target**: v0.54.18 candidate; no tag/deploy
**Baseline**: 2a02b25ae; production source is the audio sprint's tested a8bc30fc9
**Estimate**: 2–4 engineering days, about1800 implementation/test/docs LOC
**AILANG prompt version loaded: v0.16.7** (ailang prompt and --version-active)

## Scope and routing

Implement m-smt-parametric-adt-declarations.md as approved. Native SMT parametric
ADTs retain concrete arguments; constructor context comes from checked types,
not a guessed Int or global outer sort. Unsupported shapes skip honestly.
Do not expand the callee-signature gate, unit encoding, numeric model, runtime
semantics, or parked verification KPI/status split. Unrelated dirty files stay parked.

Registry audit: `ailang pkg search verification` returned firebase_auth and
deontic, both consumers of language/runtime verification rather than a compiler
SMT backend. All milestones record none: reuse internal/smt and existing pipeline
CoreTI, not an external package. Seven-day velocity analysis ran; the completed
audio sprint is the relevant measured integration example, not a sustainable
LOC/day estimate. Estimates include a refutation corpus and typed-context plumbing.

Baseline gates reused: exact production source a8bc30fc9 passed clean full
make test163 packages, lint, examples, boundaries and sizes in the previous stage.
Only documentation changed since that run; no baseline test failure is inherited.

## Milestones and ownership

### M1: Typed ADT declarations and dependency closure (~650 LOC)

**Dependencies**: none
Backend worker owns internal/smt production and associated unit tests. Root owns
CLI production wiring and shared artifacts. Failing tests must precede new code.

- [x] Generic declarations retain formal binders and concrete Result/Option arguments.
- [x] Reachable imported ADTs and contract-only/nested constructor dependencies are retained.
- [x] Unknown/open/ambiguous sorts fail with typed unsupported evidence instead of Int defaults.
- [x] Unrelated aliases and deterministic recursive declaration groups retain existing behavior.

### M2: Checked constructor context and shared verification (~650 LOC)

**Dependencies**: M1
Backend worker extends options/context and encoder; root threads pipeline CoreTI
through verify/ai-check/package entry points using the agreed interface.

- [x] Exact imported ResourceLimit guard verifies with Err payload AudioError and Ok payload Int.
- [x] Deliberately broken success value produces a genuine counterexample.
- [x] Nested/two generic instances and requires/ensures roots use their own concrete context.
- [x] Existing unsupported callee signatures and unit/higher-order boundaries remain honest skips.

### M3: CLI regression corpus, example and release docs (~400 LOC)

**Dependencies**: none for test construction; M1/M2 for final passing verification
CLI-test worker owns new cmd/ailang/smt_parametric_adt_test.go; root owns example,
manifest, public guide and changelog. Worker must record existing failure first.

- [x] Real CLI single-file/package/ai-check paths agree on valid/refuted/unsupported cases.
- [x] New runnable contracts/result_error_guard.ail checks/runs with inline result tests.
- [x] Showcase contracts: include checkSize ensures matching Ok/Err; verify actual script.
- [x] Showcase effects: include IO on main; checkSize and result predicates remain pure.
- [x] Showcase inline tests: include negative,zero and positive size cases.
- [x] Public docs/changelog describe actual supported sorts and retained limitations.

### M4: Integrated gates and independent evaluation (~100 LOC)

**Dependencies**: M1, M2, M3
Root integrates; independent evaluator audits code and adversarial proof/refutation
cases without authoring implementation. No task approvals/messages or release actions.

- [ ] Focused SMT/CLI suite and deterministic count20 declaration tests pass; Z3 availability guarded.
- [ ] Milestone checkpoint, lint, boundaries, sizes, examples/manifest and changelog gates pass.
- [ ] Exact clean implementation snapshot passes full evaluator make test and independent review.
- [ ] Completed JSON, implemented design/companion plan and release evidence are committed.

## Schedule and execution

Days1–2: backend M1→M2 chain; independent CLI regression construction and root
CoreTI/source integration. Days3–4: negative/nested/contract-only controls and
full gates, with bounded independent review. This is an estimate, not a delay
before execution: all work is authorized now. Shared checkout uses explicit
file ownership; workers do not switch branches, stage unrelated files or commit.
Root commits exact task paths and evaluates a clean non-TMPDIR worktree.
