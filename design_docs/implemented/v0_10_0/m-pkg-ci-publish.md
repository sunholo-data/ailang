# M-PKG-CI-PUBLISH: Auto-Publish Packages via Cloud Build

**Status**: Planned
**Priority**: P1 (High — manual publishing is error-prone, stale index already caused DX issues)
**Estimated**: 1 day
**Dependencies**: Registry validator index fix (in-progress, same PR)
**Milestone ID**: M-PKG-CI-PUBLISH
**Created**: 2026-03-24
**Source**: sunholo/logging v0.2.0 published with Debug effect but registry index still showed IO and old description — revealed both an index update bug and the lack of CI for ailang-packages

---

## Problem Statement

Publishing AILANG packages is entirely manual: `cd packages/foo && ailang publish`. This causes:

1. **Stale registry**: Packages get committed to GitHub but never published (or published with wrong metadata)
2. **Index bug**: The validator's `updateIndex` doesn't update `AISummary`, `Tags`, `Effects`, `Stability`, `Exports`, or `HasAgentDoc` for existing packages — only for first-time publishes (fix included in this work)
3. **No CI/CD**: The `ailang-packages` repo has zero automation — every other repo (ailang, ailang-demos) has Cloud Build triggers

## Solution

### Part 1: Fix Registry Validator Index Bug

**File**: `cmd/registry-validator/main.go` (lines 358-373 in `tryUpdateIndex`)

Add 6 missing field updates to the existing-entry update path:

```go
// After existing URL updates, add:
index.Packages[i].AISummary   = getMetaString(manifest.Metadata, "ai_summary")
index.Packages[i].Tags        = getMetaStringSlice(manifest.Metadata, "tags")
index.Packages[i].Effects     = manifest.Effects.Max
index.Packages[i].Stability   = manifest.Stability.Level
index.Packages[i].Exports     = manifest.Exports.Modules
index.Packages[i].HasAgentDoc = meta.Manifest.HasAgentDoc
```

**Already done** — committed to dev branch, deploys automatically via `cloudbuild-trigger-ailang.yaml`.

### Part 2: Cloud Build Pipeline for ailang-packages

**New file**: `ailang-multivac/cloudbuild-trigger-packages.yaml`

Pipeline steps:
1. **Clone ailang-packages** (shallow, main branch)
2. **Download ailang binary** from latest GitHub release (`linux.x64.ailang.tar.gz`)
3. **Detect changed packages** via `git diff HEAD~1 --name-only | grep "packages/.*/ailang.toml"`
4. **Publish each changed package**: `cd packages/<name> && ailang publish`

```yaml
substitutions:
  _PACKAGES_REPO: https://github.com/sunholo-data/ailang-packages.git
  _AILANG_REPO: sunholo-data/ailang

options:
  logging: CLOUD_LOGGING_ONLY

steps:
  # Clone packages repo
  - id: clone-packages
    name: gcr.io/cloud-builders/git
    args: ['clone', '--depth', '2', '--branch', 'main', '${_PACKAGES_REPO}', '/workspace/packages']

  # Download ailang binary from latest release
  - id: download-ailang
    name: gcr.io/cloud-builders/curl
    entrypoint: bash
    args:
      - -c
      - |
        # Get latest release download URL
        RELEASE_URL=$(curl -sL \
          -H "Authorization: Bearer $$GITHUB_TOKEN" \
          "https://api.github.com/repos/${_AILANG_REPO}/releases/latest" \
          | grep "browser_download_url.*linux.x64.ailang.tar.gz" \
          | cut -d '"' -f 4)
        curl -sL "$$RELEASE_URL" | tar xz -C /workspace/
        chmod +x /workspace/ailang
    secretEnv: ['GITHUB_TOKEN']
    waitFor: ['-']

  # Detect changed packages and publish
  - id: publish-changed
    name: gcr.io/cloud-builders/git
    entrypoint: bash
    args:
      - -c
      - |
        cd /workspace/packages
        CHANGED=$(git diff HEAD~1 --name-only | grep "packages/.*/ailang.toml" | sed 's|/ailang.toml||' || true)

        if [ -z "$$CHANGED" ]; then
          echo "No package manifests changed — nothing to publish"
          exit 0
        fi

        FAILED=0
        for pkg_dir in $$CHANGED; do
          PKG_NAME=$(grep '^name' "$$pkg_dir/ailang.toml" | head -1 | sed 's/.*= *"\(.*\)"/\1/')
          PKG_VERSION=$(grep '^version' "$$pkg_dir/ailang.toml" | head -1 | sed 's/.*= *"\(.*\)"/\1/')
          echo "=== Publishing $$PKG_NAME@$$PKG_VERSION ==="

          cd /workspace/packages/$$pkg_dir
          if /workspace/ailang publish; then
            echo "✓ Published $$PKG_NAME@$$PKG_VERSION"
          else
            echo "✗ Failed to publish $$PKG_NAME@$$PKG_VERSION"
            FAILED=1
          fi
          cd /workspace/packages
        done

        exit $$FAILED
    secretEnv: ['AILANG_REGISTRY_API_KEY']
    env:
      - 'AILANG_REGISTRY_VALIDATOR=https://ailang-registry-validator-mdpoxgrptq-ew.a.run.app'
    waitFor: ['clone-packages', 'download-ailang']

availableSecrets:
  secretManager:
    - versionName: projects/ailang-registry/secrets/ailang-registry-api-key/versions/latest
      env: AILANG_REGISTRY_API_KEY
    - versionName: projects/ailang-multivac-deploy/secrets/github-token/versions/latest
      env: GITHUB_TOKEN
```

### Part 3: Cloud Build Trigger (Terraform or Manual)

Create a trigger in the `ailang-multivac-deploy` project pointing at `ailang-packages` repo, `main` branch, filtered to `packages/**/ailang.toml` changes.

**Option A — Manual (fastest)**:
```bash
gcloud builds triggers create github \
  --project=ailang-multivac-deploy \
  --repo-name=ailang-packages \
  --repo-owner=sunholo-data \
  --branch-pattern="^main$" \
  --build-config=cloudbuild-trigger-packages.yaml \
  --name=ailang-packages-publish \
  --description="Auto-publish changed packages to registry" \
  --region=europe-west3 \
  --service-account="projects/ailang-multivac-deploy/serviceAccounts/sa-cloudbuild@ailang-multivac-deploy.iam.gserviceaccount.com" \
  --included-files="packages/**/ailang.toml"
```

**Option B — Terraform** (add to `terraform-registry/`):
```hcl
resource "google_cloudbuild_trigger" "packages_publish" {
  project  = "ailang-multivac-deploy"
  location = "europe-west3"
  name     = "ailang-packages-publish"

  github {
    owner = "sunholo-data"
    name  = "ailang-packages"
    push {
      branch = "^main$"
    }
  }

  included_files = ["packages/**/ailang.toml"]
  filename       = "cloudbuild-trigger-packages.yaml"

  service_account = "projects/ailang-multivac-deploy/serviceAccounts/sa-cloudbuild@ailang-multivac-deploy.iam.gserviceaccount.com"
}
```

### Part 4: Rebuild Index (One-Time Fix)

After the validator deploys with the index fix, trigger a full index rebuild to fix all stale entries:

```bash
curl -X POST "https://ailang-registry-validator-mdpoxgrptq-ew.a.run.app/rebuild-index" \
  -H "X-API-Key: $AILANG_REGISTRY_API_KEY"
```

## Secrets Required

| Secret | Location | Purpose |
|--------|----------|---------|
| `ailang-registry-api-key` | `ailang-registry` project (already exists) | Publish auth |
| `github-token` | `ailang-multivac-deploy` project (already exists) | Download ailang release binary |

No new secrets needed — both already exist in Secret Manager.

## Files to Create/Modify

| File | Repo | Change |
|------|------|--------|
| `cmd/registry-validator/main.go` | ailang | Fix 6 missing index fields (**done**) |
| `cloudbuild-trigger-packages.yaml` | ailang-multivac | New Cloud Build pipeline |
| `terraform-registry/triggers.tf` | ailang-multivac (optional) | Terraform-managed trigger |

## Verification

1. Push the validator fix → auto-deploys via existing `ailang-dev` trigger
2. Run `curl -X POST .../rebuild-index` → verify `ailang search logging` shows `[Debug]`
3. Create trigger (manual or Terraform)
4. Bump `test_pkg` version in ailang-packages, push to main → verify Cloud Build publishes it
5. `ailang search test_pkg` → verify new version and updated metadata

## Risks

- **Cloud Build secret access**: The `sa-cloudbuild` SA needs `secretmanager.secretAccessor` on the `ailang-registry` project's secrets. It already has folder-level roles so this should work, but verify.
- **ailang binary compatibility**: The downloaded release binary must match the validator's version for consistent compilation. If they diverge, dry-run may pass locally but fail on validator.
- **409 Conflict**: Re-publishing an already-published version returns 409. The pipeline should treat this as success (idempotent), not failure.
