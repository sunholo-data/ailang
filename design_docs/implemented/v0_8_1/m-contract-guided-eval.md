# M-CONTRACT-EVAL: Contract-Guided Evaluation Harness

**Status**: PLANNED
**Target**: v0.8.1
**Priority**: P1 (High) — enables ARC-paper-aligned verification experiments
**Estimated**: 12-16h (4 changes, building on existing infrastructure)
**Dependencies**:
- SMT verification (`ailang verify`, `ailang ai-check`) — COMPLETE (v0.8.0)
- Agentic eval (`ailang eval-suite --agent`) — COMPLETE
- Self-repair loop (`repair.go`) — COMPLETE
- Bounded recursion / ADT matching — COMPLETE (v0.8.0)
- 17 contract-guided benchmarks — ready in `demos/benchmarks/contract_guided/`

---

## Motivation

The ARC paper (arxiv:2511.09008v1) demonstrated that Z3 verification feedback improves
LLM code generation validity from 10.8% to 43.9% in 3 iterative rounds. AILANG already
has all the infrastructure components — SMT verification, agentic eval with multi-turn
iteration, self-repair with error categorization, multi-model comparison — but they
aren't connected. This design doc bridges the gap with targeted additions that **build
on existing iteration mechanisms** rather than creating new ones.

### Existing Infrastructure (what we build on)

| Component | Location | What it does |
|-----------|----------|-------------|
| **Self-repair** (standard mode) | `repair.go` | 0-shot → error categorization → 1-shot repair. Uses `CategorizeErrorWithCode()` → `FormatRepairPrompt()` to give structured feedback. |
| **Agent iteration** (agent mode) | `agent_runner.go` | Multi-turn agentic coding via Claude Code / Gemini CLI. Agent has `Bash`, `Read`, `Write`, `Edit`, `Grep` tools and can run `ailang check`, `ailang run` in its workspace. Iterates until correct or timeout. |
| **Prompt templates** | `templates/agent_task_{lang}.txt` | `{{PLACEHOLDER}}` substitution for task-specific instructions. |
| **Validation pipeline** | `runner.go` | Compile → Runtime → Stdout match (`CompileOk` / `RuntimeOk` / `StdoutOk`). |
| **ai-check command** | `cmd/ailang/ai_check.go` | Unified check+verify in single JSON output. |

**Key insight**: We don't need to build new iteration infrastructure. We need to make
Z3 results available to the existing iteration mechanisms.

### Experiment design (4 conditions)

| Condition | Contracts in prompt | Z3 verify | Z3 feedback | Devtools prompt |
|-----------|--------------------:|:---------:|:-----------:|:---------------:|
| baseline  | No                  | No        | No          | No              |
| contract  | Yes                 | Yes       | No          | No              |
| iterative | Yes                 | Yes       | Yes         | No              |
| full      | Yes                 | Yes       | Yes         | Yes             |

---

## Change 1: `contract_spec` Field in Benchmark YAML

### Problem

No way to include Z3-verifiable contract specifications in benchmark definitions.
Agents currently receive only `task_prompt` (natural language description).

### Solution

Add optional `contract_spec` field to `BenchmarkSpec`. When present and `--verify`
is active, it's injected into the prompt using the existing template placeholder system.

**File**: `internal/eval_harness/spec.go`

```go
type BenchmarkSpec struct {
    // ... existing fields ...
    ContractSpec string `yaml:"contract_spec"` // Optional: AILANG contract specification
}
```

**Benchmark YAML example**:
```yaml
id: contract_promo_rebate
difficulty: medium
languages: [ailang]
entrypoint: main
caps: [IO]
expected_stdout: |
  PROMO-SUMMER2024: true
  HELLO: false

task_prompt: |
  Implement a promo code validator. A valid promo code starts with "PROMO-"
  and is at least 10 characters long.

contract_spec: |
  import std/string (startsWith, length as strLen)

  export pure func isValidPromo(code: string) -> bool
    ensures { result == (startsWith(code, "PROMO-") && strLen(code) >= 10) }
```

### Prompt injection (both modes)

Add `{{CONTRACT_SPEC}}` placeholder to the task prompt template. When `--verify` is
active and `contract_spec` exists, it expands to:

```
FORMAL SPECIFICATION (your solution MUST satisfy these contracts):
```ailang
{contract_spec}
```
Run `ailang ai-check solution.ail` to verify your solution against these contracts.
```

When `contract_spec` is absent or `--verify` is not active, `{{CONTRACT_SPEC}}` expands
to empty string (backward compatible, baseline condition).

**File**: `internal/eval_harness/templates/agent_task_ailang.txt` — add `{{CONTRACT_SPEC}}`

---

## Change 2: `--verify` Post-Validation + Self-Repair Integration

### Problem

After code generation, there's no way to run contract verification and record results.
The eval harness only checks compile/runtime/stdout.

### Solution

Two integration points, matching the two eval modes:

#### Standard mode: Z3 as a repair error category

Add Z3 verification as a new step in the validation pipeline. When verification fails,
treat it like any other error in the existing self-repair loop (`repair.go`).

**File**: `internal/eval_harness/repair.go`, `errors.go`

**New error code**: `VERIFY_COUNTEREXAMPLE`

```go
// In errors.go — add new category
"VERIFY_COUNTEREXAMPLE": {
    Title: "Contract verification failed — Z3 found counterexample",
    Why:   "Your function does not satisfy its ensures clause for all inputs.",
    How:   "Read the counterexample values below and fix the logic error.",
}
```

**Integration in repair.go** (after compile succeeds, before runtime):
```go
// Existing flow: first attempt → check compile → [repair if error]
// NEW: after compile succeeds, also verify contracts
if verifyFlag && spec.ContractSpec != "" && firstResult.CompileOk {
    verifyResult := runAICheck(solutionPath, verifyTimeout)
    if verifyResult.Verify.Counterexample > 0 {
        // Use EXISTING repair mechanism — Z3 counterexample is just another error
        hint := formatZ3RepairHint(verifyResult)
        repairPrompt := prompt + "\n\n" + FormatRepairPrompt(
            "VERIFY_COUNTEREXAMPLE", hint,
            spec.ID, "ailang", firstResult.Code, formatZ3Stderr(verifyResult),
        )
        // Existing 1-round repair runs here
    }
}
```

This gives standard mode a **Z3-informed repair attempt** using the existing infrastructure.
No new loop needed — the repair mechanism already handles one retry with error feedback.

#### Agent mode: Verification instructions in prompt

The agent already has multi-turn iteration. We just need to tell it `ailang ai-check`
exists. This goes in the prompt template (Change 1 `{{CONTRACT_SPEC}}` expansion).

The agent will naturally:
1. Write code
2. Run `ailang ai-check solution.ail`
3. See counterexamples in output
4. Fix code
5. Re-verify

No new code needed in `agent_runner.go` — the agent loop handles this.

#### Post-hoc verification (both modes)

After the final solution is produced (whether standard or agent mode), run
`ailang ai-check --json` as a **recording step** to capture verification metrics.

**File**: `internal/eval_harness/runner.go` (standard), `agent_runner.go` (agent)

**Extended RunMetrics** (`internal/eval_harness/metrics.go`):
```go
type RunMetrics struct {
    // ... existing fields ...

    // Contract verification results (populated when --verify is active)
    VerifyOk           bool   `json:"verify_ok"`            // All contracts verified
    VerifyVerified     int    `json:"verify_verified"`       // Count of verified functions
    VerifyCounterex    int    `json:"verify_counterexample"` // Count of counterexamples
    VerifySkipped      int    `json:"verify_skipped"`        // Count of skipped functions
    VerifyErrors       int    `json:"verify_errors"`         // Count of Z3 errors
    VerifyJSON         string `json:"verify_json,omitempty"` // Full ai-check JSON output
}
```

**Validation flow** (extended):
```
Standard mode:
  Generate → Compile → Verify → [Repair if error] → Runtime → Stdout
               │          │            │                 │         │
               CompileOk  VerifyOk    (uses existing    RuntimeOk StdoutOk
                                       repair loop)

Agent mode:
  Agent iterates freely (runs ailang check, ai-check, run as tools)
  → Post-hoc: Verify final solution → Record metrics
               │
               VerifyOk
```

**CLI flags**:
```
--verify                 Enable contract verification
--verify-timeout 5s      Per-function Z3 timeout (default: 5s)
```

**Metrics enabled**:
- **Verify@1**: P(all contracts verified on first attempt)
- **Soundness**: P(StdoutOk | VerifyOk) — how often verified code is actually correct
- **False positive rate**: P(!StdoutOk & VerifyOk) — verified but wrong output
- **Repair convergence**: P(VerifyOk after repair | !VerifyOk before repair) — standard mode

---

## Change 3: Fix `ailang eval` CWD Dependency

### Problem

`ailang eval` only works when CWD is the ailang repo directory. Path resolution for
`benchmarks/*.yml` and `std/` uses `os.Getwd()`, failing from other directories.

### Solution

Resolve benchmark paths relative to the `ailang` binary location (using `os.Executable()`),
not CWD. For stdlib, already handled via `--stdlib-path` flag.

**File**: `internal/eval_harness/runner.go`

**Current** (broken):
```go
cwd, _ := os.Getwd()
specPath := filepath.Join(cwd, "benchmarks", specFile+".yml")
```

**Fixed**:
```go
// Resolve benchmark dir from binary location
binPath, _ := os.Executable()
binDir := filepath.Dir(binPath)
projectRoot := findProjectRoot(binDir) // Walk up to find go.mod

// Or use explicit flag (takes precedence)
benchmarkDir := flagBenchmarkDir
if benchmarkDir == "" {
    benchmarkDir = filepath.Join(projectRoot, "benchmarks")
}

specPath := filepath.Join(benchmarkDir, specFile+".yml")
```

**New CLI flag**:
```
--benchmark-dir <path>   Directory containing benchmark YAML files
```

This allows:
```bash
# From any directory:
ailang eval-suite --benchmark-dir /path/to/ailang/benchmarks --models gpt5-mini
# Or with contract benchmarks from demos:
ailang eval-suite --benchmark-dir demos/benchmarks/contract_guided --models gpt5-mini --verify
```

---

## Change 4: Devtools Prompt Integration

### Problem

In the "full" experiment condition, agents need both the language syntax prompt and the
devtools toolchain prompt. Currently no way to inject the devtools prompt into eval.

### Solution

Add `--devtools-prompt` flag to `ailang eval-suite`. When active, the devtools prompt
(from `ailang devtools-prompt --compact`) is appended to the system prompt.

**File**: `cmd/ailang/eval_suite.go`, `internal/eval_harness/agent_prompt.go`

```go
if devtoolsPromptFlag {
    devtoolsContent, err := devtoolsPrompt.LoadPrompt("v0.8.0-compact")
    systemPrompt = systemPrompt + "\n\n" + devtoolsContent
}
```

This gives agents awareness of `ailang ai-check`, `ailang verify --json`, debug flags,
and all other toolchain commands — enabling better self-directed iteration.

---

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/eval_harness/spec.go` | Add `ContractSpec` field | ~5 |
| `internal/eval_harness/metrics.go` | Add verify fields to RunMetrics | ~15 |
| `internal/eval_harness/errors.go` | Add `VERIFY_COUNTEREXAMPLE` error category + Z3 hint formatter | ~30 |
| `internal/eval_harness/repair.go` | Add verify step before repair decision | ~20 |
| `internal/eval_harness/runner.go` | Add post-hoc verify, fix CWD resolution | ~40 |
| `internal/eval_harness/agent_runner.go` | Add post-hoc verify recording | ~15 |
| `internal/eval_harness/agent_prompt.go` | Add `{{CONTRACT_SPEC}}` placeholder, devtools prompt | ~20 |
| `internal/eval_harness/templates/agent_task_ailang.txt` | Add `{{CONTRACT_SPEC}}` section | ~10 |
| `cmd/ailang/eval_suite.go` | Add `--verify`, `--benchmark-dir`, `--devtools-prompt` flags | ~25 |

**Total**: ~180 LOC

---

## Implementation Order

1. **Change 3** (CWD fix) — Quick win, unblocks running evals from any directory
2. **Change 1** (`contract_spec` field + prompt template) — Schema + prompt wiring
3. **Change 2** (`--verify` + self-repair integration) — Core verification, builds on 1
4. **Change 4** (devtools prompt) — Enables "full" experiment condition

---

## Verification Plan

### Unit Tests

```go
// spec_test.go: ContractSpec field parses from YAML
func TestBenchmarkSpec_ContractSpec(t *testing.T) { ... }

// errors_test.go: Z3 counterexample categorized correctly
func TestCategorizeError_VerifyCounterexample(t *testing.T) { ... }

// metrics_test.go: VerifyOk field serializes/deserializes
func TestRunMetrics_VerifyFields(t *testing.T) { ... }

// runner_test.go: CWD-independent path resolution
func TestRunner_BenchmarkDirResolution(t *testing.T) { ... }
```

### Integration Tests

```bash
# Change 3: CWD independence
cd /tmp && ailang eval-suite --benchmark-dir /path/to/benchmarks \
  --benchmarks fizzbuzz --models gpt5-mini --langs ailang

# Change 2: Verify flag (standard mode with repair)
ailang eval-suite --benchmarks contract_abs_value --verify --self-repair \
  --benchmark-dir demos/benchmarks/contract_guided --models gpt5-mini

# Change 2: Verify flag (agent mode — agent iterates naturally)
ailang eval-suite --agent --benchmarks contract_abs_value --verify \
  --benchmark-dir demos/benchmarks/contract_guided --models claude-sonnet-4-5

# Change 4: Full devtools condition
ailang eval-suite --agent --verify --devtools-prompt \
  --benchmark-dir demos/benchmarks/contract_guided --models claude-sonnet-4-5
```

### Experiment (full 4-condition run)

```bash
BENCHDIR=demos/benchmarks/contract_guided
MODELS=claude-sonnet-4-5,gpt5-mini,gemini-2-5-flash

# Condition 1: Baseline (no contracts, no verify)
ailang eval-suite --agent --benchmark-dir $BENCHDIR --models $MODELS \
  --output eval_results/contract_exp/baseline

# Condition 2: Contract-guided (contracts in prompt, verify result)
ailang eval-suite --agent --benchmark-dir $BENCHDIR --models $MODELS --verify \
  --output eval_results/contract_exp/contract

# Condition 3: Contract + devtools (agent knows about ai-check, verify commands)
ailang eval-suite --agent --benchmark-dir $BENCHDIR --models $MODELS --verify \
  --devtools-prompt --output eval_results/contract_exp/iterative

# Condition 4: Full (+ compact syntax prompt)
ailang eval-suite --agent --benchmark-dir $BENCHDIR --models $MODELS --verify \
  --devtools-prompt --prompt-version v0.7.4-compact \
  --output eval_results/contract_exp/full

# Compare results
ailang eval-compare eval_results/contract_exp/baseline eval_results/contract_exp/contract
ailang eval-compare eval_results/contract_exp/contract eval_results/contract_exp/iterative
```

---

## Risks

| Risk | Mitigation |
|------|-----------|
| Z3 timeout during eval slows suite | Default 5s timeout, `--verify-timeout` flag |
| Agent doesn't discover `ailang ai-check` on its own | Devtools prompt includes it; `{{CONTRACT_SPEC}}` mentions it explicitly |
| Standard mode repair of Z3 errors less effective than agent iteration | Expected — standard mode is 1-round repair, agent mode is multi-turn. We measure both. |
| Path resolution breaks on different OS | Use `filepath.Join`, test on CI |
| `contract_spec` conflicts with `task_prompt` | Clear separation in prompt template, distinct section header |

---

## Non-Goals

- **New iteration loop** — We use existing self-repair (standard) and agent iteration (agent mode)
- **Custom contract language** — Uses existing AILANG `requires`/`ensures` syntax
- **Redundant generation** — Phase 2 of M-VERIFY, separate design doc
- **Cloud eval workers** — Separate `m-cloud-eval-workers.md` design doc
- **New benchmark creation** — 17 benchmarks already exist in demos

---

## Design Rationale: Why Not a Custom Z3 Feedback Loop?

The initial design considered a `--z3-feedback N` flag that builds a custom iteration
loop around the agent executor. This was rejected because:

1. **Standard mode already has self-repair** (`repair.go`). Z3 counterexamples are just
   another error category, like `PAR_001` (syntax error) or `CAP_001` (missing capability).
   The existing `CategorizeErrorWithCode()` → `FormatRepairPrompt()` pipeline handles this.

2. **Agent mode already iterates**. The agent has `Bash` access and can run
   `ailang ai-check` on its own. Building an outer loop would fight the agent's own
   iteration — the agent might fix the code before our outer loop re-checks.

3. **Simpler experiment design**. Instead of N configurable rounds, we measure:
   - Standard mode: "Did the 1-round repair fix the Z3 error?" (binary)
   - Agent mode: "Did the agent converge to verified code?" (binary, agent controls rounds)

4. **Fewer moving parts**. The only new code is error categorization + prompt injection,
   which integrates cleanly with existing infrastructure (~180 LOC vs ~300+ LOC).
