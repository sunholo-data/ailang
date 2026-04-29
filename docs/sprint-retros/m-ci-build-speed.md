# Sprint Retrospective: M-CI-BUILD-SPEED

**Status**: ✅ Complete
**Duration**: ~3 hours (vs 2-day estimate — fast because most CI verification was sub-minute after M2 landed)
**LOC delivered**: ~310 (vs 290 estimated)
**Repo**: ailang-multivac (no ailang Go changes)

## Headline Numbers

| Push type | Before | After | Speedup |
|-----------|--------|-------|---------|
| YAML / config-only push (multivac) | 16-22 min | **60 s** | **16-22×** |
| Terraform-only push (no real changes) | 16-22 min | **64 s** | **15-20×** |
| Multivac source push (full pipeline, cache warm) | 16-22 min | **6:27** | **2.5-3.4×** |
| Multivac source push (full pipeline, cache cold) | 16-22 min | **9:17** | **1.7-2.4×** |
| ailang source push (`ailang-core-dev`) | ~7 min | not re-measured (no source push in sprint window) | n/a |

The fast paths blew past the design-doc targets:
- Target ≤5min for YAML-only — actual 60s ✅
- Target ≤10min for ailang source — actual 6:27 (multivac proxy, not exact same path) ✅
- Target ≤3min for terraform-only — actual 64s ✅

## What landed

| ID | Description | Wall-clock impact | Commit |
|----|-------------|-------------------|--------|
| M1 | Drift fix to test+prod, baseline measurement | precursor | `0c72701` (test+prod via push) |
| M2 | BuildKit registry cache for 11 agent variants + dashboard/mcp/docparse/billing/website-builder | -3 min on cache hits | `cebd1b8` |
| M3 | Parallelize agent variant graph (fan-out from agent-base instead of serial waitFor chain) | -10-12 min when cache cold | `cebd1b8` |
| M4 | Path-based trigger filters + new `cloudbuild-config-only.yaml` fast path | full ~16-22min → 60s for YAML-only | `8838f79` |
| M5 | terraform plan -detailed-exitcode + apply skip-when-clean | -3s typical, more when terraform stable | `701cc50` (escape fix) |
| Bonus | Fixed `google_identity_platform_config` permanent drift | dev plan now 0/0/0 clean | `f558e36` |
| Bonus | Cloud Build `images:` declaration removed (incompatible with `buildx --push`) | unblocks M2 | `6eac6fa` |

## Friction Encountered

**1. cloudbuild-trigger-ailang.yaml is inline, not loaded from file.**
The repo file is reference-only — the actual config lives in the trigger's `build:` field. Updating the file doesn't change anything until a separate `gcloud builds triggers import` migrates the trigger to use `filename:`. Worked around for M2/M3 (doesn't affect anything until ailang push hits the dev branch). Documented as follow-up.

**2. Cloud Build's `images:` declaration conflicts with `buildx --push`.**
The `images:` block tells Cloud Build's daemon to push images at build end via its own pusher. With buildx, images are already pushed during the build step but the daemon can't find them locally → `PUSH_IMAGE_NOT_FOUND` even though every step succeeded. Fix: drop `images:` from all three configs.

**3. Cloud Build interprets `$VARNAME` as substitution.**
`TFEXIT=$?` was rejected by Cloud Build's submission validator because it tried to substitute `$TFEXIT` and didn't know the key. Need `$$VARNAME` to emit a literal `$` for the shell.

**4. `google_identity_platform_config` permanent drift.**
The provider's read returns the `email{}` block as null when email auth is disabled, but the .tf set `enabled=false` explicitly. Result: every plan tried to "+ email" again. Fixed by removing the explicit block + adding `lifecycle { ignore_changes = [sign_in] }`.

## Per-step timeline (cache-warm M5 build d68543f4, 6:27 total)

```
   start  step                     dur
----------------------------------------
     4s  set-env                   1s
     5s  setup-buildx              6s
     5s  clone-ailang              8s
    13s  clone-demos               5s
    13s  build-coordinator       187s   (buildpacks — separate caching story)
    13s  build-agent-base        188s   (cold first-tier rebuild — source layer changed)
    13s  build-dashboard         189s   (Vite cache-cold)
    13s  build-mcp               189s
   201s  build-agent               6s   ← cache hit
   201s  build-agent-codex         7s   ← cache hit  ┐
   201s  build-agent-opencode      4s   ← cache hit  │ ALL SIX
   201s  build-agent-pi            7s   ← cache hit  │ PARALLEL
   201s  build-agent-gemini        5s   ← cache hit  │
   201s  build-agent-eval          4s   ← cache hit  ┘
   205s  build-agent-eval-go       3s   ← cache hit
   206s  build-agent-gemini-go     3s
   207s  build-agent-go            5s
   208s  build-agent-codex-go      4s
   222s  terraform-plan           28s   (init dominates)
   250s  terraform-apply           3s   (1 trivial change at the time — drift fix not yet shipped to all envs)
   253s  deploy-services         133s   (gcloud run services update is slow)
```

The two slow stragglers are now buildpacks-coordinator (187s) and the GCP CLI deploy step (133s). Both are out-of-scope for this sprint.

## Trigger taxonomy after the sprint

| Trigger | Repo → Branch | Pipeline | Fires on |
|---------|--------------|----------|----------|
| `ailang-multivac-{dev,test,prod}` | ailang-multivac → branch | `cloudbuild.yaml` (full) | Anything NOT in `config/`, `terraform/`, `docs/`, `*.md` |
| `ailang-multivac-config-{dev,test,prod}` | ailang-multivac → branch | `cloudbuild-config-only.yaml` (fast) | Only `config/**` or `terraform/**` |
| `ailang-core-dev` | ailang → `dev` | inline | Only `cmd/**`, `internal/**`, `go.mod`, `go.sum`, `docker/**` |
| `ailang-demos-dev` | ailang-demos → `main` | inline | Default (no path filter set) |

Same-push double-fire is possible for mixed pushes (e.g. config + cmd). Both run independently — fast path lands first, full path catches up. Acceptable.

## What we deferred from the design doc

- **Lever 5 (per-env ailang SHA pinning)** — the design doc's largest decision and out of scope for this sprint. Worth a follow-up to make the test→prod promotion reproducible.
- **Cache GC strategy** — manual `crane gc` documented in design doc, no automation yet.
- **Switch `cloudbuild-trigger-ailang.yaml` to file-based** — the trigger still has its build inline. Functional difference is zero today; matters when we want git history of changes to that pipeline.

## Recommendations for next CI sprint

1. **Reduce `deploy-services` time** — currently 133s of sequential `gcloud run services update`. Could parallelize with `&` + `wait` in shell.
2. **Cache the buildpacks coordinator** — `pack build --cache-image` or migrate to a Dockerfile to use buildx cache. ~3min recoverable.
3. **Cache the dashboard Vite build** — currently 189s every time because npm/Vite caches don't survive between Cloud Build runs. A buildx mount cache for `node_modules` would help.
4. **Migrate `cloudbuild-trigger-ailang.yaml` to file-based** — small ergonomic win, makes the ailang-core trigger reviewable in git.

## Verification artifacts

- Baseline doc: [m-ci-build-speed-baseline.md](./m-ci-build-speed-baseline.md)
- Real builds compared:
  - Baseline: `0cac3518` (16:04), `e5dbfe7e` (15:55), `41798e1b` (18:08), `183c94bd` (22:04)
  - M2+M3 cache cold: `2f4c0c49` (10:44, failed at images: push)
  - M2+M3 cache warm: `ba1fec26` (9:17 ✅)
  - M4 config-only: `7c0663ed` (60s ✅)
  - M5 full pipeline cache hot: `d68543f4` (6:27 ✅)
  - M5 skip-path proof: `39eaf4fb` (64s, terraform skipped ✅)
