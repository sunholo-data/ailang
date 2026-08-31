# M-FEATURE-PROVENANCE-CHAINS: one chain per feature, so a design doc, its sprint plan, its execution and its evaluation are one queryable object

**Status**: PROPOSED — grain ratified by Mark 2026-08-31 (attended: "one chain per feature"). Mechanism below is the draft.
**Target**: next sprint window.
**Priority**: P1 for the observability spine — `m-mission-elo-routing` is blocked on it (it has no substrate without this).
**Dependencies**: M-MISSION-LOOP-UNIFIED-TELEMETRY (landed `56b449d01`/`088176104`/`769d920a0`), `8131b4101` (session capture + Firestore stage counts, this session).

---

## Problem statement

We have built chain provenance three times and it has never produced a queryable feature record.
The reason is not that the pieces are missing — nearly all of them exist and work. It is that
**every link in the chain fails silently**, so each attempt looked wired and produced nothing, and
the next attempt rebuilt a layer that was already fine.

Measured first-party 2026-08-31 on this laptop and prod Firestore:

| Link | Believed | Measured |
|---|---|---|
| Eval runs → chains | works | **works** — 184 chains, 12,916 stages |
| Mission iterations → chains | posted every iteration | **0 ever posted**; no spool file exists either, so `chains post-iteration` was never *invoked* |
| Interactive sessions → sessions/tools | captured by hooks | **2 tool rows since April**, both 2026-06-16 |
| Message-driven work → chains | broken/empty | **works** — 99 prod chains WITH stages, real agents/costs |
| Cloud chain stage counts | 0 stages | stages exist; the counter was never incremented |

Four independent silent failures, each with a different owner:

1. **`ailang server` was not running and nothing said so.** It owns `/api/hooks/claude`;
   `coordinator_hook.sh` swallows connection failure by design (`|| true`) so local sessions stay
   quiet. Capture had been off since 2026-04-14 and no surface reported it.
2. **`session_tools` FK-rejected every tool event** for any session that predated the server, while
   the endpoint still answered `200 {"status":"ok"}`. Fixed in `8131b4101`.
3. **Firestore never incremented `stages_completed`**, and its `ListChains` reports that field
   rather than counting stage docs — so a populated store read as empty. Fixed in `8131b4101`.
4. **`chains post-iteration` is a model instruction, not driver code** — `.claude/skills/mission-control/SKILL.md:2815`,
   Gate 4, terminating in `|| true`. A skipped step is indistinguishable from a successful one.

**And the constraint that defeats the obvious design.** Correlation reaches the hook through
`$AILANG_CHAIN_ID` (`scripts/hooks/coordinator_hook.sh:20-23`), read from the hook process's own
environment. Measured this session: **environment does not persist between Claude Code tool calls** —
`export AILANG_CHAIN_ID=x` in one Bash call is empty in the next. So an interactive or model-driven
session can NEVER correlate itself by exporting a variable. The env path works only when something
outside the session sets it at launch (the coordinator, `mission-control.sh`). Every previous attempt
that assumed "the agent exports its chain id" was unimplementable, and failed quietly.

## Verification Log

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Interactive capture works once the server runs | started `ailang server`, ran tool calls | this session went 0 → 21 `session_tools` rows with start/end/success |
| V2 | Env cannot carry correlation in-session | `export X` in call A, read in call B | call B reads empty — the env path is launch-time only |
| V3 | Cloud stages exist | Firestore REST `runQuery` on `obs_chain_stages` | stages present with `agent_id`, `status`, `cost` ($0.978); `chains view` fails on a MISSING COMPOSITE INDEX (`chain_id, stage_number`) |
| V4 | `sessions` already carries the provenance spine | `PRAGMA table_info(sessions)` | `task_id, chain_id, stage_id, message_id` present; 6,468/6,508 rows carry chain+stage |
| V5 | No lookup by source_ref exists | grep `SourceRef` in `internal/observatory` | none — get-or-create by feature key must be built |
| V6 | `provider` is settable but unset on the coordinator path | `StageCreateRequest.Provider` vs `daemon_tasks_polling.go:292` | field exists; coordinator omits it; `AgentConfig` has no provider field, so it is NOT known at stage-create time |
| V7 | Sprint records cannot name the executed model | sweep 337 sprint JSONs | `planner_model` 11, `planner_lane` 4, `milestones` 13 — no record names the executor |

## Scope

**IN**: a `feature` chain source type keyed on a stable feature id; idempotent get-or-create by
`(source_type, source_ref)`; a `ailang chains feature` CLI that opens/closes workflow stages; a
**correlation state file** the hook reads so in-session work attaches without env; setting `provider`
where it is actually known; the missing Firestore composite index declared as code; a loud
capture-health surface so silence is never mistaken for success.

**OUT**: retiring `post-iteration` (it becomes a stage-writer against a feature chain, unchanged in
shape); cloud dual-write enablement (`AILANG_CHAINS_CLOUD` is a config act, tracked separately);
stale-data cleanup (explicitly sequenced AFTER correctness, per Mark 2026-08-31); the ELO fit that
consumes this.

## Design decisions

### D1 — A chain is a FEATURE; stages are workflow steps

`ChainSourceFeature ChainSourceType = "feature"`, `SourceRef` = a stable feature id (design-doc slug,
e.g. `m-feature-provenance-chains`). Stages are the workflow: `design`, `plan`, `execute`, `evaluate`
— `stage_number` ordering them, `agent_id` naming the role, `provider`/model naming the lane, and
`session_id` linking the session that did the work. A feature spanning three sessions and two weeks
is ONE row in `ailang chains list` with four stages and three linked sessions.

Rationale: it is the grain the question is asked at ("what happened to this feature?"), and it is the
only grain that survives session boundaries. Iteration- and session-grained chains answer questions
nobody asks twice.

### D2 — Get-or-create is idempotent on `(source_type, source_ref)`, and it is the ONLY way to open a feature chain

`GetOrCreateChainBySourceRef` on the `Backend` interface, implemented for SQLite and Firestore, with
a UNIQUE index on `(source_type, source_ref)`. Re-running any workflow step re-attaches to the same
chain rather than forking a new one. Without this, a retried sprint silently creates a second feature
chain and the provenance splits in half — the failure mode that makes a chain store untrustworthy
rather than merely incomplete.

### D3 — Correlation reaches in-session work through a STATE FILE, not the environment

Because of V2, `coordinator_hook.sh` gains a fallback: when `$AILANG_CHAIN_ID` is empty, read
`$AILANG_STATE/current-chain.json` (`{chain_id, stage_id, feature, opened_at}`), scoped per workspace.
`ailang chains feature <id> --stage execute` writes that file and prints what it wrote; `--close`
removes it. Env still WINS when set, so coordinator- and mission-launched sessions are unaffected.

This is the load-bearing decision. It is what lets an interactive session — this one — attach itself
to a feature, which no previous design could express.

### D4 — `provider` is written where it is known, and left NULL where it is not

V6: the coordinator does not know the executor at stage-create time, and `AgentConfig` carries no
provider field. So the fix is NOT to pass a guess at create. `provider` is set by an explicit
`UpdateStageProvider` at the point the executor is actually selected. A stage whose lane is genuinely
unknown keeps `provider` NULL and is reported as unknown — never defaulted to `claude`, which would
be a §2 silent fallback on the exact column a routing decision would later read.

### D5 — Capture health is a surface, not an assumption

`ailang chains health` gains a **capture** section: is the server reachable, when did the last
session/tool/stage row land, and is any workflow step open with no session attached. The four defects
above all shared one property — the system reported success while producing nothing. A green health
line that would have gone red is the deliverable, not a nicety.

### D6 — The Firestore composite index is declared as code

V3: `chains view --remote gcp` fails on a missing `(chain_id, stage_number)` index. A
`firestore.indexes.json` enters the terraform repo so the index is reproducible rather than
hand-created, and the read path reports index errors as errors — never as "0 stages".

## Milestones

**M1 — get-or-create + feature source type** (`internal/observatory`): `ChainSourceFeature`, UNIQUE
index migration, `GetOrCreateChainBySourceRef` on Backend + SQLite + Firestore, idempotency tests
(same key twice ⇒ one chain, same id).

**M2 — `ailang chains feature` CLI**: open/close a workflow stage on a feature chain, write and
remove the D3 state file, `--json`. Fail-closed on an unknown stage name.

**M3 — hook correlation fallback**: `coordinator_hook.sh` reads the state file when env is empty;
env wins when set. Test arms for both orders and for a corrupt/missing file.

**M4 — provider where known** (D4) + `UpdateStageProvider`, wired at executor selection.

**M5 — capture health** (D5) and the index-as-code (D6).

**M6 — workflow wiring**: `design-doc-creator`, `sprint-planner`, `sprint-executor`,
`sprint-evaluator` open their stage; `mission-control.sh` posts from the DRIVER, not the skill, so a
skipped step cannot be silent.

## Acceptance criteria

| AC | Claim (rc-observable) |
|----|----|
| AC1 | Two `GetOrCreateChainBySourceRef` calls with the same key return the same chain id; a UNIQUE violation is impossible by construction |
| AC2 | `ailang chains feature X --stage design` then `--stage execute` yields ONE chain with two ordered stages |
| AC3 | With env unset and the state file present, a tool event lands with the file's `chain_id`; with BOTH set, env wins |
| AC4 | A stage whose provider is unknown reads NULL and renders `unknown` — never `claude` |
| AC5 | `chains health` goes RED when the server is unreachable or no row has landed in N hours |
| AC6 | `chains view --remote gcp` returns stages (index present) or a NAMED index error — never a silent 0 |
| AC7 | A feature worked in 3 separate sessions shows 3 linked sessions on one chain |

**Mutation arms**: MU-1 make get-or-create always create → AC1 RED. MU-2 make the hook prefer the
file over env → AC3 RED. MU-3 default an unknown provider to `claude` → AC4 RED. MU-4 swallow the
Firestore index error and return 0 → AC6 RED.

## Risks

- **A fifth silent failure.** The mitigation is D5: every link gets a health assertion, so the next
  one is visible in a day rather than a quarter.
- **State-file staleness** — a crashed session leaves `current-chain.json` behind and later work
  attaches to a finished feature. Mitigate with `opened_at` + a staleness bound, reported by health.
- **Two write paths to one chain** (driver post + in-session stages) can double-count. Stage ids are
  pinned and get-or-create is idempotent, but the rollup needs a test that re-posting is a no-op.
