# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last updated:** 2026-09-06 (iteration 333)

## Release
- Latest: **v0.35.1**. `dev` is ahead; no release owed by the loop (standing rule 3: stop at ready-to-release).

## Goal distance
- **N = 12** design docs remain before v1.0.0 (was **13**, **−1**): `m-compile-cache-unverified-artifacts`
  LANDED complete this iteration (M4 of 4). Unit ratified by `D-51`/`D-53`.

## In flight / next
1. `m-cachesrc-cognitive-complexity` — SonarCloud new-code maintainability red inherited by M2–M4; NEW and ours.
2. `m-cache-sanitize-module-id-windows-colon` — **NEW, iter-333**: the compile artifact cache publishes
   *nothing* on Windows; `sanitizeModuleID` leaves the drive-letter colon in the directory name.
3. `m-gate1-shared-clone-ref-drift` — `origin/dev` advances when a sibling fetches; Gate 1's verdict expires.
4. `ci-red-mission-loop-workbench` — **motoko's**; resume predicate = `#1055` merged.

## Loop cadence + routing
- launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1` at `origin/dev`.
- Controller `claude:claude-opus-5` · designer ROTATION (next = `pi:ollama/deepseek-v4-flash:0731-cloud`;
  pointer parked at `codex:gpt-6-astra`, unspent since 329) · planner `codex:gpt-5.6-sol` ·
  executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
- Known routing defect, 7th instance: `derive-planner-lane.sh` answers `opus fail-closed:planner-lane-field-missing`
  while the spawn-pin hook would deny that spawn. Fix belongs in the tool; row queued.

## Parked on Mark
- **`D-55`** (only OPEN row of 55) — adversarial gob-decode bound for the compile-cache design.
  Its default (a) has now carried the sprint to completion; the ask is whether to ratify that or
  re-open scope. The loop may not resolve its own row.

## Quota posture
- `metered=$0.00` this iteration of the `$5` ceiling. No quorum spend (doc already through the gate),
  no GPU, no `rig.lock`. Fable designer budget UNSPENT for a fourth consecutive iteration.
