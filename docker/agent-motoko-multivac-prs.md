# M-MOTOKO-EXECUTOR-ADAPTER — ailang-multivac PR Checklist (Pillar 2)

This file documents the cross-repo PRs that complete Pillar 2 of M-MOTOKO-EXECUTOR-ADAPTER. The in-repo work (Dockerfile.agent-motoko + `knownVariants["motoko"] = true`) shipped with the main sprint commit. The two ailang-multivac PRs below are the cloud-deployment side and require ailang-multivac repo access.

**Target dev project**: `ailang-multivac-dev` (matches pi/codex/opencode precedent).

**v0.23.0 update (M-COORD-TAG-ROUTING-LASTMILE, 2026-05-27)**: this PR set was originally specified during M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) but never executed. M-COORD-TAG-ROUTING-LASTMILE re-activates it because tag-routed `requires:["agent:motoko"]` messages (now supported by both the CLI `--requires` flag and the local HTTP `POST /api/messages` path) have no cloud-fallback Job to dispatch to. Three v0.23.0-specific updates land in this revision of the doc:

1. **Coordinator pre-flight check** (PR #0, new) — the cloud `ailang-coordinator` Cloud Run service must be on v0.22.0+ before this PR set can be exercised end-to-end. v0.22.0 shipped the `requires` field on `postMessageRequest` + the `PubSubInboxAdapter` tag-subset filter. As of 2026-05-27 the deployed image is from 2026-04-28 (pre-v0.21.0), so it must be redeployed via cloudbuild before M3 verification.
2. **OPENROUTER secret reality check** (PR #3, new) — `ailang-multivac` (prod) had NO `ailang-openrouter-api-key` secret as of 2026-05-27; only `ailang-multivac-dev` has `ailang-dev-openrouter-api-key` (created today). The motoko Job in this PR set targets the dev project for now; prod motoko cloud-dispatch is deferred until OpenRouter capacity + cost-control patterns are validated in dev (matches `motoko-or-gemma-4-26b` model's `budgets: max_cost_usd: 0.30` precedent).
3. **`worker_tags` on the agent entry** (PR #2 addition) — the new Cloud Run Job's coordinator config must declare `worker_tags: [agent:motoko]` so the cloud coordinator's tag matcher recognises it as a valid dispatch target for tag-routed sends. Without this, `requires:["agent:motoko"]` messages hit the Studio-or-nothing routing path.

In-repo M-COORD-TAG-ROUTING-LASTMILE work shipped: cloudbuild-dev.yaml's `build-agent-motoko` step lands the image in `ailang-multivac-dev`'s artifact registry as soon as the cloudbuild trigger fires on dev push.

## PR #0 — Redeploy `ailang-coordinator` to v0.22.0+ (operational, not code)

```bash
# From any host with ailang-multivac IAM:
gcloud builds submit \
  --project=ailang-multivac \
  --config=cloudbuild.yaml \
  --substitutions=_TARGET_PROJECT=ailang-multivac \
  /path/to/ailang   # the in-repo cloudbuild builds + pushes + redeploys coordinator

# Verify:
gcloud artifacts docker images list \
  europe-west1-docker.pkg.dev/ailang-multivac/ailang/coordinator \
  --include-tags --filter=tags:latest --format='value(createTime,tags)'
# Expected: createTime within last hour, tags including 'latest'

# Probe the redeployed coordinator's version (if the daemon's /version
# endpoint reports build info — adapt to whatever's actually exposed):
curl -s https://ailang-coordinator-ao6kuhcibq-ew.a.run.app/health
# 200 OK with no body confirms the listener is up post-redeploy.
```

This is a prerequisite for end-to-end testing of the v0.22.0 `requires` field. None of PR #1/#2/#3 below can be E2E-validated until this lands.

## PR #1 — Build pipeline (`cloudbuild.yaml` + `cloudbuild-images.yaml`)

**Per EXECUTOR_SHAPE.md §6**, both files MUST be updated together. Historical drift between these two configs has been the recurring failure mode for new executor variants.

### Changes to `cloudbuild.yaml`

Add a `build-agent-motoko` step mirroring `build-agent-pi`:

```yaml
- id: build-agent-motoko
  name: gcr.io/cloud-builders/docker
  args:
    - build
    - --build-arg=PROJECT=$_TARGET_PROJECT
    - --tag=${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-motoko:latest
    - --file=docker/Dockerfile.agent-motoko
    - .
  waitFor: [build-agent-base]   # depends on agent-base being available
```

Add `push-agent-motoko`:

```yaml
- id: push-agent-motoko
  name: gcr.io/cloud-builders/docker
  args: [push, "${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-motoko:latest"]
  waitFor: [build-agent-motoko]
```

Update `push-images.waitFor` to include `push-agent-motoko`. Update top-level `images:` list to include `${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-motoko:latest`.

### Changes to `cloudbuild-images.yaml`

Same `build-agent-motoko` + `push-agent-motoko` steps + same `images:` list update. This is the manual image-only pipeline (no terraform); it must stay in lockstep with `cloudbuild.yaml` so manual rebuilds produce the same image set.

### Verification

```bash
# After PR #1 merges to dev:
gcloud builds submit \
  --project=ailang-multivac-dev \
  --config=cloudbuild-images.yaml \
  --substitutions=_TARGET_PROJECT=ailang-multivac-dev \
  .

# Verify image landed:
gcloud artifacts docker images list \
  europe-west1-docker.pkg.dev/ailang-multivac-dev/ailang \
  --filter="package~agent-motoko" --include-tags
```

## PR #2 — Cloud Run Job (`terraform/cloud_run_jobs.tf`)

Add an `agent-motoko` Cloud Run Job mirroring the `agent-pi` job structure. Critical: secret bindings MUST exclude `ANTHROPIC_API_KEY` (cost-control rule; pi precedent).

```hcl
resource "google_cloud_run_v2_job" "agent_executor_motoko" {
  name     = "agent-executor-motoko"
  location = local.region

  template {
    template {
      service_account = local.executor_sa
      vpc_access      { connector = local.vpc_connector }

      # v0.23.0 (M-COORD-TAG-ROUTING-LASTMILE): the cloud coordinator
      # matches this Job against `requires:["agent:motoko"]` messages via
      # the tag matcher shipped in v0.22.0. Worker tag advertisement
      # happens via the coordinator's config (PR #2 addendum below) — NOT
      # a label on this Job resource. Documented here so future readers
      # don't try to add `labels = { worker_tag = ... }` to the Job spec.
      max_retries = 1   # motoko is non-idempotent in cost; one retry max

      containers {
        image = "${local.image_base}/agent-motoko:${var.agent_image_tag}"

        resources {
          limits = {
            cpu    = "2"
            memory = "4Gi"
          }
        }

        # COST-CONTROL: bind only OpenRouter + OpenAI + Gemini.
        # NEVER bind ANTHROPIC_API_KEY here. Per EXECUTOR_SHAPE.md §8,
        # motoko's Anthropic models route via OpenRouter; binding the
        # native key would re-expose pay-per-token billing. Pi precedent:
        # agent_executor_pi deliberately omits ANTHROPIC_API_KEY.
        env {
          name = "OPENROUTER_API_KEY"
          # v0.23.0: in DEV project `local.openrouter_secret` resolves to
          # `ailang-dev-openrouter-api-key` (added 2026-05-27). PROD doesn't
          # yet have an openrouter secret — defer prod motoko cloud-dispatch
          # until OpenRouter capacity + cost-control are validated in dev.
          value_source { secret_key_ref { secret = local.openrouter_secret; version = "latest" } }
        }
        env {
          name = "OPENAI_API_KEY"
          value_source { secret_key_ref { secret = local.openai_secret; version = "latest" } }
        }
        env {
          name = "GEMINI_API_KEY"
          value_source { secret_key_ref { secret = local.gemini_secret; version = "latest" } }
        }
      }
    }
  }
}
```

If you also need the `-apikey` variant pattern (per other executors' user-API-key job), mirror the `agent_executor_pi_apikey` block with the same secret bindings.

### Verification

```bash
# After PR #2 merges to dev + cloudbuild produces the image:
cd /path/to/ailang-multivac
terraform init
terraform plan -var="agent_image_tag=latest"
terraform apply -var="agent_image_tag=latest"

# Verify the Job exists:
gcloud run jobs describe agent-executor-motoko \
  --region=europe-west1 --project=ailang-multivac-dev

# Coordinator dispatch smoke (requires the AILANG repo's coordinator daemon
# pointed at the dev project):
ailang messages send coordinator '{
  "type": "task",
  "directive": "Print hello world to stdout",
  "executor_variant": "motoko"
}' --title "motoko Cloud Run smoke"

# v0.23.0: also smoke the new tag-routing path (M-COORD-TAG-ROUTING-LASTMILE):
ailang messages send eval-rig "Print PONG and exit" \
  --requires agent:motoko --from sprint-executor \
  --title "tag-routed motoko cloud smoke"
# (this hits the local daemon's /api/messages, which publishes to Pub/Sub
# with `requires:["agent:motoko"]`. The cloud coordinator subscription
# claims it, dispatches to ailang-agent-executor-motoko Job. Verify via
# `ailang chains view` on the resulting chain ID.)

# Watch the chain:
ailang chains list --agent coordinator --since 5m
```

### PR #2 addendum — coordinator agent config (v0.23.0)

Update the cloud `ailang-coordinator`'s mounted config (the file referenced by `AILANG_CONFIG=/etc/ailang-config/config.yaml` on the Cloud Run service) to add a `motoko` agent entry under `coordinator.agents:`. Mirror the existing `claude` / `codex` entries:

```yaml
coordinator:
  agents:
    # ... existing claude/codex/opencode/pi entries ...
    - id: motoko
      label: "Motoko (AILANG-native, OpenRouter)"
      inbox: motoko
      provider: motoko
      executor_job: ailang-agent-executor-motoko
      worker_tags:
        - agent:motoko
```

The `worker_tags` field activates tag-routed dispatch via the M-COORD-MULTI-HOST-WORKERS matcher. Without it, `requires:["agent:motoko"]` won't find this cloud Job and will only route to the Studio (or fail if the Studio is offline).

## PR #3 — Prod `ailang-openrouter-api-key` secret (DEFERRED to a follow-up sprint)

Currently `ailang-multivac` (prod) has no OpenRouter secret. The motoko Cloud Run Job in PR #2 targets `ailang-multivac-dev` (which has `ailang-dev-openrouter-api-key`, created 2026-05-27). To extend cloud motoko dispatch to PROD:

```hcl
# tools/terraform/secrets.tf (or wherever ailang-anthropic-api-key is defined)
resource "google_secret_manager_secret" "openrouter_api_key" {
  project   = local.prod_project
  secret_id = "ailang-openrouter-api-key"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "openrouter_executor_access" {
  project   = google_secret_manager_secret.openrouter_api_key.project
  secret_id = google_secret_manager_secret.openrouter_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${local.executor_sa}"
}
```

After `terraform apply` the secret resource is empty — upload the value once with:

```bash
gcloud secrets versions add ailang-openrouter-api-key \
  --project=ailang-multivac \
  --data-file=/path/to/openrouter-key.txt
```

**Deferral rationale**: motoko on cloud is OS-model focused (gemma-4-26b, glm-5, deepseek-v4-flash), which the per-Job `budgets: max_cost_usd: 0.30` cap (from `motoko-or-gemma-4-26b`'s models.yml entry) already bounds. PROD motoko dispatch isn't blocking anyone today; the eval-rig Studio handles the workload via bare-metal claim. Cost analysis on dev throughput → decision to enable prod is a follow-up.

## Acceptance gate (v0.23.0 update)

- [ ] **PR #0 (operational)**: Cloud `ailang-coordinator` image timestamp shows post-v0.22.0 deploy (current image is 2026-04-28, must be updated)
- [ ] **PR #1 merged**: `gcloud artifacts docker images list` shows `agent-motoko:latest` built within 1h of dev push
- [ ] **PR #2 merged**: `terraform apply` succeeds; `agent-executor-motoko` Job exists in dev; coordinator config has `motoko` agent with `worker_tags: [agent:motoko]`
- [ ] **End-to-end smoke** (M-COORD-TAG-ROUTING-LASTMILE M4 scenario 3): `--requires agent:motoko` sent with Studio offline (`launchctl bootout`) is claimed by the cloud motoko Job and completes
- [ ] **(Deferred)** PR #3 prod openrouter secret — gated on dev throughput / cost analysis

## Rollback plan

If the build pipeline produces a broken image:
1. `gcloud artifacts docker images delete <image>` to remove the bad tag
2. `terraform apply` reverts to the previous tag if `agent_image_tag` is pinned
3. `knownVariants["motoko"] = false` (revert in `internal/dispatch/cloudrun/dispatcher.go`) to make the coordinator reject motoko-targeted dispatches with a clear error rather than 5xx-ing on missing image

## Cross-references

- In-repo work: `docker/Dockerfile.agent-motoko`, `internal/dispatch/cloudrun/dispatcher.go::knownVariants`
- EXECUTOR_SHAPE.md §6 (cloudbuild drift warning), §7 (Cloud Run Job pattern), §8 (cost-control rule)
- Pi precedent (closest analogue): `docker/Dockerfile.agent-pi` + ailang-multivac's `agent_executor_pi` Terraform block
