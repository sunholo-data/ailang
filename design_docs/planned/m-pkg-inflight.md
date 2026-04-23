# M-PKG-INFLIGHT: Package-in-Flight Tracking for Registry Workflows

**Status**: Planned
**Target**: v1.0.0
**Priority**: P1 — required for laptop visibility into agent-driven package updates
**Estimated**: 2-3 days
**Source**: Need to know when agents are publishing/updating ailang packages

---

## Problem

When a sprint-executor agent finishes implementing a feature, it publishes an updated ailang package to the registry. Currently:

1. There is no concept of a package being "in flight" — the registry only knows about successfully published versions
2. The laptop daemon (M-MAC-NOTIFY-DAEMON) has no package-specific event to subscribe to — it only sees generic task events
3. There is no link between a `task_id` and the package version it produces — you can't answer "which task published `sunholo/registry_validator@0.3.1`?"
4. Failed publishes vanish silently — the only record is the coordinator task failure log

This creates a blind spot: package work in progress is invisible until it lands in the registry index.

---

## Solution: In-Flight State in Firestore + Package Events on Pub/Sub

### Firestore: `package_builds` collection

A new Firestore collection tracks package publish attempts from task start to registry landing:

```
package_builds/{build_id}
  build_id:       string      // "{task_id}-{vendor}-{name}" 
  task_id:        string      // coordinator task that triggered this
  agent_id:       string      // which agent published
  vendor:         string      // package vendor
  name:           string      // package name
  version:        string      // version being published (from manifest)
  status:         string      // "building" | "validating" | "published" | "failed"
  started_at:     timestamp
  completed_at:   timestamp   // null if in progress
  error:          string      // null if success
  artifact_path:  string      // GCS path to build artifacts (from M-ARTIFACT-STORAGE)
  registry_url:   string      // final GCS URL if published
  workspace:      string      // coordinator workspace
```

### Pub/Sub: package events on `TopicEvents`

Extend `TaskStreamEvent` with a new `stream_type: "package"` and add a `PackageEvent` wrapper:

```go
type PackageEvent struct {
    BuildID     string `json:"build_id"`
    Vendor      string `json:"vendor"`
    Name        string `json:"name"`
    Version     string `json:"version"`
    Status      string `json:"status"`       // "building"|"validating"|"published"|"failed"
    RegistryURL string `json:"registry_url,omitempty"`
    Error       string `json:"error,omitempty"`
}
```

Published to `TopicEvents` with attribute `stream_type=package` so the laptop daemon can filter.

---

## Implementation

### 1. Registry validator fires events (`cmd/registry-validator/main.go`)

When a publish request arrives, the validator:
1. Creates a `package_builds` Firestore doc with `status: "validating"`
2. Publishes a `PackageEvent{status: "validating"}` to Pub/Sub
3. On success: updates Firestore to `published`, publishes `PackageEvent{status: "published", registry_url: ...}`
4. On failure: updates Firestore to `failed`, publishes `PackageEvent{status: "failed", error: ...}`

The `task_id` and `agent_id` come from request headers set by the executor (`X-Ailang-Task-ID`, `X-Ailang-Agent-ID`).

### 2. Executor sends task headers when publishing (`internal/executor/claude/claude.go`)

The executor already sets env vars before running Claude. Add:
```bash
AILANG_TASK_ID=task-inbox_17
AILANG_AGENT_ID=sprint-executor
```

The `ailang` CLI's `registry publish` command reads these and forwards as headers to the validator. This creates the provenance link without requiring changes to agent prompts.

### 3. Laptop daemon subscribes to package events (`cmd/ailang/daemon.go`)

In `M-MAC-NOTIFY-DAEMON`, the daemon already subscribes to `SubEventsLaptop`. Add a handler for `stream_type=package`:

```go
case "package":
    var pkg PackageEvent
    json.Unmarshal(msg.Data, &pkg)
    switch pkg.Status {
    case "published":
        notify.Fire(Notification{
            Title: "📦 Package published",
            Body:  fmt.Sprintf("%s/%s v%s", pkg.Vendor, pkg.Name, pkg.Version),
            URL:   pkg.RegistryURL,
            Sound: "Ping",
        })
    case "failed":
        notify.Fire(Notification{
            Title: "❌ Package failed",
            Body:  fmt.Sprintf("%s/%s v%s — %s", pkg.Vendor, pkg.Name, pkg.Version, pkg.Error),
            Sound: "Basso",
        })
    }
```

### 4. `ailang packages status` CLI command

```
ailang packages status [--workspace WORKSPACE] [--limit 10]

Recent package builds:
  sunholo/registry_validator  v0.3.1  published  2026-04-23T09:20  task-inbox_17
  sunholo/stdlib              v0.2.0  failed     2026-04-23T08:51  task-inbox_16  — compile error: ...
  sunholo/docparse            v0.1.0  building   2026-04-23T09:35  task-inbox_18  (in progress)
```

Reads from `package_builds` Firestore collection. Useful for debugging and reviewing what agents have published.

### 5. Dashboard: packages tab

The dashboard surfaces `package_builds` as a "Packages" tab alongside the existing Tasks view. Each row links to the artifact bucket path (session.jsonl, metrics.json) via the `artifact_path` field populated by M-ARTIFACT-STORAGE.

---

## Provenance chain (linking task → package version)

This design answers: "which task produced package X?" and "which package did task Y produce?"

```
Firestore: tasks/{task_id}
  └── artifact_gcs_path: "tasks/task-inbox_17"

Firestore: package_builds/task-inbox_17-sunholo-registry_validator
  ├── task_id: "task-inbox_17"
  ├── artifact_path: "tasks/task-inbox_17"   (same GCS prefix — session.jsonl, metrics.json)
  └── registry_url: "gs://ailang-registry/packages/sunholo/registry_validator/0.3.1/"
```

---

## Dogfooding: first test with `sunholo/test_package`

To validate the end-to-end flow, use the `sunholo/test_package` package (simple, fast to publish):

1. Send a task to `sprint-executor`: "bump sunholo/test_package to v0.0.2, add a `greet` function"
2. Watch `ailang daemon run --dry-run` — should see `PackageEvent{status: "validating"}` then `published`
3. Verify `package_builds` Firestore doc exists with task_id link
4. Verify Mac notification fires: "📦 Package published — sunholo/test_package v0.0.2"
5. `gsutil ls gs://ailang-registry/packages/sunholo/test_package/0.0.2/` confirms it landed

This proves: task → build → validate → publish → notify → GCS all work end-to-end.

---

## Non-goals

- Package dependency graph visualisation
- Multi-agent concurrent publish coordination (phase 2)
- Rollback / unpublish (registry is intentionally immutable)
