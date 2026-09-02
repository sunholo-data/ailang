# M-COORDINATOR-EXECUTION-TRUST: make "a job ran" mean "work was attempted, and we know the outcome"

**Status**: Planned — attended, standalone critical-infrastructure work, deliberately off the mission queue (Mark, 2026-09-02).
**Scope**: **M1a + M2 + M3 (landed 2026-09-02) + M4 (queue visibility) + M5 (send-path misrouting) + M6 (pi extension self-collision) — M5 and M6 both found by the end-to-end test.** The general per-repo manifest system split to [M-PACKAGE-PROTOCOL-MANIFESTS](m-package-protocol-manifests.md) 2026-09-02 (Mark, attended) — three quorum rounds put every unresolved objection there, while M2/M3's closed and stayed closed.
**Rulings**: **All of D1–D4 ruled by Mark 2026-09-02 (attended)** — see Decisions. D2 REVERSED the author's recommendation and improved the design. Design freeze is CLOSED; sprint may proceed.
**Target**: v0.35.0
**Priority**: P0. Every autonomous lane downstream — package agents, cascade, feedback triage, cross-repo handoffs — dispatches through this path and has been producing nothing for six days without saying so.
**Estimated**: ~3 days for M1a-M3 (**landed** `79082a16f`, `b1ade7e6e`, `28002af1e`) + ~0.5d for M4 + rollout
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
| plugin → pi startup | Any task whose workspace IS the AILANG repo dies before turn 1 | the globally installed pi suite collides with the repo's own `.pi/extensions/` on tool names (V30) | **M6** |
| gate → executor | Agent reads the repo, diagnoses the bug correctly, writes nothing, exits 0 | The session-protocol gate is baked into the pi executor image at `/home/ailang/.pi` and arms in **every** workspace; the disarm is a tool call the pinned executor model never makes | **M1** |
| executor → completion | 4 of 4 recent tasks report `completed` with `changed_files: null` and no pushed branch | A no-op path added for acknowledge-only probes returns `nil`, so `publishCompletion("completed", …)` fires for both "nothing to do" and "structurally prevented" | **M2** |
| send → inbox | A message is delivered to a DIFFERENT inbox than addressed, and says `✓ Message sent` | a flag value starting with `-` is refused by `normalizeArgsForFlags`, shifting every later token (V29) | **M5** |
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
| **V18** | **The tier source is untrusted.** A closed `TaskType` enum exists (`provider.go:11-21`, incl. `TaskTypeUnknown`) and is on the Task (`store.go:16`) — **but it is assigned by `classifyTaskType(task.Content)` (`analyzer.go:31,65`), a substring match over sender-controlled message text.** The bug-fix branch is tested FIRST with the broadest list: `bug, fix, error, crash, broken, issue, problem, fail, wrong` | read the classifier | **REFUTES the doc's original M1.** "The task type is already on the record" was true; "so the classifier need not guess" was false. Nearly every inbound message classifies tier 1 |
| **V19** | **The real consumers of completion status are Go switch sites, not the four reporting surfaces the doc named.** `pubsub_completion_handler.go` :85 (terminality), :95 (`Success:`), :111 (`switch`), :169, :186; `observatory_sync.go` :66/:136/:138; `daemon_tasks_worktrees.go:103` (orphan-worktree cleanup); `event_handler.go:238`; `store_sqlite_queries.go` :185/:229 | `grep -rn "\.Status\b"` across `internal/coordinator`, `internal/pubsub`, `cmd/ailang` | **Corrects M2.** The named list was wrong *in kind* |
| **V20** | The terminality check at `pubsub_completion_handler.go:85` is a **literal string comparison** — `task.Status == "completed" \|\| "failed" \|\| "cancelled"` — not an enum switch | read the site | A new `no_changes` would **not** register as terminal and the task could be re-processed. The exact silent failure D3 was chosen to avoid |
| **V21** | There is **no central prerequisite floor** in the gate today — `headlessPrerequisitesMet` is a single hard-coded pair with nothing above it to inherit from | read `.pi/extensions/session-protocol-gate.ts` | **Confirmed (negative).** So "repo-declared protocol" as originally written had nothing to be bounded by |
| **V22** | **No attempt counter exists on the task.** `grep -n "Attempt\|Retries\|RetryCount" internal/coordinator/store.go` → empty | read the Task struct | **Confirmed (negative).** D4's 2-execution cap has nowhere to persist; it must be added, not assumed |
| **V23** | **Four independent components can already decide a task is dead**: `stale_task_detector.go`, `pubsub_completion_handler.go`, `daemon_stranded_approvals.go`, `daemon_tasks_worktrees.go` | grep across `internal/coordinator` for status-mutating recovery paths | Any two of them gaining re-dispatch would breach the cap — hence the single-owner rule in M3 |
| **V24** | **The "always-PR containment" claim in Risks was FALSE as stated.** A direct-push path exists: `AILANG_PUSH_BRANCH` (`coordinator_cloud.go:220,294-296`) makes the agent work on the cloned branch with **no coordinator branch and no PR**. It is set by the dispatcher from `params.PushBranch` (`dispatch/cloudrun/dispatcher.go:206`), so it is coordinator-controlled, not sender-controlled — but containment is a property of that parameter, not of the system | read both sites | **REFUTES the risk-row mitigation.** Tier 1 must not be granted on any path where `PushBranch` is set |
| **V25** | **A sender chooses their own inbox.** `to_inbox` is a plain field on `ailang messages send <inbox> …`, persisted verbatim (`messaging/inbox.go:183`) | read the send path | **Confirms gpt5-6-sol.** Tier may NOT map from inbox. The trusted link is inbox → *registered agent* (coordinator config) → tier; the sender controls the first arrow only |
| **V26** | **"Rebuild the images" is TWO pipelines, not one.** `cloudbuild-dev.yaml` in this repo builds coordinator, agent-base, agent, agent-go, agent-motoko — **no `agent-pi`**. The pi image is built by the *multivac* repo's `cloudbuild.yaml` (`-f /workspace/ailang/docker/Dockerfile.agent-pi`, cloning ailang `--branch dev`) | read both build configs; `gcloud artifacts docker images list` confirms agent-pi exists and is built elsewhere | **CORRECTS the first Rollout draft.** M1a lives in the pi image and 32 of 35 agents are provider=pi, so an ailang-repo build alone ships M1a to nobody |
| **V27** | **`ailang-multivac` is PROD, and the automatic builds do not touch it.** `ailang-core-dev` runs with `_TARGET_PROJECT=ailang-multivac-dev`, `_PREFIX=ailang-dev`. A full parallel plane exists there (`ailang-dev-coordinator`, `ailang-dev-agent-executor-*`) with its own Firestore and an `ailang-dev` topic prefix. Prod updates via the `ailang-multivac-prod` trigger or `promote-to-prod` ("copy images, no rebuild") | `gcloud builds describe` substitutions; `gcloud run services/jobs list --project ailang-multivac-dev` | **CORRECTS the first Rollout draft**, which read as though a dev build reached the plane this doc audits. It does not — and the dev plane is therefore a safe place to test |
| **V28** | **CORRECTION — the ack is INTERMITTENT, not never.** This doc said the pinned executor model "never makes" the `session_protocol_ack` call. Measured on the dev plane 2026-09-02 (`task-f8213acd`, same model `deepseek-v4-flash-0731`): the model hit the gate, then **called `session_protocol_ack` in turn 1** and unlocked. `task-4e415b46` on 08-31 never called it across twelve minutes | dev-plane executor logs, both runs | **Falsifies the doc's own wording.** The gate is a *flaky* blocker, not a deterministic one — which is worse for diagnosis, because the failure then looks like model variance rather than a harness defect. M1a's floor is still the right fix: it removes the coin-flip |
| **V29** | **NEW DEFECT, same class as this doc's thesis: a message can be delivered to the WRONG INBOX while reporting success.** `normalizeArgsForFlags` (`cmd/ailang/messages_send.go:448`) refuses to consume a flag value beginning with `-`, so the value falls through to positional and every later token shifts. Minimal repro: `ailang messages send diag-argparse "body" --title "--help is inconsistent" --from "diag-sender"` → the message lands in inbox **`diag-sender`**, `from_agent` falls back to `cli`, `title` becomes the literal string `--from`, and the CLI prints `✓ Message sent`. Control with a title not starting with `-` routes correctly | reproduced 5×, isolated to the dash-prefixed flag VALUE (not the body) | **Found by the end-to-end test this doc motivated.** Any report whose title starts with `--help`, `-v`, `--json`… is silently misrouted |
| **V30** | **BLOCKING: every pi executor task whose workspace is the AILANG repo dies at startup.** `ailang pi install` materialises the suite into `~/.pi/agent/extensions/` — its own help text says *"global — every repo on this machine"* — and the AILANG repo ALSO ships `.pi/extensions/` with the same files. pi loads both and refuses: `Failed to load extension ".../ailang-lsp-lite.ts": Tool "ailang_check" conflicts with /workspace/task-X/.pi/extensions/ailang-lsp-lite.ts`, then `exit status 1` | dev plane 2026-09-02: `task-6825b3fc` and `task-48a365bb` (workspace `sunholo-data/ailang`) both failed identically; `task-f8213acd` (workspace `sunholo-data/ailang-packages`) did not | **Structural, and pre-existing** — the Dockerfile has always run `ailang pi install`. It went unseen because prod pi tasks target package repos. It takes out `design-doc-creator`, `sprint-planner`, `sprint-evaluator` and `coordinator` — the whole AILANG-repo pipeline on the pi provider |
| **V31** | **The sanctioned promote path CANNOT ship the pi executor to prod.** `cloudbuild-promote.yaml` in the multivac repo mentions `agent-pi` **zero times**. `_SERVICE=agent` promotes `agent:latest` and redeploys only `ailang-agent-executor`; even `_SERVICE=all` covers just coordinator, agent, dashboard, mcp, docparse, billing and website-builder. The codex, opencode, gemini, eval and **pi** variants have no promote case at all | read `cloudbuild-promote.yaml`; `grep -c agent-pi` → 0; promote run `2a66a519` failed at `validate` for a missing `_SERVICE` | **32 of the 35 prod agents run the pi variant.** The only route to prod for their image is a full `ailang-multivac-prod` build, which rebuilds and redeploys *everything* — so a one-image fix is not expressible as a one-image deploy |

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
| A4: Explicit Authority | **+1** | Raised from 0 after D1/D2 were ruled. The tiered model makes authority *stronger*, not weaker: tier-2 (feature/semantics) work now has **no** auto-path and requires an explicit recorded ack, where today a model that skips the ack simply does nothing and the clause is never tested. D2 extends the gate to every repo the executor touches. See the honest cost below. |
| A5: Bounded Verification | +1 | Each milestone lands a mutation arm that fails RED first; today the whole path has zero test coverage (V10). |
| A6: Safe Concurrency | 0 | No concurrency change. Retry in M3 is strictly sequential within one task. |
| A7: Machines First | +1 | The consumer of every one of these signals is a machine — the coordinator, the sweep, the dashboard. A status that cannot distinguish two outcomes is unanalysable by construction. |
| A8: Minimal Syntax | 0 | No language surface touched. |
| A9: Cost Visibility | +1 | M3's recorded chain link is what makes per-task cost attributable to the model that actually burned the tokens. |
| A10: Composability | +1 | M3 reuses the existing `roles:` chains rather than adding a second routing table — the thing M-MODEL-REGISTRY-SINGLE-SOURCE deleted. |
| A11: Structured Failure | +1 | A stalled provider becomes a typed, retried, recorded transport failure instead of a dead task. |
| A12: System Boundary | +1 | D2 makes the AILANG work-routing protocol stop at the AILANG repo boundary, where its authority actually ends. |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — retry order is the declared chain, in declared order.
- [x] A3 (Effects): no hidden side effects; M2 exists to *reveal* one.
- [x] A4 (Authority): no ambient access granted — argued below.
- [x] A7 (Machines First): the whole doc optimises machine analysability.

**On A4, stated plainly, because this is the row a reviewer should attack.** The honest cost of
auto-disarm is not the verification — `headlessPrerequisitesMet()` still runs, and the gate's own
design doc grounds headless authority in *"observable protocol steps in session history"*, not in
the ack call. The cost is the **assertion**: today an ack in the session log means a model
consciously claimed it had done the protocol; under tier-1 auto-disarm, mutation can occur
without that claim.

Three things follow, and the design answers each:

1. **The prerequisites are substring checks, not comprehension checks.** They prove a `read` of a
   path containing `CLAUDE.md` happened and a bash command containing `ailang messages` ran. They
   do not prove anything was understood. **This is equally true before and after auto-disarm** —
   removing the ceremony does not weaken a check that was already weak. If we want a stronger
   gate, the lever is the prerequisite set, not the ack.
2. **The "feature/semantics work needs a design doc + sprint plan" clause is unverifiable today
   and unenforced in practice.** Nothing in `headlessPrerequisitesMet()` checks it; the ack only
   made a model *assert* it. Tier 2 is what actually closes this: no auto-path, explicit ack
   required, and the evidence checked rather than asserted.
3. **Under D2 the gate is now the only thing standing between a cloud agent and a foreign repo.**
   That makes it more load-bearing, not less — which is the argument for strengthening the
   prerequisites (Future Work) rather than for keeping a ceremony that a flash-class model
   demonstrably will not perform.

The interactive human-confirm path is untouched throughout.

---

## Goals

**Primary Goal:** A cloud task either does work, or says precisely why it could not — and a
transport failure gets a second lane before it becomes a dead task.

### The goal behind the goal (Mark, attended 2026-09-02)

> *"I am trying to get to a state where you using `ailang messages` to fix background tasks is
> the preferred way."*

That is what this doc is ultimately for, and it changes what "done" means: not "the seams are
fixed" but **"a person reaches for the message plane instead of an interactive session."** Three
conditions have to hold for that, and naming them is what stops the work stopping one short:

| # | Condition | State |
|---|---|---|
| 1 | A message reliably becomes a running job | **Done** — measured 2–8s, 4/4 (see the seam table) |
| 2 | A finished job says honestly what it did | **M2**, once deployed. Today `completed` still means "exited 0" |
| 3 | **You can see the queue without asking an agent to go and look** | **Not addressed — M4 below** |

Condition 3 is the one this doc did not originally have, and it is the one that decides
*preference*. Everything measured in this session was measured by a human-driven session running
`gcloud` and `ailang` by hand. A plane you must ask an agent to inspect is not a plane you delegate
to; it is one more thing to supervise. **M1a/M2/M3 make the loop correct. M4 is what makes it
trustable without supervision.**

**Success Metrics:**
1. A pi executor run against a clean, non-AILANG workspace can write a file. (Today: cannot. No test asserts it either way.)
2. Zero tasks report `completed` while having produced no diff and no branch; that outcome has its own terminal state and appears in `ailang messages list --json`.
3. A task whose first model link stalls or 429s completes on the next link, and the completion records which link ran.
4. At least one real inbound report reaches a pushed branch, end to end, unattended — **proved on the dev plane first** (V27).
5. **One command answers "what is in flight, what is stuck, and what did the fleet do overnight"**, and its "should be zero" number is actually reachable.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **D1** — How a session disarms the gate | Determines whether the permission system stays a real, recorded approval or degrades to a checkbox | **human** | design | high |
| **D2** — Does the gate apply outside the AILANG repo? | Decides whether this is an AILANG-repo convention or the permission layer for the whole package ecosystem | **human** | design | high |
| **D3** — Shape of the no-op outcome: new status value vs. flag on `completed` | Status is banked historical data; every consumer that switches on it is affected | **human** | design | high |
| **D4** — Where retry lives: in-container vs. coordinator re-dispatch | Re-dispatch costs a fresh 24k-file clone and image pull per attempt (~60s) | **human** | design | high |
| **D5** — What counts as a retryable transport failure | Too broad and we retry a refusal into a spent bucket; too narrow and the idle-stall stays fatal | agent | design | med |
| **D6** — Convert the 35 `model:` pins to `role:` | Without it M3 ships correct code that never executes (V13) | agent | compile | med |

### Rulings — 2026-09-02, Mark, attended

**D1 — RULED: tiered. The ack stays; auto-disarm is a floor, not a replacement.**
Mark's steer: *"I want the permission system to be acked and approved by models such as
yourself in a mission loop or in a session like this."* So the ack is the mechanism, not a
ceremony to delete — and the fix for the executor deadlock is a floor beneath it, not a
substitute for it.

| Tier | Work | Disarm path |
|---|---|---|
| **1 — routine** (bug-fix, triage, reply, docs) | Mutation allowed once `headlessPrerequisitesMet()` is satisfied. The ack is still offered and still **recorded when made** — a capable model's explicit approval remains a first-class, logged event. | auto, on verified evidence |
| **2 — feature / semantics** | No auto-path. Requires an explicit ack **plus** the design-doc + sprint-plan evidence the gate's own description already demands. A model that cannot make the call cannot do tier-2 work. | ack only |

This is what makes the ack meaningful rather than decorative: it stops being the thing that
blocks a flash model from fixing a typo, and starts being the thing that gates feature work.

**D2 — RULED, REVERSING the author's recommendation: the gate applies everywhere.**
Mark: *"gates apply outside of ailang repo — we want a general ailang message system for our
packages as well."* Two consequences, one good and one new:

- **Good:** Conflict Surface #2 dissolves. Commit attribution stays coupled to the gate, nothing
  needs splitting out, and the highest-rated risk in this doc is deleted outright.
- **New requirement:** the protocol must become **per-repo declared, not AILANG-hardcoded.**
  `headlessPrerequisitesMet()` currently demands a `CLAUDE.md` read and an `ailang messages`
  call. That is an AILANG-repo convention being enforced on repos that never agreed to it. A
  package repo declares its own protocol (default: the AILANG one) and the gate reads that.
  **This changes what the doc is:** not a bug fix to an executor, but the permission layer for
  the AILANG package ecosystem.

**D3 — RULED: the new status value (`no_changes`).** Mark took the fail-safe direction. A
consumer that has never heard of this outcome reads it as *not* `completed` and therefore as
not-success — which surfaces the problem instead of hiding it. The cost is real and accepted:
every consumer that switches on status (the backstop sweep, the dashboard, `messages health`,
the banked task corpus) needs a case, and M2 must enumerate them rather than discover them.

**D4 — RULED: two-tier retry, cheapest first.**
Mark: *"retry in container first, try one re-dispatch if they fail."* This maps exactly onto the
failure taxonomy, better than either option originally offered:

| Failure class | Example | Handled by | Cost |
|---|---|---|---|
| model / provider | idle stall (V14), 429, provider 5xx | **in-container**: walk the chain inside the existing 15m timeout | ~0 — no re-clone, no image pull |
| infrastructure | container died, OOM, Cloud Run preemption | **one re-dispatch**, on the next chain link | full clone + pull (~60s) |

In-container retry is *impossible by definition* for the infra class — the container is gone —
which is precisely why the second tier exists. **Hard cap: 2 Cloud Run executions per task.**

### Design Freeze

Every `high` row above. **sprint-executor must PAUSE if any is unchecked.**

- [x] **D1** — tiered: ack stays as the recorded approval; auto-disarm is the tier-1 floor
- [x] **D2** — gate applies everywhere; protocol becomes per-repo declared
- [x] **D3** — new status value `no_changes` (fails safe; enumerate consumers, do not discover them)
- [x] **D4** — in-container chain walk, then at most one re-dispatch; cap 2 executions/task

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

**2. ~~Commit attribution is coupled to the gate~~ — DISSOLVED by the D2 ruling.**
The original risk: attribution *"lives in the GATE (not a separate extension) because the gate's
`tool_call` handler is the single guaranteed observer of every bash call"*, so scoping the gate
off in foreign repos would have silently dropped `Co-Authored-By` from those commits. **D2 keeps
the gate loaded everywhere, so the coupling is never broken and nothing needs splitting out.**
Recorded rather than deleted: it is the reason a future "just disable it over there" shortcut is
not available, and MU-4 stays as the guard.

**2b. The protocol itself is AILANG-specific and now runs everywhere (new, from D2).**
`headlessPrerequisitesMet()` hard-codes a `CLAUDE.md` read and an `ailang messages` call. Under
D2 that convention is enforced on package repos that never adopted it — and a repo with no
`CLAUDE.md` cannot satisfy it at all.

**⚠ Corrected after quorum round 1 — the first revision left a contradiction here that would have
guaranteed Success Metric 1 fails.** This section said a repo declaring nothing "gets today's
AILANG behaviour, unchanged", while M1 said a foreign workspace with no manifest and no
`CLAUDE.md` still satisfies the floor. Both cannot hold: an unconfigured foreign repo inheriting
the AILANG default blocks forever looking for a file it does not have. **The AILANG prerequisite
set is NOT the default — it is what the AILANG repo's own manifest declares.** The default is the
generic floor.

*Must still work:* (a) the AILANG repo, which ships a manifest declaring today's set, behaves
exactly as it does now; (b) a foreign repo with no manifest satisfies the generic floor and can
write.

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

**5b. The retry owner overlaps four existing recovery paths (added after quorum round 1).**
`stale_task_detector.go`, `pubsub_completion_handler.go`, `daemon_stranded_approvals.go` and
`daemon_tasks_worktrees.go` can each already move a task toward a terminal state (V23). The
original Conflict Surface covered model routing and omitted this overlap entirely — the reviewer
was right that "who may decide an execution is missing" is the load-bearing question, not "which
model runs next".
*Must still work:* exactly one component re-dispatches; the other three keep their current
behaviour and gain nothing.

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

### M1a — A permission system with a trusted tier and a built-in floor (D1 + D2)

Two changes, one to each axis.

**Tiering (D1), with a trusted authority boundary.** Quorum round 0 blocked the first draft here
and was right; V18 then showed the hole is worse than the objection stated.

> **The first draft said "the task type is already on the record, so the classifier need not
> guess." The field is on the record. It is also computed by `classifyTaskType()`, a substring
> match over sender-controlled message content, whose bug-fix branch is tested first against
> `bug, fix, error, crash, broken, issue, problem, fail, wrong`. Almost every inbound message
> classifies as tier 1 — so "feature work has no auto-path" would have been vacuous, and any
> sender could have chosen their own tier by word choice.**

The tier is therefore **not** read from `classifyTaskType`:

1. **Source.** `work_tier` is a closed enum set by the **coordinator at dispatch time** from
   trusted routing metadata (agent config + inbox), carried on the task, and **not derivable from
   message content**. The executor and the model can read it and can neither set nor downgrade it.
2. **Fail-closed.** Missing, unknown, conflicting, or model-supplied values resolve to **tier 2**
   and prohibit mutation pending an explicit ack plus verified design evidence. `classifyTaskType`
   keeps its existing job — shaping the system prompt (`meta_prompt.go:19`) — and gains no
   authority it does not already have.
3. **Floor.** The gate always enforces a **centrally defined minimum prerequisite set**. There is
   none today (V21), so M1 creates it.

**Generalisation (D2) — scoped to M1a, with the manifest system split out.** After three quorum
rounds landed every hard objection on repo-published manifests, that half is now
[M-PACKAGE-PROTOCOL-MANIFESTS](m-package-protocol-manifests.md). **M1a contains no
repo-controlled content at all**, which removes the entire "a repo declares an empty protocol and
unlocks itself" surface from this sprint.

M1a ships **two built-in prerequisite sets**, selected by trusted coordinator metadata:

| Set | Applies when | Contents |
|---|---|---|
| **Generic floor** | default, and for every non-AILANG workspace | satisfiable with no `CLAUDE.md` and no AILANG-specific file — this is what makes Success Metric 1 reachable in a clean foreign repo |
| **AILANG set** | the workspace is the AILANG repo, per the agent config's `workspace` field | today's behaviour exactly: a `CLAUDE.md` read plus an `ailang messages` call |

Selection is by **trusted coordinator metadata, never by a file in the clone** — so no repo can
influence which set governs it. The gate still loads in every workspace (D2 holds), and
`shouldBlock` stops being AILANG-specific without anything becoming repo-controlled.

**Tier 1 is refused outright on any dispatch where `PushBranch` is set (V24).** The direct-push
path has no PR containment, so it never gets the auto-disarm floor — it gets tier 2.

The failure M1 fixes remains the one measured in V6 — the model satisfied both prerequisites in
three turns and stayed blocked for twelve minutes. Tier 1 is that floor. Tier 2, now that its
input is trustworthy, is where the ack finally does work it was never doing.

### M2 — A no-op is an outcome, not a silence (D3 still open)

At `coordinator_cloud.go:465` and :475 the "no commits" path returns `nil`. Keep the branch and
PR suppression exactly as-is (V9) and change only what is *reported*. The distinction that
matters is **intent**: an acknowledge-only task is *supposed* to change nothing; a bug-fix task
that changes nothing has failed.

**⚠ That intent may NOT come from `TaskType` (quorum round 2, gpt5-6-sol).** V18 proved
`classifyTaskType` is sender-controlled, and M2 inherited the same mistake M1 was rewritten to
remove: a sender writing "fix" in a message would classify their own task's expectations. The
expectation must come from the same trusted dispatch metadata that carries `work_tier` — one
authority boundary, used by both milestones.

**D3, reframed as one question:** when a consumer that has never heard of this outcome reads it,
which way should it be wrong?

| Option | An old consumer sees | Fails |
|---|---|---|
| New status value `no_changes` | *not* `completed` → counts as not-success | **safe** |
| Boolean flag on `completed` | `completed` → counts as success | **unsafe** |

**RULED: the new status value `no_changes`** — it fails in the direction that surfaces the
problem rather than hiding it, which is the entire thesis of this doc and its predecessor.

Quorum round 0 blocked on the consumer list and was right: the first draft named four *reporting
surfaces* (sweep, dashboard, `messages health`, banked corpus) and asserted they were exhaustive
without proof. **V19 shows the real consumers are in-process Go switch sites**, and the list was
wrong in kind. Enumerated, each needs a case:

| Site | What it decides | Risk if missed |
|---|---|---|
| `pubsub_completion_handler.go:85` | **terminality** — and it is a literal string compare, not an enum switch (V20) | `no_changes` not recognised as terminal → task re-processed. **The exact silent failure D3 was chosen to avoid** |
| `pubsub_completion_handler.go:95` | `Success: completion.Status == "completed"` | a no-op counts as neither success nor failure — intended, but must be deliberate |
| `pubsub_completion_handler.go:111` | the completion `switch` | unhandled branch |
| `pubsub_completion_handler.go:169,186` | banked payload + notification title | reporting only |
| `observatory_sync.go:66,136,138` | `convertTaskStatus`, agent assignment status | dashboard mis-state |
| `daemon_tasks_worktrees.go:103` | **orphaned-worktree cleanup keyed on status** | worktrees leak for every no-op task |
| `event_handler.go:238`, `store_sqlite_queries.go:185,229` | event + persistence round-trip | value fails to survive a reload |

**AC:** a test enumerates the switch sites and fails when a new status value is added without a
case at each — otherwise this list goes stale the first time someone adds a status.

### M3 — The fallback chain, cheapest tier first (D4)

`ResolveModel` returns the resolved `[]ModelRef`, not `chain[0]`. Retry is two-tier, per Mark's
ruling, matched to the failure taxonomy:

- **In-container (model/provider class).** The executor walks the chain inside the existing 15m
  timeout on an idle-stall, 429 or provider 5xx (D5 names the predicate — a named list, never a
  substring match on `"429"`, which would retry into a spent bucket). Costs nothing extra: no
  re-clone, no image pull.
- **One re-dispatch (infrastructure class).** If the container itself dies — OOM, preemption, a
  job execution that never reports — the coordinator re-dispatches **once**, on the next chain
  link. In-container retry is impossible by definition here, which is exactly why this tier
  exists. **Hard cap: 2 Cloud Run executions per task.**

**The cap needs an owner and a place to live (quorum round 1, gpt5-6-sol — accepted).** A cap that
is only a sentence is not a cap: **V22 confirms no attempt counter exists on the task today**, and
**V23 lists four independent paths that can already decide a task is dead.** Two of them
re-dispatching would exceed the cap and duplicate work nondeterministically. So:

1. **Single owner.** The **stale-task detector** is the sole component permitted to re-dispatch.
   The backstop sweep, the completion handler, the stranded-approval sweep and worktree cleanup
   observe and report; none of them re-dispatch. Stated as a rule because today none of them
   re-dispatch *by accident of not having the feature*, not by design.
2. **Durable state.** `attempt_count` and `chain_link_index` are persisted **on the task row**, in
   the same write that transitions its status, so the cap survives a coordinator restart and a
   scale-to-zero. A task at `attempt_count >= 2` is never re-dispatched by anything.
3. **A bounded definition of dead.** An execution is dead when it has published no completion and
   its age exceeds the task timeout plus a declared grace. `getTaskAge` refuses to act on an
   unknowable age (fixed in `e0b12bf5f`) — that refusal is the precondition this depends on, and
   MU-13b asserts it still holds.
4. **Serialized.** Re-dispatch is a compare-and-set on `attempt_count`; a loser logs and does
   nothing. Not a lock — a losing writer must be observable.

This moves reconciliation from the Non-Goals list *for the retry path specifically*. The general
job-success-vs-task-success seam stays with the predecessor doc; **owning the cap does not mean
owning that.**

The completion records which link ran, in which tier. Per D6, the cloud config's `model:` pins
become `role:` wherever the pin is just "the default everyone got" — otherwise M3 ships correct
and dead (V13). Chain semantics stay in `modelreg.ResolveRole`; nothing here re-creates the
routing table M-MODEL-REGISTRY-SINGLE-SOURCE deleted.

### M4 — The queue is visible without an agent going to look (condition 3)

Scoped deliberately small: this is the difference between a loop that works and a loop you
*prefer*, not a new subsystem.

1. **Make `messages health`'s headline number reachable.** It reports "routable and never
   dispatched"; every one of the 7 it flagged this session had in fact been dispatched, because a
   dispatch never marks its message read. A number that can never be zero stops being read — and
   this one is the plane's only self-report.
2. **A fleet activity view.** "In the last N hours: messages in, tasks created, executions,
   outcomes by status (now including `no_changes`), and anything stuck past its timeout." Every
   input already exists — this session assembled exactly this by hand from `gcloud run jobs
   executions list`, coordinator logs and Firestore. The work is joining them, not gathering them.
3. **Say which plane you are looking at.** `messages health` already prints its store; it must
   also distinguish dev from prod, because V27 shows the two are separate and confusing them is
   how a "nothing happened" conclusion gets drawn about the wrong plane.

**Not in scope:** a dashboard, alerting, or anything requiring a deploy of its own. A CLI that
prints the truth is the whole ask.

### M5 — A message reaches the inbox it was addressed to (V29)

Found by the end-to-end test, and it belongs in this doc rather than a separate one because it is
the same defect the whole document is about, one layer earlier: **the send path reports success
and does something else.** Every other milestone here assumes the message arrived where it was
sent; this is the assumption underneath them.

`normalizeArgsForFlags` guards against consuming the *next flag* as a value by refusing any token
starting with `-`. That guard is wrong for a legitimate value that happens to start with a dash,
and dash-leading values are common in exactly this system — bug reports about flags.

**Fix:** decide whether the next token is a value by checking it against the *known flag set*, not
by a bare `-` prefix. Equally important: **an unconsumed flag must be an error, not a silent
shift.** The routing outcome must never be a side effect of a parse that half-failed.

**Arms:** a title beginning with `--help` routes to the addressed inbox, keeps its `--from`, and
keeps its title; a genuinely missing flag value errors instead of shifting; the existing
non-dash cases stay byte-identical.

### M6 — The globally installed pi suite must not collide with the repo that owns it (V30)

The most consequential thing the end-to-end test found, and the one that makes M1a unobservable
on this repo: **a pi task cloning `sunholo-data/ailang` never reaches turn 1.** The suite is
installed globally into `~/.pi/agent/extensions/` so that it applies in every workspace — which
is exactly what D2 wanted — and the AILANG repo carries the same files in `.pi/extensions/`, so pi
sees two registrations of `ailang_check` and exits 1.

Note the shape: **this is the same global-install decision that makes the gate apply everywhere.**
D2 got the goal right; the mechanism has a self-collision nobody had exercised, because prod pi
tasks all target package repos.

**Options, in preference order:**
1. Make the loader prefer the repo-local copy and skip the global one on a name clash — a
   workspace that ships its own suite is stating a preference, and honouring it is strictly more
   correct than failing.
2. Have `ailang pi install` skip files the workspace already provides.
3. Detect the AILANG repo at execute-job time and not inject the global suite. Weakest: it special-
   cases one repo, and every package repo that later adds `.pi/extensions/` hits the same wall.

**Arm:** a pi executor run whose workspace is the AILANG repo reaches turn 1. There is no such
test today, which is precisely why four agents were dead on this provider without anyone knowing.

### Files to Modify/Create

**Modified:**
- `.pi/extensions/session-protocol-gate.ts` — work tiering; tier-1 auto-disarm; repo-declared prerequisite sets (~70 LOC). **No attribution split — D2 keeps the gate loaded everywhere.**
- `cmd/ailang/pi_assets/session-protocol-gate.ts` — **regenerated by `make pi-assets`, never hand-edited** (V15)
- `cmd/ailang/coordinator_cloud.go` — no-op classification at the two return sites and the `publishCompletion` call (~30 LOC)
- `internal/coordinator/pubsub_completion_handler.go` — carry the new outcome (~10 LOC)
- `internal/coordinator/model_resolution.go` — return the chain; fix the V16 comment (~40 LOC)
- `internal/coordinator/daemon_tasks_exec.go` (:204), `daemon_tasks_exec_run.go` (:252) — chain-aware dispatch; the one infra-class re-dispatch and its 2-execution cap (~80 LOC)
- `gs://ailang-multivac-ailang-config/config.yaml` — `model:` → `role:` per D6 (config, not code)

**New:**
- `.pi/extensions/.session-gate-tiers.test.ts` — M1 arms MU-1..MU-7 (~140 LOC)
- `cmd/ailang/coordinator_cloud_completion_test.go` — M2 arms; **the first test to touch this path at all** (V10) (~120 LOC)
- `internal/coordinator/model_chain_test.go` — M3 arms (~100 LOC)

---

## Testing Strategy

House rule: **every arm is verified RED before its fix.** A guard that has never failed is not a
guard, and this doc exists because three code paths had none.

| MU | Mutation | Arm |
|----|----------|-----|
| MU-1 | Re-require the ack for tier-1 work | `TestTier1DisarmsOnVerifiedPrerequisites` |
| MU-2 | Drop the `ctx.hasUI` branch | `TestInteractiveStillRequiresHumanConfirm` |
| MU-3 | Allow tier-2 work through the tier-1 auto-path | `TestFeatureWorkHasNoAutoPath` (D1's load-bearing arm) |
| MU-4 | Remove the ack recording when a model does call it | `TestAckIsStillRecordedWhenMade` (keeps the audit line real) |
| MU-5 | Let a repo manifest remove or weaken a floor prerequisite | `TestManifestMayOnlyAddToTheFloor` (**quorum round 0, gpt5-6-sol**) |
| MU-5b | Fall back to the default on a malformed/unsupported manifest | `TestMalformedManifestBlocksMutation` (no silent fallback) |
| MU-5c | Make the floor require an AILANG-specific file | `TestFloorSatisfiableInAForeignRepo` |
| MU-5d | Derive `work_tier` from `classifyTaskType` / message content | `TestTierIsNotDerivableFromMessageContent` (**V18 — the load-bearing arm**) |
| MU-5e | Default a missing/unknown/model-supplied tier to tier 1 | `TestUnknownTierFailsClosedToTier2` |
| MU-6 | Make the gate inert outside the AILANG repo | `TestGateAppliesInForeignWorkspace` (D2 — the reversed decision) |
| MU-7 | Detach attribution from the gate handler | `TestAttributionObservesEveryBashCall` (Conflict Surface #2) |
| MU-8 | Report a zero-diff bug-fix task as `completed` | `TestNoDiffTaskIsNotCompleted` |
| MU-8b | Omit `no_changes` from the terminality check at :85 | `TestNoChangesIsTerminal` (**V20**) |
| MU-8c | Add a status value with no case at a switch site | `TestEveryStatusConsumerHandlesEveryStatus` (keeps V19's list from going stale) |
| MU-9 | Report an acknowledge-only probe as failed | `TestAcknowledgeOnlyTaskStillCompletesCleanly` (Conflict Surface #4) |
| MU-10 | Return `chain[0]` from `ResolveModel` | `TestResolveModelReturnsWholeChain` |
| MU-11 | Treat a model refusal as retryable | `TestOnlyTransportFailuresAdvanceTheChain` (D5) |
| MU-12 | Re-dispatch on a model-class failure | `TestOnlyInfraFailuresReDispatch` (D4 tier boundary) |
| MU-13 | Allow a third Cloud Run execution | `TestTwoExecutionCapIsHard` (D4 cost cap) |
| MU-13b | Let an unknowable task age trigger re-dispatch | `TestUnknownAgeNeverReDispatches` (**depends on `e0b12bf5f`; asserts that refusal still holds**) |
| MU-13c | Give a second component the ability to re-dispatch | `TestStaleDetectorIsTheSoleReDispatcher` (**quorum round 1**) |
| MU-13d | Keep `attempt_count` in memory instead of on the task row | `TestCapSurvivesCoordinatorRestart` (**V22**) |
| MU-13e | Make the AILANG prerequisite set the default for unconfigured repos | `TestForeignRepoWithNoManifestCanWrite` (**quorum round 1, gemini — the contradiction**) |
| MU-14 | Drop the link field from the completion | `TestCompletionRecordsWhichLinkRan` |

**Integration (the one that would have caught all of this):** a pi executor run against a clean
throwaway workspace that must produce a diff. Metric 1 above is exactly this test.

**Manual:** re-send one real `mcp-public` report and confirm a pushed branch.

---

## Success Criteria

- [ ] Pi executor writes a file in a clean non-AILANG workspace (integration arm)
- [ ] Tier-2 (feature/semantics) work still has no auto-path (MU-3)
- [ ] An explicit ack is still recorded when a capable model makes one (MU-4)
- [ ] Interactive TUI gate behaviour byte-identical (MU-2)
- [ ] The gate applies in a foreign workspace, with that repo's declared protocol (MU-5, MU-6)
- [ ] Commit attribution present on a foreign-repo commit (MU-7)
- [ ] Zero-diff bug-fix task reports its own terminal state (MU-8); acknowledge-only probe unaffected (MU-9)
- [ ] A stalled first link completes on the second in-container, with the link recorded (MU-10/11/14)
- [ ] Only infra-class failures re-dispatch, and never more than twice total (MU-12/13)
- [ ] `make pi-assets` leaves the tree clean
- [ ] One real inbound report reaches a pushed branch, unattended
- [ ] CHANGELOG.md updated; this doc moved to `implemented/v0_35_0/`

---

## Predecessor status — M-MESSAGE-PLANE-TRUST, re-measured 2026-09-02

Asked directly: is it done, and should it be finished first? **Not done, and no — but its open
items now have answers, and one of its milestones has been silently completed.**

| Milestone | State 2026-09-02 | Evidence |
|---|---|---|
| M1 backstop sweep | **Landed**, running in `report` mode in prod | `91716b092`; sweep log lines every 5m |
| M2 terminal-state reconciliation | **Landed and working** | **Zero** coordinator/task chains remain `active` — the 92-chain corpus M5 was written to clean up is already closed |
| M3 health surface | **Landed** | `ailang messages health` answers against prod |
| M4 re-announce the stranded | **Open, and now unblocked — but must sequence AFTER this doc's M1** | see below |
| M5 cleanup | **Open, and its scope is WRONG** | see below |

**The measurement M4 and M5 were gated on has now run.** That doc's condition was: *"Run the
sweep in `report` mode for a day; its count is what decides whether `dispatch` is worth enabling
and whether re-announcing the stranded messages is safe."* It has run since 08-31. The count is
**7**, of which 4 are outcome notices (which `e0b12bf5f` excludes and which are unread-and-routable
by construction) and 3 were in fact dispatched. **Push dropped nothing.** The premise behind M4's
"don't dispatch into an 88% failure rate" is also gone.

**So M4 is safe — and still should not run yet.** Re-announcing real work into an execution layer
that cannot write files reproduces exactly the silence that made it stranded in the first place.
**M4 sequences after this doc's M1**, and then becomes a five-minute job.

**M5's scope needs correcting, not executing.** Its target — 92 stuck-active coordinator chains —
is already closed by M2. But `chains active` still reaches back to 2026-08-28, and every one of
those rows is an `eval_suite` (16) or `manual:mission` (4) chain. **AC6 still fails, for a
different reason than M5 assumed:** those producers have no closer at all, where the stale-task
detector covers coordinator jobs only. That is a new finding about a different code path, not a
cleanup chore.

**Recommendation: do not stop to finish it.** Two things belong to it and land naturally here:
(a) the coordinator rebuild in Rollout below ships its already-written `e0b12bf5f`, and
(b) M5 gets rewritten with the corrected finding. Its AC3 and AC6 both still fail, so it stays in
`planned/` — moving it to `implemented/` today would bank a false completion, which is the exact
failure mode both documents exist to prevent.

## Rollout — two pipelines, two planes

**The first draft of this section was wrong twice, and the corrections are V26/V27.** It read as
though one rebuild reached the plane this doc audits. It does not.

| Change | Built by | Lands in | Reaches prod how |
|---|---|---|---|
| M2, M3 (coordinator) | `ailang-core-dev`, on push to this repo's `dev` — **automatic** | `ailang-multivac-dev` | promote |
| **M1a (gate, `agent-pi`)** | **multivac repo's `cloudbuild.yaml`** — not this repo's pipeline at all | `ailang-multivac-dev` (dev trigger) | promote |

**Consequence worth stating plainly: an ailang-repo build alone ships M1a to nobody**, because the
gate lives in the pi image and 32 of the 35 cloud agents are `provider: pi`.

**`ailang-multivac` is production.** The automatic dev builds do not touch it. Prod moves via the
`ailang-multivac-prod` trigger or `promote-to-prod` ("copy images, no rebuild"). **That is a
deliberate human step and this doc does not authorise it.**

**And promote cannot carry this change (V31).** `promote-to-prod` has no case for `agent-pi` —
nor for codex, opencode, gemini or eval. It can promote the coordinator (M2/M3) but not the gate
(M1a), even though 32 of 35 prod agents run pi. Shipping M1a to prod today therefore means a full
`ailang-multivac-prod` build: every image, every service. **That asymmetry is itself worth fixing
— a promote path that omits the variant most agents use will keep forcing all-or-nothing prod
deploys.**

### Order

1. Push to this repo's `dev` — coordinator (M2/M3) builds and deploys to the dev plane by itself.
2. Run the multivac **dev** build — this is what carries M1a into `agent-pi`.
3. **Test on the dev plane, not prod.** `ailang-dev-coordinator` reads `ailang-multivac-dev`
   Firestore with topic prefix `ailang-dev`, so a probe there cannot touch the prod inbox or
   dispatch against real package repos. This is the correct place to prove the loop end to end.
4. Only then promote, and only on a human's call. The current prod coordinator image also
   predates `e0b12bf5f`, so the promote ships that too — verify the sweep's notice filter in prod
   logs before flipping `AILANG_BACKSTOP_SWEEP` back to `dispatch`.
5. Budget a deploy retry: revision `00065-spk` failed a startup probe on an image digest
   identical to the serving `00064`.

## Deferred Decisions

- **D5's exact retryable-failure predicate** — agent may choose, but it must be a named list, not a substring match. Prior art warns specifically against this: an `isRetryableError` that returns true on any `"429"` substring will retry into a spent quota bucket.
- **Whether the no-op classifier reads task type or a declared expectation** — agent may choose; both are on the record.
- **Retry budget (how many links, what total wall-clock)** — agent may choose within the existing 15m task timeout.

## Non-Goals

- **The remaining M-MESSAGE-PLANE-TRUST rows** — reconciling job success against task success at the sweep layer; its own M4/M5. Same family, that doc's scope — see Predecessor status. **Marking a dispatched message read has MOVED here**, into this doc's M4: it is not a tidy-up, it is what makes the plane's only self-report legible.
- **Executor GitHub `422` on issue creation** — real (the agent burned ~2 of 15 minutes on it) but orthogonal; file separately.
- **`ANTHROPIC_API_KEY` unset on the coordinator** (feedback-gate classifier fail-closed) — ops, not design.
- **Re-filing the docparse redeploy message** — real, security-relevant, and tracked in that doc's "still owed" list.
- **OTLP unreachable from the executor** — observability gap, not an execution gap.
- **Making the executor model stronger.** The model was not the problem (V6). Prior art on this repo is unambiguous: this class has been a harness bug every time so far.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| M1 is read as "weakening a security gate" | High | A4 argued explicitly above; the verifiable prerequisite is *kept*, only the self-attested ceremony is dropped, and the human-confirm path is untouched. D1(B) is a one-line alternative if the argument is rejected. |
| ~~D2 silently drops commit attribution~~ | — | **Deleted by the D2 ruling** — the gate stays loaded everywhere, so the coupling is never broken. MU-7 keeps it that way. |
| D2 enforces AILANG conventions on repos that never adopted them | High | Repo-declared prerequisite sets, AILANG as default (Conflict Surface #2b, MU-5). |
| Tier-1 auto-disarm is read as removing the permission system | Med | It adds a floor beneath the ack, and tier 2 enforces a clause that is unenforced today (A4 rationale). MU-3 is the arm. |
| **A sender picks their own tier by word choice** | **High** | **Found by quorum round 0 + V18.** `work_tier` comes from trusted dispatch metadata, never from message content; fail-closed to tier 2. MU-5d/MU-5e. |
| **A repo declares an empty protocol and unlocks itself** | **High** | **Found by quorum round 0.** Manifests may only ADD to a central floor; malformed manifests block rather than fall back. MU-5/MU-5b. |
| A new status value is added later with no case at a switch site | Med | MU-8c asserts exhaustiveness rather than trusting V19's list to stay current. |
| Re-dispatch doubles cost on a bad day | Med | Hard cap of 2 executions per task, infra-class only (MU-12, MU-13). |
| The two gate copies drift | Med | MU covered by `make pi-assets` in CI; V15 records they are in sync today. |
| M3 re-creates a second routing table | Med | Chain semantics stay in `modelreg`; `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` is the guard. |
| M3 ships correct and dead because every agent is pinned | High | D6 is a freeze-adjacent item and Metric 3 cannot pass without it. |
| A new status value breaks existing queries | Med | D3 is a human decision precisely because it is banked schema; enumerate consumers (sweep, dashboard, `messages health`) before choosing. |
| Fixing M1 reveals that agents now do *wrong* work unattended | **High** | ~~Every cloud task already opens a PR and never auto-merges~~ — **FALSE, see V24.** A direct-push path exists (`AILANG_PUSH_BRANCH`). It is coordinator-set, not sender-set, but containment is a property of that parameter rather than of the system. **Tier 1 must be refused on any dispatch where `PushBranch` is set.** Found by quorum round 2 (gemini-3-1-pro). |

## Quorum

**Attended session. Triggers #1, #2 and #3 fire** — M1 overrides machinery shared with interactive
sessions (Conflict Surface #1), D3 changes banked-data schema, and a design-freeze item remains
open. **D1, D2 and D4 are now ruled**, so reviewers would see decided premises. **Round 0 — 2026-09-02, BLOCKED. Both reviewers rejected; both were right.**
Artifact: `.ailang/state/mission-quorum/m-coordinator-execution-trust-2026-09-02T06-10-11Z.json`
($0.087, 2/2 present, no absences).

| Reviewer | Objection | Disposition |
|---|---|---|
| `gpt5-6-sol` | M1 defines no trusted authority boundary for the work tier, and a repo-declared protocol could weaken or empty the gate | **ACCEPTED IN FULL.** V18 then showed it was worse than stated: the tier field is computed by substring match over sender-controlled content. M1 rewritten — trusted dispatch metadata, fail-closed to tier 2, central floor a manifest may only extend |
| `gemini-3-1-pro` | The four named status consumers were asserted exhaustive without a Verification Log row | **ACCEPTED IN FULL.** V19/V20 enumerate the real sites; the list was wrong in kind, and `:85` is a string compare that would not have recognised `no_changes` as terminal |

Neither objection was argued. The doc was technically exempt from the Conflict Surface rule and
wrote one anyway; the same instinct applies here — a reject that lands on substance is worth more
than the round it costs. **Round 1 is the re-quorum-ONCE guardrail: if it blocks again, this hands
over rather than grinding.**

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
- **Strengthen the prerequisites.** They are substring checks: a `read` of a path containing `CLAUDE.md`, a bash command containing `ailang messages`. They prove the commands were issued, not that anything was understood. This is the real lever on gate strength — and under D2 it now guards every package repo, so it matters more than it did.
- **Close `eval_suite` and `manual:mission` chains.** Neither producer has a closer; AC6 of the predecessor fails on them (see Predecessor status).

---

**Document created**: 2026-09-02
**Last updated**: 2026-09-02
