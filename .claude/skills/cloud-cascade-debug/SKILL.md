---
name: cloud-cascade-debug
description: Investigate why an `ailang publish` cascade didn't fire (or fired and silently failed) on multivac. Inspects gcloud logs across the registry validator, the Pub/Sub `ailang-cascade` topic, and the coordinator's task queue. Use when the user says "auto-cascade didn't work", "why didn't multivac pick this up", "check cloud logs for cascade", or "the dependents didn't get bumped".
model: opus
---

# AILANG Cloud Cascade Debugger

After `ailang publish` of a package whose dependents should auto-bump (the
M-PKG-AUTONOMOUS-CASCADE-SAFE pipeline), trace the message through:
**registry validator → ailang-cascade Pub/Sub topic → coordinator subscriber
→ task dispatch → Cloud Run Job execution**. Surfaces the first failing
hop with timestamps + log lines.

## When to use

User says any of:
- "auto-cascade didn't work" / "auto-cascade didn't fire"
- "why didn't multivac pick this up"
- "the dependents didn't get bumped"
- "check the cloud logs for the cascade"
- "I had to manually republish — what happened cloud-side?"

## Quick triage (60-second sanity check)

```bash
# Which project did the publish actually target?
echo "AILANG_CLOUD_PROJECT=$AILANG_CLOUD_PROJECT"
echo "AILANG_TOPIC_PREFIX=$AILANG_TOPIC_PREFIX"
```

If `AILANG_CLOUD_PROJECT=ailang-multivac-dev` (or `*-dev`), cascade messages
went to the **dev** project's topic — production coordinator never sees them.
That alone explains "auto-cascade didn't fire" for prod consumers. Keep
investigating in the dev project then.

## The pipeline (what should happen)

```
1. ailang publish FOO@2.1.0
   ├─→ POST /publish to ailang-registry-validator
   │   (project: ailang-registry; logs in europe-west1)
   │
   └─→ Pub/Sub publish to ailang-cascade topic
       (project: $AILANG_CLOUD_PROJECT; topic: $PREFIX-cascade)
           │
           └─→ Push subscription delivers to:
               https://<coordinator>/pubsub/push
                   │
                   └─→ Coordinator creates task per dependent
                       Looks up agent for inbox "pkg:sunholo/<dep>"
                       Dispatches to Cloud Run Job with env:
                         AILANG_TASK_ID=task-XXX
                         AILANG_AGENT_ID=<resolved agent>
                         (etc., see internal/dispatch/cloudrun/dispatcher.go:117+)
                           │
                           └─→ Job starts, runs the bump, publishes
                               completion to ailang-completions
```

Any hop can fail silently — the publish CLI prints "Cascade-topic
notification published" before any subscriber acks, so success messages
don't mean a dependent actually bumped.

## Workflow

### Step 1 — Identify the publish you're investigating

Get the timestamp + project from the publish output (or from
`ls -lt ~/.ailang/cache/registry/<vendor>/<name>/`):

```
✓ Published sunholo/motoko_ext_abi@2.1.0
→ Cascade-topic notification published for sunholo/motoko_ext_omnigraph
  (root=sunholo/motoko_ext_abi@2.1.0, class=A)
```

The cascade message IDs printed there (e.g. `msg_20260511_211114_88b4f765`)
are the ones to grep for. Convert local time to UTC for log queries.

### Step 2 — Check the validator did receive the publish

```bash
gcloud logging read \
  'resource.type="cloud_run_revision" AND
   resource.labels.service_name="ailang-registry-validator" AND
   httpRequest.requestMethod="POST" AND
   timestamp>="2026-05-11T19:00:00Z" AND
   timestamp<="2026-05-11T19:30:00Z"' \
  --project ailang-registry --limit 20 \
  --format="value(timestamp,httpRequest.requestUrl,httpRequest.status)"
```

Expected: HTTP 200 to `/publish` at the time of your publish. If 4xx/5xx,
the publish failed before cascade messages were emitted — investigate
validator logs directly.

### Step 3 — Confirm cascade messages were emitted

The publish CLI logs each cascade message it emits. Cross-check against
what landed in Pub/Sub. The topic is `<PREFIX>-cascade`:

```bash
PROJECT="${AILANG_CLOUD_PROJECT:-ailang-multivac-dev}"
PREFIX="${AILANG_TOPIC_PREFIX:-ailang-dev}"

gcloud pubsub topics list --project "$PROJECT" \
  --filter="name~cascade" --format="value(name)"
```

If the topic doesn't exist for that project: the cascade publisher silently
no-ops (see `cmd/ailang/pkg_publish.go::newCascadePublisher`). The publish
CLI may still print "Cascade-topic notification published" because it
prints from the legacy-inbox-write path even when the cloud publisher is
nil — confusing but documented.

### Step 4 — Find the coordinator that subscribes

```bash
gcloud pubsub subscriptions list --project "$PROJECT" \
  --filter="topic~cascade" \
  --format="value(name,pushConfig.pushEndpoint)"
```

Push endpoint will be a Cloud Run URL like
`https://ailang-dev-coordinator-XXX-ew.a.run.app/pubsub/push`.

### Step 5 — Check the coordinator received and processed each cascade message

```bash
# Substitute the time window of your publish
gcloud logging read \
  'resource.type="cloud_run_revision" AND
   resource.labels.service_name="ailang-dev-coordinator" AND
   timestamp>="2026-05-11T19:10:00Z" AND
   timestamp<="2026-05-11T19:30:00Z" AND
   (textPayload=~"cascade" OR textPayload=~"task-" OR textPayload=~"failed")' \
  --project "$PROJECT" --limit 50 \
  --format="value(timestamp,textPayload)"
```

Look for log lines:

| Pattern | Meaning |
|---|---|
| `Created task task-XXX (... inbox: pkg:sunholo/...)` | coordinator received the cascade message |
| `Cloud dispatch: task task-XXX → Cloud Run Job (agent: , ...)` | **agent: empty** = no agent registered for the inbox; will fail |
| `task task-XXX → failed (error=...)` | the job ran and exited with an error |
| `task task-XXX already in terminal state "failed", skipping` | repeated retries from Pub/Sub redelivery |
| (no logs at all) | coordinator scaled to zero AND push subscription couldn't wake it (rare) |

### Step 6 — If "agent: " is empty, the agent registry isn't routing this inbox

The coordinator's agent registry is loaded from `~/.ailang/config.yaml`
(or the equivalent on the cloud instance — check via
`coordinator describe-config` if available). The cascade tasks need a
catch-all agent with `inbox: "pkg:*"` or per-package inboxes:

```yaml
coordinator:
  agents:
    - id: package-cascade-bumper
      label: "Package Cascade Bumper"
      inbox: "pkg:*"           # wildcard
      capabilities: [code, package]
```

Without this, cascade tasks get created but never dispatched with a valid
`AILANG_AGENT_ID`, and the executor's startup check
([cmd/ailang/coordinator_cloud.go:171](cmd/ailang/coordinator_cloud.go#L171))
fails them all.

### Step 7 — If the job ran and failed for another reason

Pull the failed job's logs by task ID:

```bash
TASK_ID="task-cb56ee02"
gcloud logging read \
  "resource.type=\"cloud_run_job\" AND
   labels.\"run.googleapis.com/execution_name\"=~\"$TASK_ID\"" \
  --project "$PROJECT" --limit 100 \
  --format="value(timestamp,textPayload)"
```

Or by execution name (faster):

```bash
gcloud run jobs executions list --project "$PROJECT" \
  --region europe-west1 --limit 20 \
  --format="value(metadata.name,status.completionTime,status.conditions[0].message)"
```

## Common failure modes (cheat-sheet)

| Symptom | Likely cause | Fix |
|---|---|---|
| Publish prints "Cascade-topic notification published" but no coordinator logs | `AILANG_CLOUD_PROJECT` points to a project with no `*-cascade` topic | Check env vars; the cloud publisher silently no-ops when topic absent |
| Coordinator logs `Created task ... (agent: , inbox: pkg:...)` then `failed (error=AILANG_AGENT_ID required)` | No agent registered for `pkg:*` inboxes in coordinator config | Add `package-cascade-bumper` agent entry |
| Coordinator hasn't logged ANYTHING since hours before the publish | Cloud Run revision crashed; push delivery failing silently | `gcloud run revisions list`; check latest revision health |
| Tasks created but never dispatched (`Cloud dispatch: published task ... to Pub/Sub only (no dispatcher, ...)`) | `cloudDispatcher` not registered (CLI didn't init `cloudrun.NewDispatcher`) | Restart coordinator with proper env (`AILANG_CLOUD_REGION`, etc.) |
| Job dispatched but immediately fails with `AILANG_AGENT_ID required` | Same as the empty-agent case above | Same fix |
| Job dispatched, runs for hours, never publishes completion | Job hung mid-execution; check Cloud Run Job execution logs | `gcloud run jobs executions describe` |

## One-shot triage script

```bash
.claude/skills/cloud-cascade-debug/scripts/triage_cascade.sh \
  <vendor/name>@<version> [time-window-hours]
```

Bundles steps 2-5 into a single output. Defaults to a 1-hour window
ending at NOW. Pass `--project` to override the project (default:
`$AILANG_CLOUD_PROJECT`).

## Notes

- Production project = `ailang-multivac` (no `-dev` suffix); dev project =
  `ailang-multivac-dev`. Check `AILANG_CLOUD_PROJECT` carefully — local
  laptops typically point at dev.
- The validator service lives in `ailang-registry` (separate project)
  but the cascade pipeline lives in whichever project the publisher
  targets via `AILANG_CLOUD_PROJECT`.
- All of this is europe-west1.
- The cloud publisher is best-effort — a failure to write to the cascade
  topic does NOT abort `ailang publish` (the package still publishes).
  Cascade is "fire and forget" by design.
