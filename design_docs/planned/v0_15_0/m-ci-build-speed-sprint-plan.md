# Sprint Plan: M-CI-BUILD-SPEED (Cut multivac CI from ~25min to ~5min)

## Summary

Apply 4 stacked optimizations to `ailang-multivac` Cloud Build pipelines so that a one-line config change ships to prod in ≤5min instead of ≤25min. Levers: BuildKit registry cache, parallelize the agent variant graph, path filters per trigger, terraform skip-when-clean. Defers lever 5 (per-env ailang SHA pinning) — the design doc flags it as the largest decision and we don't need it to hit the headline numbers.

**Design doc:** [m-ci-build-speed.md](m-ci-build-speed.md)
**Duration:** 2 working days of edits, ~3 days wall-clock (validation rounds are CI-bound — each tweak needs a real Cloud Build run to measure)
**Total LOC:** ~190 (mostly YAML in `ailang-multivac` repo + `gcloud builds triggers` updates)
**Risk Level:** Medium — CI YAML changes are easy to roll back but bad changes block deploys until reverted; staged rollout dev → test → prod with measurement at each step
**Repo:** primarily `ailang-multivac` (cloudbuild.yaml, cloudbuild-trigger-ailang.yaml, cloudbuild-images.yaml, gcloud trigger configs); no ailang code changes expected

---

## Current Status Analysis

### Baseline measurements (2026-04-28, M2 deploy)

| Push type | Current wall-clock | Target |
|-----------|--------------------|--------|
| YAML-only (config/templates) | ~25min | ≤5min |
| ailang source change (one-line in `internal/coordinator/`) | ~25min | ≤10min |
| Terraform-only change | ~25min | ≤3min |
| Full clean rebuild | ~25min | ≤15min (best-case after cache + parallel) |

### Pipeline steps (cloudbuild.yaml, 30 steps total)

The chain that bottlenecks everything: `clone-ailang → build-agent-base → push-agent-base → build-agent → push-agent → build-agent-go → build-agent-codex → push-agent-codex → build-agent-codex-go → ...`. Almost every variant gates on `push-agent-base` so they fan out, but within Claude lineage they're serial. Pushes are explicit steps that block downstream builds.

### Precursor: Cloud Run drift fix already shipped

Commit `0c72701` in `ailang-multivac` dev branch added `lifecycle { ignore_changes = [client, client_version] }` to all 14 `google_cloud_run_v2_job` and 6 `google_cloud_run_v2_service` resources, eliminating the ping-pong drift where `terraform plan` always showed 9 changes that flipped back and forth between gcloud and tf. Now `terraform plan dev` shows 1 change (Firebase identity_platform — separate issue, deferred). M1 verifies this lands cleanly in test+prod.

---

## Proposed Milestones

### M1 — Promote drift fix + establish baseline measurement (~0 LOC, ~1h)

**Goal:** Ship the existing `0c72701` Cloud Run drift fix to test+prod, then take a baseline timing measurement we'll compare against at sprint end.

**Tasks:**
- Merge `multivac` dev → test → prod (drift fix)
- Verify each env's `terraform plan` returns ≤1 change on consecutive runs (no ping-pong)
- Capture timing of 3 baseline pushes recorded in `docs/sprint-retros/m-ci-build-speed-baseline.md`:
  - Push 1: A YAML-only change (already measured today: ~25min)
  - Push 2: A no-op ailang push (will measure)
  - Push 3: A no-op multivac push (will measure)
- These 3 numbers are the "before" the sprint compares against

**Acceptance:**
- [ ] dev/test/prod all show "Plan: 0 to add, ≤1 to change, 0 to destroy" on `terraform plan` (the 1 = identity_platform, separate issue)
- [ ] No drift on consecutive `terraform plan` runs (idempotent)
- [ ] Baseline timings captured in `docs/sprint-retros/m-ci-build-speed-baseline.md`

---

### M2 — BuildKit registry cache for agent variant images (~80 LOC, ~4h)

**Goal:** Stop rebuilding `agent-base` (Debian + Node + Claude/Codex/Gemini CLIs) from scratch on every push. Cache hits should make the agent build phase drop from ~14min to ~3min on no-source-change runs.

**Tasks:**
- Add `setup-buildx` step at the start of `cloudbuild.yaml` (creates a buildx builder with registry cache support)
- Convert `build-agent-base` from `docker build` to `docker buildx build --cache-from "type=registry,ref=$AR/agent-base:cache" --cache-to "type=registry,ref=$AR/agent-base:cache,mode=max" --push`
- Same conversion for all 10 agent variants (`agent`, `agent-go`, `agent-codex`, `agent-codex-go`, `agent-opencode`, `agent-pi`, `agent-gemini`, `agent-gemini-go`, `agent-eval`, `agent-eval-go`)
- Drop the now-redundant `push-agent`, `push-agent-codex`, `push-agent-gemini`, `push-agent-eval` steps (pushes happen as part of buildx)
- Same conversion for `cloudbuild-trigger-ailang.yaml` inline (the ailang-core-dev trigger that rebuilds coord/agent/dashboard on ailang push)
- Same conversion for `cloudbuild-images.yaml` (the manual image-build trigger)
- Document cache lifecycle: weekly `crane gc` cron via a small `cloudbuild-cache-gc.yaml` (or just leave for follow-up if scope creeps)

**Acceptance:**
- [ ] `make docker-build` locally still works (uses the same registry cache)
- [ ] Back-to-back identical builds: 2nd run completes ≥3× faster than 1st (validate via two consecutive `gcloud builds triggers run`)
- [ ] Build logs show `CACHED [N/M]` lines per stage (visible cache hits)
- [ ] Image digests for cached vs from-scratch builds match for identical input SHAs (spot-check via `crane manifest`)
- [ ] Push-only `push-*` steps removed from cloudbuild.yaml; build steps include `--push`

**Risk:** BuildKit cache image grows unboundedly without GC. Mitigation: `mode=max` + a future GC job; for now document the manual `crane gc` command in `ailang-multivac/Makefile`.

---

### M3 — Parallelize agent variant builds (~50 LOC, ~2h)

**Goal:** Today `build-agent-go` waits for `push-agent` even though both extend `agent-base`. Fan everything out from `build-agent-base` and let Cloud Build run them in parallel.

**Tasks:**
- Edit cloudbuild.yaml `waitFor` for each agent variant to point at `build-agent-base` (since pushes are now embedded in the build step from M2, there's no `push-agent-base` to gate on either)
- Same for cloudbuild-trigger-ailang.yaml inline
- Same for cloudbuild-images.yaml
- Verify `push-images` final step still gates correctly on the union of all variants
- Verify `terraform-plan` still waits for `push-images`
- Document the dependency graph in a comment block at the top of cloudbuild.yaml

**Acceptance:**
- [ ] Cloud Build timeline view shows ≥8 agent variants running concurrently after M3 (vs 1-2 today)
- [ ] Wall-clock for agent build phase drops to `max(any single variant)` (~3-4min) instead of `sum(all variants)` (~14min)
- [ ] No stale-image hazard: `push-images` still depends on every variant having pushed before terraform-plan starts
- [ ] dev/test/prod all still work end-to-end after the change

**Risk:** Cloud Build's default machine type has 8 vCPU. With 11 parallel image builds plus dashboard/mcp/docparse/billing/website-builder, we'd hit the CPU ceiling and serialize implicitly. Watch the timeline; if it doesn't speed up, bump to E2_HIGHCPU_32 in cloudbuild.yaml `options.machineType` for the duration of the run.

---

### M4 — Path-based trigger filters (~40 LOC, ~3h)

**Goal:** A push that only changes `config/config.cloud.yaml` shouldn't rebuild 11 Docker images. Add `includedFiles` / `ignoredFiles` to triggers and a config-only fast path.

**Tasks:**
- Identify the path patterns each trigger should care about (matrix in design doc Lever 3)
- Update 3 triggers via `gcloud builds triggers update`:
  - `ailang-multivac-dev`, `ailang-multivac-test`, `ailang-multivac-prod`: `ignoredFiles: ["**/*.md", "docs/**", "scripts/**"]`
  - `ailang-core-dev`: `includedFiles: ["cmd/**", "internal/**", "go.mod", "go.sum", "docker/**"]`
- Add a new trigger `ailang-multivac-config-{dev,test,prod}` filename `cloudbuild-config-only.yaml` that runs only `terraform-plan → terraform-apply → deploy-services` when `includedFiles: ["config/**", "terraform/**"]`
- The original triggers get `ignoredFiles: ["config/**", "terraform/**"]` so they don't double-fire
- Document in `ailang-multivac/CLAUDE.md` section 11 (Cloud Build) which trigger fires for which path

**Acceptance:**
- [ ] A push of only `config/config.cloud.yaml` to dev fires the new config-only trigger (visible in `gcloud builds list`)
- [ ] That config-only build completes in <3min wall-clock
- [ ] A push of only `cmd/ailang/messages.go` (ailang side) fires `ailang-core-dev`, NOT a multivac trigger
- [ ] A push of only `docs/**` doesn't fire any rebuild trigger
- [ ] Trigger filters documented in CLAUDE.md

**Risk:** Path filter regression — if a real source dependency is excluded, builds won't rebuild when they should. Mitigation: conservative allow-lists (any `cmd/**` or `internal/**` change triggers full rebuild, not specific files); explicit deny-list only for known-irrelevant paths (`docs/**`, `*.md`, `scripts/**`).

---

### M5 — Terraform skip when no changes (~20 LOC, ~1h)

**Goal:** Many CI runs have nothing to change in terraform — but `terraform apply` still runs (~2min). Use `terraform plan -detailed-exitcode` to gate apply.

**Tasks:**
- Modify `terraform-plan` step in cloudbuild.yaml: add `-detailed-exitcode -out=plan.out` to plan command, capture exit code in a workspace file
- Modify `terraform-apply` step: read exit code, only apply if `=2`, exit 0 if `=0` (no changes), fail if `=1` (real error)
- Same gate for `deploy-services`: skip if terraform skipped
- Document the exit-code semantics inline in cloudbuild.yaml
- Same change in cloudbuild-config-only.yaml from M4

**Acceptance:**
- [ ] A push that changes nothing terraform-tracked (e.g. only template content) skips both terraform-apply AND deploy-services (visible "skipped — no changes" in build log)
- [ ] A push with real terraform changes still runs apply + deploy normally
- [ ] A push with broken terraform (syntax error in tf) fails loud at plan, doesn't silently skip

---

### M6 — End-to-end measurement + retro (~0 LOC, ~1h)

**Goal:** Prove we hit the headline numbers and capture the savings.

**Tasks:**
- Re-run the same 3 push types from M1 baseline:
  - YAML-only push to dev
  - ailang source push to dev
  - Multivac terraform-only push to dev
- Record new timings in `docs/sprint-retros/m-ci-build-speed.md`
- Compare baseline → after; calculate % wall-clock reduction
- Update CHANGELOG.md with the savings (helps future sprint estimates)
- Move design doc + sprint plan from `design_docs/planned/v0_15_0/` to `design_docs/implemented/v0_15_x/`

**Acceptance:**
- [ ] YAML-only push completes in ≤5min (target from design doc)
- [ ] ailang source push completes in ≤10min
- [ ] Retro doc captures before/after for all 3 push types
- [ ] CHANGELOG entry under v0.15.x section
- [ ] Design doc moved to `implemented/`

---

## Success Metrics

- **YAML-only changes ship in ≤5min** (currently ~25min) — 5× speedup
- **ailang source changes ship in ≤10min** (currently ~25min) — 2.5× speedup
- **Terraform skip works** — `Plan: 0 to add, 0 to change, 0 to destroy` runs skip apply + deploy
- **No correctness regression** — image digests match between cached and from-scratch builds for identical SHAs
- **Cloud Run drift stays at ≤1 change/plan** (M1 precursor verified) and doesn't regress

## Dependencies

- The Cloud Build SA (`sa-cloudbuild@ailang-multivac-deploy.iam.gserviceaccount.com`) needs `roles/artifactregistry.writer` on the `:cache` tags — already has admin so no change needed
- buildx is bundled in `gcr.io/cloud-builders/docker` — no new builder image needed
- `gcloud builds triggers update` requires `roles/cloudbuild.builds.editor` (the user has owner on the deploy project)
- Drift fix from `0c72701` should be in test+prod before M2 starts (so we don't conflate drift noise with cache-hit measurement) — that's M1

## Open Questions

1. **Should we also touch `cloudbuild-promote.yaml`?** It already does the optimal "no rebuild, crane copy + redeploy" path. Recommend NO — it's already fast enough.
2. **GC strategy for cache images?** Recommend documenting `crane gc` command in Makefile + revisit if cache size becomes a real cost (low priority — Artifact Registry is cheap).
3. **Per-env ailang SHA pinning (Lever 5 in design doc)?** Defer per the design doc. Worth a follow-up sprint after this lands; relevant for the test→prod promotion correctness story.

## Notes

- All milestones except M1 and M6 touch only `ailang-multivac` (no ailang Go code). M1 is the precursor verification (drift fix already committed).
- Each milestone needs at least one real Cloud Build run to verify; budget ~25min/run for the first verification round, ~5min/run after M2-M3 land.
- M2 + M3 are the headline wins. M4 is the YAML-only ergonomic win. M5 is small cleanup. They can all land sequentially in one branch or as separate PRs — recommend separate PRs since each is independently verifiable.
- After this sprint, expect the "fix one-line bug → ship to prod via dev/test promotion" cycle to drop from ~75min to ~15min.
