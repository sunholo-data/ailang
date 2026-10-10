# Claude API credit deployment review — M0/M6

**Recorded:** 2026-10-10. **State:** review candidate; no infrastructure applied,
images built/pushed, credit account enabled, or billable inference performed.

## Confirmed grant and remaining evidence

Mark identified the provider credential as
`projects/ailang-multivac/secrets/ailang-anthropic-api-key/versions/latest`.
He confirmed that the full $200 is available and allocated to this workload and
expires on 2026-10-28. The conservative admission expiry is
`2026-10-28T00:00:00Z`, pending the provider's exact expiry timestamp/timezone.
This record contains no credential value and does not itself create a grant.

The Console organization ID, dedicated workspace ID, grant/cycle identifier and
start timestamp still need recording. A secret resource name cannot establish its
organization, workspace, or allocation. Activation and the live canary remain
blocked until those identities, provider-side caps and grant evidence agree.
No calendar rollover or key replacement renews the AILANG allowance.

## Actual cloud deployment inventory

### Installed CLI compatibility blocker

The opt-in `TestClaudeCLIProtocolSmoke` uses the installed Claude CLI **2.1.295**,
an empty home directory, no tools, synthetic credentials and a localhost gateway
with a fake-only upstream. On 2026-10-10 it failed before any reservation or
upstream inference. Despite experimental features being disabled, the CLI sends:

| Required beta header | Current decision |
|---|---|
| `interleaved-thinking-2025-05-14` | Rejected pending a reviewed supported contract |
| `mid-conversation-system-2026-04-07` | Rejected pending a reviewed supported contract |
| `claude-code-20250219` | Rejected pending a reviewed supported contract |
| `effort-2025-11-24` | Rejected pending a reviewed supported contract |

Pinning `anthropic-beta` through `ANTHROPIC_CUSTOM_HEADERS` did not override the
CLI's native list, so the ineffective override was removed. The guarded child
also disables tool search and separate server auto-mode review; those controls
do not remove this mandatory list. Default Go tests passing therefore does not
establish actual client compatibility. Diagnostic output is retained at
`/private/tmp/claude-credit-cli-protocol-final.log`.

Automatic approval review rejected the proposed four-header allowlist expansion:
it could weaken the fail-closed billing guard by admitting unverified features
with different pricing or unbounded usage. The rejected change was not applied
and the headers were not stripped or bypassed. M4 and activation remain blocked.

A concrete compatibility follow-up would review those four features individually,
retain the strict Messages-only body contract and helper model pinning, prove the
same maximum-cost reservation covers every admitted feature, and rerun this
fake-only test before a reserved live canary. Allowlist expansion needs explicit
approval and pricing evidence; approval alone cannot establish the spending bound.
Anthropic warns that beta features may have different pricing. The documented
thinking/output hard cap and effort controls are useful evidence, but do not by
themselves verify every flag in this CLI's mandatory list.
[Beta headers](https://platform.claude.com/docs/en/api/beta-headers),
[thinking and cost](https://platform.claude.com/docs/en/build-with-claude/thinking-steering-and-cost),
[effort](https://platform.claude.com/docs/en/build-with-claude/effort),
[gateway protocol](https://code.claude.com/docs/en/llm-gateway-protocol).

Read-only `gcloud run jobs/services/revisions describe` queries on 2026-10-10:

| Environment | Existing API-key job | Runtime identity | Image |
|---|---|---|---|
| Prod, `ailang-multivac`, `europe-west1` | `ailang-agent-executor-apikey` | `ailang-agent-external@ailang-multivac.iam.gserviceaccount.com` | `europe-west1-docker.pkg.dev/ailang-multivac/ailang/agent@sha256:83dd4c6deec4d5992b84fccbe3f3ebaedea4bba74c7c0f09efdead4291a98468` |
| Dev, `ailang-multivac-dev`, `europe-west1` | `ailang-dev-agent-executor-apikey` | `ailang-dev-agent-external@ailang-multivac-dev.iam.gserviceaccount.com` | `europe-west1-docker.pkg.dev/ailang-multivac-dev/ailang/agent:latest` |

| Coordinator | Ready revision | Immutable image | Identity |
|---|---|---|---|
| `ailang-coordinator` | `ailang-coordinator-00194-bhf` | `europe-west1-docker.pkg.dev/ailang-multivac/ailang/coordinator@sha256:9fa1e4919498901518235e1534b76a372a4a8513eeeed2e5cefaa86d1e476f53` | `ailang-coordinator@ailang-multivac.iam.gserviceaccount.com` |
| `ailang-dev-coordinator` | `ailang-dev-coordinator-03424-tg2` | `europe-west1-docker.pkg.dev/ailang-multivac-dev/ailang/coordinator@sha256:e1f2d5b8af75e99cf88b26b37f13489fb1b9c4ae5fbb8860852d3345afdbdf27` | `ailang-dev-coordinator@ailang-multivac-dev.iam.gserviceaccount.com` |

Existing artifact buckets are `ailang-multivac-ailang-artifacts` and
`ailang-multivac-dev-ailang-artifacts`. Production Firestore currently has
`(default)`, `docparse` and `website-builder`, all in `europe-west1`.
The Anthropic secret resource exists; its resource creation time is
2026-04-01, which says nothing about the new key's current version or owner.
No secret payload or sensitive environment value was read.

Infrastructure owner is the private sibling checkout
`/Users/voightkampff/dev/sunholo-data/ailang-multivac`, specifically
`terraform/cloud_run_jobs.tf`, `terraform/iam.tf`, `terraform/agent_egress.tf`,
`terraform/billing.tf`, `cloudbuild-config-only.yaml` and
`scripts/cloudbuild-lib.sh`. Its `CLAUDE.md` requires branch-driven CI for
persistent changes, dev → test → prod infrastructure promotion, and source
release/tag promotion for images. No local `terraform apply` is appropriate.
The existing infrastructure rule against provider keys in executor jobs remains
satisfied: this lane gives jobs gateway capabilities only.

## Security findings that block activation

The live production IAM policy grants `roles/datastore.user` to both the existing
API-key executor and coordinator. Other executor lanes also have this role in
Terraform. Putting credit records in the existing production default database
would let a job forge grants or accounting. A collection name is not an IAM
boundary for Firestore server clients. The review candidate therefore uses a
**separate private authority project**, proposed `ailang-credit-authority`, whose
Firestore database is accessible to the gateway SA alone among runtime services.
Both dev and prod use this same authority; their jobs/coordinators have no ledger
IAM binding there. Verify project/folder/organization inherited roles as well.

Live production `roles/secretmanager.secretAccessor` project bindings currently
allow these runtime identities to retrieve the provider key:

- `ailang-coordinator@ailang-multivac.iam.gserviceaccount.com`
- `ailang-dashboard@ailang-multivac.iam.gserviceaccount.com`
- `ailang-website-builder@ailang-multivac.iam.gserviceaccount.com`
- `ailang-billing@ailang-multivac.iam.gserviceaccount.com`

The concrete `runtime-secret-iam.patch` replaces those four broad grants with
explicit service-required secret bindings. It includes the dashboard's separate
approval signing key where enabled. Apply through infrastructure CI, inventory
runtime secret fetches, and regression-check those services. Granting gateway
access on the provider secret does **not** revoke inherited project-wide access.

Folder `389195706883` additionally has inherited secret-reader grants for
`sa-cloudbuild@ailang-multivac-deploy.iam.gserviceaccount.com` and
`sa-cloudbuild@multivac-deploy.iam.gserviceaccount.com`. The latter belongs to the
other Multivac estate according to the infrastructure guide. Trusted deployment
administrators can change IAM; they are outside the runtime request budget
boundary. Audit and explicitly approve or narrow those control-plane grants
before claiming the key has a closed allocation. Do not quietly treat them as
absent. These observations are read-only, not a full organization IAM audit.

## Concrete deployment candidate

[The candidate module](../../deploy/claude-credit-gateway/main.tf) and its README
provide two stages, both off by default. Provisioning creates the private authority,
gateway/job SAs, exact provider/signing secret reader lists, and scoped job access.
It creates a signing secret **resource only**; its random value of at least 32
bytes must be provisioned securely, never through Terraform values/state.
Runtime creation follows the IAM audit and verified image release.

The shared production gateway is `ailang-claude-credit-gateway`, running the
verified coordinator buildpack image with `credit-gateway serve` arguments.
There is no need to install Claude CLI in the gateway. The optional minimal
Dockerfile is a build candidate; it has not been built or vulnerability-scanned.
Cloud Run requires both Google IAM invocation and application authorization:
coordinators issue task-bound capabilities, jobs present their own Google ID token
and signed capability, and separate operator identities confirm/enable/disable
accounts. Provider and signing keys mount only in the gateway.

Each environment gets one dedicated guarded SA, `${prefix}-claude-credit`, shared
by default/go jobs named `${prefix}-agent-executor-claude-credit` and
`${prefix}-agent-executor-go-claude-credit`. Existing OAuth/API-key jobs are not
repurposed. The dedicated SA can read its GitHub credential and its task/artifact
stores, publish completion/events, and invoke the gateway. It has no provider,
OAuth, alternative model, signing, or authority-ledger access. CLI helper models
are pinned to Haiku 5.5; other billable routes/features fail closed. Tasks are at
most 50 minutes with zero platform retries; a failed attempt needs new admission.

Existing egress policy is preserved through explicit VPC/subnet inputs where it
is active. Infrastructure comments say the fleet lock is currently disabled;
this review does not claim a deployed network deny rule. When enabling it, allow
the gateway run.app endpoint and required task/telemetry APIs. Direct provider
connectivity alone cannot bill this account because the guarded job lacks its
key. IAM, rather than hooks or a prompt, prevents retrieval of that key.

## Rollout and retained verification

1. Review the module and IAM patch in the infrastructure repo. Assign one state
   owner to the shared authority and cross-project jobs, avoiding duplicate module
   application from both environment states. The production infrastructure stack
   may own the shared module; the cross-project resources must be explicit.
2. Plan target resources read-only and check existing resources/import requirements.
   Integrate deployment into branch-driven CI. Keep every Claude credit registry
   entry disabled while infrastructure and image release paths are completed.
3. Provision resources with `provision=true`, `deploy_runtime=false`. Remove broad
   runtime secret-reader grants, audit inheritance, and securely create the signing
   secret value. Do not fetch either key into an agent transcript.
4. Release source through test and promote verified coordinator/agent/agent-go
   digests. Extend release job/service maps for the new guarded resources; a new
   module alone does not teach the current pipeline to roll them.
5. Deploy runtime with verified organization/workspace/operator identities and
   images. Confirm the canonical account remains disabled after restart. Confirm
   gateway audience and Google token behavior with nonbillable admin/status calls.
6. Record the actual grant/cycle/start/expiry/evidence through
   `ailang coordinator credits confirm --remote gcp --gateway <origin>`. Review
   status before the separately audited `credits enable --yes` action.
7. Run the reserved canary only after all preceding gates pass. Its aggregate $5
   ceiling must include settled **and unresolved/reserved** exposure, be enforced
   server-side across all smoke attempts (implemented by the canary ledger), and sit inside the same $190 operating
   and $6 UTC daily ceilings. Per-task maximum remains $2; parallelism remains two.
   Do not infer this aggregate limit from a per-task flag.
8. Reconcile provider billing, request IDs and hidden CLI inference against the
   ledger, then use audited `credits promote` to lift the canary sublimit and promote selected registry entries. Unknown billable categories,
   unsupported beta/header behavior, or mismatches block promotion.

Rollback uses the audited `credits disable` command and disabled registry entries.
It stops new admission while retaining settled, reserved and unresolved exposure.
Key or signing-key rotation does not replace the account or clear exposure.
Fresh confirmed grants use a new stable grant ID and never erase old ambiguity.

Validation performed: Terraform fmt and validate using installed Google providers
5.45.2; the IAM patch parses/formats and passes `git apply --check` against the
inspected infrastructure checkout. Neither a live Terraform plan/apply nor a
container build, deployment, emulator integration, provider billing reconciliation,
live canary, release-pipeline update or CI gate is claimed by this artifact.
