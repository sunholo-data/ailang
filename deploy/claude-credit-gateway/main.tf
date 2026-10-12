# REVIEW CANDIDATE ONLY. Copy into ailang-multivac and use its branch-driven CI.
# No provider API key or signing key value is accepted by Terraform.
terraform {
  required_version = ">= 1.5"
  required_providers {
    google      = { source = "hashicorp/google", version = "~> 5.0" }
    google-beta = { source = "hashicorp/google-beta", version = "~> 5.0" }
  }
}
locals {
  provision_count = var.provision ? 1 : 0
  runtime_count   = var.provision && var.deploy_runtime ? 1 : 0
  active_envs     = var.provision ? var.environments : {}
  runtime_jobs = var.provision && var.deploy_runtime ? merge([
    for env, cfg in var.environments : {
      "${env}/default" = { env = env, cfg = cfg, variant = "", image = cfg.agent_image }
      "${env}/go"      = { env = env, cfg = cfg, variant = "-go", image = cfg.agent_go_image }
    }
  ]...) : {}
  service_name = "ailang-claude-credit-gateway"
  gateway_url  = var.provision ? "https://${local.service_name}-${data.google_project.production[0].number}.${var.region}.run.app" : ""
  coordinators = join(",", [for env, cfg in var.environments : "${cfg.coordinator}=${cfg.prefix}-claude-credit@${cfg.project}.iam.gserviceaccount.com"])
  callers      = var.provision ? toset(concat(tolist(var.operators), [for cfg in var.environments : cfg.coordinator], [for cfg in var.environments : "${cfg.prefix}-claude-credit@${cfg.project}.iam.gserviceaccount.com"])) : toset([])
}
# Preconditions reject unsafe review inputs before resource mutation.
resource "terraform_data" "review_gate" {
  input = { provision = var.provision, runtime = var.deploy_runtime }
  lifecycle {
    precondition {
      condition     = !var.deploy_runtime || var.provision
      error_message = "Runtime requires provision=true."
    }
    precondition {
      condition     = !var.provision || (var.billing_account != "" && var.authority_project_id != var.production_project && alltrue([for cfg in var.environments : var.authority_project_id != cfg.project]))
      error_message = "Authority must be a separate private project, with explicit billing account."
    }
    precondition {
      condition     = !var.deploy_runtime || (var.organization != "" && var.workspace != "" && length(var.operators) > 0 && length(var.environments) > 0)
      error_message = "Runtime requires verified organization/workspace, operators and environment identities."
    }
    precondition {
      condition     = !var.deploy_runtime || (can(regex("@sha256:[0-9a-f]{64}$", var.gateway_image)) && alltrue([for cfg in var.environments : can(regex("@sha256:[0-9a-f]{64}$", cfg.agent_image)) && can(regex("@sha256:[0-9a-f]{64}$", cfg.agent_go_image))]))
      error_message = "All runtime images require verified immutable digests, never latest."
    }
    precondition {
      condition     = alltrue(flatten([for cfg in var.environments : [for name in keys(cfg.config_env) : !can(regex("KEY|TOKEN|SECRET|AUTH|CREDIT|ANTHROPIC", name))]]))
      error_message = "config_env cannot inject credentials or override guarded lane controls."
    }
  }
}
resource "google_project" "authority" {
  count               = local.provision_count
  project_id          = var.authority_project_id
  name                = "AILANG Claude credit authority"
  folder_id           = var.folder_id
  billing_account     = var.billing_account
  auto_create_network = false
  lifecycle { prevent_destroy = true }
  depends_on = [terraform_data.review_gate]
}
resource "google_project_service" "authority_firestore" {
  count              = local.provision_count
  project            = google_project.authority[0].project_id
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}
resource "google_firestore_database" "authority" {
  count                   = local.provision_count
  project                 = google_project.authority[0].project_id
  name                    = "(default)"
  location_id             = var.region
  type                    = "FIRESTORE_NATIVE"
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
  depends_on              = [google_project_service.authority_firestore]
  lifecycle { prevent_destroy = true }
}
# The gateway runs in prod; its private ledger is in a different project.
data "google_project" "production" {
  count      = local.provision_count
  project_id = var.production_project
}
resource "google_service_account" "gateway" {
  count        = local.provision_count
  project      = var.production_project
  account_id   = "ailang-claude-credit-gateway"
  display_name = "Claude API credit authority (provider key only here)"
}
resource "google_project_iam_member" "gateway_ledger" {
  count   = local.provision_count
  project = google_project.authority[0].project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.gateway[0].email}"
}
resource "google_secret_manager_secret" "signing" {
  count     = local.provision_count
  project   = var.production_project
  secret_id = "ailang-claude-credit-signing-key"
  replication {
    auto {}
  }
  lifecycle { prevent_destroy = true }
}
# Authoritative secret-level reader lists. Existing project/folder broad grants
# MUST be removed separately; these bindings cannot negate inherited access.
resource "google_secret_manager_secret_iam_binding" "provider_key" {
  count     = local.provision_count
  project   = var.production_project
  secret_id = "ailang-anthropic-api-key"
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.gateway[0].email}"]
}
resource "google_secret_manager_secret_iam_binding" "signing_key" {
  count     = local.provision_count
  project   = var.production_project
  secret_id = google_secret_manager_secret.signing[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.gateway[0].email}"]
}
resource "google_cloud_run_v2_service" "gateway" {
  count    = local.runtime_count
  project  = var.production_project
  name     = local.service_name
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"
  # The stable URL is also the application JWT audience. Cloud Run otherwise
  # accepts only its generated service URL, which differs on this deployment.
  custom_audiences = [local.gateway_url]
  template {
    service_account                  = google_service_account.gateway[0].email
    timeout                          = "660s"
    max_instance_request_concurrency = 4
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    containers {
      image = var.gateway_image
      # Keep the coordinator buildpack launcher; never override command.
      args = ["credit-gateway", "serve", "--ledger-project", var.authority_project_id, "--gateway-url", local.gateway_url, "--provider-key-file", "/secrets/provider/key", "--signing-key-file", "/secrets/signing/key", "--operators", join(",", sort(tolist(var.operators))), "--coordinators", local.coordinators, "--organization", var.organization, "--workspace", var.workspace, "--listen", ":8080"]
      resources {
        limits   = { cpu = "1", memory = "512Mi" }
        cpu_idle = true
      }
      ports { container_port = 8080 }
      volume_mounts {
        name       = "provider-key"
        mount_path = "/secrets/provider"
      }
      volume_mounts {
        name       = "signing-key"
        mount_path = "/secrets/signing"
      }
    }
    volumes {
      name = "provider-key"
      secret {
        secret = "projects/${var.production_project}/secrets/ailang-anthropic-api-key"
        items {
          version = "latest"
          path    = "key"
        }
      }
    }
    volumes {
      name = "signing-key"
      secret {
        secret = google_secret_manager_secret.signing[0].id
        items {
          version = "latest"
          path    = "key"
        }
      }
    }
  }
  depends_on = [google_project_iam_member.gateway_ledger, google_firestore_database.authority, google_secret_manager_secret_iam_binding.provider_key, google_secret_manager_secret_iam_binding.signing_key, terraform_data.review_gate]
  lifecycle { ignore_changes = [client, client_version, template[0].containers[0].image] }
}
resource "google_cloud_run_v2_service_iam_member" "gateway_caller" {
  for_each = var.deploy_runtime ? local.callers : toset([])
  project  = var.production_project
  location = var.region
  name     = google_cloud_run_v2_service.gateway[0].name
  role     = "roles/run.invoker"
  member   = contains(var.operators, each.value) && !endswith(each.value, ".iam.gserviceaccount.com") ? "user:${each.value}" : "serviceAccount:${each.value}"
}
# One dedicated guarded SA per environment, shared by default/go images. No
# authority project or provider/signing secret permissions belong to these SAs.
resource "google_service_account" "job" {
  for_each     = local.active_envs
  project      = each.value.project
  account_id   = "${each.value.prefix}-claude-credit"
  display_name = "Claude guarded executor; no provider credentials"
}
resource "google_project_iam_member" "job_tasks" {
  for_each = local.active_envs
  project  = each.value.project
  role     = "roles/datastore.user"
  member   = "serviceAccount:${google_service_account.job[each.key].email}"
}
resource "google_pubsub_topic_iam_member" "job_events" {
  for_each = var.provision ? merge([for env, cfg in var.environments : { for topic in ["completions", "events"] : "${env}/${topic}" => { env = env, cfg = cfg, topic = topic } }]...) : {}
  project  = each.value.cfg.project
  topic    = "${each.value.cfg.topic_prefix}-${each.value.topic}"
  role     = "roles/pubsub.publisher"
  member   = "serviceAccount:${google_service_account.job[each.value.env].email}"
}
resource "google_secret_manager_secret_iam_member" "job_github" {
  for_each  = local.active_envs
  project   = each.value.project
  secret_id = each.value.github_secret
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.job[each.key].email}"
}
resource "google_storage_bucket_iam_member" "job_artifacts" {
  for_each = local.active_envs
  bucket   = each.value.artifacts_bucket
  role     = "roles/storage.objectAdmin"
  member   = "serviceAccount:${google_service_account.job[each.key].email}"
}
resource "google_storage_bucket_iam_member" "job_artifacts_mount" {
  for_each = local.active_envs
  bucket   = each.value.artifacts_bucket
  role     = "roles/storage.legacyBucketReader"
  member   = "serviceAccount:${google_service_account.job[each.key].email}"
}
resource "google_project_iam_member" "job_trace" {
  for_each = local.active_envs
  project  = each.value.project
  role     = "roles/cloudtrace.agent"
  member   = "serviceAccount:${google_service_account.job[each.key].email}"
}
resource "google_cloud_run_v2_job" "guarded" {
  provider = google-beta
  for_each = local.runtime_jobs
  project  = each.value.cfg.project
  name     = "${each.value.cfg.prefix}-agent-executor${each.value.variant}-claude-credit"
  location = var.region
  template {
    task_count = 1
    template {
      service_account = google_service_account.job[each.value.env].email
      timeout         = "3300s" # 50m maximum agent run plus cleanup; no task retry.
      max_retries     = 0
      dynamic "vpc_access" {
        for_each = each.value.cfg.egress == null ? [] : [each.value.cfg.egress]
        content {
          egress = "ALL_TRAFFIC"
          network_interfaces {
            network    = vpc_access.value.network
            subnetwork = vpc_access.value.subnetwork
          }
        }
      }
      containers {
        image = each.value.image
        resources { limits = { cpu = "2", memory = each.value.variant == "-go" ? "8Gi" : "4Gi" } }
        dynamic "env" {
          for_each = merge(each.value.cfg.config_env, { AILANG_CLOUD_PROJECT = each.value.cfg.project, AILANG_CLOUD_REGION = var.region, AILANG_STORAGE = "gcp", AILANG_TOPIC_PREFIX = each.value.cfg.topic_prefix, AILANG_AUTH_MODE = "apikey", AILANG_PROVIDER = "claude", AILANG_IMAGE_PROVIDER = "claude", AILANG_CLAUDE_CREDIT_ACCOUNT = "anthropic-api-credits", AILANG_MAX_COST_USD = "2", AILANG_GIT_MODE = "guardrails" })
          content {
            name  = env.key
            value = env.value
          }
        }
        env {
          name = "GITHUB_TOKEN"
          value_source {
            secret_key_ref {
              secret  = each.value.cfg.github_secret
              version = "latest"
            }
          }
        }
        volume_mounts {
          name       = "artifacts"
          mount_path = "/artifacts"
        }
      }
      volumes {
        name = "artifacts"
        gcs {
          bucket    = each.value.cfg.artifacts_bucket
          read_only = false
        }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.job_github, google_cloud_run_v2_service.gateway, terraform_data.review_gate]
  lifecycle { ignore_changes = [client, client_version, template[0].template[0].containers[0].image] }
}
resource "google_cloud_run_v2_job_iam_member" "dispatch" {
  for_each = local.runtime_jobs
  project  = each.value.cfg.project
  location = var.region
  name     = google_cloud_run_v2_job.guarded[each.key].name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${each.value.cfg.coordinator}"
}
resource "google_service_account_iam_member" "dispatch_act_as" {
  for_each           = local.active_envs
  service_account_id = google_service_account.job[each.key].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${each.value.coordinator}"
}
output "gateway_url" { value = local.gateway_url }
output "authority_project" { value = var.authority_project_id }
