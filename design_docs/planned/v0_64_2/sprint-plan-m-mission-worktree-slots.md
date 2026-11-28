# Sprint Plan: M-MISSION-WORKTREE-SLOTS — reusable mission worktree slots

**Design doc:** [m-mission-worktree-slots.md](m-mission-worktree-slots.md) (ratified 2026-11-28)
**Sprint JSON:** `.ailang/state/sprints/sprint_M-MISSION-WORKTREE-SLOTS.json`

## Placement and Human-Decision Rules

**There are no human-decision gates in this sprint.** D1–D4 were ratified on 2026-11-28 (design
doc "Design Freeze"); they are copied below as rulings, not gates.

| Ruling | Value |
|---|---|
| D1 dirty slot | **salvage-then-reset**: staged diff, unstaged diff, `index` copy, untracked tar, HEAD + branch → `~/.ailang/state/slots/<slot>/salvage-<ts>/`, then reset |
| D2 slot root | `~/.ailang-slots/<repo>/<mission>/<role>-<n>`. Must stay outside `~/dev/sunholo-data` (weekly checkout-audit deletes idle ≥ 7 d trees there) and outside every sweep `PROTECT` prefix |
| D3 Go cache | mission-only `GOCACHE=$HOME/.cache/ailang-mission-go`; size recorded per fire; one notice above **40 GB**; **never** auto-cleaned |
| D4 sweep | keep `~/.ailang-driver-pin/` prefix protection; **add** the slot root to `PROTECT`; sweep the fixed clone list `ailang`, `ailang-world`, `ailang-docs`, `ailang-motoko` |

## Summary

Replace per-iteration fresh worktrees with lock-held, reusable slots owned by the fire's driver,
so mission checkouts become incremental (about 1% of files per day of churn), Go cache keys stay
stable, and the sweep can see and bound every mission tree.

**Duration:** 3 days (4 milestones, each sized for one 60–90 min executor run)
**Dependencies:** none
**Risk level:** Medium. M3 changes the live launchd driver and M4 changes the shared mission skill.
Both affect every mission on its next fire.
**Execution:** **attended** (memory: harness debt is cleared attended, and loops restart only on a
stable base). The mission skill loads from the **main** checkout, not the pin, so M4 is live only
once the main checkout carries it.

## Current Status Analysis

### Already in place (reused, not rebuilt)
- ✅ `tools/launchd/mission-worktree.sh` (213 lines): `add` / `wait`, identity-checked owner records
  (`identity()`, `birth()`, `register()`), detached bounded `_writer`, half-built-tree protection.
  Tests: `tools/launchd/test_mission_worktree.sh` (457 lines), wired in `make/test.mk` L83.
- ✅ `pin-root.sh` L272–275 already reuses its tree with `checkout --detach --force`. The slot
  reset copies this pattern.
- ✅ `scripts/worktree_sweep.sh` (149 lines) + `scripts/test_worktree_sweep.sh` (wired L134);
  `scripts/hooks/git_health.sh` (77 lines) + `test_git_health.sh` (L131).
- ✅ One-off clean-up done 2026-11-28: 353 → 89 worktrees across ailang + ailang-world; free disk
  76 → 146 GB.

### Velocity
The velocity script's 7-day window is dominated by docs merges and tells us little. The closest
comparable is the bash mission tooling: `mission-worktree.sh` plus its tests (~670 lines) landed in
about one mission iteration. Planning figure: **~250–300 LOC/day** of bash + tests, with a 25%
buffer.

### Remaining from the design doc
- 📋 Component 1 (acquire / held / release-fire, owner, steal, guards): ~320 LOC
- 📋 Component 1 step 4–5 (reset, salvage, proof): ~250 LOC
- 📋 Components 3 + 4 (driver trap, GOCACHE, sweep): ~145 LOC
- 📋 Component 2 (gate wiring) + rollout: ~90 lines

## Registry Reuse Audit

| Milestone | Searches (`ailang pkg search`) | Inspected | Decision | Reason |
|---|---|---|---|---|
| M1 | `worktree`, `git checkout`, `lock`, `slot`, `cache` | none relevant (only `lock` matched, on unrelated physics/audio/social packages) | **none** | rig-local bash launchd tooling; no AILANG package covers git worktrees or file locks |
| M2 | same | same | **none** | same |
| M3 | same | same | **none** | edits to existing driver/sweep shell scripts |
| M4 | same | same | **none** | skill-text wiring + changelog |

## Proposed Milestones

### M1: Slot ownership: `acquire` / `held` / `release-fire`

**Goal:** Fire-owned, lock-held slots with safe stealing and location guards. First creation goes
through the existing bounded writer. No reset logic yet: an existing slot is reused only if clean
at the requested SHA, otherwise `failed:reset_not_implemented`. That keeps M1 bisectable, and
nothing calls `acquire` until M4.

**Estimated:** ~140 implementation + ~180 tests = **~320 LOC**
**Files:**
- `tools/launchd/mission-worktree.sh`: new verbs `acquire MISSION ROLE SHA [BRANCH]`,
  `held FIRE_ID ROLE`, `release-fire FIRE_ID`. Keep the file under ~450 lines; if it grows past
  that, move the slot verbs to `tools/launchd/lib/mission-slot.sh`, sourced by it.
- `tools/launchd/test_mission_slots.sh` (**new**; the existing test file is already 457 lines) +
  add it to the `LAUNCHD_…` list in `make/test.mk` next to L83.

**Tasks:**
1. Owner model: parse driver PID from `MISSION_FIRE_ID` (`<mission>-<epoch>-<pid>`); record
   `fire_id`, PID, `birth()`, role. No `MISSION_FIRE_ID` → `failed:no_fire_owner`.
2. Root guards: refuse a slot root under `/tmp` / `/private/tmp`, under any `PROTECT` prefix of
   `scripts/worktree_sweep.sh` (read the same list, do not copy it), or in the checkout-audit
   scan set (directly under `~/dev/sunholo-data`, or in a `*-worktrees` / `*-wt` / `*-worktree`
   folder there) → `failed:slot_root_protected`. Root from `AILANG_SLOT_ROOT` is **not** added:
   fixed `~/.ailang-slots` per D2; tests override `HOME`.
3. Free-slot scan `<role>-1..N` (N from a per-role default table: exec 2, plan 1, record 1,
   judge 2). Held = live driver PID with matching birth (inconclusive = held), **or** any process
   cwd inside (`lsof -d cwd`; an `lsof` failure means held).
4. Lock: `mkdir <slot>.lock` + tmp/`mv` owner file. Steal: `mv <slot>.lock <slot>.lock.dead.$$`,
   re-read the owner, check it is the judged dead identity (else move it back), then `mkdir`.
5. First creation: launch the existing `_writer` and print `pending <path>`; `wait` unchanged.
6. `held FIRE_ID ROLE` prints the path(s) this fire holds for the role (exit 1 if none).
   `release-fire FIRE_ID` removes only locks whose `fire_id` matches.

**Acceptance criteria:**
- [ ] TST-SLOT-01, TST-SLOT-04, TST-SLOT-05, TST-SLOT-09, TST-SLOT-12, TST-SLOT-13, TST-SLOT-14 pass in `tools/launchd/test_mission_slots.sh` (each test carries its ID in a comment, so `check_acceptance_criteria.sh` can find it)
- [ ] TST-SLOT-04 uses the real topology: acquire in shell A, which exits; driver stand-in D alive; **no cwd inside**; shell B in another process group (`set -m`) acquires
- [ ] TST-SLOT-11: `tools/launchd/test_mission_worktree.sh` passes **unmodified** (`git diff --exit-code` on it)
- [ ] `/bin/bash -n` clean; runs under `/bin/bash` 3.2 (no `declare -A`, `${v,,}`, GNU `timeout`, relative `find -newermt`)
- [ ] `make test-launchd-drivers` green

**Risks:**
- PID reuse on a long-uptime rig. Mitigation: birth-time match, as `identity()` already does.
- `lsof` is slow under load. Mitigation: bound it with the sweep's `bounded 30` pattern; timeout = held.

### M2: Reset + salvage + cleanliness proof

**Goal:** Reuse an existing slot at a new SHA: salvage dirt, reset in place, prove it clean. This
is the Go-cache and checkout win.

**Estimated:** ~100 implementation + ~150 tests = **~250 LOC**
**Depends on:** M1
**Files:** `tools/launchd/mission-worktree.sh` (or `lib/mission-slot.sh`), `tools/launchd/test_mission_slots.sh`

**Tasks:**
1. In the `_writer` path for an existing slot: dirty check → salvage to
   `~/.ailang/state/slots/<slot>/salvage-<ts>/` (`git diff --binary --cached`, `git diff --binary`,
   `cp <gitdir>/index`, `git ls-files -o --exclude-standard -z | xargs -0 tar -cf`, `INFO` with HEAD
   + branch). Any `rm` of a salvage artefact uses `"${d:?}/${x:?}"` (rig safety check).
2. `git checkout --quiet --detach --force <SHA>` → `git clean -ffdx` → optional
   `git switch -c <BRANCH>`; an existing branch gives `failed:branch_exists` (never `-C`).
3. Proof: `rev-parse HEAD == SHA` and empty `status --porcelain`; otherwise remove the slot +
   registration and report `failed:not_clean_after_reset`.
4. One log line per reset to `~/.ailang/state/worktree-sweep.log` style (`[ts] [repo] slot reset …
   salvage=<dir|none>`).

**Acceptance criteria:**
- [ ] TST-SLOT-02, TST-SLOT-03, TST-SLOT-07, TST-SLOT-08 pass
- [ ] TST-SLOT-03 includes the "stage B, restore A in the working file" case and restores the index byte-identically from the salvage
- [ ] TST-SLOT-02 asserts only ≈K files' mtimes changed for a K-file SHA delta (use `perl stat`, not GNU `stat`)
- [ ] M1 tests still pass; `make test-launchd-drivers` green
- [ ] Mutation check (memory: mutate your own tests): replace `--cached` salvage with `git diff HEAD` → TST-SLOT-03 fails; skip the porcelain proof → TST-SLOT-08 fails. Do it via a scratchpad copy, never `git checkout <file>`

**Risks:**
- `git clean -ffdx` deletes ignored build outputs (e.g. `bin/ailang`). Intended: they rebuild from a warm cache. If a mission measures it as slow, add `-e` excludes per mission (Deferred Decision).

### M3: Driver trap + mission GOCACHE + sweep coverage

**Goal:** The driver releases its slots on exit and isolates the Go cache; the sweep protects
slots and covers every listed clone.

**Estimated:** ~75 implementation + ~70 tests = **~145 LOC**
**Depends on:** M1 (`release-fire`)
**Files:**
- `tools/launchd/mission-control.sh`: next to `MISSION_FIRE_ID` (L2431):
  `trap '… release-fire "$MISSION_FIRE_ID"' EXIT`, `export GOCACHE="$HOME/.cache/ailang-mission-go"`,
  `du -sk` recorded in the fire's log/record line, one `_mc_notice_*` notice when above 40 GB
  (reuse `_mc_notice_title` / `_mc_drain_notices`, L202–226; check `_mc_notice_suppressed` so it
  does not repeat every fire).
- `scripts/worktree_sweep.sh`: add `$HOME/.ailang-slots/` to `PROTECT` (L53); new `--repo <path>`
  repeat flag (default: the current clone, unchanged).
- `scripts/hooks/git_health.sh`: add the slot root to the protect list (L71); loop the sweep over
  `ailang`, `ailang-world`, `ailang-docs`, `ailang-motoko` under `~/dev/sunholo-data` when present.
- `scripts/test_worktree_sweep.sh`, `scripts/hooks/test_git_health.sh`: new cases.

**Acceptance criteria:**
- [ ] TST-SLOT-10 passes (idle clean slot under the slot root is KEPT by `--apply`)
- [ ] New sweep case: `--repo A --repo B` judges both clones and logs each with its own `[repo]` tag
- [ ] `git_health.sh` test: the sweep is spawned once per present clone and never for an absent one; still exits 0 and still skips when `AILANG_MISSION_STAGE` / `AILANG_TASK_ID` are set
- [ ] Driver: a test (or `tools/launchd/test_pin_root.sh`-style harness) shows the `EXIT` trap fires `release-fire` with the fire's id after the pin re-exec, and the exported `GOCACHE` reaches a child (`/bin/bash -c 'go env GOCACHE'` equals the mission path)
- [ ] Existing `scripts/test_worktree_sweep.sh` case "protected mission pin → kept" (L66) passes unchanged
- [ ] `make test-launchd-drivers` green; `/bin/bash -n tools/launchd/mission-control.sh`

**Risks:**
- **Live on the next fire after this reaches origin/dev** (the pin re-execs committed code).
  `release-fire` with no slots is a no-op; the GOCACHE export means each mission's **first
  build is cold** (one-time, several minutes). Mitigation: land M3 when no mission is mid-sprint,
  and watch one fire.
- Sweeping `ailang-world` from attended sessions now runs its 24-h-safe rules there. The idle
  threshold stays 2 h (default). Mitigation: the same five rules that already run on ailang; the
  world mission's live trees are protected by rule 2 (cwd) and 3 (idle).

### M4: Gate wiring + rollout + measurement

**Goal:** Mission recipes use slots; measure the cache win; document.

**Estimated:** ~60 lines skill text + ~30 changelog/docs = **~90 lines**
**Depends on:** M1, M2, M3
**Files:**
- `.claude/skills/mission-control/resources/gate-3-route.md`: sprint/executor (L651–694),
  planner (L232–241) and judge/evaluator recipes → `acquire` + `wait` loop + `held` for re-entry;
  remove recipe-level cleanup/`trap`; keep the half-built-tree rules, scoped to "first creation
  of a slot". Planner worktree: use role `plan`, detached (no BRANCH), at local `HEAD` as today.
- `.claude/skills/mission-control/resources/gate-4-record.md` (L288): record tree → role `record`.
- `.claude/skills/mission-control/resources/ref-drift.md`: the base-SHA measurement feeds `acquire`.
- `changelogs/unreleased/2026-11-2X-mission-worktree-slots.md` (`### Changed — …`).
- `docs/internal/runbooks/`: short "mission worktree slots" runbook (where slots live, how to
  read a salvage, how to restore `refs/salvage/*`).

**Acceptance criteria:**
- [ ] `grep -n 'mission-worktree.sh" add\|worktree add' .claude/skills/mission-control/resources/*.md` shows no remaining per-iteration creation outside "first creation" wording
- [ ] TST-SLOT-06 measured on the rig and recorded in the design doc: second build on an unchanged SHA in the same slot gains ≈0 bytes in `~/.cache/ailang-mission-go`; control: a fresh path gains ≥1 GB
- [ ] One real fire per migrated mission (fleet first, then triage, v1, world) observed: slot path used, `release-fire` logged, worktree count unchanged after the fire
- [ ] `make check-changelog`, `make check-skills`, `make check-referenced-paths` green
- [ ] Main checkout updated (skills load from the main worktree; memory `project_mission_skills_load_from_main_worktree`)

**Risks:**
- A recipe still using `add` in a mission not yet migrated works unchanged (`add` / `wait` kept). Rollback = revert M4's skill commit; M1–M3 are inert without it.

## Success Metrics

- Worktrees per repo ≤ pins + slots after two weeks live (today 45 ailang / 44 world; slots add at most ~6 per mission)
- Mission GOCACHE stays under 40 GB with no manual clean-up
- Slot reset rewrites ≈ `git diff --name-only` files (TST-SLOT-02) instead of 31,441
- All TST-SLOT-01..14 pass; `make test-launchd-drivers` green
- Docs: changelog fragment + runbook; the design doc moves to `implemented/` via `move_to_implemented.sh`

## Dependencies

- None external. M2 and M3 depend on M1; M4 depends on all three.

## Open Questions

- None. D1–D4 are ruled. Deferred Decisions in the design doc (D5 `-trimpath`, D6 slot counts,
  `git clean -e`, salvage retention) are executor latitude, not gates.

## Notes

- Correction made while planning: the design's example wrote the slot path to a per-iteration
  state file (`$ITER_STATE`) that the driver does not have (design V16). It is replaced by
  `held FIRE_ID ROLE`, which reads the lock owner records.
- `check-file-sizes` covers Go only; the bash files are kept in the 200–500 sweet spot by the
  `lib/mission-slot.sh` split rule in M1.
- Today's salvage (`~/.ailang/state/worktree-salvage/2026-11-28{,-world}/`, `refs/salvage/*`)
  is outside slot salvage; leave it for the retention rule.
