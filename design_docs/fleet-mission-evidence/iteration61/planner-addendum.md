PLANNER VERDICT: ADDENDUM OK

# Landing Addendum — M-FLEET-ZSH-SAFE-MISSION-LIBS, fleet iteration 61 (planner role)

Ticket `quorum-reader:zsh-nomatch`. This is not a new plan. Plan + sprint JSON are unchanged on `origin/fleet/i60-zsh-libs-land`
(`bd9a41cf3` plan, `dcab63a3c` impl, base `bf22f4943`, PR #1942). The only thing that changed since iteration 60's park:
dev now carries the CI Python prefetch, so the inherited `test` red is gone. Remaining work = refresh, gates, merge, resolve, close.
All commands were run read-only from the pin (`/Users/voightkampff/.ailang-driver-pin/fleet`, detached at `c8e348d08`), with `/bin/bash -c` where zsh would misparse.

## A. Verification I ran (command → output)

1. `git fetch -q origin`; `git rev-parse origin/dev origin/fleet/i60-zsh-libs-land HEAD` → dev `c8e348d08736e6b273efb73175ae706ce82eef62`,
   branch `dcab63a3c006bc5455bd93b3d124e52e695e887c`, HEAD = dev. `git ls-remote origin` → `refs/heads/dev c8e348d08…`, `fleet/i60-zsh-libs-land dcab63a3c…`, no `fleet/i61*` ref yet.
2. `git log --oneline bf22f4943..origin/dev` → exactly 2 commits: `c8e348d08 build(deps): bump fast-uri … in /docs (#1943)` (dependabot; `--stat` = `docs/package-lock.json` only),
   `d8e47684c Maintenance instrument … (#1940)`. `git merge-base origin/dev origin/fleet/i60-zsh-libs-land` → `bf22f4943`.
3. Branch files: `git diff --name-only origin/dev...origin/fleet/i60-zsh-libs-land` → 7 paths: sprint JSON, `changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md`,
   plan md, `tools/launchd/lib/{pi-ext-args,quorum-artifacts}.sh`, `tools/launchd/test_{pi_ext_args,quorum_artifacts}.sh`.
4. **Intersection (item 2)**: `comm -12 <(git diff --name-only bf22f4943 origin/dev | sort) <(git diff --name-only origin/dev...origin/fleet/i60-zsh-libs-land | sort) | wc -l` → **0**.
   **Positive control**: same command with `.github/workflows/ci.yml` appended to the right-hand list → prints `.github/workflows/ci.yml` (instrument can see a hit).
   Extra: `git diff --name-only bf22f4943 origin/dev -- tools/launchd` → 0 lines; `-- tools/ .agents/ .claude/ changelogs/` → only `tools/pi-extensions/*` (4) and two new
   changelog fragments (`2026-10-23-maintenance-adapter.md`, `2026-10-26-ci-python-prefetch.md`). The Gate-2 skill copies read by arm `zsh_gate2_export_only` are untouched;
   `scripts/` (other `test-launchd-drivers` suites) not in dev's diff.
5. Trial merge, no writes: `git merge-tree bf22f4943 origin/dev origin/fleet/i60-zsh-libs-land | grep -cE '^(changed in both|added in both|removed in)'` → **0** (cherry-pick will be clean).
6. **#1940 prefetch (item 3)**: `git show d8e47684c -- .github/workflows/ci.yml` → two hunks adding, after "Set up uv", step
   `name: Prefetch Python for offline eval-harness tests` / `timeout-minutes: 5` / `run: uv python install 3.12`, commented "Prepare the eval runner's pin before tests poison
   network access. Keep this request in sync with PinnedPythonVersion in python.go." Hunk 1 sits in job `test` (`runs-on: ubuntu-latest`, ci.yml ~L96–123); hunk 2 in `test-windows`.
   `git show d8e47684c -- .github/workflows/build.yml` → same step added (build.yml L61, job `build`, matrix) — this also covers iteration 60's `Build ubuntu-latest` red.
   `git grep '"3.12"' origin/dev -- '*.go'` → `internal/eval_harness/python.go:23 // PinnedPythonVersion returns "3.12".` (pin and prefetch agree).
7. **Controller CI claim**: `gh run view 40288893796` → push/dev at `d8e47684c`, completed/failure; jobs: test, lint, docs, build, launchd drivers (bash 3.2), float determinism,
   govulncheck = success; **test-windows = failure**. `gh api …/jobs/121459024865` (test) → labels `ubuntu-latest`, step 5 "Prefetch Python…" success, step 14 "Run tests with timeout (M-DX11)" success.
   Job log grep → `Image: ubuntu-26.04`, `Installed Python 3.12.15 in 640ms`, `--- PASS: TestRunPythonSolution_WiresCliArgs / _WiresStdin / _NoSpecInputsIsBenign`, `--- PASS: TestPythonRunner (0.02s)`,
   `ok github.com/sunholo-data/ailang/internal/eval_harness`. test-windows log → `--- FAIL: TestShadowGradeHangIsUngradedTimeout (11.73s)`, `FAIL …/cmd/ailang`.
   `gh run list --commit d8e47684c…` → `40288894078 Build and Release push success`. **Claim CONFIRMED.**
8. `gh api repos/sunholo-data/ailang/branches/dev/protection/required_status_checks` → `{"contexts":["test","lint","build","docs-gate","UI build gate"],"strict":false}` — test-windows not required. CONFIRMED.
9. `gh run view 40296349998` → workflow_dispatch on dev `c8e348d08`, **in_progress** at read time (changes/float/govulncheck success; test/lint/launchd/test-windows pending).
10. `gh pr view 1942/1932 --json …` → #1942 OPEN, head `dcab63a3c`, MERGEABLE, 0 comments; #1932 OPEN, head `6268fb0b8` (`fleet/i57-zsh-safe-libs`), 0 comments.
    `gh api repos/…/issues/{1942,1932}/comments --jq length` → `0`, `0` (baseline counts for step 5).
11. Old-head blob SHAs (`git ls-tree -r origin/fleet/i60-zsh-libs-land -- <7 files>`), the expected values for the identity check:
    `acf7e58c6bfd` sprint JSON · `7a5f85b49eee` changelog · `8455c67d9d1e` plan · `a4a9acf0389c` lib/pi-ext-args.sh · `adaead480660` lib/quorum-artifacts.sh ·
    `f51408cb012b` test_pi_ext_args.sh · `7b11a4e0f94b` test_quorum_artifacts.sh.
12. `ailang mission ticket resolve --help` → `usage: ailang mission ticket resolve <signature> --resolution TEXT [--sha SHA]` (charter `design_docs/fleet-mission.md:105` matches).
    Log line 1617 (iteration 58): scope guard refused force-with-lease on the old branch → new branch + replacement PR. Fleet merges land as merge commits (`git log --merges`: #1941, #1937, #1936).

## B. Ordered landing steps (executor/controller)

0. **Preflight**: `gh auth status` = `sunholo-voight-kampff`; re-read `git ls-remote origin refs/heads/dev`. If dev moved past `c8e348d08`, redo A4 (intersection + control) and A5 against the new tip before continuing; any non-empty intersection = stop and replan.
1. **Refresh — yes, as a NEW branch** (iteration-58 precedent; never force the old branch). In an isolated worktree off the pin: `git worktree add --detach <wt> origin/dev`;
   `git cherry-pick bd9a41cf3 dcab63a3c` (expect clean, A5); push `HEAD:refs/heads/fleet/i61-zsh-libs-land` through the active per-fire scope guard (guard not edited/disabled);
   `gh pr create --base dev --head fleet/i61-zsh-libs-land` titled as the fix, body citing ticket, supersedes #1942 and #1932, the iteration-57 judge (PASS 94 @ `6268fb0b8`) and iteration-60 evaluator (CODE ACCEPT 93 @ `dcab63a3c`).
   Retain `fleet/i60-zsh-libs-land`, `fleet/i57-zsh-safe-libs`, `fleet/i60-before-refresh`.
2. **Identity checks the executor must show (verbatim output)**:
   - `git range-diff bf22f4943..dcab63a3c origin/dev..<new-head>` → `1: bd9a41cf3 = 1: <new>` and `2: dcab63a3c = 2: <new>` (both `=`; any `!` = stop).
   - `git diff --name-only origin/dev...<new-head>` → exactly the 7 paths in A3.
   - Per-file blob compare, all 7: `for f in <7 paths>; do a=$(git rev-parse dcab63a3c:"$f"); b=$(git rev-parse <new-head>:"$f"); [ "$a" = "$b" ] && echo "IDENTICAL $f" || echo "DIFF $f"; done` under `/bin/bash`
     → 7× IDENTICAL, matching A11's prefixes. Rev-parse both SHAs in the same block (never hand-expand).
   - Evidence chain: `6268fb0b8` (judged) ≡ `dcab63a3c` (evaluated, iteration 60) ≡ `<new-head>` by `=` patches + identical blobs ⇒ the iteration-57/60 review verdicts carry; no source rescoring.
3. **Focused tests** at `<new-head>`, outside the sandbox, under `/bin/bash` (plan's exact form):
   `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh` → rc 0, all 5 zsh arms PASS (`zsh_quorum_export_only/local_only/absent/options`, `zsh_gate2_export_only`);
   `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh` → rc 0, `17 passed, 0 failed` (incl. `zsh_pi_empty/two_ts/options`). No `SKIP (no /bin/zsh)` lines on Darwin.
   Then `make check-changelog` rc 0 (dev added 2 fragments; must still pass) and `git diff --check origin/dev...<new-head>` rc 0.
4. **Controller done-gate** (mission-loop-change five-line pre-flight; shapes from iteration-60 `source-identity-and-lanes.md`):
   a. `env -i HOME="$HOME" PATH="$PATH" make test-launchd-drivers` at `<new-head>`, OUTSIDE the sandbox → rc 0, final line `launchd drivers: tests + bash 3.2 syntax OK`.
   b. Dry-run healthy: idle, armed sibling profile (stapledon precedent; verify pidfile absent before/after), `AILANG_DRIVER_PINNED=<new-head> MISSION_PROFILE=stapledon MISSION_DRY_RUN=1` → rc 0, `lanes=ok`.
   c. Dry-run degraded: same + a bogus executor model (`MISSION_EXECUTOR_MODEL=codex:bogus MISSION_PROBE_TIMEOUT=10`) → rc 0, `lanes=DEGRADED(1)` naming the lane + fallback. Unset ambient `MISSION_WORKDIR` (iteration-60 cosmetic leak).
   d. No plist/binary/mission reload; shell-only change reaches missions via pinned `origin/dev` re-exec, so the merge IS the reach. Confirm no mission mid-fire before merging.
5. **CI gate (exact head, then exact merge)**: bounded poll (≤30 min, fail loudly on expiry) of check-runs on `<new-head>` until the required set `test, lint, build, docs-gate, UI build gate` is all
   `success` (also expect `launchd drivers (bash 3.2)` green — it runs the changed zsh arms). A red `test-windows` with `TestShadowGradeHangIsUngradedTimeout` is dev-inherited (A7), non-required:
   record it, do not block on it, do not rerun-to-green it. Any required red = park (never force).
   Merge: `gh pr merge <N> --merge --match-head-commit <new-head full SHA>`; record merge SHA via `git ls-remote origin refs/heads/dev` + `git rev-parse`.
   Then poll the push-triggered `CI` and `Build and Release` runs whose `headSha` == merge SHA (not a later dev head); required jobs success. Optionally record outcome of dispatched run 40296349998 (dev `c8e348d08`) as the base control.
6. **Ticket resolve** (only after 5 green): `ailang mission ticket resolve quorum-reader:zsh-nomatch --resolution "zsh NOMATCH abort fixed: setopt local_options null_glob guard in mc_quorum_lookup and _mc_pi_ext_args; landed via PR #<N> (refresh of #1942/#1932)" --sha <merge sha on origin/dev>`.
   Verify open-ticket count drops by exactly one and the auto-reply reaches the filer (stapledon). Clause-2 <48h window closed 2026-10-24 — no turnaround claim.
7. **Close superseded PRs** (after merge): for each of #1942 then #1932: `gh pr comment <n> --body "Superseded by #<N>, merged as <merge sha>; identical patches (range-diff =, 7/7 blobs identical)."`;
   verify `gh api repos/sunholo-data/ailang/issues/<n>/comments --jq length` went **0 → 1** (A10 baseline) before `gh pr close <n>` (no `--delete-branch`); verify `state: CLOSED`.

## C. What the refresh could invalidate (plan acceptance criteria)

- **Plan/JSON content**: none — both are byte-identical blobs (gate 2). The plan's "At HEAD `d25d9b8d0`" and sprint JSON base-gate logs are historical evidence, not landing criteria.
- **Final gates are base-dependent and must be re-run, not carried**: `make test-launchd-drivers` (whole-suite; dev changed none of its suites per A4, but rc is a property of the tree),
  `make check-changelog` (dev added fragments), `git diff --check`. The iteration-60 rc 0 at `dcab63a3c` does NOT transfer to `<new-head>`.
- **"Existing make wiring reaches zsh arms on macos-latest without CI edits"**: dev changed `.github/workflows/ci.yml`; confirm the `launchd drivers (bash 3.2)` job on `<new-head>` is green and its log shows the zsh arm names (not SKIP).
- **Mutation evidence** (15 mutants, iteration 57) carries only through the blob-identity proof; if any blob differs, the review evidence is void and a fresh evaluator is required.
- **Out-of-scope `.github/workflows`**: still honored — the CI fix arrived via dev (#1940), not this branch; `git diff --name-only` must show no workflow path.
- Sprint JSON top-level `"status": "pending"` remains landing bookkeeping (record commit), not a blocker.

Verdict: ADDENDUM OK — the two new dev commits touch none of the 7 files (intersection 0, control fired), the trial merge is conflict-free, and the
iteration-60 blocker is removed upstream (prefetch verified in source and in the ubuntu-26.04 job log). The one open item is that dispatched run 40296349998 was still in progress when I read it; the exact-head PR run decides the merge regardless.
