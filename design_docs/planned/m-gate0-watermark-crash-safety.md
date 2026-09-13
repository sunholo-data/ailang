# Gate-0 Watermark Crash Safety

**Status**: Planned
**Target**: v0.38.x (loop-protocol change — no compiler/runtime surface)
**Priority**: P0 (silent loss of a human answer; observed twice)
**Estimated**: 1–2 days
**Dependencies**: None (builds on `scripts/mission_directives.sh` and the mission-control Gate-0 preflight skill)
**Source**: Issue #981, filed at Mark's request 2026-08-31: *"a fire that dies must not advance the watermark"*. Two of Mark's allowlisted answers were lost this way.

## Problem Statement

Gate 0 (`.claude/skills/mission-control/resources/gate-0-preflight.md`, step 6) currently instructs the controller:

> After triaging, write the newest processed `createdAt` to `"$WATERMARK"` … **before routing**, so a crashed iteration re-reads.

The intent was crash safety, but the ordering achieves the opposite. A fire that advances the watermark and then dies — before routing the directive into the iteration's work (ledger row update, unpark, queue pick) — leaves the next fire reading a watermark that says "already seen". The directive is re-fetched by **no one** and is silently and permanently skipped. This is exactly the failure Mark reported: two allowlisted human answers were lost.

**Current crash window (mark-before-route):**

```
fetch directives ──> triage ──> WRITE WATERMARK ──╳ CRASH ──> route (never happens)
                                                    next fire reads: "nothing new"
```

**Secondary defect — re-triage is not actually idempotent.** The skill's rationale claims "re-triage is idempotent", but idempotency only holds if a re-read directive is *deduplicated against work already done*. Nothing deduplicates: `scripts/mission_directives.sh` emits directive text ordered by fetch, carrying no comment ID, and the skill relies on `createdAt` + watermark ordering to avoid double-processing. Ordering is not identity. An unrotated watermark, a clock skew, or process-then-mark re-fetching an old directive (see below) makes the same comment appear "new" twice — iter-106 already demonstrated this class (a 5-day-stale literal watermark re-surfaced an actioned directive, which by rank would have re-run a landed sprint).

**Impact:** the human's only asynchronous channel into the loop silently drops answers — the worst failure mode this protocol can have, because it is invisible by construction (no error, no log line, no re-ask: the row stays OPEN but the `DECISIONS FOR MARK` report suppresses nothing while the answer sits unread).

## Goals

**Primary Goal:** No crash of a mission-control fire, at any point in Gate 0, can permanently lose or double-execute an allowlisted human directive.

**Success metrics:**
- Killing a fire at every step of the Gate-0 directive path (simulated in tests) never loses a directive: the next fire re-reads and processes it exactly once.
- Directive processing is keyed on comment identity, not timestamp ordering, so re-triage cannot double-execute a directive (no re-run landed sprint).
- One shared mechanism serves all four missions (v1 + three others) with no per-mission hand-rolled variant.
- A driver-suite test arm exercises the protocol (kill-at-each-step and dedupe assertions) and fails on regression.

## Proposed Design

### Ruling: process-then-mark, with comment-ID dedupe

Adopt **process-then-mark** as the crash-safety invariant:

> The watermark may only be advanced to a directive's `createdAt` **after the routing of that directive has fully landed** — the ledger row is committed (row updated `OPEN → RESOLVED` where applicable, decision recorded), the unpark/queue-pick consequence is recorded, and the iteration report draft acknowledges the comment. Until all of that, the watermark must not move past it.

This inverts the current order and shrinks the crash window to the same shape as any other idempotent replay: a crash *before* marking leaves the watermark behind, the next fire re-fetches the directive, and the dedupe key makes re-processing a no-op rather than a re-execution.

```
fetch directives ──> triage ──> route (ledger+report draft) ──> WRITE WATERMARK
                                   ╳ CRASH anywhere left of the mark
                                   next fire re-reads: directive re-appears,
                                   dedupe by comment id: already-routed = no-op
```

Why not the alternatives:
- **Two-phase pending-ack marker** (a pending file the next fire reconciles): strictly more state (a second marker file *per issue*, plus its lifecycle: write, reconcile, delete, and handling of a crash *between* pending-write and ack-write) and it re-solves the same problem with weaker guarantees. It is also the spool-file pattern the 2026-09-07 `_mc_notify` incident already measured as rotting in place. Rejected — process-then-mark with dedupe gives the same safety with zero new state.
- **Keep mark-before-route but re-triage idempotently**: requires the *same* comment-ID dedupe machinery as the ruling, but additionally demands that every routing action (ledger flips, unparks, sprint launches) be individually idempotent — a much larger surface than one Gate-0 check. The current protocol's rationale already assumed this and it was never built. Rejected as strictly harder for the same guarantee.

### What re-triage idempotency requires: dedupe by comment id

`scripts/mission_directives.sh` currently prints `author @ createdAt:\n body` blocks and the allowlist summary to stderr. It must additionally print, for each kept comment, a **stable comment id** (GitHub's `.id` field from the `gh --json comments` payload — globally unique, immutable, unaffected by clock skew, watermark staleness, or issue rotation).

Gate 0 then:

1. Fetches directives with ids.
2. For each directive, checks whether the comment id is already recorded as processed. **Processed-record location: the existing watermark-adjacent state only** — the single derived-path marker files already under `~/.ailang/state/`. The natural carrier is the watermark file's sibling: extend the *existing* watermark file format to carry `createdAt` plus a list of processed comment ids (a small, versioned record — see "No new state store" below). There is no second store, no pending file, no per-mission sidecar.
3. If the id is unknown → this is a new human answer: route it fully (per the decision-recording contract: update the ledger row in the same iteration, record dated evidence, prepare the report acknowledgment).
4. If the id is already present → re-read it for context, but take **no routing action**; if the watermark entry for it is missing (the crash case: routed but never marked), advance only the bookkeeping — record the id and the `createdAt` watermark — and note in the report that a crashed prior fire's directive was reconciled.
5. After all directives in the batch are routed (or confirmed already-routed), write the new watermark record: newest processed `createdAt` **and** the processed-id set (bounded: ids older than the watermark are redundant once the watermark covers them, so the file stays small — ids only coexist with a trailing watermark while a crash is outstanding).

**Rule the ordering explicitly in the skill text:** a directive is *processed* when its routing has landed, and the watermark write is the LAST Gate-0 directive action. The skill's existing line — "write … before routing, so a crashed iteration re-reads" — is replaced by this ruling; the parenthetical rationale ("re-triage is idempotent; dropping a human answer is not") is kept but is now backed by mechanism instead of assumption.

### All four missions share the protocol (no per-mission rolling)

The protocol lives in exactly two places, both already mission-independent:

- **`scripts/mission_directives.sh`** — the shared fetcher all missions already call (it derives everything from `MISSION_REPO`, `MISSION_DIRECTIVE_AUTHORS`, `--issue`), extended to emit comment ids and to offer the processed-check/watermark-record subcommands (`--mark <id> <createdAt>` style helpers), so no mission's controller ever hand-writes the jq/mark logic. The allowlist, self-direction guard, and set-but-empty refusal all remain in the script.
- **`.claude/skills/mission-control/resources/gate-0-preflight.md`** — the single shared preflight all four missions run, updated to state the process-then-mark ruling and the dedupe check in place of the current mark-before-route sentence.

The watermark paths stay derived per mission (`mission-${MISSION_GH_ISSUE}-last-seen`, including the V1 legacy namespacing exception in `tools/launchd/mission-control.sh`); only the *protocol* is shared, which is the existing doctrine (the 2026-09-01 attended-ruling clause (f) already notes "all four missions run this one skill file"). The change explicitly forbids missions from adding local jq fragments, per the same "the allowlist is enforced in code, not prose" precedent.

### No new state store beyond existing marker files

The watermark file's format gains a small, versioned extension (e.g., first line remains the `createdAt` watermark for backwards compatibility with any tooling that reads it raw; subsequent lines carry processed comment ids pending watermark coverage). If the file contains only a bare timestamp (legacy), Gate 0 treats all fetched directives as unprocessed and processes them per the contract — safe, because re-triage with the ruling above routes first and marks after, and the decision-recording contract already prevents double-asking. Rotation handling (weekly issue rotation, `-prev` catch in Gate 5) is unchanged: ids are issue-scoped by their presence alongside the issue-scoped watermark.

### Verification arm in the driver test suite

The driver test suite gains `tools/tests/test_gate0_watermark.sh` (alongside the existing `scripts/test_mission_answer.sh` pattern — mutation-proven arms, same conventions):

1. **Kill-at-each-step arm.** Drive a stubbed Gate-0 directive cycle (fixture issue JSON with two allowlisted comments; stub `gh`) and abort the sequence at each step — after fetch, after triage, after routing directive 1 but before mark, mid-mark. For every kill point, run the *next* fire against the state: exactly the unprocessed directives surface, processed ones are no-ops, no directive is ever lost, and no ledger action fires twice.
2. **Dedupe arm.** Present the same comment twice (stale watermark, clock-skewed `createdAt`, issue-rotation overlap): the second presentation takes no routing action.
3. **Legacy-state arm.** A bare-timestamp watermark file (no ids) upgrades cleanly and does not re-execute a landed directive given ledger evidence.
4. **Mutation arms** (one per guarantee, per the `test_mission_answer.sh` doctrine): removing the id emission, removing the processed-check, or reordering mark-before-route must each fail a specific arm.
5. **Multi-mission arm.** Two mission env profiles (v1 legacy paths + a namespaced mission) run the same protocol, proving the shared-script constraint — no hand-rolled variant passes.

## Non-Goals

- No change to the allowlist, self-direction guard, attended-ruling channel, or decision-recording contract in `gate-0-preflight.md` — those are untouched; only the ordering ruling and dedupe check change.
- No new state store, no pending/ack files, no database, no background reconciliation process.
- No compiler, runtime, or stdlib surface — this is a **loop-protocol change** confined to the mission-control skill text, `scripts/mission_directives.sh`, and a driver test.
- No change to Gate-5 rotation semantics or the `_mc_notify` spool mechanism (though that incident informed the rejection of the two-phase marker).

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Legacy bare-timestamp watermarks everywhere at rollout | Legacy-state arm + treating bare files as "all fetched directives unprocessed" (correct under the contract: routing is the ledger's job, and the ledger is the authority) |
| Watermark file grows with pending ids | Ids are dropped once the watermark's `createdAt` covers their `createdAt` — pending ids exist only in the crash window |
| Controller "paraphrases the pipeline" around the script | The skill text already forbids hand-rolled jq for directives; the mutation arms enforce the script is the only path |
| Crash *after* the ledger flip but before the report acknowledgment | The next fire re-sees the id as unmarked, reconciles bookkeeping only, and re-acknowledges in ITS report — Mark still sees the channel worked (no directive silently vanishes; worst case is a duplicated acknowledgment line, which is the correct failure direction) |

## Testing Strategy

- `tools/tests/test_gate0_watermark.sh` arms 1–5 above, mutation-proven (each arm names its sole killer).
- A dry-run checklist for the first real fire post-landing: read the watermark file (format upgrade), verify one iteration processes and marks normally, kill a fire manually mid-Gate-0 and verify the next fire reconciles.
- Existing `scripts/test_mission_answer.sh` continues to pass unchanged (proves no interference with the attended-ruling guard).

## Rollout Plan

1. Extend `mission_directives.sh` with id emission + mark/processed helpers (backwards compatible).
2. Flip the `gate-0-preflight.md` step-6 ordering to process-then-mark + dedupe check.
3. Add `tools/tests/test_gate0_watermark.sh`; wire into the driver suite.
4. Land as one atomic change; next-fire observation on all four missions before closing #981.