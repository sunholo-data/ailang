# Fleet i119 — executor fix round r2 (unowned-dirty handler tests)

Worktree: `/Users/voightkampff/.ailang-driver-pin/.fleet-i119-build`, branch `fleet/i119-unowned-dirty`, base HEAD `1feb66036`.
Scope: tests only. `unowned-dirty.ts` was not edited.

## Change

Appended two handler tests to `.pi/extensions/.unowned-dirty.test.ts`:

1. `handler: own edit through a symlinked cwd matches git's physical toplevel` — cwd is a symlink to the repo; git's `rev-parse` reports the physical toplevel; asserts exactly one warning naming `theirs.go`. Targets m7 (`[...ownFiles].map(realOrSelf)` removed).
2. `handler: outside a git repo it neither reads git status nor warns` — `rev-parse` returns code 128; asserts the only exec call is `rev-parse` and no notification is sent. Targets m10 (`if (top.code !== 0) return;` removed).

Imports: no changes needed. `test`, `assert`, `mkdtempSync`, `mkdirSync`, `symlinkSync`, `writeFileSync`, `tmpdir`, `join`, `unownedDirtyExtension` (default export), and `realOrSelf` were already imported.

Both new tests passed on the unmutated code on the first run, so no expectation was adjusted.

## Commands and results

| Step | Command | rc | Result |
|---|---|---|---|
| Unit gate (unmutated) | `cd .pi/extensions && node --test .unowned-dirty.test.ts` | 0 | pass 8, fail 0 |
| m7 | scratch `/tmp/fleet_i119_r2_m7`: `perl -pi -e 's/\[\.\.\.ownFiles\]\.map\(realOrSelf\)/[...ownFiles]/'` | 1 | pass 7, fail 1 (`handler: own edit through a symlinked cwd…`) |
| m10 | scratch `/tmp/fleet_i119_r2_m10`: `perl -0pi -e 's/\t\tif \(top\.code !== 0\) return;[^\n]*\n//'` | 1 | pass 7, fail 1 (`handler: outside a git repo…`) |
| Original drill | `/bin/bash /tmp/fleet_i119_mutants.sh <worktree>` | 0 | all six mutants fail >= 1 (table below) |
| Pi test target | `make -C <worktree> test-pi-extensions` | 0 | all suites pass; unowned-dirty 8/8 |

Both m7 and m10 were confirmed as real mutations: `cmp` reported a difference from the original, and `grep -c "top.code !== 0"` on the m10 copy returned 0.

## Mutant table (original drill, `fleet_i119_mutants.sh`)

| Mutant | fail | pass | Failing tests |
|---|---|---|---|
| M1_no_prefix_strip | 4 | 4 | relative-path handler; symlink handler; wiring (absolute own paths); wiring (symlink) |
| M2_realOrSelf_identity | 2 | 6 | symlink handler; wiring (symlink) |
| M3_callsite_reverted | 2 | 6 | relative-path handler; symlink handler |
| M4_top_not_trimmed | 2 | 6 | relative-path handler; symlink handler |
| M5_early_return_inverted | 3 | 5 | relative-path handler; non-repo handler; symlink handler |
| M6_always_append_slash | 1 | 7 | wiring (absolute own paths) |

All six still fail >= 1.

## Deviations and notes

- The first m10 attempt was in the same shell invocation as m7 and aborted on zsh's `PIPESTATUS` (not a valid variable in zsh). The m7 result above came from the first invocation. m10 was rerun alone and its result is the one recorded here. No code was affected.
- The `make test-pi-extensions` run also prints a `fatal: not a git repository` line from `.workspace-trust.test.ts`. That line comes from an existing test and is not caused by this change; that suite passes 13/13.
- `unowned-dirty.ts` was not touched. Scratch copies live only under `/tmp/fleet_i119_r2_m7` and `/tmp/fleet_i119_r2_m10`.
