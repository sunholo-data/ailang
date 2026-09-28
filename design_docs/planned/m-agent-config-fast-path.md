# M-AGENT-CONFIG-FAST-PATH

**Status:** Implemented in `ailang-multivac` 2026-09-13, **pending one manual step**:
the triggers stack is applied as `m@sunholo.com` (`make triggers-apply`), which this
session cannot run.
**Requested by:** Mark, attended 2026-09-13 — "why does it take so much to change a
configuration for an agent? I'd like it easier", then "do B now".
**Companion:** `ailang coordinator agent-set` (option A, shipped `0f5b24d00`) makes the
operator do one step; this makes the machine do less work per step.

## Problem

Changing one field on one agent — a model pin — cost about ten minutes and six
interactions. The interactions are A's problem. This document is about the ten minutes.

`config/**` and `terraform/**` share a trigger
(`ailang-multivac-config-{dev,test,prod}` → `cloudbuild-config-only.yaml`), so **every**
agent registry edit runs `terraform init → plan → apply → roll coordinator + mcp`,
three times, once per environment. For a config-only change the plan is a guaranteed
no-op (`exit 0`, apply skipped) and `init` against the environment's remote backend
dominates the build.

None of that validates the agent entry. An agent registry entry is **data the
coordinator reads at startup**, not infrastructure; it inherits terraform's ceremony
because it lives in the same directory as terraform's state. The checks that actually
catch agent mistakes are local and take a second: strict YAML (`UnknownConfigKeys`),
the registry load, and `ailang coordinator agent-check`.

## What is NOT the problem

- **The dev → test → prod ladder.** Each rung regenerates that environment's bucket and
  rolls its coordinator. Skipping one leaves that environment behind, and its next
  config build silently reverts the change — that is exactly how six `daneel-design-*`
  agents were deleted on 2026-09-10. The ladder stays.
- **The roll.** The gcsfuse mount sees a new file in seconds, but the registry is built
  **once at startup**, so routing follows the config loaded then until a new revision
  serves (measured 2026-09-10: an inbox declared `triage_only` still bounced after a
  successful write). Removing the roll is option C and is deliberately not done here.

## Design

A second trigger on the same branch ladder, for `config/config.cloud.yaml` alone:

| | `ailang-multivac-config-*` | `ailang-multivac-agents-*` (new) |
|---|---|---|
| fires on | `config/**`, `terraform/**` minus `config.cloud.yaml` | `config/config.cloud.yaml` |
| runs | terraform init → plan → apply → roll coordinator + mcp | upload config.yaml → roll coordinator |
| for | templates, infrastructure | agent registry entries |

**Terraform stays consistent, and this is the load-bearing detail.** The config object is
terraform-managed — `terraform/config_storage.tf`,
`google_storage_bucket_object.coordinator_config`, with
`source = ../config/config.cloud.yaml` — and terraform detects change by content hash
against that file. The fast path uploads **that exact file** to **that exact object**, so
afterwards the object already holds what terraform would upload and the next full apply
is a no-op for it. Uploading anything else — rendered, merged, reformatted, or
`gsutil -Z` compressed — would be permanent drift that the next apply silently reverts.

Only the coordinator is rolled: it is the only service that reads the agent registry.
The MCP reads the same bucket but not this part of it, and rolling it here would double
the build for nothing.

### Two assertions the pipeline makes about itself

Both exist because this repo's recurring defect is a step that reports OK without doing
the thing:

- The target bucket must exist **before** anything is written. A `cp` to a missing
  bucket fails loudly, but a `cp` to the wrong existing bucket succeeds and deploys
  nothing.
- The uploaded object's md5 must equal the local file's. An upload that reports success
  and leaves the old object is indistinguishable from a no-op deploy.

## What this does not fix

A push that changes both `terraform/` and `config.cloud.yaml` fires both triggers and
does the roll twice. Correct, mildly wasteful, and not worth a coordination mechanism.

## Implementation

Landed in `ailang-multivac`:

- `cloudbuild-agents-only.yaml` — verify target → upload with md5 assertion → roll.
- `terraform-triggers/triggers.tf` — `google_cloudbuild_trigger.agents` for each env, and
  `config/config.cloud.yaml` added to the `config` trigger's `ignored_files`. Note
  `ignoredFiles` only suppresses when **every** changed file is ignored, so a mixed push
  still fires the terraform path.

**Remaining step, for Mark:** `make triggers-apply` in `ailang-multivac`, as
`m@sunholo.com`. Until then the new triggers do not exist and agent changes keep taking
the terraform path — which is the safe direction for a half-applied state: the old route
still works, so nothing is broken while this waits.

`agent-set`'s ladder waits on the `ailang-multivac-config-*` triggers by name. When the
new triggers are live it must wait on `ailang-multivac-agents-*` instead, or it will
watch a build that never fires and time out — a one-line change, deliberately held until
the triggers exist so the two cannot disagree in between.

## Validation criteria

- A push touching only `config.cloud.yaml` fires `ailang-multivac-agents-<env>` and
  **not** `ailang-multivac-config-<env>`.
- The build is materially faster than the terraform path — the claim to check, not
  assume, since `init` time is the whole premise.
- `terraform plan` immediately afterwards reports **no changes** to
  `google_storage_bucket_object.coordinator_config`. If it wants to re-upload, the fast
  path is writing different bytes and the design is wrong.
- `ailang coordinator agent-check <id>` reports `repo/live in sync` and a coordinator
  rolled after the write.
- A push touching `terraform/` still fires the terraform path.
- A push touching both fires both, and neither leaves the environment behind.

## Alternatives considered

**Hot-reload the registry** (option C): the coordinator watches the mounted file instead
of reading it once. Removes the roll entirely — but the roll is currently what makes
"written" and "in effect" the same event, and losing that returns us to a world where
the bucket can disagree with what is running. Worth doing later, with a version-stamped
reload that says which generation is live, and not before.

**Move agent entries to their own file** (`config/agents/*.yaml`): cleaner separation and
would make the path filter obvious rather than a single-file special case. Rejected for
now only because it moves ~1300 lines of commented rulings, and this change is meant to
be small enough to verify by reading.
