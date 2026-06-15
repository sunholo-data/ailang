# M-APPLE-CONTAINER-LOCAL-EVAL-SANDBOX

**Status**: Planned (blocked — requires macOS 26, GA ~autumn 2026)
**Target**: Post-macOS-26-GA (likely v0.26+)
**Priority**: P2 - Low (no action until macOS 26 is GA)
**Estimated**: 3 days
**Dependencies**: macOS 26 (Tahoe) GA release; Apple Silicon hardware

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval infrastructure, not a language feature — axioms apply to how the harness models execution environments.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Containers give reproducible starting state — same image = same eval environment every run |
| A2: Replayability | +1 | Fixed container image tag = same run conditions; failed runs can be replayed identically |
| A3: Effect Legibility | 0 | No impact on language-level effect tracking |
| A4: Explicit Authority | +1 | Eval agent process has bounded filesystem scope (container root), not host access |
| A5: Bounded Verification | 0 | No impact on type checking |
| A6: Safe Concurrency | +1 | Isolated containers prevent concurrent eval runs from polluting each other's workspaces |
| A7: Machines First | +1 | Reproducible environments produce more trustworthy machine-readable benchmark results |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | Container overhead negligible; no change to token/cost tracking |
| A10: Composability | +1 | Slots into existing `CapRemoteSandbox` pattern — same abstraction, local flavor |
| A11: Structured Failure | +1 | Container exit codes + OCI runtime errors are structured failure signals |
| A12: System Boundary | +1 | VM-level boundary makes "eval process accesses host" impossible by construction |

**Net Score: +8** → **Decision: Proceed (when prerequisite met)**

### Hard Violation Check

- [x] A1 (Determinism): Fixed image = deterministic start state
- [x] A3 (Effects): No hidden side effects; eval effects stay inside container
- [x] A4 (Authority): Bounded to container filesystem — no ambient host access
- [x] A7 (Machines First): Improves result quality for machine consumers

## Problem Statement

Local eval runs for agentic benchmarks use filesystem-only isolation: each task gets a temp directory (`WorkspaceDir/spec_lang_executor_pid`). For non-agentic benchmarks (compile-and-check) this is fine. For **agentic evals** where the AI agent edits real files over multiple turns, this has gaps:

**Current State:**
- Agentic eval runs share the host filesystem — a buggy agent can write outside its workspace dir
- No guarantee the host environment is clean between runs (pre-installed binaries, env vars, stale artefacts from prior runs)
- The `CapRemoteSandbox` capability exists to model clean-slate isolation for cloud executors (Vertex Managed Agents) but there is no local equivalent
- Docker/OrbStack can provide local containers but add daemon overhead and subscription cost

**Impact:**
- Affects agentic benchmark reliability — false passes/fails caused by environment bleed
- Hard to reproduce "clean machine" failures locally
- Becoming more relevant as the agent eval suite grows

## Goals

**Primary Goal:** Give agentic eval tasks a clean VM-per-run using Apple Container, matching the isolation guarantee of cloud sandbox executors — without external daemon dependencies.

**Success Metrics:**
- Each agentic benchmark task starts in a pristine OCI container with no state from prior runs
- No file writes escape the container's root filesystem
- Eval run time increases by < 20% (container boot is sub-second per Apple's design)
- `run_eval_baseline.sh --sandbox apple-container` flag enables the mode opt-in

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Container image to use as base | Determines what tools are pre-installed (Go, ailang binary, etc.) | human | design | high |
| Per-task vs per-suite container lifecycle | Per-task = maximum isolation, higher overhead; per-suite = faster but less clean | human | design | med |
| How to get the ailang binary into the container | Build into image at eval time vs mount from host | agent | compile | med |
| Whether to add `CapLocalSandbox` capability or reuse `CapRemoteSandbox` | Determines code path; local sandbox can read the container filesystem directly | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Base container image** — likely `ubuntu:24.04` + Go toolchain, but needs confirming against what the benchmarks actually need
- [ ] **Per-task vs per-suite lifecycle** — per-task is the safe default but may be too slow for 49-benchmark suites; measure before deciding

## Solution Design

### Overview

Add an optional `apple-container` sandbox mode to the local eval harness. When enabled, each benchmark task runs inside a fresh OCI container (via `container run --rm`) instead of a bare temp directory. The eval harness starts the container, mounts the task workspace in, runs the executor, then discards the container. The rest of the eval pipeline (validation, scoring, result export) is unchanged.

### Architecture

**Components:**

1. **`internal/eval_harness/container_runner.go`** (new): wraps a local executor invocation in `container run --rm --workdir /task <image> <cmd>`. Exposes the same `ExecutorResult` interface as the direct runner. Detects whether `container` binary is on PATH and macOS version ≥ 26; fails loudly if not.

2. **`internal/eval_harness/agent_runner_multi.go`** (modify): add `CapLocalSandbox` branch alongside existing `CapRemoteSandbox`. When the runner config has `SandboxMode: "apple-container"`, routes through `container_runner.go` instead of bare subprocess.

3. **`run_eval_baseline.sh`** (modify): add `--sandbox <mode>` flag. `--sandbox apple-container` sets `AILANG_EVAL_SANDBOX=apple-container` env var which the Go harness reads.

4. **`eval/base.Dockerfile`** (new): minimal image — Ubuntu + Go toolchain + pre-built ailang binary. Built once and tagged; the eval script verifies the image exists before running.

### Implementation Plan

**Phase 1: Container runner** (~4 hours)
- [ ] Write `container_runner.go` — detect binary, check macOS version, `container run --rm` wrapper
- [ ] Unit test: mock `container` binary, verify args constructed correctly
- [ ] Fail-loud path: if `container` not found or macOS < 26, return explicit error (not silent fallback)

**Phase 2: Harness integration** (~3 hours)
- [ ] Add `CapLocalSandbox` to `executor/capability.go`
- [ ] Wire `SandboxMode` config field in `agent_runner_multi.go`
- [ ] Integration test with a real benchmark task against a lightweight image

**Phase 3: Image + CLI flag** (~4 hours)
- [ ] Write `eval/base.Dockerfile`
- [ ] Add `--sandbox` flag to `run_eval_baseline.sh`
- [ ] Update `ailang eval` CLI to expose `--sandbox` flag
- [ ] Document in evaluation guide

### Files to Modify/Create

**New files:**
- `internal/eval_harness/container_runner.go` — container invocation wrapper (~120 LOC)
- `eval/base.Dockerfile` — base eval image (~20 LOC)

**Modified files:**
- `internal/eval_harness/agent_runner_multi.go` — add `CapLocalSandbox` branch (~30 LOC)
- `internal/executor/capability.go` — add `CapLocalSandbox` constant (~5 LOC)
- `.claude/skills/post-release/scripts/run_eval_baseline.sh` — add `--sandbox` flag (~15 LOC)
- `docs/docs/guides/evaluation/running-evals.md` — document sandbox mode

## Examples

### Example 1: Running agentic evals with sandbox

**Before (bare filesystem):**
```bash
ailang eval run --tier core --benchmarks code-editing
# Agent edits happen directly on host temp dirs
# /tmp/ailang_eval_abc123/solution.ail may persist if cleanup fails
```

**After (Apple Container sandbox):**
```bash
ailang eval run --tier core --benchmarks code-editing --sandbox apple-container
# Each task: container run --rm ailang-eval-base:latest
# Container exits → filesystem wiped automatically
```

### Example 2: `run_eval_baseline.sh` for post-release

```bash
# Standard run (unchanged behaviour)
.claude/skills/post-release/scripts/run_eval_baseline.sh --tier core

# Sandboxed agentic run (macOS 26+ only)
.claude/skills/post-release/scripts/run_eval_baseline.sh --tier core --sandbox apple-container
```

### Example 3: container_runner.go detection

```go
// container_runner.go
func NewContainerRunner() (*ContainerRunner, error) {
    if err := checkMacOS26(); err != nil {
        return nil, fmt.Errorf("apple-container sandbox requires macOS 26+: %w", err)
    }
    if _, err := exec.LookPath("container"); err != nil {
        return nil, fmt.Errorf("apple-container sandbox: 'container' binary not in PATH — install from github.com/apple/container")
    }
    return &ContainerRunner{image: "ailang-eval-base:latest"}, nil
}
```

## Success Criteria

- [ ] `container_runner.go` detects missing binary or wrong macOS version and returns a clear error (not a silent fallback to bare runner)
- [ ] A benchmark task run with `--sandbox apple-container` produces the same pass/fail result as bare mode for a known-passing test
- [ ] No files written by the container agent are visible on the host after the run
- [ ] `make test` passes (all existing eval harness tests unaffected)
- [ ] Documentation updated in evaluation guide

## Testing Strategy

**Unit tests:**
- `container_runner_test.go`: mock `container` binary, verify correct `run --rm --workdir` args
- Version detection: mock `sw_vers` output, verify correct error on < 26
- Capability routing: `agent_runner_multi_test.go` with `SandboxMode: "apple-container"` config

**Integration tests:**
- Run one real benchmark (smallest available) against `ubuntu:24.04` to confirm end-to-end flow works
- Verify workspace dir on host is clean post-run

**Manual testing (macOS 26 only):**
- Run full `--tier smoke` suite with `--sandbox apple-container`
- Confirm runtime overhead is < 20% vs bare mode
- Confirm parallel eval tasks don't interfere (concurrent containers)

## Deferred Decisions

- **Which Linux base image** — agent may choose `ubuntu:24.04` or `debian:bookworm-slim` based on dependency fit
- **Whether to cache the built image between runs** — agent may implement a `--rebuild-image` flag vs always using cached
- **Parallel container limit** — agent may choose to mirror the existing `maxConcurrentTasks` config

## Non-Goals

- **Windows/Intel Mac support** — Apple Container is Apple Silicon + macOS 26 only; out of scope
- **Replacing OrbStack/Docker for dev workflow** — this is eval-only; no change to how contributors run things locally
- **Full reproducible build image** (pinning exact Go version, ailang commit, etc.) — nice-to-have but not required for v1 of this feature
- **Cloud eval workers** — that's a separate design doc ([m-cloud-eval-workers](v0_13_0/m-cloud-eval-workers.md))

## Timeline

**Single sprint (~2 days) once macOS 26 is GA:**
- Day 1: Phase 1 + Phase 2 (container runner + harness integration)
- Day 2: Phase 3 (Dockerfile, CLI flag, docs) + testing

**Prerequisite gate:** Do not start until `sw_vers -productVersion` on dev machine returns 26.x.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| macOS 26 beta instability breaks dev machine | High | Do not upgrade until GA; doc is parked until then |
| `container` API changes before GA | Med | Apple tagged 1.0.0 but notes minor-version breaking changes; pin to a release tag in the Dockerfile |
| Per-task container boot overhead makes 49-benchmark suite too slow | Med | Measure in Phase 2 integration test; fall back to per-suite lifecycle if > 20% overhead |
| `ailang` binary inside container out of sync with host version | Low | Build image from same commit hash as eval run; add version check at container startup |

## Related Documents

**Implemented (patterns this builds on):**
- [design_docs/implemented/v0_5_6/m-eval-process-guardrails.md](../implemented/v0_5_6/m-eval-process-guardrails.md) — existing eval process isolation patterns
- [design_docs/implemented/v0_7_0/m-eval-ollama-local-models.md](../implemented/v0_7_0/m-eval-ollama-local-models.md) — precedent for local-only eval infra

**Planned (check for overlap):**
- [design_docs/planned/m-eval-local-ollama.md](m-eval-local-ollama.md) — local Ollama eval (different concern; no overlap)
- [design_docs/planned/v0_13_0/m-cloud-eval-workers.md](v0_13_0/m-cloud-eval-workers.md) — cloud workers (complementary, not overlapping)

## References

- [Apple Container GitHub](https://github.com/apple/container) — source and releases
- [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec) — container image format
- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Evaluation Guide](/docs/docs/guides/evaluation/) — current eval runner docs

## Future Work

- **Pinned image registry**: publish `ailang-eval-base` images to a registry per release, so eval runs can reference `ailang-eval-base:v0.26.0` for exact reproducibility
- **`CapLocalSandbox` for non-Apple platforms**: once Linux has a comparable lightweight VM runtime, the capability abstraction makes adding it straightforward

---

**Document created**: 2026-06-11
**Last updated**: 2026-06-11
**Revisit trigger**: macOS 26 GA release (expected autumn 2026)
