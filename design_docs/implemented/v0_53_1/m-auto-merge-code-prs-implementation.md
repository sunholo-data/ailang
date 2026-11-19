# M-AUTO-MERGE-CODE-PRS implementation

Refs #1599

Repository implementation is complete; independent sprint evaluation and live
Daneel deployment evidence are pending. No new GitHub issue was opened.
Execution branch: `coordinator/task-5fd61a94` (the execution checkout; the planner
handoff named `coordinator/task-53d1dbd7`).

## Delivered

- M1: four optional agent fields, trusted daemon-to-job plumbing, newline-separated
  checks, registry/dispatch/job validation, YAML/JSON and default regression tests.
  Only the approver secret name enters the dispatch spec.
- M2: one production scope guard with docs/code modes, real git fixtures including
  spaced image filenames, base SHA resolution, paginated check-run enumeration,
  missing/malformed/API-error refusals. GitHub remains the native SQUASH merge actor.
- M3: job-side secret fetch, expected/actual/author identity preflight, case-insensitive
  comparison, final-head-pinned review after enable, confirmed-review validation,
  disable-on-review-failure cleanup and explicit cleanup diagnostics. PR audit records
  configured intent separately from success and preserves existing PR bodies on retry.
  Label failures warn; body/log audit persists. Native enable also requires a confirmed
  GraphQL mutation result, rather than accepting an empty response.
- M4: registry-loader verification of both guide examples, generated environment
  reference, changelog and Daneel permissions/protection/refusal/staging runbook.

The only change to shared diff enumeration is NUL-separated git output, preserving
filenames containing spaces. New wrapper logic lives in the planned sibling file;
configuration validation and env forwarding also use small cohesive sibling files.
All touched Go files remain under 800 lines.

## Validation

Commands use Go and locally installed make/jq/lint tools in PATH. Tool installation
was outside the repository. The sprint JSON validator passed before implementation.

| Check | Result |
|---|---|
| `go test ./internal/config ./internal/dispatch/cloudrun` | Pass |
| `go test ./internal/coordinator -run 'AgentConfigAutoMergeCode\|AutoMerge'` | Pass |
| `go test ./internal/coordinator -run AgentConfigAutoMerge` | Pass, including guide examples through real registry loader |
| `go test ./cmd/ailang -run 'AutoMerge\|Approv\|RequiredCheck\|AgentConfig'` | Pass |
| Broad plan filter on coordinator and wrapper | Wrapper passes; existing SQLite coordinator tests fail with CGO unavailable |
| `make test-core` | Non-SQLite core packages pass; existing effects brain-store tests fail with CGO unavailable |
| `make check-boundaries check-file-sizes` | Pass |
| `make check-git-exec` | Pass |
| `make docs-env` | Pass; generated reference updated |
| `make lint` | First cold run timed out at five minutes with zero issues; warm rerun passes with zero issues |
| `git diff --check` | Pass |

CGO limitation, verbatim: `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3
requires cgo to work. This is a stub`. This executor has no C compiler; the approved
plan explicitly allows these environment failures to be recorded separately.
No full `make test` was run, per the sprint plan; CI remains required.

Focused statement coverage: registry code config validation 100%, wrapper config
resolution/validation and audit formatting 100%, check enumeration 91.9%, identity
preflight 92.9%, approving review 100%, enable/approve sequence 92.5%. This is helper
coverage, not a repository-wide coverage claim. API tests cover rollback, cleanup
failure, missing evidence, secret failure, wrong identity, existing-PR retries,
label failure and request order. New tests use native paths only for filesystem
operations; API/path assertions use git's slash paths. No external SMT binary or
platform-specific golden was introduced.

## Deployment and evaluation handoff

Pending: Mark confirms required checks are enforced by the target ruleset and
last-push approval is required; provision repo-limited second-user PAT and secret
access; run in-scope, out-of-scope and failing-check staging fixtures; record one
staging merge followed by ten clean production merges. No live registry, ruleset,
secret, IAM or site workflow changes are in this sprint.

The code-writing process shares the job service account that reads the approver
secret. The second GitHub identity is not a separately isolated execution principal;
the target ruleset is the merge boundary. See the coordinator guide's runbook.

Coordinator completion markers hand this branch and the sprint JSON to the
independent sprint-evaluator. The executor does not self-approve its work or move
the design into implemented before that evaluation.

## PR body

Refs #1599

Enable per-agent code PR auto-merge through trusted config, declared scope,
base check-name evidence, second-user approval and durable PR audit. Keep docs
mode as the default. On approval failure, attempt to disable native auto-merge
and report cleanup failure explicitly.

Validation: focused wrapper/config/dispatch/registry and guide-loader tests pass;
architecture, file-size and git-exec gates pass. Existing SQLite-dependent checks
are limited by this executor's CGO-unavailable build. Live Daneel staging and
production activation remain pending; repository protections must enforce the
configured check names.


## Retrospective

Four dependent milestones were implemented sequentially. The environment needed
PATH correction and temporary make/jq/lint installation; cold dependency loading
exceeded lint's five-minute timeout, and a warm rerun passed. Focused tests and
coverage avoided the unavailable SQLite runtime; broad plan checks surfaced that
limitation explicitly. No new production polling loop, store or dependency was
introduced. The remaining evidence belongs to CI, independent evaluation and the
human-confirmed deployment checklist, not to repository implementation.
