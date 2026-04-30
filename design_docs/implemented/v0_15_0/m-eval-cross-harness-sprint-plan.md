# M-EVAL-CROSS-HARNESS Sprint Plan

**Sprint ID**: M-EVAL-CROSS-HARNESS
**Design doc**: [m-eval-cross-harness-comparison.md](m-eval-cross-harness-comparison.md)
**Duration**: 2 days (~10 hours)
**Risk**: Low — purely additive, no breaking changes, defaults unchanged
**Target**: v0.15.x

## Goal

Surface harness-induced benchmark deltas for the same model by adding cross-harness model pairs (`opencode-sonnet-4-6`, `opencode-gemini-3-flash`) to `models.yml`, a `harness_suite` composite, and a `--group-by model-family` flag to `eval-matrix` that clusters pairs and shows pass/cost/duration deltas.

## Key Design Decisions (Already Resolved)

- **`model_family`**: Explicit YAML field in `ModelConfig`, not inferred from `api_name` (fragile due to opencode's `anthropic/` prefix)
- **`eval-matrix` grouping**: `--group-by model-family` flag, default off — existing output format unchanged

## Milestones

### M1: ModelConfig + models.yml + harness_suite (~35 LOC, ~2 hours)

**Files**: `internal/eval_harness/models.go`, `internal/eval_harness/models.yml`, `cmd/ailang/eval_suite.go`

**What**:
1. Add `ModelFamily string \`yaml:"model_family"\`` to `ModelConfig` struct
2. Add `opencode-sonnet-4-6` entry to `models.yml` (follows `opencode-haiku` pattern, `agent_model_name: "anthropic/claude-sonnet-4-6"`)
3. Add `opencode-gemini-3-flash` entry to `models.yml` (`agent_model_name: "google/gemini-3-flash-preview"`)
4. Add `model_family` to 5 existing entries: `claude-sonnet-4-6`, `gemini-3-flash`, `claude-haiku-4-5`, `opencode-haiku`, and both new entries
5. Add `harness_suite` composite (4 models: sonnet+opencode-sonnet, gemini+opencode-gemini)
6. Wire `harness_suite` in `expandModelSuite` in `eval_suite.go`

**Acceptance criteria**:
- `ModelConfig.ModelFamily` parses from `model_family` YAML key (0 = no grouping)
- `opencode-sonnet-4-6` health check passes (`ailang eval-suite --agent --models opencode-sonnet-4-6 --benchmarks fizzbuzz --dry-run`)
- `opencode-gemini-3-flash` health check passes
- `ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz --dry-run` shows 4 models × 1 benchmark × 2 languages = 8 planned runs
- `TestTTFTConfigParsed`-style test verifies `ModelFamily` parses correctly for `claude-sonnet-4-6` and `opencode-sonnet-4-6`
- All existing tests pass

### M2: Write model_family to result JSON + eval-matrix --group-by (~80 LOC, ~4 hours)

**Files**: `internal/eval_harness/agent_runner.go`, `internal/eval_harness/metrics.go` (or wherever JSON is saved), `cmd/ailang/eval_tools.go`, `cmd/ailang/eval_matrix_sections.go`

**What**:
1. Add `ModelFamily string \`json:"model_family,omitempty"\`` to `AgentBenchmarkResult`
2. Populate `ModelFamily` from `GlobalModelsConfig` lookup (same `ConfigKey` pattern used for TTFT) when saving result JSON
3. Add `--group-by` flag to `ailang eval-matrix` (accepts `"model-family"`)
4. When `--group-by model-family`: load all results, group by `model_family`, render family clusters with delta row (pass diff, cost diff, duration diff)
5. Delta row format: `Δ (opencode−claude): fizzbuzz/py: 0, fib/ail: −1 | avg: −33% | cost: −$0.003 | dur: +33s`

**Acceptance criteria**:
- `model_family` field appears in saved result JSON for models that have it set
- `ailang eval-matrix <dir> --group-by model-family` renders grouped output with delta rows
- Models without `model_family` fall through to ungrouped display (no crash)
- `ailang eval-matrix <dir>` (no flag) output is byte-identical to pre-sprint (regression test)
- Delta row shows correct arithmetic: if claude scores 3/3 and opencode scores 2/3, Δ = −1

### M3: End-to-end verification (~0 LOC, ~4 hours)

**What**: Run real cross-harness eval on fizzbuzz + sort_list, verify grouped output renders correctly and delta data is meaningful.

```bash
make quick-install
ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz,sort_list --agent-parallel 2
ailang eval-matrix eval_results/ --group-by model-family
```

**Acceptance criteria**:
- At least one model family has ≥1 benchmark result in both harness rows
- Delta row renders without panic
- `ailang eval-suite --agent --models harness_suite --benchmarks fizzbuzz --dry-run` still shows 8 planned runs after install
- `make test ./internal/eval_harness/... ./cmd/ailang/...` green

## Success Metrics

- [ ] `harness_suite` dry-run resolves 4 models (sonnet, opencode-sonnet, gemini, opencode-gemini)
- [ ] `model_family` written to result JSON for all harness_suite models
- [ ] `eval-matrix --group-by model-family` shows paired rows + delta without error
- [ ] No regression in existing `eval-matrix` default output
- [ ] All harness + eval_harness tests green

## Non-Goals (this sprint)

- Statistical significance testing across repeated runs
- Grouping for non-agent (0-shot) eval results
- Adding further harnesses beyond opencode pairs
- Prompt-level diff between harnesses
