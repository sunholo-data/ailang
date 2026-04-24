# M-EXECUTOR-VARIANTS: Per-Agent Docker Image Variants

**Status**: Planned
**Target**: v1.1.0
**Priority**: P2 — quality-of-life; current monolithic image is a temporary workaround
**Estimated**: 2–3 days
**Dependencies**: None (coordinator code already wires codex/opencode executors)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Image tags are pinned strings; variant resolution is a pure map lookup |
| A2: Replayability | +1 | Variant name logged per execution; jobs are reproducible from task record |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | +1 | Each variant declares exactly which CLIs are present — no ambient surprises |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Variant is machine-readable in config; no guessing from agent name |
| A8: Minimal Syntax | 0 | One new config field (`executor_variant`); no new language syntax |
| A9: Cost Visibility | +1 | Smaller images → lower pull latency; explicit variant → auditable runtime cost |
| A10: Composability | +1 | Variants layer via `FROM`; adding a new variant doesn't touch existing images |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | +1 | Makes the container tool boundary explicit and config-driven |

**Net Score: +7** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

---

## Problem Statement

**Current State:**

All Cloud Run agent jobs run the same `agent:latest` Docker image. The current image contains:

- Node.js 22 (required by all CLI tools)
- Claude CLI (`@anthropic-ai/claude-code`)
- Gemini CLI (installed with `|| true` — fragile silent failure)
- **Go toolchain 1.25** (~800MB) — only needed by `sprint-executor` to build/test the ailang repo
- `ailang` binary
- `ailang_bootstrap` plugin pre-clone

The Go toolchain was baked in as a workaround after agents OOM'd downloading Go at runtime. This bloated every agent job, even those that only edit markdown.

**Codex and opencode executors exist in Go code but not in any image.** The coordinator (`internal/coordinator/provider_executor.go`) imports both packages at compile time. `AILANG_EXECUTOR=codex` and `AILANG_EXECUTOR=opencode` will resolve correctly at runtime — but the CLIs are not installed in any Docker image, so those tasks will fail.

**Impact:**
- Every agent pulls ~2GB even if it only needs Node + one CLI
- Gemini CLI install is silently optional (`|| true`), making failures invisible
- No path to using codex or opencode executors in cloud tasks
- Future executor additions (aider, copilot) will further bloat the single image

---

## Goals

**Primary Goal:** Replace the monolithic `agent:latest` image with a small set of purpose-built variants, each containing only the tools a given agent needs.

**Success Metrics:**
- `agent:latest` (claude, no Go) is ≤ 600MB uncompressed
- `agent-go:latest` (claude + Go) is ≤ 1.5GB
- `sprint-executor` uses `agent-go:latest`; markdown agents use `agent:latest`
- Codex and opencode CLIs are installable via a dedicated variant
- A new executor variant can be added via new Dockerfile + one dispatcher map entry — no coordinator code change required
- All existing agents continue working after rollout (zero-downtime migration)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Variant naming convention | Names appear in config files, CI, and monitoring | human | design | high |
| Image layering DAG | Layering order affects build parallelism and cache invalidation | agent | design | high |
| How `executor_variant` maps to image tag | Determines dispatcher code and config contract | human | design | med |
| Auth secrets per variant | Wrong secrets = broken agents in prod | human | design | high |
| Migration: keep current image or hard-cut | Determines rollout risk | human | deploy | med |

### Design Freeze

Before implementation begins:

- [x] **Variant naming convention** — chosen: `{executor}[-go]` pattern (see Variant Matrix below)
- [x] **Image layering DAG** — chosen: `base → {executor} → {executor}-go` (see Dockerfile section)
- [x] **Auth model** — chosen: per-variant secrets section in Terraform; same GCS config, different Secret Manager entries per executor (see Auth section)
- [x] **Migration approach** — **hard cut**: current `agent:latest` becomes `agent-go:latest` (rename in CI tag); new slim `agent:latest` is built from `Dockerfile.agent` without Go. No parallel rollout needed — current cloud agents are not in active production use.

---

## Solution Design

### Overview

Each agent in `config.cloud.yaml` declares an `executor_variant` string. The dispatcher resolves this to a Docker image tag and passes it as a container override when launching the Cloud Run Job. No new job templates are needed — the existing `agent-executor` job supports image overrides via the Cloud Run Jobs API.

The variant name encodes which executor CLI(s) are installed and whether the Go toolchain is present. The `ailang` binary is present in all variants.

### Variant Matrix

| Variant | Image tag | Executor CLIs | Go | AILANG binary | Primary use |
|---------|-----------|---------------|----|---------------|-------------|
| `default` | `agent:latest` | claude | — | ✓ | docs, markdown, packages |
| `go` | `agent-go:latest` | claude | ✓ | ✓ | ailang repo, Go codebases |
| `gemini` | `agent-gemini:latest` | gemini | — | ✓ | gemini-provider agents |
| `gemini-go` | `agent-gemini-go:latest` | gemini | ✓ | ✓ | gemini + Go repo |
| `codex` | `agent-codex:latest` | codex | — | ✓ | OpenAI-based agents |
| `codex-go` | `agent-codex-go:latest` | codex | ✓ | ✓ | codex + Go repo |
| `opencode` | `agent-opencode:latest` | opencode | — | ✓ | opencode-based agents |
| `eval` | `agent-eval:latest` | claude + gemini + codex + opencode | — | ✓ | eval-suite, cross-harness benchmarks |
| `eval-go` | `agent-eval-go:latest` | all CLIs | ✓ | ✓ | eval-suite against Go repos |

**Omitting `executor_variant`** defaults to `"default"` → `agent:latest`.

### Agent → Variant Mapping (initial)

| Agent | Variant | Reason |
|-------|---------|--------|
| `design-doc-creator` | `default` | Markdown only; Claude CLI |
| `sprint-planner` | `default` | Markdown only; Claude CLI |
| `sprint-executor` | `go` | Runs `make test`, `go build` on ailang repo |
| `stapledon-design-doc` | `default` | Markdown; no Go needed (non-Go repo) |
| `stapledon-sprint-planner` | `default` | Same |
| `stapledon-sprint-executor` | `default` | Stapledons Voyage is not a Go repo |
| `website-builder` | `default` | Node/HTML edits; Claude CLI |
| `eval-suite` (future) | `eval-go` | Needs all CLIs + Go for ailang benchmarks |
| `codex-agent` (future) | `codex` | OpenAI executor |
| `opencode-agent` (future) | `opencode` | opencode executor |

### Architecture

#### 1. Docker image DAG

```
Dockerfile.agent-base  (Node 22 + git + curl + ailang binary + bootstrap clone)
    │
    ├─ Dockerfile.agent          → agent:latest      (+ claude CLI)
    │       │
    │       └─ Dockerfile.agent-go  → agent-go:latest  (+ Go toolchain)
    │
    ├─ Dockerfile.agent-gemini   → agent-gemini:latest  (+ gemini CLI)
    │       │
    │       └─ Dockerfile.agent-gemini-go → agent-gemini-go:latest (+ Go)
    │
    ├─ Dockerfile.agent-codex    → agent-codex:latest   (+ codex CLI)
    │       │
    │       └─ Dockerfile.agent-codex-go  → agent-codex-go:latest  (+ Go)
    │
    ├─ Dockerfile.agent-opencode → agent-opencode:latest (+ opencode CLI)
    │
    └─ Dockerfile.agent-eval     → agent-eval:latest    (all CLIs)
            │
            └─ Dockerfile.agent-eval-go → agent-eval-go:latest (+ Go)
```

The base image carries everything every agent always needs. Executor images layer one CLI on top. Go images layer on top of their executor image. This maximises Docker layer caching: a Go toolchain change only rebuilds the Go-suffix images; a claude CLI update only rebuilds claude-family images.

#### 2. Coordinator config (`config/config.cloud.yaml` in `ailang-multivac`)

Add optional `executor_variant` field per agent. Omitting it defaults to `"default"`.

```yaml
coordinator:
  agents:
    - id: sprint-executor
      executor_variant: go       # needs go build/test
      provider: claude
      ...

    - id: design-doc-creator
      # executor_variant omitted → defaults to "default" (agent:latest)
      provider: claude
      ...

    - id: eval-suite-agent
      executor_variant: eval-go  # needs all CLIs + Go
      provider: gemini            # AILANG_EXECUTOR can still be set per-task
      ...
```

#### 3. Agent registry (`internal/coordinator/agent_registry.go`)

Add field to `AgentConfig`:

```go
type AgentConfig struct {
    // ... existing fields ...
    ExecutorVariant string `yaml:"executor_variant" json:"executor_variant,omitempty"`
}
```

#### 4. Dispatch params (`internal/coordinator/cloud_dispatcher.go`)

Add field to `DispatchParams`:

```go
type DispatchParams struct {
    // ... existing fields ...
    ExecutorVariant string // e.g. "default", "go", "codex", "eval-go"
}
```

Wire it in `daemon_tasks_exec.go` where `DispatchParams` is built from the `AgentConfig`.

#### 5. Dispatcher (`internal/dispatch/cloudrun/dispatcher.go`)

**Implementation note:** The Cloud Run Jobs API `ContainerOverride` does not support an `Image` field — only `Name`, `Args`, `Env`, `ClearArgs`. Per-execution image selection is done via **job template selection** (same pattern as the existing OAuth/apikey split).

Each variant maps to a separate Cloud Run Job in Terraform with the image baked in. The dispatcher selects the job by name:

```go
// knownVariants is the set of valid executor_variant values.
// Job naming: {prefix}-agent-executor[-{variant}][-apikey]
var knownVariants = map[string]bool{
    "": true, "default": true, "go": true,
    "gemini": true, "gemini-go": true,
    "codex": true, "codex-go": true,
    "opencode": true, "eval": true, "eval-go": true,
}

func jobSuffixForVariant(variant, authMode string) (string, error) {
    if !knownVariants[variant] {
        return "", fmt.Errorf("unknown executor_variant %q — check config.cloud.yaml", variant)
    }
    suffix := "agent-executor"
    if variant != "" && variant != "default" {
        suffix += "-" + variant
    }
    if authMode == "apikey" {
        suffix += "-apikey"
    }
    return suffix, nil
}
```

Replace the existing dual-suffix logic:

```go
jobSuffix, err := jobSuffixForVariant(params.ExecutorVariant, params.AuthMode)
if err != nil {
    return err
}
jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s-%s",
    d.projectID, d.region, d.prefix, jobSuffix)
```

Examples: `("go", "oauth")` → `{prefix}-agent-executor-go`; `("codex", "apikey")` → `{prefix}-agent-executor-codex-apikey`.

**Terraform implication:** Each variant requires its own Cloud Run Job in `cloud_run_jobs.tf` with the image baked in. Use `for_each` over a variant map to minimise repetition.

### Dockerfiles

#### `docker/Dockerfile.agent-base` (new)

```dockerfile
# AILANG Agent Base — minimal shared layer
# No executor CLIs. Every variant builds FROM this.

# ── Build stage ──────────────────────────────────────────────
FROM golang:1.25-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /ailang ./cmd/ailang/

# ── Runtime stage ────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git curl gnupg \
    && rm -rf /var/lib/apt/lists/*

# Node.js (required by all AI CLI tools)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# ailang binary
COPY --from=builder /ailang /usr/local/bin/ailang

# Git identity for automated commits
RUN git config --system user.name "ailang-agent" \
    && git config --system user.email "ailang-agent@noreply.github.com"

# Pre-clone shared skills plugin
ARG AILANG_PLUGIN_REPO=https://github.com/sunholo-data/ailang_bootstrap.git
RUN git clone --depth 1 ${AILANG_PLUGIN_REPO} /plugins/ailang_bootstrap 2>/dev/null || true

ENV COORDINATOR_MODE=cloud

RUN useradd -m -s /bin/bash ailang \
    && mkdir -p /workspace /plugins \
    && chown -R ailang:ailang /workspace /plugins

USER ailang
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/ailang"]
CMD ["coordinator", "execute-job"]
```

#### `docker/Dockerfile.agent` (updated — slim, no Go)

```dockerfile
# AILANG Agent — Claude CLI executor (default variant)
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code
```

#### `docker/Dockerfile.agent-go` (new — replaces Go in Dockerfile.agent)

```dockerfile
# AILANG Agent (Go variant) — Claude CLI + Go toolchain
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent:latest

COPY --from=golang:1.25-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/ailang/go"
```

#### `docker/Dockerfile.agent-gemini` (new)

```dockerfile
# AILANG Agent (Gemini variant) — Gemini CLI executor
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

# Gemini CLI — install must succeed (no || true)
RUN npm install -g @google/gemini-cli
```

#### `docker/Dockerfile.agent-codex` (new)

```dockerfile
# AILANG Agent (Codex variant) — OpenAI Codex CLI executor
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

# OpenAI Codex CLI
RUN npm install -g @openai/codex
```

#### `docker/Dockerfile.agent-opencode` (new)

```dockerfile
# AILANG Agent (opencode variant) — opencode executor
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

# opencode CLI (binary release or npm — verify package name before implementing)
RUN npm install -g opencode-ai
```

#### `docker/Dockerfile.agent-eval` (new)

```dockerfile
# AILANG Agent (eval variant) — all executor CLIs for cross-harness benchmarks
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

RUN npm install -g @anthropic-ai/claude-code \
    @google/gemini-cli \
    @openai/codex \
    opencode-ai
```

**Go variants** (`Dockerfile.agent-gemini-go`, `Dockerfile.agent-codex-go`, `Dockerfile.agent-eval-go`) all follow the same pattern as `Dockerfile.agent-go`: `FROM` the executor image + `COPY golang`.

### Auth Patterns per Variant

**Critical:** The agent executor uses `CLAUDE_CODE_OAUTH_TOKEN` for Claude auth — NOT `ANTHROPIC_API_KEY`. The OAuth token uses the Claude Max subscription (flat fee). Using the API key is pay-per-token and can produce large bills.

| Variant | Auth mechanism | Secret Manager key | Notes |
|---------|---------------|-------------------|-------|
| `default`, `go` | `CLAUDE_CODE_OAUTH_TOKEN` | `{prefix}-claude-code-oauth-token` | Current; do not change |
| `gemini`, `gemini-go` | Application Default Credentials | Workload Identity on Cloud Run SA | Same ADC as coordinator |
| `codex`, `codex-go` | `OPENAI_API_KEY` | `{prefix}-openai-api-key` (new secret) | Injected via Cloud Run job env |
| `opencode`, `opencode-go` | Provider-dependent | `{prefix}-openai-api-key` or OAuth | opencode supports multiple providers; wire per deployment |
| `eval`, `eval-go` | All of the above | All secrets | Job SA must have access to all secrets |

The Terraform `cloud_run_jobs.tf` in `ailang-multivac` must add the new secrets and mount them into the appropriate job template(s). Since all variants share a single job template (container override selects the image), secrets must be available in that job's environment — variants that don't use a secret will simply ignore it.

**Alternative:** Create separate job templates per executor family (`agent-executor-claude`, `agent-executor-gemini`, etc.) and select the template in the dispatcher alongside the variant. This avoids mounting unused secrets but adds Terraform complexity. Deferred decision — agent may choose which approach based on secret count at implementation time.

### CI: Build order (`cloudbuild-images.yaml` in `ailang-multivac`)

Images must be built in dependency order. Add to the `steps` array:

```yaml
# 1. Base image (no executor CLI, no Go) — parallel with coordinator/dashboard
- id: build-agent-base
  name: gcr.io/cloud-builders/docker
  args: [build, -t, "${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-base:latest",
         -f, /workspace/ailang/docker/Dockerfile.agent-base,
         --build-arg, PROJECT=$_TARGET_PROJECT, /workspace/ailang]
  waitFor: ['clone-ailang']

# 2. Executor-specific images — each waits for base
- id: build-agent          # claude CLI (slim — no Go)
  waitFor: ['build-agent-base']
  # ... args as above

- id: build-agent-gemini
  waitFor: ['build-agent-base']
  # ...

- id: build-agent-codex
  waitFor: ['build-agent-base']
  # ...

- id: build-agent-opencode
  waitFor: ['build-agent-base']
  # ...

- id: build-agent-eval
  waitFor: ['build-agent-base']
  # ...

# 3. Go variants — each waits for its executor image to be pushed
# (because Dockerfile.agent-go uses FROM agent:latest)
- id: push-agent-base-family   # push base + executor images first
  # ...

- id: build-agent-go
  waitFor: ['push-agent-base-family']   # needs agent:latest pushed first
  # ...

- id: build-agent-gemini-go
  waitFor: ['push-agent-base-family']
  # ...

# etc. for codex-go, eval-go
```

**Important:** Go variant Dockerfiles use `FROM .../agent:latest` (pulls from registry). They cannot be built in parallel with `agent:latest` — they must wait for it to be pushed. This adds one sequential step to the CI pipeline. The push-before-build pattern is already established by `build-coordinator` in the existing `cloudbuild-images.yaml`.

---

## Implementation Plan

**Phase 1: Base image + slimmed default** (~6 hours)

- [ ] Create `docker/Dockerfile.agent-base` (extract current Dockerfile.agent minus CLIs and Go)
- [ ] Update `docker/Dockerfile.agent` to `FROM agent-base + claude CLI install`
- [ ] Create `docker/Dockerfile.agent-go` (`FROM agent + Go toolchain`)
- [ ] Update `cloudbuild-images.yaml` to build base → agent → agent-go in order
- [ ] Update `cloudbuild-trigger-ailang.yaml` similarly
- [ ] Add `executor_variant` to `AgentConfig` in `internal/coordinator/agent_registry.go`
- [ ] Add `ExecutorVariant` to `DispatchParams` in `internal/coordinator/cloud_dispatcher.go`
- [ ] Wire variant from agent config → dispatch params in `daemon_tasks_exec.go`
- [ ] Add `variantImages` map + `imageForVariant()` to `internal/dispatch/cloudrun/dispatcher.go`
- [ ] Pass `image` in `ContainerOverride` in dispatcher
- [ ] Set `executor_variant: go` on `sprint-executor` in `config.cloud.yaml`
- [ ] Add `AILANG_IMAGE_BASE` env var to coordinator Cloud Run service (Terraform)

**Phase 2: Codex + opencode variants** (~4 hours)

- [ ] Verify npm package names for codex and opencode CLIs
- [ ] Create `docker/Dockerfile.agent-codex` and `docker/Dockerfile.agent-codex-go`
- [ ] Create `docker/Dockerfile.agent-opencode`
- [ ] Add `{prefix}-openai-api-key` secret to Terraform secrets.tf (with empty placeholder)
- [ ] Mount `OPENAI_API_KEY` in agent-executor job template
- [ ] Add build steps to CI for codex and opencode images
- [ ] Test codex executor end-to-end (`AILANG_EXECUTOR=codex` + `AILANG_PROVIDER=codex`)

**Phase 3: Gemini + eval variants** (~4 hours)

- [ ] Create `docker/Dockerfile.agent-gemini` (remove the `|| true` from current install)
- [ ] Create `docker/Dockerfile.agent-gemini-go`
- [ ] Create `docker/Dockerfile.agent-eval` and `docker/Dockerfile.agent-eval-go`
- [ ] Add Gemini auth handling to executor job (ADC via Workload Identity — verify this works)
- [ ] Add build steps to CI
- [ ] Set `executor_variant: gemini` on any gemini-provider agents in config

### Files to Modify/Create

**New files (in `ailang` repo):**
- `docker/Dockerfile.agent-base` — ~40 LOC
- `docker/Dockerfile.agent-go` — ~8 LOC
- `docker/Dockerfile.agent-gemini` — ~8 LOC
- `docker/Dockerfile.agent-codex` — ~8 LOC
- `docker/Dockerfile.agent-opencode` — ~8 LOC
- `docker/Dockerfile.agent-eval` — ~10 LOC
- (+ `-go` variants for gemini, codex, eval — ~8 LOC each)

**Modified files (in `ailang` repo):**
- `docker/Dockerfile.agent` — remove Go toolchain, add `FROM agent-base`; ~-10 LOC
- `internal/coordinator/agent_registry.go` — add `ExecutorVariant` field; ~+3 LOC
- `internal/coordinator/cloud_dispatcher.go` — add `ExecutorVariant` to `DispatchParams`; ~+3 LOC
- `internal/coordinator/daemon_tasks_exec.go` — wire variant into params; ~+3 LOC
- `internal/dispatch/cloudrun/dispatcher.go` — add map + image override; ~+25 LOC

**Modified files (in `ailang-multivac` repo):**
- `config/config.cloud.yaml` — add `executor_variant` to sprint-executor + any gemini agents
- `cloudbuild-images.yaml` — add build steps for all new variants
- `cloudbuild-trigger-ailang.yaml` — same
- `terraform/secrets.tf` — add `openai-api-key` secret resource
- `terraform/cloud_run_jobs.tf` — mount new secrets; add `AILANG_IMAGE_BASE` env var to coordinator service

---

## Examples

### Before: every agent runs the same fat image

```yaml
# All agents → agent:latest (~2GB, Go + claude + gemini)
- id: design-doc-creator
  provider: claude
  # no executor_variant — implicitly uses agent:latest

- id: sprint-executor
  provider: claude
  # no executor_variant — also uses agent:latest (has Go, correct by accident)
```

### After: each agent declares its needs

```yaml
- id: design-doc-creator
  provider: claude
  # executor_variant omitted → "default" → agent:latest (~600MB, claude only)

- id: sprint-executor
  provider: claude
  executor_variant: go       # agent-go:latest (~1.4GB, claude + Go)

- id: eval-suite-agent
  provider: gemini
  executor_variant: eval-go  # agent-eval-go:latest (all CLIs + Go)

- id: codex-agent            # future
  provider: codex
  executor_variant: codex    # agent-codex:latest (codex CLI only)
```

### Adding a new executor variant (no coordinator change)

```
1. Create docker/Dockerfile.agent-aider
2. Add "aider" entry to variantImages map in dispatcher.go
3. Add build step to cloudbuild-images.yaml
4. Set executor_variant: aider in config.cloud.yaml for the agent
```

Zero changes to coordinator logic.

---

## Success Criteria

- [ ] `agent:latest` does NOT include Go toolchain; `docker image inspect` confirms
- [ ] `sprint-executor` uses `agent-go:latest` in Cloud Run Job execution logs
- [ ] `design-doc-creator` uses `agent:latest` (slim) in logs
- [ ] `codex` executor resolves its CLI in `agent-codex:latest` (`codex --version` succeeds in job)
- [ ] Unknown `executor_variant` returns an error at dispatch time (not silently falls back)
- [ ] All currently-deployed agents continue working after rollout
- [ ] CI builds all images in correct dependency order (base before executors, executors before Go variants)
- [ ] `CHANGELOG.md` updated with image size before/after

---

## Testing Strategy

**Unit tests:**
- `imageForVariant()` — table-driven test covering all known variants, unknown variant returns error, empty string returns `agent:latest`
- `DispatchParams` serialization round-trip with `ExecutorVariant`

**Integration tests:**
- `internal/dispatch/cloudrun/dispatcher_test.go` — verify `ContainerOverride.Image` is set correctly for `go` variant
- Build all Dockerfiles in CI (build failure = test failure)

**Manual testing (per phase):**
- Trigger a `design-doc-creator` task; verify Cloud Run Job execution log shows `agent:latest`
- Trigger a `sprint-executor` task; verify `agent-go:latest`; run `go version` in the task to confirm Go is present
- Phase 2: trigger a codex task; confirm codex CLI responds

---

## Deferred Decisions

- **Separate job templates vs shared job + secrets** — which approach for auth isolation between executor families. Agent may choose based on secret count at implementation time. If ≤ 6 secrets total, shared job is fine; otherwise split templates.
- **Exact npm package names for opencode** — verify `opencode-ai` or alternate package name before writing Dockerfile.
- **`agent-base` push-to-registry timing** — whether to push `agent-base:latest` as a pullable image or keep it build-time only. Agent may decide; recommendation is to push it so developers can pull and inspect locally.
- **Go version pinning** — whether `agent-go` should pin Go version explicitly in the tag (`agent-go-1.25:latest`) or just track `golang:1.25-bookworm`. Agent may choose; recommendation is to track the minor version tag.

---

## Non-Goals

- **Dynamic variant selection at task runtime** — the variant is set in config, not inferred from task content. Over-engineered and a security surface.
- **Per-task image overrides via API** — same reason.
- **Variant-specific CPU/memory limits** — handle via separate Terraform job resources if ever needed.
- **Auto-detecting which CLIs a task needs** — this design requires explicit declaration in config.

---

## Migration Path

Hard cut — current cloud agents are not in active production use, so no parallel rollout is needed.

1. **Update `cloudbuild-images.yaml`**: add `agent-base` build step; rename existing `agent` build step to also tag as `agent-go:latest`; add a new slim `agent:latest` build step (FROM agent-base + claude CLI only)
2. **Build and push all new images** in one CI run — `agent:latest` is now slim, `agent-go:latest` carries Go
3. **Set `executor_variant: go` for `sprint-executor`** in `config.cloud.yaml` and upload
4. **Redeploy coordinator** — picks up new config; `sprint-executor` tasks now use `agent-go:latest`
5. **Build and push codex/opencode images** — no agents use them yet; available for future config wiring
6. Done — no phased cutover required

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Go variant image not pushed before `sprint-executor` task fires | High | Deploy config change after image is confirmed pushed in CI |
| `image` field in ContainerOverride rejected by Cloud Run API | High | Test with a single `go` variant before full rollout |
| Unknown variant silently falls back | Med | `imageForVariant()` returns error on unknown variant (no silent fallback principle) |
| Gemini CLI package name changes | Low | Install without `|| true`; CI build failure catches breakage early |
| opencode npm package name wrong | Med | Verify package name in Phase 2 before writing Dockerfile |
| New secrets not yet set in Secret Manager | Med | Use `scripts/setup-secrets.sh` after Terraform apply; document in migration guide |

---

## Timeline

**Day 1** (~8 hours):
- Phase 1: base image + slimmed agent + agent-go + dispatcher image override
- CI wired; deploy to dev

**Day 2** (~4 hours):
- Phase 2: codex + opencode variants + secrets wiring
- Test codex end-to-end in dev

**Day 3** (~4 hours):
- Phase 3: gemini + eval variants
- Migration step 5 (remove Go from `agent:latest`)
- Production rollout + `CHANGELOG.md`

**Total: ~2–3 days**

---

## Related Documents

- [EXECUTOR_SHAPE.md](../../docs/internal/EXECUTOR_SHAPE.md) — executor contract (codex/opencode already wired in coordinator)
- [design_docs/planned/v0_15_0/m-executor-variants.md](../v0_15_0/m-executor-variants.md) — original rough draft (superseded by this doc)
- [design_docs/planned/v0_13_0/m-copilot-cli-integration.md](../v0_13_0/m-copilot-cli-integration.md) — future Copilot CLI executor (same pattern)
- [ailang-multivac CLAUDE.md](../../..) — auth model: OAuth token, never API key for agent executor
- [design_docs/implemented/v0_8_1/m-process-exec.md](../../implemented/v0_8_1/m-process-exec.md) — process execution patterns

---

**Document created**: 2026-04-23
**Last updated**: 2026-04-23
