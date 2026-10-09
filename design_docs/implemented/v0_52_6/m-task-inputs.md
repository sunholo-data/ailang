# M-TASK-INPUTS: Typed Task Inputs — Fetch a Named `{repo, ref, path}` Into the Workspace Before the Agent Runs

Refs #1600

**Status**: Implemented — independent evaluation passed 95/100 on 2026-10-09. Cloud deployment and the Daneel migration remain pending.
**Target**: v0.52.6
**Priority**: P2 (matches `priority:P2` on the issue)
**Estimated**: 5–7 engineering days (message plane ~2, registry/dispatch enforcement ~1.5, job fetch ~2, site-agent migration + docs ~1, regression/edge tests throughout)
**Dependencies**: None hard. Complements planned [M-EXECUTOR-ENV-HARDENING](../../planned/v0_49_1/m-executor-env-hardening.md) (this feature is one of the "narrowings" its D2 anticipates) and implemented [M-AGENT-AILANG-ONLY-EXECUTION](../v0_39_0/m-agent-ailang-only-execution.md) (the `ailang_only` lane the site agent will move to).
**Routing**: AILANG fix under [PROGRAM](../../PROGRAM.md) — coordinator/harness Go only; no motoko core change, no language change.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The fetched content is pinned by an explicit `ref` (and optionally `sha256`); the resolved commit is recorded so a run can state what it saw. The ref may be a moving branch — that nondeterminism is the sender's explicit choice, surfaced in the completion report, not ambient. |
| A2: Replayability | +1 | Fetched provenance (repo@ref → resolved commit, dest, verified checksums) is logged and published on the completion report, so a banked run states which bytes it was handed. |
| A3: Effect Legibility | +1 | File delivery into an agent's workspace stops being a shell side effect (`git fetch` with a fleet token) and becomes a typed, inspectable field on the task, validated and granted before dispatch. |
| A4: Explicit Authority | +1 | Repo reach is a typed grant (`inputs_allow`) on the trusted agent registry entry — the same authority boundary as `work_tier` (M-COORDINATOR-EXECUTION-TRUST V18: trusted registry, never message content). A message cannot point an agent at an arbitrary repo. |
| A5: Bounded Verification | +1 | Allowlist check, dest validation, symlink/overwrite refusal, and sha256 verification are pure functions unit-testable with temp git repos — no live model, no Cloud Run. |
| A6: Safe Concurrency | 0 | Fetches run sequentially in the job parent before the executor starts; no concurrency surface changes. |
| A7: Machines First | +1 | Refusals are loud, structured, and name the offending repo/field; a sender (Daneel) can machine-act on the failure reason without scraping logs. |
| A8: Minimal Syntax | +1 | No language syntax. Typed Go structs + one YAML list + one JSON message field. |
| A9: Cost Visibility | 0 | No cost-system change. (A task that would previously need a shell-capable lane now runs cheaper, but that is not measured here.) |
| A10: Composability | +1 | Reuses existing seams end-to-end: the plugin-clone pattern, `excludeFromGit`, `ErrDispatchPermanent` → `MarkTaskFailed` + `postTaskResult`, the DispatchParams → env → job-config plumbing, and the `ailang_only` tool profile. |
| A11: Structured Failure | +1 | An input naming a repo outside `inputs_allow` fails the task permanently pre-dispatch with the repo named on the thread; a checksum mismatch fails the task before the agent sees the bytes. No silent skips. |
| A12: System Boundary | +1 | The fetch runs in the job parent with the job's existing credential; the agent child never receives the credential (the boundary M-EXECUTOR-ENV-HARDENING E5/D2 draws — the fleet token stays in the parent). |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — content is pinned by explicit ref, and the resolved commit is recorded
- [x] A3 (Effects): no hidden side effects — delivery is a declared, validated task field, and the default destination is excluded from the commit
- [x] A4 (Authority): no ambient access granted — repo reach requires a per-agent registry grant; the credential is used once, parent-side, for allowlisted repos only
- [x] A7 (Machines First): not optimizing for human convenience; the failure reasons are machine-parseable

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

Agents need files from repos other than their workspace repo at task start. Today the only delivery paths are (a) paste the content into the message body, or (b) give the agent a shell so it can fetch the second repo itself.

**Current State** (all verified on `origin/dev`, triage 2026-10-08 @ `658ff76a3`, re-verified on this branch @ `1dfd5615` — see Verification Log):

- `executeCloudTask` clones exactly one repo at one branch: `baseBranch`, overridden only by `AILANG_PUSH_BRANCH` (`cmd/ailang/coordinator_cloud.go`, `executeCloudTask`; `config.PushBranch()` in `internal/config/job.go:112`). Nothing per task. No `inputs` handling exists anywhere in `coordinator_cloud.go` (grep: 0 matches).
- A message carries text only: `InboxMessage` has `Payload` plus routing/GitHub/embedding fields — no file or repo-reference field (`internal/messaging/inbox.go:17-42`). Daneel caps message blocks at 16000 bytes (issue #1600 claim, sender-side), so attachments cannot ride the body.
- An `ailang_only` agent has no shell: the profile expands to `AilangRead, AilangEdit, AilangWrite, AilangCheck, AilangRun, BuiltinsSearch, ExamplesSearch, AilangCLI` — explicitly no `Bash` (`internal/executor/toolpolicy.go:51-56`). Such an agent cannot fetch a second repo even if it knows the URL.
- So Daneel's site agent (M-DANEEL-SITE-PUBLISH, sunholo-data/daneel#335) gets its poster/photo/PDF attachments into the site repo today only via the pi lane with a shell — and that shell carries the fleet token, whose reach spans every sunholo-data repo. The daneel design records that as a D11 (narrowest-grant) violation accepted for v1 only, with this feature as the narrowing.
- Side effect: the writer and design agents read host-pushed sources from `daneel-memory` `dev` only because that repo happens to be their checkout — an ambient dependency, not a declared input.

**Impact:**

- **Security**: a fleet-token-carrying shell in the site agent is the standing exception to the narrowest-grant doctrine (M-EXECUTOR-ENV-HARDENING E5 documents how reachable that token is from child processes).
- **Capability**: `tool_policy: ailang_only` is unusable for any agent that needs external files; the site agent cannot be moved to the contained lane until inputs exist.
- **Determinism**: agents depending on "whatever is on the `dev` branch right now" have no recorded input provenance.

## Goals

**Primary Goal:** Let a task message name typed inputs — `{repo, ref, path, dest}` — that the job fetches into the workspace before the agent runs, under a per-agent registry allowlist, so Daneel's site agent can drop its shell and run `tool_policy: ailang_only`.

**Success Metrics:**

- A cloud task with `inputs: [{repo: sunholo-data/daneel-memory, ref: incoming/DNL-abc123, path: incoming/DNL-abc123}]` places those files in the workspace with no shell in the agent lane, and the agent's completion report records the fetched repo@ref and resolved commit.
- A message naming a repo outside `inputs_allow` is refused **before dispatch**; the task fails permanently with the offending repo named (not requeued, not silently dropped).
- Zero behavior change when no inputs are present: existing dispatch, env, clone, and completion flows are byte-identical (regression test).
- A sha256 mismatch (per-input `sha256`, or a `manifest.sha256` present in the copied root) fails the task before the agent sees the bytes.
- The site agent's registry entry can be flipped to `tool_policy: ailang_only` + `inputs_allow: [sunholo-data/daneel-memory]` and complete its publish flow (Daneel-side verification, daneel#335 follow-up).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: `inputs` is a typed field end-to-end (message → task record → dispatch env → job), not JSON embedded in the payload text | Embedded-payload JSON is the existing precedent for `siteSlug`/`briefID` (M-HARNESS-COMMIT-CONTRACT) but cannot be allowlisted, validated, or surfaced in tooling; the issue explicitly asks for typed | human | design | high |
| D2: The repo allowlist (`inputs_allow`) lives on the **trusted agent registry entry** and is enforced at dispatch; message content can never widen it | Same authority boundary as `work_tier` (M-COORDINATOR-EXECUTION-TRUST V18/V25: trusted registry, never message content, never sender-chosen inbox). If this moved to message content, a mailed message could point the job's credential at any repo | human | design | high |
| D3: The fetch runs in the **job parent** (`executeCloudTask`), after clone/branch creation and before the executor, using the job's existing credential; the child never sees the credential | Keeps the parent/child credential boundary of M-EXECUTOR-ENV-HARDENING; doing it in the child would reintroduce the shell/token this feature exists to remove | human | design | med |
| D4: Default destination is `.incoming/<n>/`, excluded from git via `.git/info/exclude`; an explicit `dest` places files in the tree (new-files-only, harness-instruction paths denied) | Excluded-by-default keeps read-only use cases (design/writer agents) from polluting diffs; explicit dest is the only way an `ailang_only` agent gets **binary** files into the site tree (AilangWrite is a text tool) — but an unrestricted dest would let a message overwrite `AGENTS.md`-style instruction files, i.e. inject the agent's own context | human | design | med |
| D5: v1 accepts **branch/tag refs only** (clone `--branch` semantics); raw commit SHAs are refused loudly; content pinning is via optional `sha256` (per input) or `manifest.sha256` in the copied root | SHA fetches need `uploadpack.allowReachableSHA1InWant` and complicate the shallow-clone path; a sha256 pin is stronger than a git SHA for the "these exact bytes" guarantee the site use case wants | agent | design | low |
| D6: An allowlist violation is a **permanent** dispatch error (`ErrDispatchPermanent` → `MarkTaskFailed` + `postTaskResult`), not a retryable one | Retrying cannot fix an unauthorized repo; the existing permanent-error path already fails the task and posts the reason to its thread (`daemon_tasks_exec.go:376-386`) | agent | design | low |
| D7: Cloud lane only in v1; the local (`execution_lane: local`) worktree lane keeps its current behavior | The motivating agent (Daneel site agent) is cloud; the local lane has a different fetch point (bare-metal checkout) and no Cloud Run env plumbing | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — typed field end-to-end (this doc; approved via #1600 review)
- [x] D2 — allowlist on the trusted registry, enforced at dispatch
- [x] D3 — fetch in the job parent, before the executor
- [x] D4 — default dest excluded from git; explicit dest new-files-only with an instruction-path denylist
- [x] D7 confirmation is design-level: local-lane support is intentionally deferred (a sprint may not expand scope into it without a new decision) — **Ruled 2026-10-08 by Mark: confirmed: cloud lane only in v1.**

## Solution Design

### Overview

Add an optional typed `inputs` list to the task message. The coordinator validates it at ingestion, stores it on the task record, and — at dispatch — checks every input's `repo` against the target agent's `inputs_allow` registry grant. Allowed inputs travel to the Cloud Run Job as one JSON env var. The job's parent process (which already holds the git credential) shallow-clones each allowlisted repo at the named ref into a temp dir, verifies checksums, copies the named path into the workspace (excluded from git by default), and only then starts the agent. Refusals at any stage are loud, structured, and permanent — the agent never runs on a task whose inputs could not be delivered in full.

### Architecture

**The typed input** (new `internal/coordinator/task_inputs.go`):

```go
// TaskInput names content in another repository that the job fetches into
// the workspace before the agent runs. Refs #1600.
type TaskInput struct {
    Repo  string `json:"repo"`            // exact "owner/repo" — must be in the agent's inputs_allow
    Ref   string `json:"ref"`             // branch or tag name (v1: no raw SHAs — D5)
    Path  string `json:"path"`            // path within the repo at that ref; "" = whole tree
    Dest  string `json:"dest,omitempty"`  // workspace-relative destination; "" = .incoming/<n>/ (excluded)
    SHA256 string `json:"sha256,omitempty"` // pin for single-file inputs; verified post-fetch
}
```

Validation (`Validate()`, pure): `repo` must match `^[\w.-]+/[\w.-]+$` (exact-match allowlist key, no wildcards); `ref` must be a plausible branch/tag name (no leading `-`, no whitespace, no `..`); `path`/`dest` must be clean relative paths with no `..` component; at most 16 inputs per task; `sha256` (when set) requires `path` to name a single file.

**Components and data flow:**

1. **Ingestion** (typed field enters the plane):
   - `POST /api/messages` gains `inputs: []TaskInput` on `postMessageRequest` (`internal/coordinator/daemon_http.go`); invalid inputs → 400 naming the field and the reason.
   - `ailang messages send` gains `--inputs-file <path>` (a JSON file of `[]TaskInput`), validated at parse (`cmd/ailang/messages_send.go`). A file rather than an inline flag: the list is structured JSON and senders (Daneel) generate it.
   - `messaging.InboxMessage` gains `Inputs []TaskInput` (`json:"inputs,omitempty"`), persisted as a JSON text column:
     - SQLite: new `inputs TEXT` column via the next migration (keyed on the chain's current terminal state `"1.9.0"`, recording `"1.10.0"`; `ALTER TABLE inbox_messages ADD COLUMN inputs TEXT DEFAULT '[]'` — same shape as the envelope-column migration v1.7.0→v1.8.0), plus the column in `schema.go`'s `CREATE TABLE` and in **all three** hand-written SQL lists in `internal/messaging/inbox.go` (insert ~line 242, list ~line 294, get ~line 428 — an omitted list is a silently dropped field, the exact trap the `FinalizationLedger` comment in `store.go` documents).
     - Firestore: `inboxToMap`/`mapToInbox` in `internal/storage/firestore/messaging_convert.go` (hand-written map — same trap).
   - All three `InboxMessage → coordinator.Message` converters copy `Inputs`: the Pub/Sub hydration in `pubsub_adapter.go` (`HandleNotification`, after `fullMsg` hydration), `message_adapter.go` `ListUnread`, and — for the watcher path — `watcher.go`.

2. **Task record**: `TaskRecord.Inputs []TaskInput` (`internal/coordinator/store.go`, next to `Workspace`/`BaseBranch`), set at **both** TaskRecord-creation sites in `daemon_tasks_polling.go` (~line 207 and ~line 511 — the two sites that already mirror `GithubIssue`). Persisted in:
   - `internal/coordinator/store_sqlite_queries.go` (insert + select column lists) and `store_sqlite_schema.go`
   - `internal/storage/firestore/coordinator_convert.go` `taskToMap`/`mapToTask` — the hand-written Firestore mapping where a missing field is silently dropped on read (documented in `store.go`).

3. **Registry grant**: `AgentConfig.InputsAllow []string` (`yaml:"inputs_allow"`, `internal/coordinator/agent_registry.go`), exact `owner/repo` strings. Shown by `coordinator agents <id>` beside `tool_policy`. An agent with no `inputs_allow` accepts **no** inputs (deny-by-default — the zero value is the loud direction, same as `AcknowledgeOnly`).

4. **Dispatch enforcement** (`internal/coordinator/daemon_tasks_exec.go`, where the agent registry and the task are both visible — the single point where D2 can hold):
   ```go
   if len(task.Inputs) > 0 {
       if err := ValidateInputsForAgent(task.Inputs, agent.InputsAllow); err != nil {
           // wraps coordinator.ErrDispatchPermanent
       }
   }
   ```
   A violation flows through the existing permanent-error path: `handleDispatchError` → `MarkTaskFailed` + `postTaskResult`, so the sender sees `"inputs: repo %q is not in inputs_allow for agent %q"` on the thread (A7/A11).
   Allowed inputs are serialized to `DispatchParams.Inputs` (JSON) → Cloud Run env **`AILANG_TASK_INPUTS`** (`internal/dispatch/cloudrun/dispatcher.go`, alongside `AILANG_SITE_SLUG`; reader `config.TaskInputs()` in `internal/config/job.go`, plus a row in that file's env-var documentation table). The list is metadata only (≤16 entries of a few fields each), so the env var stays tiny.

5. **Job fetch** (new `cmd/ailang/coordinator_cloud_inputs.go`, called from `executeCloudTask` as a "Step 2.5" — after clone and branch creation, before the executor):
   - For each input, in order: `git clone --depth 1 --branch <ref> https://github.com/<repo> <tmpdir>` (the same shallow-clone shape the plugin-repo step already uses), resolve `HEAD` of the clone to a commit for provenance.
   - Verify: per-input `sha256` (single-file inputs), and if the copied root contains `manifest.sha256`, verify **every** entry (`sha256sum` output format). A present-but-invalid manifest is a hard failure — never a silent skip.
   - Copy `path` → `workDir/<dest>`:
     - `dest` empty → `.incoming/<n>/`, then `excludeFromGit(workDir, ".incoming/")` (the helper `coordinator_cloud_github.go:432` already exists and is worktree-aware).
     - `dest` set → refuse if any target path already exists in the clone (inputs create files, never overwrite); refuse destinations resolving inside `.git/` or any harness-instruction path (`AGENTS.md`, `CLAUDE.md`, `.claude/`, `.pi/`, `.agents/`, `.mcp.json`) — a message must not be able to write the agent's own instructions.
     - Refuse symlink entries encountered during the copy (a symlinked `../..` path would escape the destination); enforce a total-size cap (see Deferred Decisions for the value).
   - Failures return an error → the task fails loudly with the input index and reason; **the agent does not start on partial delivery** (an agent acting on half its inputs produces wrong work confidently — same reasoning as the hydration-refusal precedent in `pubsub_adapter.go`).
   - The parent's existing credential does the fetch: the `GITHUB_TOKEN`-backed global credential helper, or the agent's SSH deploy key when one is configured (see Risks for the deploy-key interplay).
   - Provenance (`repo@ref → <resolved-commit>`, dest, verified checksums) is printed to the job log and appended to the completion report so a human reviewing the PR can tell fetched inputs from agent-authored changes.

6. **Site-agent migration** (config, not code): the registry entry gains `tool_policy: ailang_only`, `inputs_allow: [sunholo-data/daneel-memory]`; Daneel's send path adds `inputs` (daneel repo, daneel#335). The lane's shell and its fleet token go away.

### Implementation Plan

**Phase 1: Typed field through the message plane** (~2 days)
- [ ] `TaskInput` type + `Validate` + JSON round-trip (`internal/coordinator/task_inputs.go`)
- [ ] `InboxMessage.Inputs`: struct field, SQLite migration (new `inputs` column), `schema.go` CREATE TABLE, all 3 SQL lists in `inbox.go`, Firestore converters
- [ ] `POST /api/messages` `inputs` field; `messages send --inputs-file`
- [ ] `Message.Inputs` + the 3 converters (pubsub hydration, `message_adapter`, `watcher`)
- [ ] Converter round-trip tests on BOTH backends (SQLite + Firestore) — the silently-dropped-field trap

**Phase 2: Task record, registry grant, dispatch enforcement** (~1.5 days)
- [ ] `TaskRecord.Inputs` + SQLite and Firestore task converters; both `daemon_tasks_polling.go` creation sites
- [ ] `AgentConfig.InputsAllow` (yaml + `coordinator agents` readout)
- [ ] `ValidateInputsForAgent` + wiring into `buildDispatchParams` as `ErrDispatchPermanent`; test: refusal fails the task, names the repo, does not requeue
- [ ] `DispatchParams.Inputs` → `AILANG_TASK_INPUTS` env + `config.TaskInputs()` + env docs table

**Phase 3: Job fetch** (~2 days)
- [ ] `fetchTaskInputs(ctx, workDir, inputs)` seam in `coordinator_cloud_inputs.go` (pure enough to test against local temp git repos), called from `executeCloudTask` Step 2.5
- [ ] Checksum verification (per-input sha256; `manifest.sha256` when present), symlink/overwrite/instruction-path/size refusal
- [ ] Provenance in job log + completion report
- [ ] Tests with temp git repos: happy path, excluded default dest, explicit dest, every refusal, zero-inputs regression

**Phase 4: Migration + docs** (~1 day)
- [ ] Site-agent registry entry flip (coordinator config repo) — behind a staged rollout
- [ ] `docs/docs/guides/coordinator.md` + agent-messaging guide: `inputs`, `inputs_allow`, failure reasons
- [ ] Daneel-side follow-up tracked on daneel#335 (out of this repo)

### Files to Modify/Create

**New files:**
- `internal/coordinator/task_inputs.go` (~150 LOC) — `TaskInput`, `Validate`, `ValidateInputsForAgent`, JSON helpers
- `internal/coordinator/task_inputs_test.go` (~250 LOC)
- `cmd/ailang/coordinator_cloud_inputs.go` (~200 LOC) — `fetchTaskInputs` + copy/verify/exclude
- `cmd/ailang/coordinator_cloud_inputs_test.go` (~250 LOC)

**Modified files:**
- `internal/messaging/inbox.go` (+15/-6) — struct field; 3 SQL column lists
- `internal/messaging/schema.go` (+2) — CREATE TABLE column
- `internal/messaging/schema_migrations.go` (+35) — new `inputs` column migration
- `internal/storage/firestore/messaging_convert.go` (+6) — inboxToMap/mapToInbox
- `internal/storage/firestore/coordinator_convert.go` (+6) — taskToMap/mapToTask
- `internal/coordinator/watcher.go` (+3) — Message field + copy
- `internal/coordinator/message_adapter.go` (+2) — converter
- `internal/coordinator/pubsub_adapter.go` (+2) — hydration copy
- `internal/coordinator/daemon_tasks_polling.go` (+4) — both TaskRecord sites
- `internal/coordinator/store.go` (+6) — TaskRecord field
- `internal/coordinator/store_sqlite_queries.go` (+8) — insert/select lists
- `internal/coordinator/store_sqlite_schema.go` (+3)
- `internal/coordinator/agent_registry.go` (+4) — InputsAllow
- `internal/coordinator/daemon_tasks_exec.go` (+12) — enforcement
- `internal/coordinator/cloud_dispatcher.go` (+8) — DispatchParams field
- `internal/dispatch/cloudrun/dispatcher.go` (+6) — env var
- `internal/config/job.go` (+10) — const, reader, docs row
- `cmd/ailang/coordinator_cloud.go` (+8) — Step 2.5 call
- `internal/coordinator/daemon_http.go` (+8) — postMessageRequest field
- `cmd/ailang/messages_send.go` (+25) — `--inputs-file`
- docs guides (coordinator, agent-messaging)

## Examples

### Example 1: Daneel site publish (the motivating case)

**Before** (site agent needs a shell carrying the fleet token to fetch its own attachments):
```
registry: site agent, tool_policy: full (shell), lane: pi
message:  "Publish the attached poster to the landing page"   # attachments unreachable without shell
```

**After** (no shell; the job fetches under the narrowest grant):
```yaml
# agent registry entry
- id: site
  tool_policy: ailang_only
  inputs_allow:
    - sunholo-data/daneel-memory        # exact match — the only repo this agent may be fed
```
```json
// POST /api/messages
{ "inbox": "site", "title": "Publish poster DNL-abc123",
  "content": "Publish the attached poster to the landing page",
  "inputs": [ { "repo": "sunholo-data/daneel-memory",
                "ref": "incoming/DNL-abc123",
                "path": "incoming/DNL-abc123",
                "dest": "content/images/dnl-abc123/" } ] }
```
The job clones the site repo, shallow-clones `daneel-memory` at `incoming/DNL-abc123`, verifies the manifest, copies the files to `content/images/dnl-abc123/` (new files — binary content the `ailang_only` agent could not have written with its text tools), records `daneel-memory@incoming/DNL-abc123 → <commit>` on the completion report, and starts the agent, which edits the landing page with AilangEdit.

### Example 2: Read-only reference material (design/writer agents)

```json
{ "inputs": [ { "repo": "sunholo-data/daneel-memory", "ref": "dev", "path": "notes/marketing-brief" } ] }
```
Empty `dest` → files land under `.incoming/1/`, excluded from git: the agent reads them with AilangRead (inside the policy root), the diff and PR stay clean, and the ambient "it happens to be my checkout" dependency becomes a declared input.

### Example 3: Refusal (the narrowest grant holding)

```json
{ "inputs": [ { "repo": "sunholo-data/private-infra", "ref": "main", "path": "terraform" } ] }
```
Dispatch refuses permanently; the task's thread carries: `inputs: repo "sunholo-data/private-infra" is not in inputs_allow for agent "site"`. The agent never runs; the job's credential is never pointed at the repo.

## Success Criteria

- [ ] Cloud task with inputs delivers files before the agent starts; provenance (repo@ref, resolved commit, dest, checksums) appears in the job log and completion report
- [ ] Repo outside `inputs_allow` → task failed pre-dispatch with the repo named (unit test asserts `ErrDispatchPermanent` + no requeue)
- [ ] Zero inputs → dispatch params, env vars, clone, and completion flows unchanged (regression test)
- [ ] sha256 / `manifest.sha256` mismatch → task fails before the executor starts; present-but-invalid manifest never silently passes
- [ ] Dest traversal (`..`), symlink entries, overwrite of existing files, and instruction-path destinations (`AGENTS.md` etc.) all refused (unit tests with temp git repos)
- [ ] Default dest excluded from the agent's diff (test asserts `excludeFromGit` was applied and the PR contains no `.incoming/` files)
- [ ] SQLite AND Firestore round-trip tests for both `InboxMessage.Inputs` and `TaskRecord.Inputs` (the silently-dropped-field trap)
- [ ] Site agent registry entry runs `tool_policy: ailang_only` + `inputs_allow` end-to-end on a staged publish (Daneel-side verification per daneel#335)
- [ ] All tests passing (`make test`); coordinator + messaging guides updated

## Testing Strategy

**Unit tests:**
- `TaskInput.Validate`: repo/ref/path/dest formats, count cap, sha256-requires-file
- `ValidateInputsForAgent`: empty allowlist denies; exact-match only; error text names repo and agent
- Converter round-trips: SQLite + Firestore, `InboxMessage` and `TaskRecord` (field survives both directions)
- `messages send --inputs-file`: valid file, malformed JSON (exit 1 with reason)

**Integration tests (local temp git repos, no network):**
- `fetchTaskInputs`: happy path; excluded default dest; explicit new-files dest; every refusal (bad ref, missing path, checksum mismatch, symlink, overwrite, instruction path, size cap); partial-delivery → error before any agent-visible placement is relied on
- Dispatch: task with disallowed input fails permanently and posts the reason; task with allowed input sets `AILANG_TASK_INPUTS`
- Regression: zero-input dispatch produces identical env/params as before the change

**Manual testing:**
- One staged cloud dispatch to the site agent with a real `daneel-memory` input branch (the triage harness from #1600 can be reused), verifying the completion report and the PR diff

## Deferred Decisions

The following are intentionally left open for the implementer:

- Total-size cap value for all inputs in a task — agent may choose (256 MiB suggested); it must be a named constant, not per-site magic numbers
- Exact shape of the provenance block in the completion report (log lines + report footer are the minimum) — agent may choose, but repo@ref and resolved commit must be present
- Whether the fetch uses `git archive` over HTTPS for single-file inputs instead of a clone — agent may choose if it stays shallow and credential-identical
- `coordinator agents` display formatting for `inputs_allow` — agent may choose

## Non-Goals

**Not attempted in this feature:**
- Local-lane (`execution_lane: local`) input fetching — different fetch point, no env plumbing; deferred until an agent actually needs it (D7)
- Per-repo scoped credentials for the fetch (e.g. a dedicated read-only deploy key per `inputs_allow` repo) — the natural hardening once more than one agent uses inputs; today the fleet token already reads the allowlisted repos, and this design strictly **narrows** how it is used (parent-only, allowlist-gated)
- Mutable-ref pinning to git SHAs — D5; `sha256` is the content pin
- Making inputs part of the AILANG language or the coordinator's task *content* — inputs are transport metadata, never prompt text
- General attachment storage (files riding the message store itself) — explicitly out; the issue's design keeps bytes in git, which gives versioning, provenance, and existing credentials for free

## Timeline

**Week 1** (~5.5 days):
- Phase 1: message plane (2d)
- Phase 2: registry + dispatch enforcement (1.5d)
- Phase 3 start: job fetch seam + refusals (2d)

**Week 2** (~2 days):
- Phase 3 finish: checksums, provenance, regression tests
- Phase 4: docs, staged site-agent flip, Daneel handoff

**Total: ~7.5 engineering days across 2 weeks** (initial estimate ~4d, doubled per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fetched content is prompt-injectable (a mailed poster's filename or a manifest could carry instructions an agent reads) | High | Inputs are data, not instructions: default dest is outside the diff; the agent's AGENTS.md must say `.incoming/` is untrusted data (migration step); instruction-path destinations are denied outright (D4) |
| Explicit-dest inputs become committable content on direct-push (`skip_approval`) agents even if the agent does nothing | Med | New-files-only, allowlisted repo, provenance in the completion report names input-sourced files so a human can distinguish them; site inbox is Daneel-authenticated, not public |
| Env-var size for `AILANG_TASK_INPUTS` | Low | Metadata only (≤16 small objects); bytes never travel in env |
| Agents using SSH deploy keys: the key is host-alias-bound to the workspace repo, so the parent's fetch of a *second* repo needs the fleet token lane | Med | Documented: deploy-key agents keep fleet-token reads for inputs (read-only usage, parent-only); a per-repo read-only key is the listed follow-up hardening |
| Converter drift (a hand-written SQL list or Firestore map omits `inputs` → field silently dropped, task runs without its files) | Med | Both-backend round-trip tests are success criteria (the `FinalizationLedger` precedent exists precisely because this shipped before) |
| Partial delivery (first input fetches, second fails) | Med | All-or-nothing before the executor starts; task fails loudly (A11) |

## Related Documents

**Implemented (may inform design):**
- [M-AGENT-AILANG-ONLY-EXECUTION](../v0_39_0/m-agent-ailang-only-execution.md) (v0.39.0) — the `ailang_only` lane this feature enables for the site agent
- [M-EXECUTOR-POLICY-HARDENING](../v0_41_0/m-executor-policy-hardening.md) (v0.41.0) — the sandboxed file tools (`AilangRead`/`AilangWrite`) the site agent will use on the fetched files

**Planned (check for overlap):**
- [M-EXECUTOR-ENV-HARDENING](../../planned/v0_49_1/m-executor-env-hardening.md) (v0.49.1) — E5/D2 own the credential boundary this feature's parent-side fetch respects; D2 anticipates "scoped per-task token" as its own follow-up
- [M-COORDINATOR-EXECUTION-TRUST](../../planned/m-coordinator-execution-trust.md) — the trusted-registry authority pattern (`work_tier`) that `inputs_allow` copies

**Duplicate gate:** SimHash/neural searches for "task inputs", "fetch repo files workspace", and "daneel site agent attachments" found no implemented or planned doc covering typed task inputs (closest hits are the four docs above, all adjacent — credential boundary, tool policy — none covers input delivery). Distinct by scope: this doc owns the message → dispatch → fetch path only.

## References

- **Issue**: #1600 (labels: feature, from:daneel, priority:P2, area:messaging) — reported by daneel via ailang messages; triage 2026-10-08 verified the defect live on origin/dev `658ff76a3`
- **Daneel side**: sunholo-data/daneel#335 (M-DANEEL-SITE-PUBLISH) — records the accepted D11 narrowest-grant violation this feature retires
- [Design Axioms](/docs/references/axioms)
- [MOTOKO.md](../../../MOTOKO.md) — harness lanes (pi lane, fleet token)

## Future Work

- Per-repo read-only credentials for input fetches (dedicated deploy keys per `inputs_allow` entry) — the full narrowest-grant endgame
- Local-lane input fetching if a bare-metal agent ever needs it (D7)
- Input revocation/expiry (an `expires` field) if input branches are pruned before tasks run — not observed as a failure yet

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08

## Verification Log

Every load-bearing claim above, checked against the code (branch `coordinator/task-3091adbc` @ `1dfd5615`, re-verifying the 2026-10-08 triage on origin/dev @ `658ff76a3`):

| # | Claim | Check | Result |
|---|-------|-------|--------|
| V1 | No `inputs` field/handling exists in `cmd/ailang/coordinator_cloud.go` (negative-existence; the issue's core premise) | `grep -ci "inputs" cmd/ailang/coordinator_cloud.go` | **0 matches** — confirmed |
| V2 | No `InputsAllow` / typed `TaskInput` exists anywhere in the coordinator today (negative-existence) | `grep -rn "InputsAllow\|TaskInput" internal/ cmd/ --include="*.go"` | **empty** — confirmed |
| V3 | `executeCloudTask` clones one repo at one branch, per-task override only via `AILANG_PUSH_BRANCH` | Read `coordinator_cloud.go` `executeCloudTask`: `config.PushBranch()` overrides `baseBranch`; single `git clone --branch` | Confirmed (issue's "coordinator_cloud.go:323-333" describes this function) |
| V4 | `daemon_tasks_exec.go:226` sets PushBranch from the agent's `merge_branch` | Read `daemon_tasks_exec.go:226-229` | Confirmed — `params.PushBranch = agent.MergeBranch` at line 227 |
| V5 | Messages carry text only; `InboxMessage` has no file/repo-reference field | Read `internal/messaging/inbox.go:17-42` struct | Confirmed (payload + routing/GitHub/embedding fields only) |
| V6 | "Daneel caps message blocks at 16000 bytes" / "fleet token already reads daneel-memory (multivac, 11 Sept)" | Sender-side facts from issue #1600 body | **Cited as issue claims** — not re-verifiable in this repo |
| V7 | `ailang_only` profile has no shell | Read `internal/executor/toolpolicy.go:51-56` (`ProfileTools`) | Confirmed — no `Bash`, no native Read/Write |
| V8 | There are **two** `TaskRecord` creation sites that mirror message fields | `grep -n "GithubIssue:" internal/coordinator/daemon_tasks_polling.go` | Confirmed — lines 207 and 511 |
| V9 | There are **three** `InboxMessage → Message` converters | Read `pubsub_adapter.go` (hydration), `message_adapter.go` (ListUnread), `watcher.go` (messageToTask) | Confirmed — all three copy fields manually |
| V10 | Inbox SQL column lists exist at 3 hand-written sites; a missed list silently drops the field | `grep -n "INSERT INTO inbox_messages\|SELECT id, message_id" internal/messaging/inbox.go` | Confirmed — lines 242, 294, 428; plus the `FinalizationLedger` comment in `store.go` documenting the trap |
| V11 | TaskRecord persistence has hand-written SQLite AND Firestore converters | `store_sqlite_queries.go:50,79`; `internal/storage/firestore/coordinator_convert.go:20,102` | Confirmed |
| V12 | Next SQLite messaging migration: chain's terminal recorded version is `1.9.0`; ALTER-TABLE-add-column precedent exists (envelope column, v1.7.0→v1.8.0) | Read `schema_migrations.go:77-90` (last chain step keys on `"1.9.0"`, re-runs the rebuild, error text says "v1.10.0" but the stored version stays `"1.9.0"` — line 577), `migrateV170ToV180` | Confirmed — new migration keys on `"1.9.0"`, records `"1.10.0"` |
| V13 | `excludeFromGit` helper exists and is worktree-aware (`.git/info/exclude` via `rev-parse --git-dir`) | Read `coordinator_cloud_github.go:432-450` | Confirmed — reusable as-is |
| V14 | Shallow-clone of a second repo inside the job already has a precedent (plugin repo) | Read `executeCloudTask` Step 0 (`git clone --depth 1 <pluginRepo> <dir>`) | Confirmed |
| V15 | Permanent dispatch errors fail the task and post the reason to its thread | Read `daemon_tasks_exec.go:376-386` (`ErrDispatchPermanent` → `MarkTaskFailed` + `postTaskResult`); `cloud_dispatcher.go:8-13` | Confirmed |
| V16 | The job credential is the fleet token in a global credential helper readable from child processes (why the fetch must stay parent-side) | M-EXECUTOR-ENV-HARDENING E5 (`coordinator_cloud.go` git config --global credential.helper with token literal) | Confirmed — cited from the planned doc, re-verified in code |
| V17 | The `ailang_only` policy root is the workspace, so `.incoming/` inside it is reachable by `AilangRead` | Read `internal/executor/agent_policy.go:42` (rel-path check vs workspace) | Confirmed |
| V18 | Empty `inputs_allow` must mean "no inputs" (deny-by-default zero value) | Precedent read: `AcknowledgeOnly` comment in `agent_registry.go` ("stated so the Go zero value is the LOUD direction") | Confirmed as the pattern to follow |
| V19 | Related-doc coverage gate: no existing doc covers typed task inputs | `ailang docs search` (SimHash, 1887 docs): "task inputs", "daneel site agent private branch attachments", manual greps of planned/ for "inputs", "daneel" | No duplicate/coverage hit ≥ threshold — proceed; adjacent docs cited in Related Documents |
| V20 | Create-script search step works but the script exits 1 when implemented-search returns no matches (`set -e` + `pipefail`: `merge_results`'s grep exits 1 on empty input) | Ran `create_planned_doc.sh m-task-inputs v0_52_6` → EXIT=1 after "(none found)"; traced `bash -x` | Confirmed — script bug recorded; scaffold filled manually from the script's own template (lines 209–449) |
| V21 | `AilangWrite` is a text tool (motivates D4's explicit-dest path for binary inputs): content is a JSON string written verbatim | Read `internal/policytool/fs_ops.go:120-143` (`write`: `WriteString(req.Content)`, transfer cap, no binary encoding surface) | Confirmed — binary placement must be done by the job's copy, not the agent's tools |

## Maintainer rulings (Ruled 2026-10-08 by Mark)

All decisions marked `human` in the decisions table (D1 typed field, D2 registry-side `inputs_allow`, D3 fetch in the job parent, D4 `.incoming/<n>/` default, D7 cloud lane only) are ratified as written.
