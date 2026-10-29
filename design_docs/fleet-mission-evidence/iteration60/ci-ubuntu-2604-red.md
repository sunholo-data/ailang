# Iteration 60 — CI infra red evidence (the merge park)

## What blocks the merge

PR #1942 (candidate `dcab63a3c006bc5455bd93b3d124e52e695e887c`, content-identical twin of the
judged iteration-57 head `6268fb0b8`) cannot merge: the **required** `test` check fails on the PR
head, twice, identically, and the failure is not caused by the candidate.

## The failing tests (both runs, identical)

Run 40258947803 job 121366866585 (attempt 1) and rerun job 121370813839 (attempt 2, `--failed`):

- `TestRunPythonSolution_WiresCliArgs` — stderr: `Failed to download https://github.com/astral-sh/python-build-standalone/releases/download/20261021/cpython-3.12.15%2B20261021-x86_64-unknown-linux-gnu-install_only_stripped.tar.gz` → `tunnel error: failed to create underlying connection` → `Connection refused (os error 111)`
- `TestRunPythonSolution_WiresStdin` — same signature
- `TestRunPythonSolution_NoSpecInputsIsBenign` — same signature
- `TestPythonRunner` — same signature
- Package verdict: `FAIL github.com/sunholo-data/ailang/internal/eval_harness`

`Build ubuntu-latest` (Build and Release run 40258947775) fails with the **identical** four-test
signature; the macos/windows matrix legs are fail-fast casualties (cancelled).

## Why the download is refused — by design

`.github/workflows/ci.yml` "Run tests with timeout (M-DX11)" sets, deliberately (hermeticity guard):
`HTTP_PROXY=http://127.0.0.1:9`, `HTTPS_PROXY=http://127.0.0.1:9`, `NO_PROXY=localhost,127.0.0.1`,
`GOPROXY=off`, and asserts all four. Any test that must download at run time cannot.

## Why this started between 10-23 and 10-26 — the runner image

- Dev's green `test` run on the SAME Go code (`bf22f4943`, CI run 40024034450, 2026-10-23) ran on
  **`ubuntu-24.04`** (image string in job log) and shows `--- PASS: TestPythonRunner (0.03s)` — the
  preinstalled python3.12 satisfied `uv run --python 3.12` (pin is `PinnedPythonVersion() == "3.12"`,
  `internal/eval_harness/python.go`), so nothing downloaded.
- Both failing PR runs ran on **`ubuntu-26.04`** (image string in job log). No satisfying Python is
  preinstalled there, so `uv` must download the pinned build-standalone — and the poison proxy
  refuses it.

GH's `ubuntu-latest` moved 24.04 → 26.04 between 2026-10-23 and 2026-10-26. The candidate touches
**zero** Go files (`git diff --name-only origin/dev..HEAD` = 7 shell/docs paths), so the code under
test is byte-identical to dev's green head. This is the "runner-image change un-hiding a latent
bug" class: the hermeticity guard has always forbidden the download; the old image made the
download unnecessary.

## Rule-3d attribution (negative controls)

- Same check, same tests, green on dev head 10-23 (control fired: `PASS: TestPythonRunner (0.03s)`
  in dev job 120666198313's log), red on the PR 10-26 twice — with identical code. Not the
  candidate's doing.
- Not the pricing red: dev's required checks are ALL green at `bf22f4943` (check-runs read
  2026-10-26 ~02:16Z: test/lint/build/docs-gate/UI build gate = success). D-FLEET-17's inherited
  red is repaired upstream; this is a NEW, different red.
- Deterministic on the image: two failures, identical bytes in stderr; a third rerun would be
  p-hacking the runner pool.

## Why the fleet cannot fix it

The fix needs `.github/workflows/ci.yml` (e.g. install/cache the uv-managed Python 3.12 in an
earlier step, before the poisoned test step) — CI workflows are explicitly outside the fleet's
write authority (charter Authority: "Anything else is refused, including CI workflows"; the
per-fire pre-push scope guard enforces the allowlist). `internal/eval_harness` is likewise outside
the allowlist. → parks for Mark as D-FLEET-30; handed to the repo owner (mission-v1) on the
cross-mission channel.

## Worked-around nothing, forced nothing

No admin merge, no bypass of branch protection, no scope-guard change. PR #1942 and PR #1932 stay
open; backup branch `fleet/i60-before-refresh` preserves the old head `6268fb0b8`. All landing
evidence is banked so the next fire (or the first fire after the workflow fix) lands without
re-deriving it: designer CONFIRMED, planner ADDENDUM OK, executor PASS (21/21 + 17/17, zsh arms
green), evaluator CODE ACCEPT 93/100, `env -i make test-launchd-drivers` rc 0 outside the sandbox,
dry-run healthy `lanes=ok` AND degraded `DEGRADED(1)` at `AILANG_DRIVER_PINNED=dcab63a3c`.