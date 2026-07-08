# M-TRACE-FEEDBACK: Execution-Trace Feedback Loop for Harness Diagnostics

**Status**: Planned
**Target**: v0.23.0
**Priority**: P1 - Medium
**Estimated**: 1 week
**Dependencies**: OTEL trace pipeline (already shipped), `ailang dashboard` (v0.12+)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Reads deterministic OTEL traces; diagnostic classification is pure/reproducible |
| A2: Replayability | +1 | Traces are the replay substrate; this feature surfaces them |
| A3: Effect Legibility | +1 | Makes hidden latency, retries, and tool costs visible as typed failures |
| A4: Explicit Authority | 0 | Read-only pipeline in v1; no new authority granted |
| A5: Bounded Verification | +1 | Failure taxonomy is locally decidable per trace |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured JSON diagnostic report, machine-parseable |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Surfaces token costs, latency, and retry overhead per failure class |
| A10: Composability | +1 | Composes with `ailang dashboard` and future Evolution Agent |
| A11: Structured Failure | +1 | Failure taxonomy is typed; each class maps to a defined remediation |
| A12: System Boundary | +1 | Explicit boundary between observation (traces) and prescription (report) |

**Net Score: +9** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — classifier reads existing trace data
- [x] A3 (Effects): No hidden side effects — read-only pipeline
- [x] A4 (Authority): No ambient access — reads from trace store only
- [x] A7 (Machines First): Output is machine-first JSON, not human narrative

## Problem Statement

AILANG's OTEL trace pipeline captures rich telemetry per coordinator chain: prompts, token usage, latency per stage, tool call arguments, sandbox snapshots, test pass/fail outcomes, and error messages. This data exists and is queryable via `ailang trace list`.

**Current State:**
- Telemetry is visible in `ailang dashboard` and `ailang trace list` — humans can scroll and notice patterns
- No automated classification of failure modes: a human must read multiple traces and infer "this class of tasks keeps timing out on the verify stage"
- No prescription layer: even after a human identifies a pattern, there is no structured handoff to a harness fix
- The gap from observation → diagnosis → fix is entirely manual

**Impact:**
- Harness regressions (brittle tools, weak validators, wrong retry policies) are caught late — after multiple failed chains
- No feedback loop from eval failures back to the harness components that caused them
- "Code as Agent Harness" (arXiv:2605.18747) identifies this exact loop — "deep telemetry as optimization substrate" — as cutting-edge practice. Production harnesses (Cursor Composer, OpenAI Codex) are using trace data as training signal. AILANG has the data; the pipeline is missing.

## Goals

**Primary Goal:** Build a pipeline that ingests AILANG OTEL traces, classifies failure modes by a defined taxonomy, and produces a structured "harness diagnostic report" that names the failing harness component and suggests a remediation.

**Success Metrics:**
- `ailang trace diagnose` command produces a diagnostic report in <5s for up to 1000 traces
- Diagnostic report correctly classifies ≥80% of known failure patterns (validated against labeled test set)
- Report format is machine-readable JSON + human-readable Markdown
- At least one harness fix is identified and acted on within one sprint of enabling this feature

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Failure taxonomy: 4 classes or extensible registry? | Fixed taxonomy is simpler but may not cover new failure types | human | design | high |
| Classification: rule-based or LLM-assisted? | LLM adds AI cost and non-determinism; rules are auditable | human | design | high |
| Output target: `ailang dashboard` integration or separate command? | Determines UX and coupling | agent | compile | med |
| Prescription format: free-text or typed remediation struct? | Typed enables Evolution Agent automation later | human | design | med |

### Design Freeze

Before implementation begins:

- [ ] Failure taxonomy finalized (4 canonical classes below, or extended set)
- [ ] Classification strategy: rule-based (recommended) vs LLM-assisted

## Solution Design

### Overview

A three-stage pipeline:

```
ailang trace list --hours N
    ↓  [ingest]
TraceClassifier  →  FailureClass × HarnessComponent × Evidence
    ↓  [route]
DiagnosticReport (JSON + MD)
    ↓  [future: Evolution Agent]
HarnessPatch proposal
```

### Architecture

**Components:**

1. **TraceIngestor** (`internal/diagnostics/ingestor.go`): Reads spans from the trace store via the existing `ailang trace` query API. Filters to chains with `status=failed` or `duration > threshold`. Groups by coordinator task type.

2. **FailureClassifier** (`internal/diagnostics/classifier.go`): Rule-based classification against the 4-class taxonomy (see below). Each rule is a predicate over span attributes. Returns `FailureClass`, the `HarnessComponent` implicated, and supporting evidence spans.

3. **DiagnosticReportWriter** (`internal/diagnostics/report.go`): Renders the classified failures into a structured JSON report and a human-readable Markdown summary. Groups by failure class, ranks by frequency.

4. **CLI command** (`cmd/ailang/trace_diagnose.go`): `ailang trace diagnose [--hours N] [--output json|md]`

**Failure Taxonomy (4 canonical classes):**

| Class | Signature | Implicated Component | Remediation Direction |
|-------|-----------|---------------------|----------------------|
| `MISSING_CONTEXT` | Executor asked for file/symbol not in context; hallucinated content | Coordinator context-packing | Widen context window or add repo-index retrieval |
| `BRITTLE_TOOL` | Tool call succeeded but output caused downstream failure; retry loop > 2 | Tool adapter | Tighten tool output schema or add output validation |
| `WEAK_VALIDATOR` | Tests passed but chain marked failed by human review; or test suite coverage < threshold | Eval harness / oracle | Add property tests or integration test layer |
| `WRONG_RETRY_POLICY` | Chain retried >3× with identical prompt/context; no convergence | Coordinator retry logic | Add backoff, context mutation, or human escalation gate |

### Implementation Plan

**Phase 1: Ingestor + Classifier** (~2 days)
- [ ] `internal/diagnostics/ingestor.go` — query trace store, return `[]Span` per chain
- [ ] `internal/diagnostics/classifier.go` — 4-class rule engine
- [ ] Unit tests: labeled fixture traces for each failure class
- [ ] `make test` passes

**Phase 2: Report Writer + CLI** (~1.5 days)
- [ ] `internal/diagnostics/report.go` — JSON + Markdown renderer
- [ ] `cmd/ailang/trace_diagnose.go` — CLI wiring
- [ ] Integration test: run against real trace store in CI with known-bad chains
- [ ] `ailang trace diagnose --help` works

**Phase 3: Dashboard Integration** (~1.5 days)
- [ ] Add "Diagnostics" tab to `ailang dashboard` showing latest report
- [ ] Link each failure entry to its raw trace via `ailang trace show SPAN_ID`
- [ ] Documentation update: `docs/docs/guides/telemetry.md`

### Files to Modify/Create

**New files:**
- `internal/diagnostics/ingestor.go` — trace ingest (~150 LOC)
- `internal/diagnostics/classifier.go` — failure taxonomy rules (~200 LOC)
- `internal/diagnostics/report.go` — JSON/MD rendering (~120 LOC)
- `internal/diagnostics/classifier_test.go` — fixture-based unit tests (~150 LOC)
- `cmd/ailang/trace_diagnose.go` — CLI command (~80 LOC)

**Modified files:**
- `cmd/ailang/main.go` — register `trace diagnose` subcommand (~10 LOC)
- `internal/dashboard/` — add Diagnostics tab (~100 LOC)
- `docs/docs/guides/telemetry.md` — document `trace diagnose` command (~30 LOC)

## Examples

### Example 1: Running the Diagnostic

```bash
$ ailang trace diagnose --hours 24

Harness Diagnostic Report — last 24h (2026-05-21)
Chains analyzed: 47  |  Failed: 12  |  Classified: 11/12

MISSING_CONTEXT (5 chains, 42%)
  Component: Coordinator context-packing
  Evidence:  executor asked for internal/types/checker.go in 5 chains; file not in context window
  Remedy:    Add internal/types/ to default context for type-checker tasks

BRITTLE_TOOL (4 chains, 33%)
  Component: gopls tool adapter
  Evidence:  gopls returned 0 references for 3 renamed symbols; executor accepted silently
  Remedy:    Add reference-count validation to gopls tool output schema

WRONG_RETRY_POLICY (2 chains, 17%)
  Component: Coordinator retry logic
  Evidence:  same prompt retried 4× without context change on 2 benchmark tasks
  Remedy:    Add context-mutation step (inject failing test output) before retry 2+

Unclassified: 1 chain (chain_id: abc123) — inspect with: ailang trace show abc123
```

### Example 2: JSON Output for Automation

```bash
$ ailang trace diagnose --hours 24 --output json | jq '.failures[0]'
{
  "class": "MISSING_CONTEXT",
  "count": 5,
  "component": "coordinator/context-packing",
  "evidence_span_ids": ["span_abc", "span_def"],
  "remedy": "Widen context to include internal/types/ for type-checker tasks",
  "chains": ["chain_001", "chain_003", "chain_007", "chain_012", "chain_019"]
}
```

## Success Criteria

- [ ] `ailang trace diagnose` produces a report in <5s for 1000 traces
- [ ] All 4 failure classes correctly identified in labeled fixture test set (≥80% accuracy)
- [ ] JSON output is valid and schema-stable across runs (deterministic for same input)
- [ ] Markdown report is human-readable without post-processing
- [ ] Dashboard "Diagnostics" tab shows latest report
- [ ] At least one real harness fix identified from first production run
- [ ] All tests passing (`make test`)
- [ ] Documentation updated (`docs/docs/guides/telemetry.md`)

## Testing Strategy

**Unit tests:**
- One labeled fixture trace per failure class (stored in `internal/diagnostics/testdata/`)
- Classifier correctly categorizes each fixture
- Report writer produces valid JSON for each class

**Integration tests:**
- Run `ailang trace diagnose` against CI trace store after a known-bad eval run
- Verify at least one `BRITTLE_TOOL` or `MISSING_CONTEXT` classification

**Manual testing:**
- Run against 24h of production traces, review report for plausibility
- Confirm each `remedy` field suggests a real, actionable change

## Deferred Decisions

- LLM-assisted classification for the `Unclassified` bucket — agent may add after v1 ships
- Prescription → auto-PR generation (Evolution Agent) — deferred to future milestone, see [Future Work](#future-work)
- Threshold tuning for `WRONG_RETRY_POLICY` (currently: >3 retries) — agent may adjust based on production data

## Non-Goals

- **Auto-applying remediations** — this doc is observation + prescription only; applying is the Evolution Agent (future)
- **LLM-based classification in v1** — rule-based is auditable and deterministic; LLM is deferred
- **Real-time streaming diagnostics** — batch report over a time window is sufficient

## Timeline

**Week 1** (~5 days):
- Phase 1: Ingestor + Classifier (days 1–2)
- Phase 2: Report Writer + CLI (days 3–4)
- Phase 3: Dashboard integration (day 5)
- Documentation + tests throughout

**Total: ~5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Failure taxonomy doesn't cover real production failures | Med | Start with `Unclassified` bucket; promote patterns to named classes after first run |
| Trace store query performance at 1000+ chains | Med | Add `LIMIT` and time-window filters; index on `status` and `timestamp` |
| Classification accuracy < 80% | Med | Collect labeled examples from first production run; tune rules before v2 |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_1/trace-test.md](design_docs/implemented/v0_7_1/trace-test.md) — original trace system design
- [design_docs/implemented/v0_11_1/m-wasm-trace.md](design_docs/implemented/v0_11_1/m-wasm-trace.md) — trace integration patterns

**Planned (same cluster):**
- [design_docs/planned/v0_23_0/m-oracle-adequacy.md](design_docs/planned/v0_23_0/m-oracle-adequacy.md) — Doc 5: evidence bundles improve `WEAK_VALIDATOR` taxonomy
- [design_docs/planned/v0_23_0/m-harness-state.md](design_docs/planned/v0_23_0/m-harness-state.md) — Doc 3: shared state enables `MISSING_CONTEXT` detection

## References

- **Ning et al. (2026).** Code as Agent Harness. arXiv:[2605.18747](https://arxiv.org/abs/2605.18747) — §AHE "deep telemetry as optimization substrate"; §sec_4 "production harnesses as training data"
- [Design Axioms](/docs/references/axioms)
- [Telemetry Guide](../../../docs/docs/guides/telemetry.md)

## Future Work

- **Evolution Agent**: reads diagnostic reports and proposes harness patches as GitHub PRs — this doc is the prerequisite
- **Training data pipeline**: route classified traces to fine-tuning dataset (paper's §sec_4 finding: "harnesses are the dominant source of next-gen training data")
- **Per-model diagnostic breakdown**: classify failures by model/executor camp

---

**Document created**: 2026-05-21
**Last updated**: 2026-05-21
