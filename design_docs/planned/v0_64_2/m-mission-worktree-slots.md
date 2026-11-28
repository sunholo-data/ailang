# M-MISSION-WORKTREE-SLOTS — reusable worktree slots for mission loops

**Status**: Planned — r3, awaiting human ratification (quorum r1 + r2 blocked; every objection addressed, see Quorum)
**Target**: v0.64.2
**Priority**: P1 (rig disk + I/O; recurring, compounding)
**Estimated**: 2–3 days
**Dependencies**: None. Reuses `tools/launchd/mission-worktree.sh` (identity/lock machinery),
`tools/launchd/lib/pin-root.sh` (the reuse pattern it already applies to driver pins) and
`scripts/worktree_sweep.sh`.
**Lane** (PROGRAM.md): harness / mission-loop infrastructure. Not an AILANG language change and
not a motoko core change.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms). This is harness
infrastructure, so most axioms are untouched; it is scored for the gates anyway.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Every mission tree is at the same path with a proven-clean state at an exact SHA; no CWD-shaped test variance between iterations |
| A2: Replayability | 0 | Iteration provenance (`base=<sha>`) unchanged |
| A3: Effect Legibility | +1 | Salvage-then-reset logs every destructive step with what was saved; today trees pile up silently |
| A4: Explicit Authority | 0 | No new authority; sweep protection is unchanged for pins and only *adds* the slot root |
| A5: Bounded Verification | 0 | n/a |
| A6: Safe Concurrency | +1 | Per-slot locks owned by the fire's driver (PID + birth), released only by the driver's EXIT trap, replace "unique path = no collision by luck" (TST-SLOT-04/05/12) |
| A7: Machines First | 0 | Unattended loops are the beneficiary; no human-convenience trade |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +1 | Mission GOCACHE size recorded per iteration with a threshold notice |
| A10: Composability | 0 | `add` / `wait` remain; `acquire` / `release` compose with them |
| A11: Structured Failure | +1 | `busy` / `failed:<why>` outputs; unclean reset deletes the slot and parks loudly |
| A12: System Boundary | 0 | n/a |

**Net Score: +5** → **Decision: Move forward**

- [x] A1: no implicit nondeterminism introduced
- [x] A3: no hidden side effects (every reset is logged with its salvage)
- [x] A4: no ambient access granted
- [x] A7: not optimizing for human convenience

## Problem Statement

Every mission iteration makes **brand-new, full git worktrees at a new path**: a sprint worktree,
a planner worktree, a record worktree and judge/evaluator worktrees (`gate-3-route.md` L651–694,
`gate-4-record.md` L288). Two costs follow, and the sweep cannot clean up after either.

**1. Fresh path = full Go rebuild, every time.** Without `-trimpath`, Go mixes the package's
absolute source directory into the build-cache action ID:

```
go1.26.11 src/cmd/go/internal/work/exec.go:296-299
    // -trimpath is not set and no other rewrite rules apply,
    // so the object file may refer to the absolute directory
    // containing the package.
    fmt.Fprintf(h, "dir %s\n", p.Dir)
```

So every one of the repo's ~222 Go packages misses the cache in a new worktree, is recompiled and
is written to `~/Library/Caches/go-build` again. Go trims entries only after **5 days** unused
(`cache.go` L348–350: `mtimeInterval=1h`, `trimInterval=24h`, `trimLimit=5d`), so the cache settles
at about *5 days of iterations × one full rebuild each*.

**2. A full checkout per tree.** The ailang repo has 31,441 tracked files (about 386 MB on disk per
worktree). Each `worktree add` writes all of them, and fseventsd and Spotlight then index all of
them. `gate-3-route.md` L717–736 already records the side effects: about 2 minutes per checkout,
foreground-timeout kills, half-built trees, and iteration 234's lock-removal accident. The
`mission-worktree.sh` helper exists only to make that slow checkout survivable.

**3. The throwaway trees are invisible to the sweep.**
- `scripts/worktree_sweep.sh:53` protects the whole `~/.ailang-driver-pin/` prefix (and
  `scripts/hooks/git_health.sh:71` passes it again). That prefix was meant for the driver pins.
- But gate-3/gate-4 put new trees as **siblings of `$REPO`**, and on a pinned pass `$REPO` *is* the
  pin (`~/.ailang-driver-pin/fleet`). So every fleet sprint, record and judge tree lands inside the
  protected prefix and is never removed.
- The sweep also runs only from an **attended** SessionStart, and only in the clone that session
  starts in (`git_health.sh` L22–23, L64–76). `ailang-world` therefore has never been swept.

### Measured (studio rig, 2026-11-28)

| Measurement | Value | Command |
|---|---|---|
| Go build cache size | 94 GB | `du -sh ~/Library/Caches/go-build` |
| Go cache written or used in the last 24 h | **43 GB** | `find … -mindepth 2 -type f -mtime -1` summed |
| Go `compile` writes, one night | **549.76 GB in 8.4 h** (18.19 MB/s) | `/Library/Logs/DiagnosticReports/compile_2026-11-25-012627_*.diag` |
| ailang worktrees registered | 162 (133 under `~/.ailang-driver-pin`) | `git worktree list \| wc -l` |
| `~/.ailang-driver-pin` | 49 GB, 52 dirs, oldest 2 Oct | `du -sh`, `ls -lt` |
| ailang-world worktrees | 191 registered; 140 `.wt-world-*` dirs = 14 GB | `git -C ailang-world worktree list`, `du -sch` |
| New mission worktree dirs in 24 h | 8 fleet + 17 world | `find … -maxdepth 1 -mtime -1` |
| Disk free before clean-up | 76 GB of 926 GB (92% full) | `df -g /System/Volumes/Data` |
| Sustained disk I/O while idle-ish | 30–50 MB/s, 1.4–2.8k IOPS | `iostat -d -w 5 disk0` |
| ailang churn per day | 47 commits → **337 of 31,441 files (1.1%)**, 35 of 222 Go packages | `git diff --name-only <24h-ago> origin/dev` |
| ailang-world churn per day | 107 of 7,496 files | same |

The last two rows are why this design works: a tree reused at a fixed path and moved forward by
one day touches about 1% of the files, and Go rebuilds only the changed packages and their
dependents.

**One-off clean-up already done (2026-11-28, attended):** 69 landed, clean, idle worktrees removed
using the sweep's own five rules with only the seven real pins protected. Go cache entries unused
for more than 1 day deleted. Free space went from 76 GB to 118 GB. 92 trees were kept, all with
uncommitted changes, unlanded work or a live process; they need a human or the salvage rule below.
Without this design, the space comes back within days.

## Goals

**Primary Goal:** Mission iterations reuse a small, fixed set of worktree slots. Checkouts become
incremental, Go cache keys stay stable, and mission worktree disk use has a fixed upper bound.

**Success Metrics:**
- Worktrees per repo stay ≤ (pins + slots), independent of iteration count. Today they grow
  without bound (162 / 191).
- Go cache growth from missions: a second iteration on an unchanged SHA writes ≈0 new cache
  entries (TST-SLOT-06). The 24 h cache working set drops from 43 GB toward one rebuild of the
  daily churn.
- Preparing a slot for a SHA one day newer takes seconds, not about 2 minutes, and rewrites about
  as many files as `git diff --name-only` reports.
- No mission tree can be created under a protected prefix again (`acquire` refuses it), so the
  pin prefix holds only pins, and every listed clone, `ailang-world` included, is swept.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: what happens to a **dirty slot** at acquire time (leftover from a killed run): *salvage-then-reset* vs *park* | Salvage keeps loops moving but resets a tree; park is today's "never repair" stance and would wedge the slot | human | design | med |
| D2: slot root location (`~/.ailang-slots/<repo>/<mission>/<role>-<n>`) | Must be outside `/tmp` (gate-3 L696), outside every protected pin, and stable | human | design | med |
| D3: mission-only `GOCACHE` (`~/.cache/ailang-mission-go`) with a size alarm, vs keep the shared user cache | Keeps mission growth from pushing out attended builds; costs one cold start | human | design | low |
| D4 (revised r2): sweep keeps the `~/.ailang-driver-pin/` prefix, adds the slot root, sweeps a fixed list of clones; `acquire` refuses a slot root under a protected prefix | Shared machinery attended sessions run; round 1's narrowing to a registry was withdrawn as a pin-deletion hazard | human | design | low |
| D5: `-trimpath` for mission builds | Would make the cache path-independent, but 15 test files use `runtime.Caller` | agent (deferred; see below) | compile | low |
| D6: slots per role (proposal: exec×2, plan×1, record×1, judge×2) | Too few causes parking under concurrency; too many wastes disk | agent | runtime | low |

### Design Freeze

- [ ] D1 — salvage-then-reset (recommended) or park
- [ ] D2 — slot root path
- [ ] D3 — mission-only GOCACHE: yes/no; if yes, alarm threshold (proposal 40 GB)
- [ ] D4 — keep pin-prefix protection, add slot root, sweep `ailang`/`ailang-world`/`ailang-docs`/`ailang-motoko`

## Solution Design

### Overview

Use the pattern the driver pin already relies on (`pin-root.sh` L260–284: when the worktree
exists, `git checkout --quiet --detach --force <target>`; otherwise `worktree add` once) for
**every** mission worktree, under a lock, with a cleanliness proof.

### Architecture

**Components:**

1. **`mission-worktree.sh acquire / release`** (extends the existing helper; it already has
   identity-checked owner records, process-group isolation and a bounded background writer).
   - **Who owns a slot (revised after quorum round 1).** The owner is **the fire, not a
     process that touches the slot**. The `acquire` call, the Bash tool call that ran it and the
     sub-agent working in the tree are all short-lived or in other process groups, and between
     tool calls no process may have its cwd in the tree. A lock keyed on any of them would read
     "owner dead" while a live role is still using the slot. The process that does live exactly
     as long as the iteration is the **driver**: `mission-control.sh` computes
     `MISSION_FIRE_ID="${MISSION_NAME}-$(date +%s)-$$"` (L2431, exported), launches the
     controller in the background and `wait`s on it (L2689, L2725), and keeps its own PID across
     controller retries. So the lock records `fire_id`, the driver PID parsed from it, that PID's
     birth time (`ps -o lstart`, as `birth()` already does) and the role.
     `acquire` **refuses** without `MISSION_FIRE_ID` (`failed:no_fire_owner`); there is no
     fallback owner.
   - `acquire MISSION ROLE SHA [BRANCH]` → prints a slot path, or `busy` / `failed:<why>`.
     1. Pick the first free `<role>-<n>` under the slot root. A slot is free only when its lock
        is absent, or its recorded driver is **proven dead**: the PID is gone, or alive with a
        different birth time. Inconclusive means held, as in `identity()`. A slot held by a live
        fire is never reclaimed, **including by the same fire**: a second acquire for the same
        role gets `<role>-2`. Independently, a slot with any process cwd inside (the sweep's
        `lsof` rule) is busy. That is a second guard against orphans like a stray `go test`, not
        the ownership test.
     2. Take the lock: atomic `mkdir <slot>.lock`, then write `owner` (`fire_id`, driver PID,
        birth, role) via tmp + `mv`, as `register()` does. **Stealing a dead lock (r2):**
        `<slot>.lock` already exists, so `mkdir` alone cannot arbitrate between two acquirers
        that both saw a dead owner. The steal is `mv <slot>.lock <slot>.lock.dead.<own-pid>`.
        `rename(2)` of one source succeeds for exactly one mover; the loser's `mv` fails and
        it moves on to the next slot. The winner re-reads the moved `owner` and checks it is
        still the dead identity it judged; if not, it moves it back and goes on. Then it does the
        normal `mkdir`. Two acquirers that both find no lock are already arbitrated by `mkdir`.
     3. **All tree work runs in the existing bounded background writer (r2).** First creation
        (`worktree add`, about 2 minutes) and the reset of an old slot can both outlive one
        foreground Bash tool call. So `acquire` takes the lock, launches the `_writer` (detached,
        `MISSION_WORKTREE_TIMEOUT`) and prints `pending <path>`. The recipe then polls
        `mission-worktree.sh wait <path> 45` until `ready` / `failed:*`, exactly as for `add`
        today. One protocol for both cases; no new timeout surface.
     4. If the slot exists, the writer **dirty-checks first** (D1). On dirt it salvages to
        `~/.ailang/state/slots/<slot>/salvage-<ts>/`:
        - `HEAD`, the branch name;
        - **staged** changes `git diff --binary --cached`;
        - **unstaged** changes `git diff --binary` (index → worktree; the two diffs are kept
          separate so "stage B, then restore A in the file" loses nothing, which the round-2
          reviewer showed `git diff HEAD` alone would);
        - a byte copy of the worktree's `index` file;
        - a tar of untracked files.

        It logs one line. Then `git checkout --detach --force <SHA>` → `git clean -ffdx` → if a
        BRANCH was given, `git switch -c <BRANCH>`. A name that already exists is a refusal,
        never `-C`: an existing branch can hold unpushed commits.
     5. **Prove it.** `git rev-parse HEAD` must equal SHA and `git status --porcelain` must be
        empty. Otherwise `failed:not_clean_after_reset`, the slot is deleted (the next acquire
        recreates it), and the caller parks, exactly as with `add` today.
   - `release-fire FIRE_ID` → drops every lock whose `fire_id` matches. The trees stay in place
     (that is the point). **Only the driver calls it**, from its own `EXIT` trap. The driver is
     one persistent bash process, so the trap fires when the fire ends, not when a tool call
     returns. A recipe never releases.
   - A driver killed with `SIGKILL` runs no trap; its locks are reclaimed by step 1's
     dead-driver proof on the next acquire, with salvage.
   - `add` / `wait` keep working, so missions not yet migrated are unaffected.

2. **Gate wiring.** `gate-3-route.md` (sprint, planner, judge, evaluator), `gate-4-record.md`
   (record) and `ref-drift.md` replace `WT="$(cd .. && pwd)/.wt-…"` + `mission-worktree.sh add`
   with `acquire` followed by the existing `wait` loop (Component 1 step 3). The recipe writes
   `WT` into the iteration's state file so later tool calls and sub-agents re-read the path
   instead of re-acquiring. **No `trap` in any recipe**: release is the driver's job
   (Component 1). The "measure `origin/dev` immediately before, use the full SHA" rule
   (gate-3 L653–694) is unchanged: `acquire` takes that SHA.

3. **Sweep: keep the pin prefix, add the slot root (revised r2).** Round 1 narrowed protection
   to a registry of exact pin paths. Round 2 showed that opens a window where a live pin is
   sweep-visible: a pin created before its registry write, or one of the seven existing pins
   whose mission has not fired since the change. **Dropped.** Slots move every per-iteration
   tree *out* of `~/.ailang-driver-pin/`, so that prefix again holds only pins and can stay
   protected as it is. Changes:
   - add the slot root to `PROTECT` in `worktree_sweep.sh:53` and `git_health.sh:71` (slots are
     lifecycle-managed by `acquire`);
   - `acquire` **refuses** a slot root inside any protected prefix (`failed:slot_root_protected`),
     so trees cannot hide there again;
   - the leftover per-iteration trees under the pin prefix are a one-time attended migration
     (69 + 48 already done 2026-11-28; the remaining 35 are November work);
   - the sweep gains a `--repo <path>` repeat flag, and `git_health.sh` sweeps each clone named
     in a fixed list in the hook (`ailang`, `ailang-world`, `ailang-docs`, `ailang-motoko`), so
     `ailang-world` is covered.

   No pin is ever more exposed than it is today.

4. **Mission GOCACHE (D3).** `mission-control.sh` exports `GOCACHE=$HOME/.cache/ailang-mission-go`
   next to `MISSION_FIRE_ID` (L2431). That is after the pin re-exec (`pin-root.sh` is sourced at
   L1248 and `exec`s the pinned driver at its L362, same PID), so the export, the fire id and the
   `EXIT` trap are all set in the pinned process (V14). At fire start it records `du -sk` of that cache in the iteration record, and
   posts one notice if it is over the threshold. **No automatic `go clean -cache`**: a concurrent
   mission's `go test` would lose its cache mid-build (seen live 2026-11-28: a world-mission
   `go test ./host/verifygate/` was running during the clean-up). Go's 5-day trim plus stable
   paths bound it.

### Implementation Plan

**Phase 1: slots in the helper** (~1 day)
- [ ] `acquire` / `release` in `tools/launchd/mission-worktree.sh` reusing `identity()`,
      `register()` and the bounded writer
- [ ] Salvage-then-reset (D1), the cleanliness proof, `busy` / `failed:*` outputs
- [ ] Tests TST-SLOT-01..05, 07, 08, 12, 13, 14 in `tools/launchd/test_mission_worktree.sh` (bash 3.2, rig
      `find` is bfs, BSD tools)

**Phase 2: gate wiring + GOCACHE** (~0.5–1 day)
- [ ] gate-3 / gate-4 / ref-drift recipes call `acquire` / `release`. The skill text keeps the
      half-built-tree rules, now scoped to "first creation of a slot".
- [ ] `mission-control.sh`: `release-fire` EXIT trap next to `MISSION_FIRE_ID` (L2431)
- [ ] `mission-control.sh`: mission `GOCACHE` export and size alarm (if D3 = yes)
- [ ] TST-SLOT-06 measured on the rig

**Phase 3: sweep + migration** (~0.5 day)
- [ ] `worktree_sweep.sh` / `git_health.sh`: add the slot root to `PROTECT` (pin prefix unchanged); `--repo` for the fixed clone list;
      `acquire` refuses a slot root under a protected prefix (TST-SLOT-09, 10)
- [ ] One attended migration run: sweep `ailang` + `ailang-world` with the new rules (dry run
      first, log kept), and list the remaining KEEP rows for a human

### Files to Modify/Create

**Modified files:**
- `tools/launchd/mission-worktree.sh` - `acquire` / `release`, salvage, cleanliness proof (~+120 LOC; the file is ~200 now)
- `tools/launchd/test_mission_worktree.sh` - TST-SLOT-01..08 (~+150 LOC)
- `tools/launchd/lib/pin-root.sh` - **no change** (r2: registry withdrawn)
- `tools/launchd/mission-control.sh` - mission `GOCACHE` export + size line in the record, `release-fire` EXIT trap (~+25 LOC)
- `scripts/worktree_sweep.sh` - slot root added to `PROTECT`, `--repo` (~+25 LOC)
- `scripts/test_worktree_sweep.sh` - TST-SLOT-10 + `--repo` case (~+40 LOC)
- `scripts/hooks/git_health.sh` - add slot root to the protect list, sweep the fixed clone list (~+15 LOC)
- `.claude/skills/mission-control/resources/gate-3-route.md` - recipes → `acquire` / `release` (~±40 lines)
- `.claude/skills/mission-control/resources/gate-4-record.md` - record worktree → `acquire` (~±10 lines)
- `.claude/skills/mission-control/resources/ref-drift.md` - same (~±5 lines)

## Conflict Surface (shared machinery)

Not a parser or typechecker change, but D4 and Component 1 change machinery other lanes rely on
(quorum trigger 2).

| Shared thing | Who else relies on it | Decision | What must still work |
|---|---|---|---|
| `mission-worktree.sh add` / `wait` | every mission's gate-3 recipe; world/stapledon via `MISSION_DRIVER_ROOT` | **reuse**: add verbs; `add` / `wait` unchanged | existing `test_mission_worktree.sh` cases pass unmodified (TST-SLOT-11) |
| sweep protection rule | attended SessionStart (`git_health.sh`), `test_worktree_sweep.sh` | **reuse**: pin prefix kept; slot root *added* | "protected mission pin → kept" test (L66) passes unchanged; no pin is more exposed than today |
| driver pin refresh in `pin-root.sh` | every launchd mission | **reuse**, untouched (r2) | `tools/launchd/test_pin_root.sh` passes unchanged |
| per-path Claude Code project state (`~/.claude/projects/<path>`) | mission roles whose cwd is a worktree | **changes**: one path per slot instead of per iteration | a role's auto-memory from a previous iteration in the same slot becomes visible to the next one. Accepted: mission roles are unattended and read the charter, not memory. Noted as a risk |
| "worktree must be a sibling of `$REPO`" (gate-3 L235) | the planner recipe | **override** for slots: the root is fixed by D2 | the stated reason (derive, don't hardcode, never `/tmp`) still holds |

## Examples

**Before (gate-3 sprint worktree):**
```bash
WT="$(cd .. && pwd)/.wt-${MISSION_NAME}-iter<N>-<item>"
bash "$MW" add "$BRANCH" "$newsha" "$WT"          # full 31k-file checkout, new path, cold Go cache
until out=$(bash "$MW" wait "$WT" 45); [ "$out" != pending ]; do :; done
[ "$out" = ready ] || { echo "PARK worktree_incomplete: $out"; exit 1; }
# …never removed: sits under ~/.ailang-driver-pin/ forever
```

**After:**
```bash
# one Bash tool call; owner = this fire's driver (MISSION_FIRE_ID), not this shell
out=$(bash "$MW" acquire "$MISSION_NAME" exec "$newsha" "$BRANCH")   # returns at once: "pending <path>" | busy | failed:*
case "$out" in "pending "*) WT=${out#pending };; *) echo "PARK slot: $out"; exit 1;; esac
until r=$(bash "$MW" wait "$WT" 45); [ "$r" != pending ]; do :; done   # same loop as add today
[ "$r" = ready ] || { echo "PARK slot_incomplete: $r"; exit 1; }
echo "exec_wt=$WT" >> "$ITER_STATE"      # later calls and sub-agents read the path from here
# same path every iteration → ~1% of files rewritten, Go cache hits for unchanged packages
```

**Driver (`mission-control.sh`, once, near L2431):**
```bash
trap 'bash "$MC_DRIVER_ROOT/tools/launchd/mission-worktree.sh" release-fire "$MISSION_FIRE_ID"' EXIT
```

## Success Criteria

- [ ] Acquire on an existing slot moves HEAD to the exact SHA with an empty `status --porcelain`, without a full checkout (TST-SLOT-01, 02)
- [ ] A dirty slot is salvaged to state, then reset; the salvage restores byte-identically (TST-SLOT-03)
- [ ] A slot held by a live fire is never reset, even with no process inside and the acquiring shell gone; a dead driver's slot is reclaimed with salvage; an unreadable birth counts as held (TST-SLOT-04, 05)
- [ ] The driver's EXIT trap releases exactly its own fire's locks; acquire without a fire owner refuses (TST-SLOT-12, 13)
- [ ] An existing branch name is refused, never force-moved (TST-SLOT-07)
- [ ] A reset that does not prove clean deletes the slot and reports `failed:not_clean_after_reset` (TST-SLOT-08)
- [ ] Second build on an unchanged SHA in the same slot writes ≈0 cache bytes (TST-SLOT-06, rig measurement)
- [ ] A slot root under a protected prefix is refused; slots under the slot root are kept by the sweep; the sweep covers every listed clone (TST-SLOT-09, 10)
- [ ] Two acquirers racing to steal the same dead lock: exactly one wins, the other gets the next slot or `busy` (TST-SLOT-14)
- [ ] Existing `add` / `wait` behaviour unchanged (TST-SLOT-11)
- [ ] After two weeks live: ailang + ailang-world worktree counts ≤ pins + slots; mission GOCACHE under its threshold (manual, recorded in the mission log)

## Testing Strategy

| ID | Level | Given / When / Then |
|----|-------|---------------------|
| TST-SLOT-01 | integration | Given no slot, when `acquire m exec SHA` runs, then a worktree is created at `<root>/<repo>/m/exec-1` with HEAD=SHA and empty porcelain |
| TST-SLOT-02 | integration | Given slot exec-1 at SHA1 and SHA2 differing in K files, when `acquire … SHA2` runs, then HEAD=SHA2, porcelain is empty, and only ≈K files' mtimes changed |
| TST-SLOT-03 | integration | Given exec-1 with (a) a modified tracked file, (b) an untracked file, and (c) a change B **staged** while the working file was restored to HEAD's A, when acquired, then `salvage-<ts>/` holds the cached diff, the worktree diff, the `index` copy and the untracked tar, and applying them to a fresh tree at the old HEAD reproduces (a)–(c) byte-identically, including B in the index; the slot is clean at SHA |
| TST-SLOT-04 | integration | *(real topology)* Given a stand-in driver D (a long-lived process whose PID is in `MISSION_FIRE_ID`) and exec-1 acquired by a **separate shell A that has exited**, with **no process cwd inside exec-1**, when a shell B in another process group runs acquire, then B gets exec-2 (or `busy`) and exec-1 is untouched (sha of a tracked file + an untracked marker unchanged) |
| TST-SLOT-05 | integration | Given the TST-SLOT-04 setup, when D is killed (no trap runs), then the next acquire reclaims exec-1 with salvage; when D's PID is reused by a process with a different birth, reclaimed; when birth cannot be read, held |
| TST-SLOT-12 | integration | Given D holding exec-1 and record-1, when D exits normally through its `EXIT` trap, then `release-fire` drops both locks, the trees stay, and a lock owned by another fire is untouched |
| TST-SLOT-13 | integration | Given `MISSION_FIRE_ID` unset, when acquire runs, then it fails `failed:no_fire_owner` and creates no lock or tree |
| TST-SLOT-14 | integration | Given exec-1 locked by a dead driver, when two acquirers in separate process groups start together (barrier), then exactly one `mv` of `exec-1.lock` succeeds, exactly one acquirer owns exec-1, the other gets exec-2 or `busy`, and exec-1 was reset at most once (salvage dir count = 1) |
| TST-SLOT-06 | e2e (rig) | Given exec-1 built once at SHA, when `release` + `acquire … SHA` + `go build ./...` run again, then the cache gains ≈0 bytes (`du -sk` before/after; control: a fresh path gains ≥1 GB) |
| TST-SLOT-07 | integration | Given branch B exists, when `acquire … SHA B` runs, then it fails `failed:branch_exists` and B's ref is unchanged |
| TST-SLOT-08 | integration | Given a reset that leaves porcelain non-empty (injected), when acquire finishes, then the slot dir is gone and the output is `failed:not_clean_after_reset` |
| TST-SLOT-09 | integration | Given a slot root configured under `~/.ailang-driver-pin/` (or any `PROTECT` prefix), when acquire runs, then it fails `failed:slot_root_protected` and creates no lock or tree |
| TST-SLOT-10 | integration | Given an idle, clean slot under the slot root, when the sweep runs `--apply`, then it is kept (lifecycle-managed) |
| TST-SLOT-11 | integration | *(regression surface)* Given the existing `test_mission_worktree.sh` cases, when run unmodified, then all pass |

Tests run under `/bin/bash` 3.2 with BSD tools (rig constraints: no `declare -A`, bfs `find`, no
GNU `timeout`), and real `git` only in temp repos.

## Deferred Decisions

- D5 `-trimpath` — agent may add it **only** after running the 15 `runtime.Caller` test files
  (`grep -rln runtime.Caller --include='*_test.go'`) green under `-trimpath`. Slots make it
  unnecessary for the cache problem.
- D6 slot counts per role — agent chooses; start with exec×2, plan×1, record×1, judge×2 and adjust from `busy` counts.
- Whether `git clean` excludes heavy ignored dirs (e.g. `node_modules` for the docs mission) — agent may add `-e` per mission if measured.
- Retention for `salvage-*` dirs — agent may pick (proposal: 14 days, then the sweep deletes).

## Non-Goals

- Automatic `go clean -cache`: unsafe with concurrent missions (see Component 4).
- Spotlight exclusion of slot / cache dirs: needs System Settings or sudo; documented for the operator instead.
- Changing the driver-pin mechanism itself (`pin-root.sh` already reuses its tree).
- Coordinator task worktrees (`~/.ailang/state/worktrees/`): their own lifecycle; untouched.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A reset destroys a crashed iteration's uncommitted work | High | D1 salvage before reset; branches (and their commits) live in the common repo and survive the reset; `-C` is never used |
| A stray process (e.g. an orphaned `go test`/`godot`) still has cwd in the slot | Med | `lsof` cwd check makes the slot busy; never reset under a live process |
| State leaks between iterations via ignored files | Med | `git clean -ffdx` + the porcelain proof; ignored build outputs are removed every time |
| Shared per-path Claude project memory across iterations | Low | Accepted (see Conflict Surface); revisit if a role starts reading memory |
| Concurrency between roles of one iteration | Med | One lock per slot, N slots per role; `busy` parks, never waits unbounded (see memory: unbounded waits wedge the loop) |

## Quorum

Triggers fired: **1** (design-freeze items D1–D4) and **2** (overrides the shared sweep
protection rule).

**Round 1 (2026-11-28): BLOCKED 3/3.** `oc-kimi-k3`, `codex:gpt-6.2-astra` and
`gemini-4-carbon` all raised the same objection, and it was correct: the lock owner was the
short-lived `acquire` process, and the example's recipe-level `trap … EXIT` would have released
the slot when the tool call returned. With no cwd inside between tool calls, a concurrent acquire
would have reset a live role's tree. **Fixed:** the owner is the fire's driver (V11), release is
`release-fire` from the driver's own `EXIT` trap, recipes never release, and TST-SLOT-04/05/12/13
model the real acquire-in-shell-A / use-in-session-B topology. Artifact:
`.ailang/state/mission-quorum/m-mission-worktree-slots-2026-11-28T14-41-18Z.json`.
**Round 2 (2026-11-28): BLOCKED 3/3** on new objections, all addressed in r3 below. Per the re-quorum-ONCE guardrail no round 3 was run; **the doc goes to the human for ratification** with these resolutions:
- `oc-kimi-k3`: the registry narrowing could expose a live pin to the sweep → **registry withdrawn**; pin-prefix protection kept as is; slots live outside it and `acquire` refuses to put them inside (Component 3, TST-SLOT-09).
- `codex:gpt-6.2-astra`: `git diff HEAD` loses "stage B, restore A" → salvage keeps the cached and worktree diffs separately plus a copy of `index` (step 4, TST-SLOT-03).
- `gemini-4-carbon`: (1) GOCACHE "before re-exec" contradicted V11 → corrected; exports and trap sit after the pin re-exec (V14), and the three trap-setting libs are executed, not sourced (V13). (2) steal race on an existing `.lock` → `rename(2)`-arbitrated steal (step 2, TST-SLOT-14). (3) first creation exceeds the foreground tool limit → `acquire` always returns `pending` and the existing `wait` loop polls (step 3).

Artifact: `.ailang/state/mission-quorum/m-mission-worktree-slots-2026-11-28T14-47-23Z.json`.

## Related Documents

**Planned (check for overlap):**
- [design_docs/planned/m-mission-runtime-contract.md](design_docs/planned/m-mission-runtime-contract.md) (0.36): runtime contract for missions; no worktree lifecycle overlap found
- [design_docs/planned/m-mission-iteration-sprint-plan.md](design_docs/planned/m-mission-iteration-sprint-plan.md) (0.38)

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-coord-stable.md](design_docs/implemented/v0_6_2/m-coord-stable.md) (0.31): coordinator worktree lifecycle (kept separate; non-goal)

No related doc scored ≥ 0.45; there is no duplicate.

## Verification Log

| # | Claim | Verified by | Result |
|---|-------|-------------|--------|
| V1 | Go keys the cache on the package dir without `-trimpath` | `grep -n 'dir %s' $(go env GOROOT)/src/cmd/go/internal/work/exec.go` | L299, guarded by "-trimpath is not set" (L296) |
| V2 | Go trims unused entries after 5 days | `grep -n trimLimit …/cache/cache.go` | `trimLimit = 5 * 24 * time.Hour` (L350) |
| V3 | Sweep protects the whole driver-pin prefix | `sed -n 53p scripts/worktree_sweep.sh`; `git_health.sh` L71 | confirmed, both sites |
| V4 | Fleet throwaway trees land inside the pin prefix | `git worktree list`: 133 of 162 under `~/.ailang-driver-pin`; pin-root L260 sets the pin path; gate-3 L679 derives siblings of `$REPO` | confirmed |
| V5 | Sweep never runs unattended or outside the session's clone | `git_health.sh` L22–23 (exit on `AILANG_MISSION_STAGE` / `AILANG_TASK_ID`), L70 `cd "$ROOT"` | confirmed; ailang-world has 191 registered trees |
| V6 | Pin already uses checkout-in-place reuse | `pin-root.sh` L272–275 | `git checkout --quiet --detach --force "$target"` |
| V7 | 15 test files use `runtime.Caller` | `grep -rln runtime.Caller --include='*_test.go' . \| grep -v .claude/worktrees \| wc -l` | 15 |
| V8 | No existing slot/reuse verb in mission-worktree.sh | `grep -n -E 'acquire\|release\|reuse' tools/launchd/mission-worktree.sh` | no verbs beyond `add`, `wait`, internal `_writer` / `_checkout` (the two "release" hits are comments about the writer handshake) |
| V9 | Daily churn is ~1% of files | `git diff --name-only $(git rev-list -1 --before='24 hours ago' origin/dev) origin/dev \| wc -l` | 337 / 31,441; world 107 / 7,496 |
| V10 | Mission overlap guard is per mission, not per role | `mission-control.sh` L2690 pidfile | per-mission only, so per-slot locks are needed |
| V11 | The driver outlives the controller and every role, and keeps its PID across retries | `mission-control.sh` L2431 (`MISSION_FIRE_ID=…-$$`, exported), L2672–2689 (controller backgrounded with `&`), L2725 (`wait "$CONTROLLER_PID"`) | confirmed; `exec` happens only inside the `( … ) &` child subshell |
| V12 | Recipes run as separate, ephemeral Bash tool calls | quorum round 1 (all three reviewers); gate-3 L733 already requires `run_in_background` + polling because one call cannot span the work | confirmed: a recipe-level `trap … EXIT` would release immediately |
| V13 | `mission-control.sh` has no existing `trap` the new `EXIT` trap would replace | `grep -c -E "trap " tools/launchd/mission-control.sh` | 0 (the only hit for `trap\b` is a prose comment at L776); a sourced lib that sets one later must chain, not replace |
| V14 | `MISSION_FIRE_ID`, the GOCACHE export and the new trap are set in the pinned driver | `mission-control.sh` L1247–1248 source `pin-root.sh`, which `exec /bin/bash "$wt/tools/launchd/$script"` (pin-root L362, same PID); `MISSION_FIRE_ID` is assigned later at L2431 | confirmed: after re-exec |
| V15 | No sourced lib sets a trap | `grep -l -E '\btrap ' tools/launchd/lib/*.sh` → `idle-fingerprint.sh`, `run-suites.sh`, `triage-fingerprint.sh`; mission-control.sh runs these via `/bin/bash …` (L1740, L1787, L2451), never `.`; the sourced set (L944–950, 1248, 2102, 2660–2664) has none | confirmed |

---

**Document created**: 2026-11-28
**Last updated**: 2026-11-28
