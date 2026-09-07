---
title: Durable mission iterations
description: Run one approved work item with durable stage ownership, quota admission, and validated artifacts.
---

# Durable mission iterations

`ailang mission iterate` advances one explicit, approved work item through its
remaining stages. The binary owns dispatch, resource limits, durable ownership,
artifact checks, and recovery. Successful provider execution alone does not mean
the work item is accepted. The result is a candidate for the existing landing
workflow; iteration does not merge or publish it.

The initial runtime supports macOS and Linux. Windows execution fails before
dispatch because receipt syncing and descendant cleanup are not supported there.
This opt-in path does not activate or replace a mission's live schedule by itself.

## Configure local placement

Create `~/.config/ailang/mission-runtime.toml` with exactly these fields:

```toml
version = 1
state_db = "/absolute/local/state/mission-runtime.sqlite"
workspace_root = "/absolute/local/mission-workspaces"
```

Both paths must be absolute. The workspace root must be outside the source
repository. The database filename cannot contain `?` or `#`. Missing configuration,
unknown keys, and unsupported versions fail with an error. There is no command-line
database override: iterate, status, resume, and cancel use this same binding.

`missions/*.toml` remains the mission registry; `models.yml` remains model policy.
For first admission from a foreign project, set `AILANG_MISSION_REGISTRY` to the
absolute directory containing the existing fleet registry. For example:

```bash
AILANG_MISSION_REGISTRY=/absolute/ailang/missions \
  ailang mission iterate --work-item /absolute/reviewed-work-item.json --dry-run
```

The registry entry supplies the mission's source checkout. Execution uses isolated
stage worktrees below the configured workspace root. Resume reads saved repository
and model inputs; changing today's registry does not reroute an admitted item.

## Review and run a work item

The input is strict version 1 JSON. It includes the mission/work IDs, normalized
Git origin, full base commit, brief, allowed paths, acceptance criteria, stage
instructions, required artifacts, authority references, verification commands,
and positive time/token/cost limits. `full-v1` accounts for designer, planner,
executor, and evaluator in that order. Approved earlier artifacts can be supplied
as prerequisites, allowing a work item to start at execution.

```bash
ailang mission iterate --work-item /absolute/reviewed-work-item.json --dry-run
ailang mission iterate --work-item /absolute/reviewed-work-item.json
ailang mission status docs --work-item item-1 --json
```

Dry-run checks local inputs and resolves model routes without provider health
checks, quota probes, database creation, receipt creation, or worktree creation.
It prints the resolved input snapshot. Actual execution checks shared quota policy
before constructing an executor and immediately before dispatch. Unknown or
exhausted protected quota produces a waiting state.

The runtime requires committed, scoped artifacts and the reserved untracked
`stage-result.json` protocol output. It runs frozen checks against exact candidate
commits and records acceptance evidence outside author worktrees. The evaluator
must be independent of the actual author routes and preserve the candidate tree.

## Inspect, resume, and cancel

```bash
ailang mission status docs --work-item item-1
ailang mission resume docs --work-item item-1
ailang mission cancel docs --work-item item-1 --version 7
```

Use the version from a fresh status response when cancelling. Status opens an
existing database read-only without schema migration. Its JSON includes phase,
reason, next action, current stage, version, lease/deadline, route provenance, and
accepted artifact digests. It omits saved prompts, owner tokens, and transcripts.

Resume preserves frozen input, routes, accumulated limits, and accepted stages.
It does not steal a live lease or blindly repeat a dispatched stage. Repeating a
completed item returns its existing evidence without another provider call.
Changed work-item input requires a reviewed successor item.

Cancellation fences parent and child state before termination. A running owner
observes the lost fence and stops its owned process group. If termination cannot
be confirmed, status remains `needs_reconciliation` and retains mission admission.
Do not interpret the cancel command's return as proof an external process died.
There is no automatic rerun or automatic acceptance for an ambiguous outcome.

| Exit | Meaning |
| --- | --- |
| 0 | Validated completion or successful read-only command |
| 2 | Invalid input or local configuration |
| 3 | Waiting, including quota, decision, or active ownership |
| 4 | Reconciliation required |
| 5 | Execution or verification failure |
| 130 | Cancelled |

## Hermetic example

The command-level fixture creates a disposable Git repository, injects model and
quota policy, and counts attempted executor construction. It exercises dry-run,
quota refusal, and frozen resume without credentials or inference:

```bash
go test ./cmd/ailang -run TestMissionIterationDryRunQuotaAndFrozenResume -count=1 -timeout=120s
```

Runtime artifact/crash fixtures live in `internal/mission/iteration`. A live Docs
canary requires the separate reviewed activation packet; passing these fixtures
does not establish live adoption or productivity.
