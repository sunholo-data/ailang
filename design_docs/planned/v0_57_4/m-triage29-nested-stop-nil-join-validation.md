# Nested stop readiness validation

The attended retry of `task-c3696d7b` applies variant A1 from the
[existing diagnosis](m-triage29-nested-stop-nil-join.md) and the
[sprint plan approved in PR #2135](https://github.com/sunholo-data/ailang/pull/2135).
Base `dev` revision: `ec3ad8b99ab352c17c4b04abe970d26ecd6935b3`.

## Fixture review

The outer marker is written after the outer helper binds cleanup. The nested
marker is written after the nested helper binds its own cleanup. The existing
five-second poll with one-millisecond sleeps now waits for both files. The
120 ms stop context, 370 ms upper bound, non-nil Join requirement, both
remaining-budget assertions, and process absence assertion are preserved.
The diff is confined to the readiness poll and its diagnostic in the Go test.

## Historical evidence

The diagnosis records the fast nil-Join failures in macOS Build jobs
[127178632414](https://github.com/sunholo-data/ailang/actions/runs/42007729357/job/127178632414)
and [123447755475](https://github.com/sunholo-data/ailang/issues/2069#issuecomment-6599231990),
along with the earlier fleet recurrence. The unchanged test on the green
direct parent took 0.15 s and 0.19 s.

The prior synthetic pre-Bind stop returned nil in 12/12 runs, and its
strengthened-gate control returned non-nil with both remaining-budget files
in 5/5 runs. The original gate had zero natural nil-Join reproductions across
the prior Linux batches of 40, 40, and 240 runs. Those batches characterize
the historical Linux behavior. The synthetic experiment establishes the
mechanism under a forced interleaving.

## Local evidence

Go 1.26.11, Linux amd64. Raw command logs are retained locally under
`/tmp/triage29-validation/`.

The ordinary container smoke run passed in **0.38 s** (package 0.481 s).
There was no failing assertion or surviving fixture PID to inspect. Passing
the unchanged assertions confirms a non-nil Join, both `.remaining` and
`.nested.remaining` files with integer budgets in 0–120 ms, and final
process absence. The historical container zombie failure is recorded in
the diagnosis separately from the early macOS nil-Join.

The focused repetitions ran serially in an isolated Linux PID namespace
with a Python PID 1 that reaped adopted children using `os.wait()`. The
namespace used `unshare --user --map-current-user --pid --fork --mount-proc`
to keep the current UID and mount the namespace's process table.

`go test ./internal/effects -run '^TestRuntimeNestedStopUsesRemainingDeadline$' -count=50 -v`
passed **50/50** with zero nil-Join outcomes. Individual runs took
**0.20–0.40 s** (package 15.670 s), consistent with exercising the stalled
cleanup and absence checks. The historical 0.14 s lower end is an
observation. The run used the existing assertions and deadlines.

`go test ./internal/effects -count=1` passed in **55.583 s** in the same
reaping namespace. Independent confirmation on a separate reaping host
is pending. The isolated namespace supplied the Linux results above.

`make check-boundaries`, `make check-file-sizes`, and `make check-changelog`
passed. The changed test is gofmt-clean, and `git diff --check` passed.
`golangci-lint run --timeout=5m ./internal/effects` reported zero issues.
Repository-wide `make lint` timed out while loading packages (exit 4).

The first `make test-changed` attempt passed the changed package and then
lost the pipeline test process to a kill signal during the core suite.
A serial retry exposed seven pipeline test failures caused by the inherited
`AILANG_STDLIB_PATH` override picking the repository stdlib in place of
the tests' temporary fixtures. A focused rerun of those tests with the
override cleared passed. The full clean-environment run reached the
pipeline suite and was killed again. The container's 8 GiB memory cgroup
reported two OOM kills. The final **passing** `make test-changed` run
used `GOMAXPROCS=2`, `GOFLAGS=-p=1`, `GOMEMLIMIT=512MiB`, and
`GOGC=50` with the stdlib override cleared and all tests enabled.
The changed effects package took 37.612 s and the core pipeline package
took 426.905 s.

## Passive macOS soak

Pending, zero of ten qualifying legs at review handoff. The patch is on the
local review branch, and the normal `Build and Release` workflow will run
on a PR targeting `dev` and on subsequent pushes to `dev` containing the
fix. Record the run ID, job ID, commit, fixture outcome, and fixture duration
for each consecutive `Build macos-latest` leg. Reference issue #2069 for
recurrence bookkeeping.
