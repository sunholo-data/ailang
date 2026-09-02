# M-COORDINATOR-EXECUTION-TRUST: make "a job ran" mean "work was attempted, and we know the outcome"

**Status**: Planned — not queued on any mission; this is attended, standalone critical-infrastructure work (Mark, 2026-09-02)
**Target**: v0.35.0
**Priority**: P0. Every autonomous lane downstream — package agents, cascade, feedback triage, cross-repo handoffs — dispatches through this path and has been producing nothing for six days without saying so.
**Estimated**: ~3 days (M1 ~0.5d, M2 ~0.5d, M3 ~1.5d, rollout ~0.5d)
**Dependencies**: none blocking. Builds directly on [m-message-plane-trust.md](m-message-plane-trust.md) M1–M3, which landed 2026-08-31 and fixed the layer below (delivery).

---

## Problem statement

[M-MESSAGE-PLANE-TRUST](m-message-plane-trust.md) fixed delivery. It worked: measured
2026-09-02, **every message addressed to a registered agent inbox produced a Cloud Run job
within 2–8 seconds**, four for four, including one full external round trip (a `mcp-public`
bug report → job → agent reply back to the reporter, 4m 32s end to end).

And in six days that path has changed **zero files**.

The layer below reports success at every step, so the layer above cannot tell "the agent did the
work" from "the agent was structurally prevented from doing any". Same pattern as its
predecessor, one level up: **the parts work, the outcome is unobservable.**

| Seam | Symptom | Root cause | Milestone |
|---|---|---|---|
| gate → executor | Agent reads the repo, diagnoses the bug correctly, writes nothing, exits 0 | The session-protocol gate is baked into the pi executor image at `/home/ailang/.pi` and arms in **every** workspace; the disarm is a tool call the pinned executor model never makes | **M1** |
| executor → completion | 4 of 4 recent tasks report `completed` with `changed_files: null` and no pushed branch | A no-op path added for acknowledge-only probes returns `nil`, so `publishCompletion("completed", …)` fires for both "nothing to do" and "structurally prevented" | **M2** |
| agent → model | A stalled provider kills the task outright — no second attempt | `ResolveModel` returns `chain[0]` and discards the rest; all 35 cloud agents carry a `model:` pin, so the chain never resolves at all | **M3** |

**The unifying defect is the same one the sibling doc named, moved up a layer.** Each of the
three reports success on the happy path and silence on the failure path — and this time the
silence is louder, because the *whole point* of the layer below was to make dispatch trustworthy.
We now have a plane that provably delivers, into an execution layer that provably does nothing,
and one number (`completed`) that says both are fine.

**Impact.** Six days of external bug reports from `mcp-public` and `aitana-platform` were
received, analysed correctly, and dropped. Store-wide the picture is worse than the recent
window suggests: **1,249 completion records, 16 marked `completed`, 6 that ever changed a
file** — four of those six touched only `AGENTS.md`. The last task to change real code was
2026-08-27.

---

## Verification Log

Every load-bearing claim below was checked against the code or against prod
(`ailang-multivac`) on 2026-09-02, not inferred from behaviour.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Dispatch works: a message to a registered inbox produces a job in seconds | `gcloud run jobs executions list` timestamps against message `created_at` | **Confirmed** — 4/4: +2s, +8s, +2s, +2s |
| V2 | The gate ships in the pi executor image, globally, not per-repo | `docker/Dockerfile.agent-pi` runs `ailang pi install`, which installs the embedded suite into `/home/ailang/.pi` (`cmd/ailang/pi_setup.go:312`) | **Confirmed** |
| V3 | The gate therefore applies to workspaces that never opted in | `gh api repos/sunholo-data/ailang-parse/contents/.pi/extensions` → **404**; the gate still fired in that clone | **Confirmed** |
| V4 | The gate blocks exactly `edit`, `write`, and non-allowlisted `bash` | `shouldBlock()` in `.pi/extensions/session-protocol-gate.ts` — read-only tools are never blocked | **Confirmed** |
| V5 | The only disarm is an explicit `session_protocol_ack` tool call | `pi.registerTool` + the `tool_call` interceptor; `acked` flips nowhere else | **Confirmed** |
| V6 | The executor model **met** the headless prerequisites but never called the ack | Job log for `task-4e415b46`: turn 2 `read: CLAUDE.md`, turn 3 `bash: ailang messages list --unread` — both conditions `headlessPrerequisitesMet()` checks — then turn 4 blocked again | **Confirmed** |
| V7 | There is **no** environment escape hatch in the gate | `grep -c "env" .pi/extensions/session-protocol-gate.ts` → **0** | **Confirmed (negative)** |
| V8 | A no-op run is deliberately reported as `completed` | `cmd/ailang/coordinator_cloud.go:465` — "no commits to push (agent made no changes)" returns `nil` error → `publishCompletion("completed", …)` at :205 | **Confirmed** |
| V9 | That path was added on purpose, for a real reason | Same comment: added 2026-08-26 for `task-90d5eeef`, an acknowledge-only probe, to stop orphan branches and `422 "No commits between…"` PR failures | **Confirmed — do not naively revert** |
| V10 | No test exercises the completion-status decision | `grep -rn "publishCompletion" --include="*_test.go"` → empty. Positive control, same grep shape: `injectAgentsMD` → 2 hits in `cmd/ailang/agents_md_exclude_test.go` | **Confirmed (negative), instrument proven** |
| V11 | `ResolveModel` discards the chain tail | `internal/coordinator/model_resolution.go:42` — `return chain[0].ModelName, nil`; its own comment: *"retry chains are a driver concern"* | **Confirmed** |
| V12 | The roles table already holds real chains | `internal/modelreg/models.yml:4620` — `executor: [gpt5-6-sol, opencode-or-deepseek-v4-flash]` | **Confirmed** |
| V13 | The chain never fires anyway: all cloud agents are pinned | Parsed the live `gs://ailang-multivac-ailang-config/config.yaml`: **35/35 carry `model:`, 0 carry `role:`** (30 × deepseek-v4-flash, 2 × gpt-5.6-sol, 1 each kimi-k3 / minimax-m3 / glm-5.3-flash) | **Confirmed** |
| V14 | A transport stall kills a task with no retry | `task-c429f9e0` completion payload: `"executor failed: pi task failed: pi idle for 3m0.006592232s mid-generation (no output)"`, status `failed` | **Confirmed** |
| V15 | The two gate copies are currently byte-identical, and a `make` target syncs them | `diff -q .pi/extensions/session-protocol-gate.ts cmd/ailang/pi_assets/session-protocol-gate.ts` → silent; `Makefile:368` defines `pi-assets` | **Confirmed** |
| V16 | The guard protecting the deleted routing table exists — **under a different name than the code cites** | `model_resolution.go:21` cites `TestCloudAgents_ResolveIdenticallyWithoutModelRouting`, which exists nowhere. The real guard is `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` (`agents_fixture_test.go:56`) | **Corrected** — fix the comment in M3 |
| V17 | Nothing in this doc can reach prod on the current images | Latest coordinator image built **2026-08-31T11:13:36Z**; prod rev `00064-4w6` still counts outcome notices in the backstop sweep, which `e0b12bf5f` skips | **Confirmed — see Rollout** |

**Not verified, deliberately deferred:** why executor GitHub writes return `422` on every REST
and GraphQL issue-create (token scopes read healthy). It cost the agent ~2 of its 15 minutes but
is not on this doc's critical path — see Non-Goals.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A task's reported status becomes a function of what the run actually produced, not of whether a gate happened to be armed. Today the same work yields `completed` either way. |
| A2: Replayability | +1 | M3 records **which chain link actually ran** on the completion payload; today a retry (when one is added) would be unattributable, and the banked model field would be a guess. |
| A3: Effect Legibility | +1 | M2 makes "this run mutated nothing" an explicit, queryable outcome instead of an absence. |
| A4: Explicit Authority | **0** | The most contestable row — see below. M1 removes a *ceremony* (the self-attested ack call) while keeping the *verifiable evidence* (`headlessPrerequisitesMet`). Net neutral on authority, and D2 arguably increases it. |
| A5: Bounded Verification | +1 | Each milestone lands a mutation arm that fails RED first; today the whole path has zero test coverage (V10). |
| A6: Safe Concurrency | 0 | No concurrency change. Retry in M3 is strictly sequential within one task. |
| A7: Machines First | +1 | The consumer of every one of these signals is a machine — the coordinator, the sweep, the dashboard. A status that cannot distinguish two outcomes is unanalysable by construction. |
| A8: Minimal Syntax | 0 | No language surface touched. |
| A9: Cost Visibility | +1 | M3's recorded chain link is what makes per-task cost attributable to the model that actually burned the tokens. |
| A10: Composability | +1 | M3 reuses the existing `roles:` chains rather than adding a second routing table — the thing M-MODEL-REGISTRY-SINGLE-SOURCE deleted. |
| A11: Structured Failure | +1 | A stalled provider becomes a typed, retried, recorded transport failure instead of a dead task. |
| A12: System Boundary | +1 | D2 makes the AILANG work-routing protocol stop at the AILANG repo boundary, where its authority actually ends. |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — retry order is the declared chain, in declared order.
- [x] A3 (Effects): no hidden side effects; M2 exists to *reveal* one.
- [x] A4 (Authority): no ambient access granted — argued below.
- [x] A7 (Machines First): the whole doc optimises machine analysability.

**On A4, stated plainly, because this is the row a reviewer should attack.** The gate's own
design doc ([m-dx-session-protocol-gate.md](v0_35_0/m-dx-session-protocol-gate.md)) claims
A4 +1 on the grounds that mutating tools unlock behind *"verifiable prerequisites (human
`ctx.ui.confirm` in TUI, V11; **observable protocol steps in session history** in headless mode,
V10) — not bare self-attestation."* In headless mode the authority therefore comes from
`headlessPrerequisitesMet()`, and the ack call is a *wrapper* around that check, not an
independent source of authority. M1 keeps the check and drops the wrapper. Nothing that was
verified stops being verified. **The interactive human-confirm path is untouched.** If a
reviewer disagrees, D1 option (B) preserves the ack call exactly and is a one-line
alternative — that is why it is a design-freeze item and not an implementer's choice.

---

## Goals

**Primary Goal:** A cloud task either does work, or says precisely why it could not — and a
transport failure gets a second lane before it becomes a dead task.

**Success Metrics:**
1. A pi executor run against a clean, non-AILANG workspace can write a file. (Today: cannot. No test asserts it either way.)
2. Zero tasks report `completed` while having produced no diff and no branch; that outcome has its own terminal state and appears in `ailang messages list --json`.
3. A task whose first model link stalls or 429s completes on the next link, and the completion records which link ran.
4. At least one real inbound report from `mcp-public` / `aitana-platform` reaches a pushed branch, end to end, unattended.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **D1** — How a headless session disarms: (A) auto-disarm when `headlessPrerequisitesMet()` is satisfied · (B) keep the ack call, inject an explicit instruction into the executor system prompt · (C) both | Determines whether the gate's A4 claim survives, and whether the fix depends on a flash-class model following an instruction — the exact thing that already failed | **human** | design | high |
| **D2** — Does the gate apply to workspaces outside the AILANG repo at all? | The AILANG work-routing protocol has no authority over `ailang-parse`. But the gate's `tool_call` handler is also the **single guaranteed observer for commit attribution** — scoping it off silently drops `Co-Authored-By` from every foreign-repo commit | **human** | design | high |
| **D3** — Shape of the no-op outcome: a new status value (`no_changes`) vs. a boolean on `completed` | Status is banked historical data. A new enum value ripples to the sweep, the dashboard, `messages health` and every existing query; a boolean does not, but is easier to ignore | **human** | design | high |
| **D4** — Where the retry chain lives: coordinator re-dispatch (new Cloud Run execution per link) vs. in-container retry (one execution, executor walks the chain) | Re-dispatch costs a fresh 24k-file clone and image pull per link (~60s); in-container retry is cheap but the coordinator cannot see which link ran unless the executor reports it | **human** | design | high |
| **D5** — What counts as a retryable transport failure | Too broad and we retry a real model refusal into a spent bucket; too narrow and V14's idle-stall stays fatal | agent | design | med |
| **D6** — Whether to convert the 35 `model:` pins to `role:` in the cloud config | Without it M3 ships correct code that never executes (V13) | agent | compile | med |

### Design Freeze

Every `high` row above. **sprint-executor must PAUSE if any is unchecked.**

- [ ] **D1** — headless disarm mechanism
- [ ] **D2** — gate scope outside the AILANG repo, *and* where commit attribution goes if the gate is scoped off
- [ ] **D3** — no-op outcome shape (new status vs. flag)
- [ ] **D4** — retry location: coordinator re-dispatch vs. in-container

**Recommendation, for the record:** D1 = (A), D2 = scope off but **first** split commit
attribution into its own extension so it survives, D3 = new status value (the banked-schema cost
is real but a boolean will be ignored by exactly the queries that matter), D4 = in-container,
with the executor reporting the link on the completion.

---

## Conflict Surface

Not strictly required — this touches no parser, typechecker or codegen path. Written anyway,
because quorum trigger **#2 fires** (M1 overrides machinery shared with every interactive pi
session in this repo), and because the sibling doc's worst bug came from a guard whose
correctness argument named a premise a later milestone quietly falsified.

**1. The gate is shared with Mark's own interactive sessions.**
`shouldBlock` and the `acked` flag serve both the TUI and the headless executor. Changing the
headless path must leave `ctx.hasUI` behaviour byte-identical.
*Must still work:* an interactive pi session in this repo still requires the human `confirm`
keypress before `edit`/`write` unlock.

**2. Commit attribution is coupled to the gate, by explicit design.**
From `session-protocol-gate.ts`: attribution *"lives in the GATE (not a separate extension)
because the gate's `tool_call` handler is the single guaranteed observer of every bash call — a
separate extension's handler can be skipped when the gate blocks (runner.js returns on first
block)."* **So D2's "just don't load the gate in foreign repos" silently removes
`Co-Authored-By` from every commit those agents make.** The reasoning that put attribution
inside the gate is sound and still holds; it simply did not anticipate the gate becoming
conditional. Split attribution out *before* scoping the gate, or scope only `shouldBlock` while
leaving the extension registered.

**3. There are two copies of the gate and a `make` target between them.**
`.pi/extensions/` is the source; `cmd/ailang/pi_assets/` is what `ailang pi install` bakes into
the image (`Makefile:368`, `make pi-assets`). They are byte-identical today (V15). Editing one
and shipping without the sync produces an executor image whose gate does not match the repo —
invisible, because both files exist and both look right.
*Must still work:* `make pi-assets` leaves the tree clean after a gate edit.

**4. The no-op path exists for a real caller (V9).**
`task-90d5eeef` was an acknowledge-only probe: correct behaviour is a clean completion with no
branch. A naive "no diff ⇒ failed" re-creates the orphan-branch and `422` failures that
2026-08-26 fixed.
*Must still work:* an acknowledge-only task completes cleanly, creates no branch, and is not
reported as an error.

**5. M3 must not resurrect the deleted routing table.**
`model_routing` was removed by M-MODEL-REGISTRY-SINGLE-SOURCE M7 precisely because it answered
"which model runs this role?" in a second place. Chain semantics belong in
`modelreg.ResolveRole`, consumed by `ResolveModel` — not in a new coordinator-side table.
*Must still work:* `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable`
(`agents_fixture_test.go:56`) — **note the name; the comment at `model_resolution.go:21` cites a
test that does not exist (V16), and that stale citation is itself a small instance of this
doc's thesis.**

**6. What deliberately changes.** A `completed` status stops meaning "the executor exited 0".
Any consumer treating `completed` as "work landed" was already wrong; after M2 it will be
*visibly* wrong, which is the point.

---

## Solution Design

### M1 — The gate must not silently disable the agent it is protecting

Headless runs disarm on **verified evidence**, not on a ceremony. When `ctx.hasUI` is false and
`headlessPrerequisitesMet()` is satisfied, `acked` flips without waiting for the tool call; the
tool stays registered so a model that *does* call it still works, and the interactive path is
untouched. Per D2, `shouldBlock` additionally goes inert when the workspace is not the AILANG
repo — with attribution split out first (Conflict Surface #2).

The failure this fixes is not "the model was too weak". The model satisfied both prerequisites
in three turns and then spent twelve minutes blocked. **The gate had all the evidence it needed
and required a password anyway.**

### M2 — A no-op is an outcome, not a silence

At `coordinator_cloud.go:465` and :475 the "no commits" path returns `nil`. Keep the branch and
PR suppression exactly as-is (V9) and change only what is *reported*: a run that produced no
diff and pushed no branch gets its own terminal state (D3), carried through
`pubsub.TaskCompletion` to `pubsub_completion_handler.go:172`.

The distinction that matters is **intent**: an acknowledge-only task is *supposed* to change
nothing; a bug-fix task that changes nothing has failed. The task type is already on the record,
so the classifier does not need to guess.

### M3 — The coordinator gets the fallback chain the mission loops already have

`ResolveModel` returns the resolved `[]ModelRef`, not `chain[0]`. The dispatcher walks it: on a
**transport-class** failure (idle-timeout, zero-byte completion, 429, provider 5xx) it advances
to the next link; on a model-class outcome (a real refusal, a wrong answer) it does not. The
completion records the link that ran. Per D6, the cloud config's `model:` pins become `role:`
wherever the pin is just "the default everyone got" — otherwise M3 ships correct and dead (V13).

This is deliberately the same posture the mission loops already run under, where a probe-failed
lane falls to the next entry and the fallback is recorded LOUDLY. The coordinator dispatches
unattended, so it needs this *more* than the driver does, not less — the comment claiming retry
chains are "a driver concern" was written when the driver was the only dispatcher.

### Files to Modify/Create

**Modified:**
- `.pi/extensions/session-protocol-gate.ts` — headless auto-disarm; workspace scoping; attribution split per D2 (~50 LOC)
- `cmd/ailang/pi_assets/session-protocol-gate.ts` — **regenerated by `make pi-assets`, never hand-edited** (V15)
- `cmd/ailang/coordinator_cloud.go` — no-op classification at the two return sites and the `publishCompletion` call (~30 LOC)
- `internal/coordinator/pubsub_completion_handler.go` — carry the new outcome (~10 LOC)
- `internal/coordinator/model_resolution.go` — return the chain; fix the V16 comment (~40 LOC)
- `internal/coordinator/daemon_tasks_exec.go` (:204), `daemon_tasks_exec_run.go` (:252) — chain-aware dispatch (~60 LOC)
- `gs://ailang-multivac-ailang-config/config.yaml` — `model:` → `role:` per D6 (config, not code)

**New:**
- `.pi/extensions/.session-gate-headless.test.ts` — M1 arms (~80 LOC)
- `cmd/ailang/coordinator_cloud_completion_test.go` — M2 arms; **the first test to touch this path at all** (V10) (~120 LOC)
- `internal/coordinator/model_chain_test.go` — M3 arms (~100 LOC)

---

## Testing Strategy

House rule: **every arm is verified RED before its fix.** A guard that has never failed is not a
guard, and this doc exists because three code paths had none.

| MU | Mutation | Arm |
|----|----------|-----|
| MU-1 | Re-require the ack call in headless mode | `TestHeadlessDisarmsOnVerifiedPrerequisites` |
| MU-2 | Drop the `ctx.hasUI` branch | `TestInteractiveStillRequiresHumanConfirm` |
| MU-3 | Arm the gate in a non-AILANG workspace | `TestGateInertOutsideOwningRepo` |
| MU-4 | Remove the attribution split | `TestAttributionSurvivesGateScoping` (Conflict Surface #2) |
| MU-5 | Report a zero-diff bug-fix task as `completed` | `TestNoDiffTaskIsNotCompleted` |
| MU-6 | Report an acknowledge-only probe as failed | `TestAcknowledgeOnlyTaskStillCompletesCleanly` (Conflict Surface #4) |
| MU-7 | Return `chain[0]` from `ResolveModel` | `TestResolveModelReturnsWholeChain` |
| MU-8 | Treat a model refusal as retryable | `TestOnlyTransportFailuresAdvanceTheChain` |
| MU-9 | Drop the link field from the completion | `TestCompletionRecordsWhichLinkRan` |

**Integration (the one that would have caught all of this):** a pi executor run against a clean
throwaway workspace that must produce a diff. Metric 1 above is exactly this test.

**Manual:** re-send one real `mcp-public` report and confirm a pushed branch.

---

## Success Criteria

- [ ] Pi executor writes a file in a clean non-AILANG workspace (MU-3, integration arm)
- [ ] Interactive TUI gate behaviour byte-identical (MU-2)
- [ ] Commit attribution present on a foreign-repo commit after D2 (MU-4)
- [ ] Zero-diff bug-fix task reports its own terminal state (MU-5); acknowledge-only probe unaffected (MU-6)
- [ ] A stalled first link completes on the second, with the link recorded (MU-7/8/9)
- [ ] `make pi-assets` leaves the tree clean
- [ ] One real inbound report reaches a pushed branch, unattended
- [ ] CHANGELOG.md updated; this doc moved to `implemented/v0_35_0/`

---

## Rollout — nothing here reaches prod without a rebuild

**Read before starting.** The latest coordinator image was built **2026-08-31T11:13:36Z** and
already predates a fix we believe is live: prod (`ailang-coordinator-00064-4w6`) still counts
outcome notices in the backstop sweep, which `e0b12bf5f` skips (V17). `AILANG_BACKSTOP_SWEEP=report`
is currently the only thing preventing a repeat of the 1,146-message amplification incident.

1. Rebuild **both** images — coordinator and `agent-executor-pi`. The gate lives in the executor image; M2/M3 live in the coordinator.
2. Verify the sweep's notice filter in prod logs before flipping `AILANG_BACKSTOP_SWEEP` back to `dispatch`.
3. Deploys here are flaky: revision `00065-spk` failed its startup probe on an image digest **identical** to the serving `00064`. Budget a retry; do not read one failed deploy as a bad build.

---

## Deferred Decisions

- **D5's exact retryable-failure predicate** — agent may choose, but it must be a named list, not a substring match. Prior art warns specifically against this: an `isRetryableError` that returns true on any `"429"` substring will retry into a spent quota bucket.
- **Whether the no-op classifier reads task type or a declared expectation** — agent may choose; both are on the record.
- **Retry budget (how many links, what total wall-clock)** — agent may choose within the existing 15m task timeout.

## Non-Goals

- **The remaining M-MESSAGE-PLANE-TRUST rows** — marking a dispatched message read, reconciling job success against task success at the sweep layer. Same family, that doc's scope.
- **Executor GitHub `422` on issue creation** — real (the agent burned ~2 of 15 minutes on it) but orthogonal; file separately.
- **`ANTHROPIC_API_KEY` unset on the coordinator** (feedback-gate classifier fail-closed) — ops, not design.
- **Re-filing the docparse redeploy message** — real, security-relevant, and tracked in that doc's "still owed" list.
- **OTLP unreachable from the executor** — observability gap, not an execution gap.
- **Making the executor model stronger.** The model was not the problem (V6). Prior art on this repo is unambiguous: this class has been a harness bug every time so far.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| M1 is read as "weakening a security gate" | High | A4 argued explicitly above; the verifiable prerequisite is *kept*, only the self-attested ceremony is dropped, and the human-confirm path is untouched. D1(B) is a one-line alternative if the argument is rejected. |
| D2 silently drops commit attribution | High | Conflict Surface #2 + MU-4. Split attribution **before** scoping the gate. |
| The two gate copies drift | Med | MU covered by `make pi-assets` in CI; V15 records they are in sync today. |
| M3 re-creates a second routing table | Med | Chain semantics stay in `modelreg`; `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` is the guard. |
| M3 ships correct and dead because every agent is pinned | High | D6 is a freeze-adjacent item and Metric 3 cannot pass without it. |
| A new status value breaks existing queries | Med | D3 is a human decision precisely because it is banked schema; enumerate consumers (sweep, dashboard, `messages health`) before choosing. |
| Fixing M1 reveals that agents now do *wrong* work unattended | Med | Every cloud task already opens a PR and never auto-merges — the always-PR posture is the existing containment, and it is unchanged here. |

## Quorum

**Attended session. Triggers #2 and #3 fire** — M1 overrides machinery shared with interactive
sessions (Conflict Surface #1), and D3 changes banked-data schema. Trigger #1 also fires: four
design-freeze items await a human ruling. **Recommendation: run the quorum before planning**,
after D1–D4 are ruled on, so reviewers see decided premises rather than open questions.

```bash
ailang design-quorum design_docs/planned/m-coordinator-execution-trust.md \
  --reviewers gpt5-6-sol,gemini-3-1-pro \
  --controller-verdict pass --controller-note "<in-session judgement>"
```

## Related Documents

- [m-message-plane-trust.md](m-message-plane-trust.md) — **the direct predecessor.** Fixed delivery; this doc fixes what happens after delivery. Its "still owed" list and this doc's Non-Goals are deliberately disjoint.
- [m-message-plane-fail-loud.md](../implemented/v0_35_0/m-message-plane-fail-loud.md) — the original "seams that report success while doing nothing" thesis.
- [v0_35_0/m-dx-session-protocol-gate.md](v0_35_0/m-dx-session-protocol-gate.md) — the gate's own design, including the A4 claim M1 must preserve.
- [v0_35_0/m-dx-pi-harness.md](../implemented/v0_35_0/m-dx-pi-harness.md) — pi extension distribution (V2's `ailang pi install` path).
- [v0_35_0/m-coordinator-route-authority-recovery.md](v0_35_0/m-coordinator-route-authority-recovery.md) (0.40) — adjacent coordinator work; distinct surface (routing authority, not execution outcome).
- [implemented/v0_22_0/m-coord-codex-executor.md](../implemented/v0_22_0/m-coord-codex-executor.md) (0.44) — executor-variant prior art for M3's provider/variant coupling.
- `docs/internal/message-plane-topology.md` — the map.

## Future Work

- Reconcile Cloud Run job success against task success at the sweep layer (the seam M2 makes visible but does not close).
- Per-task cost attribution keyed on the recorded chain link (M3 makes this possible; nothing consumes it yet).
- Extend the chain posture to the `codex` and `motoko` executor variants — M3 is scoped to the 32 `pi` agents.

---

**Document created**: 2026-09-02
**Last updated**: 2026-09-02
