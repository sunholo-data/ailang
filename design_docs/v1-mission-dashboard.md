# Mission Dashboard — V1
*Snapshot, overwritten every iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`.
Last written: iteration 334, 2026-09-06.*

## Where the mission is
- **Goal (`D-51`)**: **N = 12** design docs before v1.0.0. Iteration 334 moved it **±0** — designed
  and planned, zero milestones executed, and it ABSORBS two queue rows rather than adding a 13th.
- **Release** v0.35.1 · `dev` CI **GREEN** at `e50066037` (16 checks, 0 not-green).
- Last complete sprint: `m-compile-cache-unverified-artifacts`, 4/4, iter-330→333.

## In flight
- **`m-cache-module-id-encoding`** [IN-SPRINT, 0 of 4]. Design + plan landed and judged this
  iteration (`sonnet` PASS 85/100, zero blocking). The compile artifact cache publishes **nothing
  on Windows** (drive-letter colon survives into the directory name) and its mapping is not
  injective anywhere (`a/b` = `a__b`). Encoding `m-<slug>-<16hex>`. **Resume: execute M1** (0.35 d).
  ⚠ The plan flags **M3 and M4 as having no non-vacuous mutation of their own diff** — read its
  non-vacuity ledger before executing either.

## Up next (banked)
1. `m-cache-module-id-encoding` **M1** — in-sprint, smallest, unblocks M2's wiring.
2. `m-gate1-shared-clone-ref-drift` — Gate 1's sync verdict expires mid-iteration (one shared
   `.git` across missions); 2 first-party instances.
3. `m-pi-runner-worktree-assertion-vacuous-on-revision` — NEW: the pi runner returns `ok` for a run
   that wrote nothing, whenever the target file was already dirty.

## Loop and routing
- launchd, pinned worktree `~/.ailang-driver-pin/v1`; iteration 334 ran ~04:33–06:00 CEST.
- Designer **rotation** `claude-fable-5-1` → `gpt-6-astra` → `pi/deepseek-v4-flash` (pointer now
  past deepseek). Planner + executor `codex:gpt-5.6-sol`, evaluator `sonnet`.
- Quorum: `gpt6-astra`, `gemini-3-1-pro`, `oc-glm-5-2`.
- Standing routing defect, **7 instances**: `resolve-role-spawn.sh planner` answers
  `agent-tool opus fail-closed:planner-lane-field-missing` for essentially every real pick and the
  spawn-pin hook then denies it. Route straight to the pin; the fix belongs in the TOOL.

## Parked on Mark
- **`D-55`**, the only OPEN row of 55 — bound adversarial gob decode, or ship the correctness fix?
  Its pre-registered default (a) already carried the whole compile-cache sprint to completion, so
  **nothing is blocked on it**; it stays OPEN only because the loop may not resolve its own row.

## Quota posture
- `metered=$0.28` of a $5/iteration ceiling, all of it quorum reviewers. Fable: ONE bounded
  revision run (ceiling is one DOC; authoring ran on the flat-rate pi lane). codex + pi probed
  rc=0. No GPU, no `rig.lock`.
