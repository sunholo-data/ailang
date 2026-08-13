# HANDOVER — M-MISSION-LOOP-UNIFIED-TELEMETRY (M2 + M3)

**Written**: 2026-08-13 · **Base commit**: `1d0e0837f` on `dev` (pushed)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json`
**Design doc**: [m-mission-loop-unified-telemetry.md](m-mission-loop-unified-telemetry.md)
**Sprint plan**: [m-mission-loop-unified-telemetry-sprint-plan.md](m-mission-loop-unified-telemetry-sprint-plan.md)

Start with `.claude/skills/sprint-executor/scripts/session_start.sh M-MISSION-LOOP-UNIFIED-TELEMETRY`.

## Status

| Milestone | State |
|-----------|-------|
| M1 session-keyed chain linkage | ✅ **PASS** — landed `56b449d01` |
| **M2 mission stage accounting** | ⬜ **PENDING — start here** |
| M3 node-generic cloud routing | ⬜ pending (depends on M2) |

**All three Design Freeze items are RATIFIED by Mark (2026-08-13).** Do not re-open them; they are
recorded verbatim with his reasoning in the sprint JSON's `design_freeze` block:

1. **Dual-write, NODE-GENERIC** — "this server, laptop, cloud, other nodes in the future". Nothing may
   hardcode "the rig"; the node is a parameter.
2. **Never block** when the cloud is unreachable — "at least until we harden availability".
3. **Local analysis reads cloud, OPT-IN** — default stays local so offline nodes stay first-class.

## M2 — the whole point of the sprint

A mission iteration spans Anthropic + OpenAI + OpenRouter and reports **`$0.0000` / 0 tokens** while
holding **$0.1077** across its own stages. Measured on `manual:mission:v1/iter-190`:

| Stage | Provider | Status | Cost | Tokens |
|-------|----------|--------|------|--------|
| controller | `quota:opus` | `pending` | — | — |
| designer | `quota:codex` | `pending` | — | — |
| quorum-r1 | openrouter ×2 | `pending` | $0.0570 | **0** |
| quorum-r2 | openrouter ×2 | `pending` | $0.0507 | **0** |

**Two defects, DIFFERENT owners.** Fixing "mission accounting" as one thing will patch one and miss
the other — that split is the milestone's main insight:

- **(a) Writer-side — status.** `PostIteration` (`internal/observatory/iteration_post.go:103-128`)
  calls `CreateStage` → `UpdateStageMetrics` → `UpdateStageEvalAssessment` and **never calls
  `UpdateStageStatus`**, so every stage keeps `CreateStage`'s `StageStatusPending` default
  (`store_chains.go:292`). `IterationStage` has **no `Status` field** — add one. Vocabulary:
  `pending`, `running`, `awaiting_approval`, `completed`, `failed` (`models_chains.go:65-69`).
- **(b) Caller-side — tokens.** `IterationStage` **already carries** `TokensIn`/`TokensOut`/`CostUSD`,
  and `iteration_post.go:116` **already passes them** to `UpdateStageMetrics`. The zeros come from
  the poster. Entry point is `ailang chains post-iteration`
  (`cmd/ailang/chains_post.go:44`), which reads a JSON payload; the supplier is the
  **mission-control skill**.

Then aggregate stage cost/tokens into the chain total.

### The criterion that blocks the shortcut

Setting every stage to `completed` would satisfy "no stage remains pending" **and hide real
failures**. A failed stage must read back `failed`. This is an acceptance criterion, not a
preference.

Also required: **version skew.** A payload omitting `Status` must keep working — the skill and the
CLI ship independently.

## M3 — smaller than the design doc says

The design doc estimated ~8h; planning found ~4h, because:

- `cmd/ailang/chains_post.go:59` hardcodes `observatory.NewSQLiteBackendFromPath(DefaultDatabasePath())`
  and is the **single** place the mission loop's backend is chosen.
- `chainsPostIterationCommand` **already wraps** that write in the bounded+loud spool
  (`internal/observatory/spool.go`: 100 entries / 1 MiB, warns on every buffering event, fail-soft).

So the ratified never-block requirement is **already satisfied structurally**. Extend the existing
wrapper to a second backend; do not build fail-soft behaviour from scratch, and do not invent a
policy — that spool exists specifically to answer a quorum objection about unbounded silent fallbacks.

Reuse `internal/storage.NewBackends`, which already resolves local/gcp/hybrid from `AILANG_STORAGE`.
Do not add a second selection mechanism.

## Traps that already cost time today

**1. Two functions carry an identical chain-extraction block.** `convertLogToSpan` (~line 445) and
`convertSpan` (~line 693) in `otlp_receiver.go`. My first M1 patch went into the logs path; the
direct-lookup test passed while the end-to-end test stayed red. **Broadcast exports traces, not
logs.** Always confirm which function you edited.

**2. Deployed observatories are FIRESTORE, not SQLite.** Both `ailang-dashboard` and
`ailang-dev-dashboard` run `AILANG_STORAGE=gcp`. `internal/observatory` migrations run **only** from
the SQLite paths, so anything written there silently does nothing to dev or prod. This already cost
one shipped-but-ineffective migration (`migrate_v18`) and a follow-up command
(`ailang observatory repair-ids`). See `.claude/rules/cloud-endpoints.md`.

**3. `make test` is NARROWER than CI.** `test-nightly-classifier`, `verify-examples`,
`check-file-sizes`, `check-boundaries`, `check-changelog`, `check-skills`, `check-golden-drift`,
`test-regression-guards`, `verify-no-shim` are separate CI steps. A green `make test` preceded two
red CI runs today. Run those targets before pushing.

**4. `dev` is a shared branch with concurrent agents.** Stage explicit paths — never `git add -A`.
An over-broad add swept another agent's uncommitted `models.yml` into an unrelated commit and caused
a merge conflict.

## After the sprint

Hand to `sprint-evaluator`. Then two follow-ups exist, both deliberately **out of scope here**:

- **Collaboration-hub re-baseline** (Mark asked for this as a SEPARATE thread):
  `design_docs/planned/v1_1_0/global-collaboration-hub.md` is the multi-node/cloud-native vision, but
  its six Design Freeze checkboxes are all unticked while at least three are resolved in shipped code
  (`internal/storage.Backends`, the Firestore stores, `AILANG_TOPIC_PREFIX`). It reads as
  "not started" when it is partly built. Re-baseline before writing anything new.
- **`ailang messages` cloud-first.** Local and cloud inboxes hold **different populations** — 20
  internal loop messages here, 55 external (`mcp-public`, package feedback) in cloud, neither visible
  to the other.

**Corrections Mark made that are NOT yet reflected in any doc**: evals-as-Cloud-Run-jobs already
works (just unwired on this server), and Discord already works for some public feedback. Missing
design docs ≠ missing features — do not infer capability from the doc corpus.
