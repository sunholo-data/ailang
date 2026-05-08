# M-MOTOKO-EXECUTOR-ADAPTER — ailang-multivac PR Checklist (Pillar 2)

This file documents the cross-repo PRs that complete Pillar 2 of M-MOTOKO-EXECUTOR-ADAPTER. The in-repo work (Dockerfile.agent-motoko + `knownVariants["motoko"] = true`) shipped with the main sprint commit. The two ailang-multivac PRs below are the cloud-deployment side and require ailang-multivac repo access.

**Target dev project**: `ailang-multivac-dev` (matches pi/codex/opencode precedent).

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

# Watch the chain:
ailang chains list --agent coordinator --since 5m
```

## Acceptance gate (before sprint M5 can use the cloud Job)

- [ ] PR #1 merged; `gcloud artifacts docker images list` shows `agent-motoko:latest`
- [ ] PR #2 merged; `terraform apply` succeeds; `agent-executor-motoko` Job exists in dev
- [ ] One coordinator-dispatched task with `executor_variant: motoko` completes end-to-end
- [ ] Job logs show `motoko --version` resolves and the JSONL gets written

## Rollback plan

If the build pipeline produces a broken image:
1. `gcloud artifacts docker images delete <image>` to remove the bad tag
2. `terraform apply` reverts to the previous tag if `agent_image_tag` is pinned
3. `knownVariants["motoko"] = false` (revert in `internal/dispatch/cloudrun/dispatcher.go`) to make the coordinator reject motoko-targeted dispatches with a clear error rather than 5xx-ing on missing image

## Cross-references

- In-repo work: `docker/Dockerfile.agent-motoko`, `internal/dispatch/cloudrun/dispatcher.go::knownVariants`
- EXECUTOR_SHAPE.md §6 (cloudbuild drift warning), §7 (Cloud Run Job pattern), §8 (cost-control rule)
- Pi precedent (closest analogue): `docker/Dockerfile.agent-pi` + ailang-multivac's `agent_executor_pi` Terraform block
