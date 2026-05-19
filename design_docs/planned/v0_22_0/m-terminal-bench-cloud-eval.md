# M-TERMINAL-BENCH: Terminal-Bench Cloud Evaluation (AILANG vs Python)

**Status**: Planned
**Target**: v0.22.0
**Priority**: P1 (High) — Strategic external benchmark validation
**Estimated**: 4-6 weeks (phased)
**Dependencies**:
- Multivac Cloud Run Jobs infrastructure (deployed, see [ailang-multivac/terraform/cloud_run_jobs.tf](../../../../ailang-multivac/terraform/cloud_run_jobs.tf))
- `internal/eval_harness/` agent-based evaluation (complete)
- Motoko/Claude executor abstraction (complete, v0.18+)
- AILANG runtime + caps (complete)

---

## Executive Summary

Integrate [Terminal-Bench](https://www.tbench.ai/) as a third-party benchmark to measure the value of **the AILANG language** (not just the AILANG harness) versus Python, holding the underlying model constant. All runs execute on ailang-multivac Cloud Run Jobs — developers never need Docker locally.

**Key framing**: Terminal-Bench primarily benchmarks the *agent harness* (how a model operates a terminal to complete a task). For our purposes, we hold the harness and model constant and vary only the **implementation language** the agent is instructed to use. This isolates "does AILANG produce more reliably-correct programs than Python for the same task and model?"

**Hypothesis**: On the build-a-program subset of Terminal-Bench, same model + AILANG will pass equal-or-more tasks than same model + Python, primarily because explicit effects and deterministic semantics catch a class of bugs at compile time that Python only surfaces at runtime.

---

## Research Foundation

### Terminal-Bench Overview

Terminal-Bench is an open agentic benchmark where an LLM agent operates a real Linux terminal inside a per-task Docker container, scored by deterministic grading scripts that check final filesystem/process/output state. Tasks ship as container images with:
- A goal description (natural language)
- A pre-built container environment (FROM image + setup)
- A grader script (typically pytest or bash) that produces pass/fail

Tasks span: filesystem recovery, log analysis, service debugging, program building, data transformation. Roughly **20-40% of tasks** ("build-a-program" subset) have the property that the deliverable is *a program the agent writes*, where the choice of implementation language is free.

### Why Terminal-Bench for AILANG

| Task Property | AILANG Advantage | Expected Signal |
|---------------|------------------|------------------|
| **Programs invoked from shell** | AILANG binary is a single executable, capability flags grade-able | Parity or better |
| **Effect-typed I/O** | Compile-time catches missing `--caps IO,FS` rather than runtime crash | Fewer silent passes-then-fails |
| **Deterministic execution** | Same input → same output, no flakiness in re-grading | Lower variance across attempts |
| **No hidden state** | Module imports explicit, no `sys.path` foot-guns | Fewer "works on my machine" failures |

**What we will NOT learn from Terminal-Bench**: anything about AILANG's edit-loop or REPL ergonomics — the harness itself is the variable being held constant.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same model + task → reproducible attempt (TB tasks are containerized, AILANG runs deterministic) |
| A2: Replayability | +1 | Cloud Run Job logs + AILANG eval results store give full replay |
| A3: Effect Legibility | 0 | Benchmarking feature, not a language change |
| A4: Explicit Authority | +1 | AILANG arm requires explicit `--caps` declaration; Python arm has ambient authority — and this difference is part of what we measure |
| A5: Bounded Verification | +1 | Per-task grading verdicts are local and independent |
| A6: Safe Concurrency | +1 | Job-per-task, no shared state between attempts |
| A7: Machines First | +1 | All comparison results in structured JSON; no human prose in critical path |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +1 | Per-attempt token + Cloud Run Job cost tracked into existing eval results store |
| A10: Composability | +1 | Reuses existing eval harness + multivac infra; no new persistence layer |
| A11: Structured Failure | +1 | Task verdicts categorize: compile fail / runtime fail / wrong output / timeout |
| A12: System Boundary | +1 | TB container ↔ AILANG runtime boundary explicit (one Cloud Run Job = one boundary crossing) |

**Net Score: +9** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Tasks are seeded, AILANG is deterministic, TB grading is mechanical
- [x] A3 (Effects): No new effects introduced
- [x] A4 (Authority): AILANG arm enforces caps; we don't loosen Python's ambient model either
- [x] A7 (Machines First): JSON throughout

---

## Problem Statement

### Current State (v0.20.x)

- AILANG evals run against **our own benchmark set** (47 single-file tasks plus cross-language matrix)
- No third-party validation: when AILANG outperforms Python on our benchmarks, skeptics can argue the benchmarks are tuned to AILANG's strengths
- Cross-language matrix exists ([eval-lang-jsgo](../../implemented/v0_15_0/m-eval-lang-jsgo-sprint-plan.md)) but is still our taxonomy
- No leaderboard presence for AILANG anywhere — no public datapoint a skeptic can grep for

### Impact

| Audience | Without TB integration | With TB integration |
|----------|------------------------|---------------------|
| AILANG skeptics | "Your benchmarks are tuned to your language" | "Public benchmark, same model, language is the only variable" |
| Funders / partners | No external validation | Comparable datapoint against industry-standard agent benchmark |
| Academic publication | Self-benchmarked claims | Citable third-party signal |
| Internal sprint planning | Hard to spot real regressions in language design | TB delta becomes a tripwire for releases |

---

## Goals

**Primary Goal**: Run a defined subset of Terminal-Bench tasks twice (Python arm, AILANG arm) on the same model on Cloud Run Jobs, and produce a delta report that lands in AILANG's eval results store.

**Success Metrics**:
- ≥50 TB tasks (build-a-program subset) runnable in both arms
- Full matrix (50 tasks × 4 models × 2 arms × 3 attempts) completes in <2 hours wall-clock on Cloud Run Jobs
- Cost ≤ $25/run for the full matrix
- Delta report auto-generated and viewable in the dashboard
- Zero local Docker required for developer or CI

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Leaderboard-target vs eval-substrate framing | Determines whether we follow TB's exact protocol (and submit) or cherry-pick tasks (and stay internal) | human | design | high |
| Task subset selection criteria | "Build-a-program" is fuzzy — need explicit predicate (e.g., grader uses `subprocess.run` against an artifact path) | human + agent | design | med |
| Model matrix scope | 4 models × 2 arms × 50 tasks × 3 attempts ≈ 1,200 attempts/run. Adding a 5th model is +25%. | human | design | med |
| TB task image hosting | Mirror upstream images to our Artifact Registry, or fetch upstream on every run | agent | design | low |
| AILANG runtime injection mechanism | Layer `ailang` binary into the TB container via init container, or build a per-task derivative image | agent | compile | med |
| Cost ceiling per run | Without a cap, a full matrix could spend unbounded on retries | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Leaderboard-target vs eval-substrate framing (drives whether AILANG arm follows TB's exact protocol)
- [ ] Cost ceiling per matrix run (dollar value)
- [ ] Initial model matrix (e.g., claude-sonnet-4.6, gemini-3-flash, deepseek-v4-flash, gpt-5-mini)

---

## Solution Design

### Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        TERMINAL-BENCH CLOUD EVAL                              │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ailang eval-tb run --matrix tb-50.yml --models claude,gemini,deepseek      │
│              │                                                               │
│              ▼                                                               │
│   ┌──────────────────────┐                                                   │
│   │ TB matrix planner    │  expands → (task, model, arm, attempt) tuples     │
│   └──────────┬───────────┘                                                   │
│              │                                                               │
│              ▼                                                               │
│   ┌──────────────────────┐                                                   │
│   │ Cloud Run Jobs       │  one job execution per tuple                      │
│   │ (multivac infra)     │  ─ pulls TB task image                            │
│   │                      │  ─ injects ailang binary (AILANG arm only)        │
│   │                      │  ─ runs executor (claude/motoko) inside container │
│   │                      │  ─ TB grader produces verdict                     │
│   │                      │  ─ publishes verdict + cost + tokens              │
│   └──────────┬───────────┘                                                   │
│              │                                                               │
│              ▼                                                               │
│   ┌──────────────────────┐                                                   │
│   │ Eval results store   │  Firestore + GCS (existing AILANG infra)          │
│   └──────────┬───────────┘                                                   │
│              │                                                               │
│              ▼                                                               │
│   ┌──────────────────────┐                                                   │
│   │ Dashboard report     │  per-task verdict table, arm-delta, cost          │
│   └──────────────────────┘                                                   │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Architecture

**Components:**

1. **TB task catalog** (`internal/eval_harness/tb_catalog.go`)
   Inspects upstream Terminal-Bench repo, filters to "build-a-program" subset using a predicate function (see Deferred Decisions), and produces `tb-NN.yml` matrix definitions. Catalog is regenerated, not hand-curated; the predicate is the source of truth.

2. **TB matrix planner** (`internal/eval_harness/tb_matrix.go`)
   Expands a matrix YAML into the cross-product of (task, model, arm, attempt) tuples. Emits dispatch records to Pub/Sub for Cloud Run Job consumption. Mirrors existing `eval-suite` dispatch pattern.

3. **TB Cloud Run Job template** (`ailang-multivac/terraform/cloud_run_jobs_tb.tf`)
   New job template based on existing `agent_executor`, with these differences:
   - Base image is the TB task image (passed as `TB_TASK_IMAGE` override per execution)
   - AILANG arm: init step injects `ailang` binary + stdlib into `/usr/local/bin/`
   - Python arm: container as-shipped, no modifications
   - Same OAuth/API-key auth split as existing `agent_executor`

4. **Executor adapter** (`internal/eval_harness/tb_adapter.go`)
   Translates between AILANG's executor protocol (claude / motoko / etc.) and Terminal-Bench's expectation that the agent issues shell commands. The adapter prompts the model with the TB task description plus an arm-specific preamble:
   - Python arm: "Write Python. The grader will invoke your solution as described."
   - AILANG arm: "Write AILANG. Run with `ailang run --caps <caps> file.ail`. Declare all effects explicitly."

5. **Verdict capture** (`internal/eval_harness/tb_verdict.go`)
   Runs the TB grader unchanged, captures pass/fail + grader stdout/stderr + any artifacts the grader inspects. Persists to existing eval results schema with two new fields: `tb_task_id`, `arm` (python|ailang).

6. **Dashboard view** (`ui/dashboard/tb-results.tsx`)
   Per-task table: task / model / Python verdict / AILANG verdict / arm-delta. Summary row: aggregate pass rate per arm. Cost breakdown per arm.

### Implementation Plan

**Phase 1: Catalog + Matrix (~12 hours)**
- [ ] Clone Terminal-Bench upstream into a vendored fixtures path (or pin a release tag)
- [ ] Implement build-a-program predicate (see Deferred Decisions for candidates)
- [ ] Generate first `tb-50.yml` matrix file
- [ ] Unit tests on predicate against ≥20 known tasks (hand-labeled)

**Phase 2: Cloud Run Job Plumbing (~20 hours)**
- [ ] New job template in multivac terraform (`cloud_run_jobs_tb.tf`)
- [ ] AILANG-injection init step (sidecar or single-image — agent decides)
- [ ] Pub/Sub dispatch topic for TB tuples (or reuse existing if back-pressure tolerates)
- [ ] End-to-end smoke test: 1 task × 1 model × 2 arms

**Phase 3: Executor Adapter (~16 hours)**
- [ ] Arm-specific preamble injection in `agent_prompt.go`
- [ ] AILANG-arm prompt enforces `--caps` declaration + module syntax (cite `ailang prompt`)
- [ ] TB grader invocation wrapper
- [ ] Verdict normalization (TB → AILANG eval schema)

**Phase 4: Results + Dashboard (~12 hours)**
- [ ] Schema migration for `tb_task_id` + `arm` fields
- [ ] Dashboard TB view (per-task table + arm-delta)
- [ ] CSV/JSON export from `ailang eval-tb export`

**Phase 5: First Real Run + Report (~10 hours)**
- [ ] 50 tasks × 4 models × 2 arms × 3 attempts on Cloud Run Jobs
- [ ] Cost reconciliation against ceiling
- [ ] Public report doc under `docs/docs/benchmarks/terminal-bench.md`

### Files to Modify/Create

**New files:**
- `internal/eval_harness/tb_catalog.go` — TB task catalog + predicate (~250 LOC)
- `internal/eval_harness/tb_matrix.go` — matrix expansion + dispatch (~200 LOC)
- `internal/eval_harness/tb_adapter.go` — executor ↔ TB harness shim (~300 LOC)
- `internal/eval_harness/tb_verdict.go` — verdict capture + normalization (~150 LOC)
- `cmd/ailang/eval_tb.go` — CLI: `ailang eval-tb run|export|status` (~200 LOC)
- `ailang-multivac/terraform/cloud_run_jobs_tb.tf` — TB job template (~150 LOC)
- `ailang-multivac/docker/tb-injector/` — AILANG injector layer (Dockerfile + entrypoint, ~80 LOC)
- `ui/dashboard/tb-results.tsx` — dashboard view (~300 LOC)
- `docs/docs/benchmarks/terminal-bench.md` — public results doc (~200 LOC)

**Modified files:**
- `internal/eval_harness/agent_prompt.go` — arm-specific preamble (~50 LOC)
- `internal/eval_harness/models.go` — track TB-eligible flag per model (~20 LOC)
- `internal/eval_harness/metrics.go` — add `tb_task_id`, `arm` fields (~30 LOC)
- `cmd/ailang/eval.go` — register `eval-tb` subcommand (~10 LOC)

---

## Examples

### Example 1: Single task, both arms

**Command:**
```bash
ailang eval-tb run \
  --task tb:build-csv-summarizer \
  --models claude-sonnet-4-6 \
  --arms python,ailang \
  --attempts 3
```

**Behavior:**
- 6 Cloud Run Job executions dispatched (3 attempts × 2 arms)
- Python arm: model writes `summarize.py`, grader runs `python summarize.py <fixture>`
- AILANG arm: model writes `summarize.ail`, grader runs `ailang run --caps IO,FS --entry main summarize.ail`
- Both arms graded by **the same TB grader script unchanged**

**Result (JSON, abridged):**
```json
{
  "task_id": "tb:build-csv-summarizer",
  "model": "claude-sonnet-4-6",
  "python":  {"pass_rate": 0.67, "avg_cost_usd": 0.012, "avg_tokens": 4_200},
  "ailang":  {"pass_rate": 1.00, "avg_cost_usd": 0.011, "avg_tokens": 3_800},
  "delta":   {"pass_rate": +0.33, "verdict": "ailang_better"}
}
```

### Example 2: Full matrix run

**Command:**
```bash
ailang eval-tb run --matrix tb-50.yml --cost-ceiling 25.00
```

**Behavior:**
- Planner expands to 1,200 tuples
- Pub/Sub dispatches to Cloud Run Jobs (concurrency capped by job template)
- Real-time progress in dashboard
- Auto-aborts if running cost exceeds `--cost-ceiling`
- On completion, generates `tb-results-<timestamp>.md` summary

---

## Success Criteria

- [ ] `ailang eval-tb run --task <id> --arms python,ailang` works end-to-end on Cloud Run Jobs without any local Docker
- [ ] At least 50 TB tasks classified as build-a-program by the predicate
- [ ] Predicate has zero false-negatives on a hand-labeled set of 20 known build-a-program tasks
- [ ] Verdicts land in existing eval results store with `tb_task_id` + `arm` fields
- [ ] Dashboard shows per-task arm-delta and aggregate pass-rate-by-arm
- [ ] First public report published under `docs/docs/benchmarks/terminal-bench.md`
- [ ] Cost ceiling enforced (run aborts cleanly when exceeded)
- [ ] All tests passing (`make test`)
- [ ] CLI documented in `ailang prompt` and `ailang eval-tb --help`

---

## Testing Strategy

**Unit tests:**
- TB predicate against a corpus of hand-labeled tasks (20+ positive, 20+ negative)
- Matrix expansion: cross-product correctness, dedup, attempt counting
- Verdict normalization: TB grader exit codes → AILANG eval schema
- Arm-specific preamble: AILANG arm preamble contains `--caps` and module syntax cue; Python arm does not

**Integration tests:**
- One real TB task end-to-end on Cloud Run Jobs (gated by `AILANG_INTEGRATION_GCP=1`)
- Pub/Sub dispatch + Cloud Run Job consumption round-trip
- Cost-ceiling enforcement (synthetic dispatch loop)

**Manual testing:**
- First 5-task smoke run, eyeball the dashboard view
- Compare AILANG-arm artifact code against `ailang prompt` for syntactic validity

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Build-a-program predicate definition** — agent may choose between (a) static analysis of grader scripts for `subprocess` invocations against an artifact path, (b) presence of an `EXPECTED_*` env var pointing at agent-written file, (c) explicit upstream task tag if/when Terminal-Bench adds one. Predicate must be testable.
- **AILANG runtime injection mechanism** — agent may choose between init-container layering, a per-task derivative image baked into Artifact Registry, or runtime download from a signed URL. Constraint: TB task images must not be modified upstream-source-of-truth.
- **Image hosting** — agent may choose between mirroring TB upstream images to our Artifact Registry (faster cold start, storage cost) or pulling per-execution (always fresh, slower). Default to mirror with monthly refresh unless storage budget pushes back.
- **Pub/Sub topic split** — agent may choose to reuse existing `agent-tasks` topic or create dedicated `tb-tasks`. Reuse if back-pressure on existing topic is acceptable.
- **Attempt scheduling** — agent may choose sequential or parallel attempts within a tuple. Default: parallel (faster wall-clock), but parallel can mask flakiness.

## Non-Goals

**Not attempted in this feature:**
- **Operate-the-terminal tasks** — TB tasks that require shell debugging (no language choice) are out of scope; tracked separately if we ever want to benchmark our agent harness directly.
- **Submitting to the public TB leaderboard** — depends on the leaderboard-target framing decision; if greenlit, becomes a follow-up doc, not part of this milestone.
- **AILANG-side TB grader rewriting** — TB grading scripts run unchanged. We do not "improve" their graders.
- **Multi-file TB tasks** — initial scope is single-artifact builds; multi-file is a follow-up once single-artifact arm-delta is established.
- **GPU model runtime** — TB Cloud Run Jobs use CPU; GPU model serving is M-CLOUD-EVAL territory.

---

## Timeline

**Week 1** (~32 hours):
- Phase 1: Catalog + Matrix
- Phase 2: Cloud Run Job plumbing (terraform + smoke test)

**Week 2** (~28 hours):
- Phase 3: Executor adapter
- Phase 4 partial: schema migration + dashboard scaffold

**Week 3** (~20 hours):
- Phase 4 finish: dashboard view
- Phase 5: first real matrix run + cost reconciliation

**Week 4** (~10 hours):
- Public report doc
- Buffer for cost ceiling tuning, dashboard polish, dependent docs

**Total: ~90 hours across 4 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Predicate misclassifies tasks; AILANG arm fails because task isn't actually a build-a-program task | High | Hand-label first 50, predicate must agree with labels before matrix run. Audit failures in pilot run. |
| TB upstream changes task format mid-run | Med | Pin to a specific TB release tag in vendor manifest; bump deliberately. |
| Cost overrun on first matrix run | High | Cost-ceiling flag with hard abort; start with 5-task run before 50-task run. |
| AILANG-arm preamble too prescriptive → measures prompt engineering rather than language | High | Keep arm preambles minimal: language name + canonical invocation only. No task-specific hints in either arm. |
| AILANG binary injection breaks TB task assumptions about container contents | Med | Inject under `/opt/ailang/` (non-FHS path), only add to PATH, don't overwrite anything. |
| Same-model variance dominates the AILANG-vs-Python signal | High | Run ≥3 attempts per tuple, report variance not just means; require delta > 2σ before claiming "ailang_better". |
| TB image pull rate-limits us | Low | Mirror to our Artifact Registry. |
| Skeptics dismiss results because we cherry-picked tasks | Med | Publish predicate + hand-labels alongside results; allow third party to re-run with their own subset. |

---

## Conflict Surface

This design **does not touch parser/lexer/AST/types/elaborate/iface/codegen/eval/vm/effects/exec.go**. It is a pure benchmarking + infrastructure addition.

The closest internal contract surface is `internal/eval_harness/`:
- New fields on result records (`tb_task_id`, `arm`) — additive only, no breaking changes to existing eval-suite consumers
- New CLI subcommand (`eval-tb`) — disjoint from existing `eval-suite` namespace
- New Pub/Sub topic (or namespace within existing) — disjoint from existing executor dispatch

**Existing programs that MUST still work post-change:**
1. `ailang eval-suite run --tier core` — unchanged
2. `ailang eval-suite run --agent --benchmarks ...` — unchanged
3. Existing dashboard views (cross-language matrix, by-harness, by-version) — unchanged; TB view is additive
4. Existing Cloud Run `agent_executor` job — unchanged; TB job is a new template alongside it

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_15_0/m-eval-lang-jsgo-sprint-plan.md](../../implemented/v0_15_0/m-eval-lang-jsgo-sprint-plan.md) — cross-language matrix pattern; this doc extends the pattern to a third-party benchmark.
- [design_docs/implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md](../../implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md) — local eval; complemented by cloud-side execution here.

**Planned (relevant adjacencies):**
- [design_docs/planned/v0_13_0/m-locobench-long-context-benchmark.md](../v0_13_0/m-locobench-long-context-benchmark.md) — same AILANG-vs-Python integration pattern, different upstream benchmark (LoCoBench is long-context SWE, TB is agentic terminal). Shares schema and dispatch infrastructure.
- [design_docs/planned/v0_13_0/m-cloud-eval-workers.md](../v0_13_0/m-cloud-eval-workers.md) — distributed cloud eval workers; this doc reuses the same Cloud Run Jobs substrate.
- [design_docs/planned/v0_17_0/m-bench-motoko-executor.md](../v0_17_0/m-bench-motoko-executor.md) — benchmark Motoko executor; TB will be runnable under either claude or motoko executor.

## References

- [Design Axioms](/docs/references/axioms) — 12 non-negotiable principles
- [Terminal-Bench upstream](https://www.tbench.ai/) — benchmark home + leaderboard
- [Multivac Cloud Run Jobs config](../../../../ailang-multivac/terraform/cloud_run_jobs.tf) — existing job-per-task infrastructure
- [`ailang prompt`](../../../prompts/) — canonical AILANG syntax used in AILANG-arm preamble

## Future Work

- **Operate-the-terminal subset** — for tasks where the agent must debug a service rather than write a program, AILANG has no surface area; but our agent harness (motoko executor) could be benchmarked there as a separate exercise.
- **Submit to public leaderboard** — once the eval-substrate run is stable, follow upstream submission rules to publish AILANG-arm scores.
- **TB v3+ migration plan** — pin to current TB release; upgrade as a tracked sub-task with regression check on the predicate corpus.
- **Multi-file TB tasks** — natural follow-up once single-artifact arm-delta is established.

---

**Document created**: 2026-05-19
**Last updated**: 2026-05-19

DESIGN_DOC_PATH: design_docs/planned/v0_22_0/m-terminal-bench-cloud-eval.md
