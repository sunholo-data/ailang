# Sprint Plan — M-COORDINATOR-EXECUTION-TRUST (M1a + M2 + M3)

**Design doc**: [m-coordinator-execution-trust.md](m-coordinator-execution-trust.md)
**Split-out sibling (NOT in this sprint)**: [m-package-protocol-manifests.md](m-package-protocol-manifests.md)
**Created**: 2026-09-02 · attended session, main checkout, commits straight to `dev`
**Risk**: medium — three code areas, one of them a permission boundary
**Estimated**: ~670 LOC across 3 milestones + 1 rollout step

## Goal

A cloud task either does work, or says precisely why it could not — and a transport failure gets
a second lane before it becomes a dead task.

## Why the order is fixed

**M1a first, and not for convenience.** Until the gate stops blocking, nothing else in this sprint
is observable: M2's new status value would report on runs that were structurally prevented from
doing anything, and M3's retry would re-run a deadlock on a second model. M1a is what makes M2 and
M3 measurable rather than theoretical.

## Milestones

### M1a — Trusted work tier + built-in prerequisite floor (~230 LOC, ~1 day)

The permission boundary. Every quorum objection that survived three rounds was about letting
repo-published content into this decision; M1a contains none.

**Tasks**
1. `work_tier` closed enum (`tier1`/`tier2`) on the task; persisted; set by the coordinator from
   its **own registry** — inbox → *registered agent* → tier. Never from message content (V18),
   never from the sender-chosen inbox directly (V25).
2. Fail closed: missing, unknown, conflicting or model-supplied → `tier2`.
3. **Refuse tier 1 when `PushBranch` is set** (V24 — the direct-push path has no PR containment).
4. Gate: two built-in prerequisite sets. Generic floor (satisfiable with no `CLAUDE.md`), AILANG
   set (today's behaviour) selected by the agent config's `workspace` field.
5. Tier 1 auto-disarms on floor satisfaction; the ack tool stays registered and its call is still
   recorded. Tier 2 has no auto-path.
6. `make pi-assets` to sync `cmd/ailang/pi_assets/`.

**Acceptance**
- A pi executor run in a clean non-AILANG workspace writes a file.
- Interactive TUI confirm path byte-identical.
- Tier is not derivable from message content or from a sender-chosen inbox.
- An unknown/absent tier resolves to tier 2.
- A `PushBranch` dispatch never gets tier 1.

**Arms**: MU-1, MU-2, MU-3, MU-4, MU-5c, MU-5d, MU-5e, MU-7, plus a `PushBranch` refusal arm.

### M2 — `no_changes` terminal status (~180 LOC, ~0.5 day)

**Tasks**
1. Add `no_changes` to the completion status set; classify at `coordinator_cloud.go:465` and `:475`
   using the same trusted metadata as M1a — **not** content-derived `TaskType` (quorum round 2).
2. Add a case at **every** site in V19, starting with the terminality string-compare at
   `pubsub_completion_handler.go:85` (V20) and the worktree cleanup at `daemon_tasks_worktrees.go:103`.
3. Exhaustiveness test so the V19 list cannot go stale.

**Acceptance**
- Zero-diff bug-fix task reports `no_changes`; acknowledge-only probe still completes cleanly with
  no branch (Conflict Surface #4).
- `no_changes` is recognised as terminal and survives a persistence round-trip.
- Adding a status value with no case at a switch site fails the suite.

**Arms**: MU-8, MU-8b, MU-8c, MU-9.

### M3 — Two-tier retry with a durable cap (~260 LOC, ~1.5 days)

**Tasks**
1. `ResolveModel` returns the full `[]ModelRef`; fix the stale test citation at
   `model_resolution.go:21` (V16).
2. In-container chain walk on transport-class failure (named list, never a `"429"` substring).
3. `attempt_count` + `chain_link_index` persisted **on the task row in the status-transition
   write** (V22 — no counter exists today).
4. **Stale-task detector is the sole re-dispatcher** (V23 — four components can already decide a
   task is dead). Compare-and-set; losing writer logs and does nothing.
5. Completion records which link ran, in which tier.
6. Convert cloud-config `model:` pins to `role:` (D6) — otherwise M3 ships correct and dead (V13).

**Acceptance**
- A stalled first link completes on the second in-container, link recorded.
- Only infra-class failures re-dispatch; never more than 2 executions per task; the cap survives a
  coordinator restart.
- An unknowable task age never triggers re-dispatch.
- `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` still passes.

**Arms**: MU-10 … MU-14, MU-13b/c/d.

## Method

**Every arm is verified RED before its fix.** This sprint exists because three code paths had no
coverage at all (V10). An arm that has never failed is not an arm.

## Rollout (~0.5 day, after the code lands)

Rebuild **both** images — the gate is in `agent-executor-pi`, M2/M3 are in the coordinator. The
current coordinator image (2026-08-31T11:13:36Z) also predates `e0b12bf5f`, so this rebuild ships
that too. Verify the sweep's notice filter in prod, then flip `AILANG_BACKSTOP_SWEEP` back to
`dispatch`. Budget a deploy retry — `00065-spk` failed a startup probe on an identical digest.

**Then, and only then**, M-MESSAGE-PLANE-TRUST M4 becomes a five-minute job: re-file the docparse
redeploy message and watch it reach a pushed branch.

## Out of scope

Per-repo manifests (split doc) · the predecessor's M4/M5 · executor GitHub `422` ·
`ANTHROPIC_API_KEY` on the coordinator · OTLP from the executor.

## Success metric for the sprint as a whole

One real inbound report from `mcp-public` reaches a pushed branch, unattended. Nothing else
proves the layer works.
