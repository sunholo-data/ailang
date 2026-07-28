# Release State JSON Schema

This document defines the JSON format for release state tracking, enabling multi-session release workflows.

Based on [Anthropic's long-running agent patterns](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents).

## Purpose

Release processes can take hours (eval baselines, dashboard updates). The JSON state file enables:
- Resuming interrupted releases
- Tracking which steps are complete
- Skipping completed steps on retry
- Clear progress reporting

## File Location

```
.ailang/state/release_<version>.json
```

**Examples:**
- `.ailang/state/release_v0.4.6.json`
- `.ailang/state/release_v1.0.0.json`

## Schema Definition

```json
{
  "version": "string",
  "started": "ISO 8601 timestamp",
  "last_updated": "ISO 8601 timestamp",
  "correlation_id": "string",
  "status": "in_progress | completed | failed",
  "steps": [
    {
      "id": "string",
      "name": "string",
      "status": "pending | in_progress | completed | failed | skipped",
      "started": "ISO 8601 timestamp | null",
      "completed": "ISO 8601 timestamp | null",
      "result": "string | null",
      "error": "string | null"
    }
  ],
  "eval_baseline": {
    "status": "not_started | running | completed | failed",
    "total_benchmarks": "number",
    "completed_benchmarks": "number",
    "output_dir": "string | null",
    "started": "ISO 8601 timestamp | null",
    "completed": "ISO 8601 timestamp | null"
  }
}
```

## Example States

### Initial State

```json
{
  "version": "v0.4.6",
  "started": "2025-01-27T10:00:00Z",
  "last_updated": "2025-01-27T10:00:00Z",
  "correlation_id": "release_v0.4.6",
  "status": "in_progress",
  "steps": [
    {
      "id": "pre_release_checks",
      "name": "Pre-release validation",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    },
    {
      "id": "git_tag",
      "name": "Create git tag",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    },
    {
      "id": "github_release",
      "name": "Create GitHub release",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    },
    {
      "id": "eval_baseline",
      "name": "Run eval baseline",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    },
    {
      "id": "dashboard_update",
      "name": "Update benchmark dashboard",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    }
  ],
  "eval_baseline": {
    "status": "not_started",
    "total_benchmarks": 264,
    "completed_benchmarks": 0,
    "output_dir": null,
    "started": null,
    "completed": null
  }
}
```

### In Progress (Eval Running)

```json
{
  "version": "v0.4.6",
  "started": "2025-01-27T10:00:00Z",
  "last_updated": "2025-01-27T14:30:00Z",
  "correlation_id": "release_v0.4.6",
  "status": "in_progress",
  "steps": [
    {
      "id": "pre_release_checks",
      "name": "Pre-release validation",
      "status": "completed",
      "started": "2025-01-27T10:00:00Z",
      "completed": "2025-01-27T10:15:00Z",
      "result": "All checks passed",
      "error": null
    },
    {
      "id": "git_tag",
      "name": "Create git tag",
      "status": "completed",
      "started": "2025-01-27T10:15:00Z",
      "completed": "2025-01-27T10:16:00Z",
      "result": "Tag v0.4.6 created and pushed",
      "error": null
    },
    {
      "id": "github_release",
      "name": "Create GitHub release",
      "status": "completed",
      "started": "2025-01-27T10:16:00Z",
      "completed": "2025-01-27T10:20:00Z",
      "result": "Release v0.4.6 published",
      "error": null
    },
    {
      "id": "eval_baseline",
      "name": "Run eval baseline",
      "status": "in_progress",
      "started": "2025-01-27T10:20:00Z",
      "completed": null,
      "result": "Running... 150/264 benchmarks complete",
      "error": null
    },
    {
      "id": "dashboard_update",
      "name": "Update benchmark dashboard",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    }
  ],
  "eval_baseline": {
    "status": "running",
    "total_benchmarks": 264,
    "completed_benchmarks": 150,
    "output_dir": "eval_results/baselines/v0.4.6",
    "started": "2025-01-27T10:20:00Z",
    "completed": null
  }
}
```

### Failed (Can Resume)

```json
{
  "version": "v0.4.6",
  "started": "2025-01-27T10:00:00Z",
  "last_updated": "2025-01-27T16:00:00Z",
  "correlation_id": "release_v0.4.6",
  "status": "failed",
  "steps": [
    {
      "id": "eval_baseline",
      "name": "Run eval baseline",
      "status": "failed",
      "started": "2025-01-27T10:20:00Z",
      "completed": "2025-01-27T16:00:00Z",
      "result": "Partial: 150/264 benchmarks complete",
      "error": "Session timeout - can resume with --skip-existing"
    },
    {
      "id": "dashboard_update",
      "name": "Update benchmark dashboard",
      "status": "pending",
      "started": null,
      "completed": null,
      "result": null,
      "error": null
    }
  ],
  "eval_baseline": {
    "status": "failed",
    "total_benchmarks": 264,
    "completed_benchmarks": 150,
    "output_dir": "eval_results/baselines/v0.4.6",
    "started": "2025-01-27T10:20:00Z",
    "completed": "2025-01-27T16:00:00Z"
  }
}
```

## Resuming Interrupted Releases

```bash
# Check release state
cat .ailang/state/release_v0.4.6.json | jq '.steps[] | select(.status == "pending" or .status == "failed")'

# Resume eval baseline (skips completed benchmarks)
ailang eval-suite --full --skip-existing \
  --output eval_results/baselines/v0.4.6

# Update state after manual recovery
jq '.steps[] |= if .id == "eval_baseline" then .status = "completed" | .completed = "2025-01-27T17:00:00Z" else . end' \
  .ailang/state/release_v0.4.6.json > /tmp/release_update.json
mv /tmp/release_update.json .ailang/state/release_v0.4.6.json
```

## Usage with release-manager

The release-manager skill can create and update this state file:

```bash
# Start release (creates state file)
# Use release-manager skill: "Create release v0.4.6"

# State file created at: .ailang/state/release_v0.4.6.json
# Correlation ID: release_v0.4.6

# If interrupted, resume with:
# Use release-manager skill: "Resume release v0.4.6"
```

## Integration with post-release

The post-release skill can check for existing state and resume:

```bash
# Check if release already started
if [ -f ".ailang/state/release_v0.4.6.json" ]; then
  echo "Found existing release state, resuming..."
  # Skip completed steps
  # Use --skip-existing for eval baseline
fi
```

## Benefits

1. **Resume after timeout**: Long eval baselines can be interrupted and resumed
2. **Skip completed work**: Don't repeat expensive operations
3. **Clear progress**: See exactly what's done and what's pending
4. **Audit trail**: Complete record of release process
5. **Debug failures**: Understand where release broke
