# Sprint Plan: M-EXEC-PI (Pi Coding Harness Executor)

**Status**: Planned
**Target**: v0.14.2
**Estimated**: 5 working days (~1 week, single sprint with internal parallelization)
**Priority**: P2
**Role**: Adds a fifth CLI-subprocess executor (`pi`) following the uniform shape established by [M-EXEC-EXPAND](../../implemented/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md). First sprint to exercise the **two-pillar contract** (local executor + cloud deployment) end-to-end; updates [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md) so the contract is complete for the next executor.

## Source-of-Truth Design Doc

- [m-exec-pi-harness.md](m-exec-pi-harness.md) — full design (axiom scoring, Pillar 1 + Pillar 2 phases, files, risks). This plan drives execution against that doc.

---

## Why This Sprint (and Why Now)

- **Validates the two-pillar contract**: `EXECUTOR_SHAPE.md` was extended this week to fold in the cloud-deployment pillar. Pi is the first executor authored against that updated contract — if anything is unclear, surfacing it here is much cheaper than discovering it on the sixth executor.
- **Cross-harness signal**: pi is deliberately minimal (no MCP, no sub-agents, no permission popups). Pairing pi-claude-haiku-4-5 with claude-haiku-4-5 (claude-code, full-featured) and opencode-haiku (opencode, mid-featured) gives the `model_family` axis a clean three-point baseline for "harness richness vs. completion rate."
- **Cheap follow-on leverage**: once pi is wired, the 11 additional providers it supports (Bedrock, Azure, Groq, Mistral, …) become incremental `models.yml` entries with no executor work.
- **Low risk**: the recipe is now mature (5th executor, second worked example after opencode). The only genuine unknown is pi's NDJSON event schema — gated by Phase 0.

---

## Velocity Calibration

**Reference points** (from [internal/executor/](../../../internal/executor/)):
- `opencode/opencode.go` = 517 LOC impl + 200+ LOC tests, shipped in M-EXEC-EXPAND Sprint 2 (Days 8–10, ~3 days for the core)
- `codex/codex.go` = 665 LOC impl + 280+ LOC tests, shipped in M-EXEC-EXPAND Sprint 1 (Days 1–3)
- M-EXEC-EXPAND blended: ~2,620 LOC across 11.5 days = **~228 LOC/day**

**Why this sprint is faster than precedent:**
- Contract is mature (second worked example after opencode, not the first)
- No `EXECUTOR_SHAPE.md` authoring (already exists; just append a "Cloud Deployment" was already done as a parallel edit — only refinements needed)
- Microrag frontends (M4A/M7A in M-EXEC-EXPAND, ~350 LOC) are out of scope here
- Docker variant pattern is established (mirror `Dockerfile.agent-opencode` exactly, 10 LOC)

**Planning target**: 200 LOC/day sustained.

**Total sprint estimate**: ~1,095 LOC (impl + tests + docs + Dockerfile + cloudbuild + terraform) → 5 days at 200 LOC/day with the Phase 0 spike absorbing half a day.

---

## Milestone Breakdown

Seven milestones across one week, structured: **fixture spike → executor core → tests → local wiring → Docker variant → cloud deployment → contract update**.

| # | ID | Title | Est. LOC | Depends on | Parallel with |
|---|----|-------|----------|-----------|----------------|
| M1 | M1_FIXTURE_SPIKE | Capture pi NDJSON fixtures + design-freeze gate | ~50 | — | — |
| M2 | M2_EXECUTOR_CORE | `pi.go` driver + NDJSON parser | 500 | M1 | M5, M6 (drafts) |
| M3 | M3_TESTS | Registration + fixture replay + mock binary + gated live test | 250 | M2 | — |
| M4 | M4_LOCAL_WIRING | `factory.go` config + blank import + `models.yml` + README + local smoke | 220 | M3 | — |
| M5 | M5_DOCKER_VARIANT | `Dockerfile.agent-pi` + local docker build verify | 10 | M1 (npm pkg name confirmed) | M2, M3 |
| M6 | M6_CLOUD_DEPLOY | `cloudbuild-images.yaml` + `cloud_run_jobs.tf` + secrets in `ailang-multivac` | 65 | M5 (image must build) | M3, M4 (drafts) |
| M7 | M7_CONTRACT_UPDATE | Refine `EXECUTOR_SHAPE.md` Cloud Deployment section + CHANGELOG | 50 | M6 | — |

**Total**: ~1,145 LOC (impl + tests + README + Dockerfile + cloudbuild + terraform + contract refinements)
**Schedule**: 5 working days with parallelization

---

## Parallelization Map

The two pillars are **largely independent** once M1 (fixture spike) confirms the executor is viable. Concretely:

```
Day 1: M1 (gate)
Day 2:  ┌─ M2 (executor core) ────────────────┐
        └─ M5 (Dockerfile draft) ─ verify ──┘  (~30 min, then idle until M6)
Day 3:  ┌─ M2 finishes / M3 (tests) ─────────┐
        └─ M6 (cloudbuild + tf drafts) ─────┘  (no apply yet — needs image first)
Day 4: M4 (local wiring + smoke), then M5 final docker push, M6 apply + smoke
Day 5: M7 (contract refinements + CHANGELOG + doc move)
```

**Why this works**:
- M5 (Dockerfile) is 10 LOC mirroring `agent-opencode` exactly; it doesn't read pi's parser code.
- M6 (cloudbuild + terraform) is YAML + HCL referencing image names; it doesn't depend on Go internals.
- Both can be authored in parallel with M2/M3 by the same agent (independent file edits) or by a second agent on a separate branch.

**The only true serialization point**: M6 cannot `terraform apply` until M5's image is in Artifact Registry. So M6 is *drafted* in parallel and *applied* after M5 completes.

---

## Risk Gates

### Gate 1 — End of M1 (Fixture Spike): Schema Viability

**Decision criterion**: Did `pi --mode json` produce a parseable, stable NDJSON event stream?

- **Pass** → Proceed to M2.
- **Fail (no JSON mode, only print mode)** → Halt sprint. File a message documenting the gap; consider deferring until pi adds a stable schema.
- **Marginal (schema works but is unversioned/visibly fluid)** → Proceed but capture *two* fixtures from different prompts to bound the variance; document the version pinning explicitly in `internal/executor/pi/README.md`.

### Gate 2 — End of M5 (Docker Build): Image Viability

**Decision criterion**: Does `agent-pi:dev` build locally, and does `pi --version` succeed inside the container?

- **Pass** → Proceed to M6.
- **Fail (npm install fails inside agent-base)** → Likely a Node.js version mismatch with `agent-base`. Either pin a Node version in `Dockerfile.agent-pi` or escalate to update `agent-base`. Do not proceed to M6 with a broken image.

### Gate 3 — End of M6 (Cloud Smoke): End-to-End Routing

**Decision criterion**: Does a coordinator-dispatched task with `--executor pi` against the Cloud Run Job complete successfully in dev?

- **Pass** → Proceed to M7 (contract update + doc move).
- **Fail (job runs but errors)** → Capture the failure mode (auth? missing secret? schema drift?) and triage. Cloud-pillar success criteria stay open until resolved; M7 can ship contract refinements regardless.

---

## Sprint — M-EXEC-PI (Days 1–5)

### M1: Fixture Capture + Design-Freeze Gate

**Goal**: Install pi, capture representative NDJSON fixtures, confirm the executor is buildable. **This is a hard prerequisite for M2** — writing a parser without a captured event stream is writing against guesses.

**Files created:**
- `internal/executor/pi/testdata/fizzbuzz.ndjson` — captured `pi --mode json -p "write fizzbuzz in python"` output
- `internal/executor/pi/testdata/tool_use.ndjson` — captured `pi --mode json -p "create file hello.txt"` output

**Tasks:**
- [ ] `npm install -g @mariozechner/pi-coding-agent`; pin version
- [ ] Set `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` locally
- [ ] Run two trial prompts; capture stdout to `testdata/`
- [ ] Cross-reference the live event shape against pi-mono `docs/json.md`; flag drift in a brief notes file
- [ ] Decide model-flag format (provisional pick: provider-prefix shorthand `openai/gpt-4o`)
- [ ] Confirm npm package name + binary name (`pi`) for M5

**Acceptance criteria:**
- [ ] Two NDJSON fixtures land at `internal/executor/pi/testdata/`
- [ ] Each fixture is non-empty and parses as line-delimited JSON
- [ ] Tool-use fixture contains at least one tool-call event
- [ ] Schema notes documented inline in `internal/executor/pi/README.md` (stub OK, expanded in M4)
- [ ] **Gate decision recorded**: pass / marginal / fail with rationale

**Dependencies**: none
**Estimate**: 50 LOC (fixture lines + README stub) — ~½ day

---

### M2: Pi Executor Core

**Goal**: Author `internal/executor/pi/pi.go` by copy-modifying `internal/executor/opencode/opencode.go` (closest structural analog: TypeScript-based, multi-provider, NDJSON output); port the parser against the M1 fixtures.

**Files created:**
- `internal/executor/pi/pi.go` — Executor implementation (~500 LOC, mirroring [opencode.go](../../../internal/executor/opencode/opencode.go))

**Tasks:**
- [ ] Scaffold from `opencode.go`: copy file, rename type/function/literal references
- [ ] Implement `Name()`, `Capabilities()`, `CostModel()`, `HealthCheck()`, `Close()`
- [ ] Implement argv construction with provider-prefix model shorthand (e.g., `--model anthropic/claude-haiku-4-5`)
- [ ] Implement NDJSON parser: per-event-type switch driven by M1 fixtures; tolerate non-JSON preamble; preserve unknown fields in `Result.ProviderData`
- [ ] Implement `Execute()` and `ExecuteStreaming()` (the latter wires events to `EventHandler`)
- [ ] Add `Register()` + `init()` calling `executor.GlobalFactory().Register("pi", builder)`

**Acceptance criteria:**
- [ ] All 7 methods of `executor.Executor` implemented
- [ ] Parser accepts both M1 fixtures without panic
- [ ] Token totals (`InputTokens`, `OutputTokens`) populated from terminal events
- [ ] `make build` succeeds; package compiles
- [ ] No `default:` returning fake data in any switch (per project rules)

**Dependencies**: M1 (fixtures are the parser's specification)
**Estimate**: 500 LOC — ~1 day

---

### M3: Tests

**Goal**: Cover registration, fixture replay, mock-binary streaming end-to-end, gated live run, healthcheck.

**Files created:**
- `internal/executor/pi/pi_test.go` — ~250 LOC

**Tasks:**
- [ ] `TestRegister` — `init()` registers; calling `Register()` again is idempotent
- [ ] `TestParseFizzbuzzFixture` — replay `testdata/fizzbuzz.ndjson`; assert NumTurns, ToolCallCount, InputTokens, OutputTokens
- [ ] `TestParseToolUseFixture` — replay `testdata/tool_use.ndjson`; assert tool events fire on the handler
- [ ] `TestParseTolerateNonJSONPreamble` — feed a stream with leading non-JSON; assert clean parse
- [ ] `TestArgvConstruction` — build argv for various model flag formats; assert exact flag order
- [ ] `TestStreamingEndToEnd_MockBinary` — POSIX shell script writes a known NDJSON stream; full pipeline verified
- [ ] `TestHealthCheck_OK` (mock binary) and `TestHealthCheck_BadPath` (negative)
- [ ] `TestLiveRun_Pi` — gated on `AILANG_PI_LIVE=1` and `pi` on PATH; trivial prompt, asserts non-empty output

**Acceptance criteria:**
- [ ] `go test ./internal/executor/pi/... -run "."` passes locally
- [ ] Test coverage ≥ 70% for `internal/executor/pi/`
- [ ] `TestLiveRun_Pi` skips cleanly with a clear message when binary or env-var absent
- [ ] `make lint` clean

**Dependencies**: M2
**Estimate**: 250 LOC — ~½ day

---

### M4: Local Wiring + Smoke

**Goal**: Wire pi into the global config, coordinator blank-import list, models.yml; document; smoke-test end-to-end through eval-suite.

**Files modified:**
- [`internal/executor/factory.go`](../../../internal/executor/factory.go) — add `PiPath` (default `"pi"`) and `PiModel` (default `"anthropic/claude-haiku-4-5"`) fields and defaults (~6 LOC)
- [`internal/coordinator/provider_executor.go`](../../../internal/coordinator/provider_executor.go) — one blank import line (1 LOC)
- [`internal/eval_harness/models.yml`](../../../internal/eval_harness/models.yml) — four new entries (~60 LOC):
  - `pi-claude-haiku-4-5` (model_family: `claude-haiku-4-5`)
  - `pi-claude-sonnet-4-6` (model_family: `claude-sonnet-4-6`)
  - `pi-gpt5-4` (model_family: `gpt5-4`)
  - `pi-gemini-3-flash-preview` (model_family: `gemini-3-flash`)

**Files created:**
- `internal/executor/pi/README.md` — flags used, auth env vars, event schema notes, cost model, known limits (~150 LOC)

**Tasks:**
- [ ] `factory.go` Config + DefaultConfig updates
- [ ] Blank import in `provider_executor.go`
- [ ] Four `pi-*` entries in `models.yml`; verify each pairs with an existing `model_family`
- [ ] **Composite-suite policy** (verified against current `models.yml`, lines 864–909):
  - Add `pi-claude-sonnet-4-6` to `harness_suite` (extends the sonnet-4-6 family from 2 to 3 columns — headline cross-harness signal)
  - **Do not** add pi entries to `lang_harness_suite` in this sprint (would raise post-release Explorer cost ~25%; gate on a separate decision once pi cost data exists)
  - **Do not** add pi to `agent_suite` in this sprint (current 4-member shape is "one rep per harness"; expanding to 5 is a separate calibration call)
- [ ] Author `internal/executor/pi/README.md` (use `internal/executor/opencode/README.md` as template)
- [ ] Local smoke: `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` (with `pi` installed + `ANTHROPIC_API_KEY` set)
- [ ] `executor.GlobalFactory().ListAvailable()` includes `"pi"`

**Acceptance criteria:**
- [ ] `make test && make lint` clean
- [ ] Local smoke run produces a `RunMetrics` row schema-identical to other executors
- [ ] `provider_executor.go` diff is **exactly 1 line added** (the blank import)
- [ ] `executor.GlobalFactory().ListAvailable()` returns 5 names: claude, codex, gemini, opencode, pi
- [ ] README documents flags, auth, schema, cost model, known limits

**Dependencies**: M3
**Estimate**: ~220 LOC (60 YAML + 6 Go + 1 import + 150 README + smoke verify) — ~1 day

---

### M5: Docker Variant

**Goal**: Author `docker/Dockerfile.agent-pi` mirroring `Dockerfile.agent-opencode`; verify locally.

**Files created:**
- `docker/Dockerfile.agent-pi` — ~10 LOC, identical structure to `Dockerfile.agent-opencode` with the npm package swapped

**Tasks:**
- [ ] Author Dockerfile (`FROM agent-base`, `USER root`, `npm install -g @mariozechner/pi-coding-agent`, `USER ailang`)
- [ ] Local build: `docker build -f docker/Dockerfile.agent-pi --build-arg PROJECT=<dev-project> -t agent-pi:dev .`
- [ ] Verify `pi --version` runs inside the image
- [ ] (Defer per Design Freeze decision) Skip `Dockerfile.agent-pi-go` unless an AILANG benchmark targeted at pi compiles Go inside the agent

**Acceptance criteria:**
- [ ] `docker build` succeeds
- [ ] `docker run --rm agent-pi:dev pi --version` outputs a version string
- [ ] Dockerfile diff vs `Dockerfile.agent-opencode` is minimal (only npm package name differs)

**Dependencies**: M1 (npm package name confirmed)
**Estimate**: 10 LOC — ~½ day (mostly verification, not authoring)

**Parallelizable**: yes — can be drafted on Day 2 in parallel with M2/M3.

---

### M6: Cloud Deployment (`ailang-multivac` repo)

**Goal**: Add a Cloud Build step that produces `agent-pi:latest`; add a Cloud Run Job referencing it; bind multi-provider secrets; smoke-test end-to-end.

**Files modified (in [`ailang-multivac`](https://github.com/sunholo-data/ailang-multivac) repo, separate PR):**
- `cloudbuild-images.yaml` — add `build-agent-pi` + `push-agent-pi` steps (mirror `build-agent-opencode`); ~25 LOC
- `terraform/cloud_run_jobs.tf` — add `agent-pi` Cloud Run Job container blocks with multi-provider secret bindings (~40 LOC)

**Tasks:**
- [ ] Author `build-agent-pi` + `push-agent-pi` steps in `cloudbuild-images.yaml` mirroring the opencode block exactly (same `--build-arg PROJECT=$_TARGET_PROJECT`, same registry path)
- [ ] Author Cloud Run Job container block in `terraform/cloud_run_jobs.tf` referencing `${local.image_base}/agent-pi:${var.agent_image_tag}`
- [ ] Bind secrets: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `GOOGLE_APPLICATION_CREDENTIALS` (mounted file)
- [ ] `terraform plan` then `terraform apply` in dev project; verify Job is created
- [ ] Trigger Cloud Build pipeline; confirm `agent-pi:latest` lands in Artifact Registry
- [ ] Smoke-test: dispatch a coordinator task with `--executor pi` against the Cloud Run Job; confirm completion

**Acceptance criteria:**
- [ ] Cloud Build run produces `agent-pi:latest` in Artifact Registry
- [ ] `terraform apply` creates the `agent-pi` Cloud Run Job in dev with all four secrets bound
- [ ] Coordinator-dispatched smoke task completes successfully end-to-end
- [ ] Cross-repo PR description tags the `ailang` PR's commit SHA for traceability

**Dependencies**: M5 (image must build locally first)
**Estimate**: 65 LOC — ~1 day (cross-repo, includes apply + smoke)

**Parallelizable**: drafts can land on Day 3 in parallel with M3 — apply waits on M5.

---

### M7: Contract Refinement + CHANGELOG + Doc Move

**Goal**: Refine `EXECUTOR_SHAPE.md` based on lessons from this sprint (any missed touchpoints, ambiguous flag mappings); land CHANGELOG entry; move design doc to `implemented/v0_14_2/`.

**Files modified:**
- [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md) — refinements only (the Cloud Deployment section was added during the design-doc work; M7 captures any gaps surfaced by actually running the recipe) (~30 LOC)
- [`changelogs/v0.10-current.md`](../../../changelogs/v0.10-current.md) — v0.14.2 entry detailing M-EXEC-PI (~20 LOC)

**Files moved:**
- `design_docs/planned/v0_14_2/m-exec-pi-harness.md` → `design_docs/implemented/v0_14_2/m-exec-pi-harness.md` (status: Implemented)
- `design_docs/planned/v0_14_2/m-exec-pi-harness-sprint-plan.md` → `design_docs/implemented/v0_14_2/m-exec-pi-harness-sprint-plan.md`

**Tasks:**
- [ ] Audit `EXECUTOR_SHAPE.md` against the actual sequence of work performed; add any missing touchpoints
- [ ] If any contract item was ambiguous and led to a mistake during the sprint, document the resolution
- [ ] CHANGELOG entry under v0.14.2 covering Pillar 1 (executor) + Pillar 2 (cloud) + contract update
- [ ] Update front-matter of `m-exec-pi-harness.md`: Status → Implemented
- [ ] Add implementation report section to the design doc (what was actually built, deviations, LOC actuals)
- [ ] Move both docs to `design_docs/implemented/v0_14_2/`

**Acceptance criteria:**
- [ ] `EXECUTOR_SHAPE.md` reflects all touchpoints actually used
- [ ] CHANGELOG entry references both this sprint plan and the design doc
- [ ] Both docs landed in `implemented/v0_14_2/`
- [ ] `make ci` passes

**Dependencies**: M6 (cloud smoke must pass before declaring sprint complete)
**Estimate**: 50 LOC (refinements + CHANGELOG + status updates) — ~½ day

---

## Success Metrics (measurable on v0.14.2 release)

**Pillar 1 — Local executor:**
- [ ] `executor.GlobalFactory().ListAvailable()` returns `["claude", "codex", "gemini", "opencode", "pi"]` (order-independent)
- [ ] Four `pi-*` entries in `models.yml` each pair with an existing `model_family` group
- [ ] [`provider_executor.go`](../../../internal/coordinator/provider_executor.go) gains exactly 1 line (one blank import)
- [ ] `ailang eval-suite --models pi-claude-haiku-4-5 --benchmarks fizzbuzz` produces a `RunMetrics` row schema-identical to other executors
- [ ] `TestLiveRun_Pi` skips cleanly when binary absent
- [ ] Test coverage ≥ 70% for `internal/executor/pi/`

**Pillar 2 — Cloud deployment:**
- [ ] `agent-pi:latest` exists in Artifact Registry (dev project)
- [ ] Cloud Run Job `agent-pi` exists in dev with all four provider secrets bound
- [ ] Coordinator-dispatched smoke task targeting `agent-pi` Cloud Run Job completes end-to-end

**Contract:**
- [ ] `EXECUTOR_SHAPE.md` reflects all cloud-side touchpoints actually exercised
- [ ] CHANGELOG entry under v0.14.2

---

## High-Impact Decisions (carried from design doc)

| Decision | Resolved | Note |
|----------|----------|------|
| Pi as CLI-subprocess executor (not RPC/SDK) | ✅ | Design doc §High-Impact Decisions |
| Use `--mode json` over `-p` | ✅ | M1 captures fixtures via `--mode json` |
| Capture fixtures before parser work | ✅ | M1 is design-freeze gate |
| Initial 4 model entries (haiku-4-5, sonnet-4-6, gpt5-4, gemini-3-flash) | ✅ | M4 |
| Provider-prefix shorthand for `agent_model_name` | ⏳ | Confirmed in M1 |
| Auth via env-var passthrough only | ✅ | M2 + M6 secret bindings |
| Ship `agent-pi` Docker variant in this sprint | ✅ | M5 |
| Defer `agent-pi-go` to follow-up | ✅ | Per Design Freeze gate |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pi NDJSON schema is unversioned / fluid (pre-1.0 CLI) | Medium | M1 is a hard gate; capture two fixtures from different prompts; pin pi version in README; preserve unknown fields in `ProviderData` |
| `npm install -g @mariozechner/pi-coding-agent` fails inside `agent-base` (Node version mismatch) | Medium | M5 Gate 2; if this fires, either pin Node version in `Dockerfile.agent-pi` or escalate `agent-base` update |
| Tool-call event shape differs from claude/codex/opencode patterns enough to require rethinking `EventHandler` | Low–Medium | M1 captures a tool-use fixture explicitly; if shape is incompatible, escalate as a contract change before writing the parser |
| Cloud-deployment cross-repo PR (`ailang-multivac`) lags behind local-executor PR | Medium | M5/M6 drafts begin Day 2; tag cloud PR with local PR's SHA; dev-only apply first, prod promotion is post-sprint |
| Multi-provider secret binding misconfigured (missing one key) | Low | M6 acceptance criterion explicitly verifies all four secrets present in Cloud Run Job env |
| Schema discovery in M1 reveals pi has no usable JSON mode | High | Halt sprint cleanly at M1 Gate 1; design doc remains in `planned/`; revisit when pi adds stable schema |
| LOC overrun on `pi.go` (500 LOC estimate) | Low | opencode reference is 517 LOC → estimate matches; buffer is ~½ day on Day 5 |

---

## Timeline

**Day 1**: M1 — Fixture capture + design-freeze gate (~½ day; remaining ½ day = M5 Dockerfile draft if Gate 1 passes)

**Day 2**: M2 — Pi executor core (~1 day, ~500 LOC)
- Parallel: M5 Dockerfile authored + local build verified (~½ hour)

**Day 3**: M3 — Tests (~½ day, ~250 LOC); start M4 wiring (~½ day)
- Parallel: M6 cloudbuild + terraform drafts (~½ day; not applied yet)

**Day 4**: M4 finishes — local smoke (~½ day); M5 image push to Artifact Registry; M6 `terraform apply` + cloud smoke (~½ day)

**Day 5**: M7 — Contract refinements + CHANGELOG + doc move (~½ day); buffer (~½ day)

**Total: ~1,145 LOC across 5 working days** (Pillar 1 ≈ 990 LOC, Pillar 2 ≈ 75 LOC, contract ≈ 50 LOC, fixtures ≈ 30 LOC)

---

## Non-Goals (from design doc)

- **`pi --mode rpc`** — JSON-RPC over stdin/stdout for embedded use
- **Pi SDK embedding** — Node bindings in-process
- **Pi extensions / custom TypeScript modules** — out of scope
- **Provider expansion beyond the initial four** (Bedrock, Azure, Mistral, Groq) — incremental `models.yml` additions in a follow-up
- **`agent-pi-go` Docker variant** — defer until a benchmark requires Go toolchain inside the pi container
- **Cross-harness comparison report templates** — tracked under [m-eval-cross-harness-comparison.md](../v0_15_0/m-eval-cross-harness-comparison.md)
- **Microrag frontend for pi** — orthogonal; if needed, ship in a follow-up modeled on M-EXEC-EXPAND M4A/M7A

---

## Follow-on Sprints

After this sprint lands, two tracks become easier:

1. **Pi provider expansion** — add Bedrock/Azure/Groq/Mistral entries to `models.yml` (incremental; no executor work).
2. **Cross-harness analysis** — feed pi entries into [m-eval-cross-harness-comparison.md](../v0_15_0/m-eval-cross-harness-comparison.md) so the "minimal-harness vs. full-harness" axis becomes a first-class report dimension.

---

**Created**: 2026-04-27
**Author**: sprint-planner (Claude Opus 4.7)
