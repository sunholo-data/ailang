# Sprint Plan — M-CHAINS-EXECUTOR-TRANSCRIPTS

**Design doc:** [m-chains-executor-transcripts.md](m-chains-executor-transcripts.md)  
**Sprint ID:** `M-CHAINS-EXECUTOR-TRANSCRIPTS`  
**Target:** v0.39.1 · P1  
**Planned at:** `87d50e4c`, 2026-09-16  
**Duration:** **1.5 engineering days / about 12 hours**  
**Estimated change:** **~1,050 LOC** (about 440 implementation, 540 tests, 70 migration/docs)  
**Risk:** Medium-high (new Cloud Run authority and a cross-store schema/write path)

## 0. Binding decisions and scope correction

Mark ratified the design freeze in an attended session on 2026-09-16. The executor must not reopen
these decisions:

- **D1:** Cloud Run writes `obs_chat_messages` directly. There is no GCS/Pub/Sub fallback in this
  sprint. O1, the job-SA Firestore permission verification/grant in `ailang-multivac` terraform,
  is the first M3 task and a hard gate on every later M3 task.
- **D2:** add `reason_tokens` through observatory `migrate_v22` and both storage backends.
- **D3:** the recorder and `ChatTurn` live in `internal/executor`; executor must not import
  observatory.
- **D4:** per-turn usage is delivered through the optional `TurnUsageHandler` interface.
- **D5:** both existing writers (`importer_motoko.go` and `claudehistory/importer.go`) migrate to
  `Backend.PutChatMessages` **in this sprint**. `ChatMessage` must first gain the fields required to
  preserve their current data. The end state is one chat-writing abstraction, not three.

The design doc's Deferred Decisions, Non-Goals, Timeline, Risks, and Future Work still contain
pre-ratification text that defers D5 and quotes the old one-day estimate. M4 updates those sections;
the checked Design Freeze above is authoritative until then.

## 1. Baseline and capacity

- Working tree was clean when planned; HEAD contains only the ratification commit in the last seven
  days, so repository-local seven-day LOC velocity is not informative.
- The design estimated 8 hours / 500–700 LOC before D5. Mark explicitly budgeted about half a day
  for importer convergence. This plan therefore budgets 12 hours and ~1,050 LOC.
- Code inspection confirms two raw-SQL non-test writers, no `Backend.PutChatMessages`, no Firestore
  writer, and schema migration v21 as the current tip.
- The two importers currently preserve `content_text`, `content_thinking`, `request_id`, and
  `stage_id`; the canonical message type/write method must preserve those fields rather than silently
  dropping them.
- `go` is absent in this planning workspace (`go: command not found`), so package baselines could not
  be executed here. M0 requires a Go-enabled executor and records clean base results before edits.

## 2. Milestone map

| Milestone | Outcome | Estimate | Depends on |
|---|---|---:|---|
| M0 | Execution preflight and green baselines | 0.5h / 0 LOC | — |
| M1 | Executor-local transcript recorder, pi usage, and cap mutation proof | 3.5h / 410 LOC | M0 |
| M2 | migrate_v22, canonical backend writer, and both importer migrations | 4.0h / 390 LOC | M0 |
| M3 | Rig + Cloud Run persistence, with O1 verified before cloud code | 3.0h / 210 LOC | M1, M2, **O1** |
| M4 | End-to-end checks and design-doc reconciliation | 1.0h / 40 LOC | M1–M3 |

M1 and M2 may proceed in either order after M0. M3 must not begin until their APIs are stable; within
M3, O1 is performed and evidenced before any Cloud Run writer work.

## 3. Milestones

### M0 — Preflight and discriminating baselines

**Files:** none.

Tasks:

1. Use a workspace with the repository-required Go version; record `go version`.
2. Run, without piping away return codes:
   `go test ./internal/executor/... ./internal/observatory/... ./internal/claudehistory/... ./internal/storage/firestore/... ./internal/coordinator/... ./cmd/ailang/...`.
3. Record `make check-boundaries` and `git status --short`. Do not absorb unrelated failures into
   this sprint; distinguish red-at-base gates from regressions.

Acceptance criteria:

- [ ] Go is available and relevant package baselines have an informative result.
- [ ] Any baseline failure is recorded with its exact command and excluded only with user approval.
- [ ] No unrelated working-tree changes are overwritten.

### M1 — Executor transcript recorder and pi usage

**Files:** `internal/executor/transcript.go` (new), `internal/executor/transcript_test.go` (new),
`internal/executor/pi/pi.go`, pi fixture/tests, and the local handler-construction site.

Build:

- Add executor-local `ChatTurn`, `TurnUsage`, `TurnUsageHandler`, and decorator-style
  `TranscriptRecorder`. It forwards every event unchanged while forming ordered assistant/user
  content-block turns.
- Cap each tool result at exactly 8 KiB with an explicit truncation marker and assistant text at
  64 KiB. Return a defensive snapshot, not mutable recorder-owned storage.
- Drive optional per-turn usage from pi `message_end`; bank `tokens_out = output - reasoning` and
  `reason_tokens = reasoning`. Missing usage remains zero.
- Set pi's `Result.Transcript` from its existing transcript buffer and expose the recorder's structured
  turns to the coordinator path without importing observatory.

Acceptance criteria:

- [ ] A synthetic five-turn replay proves role order, content-block order, forwarding, usage, and a
  defensive returned slice.
- [ ] Pi fixtures prove reasoning/output disjointness and prove absent usage remains zero.
- [ ] `Result.Transcript` is non-empty for a successful pi fixture run.
- [ ] A 456-turn generated stream remains bounded and every stored tool result is at most the cap
  including its marker.
- [ ] **Required cap mutation:** first run the named cap test green; change only
  `maxToolResultBytes` from `8192` to `16384` while using a fixture between those sizes, assert the
  named test fails because the stored byte length/marker is wrong, restore `8192`, and assert it is
  green again. Record all three return codes. A compile failure does not count as a killed mutant.
- [ ] Relevant executor/pi tests, gofmt, and vet pass.

### M2 — One canonical chat writer and importer convergence

**Files:** `internal/observatory/migrate_v22.go` + tests (new), `migrate.go`, schema definitions,
`store_chat.go`, `backend.go`, `backend_sqlite.go`, Firestore observatory helpers/tests,
`importer_motoko.go`, `internal/claudehistory/importer.go`, and importer tests/fixtures.

Build:

- Add additive migration v22 for `reason_tokens INTEGER NOT NULL DEFAULT 0`; update fresh schema and
  all query/scan mappings.
- Extend `observatory.ChatMessage` to losslessly represent the existing importer columns:
  `content_text`, `content_thinking`, `request_id`, and `stage_id`, plus `reason_tokens`.
- Add `Backend.PutChatMessages(ctx, msgs)` and real SQLite and Firestore implementations. SQLite uses
  one transaction; Firestore uses deterministic document IDs, includes `expire_at`, and preserves
  all correlation/usage/content fields. Empty input is a no-op; partial/invalid data fails loudly.
- Refactor both motoko and claudehistory importers to build `ChatMessage` values and call the canonical
  method. Preserve their surrounding import-status/session transaction semantics or explicitly make
  the new boundary atomic; do not create a partial-success silent fallback.
- Remove all direct non-test `INSERT INTO chat_messages` calls outside `PutChatMessages`.

Acceptance criteria:

- [ ] v21→v22 and fresh-database tests show `reason_tokens=0` for old rows and round-trip non-zero data.
- [ ] SQLite insert/replay of identical IDs is idempotent and queries by task and session round-trip
  every rich field.
- [ ] Firestore mapping/writer tests prove deterministic IDs, `expire_at`, reason/cache tokens, rich
  content fields, and propagated write errors (emulator if available; otherwise a fake client seam).
- [ ] Existing motoko and claudehistory fixtures produce field-equivalent rows before/after the
  refactor, including content/thinking/request/stage correlation.
- [ ] `rg -n 'INSERT (OR REPLACE )?INTO chat_messages' internal cmd` reports only the canonical
  SQLite writer, migrations/schema fixtures, and tests—no importer writer.
- [ ] Compile-time assertions show both SQLite and Firestore implementations satisfy `Backend`.

### M3 — Rig and cloud persistence; O1 is the hard gate

**Files:** coordinator handler/finalize files, `cmd/ailang/coordinator_cloud_executor.go`, cloud
writer tests; external `ailang-multivac` terraform for job-SA grant and chat TTL policy.

Tasks in mandatory order:

1. **O1 — before any other M3 work:** identify the exact Cloud Run execute-job service account from
   `ailang-multivac/terraform/cloud_run_jobs.tf`; inspect its effective IAM policy with
   `gcloud projects get-iam-policy`; record whether it already has the minimum
   `datastore.documents.create/update` authority. If absent, add the scoped terraform grant, apply it
   through the multivac workflow, and re-query effective policy. **Do not start task 2 until O1 is
   verified.** If access to the repo/project/apply path is unavailable, stop M3 as blocked—D1 does
   not authorize falling back to GCS or Pub/Sub.
2. Add/verify the 30-day `obs_chat_messages.expire_at` TTL policy in the same infrastructure change.
3. Rig: wrap the local executor handler and persist recorder turns through `PutChatMessages` at the
   completion/finalize boundary with deterministic IDs and task/session/chain correlation.
4. Cloud: wrap `cloudEventHandler`, construct the project-pinned Firestore observatory backend, and
   flush each completed turn directly so a killed job retains prior turns. Writes must be idempotent
   and errors must be surfaced; do not wait until job completion.

Acceptance criteria:

- [ ] O1 evidence names the service account, terraform resource/role, applied revision, and effective
  policy result; its timestamp precedes the first M3 cloud-code change.
- [ ] An integration test proves repeated local finalize produces one logical row per deterministic ID.
- [ ] A cloud-handler test proves a completed turn triggers a write immediately and a later cancellation
  does not erase it; a failed write is observable.
- [ ] Project selection follows the repository's explicit observatory/project-pinning rules, with no
  ambient/default-project fallback.
- [ ] TTL policy is present and targets `obs_chat_messages.expire_at` at 30 days.

### M4 — Live acceptance and documentation reconciliation

**Files:** `design_docs/planned/v0_39_1/m-chains-executor-transcripts.md`, changelog entry if required.

Acceptance criteria:

- [ ] A local pi task is visible through `ailang chains chat <chain> --stage N` with ordered content
  and non-zero pi usage.
- [ ] A cloud pi task is visible through the remote chains-chat path; a deliberately cancelled task
  retains all turns completed before cancellation.
- [ ] For a fixture/live pi run,
  `sum(tokens_out) + sum(reason_tokens) == Result.OutputTokens`.
- [ ] The design doc no longer says D5 is deferred or out of scope and reports the ratified 1.5-day
  budget/actual outcome consistently.
- [ ] `go test` for all touched packages, `go vet` for touched packages, `make check-boundaries`,
  `make fmt`, and the informative repository build gate are green.

## 4. Risks and controls

| Risk | Control |
|---|---|
| O1 authority missing or unverifiable | O1 is a hard, evidenced precondition to M3; stop M3 rather than silently change D1. |
| Importer migration loses rich columns | Expand `ChatMessage` first and use before/after fixtures that compare every existing field. |
| Import status commits while chat write fails | Preserve atomicity where possible; otherwise fail before marking import complete and test retry. |
| Recorder cap test is vacuous | Required 8KiB→16KiB compiling mutation must be killed and restored. |
| Firestore amplification / unbounded retention | One write per completed turn, deterministic IDs, 30-day TTL. |
| Cancellation loses buffered turns | Cloud writer flushes on completed turns, not only at task completion. |

## 5. Definition of done and handoff

- All M0–M4 criteria pass and the progress JSON contains no placeholders.
- The final writer audit demonstrates one abstraction (`Backend.PutChatMessages`) and no raw importer
  inserts.
- O1 and live cloud checks are evidence-bearing, not inferred from unit tests.
- This plan does **not** authorize implementation. Per repository routing, Mark must approve this plan
  and then explicitly say **“execute sprint”** before `sprint-executor` starts.

