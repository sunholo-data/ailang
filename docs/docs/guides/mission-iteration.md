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

### A complete example

Every field below is present so the JSON decodes, but every revision, SHA-256,
repository and path is a fabricated, syntactically valid placeholder — **this
example is illustrative only**. It is not an approval record, and iterate must
never be pointed at it. It mirrors the structure of the strict decoder's own
fixture (`cmd/ailang/testdata/mission-iteration/work-item.json`) with two
supplied prerequisites (`designer`, `planner`) so the work item starts at
`executor`, matching `full-v1`'s four ordered roles.

```json
{
  "version": 1,
  "mission_id": "docs",
  "work_item_id": "item-1",
  "repository": "github.com/example-org/example-docs",
  "base_revision": "aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111",
  "brief": "Illustrative only: add one example page under docs/ following the design and plan bound by the authority reference below.",
  "allowed_paths": ["docs/"],
  "workflow": "full-v1",
  "stages": [
    {
      "id": "executor",
      "role": "executor",
      "instructions": "Follow the frozen brief. Change only paths under docs/. Write the untracked stage-result.json protocol file before finishing.",
      "required_artifacts": ["docs/example-guide.md"],
      "authority_refs": [
        {
          "revision": "dddd4444dddd4444dddd4444dddd4444dddd4444",
          "path": "docs/approval.md",
          "locator": "example-operator-approved-20260101",
          "sha256": "3333333333333333333333333333333333333333333333333333333333333333",
          "artifact_digest": "2222222222222222222222222222222222222222222222222222222222222222"
        }
      ],
      "limits": { "timeout_seconds": 1800, "max_tokens": 70000, "max_cost_usd": 3 }
    },
    {
      "id": "evaluator",
      "role": "evaluator",
      "instructions": "Check the exact candidate commit against every frozen acceptance criterion. Preserve candidate HEAD. Write the untracked stage-result.json protocol file.",
      "required_artifacts": ["docs/example-guide.md"],
      "authority_refs": [
        {
          "revision": "dddd4444dddd4444dddd4444dddd4444dddd4444",
          "path": "docs/approval.md",
          "locator": "example-operator-approved-20260101",
          "sha256": "3333333333333333333333333333333333333333333333333333333333333333",
          "artifact_digest": "2222222222222222222222222222222222222222222222222222222222222222"
        }
      ],
      "limits": { "timeout_seconds": 1200, "max_tokens": 30000, "max_cost_usd": 2 }
    }
  ],
  "prerequisites": [
    {
      "role": "designer",
      "artifact": {
        "commit": "bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222",
        "path": "docs/designer.md",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111"
      },
      "authority_refs": [
        {
          "revision": "dddd4444dddd4444dddd4444dddd4444dddd4444",
          "path": "docs/approval.md",
          "locator": "example-operator-approved-20260101",
          "sha256": "3333333333333333333333333333333333333333333333333333333333333333",
          "artifact_digest": "1111111111111111111111111111111111111111111111111111111111111111"
        }
      ],
      "author_models": ["claude-sonnet-5"]
    },
    {
      "role": "planner",
      "artifact": {
        "commit": "cccc3333cccc3333cccc3333cccc3333cccc3333",
        "path": "docs/planner.md",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222"
      },
      "authority_refs": [
        {
          "revision": "dddd4444dddd4444dddd4444dddd4444dddd4444",
          "path": "docs/approval.md",
          "locator": "example-operator-approved-20260101",
          "sha256": "3333333333333333333333333333333333333333333333333333333333333333",
          "artifact_digest": "2222222222222222222222222222222222222222222222222222222222222222"
        }
      ],
      "author_models": ["claude-sonnet-5"]
    }
  ],
  "verification": [
    {
      "id": "whitespace",
      "argv": ["git", "diff", "--check", "aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111"],
      "cwd": ".",
      "timeout_seconds": 30
    }
  ],
  "limits": { "timeout_seconds": 3600, "max_tokens": 100000, "max_cost_usd": 5 },
  "acceptance_criteria": [
    { "id": "example-complete", "text": "The illustrative artifact demonstrates every required WorkItem v1 field." }
  ]
}
```

### Replace every illustrative value with a real committed input

- **`mission_id`/`work_item_id`/`repository`** — the real registry mission ID and
  the exact normalized Git origin (`ValidateRepository` compares `git remote
  get-url origin`, lower-cased host, `.git`/trailing-slash stripped) of the
  checkout iterate will actually run in. A mismatch fails before dispatch.
- **`base_revision`, prerequisite `artifact.commit`, and every `authority_refs[].revision`**
  — full commit IDs from the real history, via `git rev-parse <ref>`. `iterate`
  requires the prerequisite and authority revisions to be ancestors of
  `base_revision` (`merge-base --is-ancestor`); an unrelated or future commit fails.
- **every `sha256` and `artifact_digest`** — the SHA-256 of the committed file's
  raw bytes, via `git show <rev>:<path> | shasum -a 256`. This is **not** a Git
  object ID: `git rev-parse <rev>:<path>` (or `git hash-object`) hashes a
  `blob <len>\0`-prefixed object under SHA-1 (or SHA-256 for a sha256-format
  repo), a different value from `sha256(content bytes)` computed here. Use
  `git show`, not `git rev-parse` or `git ls-tree`'s blob column, to get the
  field this schema wants.
- **`author_models`** — the exact registry model identity that actually
  produced each prerequisite, not an aspiration. `iterate` binds a stage's
  accepted result to the one dispatch attempt that completed
  (`completedRoute`), and rejects a route whose model was never in the
  requested list — you cannot backfill this field with a guess after the fact.
- **every `path`** (`artifact.path`, `authority_refs[].path`, `required_artifacts`)
  — the real repository-relative path of that role's actual committed
  deliverable or approval document, e.g. the real design doc instead of
  `docs/designer.md`. Every `required_artifacts` entry must already fall under
  `allowed_paths`.

### How authority binds, and what it never does

An `authority_refs[]` entry has two hashes that mean different things:
`sha256` is the content hash of the **approval document itself** at
`revision`+`path`; `artifact_digest` is the content hash of the **artifact
being authorized** — for a prerequisite, the decoder itself requires
`artifact_digest` to equal that prerequisite's own `artifact.sha256`
(`Spec.Validate`), before any Git call runs.

Stages have no `artifact` field, so the decoder does not force a stage's
`artifact_digest` to match anything; above, both stages reuse the planner's
digest because executing the approved plan is what their authority actually
covers — bind each stage reference to whichever already-approved digest your
project's authority record genuinely authorizes, not to this example's choice.

At verification time `VerifyPrerequisites`/`verifyAuthorities` re-derive both
hashes from real Git blobs, and additionally require the approval document to
be **byte-identical between the cited `revision` and the work item's own
`base_revision`** (a stale or edited approval fails), and require the
`locator` string to appear **literally inside that document's committed
text** — the locator is not a label you invent afterward; it must already be
words in a real commit. None of this manufactures approval: an unreachable
revision, a changed hash, or an absent locator fails verification instead of
producing one. Recording a locator that matches nothing committed is not
authority; it is a validation failure waiting to happen.

### Evaluator independence

Independence is judged by the actual completed route's vendor and model, the
same provenance `author_models` is checked against — not by which client
library or endpoint carried the request. Routing the same underlying model
through a second transport does not make the evaluator independent of an
author who used that vendor; choose an evaluator with a genuinely different
model/vendor pairing from every author role it reviews.

### What you will actually see

Invalid local input or configuration exits `2` immediately, before any quota
or dispatch check. An unknown or exhausted protected quota leaves the item
`waiting` (exit `3`) for a later retry rather than failing it. An
unconfirmed cancellation or an interrupted stage leaves it
`needs_reconciliation` (exit `4`) with mission admission retained, never an
automatic rerun. Status reports which of these applies:

```text
# illustrative status excerpt, not a command or real output
{"phase": "waiting", "reason": "quota_exhausted", "next_action": "retry_after_quota"}
{"phase": "needs_reconciliation", "reason": "owner_fence_lost", "next_action": "operator_review"}
```

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
