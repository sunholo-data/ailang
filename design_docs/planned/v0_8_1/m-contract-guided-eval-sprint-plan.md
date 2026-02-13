# Sprint Plan: M-CONTRACT-EVAL — Contract-Guided Evaluation Harness

**Sprint ID**: M-CONTRACT-EVAL
**Design Doc**: [m-contract-guided-eval.md](m-contract-guided-eval.md)
**Duration**: 2 days (~12-16 hours)
**Risk Level**: Low-Medium (builds on existing infrastructure, ~180 LOC new code)
**Total LOC Estimate**: ~250 (180 impl + 70 tests + benchmark YAML files)

---

## Sprint Summary

Connect AILANG's existing SMT verification infrastructure (`ailang ai-check`) to the
eval harness's existing self-repair and agent iteration mechanisms. This enables
ARC-paper-aligned experiments measuring how Z3 contract feedback improves LLM code
generation validity.

**Key principle**: No new iteration loops — Z3 becomes another error category in
standard mode's repair loop, and agent mode just learns `ailang ai-check` exists
via prompt template injection.

---

## Current Status Analysis

### Completed Dependencies
- SMT verification (`ailang verify`, `ailang ai-check --json`) — v0.8.0
- Agentic eval (`ailang eval-suite --agent`) — Complete
- Self-repair loop (`repair.go`, `errors.go`) — Complete (8 error categories)
- Bounded recursion / ADT matching — v0.8.0
- Compact prompts (`ailang prompt --compact`, `ailang devtools-prompt --compact`) — v0.8.0

### Infrastructure Assessment
| Component | File | Lines | Status |
|-----------|------|-------|--------|
| BenchmarkSpec | `spec.go` | 182 | Needs `ContractSpec` field |
| RunMetrics | `metrics.go` | 174 | Needs verify fields |
| Error taxonomy | `errors.go` | 195 | Needs `VERIFY_COUNTEREXAMPLE` |
| Self-repair | `repair.go` | 165 | Needs verify step before repair |
| Runner | `runner.go` | 445 | Needs post-hoc verify + CWD fix |
| Agent runner | `agent_runner.go` | 491 | Needs post-hoc verify recording |
| Prompt gen | `agent_prompt.go` | ~300 | Needs `{{CONTRACT_SPEC}}` |
| Template | `agent_task_ailang.txt` | 57 | Needs contract section |
| CLI flags | `eval_suite.go` | 544 | Needs `--verify`, `--benchmark-dir`, `--devtools-prompt` |

### Velocity (last 14 days)
- SMT bounded recursion + cross-function + records + strings + lists: ~1,380 LOC impl + ~1,460 LOC tests
- Trace export phases 1-4: ~1,690 LOC
- AI devtools workflow (Tier 2): ~300 LOC
- Estimated pace: ~250-400 LOC/day

---

## Milestones

### M1: CWD Fix + CLI Flags (~2h)

**Goal**: Fix `ailang eval` CWD dependency and add new CLI flags.

**Files**:
- `internal/eval_harness/runner.go` — Fix path resolution (~20 LOC)
- `cmd/ailang/eval_suite.go` — Add `--verify`, `--verify-timeout`, `--benchmark-dir`, `--devtools-prompt` flags (~25 LOC)

**Tasks**:
1. Add `findProjectRoot()` function to `runner.go` (walk up to find `go.mod`)
2. Replace `os.Getwd()` path resolution with `os.Executable()` + `findProjectRoot()`
3. Add `--benchmark-dir` flag that takes precedence over auto-resolution
4. Add `--verify`, `--verify-timeout`, `--devtools-prompt` flags to eval_suite.go
5. Thread flags through to runner/agent_runner via config structs
6. Test: run `ailang eval-suite` from `/tmp` with `--benchmark-dir`

**Acceptance Criteria**:
- `cd /tmp && ailang eval-suite --benchmark-dir /path/to/benchmarks --benchmarks fizzbuzz --models gpt5-mini` works
- `--verify`, `--verify-timeout`, `--benchmark-dir`, `--devtools-prompt` flags parse correctly
- All existing tests pass

**Estimated LOC**: ~45

---

### M2: Contract Spec + Prompt Template (~3h)

**Goal**: Add `contract_spec` field to benchmark YAML and wire it into prompt generation.

**Files**:
- `internal/eval_harness/spec.go` — Add `ContractSpec` field (~5 LOC)
- `internal/eval_harness/agent_prompt.go` — Add `{{CONTRACT_SPEC}}` expansion (~15 LOC)
- `internal/eval_harness/templates/agent_task_ailang.txt` — Add contract section (~10 LOC)
**Tasks**:
1. Add `ContractSpec string \`yaml:"contract_spec"\`` to `BenchmarkSpec`
2. Create `expandContractSpec()` function: returns formatted contract block when spec present and `--verify` active, empty string otherwise
3. Add `{{CONTRACT_SPEC}}` to `agent_task_ailang.txt` template (after task description, before expected output)
4. Wire `--verify` flag to prompt expansion (pass verify flag through to prompt builder)
5. Test: YAML parsing of `contract_spec` field using existing benchmarks from `/Users/mark/dev/sunholo/demos/benchmarks/contract_guided/`
6. Test: prompt expansion with/without `--verify` flag

**Note**: 17 contract-guided benchmarks already exist at `/Users/mark/dev/sunholo/demos/benchmarks/contract_guided/` with proper `contract_spec` fields. No new benchmark creation needed.

**Acceptance Criteria**:
- `BenchmarkSpec` parses `contract_spec` from YAML
- With `--verify`: contract spec appears in agent prompt
- Without `--verify`: contract spec section absent (backward compatible)
- Existing contract benchmarks parseable with new `ContractSpec` field

**Estimated LOC**: ~50 (impl)

---

### M3: Verify Integration + Self-Repair (~5h)

**Goal**: Run `ailang ai-check` as verification step, integrate with self-repair, record metrics.

**Files**:
- `internal/eval_harness/errors.go` — Add `VERIFY_COUNTEREXAMPLE` error code + hint formatter (~30 LOC)
- `internal/eval_harness/repair.go` — Add verify step after compile succeeds (~20 LOC)
- `internal/eval_harness/metrics.go` — Add verify fields to `RunMetrics` (~15 LOC)
- `internal/eval_harness/runner.go` — Add post-hoc verify for standard mode (~20 LOC)
- `internal/eval_harness/agent_runner.go` — Add post-hoc verify for agent mode (~15 LOC)

**Tasks**:
1. Add `VERIFY_COUNTEREXAMPLE` to error taxonomy in `errors.go` with repair hint
2. Add `formatZ3RepairHint()` function that extracts counterexample values from `ai-check --json` output
3. Add `runAICheck()` helper that shells out to `ailang ai-check --json` and parses result
4. In `repair.go`: after compile succeeds + before runtime, if `--verify` active and `contract_spec` present, run verification. If counterexample found, trigger existing repair with Z3-specific hint
5. Add verify fields to `RunMetrics`: `VerifyOk`, `VerifyVerified`, `VerifyCounterex`, `VerifySkipped`, `VerifyErrors`, `VerifyJSON`
6. In `runner.go`: after final solution produced (standard mode), run post-hoc `ai-check --json` to record verify metrics
7. In `agent_runner.go`: after agent finishes, run post-hoc `ai-check --json` to record verify metrics
8. Test: error categorization of Z3 counterexample
9. Test: metrics JSON serialization/deserialization with verify fields
10. Test: repair flow with verify step (mock ai-check output)

**Acceptance Criteria**:
- Standard mode: code that fails verification triggers repair with Z3 hint
- Agent mode: post-hoc verification recorded in metrics
- Verify metrics correctly populated in result JSON
- All existing tests pass (verify fields are optional, zero values for non-verify runs)

**Estimated LOC**: ~100 (impl) + ~40 (tests)

---

### M4: Devtools Prompt + Smoke Test (~2h)

**Goal**: Enable "full" experiment condition and validate end-to-end.

**Files**:
- `cmd/ailang/eval_suite.go` — Wire `--devtools-prompt` to prompt builder (~10 LOC)
- `internal/eval_harness/agent_prompt.go` — Load and append devtools prompt (~10 LOC)

**Tasks**:
1. When `--devtools-prompt` flag active, load compact devtools prompt from embedded prompts
2. Append devtools prompt to system prompt in agent mode
3. Smoke test: run standard mode with `--verify` on contract benchmark
4. Smoke test: run agent mode with `--verify --devtools-prompt` on contract benchmark
5. Verify metrics JSON contains verify fields

**Acceptance Criteria**:
- `--devtools-prompt` appends devtools reference to agent system prompt
- Standard mode eval with `--verify` completes without errors
- Agent mode eval with `--verify --devtools-prompt` completes without errors
- Result JSON contains `verify_ok`, `verify_verified`, etc. fields

**Estimated LOC**: ~20 (impl) + ~10 (test)

---

## Implementation Order

```
M1 (CWD fix + flags) ──► M2 (contract spec + template) ──► M3 (verify + repair) ──► M4 (devtools + smoke test)
        2h                          3h                              5h                        2h
```

All milestones are sequential — each builds on the previous.

---

## Success Metrics

- [ ] `make test` passes (0 regressions)
- [ ] `make lint` passes (0 issues)
- [ ] `ailang eval-suite` works from any directory with `--benchmark-dir`
- [ ] Contract benchmark YAML with `contract_spec` field parses correctly
- [ ] Standard mode `--verify` triggers Z3 repair on counterexample
- [ ] Agent mode `--verify` records post-hoc verification metrics
- [ ] `--devtools-prompt` injects devtools reference into agent prompt
- [ ] Result JSON contains verify fields (`verify_ok`, `verify_verified`, etc.)

---

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Z3 not installed on eval machine | Medium | Check Z3 availability before verify, skip gracefully if absent |
| `ai-check --json` output format changes | Low | Already implemented in v0.8.0, stable interface |
| Contract benchmarks in external dir | Low | 17 benchmarks at `~/dev/sunholo/demos/benchmarks/contract_guided/`; `--benchmark-dir` flag handles this |
| Agent ignores `ailang ai-check` in prompt | Low | Expected for baseline condition; devtools prompt helps in "full" condition |
| Self-repair on Z3 errors is ineffective | Low | Expected — this is what we're measuring. Binary metric: did repair fix it or not |

---

## Notes

- 17 contract-guided benchmarks exist at `/Users/mark/dev/sunholo/demos/benchmarks/contract_guided/` (outside the ailang repo). The `--benchmark-dir` flag (M1) is essential for pointing eval to this external directory.
- Agent mode requires no code changes to `agent_runner.go` for iteration — only post-hoc verification recording is needed.
- The `runAICheck()` helper shells out to `ailang ai-check --json` rather than calling the verify pipeline directly in Go, keeping the integration simple and reusing the existing CLI command.
