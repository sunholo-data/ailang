# M-CI-BUILD-SPEED: Cut multivac build time from ~25min to ~5min

**Status**: Planned
**Target**: v0.15.x
**Priority**: P2 (DX win, blocks no shipping work but every release/promotion pays the tax)
**Estimated**: 1 day, ~150 LOC of CI YAML + small Dockerfile tweaks
**Dependencies**: none (CI-only change)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Pure infra/DX work. No language semantics change.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A2: Replayability | 0 | Cache hits don't change output; same digests if inputs unchanged |
| A7: Machines First | **+2** | The agents own multivac via the coordinator pipeline; faster CI compounds across every M-* sprint |
| A11: Structured Failure | **+1** | Path filters give "skipped: no relevant files changed" as a first-class outcome instead of a 25min no-op |

**Net Score: +3** → **Decision: Move forward**

---

## Problem Statement

`ailang-multivac/cloudbuild.yaml` runs ~30 sequential steps on every push to dev/test/prod:

```
clone-ailang → clone-demos → build-coordinator
              ↓
build-agent-base → push-agent-base
              ↓
build-agent → push-agent → build-agent-go → build-agent-codex → push-agent-codex
              ↓
build-agent-codex-go → build-agent-opencode → build-agent-pi
              ↓
build-agent-gemini → push-agent-gemini → build-agent-gemini-go
              ↓
build-agent-eval → push-agent-eval → build-agent-eval-go
              ↓
build-dashboard → build-mcp → clone-docparse → build-docparse
              ↓
build-billing → build-website-builder → push-images
              ↓
terraform-plan → terraform-apply → deploy-services
```

Real measurements (2026-04-28 prod build of M-PKG-FEEDBACK-LOOP M2 — a YAML-only change):

| Phase | Wall time | What it actually did |
|-------|-----------|----------------------|
| Clone repos | ~30s | shallow clone ailang + demos + docparse |
| Build 11 agent variant images | ~14min | Each pulls Debian + Node.js + (Claude/Codex/Gemini/opencode) CLI + npm packages, all from scratch |
| Build dashboard + mcp + docparse + billing + website-builder | ~6min | dashboard does Go + React/Vite |
| Push images to Artifact Registry | ~2min | ~3GB total |
| Terraform plan + apply | ~2min | Many resource diffs (10 changes per dev plan today) |
| Deploy 5 Cloud Run services | ~1min | Sequential `gcloud run services update` |
| **Total** | **~25min** | for a one-line YAML change |

### Why it's so slow

1. **No Docker layer cache between builds.** Every push rebuilds `agent-base` from scratch — same Debian apt-get, same Node.js download, same `npm install -g @anthropic/claude-code` every time. None of the agent variant images share a `--cache-from` reference. Each variant pays the agent-base cost again.
2. **Pure dependency chains.** `build-agent` waitFor `push-agent-base`. `build-agent-go` waitFor `push-agent`. The variant images form a chain even though codex/gemini/opencode could fan out from `agent-base` directly in parallel.
3. **Pushes block the next build.** `push-agent` runs serially after `build-agent`, blocking `build-agent-go` from starting. Push is I/O-bound; build is CPU-bound. They could overlap.
4. **No path filtering on triggers.** A change to `config/config.cloud.yaml` rebuilds 11 Docker images that have no source dependency on it.
5. **Terraform always runs.** Even when no terraform-tracked resource changed, plan + apply still execute (~2min).

### Why it matters

- Every release waits 25min to land in dev, then again for test, then again for prod (~75min wall-clock per release end-to-end). M-PKG-FEEDBACK-LOOP M2 cost ~3 hours of wall-clock to ship from `git commit` → working in prod, mostly waiting on CI.
- Agent-driven sprints cycle through dev→test→prod multiple times per sprint when fixing infra bugs (we hit two during M2 — cloud-dispatch directive + firestore index). Each cycle is 25min lost.
- Promote-to-prod (test→prod image copy via crane) takes ~30s, proving the rebuild is unnecessary when the source SHA is the same.

---

## Goals

1. **A YAML-only change ships in <5min wall-clock**, not 25min
2. **An ailang-source change ships in <10min wall-clock**, not 25min
3. **Path filters skip build steps entirely when their inputs didn't change** (e.g. demos repo change → don't rebuild ailang agents)
4. **Cache hits visible in build logs** so future drift can be diagnosed
5. **No correctness regression** — the cached image must be byte-identical to a from-scratch build of the same inputs

## Non-goals

- Replacing Cloud Build with another CI system
- Switching from buildpacks (coordinator) to Dockerfile or vice versa
- Restructuring the agent variant taxonomy (still build all 11)
- Pre-built base images stored outside the project (would dodge cache invalidation but adds cross-project ownership)

---

## Proposed Approach

### Lever 1: BuildKit cache mounts + `--cache-from`

Switch the agent build steps from `docker build` to `docker buildx build` with a registry-backed cache:

```yaml
- id: build-agent-base
  name: gcr.io/cloud-builders/docker
  entrypoint: bash
  args:
    - -c
    - |
      docker buildx build \
        --cache-from "type=registry,ref=$AR/agent-base:cache" \
        --cache-to   "type=registry,ref=$AR/agent-base:cache,mode=max" \
        --tag "$AR/agent-base:latest" \
        --tag "$AR/agent-base:${SHORT_SHA}" \
        --push \
        -f /workspace/ailang/docker/Dockerfile.agent-base \
        /workspace/ailang
```

Each agent variant gets the same treatment, with `--cache-from` pointing to its own `:cache` tag AND its parent's `:cache`. This works because `docker buildx` understands layer hashing across separate image lineages when fed the right cache hints.

**Expected impact:** the 14-min agent build phase drops to **~3min** for a no-source-change run (cache hits everywhere) and **~6min** for an ailang-source change (only the layers after `COPY ailang/` rebuild, cache hits for everything below).

### Lever 2: Parallelize what doesn't actually depend

Today, `build-agent-go` waits for `push-agent`. But `agent-go` and `agent` both extend `agent-base` — they don't depend on each other. The dependency graph should be:

```
build-agent-base
       │
       ├──→ build-agent       ──→ push-agent       (Claude variant)
       ├──→ build-agent-go    ──→ push-agent-go    (Claude + Go)
       ├──→ build-agent-codex ──→ push-agent-codex (Codex variant)
       ├──→ build-agent-codex-go
       ├──→ build-agent-opencode
       ├──→ build-agent-pi
       ├──→ build-agent-gemini
       ├──→ build-agent-gemini-go
       ├──→ build-agent-eval
       └──→ build-agent-eval-go
                    ↓ (all in parallel)
              push-images (fan-in)
```

Replace the sequential `waitFor: ['<previous>']` with `waitFor: ['build-agent-base']` on every variant. Cloud Build runs steps in parallel by default when they don't share `waitFor` dependencies. Likewise: `build-dashboard`, `build-mcp`, `build-docparse`, `build-billing`, `build-website-builder` all fan out from `clone-ailang` (or their respective clones) and can run in parallel.

**Expected impact:** wall-clock for the build phase drops from `sum(all variants)` to `max(any single variant build time)`. With 11 variants and Cloud Build's machine type having 8 vCPUs, expect **~4× wall-clock reduction** on top of the cache wins. Cloud Build supports up to 100 concurrent steps per build — we're nowhere near the cap.

Push steps need rethinking too: today `push-agent` is its own step that blocks downstream. With `--push` baked into `docker buildx build`, the push happens as part of the build step and the next variant doesn't have to wait.

### Lever 3: Path-based trigger filters

Add `includedFiles` / `ignoredFiles` to triggers so a YAML-only change skips the image rebuild entirely:

| Change touches | What rebuilds |
|----------------|---------------|
| `config/config.cloud.yaml`, `config/templates/**` | terraform-apply only (uploads new YAML/templates to GCS, redeploys Cloud Run with same image) |
| `terraform/**.tf`, `terraform/environments/**` | terraform-apply only |
| (ailang-side) `docker/Dockerfile.*` | full rebuild of affected variant (and its descendants) |
| (ailang-side) `cmd/**`, `internal/**`, `go.mod`, `go.sum` | rebuild coordinator + agent variants (Go binary changed) |
| `cloudbuild*.yaml` | next build picks up the changes; no special handling needed |

Implementation: each Cloud Build trigger gets `includedFiles` matching the patterns above. For pushes that match nothing relevant, the trigger emits `no-op` skip (a 5-second build that does nothing).

**Expected impact:** YAML-only changes drop to **~2min total** (terraform plan + apply + deploy-services, no image work).

### Lever 4: Skip terraform when nothing changed

Add a `terraform plan -detailed-exitcode` gate before `terraform apply`:
- exit 0 → no changes, skip apply, skip deploy-services
- exit 2 → changes pending, run apply + deploy
- exit 1 → real error, fail loud

Saves ~2min on the large fraction of CI runs where terraform has nothing to do.

### Lever 5: Build the right thing from the right SHA

`cloudbuild.yaml` clones ailang from `dev` regardless of which env it's deploying to. That means a multivac-prod push picks up whatever happens to be on ailang `dev` at build time, not what was last validated in test. Two side effects:

1. The "deploy validated test images to prod" workflow would actually rebuild from a *newer* ailang dev SHA than what test was running. This is a correctness gap, not just a perf gap.
2. The build can't take a fast path when ailang-side hasn't changed since the last successful prod build of that SHA.

Fix: clone ailang from a SHA pinned per-environment (`_AILANG_SHA: ${SHORT_SHA}` for the multivac-dev trigger, recorded SHA from the last successful test apply for prod). Combined with a per-SHA image tag (`agent:sha-1234567`), the prod build can detect "this SHA already has images in test AR" and skip ahead to the crane-promote path automatically.

This is the largest design decision in the doc — defer to a follow-up if time-pressed; levers 1+2+3+4 already deliver the headline numbers.

---

## Acceptance Criteria

- [ ] A YAML-only change to `ailang-multivac/config/config.cloud.yaml` ships in **≤5min** wall-clock from `git push` to Cloud Run revision serving (measured via `gcloud builds describe ... --format='value(duration)'`)
- [ ] An ailang-source change (one-line edit in `internal/coordinator/`) ships in **≤10min** wall-clock end-to-end
- [ ] Path-filtered triggers skip image rebuild when only YAML changed (verify by checking build steps include only terraform + deploy)
- [ ] Cache hit ratio visible in build logs (`buildx` reports `CACHED [N/M]` per stage)
- [ ] No image digest divergence vs from-scratch build for identical input SHAs (spot-check via `crane manifest`)
- [ ] Existing dev/test/prod promotion semantics preserved (push to `dev`/`test`/`prod` branches still triggers the right pipelines)
- [ ] `make` targets in `ailang-multivac/Makefile` updated to use the same buildx cache when running locally (so `make docker-agent` picks up the registry cache)

---

## Risks

| Risk | Mitigation |
|------|------------|
| BuildKit cache image grows unboundedly | `mode=max` with weekly GC via `crane gc` cron; cache images excluded from the 5-version retention rule |
| Cache poisoning if a transient build pushes bad layers | `--cache-to` only on success steps; explicit `:cache-WIP` namespace for in-flight builds |
| Path filter misses a real dependency (e.g. `go.mod` change not in pattern) | Conservative defaults (any `cmd/**` or `internal/**` change triggers full rebuild); explicit allow-list, not deny-list |
| Cloud Build parallel step limits surprise us | Document the 100-step cap; current proposal stays under 30 |
| Terraform `-detailed-exitcode` flake | Wrap in retry; fall back to always-apply if `plan` fails |
| Lever 5 (per-env ailang pinning) breaks the clone-dev convention | Defer to a follow-up sprint; ship 1-4 first |

---

## Out of Scope

- Migrating coordinator off buildpacks (separate decision — buildpacks are already fast enough)
- Replacing the agent variant taxonomy with a runtime-flag-driven single image (would simplify CI but reshape the executor model — too big a change for this sprint)
- Adopting reusable workflows / a shared base build template across the multiple cloudbuild*.yaml files (good follow-up; out of scope here)

---

## Validation Plan

1. **Baseline measurement:** record current wall-clock for 3 representative pushes (YAML-only, ailang-source, dependency bump) — done in the M2 deploy retros (25min/25min/25min)
2. **Land levers 1+2 first** (cache + parallel) — validate against same 3 push types
3. **Land lever 3** (path filters) — validate YAML-only push hits the no-image-rebuild path
4. **Land lever 4** (terraform skip) — validate a config-only push skips terraform-apply
5. **Compare digests** for identical input SHAs across cached vs uncached runs — must match

## Open Questions

1. **Should we use Cloud Build's [private pools](https://cloud.google.com/build/docs/private-pools/private-pools-overview)?** The default pool's machine type is `E2_HIGHCPU_8` which is fine for parallel builds. Private pools would let us use bigger machines (faster individual builds) but cost more. Recommend NO unless lever 1+2 doesn't hit the targets.
2. **Lever 5 — pinning ailang SHA per environment** — is the correctness gap real enough to justify the sprint cost, or do we just trust that ailang `dev` is always in a deployable state? Reasonable to defer.
3. **Should `make docker-agent` use the same registry cache?** Yes for consistency, but adds a `--cache-from` that requires the developer to be authenticated to the AR. Worth it.

---

## Notes

- The `promote-to-prod` cloudbuild config (`cloudbuild-promote.yaml`) already does the "no rebuild, just copy + redeploy" path via `crane` — this design extends that thinking to dev→test (it already happens by accident when ailang `dev` hasn't changed) and to within-env retries.
- After this sprint, expect the "fix a one-line bug → ship to prod" cycle to compress from ~75min to ~15min, which makes the autonomous-package-agents loop feel snappy enough that the user doesn't context-switch away while waiting.
