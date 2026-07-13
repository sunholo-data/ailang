# Sprint Plan: M-LAMBDA-OPEN-RECORD-PATTERN

> Planned against **HEAD 999bd629f (v0.29.2, -151 dev commits)**. Every code claim
> below was re-verified at this HEAD (the design doc dates from 2026-05-20 / v0.22.0
> and several of its line numbers and one of its hypotheses had drifted — see
> "Reality-Check" below). **Target: v0.29.x patch / v0.30.0.**

## Summary

Make `\obj. match obj { {name, ...} => name }` produce an **open** parameter type
`{name: τ | r}` so callers may pass records with extra fields, exactly as top-level
`let ... in match` already does. The user explicitly wrote `{name, ...}` but the
open/closed intent is erased at the AST→Core boundary, so the lambda parameter
collapses to a **closed** `{name: τ}` and the call site is rejected. Secondary DX
defect: the resulting error's Hint tells the user to write `{name, ...}` — which they
already wrote.

**Duration:** ~1 day (1.0–1.5 days with buffer)
**Dependencies:** None
**Risk Level:** Medium (row-variable over-generalization is the key risk)

---

## Reality-Check vs the Design Doc (verified at HEAD)

The controller asked me to re-verify every premise. Results:

| Design-doc claim | Verdict at HEAD | Evidence |
|---|---|---|
| **H1**: `core.RecordPattern` has NO `Rest` field; elaborator drops open/closed intent | ✅ **CONFIRMED** | `internal/core/core.go:380-387` — struct is `{ Fields map[string]CorePattern }`, no `Rest`. `internal/elaborate/patterns.go:207-219` builds `&core.RecordPattern{Fields: fields}` and never reads `p.Rest` (the AST *does* carry it — `internal/ast/ast_patterns.go:69`). |
| Cited line `patterns.go:177-189` | ⚠️ **DRIFTED** — the `ast.RecordPattern` case is now at **`patterns.go:207-219`** | direct read |
| **H2**: `checkPattern` uses a rowVar only when scrutinee is unresolved | ✅ still structurally true, but the `*TRecord` branch is a real second gap: when the scrutinee already resolves to a **closed** `TRecord`, `fieldTypes = recTy.Fields` is taken verbatim and `recTy.Row` is ignored, with no way to force openness (no Rest flag reaches here) | `internal/types/typechecker_patterns.go:394-416` (cited as 365-387 — **DRIFTED**) |
| **H3 (doc's "most likely")**: generalize doesn't quantify the ROW variable → param collapses to closed at lambda-exit | ⚠️ **PARTIALLY REFUTED as the *primary* cause.** The `CoreTypeChecker` path (the one `ailang check` uses) *does* hardcode `RowVars: []string{} // Simplified for now` in `generalizeWithConstraints` (`typechecker_functions.go:462-464`), AND `Scheme.Instantiate` only re-freshens row vars at `EffectRow` kind, never `RecordRow` (`types_v2.go:541-546`) — so generalization is *a* gap. **BUT** a direct-applied lambda (IIFE, no let → no generalize) **still fails** (probe below). So the collapse happens *before/without* generalization; H3 alone cannot explain it. |

### Probe matrix (built `go build -o /tmp/plan-ailang ./cmd/ailang`, module-wrapped `check`)

| Shape | Expected | Observed at HEAD |
|---|---|---|
| lambda + `{name, ...}` + extra-field caller | PASS | ❌ **FAIL** ("record field mismatch: expected 1, got 2") — **the bug** |
| top-level `let big = {...} in match big { {name, ...} }` | PASS | ✅ PASS |
| lambda + `{name}` (closed) + extra-field caller | FAIL | ✅ FAIL (correct — must stay) |
| lambda + `{name, ...}` + matching-shape caller | PASS | ✅ PASS |
| **IIFE** `(\obj. match obj { {name, ...} => name })({name,id})` (no let/generalize) | PASS | ❌ **FAIL** — *localizes the collapse to the pattern/param path, NOT generalize* |
| `func getName(obj: {name: string \| r})` + open pattern + extra | PASS | ✅ PASS — *the open-record machinery & unification are healthy; only the pattern→param inference drops openness* |

**Conclusion on mechanism:** The lambda parameter is a fresh `TVar` (`typechecker_functions.go:47`).
`checkPattern` branch #2 emits `paramTVar ~ {name: τ | rowVar}` — an *open* constraint.
Yet the call site rejects the extra field, and the IIFE (no generalization) fails too.
So the open row is being **closed to empty before/at the application unify**, independent of
generalization. The doc's H3 is a *real secondary* gap (needed for the *let-bound* variant to
stay open once M1 lands) but is **not** the primary cause. The primary fix is structural:
**carry the `Rest` flag into Core and honor it in `checkPattern` so the param genuinely becomes an
open record**, then confirm openness survives solve/defaulting (and, for the let-bound case,
generalization).

---

## Current Status Analysis

### Velocity (recent, from git + CHANGELOG)
Recent milestones (M-SCHEME-IMPORT-PRESERVE-ADT-HEAD sibling, brain-vectors, etc.) land
~150–250 LOC/milestone/session including tests. This sprint is small and surgical
(one struct field + two type-checker sites + tests); ~1 day is realistic.

### Related implemented work to not regress
- `internal/types/row_unification.go`, `unification_records.go` (open/closed row unify — healthy per probe).
- M-SCHEME-IMPORT-PRESERVE-ADT-HEAD (`design_docs/implemented/v0_22_0/`) — the sibling that
  removed the over-polymorphism hiding this bug. Its tests MUST stay green.

---

## Proposed Milestones

### Milestone 1 — Carry `Rest` into Core + honor it in `checkPattern` (core fix)

**Goal:** Stop erasing open/closed intent. `\obj. match obj { {name, ...} => name }` must
type as `({name: τ | r}) -> τ`; the IIFE and the extra-field caller must pass. Closed
`{name}` must remain closed (still reject extra fields).

**Files & LOC:**
- `internal/core/core.go` (~+3): add `Rest bool` to `RecordPattern`; update `String()`.
- `internal/core/gob.go` — verify `gob.Register(&RecordPattern{})` still fine (no schema break; new bool zero-values on old blobs). Check the cache/gob round-trip. (~0–5)
- `internal/elaborate/patterns.go:207-219` (~+1): propagate `Rest: p.Rest`.
- `internal/types/typechecker_patterns.go:394-416` (~+15): when `p.Rest == true`, ALWAYS
  build the open constraint with a fresh `freshRecordRow()` **even when `scrutType` already
  resolves to a `*TRecord`** — i.e. do not take `recTy.Fields` verbatim; instead constrain
  `scrutType ~ {matched fields | freshRow}` so the param stays open. When `Rest == false`,
  keep today's closed behavior exactly (this is what preserves the correct closed-record
  rejection). Nested open patterns (`{user: {name, ...}, ...}`) must recurse correctly.
- Grep for other `core.RecordPattern` constructors/consumers (eval, exhaustiveness,
  typedast lowering, SMT codegen) that may need to tolerate the new field. (~audit only)

**Tasks (Day 1 morning):**
1. Add field + elaborator propagation; `go build`.
2. Rewire `checkPattern` `*core.RecordPattern` branch on `Rest`.
3. Instrument (temporary `DEBUG` print of the solved param type) to CONFIRM the param
   resolves to an *open* `TRecord` for the IIFE case — this pins whether M1 alone closes the bug
   or whether M2 (generalize/instantiate row vars) is also required for the *let-bound* case.

**Acceptance:**
- [x] IIFE `(\obj. match obj { {name, ...} => name })({name,id})` → PASS.
- [x] lambda + `{name}` (closed) + extra caller → still FAIL.
- [x] `make build` clean; no gob/cache round-trip breakage.

**Risk:** Over-forcing openness could make closed `{name}` accept extra fields (soundness
regression, reversing the M-SCHEME-IMPORT strictness). Mitigation: gate strictly on
`Rest == true`; the closed-record probe is a required red-line test.

---

### Milestone 2 — Preserve openness through generalize/instantiate (let-bound variant)

**Goal:** The *let-bound* form `let getName = \obj. match ... in getName({name,id})` must ALSO
pass. Per the probe, the IIFE is fixed by M1, but the let-bound form is generalized by
`CoreTypeChecker.generalizeWithConstraints`, which today discards record row vars
(`RowVars: []string{}`), and `Scheme.Instantiate` only refreshes row vars at `EffectRow`
kind. **Only do the work here that M1's instrumentation proves is still needed.**

**Files & LOC:**
- `internal/types/typechecker_functions.go:427-468` (~+15): quantify free **record** row vars
  in the scheme (mirror the already-correct `inference_helpers.go:128-166 generalize`, which
  DOES collect `generalizedRowVars`), withholding rows free in the enclosing env exactly like
  type vars. Populate `Scheme.RowVars`.
- `internal/types/types_v2.go:533-559` (`InstantiateWithConstraints`) (~+8): allocate fresh
  row vars **at the correct kind** (`RecordRow` vs `EffectRow`). Today all quantified row vars
  are freshened as `EffectRow` — wrong for record rows. Determine kind from the scheme (track
  kind alongside the var, or infer from usage).

**Tasks (Day 1 afternoon):**
1. If M1's instrument shows the let-bound param already stays open (e.g. because the value is
   a syntactic lambda and generalization is value-restricted / the row is env-free), **M2 may
   shrink to a no-op + a regression test only** — record that in the JSON notes.
2. Otherwise quantify record row vars + fix instantiate kinds.

**Acceptance:**
- [x] let-bound `getName({name,id})` → PASS.
- [x] No over-generalization: a genuinely incompatible record is still rejected (add a
      negative probe, e.g. call with a record MISSING `name`).

**Risk (TOP RISK):** Row-variable over-generalization in HM-with-rows — quantifying a row var
that is actually shared/constrained elsewhere can accept incompatible records. Mitigation:
mirror the vetted `inference_helpers.generalize` withholding logic exactly; add the
missing-field negative test; run the full `internal/types` + `internal/pipeline` suites and
`-race` on the new test.

---

### Milestone 3 — Misleading-hint DX fix (open-record hint suppression)

**Goal:** The error Hint at `internal/types/unification_records.go:516-520` suggests
`{name | r} or {name, ...}` whenever there are extra fields and no missing fields. When the
user ALREADY wrote an open pattern (now that M1 lands and it works, this fires far less), or
in any residual closed-vs-extra mismatch where the suggested syntax equals what's on the page,
the hint is actively misleading.

**Approach (pick the lighter that survives M1):**
- After M1, the *primary* reproducer no longer errors, so the misleading hint mostly
  disappears on its own. Remaining case: closed `{name}` + extra caller still errors and still
  shows the hint — which is now *correct* (telling them to add `...`). So the hint is fine there.
- **Minimal, honest change:** verify with a probe that post-M1 the hint only appears in cases
  where following it would actually help (closed pattern / plain closed-record annotation). If
  a case remains where the emitted `expected` set already appears open in source, suppress or
  reword. If none remains, this milestone reduces to a **regression test asserting the hint no
  longer fires on the open-pattern reproducer** + a one-line note.

**Files & LOC:**
- `internal/types/unification_records.go:516-520` (~0–8, likely 0): only touch if a probe shows
  a genuinely misleading residual case.
- Test: assert Hint absent on the fixed open reproducer.

**Acceptance:**
- [x] Post-fix, the open-pattern reproducer produces NO error (so no hint) — asserted by test.
- [x] Closed-pattern-+-extra still shows the (now-correct) hint.

**Risk:** Low. Don't over-engineer source-position plumbing into the unifier; the structural M1
fix removes the misleading path.

---

## Regression Test Plan — the 4-shape matrix

Place per current conventions in **`internal/pipeline/lambda_open_record_test.go`**
(`package pipeline_test`, `pipeline.Run(cfg, src)` with `Code:`/`Filename:""`, as in
`poly_arithmetic_test.go`). These are end-to-end and exercise elaborate→typecheck→solve.

| Test | Code (module-less `Code:` form) | Expect |
|---|---|---|
| `TestLambdaOpenRecord_ExtraField_Pass` | `let getName = \obj. match obj { {name, ...} => name } in getName({name: "Grace", id: 123})` | PASS → "Grace" |
| `TestLambdaOpenRecord_MatchingShape_Pass` | same lambda, caller `{name: "Grace"}` | PASS → "Grace" |
| `TestLambdaClosedRecord_MatchingShape_Pass` | `{name}` pattern, caller `{name: "Grace"}` | PASS → "Grace" |
| `TestLambdaClosedRecord_ExtraField_Fail` | `{name}` pattern, caller `{name: "Grace", id: 123}` | **type error** (assert error non-nil; the strictness must NOT be reversed) |
| `TestLambdaOpenRecord_IIFE_ExtraField_Pass` | IIFE form (no let) — guards M1 independently of M2 | PASS |
| `TestLambdaOpenRecord_MissingField_Fail` | open pattern, caller record LACKS `name` | **type error** (guards M2 over-generalization) |
| `TestLambdaOpenRecord_HintNotMisleading` | run the open reproducer, assert NO "Use open record syntax" hint in output | no hint |

(If `pipeline.Run` cannot easily distinguish "type error" vs runtime for the FAIL cases, assert
on the returned `err` / result error string as the sibling tests do.)

**Restore the example:** `examples/runnable/record_patterns.ail` — revert the
lines 82–94 workaround back to `{name, ...}` in the lambda with an extra-field caller
(the file's own comment at 84–87 documents the workaround). Must pass `make verify-examples`.

---

## Success Metrics
- All 7 new tests pass, including the two **negative** (closed+extra, missing-field) tests.
- `make test` green (esp. `internal/types`, `internal/pipeline`, `internal/elaborate`).
- `make verify-examples` green with restored `record_patterns.ail`.
- `go test -race ./internal/pipeline/ -run LambdaOpenRecord` green.
- No regression in M-SCHEME-IMPORT-PRESERVE-ADT-HEAD tests.
- `make lint` clean.

## Verification Steps (sprint-executor)
1. `go build ./...` after each milestone.
2. `go test ./internal/types/... ./internal/pipeline/... ./internal/elaborate/...`
3. `go test -race ./internal/pipeline/ -run LambdaOpenRecord`
4. `make verify-examples`
5. `make test && make lint`
6. Re-run the 6-row probe matrix above via a temp `check` binary and confirm every row matches
   "Expected".

## Risk Register
| Risk | Severity | Mitigation |
|---|---|---|
| **Row-var over-generalization** (M2) accepts incompatible records | **High (top risk)** | Mirror vetted `inference_helpers.generalize` withhold logic; missing-field negative test; full type suite + `-race`. |
| Closed-record strictness silently reversed (M1) | High | Gate openness strictly on `Rest==true`; closed+extra red-line test. |
| gob/cache schema break from new `Rest bool` on `core.RecordPattern` | Medium | New bool zero-values on old blobs; verify cache round-trip test; bump cache key if needed. |
| Instantiate row-kind fix (M2) mis-kinds effect rows | Medium | Track kind per quantified var; run effect-heavy tests (`effect_soundness_test.go`). |
| Other examples silently relied on the hidden over-polymorphism | Medium | Full `make verify-examples` sweep post-fix. |

## Open Questions
- Does M1 alone fix the let-bound reproducer (value-restriction may keep the row env-free),
  or is M2 strictly required? **Resolved during M1 via instrumentation** — the plan is
  structured so M2 shrinks to a test-only milestone if M1 suffices.
- Kind tracking in `Scheme.RowVars`: add a parallel `RowVarKinds []Kind`, or infer kind from
  first usage in `Type`? (Executor's call; prefer explicit kind tracking for determinism.)

## Notes / Assumptions
- Cited line numbers in the design doc are stale (v0.22.0); corrected numbers are in the
  Reality-Check table. Executor should re-grep, not trust either doc blindly.
- The design doc's H3 ("generalize is most likely") is **downgraded** to a secondary gap; H1
  (Core drops Rest) is the confirmed structural cause. Milestones are ordered accordingly.
- Out of scope (unchanged from design doc): open-record patterns in non-match `let` bindings;
  row subtyping.
