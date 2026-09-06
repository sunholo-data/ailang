# M-GATE1-SHARED-CLONE-REF-DRIFT Sprint Plan — post-split recovery

**Design**: `design_docs/planned/v0_36_0/m-gate1-shared-clone-ref-drift.md`
**Status**: IN PROGRESS — M1 complete at `20201e1d9`; M2/M3 replanned after upstream `c1212b3c` split the operational mission-control skill
**Duration**: 3 days total (1 complete, 2 remaining)
**Estimated size**: 302 LOC total (152 actual M1 + 110 M2 + 40 M3), about 101 LOC/day
**Risk**: low implementation risk; recovery/base-alignment is the primary execution risk

## Recovery boundary

Commit `c1212b3c` replaced the operational 2,781-line `.claude/skills/mission-control/SKILL.md` monolith with a 560-line index and authoritative per-gate resources. This changes only the M2/M3 edit locations. The reviewed helper contract, conflict direction, gate behavior, acceptance semantics, and both quorum-R2 fixes remain frozen.

Before the executor starts, the controller must provide one clean execution base containing both:

- the upstream gate-resource split (`c1212b3c` or a descendant), including `gate-1-observe.md`, `gate-3-route.md`, `gate-3b-ci-green.md`, and `gate-4-record.md`; and
- committed M1 `20201e1d9`, including `mission-base.sh`, its non-vacuity test, and the `make test-launchd-drivers` wiring.

The uncommitted old-monolith M2 diff and the existing `.snap/M2/` material are **abandoned evidence, not implementation input**. The executor must not copy, merge, apply, restore, or otherwise carry either forward. The controller must clear or quarantine them before execution; M2 must be authored afresh against the split resources and produce a fresh split-resource snapshot. Do not edit the root `SKILL.md` to salvage any old hunk.

No quorum rerun is permitted: the design amendment is a file-map recovery, not a new direction.

## Current status

| Milestone | State | Size | Dependency | Authoritative files |
|---|---:|---:|---|---|
| M1 — measurement helper and non-vacuity test | COMPLETE (`20201e1d9`) | 152 actual LOC | none | `tools/launchd/mission-base.sh`, `tools/launchd/test_mission_base.sh`, `make/test.mk` |
| M2 — Gate 1/Gate 3 wiring and rationale | PENDING | ~110 LOC | M1 + clean post-split base | `resources/gate-1-observe.md`, `resources/gate-3-route.md`, new `resources/ref-drift.md` |
| M3 — Gate 3b/Gate 4 wiring and final re-proof | PENDING | ~40 LOC | M2 | `resources/gate-3b-ci-green.md`, `resources/gate-4-record.md` |

M1 remains complete and unchanged. Its implementation must not be regenerated. Verify it after base alignment, but do not reopen its reviewed code unless a check proves the committed content was lost.

## M1 — COMPLETE: helper + test

Commit `20201e1d9` added 55 lines in `mission-base.sh`, 96 lines in `test_mission_base.sh`, and one makefile line. It preserves the two R2 fixes:

- `record()` snaps exactly once into `rec`;
- `last()` exits 1 when no matching label exists; and
- `drift()` explicitly rejects an empty `old` and returns 2 for both no-record cases.

Post-alignment verification only:

```bash
git merge-base --is-ancestor 20201e1d9 HEAD
/bin/bash -n tools/launchd/mission-base.sh
/bin/bash -n tools/launchd/test_mission_base.sh
/bin/bash tools/launchd/test_mission_base.sh
grep -n 'test_mission_base.sh' make/test.mk
```

Expected: ancestry check succeeds, both syntax checks succeed, the test reports all eight arms green, and the make target contains the test invocation.

## M2 — Gate 1/Gate 3 wiring + rationale (~110 LOC)

### Files

- Modify `.claude/skills/mission-control/resources/gate-1-observe.md`.
- Modify `.claude/skills/mission-control/resources/gate-3-route.md`.
- Create `.claude/skills/mission-control/resources/ref-drift.md`.
- Do not modify `.claude/skills/mission-control/SKILL.md`.

### Work

1. In `gate-1-observe.md`, immediately after the existing Gate-1 fetch/rev-parse sync block, add the reviewed `mission-base.sh record gate1` command and echo the full SHA/read-time pair.
2. In `gate-3-route.md`, immediately before an `origin/dev`-based worktree is created, call `mission-base.sh snap`, compare the fresh SHA with `last gate1`, invoke `drift gate1` on disagreement, create from the fresh `$newsha`, and carry `base=$base` into provenance/routing evidence. Preserve the reviewed re-read-once/re-run-the-affected-gate/abort-only-on-invalidated-integrity protocol.
3. Create `resources/ref-drift.md` with the two measured iterations, why shared-clone ref movement is silent, the disagreement protocol, and the non-vacuity recipe. Link it from both Gate 1 and Gate 3 resources at the helper call sites.
4. Keep Gate 2 at the reviewed light-observation choice: it may use `mission-base.sh snap` in its pick note, but do not introduce a durable `base-gate2` stamp. M2 does not own `gate-2-pick.md`; defer this observation unless the controller explicitly schedules it later.

### Acceptance

- `gate-1-observe.md` contains `mission-base.sh record gate1` adjacent to its existing sync block.
- `gate-3-route.md` contains `mission-base.sh snap`, `mission-base.sh last gate1`, and `mission-base.sh drift gate1`, and requires the worktree base to be the fresh full SHA.
- Both resources link to `resources/ref-drift.md`; the new resource exists and contains the two-instance rationale plus drift protocol.
- The root `SKILL.md` is byte-identical to the clean execution base and remains at or below the historical 2,781-line no-growth ceiling (expected current size: 560).
- `make check-context-docs` exits 0 and all new links resolve.
- S1–S5 routing literals remain in `gate-3-route.md`: `resolve-role-spawn.sh`, `MISSION-ROLE:`, `enum in this build lists`, `now \`claude:claude-fable-5-1\` → \`codex:gpt-6-astra\` → \`pi:ollama/deepseek-v4-flash:0731-cloud\` → repeat`, and `ASTRA IS ALSO A QUORUM REVIEWER`.
- The scrubbed CI-parity suite exits 0:

```bash
env -i HOME="$HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb make test-launchd-drivers
```

- A fresh `.snap/M2/` is produced from the split-resource implementation. It must contain the cumulative M1 files plus `gate-1-observe.md`, `gate-3-route.md`, and `ref-drift.md`; it must not contain the root `SKILL.md` or content recovered from the abandoned snapshot.

## M3 — Gate 3b/Gate 4 wiring + final re-proof (~40 LOC)

### Files

- Modify `.claude/skills/mission-control/resources/gate-3b-ci-green.md`.
- Modify `.claude/skills/mission-control/resources/gate-4-record.md`.
- Do not modify the root `SKILL.md`; do not reopen M1.

### Work

1. In `gate-3b-ci-green.md`, replace the direct poll-target read with the reviewed `record gate3b` form, derive the full SHA from the returned record, compare against Gate 1, and retain the existing SHA-addressed CI lookup and bounded poll. Link to `ref-drift.md` near this rule.
2. In `gate-4-record.md`, retain the existing fetch plus `git rev-parse dev origin/dev` re-confirmation. Route the record-time base through `mission-base.sh record gate4`, and require the human log's Routing-evidence row to contain `base=<full-sha>@<iso>`. Link to `ref-drift.md` near this rule. Do not add a redundant second fetch/re-read.
3. Record the Gate-2 decision as deferred-with-reason; no `base-gate2` durable stamp is added in this sprint.

### Acceptance

- `gate-3b-ci-green.md` contains `mission-base.sh record gate3b`, derives the poll target from that same read, retains full-SHA selection, and links to `ref-drift.md`.
- `gate-4-record.md` contains `mission-base.sh record gate4`, preserves its existing fetch/re-confirmation, requires `base=<sha>@<iso>` in Routing evidence, and links to `ref-drift.md`.
- The root index is unchanged from the clean execution base; `make check-context-docs` exits 0; every new link resolves.
- All five S-guard literals still match in `gate-3-route.md`, and the scrubbed `make test-launchd-drivers` gate exits 0.
- Final non-vacuity re-proof runs last: steady returns 0; a scratch `git update-ref refs/remotes/origin/dev ...` mutation makes `drift gate1` return 1 with `DRIFT old -> new`; absent-file and missing-label cases both return 2 with `no base-gate1 record yet`.
- A fresh cumulative `.snap/M3/` contains the M1 files, all five split-resource files, and no root `SKILL.md`.

## Ratchets, guards, and exclusions

- Historical root ratchet: never bump `scripts/context_docs_baseline.txt` or `scripts/context_docs_links_baseline.txt`; root `SKILL.md` remains unchanged and ≤2781 lines.
- Operative context gates: `make check-context-docs`, new-link resolution, and unchanged root-index pointers to the four authoritative gate resources.
- S-guard location: all five guarded routing literals live in `resources/gate-3-route.md` after the split; check there, not in the root index.
- Do not modify `.agents/skills/mission-control/SKILL.md`, `mission-heartbeat.sh`, `mission-control.sh`, `pin-root.sh`, `test_mission_routing.sh`, either context baseline, or quorum receipts.
- Bash 3.2 only: no associative arrays, `${v,,}`, or GNU `timeout`; `snap` remains read-only and never fetches.
- Executor may edit only the five split resources named above and snapshot artifacts. It must not run `git add`, `git commit`, `git checkout`, or `git push`.

## Executor handoff

Handoff only after the controller has proven the clean base contains the split plus M1 and has neutralized the abandoned monolith diff/snapshot. The executor should implement M2, run its complete gates, emit a fresh M2 snapshot, then implement M3 and run the final mutation re-proof as its last check. The controller builds one commit for M2 and one for M3 from the fresh snapshots; M1 remains the existing commit.

SPRINT_PLAN_PATH: design_docs/planned/v0_36_0/m-gate1-shared-clone-ref-drift-sprint-plan.md
SPRINT_JSON_PATH: design_docs/planned/v0_36_0/m-gate1-shared-clone-ref-drift-sprint.json
