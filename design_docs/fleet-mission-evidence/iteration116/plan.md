# Fleet i116 plan — land the D-FLEET-18 A guard arm ALONE (fresh branch from origin/dev)

Planner: 2026-11-16, read-only against pin `b7941bbb5` (= origin/dev). Throwaway checks ran in
`/tmp/fleet_i116_plancheck` (removed afterwards).

## Findings (evidence)
1. **Ruling is already on origin/dev.** `design_docs/fleet-mission.md:165` (dev) and `:161`
   (`origin/fleet/i115-record` 566263001) carry D-FLEET-18 `RESOLVED`, "ANSWERED — A ... attended
   2026-11-16". The same cell still says "The row stays OPEN; the ask is unchanged" (stale prose
   inside the row, not this PR's job). It also cites `make/test.mk:117` and `:128,140-141`. Those
   lines are now `:136` (the target) and `:147-148` (the `bash -n` lines for the probe pair). Fix
   both in the record, not in the arm PR.
2. **The arm applies unchanged.** merge-base(875581f46, origin/dev) = `aebc3dcbb`.
   `git log aebc3dcbb..origin/dev` returns 0 commits for `tools/launchd/githooks/pre-push` and
   0 for `tools/launchd/test_mission_scope_guard.sh`. `design_docs/fleet-mission.md` has 15 commits,
   all record or ruling edits, and the Authority block is unchanged: dev `:79-87` matches the base
   and i115-record byte for byte. `git cherry-pick --no-commit 875581f46` on origin/dev gave rc 0
   ("Auto-merging design_docs/fleet-mission.md", no conflict, 4 files, +16). The patched suite
   passed all 3 PASS lines in about 1 s.
3. **The arm matches the ruling exactly.** It adds one `case` arm to `_scope_is_fleet_support`
   (pre-push `:29-36`, inserted after `:32`). The arm matches the two literal paths only, with no glob.
   Verdict logic: `_scope_verdict` `:50-63`. Fleet is refused unless the path is harness
   (`:14-23`) or support. For product loops, only harness paths are refused (`:60`). So support
   placement admits the fleet and leaves the product loops' access unchanged. Measured
   before → after with `_scope_verdict`:
   | mission | motoko_connection_probe.sh | test_motoko_connection_probe.sh | qwen38-motoko-rerun.sh |
   |---|---|---|---|
   | fleet | refuse → **allow** | refuse → **allow** | refuse → refuse |
   | motoko / v1 / docs / attended("") | allow → allow | allow → allow | allow → allow |
   The charter line it adds (between `make/test.mk` and `tools/pi-extensions/sandbox/**` at dev
   `:84-85`) follows the D-FLEET-5 and D-FLEET-6 annotation style. No rewording is needed.
4. **Landing precondition met.** The PINNED guard (dev `pre-push`) admits all 4 diff paths for
   `MISSION_NAME=fleet`. `tools/launchd/*` and `design_docs/fleet-mission*` are harness (`:16`,
   `:20`), `changelogs/*` is support (`:31`), and the fragment name dated 2026-11-16 is also
   admitted. Measured via `MISSION_SCOPE_LIB=1 . pre-push; _scope_verdict fleet <p>`, the same call
   the suite's `check` helper makes (test `:16-20`).
5. **Push mechanics.** For a new remote branch, `remote_sha`=0, so the base is
   `merge-base(local, origin/dev)` (pre-push `:80-86`). On a fresh branch from current origin/dev,
   the diff is exactly the 4 paths. Rebasing and force-pushing `fleet/i83-dfleet18-scope-arm`
   would diff from old SHA 875581f46. That diff would sweep in about 684 dev commits, including the
   `internal/types` core paths from #2363, and be refused. The brief's i115 (a) premise holds.
6. **The test arms leave 3 mutants alive (new finding).** Mutation drill run in the throwaway, 15 mutants:
   12 are killed by the existing ten arms. These 3 SURVIVE:
   M12 `tools/eval/*test_motoko_connection_probe.sh` (path-2 basename-prefix glob),
   M13 `*/motoko_connection_probe.sh` (path-1 any-directory), M14 `*/test_motoko_connection_probe.sh`.
   Three added arms kill all three, verified with the suite still green on the correct hook:
   ```
   check fleet tools/eval/xtest_motoko_connection_probe.sh refuse # kills: path 2 basename-prefix glob
   check fleet scripts/motoko_connection_probe.sh refuse # kills: path 1 directory wildcard
   check fleet scripts/test_motoko_connection_probe.sh refuse # kills: path 2 directory wildcard
   ```
   Insert them directly after `check fleet tools/eval/xmotoko_connection_probe.sh refuse ...`.
7. PR #2110 is still OPEN and draft at 875581f46. Its title and body say PROPOSED. Close it as
   superseded after landing. `fleet/i116*` does not exist on origin yet.

## Executor steps (fresh worktree; never touch the pin; never --no-verify)
```bash
cd ~/.ailang-driver-pin/fleet && git fetch -q origin dev
W=/tmp/fleet_i116_arm; git worktree add -b fleet/i116-dfleet18-scope-arm "$W" origin/dev
cd "$W" && git cherry-pick --no-commit 875581f46          # expect rc 0, 4 paths staged
git mv changelogs/unreleased/2026-11-04-fleet-scope-arm-dfleet18.md \
       changelogs/unreleased/2026-11-16-fleet-scope-arm-dfleet18.md
# edit the fragment (below); add the 3 arms (finding 6) to tools/launchd/test_mission_scope_guard.sh
/bin/bash tools/launchd/test_mission_scope_guard.sh        # 3 PASS lines
make check-changelog                                       # fragment shape (unpiped!)
git diff --cached --name-only origin/dev                   # exactly 4 paths, nothing under tools/eval/
```
Fragment text (date it today; no longer PROPOSED):
```
### Changed — Fleet scope guard admits the two Motoko probe files (D-FLEET-18, ruled A 2026-11-16)

The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
exactly `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`,
per Mark's attended ruling D-FLEET-18 = A (2026-11-16). They are support paths, not harness, so product
missions (v1, motoko, docs...) keep their access; every other `tools/eval/` path stays refused for
the fleet. Guard arm, Authority line and 13 regression arms only; under the iteration-16 rule the
#13 probe flake fixes follow in a separate change once this lands on the pinned `origin/dev` guard.
```
Commit message: drop "PROPOSED, pending ruling" and "Do not merge before the ruling". State "ruled A
2026-11-16 (attended)". Keep the Co-Authored-By trailer. Then run the done-gate (below),
`git push -u origin fleet/i116-dfleet18-scope-arm` (from the mission env, which goes through the
pinned guard), open a PR, wait for checks with a bounded `date +%s` deadline, and merge. After the
merge: `gh pr close 2110 --comment "superseded by #<new>"`. Deleting the old branch is optional,
and a delete pushes no content (pre-push `:79`).

## Mutation drill (each must turn the suite red at the named line; restore with cp + cmp, never git checkout)
| # | mutant in pre-push | killed by check |
|---|---|---|
| M1 | drop path 1 from the arm | `fleet tools/eval/motoko_connection_probe.sh allow` |
| M2 | drop path 2 | `fleet tools/eval/test_motoko_connection_probe.sh allow` |
| M3 | move pair to `_scope_is_harness` | `motoko tools/eval/motoko_connection_probe.sh allow` |
| M4 | `tools/eval/*)` | `fleet tools/eval/qwen38-motoko-rerun.sh refuse` |
| M5 | `motoko_connection_probe.sh*` | `fleet ...motoko_connection_probe.sh.bak refuse` |
| M6 | `test_motoko_connection_probe.sh*` | `fleet ...test_motoko_connection_probe.sh-x refuse` |
| M7 | `tools/eval/*motoko_connection_probe.sh` | `fleet tools/eval/xmotoko_connection_probe.sh refuse` |
| M8 | `tools/eval/*motoko*)` | `fleet tools/eval/qwen38-motoko-rerun.sh refuse` |
| M9 | add `tools/eval/*` to harness | `motoko tools/eval/motoko_connection_probe.sh allow` |
| M10 | pair classed `_scope_is_core` | `fleet tools/eval/motoko_connection_probe.sh allow` |
| M11 | delete the whole new line | `fleet tools/eval/motoko_connection_probe.sh allow` |
| M12 | `tools/eval/*test_motoko_connection_probe.sh` | NEW `fleet tools/eval/xtest_motoko_connection_probe.sh refuse` |
| M13 | `*/motoko_connection_probe.sh` | NEW `fleet scripts/motoko_connection_probe.sh refuse` |
| M14 | `*/test_motoko_connection_probe.sh` | NEW `fleet scripts/test_motoko_connection_probe.sh refuse` |
| M15 | `tools/eval/*_probe.sh)` | `fleet tools/eval/xmotoko_connection_probe.sh refuse` |
All 15 were measured red in the planner's throwaway, with M12-M14 checked against the 3 new arms.
The executor re-runs them on the real branch.

## Done-gate
1. `/bin/bash tools/launchd/test_mission_scope_guard.sh`: 3 PASS lines (verdicts, e2e env-only
   hooksPath push, driver wiring). The e2e arm is what exercises the surface that actually runs.
2. `make test-launchd-drivers` (~330 s, unpiped, bounded wait). It `bash -n`s the probe pair
   (`make/test.mk:147-148`) and runs the scope suite (`:118`). The pair is unchanged here, so its
   known flake (#13) may appear. Compare any red arm against origin/dev before calling it this PR's.
3. `make check-changelog`.
4. Dry-run. The driver (`mission-control.sh`) is NOT edited, and the hook only runs at `git push`, so
   a driver dry-run cannot see this change. It is charter compliance and reach hygiene, not
   discrimination. Run it once healthy and once degraded (`MISSION_EXECUTOR_MODEL=codex:bogus`,
   driver `:2074-2075`), with `AILANG_DRIVER_PINNED=<new branch sha>`, under an idle, armed sibling.
   At plan time, `triage` had neither `mission-triage.disabled` nor `mission-triage.pid` in `~/.ailang/state`.
   v1 (`mission-control.disabled`), docs and motoko are disabled. Fleet and stapledon hold pidfiles.
   Re-check at run time. If none is idle, use the charter's synthetic-HOME `MISSION_PROFILE=fleet`
   recipe. Record the profile that ran.
5. Reach (after merge): refresh the pin, then
   `MISSION_SCOPE_LIB=1 . <pin>/tools/launchd/githooks/pre-push; _scope_verdict fleet tools/eval/motoko_connection_probe.sh`
   must print nothing (allow). Until this is true, no #13 probe change may be pushed.

## Risks
- Scope creep: any `tools/eval/*` edit in this PR would be refused by the pinned guard (it is not
  admitted until the merge) and would break the ships-ALONE rule. Keep the PR to 4 paths.
- Stale `origin/dev` ref at push time widens the merge-base diff. Fetch before cutting the branch.
- `fleet-mission.md` edits from the parallel record branch (#2370, i115-record) touch other
  sections. The Authority block is identical there, so the merge order does not matter, but
  rebase the record after this lands.
- The test-launchd-drivers flake is the very #13 this ruling unblocks. A red probe arm is
  pre-existing unless it reproduces only on this branch.
- Do not reload any launchd job mid-iteration (done-gate line 5 of the skill).
