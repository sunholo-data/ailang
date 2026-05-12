# Cloud coordinator config: how it works, how to update it

The cloud coordinator daemon (Cloud Run service `ailang{,-dev}-coordinator`)
loads its agent registry from a per-project GCS bucket, mounted via gcsfuse.
This doc covers:
- Where the config actually lives in each project
- How to add an agent for a new package family (the common case)
- The known gap: cascade tasks fail silently when no agent matches a
  `pkg:*` inbox (tracked in
  [M-COORDINATOR-INBOX-WILDCARDS](../../design_docs/planned/v0_19_0/m-coordinator-inbox-wildcards.md))

## TL;DR

| Project | Config bucket | Coordinator service |
|---|---|---|
| `ailang-multivac-dev` | `gs://ailang-multivac-dev-ailang-config/config.yaml` | `ailang-dev-coordinator` |
| `ailang-multivac` (prod) | `gs://ailang-multivac-ailang-config/config.yaml` | `ailang-coordinator` |

Update flow:
```bash
# 1. Pull current config
gsutil cp gs://ailang-multivac-dev-ailang-config/config.yaml /tmp/cfg.yaml

# 2. Edit /tmp/cfg.yaml — add agents under coordinator.agents
nano /tmp/cfg.yaml

# 3. Push back
gsutil cp /tmp/cfg.yaml gs://ailang-multivac-dev-ailang-config/config.yaml

# 4. Restart the coordinator so it re-reads
#    (gcsfuse caches; the simplest cycle is a revision restart)
gcloud run services update ailang-dev-coordinator \
  --project ailang-multivac-dev --region europe-west1 \
  --update-labels=config-restart=$(date +%s)

# 5. Verify with the cloud-cascade-debug skill
.claude/skills/cloud-cascade-debug/scripts/triage_cascade.sh \
  sunholo/<your-pkg>@<version> 1
```

## Why this matters

`ailang publish` emits a Pub/Sub message per dependent to the cascade topic
(`ailang{-dev}-cascade`). The coordinator subscribes, creates a task per
message, looks up the right agent via `AgentRegistry.GetAgentForInbox(
"pkg:vendor/name")`, and dispatches to a Cloud Run Job with
`AILANG_AGENT_ID` set from the resolved agent.

If no agent is registered for that exact inbox, the task is created with
`agent: ""` and the Cloud Run Job exits 1 immediately:

```
error=AILANG_AGENT_ID environment variable is required
```

The publishing CLI prints `Cascade-topic notification published` regardless,
so failures are invisible at publish time.

## Adding a new package agent (today, exact-match only)

Until [M-COORDINATOR-INBOX-WILDCARDS](../../design_docs/planned/v0_19_0/m-coordinator-inbox-wildcards.md)
ships, every package needs its own entry. Append under
`coordinator.agents`:

```yaml
- id: pkg-sunholo-mypackage
  label: "pkg sunholo/mypackage"
  inbox: "pkg:sunholo/mypackage"
  workspace: sunholo-data/ailang-packages
  merge_branch: main
  capabilities: [code, package]
  # Optional but recommended for cascade work:
  default_provider: claude
  max_cost_usd: 0.05
```

Add one entry per package in the family. For the motoko_ext family
(13 packages) that's 13 entries. This is the manual tax that the wildcard
work removes.

## Adding a new package agent (after wildcards ship)

After M-COORDINATOR-INBOX-WILDCARDS, one entry per FAMILY:

```yaml
- id: pkg-motoko-ext-cascade-bumper
  label: "Cascade bumper for motoko_ext_*"
  inbox: "pkg:sunholo/motoko_ext_*"   # trailing * = family glob
  workspace: sunholo-data/ailang-packages
  merge_branch: main
  capabilities: [code, package, cascade]
```

Longest-prefix-wins, so an explicit `pkg:sunholo/motoko_ext_abi` entry
still overrides the family glob if you want different settings for one
specific package.

## Dev-vs-prod promotion

Pattern: changes go to dev first, validated, then promoted. Dev is the
target of laptop publishes (`AILANG_CLOUD_PROJECT=ailang-multivac-dev`).

```bash
# Promote dev → prod
gsutil cp gs://ailang-multivac-dev-ailang-config/config.yaml /tmp/dev-cfg.yaml
# Manually merge into prod (don't blind-overwrite — prod has its own
# project_id/secret-name/repo overrides)
gsutil cp gs://ailang-multivac-ailang-config/config.yaml /tmp/prod-cfg.yaml
# ... merge the new agents block ...
gsutil cp /tmp/prod-cfg.yaml gs://ailang-multivac-ailang-config/config.yaml
gcloud run services update ailang-coordinator \
  --project ailang-multivac --region europe-west1 \
  --update-labels=config-restart=$(date +%s)
```

## Diagnostics

Use the `cloud-cascade-debug` skill to triage cascade failures:

```bash
.claude/skills/cloud-cascade-debug/scripts/triage_cascade.sh \
  sunholo/motoko_ext_abi@2.1.0 12
```

Inspects validator HTTP logs, cascade Pub/Sub topic state, coordinator
task lifecycle, and recent Cloud Run Job execution status. The "Tasks
created with empty `agent: ` field" line is the signature of a missing
agent registration.

## Open questions

- **Hot-reload**: today gcsfuse caches the config; updates require a
  Cloud Run revision restart (the `--update-labels=config-restart=...`
  trick). The proper fix is a coordinator-side filewatcher + atomic
  swap. Not in scope for v0.19.0; tracked under M2 of the wildcards doc.
- **Wildcard precedence semantics**: longest-prefix wins is intuitive
  but a single `pkg:*` catch-all + a per-package override has potential
  for surprise. Document precedence rules clearly when wildcards land.
- **Failure visibility**: even with wildcards, `agent: ""` empty-AgentID
  bugs can still happen (e.g. typo in inbox name). The wildcards doc
  proposes a refusal-to-dispatch + structured warning; pairs naturally
  with a CLI-side "publish saw zero confirmed cascades" warning.
