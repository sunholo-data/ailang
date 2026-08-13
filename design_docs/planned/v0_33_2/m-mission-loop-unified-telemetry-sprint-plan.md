# Sprint Plan: M-MISSION-LOOP-UNIFIED-TELEMETRY

**Design doc**: [m-mission-loop-unified-telemetry.md](m-mission-loop-unified-telemetry.md)
**Sprint ID**: `M-MISSION-LOOP-UNIFIED-TELEMETRY`
**Created**: 2026-08-13
**Duration**: ~2 days (~16 hours)
**Risk**: Medium — M1 widens an attribute that Claude Code currently owns; M3 changes where mission telemetry is written
**Total LOC estimate**: ~500 (implementation ~260, tests ~240)

## Summary

**Goal**: One cloud-side record per mission iteration, queryable end to end across Anthropic, OpenAI,
OpenRouter and local providers.

**All three Design Freeze items are RATIFIED** (Mark, 2026-08-13): dual-write and node-generic, never
block when the cloud is unreachable, local analysis reads cloud opt-in. No pause point.

## Planning discovery — the estimates moved

Reading the call sites during planning changed two of the design doc's three estimates, both downward,
because the mechanisms already exist:

| Milestone | Doc estimate | Planned | Why |
|-----------|--------------|---------|-----|
| M1 linkage | ~6h / 180 LOC | ~6h / 190 LOC | unchanged |
| M2 accounting | ~8h / 200 LOC | ~6h / 190 LOC | `IterationStage` **already has** `TokensIn`/`TokensOut`/`CostUSD`; only `Status` is missing |
| M3 cloud routing | ~8h / 80 LOC | ~4h / 120 LOC | ONE hardcoded backend at `chains_post.go:59`, and the spool already wraps it |

**The spool discovery is the important one.** `chainsPostIterationCommand` already buffers on write
failure, loudly and boundedly. The ratified "never block" requirement is therefore **already
satisfied structurally** — M3 extends the existing wrapper to a new backend rather than building
fail-soft behaviour from scratch.

## Milestones

### M1_SESSION_KEYED_CHAIN_LINKAGE (~190 LOC, ~6h)

Make a Broadcast span resolve to its chain.

**Tasks:**
- [x] Register the correlation `session_id` as a `sessions` row bound to `chain_id` + `stage_id`
      when a mission/eval stage dispatches an OpenRouter call.
- [x] In `convertSpan`, resolve `chain_id` via the `sessions` table when `session.id` is present and
      no explicit `ailang.chain_id` was supplied.
- [x] `ailang.chain_id` keeps precedence — the new path only fires when it is absent.

**Acceptance criteria:**
- An OTLP/JSON span carrying `session.id` matching a seeded session resolves to that session's
  `chain_id` and `stage_id`.
- **NEGATIVE CONTROL — the one that matters**: a Claude Code span carrying `session.id` and NO chain
  resolves exactly as it does today. That attribute belongs to Claude Code; widening it is the
  primary regression risk in this sprint.
- An explicit `ailang.chain_id` WINS over a conflicting `session.id` lookup — asserted, not assumed.
- A `session.id` with no matching row leaves `chain_id` empty and does not error.
- `go test ./internal/observatory/...` green.

### M2_MISSION_STAGE_ACCOUNTING (~190 LOC, ~6h)

Two defects with **different owners**. A single "fix mission accounting" change would patch one and
miss the other, so they are separate tasks with separate criteria.

**Tasks:**
- [x] **Writer-side (status)**: `PostIteration` creates stages and never transitions them, so they
      keep `CreateStage`'s `StageStatusPending` default. Add a `Status` field to `IterationStage`
      and call `UpdateStageStatus`. Vocabulary available: `pending`, `running`,
      `awaiting_approval`, `completed`, `failed`.
- [x] **Caller-side (tokens)**: `IterationStage` ALREADY carries `TokensIn`/`TokensOut`;
      `UpdateStageMetrics` already receives them. The zeros come from the poster. Supply real token
      counts from the mission-control skill.
- [x] Aggregate stage cost/tokens into the chain total.

**Acceptance criteria:**
- A posted stage with a terminal status reads back as that status, NOT `pending`.
- **A failed stage reads back `failed`** — blanket-completing every stage would hide real failures
  and is explicitly wrong.
- A stage posted with tokens reads back with those tokens (the path is already wired; this asserts it).
- Chain total equals the sum of its stages — regression fixture built from the real iter-190 shape:
  4 stages, 3 providers, two stages with cost-but-zero-tokens.
- A post that omits `Status` still works and defaults to today's behaviour — the mission-control
  skill and the CLI ship at different times, so an unversioned payload must not break.

### M3_NODE_GENERIC_CLOUD_ROUTING (~120 LOC, ~4h)

**Tasks:**
- [x] Make the backend at `chains_post.go:59` selectable rather than hardcoded SQLite, reusing
      `internal/storage.NewBackends` which already resolves local/gcp/hybrid from `AILANG_STORAGE`.
- [x] Dual-write: local AND cloud, per the ratified decision.
- [x] Confirm the EXISTING spool covers a cloud write failure — extend, do not replace.
- [x] Opt-in remote read for analysis; local stays the default.

**Acceptance criteria:**
- With cloud configured, a posted iteration appears in BOTH stores.
- **Cloud unreachable → the post is spooled, a loud stderr notice fires, and the command exits 0.**
  Asserted against a deliberately-broken cloud config, since never-block is the ratified requirement.
- The spool stays bounded — its existing 100-entry / 1 MiB caps still hold with cloud failures added.
- With no cloud configured, behaviour is byte-identical to today.
- Node-generic: nothing hardcodes "the rig". The node is a parameter.

## Out of Scope

- Dashboard refactor for mission control (Mark's stated follow-up; depends on this data existing).
- `ailang messages` cloud-first (separate concern, own design).
- Backfilling historical mission chains.

## Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M1 widening `session.id` mis-attributes Claude Code spans | **High** | `ailang.chain_id` keeps precedence; negative-control test is an acceptance criterion, not a nice-to-have |
| M2 blanket-completes stages, hiding failures | **High** | A failed stage must read back `failed` — explicit criterion |
| M3 dual-write doubles the failure surface | Med | Existing bounded+loud spool covers it; never-block asserted against a broken config |
| Version skew between the skill and the CLI | Med | An unversioned payload must keep working — explicit criterion |
