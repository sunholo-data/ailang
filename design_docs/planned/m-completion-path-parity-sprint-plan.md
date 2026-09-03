# Sprint Plan — M-COMPLETION-PATH-PARITY (M1 – M5)

**Design doc**: [m-completion-path-parity.md](m-completion-path-parity.md) (r14, 53 verification rows)
**Sibling (NOT in this sprint)**: [m-coordinator-execution-trust.md](m-coordinator-execution-trust.md) — owns the tier model (D4)
**Created**: 2026-09-03 · attended session, main checkout, commits straight to `dev`
**Rulings in force**: D2 per-edge · D3 abandon · D5 terminal-nothing-follows (Mark, attended, 2026-09-03)
**Status**: **M0, M0b, M1 (cloud half), M2, M3, M4, M5 LANDED on `dev`** (`b5dd366cb`, `d4aa5757c`, `092a27413`, `c22bf06b0`, `8e72dc630`). ~3,500 LOC, 60 test arms. **M1b — consolidating the DAEMON call site onto the orchestrator — is deliberately NOT done**; see below.
**Risk**: **high** — the extraction touches the one completion path that currently works. Thirteen quorum rounds falsified fourteen claims about this code; assume the fifteenth is in here somewhere.
**Estimated**: ~5 days, ~1,150 LOC across 5 milestones + reconciliation + rollout

## Goal

The pipeline that is already configured in prod — `design-doc-creator → sprint-planner →
sprint-executor → sprint-evaluator` — actually runs, end to end, on real work, with the artefact
landing in the right repo.

## Why the order is fixed

**M0 first, and it is not optional.** The daemon path is the *working* one, and the extraction's
whole safety argument is that it preserves that behaviour column-for-column. That claim is worthless
unless the current behaviour is pinned **before** anything moves. V43/V45/V48 each corrected the
matrix *after* it was written down — three times in three rounds. M0 turns the matrix into a test.

**M1 cannot precede M0b.** Every effect the orchestrator replays must be idempotent first, and
V31/V32/V34/V31fs/V32fs/V34fs showed that **three of them are not, on both backends, differently**.
Building the orchestrator on today's primitives would crash-loop on SQLite approvals and double-count
cost everywhere.

**M3 before any new auto-approved edge (D2 + C3).** An auto-approved handoff released from a card
showing "Files (0)" is the failure #921 was written for, with a robot on the other end.

---

## Milestones

### M0 — Pin the daemon's current behaviour (~120 LOC, ~0.5 day)

A characterisation test asserting the D1 outcome × effect matrix against the **existing** daemon
path, before a line moves. All four outcome columns, all thirteen effects — including effect 13
(`updateStageError`), which was missing from every matrix until round 12.

- RED-first is inverted here on purpose: **M0 must pass on unmodified `dev`.** If it does not, the
  matrix is still wrong and the sprint stops until it is right.
- Covers: `completed`/skip_approval, `completed`/normal, `failed`. `no_changes` has no daemon arm
  (V48) — asserted absent, not skipped.

**Done when**: the matrix is executable and green against untouched code.

### M0b — Idempotent store primitives, both backends (~320 LOC, ~1.5 days)

Three primitives, each in SQLite **and** Firestore, because they fail differently in each (V31 vs
V31fs, V34 vs V34fs):

| Primitive | SQLite today | Firestore today | Fix |
|---|---|---|---|
| `CreateApprovalIfAbsent` | bare `INSERT` → UNIQUE violation (V31) | `Doc(id).Set` → **overwrites a resolved approval back to `pending`** (V31fs) | `ON CONFLICT DO NOTHING` / `Create` treating `AlreadyExists` as success |
| `PutMessageIfAbsent` | bare `INSERT` → errors (V34) | `Doc(id).Set` → overwrites a read message (V34fs) | same shape |
| `SetStageMetrics` / `SetChainMetrics` | `SET cost = cost + ?` (V32) | `firestore.Increment` (V32fs) | absolute; chain total recomputed atomically |

The chain recompute is **one atomic operation, never read-then-write**: on SQLite a single row-value
`UPDATE … SET (total_cost, total_tokens, total_turns) = (SELECT …)`; on Firestore inside
`RunTransaction` (V26) — the one transaction in the design, and it is within a single store.

Plus the ledger's home: `ALTER TABLE tasks ADD COLUMN finalization TEXT` (V41) **and**
`TaskRecord.Finalization` with entries in **both** directions of the hand-written Firestore converter
(V44) — omit either and the ledger is silently dropped on read, and the protocol quietly does nothing.

**Mutation arms**: apply each primitive twice; assert the second application changes nothing. For
metrics, assert the **value**, not that a write occurred — a double-count is invisible to a write-count
assertion.

### M1 — The orchestrator and the ledger (~330 LOC, ~1.5 days)

`internal/coordinator/task_finalize.go`: `FinalizeTaskCompletion`, the `ExecutionStrategy` interface
with an explicit `Kind()` (no `Cleanup` no-op body), and the per-effect ledger.

- **Outcome-driven, not unconditional** (V43). The D1 matrix is the specification; M0 is its test.
- **Status writes are compare-and-set**, ordered to advance only. A stale replay must never regress a
  task the approval processor has already advanced; a non-matching CAS records `superseded`.
- **The claim carries the payload**, or recovery cannot recover — the stale-task detector reads only
  the database and has no `TaskCompletion`.
- **A Pub/Sub redelivery proceeds immediately.** It carries the payload and every write is CAS-guarded,
  so it cannot corrupt. Making a 60s redelivery (V22) wait out a 10-minute lease would be an
  unbounded wait; the threshold governs the *sweep* only, which reuses the 2-minute stale-task
  detector already running in prod (V46).
- Both call sites route through it: `daemon_tasks_exec_run.go` (daemon strategy) and
  `pubsub_completion_handler.go` (cloud strategy).

**Arms**: crash injection on both sides of every ledger write; concurrent daemon/Pub-Sub finalisation
converging to exactly one approval, one stage transition, one metrics value, one handoff; a CAS-regression
arm proving an approved task is never reset.

### M2 — Cloud stages advance (~90 LOC, ~0.5 day)

Effects 6–9 and 13 run on the cloud path, so a stage leaves `pending` and carries metrics. Cost goes
through **M0b's absolute setters** — explicitly *not* `updateChainMetrics`, which V32 proved increments.

**Acceptance**: a cloud task's stage reaches a terminal status with non-zero metrics. This has never
happened in the system's history.

### M3 — A cloud approval carries its diff (~180 LOC, ~1 day)

The coordinator has no clone and cannot run `git diff` at any price (V37). The **executor** does the
work:

- `git rev-parse HEAD` after the push at `coordinator_cloud.go:529`, which captures no SHA today (V40).
- `git diff --stat` and a capped `git diff <base>..<head>` — **new invocations**;
  `discoverChangedFilesFromCommit` returns filenames only (V42), so this is not an extension of an
  existing path.
- `BaseCommit` (= `clonePoint`), `HeadCommit`, `DiffStat` and the capped `Diff` join `TaskCompletion`.

Two immutable SHAs, so the diff is identical on replay. Missing either ⇒ an explicit
`diff_unavailable` reason on the approval, never a confident "Files (0)".

### M4 — Reconcile the leak (~70 LOC, ~0.5 day) — D3

One-shot pass marking the 315 pre-fix `active` chains **abandoned**, creating no synthetic stage
transitions. Carries `ALTER TABLE execution_chains ADD COLUMN reason TEXT` (V36). Plus a recurring
check flagging any chain still `active` whose task is terminal — so the next occurrence of this class
surfaces in hours, not the four months this one took.

### M5 — The anti-drift arm (~40 LOC, ~0.5 day)

Assert both call sites route through the orchestrator, and that the configured observatory backend
implements chain writes — the GCP backend no-ops every one of them and returns `nil` (V20), so a
backend swap would erase the chain model with no signal.

**This defect existed because nothing could see the two paths side by side.** Without M5 the class
returns.

---

## What remains: M1b, and why it was left

M1 wired the **cloud** call site only. That is the half that did nothing, so the
blast radius was zero-to-positive: handoffs, approvals, chain and stage
progression all start working where they previously did not, and the daemon path
— the one that works today — was not touched. M0's structural matrix still guards
it and still passes.

Consolidating the daemon call site is a pure refactor of working code, and it is
the single riskiest edit in the sprint. It also invalidates M0 as written: that
test asserts which branch of `executeTask` each effect sits in, and after the
extraction the effects live in the orchestrator instead. M0 would need retargeting
to a call-site guard in the same commit — which is exactly the kind of "change the
code and its test together" move that makes a regression invisible.

**Until M1b lands, the parity guarantee is one-directional.** M5 asserts every
daemon effect has an orchestrator counterpart, so consolidation cannot silently
drop behaviour. It does NOT assert the daemon routes through the orchestrator, so
a new effect added to the daemon alone would still diverge — more slowly, and now
detectably, but the class is not fully closed.

## Method

- RED-first for every arm **except M0**, which must pass on untouched `dev` (see above).
- `make test` between milestones; `make test-pi-extensions` unaffected.
- Commit straight to `dev` in the main checkout. Never branch there.
- Verify the **artefact, not the pipeline** — and after r1's lesson, **not the aggregate either**:
  read chain *detail*, never the list, whose `stage_count` under-reports (O1/M6).

## Rollout — verified trigger mechanics, not remembered ones

Checked per-trigger with `gcloud builds triggers describe` on 2026-09-03.
**`gcloud builds triggers list` prints the BRANCH and TAG columns EMPTY for every
trigger** — that empty output is what produced an unplanned production deploy on
2026-09-02, so `list` is never sufficient.

**The two repos promote differently, and this change touches both.**

| Repo | Trigger | Fires on | Builds |
|---|---|---|---|
| ailang | `ailang-core-dev` | branch `^dev$` | coordinator, agent-base, agent, agent-go, dashboard, mcp → **dev only** |
| ailang | `ailang-core-test-release` | **tag `^v.*`** | `cloudbuild-dev.yaml` |
| ailang | `ailang-core-release` | **tag `^v.*`** | `cloudbuild-release.yaml` |
| multivac | `ailang-multivac-{dev,test,prod}` | branch `^dev$`/`^test$`/`^prod$` | `cloudbuild.yaml` — **this is what builds agent-pi** |
| multivac | `ailang-multivac-config-{dev,test,prod}` | same branches, `config/**` + `terraform/**` | config + terraform apply |
| multivac | `promote-to-prod` | **manual only** | image copy |

Mark 2026-09-03: multivac is migrating to tags as well, but not yet stable —
until then it stays branch-led, and pushing its `prod` branch is a real
production deploy.

**So this sprint ships in two pieces:**

1. **Coordinator** — M0b, M1, M2, M4, M5 and the consuming half of M3. Built by
   `cloudbuild-dev.yaml`; already on **dev**. Reaching test/prod needs a `v*`
   **tag** on the ailang repo.
2. **Executor** — M3's producing half lives in `cmd/ailang/coordinator_cloud.go`,
   which runs inside **agent-pi**. `cloudbuild-dev.yaml:214` takes agent-pi's base
   *from the registry* and does not build it, so this needs a **multivac** build
   as well.

**Shipping half of M3 degrades safely but should not be left standing:** the
coordinator would look for `BaseCommit`/`HeadCommit` that the executor is not yet
sending, and record `diff_unavailable` on the approval — honest, but it is the
absence #921 was written about, and D2's per-edge auto-approval must not be
widened while it holds.

**Verify the deployed revision, not the build status.** A green pipeline has meant
"the thing I wanted is true" and been wrong three times in this programme.

## Live proof — the sprint's own success metric

Send a real message to `design-doc-creator`, approve the merge, and observe:

1. a `sprint-planner` task dispatched with the correct `parent_task_id` and `chain_id`;
2. the chain reading `design-doc-creator -> sprint-planner`;
3. the artefact landing in the correct repo.

**The obvious candidate for that first real message is this doc's own next milestone.** Mark noted it
2026-09-03: the design → sprint-plan → execute chain *is* the pipeline being repaired, so the first
thing it runs unsupervised can be its own follow-on work. That is a genuine end-to-end proof rather
than a synthetic one — and a fair test, because a failure would be immediately visible in a repo we
read every day.

## Out of scope

- Effect 10 (`ProcessStageCompletion`) — 12 non-idempotent comment POSTs (V29), inert for every
  message-sourced task (V29b). Stays daemon-only and unchanged. Own doc.
- Reply-to-sender routing for completions — a real gap, a different doc.
- The GCP backend's silent no-ops (V20) — named, guarded against by M5, not fixed.
- O1/M6, the chain list view's `stage_count` under-reporting — a read-path defect; **no milestone here
  depends on it**, and conflating it with write-path work is what produced r1's false headline.
- `work_tier` for pipeline agents (D4) — the sibling doc owns it.

## Success metric for the sprint as a whole

`agent_flow` is non-empty for at least one prod chain — the first time in the system's history — and
the full four-agent chain completes once on real work with no duplicate handoff, no double-counted
cost, and no approval regressed by a redelivery.
