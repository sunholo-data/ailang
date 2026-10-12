# Sprint Plan: M-TASK-INPUTS — Typed cloud task inputs

Refs #1600

**Design:** [m-task-inputs.md](m-task-inputs.md) (approved handoff task-3091adbc).  
**Target:** v0.52.6 · **Priority:** P2 · **Status:** Completed locally on 2026-10-09; independent evaluation passed 95/100. Staged cloud deployment and Daneel migration pending.
**Duration:** 8 engineering days across two weeks, approximately 48–64 hours.  
**Estimate:** 2,100 LOC: 1,150 implementation/docs + 950 tests; **Risk:** High at filesystem and authority boundaries.

## Goal and scope

Deliver declared external repository files to a cloud workspace before the agent starts, with a trusted per-agent repo grant, checksums and provenance. Preserve empty-input behavior. This implements the approved cloud-only design; local fetching, per-repo new credentials, raw SHA refs and language changes remain deferred. A local task that declares inputs must fail explicitly rather than run without its files.

## Evidence and planning assumptions

Read issue #1600 and its comments through the GitHub API on 2026-10-08 (comments array empty). The issue describes Daneel attachments pushed to a private incoming branch and a site agent currently needing a token-bearing shell. No new issue is needed. Current checkout is coordinator/task-c2fc077e, clean at planning start, version v0.52.5; inherited design is committed at 35c4dfb3. The approved handoff came from coordinator/task-3091adbc; work stays on this checkout.

The seven-day velocity script found only the design commit and no usable LOC metrics; this shallow history cannot support a measured LOC/day claim. Current changelog records v0.52.5 record-binding fixes and v0.52.4 file-handoff work, but provides no comparable elapsed implementation time. Use an explicit planning assumption of roughly 260 changed LOC/day, including tests, with eight days allocated versus the design's 5–7-day header and 7.5-day detailed timeline. Estimates include approximately 25% buffer over six productive days. Re-estimate after M1 rather than treating this as observed throughput. Repository-wide coverage generation is deferred to implementation; this planning task runs no implementation tests.

Code review confirms the message adapters depend on messaging, while Cloud Run dispatch imports coordinator. Defining TaskInput only in coordinator as the design sketch suggests would create a cycle when InboxMessage uses it. M1 defines the canonical transport type in internal/messaging/task_inputs.go and exposes a coordinator alias if needed; trusted-agent authorization stays in coordinator. This preserves the wire design and package layering.

The current code already uses envTokenCredentialHelper, which reads GITHUB_TOKEN at call time, and executor environment filtering; the design's claim that the global helper embeds the token is stale. Reuse these implemented controls and test them rather than rebuilding environment hardening. A workspace-scoped child git credential must not grant reads of the external input repository. Fetch clones are temporary and kept outside the agent workspace.

Treat D7's unchecked confirmation box as resolved by the user's approval of the design's explicit cloud-only non-goal, without expanding scope. The issue prose mentions repo/ref allowlisting; the approved v1 grant is exact repo-only, with ref syntax validation and checksum content pinning. Ref-specific authorization is not silently added.

## Registry reuse audit

Ran `ailang pkg search 'task inputs'`, `ailang pkg search 'git'`, and `ailang pkg search 'checksum'`: all returned no packages. The binary emitted a stale-source warning, so this is a recorded registry query rather than evidence of current compiler behavior. No candidates exist to inspect with pkg info/docs. Every milestone records `none` in the JSON, since this is Go host plumbing. Reuse the existing message/task stores, converters, dispatch permanent-error handling, gitexec wrapper, credential helper, excludeFromGit and completion reporting.

## Milestones

### ✅ M1_TYPED_MESSAGE_PLANE: Define the shared typed input contract and preserve validated inputs through every message ingress, store and adapter.

**Estimate:** 360 implementation/docs + 290 tests = 650 LOC; 2.5 days.  
**Dependencies:** None.  
**Registry reuse:** none (audit above).

**Files to create/update:** `internal/messaging/task_inputs.go`, `internal/coordinator/task_inputs.go`, `internal/messaging/inbox.go`, `internal/messaging/schema.go`, `internal/messaging/schema_migrations.go`, `internal/storage/firestore/messaging_convert.go`, `internal/coordinator/daemon_http.go`, `cmd/ailang/messages_send.go`, `internal/coordinator/message_adapter.go`, `internal/coordinator/pubsub_adapter.go`, `internal/coordinator/watcher.go`. Add adjacent focused test files for modified seams.

**Acceptance criteria:**

- [x] One canonical TaskInput type is usable by messaging and coordinator without an import cycle; JSON field names match the approved design.
- [x] Validation rejects malformed repo/ref, raw SHA refs, traversal, invalid digest and more than 16 inputs; HTTP returns 400 and CLI rejects malformed inputs files.
- [x] SQLite insert/list/get and Firestore map round trips preserve all fields; legacy records decode with no inputs, malformed stored JSON returns an error.
- [x] Migration from the current terminal schema preserves existing messages and inputs across reopening; fresh schema and repeated startup work.
- [x] HTTP, CLI, Pub/Sub hydration, polling adapter and watcher preserve the same ordered input list.

### ✅ M2_TASK_GRANTS_DISPATCH: Persist task inputs and enforce the trusted registry grant before cloud dispatch, then carry metadata to the job.

**Estimate:** 250 implementation/docs + 200 tests = 450 LOC; 1.5 days.  
**Dependencies:** M1_TYPED_MESSAGE_PLANE.  
**Registry reuse:** none (audit above).

**Files to create/update:** `internal/coordinator/store.go`, `internal/coordinator/store_sqlite_queries.go`, `internal/coordinator/store_sqlite_schema.go`, `internal/storage/firestore/coordinator_convert.go`, `internal/coordinator/daemon_tasks_polling.go`, `internal/coordinator/agent_registry.go`, `internal/coordinator/daemon_tasks_exec.go`, `internal/coordinator/cloud_dispatcher.go`, `internal/dispatch/cloudrun/dispatcher.go`, `internal/config/job.go`. Add adjacent focused test files for modified seams.

**Acceptance criteria:**

- [x] Both task creation sites copy inputs; SQLite and Firestore task round trips preserve fields, order and legacy empty values.
- [x] inputs_allow accepts exact owner/repo entries from the trusted agent registry; empty grants deny all requested inputs and sender content cannot widen the grant.
- [x] Unauthorized repo wraps ErrDispatchPermanent, calls MarkTaskFailed and posts the named repo/agent reason without dispatch or requeue.
- [x] Allowed cloud inputs serialize as AILANG_TASK_INPUTS; malformed env JSON fails explicitly; absent inputs preserve previous dispatch params/env behavior.
- [x] Local tasks with inputs fail explicitly as unsupported before executor startup; no local fetch behavior is introduced.

### ✅ M3_PARENT_FETCH_PROVENANCE: Fetch and verify all inputs in the cloud job parent before starting the executor, preserving workspace and credential boundaries.

**Estimate:** 440 implementation/docs + 360 tests = 800 LOC; 2.5 days.  
**Dependencies:** M2_TASK_GRANTS_DISPATCH.  
**Registry reuse:** none (audit above).

**Files to create/update:** `cmd/ailang/coordinator_cloud_inputs.go`, `cmd/ailang/coordinator_cloud_inputs_test.go`, `cmd/ailang/coordinator_cloud.go`, `cmd/ailang/coordinator_cloud_github.go`, `cmd/ailang/coordinator_cloud_executor.go`. Add adjacent focused test files for modified seams.

**Acceptance criteria:**

- [x] Branch/tag shallow clones resolve HEAD and deliver a file, directory or whole tree in input order before the executor starts; missing refs/paths fail loudly with the input index.
- [x] Default .incoming/<n>/ is excluded from git; explicit destinations create new files only; copied whole trees omit .git metadata.
- [x] Source and destination traversal, source symlinks including ancestor components, destination symlinks, existing-file collisions and harness instruction paths at any depth are rejected.
- [x] Single-file sha256 and every manifest.sha256 entry are checked; malformed manifests, missing entries, escaping entries and mismatches fail before executor invocation.
- [x] A named 256 MiB aggregate copied-byte cap is enforced while reading; all inputs stage and verify before delivery, failures clean temporary clones and never start the executor.
- [x] Provenance in job logs and completion report includes repo, requested ref, resolved commit, effective dest and verified digests; failure reports retain indexed reasons.
- [x] Parent uses the existing HTTPS token helper for allowlisted input reads; tests prove the child does not receive the fleet token or input clone credential material, including the workspace SSH deploy-key case.
- [x] Zero-input tasks retain previous clone, executor and completion behavior; temp-repo integration tests require no network or model.

### ✅ M4_DOCS_EXAMPLES_ACCEPTANCE: Document task input configuration, add runnable transport examples and bank end-to-end acceptance evidence with a staged Daneel follow-up.

**Estimate:** 100 implementation/docs + 100 tests = 200 LOC; 1.5 days.  
**Dependencies:** M3_PARENT_FETCH_PROVENANCE.  
**Registry reuse:** none (audit above).

**Files to create/update:** `docs/docs/guides/agent-messaging.md`, `docs/docs/guides/coordinator.md`, `docs/docs/guides/coordinator-setup.md`, `examples/task_inputs/inputs.json`, `examples/task_inputs/agent-config.yaml`, `examples/task_inputs/README.md`. Add adjacent focused test files for modified seams.

**Acceptance criteria:**

- [x] Examples show default excluded inputs, explicit binary placement and exact inputs_allow with tool_policy ailang_only; JSON examples validate with the production validator and sample config parses.
- [x] Guides document --inputs-file, HTTP inputs, AILANG_TASK_INPUTS, branch/tag-only refs, checksums, copied-byte cap, permanent errors, untrusted data and cloud-only support.
- [x] Focused package tests and make test, make lint, make check-boundaries pass; changed validators/copy code target at least 85 percent statement coverage with refusal paths asserted.
- [x] Bank a no-network end-to-end ingress-to-job fixture showing bytes/provenance and denied-input no-dispatch behavior; attach a staged cloud publish checklist for daneel#335.
- [x] When deployment access exists, verify the site agent with a real daneel-memory branch, no shell, correct PR diff and provenance; otherwise record the external acceptance as pending without claiming the migration completed.

## Day-by-day execution

| Day | Work and checkpoint |
|---|---|
| 1 | M1: shared type and table-driven format/count validation; CLI/HTTP ingress and old/new SQLite migration fixtures. |
| 2 | M1: message SQLite lists and Firestore maps; Pub/Sub, adapter and watcher preservation; targeted round trips. |
| 3 | Finish M1 (half day); M2 task stores and both task-creation sites; registry validation and display. |
| 4 | Finish M2: exact-grant authorization, permanent failure/thread posting, local refusal and env reader/dispatcher tests. |
| 5 | M3: injectable clone seam using gitexec, branch/tag resolution, bounded staging and safe copy with collision tests. |
| 6 | M3: digests/manifests, symlink/ancestor and instruction-path refusals, aggregate cap and cleanup. |
| 7 | Finish M3 (half day): report provenance, executor-not-started and child credential tests; start M4 examples/guides. |
| 8 | Finish M4: no-network end-to-end regression, full required checks, coverage review and staged cloud acceptance evidence or explicit external pending record. |

## Validation and success metrics

Run focused tests after each milestone: `go test ./internal/messaging/... ./internal/coordinator/... ./internal/storage/firestore/... ./internal/dispatch/cloudrun/... ./internal/config/... ./cmd/ailang/...` (select changed packages during the inner loop). M3 additionally runs `go test ./internal/executor/...` for credential-boundary regressions. Before handoff to sprint-evaluator run `make test`, `make lint`, `make check-boundaries` and format changed Go files with gofmt. Target >=85% coverage in new pure validation and copy helpers; assert every security refusal and both storage backends, rather than depending on a repository aggregate. Inspect migrations with fresh DB, old DB, reopen and repeated startup fixtures. No network/model is required for automated acceptance.

Examples are JSON/YAML transport examples, not .ail code: `examples/task_inputs/inputs.json`, `agent-config.yaml` and README instructions must validate through production paths. Include directory/file, default/explicit destination and denial cases. Plan the staged cloud check as external evidence, never as a substitute for automated tests.

## Risks, dependencies and executor choices

- Persistence drift: enumerate every insert/select/map and all three adapters; assert read-back through normal store APIs. Inspect the unusual existing messaging terminal migration before assigning the next version; do not drop/rebuild data or assume its error-text version is the stored version.
- Grant bypass: authorize at trusted dispatch, validate again in the job for malformed transport, and test that sender-chosen text/inbox cannot expand inputs_allow. No fleet credential is passed to the agent for input access.
- Filesystem escapes: use Lstat on source/destination ancestors and entries; validate manifest entries relative to copied root; deny instruction filenames/directories at any depth; omit .git when copying whole trees. Stage every input and check cross-input collisions before placement. The executor must never start after any failure; rollback incomplete placement where practical.
- Resource bounds: choose the design's suggested named 256 MiB aggregate copied-byte cap and enforce it during reads. Shallow clone size itself is not bounded by that cap; use job context cancellation and existing job resource limits, document this distinction, and avoid claiming a network-transfer cap.
- SHA256 shape: syntax checks validate 64 hex digits; fetched filesystem checks establish that an input with a per-file digest actually names a regular file. A whole-tree or directory input can use a manifest instead.
- External rollout: Daneel sender and live site registry/AGENTS.md live outside this repo. Document required inputs_allow, ailang_only and untrusted-data instructions and hand off against existing daneel#335. Live rollout requires deployment access and a staged publish; do not modify production registry configuration merely to complete local planning.

## Completion and approval boundary

This sprint produces code, tests, transport examples and guides in this repository. Live site-agent migration evidence is tracked separately if credentials/deployment access are absent; final evaluator must state that limit. No unresolved design question blocks planning. Narrower per-repo credentials and local fetching require later work.

PR body must include **Refs #1600**; link only the existing issue and do not open a new one. Suggested PR body: “Plan typed cloud task inputs across messaging, trusted dispatch and parent-side fetch, with storage round trips, safe delivery, checksums and provenance. Refs #1600. Validation: sprint JSON schema/milestone checks and git diff whitespace check.”

The coordinator consumes the plan and JSON markers and presents this sprint for approval. Merging the coordinator sprint-plan PR is the execution gate documented by the sprint-planner skill; do not self-approve or start sprint-executor from this planning task. After approval, executor reads the JSON in milestone order; evaluator checks the criteria against the approved design and records any external acceptance still pending.

## Execution evidence — 2026-10-09

All four milestones are implemented locally after the cloud executor was blocked
before implementation (missing make/CGO toolchain). The companion sprint JSON and
`.ailang/state/sprints/M-TASK-INPUTS/validation.json` bank acceptance and all required
gates. Canonical validation coverage is 98.39%; filesystem/manifest helper coverage
is 90.81%. The no-network fixture delivers actual binary bytes from temporary
branch/tag repos, resolves commits, verifies git exclusion/provenance and cleans
clones; transport/store/denial and credential boundary regressions also pass.

The estimate was 2,100 changed LOC over eight engineering days. Actual implementation
was one attended agent session (approximately 47.6 minutes through independent
review and checks); this is not a new measured human LOC/day baseline. The existing
cloud execution function was moved into a companion file to keep every file under
800 lines. Implementation commit `0b5993561` changes 71 files with 3,301 insertions and
648 deletions, including the move, docs and banked coverage/evidence.

The final external criterion is satisfied by recording rollout as **pending**, not
by claiming a live publish. Deployment of coordinator/executor builds, site registry
changes and Daneel sender migration remain external work; the staged checklist is
in `examples/task_inputs/README.md`. The codex-go Dockerfile installs make, gcc and
libc6-dev; its image was not built locally because Docker is unavailable.
