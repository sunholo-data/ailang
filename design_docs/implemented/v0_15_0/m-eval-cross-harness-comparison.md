# M-EVAL-CROSS-HARNESS: Cross-Harness Benchmark Comparison

**Status**: Implemented (v0.15.0)
**Target**: v0.15.x
**Priority**: P1 (Medium — strategic eval insight, not a release blocker)
**Estimated**: 2 days
**Dependencies**: opencode executor (`internal/executor/opencode/`), `agent_suite` in models.yml

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval infrastructure only; no language semantics changes |
| A2: Replayability | +1 | Harness is an explicit eval dimension; paired runs are reproducible |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | Each harness is a named executor; no new capabilities |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Reveals which harness wrapping best serves machine-readable AILANG; direct thesis relevance |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Harness cost-per-run becomes a first-class metric; opencode vs claude cost delta visible |
| A10: Composability | +1 | Harness is just another dimension in models.yml, composes with existing model/benchmark/language axes |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No new boundary crossings |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Reveals harness quality signal for machine audience

---

## Problem Statement

**Current State:**

The eval harness treats each `models.yml` entry as an independent data point. `claude-sonnet-4-6` (via the `claude` CLI) and `opencode-haiku` (via `opencode`) are presented side by side in `eval-matrix` without any grouping by underlying model identity. This makes it impossible to answer the question: **"Does harness wrapping change benchmark outcomes for the same model?"**

This matters because:
- `opencode` wraps Anthropic/Google models through its own prompt template, tool schema format, and context management layer
- `claude` CLI talks directly to the Anthropic API with Claude Code's native tool-use protocol
- The same `claude-sonnet-4-6` weights may produce different benchmark scores depending on which harness calls them, due to system prompt differences, tool round-trip overhead, and context window handling

We currently have no data on this delta, and no infrastructure to collect it.

**Concrete harness pairs that can be tested today:**

| Model family | Harness A | models.yml key | Harness B | models.yml key |
|---|---|---|---|---|
| claude-sonnet-4-6 | `claude` CLI | `claude-sonnet-4-6` | `opencode` | `opencode-sonnet-4-6` (new) |
| gemini-3-flash | `gemini` CLI | `gemini-3-flash` | `opencode` | `opencode-gemini-3-flash` (new) |
| claude-haiku-4-5 | `claude` CLI | `claude-haiku-4-5` | `opencode` | `opencode-haiku` (exists) |

**Impact:**
- We don't know if opencode's system prompt wrapping helps or hurts benchmark performance
- `eval-matrix` conflates harness and model, making cross-harness comparison manual and error-prone
- Harness selection (which CLI to use for production agent tasks) is currently based on intuition rather than data

---

## Goals

**Primary Goal:** Run the same models through multiple harnesses and surface harness-induced benchmark deltas as a first-class eval dimension.

**Success Metrics:**
- `opencode-sonnet-4-6` and `opencode-gemini-3-flash` entries added to `models.yml`
- `claude-sonnet-4-6` vs `opencode-sonnet-4-6` paired results available for ≥5 benchmarks
- `eval-matrix` groups results by model family (showing harnesses as sub-rows) when a `--group-by model-family` flag is passed
- Harness overhead (duration delta, cost delta) recorded as `harness_overhead_ms` in result JSON
- A `harness_suite` composite in models.yml containing both harness pairs for one-command cross-harness runs

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `model_family` field in ModelConfig vs inferred from `api_name` | Determines how `eval-matrix` groups harness pairs; inferred is brittle if api_names diverge | human | design | med |
| `harness_suite` composite scope — one pair or all pairs | Broader scope = more data but higher cost; one pair (sonnet) is cheapest first validation | human | design | low |
| Grouping in `eval-matrix` — new flag vs default behaviour | Always-on grouping changes existing output format; flag is safer | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **`model_family` field**: Add explicit `model_family: "claude-sonnet-4-6"` to ModelConfig YAML, OR infer from `api_name` (simpler but fragile). Recommendation: explicit field.
- [ ] **`eval-matrix` grouping**: New `--group-by model-family` flag (default off) vs always group when `model_family` matches. Recommendation: flag, default off.

---

## Solution Design

### Overview

Three additive changes, no breaking modifications:

1. **`models.yml`**: Add `opencode-sonnet-4-6` and `opencode-gemini-3-flash` entries + optional `model_family` field + `harness_suite` composite
2. **`ModelConfig`**: Add `ModelFamily string` field parsed from `model_family` YAML key
3. **`eval-matrix`**: Add `--group-by model-family` flag that clusters matching families and shows harness delta rows

### Architecture

```
models.yml
  claude-sonnet-4-6:
    model_family: "claude-sonnet-4-6"
    agent_cli: "claude"
    ...
  opencode-sonnet-4-6:          ← new
    model_family: "claude-sonnet-4-6"   ← same family
    agent_cli: "opencode"
    agent_model_name: "anthropic/claude-sonnet-4-6"
    ...

harness_suite:                  ← new composite
  - "claude-sonnet-4-6"
  - "opencode-sonnet-4-6"
  - "gemini-3-flash"
  - "opencode-gemini-3-flash"

eval-matrix --group-by model-family
  ┌─────────────────────────────────────────────────────┐
  │ claude-sonnet-4-6 (family)                          │
  │   claude harness:   fizzbuzz ✓  fib ✓  sort ✓      │
  │   opencode harness: fizzbuzz ✓  fib ✗  sort ✓      │
  │   Δ harness: -1 pass, +450ms, -$0.002               │
  └─────────────────────────────────────────────────────┘
```

### Components

1. **`ModelConfig.ModelFamily`** — new optional string field. When two models share the same `model_family`, `eval-matrix` can pair their results. Zero value = no grouping.

2. **New `models.yml` entries** — `opencode-sonnet-4-6` and `opencode-gemini-3-flash`. Pattern follows existing `opencode-haiku` entry. `agent_model_name` uses opencode's provider-prefixed format (`anthropic/claude-sonnet-4-6`, `google/gemini-2.5-flash`).

3. **`harness_suite` composite** — like `agent_suite` and `ollama_suite`, expands to all cross-harness pairs. Allows `ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz` to run all pairs in one command.

4. **`eval-matrix --group-by model-family`** — reads `ModelFamily` from result JSON metadata, clusters rows by family, adds a delta row showing pass-rate diff, duration diff, cost diff between harnesses.

5. **`harness_overhead_ms` in result JSON** — already available as `DurationMS` per result; the grouping logic computes delta, no new fields needed in `AgentBenchmarkResult`.

### Implementation Plan

**Phase 1: models.yml + ModelConfig** (~2 hours, ~30 LOC)
- [ ] Add `ModelFamily string` to `ModelConfig` struct (`internal/eval_harness/models.go`)
- [ ] Add `model_family` YAML tag
- [ ] Add `opencode-sonnet-4-6` entry to `models.yml` (follows `opencode-haiku` pattern)
- [ ] Add `opencode-gemini-3-flash` entry to `models.yml`
- [ ] Add `model_family` to existing `claude-sonnet-4-6`, `gemini-3-flash`, `claude-haiku-4-5`, `opencode-haiku` entries
- [ ] Add `harness_suite` composite and wire `expandModelSuite` in `eval_suite.go`
- [ ] Tests: verify `ModelFamily` parses, `harness_suite` expands correctly

**Phase 2: eval-matrix grouping** (~3 hours, ~80 LOC)
- [ ] Add `--group-by` flag to `ailang eval-matrix` command
- [ ] When `--group-by model-family`: load `ModelFamily` from result JSON metadata (need to write it there first)
- [ ] Cluster results by family, render grouped table with delta row
- [ ] Write `model_family` into result JSON at save time (add field to `AgentBenchmarkResult` or metadata)
- [ ] Tests: grouped output formatting

**Phase 3: end-to-end verification** (~2 hours)
- [ ] Run `ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz,sort_list --agent-parallel 2`
- [ ] Verify paired results appear in `eval-matrix --group-by model-family`
- [ ] Document delta findings in design doc

### Files to Modify/Create

**Modified files:**
- `internal/eval_harness/models.go` — add `ModelFamily string` to `ModelConfig` (~5 LOC)
- `internal/eval_harness/models.yml` — add 2 new model entries + `model_family` on 6 existing entries + `harness_suite` (~30 LOC)
- `internal/eval_harness/agent_runner.go` — add `ModelFamily string` to `AgentBenchmarkResult` (~3 LOC)
- `internal/eval_harness/metrics.go` — write `ModelFamily` into saved JSON (~5 LOC)
- `cmd/ailang/eval_suite.go` — wire `harness_suite` in `expandModelSuite` (~5 LOC)
- `cmd/ailang/eval_matrix.go` — add `--group-by` flag + grouped rendering (~80 LOC)

---

## Examples

### Example 1: Running a cross-harness comparison

```bash
# Run all cross-harness pairs against fizzbuzz
ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz --agent-parallel 2

# View grouped results
ailang eval-matrix eval_results/ --group-by model-family
```

**Output:**
```
Model Family: claude-sonnet-4-6
  Harness          | fizzbuzz/py | fizzbuzz/ail | sort/py | Avg   | Cost    | Duration
  claude           |     ✓       |      ✓       |    ✓    | 100%  | $0.012  | 45s
  opencode         |     ✓       |      ✗       |    ✓    |  67%  | $0.009  | 78s
  Δ (opencode-claude) |   0     |     -1       |    0    | -33%  | -$0.003 | +33s

Model Family: gemini-3-flash
  Harness          | fizzbuzz/py | fizzbuzz/ail | sort/py | Avg   | Cost    | Duration
  gemini           |     ✓       |      ✓       |    ✗    |  67%  | $0.001  | 38s
  opencode         |     ✓       |      ✓       |    ✓    | 100%  | $0.001  | 52s
  Δ (opencode-gemini) |   0     |      0       |   +1    | +33%  | $0.000  | +14s
```

### Example 2: Adding opencode-sonnet-4-6 to models.yml

```yaml
opencode-sonnet-4-6:
  api_name: "claude-sonnet-4-6"
  model_family: "claude-sonnet-4-6"
  provider: "anthropic"
  description: "Claude Sonnet 4.6 via opencode harness — cross-harness comparison pair"
  env_var: "ANTHROPIC_API_KEY"
  agent_cli: "opencode"
  agent_model_name: "anthropic/claude-sonnet-4-6"
  max_output_tokens: 8192
  pricing:
    input_per_1k: 0.003
    output_per_1k: 0.015
```

---

## Success Criteria

- [ ] `opencode-sonnet-4-6` and `opencode-gemini-3-flash` in `models.yml`, both pass health check
- [ ] `harness_suite` composite resolves to 4 models in dry-run
- [ ] `ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz --dry-run` shows 4 models × 1 benchmark × 2 languages = 8 planned runs
- [ ] After a real run, `ailang eval-matrix eval_results/ --group-by model-family` shows grouped output with delta rows
- [ ] `model_family` field saved in result JSON and readable by eval-matrix
- [ ] All existing `eval-matrix` output unchanged when `--group-by` is not passed
- [ ] `make test ./internal/eval_harness/... ./cmd/ailang/...` passes

---

## Testing Strategy

**Unit tests:**
- `ModelFamily` parses from YAML (add to `TestModelConfigParsed` or equivalent)
- `harness_suite` expands correctly via `expandModelSuite`
- Grouped rendering produces correct delta rows (table formatter test with fixture data)

**Integration tests:**
- Dry-run confirms 8 planned runs for `harness_suite` × `fizzbuzz` × 2 languages
- Grouped `eval-matrix` output is stable (no panic on missing `model_family`)

**Manual testing:**
- Run actual cross-harness eval on at least fizzbuzz to get real delta data
- Verify delta row numbers are correct by manual calculation

---

## Deferred Decisions

- **Delta metric format** — whether to show absolute diff or % diff in the Δ row; agent may choose whatever is clearest
- **Color coding in terminal output** — green for positive delta, red for negative; agent may choose or omit
- **Whether to write `harness_overhead_ms` as a separate JSON field** — `DurationMS` diff is sufficient for now; agent may add if the grouping logic makes it natural

---

## Non-Goals

**Not attempted in this feature:**
- Automated statistical significance testing (t-test across repeated runs) — separate sprint
- Harness comparison for non-agent (0-shot) eval — agent eval only
- Adding new harnesses beyond opencode — only opencode vs native CLI pairs in scope
- Prompt-level diff between harnesses (what exactly differs in system prompts) — observability sprint

---

## Timeline

**Day 1** (~5 hours):
- Phase 1: models.yml + ModelConfig + harness_suite + tests

**Day 2** (~5 hours):
- Phase 2: eval-matrix --group-by flag + grouped rendering + integration test
- Phase 3: end-to-end verification run + document findings

**Total: ~2 days (~10 hours)**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| opencode's `anthropic/` model name format changes | Med | Pin in models.yml; easy to update if API changes |
| Cross-harness run cost doubles eval spend | Med | `harness_suite` is opt-in; not added to `agent_suite` |
| gemini-3-flash hitting quota errors under opencode | Low | Same quota as native gemini CLI; quota errors recorded as failures, not crashes |
| `eval-matrix` grouped output breaks existing CI parsers | Low | Flag is opt-in (`--group-by`); default output unchanged |

---

## Related Documents

**Implemented (inform design):**
- [m-eval-cross-language-benchmark-sprint-plan.md](../../implemented/v0_11_0/m-eval-cross-language-benchmark-sprint-plan.md) — pattern for adding new eval dimensions
- [m-exec-expand-codex-opencode.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) — opencode executor implementation

**Planned (check for overlap):**
- [m-eval-expand-harnesses-languages.md](../v0_13_0/m-eval-expand-harnesses-languages.md) — broader harness/language expansion; this doc is a focused subset
- [m-cloud-eval-workers.md](../v0_13_0/m-cloud-eval-workers.md) — cloud eval parallelism; orthogonal but `harness_suite` pairs will benefit from parallel workers
- [m-ollama-local-eval.md](m-ollama-local-eval.md) — pattern for `ollama_suite` composite; `harness_suite` follows same approach

---

**Document created**: 2026-04-23
**Last updated**: 2026-04-23
