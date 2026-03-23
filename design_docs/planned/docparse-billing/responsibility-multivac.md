# M-BILLING: ailang-multivac Responsibility

**Parent**: [m-billing-docparse-billing-agent-payment.md](m-billing-docparse-billing-agent-payment.md)
**Repo**: `sunholo-data/ailang-multivac` (private)
**Working directory**: `/Users/mark/dev/sunholo/ailang-multivac`

---

## Scope

Add Terraform infrastructure for the new billing Cloud Run service. This follows the existing patterns in `docparse.tf`, `iam.tf`, `secrets.tf`, and the Cloud Build pipeline.

---

## Existing Infrastructure Context

The billing service joins an existing set of Cloud Run services:

| Service | Terraform | Service Account | Firestore DB |
|---------|-----------|----------------|--------------|
| Coordinator | `cloud_run.tf` | `sa-coordinator` | `(default)` |
| Dashboard | `cloud_run.tf` | `sa-dashboard` | `(default)` |
| Agent Executor | `cloud_run_jobs.tf` | `sa-agent` | `(default)` |
| Website Builder | `cloud_run_website.tf` | `sa-website-builder` | `website-builder` |
| **DocParse API** | `docparse.tf` | `sa-docparse` | `docparse` |
| **Billing (NEW)** | `billing.tf` | `sa-billing` | `docparse` (shared) |

Key: The billing service shares the `docparse` Firestore database but has its own service account with minimal permissions.

---

## New File: `terraform/billing.tf`

### Cloud Run Service

```hcl
# Billing Service — Cloud Run + Stripe integration
#
# Deploys the billing service with:
#   1. Cloud Run service (authenticated, not public)
#   2. Service account with Firestore + Secret Manager access
#   3. Stripe secrets injected as env vars
#
# Uses the existing `docparse` Firestore database for billing collections.
# Source: ailang-packages billing-service-api package

resource "google_cloud_run_v2_service" "billing" {
  count    = var.bootstrap ? 0 : 1
  name     = "${var.prefix}-billing-api"
  location = var.region

  template {
    service_account = google_service_account.billing.email

    scaling {
      min_instance_count = var.billing_min_instances
      max_instance_count = 5
    }

    containers {
      image = "${local.image_base}/${var.prefix}-billing:${var.billing_image_tag}"

      resources {
        limits = {
          cpu    = var.billing_cpu
          memory = var.billing_memory
        }
      }

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }
      env {
        name  = "FIRESTORE_DATABASE"
        value = var.docparse_firestore_database  # shared with DocParse
      }
      env {
        name = "STRIPE_SECRET_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.stripe_secret_key.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "STRIPE_WEBHOOK_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.stripe_webhook_secret.secret_id
            version = "latest"
          }
        }
      }

      startup_probe {
        tcp_socket {
          port = 8080
        }
        initial_delay_seconds = 5
        period_seconds        = 10
        failure_threshold     = 5
      }

      ports {
        container_port = 8080
      }
    }

    timeout = "30s"  # billing operations are fast (unlike parse)
  }

  depends_on = [
    google_project_service.apis,
    google_artifact_registry_repository.ailang,
  ]
}
```

### IAM: Public webhook endpoint, authenticated everything else

```hcl
# Public access for Stripe webhooks — app-layer auth via signature verification
# User-facing endpoints authenticated via Firebase Auth bearer token
resource "google_cloud_run_v2_service_iam_member" "billing_public" {
  count    = var.bootstrap ? 0 : 1
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.billing[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
```

**Note**: The webhook endpoint (`/billing/webhooks/stripe`) needs public access for Stripe to call it. All other endpoints are Firebase Auth-protected at the app layer. Same pattern as DocParse.

### Service Account

```hcl
resource "google_service_account" "billing" {
  account_id   = "${var.prefix}-billing"
  display_name = "Billing Service"
  description  = "Service account for the billing Cloud Run service"
}

# Firestore: read/write billing collections in docparse database
resource "google_project_iam_member" "billing_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.billing.email}"
}

# Secret Manager: read Stripe keys
resource "google_project_iam_member" "billing_secrets" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.billing.email}"
}

# Cloud Trace: request tracing
resource "google_project_iam_member" "billing_trace" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.billing.email}"
}

# Logging
resource "google_project_iam_member" "billing_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.billing.email}"
}
```

---

## New Secrets: `terraform/secrets.tf` additions

```hcl
# Stripe secret key for billing service
resource "google_secret_manager_secret" "stripe_secret_key" {
  secret_id = "${var.prefix}-stripe-secret-key"

  replication {
    auto {}
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.apis]
}

# Stripe webhook signing secret for billing service
resource "google_secret_manager_secret" "stripe_webhook_secret" {
  secret_id = "${var.prefix}-stripe-webhook-secret"

  replication {
    auto {}
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.apis]
}
```

**Post-apply**: Set secret values via:
```bash
echo -n "sk_live_..." | gcloud secrets versions add ${PREFIX}-stripe-secret-key \
  --project=${PROJECT} --data-file=-
echo -n "whsec_..." | gcloud secrets versions add ${PREFIX}-stripe-webhook-secret \
  --project=${PROJECT} --data-file=-
```

---

## New Firestore Indexes: `terraform/firestore.tf` additions

Add to the `docparse` database (alongside existing `usage_logs` and `api_keys` indexes):

```hcl
# Billing: lookup subscriptions by status + period end (for expiration checks)
resource "google_firestore_index" "billing_subscriptions_by_status" {
  project    = var.project_id
  database   = google_firestore_database.docparse.name
  collection = "subscriptions"

  fields {
    field_path = "status"
    order      = "ASCENDING"
  }
  fields {
    field_path = "currentPeriodEnd"
    order      = "ASCENDING"
  }
}

# Billing: lookup proposals by principal + status (for dedup and polling)
resource "google_firestore_index" "billing_proposals_by_principal" {
  project    = var.project_id
  database   = google_firestore_database.docparse.name
  collection = "payment_proposals"

  fields {
    field_path = "principalId"
    order      = "ASCENDING"
  }
  fields {
    field_path = "status"
    order      = "ASCENDING"
  }
  fields {
    field_path = "createdAt"
    order      = "DESCENDING"
  }
}

# Billing: lookup events by processed status (for reprocessing)
resource "google_firestore_index" "billing_events_by_processed" {
  project    = var.project_id
  database   = google_firestore_database.docparse.name
  collection = "billing_events"

  fields {
    field_path = "processed"
    order      = "ASCENDING"
  }
  fields {
    field_path = "processedAt"
    order      = "DESCENDING"
  }
}
```

---

## New Variables: `terraform/variables.tf` additions

```hcl
variable "billing_cpu" {
  description = "CPU allocation for billing service"
  type        = string
  default     = "1"
}

variable "billing_memory" {
  description = "Memory allocation for billing service"
  type        = string
  default     = "512Mi"
}

variable "billing_min_instances" {
  description = "Minimum instances for billing service"
  type        = number
  default     = 0
}

variable "billing_image_tag" {
  description = "Docker image tag for billing service"
  type        = string
  default     = "latest"
}
```

---

## Environment Config: `terraform/environments/*/terraform.tfvars` additions

### dev
```hcl
billing_cpu           = "1"
billing_memory        = "512Mi"
billing_min_instances = 0  # Scale to zero in dev
```

### prod
```hcl
billing_cpu           = "1"
billing_memory        = "512Mi"
billing_min_instances = 1  # Always-on for webhook reliability
```

---

## Cloud Build Pipeline Changes

### cloudbuild.yaml additions

Add a clone + build step for the billing service (after the existing docparse steps):

```yaml
# Clone ailang-packages (billing service source)
- name: 'gcr.io/cloud-builders/git'
  id: 'clone-ailang-packages'
  args: ['clone', '--depth', '1', '--branch', 'main',
         'https://github.com/sunholo-data/ailang-packages.git',
         '/workspace/ailang-packages']

# Build billing service image
- name: 'gcr.io/cloud-builders/docker'
  id: 'build-billing'
  args: ['build', '-t', '${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/${_PREFIX}-billing:$_IMAGE_TAG',
         '-f', '/workspace/ailang-packages/packages/billing-service-api/Dockerfile',
         '/workspace/ailang-packages']
  waitFor: ['clone-ailang-packages']

# Deploy billing service
- name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
  id: 'deploy-billing'
  entrypoint: 'gcloud'
  args: ['run', 'services', 'update', '${_PREFIX}-billing-api',
         '--region', '${_REGION}', '--project', '$_TARGET_PROJECT',
         '--image', '${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/${_PREFIX}-billing:$_IMAGE_TAG']
  waitFor: ['build-billing', 'terraform-apply']
```

### cloudbuild-images.yaml additions

Same clone + build steps for manual image builds.

---

## Outputs: `terraform/outputs.tf` additions

```hcl
output "billing_service_url" {
  description = "URL of the billing Cloud Run service"
  value       = var.bootstrap ? "" : google_cloud_run_v2_service.billing[0].uri
}
```

---

## setup-secrets.sh additions

Add to the `scripts/setup-secrets.sh` interactive flow:

```bash
# Stripe secrets
read -sp "Stripe secret key: " STRIPE_SECRET_KEY
echo
gcloud secrets versions add "${PREFIX}-stripe-secret-key" \
  --project="${PROJECT}" --data-file=<(echo -n "$STRIPE_SECRET_KEY")

read -sp "Stripe webhook secret: " STRIPE_WEBHOOK_SECRET
echo
gcloud secrets versions add "${PREFIX}-stripe-webhook-secret" \
  --project="${PROJECT}" --data-file=<(echo -n "$STRIPE_WEBHOOK_SECRET")
```

---

## Phase Mapping

| Phase | What to Do |
|-------|-----------|
| 1 | Add `billing.tf`, secrets, variables, env configs. Run `terraform apply` to create service account + secrets + indexes. |
| 2 | Add Cloud Build steps. Deploy billing service image after packages are built. |
| 3 | No infra changes (agent assistance is app-level). |
| 4 | Add monitoring dashboards, alert policies (Cloud Monitoring Terraform). |

---

## Checklist

- [ ] Create `billing.tf` with Cloud Run service + IAM + service account
- [ ] Add Stripe secrets to `secrets.tf`
- [ ] Add Firestore composite indexes to `firestore.tf` (in docparse DB)
- [ ] Add billing variables to `variables.tf`
- [ ] Add env configs to `environments/dev/terraform.tfvars` and `environments/prod/terraform.tfvars`
- [ ] Add billing output to `outputs.tf`
- [ ] Add Cloud Build steps to `cloudbuild.yaml` and `cloudbuild-images.yaml`
- [ ] Update `scripts/setup-secrets.sh` with Stripe secret prompts
- [ ] Verify `sa-billing` has correct IAM roles after apply
- [ ] Set Stripe secret values in dev environment
- [ ] Test webhook delivery to billing service URL
