# M-CI-BUILD-SPEED Baseline Measurements

**Captured:** 2026-04-28 (sprint M1)
**Purpose:** "Before" numbers M6 will compare against. All times measured from real Cloud Build runs during M-PKG-FEEDBACK-LOOP M2 deploy.

## Method

Pulled real start→finish timestamps from successful builds via `gcloud builds describe <id> --format='value(startTime,finishTime)'`. These exclude trigger-fire latency and queue time, which can add 5-30s.

## Per-trigger baseline

| Trigger | What it does | Recent build IDs | Wall-clock |
|---------|--------------|------------------|-----------|
| `ailang-multivac-dev` (full pipeline, cloudbuild.yaml) | Build all 11 agent variants + dashboard + mcp + docparse + billing + website-builder + terraform-apply + deploy-services | `0cac3518` (15:07→15:23 dev) | **~16 min** |
| `ailang-multivac-test` (full pipeline) | Same as dev, against test project | `183c94bd` (12:43→13:05 test) | **~22 min** |
| `ailang-multivac-prod` (full pipeline) | Same as dev, against prod project | `6ec3aee1` (14:25→14:41 prod) | **~16 min** |
| `ailang-core-dev` (cloudbuild-trigger-ailang.yaml inline) | Rebuild only coordinator + agent + agent-go + dashboard, redeploy to dev | `d8fd44f6` (11:57→12:04), `a4f03379` (11:11→11:17) | **~7 min** |

## Per push-type baseline

These are what M6 will re-measure after the sprint lands:

| Push type | Baseline | Target | Speedup |
|-----------|----------|--------|---------|
| YAML-only (config/templates) → multivac dev | ~16 min | ≤5 min | 3.2× |
| YAML-only → multivac test | ~22 min | ≤5 min | 4.4× |
| YAML-only → multivac prod | ~16 min | ≤5 min | 3.2× |
| ailang source change → ailang-core-dev | ~7 min | ≤10 min | already meets target (improve to ≤4 min via cache) |
| Multivac terraform-only → multivac dev | ~16 min (no skip path today) | ≤3 min (M5 terraform skip) | 5.3× |
| Full release dev → test → prod (3 hops) | ~54 min | ~15 min | 3.6× |

Note: The "25 min" worst-case I cited in conversation came from queue + rebuild + ambient drift on Cloud Run revision rolls. The real Cloud Build wall-clock floor for a full pipeline today is **~16 min** (dev/prod) and **~22 min** (test — likely warmer cache when others ran later in the day).

## Trigger structure (pre-sprint)

- 30 sequential build steps in `cloudbuild.yaml`
- 11 agent variants form a partially-serial chain: every variant `waitFor: ['push-agent-base']`, but within Claude lineage (`build-agent → push-agent → build-agent-go`) they're serial
- Pushes (`push-agent`, `push-agent-codex`, `push-agent-gemini`, `push-agent-eval`) are explicit steps blocking downstream builds
- No `--cache-from` anywhere — every layer rebuilt from scratch on every run
- No path filters on triggers — a docs-only change rebuilds 11 Docker images
- Terraform always runs both plan and apply — no skip on no-change

## Drift state (post-precursor)

After commit `0c72701` (Cloud Run drift fix), `terraform plan` against:

| Env | Add | Change | Destroy | Build |
|-----|-----|--------|---------|-------|
| dev | 0 | 1 (google_identity_platform_config — separate Firebase email-auth issue, deferred) | 0 | already shipped |
| test | 0 | 1 (same) | 0 | `41798e1b` (18 min) |
| prod | 0 | 1 (same) | 0 | `e5dbfe7e` (16 min) |

All three envs idempotent: a second consecutive `terraform plan` returns the same 0/1/0 (no ping-pong on `client`/`client_version` anymore).
