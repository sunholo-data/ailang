# M-CONTRACT-VERIFICATION-COVERAGE: Split `verify_skipped` into `skipped` vs `not_applicable`

**Status**: Planned
**Target**: v0.33.2
**Priority**: P0 (bar-gating — feeds the v1.0 headline KPI's data integrity)
**Estimated**: 1.5–2 days (2 milestones, **reader-before-writer**, each independently committable)
**Dependencies**: None. Explicitly does NOT depend on `D-29` (see [Out of Scope](#out-of-scope--the-d-29-flip-is-parked-and-stays-parked)).
**Author**: design-doc-creator role, mission iteration 255 (2026-08-23, at `origin/dev` = `30176187f`)
**Revision**: r2 — round-1 quorum verdict was **blocked** (2/2 reviewers, same defect:
the original M1→M2→M3 order was writer-before-reader and lossy in the window). This
revision applies both reviewers' fixes verbatim; see the
[Quorum verification log](#quorum-verification-log) at the end.

---

## Problem Statement

AILANG's contract verifier reports a per-function `status`, and the eval harness
aggregates it into a single flat `verify_skipped` counter. That ONE counter conflates
two semantically different outcomes:

1. **"The SMT encoder could not encode this function"** — a real coverage limitation
   of the verifier.
2. **"This function has no `ensures` clause, so there is nothing to verify"** — the
   benchmark spec's own design. The model did nothing wrong; the spec deliberately
   carries the contract in a separate *proof* function.

The observatory's `isVerifiedSuccess` predicate
(`internal/observatory/cost_per_verified_success.go:103`) requires
`VerifySkipped == 0`, so a benchmark run is disqualified from the headline
cost-per-verified-success KPI for COMPLYING with the benchmark's own contract spec.

**Measured on the frozen `v1.0` cohort** (mission iteration 254, 30 runs — provenance:
inherited from the iteration-254 mission-log entry, where it was re-derived by two
independent routes that agreed exactly; see [Provenance](#provenance-of-the-cohort-numbers)):
of 53 skips, **24** are class (2) — `no ensures clause (nothing to verify)` — and
**29** are class (1). The five functions carrying class (2) — `isBST`, `encode`,
`decode`, `toRoman`, `minor3` — are declared in the benchmarks' own `contract_spec`
with `requires` and no `ensures`; their contracts live in separate proof functions
(`roundtrip`, `singleCharEncode`, `insertPreservesBST`) which DO carry `ensures`.
All five skip in all 5 models — a property of the spec, not of any model.

This is a data-integrity defect on its own merits, whichever way the pending KPI
ruling goes (CLAUDE.md Principle 2): today the encoder-coverage figure a reader
needs — **29** — is hidden inside a **53** that is 45% not-about-the-encoder.

### Provenance of the cohort numbers

The 53/24/29 partition and the five function names are **inherited** from the
iteration-254 mission-log entry (`design_docs/v1-mission-log.md`, section
`## 254 — 2026-08-23`), where the counts were re-measured via two independent routes
(flat `verify_*` fields vs nested `verify_json.verify.results[]` arrays) with a firing
control (`roundtrip`/`isLeapYear` verified in the same files). This doc does NOT
build its correctness argument on those numbers — they are motivation. Every claim
the DESIGN depends on (predicate text, emission sites, propagation surface, live KPI
value) was re-verified first-party at HEAD `30176187f` on 2026-08-23; see the
[Verification Log](#verification-log).

---

## Goals

**Primary goal**: make the verifier's "nothing to verify" outcome a first-class,
separately-counted status (`not_applicable`) at every layer that today carries
`verify_skipped` — while keeping every existing behavior, exit code, and KPI value
**byte-identical**.

Success metrics:
1. `ailang ai-check --json` and `ailang verify --json` report a `not_applicable`
   count distinct from `skipped`; per-function results carry `status: "not_applicable"`
   for the no-ensures case.
2. Newly-banked eval rows carry `verify_not_applicable` alongside `verify_skipped`.
3. The published `v1.0` baseline KPI still reads exactly
   `cost_per_verified_success_usd = 0.7778187071999999` with `verified_successes = 3`
   (asserted by a live command, not by inspection).
4. The future `D-29` flip is a **one-line change** to `isVerifiedSuccess`, and this
   doc shows that diff without shipping it.

## Non-Goals

- Changing which runs count as verified successes (that is `D-29` — see below).
- Any change to SMT encoder coverage.
- Any edit to benchmark YAML / contract specs.
- Any SQL schema migration (none is needed — verified below).
- Any change to `ai-check`'s exit-code contract (skipped-only exits 0 today;
  not_applicable-only must also exit 0 — it is a strict refinement of "skipped").

## Out of Scope — the D-29 flip is PARKED and stays parked

Changing `isVerifiedSuccess` to **ignore** `not_applicable` would move the published
headline KPI from **$0.7778** to **$0.2121** (iteration 254's arm B) and changes a
definition Mark ratified on 2026-07-27. That is open decision **`D-29`** and is
Mark's call, not ours. This sprint must NOT ship it, and the sprint-evaluator should
treat its accidental inclusion as a failing defect (the mutation table below contains
a test whose *purpose* is to go red if the flip lands).

The design makes the flip a one-line change later. The exact diff that would
implement it, **for D-29's record only — DO NOT APPLY**:

```diff
 	return a.CompileOk && a.RuntimeOk && a.StdoutOk &&
 		a.VerifyOk &&
 		a.VerifyVerified > 0 &&
 		a.VerifyCounterex == 0 &&
-		a.VerifySkipped+a.VerifyNotApplicable == 0 &&
+		a.VerifySkipped == 0 &&
 		a.VerifyErrors == 0
```

(A matching one-line decision then exists for `isVerificationFailure`; D-29 should
rule on both together.)

---

## Verification Log

All commands run 2026-08-23 in `/Users/voightkampff/.ailang-driver-pin/v1`, a clean
worktree detached at `origin/dev` = `30176187f`. Empty/negative results carry a
known-positive control in the same call, scoped to the same path.

| # | Claim | Command | Observed output |
|---|-------|---------|-----------------|
| V1 | `isVerifiedSuccess` requires `VerifySkipped == 0` at line 103 | `Read internal/observatory/cost_per_verified_success.go` | Lines 95–105: `return a.CompileOk && a.RuntimeOk && a.StdoutOk && a.VerifyOk && a.VerifyVerified > 0 && a.VerifyCounterex == 0 && a.VerifySkipped == 0 && a.VerifyErrors == 0` — the `VerifySkipped == 0` clause is line 103. |
| V2 | `isVerificationFailure` includes `VerifySkipped > 0` at line 124 | same Read | Line 124: `return a.VerifyCounterex > 0 || a.VerifySkipped > 0 || a.VerifyErrors > 0` (guarded by `isPass` at line 121). |
| V3 | The no-ensures outcome is emitted at exactly TWO sites, both `Status: "skipped"`, with a duplicated `hasEnsures` loop | `sed -n '295,330p' cmd/ailang/ai_check.go; sed -n '285,320p' cmd/ailang/verify.go` | `ai_check.go:303-309` and `verify.go:290-296` each contain an identical `hasEnsures := false; for _, c := range meta.Contracts { if c.Kind == core.EnsuresKind {...} }` loop; `ai_check.go:311-315` and `verify.go:298-304` each emit `Status: "skipped", Reason: "no ensures clause (nothing to verify)"` and increment the skip counter. |
| V4 | Full taxonomy: FOUR skip-emission branches per CLI, only one of which is the no-ensures case | `grep -n '"skipped"' cmd/ailang/ai_check.go cmd/ailang/verify.go` plus reads of each site | `ai_check.go`: :296 (body not found in Core AST), :312 (no ensures), :346 (not SMT-encodable, joined rejection reasons); `verify.go`: :282, :300, :346 (same three); plus the shared helper `unresolvedTypeVerifyResult` (`verify_filter.go:25-28`, Status `"skipped"`, unresolvable SMT sorts) used at `ai_check.go:374` and `verify.go:389`. |
| V5 | The `VerifySkipped` propagation surface is exactly 10 non-test sites | `grep -rn "VerifySkipped" --include='*.go' cmd internal \| grep -v _test.go` (rc=0) | Exactly: `cmd/ailang/eval_benchmark_agent.go:302,:406` · `cmd/ailang/eval_benchmark.go:282` · `internal/observatory/cost_per_verified_success.go:103,:124` · `internal/observatory/models_chains.go:165` (`json:"verify_skipped"`) · `internal/eval_harness/metrics.go:104` · `internal/eval_harness/agent_runner.go:147` · `internal/eval_harness/verify.go:140,:153`. |
| V6 | `not_applicable`/`NotApplicable` appears NOWHERE in the Go tree (genuinely new value) | `grep -rin "notapplicable\|not_applicable" --include='*.go' cmd internal` → rc=1; control in same call: `grep -c "VerifySkipped" internal/eval_harness/verify.go` → `2`, rc=0 | Negative confirmed with a firing control on the same tree. |
| V7 | `verify_skipped` is NOT in either SQL schema → **no SQL migration needed**; it is persisted as JSON inside `EvalAssessment` | `grep -c "verify_skipped" internal/observatory/schema.sql` → `0` rc=1; same for `schema_chains.sql` → `0` rc=1; control in same call: `grep -c "CREATE TABLE" …` → `10` and `2` | Both negatives controlled by `CREATE TABLE` firing 10 and 2 times in the same files. |
| V8 | The live published KPI at HEAD is exactly `0.7778187071999999` with 3 verified successes, 26 verification_failures, 30 runs | `ailang chains stats --cost-per-verified-success --baseline v1.0 --json` | `"total_runs": 30, "passed_runs": 29, "verified_successes": 3, "unverified_passes": 0, "verification_failures": 26, "known_cost_usd": 2.3334561216, "cost_per_verified_success_usd": 0.7778187071999999, "available": true` |
| V9 | The eval harness parses ONLY the five verify counters from ai-check JSON — it has no `Results` field, so per-function status strings never reach harness logic | `Read internal/eval_harness/verify.go:35-42` | `AICheckVerifyResult` = `{Available bool; Verified, Counterexample, Skipped, Errors int}` — no `Results`. |
| V10 | ai-check's exit code ignores skips by contract ("skipped-only → 0") | `Read cmd/ailang/ai_check.go` (aiCheckExitCode + doc comment) | `func aiCheckExitCode(...) int { if !check.Passed \|\| verify.Counterexample > 0 \|\| verify.Errors > 0 { return 1 }; return 0 }` with the contract comment `skipped-only -> 0 (a skip is "not proved", not "disproved")`. Unit-tested in `cmd/ailang/ai_check_exit_test.go` (grep rc=0). |
| V11 | `ailang verify --strict` exits 1 when `skipped > 0 \|\| errCount > 0` — so today a no-ensures function DOES trip `--strict` | `sed -n '440,460p' cmd/ailang/verify.go` | `if *strictFlag && (skipped > 0 \|\| errCount > 0) { os.Exit(1) }` at `verify.go:457`. |
| V12 | The UI does NOT read `verify_skipped` or the KPI fields (no UI change needed) | `grep -rl "verify_skipped" ui/src` → rc=1; `grep -rn "cost_per_verified" ui/src` → rc=1; `grep -rn "costPerVerifiedSuccess" ui/src` → rc=1; control in same call: `grep -rln "React" ui/src` → fires (`ui/src/App.tsx`, `ui/src/main.tsx`, …), rc=0 | Three controlled negatives. (The "React card" note in `cost_per_verified_success.go`'s header refers to the latest.json headline object consumed elsewhere, not a `ui/src` reader of these fields.) |
| V13 | The other `"skipped"` string hits in Go are different domains, not this status | `grep -rn '"skipped"' --include='*.go' internal/ cmd/ \| grep -v _test.go \| grep -v 'ai_check.go\|verify.go'` + read of each | `internal/test/reporter.go` (test-runner statuses), `cmd/ailang/eval_suite_finalize.go:149` (job-level "already banked" skip flag in a notification payload — read in context, unrelated to verify), `cmd/registry-validator/validate.go:133` (registry domain), `cmd/ailang/verify_print.go:46,:110` (the verify human printer + JSON summary — these two ARE in scope, see design), `cmd/ailang/verify_filter.go:28` (in scope, V4). |
| V14 | No existing doc covers this; duplicate gate clears | `ailang docs search --neural "verification skipped not applicable ensures coverage"` | Top match 0.43 (`m-coverage-cross-package-attribution.md`, a `go test -coverpkg` question — unrelated domain). All matches < 0.45 ⇒ proceed per the skill's gate. `ls design_docs/planned/m-contract-verification-coverage.md` → No such file. |
| V15 | `meta.Contracts` element type for the shared helper | `sed -n '430,460p' internal/core/core.go` | `DeclMeta.Contracts []*Contract`; `EnsuresKind` is a `ContractKind` const at `core.go:445`. |
| V16 | Named tests that anchor the mutation table exist at HEAD | `grep -n "^func Test" internal/observatory/cost_per_verified_success_test.go internal/eval_harness/agent_verify_test.go internal/eval_harness/verify_nonzero_test.go` | 12 tests incl. `TestCostPerVerifiedSuccess_Table`, `TestEvalAssessment_VerifyFieldsRoundTrip`, `TestEvalAssessment_HistoricalRowIsVerificationMissing`, `TestApplyAgentVerification_PopulatesVerifiedSuccess`, `TestRunAICheckParsesJSONFromNonzeroChild`. Also `cmd/ailang/ai_check_exit_test.go`, `cmd/ailang/chains_stats_cvs_test.go` exist (`ls`). |
| V17 | The three banking sites carry the 5 verify_* fields as parallel literal blocks | `sed -n '290,310p' cmd/ailang/eval_benchmark_agent.go; sed -n '395,412p' cmd/ailang/eval_benchmark_agent.go; sed -n '272,290p' cmd/ailang/eval_benchmark.go` | All three constructor literals assign `VerifyOk/VerifyVerified/VerifyCounterex/VerifySkipped/VerifyErrors` (agent result-file block additionally `VerifyJSON`). |

**Language-claim gate (`ailang check`)**: this doc makes **no** "AILANG does/doesn't
support X" claims — it changes Go tooling around the verifier, not the language.
N/A, stated explicitly rather than skipped silently.

---

## Solution Design

### Overview

Introduce a new per-function status value **`not_applicable`** and a parallel counter
at every layer that today carries `skipped`, classified by ONE shared helper (the
no-ensures test currently duplicated across the two CLIs), with the two observatory
predicates rewritten in terms of the SUM `VerifySkipped + VerifyNotApplicable` —
arithmetically identical to today's `VerifySkipped` for both historical rows
(missing key unmarshals to 0) and new rows (the old 53 becomes 29 + 24).

**Landing order is reader-before-writer, and this is load-bearing.** The ENTIRE read
path — harness parser, metrics/result structs, all three banking sites,
`EvalAssessment`, and BOTH observatory predicates (as sums) — lands and deploys
FIRST, while the emitters still report every no-ensures case as `skipped`. Only the
final milestone changes what the emitters write. The round-1 version of this doc
ordered it the other way (emitters first) and was rejected by a 2/2 quorum: in the
emitter-first window, `ai-check` stops counting no-ensures cases in `skipped` while
the old harness parser (`AICheckVerifyResult`, five fields, no `not_applicable` —
V9) silently DROPS the new key, losing those outcomes from every row banked in the
window; and rows banked after the struct split but before the predicate rewrite
would satisfy `VerifySkipped == 0` on no-ensures-only runs — the prohibited D-29
flip, shipped by accident. Both failure modes are impossible in reader-first order.

**The ordering invariant** (stated once, defended at every commit): at EVERY commit
boundary, `VerifySkipped + VerifyNotApplicable` equals the pre-change conflated
`VerifySkipped`. While the emitters are unchanged (all of M1), `VerifyNotApplicable`
is 0 on every row — historical, new, and in-flight — so the sum is trivially equal
and every intermediate state is behaviour-identical **by construction**, not by
argument. After the emitter split (M2), a row that previously carried `skipped=53`
carries `skipped=29, not_applicable=24` and the sum — hence every predicate verdict,
exit code, and KPI value — is unchanged.

### Classification decision table

Only ONE of the four skip-emission branches is class (2). This is the whole
classification, with zero residue (V4):

| Emission branch (both CLIs) | Reason text | New status |
|---|---|---|
| Function body not found in Core AST | `function body not found in Core AST` | `skipped` (unchanged — a verifier limitation, not the spec's design) |
| **No `ensures` clause** | `no ensures clause (nothing to verify)` | **`not_applicable`** (NEW) |
| Not SMT-encodable (rejection reasons) | joined `smt.SMTRejectionReason` messages | `skipped` (unchanged) |
| Unresolvable SMT sorts (`unresolvedTypeVerifyResult`, shared) | `SMT declaration closure could not resolve a required sort (…)` | `skipped` (unchanged) |

The reason string `"no ensures clause (nothing to verify)"` is unchanged, so any
log-greps or transcripts keyed on it keep working.

### One systemic fix, not two patches (CLAUDE.md Principle 3)

The `hasEnsures` loop + no-ensures result construction is duplicated verbatim in
`cmd/ailang/ai_check.go:303-315` and `cmd/ailang/verify.go:290-304` (V3). Extract
both into `cmd/ailang/verify_filter.go`, which is already the home of the shared
`unresolvedTypeVerifyResult` helper (V4) — the established precedent for
verify-result helpers shared by the two CLIs:

```go
// hasEnsuresClause reports whether any contract clause is an ensures.
func hasEnsuresClause(contracts []*core.Contract) bool

// notApplicableVerifyResult is THE single constructor for the
// "nothing to verify" outcome. Status "not_applicable" is emitted
// nowhere else (grep-enforceable).
func notApplicableVerifyResult(funcName string) verifyResult
```

Both CLIs call these; neither retains a private copy. A later grep for
`"not_applicable"` in `cmd/ailang` must hit exactly one constructor plus the
printers/counters.

### Layer-by-layer changes (presented in LANDING order — reader layers first)

**Layer 1 (M1) — eval harness (`internal/eval_harness`)**

- `verify.go`: `AICheckVerifyResult` gains `NotApplicable int \`json:"not_applicable"\``.
  In M1 no emitter writes that key yet, so it unmarshals to 0 everywhere — the field
  is a dormant reader, which is exactly the point;
  `applyAgentVerification` adds `out.VerifyNotApplicable = result.Verify.NotApplicable`;
  `PopulateVerifyMetrics` likewise. `VerifyOk` computation is untouched (it never
  involved skips).
- `agent_runner.go` (`AgentBenchmarkResult`) and `metrics.go` (`RunMetrics`): new field
  `VerifyNotApplicable int \`json:"verify_not_applicable"\`` beside `VerifySkipped`.

**Layer 2 (M1) — banking sites (`cmd/ailang`), now via TESTABLE constructors**

All three literal blocks from V17 gain the one assignment
`VerifyNotApplicable: …` beside `VerifySkipped`:
`eval_benchmark_agent.go:302` (banked result-file block), `:406` (chains
`EvalAssessment` block), `eval_benchmark.go:282` (standard-mode block).

Round 1 recorded these three hunks as having NO killing test ("only executed by a
real eval run") and called extraction "optional hardening". The quorum rejected
that: an untested lossy assignment beside a lossy-by-omission field is precisely
where silent data loss hides. Extraction is now **required**. Each of the three
verify-field blocks moves into a named, package-level, unit-callable constructor in
the same file (exact signatures are the executor's choice; the binding requirements
are below):

```go
// eval_benchmark_agent.go — the banked result-file verify block
func bankedAgentResultVerifyFields(result *eval_harness.AgentBenchmarkResult) ...
// eval_benchmark_agent.go — the chains EvalAssessment block (agent mode)
func agentEvalAssessment(result *eval_harness.AgentBenchmarkResult, ...) observatory.EvalAssessment
// eval_benchmark.go — the chains EvalAssessment block (standard mode)
func standardEvalAssessment(metrics *eval_harness.RunMetrics, ...) observatory.EvalAssessment
```

Binding requirements, each enforceable without judgment: (a) every constructor is a
pure function callable from a unit test; (b) after extraction,
`grep -n "VerifySkipped\|VerifyNotApplicable" cmd/ailang/eval_benchmark_agent.go
cmd/ailang/eval_benchmark.go` hits ONLY inside the three constructors and their
tests — no verify-field assignment survives at a call site; (c) the new end-to-end
banking test (AC-8, mutations M-13..M-15) feeds `skipped=2, not_applicable=3`
through EACH constructor and round-trips the resulting `EvalAssessment` through
JSON, so omitting any one of the three assignments fails before merge.

**Layer 3 (M1) — observatory (`internal/observatory`)**

- `models_chains.go`: `EvalAssessment` gains
  `VerifyNotApplicable int \`json:"verify_not_applicable"\`` beside `VerifySkipped:165`.
  JSON-persisted only — **no SQL migration** (V7), and additive keys are safe on both
  the SQLite and Firestore backends.
- `cost_per_verified_success.go` — the ONLY semantic edits, both
  behavior-preserving, and both landing in M1 while every row still has
  `VerifyNotApplicable == 0` (so the sum degenerates to today's term):
  - `isVerifiedSuccess`: `a.VerifySkipped == 0` → `a.VerifySkipped+a.VerifyNotApplicable == 0`
  - `isVerificationFailure`: `a.VerifySkipped > 0` → `a.VerifySkipped+a.VerifyNotApplicable > 0`
- `CostPerVerifiedSuccessResult` (the wire contract shared by CLI/HTTP/latest.json) is
  **untouched** — no renamed or added fields, so no consumer updates.

**Layer 4 (M2, FINAL — the only writer change) — the two verifier CLIs (`cmd/ailang`)**

- `verify_filter.go`: the two shared helpers above (+ update the `verifyResult.Status`
  enum comment at `verify.go:465` to `verified, counterexample, skipped,
  not_applicable, error, unknown`).
- `ai_check.go`: `aiVerifySection` gains `NotApplicable int \`json:"not_applicable"\``;
  the no-ensures branch emits the helper's result and increments
  `section.NotApplicable` instead of `section.Skipped`.
  **`aiCheckExitCode` is deliberately NOT modified** — skips never affect exit status
  (V10) and `not_applicable` inherits that: it does not even appear in the function.
- `verify.go`: a `notApplicable` counter alongside `skipped`; the strict gate becomes
  `skipped > 0 || notApplicable > 0 || errCount > 0` — **behaviourally identical** to
  today, where the no-ensures case is inside `skipped` and trips `--strict` (V11).
  Extract the strict decision into a pure `verifyStrictFails(skipped, notApplicable,
  errCount int) bool` so the gate is unit-testable (precedent: `aiCheckExitCode` was
  extracted for exactly this reason).
- `verify_print.go`: human printer gains a `case "not_applicable"` (e.g.
  `– N/A <func> (no ensures clause)`) distinct from `⚠ SKIPPED`; `printVerifyJSON`'s
  summary struct gains `NotApplicable int \`json:"not_applicable"\``.

### Why the sum is exactly backward-compatible (the tested argument)

Historical banked rows — including the frozen `v1.0` cohort whose KPI is published —
were written before the split. They carry the conflated `verify_skipped` and no
`verify_not_applicable` key, which Go unmarshals to 0. Therefore
`VerifySkipped + VerifyNotApplicable == VerifySkipped(old)` for every historical row,
and the published KPI **cannot move**. For newly-banked rows, a run that previously
banked `skipped=53` now banks `skipped=29, not_applicable=24`, and the sum — hence the
predicate verdict — is identical. Both arms are **tested acceptance criteria** (AC-4,
AC-5 below), not remarks.

### Files to Modify (new files: tests + AC-6 fixture only)

M1 (reader path):
- `internal/eval_harness/verify.go` — `AICheckVerifyResult` field; two populate lines (+3 LOC)
- `internal/eval_harness/agent_runner.go` — `VerifyNotApplicable` field (+1 LOC)
- `internal/eval_harness/metrics.go` — `VerifyNotApplicable` field (+1 LOC)
- `cmd/ailang/eval_benchmark_agent.go` — extract two banking constructors + one
  `VerifyNotApplicable` assignment in each (~±35 LOC, extraction-dominated)
- `cmd/ailang/eval_benchmark.go` — extract one banking constructor + assignment (~±20 LOC)
- `internal/observatory/models_chains.go` — `VerifyNotApplicable` field (+1 LOC)
- `internal/observatory/cost_per_verified_success.go` — two predicate sums (±2 LOC)

M2 (writer path):
- `cmd/ailang/verify_filter.go` — shared `hasEnsuresClause` + `notApplicableVerifyResult` (~+25 LOC)
- `cmd/ailang/ai_check.go` — use helper; `NotApplicable` counter in `aiVerifySection` (~±15 LOC)
- `cmd/ailang/verify.go` — use helper; counter; strict-gate extraction (~±25 LOC)
- `cmd/ailang/verify_print.go` — human case + JSON summary field (~+10 LOC)

Both: `changelogs/v0.18-current.md` — changelog entry (required by repo rules)
- Tests: `cmd/ailang/` (M1: new banking-constructor round-trip test file;
  M2: new no-ensures classification + strict-gate tests + testdata fixture),
  `internal/eval_harness/agent_verify_test.go` (+split test),
  `internal/observatory/cost_per_verified_success_test.go` (+NA predicate + compat tests)

---

## Conflict Surface

This change touches no parser/typechecker/codegen path, but it DOES touch the eval
observability spine. What else lives on these files, and what could collide:

1. **`D-29` (open decision).** The one-line flip must NOT land here. The mutation
   table's O-1 test exists precisely to go red if it does.
2. **Concurrent planned docs on the eval surface**: `m-eval-failure-attribution.md`,
   `m-eval-validity-discipline.md`, `m-eval-w8-harness-errors-as-capability-failures.md`
   all orbit `internal/eval_harness` / observatory semantics. None (per their titles
   and the V14 search) touches the verify counters, but a sprint landing alongside
   them should rebase early (worktree stale-base churn is a known failure mode).
3. **External JSON consumers of `ailang ai-check`** (motoko's done-gate, agent
   convergence loops): the exit-code contract is unchanged (V10), the five existing
   counter keys are unchanged, and the new `not_applicable` key is additive. The
   in-repo harness parses only the counters and ignores `results[]` (V9). The
   per-function `status` string for no-ensures functions changes
   `"skipped"` → `"not_applicable"` in prompt-visible feedback text — semantically
   clearer, but it is a (minor) **measurement boundary** for token-level A/Bs; the
   sprint's mission-log entry must record the **M2** landing date, per the standing
   local-model-boundary practice.
   **Residual binary-skew case (the one hazard commit order cannot remove):** the
   harness resolves the ai-check child from PATH (`RunAICheck` defaults
   `ailangPath` to `"ailang"`, `internal/eval_harness/verify.go:48-53`), so a
   harness binary built pre-M1 driving a PATH `ailang` built post-M2 would still
   drop the `not_applicable` count — same lossy shape the quorum rejected, moved
   from the commit timeline to the deployment timeline. Within any single build
   the reader is always ≥ the writer (commit order guarantees it); across builds,
   mitigation is the repo's standing rule (`make quick-install` after building,
   dev-workflow.md) plus M2's post-merge spot-check of a nonzero
   `verify_not_applicable` on `contract_rle_roundtrip`. This is the same additive-
   JSON skew class every prior verify-field change (M4a-3) already carried; it is
   recorded, not new.
4. **`ailang verify --strict` users**: behavior preserved (no-ensures still fails
   strict, via the sum). Anyone who wants strict to tolerate `not_applicable` later
   has a one-line lever — that is deliberately NOT changed here for the same reason
   as D-29: preserve, then rule.
5. **Firestore-backed prod observatory**: `EvalAssessment` is JSON-persisted;
   additive key, no migration hook needed on either backend (V7). The KPI surface
   itself is local-only (`refuseRemoteReadForLocalOnlySurface`).
6. **`internal/test/reporter.go`, `eval_suite_finalize.go:149`,
   `registry-validator`**: carry their own unrelated `"skipped"` strings (V13) — must
   NOT be touched; listed so the executor doesn't "helpfully" rename them.
7. **Historical banked data (19,027 result files per iteration 254's log; 30-run
   frozen cohort)**: never rewritten. Compatibility is achieved entirely in the
   reader (missing key → 0), asserted by AC-4.

Programs/flows that MUST still work unchanged:
`ailang ai-check <file>` exit codes on: check-fail (1), counterexample (1),
verifier-error (1), skipped-only (0), verified-only (0) — the existing
`ai_check_exit_test.go` matrix; `ailang chains stats --cost-per-verified-success
--baseline v1.0 --json --strict` availability and value; `ailang verify --strict`
failing on a no-ensures-only module.

---

## Acceptance Criteria

Each criterion can fail. `go build ./...` is deliberately NOT used anywhere — it is
rc=1 on pristine dev (`cmd/wasm`, `gen/main` have no native main; controller-measured
2026-08-23). Scoped forms below were all measured rc=0 at base today.

**AC-1 — scoped build/vet/format gates** (each must exit 0; each CAN fail on a bad
edit to exactly the packages this sprint touches):

```bash
go build ./internal/observatory/... && go build ./internal/eval_harness/... && go build ./cmd/ailang
go vet ./internal/observatory/... ./internal/eval_harness/...
test -z "$(gofmt -l internal/observatory internal/eval_harness cmd/ailang)"
make check-changelog && make check-file-sizes && make check-boundaries
```

**AC-2 — unit tests with an ENUMERATION FLOOR.** `go test -run` exits 0 when the
selector matches nothing (measured at HEAD: `-run 'TestZzNoSuchTest'` → rc=0,
`[no tests to run]`, identical exit code to a real run). Every `-run` criterion
therefore asserts the count of top-level `=== RUN   Test` lines equals the selector
count, and a shortfall FAILS LOUDLY as an instrument failure:

```bash
go test ./internal/observatory/ -v -run \
  '^(TestCostPerVerifiedSuccess_Table|TestCostPerVerifiedSuccess_NotApplicableStillDisqualifies|TestCostPerVerifiedSuccess_NotApplicableIsVerificationFailure|TestEvalAssessment_VerifyFieldsRoundTrip|TestEvalAssessment_HistoricalRowIsVerificationMissing)$' \
  2>&1 | tee /tmp/ac2a.txt
n=$(grep -Ec '^=== RUN   Test[A-Za-z0-9_]+$' /tmp/ac2a.txt)
[ "$n" -eq 5 ] || { echo "ENUMERATION FLOOR FAILED: matched $n of 5 tests"; exit 1; }
grep -q '^ok' /tmp/ac2a.txt
```

Same pattern for `internal/eval_harness` (floor = the named
`TestApplyAgentVerification_*` + new split tests) and `cmd/ailang` (floor = the new
banking round-trip test (AC-8) plus, in M2, the classification/strict/exit tests).
A criterion whose floor reads 0 is an instrument failure, never a pass.

**AC-3 — new-row classification equivalence.** A unit test constructs two
`EvalAssessment` values with identical pass/verify evidence except:
(a) old-style `VerifySkipped=53, VerifyNotApplicable=0`;
(b) new-style `VerifySkipped=29, VerifyNotApplicable=24`.
Asserts BOTH are `!isVerifiedSuccess` and BOTH are `isVerificationFailure`, i.e. the
split cannot change any verdict. (This is the "both arms" unit test the
backward-compatibility argument requires.)

**AC-4 — historical-row compatibility, tested.** A unit test unmarshals a literal
JSON `EvalAssessment` payload WITHOUT a `verify_not_applicable` key (as every
pre-split banked row is) and asserts `VerifyNotApplicable == 0` and that the
predicate verdicts equal the pre-change truth table (extends
`TestEvalAssessment_HistoricalRowIsVerificationMissing` /
`TestEvalAssessment_VerifyFieldsRoundTrip`).

**AC-5 — the published KPI is byte-identical, measured live.** Using the
freshly-built binary (NOT the possibly-stale PATH binary — run via the repo to dodge
the known stale-PATH trap):

```bash
go run ./cmd/ailang chains stats --cost-per-verified-success --baseline v1.0 --json \
  | tee /tmp/ac5.json
python3 - <<'EOF'
import json; r = json.load(open('/tmp/ac5.json'))
assert r["cost_per_verified_success_usd"] == 0.7778187071999999, r
assert r["verified_successes"] == 3 and r["verification_failures"] == 26 and r["total_runs"] == 30, r
print("KPI byte-identical: OK")
EOF
```

Baseline observed at HEAD today (V8); any drift fails the assert. **Run at BOTH
milestone boundaries** — at the end of M1 (predicates already rewritten, emitters
unchanged: proves the sum degenerates correctly over the real 30-run cohort) and
again at the end of M2 (emitters split: proves historical rows still read
identically). Each run is a commit-gate, not a one-time check.

**AC-6 — ai-check end-to-end split.** Against a fixture module (added under
`cmd/ailang/testdata/`) containing one function with `requires`-only and one with
`ensures`, `ailang ai-check --json <fixture>` must report
`verify.not_applicable == 1`, the requires-only function's result
`status == "not_applicable"`, and process exit 0. Assert all three with `jq` + `$?`;
exit-code-only or key-presence-only checks are insufficient (a vacuous pass).

**AC-7 — no stray emitters.** `grep -rn '"not_applicable"' cmd/ailang --include='*.go' | grep -v _test.go`
hits exactly: the one constructor in `verify_filter.go`, the two JSON summary struct
tags, and (if written as a literal) the printer case — an enumerated allowlist in the
sprint plan. Control: the same grep for `"skipped"` still fires at its known sites.

**AC-8 — end-to-end banking round-trip (the quorum-required test; lands in M1).**
A new test file in `cmd/ailang` feeds a harness result carrying
`VerifySkipped=2, VerifyNotApplicable=3` through EACH of the three extracted banking
constructors (Layer 2), marshals each produced row to JSON, unmarshals it back into
`observatory.EvalAssessment` (or the banked result-file struct for the first site),
and asserts BOTH values survive as exactly 2 and 3 — per constructor, not just once.
With an enumeration floor per AC-2's pattern (3 named subtests or top-level tests;
floor reads < 3 ⇒ instrument failure, not a pass). Companion structural check:

```bash
grep -n "VerifySkipped\|VerifyNotApplicable" \
  cmd/ailang/eval_benchmark_agent.go cmd/ailang/eval_benchmark.go
```

must hit only inside the three constructors (enumerated allowlist in the sprint
plan) — no verify-field assignment may survive at a call site, so the tested
constructors are the ONLY way verify counters reach a banked row. Control: the grep
currently fires at the three literal sites (V17), proving the instrument sees
positives on these exact files.

---

## Mutation Table

Each refusal/branch the diff introduces, one mutation that must turn a NAMED test
red. Mutations prefer neutering (`if false && cond`) or value-substitution so the
mutant still compiles. Anchored to the DIFF, not just the defect.

| # | Mutation (site) | Named test that goes red |
|---|---|---|
| M-1 | In the shared helper, return `Status: "skipped"` instead of `"not_applicable"` (`verify_filter.go`) | NEW `TestAICheckNoEnsuresClassifiedNotApplicable` (cmd/ailang; drives the section builder over a no-ensures decl and asserts status + counters) |
| M-2 | At the ai-check no-ensures branch, increment `section.Skipped++` instead of `section.NotApplicable++` (`ai_check.go`) | same test — asserts `Skipped==0 && NotApplicable==1` |
| M-3 | Neuter the strict-gate NA term: `notApplicable > 0` → `false && notApplicable > 0` (`verify.go` extracted `verifyStrictFails`) | NEW `TestVerifyStrictFailsOnNotApplicable` (unit on the extracted pure function) |
| M-4 | Drop `NotApplicable` from `printVerifyJSON`'s summary struct (`verify_print.go`) | NEW `TestPrintVerifyJSONCarriesNotApplicable` (marshals the summary, asserts the `"not_applicable"` key) |
| M-5 | Neuter `out.VerifyNotApplicable = result.Verify.NotApplicable` in `applyAgentVerification` (assign 0) | NEW `TestApplyAgentVerification_SplitsNotApplicable` (fake verifier returns Skipped=2, NotApplicable=3; asserts both banked fields) |
| M-6 | Same neutering in `PopulateVerifyMetrics` | NEW `TestPopulateVerifyMetrics_SplitsNotApplicable` |
| M-7 | Remove/typo the `json:"verify_not_applicable"` tag on `EvalAssessment` (`models_chains.go`) | extended `TestEvalAssessment_VerifyFieldsRoundTrip` (round-trips a nonzero `VerifyNotApplicable`) |
| M-8 | Revert `isVerifiedSuccess` to `a.VerifySkipped == 0` alone — **this is the D-29 flip landing accidentally** | NEW `TestCostPerVerifiedSuccess_NotApplicableStillDisqualifies` (row with `VerifySkipped=0, VerifyNotApplicable=24`, otherwise verified — must NOT count as verified success) |
| M-9 | Neuter the NA term in `isVerificationFailure` | NEW `TestCostPerVerifiedSuccess_NotApplicableIsVerificationFailure` (same row must classify verification_failure, not unverified pass) |
| M-10 | Drop the `VerifySkipped` term from the new sum in `isVerifiedSuccess` (keep only NA) | existing `TestCostPerVerifiedSuccess_Table` (its skipped>0 rows would start passing) |
| M-11 | Add `verify.NotApplicable > 0` into `aiCheckExitCode` (the inverse mutation — a change where none is allowed) | existing `cmd/ailang/ai_check_exit_test.go` matrix extended with a NotApplicable-only case expecting 0 — red if exit starts failing |
| M-12 | Neuter `AICheckVerifyResult.NotApplicable` parsing (rename the json tag in `internal/eval_harness/verify.go`) | extended `TestRunAICheckParsesJSONFromNonzeroChild` (child emits `"not_applicable": 3`; asserts parsed value) |
| M-13 | Neuter the `VerifyNotApplicable` assignment in the extracted banked-result-file constructor (`eval_benchmark_agent.go`, ex-`:302` literal) — assign 0 | NEW `TestBankingConstructors_SplitRoundTrip` (AC-8; the result-file subtest asserts `Skipped==2 && NotApplicable==3` after JSON round-trip) |
| M-14 | Same neutering in the extracted agent `EvalAssessment` constructor (`eval_benchmark_agent.go`, ex-`:406` literal) | same test — the agent-assessment subtest |
| M-15 | Same neutering in the extracted standard-mode `EvalAssessment` constructor (`eval_benchmark.go`, ex-`:282` literal) | same test — the standard-mode subtest |

**Hunks with NO killer (recorded, not hidden — genuinely remaining after AC-8):**
Round 1 listed the three banking-site assignments here; AC-8's constructor
extraction + round-trip test (M-13..M-15) now kills all three, so they are OFF this
list. What genuinely remains unkilled is one step further out: the **call-site
wiring** — that the live eval paths actually INVOKE the tested constructors with the
real harness result. A mutant that deletes a constructor call (banking no verify
fields at all) is not caught by any unit test; it is caught structurally by AC-8's
grep allowlist only if the mutant re-introduces field assignments, and behaviourally
by the M2 post-merge spot-check (first banked run must show nonzero
`verify_not_applicable` on `contract_rle_roundtrip`, which also shows nonzero
`verify_skipped`/`verify_ok` wiring generally). This residual is strictly smaller
than round 1's (a dropped call zeroes ALL verify evidence — loud in any
`chains stats` read — whereas round 1's unkilled mutants zeroed only the new field —
silent under the sum). AC-6 additionally covers the ai-check→counters path
end-to-end. No claim is made that AC-8 covers the wiring; it covers the
constructors.

---

## Milestones

**Reader-before-writer.** Round 1 ordered these emitter-first and a 2/2 quorum
rejected it: in that order the intermediate states were lossy (dropped
`not_applicable` counts) and then D-29-flipping (predicates reading a shrunken
`VerifySkipped`). The order below is the reviewers' fix, applied verbatim. Each
milestone is independently committable and green under AC-1's gates on its own —
and each safety claim below is by construction under the ordering invariant
(Overview), not by argument. The alternative both reviewers allowed — landing all
layers atomically in one commit — remains acceptable to the sprint-executor if M1
turns out awkward to gate alone; what is PROHIBITED is any state where an emitter
writes `not_applicable` before every reader sums it.

**M1 — the ENTIRE read path: harness + banking + observatory, predicates as sums**
(~0.75 day)
`AICheckVerifyResult.NotApplicable`, `RunMetrics` / `AgentBenchmarkResult` fields,
both populate funcs, the three banking constructors extracted and split-aware,
`EvalAssessment` field, and BOTH predicate rewrites to
`VerifySkipped + VerifyNotApplicable` in `cost_per_verified_success.go`.
Tests: M-5..M-10, M-12, M-13..M-15 (AC-8), AC-3, AC-4; AC-5 live KPI byte-identity
run at this boundary.
**Why this commit is safe alone, by construction:** no emitter is touched, so no
process anywhere writes a nonzero `not_applicable` — the harness parser reads a key
that is never present (→ 0), every banked row old and new carries
`VerifyNotApplicable == 0`, and both predicates compute `VerifySkipped + 0`, which
is bit-identical to the pre-change expression on every row that exists. The
dangerous direction (writer without reader) is impossible because the writer does
not exist yet.

**M2 — emitter split in the two CLIs, FINAL** (~0.5 day)
Shared helpers in `verify_filter.go`; `ai_check.go` + `verify.go` classification and
counters; printers; strict-gate extraction + preservation; tests (M-1..M-4, M-11's
extension); AC-6 fixture; AC-5 re-run at this boundary; CHANGELOG entry in
`changelogs/v0.18-current.md`; mission-log measurement-boundary note recording the
M2 landing date (Conflict Surface item 3).
**Why this commit is safe, by construction:** by the time any emitter writes
`not_applicable`, every reader in the repo already sums both fields (M1 is its
declared dependency — the sprint plan must sequence it strictly after M1 merges).
A post-M2 run banks `skipped=29, not_applicable=24` where it banked `skipped=53`;
the sum, every predicate verdict, `aiCheckExitCode`, and the `--strict` gate are
unchanged (AC-3, V10, V11). Historical rows are untouched and read identically
(AC-4, AC-5). The residual cross-BUILD skew case (old harness binary + new PATH
`ailang`) is a deployment property, not a commit-order property — recorded with its
mitigation in Conflict Surface item 3.

Buffer: ~0.5 day. Total ≤ 2 days (unchanged — the work moved between milestones;
the extraction added ~0.25 day to M1 and removed the same "optional hardening"
line-item from the backlog).

---

## Testing Strategy

- Unit: all mutation-table tests above (pure functions and struct round-trips; no Z3
  and no subprocess except the existing fake-child pattern of
  `verify_nonzero_test.go`), including the quorum-required
  `TestBankingConstructors_SplitRoundTrip` (AC-8): `skipped=2, not_applicable=3`
  through each of the three extracted banking constructors, JSON round-tripped.
- Integration: AC-6 (real `ai-check` over a testdata fixture, exercising the parser →
  classifier → counter → JSON path with the real Z3 gate for the `ensures` sibling).
- System: AC-5 (live KPI over the real 318MB observatory and the real frozen cohort —
  the strongest available regression check, because it exercises unmarshal-of-
  historical-rows + predicate + cost rollup together against a known-exact value).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime change |
| A2: Replayability | +1 | Banked rows now record WHY verification didn't run, replayably |
| A3: Effect Legibility | 0 | No effects change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | The verifier's own coverage becomes a legible, unconflated number (29, not "part of 53") |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | `not_applicable` is machine-distinguishable in JSON where prose reasons had to be string-matched |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Protects the headline cost KPI's denominator from a misattributed penalty class (without moving it) |
| A10: Composability | 0 | Additive JSON keys; no contract breaks |
| A11: Structured Failure | +1 | "Nothing to verify" is no longer reported as a failure-shaped status |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** ✅ — no hard violations (A1/A3/A4/A7 all ≥ 0).

## Related Documents

- `design_docs/v1-mission-log.md` — iteration 254 entry (the diagnosis; 53 = 24 + 29 partition; D-29 filing)
- `design_docs/v1-mission-dashboard.md` — `D-29` (the parked predicate flip; this doc's Out-of-Scope section). Verified: `grep -rln "D-29" design_docs/decisions/ design_docs/v1-mission-dashboard.md` hits ONLY the dashboard — `D-29` is NOT in `decisions/`.
- `internal/observatory/cost_per_verified_success.go` header — M-COST-PER-SUCCESS-KPI (the KPI's single-definition contract this doc preserves)
- `design_docs/planned/m-eval-validity-discipline.md` (0.32 neural) — adjacent eval-comparison hygiene; no counter overlap
- `design_docs/planned/v0_33_2/m-coverage-cross-package-attribution.md` (0.43 neural, top match) — unrelated (`go test -coverpkg`); named per the duplicate-gate warn band

## References

- Verification log above (all first-party at `30176187f`, 2026-08-23)
- CLAUDE.md Principle 2 (no silent conflation of integrity-bearing data) and
  Principle 3 (one systemic fix: the shared classifier helper)

---

## Quorum verification log

**Round 1 — verdict: `blocked`.** Reviewers present: 2 of 2 (`gpt5-6-sol`,
`gemini-3-1-pro`; no absent reviewers — a full-strength N=2 reject, not a degraded
one). Both reviewers independently found the SAME defect. Neither disputed the
design direction (the skipped/not_applicable split itself was accepted); what was
rejected is the milestone ORDER. The controller verified the objections'
load-bearing premise first-party at `30176187f` before this revision:
`internal/eval_harness/verify.go:36-42` — `AICheckVerifyResult` has exactly five
fields (`Available`, `Verified`, `Counterexample`, `Skipped`, `Errors`), no
`not_applicable`, no `results[]` — so the round-1 window loss was real, not
hypothetical.

### Objection 1 — gpt5-6-sol (verbatim)

> "The milestone sequence is not behavior-preserving: after M1, `ai-check` stops
> incrementing `skipped` for no-ensures results while the old harness ignores
> `not_applicable`, so any run banked before M2 silently loses those outcomes.
> After M2 but before M3, new rows store the split while observatory predicates
> still inspect only `VerifySkipped`, effectively landing the prohibited D-29
> behavior for those rows. This contradicts the claims that each milestone is
> independently safe and that KPI semantics remain identical, and violates the
> no-silent-fallback/data-integrity axioms."

Its catch (verbatim): "The document explicitly says M1 is safe because the old
harness ignores the additive key and M2 is safe before the predicate rewrite.
Ignoring the key is precisely a lossy fallback once the emitter removes those cases
from `skipped`; the three untested banking assignments make that loss harder to
detect."

Its proposed_fix (verbatim): "Replace the milestone plan with a dependency-safe
sequence: (1) add `VerifyNotApplicable` plumbing through parser, metrics, banking,
persistence, and observatory, and change both predicates to use
`VerifySkipped + VerifyNotApplicable`, while emitters still report all no-ensures
cases as `skipped`; (2) only after that entire path is deployed, switch both
emitters to `not_applicable` and add printer/strict handling. Alternatively, land
all layers atomically. Add an end-to-end banking test that feeds
`skipped=2, not_applicable=3` through each banking constructor and round-trips the
resulting `EvalAssessment`, so omission of any of the three currently unkilled
assignments fails before merge."

### Objection 2 — gemini-3-1-pro (verbatim)

> "The proposed milestone order (M1 emitter split -> M2 harness -> M3 predicate
> preservation) temporarily ships the explicitly banned D-29 flip for any new evals
> run during the M1-M2 window. In M1, ai-check stops incrementing 'skipped' for
> no-ensures clauses, lowering the emitted skipped count. Because the observatory
> predicate is not updated to check the sum of skipped and not_applicable until M3,
> old observatory logic will erroneously evaluate 'VerifySkipped == 0' as true for
> runs containing only no-ensures skips, falsely classifying them as verified
> successes."

Its proposed_fix (verbatim): "Reverse the milestone order to build the reader
before the writer. Shift the struct field additions (M2) and predicate sum updates
(M3) into the initial milestone. Once the observatory is safely summing both
fields, ship the emitter split (M1) as the final milestone."

### Fixes applied in this revision (r2)

1. **Milestone order reversed to reader-before-writer** — both reviewers' shared
   fix, applied as written: new M1 = the entire read path (parser, metrics, banking,
   persistence, observatory, BOTH predicates as sums) with emitters untouched; new
   M2 = the emitter split, last, with printer/strict handling. The atomic-landing
   alternative gpt5-6-sol allowed is recorded in the Milestones preamble as
   acceptable; the prohibited state (writer before every reader sums) is named.
2. **The ordering invariant is stated and defended by construction** (Overview +
   per-milestone safety arguments): at every commit boundary
   `VerifySkipped + VerifyNotApplicable` equals the pre-change conflated
   `VerifySkipped`; during all of M1 the new field is 0 everywhere so equality is
   trivial.
3. **gpt5-6-sol's end-to-end banking test added as AC-8** (mutations M-13..M-15,
   `TestBankingConstructors_SplitRoundTrip`): `skipped=2, not_applicable=3` through
   EACH banking constructor, `EvalAssessment` JSON round-trip, enumeration floor
   of 3. This required promoting the round-1 "optional hardening" (constructor
   extraction at `eval_benchmark_agent.go:302,:406` / `eval_benchmark.go:282`) to a
   required M1 change; the mutation table's "Hunks with NO killer" section was
   rewritten to remove the three now-killed hunks and to name what genuinely
   remains (call-site wiring, with its distinct mitigation) without claiming AC-8
   covers it.
4. **Every "milestone N is independently safe" claim was re-derived under the new
   order** and rewritten; the round-1 claims quoted in gpt5-6-sol's catch
   ("the old harness ignores the additive key", "fields default to 0 until M1's
   emitter feeds them") are gone from the doc body.
5. Preserved unchanged from round 1 (already accepted): the D-29 out-of-scope
   section and its one-line diff, the M-8 accidental-flip tripwire, enumeration
   floors on every `go test -run` criterion, scoped build gates (no
   `go build ./...`), AC-5's live-KPI assert (controller re-measured at HEAD
   2026-08-23: rc=0, `cost_per_verified_success_usd = 0.7778187071999999`,
   `verified_successes = 3`, `verification_failures = 26`, `total_runs = 30`), and
   the Verification Log discipline. AC-5 additionally now runs at BOTH milestone
   boundaries. The D-29 predicate flip remains OUT OF SCOPE and parked.

---

### Round 2 — verdict: `blocked` (recorded by the controller, iteration 255)

Reviewers present: **2 of 2**, `absent_reviewers` empty — again full-strength.
**`gemini-3-1-pro`: PASS.** **`gpt5-6-sol`: REJECT**, on the single residual this doc
had named honestly in Conflict Surface item 3 and mitigated only by convention. The
controller's round-2 note had explicitly asked both reviewers to attack that item, so
this is the gate working, not a surprise.

**Surviving objection — `gpt5-6-sol` (verbatim):**

> "The design knowingly leaves a silent data-loss path: a pre-M1 harness can invoke a
> post-M2 `ailang` from PATH, ignore the additive `not_applicable` key, and bank a
> reduced `verify_skipped` count. An operational `make quick-install` convention and
> post-merge spot-check do not enforce compatibility or fail closed, so this P0
> data-integrity change violates the no-silent-fallback axiom."

**Its catch (verbatim):** "`RunAICheck` resolves an independently versioned PATH
executable, so reader-before-writer commit ordering does not guarantee
reader-before-writer deployment. The document explicitly admits that incompatible
binaries silently drop outcomes but treats it as residual risk rather than a release
blocker."

**Its proposed_fix (verbatim):** "Add a protocol-level compatibility mechanism before
the split. For example, make the harness request an explicit schema version
(`ailang ai-check --json-schema=2`), have the emitter produce `not_applicable` only for
schema 2, and make the M1 reader reject missing, unknown, or incompatible schema
versions rather than banking partial counters. Keep schema 1 emitting the conflated
`skipped` value for old harnesses. Alternatively, bind `RunAICheck` to the exact
same-version executable and prove that no old process can resolve a newly deployed
child. Add an integration test with old-reader/new-writer skew that must either
preserve the conflated count or fail before banking; replace the current residual-risk
paragraph with this enforced invariant."

**`gemini-3-1-pro` passed but left a non-blocking note worth keeping** (verbatim
proposed_fix): "In M1, add a loud inline comment block directly above both
`isVerifiedSuccess` and `isVerificationFailure` in `cost_per_verified_success.go`
explicitly warning that any future policy change to exclude `VerifyNotApplicable` (such
as D-29) MUST be applied to both predicates simultaneously to avoid orphan unverified
passes." Its reasoning: a partial D-29 implementation touching only the success
predicate would leave a `not_applicable`-only run classified as neither a verified
success nor a verification failure, silently spilling into `unverified_passes`. **Fold
this into M1 whenever this doc unparks** — it costs a comment block and closes a real
split-brain state. It also flagged that AC-8's grep-based constructor allowlist is
brittle to formatting/aliasing; treat that as a known limitation of that check.

**Controller measurement of the surviving objection (first-party, not inherited).**
`RunAICheck` defaults `ailangPath` to the bare string `"ailang"` and `exec.Command`
resolves it via **PATH** (`internal/eval_harness/verify.go:47-53`). **2 of 2** live
non-test call sites pass `""` — `internal/eval_harness/repair.go:76` and
`internal/eval_harness/verify.go:123` (same-scope control: `PopulateVerifyMetrics` has
2 callers; negative control on a fresh literal: 0). The parent harness and its verifier
child are therefore independently versioned. **The skew is live on this rig at the time
of writing**, not hypothetical: PATH `ailang` reports
`v0.33.1-211-g626f5e54b-dirty` while the repo/parent is `v0.33.1-216-g30176187f`. The
objection is correct and, measured, stronger than the doc stated.

**Disposition: PARKED `needs-human-review` as decision `D-30`** rather than resolved by
the controller. The narrow-refinement carve-out permits a controller-applied verbatim
fix only where the remaining objection needs no controller judgment; this one is a
choice between **(a)** a new versioned JSON wire contract, **(b)** rebinding
`RunAICheck` to `os.Executable()` — one line, and the idiom already has **9** in-repo
precedents including `cmd/ailang/replay.go:153`, but it silently changes how *every*
eval run resolves its verifier and collides with the mission loop's own mandated
scratch-build-and-prepend-`PATH` discipline — and **(c)** accepting a P0 data-integrity
residual. That is judgment, so it goes to the human (standing rule 2).

**Note for whoever unparks this:** the harness↔`ai-check` version coupling is a
**pre-existing** property of HEAD. This split does not create it; it makes it
consequential. `D-30` is worth a ruling whichever way `D-29` goes, and neither decision
blocks the other.
