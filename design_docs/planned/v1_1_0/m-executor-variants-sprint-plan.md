# M-EXECUTOR-VARIANTS Sprint Plan

**Design doc**: [m-executor-variants.md](m-executor-variants.md)
**Sprint ID**: M-EXECUTOR-VARIANTS
**Estimated**: 2–3 days (~400 LOC across Go + Docker + CI/Terraform config)
**Risk**: Low — coordinator Go changes are small; Docker/CI changes are the bulk of work

---

## Sprint Summary

Replace the monolithic `agent:latest` Docker image (~2GB, contains Go + claude + gemini) with a family of purpose-built variants. Each agent declares `executor_variant` in config; the dispatcher resolves it to the correct image at Cloud Run job dispatch time.

**Hard cut migration** — current cloud agents are not in active production use, so we build all new images in a single CI run and cut over.

**Repos touched**: `ailang` (Dockerfiles + Go coordinator code) and `ailang-multivac` (CI yaml + Terraform + config).

---

## Current State

- `Dockerfile.agent` already exists with Go baked in — becomes `Dockerfile.agent-go`
- Gemini CLI installed with `|| true` (silently fails — fix in this sprint)
- `OPENAI_API_KEY` secret resource exists in Terraform but not mounted in agent job
- Coordinator code wires codex + opencode executors via blank imports — no Go changes needed for registration
- `variantImages` map, `imageForVariant()`, and `ContainerOverride.Image` don't exist yet in dispatcher
- `ExecutorVariant` field missing from `AgentConfig` and `DispatchParams`
- `imageBase` can be constructed inside the dispatcher from its existing `d.region` + `d.projectID` fields — no new env var required

---

## Milestone Breakdown

### M1: Core Go wiring + base Dockerfiles (~4 hours)

The smallest useful slice: dispatcher understands variants, default image is now slim, sprint-executor gets `agent-go`. This milestone alone delivers the primary value (Go toolchain out of the default image).

**Go code changes (~60 LOC):**

- `internal/coordinator/agent_registry.go` — add `ExecutorVariant string` to `AgentConfig` (~3 LOC)
- `internal/coordinator/cloud_dispatcher.go` — add `ExecutorVariant string` to `DispatchParams` (~3 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — wire `agent.ExecutorVariant → params.ExecutorVariant` alongside existing `agent.AuthMode`, `agent.Model`, etc. (~3 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — add `variantImages` map, `imageForVariant()`, pass `Image` in `ContainerOverride`. Construct `imageBase` from `d.region + d.projectID` already in struct (~35 LOC)
- `internal/dispatch/cloudrun/dispatcher_test.go` — table-driven tests for `imageForVariant`, verify `Image` set in mock RunJob call (~40 LOC)

**Dockerfiles:**

- `docker/Dockerfile.agent-base` — extract Node 22 + git + ailang binary + bootstrap clone from current `Dockerfile.agent`, no CLIs, no Go (~40 LOC)
- `docker/Dockerfile.agent` — updated: `FROM agent-base:latest` + `npm install -g @anthropic-ai/claude-code` (~8 LOC; was 60 LOC — shrinks significantly)
- `docker/Dockerfile.agent-go` — `FROM agent:latest` + Go toolchain COPY (~8 LOC)

**Acceptance criteria:**
- `make test` passes (including new dispatcher tests)
- `docker build -f docker/Dockerfile.agent-base .` succeeds locally
- `docker build -f docker/Dockerfile.agent .` succeeds and does NOT include `go` binary
- `docker build -f docker/Dockerfile.agent-go .` succeeds and includes `go version`
- `imageForVariant("unknown-xyz", ...)` returns an error (not a silent default)

---

### M2: CI update + ailang-multivac wiring (~3 hours)

Wire the new images into CI and point `sprint-executor` at `agent-go`.

**`ailang-multivac/cloudbuild-images.yaml`:**

Build order: `clone-ailang` → `build-agent-base` → parallel(`build-agent`, `build-agent-gemini`, `build-agent-codex`, `build-agent-opencode`) → push base family → `build-agent-go`, `build-agent-gemini-go`, `build-agent-codex-go`, `build-agent-eval` → push all

Three new steps for Phase 1 (base, slim agent, agent-go). The existing `build-agent` step gets a second tag (`agent-go:latest`) during the transition, and the new slim `agent:latest` replaces it.

**`ailang-multivac/config/config.cloud.yaml`:**

```yaml
- id: sprint-executor
  executor_variant: go    # add this line
```

All other agents omit `executor_variant` (defaults to `default` → `agent:latest`).

**`ailang-multivac/cloudbuild-trigger-ailang.yaml`:**

Same additions as `cloudbuild-images.yaml` (parallel trigger pipeline).

**Acceptance criteria:**
- Cloud Build run succeeds with all three new images pushed
- `agent:latest` image size is ≤ 600MB (verify with `docker image inspect`)
- `agent-go:latest` image size is ≤ 1.5GB
- Coordinator config updated and uploaded to GCS
- Triggering a `design-doc-creator` task uses `agent:latest` (visible in Cloud Run job logs)
- Triggering a `sprint-executor` task uses `agent-go:latest`

---

### M3: Codex + opencode variants (~3 hours)

Add Codex (OpenAI) and opencode executor images. The secret resource already exists in Terraform; just need to mount it in the agent job.

**Dockerfiles:**

- `docker/Dockerfile.agent-codex` — `FROM agent-base:latest` + `npm install -g @openai/codex` (~8 LOC)
- `docker/Dockerfile.agent-codex-go` — `FROM agent-codex:latest` + Go toolchain (~8 LOC)
- `docker/Dockerfile.agent-opencode` — `FROM agent-base:latest` + opencode install (~8 LOC)

**`ailang-multivac/terraform/cloud_run_jobs.tf`:**

Mount `OPENAI_API_KEY` from the existing `openai_api_key` secret into the agent-executor job template. The secret resource exists; add the env var binding (~15 LOC).

**CI additions** (`cloudbuild-images.yaml`): build steps for agent-codex, agent-codex-go, agent-opencode.

**Acceptance criteria:**
- `agent-codex:latest` built and pushed to Artifact Registry
- `agent-opencode:latest` built and pushed
- `OPENAI_API_KEY` mounted in agent job (verify via `terraform plan` shows the change)
- Running `codex --version` inside a `agent-codex` container succeeds
- No existing agents broken (they use `agent:latest` which hasn't changed)

**Note**: Verify the opencode npm package name (`opencode-ai` or similar) before writing the Dockerfile. If uncertain, check `npm info opencode` first.

---

### M4: Gemini + eval variants (~3 hours)

Add Gemini executor image (fixing the silent `|| true` install) and the multi-executor eval image.

**Dockerfiles:**

- `docker/Dockerfile.agent-gemini` — `FROM agent-base:latest` + `npm install -g @google/gemini-cli` (no `|| true`) (~8 LOC)
- `docker/Dockerfile.agent-gemini-go` — `FROM agent-gemini:latest` + Go (~8 LOC)
- `docker/Dockerfile.agent-eval` — `FROM agent-base:latest` + all four CLIs (`claude-code`, `gemini-cli`, `codex`, `opencode`) (~12 LOC)
- `docker/Dockerfile.agent-eval-go` — `FROM agent-eval:latest` + Go (~8 LOC)

**CI additions**: build steps for gemini, gemini-go, eval, eval-go.

**`ailang-multivac/config/config.cloud.yaml`**: set `executor_variant: gemini` for any gemini-provider agents (if any exist).

**Acceptance criteria:**
- `agent-gemini:latest` built — install does NOT use `|| true`
- `agent-eval:latest` built — contains all four CLIs
- `gemini --version` and `codex --version` both work inside `agent-eval` container
- `CHANGELOG.md` updated with before/after image sizes
- `make test` still passes

---

## Files Summary

### `ailang` repo

| File | Change | Est. LOC |
|------|--------|----------|
| `docker/Dockerfile.agent-base` | New | ~40 |
| `docker/Dockerfile.agent` | Rewrite (slim) | ~8 |
| `docker/Dockerfile.agent-go` | New | ~8 |
| `docker/Dockerfile.agent-codex` | New | ~8 |
| `docker/Dockerfile.agent-codex-go` | New | ~8 |
| `docker/Dockerfile.agent-opencode` | New | ~8 |
| `docker/Dockerfile.agent-gemini` | New | ~8 |
| `docker/Dockerfile.agent-gemini-go` | New | ~8 |
| `docker/Dockerfile.agent-eval` | New | ~12 |
| `docker/Dockerfile.agent-eval-go` | New | ~8 |
| `internal/coordinator/agent_registry.go` | +1 field | ~3 |
| `internal/coordinator/cloud_dispatcher.go` | +1 field | ~3 |
| `internal/coordinator/daemon_tasks_exec.go` | Wire variant | ~3 |
| `internal/dispatch/cloudrun/dispatcher.go` | Add map + imageForVariant + Image override | ~35 |
| `internal/dispatch/cloudrun/dispatcher_test.go` | New tests | ~40 |
| `CHANGELOG.md` | Update | ~15 |

**Total ailang**: ~225 LOC

### `ailang-multivac` repo

| File | Change | Est. LOC |
|------|--------|----------|
| `cloudbuild-images.yaml` | Add 8 new build steps | ~80 |
| `cloudbuild-trigger-ailang.yaml` | Same additions | ~40 |
| `terraform/cloud_run_jobs.tf` | Mount OPENAI_API_KEY in agent job | ~15 |
| `config/config.cloud.yaml` | Add executor_variant to sprint-executor | ~2 |

**Total ailang-multivac**: ~137 LOC

**Grand total**: ~360 LOC

---

## Implementation Order

```
Day 1:
  M1 — Go code + base Dockerfiles (ailang repo)
  M2 (partial) — cloudbuild-images.yaml + config.cloud.yaml (ailang-multivac)

Day 2:
  M2 (complete) — trigger CI run, verify image sizes + agent dispatch
  M3 — Codex + opencode variants + secrets mount

Day 3:
  M4 — Gemini + eval variants
  Final verification, CHANGELOG update, PRs
```

---

## Key Implementation Notes

1. **`imageBase` construction in dispatcher** — use `fmt.Sprintf("%s-docker.pkg.dev/%s/ailang", d.region, d.projectID)`. No new env var needed; `d.region` and `d.projectID` are already fields on `Dispatcher`.

2. **Build order in CI** — Go-variant images (`agent-go`, `gemini-go`, etc.) use `FROM agent:latest` (not from local build context), so `agent:latest` must be pushed before those steps run. Use a push-then-build pattern: push base family, then build Go variants.

3. **`OPENAI_API_KEY` already in Terraform** — `google_secret_manager_secret.openai_api_key` exists in `secrets.tf`. Only need to add the env binding in `cloud_run_jobs.tf`, not create a new resource.

4. **No `agent_image_tag` Terraform var change** — the Terraform var `agent_image_tag` is used for the main job template. The dispatcher overrides the image per-execution via `ContainerOverride.Image`, so Terraform doesn't need to know about variants. The job template can keep `agent:latest` as its base image.

5. **opencode package name** — verify with `npm info opencode` or check the opencode GitHub repo before writing `Dockerfile.agent-opencode`. If the correct package name is different from `opencode-ai`, update accordingly.

---

## Success Criteria (full sprint)

- [ ] `agent:latest` does NOT contain Go toolchain (`go version` fails inside it)
- [ ] `sprint-executor` Cloud Run job logs show `agent-go:latest` as the container image
- [ ] `design-doc-creator` Cloud Run job logs show `agent:latest`
- [ ] `codex --version` works inside `agent-codex:latest`
- [ ] `gemini --version` works inside `agent-gemini:latest` (no `|| true`)
- [ ] All four CLIs work inside `agent-eval:latest`
- [ ] Unknown `executor_variant` value causes `Dispatch()` to return an error (verified by test)
- [ ] All existing agents continue working (no regressions in design-doc-creator, sprint-planner, etc.)
- [ ] `make test` passes
- [ ] `CHANGELOG.md` updated with image sizes before/after
