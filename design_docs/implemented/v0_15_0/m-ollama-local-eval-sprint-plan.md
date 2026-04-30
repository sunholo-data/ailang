# M-OLLAMA-LOCAL-EVAL Sprint Plan

**Sprint ID**: M-OLLAMA-LOCAL-EVAL
**Design doc**: [m-ollama-local-eval.md](m-ollama-local-eval.md)
**Duration**: 1 day (~4 hours)
**Risk**: Low — additive changes, no existing tests broken, defaults unchanged for cloud models
**Target**: v0.15.x

## Goal

Make `./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci`
complete at least 1/4 benchmarks instead of timing out on all 4.

Root cause: the single `idleTimeout` fires during Ollama's prefill window (241 s for
recursion_fibonacci with 18k-token AILANG prompt) before the model emits its first token.
Fix: split into `TTFTTimeout` (prefill budget, relaxed for local models) + `GenerationTimeout`
(per-token idle after first event, stays tight, the real quality SLA).

## Milestones

### M1: Add TTFTTimeout to executor.Task (~25 LOC, ~30 min)

**Files**: `internal/executor/executor.go`, `internal/executor/opencode/opencode.go`, `internal/executor/codex/codex.go`

**What**: Add `TTFTTimeout time.Duration` to `Task`. Implement split-timer in both executors:
- Start `ttftTimer` alongside existing `hardTimer` and `idleCheck`
- On first stdout event: stop `ttftTimer`, set `firstEventSeen = true`
- `idleCheck` only fires after `firstEventSeen` (rename semantics: now generation idle)
- `ttftTimer` fires only before first event → error "no output from model within X (prefill timeout)"

Default `TTFTTimeout = 0` → falls back to 30 s (cloud-safe default, current behaviour preserved).

**Acceptance criteria**:
- `TTFTTimeout` field exists on `executor.Task`
- opencode executor: idle timeout before first event uses `TTFTTimeout`, after first event uses `IdleTimeout`
- codex executor: same pattern
- All existing executor tests pass unchanged (defaults = 30 s TTFTTimeout, 3 m generation idle)
- Error messages distinguish "no output" (TTFT) from "stuck mid-generation" (idle)

### M2: Per-model TTFTTimeout in models.yml + ModelConfig (~20 LOC, ~30 min)

**Files**: `internal/eval_harness/models.go`, `internal/eval_harness/models.yml`

**What**: Add `TTFTTimeoutSeconds int` and `GenerationTimeoutSeconds int` to `ModelConfig`.
Parse from `ttft_timeout` / `generation_timeout` YAML keys (0 = use executor defaults).
Add values for Ollama models and `ollama_suite` composite.

```yaml
opencode-gemma4-e4b:
  ttft_timeout: 300      # measured 241s + headroom
  generation_timeout: 120

opencode-gemma4-26b:
  ttft_timeout: 480      # estimate: 2× e4b
  generation_timeout: 180

composites:
  ollama_suite:
    models: [opencode-gemma4-e4b, opencode-gemma4-26b]
    description: "Local Ollama agent models (zero cost, relaxed TTFT)"
```

**Acceptance criteria**:
- `ModelConfig.TTFTTimeoutSeconds` / `GenerationTimeoutSeconds` parse from YAML
- `opencode-gemma4-e4b` has `ttft_timeout: 300`, `generation_timeout: 120` in models.yml
- `opencode-gemma4-26b` has `ttft_timeout: 480`, `generation_timeout: 180`
- `ollama_suite` composite resolves to both Ollama models
- Cloud models have no `ttft_timeout` set (0 → executor default 30 s)

### M3: Wire per-model timeouts into agent runner + result JSON (~30 LOC, ~30 min)

**Files**: `internal/eval_harness/agent_runner_multi.go`, `internal/eval_harness/metrics.go` (or result struct)

**What**: After resolving executor+model in `RunAgentBenchmarkWithExecutor`, read
`TTFTTimeoutSeconds` / `GenerationTimeoutSeconds` from `GlobalModelsConfig` and set on task:

```go
if cfg, ok := GlobalModelsConfig.Models[config.ModelName]; ok {
    if cfg.TTFTTimeoutSeconds > 0 {
        task.TTFTTimeout = time.Duration(cfg.TTFTTimeoutSeconds) * time.Second
    }
    if cfg.GenerationTimeoutSeconds > 0 {
        task.IdleTimeout = time.Duration(cfg.GenerationTimeoutSeconds) * time.Second
    }
}
```

Add `ttft_seconds` to the benchmark result JSON so TTFT is recorded as a hardware metric.

**Acceptance criteria**:
- `opencode-gemma4-e4b` tasks get `TTFTTimeout: 300s`, `IdleTimeout: 120s`
- Cloud model tasks get `TTFTTimeout: 0` (executor default applies)
- Result JSON includes `ttft_seconds` field populated from first-event timestamp
- `TestRunAgentBenchmarkWithExecutor` (or new test) covers the timeout wiring

### M4: End-to-end verification (~0 LOC, ~1 hour)

**What**: Run the real eval and confirm at least 1 benchmark completes.

```bash
make quick-install
./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci
```

Expected: fizzbuzz (ailang or python) completes within 130 s total (measured).
recursion_fibonacci (ailang) completes within ~310 s total (measured).

Also confirm cloud models unaffected:
```bash
ailang eval-suite --agent --models claude-haiku-4-5 --benchmarks fizzbuzz --agent-parallel 1
```

**Acceptance criteria**:
- At least 1/4 Ollama benchmarks passes correctness check (not just "no timeout")
- Cloud model eval unchanged in timing and behaviour
- `ailang eval-suite --models ollama_suite --dry-run` resolves both models
- `make test ./internal/executor/... ./internal/eval_harness/...` passes

## Success Metrics

- [ ] `./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci` ≥ 1/4 pass
- [ ] Zero timeout errors of the form "idle for Xm before first token"
- [ ] Cloud model benchmarks timing unchanged (regression check)
- [ ] `ollama_suite` composite resolves end-to-end
- [ ] All executor + harness tests green

## Non-Goals (this sprint)

- `ailang eval-matrix` / `eval-summary` UI changes for TTFT display
- Lite prompt or μRAG on-demand context (separate sprint)
- 26b model verification (needs 128 GB Mac)
