# Cloud coordinator config: how it works, how to update it

The cloud coordinator daemon (Cloud Run service `ailang{,-dev}-coordinator`)
loads its agent registry from a per-project GCS bucket, mounted via gcsfuse.
This doc covers:
- Where the config actually lives in each project
- How to add an agent for a new package family (the common case)
- Inbox **patterns** (`pkg:sunholo/motoko_ext_*`), which since
  [M-COORDINATOR-INBOX-WILDCARDS](../../design_docs/planned/v0_29_0/m-coordinator-inbox-wildcards.md)
  let one entry serve a whole family
- The underscore-vs-hyphen naming trap that silently misroutes feedback

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

If no agent matches, the coordinator now **refuses to dispatch** and leaves the
message unread for triage, logging:

```
Skipping message <id>: no agent registered for inbox "<inbox>" (left unread for triage)
```

**Before 2026-08-25** it dispatched anyway with `agent: ""`, and the Cloud Run Job
exited 1 on arrival:

```
error=AILANG_AGENT_ID environment variable is required
```

The completion was then posted to inbox `""`, unreachable by every `--inbox` query —
36 such messages accumulated in prod and 787 in dev. The publishing CLI prints
`Cascade-topic notification published` either way, so the failure was invisible at
publish time. Old messages carry the second signature; grep for both.

## Adding a package agent: explicit entry

Use an explicit entry when the package needs its own scoping (`subdirectory`,
`artifact_patterns`, a tighter budget) or lives outside `ailang-packages`. An exact
`inbox:` **always wins over any pattern**, so it can override a family glob without
removing it. Append under `coordinator.agents`:

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

Prefer a family pattern (below) over one entry per package — that per-package tax is
what lost 10 cascades on an `abi 2.1.0` republish.

## Adding a package agent: family pattern (preferred)

One entry per FAMILY. A new member is routed the moment it is published, with no
config change:

```yaml
- id: pkg-motoko-ext-cascade-bumper
  label: "Cascade bumper for motoko_ext_*"
  inbox: "pkg:sunholo/motoko_ext_*"   # trailing * = family glob
  workspace: sunholo-data/ailang-packages
  merge_branch: main
  capabilities: [code, package, cascade]
```

Precedence: exact `inbox:` match, then longest matching pattern prefix
(`pkg:sunholo/motoko_ext_*` beats `pkg:sunholo/*` beats `pkg:*`), then no dispatch.
Only a trailing `*` is supported — full glob syntax and mid-pattern wildcards
(`pkg:*/motoko_*`) are deliberate non-goals.

Omit `subdirectory` on a family agent: it differs per package, and the prompt names the
package so the agent can locate `packages/<name>` itself.

## Naming: underscores vs hyphens

The registry spells package names with **underscores**; the repo directories use
**hyphens**:

| Registry package | Inbox | Directory |
|---|---|---|
| `sunholo/motoko_ext_abi` | `pkg:sunholo/motoko_ext_abi` | `packages/motoko-ext-abi` |
| `sunholo/ailang_parse` | `pkg:sunholo/ailang_parse` | `packages/ailang-parse` |

`FormatPackageInbox` prefixes `pkg:` with **no normalization and no existence check**,
so feedback submitted against the hyphen spelling mints an inbox nothing watches. On
2026-08-25 ten tickets landed in `pkg:sunholo/ailang-parse` while the agent watched
`pkg:sunholo/ailang_parse`. Always use the **registry** spelling.

## Verifying coverage

A package published with no matching entry or pattern accumulates unread mail nobody
acts on. To find them:

```bash
ailang search "" --limit 200 | grep -oE "sunholo/[a-z0-9_-]+" | sort -u   # published
grep -o 'inbox: "pkg:[^"]*"' config/config.cloud.yaml | sort -u           # routed
```

Measured 2026-08-25: 22 of 41 published packages had a curating agent; after adding the
`motoko_ext_*` family plus four orphans, 40 of 41. The remaining one, `sunholo/email`,
lives outside `ailang-packages` and needs its own workspace.

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
- ~~**Wildcard precedence semantics**~~ — RESOLVED 2026-08-25: longest-prefix
  wins with exact-match beating every pattern; see "family pattern" above.
- ~~**Failure visibility**~~ — PARTLY RESOLVED 2026-08-25: an unrouted inbox is now
  refused rather than dispatched, and the message is left unread. A typo in an inbox
  name still produces a *silent backlog* rather than a *silent failure*; the
  CLI-side "publish saw zero confirmed cascades" warning is still unbuilt, and
  `FormatPackageInbox` still applies no name validation.
