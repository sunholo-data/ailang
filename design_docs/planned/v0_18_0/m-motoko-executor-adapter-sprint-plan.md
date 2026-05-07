# Sprint Plan: M-MOTOKO-EXECUTOR-ADAPTER

**Status**: Planned
**Target**: v0.18.0
**Estimated**: 3 working days (~18 hours)
**Source-of-truth design**: [m-motoko-executor-adapter.md](m-motoko-executor-adapter.md)

> This plan drives execution against the design doc. All architectural decisions, axiom scoring, risks, and rationale live there. This file is the milestone-by-milestone schedule.

> ✅ **PREREQUISITE LANDED**: M-MOTOKO-EVAL-INSTRUMENTATION shipped 2026-05-07 (motoko_agent commit `0c006be` on PR #6). Schema v1 is live: per-step `input_tokens`/`output_tokens`/`cost_usd` + terminal `run_summary` event + `schema_version` envelope. The adapter sprint is now unblocked and can populate `Result.CostUSD`/`InputTokens`/`OutputTokens` directly from motoko's JSONL.

---

## Why this sprint, why now

- **Prerequisite chain is clear**: M-MOTOKO-EXTENSION-INTEGRATION shipped today (2026-05-07) — motoko_agent now runs against AILANG dev with all 9 motoko-ext-* packages registry-published. The build is green. PR #6 is open.
- **EXECUTOR_SHAPE.md two-pillar contract** is hardened by the v0.15.0 codex+opencode work. Pillar 1 + Pillar 2 are well-trodden paths.
- **Strategic measurement gap**: we cannot answer "does motoko's harness lift cheap models?" without running the comparison. Every day this isn't measured is a day the cheap-model arbitrage thesis remains a hypothesis.
- **Zero coordinator/factory/dispatch code changes** — adding a new executor is a single blank import + one models.yml block. The leverage is high and the risk surface is tiny.

---

## Velocity calibration

Reference points (from existing executor packages):

| Executor | Impl LOC | Test LOC | Total | Sprint length |
|---|---:|---:|---:|---:|
| `gemini/gemini.go` | 560 | 156 | 716 | 1 week ([m-exec-gemini-sprint-plan](../../implemented/v0_6_1/m-exec-gemini-sprint-plan.md)) |
| `claude/claude.go` | 773 | 626 | 1,399 | larger (auth + creds) |
| `opencode/opencode.go` | 601 | ~450 | ~1,050 | 1 week (from M-EXEC-EXPAND) |
| `pi/pi.go` | ~500 | ~350 | ~850 | 1 week (M-EXEC-PI) |

**Planning target for motoko**: ~600 LOC impl + ~400 LOC tests + ~150 LOC fixtures = ~1,150 LOC total.

At the codex+opencode sprint's blended velocity (~210 LOC/day for executor-package work plus YAML/Dockerfile/multivac wiring), that's **~5.5 days**. With the EXECUTOR_SHAPE contract pre-hardened and motoko's session JSONL already a stable shape (no fork-side spec work), we compress to **3 days**.

---

## Milestone breakdown

Five milestones across three days. Each milestone is a clean acceptance gate: stop and verify before proceeding to the next.

| # | ID | Title | Est. LOC | Phase | Depends on |
|---|---|---|---:|---|---|
| M1 | M1_PILLAR1_CORE | Package skeleton + Register/init + Execute/HealthCheck | ~350 | Pillar 1 | — |
| M2 | M2_PILLAR1_PARSER | parseSessionJSONL + ExecuteStreaming + fixture capture | ~400 | Pillar 1 | M1 |
| M3 | M3_PILLAR1_WIRING | Config fields + blank import + models.yml + README + tests | ~250 | Pillar 1 | M2 |
| M4 | M4_PILLAR2_CLOUD | Dockerfile + knownVariants + multivac PRs (cloudbuild + terraform) | ~150 | Pillar 2 | M3 |
| M5 | M5_THRESHOLD_RUN | End-to-end measurement + paired comparison + CHANGELOG + move-to-implemented | ~50 | validation | M4 |

**Total**: ~1,200 LOC across 3 working days

---

## Day-by-day

### Day 1 — Pillar 1 foundation (~8 hours)

**Morning (4h):** M1
- Scaffold `internal/executor/motoko/motoko.go` with all 7 `executor.Executor` interface methods (stub returns initially)
- `New(cfg *executor.Config) (*MotokoExecutor, error)` constructor
- `Register()` + `init()` per EXECUTOR_SHAPE §2 contract
- Subprocess spawn in `Execute` (env: `WORKDIR`, `MODEL`, `MOTOKO_CONFIG`)
- `HealthCheck` (positive + negative test cases)
- `TestRegistration_Motoko` proves factory registration works

**Acceptance gate (M1):**
- `go build ./internal/executor/motoko/` clean
- `go test ./internal/executor/motoko/ -run TestRegistration_Motoko` passes
- `go test ./internal/executor/motoko/ -run TestHealthCheck` passes (mock binary)

**Afternoon (4h):** M2 start
- `parseSessionJSONL` — line-by-line parse with `json.RawMessage` + `ProviderData` for unknowns
- Map known events (`session_start`, `native_tool_calls`, `native_tool_results`, `cost_warning`, `cost_exhausted`, `dp7_verifier_rejected`, `done`) to `Result` fields
- Capture 3-5 real session JSONL fixtures from running motoko locally (success / failure / dp7-rejected / cost-exhausted)
- Commit fixtures to `testdata/`

**Acceptance gate (M2 partial — checked at start of Day 2):**
- 3-5 real session JSONL fixtures committed
- Fixture replay test passing for at least the success case

---

### Day 2 — Wiring + Cloud setup (~6 hours)

**Morning (3h):** M2 finish + M3 start
- `ExecuteStreaming` — tail JSONL during run via goroutine + handler
- `TestParseSessionJSONL_*` — one test per fixture (success, failure, dp7-rejected, cost-exhausted)
- `TestExecute_MockBinary` — POSIX shell stand-in exercises the full streaming path
- `TestLiveRun_Motoko` (gated by `AILANG_MOTOKO_LIVE=1`)
- Achieve ≥80% coverage

**Acceptance gate (M2):**
- All `TestParseSessionJSONL_*` tests pass
- Mock-binary test passes
- Coverage report ≥80% on `motoko.go`

**Afternoon (3h):** M3
- Add `MotokoPath`/`MotokoModel`/`MotokoProfile` to `executor.Config`
- One blank-import line in `internal/coordinator/provider_executor.go`
- 4 model entries in `internal/eval_harness/models.yml`: `motoko-claude-haiku-4-5`, `motoko-claude-sonnet-4-6`, `motoko-glm-5`, `motoko-gemma-4` — all `agent_cli: motoko`
- Add the 4 entries to `agent_suite` composite
- Write `internal/executor/motoko/README.md` per EXECUTOR_SHAPE §1 schema (flags, auth, schema, limits, **trust boundary** noting motoko's autonomous bash tool)
- `TestExecutorProvider_Motoko` (in `internal/coordinator/`) proves blank-import wiring
- `TestEvalSuite_MotokoModelEntries` (in `internal/eval_harness/`) proves models.yml expansion

**Acceptance gate (M3):**
- `make test` green
- `make lint` clean
- `ailang eval-suite --models motoko-claude-haiku-4-5 --benchmarks <one-tier> --dry-run` resolves the executor and prints expected dispatch plan

---

### Day 3 — Cloud + measurement (~4 hours)

**Morning (3h):** M4
- Author `docker/Dockerfile.agent-motoko` (FROM `agent-base`, USER root, install motoko binary, USER ailang) — mirror `Dockerfile.agent-pi` exactly
- Verify `docker build -f docker/Dockerfile.agent-motoko --build-arg PROJECT=ailang-dev -t agent-motoko:dev .` succeeds locally
- Verify `docker run --rm agent-motoko:dev motoko --version` works
- Add `"motoko"` to `knownVariants` in `internal/dispatch/cloudrun/dispatcher.go`
- Open ailang-multivac PR #1: add `build-agent-motoko` + `push-agent-motoko` to BOTH `cloudbuild.yaml` AND `cloudbuild-images.yaml`; update `push-images.waitFor` + `images:` lists in **both** files (per EXECUTOR_SHAPE §6 historical-drift warning)
- Open ailang-multivac PR #2: add `agent-motoko` Cloud Run Job to `terraform/cloud_run_jobs.tf` with **only** `OPENROUTER_API_KEY` + `OPENAI_API_KEY` + `GEMINI_API_KEY` bindings (NO `ANTHROPIC_API_KEY` per Design Freeze cost-control rule)
- After both multivac PRs merge to dev: `terraform apply` → coordinator dispatch with `--executor motoko` → dev smoke

**Acceptance gate (M4):**
- `docker build` succeeds locally
- Both cloudbuild files updated and in sync (CI rebuild produces `agent-motoko:latest` in Artifact Registry)
- `terraform apply` to dev creates the `agent-motoko` Cloud Run Job
- One coordinator-dispatched task with `--executor motoko` completes successfully in dev

**Afternoon (1h, may extend):** M5
- Run `ailang eval-suite --executor motoko --models motoko-claude-haiku-4-5 --benchmarks <agent-tier suite>` locally
- Run paired comparison: `motoko-claude-haiku-4-5` vs vanilla `claude-haiku-4-5` on the same suite; compute delta (pass-rate, cost, tokens)
- Update `changelogs/v0.10-current.md` `[Unreleased]` section with M-MOTOKO-EXECUTOR-ADAPTER entry citing concrete numbers
- Move design doc + sprint plan to `design_docs/implemented/v0_18_0/`
- Update design doc status to "Implemented", add Implementation Report

**Acceptance gate (M5):**
- At least one full paired-comparison row exists in eval results
- CHANGELOG entry has concrete numbers (not placeholders)
- Design doc moved to implemented/, status updated
- Sprint JSON marked `status: "completed"`

---

## Dependency graph

```
M1 (skeleton + Execute + HealthCheck)
  └── M2 (parser + streaming + fixtures)
        └── M3 (Config + wiring + models.yml + README + tests)
              └── M4 (Dockerfile + multivac PRs + dev smoke)
                    └── M5 (threshold run + CHANGELOG + finalize)
```

Strict serial order — each milestone unblocks the next. No parallel tracks.

---

## External dependencies

| Dependency | Status | Owner | Notes |
|---|---|---|---|
| **M-MOTOKO-EVAL-INSTRUMENTATION** (motoko-side JSONL schema v1) | ✅ **shipped 2026-05-07** | sunholo-data/motoko_agent | Landed on PR #6 (`motoko-dx-compaction-pending`, commit `0c006be`). Adds per-step `tokens`+`cost_usd` + terminal `run_summary` + schema_version envelope. ~250 LOC. See [implemented design doc](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md). |
| motoko_agent fork stable on AILANG dev | ✅ green | sunholo-data/motoko_agent | PR #6 open at `arniwesth/motoko_agent`; all today's fixes flowed through |
| 9 motoko-ext-* packages registry-published | ✅ done | sunholo-data/ailang-packages | abi 1.0.0 + 8 leaf packages including a2a@0.1.1 (today) |
| ailang-multivac dev pipeline access | ⚠️ verify | mark | dev project Cloud Build + Artifact Registry must be writable for sunholo-voight-kampff |
| AILANG_MOTOKO_LIVE test gate convention | ✅ established | this sprint | mirrors `AILANG_<NAME>_LIVE=1` pattern from EXECUTOR_SHAPE §Testing |

---

## Sprint JSON (template)

When starting execution, sprint-executor will populate:

```json
{
  "sprint_id": "M-MOTOKO-EXECUTOR-ADAPTER",
  "design_doc_path": "design_docs/planned/v0_18_0/m-motoko-executor-adapter.md",
  "sprint_plan_path": "design_docs/planned/v0_18_0/m-motoko-executor-adapter-sprint-plan.md",
  "target_version": "v0.18.0",
  "status": "not_started",
  "milestones": [
    {"id": "M1_PILLAR1_CORE",   "passes": null, "completed": null, "notes": ""},
    {"id": "M2_PILLAR1_PARSER", "passes": null, "completed": null, "notes": ""},
    {"id": "M3_PILLAR1_WIRING", "passes": null, "completed": null, "notes": ""},
    {"id": "M4_PILLAR2_CLOUD",  "passes": null, "completed": null, "notes": ""},
    {"id": "M5_THRESHOLD_RUN",  "passes": null, "completed": null, "notes": ""}
  ],
  "estimated_loc": 1200,
  "actual_loc": null,
  "estimated_days": 3,
  "actual_days": null
}
```

---

## Risks (sprint-execution-specific)

The design doc captures architectural risks. Sprint-execution risks specific to the schedule:

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Capturing 3-5 representative session JSONL fixtures takes >1 hour (M2) | Medium | Low (slips Day 1 → Day 2 boundary) | Have 5 candidate task scripts ready before Day 1 starts; capture in batch |
| ailang-multivac PR review latency blocks M4 dev smoke | Medium | Medium (Day 3 slips to Day 4) | Open the multivac PRs as drafts at start of Day 2 so reviewers see them early |
| `terraform apply` to dev fails due to existing-job-state drift | Low | Medium | Run `terraform plan` first; if drift exists, address separately before this sprint |
| Threshold-measurement run produces ambiguous numbers (motoko marginally better/worse) | Medium | Low | The measurement existing IS the deliverable — even ambiguous numbers are publishable; defer interpretation to follow-up issue |
| Pinning a motoko_agent commit (Design Freeze item) requires a new motoko release tag | Low | Low (use commit SHA if no tag yet) | Acceptable to pin to a SHA on `motoko-dx-compaction-pending` for v0.1; cut a real tag for v0.2 |

---

## Done criteria (sprint-level)

- [ ] M1–M5 all `passes: true` in sprint JSON
- [ ] `make ci` green on the merge commit
- [ ] `ailang eval-suite --executor motoko --models motoko-claude-haiku-4-5 --benchmarks <one-tier>` produces a real result file with non-zero `cost_usd`
- [ ] One paired-comparison row (`motoko-X` vs vanilla `X`) exists in eval results, regardless of which direction the delta points
- [ ] Cloud dev smoke: one coordinator-dispatched task with `--executor motoko` completes end-to-end
- [ ] CHANGELOG entry under `[Unreleased]` cites concrete pass-rate + cost numbers
- [ ] Design doc + sprint plan moved to `implemented/v0_18_0/` with Implementation Report appended

---

## References

- **Design doc** (source of truth): [m-motoko-executor-adapter.md](m-motoko-executor-adapter.md)
- **Source proposal**: msg `5f2facd3` (motoko-explore, 2026-05-07)
- **Canonical contract**: [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md)
- **Pillar 1 precedent**: [m-exec-expand-codex-opencode-sprint-plan.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md)
- **Pillar 2 precedent**: [m-exec-pi-harness-sprint-plan.md](../../implemented/v0_14_2/m-exec-pi-harness-sprint-plan.md)
- **Superseded**: [`design_docs/planned/v0_17_0/m-bench-motoko-executor.md`](../v0_17_0/m-bench-motoko-executor.md)
