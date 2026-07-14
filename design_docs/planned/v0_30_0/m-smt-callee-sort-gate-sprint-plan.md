# Sprint Plan: M-SMT-CALLEE-SORT-GATE

**Design doc**: [m-smt-callee-sort-gate.md](m-smt-callee-sort-gate.md)
**Target**: v0.30.0
**Risk level**: Low
**Estimated duration**: 1 day (~4 hours)
**Total LOC estimate**: ~200 (impl + tests + example)

## Sprint Summary

**Goal:** `ailang verify` must never emit a raw Z3 solver crash because a *callee's signature*
is unencodable (e.g. `Option[float]`). It must skip the caller with a structured
`UNENCODABLE_TYPE` reason naming the callee + sort.

**Deliverables:**
1. Caller-level sort gate in the encodability check (primary fix).
2. Defense-in-depth `define-fun` sort validation (belt-and-suspenders).
3. Regression example + unit/integration tests + CHANGELOG.

## Current Status Analysis

- **Reproduced live (v0.29.2):** the `gradeNumeric`/`convertTo` case produces Z3 exit-1 with
  "unknown sort 'Option'" / "unknown constant convertTo".
- **Root cause mapped:** the fragment gate (`internal/smt/encodable.go`) only checks
  `$builtin`/stdlib call *names*, never cross-function callee *signature sorts*. Two leak paths
  (`callee_resolver.go` inline-with-undeclared-sort, `codegen_apps.go` raw-symbol fallthrough)
  and one missing guard (`validateDeclarations` skips `define-fun`).
- **Skip-with-reason infrastructure already exists** (`RejectUnencodable`, `verify.go:294-307`,
  `ErrUnresolvableTypes` graceful path `verify.go:358-371`) — we route into it.

## Milestones

### M1: Caller-level callee-sort gate (~2.5h, ~110 LOC)

**Description:** Add `firstUnencodableCalleeSort` walker to `internal/smt/encodable.go` and wire
it into `IsSMTEncodable`. When a contracted function's body calls a user/stdlib callee whose
return type or any parameter type maps to a non-primitive, non-declarable sort, reject the
*caller* with `RejectUnencodable` naming the callee + sort.

**Files:**
- `internal/smt/encodable.go` (+~60): new walker + check 7 in `IsSMTEncodable`.
- `cmd/ailang/verify.go` (+~15): pass callee signature-sort lookup + declarable-ADT set.
- `cmd/ailang/ai_check.go` (+~15): mirror the same wiring.

**Acceptance criteria:**
- `gradeNumeric`/`convertTo` repro reports `Status: skipped` + `UNENCODABLE_TYPE` naming callee `convertTo` and sort `Option`.
- No Z3 exit-status-1 / "unknown sort" / "unknown constant" on the repro.
- `cross_function.ail` (int callees) and a user-enum-returning callee still verify (not falsely rejected).

**Dependencies:** none.

### M2: Defense-in-depth validation + tests + example + docs (~1.5h, ~90 LOC)

**Description:** Extend `validateDeclarations` (`codegen.go:642`) to scan emitted `(define-fun …)`
signature sorts against primitive-or-declared, returning `ErrUnresolvableTypes` (graceful skip)
on violation. Add regression example + tests + CHANGELOG.

**Files:**
- `internal/smt/codegen.go` (+~30): scan `define-fun` param/return sorts.
- `examples/runnable/contracts/unencodable_callee_skip.ail` (new, ~20): the repro.
- `internal/smt/encodable_test.go` / codegen test (+~80): unit tests for the walker + validation.
- `CHANGELOG.md`: entry.

**Acceptance criteria:**
- `validateDeclarations` rejects a hand-crafted `(define-fun f ((x Real)) Option …)`; accepts `Real`.
- Integration test: `ailang verify` on the new example → `skipped` (assert via `--json`).
- `make test` passes; no regression in `examples/runnable/contracts/*.ail` verification.

**Dependencies:** M1 (shares the primitive-or-declarable sort predicate).

## Task Breakdown (single day)

1. M1: write `firstUnencodableCalleeSort` + declarable-sort predicate (~1h)
2. M1: wire into `IsSMTEncodable` + both call sites, emit reason (~1h)
3. M1: manual verify repro skips cleanly (~0.5h)
4. M2: extend `validateDeclarations` (~0.5h)
5. M2: example + unit + integration tests (~0.75h)
6. M2: CHANGELOG + `make test` + regression sweep (~0.25h)

## Success Metrics

- [x] Repro skips with `UNENCODABLE_TYPE` (no Z3 crash) — acceptance (`TestVerify_UnencodableCalleeSkipsNotErrors`)
- [x] `cross_function.ail`/`temperature.ail`/`record_verify.ail` still verify — regression (12 examples byte-identical pre/post; `TestVerify_CrossFunctionIntChainStillVerifies`)
- [x] `validateDeclarations` unit test green — belt-and-suspenders (`TestValidateDeclarationsRejects/Accepts*`)
- [x] `make test` — touched packages (`cmd/ailang`, `internal/smt`) green; full `go build ./...` clean
- [x] Example `unencodable_callee_skip.ail` added + verified (runs → `7.0`; verify → SKIPPED; `make verify-examples` green)
- [x] CHANGELOG updated (`changelogs/v0.18-current.md`, [Unreleased])
- [x] Fix mirrored to the `ailang ai-check` twin (not just `ailang verify`)

## Risks

| Risk | Mitigation |
|------|-----------|
| Gate falsely rejects declarable user ADT/record callees | Declarable predicate admits primitives + `adtTypes` + record sorts + `(Seq …)`; regression tests guard |
| Gate misses a transitive/indirect callee | M2 `validateDeclarations` catches any leak → graceful skip, never a crash |
| `IsSMTEncodable` signature change ripples | Only 2 call sites (grep-confirmed) |

## Open Questions

None blocking. `Hint:` string wording is agent's latitude (per design doc Deferred Decisions).

---

**SPRINT_PLAN_PATH**: design_docs/planned/v0_30_0/m-smt-callee-sort-gate-sprint-plan.md
