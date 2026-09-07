# M-COORDINATOR-WINDOWS-PACKAGE-TIMEOUT-HEADROOM: A Derived `-timeout` Budget and a Per-Package Headroom Instrument

**Status**: **REVISION 2 — design quorum round 2 BLOCKED (three rejects); this is the round-2 answer.**
Round 1 returned three rejects (all answered in the previous revision); round 2 returned three
more, again none disputing the design DIRECTION (a derived budget plus a headroom instrument). All
three round-2 objections are completeness/determinism defects with concrete, reviewer-authored
fixes, spread across three DIFFERENT surfaces — **provisioning** (`gpt6-astra`), **wiring**
(`gemini-3-1-pro`), and **conflict-surface completeness** (`oc-glm-5-2`) — rather than localising
on one. No reviewer flipped to pass. Because the objections did not localise onto one surface and
each is a single-surface, single-fix defect, the disposition is a **BOUNDED REVISION** applying
the reviewers' verbatim fixes, not a redesign and not a decomposition. This revision answers all
three in place. Written against base `1b8e7eab1` (the controller's worktree HEAD for this
iteration). The queue row's framing ("the coordinator package's baseline has grown 40%") is engaged
with and partly corrected below: the coordinator is **not** the package that blows next, and the
failing run was a whole-runner slowdown, not a coordinator regression. The durable half of this
design is the headroom instrument, not the budget number.

**Quorum log (round 1 → round 2).** Round 1 returned three rejects, each landing on a distinct
surface; all were answered in the previous revision. Round 2 returned three more, again each on a
distinct surface. The surface split is recorded here per the mission's quorum protocol so the
controller can decide whether this doc needs decomposition rather than another revision. All six
objections across both rounds are single-surface, single-fix defects; no decomposition is
warranted.

**Round 1 (answered by the previous revision):**

| Reviewer | Surface | Objection (one line) | Answered by (previous revision) |
|---|---|---|---|
| `gemini-3-1-pro` | **wiring** | Linux `set -e` aborts before `cat go_test_headroom.log`; a red job with no test logs | `|| go_rc=$?` guard in the Linux wiring + M3; pwsh audit (V16–V18) |
| `oc-glm-5-2` | **instrument robustness** | silent-fallback: a zero-package parse exits 0 with no error signal | runtime anti-vacuity guard (non-empty log + 0 packages → `::error::` + non-zero exit) + package-count floor |
| `gpt6-astra` | **measurement epistemics** | budget rests on a right-censored measurement; 1.34x is a lower bound, not "conservative" | budget relabelled a provisional operational budget; instrument ships WARN-ONLY; calibration follow-up row |

**Round 2 (answered by THIS revision):**

| Reviewer | Surface | Objection (one line) | Answered by (this revision) |
|---|---|---|---|
| `gpt6-astra` | **provisioning** | the instrument is never BUILT: Linux executes the source dir as a program, Windows invokes `headroom.exe` with no step creating it; the doc's own acceptance commands are unrunnable | explicit build step before the test suite in both jobs, output to `$RUNNER_TEMP`; every invocation updated; clean-checkout build+invoke acceptance rows |
| `gemini-3-1-pro` | **wiring** | both shells still abort before the log is printed: pwsh `$ErrorActionPreference='Stop'` + `$PSNativeCommandUseErrorActionPreference` make a failing `go test` throw; bash `set -e` is unprotected at the headroom call site | pwsh prepends `Continue` + `$false`; bash `hr_rc=0; ... || hr_rc=$?`; full step-body sweep; V16/V18 UNVERIFIED-LOCALLY, verified by the PR's own `test-windows` CI run |
| `oc-glm-5-2` | **conflict-surface completeness** | V14's search (`grep -rln 'headroom\|budget'`) is too narrow — it would only match a tool already doing this design's job, not an existing `tools/ci/` tool that parses `go test` output and could be extended | V14 broadened into a new Verification Log row (both commands + control); Conflict Surface row stating branch (a): `tools/ci/` has only `motoko_smoke.sh`, no `go test` parser, so a new tool is warranted |

**Disposition note.** The three round-2 objections did NOT localise onto one surface
(provisioning / wiring / conflict-surface completeness) and no reviewer flipped to pass, so the
disposition is a **bounded revision** applying the reviewers' verbatim fixes, not a decomposition.

**Target**: v0.36.0
**Priority**: P1 (a CI flake that reds the `test-windows` job on `dev`; the queue row is the top
NEXT row for iteration 348)
**Estimated**: 2 days (M1: 0.5d, M2: 0.5d, M3: 1d)
**Dependencies**: None. Touches `.github/workflows/ci.yml`, `internal/cihygiene/`, and a new
`tools/ci/headroom/` tool. Does **not** touch `internal/coordinator` production code.

---

## Problem

`.github/workflows/ci.yml`, job `test-windows`, runs `go test -timeout 300s ./...`. Go's
`-timeout` is a **per-test-binary** (i.e. per-package) wall-clock budget — `go help testflag`
states "If a test binary runs longer than duration d, panic" — and `go test ./...` runs up to `-p`
package binaries **concurrently** (`-p` defaults to GOMAXPROCS, i.e. the 4 cores of a
`windows-latest` runner). Each package's reported wall time therefore includes time spent
contending with its siblings for those 4 cores.

On 2026-09-07 the job failed. The controller's `gh run view <run> --job <job> --log` for the
`test-windows` job on run `34145572524` (commit `72f9cfeca`) shows:

```
panic: test timed out after 5m0s
FAIL	github.com/sunholo-data/ailang/internal/coordinator	300.520s
```

The panic names ONE test on the stack that had been running 2 seconds — i.e. the package
exhausted its binary-wide budget **cumulatively**, it did not hang. The queue row's framing is
true but incomplete: the coordinator's baseline has grown, but that is not the mechanism, and it
points at the wrong package for "what blows next".

### The controller's measurements (verified first-party where noted in the Verification Log)

Per-package wall seconds, top of each `test-windows` run, five consecutive `dev` commits
(controller-measured via `gh run view`):

| commit | cmd/ailang | internal/coordinator | internal/format | internal/eval_harness | SUM all pkgs |
|---|---|---|---|---|---|
| `81abc956d` | 144.0 | 88.7 | 116.5 | 45.5 | 650.0 (128 pkgs) |
| `8e3927950` | 126.0 | 100.8 | 84.3 | 43.4 | 599.7 (131 pkgs) |
| `98730db02` | 143.4 | 99.4 | 108.4 | 43.0 | 640.2 (131 pkgs) |
| `e5a325a20` | 172.5 | 124.4 | 114.5 | 80.9 | 739.0 (131 pkgs) |
| `72f9cfeca` | **228.7** | **TIMEOUT >300 (FAIL)** | 32.1 | 148.7 | **1080.3** (127 pkgs) |

Four findings the controller draws from that table, which this design engages with:

1. **The slowest package on Windows is `cmd/ailang`, NOT `internal/coordinator`.** cmd/ailang's
   steady state is 126–172 s and it reached **228.7 s = 76% of the 300 s ceiling** on the failing
   run. The queue row's framing ("the coordinator package's baseline has grown 40%") is true but
   is not the mechanism, and it points at the wrong package for "what blows next".
2. **The failing run was a whole-runner slowdown, not a coordinator regression.** Aggregate
   package-seconds went 599.7 → 1080.3, i.e. **1.80x**, on a commit whose entire diff is four
   markdown files.
3. **Negative control: the slowdown was NOT uniform.** `internal/format` went the other way on the
   same run (114.5 → 32.1), which is what you expect from concurrency redistribution rather than
   from a machine that is simply slower. So package wall time under `go test ./...` is partly a
   scheduling artifact, and the `-timeout` budget is being spent on contention.
4. **The ceiling sits INSIDE the measured noise band.** Worst steady-state package 172.5 s
   x observed 1.80x runner variance = **310 s > the 300 s ceiling**. So the current budget is
   already below the worst case the runner has actually produced, with zero margin, and no
   instrument anywhere reports how close any package is to it.

### The local control (darwin, this rig, at `1b8e7eab1`)

`ok internal/coordinator 13.158s` (controller measured 13.072s), 732 top-level tests, sum of
per-test durations ~12.35 s → the package is effectively **SERIAL**:
`grep -rho 't\.Parallel()' internal/coordinator/*_test.go | wc -l` = **0**, against a
known-positive control of **17** repo-wide under `internal/`. Also in that package: 68
`t.TempDir()`, 5 `os.MkdirTemp`, 9 store-open call sites (7 `OpenStore` + 2 `NewStore`; the
controller's "17 real SQLite store opens" counts a broader pattern — see V9), 13 `exec.Command`.
Four timer-bound tests account for 8.14 s of the local 13.16 s (V10).

### The Linux leg does NOT use a shorter timeout (premise check)

The task brief said "the Linux leg uses a different, much shorter timeout — check it and say what
you find." **Finding: it does not.** Both the Linux `test` job (ci.yml:101) and the Windows
`test-windows` job (ci.yml:471) run `go test -timeout 300s ./...`. The comment at ci.yml:456
("5min timeout (vs 60s on Linux)") is **stale** — the Linux leg has used 300s, not 60s, since the
M-DX11 change (the 60s reference at ci.yml:87 is historical, about a past flake). The only 60s
`-timeout` in the file is `ailang check --timeout 60s` for `.ail` files (ci.yml:292), unrelated to
`go test`. This design therefore applies the derived budget and the headroom instrument to **both**
legs, for consistency and because the Linux runner can have a slow day too.

---

## Goals

- **(a) A DERIVED budget.** Replace the unexplained `300s` with a value derived from the measured
  stimulus — worst observed steady-state package time x measured runner variance x an explicit
  safety factor — with the derivation written down IN the workflow as a comment, naming the
  commits and numbers it came from, so the next person can re-derive rather than guess.
- **(b) A HEADROOM INSTRUMENT (the durable half).** A check that parses the per-package timings
  `go test` already prints (`ok <pkg> <N>s` / `FAIL <pkg> <N>s`) and reports the top-N slowest
  packages plus each one's percentage of the budget, warning when a package crosses a stated
  fraction of the budget. Today nothing reports how close any package is to the ceiling: the job
  is green at 76% and red at 100%, with no signal in between. The instrument ships **WARN-ONLY**
  (a slow package is a data point, not a failure); it exits non-zero only when it is itself
  broken (the anti-vacuity guard).
- **(c) Scope discipline.** Explicitly NOT reducing `internal/coordinator`'s own runtime in this
  doc; the measurements show it is not the package that blows next. Queue it as a follow-up row.

## Non-goals

- **Do NOT "fix" this by blindly raising `-timeout`.** Making a flaky ceiling pass more often is
  the same defect with a longer mean time to discovery. The budget is raised **only** because the
  measurement shows the current number is below the runner's observed worst case, and it is
  paired with the headroom instrument so drift is caught before it becomes a flake.
- **Do NOT re-run the test suite.** The instrument consumes the existing `go test` output; a
  second run doubles the slowest job in the matrix.
- **Do NOT swallow `go test` output or its exit code.** A Windows failure must still print its
  panic and still fail the job.
- **Do NOT add `t.Parallel()` to `internal/coordinator`** (or share its SQLite fixtures) in this
  doc. That is real work, out of scope, and queued as a follow-up row.
- **Do NOT change `-p` parallelism** or the job-level `timeout-minutes` (45 Linux / 25 Windows);
  the derived budget (~7 min) is well within both.

---

## Design

### (a) A DERIVED budget

**Formula** (written verbatim into the workflow as a comment):

```
budget = worst_observed_steady_state_package_time
       x measured_runner_variance
       x safety_factor
```

**The arithmetic, with the measured numbers:**

| Term | Value | Source |
|---|---|---|
| worst observed steady-state package time | **172.5 s** | `cmd/ailang` at `e5a325a20` — the slowest package on any non-failing run (V1) |
| measured runner variance | **1.80x** | aggregate package-seconds 599.7 → 1080.3 s, failing run `72f9cfeca` vs best run `8e3927950` (V2) |
| safety factor | **1.34x** | coordinator per-package variance (≥2.41x) ÷ aggregate variance (1.80x) — the measured excess per-package variance the aggregate does not capture (V3) |

```
budget = 172.5 x 1.80 x 1.34 = 416.1 s  →  -timeout 416s  (PROVISIONAL)
```

**Why each term is what it is, and why the safety factor is 1.34x and not a round number:**

- The **worst steady-state package** is `cmd/ailang` (172.5 s), not the coordinator (124.4 s) —
  finding 1. Using the coordinator would under-budget the package that is actually closest to the
  ceiling in steady state.
- The **runner variance** is the whole-runner slowdown measured on the failing run (1.80x). This
  is the observed worst case, not a guess.
- The **safety factor** is the ratio of the coordinator's per-package variance to the aggregate
  variance. The coordinator's worst steady state was 124.4 s (`e5a325a20`); on the failing run it
  was cut off at the 300 s ceiling, so its per-package variance was **at least** 300.5/124.4 =
  2.41x. The aggregate was 1.80x. The ratio 2.41/1.80 = **1.34x** is the measured excess
  per-package variance that a whole-runner average hides — the coordinator, the package that
  actually blew, exceeded the aggregate by this factor. Because the coordinator was cut off at the
  ceiling, 1.34x is a **floor**, not a ceiling.

**Epistemic status — this is a PROVISIONAL operational budget, not a verified worst case.** The
reviewer `gpt6-astra` is right that the derivation rests on a right-censored measurement, and the
word "conservative" cannot be carried by a lower bound derived from a censored observation. Two of
the three inputs are not clean observations:

- **The 300.520 s coordinator reading is RIGHT-CENSORED.** It is the value at which the package
  was cut off by the 300 s ceiling, not its completion time. Its true completion time and the
  slowdown mechanism are unknown. The 1.34x safety factor is therefore a **lower bound** on the
  per-package excess variance, not a measured value.
- **The 310.5 s product (172.5 x 1.80) is an EXTRAPOLATION.** It multiplies a steady-state
  package time by a whole-runner variance measured on a different run; it is not an observed
  package runtime.

The arithmetic and the table above are the best available evidence and are kept, but the output
is relabelled a **provisional operational budget**. The reviewer's conclusion is adopted verbatim:

> "The failed coordinator measurement is right-censored; its completion time and the slowdown
> mechanism remain unknown. 416s is a provisional operational choice, not a verified worst-case
> budget. The headroom instrument initially reports warnings; a blocking threshold requires
> calibration."

The product 172.5 x 1.80 = 310.5 s is already above the old 300 s ceiling — confirming the old
budget was below the runner's observed worst case with zero margin (finding 4). The safety factor
then lifts the budget to 416.1 s, which is 105.6 s (34%) above the observed worst case. The value
`416s` is the presentation of the computed 416.1 s, not a round number chosen and justified
afterwards. Because the budget is provisional, the instrument ships WARN-ONLY (below); a blocking
threshold is a follow-up that requires calibration data (see "Out of scope / follow-up rows").

**Why a generous budget does not defeat the purpose of `-timeout`:** `-timeout` exists to catch
genuine hangs (M-DX11's intent). A hang is a test that never completes; a 416 s budget still
catches it, just ~2 min later than 300 s. The headroom instrument (b) is what catches *drift* —
a package whose steady state is growing toward the budget — so the budget can be generous without
reintroducing the "no signal between green and red" defect.

**Workflow change:** change `-timeout 300s` to `-timeout 416s` at ci.yml:101 (Linux) and
ci.yml:471 (Windows), and replace the stale comments (ci.yml:86-88 and ci.yml:456) with the
derivation above, naming the commits (`e5a325a20`, `72f9cfeca`, `8e3927950`) and the numbers, and
labelling the result a **provisional operational budget** (right-censored input, see above) so the
next person does not read "conservative" into it.

### (b) A HEADROOM INSTRUMENT

**What it does.** A small Go tool, `tools/ci/headroom`, that reads the `go test` output already
produced by the `go test ./...` step, parses each `ok <pkg> <N>s` / `FAIL <pkg> <N>s` line, and
reports:

```
HEADROOM: top 5 slowest packages (budget 416s)
  1. cmd/ailang            228.7s  (55% of budget)
  2. internal/coordinator  300.5s  (72% of budget)  [go test FAIL]
  3. internal/eval_harness 148.7s  (36% of budget)
  ...
WARNING: internal/coordinator at 72% of budget (warn threshold 75%)
```

The `[go test FAIL]` marker reflects that the package's `go test` line was `FAIL ... 300.520s`
(the package actually failed the suite) — distinct from the headroom WARN tiers, which never red
the job.

It takes the budget (in seconds) as an argument and the go-test log on stdin or as a file path.

**Thresholds — WARN-ONLY, both tiers report, neither reds the job.** Two tiers, both relative to
the budget:

- **WARN at 75% of budget** (312 s at 416 s): a package is approaching the budget. Print a
  prominent `::warning::` line.
- **WARN at 90% of budget** (374 s at 416 s): a package is near the ceiling. Print a prominent
  `::warning::` line (higher severity).

**Neither tier reds the job.** The tool exits non-zero for exactly ONE condition: a broken
instrument (the anti-vacuity guard below — a non-empty log that parses to zero packages, or a
package count below the floor). That is a broken INSTRUMENT, not a slow package; the two exit
paths are distinct and must stay distinct. A slow package is a data point, not a failure.

**Why WARN-ONLY, and why this reverses the earlier FAIL-tier draft.** The earlier draft argued for
a FAIL tier at 90% on the grounds that "advisory text gets skipped". The reviewer `gpt6-astra`
correctly reframed this: the budget is provisional, not verified, so a FAIL tier would be a new
flake source introduced by the fix for a flake — a runner with 1.80x measured variance (finding 2)
would red the job on a transient that is not drift. A blocking threshold requires calibration
data (see "Out of scope / follow-up rows"); until that data exists, the instrument reports and
the job stays green. The anti-vacuity guard is the one non-zero exit because a silent instrument
is worse than a noisy one (CLAUDE.md principle 2, no silent fallbacks).

**Anti-vacuity guard (no silent fallback).** The instrument MUST, when the input log is non-empty
(>= 1 byte) and the parser extracts ZERO package-timing records, print
`::error:: headroom: parsed 0 packages from non-empty log — parser may be stale` and exit
non-zero. This is the repo's own axiom (CLAUDE.md principle 2, no silent fallbacks) and the
mission's verification rule 3a (a search that found nothing is a claim, not a fact) aimed at the
tool being shipped. The recorded-fixture unit tests (M2) cannot detect a format change in the
wild — they use fixtures that by definition match the current format — so the RUNTIME guard, not
the fixtures, is the primary defence against format drift.

**Package-count floor (this revision's addition, argued).** Beyond the zero-package case, the
repo reports ~127–131 packages per `go test ./...` run (V2, V19). A parser that finds 3 packages
out of 130 is as broken as one that finds 0 — both are format drift or a parser bug, not a slow
package. The tool therefore also asserts a configurable floor `minPackages = 50` (well below the
observed minimum of 127, so it cannot false-positive on a legitimate run, and far above a broken
parser's output). If the parsed package count is below the floor, the tool prints
`::error:: headroom: parsed <N> packages (< floor) — parser may be stale` and exits non-zero. The
floor applies to the full-suite `go test ./...` output this tool is wired to; it is a constant in
`main.go` with its own unit test and mutation row (M2). Because the floor applies to ALL input,
the M2 acceptance fixtures below are written with ≥50 packages so they satisfy the floor (see M2).

**Constraints, and how each is met:**

- **Must NOT re-run the test suite.** The tool consumes the existing `go test` output captured to
  a log file; it never invokes `go test` itself.
- **Must not swallow `go test` output or its exit code.** The workflow step redirects `go test`
  output to a log, prints it back to stdout (so the panic is still visible), captures the exit
  code directly (no pipe), runs the headroom tool on the log, then exits with the OR of the two
  verdicts. `exit codes through pipes lie` is avoided by **not using a pipe at all** — see the
  wiring below. On the Linux leg the capture must use `|| go_rc=$?` so `set -e` does not abort the
  step before the log is printed (V17); on the pwsh leg the step must prepend
  `$ErrorActionPreference = 'Continue'` and `$PSNativeCommandUseErrorActionPreference = $false`
  so a failing native `go test` does not throw and abort before `Get-Content` (V16/V18,
  UNVERIFIED-LOCALLY — see Verification Log). The headroom call itself is guarded on both legs
  (`|| hr_rc=$?` on bash; `$LASTEXITCODE` on pwsh) so a non-zero headroom verdict cannot abort
  before the `go test` exit code is returned (round-2 fix, objection 5).
- **Must be testable on this rig.** The parser is ordinary Go with unit tests over recorded
  fixture output (V11), not something only observable in CI.
- **Windows only or every leg?** Both. Both legs use the same 300 s budget today (premise check
  above), so the derived budget and the instrument apply to both. The instrument is cheap (parses
  existing output, no re-run) and gives early warning on both. The Linux runner is faster, so
  packages sit at a lower % of budget there, but a slow Linux day is the same class of flake.

**The instrument must be BUILT before it can run (round-2 fix, objection 4).** `tools/ci/headroom/`
is a SOURCE directory; it is not itself an executable. The previous revision's wiring and
acceptance commands invoked `tools/ci/headroom` (Linux) and `tools/ci/headroom.exe` (Windows)
directly — the Linux leg executed a directory as a program, and the Windows leg invoked a binary
no step ever created. As specified, the instrument could not run and the workflow would fail
regardless of test results. This revision adds an explicit **build step BEFORE the test suite in
both jobs**, placing the executable OUTSIDE its source directory and failing loudly on a non-zero
build exit:

```bash
# Linux (bash) — build step, placed before the go test step
go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom
```

```powershell
# Windows (pwsh) — build step, placed before the go test step
go build -o "$env:RUNNER_TEMP/headroom.exe" ./tools/ci/headroom
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
```

The build step is intentionally NOT guarded with `|| ...` / `$ErrorActionPreference = 'Continue'`:
a broken instrument must fail the job loudly, not silently (CLAUDE.md principle 2). On the Linux
leg, `set -e` makes a non-zero `go build` abort the step automatically; on the pwsh leg the
explicit `if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }` makes the same failure loud even though
the test step (a separate step, its own shell) prepends `Continue`.

**Why `$RUNNER_TEMP` and not `./bin/`.** The repo already builds a `tools/`-rooted helper into a
binary before using it — `.github/workflows/ci.yml:627` runs
`go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter` (V20) — and consistency with
`./bin/` is worth something. `go build -o` creates the parent directory (V21), so `./bin/` would
work without a `mkdir -p`. This revision nevertheless places the headroom binary in `$RUNNER_TEMP`
(the GitHub Actions per-job temp directory, guaranteed to exist and writable, and cleaned up
automatically at the end of the job) for three reasons:

1. **The headroom binary is a CI-only diagnostic, not a shipped artifact.** `./bin/` is a
   repo-relative path that leaves the built binary in the source tree, where it shows up in
   `git status` and risks being committed or interfering with other tooling. `$RUNNER_TEMP` keeps
   it out of the tree entirely.
2. **Per-job isolation.** Each job gets its own `$RUNNER_TEMP`, so the Linux `test` job's
   `headroom` and the Windows `test-windows` job's `headroom.exe` never collide. A shared
   `./bin/` would be written by both jobs.
3. **Fresh-build guarantee.** Building to `$RUNNER_TEMP` guarantees the binary is rebuilt from the
   current checkout on every run; a `./bin/` path risks a stale binary if `./bin/` were ever
   committed or cached.

The govulncheck-filter precedent (ci.yml:627) is a single-job helper whose binary is consumed
within the same job; headroom is a two-job diagnostic whose binary must not be committed, which is
exactly the case `$RUNNER_TEMP` is designed for.

**Wiring (both legs), preserving the exit code.** The step becomes (build step above, then):

```bash
# Linux (bash) — test step
go_rc=0
hr_rc=0
go test -timeout 416s ./... > go_test_headroom.log 2>&1 || go_rc=$?
cat go_test_headroom.log                      # print the real output (panic, FAIL, ok lines)
"$RUNNER_TEMP/headroom" go_test_headroom.log 416 || hr_rc=$?
if [ $go_rc -ne 0 ]; then exit $go_rc; fi     # a Windows/Linux failure still fails the job
exit $hr_rc
```

```powershell
# Windows (pwsh) — test step
$ErrorActionPreference = 'Continue'
$PSNativeCommandUseErrorActionPreference = $false
go test -timeout 416s ./... *> go_test_headroom.log
$go_rc = $LASTEXITCODE
Get-Content go_test_headroom.log
& "$env:RUNNER_TEMP/headroom.exe" go_test_headroom.log 416
$hr_rc = $LASTEXITCODE
if ($go_rc -ne 0) { exit $go_rc }
exit $hr_rc
```

Redirecting to a file and reading the exit code directly (rather than piping) is what keeps the
real `go test` exit code: the pipeline's exit code is the last command's, but a direct
`>`-redirect + `$?`/`$LASTEXITCODE` is unambiguous. The log file name `go_test_headroom.log` is
distinct from `gated_integration.log` (used by the separate "Assert binary-gated integration
tests ran" step), so the two never collide.

**Why the Linux leg needs `|| go_rc=$?` and the pwsh leg needs the explicit `$ErrorActionPreference`
prepend (round-2 fix, objection 5).** GitHub Actions runs bash steps with `set -e` by default (the
Linux `test` job's `run:` block declares no `shell:` key, so it gets Actions' default
`bash -e {0}`). Under `set -e`, a failing `go test` would terminate the script before
`cat go_test_headroom.log` — a red job with no test logs, directly violating the "must not swallow
`go test` output" constraint. The `|| go_rc=$?` form captures the exit code without letting
`set -e` abort (V17). The pwsh leg has the SAME exposure, via a different mechanism: GitHub Actions
pwsh steps default to `$ErrorActionPreference = 'Stop'`, and PowerShell 7.3+ sets
`$PSNativeCommandUseErrorActionPreference = $true` by default, so a failing native `go test` throws
a terminating error and aborts the step before `Get-Content` — the same "red job with no test
logs" defect. The pwsh leg therefore prepends `$ErrorActionPreference = 'Continue'` and
`$PSNativeCommandUseErrorActionPreference = $false` so native command failures do not throw, and
reads `$LASTEXITCODE` immediately after each native command (V16/V18, UNVERIFIED-LOCALLY — see
Verification Log). The headroom call is guarded on BOTH legs (`|| hr_rc=$?` on bash;
`$LASTEXITCODE` on pwsh) so a non-zero headroom verdict (e.g. the anti-vacuity guard firing) cannot
abort the step before the `go test` exit code is returned — the round-1 fix guarded `go test` but
missed the headroom call site, which is exactly the "guard the helper, miss the call site" shape
the round-2 reviewer flagged.

**Step-body sweep (round-2 fix, objection 5).** Every command in both step bodies was audited for
the same "failure must not abort before the log is printed" exposure. Findings:

- **Linux (bash):** `go test` is guarded (`|| go_rc=$?`); the headroom call is guarded
  (`|| hr_rc=$?`, the round-2 fix); `cat go_test_headroom.log` is safe — the `>` redirect creates
  the file even when `go test` fails, so `cat` cannot fail on a missing file; the two `exit`
  statements are intentional exits, not commands whose failure must be suppressed. The build step
  is a SEPARATE step and is intentionally NOT guarded (it must fail loudly).
- **Windows (pwsh):** with `$ErrorActionPreference = 'Continue'` and
  `$PSNativeCommandUseErrorActionPreference = $false` prepended, neither `go test` nor the
  headroom call throws on a non-zero exit; `Get-Content go_test_headroom.log` is safe — the `*>`
  redirect creates the file even when `go test` fails; the two `exit` statements are intentional.
  The build step is a SEPARATE step and is intentionally NOT guarded (it must fail loudly, via the
  explicit `if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`).

**Placement.** The parser + threshold + anti-vacuity logic live in `tools/ci/headroom/main.go`
with unit tests in `main_test.go` over recorded fixtures. A **cihygiene gate test**
(`internal/cihygiene/`) asserts that ci.yml still carries the derived `-timeout 416s` value AND
the derivation comment in both legs, so the budget cannot silently drift back to a round number
(the same "tests ARE the gate" pattern as `workflow_timeouts_test.go`).

### (c) Scope discipline

Reducing `internal/coordinator`'s own runtime — adding `t.Parallel()` across 732 serial tests,
sharing SQLite fixtures, cutting the 4 timer-bound tests — is **real work and explicitly OUT OF
SCOPE** for this doc. The evidence: the coordinator is not the slowest steady-state package
(cmd/ailang is, finding 1), and the failing run was a whole-runner slowdown, not a coordinator
regression (finding 2). Speeding up the coordinator would not have prevented the failing run —
cmd/ailang at 228.7 s was the closest to the ceiling in steady state, and the whole runner was
1.80x slow. See "Out of scope / follow-up rows".

---

## Milestones

Each is independently committable and testable; each acceptance test names the exact
production-code mutation it kills.

### M1 — the derived budget (workflow comment + value + cihygiene gate)

Change `-timeout 300s` → `-timeout 416s` at ci.yml:101 and ci.yml:471. Replace the stale comments
(ci.yml:86-88, ci.yml:456) with the derivation (172.5 x 1.80 x 1.34 = 416.1 s), naming the commits
`e5a325a20`, `72f9cfeca`, `8e3927950`, and labelling the result a **provisional operational
budget** (right-censored input). Fix the stale "vs 60s on Linux" claim at ci.yml:456. Add a
cihygiene gate test `TestGoTestTimeoutIsDerived` that parses ci.yml (real YAML, per the existing
`workflow_timeouts_test.go` pattern) and asserts both `go test ./...` steps use `-timeout 416s`
and that the derivation comment (containing `172.5`, `1.80`, `1.34`) is present.

- **Acceptance (commands):**
  - `grep -n 'timeout 416s' .github/workflows/ci.yml` → two hits (lines 101 and 471).
  - `grep -n '172.5\|1.80\|1.34' .github/workflows/ci.yml` → the derivation comment is present.
  - `go test ./internal/cihygiene/` → passes.
- **Mutation killed:** revert `-timeout 416s` to `-timeout 300s` (or any round number) in either
  leg, or delete the derivation comment → `TestGoTestTimeoutIsDerived` turns red. This is the
  guard that stops the budget from silently drifting back to an unexplained constant.

### M2 — the headroom instrument (Go tool + unit tests over fixtures)

Add `tools/ci/headroom/main.go` (parse `ok <pkg> <N>s` / `FAIL <pkg> <N>s`, compute % of budget,
report top-N, apply the WARN-ONLY thresholds, and the anti-vacuity guard) and `main_test.go` with
table tests over recorded fixture output (V11): a normal run, a run with a `FAIL` line, a run with
`[no tests to run]` and `(cached)` suffixes, a run where a package crosses 75% and 90%, a
garbage/non-matching fixture (`TestEmptyParseOnNonEmptyInput`), and a below-floor fixture
(`TestPackageCountFloor`).

The tool is a SOURCE directory (`tools/ci/headroom/`), not an executable; every acceptance command
below therefore BUILDS it first (to `$RUNNER_TEMP/headroom`) and then INVOKES the built binary —
never the source directory (round-2 fix, objection 4). `$RUNNER_TEMP` is the GitHub Actions
per-job temp dir; on a local rig, `export RUNNER_TEMP=/tmp` (or any writable dir) before running
these commands. The fixtures below use ≥50 packages so they satisfy the package-count floor (50).

- **Acceptance (commands):**
  - `go test ./tools/ci/headroom/` → passes.
  - **Clean-checkout build + invoke (objection 4's core defect):** from a clean checkout (no
    `$RUNNER_TEMP/headroom` present), `go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom`
    then `"$RUNNER_TEMP/headroom" <(for i in $(seq 1 60); do printf 'ok\tgithub.com/x/pkg%d\t100s\n' "$i"; done) 416`
    → prints the report and exits 0 (60 packages ≥ floor 50; no threshold crossed).
  - `"$RUNNER_TEMP/headroom" <(for i in $(seq 1 59); do printf 'ok\tgithub.com/x/pkg%d\t100s\n' "$i"; done; printf 'ok\tgithub.com/x/pkg60\t380s\n') 416`
    → prints a `::warning::` line and exits **0** (60 packages, one at 380 s = 91% of 416, crossing
    the 90% WARN tier, but WARN-ONLY — a slow package is a data point, not a failure).
  - `"$RUNNER_TEMP/headroom" <(printf 'garbage\nnot a go test line\n') 416` → prints
    `::error:: headroom: parsed 0 packages from non-empty log — parser may be stale` and exits
    **non-zero** (anti-vacuity: a broken instrument, not a slow package).
  - `"$RUNNER_TEMP/headroom" <(for i in $(seq 1 3); do printf 'ok\tgithub.com/x/pkg%d\t100s\n' "$i"; done) 416`
    → prints `::error:: headroom: parsed 3 packages (< 50) — parser may be stale` and exits
    **non-zero** (package-count floor: a parser finding 3 of 130 packages is as broken as one
    finding 0).
- **Mutation killed:** remove the `FAIL`-line branch of the parser → a fixture with a `FAIL` line
  is not reported (the package vanishes from the report). Remove the anti-vacuity guard → the
  garbage fixture exits 0 instead of non-zero (format drift silently disables the instrument).
  Remove the package-count floor → a below-floor fixture exits 0 instead of non-zero.

### M3 — wire the instrument into the workflow (both legs, exit-code preservation)

Add the **build step** (before the `go test ./...` step) and the **headroom step** to both the
Linux `test` job and the Windows `test-windows` job, using the redirect-and-capture wiring above.
The build step must fail loudly on a non-zero build exit (a broken instrument is a broken job).
The test step must print the real `go test` output (panic included), preserve its exit code, and
add the headroom verdict. The Linux leg uses the `|| go_rc=$?` form so `set -e` cannot abort before
the log is printed; the pwsh leg prepends `$ErrorActionPreference = 'Continue'` and
`$PSNativeCommandUseErrorActionPreference = $false` so a failing native `go test` does not throw
(V16–V18, UNVERIFIED-LOCALLY — see below). The headroom call is guarded on both legs.

- **Acceptance (commands):**
  - **Clean-checkout build + invoke (objection 4):** from a clean checkout, run the exact Linux
    step body (build step, then test step) against a recorded fixture log that contains a `FAIL`
    line and a panic; assert the panic is printed to stdout AND the step exits non-zero.
  - `grep -n 'go_test_headroom.log\|RUNNER_TEMP/headroom\|tools/ci/headroom' .github/workflows/ci.yml`
    → both legs wired (build step + test step).
  - `go test ./internal/cihygiene/` → still passes (the gate from M1 is unaffected).
  - Under `bash -e`, run the exact Linux step body with a simulated failing `go test`; assert the
    log is still printed (the `|| go_rc=$?` guard prevents the `set -e` abort) AND the step exits
    with the go test code.
  - **PR CI acceptance (objection 5, the pwsh verification):** the `test-windows` job on THIS
    sprint's PR must show, in its log, BOTH the `go test` output (the `ok`/`FAIL` lines and any
    panic) AND the headroom report. This is the first-party verification of the pwsh step body on
    a real `windows-latest` runner (V16/V18 are UNVERIFIED-LOCALLY — pwsh is not installed on this
    rig).
- **Mutation killed:** drop the `if [ $go_rc -ne 0 ]; then exit $go_rc; fi` line (or the
  `$go_rc` capture) → a simulated `go test` failure exits 0 and the job goes green despite a
  failed test suite. This is the mutation that would reintroduce "exit codes through pipes lie".
  **Restore the `set -e`-unsafe form** (`go test ... ; go_rc=$?` without `||`) → a failing
  `go test` produces a red job with NO test output (the `set -e` abort fires before `cat`).
  **Remove the build step** → the headroom invocation fails (no binary at `$RUNNER_TEMP/headroom`)
  and the job reds regardless of test results (objection 4's defect).

---

## Test plan

Each row names the exact mutation it kills.

| Test | File | What it asserts | Mutation it kills |
|---|---|---|---|
| `TestGoTestTimeoutIsDerived` | `internal/cihygiene/` (new gate) | both `go test ./...` steps use `-timeout 416s`; derivation comment (`172.5`, `1.80`, `1.34`) present | reverting the budget to a round number / deleting the derivation comment (M1) |
| `TestParseGoTestOutput` | `tools/ci/headroom/main_test.go` (new) | parses `ok <pkg> <N>s`, `FAIL <pkg> <N>s`, `[no tests to run]`, `(cached)` lines | removing the `FAIL`-line branch — a failed package vanishes from the report (M2) |
| `TestPercentOfBudget` | `tools/ci/headroom/main_test.go` (new) | % of budget computed correctly (e.g. 100 s / 416 s = 24%) | wrong divisor / off-by-one in the % computation (M2) |
| `TestThresholds` | `tools/ci/headroom/main_test.go` (new) | warns at 75% and 90%, both exit 0 (WARN-ONLY) | changing a threshold constant / making a tier red the job (M2) |
| `TestTopN` | `tools/ci/headroom/main_test.go` (new) | reports top-N slowest, sorted descending | sorting bug / wrong N (M2) |
| `TestEmptyParseOnNonEmptyInput` | `tools/ci/headroom/main_test.go` (new) | non-empty log with zero parseable package lines → `::error::` and non-zero exit | removing the anti-vacuity guard — format drift silently disables the instrument (M2) |
| `TestPackageCountFloor` | `tools/ci/headroom/main_test.go` (new) | a below-floor package count (e.g. 3 of 130) → `::error::` and non-zero exit | removing the package-count floor — a parser finding 3 of 130 packages is treated as healthy (M2) |
| Build step (M3) | `.github/workflows/ci.yml` (manual, clean checkout) | `go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom` produces a runnable binary; invoking it on a ≥50-package fixture prints the report and exits 0 | removing the build step — the headroom invocation fails (no binary) and the job reds regardless of test results (M3, objection 4) |
| Workflow wiring (M3) | `.github/workflows/ci.yml` (manual, recorded fixture) | a simulated `go test` FAIL prints its panic to stdout AND the step exits non-zero | dropping the `exit $go_rc` line — a failed suite goes green (M3) |
| `set -e` wiring (M3) | `.github/workflows/ci.yml` (manual, `bash -e`) | a simulated failing `go test` still prints its log (the `|| go_rc=$?` guard) AND the step exits non-zero | restoring the `set -e`-unsafe form — a failing `go test` produces a red job with NO test output (M3) |
| pwsh wiring (M3) | `.github/workflows/ci.yml` (PR `test-windows` CI run) | the PR's Windows job shows the `go test` output AND the headroom report in the log | removing the `$ErrorActionPreference = 'Continue'` / `$PSNativeCommandUseErrorActionPreference = $false` prepend — a failing `go test` throws and aborts before the log is printed (M3, objection 5) |

---

## Conflict Surface

- **`-timeout` value and comments in ci.yml.** The value appears at ci.yml:101 and ci.yml:471,
  with comments at ci.yml:86-88 and ci.yml:456. All four must change together; the cihygiene gate
  (M1) pins the value and the derivation comment so they cannot drift independently.
- **The stale "vs 60s on Linux" comment (ci.yml:456).** This design corrects it. It is a
  documentation fix, not a behaviour change — the Linux leg already uses 300 s.
- **`gated_integration.log` vs `go_test_headroom.log`.** The "Assert binary-gated integration
  tests ran" step (ci.yml:111, ci.yml:480) runs its own `go test -count=1 -v -run ...` and tees to
  `gated_integration.log`. The headroom instrument must only parse the `go test ./...` step's
  output, captured to a distinct `go_test_headroom.log`, so the two never collide.
- **Go's `go test` output format.** The parser depends on the `ok <pkg> <N>s` / `FAIL <pkg> <N>s`
  line shape. If a future Go version changes it, the parser could silently report nothing. The
  **RUNTIME anti-vacuity check** (a non-empty log that parses to zero packages, or a package count
  below the floor, prints `::error::` and exits non-zero) is the PRIMARY defence against format
  drift — the recorded-fixture unit tests (V11) use fixtures that by definition match the current
  format and so cannot detect a change in the wild. The cihygiene gate pins the budget value and
  derivation comment, not the parser's live behaviour.
- **No existing `tools/ci/` tool parses `go test` output (branch (a), round-2 fix, objection 6).**
  The broadened search (`grep -rln 'go test\|--- PASS\|^ok[[:space:]]' tools/ci/`) returns no
  matches, and `ls -la tools/ci/` shows the directory contains exactly ONE file, `motoko_smoke.sh`
  (1771 B, executable), a bash consumer-smoke guardrail that does not parse `go test` output
  (V14). The control (`head -3 tools/ci/*`) prints motoko_smoke.sh's shebang and header comment, so
  the directory is readable and the grep instrument had something to find. Per the route-to-extension
  axiom, a NEW tool is warranted rather than an extension of an existing one — there is no existing
  `tools/ci/` tool to extend.
- **The instrument must be BUILT before it runs (round-2 fix, objection 4).** `tools/ci/headroom/`
  is a source directory, not an executable. The build step (before the test suite, output to
  `$RUNNER_TEMP`) is what makes the instrument runnable; every invocation in this doc uses the
  built binary path, never the source directory. Removing the build step reds the job regardless
  of test results.
- **pwsh step-body semantics are UNVERIFIED-LOCALLY (round-2 fix, objection 5).** pwsh is not
  installed on this rig (`command -v pwsh` → no output; control `command -v bash` → `/bin/bash`,
  V22), so the pwsh `$ErrorActionPreference = 'Continue'` / `$PSNativeCommandUseErrorActionPreference
  = $false` prepend cannot be first-party verified here. It is verified by the sprint's own PR CI
  run: the `test-windows` job on this sprint's PR executes the new step body on a real
  `windows-latest` runner, and an M3 acceptance criterion requires that job to show the `go test`
  output AND the headroom report in the log. **Residual risk:** if the pwsh guard is wrong, the
  failure mode is a red Windows job with a missing log — i.e. LOUD, not silent. That is the
  acceptable failure mode for a guard whose whole purpose is to keep the log visible.
- **Job-level `timeout-minutes`.** The derived budget (416 s ≈ 7 min) is well within the Linux
  `test` job's 45 min and the Windows `test-windows` job's 25 min. No change needed.
- **`-p` parallelism.** Unchanged. The instrument observes the output; it does not alter how
  packages are scheduled.
- **`internal/cihygiene` is "no production code: the tests ARE the gate."** The headroom *parser*
  lives in `tools/ci/headroom/` (a runnable tool), not in `internal/cihygiene/`. Only the *gate
  test* (M1) is added to `internal/cihygiene/`, preserving that package's nature.

---

## Out of scope / follow-up rows

- **Reduce `internal/coordinator` runtime (add `t.Parallel()` across 732 serial tests, share
  SQLite fixtures, cut the 4 timer-bound tests).** OUT OF SCOPE for this doc. Evidence: the
  coordinator is **not** the slowest steady-state package (cmd/ailang is, finding 1), and the
  failing run was a whole-runner slowdown, not a coordinator regression (finding 2). Speeding up
  the coordinator would not have prevented the failing run. The local control shows it is
  effectively serial (0 `t.Parallel()` vs 17 repo-wide, 68 `t.TempDir()`, 9 store opens, 13
  `exec.Command`, 4 timer-bound tests = 8.14 s of 13.16 s). **Queue as a follow-up row** for the
  controller: "M-COORDINATOR-TEST-PARALLELISM: add `t.Parallel()` to the 732 serial coordinator
  tests and share SQLite fixtures; target the 4 timer-bound tests (8.14 s) first."
- **A budget-drift alerting mechanism** (a scheduled job that runs the headroom instrument and
  files an issue when a package crosses the WARN threshold). The instrument in this doc reports
  in-band; a scheduled alert is a natural follow-up once the instrument is live.
- **Re-derive the budget after a material change to the slowest package.** The derivation is
  written into the workflow so the next person can re-derive; a follow-up could add a periodic
  re-measurement.
- **Calibrate a blocking threshold (the reviewer `gpt6-astra` ask).** The instrument ships
  WARN-ONLY because the budget is provisional. A blocking threshold requires calibration data:
  repeated `test-windows` runs at a FIXED commit with a documented longer diagnostic `-timeout`
  (e.g. 600 s, so packages are never cut off), recording runner resources, package coverage, cache
  status, completion times, and any remaining timeouts. ONLY that data can justify a blocking
  threshold, and the acceptable false-positive rate it would need must be stated (e.g. a FAIL tier
  that reds the job on fewer than 1 in 50 runs). This is N CI runs and therefore its own queue
  row, not a step in this sprint.

---

## Verification Log

Every claim below was measured on 2026-09-07 on the rig (darwin, `1b8e7eab1`) or pulled
first-party from CI job logs, not inferred. The controller's five-commit table is reproduced and
the failing run's numbers are re-derived first-party.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `cmd/ailang` is the slowest steady-state package (172.5 s at `e5a325a20`) | Controller's `gh run view` table; failing-run re-derivation below | 172.5 s is the max of the four non-failing runs' cmd/ailang column (144.0/126.0/143.4/172.5) |
| V2 | Failing run aggregate = 1080.3 s over 127 packages; 1.80x vs best run | `gh run view 34145572524 --job 101816771394 --log` piped through awk summing `(ok\|FAIL) <pkg> <N>s` | `packages=127 sum=1080.3s`; 1080.3/599.7 = 1.80x |
| V3 | Coordinator per-package variance ≥ 2.41x; safety factor 1.34x | 300.5 s (ceiling, failing run) ÷ 124.4 s (worst steady state, `e5a325a20`) = 2.41x; 2.41/1.80 = 1.34x | 1.34x is a floor (coordinator was cut off at the ceiling) |
| V4 | Failing run panic + FAIL line | `gh run view 34145572524 --job 101816771394 --log` | `panic: test timed out after 5m0s`; `FAIL github.com/sunholo-data/ailang/internal/coordinator 300.520s` |
| V5 | Failing run per-package top: cmd/ailang 228.7, eval_harness 148.7, format 32.1 | Same log, awk-sorted | `228.713s ok cmd/ailang`, `148.705s ok internal/eval_harness`, `32.103s ok internal/format` |
| V6 | Local coordinator wall time ~13 s, 732 top-level tests | `time go test -count=1 ./internal/coordinator`; `go test -list '^Test' ./internal/coordinator` | `ok .../internal/coordinator 13.158s` (controller: 13.072s); 732 tests |
| V7 | Coordinator is serial: 0 `t.Parallel()` | `grep -rho 't\.Parallel()' internal/coordinator/*_test.go \| wc -l` | `0`; control `grep -rho 't\.Parallel()' internal/*/*_test.go \| wc -l` = `17` |
| V8 | Coordinator does real IO: 68 `t.TempDir()`, 5 `os.MkdirTemp`, 13 `exec.Command` | `grep -rho ... \| wc -l` each | 68 / 5 / 13 |
| V9 | Coordinator store opens | `grep -rhoE 'OpenStore\|NewStore' internal/coordinator/*_test.go \| wc -l` | **9** (7 `OpenStore` + 2 `NewStore`). The controller's "17 real SQLite store opens" counts a broader pattern; the exact count depends on the counting method. Qualitative point (real SQLite IO) holds either way |
| V10 | Four timer-bound tests = 8.14 s of 13.16 s | `go test -count=1 -v ./internal/coordinator`, parse `--- PASS: <t> (<N>s)` | `TestIntegration_TaskExecutorWithRetry` 3.01s + `TestStoreBackedApprovalCheckpoint_Rejection` 2.02s + `TestStoreBackedApprovalCheckpoint` 2.01s + `TestCoordinatorEventHandler_RateLimitReset` 1.10s = 8.14 s (controller: 8.12 s) |
| V11 | `go test` output line shape (`ok <pkg> <N>s`, `FAIL <pkg> <N>s`, `[no tests to run]`, `(cached)`) | `go test -count=1 ./internal/cihygiene`; `go test ./internal/cihygiene`; failing-run log | `ok .../internal/cihygiene 0.748s`; `ok .../internal/cihygiene 0.567s` (cached); `FAIL .../internal/coordinator 300.520s`; `ok .../internal/coordinator 0.506s [no tests to run]` |
| V12 | Both legs use `-timeout 300s`; the "vs 60s on Linux" comment is stale | `grep -n '\-timeout' .github/workflows/ci.yml` | ci.yml:101 and ci.yml:471 both `go test -timeout 300s ./...`; ci.yml:456 comment says "vs 60s on Linux" (wrong); the only 60s `-timeout` is `ailang check --timeout 60s` (ci.yml:292) |
| V13 | `-timeout` is per test binary; `-p` defaults to GOMAXPROCS | `go help testflag`; `go help build` | "If a test binary runs longer than duration d, panic"; "`-p n` ... The default is GOMAXPROCS, normally the number of CPUs available" |
| V14 | No existing `tools/ci/` tool parses `go test` output (branch (a): a new tool is warranted, not an extension) — broadened from the round-1 keyword search per `oc-glm-5-2` | `ls -la tools/ci/`; `grep -rln 'go test\|--- PASS\|^ok[[:space:]]' tools/ci/`; control `head -3 tools/ci/*` | `tools/ci/` contains exactly ONE file, `motoko_smoke.sh` (1771 B, executable); the grep returns no matches (exit 1); control prints motoko_smoke.sh's shebang + header comment, so the directory is readable and the grep had something to find |
| V15 | `internal/cihygiene` is a "tests ARE the gate" package over workflows | Read `internal/cihygiene/workflow_timeouts_test.go` | "It has no production code: the tests ARE the gate"; parses workflows with `gopkg.in/yaml.v3`; `workflowDir = "../../.github/workflows"` |
| V16 | pwsh is not installed on this rig; the pwsh leg's safety is argued from documented PowerShell semantics, not first-party execution — **UNVERIFIED-LOCALLY, verified by the sprint's own PR CI run** (the `test-windows` job on this sprint's PR executes the new step body on a real `windows-latest` runner; an M3 acceptance criterion requires that job to show the `go test` output AND the headroom report) | `which pwsh pwsh.exe` | `command not found` (empty result; known-positive control is the bash `set -e` contrast in V17 and the `command -v bash` control in V22) |
| V17 | GitHub Actions bash steps run with `set -e`; a failing command aborts the script before the next line | `bash -e -c 'false > /tmp/x.log 2>&1; rc=$?; echo "reached rc=$rc"'` | prints nothing, `outer_rc=1`; control without `-e` prints `reached rc=1`, `outer_rc=0` |
| V18 | GitHub Actions pwsh steps default to `$ErrorActionPreference = 'Stop'`, and PowerShell 7.3+ sets `$PSNativeCommandUseErrorActionPreference = $true`, so a failing native command throws and aborts the step — **UNVERIFIED-LOCALLY** (pwsh not runnable on this rig, V16), verified by the sprint's own PR CI run (see V16) | Documented PowerShell automatic-variable semantics (pwsh not runnable on this rig, V16) | PowerShell 7.3+ makes native command failures fatal under the default `$ErrorActionPreference = 'Stop'`; the step must prepend `$ErrorActionPreference = 'Continue'` and `$PSNativeCommandUseErrorActionPreference = $false` to keep the log visible |
| V19 | Repo reports ~127–131 packages per `go test ./...` run; a floor of 50 is well below the observed minimum | Controller's five-commit table (V2) and the failing-run re-derivation | 128 / 131 / 131 / 131 / 127 packages across the five runs |
| V20 | The repo already builds a `tools/`-rooted helper into a binary before using it; the headroom build step follows this established pattern | `grep -n 'go build -o\|go run ./tools' .github/workflows/*.yml` | `.github/workflows/ci.yml:627: run: go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter` |
| V21 | `go build -o` creates the parent directory, so `./bin/` would work without a `mkdir -p`; the `$RUNNER_TEMP` choice is about tree hygiene/isolation, not feasibility | `go build -o /tmp/gobuildtest/nonexistent-dir-xyz/foo .` in a scratch module; `ls bin/` | exit 0, binary written into a previously non-existent directory; `ls bin/` → `No such file or directory` (the repo has no `./bin/` today) |
| V22 | pwsh is not installed on this rig; bash is (the control) | `command -v pwsh`; `command -v bash` | `command -v pwsh` → no output, exit 1; `command -v bash` → `/bin/bash`, exit 0 |

---

## References

- `.github/workflows/ci.yml` — the `test` (Linux) and `test-windows` jobs; `-timeout 300s` at
  lines 101 and 471; the `go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter`
  precedent at line 627.
- `internal/cihygiene/workflow_timeouts_test.go` — the "tests ARE the gate" pattern this design
  extends for the budget gate.
- `go help testflag` / `go help build` — `-timeout` (per test binary) and `-p` (GOMAXPROCS)
  semantics.
- Failing CI run: `gh run view 34145572524 --job 101816771394 --log` (commit `72f9cfeca`).
