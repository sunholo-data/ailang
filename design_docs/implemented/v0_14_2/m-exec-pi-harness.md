# M-EXEC-PI: Pi Coding Harness Executor

**Status**: Implemented
**Target**: v0.14.2
**Priority**: P2 (Medium-Low — harness diversity for cross-harness eval; not on a critical path)
**Estimated**: 5–6 days (one short sprint: ~½ day fixture-capture spike, ~3.5 days local executor + tests + wiring, ~1.5 days Docker + Cloud Run deployment pillar)
**Dependencies**: None for the local executor pillar. The cloud-deployment pillar depends on [m-executor-variants](../v1_1_0/m-executor-variants.md) (already shipped: per-executor Docker images + Cloud Run Jobs).

**Precedents**:
- [design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) — Go-side executor wiring pattern.
- [design_docs/planned/v1_1_0/m-executor-variants.md](../v1_1_0/m-executor-variants.md) — per-executor Docker image + Cloud Run Job pattern (the second pillar).

**Canonical contract**: [docs/internal/EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — the four-element shape (package layout, required symbols, blank import, `models.yml` wiring) for the in-process Go executor.

**⚠️ Contract gap surfaced by this doc**: `EXECUTOR_SHAPE.md` covers only the local Go-side recipe. Cloud deployment (Docker variant + `cloudbuild-images.yaml` + `cloud_run_jobs.tf`) is documented separately under [m-executor-variants](../v1_1_0/m-executor-variants.md) but is not yet folded into the canonical contract. As part of this work, extend `EXECUTOR_SHAPE.md` with a "Cloud deployment" section so future executors don't re-inherit this gap.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Executor wrapper only; no language semantics changes |
| A2: Replayability | +1 | Adds a fifth harness as a trace/eval dimension; per-executor runs reproducible via fixture replay |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | Auth via existing per-provider env vars; no new ambient capability |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Broadens the agent CLI surface AILANG can drive (15+ providers via one harness) |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Pi cost/token data flows through the same `executor.Result` schema; cross-harness cost comparisons extend cleanly |
| A10: Composability | +1 | Reuses the formalized `EXECUTOR_SHAPE.md` contract verbatim — no new abstractions, no shape drift |
| A11: Structured Failure | 0 | Errors funnel through the existing `Result.Error` path |
| A12: System Boundary | 0 | Subprocess boundary identical to claude/gemini/codex/opencode |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted (provider env vars only, same as existing executors)
- [x] A7 (Machines First): Expands machine audience coverage

---

## Problem Statement

**Current State:**
- Four CLI-subprocess executors are registered: `claude`, `gemini`, `codex`, `opencode` ([internal/executor/factory.go](../../../internal/executor/factory.go)).
- Cross-harness comparisons in `models.yml` exist for {claude, opencode}, {codex, opencode}, {gemini, opencode} pairs via the `model_family` grouping (e.g., `pi-claude-haiku-4-5` mirrors `claude-haiku-4-5`). Adding a fifth harness extends the comparison matrix to a new minimal-design point.
- [pi](https://pi.dev/) (`@mariozechner/pi-coding-agent`) is a deliberately minimal Claude Agent SDK-based harness that supports **15+ providers** (Anthropic, OpenAI, Google, Azure, Bedrock, Mistral, Groq, …) through one consistent CLI surface, with `pi --mode json` emitting NDJSON events for non-interactive use.
- The uniform contract in [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) was explicitly designed so a fifth executor lands as a mechanical exercise. This sprint validates that claim and adds pi as a worked second example after opencode.

**Impact:**
- Without pi, AILANG cannot evaluate the same model under a *minimal* harness vs. a *feature-rich* harness (claude-code, opencode). This is exactly the cross-harness signal the `model_family` design is meant to capture: harness behaviour as an independent variable from model choice.
- Pi's broad provider coverage (Bedrock, Azure, Groq, Mistral) opens routes to models that today have no `agent_cli` route in `models.yml`.
- Validating that a new executor truly slots in via the documented 6-step recipe (rather than re-paying integration tax) is itself the test that the contract is sound.

---

## Goals

**Primary Goal:** Land `pi` as a fifth CLI-subprocess executor that conforms to [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) **and** is deployable to Cloud Run via the m-executor-variants pattern, with no changes to the coordinator dispatch layer or the eval-suite dispatch layer.

**Success Metrics:**
- `executor.GlobalFactory().ListAvailable()` returns `["claude", "codex", "gemini", "opencode", "pi"]` after `pi` is on PATH.
- `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` completes and emits a `RunMetrics` row schema-identical to existing executors.
- Coordinator auto-discovers `pi` via the existing [`ExecutorProvider`](../../../internal/coordinator/provider_executor.go) with **zero** coordinator package changes beyond one blank-import line.
- At least four `model_family` cross-harness pairs land in `models.yml`: `pi-claude-haiku-4-5`, `pi-claude-sonnet-4-6`, `pi-gpt5-4`, `pi-gemini-3-flash-preview` — each grouped with its existing same-model entry under the same `model_family` key.
- The full 6-step recipe from EXECUTOR_SHAPE.md is followed verbatim and any deviation is documented in the implementation report when this doc moves to `implemented/`.
- **Cloud deployment**: `agent-pi:latest` (and optional `agent-pi-go:latest`) image is built by Cloud Build, pushed to Artifact Registry, and a Cloud Run Job referencing it runs end-to-end on a benchmark with secrets injected from Secret Manager.
- **Contract update**: `docs/internal/EXECUTOR_SHAPE.md` gains a "Cloud Deployment" section listing the four cloud-side touchpoints (Dockerfile, cloudbuild step, terraform job, secret bindings) so the recipe is complete for the next executor.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Pi is integrated as a CLI-subprocess executor (not via its `--mode rpc` or SDK embedding) | Preserves the single executor shape; reuses every existing dispatch path; matches claude/gemini/codex/opencode exactly | human | design | high |
| Use `--mode json` (NDJSON event stream), not `-p` (print mode) | NDJSON gives streaming events for `ExecuteStreaming`, token counts, and tool-use telemetry; `-p` returns only final text | human | design | med |
| Capture event-schema fixtures from a real `pi --mode json` run before writing the parser | Pi's JSON event schema is documented externally (`docs/json.md` in the pi-mono repo) but not yet codified in AILANG; replaying captured NDJSON makes parser tests deterministic | human | design | med |
| Initial `models.yml` coverage: `pi-claude-haiku-4-5`, `pi-claude-sonnet-4-6`, `pi-gpt5-4`, `pi-gemini-3-flash-preview` | Matches the existing `model_family` cross-harness pattern; each pi entry pairs with an already-shipping non-pi entry | human | design | low |
| Auth: pass through `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / ADC env vars unchanged into the subprocess | Pi reads provider keys exactly like its underlying SDKs; no new auth surface | agent | compile | low |
| Integration tests gated on `pi` binary presence (skip, not fail, when missing) | Standard pattern from [EXECUTOR_SHAPE.md §Testing Checklist](../../../docs/internal/EXECUTOR_SHAPE.md); CI remains reliable without an npm install step | agent | compile | low |
| Ship a Docker variant `agent-pi` (FROM `agent-base`, `npm install -g @mariozechner/pi-coding-agent`) | Required for Cloud Run coordinator routing; matches `agent-opencode` precedent exactly — same npm-install-on-base shape | human | design | med |
| Defer `agent-pi-go` (Go toolchain variant) to a follow-up unless an AILANG benchmark needs it in this sprint | `-go` variants exist for codex/gemini because AILANG eval benchmarks compile Go inside the agent; pi is not yet wired into those flows | human | design | low |
| Multi-provider secrets: bind `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY` (and ADC for Vertex) into the Cloud Run Job for `agent-pi` | Pi's whole point is multi-provider reach; a single-secret binding would defeat that | human | design | med |

### Design Freeze

Before Sprint 1 starts, these must be resolved:

- [ ] Approve targeting `pi --mode json` over `pi -p` (and explicitly defer `--mode rpc` and SDK embedding — see Non-Goals).
- [ ] Capture a representative NDJSON fixture from a real `pi --mode json` run on a trivial benchmark (e.g., fizzbuzz). Land in `internal/executor/pi/testdata/`. **This is a hard prerequisite for writing the parser** — without a captured event stream, the parser is being written against guesses.
- [ ] Decide model-flag format: `--model <pattern>` vs. `--provider X --model Y` vs. `--model openai/gpt-4o` shorthand. The pi CLI accepts all three; pick one for `agent_model_name` semantics in `models.yml` and document in `internal/executor/pi/README.md`. Provisional pick: provider-prefix shorthand (`openai/gpt-4o`) — matches opencode's existing `agent_model_name: "openai/gpt-5.4"` convention.
- [ ] Approve shipping `agent-pi` Docker variant in this sprint (vs. deferring to a follow-up). Default: ship — leaving it out means the executor works locally but cannot be routed by the Cloud Run coordinator, which is the larger production surface.
- [ ] Confirm whether `agent-pi-go` is needed in this sprint (does any benchmark targeted at pi compile Go inside the agent?). Default: defer — add only if a concrete benchmark requires it.

---

## Solution Design

### Overview

One sprint with **two pillars**:

1. **Local executor** — Apply the 6-step recipe from [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) verbatim to `pi`. No new abstractions; no coordinator or eval-harness dispatch code changes.
2. **Cloud deployment** — Add an `agent-pi` Docker variant, a Cloud Build step, and a Cloud Run Job, mirroring the `agent-opencode` precedent shipped under [m-executor-variants](../v1_1_0/m-executor-variants.md). Update `EXECUTOR_SHAPE.md` with a "Cloud Deployment" section so the contract is complete for the next executor.

### The 6-Step Recipe (Applied to Pi)

1. **Create `internal/executor/pi/`** with `pi.go` implementing the seven `Executor` methods (`Name`, `Execute`, `ExecuteStreaming`, `Capabilities`, `CostModel`, `HealthCheck`, `Close`) plus `Register()` + `init()`. Mirror [`internal/executor/opencode/opencode.go`](../../../internal/executor/opencode/opencode.go) as the structural template — opencode is the closest existing analog (TypeScript-based, multi-provider, NDJSON output).
2. **Write tests** in `internal/executor/pi/pi_test.go`: registration, fixture replay (NDJSON from `testdata/`), mock-binary streaming end-to-end, gated `TestLiveRun_Pi` (skips unless `AILANG_PI_LIVE=1` and `pi` binary on PATH), HealthCheck positive/negative.
3. **Write `internal/executor/pi/README.md`** documenting flags used, auth env vars, event schema, cost model, known limits.
4. **Add one blank import line** to [`internal/coordinator/provider_executor.go`](../../../internal/coordinator/provider_executor.go):
   ```go
   _ "github.com/sunholo-data/ailang/internal/executor/pi"
   ```
5. **Add `models.yml` entries**: `pi-claude-haiku-4-5`, `pi-claude-sonnet-4-6`, `pi-gpt5-4`, `pi-gemini-3-flash-preview`, each with `agent_cli: "pi"` and `agent_model_name: "<provider>/<model>"`. Group each under an existing `model_family` so cross-harness composites (e.g., `harness_suite`) pick them up. Optionally extend the `agent_suite` composite to include the pi entries.
6. **Run** `go test ./internal/executor/pi/... && make test && make lint`. No coordinator code change. No eval-harness code change.

### Components

1. **`internal/executor/pi/pi.go`** — CLI driver: argv construction (`pi --mode json --model <provider>/<model> [--api-key …] <prompt>`), subprocess spawn via `exec.CommandContext`, line-by-line NDJSON parser, mapping events → `EventHandler` callbacks → `Result`.
2. **`internal/executor/pi/testdata/*.ndjson`** — captured event streams from real pi runs, replayed by parser tests for deterministic CI.
3. **`internal/executor/pi/pi_test.go`** — registration, fixture replay, mock-binary streaming, gated live test, healthcheck.
4. **`internal/executor/pi/README.md`** — flags, auth, event schema notes, cost model, limits.
5. **One-line blank import** in `internal/coordinator/provider_executor.go`.
6. **Four model entries** in `internal/eval_harness/models.yml`.
7. **`Config` extension** in [`internal/executor/factory.go`](../../../internal/executor/factory.go): add `PiPath` (default `"pi"`) and `PiModel` (default `"anthropic/claude-haiku-4-5"`) fields and defaults — mirrors the existing `OpenCodePath` / `OpenCodeModel` pattern.

### Implementation Plan

**Phase 0: Fixture-capture spike (~½ day)**
- [ ] Install `pi` (`npm install -g @mariozechner/pi-coding-agent`) on a dev machine with provider keys.
- [ ] Run `pi --mode json -p "write fizzbuzz in python"` and capture NDJSON to `internal/executor/pi/testdata/fizzbuzz.ndjson`.
- [ ] Cross-reference the live event shape against [pi-mono `docs/json.md`](https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent/docs); flag any drift.
- [ ] Capture a tool-use run (e.g., `pi --mode json -p "create file hello.txt"`) to verify tool-call events parse correctly.
- [ ] **Gate**: if the schema is unversioned or unstable, raise this in the implementation report and consider deferring.

**Phase 1: Executor package (~1 day)**
- [ ] Scaffold `internal/executor/pi/pi.go` from the opencode template.
- [ ] Implement `Name()`, `Capabilities()`, `CostModel()`, `HealthCheck()`, `Close()`.
- [ ] Implement argv construction with provider-prefix model shorthand.
- [ ] Implement NDJSON parser (per-event-type switch; tolerate non-JSON preamble; preserve unknowns in `ProviderData`).
- [ ] Implement `Execute()` and `ExecuteStreaming()` (the latter wires events to `EventHandler`).
- [ ] Add `Register()` + `init()` calling `executor.GlobalFactory().Register("pi", builder)`.

**Phase 2: Tests (~½ day)**
- [ ] Registration test (`init()` registers; `Register()` idempotent).
- [ ] Fixture replay test against captured NDJSON; assert turn count, tool-call count, token totals.
- [ ] Mock-binary test (POSIX shell script writes a known NDJSON stream to stdout) — exercises full streaming path without needing real `pi`.
- [ ] Gated `TestLiveRun_Pi` — skipped unless `AILANG_PI_LIVE=1` and `pi` on PATH.
- [ ] HealthCheck positive (mock binary) and negative (bad path).

**Phase 3: Wiring (~½ day)**
- [ ] Add blank import in `internal/coordinator/provider_executor.go`.
- [ ] Add `PiPath` + `PiModel` to `executor.Config` + `DefaultConfig()`.
- [ ] Add four `pi-*` entries to `internal/eval_harness/models.yml` with `agent_cli: "pi"` and matching `model_family` keys.
- [ ] (Optional) extend `agent_suite` and any `harness_suite` composite to include pi entries.
- [ ] Write `internal/executor/pi/README.md`.

**Phase 4: Local verify (~½ day)**
- [ ] `go test ./internal/executor/pi/... -run "."`
- [ ] `make test && make lint`
- [ ] `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` (with `pi` installed + `ANTHROPIC_API_KEY` set) — full end-to-end smoke.
- [ ] Confirm `executor.GlobalFactory().ListAvailable()` includes `"pi"`.

**Phase 5: Docker variant (~½ day)**
- [ ] Author `docker/Dockerfile.agent-pi` mirroring `docker/Dockerfile.agent-opencode` exactly: `FROM agent-base`, `USER root`, `npm install -g @mariozechner/pi-coding-agent`, `USER ailang`.
- [ ] Test build locally: `docker build -f docker/Dockerfile.agent-pi --build-arg PROJECT=<dev-project> -t agent-pi:dev .`
- [ ] Verify `pi --version` runs inside the image.
- [ ] (Optional, gated on Design Freeze decision) author `docker/Dockerfile.agent-pi-go` mirroring `Dockerfile.agent-opencode-go` if it exists, or `Dockerfile.agent-codex-go` otherwise.

**Phase 6: Cloud deployment (~1 day, in `ailang-multivac` repo)**
- [ ] Add `build-agent-pi` + `push-agent-pi` steps to `cloudbuild-images.yaml` (mirror `build-agent-opencode` block).
- [ ] Add Cloud Run Job(s) in `terraform/cloud_run_jobs.tf` referencing `agent-pi:${var.agent_image_tag}`. Match the existing pattern (resource limits, service account, VPC connector).
- [ ] Bind multi-provider secrets (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, ADC) into the new Job — pi's multi-provider story requires all relevant keys present.
- [ ] `terraform plan` then `terraform apply` in the dev project; verify the job is created.
- [ ] Trigger the Cloud Build pipeline; confirm `agent-pi:latest` lands in Artifact Registry.
- [ ] Smoke-test: dispatch a coordinator task with `--executor pi` against the new Cloud Run Job; confirm completion.

**Phase 7: Contract update (~½ day)**
- [ ] Add a "Cloud Deployment" section to `docs/internal/EXECUTOR_SHAPE.md` listing the four cloud-side touchpoints: Dockerfile (and -go variant), `cloudbuild-images.yaml` step, `cloud_run_jobs.tf` job, secret bindings. Reference both this doc and `m-executor-variants` as canonical examples.
- [ ] Bump the recipe from "6 steps" to a clear two-pillar structure (local + cloud) with explicit checklists per pillar.

### Files to Modify/Create

**New files (local executor):**
- `internal/executor/pi/pi.go` — CLI driver, ~500 LOC (mirroring opencode.go's 517 LOC)
- `internal/executor/pi/pi_test.go` — tests, ~250 LOC
- `internal/executor/pi/README.md` — flags, auth, schema, limits, ~150 lines
- `internal/executor/pi/testdata/fizzbuzz.ndjson` — captured event stream
- `internal/executor/pi/testdata/tool_use.ndjson` — captured tool-call stream

**New files (cloud deployment):**
- `docker/Dockerfile.agent-pi` — pi CLI on top of agent-base, ~10 LOC
- `docker/Dockerfile.agent-pi-go` — (optional, only if benchmarks require it) ~15 LOC

**Modified files (this repo):**
- `internal/executor/factory.go` — add `PiPath` + `PiModel` fields and defaults, ~6 LOC
- `internal/coordinator/provider_executor.go` — one blank import line, ~1 LOC
- `internal/eval_harness/models.yml` — four new model entries with `agent_cli: "pi"`, ~60 LOC
- `docs/internal/EXECUTOR_SHAPE.md` — add "Cloud Deployment" section, ~50 lines

**Modified files (in `ailang-multivac` repo, separate PR):**
- `cloudbuild-images.yaml` — add `build-agent-pi` + `push-agent-pi` steps, ~25 LOC
- `terraform/cloud_run_jobs.tf` — add `agent-pi` Cloud Run Job container definitions + secret bindings, ~40 LOC

## Examples

### Example 1: Eval-Suite Cross-Harness Comparison

**Before:**
```bash
# Compare claude-haiku across two harnesses (claude-code vs opencode)
ailang eval-suite --models claude-haiku-4-5,opencode-haiku --benchmarks fizzbuzz
```

**After:**
```bash
# Same model, three harnesses — claude-code (full-featured), opencode (TUI), pi (minimal)
ailang eval-suite --models claude-haiku-4-5,opencode-haiku,pi-claude-haiku-4-5 \
                  --benchmarks fizzbuzz
```

### Example 2: Coordinator Auto-Discovery

**Before:** No diff. `ExecutorProvider` already auto-discovers; nothing to add at coordinator level.

**After:** With `pi` installed and the blank import in place:
```bash
ailang messages send coordinator "Fix bug in fizzbuzz.ail" \
                                 --executor pi --model pi-claude-sonnet-4-6
```
Routes to the pi executor with zero coordinator code change.

### Example 3: `models.yml` Entry (Pattern)

```yaml
pi-claude-haiku-4-5:
  api_name: "claude-haiku-4-5"
  provider: "anthropic"
  description: "Claude Haiku 4.5 via pi minimal harness — cross-harness comparison"
  env_var: "ANTHROPIC_API_KEY"
  agent_cli: "pi"
  agent_model_name: "anthropic/claude-haiku-4-5"   # pi provider/model shorthand
  model_family: "claude-haiku-4-5"   # pairs with claude-haiku-4-5 + opencode-haiku
  max_output_tokens: 64000
  pricing:
    input_per_1k: 0.001
    output_per_1k: 0.005
  notes: |
    Claude Haiku 4.5 evaluated via the pi minimal harness for cross-harness comparison.
    Uses pi --mode json. Pi is a deliberately minimal harness (no MCP, no sub-agents,
    no permission popups), serving as the "thin-harness" baseline against claude-code
    and opencode.
```

## Success Criteria

**Pillar 1 — Local executor:**
- [ ] `executor.GlobalFactory().ListAvailable()` includes `"pi"`.
- [ ] `internal/executor/pi/pi_test.go` passes including fixture replay against real captured NDJSON.
- [ ] `TestLiveRun_Pi` passes when `AILANG_PI_LIVE=1` and `pi` binary on PATH (skipped in standard CI).
- [ ] `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` runs end-to-end and emits a `RunMetrics` row schema-identical to other executors.
- [ ] Coordinator routes a task with `--executor pi` end-to-end (with `pi` installed locally).
- [ ] Four `pi-*` entries in `models.yml` each pair with an existing `model_family` group.
- [ ] All tests passing (`make test`).
- [ ] No new lint warnings (`make lint`).
- [ ] `internal/executor/pi/README.md` documents flags, auth, event schema, cost model, limits.
- [ ] Blank import added to `internal/coordinator/provider_executor.go`.

**Pillar 2 — Cloud deployment:**
- [ ] `docker build -f docker/Dockerfile.agent-pi` succeeds locally; `pi --version` runs inside the image.
- [ ] Cloud Build pipeline produces `agent-pi:latest` in Artifact Registry.
- [ ] `terraform apply` creates an `agent-pi` Cloud Run Job in the dev project with multi-provider secrets bound.
- [ ] A coordinator-dispatched task targeting the `agent-pi` Cloud Run Job completes successfully.

**Contract:**
- [ ] `docs/internal/EXECUTOR_SHAPE.md` has a new "Cloud Deployment" section with the four cloud-side touchpoints (Dockerfile, cloudbuild step, terraform job, secret bindings).
- [ ] Doc moved to `design_docs/implemented/v0_14_2/` with implementation report covering both pillars.

## Testing Strategy

**Unit tests** (`internal/executor/pi/pi_test.go`):
- `TestRegister` — `init()` registers; calling `Register()` again is idempotent.
- `TestParseFizzbuzzFixture` — replay `testdata/fizzbuzz.ndjson`; assert NumTurns, ToolCallCount, InputTokens, OutputTokens.
- `TestParseToolUseFixture` — replay `testdata/tool_use.ndjson`; assert tool events fire on the handler.
- `TestParseTolerateNonJSONPreamble` — feed a stream with leading non-JSON noise; assert clean parse.
- `TestArgvConstruction` — build argv for various model flag formats; assert exact flag order.
- `TestHealthCheck_OK` — mock binary responds; assert nil.
- `TestHealthCheck_BadPath` — path doesn't exist; assert error.

**Integration tests** (mock binary):
- `TestStreamingEndToEnd_MockBinary` — POSIX shell script stand-in writes a known NDJSON stream; full pipeline (Execute → handler callbacks → Result) verified.

**Live test** (gated):
- `TestLiveRun_Pi` — skipped unless `AILANG_PI_LIVE=1` and `which pi` succeeds. Hits a real provider with a trivial prompt; asserts non-empty `Result.Output` and `Success == true`.

**Manual verification:**
- `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` with pi installed and provider keys set.
- `ailang chains list` after the run; confirm a row exists with `executor: pi`.
- `ailang chains view <chain-id> --spans`; confirm tool-use events were captured.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Tool allowlist passthrough** — pi supports `--tools <list>` and `--no-tools`. Decide at implementation time whether `Task.AllowedTools` maps to `--tools` or whether tools are always permitted (matching opencode's current behaviour). Agent may choose; document in README.
- **Thinking-level mapping** — pi exposes `--thinking <off|minimal|low|medium|high|xhigh>`. Decide whether `Task.Effort` ("low"/"medium"/"high") maps directly or via a translation table. Agent may choose; document.
- **Cost-model granularity** — provider-aware cost lookup vs. delegating to the underlying provider's `models.yml` pricing. Agent may choose; deferring to `models.yml` pricing is simpler and matches opencode.

## Non-Goals

**Not attempted in this feature:**
- **`pi --mode rpc`** — JSON-RPC over stdin/stdout for embedded use. Not needed; the subprocess + NDJSON path covers every existing dispatch site (coordinator + eval-suite). RPC mode adds a long-lived process and bidirectional protocol with no eval-harness consumer.
- **Pi SDK embedding** — `@mariozechner/pi-coding-agent` SDK in-process via Node bindings. Crosses a language boundary (Go ↔ Node) for no measurable benefit over subprocess invocation.
- **Pi extensions / custom TypeScript modules** — pi's extensibility model is out of scope; we use it as a black-box CLI.
- **Provider expansion beyond the initial four** — Bedrock, Azure, Mistral, Groq, etc. can be added as additional `models.yml` entries in a follow-up once the executor is proven on the four anchor models.
- **Cross-harness comparison report templates** — orthogonal to executor wiring; tracked under [design_docs/planned/v0_15_0/m-eval-cross-harness-comparison.md](../v0_15_0/m-eval-cross-harness-comparison.md).

## Timeline

**Days 1–2** (~1.5 days): Phase 0 + Phase 1
- Fixture-capture spike against live `pi`.
- Scaffold and implement `internal/executor/pi/pi.go`.

**Day 3** (~1 day): Phase 2
- Tests (registration, fixture replay, mock binary, gated live, healthcheck).

**Day 4 morning** (~½ day): Phase 3
- Blank import, `Config` fields, `models.yml` entries, README.

**Day 4 afternoon** (~½ day): Phase 4 + Phase 5
- Local `make test && make lint`, eval-suite smoke run.
- `docker/Dockerfile.agent-pi` + local docker build verify.

**Day 5** (~1 day): Phase 6
- `cloudbuild-images.yaml` + `terraform/cloud_run_jobs.tf` updates in `ailang-multivac`.
- Apply, build, smoke-test Cloud Run Job dispatch.

**Day 6** (~½ day): Phase 7
- Update `docs/internal/EXECUTOR_SHAPE.md` with "Cloud Deployment" section.
- Move doc to `implemented/`.

**Total: ~5 days across one sprint** (3.5 days local + 1.5 days cloud + contract update)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pi NDJSON schema is undocumented in detail (only summary in `pi.dev`; full schema lives in `docs/json.md` of pi-mono repo) | Medium | Phase 0 fixture-capture spike against the real CLI is a hard prerequisite for parser work. If the schema is unversioned or visibly unstable, document and defer. |
| `npm install -g @mariozechner/pi-coding-agent` adds a Node toolchain dependency for live testing | Low | Standard pattern: live tests are gated on `AILANG_PI_LIVE=1` and binary presence. CI runs only the fixture/mock paths, which need no Node. |
| Pi event schema may drift between pi versions (pi is pre-1.0) | Medium | Pin a tested pi version in README; capture fixtures from that version; preserve unknown fields in `ProviderData` so forward-compat is best-effort but not load-bearing. |
| Pi tool-call event shape differs from claude/codex/opencode patterns enough to require rethinking `EventHandler` | Low–Medium | Phase 0 captures a tool-use fixture explicitly. If the shape is genuinely incompatible with the existing handler interface, escalate as a contract change to EXECUTOR_SHAPE.md before writing the parser. |
| Pi defaults to interactive TUI; non-interactive flags interact in non-obvious ways (`-p` vs `--mode json`) | Low | Phase 0 spike resolves; document exact invocation in README. |
| Auth surface differs across providers (Bedrock uses AWS creds, ADC for Vertex, etc.) | Low | Initial four model targets all use simple env-var auth (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, ADC). Provider expansion is deferred. |
| Cloud-deployment work crosses repo boundaries (`ailang` → `ailang-multivac`); two PRs in lockstep | Medium | Land local executor PR first; cloud PR second. Tag the cloud PR with the local-executor commit SHA in the description so the rollback story is clear. |
| Cloud Build / Terraform changes are environment-specific (dev vs prod project) | Medium | Land in dev project first; verify smoke-test passes; only then promote to prod. Match the staging pattern documented under [m-executor-variants](../v1_1_0/m-executor-variants.md). |
| `EXECUTOR_SHAPE.md` cloud-deployment section becomes stale as Cloud Run / Terraform layout evolves | Low | Section stays high-level (touchpoints + canonical examples), pointing at `m-executor-variants` for current detail. Each new executor's design doc carries the up-to-date specifics. |

## Related Documents

<!-- Auto-populated by Ollama neural search on "exec pi harness" -->

**Implemented (precedent for this work):**
- [design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) — direct precedent: added codex + opencode under the same uniform shape this doc applies.
- [design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md) — sprint structure to mirror.
- [design_docs/implemented/v0_8_1/m-process-exec.md](../../implemented/v0_8_1/m-process-exec.md) (0.44) — process exec primitives.
- [design_docs/implemented/v0_8_0/m-telemetry-hooks-handoff.md](../../implemented/v0_8_0/m-telemetry-hooks-handoff.md) (0.41) — telemetry hooks pattern.

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-exec-hierarchy-refactor.md](../v0_13_0/m-exec-hierarchy-refactor.md) (0.40) — executor hierarchy work; orthogonal but worth a sanity check.
- [design_docs/planned/v0_13_0/m-copilot-cli-integration.md](../v0_13_0/m-copilot-cli-integration.md) (0.39) — another CLI-harness integration; same pattern.
- [design_docs/planned/v0_15_0/m-eval-cross-harness-comparison.md](../v0_15_0/m-eval-cross-harness-comparison.md) — pi entries become inputs to this comparison work.

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [docs/internal/EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — the four-element contract this doc applies
- [internal/executor/opencode/opencode.go](../../../internal/executor/opencode/opencode.go) — closest structural template (TypeScript-based, multi-provider, NDJSON output)
- [internal/executor/codex/codex.go](../../../internal/executor/codex/codex.go) — alternate template (NDJSON parser with cumulative token totals)
- [internal/eval_harness/models.yml](../../../internal/eval_harness/models.yml) — see existing `opencode-*` entries for the cross-harness pairing pattern
- [pi.dev](https://pi.dev/) — official pi homepage
- [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent) — pi source + `docs/json.md` event schema
- npm: `@mariozechner/pi-coding-agent`

## Implementation Report

Sprint completed 2026-04-27 (single working day, ~5 hours active dev time
across 7 milestones). All success criteria met.

### What Was Built

| Milestone | LOC actual | Estimated | Notes |
|---|---|---|---|
| M1 — Fixture spike | ~180 | ~50 | Captured live fixtures (20 + 37 events) + executor README documenting full schema upfront |
| M2 — Executor core | 522 | ~500 | `pi.go` modeled on opencode; per-turn deltas summed from `message_end` |
| M3 — Tests | 680 | ~250 | 26 tests, 78.8% coverage (over 70% target); fixture replay validates real schema |
| M4 — Wiring + smoke | 110 | ~220 | One-line blank import + 4 `pi-*` models.yml entries + harness_suite update; smoke passed 1/1 |
| M5 — Dockerfile | 10 | 10 | Mirrors `agent-opencode`; only npm package name differs |
| M6 — Cloud deploy | 274 + 156 | ~65 | 274 LOC tf + cloudbuild-images, plus 156 LOC for the systemic cloudbuild.yaml fix |
| M7 — Contract + CHANGELOG | ~80 | ~50 | EXECUTOR_SHAPE.md refinements, CHANGELOG entry, implementation report |
| **Total** | **~2,012 LOC** | **~1,145** | 1.76× the estimate, driven mostly by the M6 systemic fix that wasn't scoped |

### Cloud Smoke Test (M6)

End-to-end verification:
- `pi-smoke-1777275079` (anthropic/claude-haiku-4-5): expected fail — "No API key found for anthropic" — confirms cost-control policy works as designed
- `pi-smoke-openai-1777275552` (openai/gpt-5.4-mini): **completed**, 2 turns, $0.0046

### Deviations from Plan

1. **`agent-pi-go` Docker variant** — deferred per Design Freeze decision; no AILANG benchmark currently targets pi with Go-toolchain needs.
2. **`agent_suite` membership** — pi NOT added (kept the 4-member shape). `harness_suite` got `pi-claude-sonnet-4-6` only.
3. **Cost-control adjustment** — `agent_executor_pi` ended up binding only `OPENAI_API_KEY` + `GEMINI_API_KEY`, not `ANTHROPIC_API_KEY`. The original design doc's "bind all three" was overruled mid-sprint per CLAUDE.md §5; pi-claude-* models work locally only.
4. **Unscoped systemic fix (in scope per "first application" decision)** — `cloudbuild.yaml` was missing 8 of 16 image build steps. Added them all so future executor variants land cleanly via the auto-trigger pipeline.

### Lessons Captured in Contract

- **EXECUTOR_SHAPE.md §6** — both `cloudbuild.yaml` AND `cloudbuild-images.yaml` must be updated for new variants (the historical drift between them caused this sprint's terraform-apply failure; explicit guard added).
- **EXECUTOR_SHAPE.md §8** — cost-control rule for cloud secret bindings: claude uses free OAuth (never bind ANTHROPIC_API_KEY); multi-provider executors with no OAuth path bind only the API-keyed providers explicitly approved for billing. Pi precedent now documented as the canonical example.
- **Recipe step 9** — explicit "add to `knownVariants`" step in `internal/dispatch/cloudrun/dispatcher.go`. Easy to miss; pi originally compiled fine without it but the coordinator would have rejected `--executor pi` at dispatch time.

## Future Work

- **Provider expansion** — once pi is proven on the four anchor models, add Bedrock, Azure, Groq, Mistral entries to `models.yml` to extend the agent-mode coverage matrix.
- **Cross-harness analysis** — feed pi entries into [m-eval-cross-harness-comparison.md](../v0_15_0/m-eval-cross-harness-comparison.md) so the "minimal-harness vs. full-harness" axis becomes a first-class report dimension.
- **Pi extension authoring** — pi's TypeScript extension surface could host an AILANG-specific telemetry/microrag shim parallel to the Claude Code bash-hook frontend (M4A/M7A in M-EXEC-EXPAND). Out of scope here; revisit if pi adoption justifies it.

---

**Document created**: 2026-04-26
**Last updated**: 2026-04-26

---

DESIGN_DOC_PATH: design_docs/planned/v0_14_2/m-exec-pi-harness.md
