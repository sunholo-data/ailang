# Fleet iteration 38 executor report

**M1 complete. M2 blocked by the filesystem sandbox protecting `.agents`.**
No git write operations, provider calls, real-home fixtures, legacy import, dispatch or activation were performed. All production edits are within the user allowlist. Evidence/scratch copies live under `.snap` and are excluded from recursive snapshots.

## M1: complete

Registered `MISSION_QUORUM_ARTIFACT_DIR` with empty default, mission-area documentation and a raw getter, plus raw policy-marker presence. The writer now resolves explicit non-default arguments first, confines marked default writes before validating overrides, validates absolute unmarked overrides (including spaces), and retains the exact relative sentinel otherwise. Suppression names the absolute local destination without exposing marker contents. Resolution creates no directories. Existing exclusive-create collision handling, JSON shape, slug and PR #1824 salvage behavior remain intact.

Added unit and real public policytool host/child tests covering valid/invalid inheritance, whitespace marker, allowed writes, denied writes with zero launches, diagnostics, invalid paths/NUL, deterministic ENOTDIR and actual permission errors, unavailable CWD, registry/raw bytes and collisions. The initial TDD run was red (missing API); final focused tests are green. Scratch mutants provide semantic red evidence.

M1 gates: **G1 rc0; G2 rc0; G3 rc0; G6 rc0; G10 rc0; G12 rc0**. All 16 M1 mutation controls went red (rc1), then green (rc0) after restoration. Commands and logs: `M1/GATES.txt`, `M1/G*.log`, `M1-TDD-red.log`, `M1-TDD-green.log`, `MUTATION.md`, `mut/M1-*-{red,green}.log`. Full cumulative post-M1 content: `M1/FILES.txt` and matching relative paths.

## M2: implementation prepared, incomplete

Added the pure bash 3.2 helper, nine-line checked startup wiring after profiles/secrets/log and before both launch paths, source Gate 2 executable reader, shell suite, mandatory Go adapter, independent writer arms and actual taskFor/BuildEnvironment fake-CLI handoff tests. Added the shell-only LAUNCHD_SUITE recipe, regenerated env docs through the existing generator (259 variables), and the required `### Fixed` changelog fragment.

The helper creates no evidence directory at startup; missions share repo namespaces, active marked hosts stop, standalone marked readers stay in sandbox CWD, and readers consult export then current-repo local fallback. Directory/read/parse errors dominate matches. Existing quorum verdict/remedy and nested schema policy remain unchanged. No new Codex forwarding rule or executor/policytool/CLI edit was made.

**Required mirror is not updated.** `/bin/bash tools/launchd/sync-agents-skills.sh` returned rc1: rsync writes/metadata updates under `.agents` were denied with `Operation not permitted`. No alternate write route or manual mirror editing was attempted. The source and mirror therefore differ; the tests deliberately reject that state. The `.agents` copy has no new executable marker. Controller must run the SAME sync tool on an authority-permitting runner, verify only the intended mirror changed, then rerun the blocked gates and mutations before accepting M2.

Final M2 acceptance gates: **G2/G4/G6/G9/G10/G11/G14 rc0**. **G1/G5/G13 rc1 — UNINFORMATIVE UNDER SANDBOX**, because the required protected mirror could not be synchronized. Sync and aggregate shell suite also rc1 with the same classification. `M2/GATES.txt` contains every command and actual rc. No gate is labelled passed based on its green planner baseline.

Twelve of sixteen shell-only named arms pass individually; four require the stale mirror and are uninformative. Real-writer `cross_tree_durable` and `attended_no_exports` pass with exact path/bytes checks. `cross_mode_fixed_order` reaches the mission-to-attended mirror read and cannot complete. All three handoff variants (clean/inherited marker/task policy), across four roles and six provider identities, pass. The actual legacy Codex call-site capture passes and reports process marker presence separately from forwarded shell overrides.

Full cumulative partial-M2 content is snapshotted under `M2/` with `FILES.txt`. Sprint JSON keeps M1 `passes:true`, M2 `passes:false`, `completed:null`, and overall `in_progress`; no false completion or activation is recorded.

## Mutations

`MUTATION.md` records **42 attempts: 35 red/restored-green controls and 7 UNINFORMATIVE UNDER SANDBOX**. All M1 rows and every M2 table row (including separate HOME, order, bash4-syntax, registry, source-only sync and three handoff mutations) were attempted on scratch file copies. Go uses `-overlay` to target the scratch copy; shell uses a disposable copied fixture tree. Working production files were never mutated. Scratch mirrors intentionally retain current bytes; acceptance gates were not redirected around the sandbox.

The seven mirror-dependent rows return rc1 before/after restoration and are explicitly not mutation kills. Rerun them after authorized sync. Every informative mutant reddened its named arm without a build failure; no surviving informative mutant required an assertion change. Actual stderr/failure logs remain under `mut/`; numbered M2 logs distinguish separate mutants of the same arm.

## Deviations and limitations

- macOS returned a deleted current directory's former path; the resolver verifies the CWD exists before naming the confined destination. The original deleted-CWD arm exposed this, and its error-return mutant now goes red/restored green.
- Hermetic fixture repos use a fake read-only `git rev-parse` seam rather than `git init`, clone or commits, honoring the absolute no-git-write instruction. Separate A/B/C fixture roots and the real Go writer still establish durable evidence visibility. Live repo/lane observations remain controller-owned.
- The plan's scoped gate list overrides the skill's broad checkpoint/commit/finalize commands. No full make test, full launchd suite, unrelated edits, doc moves or handoff messages were run.
- Snapshot JSON serialization preserves all non-progress fields, including Unicode. The M1 snapshot's equivalent Unicode escaping was normalized to avoid unrelated field churn.
- The source/mirror requirement cannot be completed in this sandbox. Independent evaluator review, updated binary/lane marker presence, live matching paths and activation remain outstanding, as required by the controller-owned rollout.

## Real-path observations

Read-only before/after manifests (`observations-before.json`, `observations-after.json`) are identical for pin, main clone, host and World evidence directories. Each manifest records existence, file count, byte line count and SHA256 over sorted basename + NUL + exact JSON bytes. No concurrent change or real-path mutation was observed. Existing full-config listener/network limitations remain excluded as specified in the plan; no attempt was made to evade them.
