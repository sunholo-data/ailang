# CLI-Subprocess Executor Shape

This is the uniform contract every CLI-subprocess executor in AILANG follows
(claude, codex, opencode, pi, motoko). Note the `managed_agents` executor is
HTTP/SSE, not a CLI subprocess, so it follows the executor interface but not the
stdout-parsing parts of this contract — see `internal/executor/managed_agents/`.

The contract has **two pillars**:

- **Pillar 1 — Local executor** (this repo, in-process Go): four elements that
  make a new executor auto-discovered by both the coordinator
  (`internal/coordinator/provider_executor.go`) and the eval harness
  (`cmd/ailang/eval_suite.go`) with zero dispatch-layer code changes.
- **Pillar 2 — Cloud deployment** (`docker/` here + `ailang-multivac` repo):
  four cloud-side touchpoints that make the same executor routable through the
  Cloud Run coordinator. Required for any executor used in production cloud
  dispatch; optional if the executor is local-only.

The total touch points outside the new executor package are: one-line blank
import in `provider_executor.go`, an `agent_cli` string in `models.yml`,
one Dockerfile under `docker/`, **two Cloud Build steps** (one in
`ailang-multivac/cloudbuild.yaml` and one in `ailang-multivac/cloudbuild-images.yaml`),
and one Cloud Run Job block in `ailang-multivac/terraform/cloud_run_jobs.tf`.

## Pillar 1 — Local Executor

The four elements below make a new executor auto-discovered locally
(coordinator + eval harness). For local-only executors, this is sufficient.
For executors that must run in Cloud Run, also complete Pillar 2 below.

## 1. Package Layout

```
internal/executor/<name>/
  <name>.go        # CLI driver: flag building, subprocess spawn, stream parse
  <name>_test.go   # Unit tests + fixture-driven streaming test + gated live test
  README.md        # Flags, auth, event schema, cost model, known limits
  testdata/        # (optional) NDJSON fixtures replayed by tests
```

Keep the package name equal to the executor name so the blank import reads
cleanly: `_ "github.com/sunholo-data/ailang/internal/executor/codex"`.

## 2. Required Symbols

The package **must** export these:

| Symbol | Signature | Purpose |
|---|---|---|
| `New(cfg *executor.Config) (*<Name>Executor, error)` | constructor | Reads `cfg.<Name>Path` / `cfg.<Name>Model`; applies defaults |
| `Register()` | `func()` | Calls `executor.GlobalFactory().Register("<name>", builder)` |
| `init()` | package init | Calls `Register()` |

The `*<Name>Executor` type **must** implement every method of
`executor.Executor` (see `internal/executor/executor.go`):

```go
Name() string
Execute(ctx, task) (*Result, error)
ExecuteStreaming(ctx, task, handler) (*Result, error)
Capabilities() []Capability
CostModel() *CostModel
HealthCheck(ctx) error
Close() error
```

**Canonical references:**
- `internal/executor/claude/claude.go:764-773` — `Register()` + `init()` pattern
- `internal/executor/codex/codex.go:541-548` — same pattern for Codex CLI
- `internal/executor/opencode/opencode.go:592-601` — same pattern for opencode CLI
- `internal/executor/pi/pi.go` — same pattern for pi CLI (multi-provider, no Go toolchain)
- `internal/executor/motoko/motoko.go:267-278` — same pattern for motoko_agent (AILANG-native; reads JSONL from `.motoko/logfile/` rather than stdout — see package README for the schema-v1 contract)

### Streaming Parser Contract

`ExecuteStreaming` reads the CLI's stdout line-by-line. Each provider emits a
different NDJSON shape, so each package has its own parser. All parsers must:

- Tolerate non-JSON lines (preamble chatter, warnings) — skip cleanly
- Use `json.RawMessage` for payloads with shifting schemas
- Preserve unknown fields in `ProviderData map[string]any` on the result
- Report final token counts to `Result.InputTokens` / `Result.OutputTokens`

**Codex-specific note:** tokens are emitted as **cumulative running totals**
per message (matching OpenAI API semantics), not per-turn deltas. Use `max()`
when aggregating, not sum (see `codex.go` — message branch uses
`if ev.Tokens.Input > inputTokens` pattern).

## 3. Coordinator Wiring (Blank Import)

Add **exactly one line** to [`internal/coordinator/provider_executor.go`](../../internal/coordinator/provider_executor.go):

```go
import (
    _ "github.com/sunholo-data/ailang/internal/executor/claude"
    _ "github.com/sunholo-data/ailang/internal/executor/codex"   // <-- add
    _ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
)
```

That's it. `ExecutorProvider` auto-discovers any name registered in the
factory via `NewExecutorProvider("<name>")`. No switch statement, no
constructor factory, no coordinator changes.

## 4. Models.yml Wiring

In [`internal/eval_harness/models.yml`](../../internal/eval_harness/models.yml),
set each model's `agent_cli` to the executor name:

```yaml
models:
  gpt5:
    api_name: gpt-5
    provider: openai
    agent_cli: "codex"           # <-- maps model to executor
    agent_model_name: "gpt-5"    # <-- optional; flag passed as --model
    ...
```

- `agent_cli: null` = model is text-only (eval-suite standard mode only)
- `agent_cli: "<name>"` = model supports agent-mode eval via that executor
- Add the model to the `agent_suite` composite if it should appear in
  `ailang eval-suite --models agent_suite`

---

## Pillar 2 — Cloud Deployment

For an executor to be routable through the **Cloud Run coordinator** (production
dispatch surface), four additional touchpoints are required. Skip this pillar
only if the executor is local-only (developer laptop or self-hosted CI).

The canonical precedent is [`design_docs/planned/v1_1_0/m-executor-variants.md`](../../design_docs/planned/v1_1_0/m-executor-variants.md),
which introduced per-executor Docker variants and Cloud Run Jobs.
[`docker/Dockerfile.agent-opencode`](../../docker/Dockerfile.agent-opencode) is
the simplest concrete template for a CLI-only variant
(no Go toolchain); [`docker/Dockerfile.agent-codex-go`](../../docker/Dockerfile.agent-codex-go)
shows the `-go` variant pattern when the agent must compile Go inside the
container (e.g., for AILANG benchmarks).

### 5. Docker Variant

Add `docker/Dockerfile.agent-<name>` mirroring `Dockerfile.agent-opencode`:

```dockerfile
ARG PROJECT
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

USER root
RUN <install command for the CLI>   # e.g., npm install -g @vendor/cli
USER ailang
```

If a benchmark targeted at this executor needs the Go toolchain (e.g., it
compiles AILANG → Go inside the agent), also add `Dockerfile.agent-<name>-go`
mirroring `Dockerfile.agent-codex-go`. Most new executors do **not** need the
`-go` variant; defer it until a concrete benchmark requires it.

### 6. Cloud Build Step (BOTH `cloudbuild.yaml` AND `cloudbuild-images.yaml`)

**⚠️ Critical**: a new executor variant must be added to **both** Cloud Build
configs in `ailang-multivac/`:

1. **`cloudbuild.yaml`** — the auto-trigger pipeline that runs on push to
   dev/test/prod. This is the file that builds images **before** running
   `terraform apply`. If a new variant is missing here, `terraform apply` will
   fail with `Image 'agent-<name>:latest' not found.` for the new Cloud Run
   Job, and Cloud Run will cache that failure (`ContainerMissing` condition)
   until the next successful apply re-validates the resource.
2. **`cloudbuild-images.yaml`** — the manual image-only pipeline (no
   terraform, no deploy). Useful for rebuilding images without paying for a
   full deploy. Must stay in sync with `cloudbuild.yaml` so manual rebuilds
   produce the same image set as auto-triggered runs.

In each file, add a `build-agent-<name>` step (and `push-agent-<name>` if a
downstream `-go` variant `FROM`s it). Mirror the `build-agent-opencode`
block exactly — same `--build-arg PROJECT=$_TARGET_PROJECT`, same registry path
(`${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-<name>:latest`).
Also add the new image to the `push-images` `waitFor` list and the top-level
`images:` declaration so it's recorded as a build artifact.

The historical drift between the two files (cloudbuild.yaml missing all
executor variants for several months) is exactly the kind of silent breakage
this contract is designed to prevent. Updating both is non-negotiable.

### 7. Cloud Run Job

In [`ailang-multivac/terraform/cloud_run_jobs.tf`](https://github.com/sunholo-data/ailang-multivac),
add Cloud Run Job container blocks referencing
`${local.image_base}/agent-<name>:${var.agent_image_tag}`. Match the existing
pattern used by `agent-opencode` / `agent-codex` / `agent-pi`:
resource limits, service account, VPC connector, env, secret bindings.

### 8. Secret Bindings

Bind the provider API keys the executor may use into the Cloud Run Job's
`env { value_source { secret_key_ref { ... } } }` blocks. Standard secrets:

- `ANTHROPIC_API_KEY` — Anthropic models
- `OPENAI_API_KEY` — OpenAI models
- `GEMINI_API_KEY` — Gemini API (alternate to ADC)
- `GOOGLE_APPLICATION_CREDENTIALS` (mounted file) — Vertex AI via ADC

**⚠️ Cost-control rule**: the **claude** executor uses a free Claude
Code OAuth token (`CLAUDE_CODE_OAUTH_TOKEN`) — it MUST NOT bind
`ANTHROPIC_API_KEY` (per `ailang-multivac/CLAUDE.md` §5; API-key billing
is pay-per-token and a busy day of agent runs can cost hundreds of
dollars). For multi-provider executors that have no OAuth path
(e.g., pi), bind only the API-keyed providers you've decided to allow
billing for; other models become dispatch-time errors with a clear
"No API key found for X" message — exactly the failure mode you want.

**Pi precedent**: `agent_executor_pi` deliberately binds only
`OPENAI_API_KEY` + `GEMINI_API_KEY`. Pi-claude-* models remain
runnable LOCALLY (where the developer's own key is in env) but fail
fast in cloud — preventing surprise Anthropic bills.

### Cloud Deployment Checklist

For a new cloud-deployable executor:

1. `docker/Dockerfile.agent-<name>` builds locally with
   `docker build -f docker/Dockerfile.agent-<name> --build-arg PROJECT=<dev-project> -t agent-<name>:dev .`
2. `<cli> --version` succeeds inside the built image
3. **Both** `cloudbuild.yaml` AND `cloudbuild-images.yaml` produce
   `agent-<name>:latest` in Artifact Registry (not just one!)
4. `terraform apply` creates an `agent-<name>` Cloud Run Job in dev
5. Coordinator-dispatched task targeting the new Job completes end-to-end
6. Promote to prod after dev smoke passes

---

## Authentication Patterns for Executors

Each executor has its own auth surface. The patterns break into three tiers:

| Tier | Method | Best for |
|---|---|---|
| **API key** | `EXECUTOR_API_KEY` env var | CI/CD, coordinator daemon, cloud workers |
| **Browser OAuth** | `<cli> login` | Developer laptop with browser |
| **Device OAuth** | `<cli> login --device-auth` | Headless / SSH / remote machines |

**The coordinator should always use env-var auth** — it is stateless, survives
container restarts, and requires no browser or cached session files on worker nodes.

**For interactive developer machines without a browser** (cloud VM, SSH session):
```bash
codex login --device-auth   # OAuth2 Device Authorization Grant (RFC 8628)
                             # Prints URL + code; authorize on any device with browser
```

Per-executor summary:

| Executor | Env var | Device flow | Notes |
|---|---|---|---|
| `claude` | `ANTHROPIC_API_KEY` | `claude login --device-auth` (Claude Pro) | Claude Code uses OAuth for subscription billing |
| `codex` | `OPENAI_API_KEY` | `codex login --device-auth` | ChatGPT Plus session OR API key; device flow for headless |
| `opencode` | `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / ADC | provider-dependent | opencode Zen subscription optional; direct provider keys work |

## Testing Checklist

For a new executor package:

1. **Registration test** — `init()` registers with factory, `Register()` idempotent
2. **Parser test** — replay a fixture NDJSON, assert token/turn/event counts
3. **Mock binary test** — POSIX shell script stand-in exercises the streaming
   path end-to-end without requiring the real CLI installed
4. **Gated live test** — `TestLiveRun_<Name>` skips unless
   `AILANG_<NAME>_LIVE=1` is set **and** the binary exists on PATH
5. **HealthCheck test** — positive case with mock, negative case with bad path

See `internal/executor/codex/codex_test.go` for the complete blueprint.

## Adding a New Executor: Two-Pillar Recipe

### Pillar 1 — Local Executor (always required)

1. Create `internal/executor/<name>/` with `<name>.go` implementing the 7
   `Executor` methods plus `Register()` + `init()`
2. Write fixture replay + mock binary tests; add gated live test
3. Write `internal/executor/<name>/README.md` (flags, auth, schema, limits)
4. Add one blank import line to `internal/coordinator/provider_executor.go`
5. Flip any `agent_cli: null` lines in `models.yml` to `"<name>"` for models
   served by this CLI; add to `agent_suite` if cross-harness eval is desired
6. Run `go test ./internal/executor/<name>/... && make test && make lint`

### Pillar 2 — Cloud Deployment (required for Cloud Run dispatch)

7. Author `docker/Dockerfile.agent-<name>` (mirror `Dockerfile.agent-opencode`);
   verify `docker build` + `<cli> --version` locally
8. Add `build-agent-<name>` (+ `push-agent-<name>` if needed) to **BOTH**
   `ailang-multivac/cloudbuild.yaml` AND `ailang-multivac/cloudbuild-images.yaml`.
   Update each file's `push-images.waitFor` and `images:` lists.
9. Add `"<name>"` to `knownVariants` in
   `internal/dispatch/cloudrun/dispatcher.go` so the coordinator accepts the
   new variant in `DispatchParams.ExecutorVariant`
10. Add a Cloud Run Job block in
    `ailang-multivac/terraform/cloud_run_jobs.tf` (one for the project-keys
    variant, one for the user-API-key variant) with the policy-appropriate
    secret bindings — see §8 above for the cost-control rule
11. Smoke-test in dev: build pipeline → `terraform apply` → coordinator
    dispatch with `--executor <name>` → completion

No coordinator code change. No eval-harness code change. No factory
modifications. The registration runs at import time, and both
`ExecutorProvider` and `eval-suite` resolve names dynamically. Cloud
deployment is purely declarative (Dockerfile + cloudbuild + terraform).

## Why this shape?

Three historical forces shaped this:

- **Coordinator auto-discovery** — an earlier refactor replaced a
  switch-statement factory with `NewExecutorProvider(name)`, so adding an
  executor is a single blank import (see the post-refactor note in
  `design_docs/planned/v0_13_0/m-coord-codex-executor.md`)
- **Eval harness decoupling** — `agent_cli` in `models.yml` is the single
  source of truth for executor routing; `eval-suite` expands composites
  (`agent_suite`, `benchmark_suite`, `dev_models`) at dispatch time
- **Schema drift tolerance** — every provider evolves its JSON shape
  independently; keeping parsers per-package and using `ProviderData` for
  forward-compat means a schema change in one vendor never touches the others

## Related Documents

- [.claude/rules/coordinator.md](../../.claude/rules/coordinator.md) — coordinator daemon + agent workflow
- [`design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md`](../../design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md) — Pillar 1 precedent: added Codex + opencode local executors
- [`design_docs/planned/v1_1_0/m-executor-variants.md`](../../design_docs/planned/v1_1_0/m-executor-variants.md) — Pillar 2 precedent: per-executor Docker variants + Cloud Run Jobs
- [`design_docs/planned/v0_14_2/m-exec-pi-harness.md`](../../design_docs/planned/v0_14_2/m-exec-pi-harness.md) — first design doc structured around the two-pillar recipe (pi executor)
- `design_docs/planned/v0_13_0/m-coord-codex-executor.md` — original (pre-refactor) proposal; superseded
