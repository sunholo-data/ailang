# Iteration 119 — executor report r1

Plan: `design_docs/fleet-mission-evidence/iteration119/plan.md`
Worktree: `/Users/voightkampff/.ailang-driver-pin/.fleet-i119-build`, branch `fleet/i119-unowned-dirty`, base `022f68f1a`.
Inputs checked before applying: `/tmp/fleet_i119_proposed.diff` sha256 `8a6dbb3c…e2e9df` and `/tmp/fleet_i119_handler_test.diff` sha256 `d0eefb88…a99a8f` (both match the plan).

## Step 1 — defect reproduction at HEAD (before applying anything)

Command (run from `.pi/extensions`, as the plan writes it):
```bash
cd /Users/voightkampff/.ailang-driver-pin/.fleet-i119-build/.pi/extensions && node -e '
import("./unowned-dirty.ts").then(({parsePorcelain, unownedDirty}) => {
  const cwd = "/repo", p = "mine.ts";
  const own = new Set([p.startsWith("/") ? p : `${cwd}/${p}`]);
  console.log("own=", [...own], "unowned=", JSON.stringify(unownedDirty(parsePorcelain(" M mine.ts\n M theirs.go\n"), own)));
})'
```
Output: `own= [ '/repo/mine.ts' ] unowned= ["mine.ts","theirs.go"]` (rc=0). The session's own `mine.ts` is wrongly reported. Matches the plan's expected output.

## Step 2 — apply patches with `git apply`

```
git apply --check /tmp/fleet_i119_proposed.diff && git apply /tmp/fleet_i119_proposed.diff    -> rc=0
git apply --check /tmp/fleet_i119_handler_test.diff && git apply /tmp/fleet_i119_handler_test.diff -> rc=0
git status --short -> M .pi/extensions/.unowned-dirty.test.ts ; M .pi/extensions/unowned-dirty.ts
```
Post-apply `unowned-dirty.ts` sha256: `0b196ae1f6fe9b0dffc2cc48dd1c3d2e4bc016df0ad88bb2d1f2111812147265`.

## Step 3 — changelog fragment

Written verbatim from the plan to `changelogs/unreleased/2026-11-17-pi-unowned-dirty-relative-paths.md` (heredoc, rc=0).

## Step 4 — gates (each run unpiped; rc is the command's own)

| Gate | Command | rc | Key output |
|---|---|---|---|
| G1 | `node --test <worktree>/.pi/extensions/.unowned-dirty.test.ts` (run from `.pi/extensions`) | 0 | `ℹ pass 6`, `ℹ fail 0` (plan: pass 6 / fail 0) |
| G2 | `make -C <worktree> test-pi-extensions` | 0 | Last row `.pi/extensions/.unowned-dirty.test.ts` → `ℹ pass 6`, `ℹ fail 0`; every file row shown has `ℹ fail 0` |
| G3 | `git apply --check -R --include='.pi/extensions/unowned-dirty.ts' /tmp/fleet_i119_proposed.diff` and `git apply --check -R /tmp/fleet_i119_handler_test.diff` | 0 / 0 | both silent, rc=0 (source equals the reviewed diff) |
| G4 | `make -C <worktree> check-changelog` | 0 | `✓ 32 changelog fragment(s) valid; changelogs/v0.32-current.md has '## [Unreleased]'` / `✓ CHANGELOG.md is index-only and links changelogs/v0.32-current.md` |
| G5 | `make -C <worktree> verify-pi-assets` | **2 (EXPECTED FAIL)** | `Files .pi/extensions/unowned-dirty.ts and cmd/ailang/pi_assets/unowned-dirty.ts differ` / `DRIFT: cmd/ailang/pi_assets is stale vs .pi/extensions — run 'make pi-assets'` / `make: *** [verify-pi-assets] Error 1` |
| G6 | `git -C <worktree> diff --name-only` | 0 | `.pi/extensions/.unowned-dirty.test.ts`, `.pi/extensions/unowned-dirty.ts` only; nothing under `cmd/`. The changelog fragment is untracked and shows in `git status` |

G5 is the attended half, as the plan says. `make pi-assets` was not run and nothing under `cmd/` was touched.

## Step 5 — mutation drill

Command: `/bin/bash /tmp/fleet_i119_mutants.sh /Users/voightkampff/.ailang-driver-pin/.fleet-i119-build` (rc=0). All six mutants applied (no `MUTATION DID NOT APPLY`).

| Mutant | Result | Red tests (planner expected) |
|---|---|---|
| M1 no prefix strip | `fail 3` / `pass 3` | handler; wiring absolute-own-paths; wiring symlinked-cwd — matches (fail 3) |
| M2 realOrSelf identity | `fail 1` / `pass 5` | wiring symlinked-cwd — matches |
| M3 call site reverted (HEAD defect) | `fail 1` / `pass 5` | handler — matches |
| M4 top not trimmed | `fail 1` / `pass 5` | handler — matches |
| M5 early return inverted | `fail 1` / `pass 5` | handler — matches |
| M6 always append slash | `fail 1` / `pass 5` | wiring absolute-own-paths — matches |

Every mutant fails at least one test, with the test the plan named.

## Step 7 — post-drill checks

- G1 re-run unpiped, rc=0, `ℹ pass 6`, `ℹ fail 0`.
- G3 re-run after the drill: both checks rc=0, so the worktree source still equals the reviewed diff.
- `git -C <worktree> diff --stat`: `.pi/extensions/.unowned-dirty.test.ts | 43 +++++-`, `.pi/extensions/unowned-dirty.ts | 23 +++++-`, 2 files changed, 64 insertions(+), 2 deletions(-).

## Deviations from the plan

1. **G1 was first run with an absolute test path** (from inside `.pi/extensions`, as the plan's `cd` step) rather than the bare `.unowned-dirty.test.ts`. Same file, same result, and the brief requires absolute paths.
2. **The first post-drill G1 re-run was piped** to `grep` and the rc was lost. I re-ran it unpiped (writing output to `/tmp/i119_g1b.txt`) and recorded rc=0. The G1b result above is the unpiped one.
3. **No deviation in the patches, changelog text, or the drill.** Both patches applied with `git apply` exactly as the plan wrote them; no hand edits.

## Verdict

All gates behave as the plan predicts. G1–G4 and G6 pass. G5 fails as expected and is the attended handoff, not an executor defect. All six mutants are killed.

Handoff line for the controller: attended: run `make pi-assets` on `fleet/i119-unowned-dirty`, push, merge.
