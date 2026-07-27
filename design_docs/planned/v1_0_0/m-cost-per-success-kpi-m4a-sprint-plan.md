# Sprint Plan: Cost-Per-Verified-Success KPI — M4a (Cohort Freeze Mechanism)

**Design doc**: [m-cost-per-success-kpi.md](m-cost-per-success-kpi.md) (quorum-cleared iter-103, carve-out)
**Predecessor sprint**: [m-cost-per-success-kpi-sprint-plan.md](m-cost-per-success-kpi-sprint-plan.md) (M1–M3, LANDED)
**Sprint ID**: `M-COST-KPI-M4A`
**Branch**: `sprint/m-cost-kpi-m4a`
**Target**: v1.0.0
**Duration**: **0.5 day** (0.75 day if the blocking finding in §Blocking Findings is repaired in-sprint)
**Risk level**: Low (mechanism) / **HIGH on the premise that a frozen cohort can currently produce a
non-zero denominator — see Blocking Finding BF-1**

---

## Execution Record (2026-07-27, branch `sprint/m-cost-kpi-m4a`)

**Status: ✅ ALL MILESTONES COMPLETE.**

| Milestone | Commit | Notes |
|---|---|---|
| M4a-0 (extraction, mandatory first) | `37c070dd9` | `eval_suite.go` 790 → 764; pure move |
| M4a-1 (`--baseline` + BF-2) | `612cb78af` | one validator shared by both sides |
| M4a-2 (manifest + `cohort_hash`) | `fa4c1d095` | + 4 in-scope defects found while wiring |
| M4a-3 (BF-1 + BF-3) | `522ad61f1` | promoted CONDITIONAL → **REQUIRED** by the controller |
| M4a-4 (docs/changelog/design-doc split) | this commit | M4 split into M4a/M4b |

**CONTROLLER AMENDMENT (pre-execution):** the mission controller *independently
verified* BF-1 at `origin/dev` and promoted **M4a-3 from `CONDITIONAL` to REQUIRED**,
making it the sprint's critical path — the freeze mechanism (M4a-1/-2) is worthless
without it, because agent runs could never populate `verify_verified > 0` and the KPI
would return `zero_denominator` forever. BF-3 was folded in for the same reason (the
manifest records `verify_timeout`, so a hardcoded value would make the artifact lie).
BF-1's premise was re-confirmed in this worktree before any edit.

**Deviations from the plan** (all additive, none reduce scope):

1. **M4a-3 executed as REQUIRED** rather than conditional — per the controller amendment.
2. **The new surface is 3 files, not 1.** The plan put the manifest in
   `eval_suite_cohort.go`; it went to a sibling `eval_suite_manifest.go` instead
   (freeze/`source_ref` vs. the manifest artifact are separate concerns), and the
   agent-config block was extracted to `eval_suite_agent.go` because `eval_suite.go`
   had crept back to **798/800** against the hard CI gate. Final: 733 lines, 67 headroom.
3. **Freeze-time manifest write failure is FATAL, not best-effort.** The plan specified
   best-effort; that is right for the *finalize* rewrite (a completed metered run must
   never be discarded) but wrong at freeze time, where nothing has been spent yet and
   running a frozen cohort without its reproducibility artifact is the silent fallback
   CLAUDE.md §2 forbids. Implemented as an asymmetry, with both halves tested.
4. **An `ai-check` error is no longer swallowed.** The dead path discarded it with `_`.
   The gate is otherwise preserved verbatim; a verification that could not run leaves the
   block all-zero (unverified pass), with the reason printed.
5. **Four in-scope defects found while wiring** (CLAUDE.md §3), each with a regression test:
   `cleanResults()` deleted the manifest it had just written; `expandModelSuite`'s
   hardcoded 7-way switch became one table shared with the manifest recorder; three
   drifted `trials < 1` clamps collapsed into one; agent-mode `RunMetrics` never carried
   the 5 `verify_*` fields, so the banked result JSON could not show verification even
   once it ran.

**Not done, deliberately** (recorded as known limitations):

- `out_of_scope_M4b` — no benchmark suite was run, no `eval-suite` invocation against
  real models, **zero metered spend**. All tests use a fake verifier; no subprocess, no Z3.
- `out_of_scope_provenance` — the subscription-vs-metered cost finding is **not fixed**
  (open product decision). M4a only refuses to hide it: the manifest's per-model
  `executors[]` makes subscription lanes auditable (verified against the real
  `agent_suite`: `claude` vs `codex` vs `opencode` are distinguishable), plus a docs caveat.
- The residual legacy Claude-headless helpers (`RunHeadlessSessionStreaming`,
  `determineSuccess`, `checkClaudeCLI`, `getErrorMessage`, `ClaudeHeadlessResult`) were
  **retained**: still test-covered, removing them also touches the fmt-hook A/B machinery,
  and they carry **no** verification logic, so they cannot drift on the axis BF-1 was about.
  Follow-up cleanup. A comment at the deletion site records this.

---

## Sprint Summary

**Goal**: Give `ailang eval-suite` a **first-class, documented cohort-freeze mechanism** so a benchmark
run can be banked under an explicit baseline cohort ref that the already-shipped
`ailang chains stats --cost-per-verified-success --baseline <id>` query actually finds — and record a
**reproducible, data-driven cohort manifest** alongside it.

This is the *write side* of the KPI. M1–M3 shipped the read side (rollup → CLI/HTTP → `latest.json` →
dashboard card). The read side is currently **unreachable**: no eval run can land in the `v1.0/` cohort,
so the KPI permanently returns `{"available":false,"reason":"empty_cohort"}`.

### In scope

1. **`--baseline <id>` on `eval-suite`** — writes `chains.source_ref` as `<baseline>/<taskID>/<mode>[/<cond>]`
   so the M1 `SourceRefPrefix` filter matches. Default (flag absent) behaviour stays **byte-identical**.
2. **`cohort_manifest.json`** — an explicit, versioned, **data-driven** manifest (models resolved from
   `agent_suite` in `models.yml`, never a hardcoded Go list) + a stable `cohort_hash`, written into the
   run's output dir and closed out with the run window at finalize.
3. **Baseline-id validation** — fail loudly on ids that would corrupt the `LIKE` prefix match.
4. Tests, `changelogs/v0.18-current.md` `[Unreleased]` entry, and docs for the documented freeze command.

### Explicitly OUT of scope

| Tag | Item | Why |
|---|---|---|
| `out_of_scope_M4b` | **Running** the baseline cohort and publishing the measured number | Real metered exposure on the OpenRouter lanes; parked on the human's spend approval. |
| `out_of_scope_provenance` | Fixing `ClassifyStageCost` rule 1 treating subscription-CLI "cost" as metered | See §Known Limitation Deliberately Not Addressed. Depends on an open product decision; must NOT be "fixed" here. |
| `out_of_scope_hash_enforcement` | Strict-mode **comparison** of the recorded `cohort_hash` against the queried cohort | Requires the manifest to be reachable from the DB (`execution_chains` has no metadata column → migration). The hash is *recorded* here; enforcing it is a follow-up. |
| `out_of_scope_db_manifest` | A `cohort_manifest` column on `execution_chains` | Schema migration; the file+`source_ref` pair is sufficient release evidence for M4a. |
| `out_of_scope_placement` | Mark's primary-landing-surface mirror for the KPI card | Still Mark-gated from the M1–M3 sprint. |

---

## Ground-Truth Verification (planner probe, 2026-07-27, worktree `.claude/worktrees/m4a` @ `0977fb89b`)

Every premise handed to the planner was re-checked against the code. Results:

| Premise | Verdict | Evidence |
|---|---|---|
| KPI query filters chains on `source_ref` prefix `<baseline>/` | ✅ CONFIRMED | `cmd/ailang/chains_stats_cvs.go:53` → `cohortSourceRefPrefix()` at `:84-92`; consumed as `SourceRefPrefix` in `internal/observatory/cost_per_verified_success.go:154` → `store_chains_eval.go:95-98` (`c.source_ref LIKE ?`) |
| `eval-suite` hardcodes `SourceRef` from `taskID` | ✅ CONFIRMED | `cmd/ailang/eval_suite.go:457` — `fmt.Sprintf("%s/%s%s", taskID, evalMode, condRef)`, inside the block at `:437-470` |
| `taskID` comes from correlation IDs or a timestamp fallback | ✅ CONFIRMED | `eval_suite.go:53` (`telemetry.ExtractCorrelationIDs()`), `:60` (`fmt.Sprintf("eval-%d", …UnixNano())`) |
| No `--source-ref` / `--baseline` / cohort flag on `eval-suite` | ✅ CONFIRMED | Flag block `eval_suite.go:104-157`; `grep -n "source-ref\|SourceRef" cmd/ailang/eval_suite*.go` → **only** the write at `:457` |
| KPI is live-unreachable → `empty_cohort` | ✅ CONFIRMED by construction | No writer can emit a `v1.0/`-prefixed `source_ref`; `cost_per_verified_success.go:201-203` returns `CVSReasonEmptyCohort` |
| `OTEL_RESOURCE_ATTRIBUTES` workaround collides with the task registry | ✅ CONFIRMED | `taskID` is also the Observatory `Task.ID` at `cmd/ailang/eval_observatory.go:54` via `createEvalTask(taskID, …)` (`eval_suite.go:435`) — hijacking it to `v1.0` would collide the task row across every freeze run |
| M1 already added the reusable cohort filter | ✅ CONFIRMED | `EvalQueryOptions.SourceRefPrefix` / `CreatedAfter` / `CreatedBefore`, with a conditional JOIN to `execution_chains` (`store_chains_eval.go:74-106`). **Reuse — do not add a second filter path.** |
| Exactly 7 contract-bearing benchmarks | ✅ CONFIRMED | `grep -l contract_spec benchmarks/` → `contract_bst_validate`, `contract_leap_year`, `contract_matrix_determinant`, `contract_rle_roundtrip`, `contract_roman_numeral`, `contract_sorted_merge`, `prompt_injection` |
| `agent_suite` = 6 cloud models, no GPU | ✅ CONFIRMED | `internal/eval_harness/models.yml:2789-2803`; accessor `ModelsConfig.GetAgentSuite()` (`models.go:299-303`), suite-token expansion `cmd/ailang/eval_suite_types.go:47-49` — **this is the data-driven source the manifest must read** |
| Verification wired for **agent** mode | ❌ **CONTRADICTED — see BF-1** | The only agent-mode `RunAICheck` call is on a **dead** code path |

---

## Blocking Findings

### BF-1 (BLOCKING for `out_of_scope_M4b`) — agent-mode contract verification is on a DEAD code path

The handed-down premise stated verification "is already wired for agent mode
(`internal/eval_harness/agent_runner.go:316` + `agent_runner_multi.go`)". **The second half is false, and
the first half is unreachable.**

- `internal/eval_harness/agent_runner.go:316-317` is the *only* agent-mode verification call:
  ```go
  if config.Verify && spec.ContractSpec != "" && language == "ailang" && validation.CompileOk {
      verifyResult, verifyRawJSON, _ = RunAICheck("", solutionPath, 5*time.Second)
  }
  ```
  It lives in `RunAgentBenchmark()` (`agent_runner.go:195`).
- `RunAgentBenchmark()` has **no caller**. `grep -rn "RunAgentBenchmark(" --include="*.go" .` returns only
  its own definition plus a *comment* at `cmd/ailang/eval_benchmark.go:129`:
  > `// Always use multi-executor runner -- legacy RunAgentBenchmark() hardcodes Claude`
  > `// and must NOT be used for non-Claude models`
- The live path is `eval_harness.RunAgentBenchmarkWithExecutor(…)` (`eval_benchmark.go:179` →
  `agent_runner_multi.go:76`). `grep -n "Verify" internal/eval_harness/agent_runner_multi.go` returns
  **zero matches**. `grep -rn "RunAICheck" internal/eval_harness/ internal/executor/` returns exactly two
  live sites: `repair.go:76` (**standard** mode only) and the dead `agent_runner.go:317`.

**Consequence**: with the freeze mechanism landed and a `--verify --agent` cohort run executed, every
banked `eval_assessment` would carry an all-zero verify block. `isVerifiedSuccess()`
(`cost_per_verified_success.go:95-105`) requires `VerifyVerified > 0`, so the cohort classifies as
`unverified_passes`, `VerifiedSuccesses == 0`, and the KPI returns
`available:false, reason:"zero_denominator"` — **forever**. This is also exactly consistent with the
established fact that no banked run has ever had `verify_verified > 0`; that is a harness gap, not an
absence of opportunity.

**Routing** (per CLAUDE.md §3 SYSTEMIC FIXES — audit before patching): this is one gap with two
manifestations (a dead correct implementation + a live implementation missing it). The fix is to move the
verification block onto the live multi-executor path and **delete** the dead one, not to duplicate it.

**Handling in this sprint**: carried as milestone **M4a-3**, marked `CONDITIONAL`. It does **not** gate
M4a-1/M4a-2/M4a-4, and it does **not** run any benchmark (still `out_of_scope_M4b`). If the controller
prefers strict scope discipline, drop M4a-3 and file it as its own item — but M4b **must not** be greenlit
for spend until BF-1 is closed, or the metered run is guaranteed to yield an unavailable KPI.

### BF-2 (non-blocking, in-sprint hardening) — the baseline id is an unescaped SQL `LIKE` pattern

`store_chains_eval.go:96-97` builds `c.source_ref LIKE ?` with `opts.SourceRefPrefix+"%"` and no
`ESCAPE` clause. SQLite `LIKE` treats `_` as a single-character wildcard, so a baseline id containing
`_` or `%` silently widens the cohort (e.g. `v1_0` also matches `v1x0/…`). Today only the read side
accepts a baseline id, so this is latent. **M4a introduces the write side, which is exactly where the
charset must be pinned** — validate at freeze time (`^[A-Za-z0-9][A-Za-z0-9.-]*$`) and reuse the same
validator on the read side so a frozen id is always a literal prefix.

### BF-3 (non-blocking, note only) — `--verify-timeout` is ignored on the agent path

`eval_suite.go:182` sets `evalVerifyTimeout` from `--verify-timeout`; it reaches only
`repairRunner.SetVerify(…)` (`eval.go:166`, `eval_benchmark.go:561`) i.e. standard mode. The agent-mode
call hardcodes `5*time.Second` (`agent_runner.go:317`). Fold the plumbing in **if and only if** M4a-3 is
executed (it is one parameter on the moved block); otherwise record it as a manifest caveat, because the
manifest claims to record the verify timeout.

---

## Known Limitation Deliberately Not Addressed (`out_of_scope_provenance`)

`internal/observatory/cost_classify.go` rule 1 treats any `stage.Cost > 0` as authoritative
`CostStatusReported`. The **subscription** `claude` CLI reports a non-zero `total_cost_usd` even when
nothing is billed (live-probed with both Anthropic API keys stripped: 10 in / 46 out tokens returned
`total_cost_usd: 0.0108355`). On this rig `ANTHROPIC_API_KEY` is deliberately absent, so every `claude`
agent run is subscription and its "cost" is a **list-price equivalent, not a metered dollar**. The design
doc's Primary Goal says *metered* dollars, so an `agent_suite` cohort spanning `claude-haiku-4-5` /
`claude-sonnet-4-6` (notional) plus the OpenRouter lanes (real) blends both under one `reported` label.

**This sprint designs no fix.** It is an open product decision (does a subscription lane belong in a
metered KPI at all, and if so at what price?). M4a's only obligation is **not to hide it**: the cohort
manifest records the resolved executor per model, which is the audit hook a reviewer needs to see that
`claude`-CLI rows are subscription lanes. A one-line caveat goes in the docs section, not in code.

---

## Velocity

- Last 3 days: 24 commits; last 10 commits: 14 files, 1,302 insertions.
- M1–M3 of this same design doc landed **in one day** for ~1,900 LOC across 3 commits
  (`9bdc9319c`, `2a2a40f31`, `2d76b2cc3`, changelog `d869ec12d`, merged into `dev`).
- M4a is a ~460 LOC sprint against a codebase the executor's predecessor just wrote. **0.5 day is
  realistic**; 0.75 day with M4a-3.

---

## Proposed Milestones

### M4a-1 — `--baseline` cohort freeze flag on `eval-suite` — ✅ DONE (`612cb78af`)
**Estimated**: ~95 impl + ~95 tests = **~190 LOC** · **~2.5 h**

**Design (single mechanism, no second code path):**

- New flag in the `eval-suite` FlagSet (`eval_suite.go`, ~line 137 near the other M-CONTRACT-EVAL flags):
  ```
  --baseline <id>   Bank this run under an immutable baseline cohort ref so
                    `ailang chains stats --cost-per-verified-success --baseline <id>`
                    finds it. Empty (default) = today's timestamp/correlation source_ref.
  ```
  Name chosen to be **symmetric with the read side** (`chains stats --baseline`), so the freeze command
  and the query command use the same word for the same value. No `--source-ref` alias (two spellings for
  one cohort key is a drift vector).
- `source_ref` composition becomes:
  - `--baseline` **absent** → `"<taskID>/<mode><condRef>"` — **byte-identical to `eval_suite.go:457` today**.
  - `--baseline v1.0` → `"v1.0/<taskID>/<mode><condRef>"` — prefix `v1.0/` matches
    `cohortSourceRefPrefix("v1.0")`, and `taskID` is *retained inside* the ref so trace/task correlation
    and per-run disambiguation survive.
- **`taskID`, `assignmentID`, `OTEL_RESOURCE_ATTRIBUTES`, and `createEvalTask(taskID, …)` are NOT touched.**
  The baseline is a *chain* cohort label, not a correlation ID. This is the whole point of the flag vs. the
  `ailang.task_id=v1.0` hack.
- Fail loudly (CLAUDE.md §2), before any metered work starts:
  - invalid baseline id (BF-2 charset) → error + exit 1;
  - `--baseline` set **without** `--verify` → error + exit 1, naming `--verify` ("a frozen cohort with no
    verification evidence can only ever produce `zero_denominator`"). This is a hard error, not a warning:
    the failure mode it prevents is an expensive metered run that yields an unusable KPI.

**Tasks**
1. Create `cmd/ailang/eval_suite_cohort.go`. **Move** the chain-creation block (`eval_suite.go:437-470`)
   into it as `createEvalChain(ctx, params) *EvalChainContext`, and add `cohortSourceRef(baselineID, taskID,
   evalMode, condRef) string` + `validateBaselineID(string) error`. See the file-size risk below — this
   extraction is mandatory, not cosmetic.
2. Register `--baseline` in `eval_suite.go` and thread it into the extracted helper.
3. Wire the two fail-loud validations immediately after `fs.Parse` (before the rig lock / API-key checks /
   any spend).
4. Export/relocate `validateBaselineID` so **`chains_stats_cvs.go` reuses it** on the read side (one
   validator, both sides — closes BF-2).

**Acceptance criteria**
- [x] `cohortSourceRef("", "eval-123", "agent", "")` == `"eval-123/agent"` (today's exact string).
- [x] `cohortSourceRef("v1.0", "eval-123", "agent", "/full")` == `"v1.0/eval-123/agent/full"`.
- [x] **Round-trip invariant test** (the load-bearing one): for a table of baseline ids, the string from
      `cohortSourceRef` matches the `LIKE` prefix produced by `cohortSourceRefPrefix` from
      `chains_stats_cvs.go:84` — i.e. writer and reader provably agree.
- [x] `validateBaselineID` rejects `""`, `"v1_0"`, `"50%"`, `"/v1.0"`, `"v1.0/x"`, and accepts `"v1.0"`,
      `"v1.0-rc1"`, `"os-rolling.2"`.
- [x] `--baseline` without `--verify` exits non-zero with a message naming `--verify`.
- [x] `taskID` / `assignmentID` / `OTEL_RESOURCE_ATTRIBUTES` / `createEvalTask` semantics unchanged
      (assert the correlation block at `eval_suite.go:52-67` is untouched in the diff).
- [x] `make check-file-sizes`, `make check-boundaries`, `make lint`, `go test ./cmd/... ./internal/observatory/...` green.

**Risks**
- **`cmd/ailang/eval_suite.go` is 790/800 lines.** `make check-file-sizes` is a CI gate at a hard 800
  (`make/code-health.mk:122-137`). Adding a flag + validation inline **would break CI.** Mitigation: the
  `:437-470` extraction in task 1 removes ~34 lines *before* ~8 are added → net ~764. Do the extraction
  **first** and re-run `make check-file-sizes` before writing anything else.

---

### M4a-2 — Recorded, data-driven cohort manifest — ✅ DONE (`fa4c1d095`)
**Estimated**: ~130 impl + ~85 tests = **~215 LOC** · **~2 h**

**Design**: `cohort_manifest.json` written into the run's `outputDir` (which already respects
`--bank-by-version`, `eval_suite.go:169-176`), only when `--baseline` is set. It is the reproducibility
artifact the design doc's final acceptance criterion demands ("the exact frozen cohort manifest and command
output; a reviewer can independently recompute it without dashboard JavaScript").

Fields — all **derived from the resolved run configuration**, never hardcoded:

| Field | Source |
|---|---|
| `baseline_id`, `source_ref_prefix` | `--baseline` + `cohortSourceRef` |
| `frozen_at`, `run_window{started_at,completed_at}` | wall clock; `completed_at` filled in at finalize |
| `eval_mode`, `languages`, `conditions` | `--agent`, `--langs`, `--conditions` (resolved lists) |
| `model_suite` | the literal `--models` token when it is a suite name (e.g. `"agent_suite"`), else `""` |
| `models[]` | the **resolved** list from `expandModelSuite` (`eval_suite_types.go:40-75`) → `ModelsConfig.GetAgentSuite()` |
| `executors[]` (per model) | `ModelsConfig.GetExecutorForModel(model)` (`models.go:360`) — also the `out_of_scope_provenance` audit hook |
| `benchmarks[]` | the **resolved** benchmark id list (post-discovery, post-`--tier`) |
| `seed`, `prompt_version`, `trials` | `--seed`, `--prompt-version`, `--trials` |
| `verify`, `verify_timeout` | `--verify`, `--verify-timeout` (see BF-3 caveat if M4a-3 is dropped) |
| `ailang_version`, `git_commit` | `internal/version` (already imported in `eval_suite.go`) |
| `cohort_hash` | SHA-256 over the canonical (sorted) identity fields |

**Re-freezability** (Mark's 2026-07-27 ratification: *"assume current cohort but this may have light
changes depending on release date"*): the manifest is **explicit and versioned but derived**. Nothing in Go
names a model. If `agent_suite` in `models.yml` changes before the release, re-running the freeze command
produces a manifest with the new members **and a different `cohort_hash`** — the drift is visible in the
artifact instead of silent. A re-freeze publishes as a **new baseline id** (`v1.0-rc2`, …), never a rewrite
of a published one.

**Tasks**
1. `CohortManifest` struct + `buildCohortManifest(...)` + `writeCohortManifest(dir, *CohortManifest)` in
   `cmd/ailang/eval_suite_cohort.go` (keeps the new surface in one ≤300-line file).
2. Call at freeze time (right after the chain is created, so `chain_id` can also be recorded) and print the
   absolute manifest path so the run's stdout is self-documenting release evidence.
3. Add `cohortManifestPath string` to `suiteSummaryParams` (`eval_suite_finalize.go:19-31`) and re-write the
   manifest with `run_window.completed_at` in `finalizeSuiteRun`. Best-effort with a **printed** note on
   failure (matching the adjacent `summary.json` convention at `:79-87`) — a manifest write failure must not
   discard a completed metered run, but it must not be silent either.
4. `cohort_hash` over sorted `models` + sorted `benchmarks` + `languages` + `eval_mode` + `seed` +
   `prompt_version` + `conditions` + `trials`. Explicitly **exclude** timestamps, `git_commit`, and
   `chain_id` so the hash identifies the *cohort*, not the *run*.

**Acceptance criteria**
- [x] No `--baseline` → **no** manifest file written, and the run is otherwise unchanged.
- [x] `--baseline v1.0` → `<outputDir>/cohort_manifest.json` exists, parses, and carries every field above.
- [x] `models[]` equals `GetAgentSuite()` from `models.yml` for `--models agent_suite`; the test reads
      `models.yml` rather than asserting a literal 6-name list, so a suite edit does **not** fail the test
      but **does** change the manifest.
- [x] `model_suite` records `"agent_suite"`; an explicit comma list records `""` with the same resolved members.
- [x] `cohort_hash` is stable across two builds of the same inputs, order-independent for `models`/
      `benchmarks`, and **changes** when a model or benchmark is added/removed.
- [x] `cohort_hash` is **unchanged** by `frozen_at` / `git_commit` / `chain_id` differences.
- [x] `run_window.completed_at` is populated after `finalizeSuiteRun`.
- [x] `executors[]` distinguishes `claude` from `opencode` / `codex` (the provenance audit hook).

**Risks**
- Manifest field creep → keep it to the table above; anything else is a follow-up.
- Writing the manifest before benchmark discovery would record an unresolved list. Mitigation: build it
  **after** the benchmark/model resolution and the `agent`-mode model filter (`eval_suite.go:~400-420`),
  before the run loop.

---

### M4a-3 — ~~`CONDITIONAL`~~ **REQUIRED** — move agent-mode verification onto the live path (closes BF-1) — ✅ DONE (`522ad61f1`)
**Estimated**: ~35 impl + ~60 tests = **~95 LOC** · **~1.5 h** · **push to 0.75 day**

Without this, M4a-1/M4a-2 ship a freeze mechanism that provably cannot yield a number. Scoped as a
**move + delete**, not a new feature:

**Tasks**
1. Move the verification block (`agent_runner.go:314-358`: the `RunAICheck` gate plus the six
   `agentResult.Verify*` assignments) into the live `RunAgentBenchmarkWithExecutor` result construction in
   `internal/eval_harness/agent_runner_multi.go`, preserving the gate verbatim
   (`config.Verify && spec.ContractSpec != "" && language == "ailang" && validation.CompileOk`).
2. **Delete** the dead `RunAgentBenchmark()` verification block (and, if nothing else references it, the
   dead function) — per `.claude/rules/coding-standards.md` this is a *rename/relocate*, so the call site
   moves with it; do not leave two copies. Run `make test` between the move and the delete.
3. Thread `verifyTimeout` through `MultiExecutorConfig` instead of the hardcoded `5*time.Second` (closes BF-3
   and makes the manifest's `verify_timeout` field truthful).
4. Test: a fake executor + a `contract_spec`-bearing spec produces `VerifyVerified > 0` and
   `VerifyOk == true` on the multi-executor path; a spec with **no** `contract_spec` leaves the block all-zero
   (→ `verification_missing`, never success); `Verify == false` leaves it all-zero.
5. End-to-end assertion (no network, no spend): a fabricated `EvalAssessment` built from a verified
   multi-executor result satisfies `observatory.isVerifiedSuccess` — i.e. the write side and the M1
   predicate agree.

**Acceptance criteria**
- [x] `grep -rn "RunAICheck" internal/eval_harness/` shows the agent call on the **live** multi-executor path.
- [x] No `RunAICheck` call remains behind a caller-less function.
- [x] `--verify-timeout` reaches the agent-mode verifier.
- [x] `make test` green; `make check-boundaries` green; no benchmark executed.

**Risks**
- Scope objection. Mitigation: it is severable — M4a-1/2/4 are independently mergeable. Record the drop
  decision in the sprint JSON `notes` if the controller cuts it, and hard-block M4b spend on it.
- `agent_runner_multi.go` is 659 lines; +35 → ~694. Under the 800 gate, but re-run `make check-file-sizes`.

---

### M4a-4 — Docs + changelog — ✅ DONE
**Estimated**: ~0 code · **~0.5 h**

**Tasks**
1. `changelogs/v0.18-current.md` `[Unreleased]`: extend the existing
   *"Added — Cost-Per-Verified-Success KPI (M-COST-PER-SUCCESS-KPI, M1–M3)"* section (line 71) with an
   **M4a** bullet group — do not open a competing section. Must state: the KPI was previously unreachable
   (`empty_cohort`) because no writer could emit a cohort-prefixed `source_ref`; the default `source_ref`
   is unchanged; `--baseline` requires `--verify`; the manifest is derived from `models.yml`;
   and the `out_of_scope_provenance` subscription-cost limitation.
2. `docs/docs/guides/evaluation/eval-loop.md`: a short **"Freezing a baseline cohort"** section giving the
   exact reproducible pair —
   ```bash
   # 1. freeze + run (metered — see the cost caveat below)
   ailang eval-suite --agent --langs ailang --verify \
     --models agent_suite \
     --benchmarks contract_bst_validate,contract_leap_year,contract_matrix_determinant,contract_rle_roundtrip,contract_roman_numeral,contract_sorted_merge,prompt_injection \
     --baseline v1.0 --seed 42 --no-rig-lock

   # 2. reproduce the published KPI from banked data
   ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict
   ```
   plus: the manifest path, that the two commands share one baseline id, the allowed id charset, and a
   two-line **cost-provenance caveat** (`claude`-CLI lanes on a key-less rig are subscription/list-price,
   not metered — `out_of_scope_provenance`).
3. `cmd/ailang/help.go`: one `eval-suite --baseline` example line near `:296-297` (CLI help is the
   single source of truth per the `cli-doc-maintainer` skill).
4. `design_docs/planned/v1_0_0/m-cost-per-success-kpi.md`: mark M4 as split — **M4a mechanism (this sprint)**
   / **M4b measured baseline (Mark-gated spend)** — and add BF-1 to Risks & Open Questions.

**Acceptance criteria**
- [x] `[Unreleased]` entry present, inside the existing KPI section.
- [x] Docs give a copy-pasteable freeze command whose baseline id matches the query command's.
- [x] The subscription-cost caveat appears in the docs and is labelled a known, unaddressed limitation.
- [x] Design doc M4 split recorded; BF-1 is on the record as an M4b blocker.
- [x] `cd docs && npm run build` (or the repo's docs check) passes.

---

## Day Plan (single 0.5–0.75 day session)

| Slot | Work |
|---|---|
| 0:00–0:20 | **Extraction first**: create `eval_suite_cohort.go`, move `eval_suite.go:437-470`, re-run `make check-file-sizes`. Do not proceed until green. |
| 0:20–1:40 | M4a-1: `--baseline`, `cohortSourceRef`, `validateBaselineID` (shared with the read side), fail-loud validations + tests (round-trip invariant first). |
| 1:40–3:20 | M4a-2: manifest struct/build/write, finalize hook, `cohort_hash` + tests. |
| 3:20–4:20 | M4a-3 (`CONDITIONAL`): move the verification block, delete the dead one, thread the timeout, tests. |
| 4:20–5:00 | M4a-4: changelog, docs, help.go, design-doc M4 split. `make lint`, `make check-boundaries`, `make check-file-sizes`, `go test ./...`. Commit per milestone. |

---

## Success Metrics

- **No behaviour change without `--baseline`**: the default `source_ref` string is byte-identical, proven
  by a test, not by inspection.
- **Writer/reader agreement is tested**, not asserted: `cohortSourceRef` output matches
  `cohortSourceRefPrefix`'s `LIKE` prefix for every id in the table.
- Manifest is derived from `models.yml` — the test reads the suite instead of pinning a literal model list.
- CI gates: `make check-file-sizes` (watch `eval_suite.go`), `make check-boundaries`, `make lint`,
  `go test ./...`.
- **Zero benchmark executions, zero metered spend** in this sprint.
- No example `.ail` file is required: this is CLI/eval-harness plumbing, not a language feature.

## Dependencies

- M1–M3 landed on `dev` (`9bdc9319c`, `2a2a40f31`, `2d76b2cc3`, `d869ec12d`) — present in this worktree's base.
- `z3 4.16.0` on PATH (only needed by M4b, not by this sprint's tests).
- Nothing else. No GPU, no `rig.lock`, no network.

## Open Questions (for the human)

1. **BF-1 routing** — fold M4a-3 into this sprint (recommended: it is a move+delete and it is the difference
   between a mechanism and a *working* mechanism), or split it out and block M4b on it?
2. **`out_of_scope_provenance`** — does a subscription `claude` lane belong in a *metered* KPI cohort at all?
   If not, the `v1.0` cohort should probably drop `claude-haiku-4-5`/`claude-sonnet-4-6` and the headline
   becomes an OpenRouter+codex number. This changes the cohort manifest, so it is worth answering **before**
   M4b spend, not after.
3. **Baseline id for the release** — `v1.0` exactly, or `v1.0-rc1` first so a re-freeze after any late
   `agent_suite` edit does not need to rewrite a published id?
