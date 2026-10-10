# Claude credit gateway infrastructure candidate

This directory contains the reusable reviewed module. The owning private
infrastructure stack provisions the shared dev/prod authority through Terraform CI.
Module defaults stay off (`provision=false`, `deploy_runtime=false`); actual
rollout status and activation evidence are linked below.
Use the private `ailang-multivac` repo's branch-driven infrastructure pipeline;
do not run a local apply or build/push production images from this directory.

Read [deployment evidence](../../design_docs/planned/m-cloud-claude-api-credit-guard-deployment.md)
for live resource names, blocking IAM findings and the outstanding grant fields.

The module creates a private Firestore project because existing executor SAs can
write the production database. Its gateway runs in `ailang-multivac`, with access
only to that private ledger and the production provider/signing secrets. The
provider secret already exists; Terraform creates the signing secret resource,
never either secret value. Provision resources first; securely add a random
signing key of at least 32 bytes before creating the Cloud Run runtime.

`runtime-secret-iam.patch` is a separate, concrete patch for
`ailang-multivac/terraform/{iam,billing}.tf`. It removes broad secret-reader grants
from coordinator, dashboard, website-builder and billing and preserves their
explicitly declared secret dependencies. Review runtime secret fetches before
promotion. Audit inherited readers too: a secret-level allowlist cannot cancel a
project/folder grant. Ordinary coordinator/task identities must never read the
provider/signing key or edit the private authority's data.

One state owner must manage the shared authority and both environments' guarded
jobs. Copy this directory into the infrastructure repo as a module, with an
explicit shared production-stack owner. Environment examples are:

```hcl
environments = {
  dev = {
    project          = "ailang-multivac-dev"
    prefix           = "ailang-dev"
    coordinator      = "ailang-dev-coordinator@ailang-multivac-dev.iam.gserviceaccount.com"
    agent_image      = "<verified dev agent@sha256 digest>"
    agent_go_image   = "<verified dev agent-go@sha256 digest>"
    github_secret    = "ailang-dev-github-token"
    topic_prefix     = "ailang-dev"
    artifacts_bucket = "ailang-multivac-dev-ailang-artifacts"
    # Pass existing local.agent_config_env and egress network/subnet from the
    # owning infrastructure stack. No model credentials belong in config_env.
  }
  prod = {
    project          = "ailang-multivac"
    prefix           = "ailang"
    coordinator      = "ailang-coordinator@ailang-multivac.iam.gserviceaccount.com"
    agent_image      = "<verified prod agent@sha256 digest>"
    agent_go_image   = "<verified prod agent-go@sha256 digest>"
    github_secret    = "ailang-github-token"
    topic_prefix     = "ailang"
    artifacts_bucket = "ailang-multivac-ailang-artifacts"
  }
}
```

Operators are exact email identities, independent from coordinators. An operator
may be a Google user or a dedicated service account; the IAM member type follows
the email suffix. With a dedicated operator SA, `coordinator credits` supports
`--impersonate-service-account`; grant only the human operator scoped Token Creator
on that SA, never an executor/coordinator. Never make the gateway unauthenticated.

Preferred service build: reuse the source release's verified coordinator buildpack
image. The Cloud Run resource passes `credit-gateway serve` args and leaves its
launcher intact. Jobs use verified released `agent` and `agent-go` digests; their
new names must enter the existing release/promote job maps. Both jobs share a new
guarded SA per environment. Jobs get only task-bound gateway capabilities and
Google identity tokens; no real provider key, OAuth token, signing key, or ledger
permissions.

After creation, the normal verified image release pipeline owns gateway/job
image updates. Terraform ignores subsequent image changes so a config apply
cannot restore an older reviewed digest over a later verified release.

The optional Dockerfile demonstrates a minimal Go-only service build. Its build
context is the full ailang checkout and Go version matches go.mod. If adopted,
add this image to the normal source release, image verification and promotion
pipeline. It is not a shortcut for an unverified production build.

Local validation of the module is `terraform fmt -check` and `terraform validate`
after `init -backend=false` with the pinned providers. Those commands do not prove
resource access, grant ownership, service compatibility or deployed isolation.
Inspect a read-only target plan through the infrastructure workflow before
provisioning, then test status/auth/kill-switch before any billable canary.
