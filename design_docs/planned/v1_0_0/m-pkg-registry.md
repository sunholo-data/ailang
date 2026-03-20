# M-PKG-REGISTRY: AILANG Package Registry on GCP

**Status**: Planned
**Target**: v1.1.0 (after v1.0.0 ships with path + git deps)
**Priority**: P1 — enables ecosystem growth beyond first-party packages
**Estimated**: 2 weeks (1 week infra/Terraform, 1 week CLI)
**Dependencies**:
- Package system Phase 1+1.5+1.5b (complete)
- ailang-packages repo (complete — seed content for registry)
- ailang-multivac Terraform infrastructure (existing)

---

## Problem Statement

AILANG packages currently require either:
- **Path deps**: clone the repo alongside your project
- **Git deps**: specify full git URL + subdir + tag in every `ailang.toml`

This works for first-party packages but doesn't scale to a community ecosystem. AI agents need:
- **Discovery**: `ailang search "auth"` → structured results with `ai_summary`
- **Simple install**: `ailang add sunholo/auth@0.1.0` (no git URL needed)
- **Quality signals**: verified contracts, effect compliance, stability level
- **Immutability**: published versions cannot be overwritten

---

## Design Goals

| Goal | Description |
|------|-------------|
| G1 — Simple install | `ailang add vendor/name@version` resolves from registry |
| G2 — AI-first discovery | Structured search results, `ai_summary`, AGENT.md serving |
| G3 — Verified packages | Automated compile + effect check + contract verification on publish |
| G4 — Immutable releases | Once published, a version cannot be changed or yanked (v1) |
| G5 — GCP-native | GCS bucket, Cloud Run validator, IAM auth — matches existing infra |
| G6 — Terraform-deployable | Multivac team deploys everything via Terraform |

## Non-Goals (v1.1)

- **No web UI** — AI agents use CLI, not browsers
- **No semver ranges** — exact versions only, no `^` or `~`
- **No dependency mirroring** — packages store their own source, not copies of deps
- **No private registries** — single public registry (private registries are Phase 3)
- **No package yanking** — immutable once published
- **No user accounts** — GCP IAM handles authentication

---

## Architecture

```
                     ailang publish
                          │
                          ▼
                  ┌───────────────┐
                  │  Cloud Run    │  Validation service
                  │  validator    │  (compile, check effects, verify contracts)
                  └───────┬───────┘
                          │ if valid
                          ▼
                  ┌───────────────┐
                  │  GCS Bucket   │  gs://ailang-registry/
                  │  (public)     │  packages/vendor/name/version/
                  └───────┬───────┘
                          │
                  ┌───────────────┐
                  │  index.json   │  Package catalog (updated on each publish)
                  └───────────────┘
                          │
               ailang search / install
                          │
                          ▼
                  ┌───────────────┐
                  │  Local cache  │  ~/.ailang/cache/registry/
                  └───────────────┘
```

### Components

1. **GCS Bucket** (`gs://ailang-registry/`) — public read, authenticated write
2. **Validation Service** (Cloud Run) — compiles, checks effects, verifies contracts
3. **Index File** (`index.json`) — machine-readable catalog of all packages
4. **CLI Commands** — `publish`, `install`, `search` in `cmd/ailang/`

---

## GCS Bucket Structure

```
gs://ailang-registry/
  index.json                                    # Package catalog
  task-views/                                   # AI Task View JSON files
    document-processing.json
    web-api.json
    gcp-integration.json
  packages/
    sunholo/
      auth/
        0.1.0/
          package.tar.gz                        # Source tarball
          metadata.json                         # Validation results + hashes
        0.2.0/
          package.tar.gz
          metadata.json
      gcp-auth/
        0.1.0/
          package.tar.gz
          metadata.json
```

### Naming Rules

- Bucket: `ailang-registry` (in the ailang GCP project)
- Package path: `packages/<vendor>/<name>/<version>/`
- Vendor names: lowercase alphanumeric + hyphens
- Package names: lowercase alphanumeric + hyphens
- Versions: semantic versioning (exact, e.g., `0.1.0`)

---

## Index File Schema (`index.json`)

Updated atomically on each publish. AI agents download this for search and resolution.

```json
{
  "schema": "ailang.registry/v1",
  "updated_at": "2026-03-20T12:00:00Z",
  "packages": [
    {
      "name": "sunholo/auth",
      "latest": "0.2.0",
      "versions": ["0.1.0", "0.2.0"],
      "ai_summary": "API key validation, HMAC signing, bearer token extraction",
      "tags": ["auth", "security", "api-key"],
      "effects": [],
      "stability": "experimental",
      "exports": ["sunholo/auth/keys", "sunholo/auth/bearer"],
      "contracts_verified": 3,
      "has_agent_doc": true
    }
  ]
}
```

**Design constraints:**
- Single file — no pagination needed for hundreds of packages
- ~1KB per package → 100 packages ≈ 100KB (cacheable)
- Client caches with `If-Modified-Since` header
- Immutable entries: version data never changes after publish

---

## Per-Version Metadata (`metadata.json`)

Computed by the validation service on publish. Not user-editable.

```json
{
  "schema": "ailang.package-metadata/v1",
  "name": "sunholo/auth",
  "version": "0.1.0",
  "published_at": "2026-03-20T12:00:00Z",
  "published_by": "sunholo-voight-kampff",
  "content_hash": "sha256:a1b2c3d4...",
  "interface_hash": "sha256:e5f6g7h8...",
  "tarball_hash": "sha256:i9j0k1l2...",
  "tarball_size_bytes": 4096,
  "validation": {
    "compiles": true,
    "effects_valid": true,
    "contracts_verified": 3,
    "contracts_total": 5,
    "contracts_skipped": 2,
    "ailang_version": "v1.0.0"
  },
  "manifest": {
    "edition": "1",
    "effects_max": [],
    "exports": ["sunholo/auth/keys", "sunholo/auth/bearer"],
    "stability": "experimental",
    "ai_summary": "API key validation, HMAC signing, bearer token extraction",
    "has_agent_doc": true
  }
}
```

---

## Validation Service (Cloud Run)

### Validation Pipeline

1. **Receive tarball** — HTTP POST with `package.tar.gz`
2. **Extract and parse** — verify `ailang.toml` is valid, all required fields present
3. **Check immutability** — reject if `vendor/name@version` already exists in bucket
4. **Check namespace** — verify caller owns the `vendor/` namespace (Firestore lookup)
5. **Compile check** — `ailang check` on all exported modules (must pass)
6. **Effect ceiling check** — verify no module exceeds declared `[effects].max`
7. **Contract verification** — `ailang verify` on all modules (best-effort, records results)
8. **Compute hashes** — content hash + interface hash + tarball hash
9. **Generate metadata.json** — record all validation results
10. **Upload to GCS** — `package.tar.gz` + `metadata.json` to versioned path
11. **Update index.json** — add/update package entry atomically

### Service Specification

```yaml
service: ailang-registry-validator
image: gcr.io/${project_id}/ailang-registry-validator
port: 8080
memory: 1Gi
cpu: 2
timeout: 300s
concurrency: 1
min_instances: 0
max_instances: 1

env:
  REGISTRY_BUCKET: ailang-registry
  GOOGLE_CLOUD_PROJECT: ${project_id}
```

### API Endpoints

```
POST /publish
  Auth: Bearer token (GCP IAM or API key)
  Body: multipart/form-data with package.tar.gz
  Response: 200 + metadata.json | 400 validation errors | 409 version exists

GET /health
  Response: 200 OK
```

### Container Image

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o ailang ./cmd/ailang/
RUN go build -o registry-validator ./cmd/registry-validator/

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y z3 ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/ailang /usr/local/bin/
COPY --from=builder /app/registry-validator /usr/local/bin/
EXPOSE 8080
CMD ["registry-validator"]
```

---

## Authentication Model

### Namespace Ownership

- `sunholo/*` — published by `sunholo-voight-kampff` service account
- Namespace mapping in Firestore: `ailang_registry_namespaces` collection
- Document: `{ vendor: "sunholo", owners: ["sa@project.iam.gserviceaccount.com"] }`
- No self-registration — namespaces manually approved (CRAN-style curation)

### For AI agents on Cloud Run

Cloud Run services get IAM identity automatically. An agent with the right service account can publish packages as part of automated workflows.

---

## CLI Commands

### `ailang publish`

```bash
cd my-package/
ailang publish
# ✓ Published sunholo/auth@0.1.0 (3 contracts verified)
```

### `ailang install vendor/name@version`

```bash
ailang install sunholo/auth@0.1.0
# Downloads, verifies hash, extracts to ~/.ailang/cache/registry/
# Adds to ailang.toml and runs ailang lock
```

### `ailang search "query"`

```bash
ailang search "auth"
# sunholo/auth@0.2.0 — API key validation, HMAC signing [Pure]
# sunholo/gcp-auth@0.1.0 — GCP ADC OAuth2 token exchange [FS, Net]

ailang search --tag gcp
ailang search --task-view web-api
```

### `ailang docs vendor/name`

```bash
ailang docs sunholo/auth
# Displays AGENT.md from the package
```

---

## Registry URL Configuration

```bash
# Default (built into ailang binary)
AILANG_REGISTRY=https://storage.googleapis.com/ailang-registry

# Override for testing
AILANG_REGISTRY=https://my-test-bucket.storage.googleapis.com
```

---

## What the Multivac Team Needs to Deploy (Terraform)

### Resources Required

| Resource | Type | Purpose |
|----------|------|---------|
| `google_storage_bucket.ailang_registry` | GCS | Package storage (public read) |
| `google_storage_bucket_iam_member.public_read` | IAM | `allUsers` → `objectViewer` |
| `google_cloud_run_v2_service.registry_validator` | Cloud Run | Validation service |
| `google_service_account.registry_validator` | SA | Validator identity |
| `google_storage_bucket_iam_member.validator_write` | IAM | Validator → bucket `objectAdmin` |
| `google_project_iam_custom_role.package_publisher` | IAM | Publisher role |
| Firestore collection: `ailang_registry_namespaces` | Firestore | Namespace ownership |

### Terraform Snippet

```hcl
# 1. GCS Bucket
resource "google_storage_bucket" "ailang_registry" {
  name          = "${var.prefix}-registry"
  project       = var.project_id
  location      = var.region
  storage_class = "STANDARD"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "registry_public_read" {
  bucket = google_storage_bucket.ailang_registry.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

# 2. Validator Service Account
resource "google_service_account" "registry_validator" {
  account_id   = "${var.prefix}-registry-validator"
  display_name = "AILANG Registry Validator"
  project      = var.project_id
}

resource "google_storage_bucket_iam_member" "validator_write" {
  bucket = google_storage_bucket.ailang_registry.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.registry_validator.email}"
}

# 3. Cloud Run Validator
resource "google_cloud_run_v2_service" "registry_validator" {
  name     = "${var.prefix}-registry-validator"
  location = var.region
  project  = var.project_id

  template {
    service_account = google_service_account.registry_validator.email

    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/ailang/registry-validator:latest"

      resources {
        limits = { memory = "1Gi", cpu = "2" }
      }

      env {
        name  = "REGISTRY_BUCKET"
        value = google_storage_bucket.ailang_registry.name
      }
    }

    timeout = "300s"
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
  }
}

# 4. Variables to add
variable "registry_bucket_name" {
  description = "GCS bucket for AILANG package registry"
  type        = string
  default     = "ailang-registry"
}
```

### Deployment Order

1. `terraform apply` — creates bucket + IAM + Cloud Run service
2. Seed `index.json`: upload empty `{"schema":"ailang.registry/v1","packages":[]}` to bucket
3. Build validator Docker image, push to Artifact Registry
4. Deploy Cloud Run with image
5. Test: `curl https://validator-url/health`
6. Seed registry: publish the 6 ailang-packages

---

## Migration Path

Existing deps continue to work. Registry deps are additive:

```toml
[dependencies]
"local/utils" = { path = "../utils" }                    # path dep (existing)
"sunholo/auth" = { git = "https://...", tag = "v0.1.0" } # git dep (existing)
"sunholo/logging" = "0.1.0"                               # registry dep (NEW)
```

Resolution priority: path > git > registry.

---

## Implementation Split

### Multivac team (Terraform + infra)

- [ ] GCS bucket with public read + IAM write
- [ ] Cloud Run validator service
- [ ] Service account + IAM bindings
- [ ] Firestore namespace collection
- [ ] Docker image build pipeline
- [ ] Seed empty index.json

### AILANG team (CLI + client)

- [ ] `cmd/ailang/pkg_publish.go`
- [ ] `cmd/ailang/pkg_install.go`
- [ ] `cmd/ailang/pkg_search.go`
- [ ] `cmd/ailang/pkg_docs.go`
- [ ] `internal/pkg/registry.go` — registry HTTP client
- [ ] `internal/pkg/tarball.go` — package tar creation/extraction
- [ ] Update resolver for `"registry"` source
- [ ] Cache: `~/.ailang/cache/registry/`

### Validator service (new Go service)

- [ ] `cmd/registry-validator/main.go`
- [ ] Tarball receive + extract
- [ ] `ailang check` + `ailang verify` integration
- [ ] `metadata.json` generation
- [ ] GCS upload + index.json update
- [ ] Namespace auth check

---

## Success Criteria

- [ ] `ailang publish` uploads to GCS with validation
- [ ] `ailang install sunholo/auth@0.1.0` works
- [ ] `ailang search "auth"` returns structured results
- [ ] `ailang docs sunholo/auth` displays AGENT.md
- [ ] Validation rejects packages that fail compilation
- [ ] Published versions are immutable
- [ ] All 6 ailang-packages seeded into registry
- [ ] Path and git deps continue to work alongside registry deps

---

## Related Documents

- [m-pkg-package-system.md](m-pkg-package-system.md) — Package system design (Phase 1+1.5+1.5b)
- [m-pkg-package-system-sprint-plan.md](m-pkg-package-system-sprint-plan.md) — Phase 1 sprint plan
- [m-pkg-phase1.5-sprint-plan.md](m-pkg-phase1.5-sprint-plan.md) — Phase 1.5 sprint plan

---

**Document created**: 2026-03-20
**Last updated**: 2026-03-20
