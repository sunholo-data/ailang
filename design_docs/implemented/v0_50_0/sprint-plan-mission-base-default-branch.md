# Sprint plan: mission-base derives the default branch (ticket `mission-base:hardcoded-origin-dev`)

**One milestone (M1). Mechanical fix. Executor: work only in `/Users/voightkampff/dev/sunholo-data/.wt-fleet-iter9`.**

## Problem

`tools/launchd/mission-base.sh:10` has `REF="${MISSION_BASE_REF:-origin/dev}"`. Stapledon's repo
(`stapledons-godot`) has default branch `main` and no `origin/dev`, so every Gate-1
`mission-base.sh record gate1` fails with `mission-base: cannot resolve origin/dev` (5 slots lost).
Rig measurement: `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` is rc 0 in all six
mission clones (five → `origin/dev`, stapledon → `origin/main`). Deriving from `origin/HEAD` is correct.
Verified in a scratch repo: with no `origin/HEAD`, that command is rc 1 with empty output; after
`git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main` it prints `origin/main`, with no remote configured.

## Files in scope (touch nothing else)

1. `tools/launchd/mission-base.sh`
2. `tools/launchd/test_mission_base.sh`
3. `changelogs/unreleased/2026-09-30-mission-base-default-branch.md` (new)
4. Optional: `.claude/skills/mission-control/resources/ref-drift.md`, one sentence only (see step 4).

Do NOT touch the relative `bash tools/launchd/...` call sites (parked ticket D-FLEET-8). No network:
no `git fetch` and no `git ls-remote`. The code must be bash 3.2.57-safe: no associative arrays, no `${v,,}`, no `timeout`.

## M1 steps

### 1. `mission-base.sh`: add `resolve_ref`, resolve once

- Line 2 header: replace `record the shared origin/dev reading` with
  `record the shared default-branch reading (MISSION_BASE_REF, else origin/HEAD's target)`.
- Line 10: replace `REF="${MISSION_BASE_REF:-origin/dev}"` with `REF=""` plus a comment saying it is
  set once by `resolve_ref` in the dispatcher.
- Add the following function above `snap()`:

```bash
resolve_ref() {  # explicit MISSION_BASE_REF wins; else origin/HEAD's symbolic target; else FAIL (no silent default)
  if [ -n "${MISSION_BASE_REF:-}" ]; then REF="$MISSION_BASE_REF"; return 0; fi
  REF=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null) && [ -n "$REF" ] && return 0
  echo "mission-base: cannot derive the base ref: refs/remotes/origin/HEAD is unset and MISSION_BASE_REF is empty." >&2
  echo "mission-base: fix: export MISSION_BASE_REF=origin/<default-branch>, or run 'git remote set-head origin --auto' in this repo." >&2
  return 1
}
```

- Dispatcher: for `snap`, `record` and `drift`, call `resolve_ref || exit 1` **before** the
  subcommand, and after the label-arg check. `last` reads only the state file, so it must not resolve
  and must not fail in a repo without `origin/HEAD`. Example:
  `record) [ $# -ge 2 ] || {…; exit 2; }; resolve_ref || exit 1; record "$2" ;;`
- `snap()` and `drift()` keep using `$REF`. Do NOT call `resolve_ref` inside `snap`: `record` runs
  `$(snap)` in a subshell and must see the same `REF`. The single-read invariant in `record` (exactly
  one `rev-parse`) is unchanged.
- `drift()` line 41: make the silent `|| return 1` loud:
  `|| { echo "mission-base: cannot resolve $REF" >&2; return 1; }`.
- Forbidden: any `|| echo origin/dev`, `:-origin/dev` or `main`/`master` guessing. Resolution is
  exactly the order above.

### 2. `test_mission_base.sh`: update the fixture

- `scratch_clone()` (line 25): after `git update-ref refs/remotes/origin/dev HEAD`, add
  `&& git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/dev`. Update the comment on lines 17-18.
  All 8 existing arms must stay green unchanged.
- Add a helper `scratch_clone_main()`: the same as above, but it creates `refs/remotes/origin/main`
  and `origin/HEAD` → `refs/remotes/origin/main`, and **no** `origin/dev`.

### 3. New test arms (append before the summary block, same `check` style)

| # | Arm name | Fixture and assertion | Mutation that turns it red |
|---|---|---|---|
| T1 | `derived-main-record` | `scratch_clone_main`; `snap` rc 0; `record gate1` rc 0; recorded SHA == `git rev-parse refs/remotes/origin/main`; `last gate1` returns it | M-a: revert line 10 / `resolve_ref` to `origin/dev` (does not resolve: rc 1) |
| T2 | `override-beats-origin-head` | `scratch_clone_main` (A), then commit B and `git update-ref refs/remotes/origin/release HEAD`; `MISSION_BASE_REF=origin/release record gate1`; recorded SHA == B and != A | M-c: ignore `MISSION_BASE_REF` (records A) |
| T3 | `no-origin-head-fails-loudly` | Clone with `origin/dev` present but **no** `origin/HEAD` (`git symbolic-ref --delete refs/remotes/origin/HEAD` on `scratch_clone`); unset MISSION_BASE_REF; `record gate1` rc 1; stderr contains `MISSION_BASE_REF` and `set-head`; state file `$t/mission-test-base` absent | M-b: silent `\|\| echo origin/dev` fallback (resolves via the existing origin/dev: rc 0) and M-a |
| T4 | `drift-uses-derived-ref` | Clone where `origin/dev` and `origin/main` are both at A and `origin/HEAD` → `origin/main`; `record gate1`; commit B and `update-ref refs/remotes/origin/main` (leave origin/dev at A); `drift gate1` rc 1, output contains `DRIFT base gate1` and B's SHA | M-d: `drift` hard-codes/rev-parses `origin/dev` (steady: rc 0). M-a also reds it |
| T5 | `last-needs-no-ref` | Clone with no `origin/HEAD`; pre-write a `base-gate1` row; `last gate1` rc 0 and prints the row's SHA | Moving `resolve_ref` to the top level (before the dispatcher) makes `last` fail |

Run every arm unset: invoke each with `env -u MISSION_BASE_REF` or prefix `MISSION_BASE_REF=`
(empty counts as unset in `resolve_ref`), so a stray export cannot mask T1, T3 or T4.

**Mutation proof (required; report it in the executor summary).** For each mutation M-a, M-b, M-c
and M-d, apply it to a **scratch copy**: `cp tools/launchd/mission-base.sh /tmp/mb.sh`, edit it,
and point `HELPER` at it via an env override such as `HELPER="${MISSION_BASE_HELPER:-$ROOT/...}"`.
Run the suite and record which arms go red. **Never** `git checkout <file>` to revert. Each of T1-T4
must go red under at least its named mutation.

### 4. Docs

- Changelog fragment `changelogs/unreleased/2026-09-30-mission-base-default-branch.md`. Its first line is
  `### Fixed — mission-base.sh hard-coded origin/dev; now derives the base from origin/HEAD (2026-09-30)`,
  followed by 3-5 lines covering: the stapledon (`main`) Gate-1 failures (5 slots), the resolution order,
  the loud failure naming both remedies, and no silent fallback.
- Optional `ref-drift.md`: after line 3's first sentence, add "(For each mission the base ref is
  `MISSION_BASE_REF` or its clone's `origin/HEAD` target: `origin/dev` for ailang repos, `origin/main`
  for stapledon.)". Do not make other edits.

## Acceptance criteria

- [ ] `grep -n 'origin/dev' tools/launchd/mission-base.sh` matches only comments, if anything; there is no code default.
- [ ] Resolution: MISSION_BASE_REF, else `origin/HEAD`, else rc 1 with both remedies on stderr, and no state row written.
- [ ] `snap`, `record` and `drift` resolve once per invocation, and `drift` compares against the same ref `record` uses.
- [ ] `last` works with no `origin/HEAD`.
- [ ] All 8 existing arms plus T1-T5 pass. Each of M-a through M-d reds at least one named arm (mutation log in the summary).
- [ ] The file is bash 3.2-clean: `/bin/bash -n` passes, and there are no `declare -A`, `${x,,}` or `timeout` uses.
- [ ] The changelog fragment passes `make check-changelog`.

## Gate commands (the controller runs these unpiped)

```bash
cd /Users/voightkampff/dev/sunholo-data/.wt-fleet-iter9
/bin/bash -n tools/launchd/mission-base.sh
/bin/bash tools/launchd/test_mission_base.sh
env -i HOME=$HOME PATH=$PATH make test-launchd-drivers
make check-changelog
```
