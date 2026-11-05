# M-SCC-SHADOW-AWARE-CALLGRAPH execution evidence

Execution: 2026-11-04, Linux amd64, Go 1.26.10. Workspace base:
`17664aba396e5a7135d1c4c962320e1e002e9724`; branch `coordinator/task-25c55223`.
The coordinator supplied a different historical planner branch; the clean provided
workspace branch was retained. No changes in `internal/types` or `internal/pipeline`.

## Binary identities

- Published baseline: `/usr/local/bin/ailang`, copied to `/tmp/scc-v0570`,
  v0.57.0, reported commit `b23cadced75c7bbf1328eebcc5ab011453fba672-dirty`.
  SHA256 `c17e147855b0fac8dce9a60760340a8ae5b2b6a54c078d68c25154fdeaa350c3`.
- Candidate: `go build -o /tmp/scc-candidate ./cmd/ailang`, vcs revision
  `b4b8a5899f36a084f6cfddc93c19a2209bdca3c6`, `vcs.modified=false`.
  SHA256 `9879b2b356e3adcbf74e9413b15839c8bdb73c3b4f82fcb20fa4c09c8aed4a2b`.
- Candidate production file `internal/elaborate/scc.go` SHA256
  `08fe748f5416595d95102a78797c089660f276ebedb57778ced36d9edcb19f3d`.
  Later commits add tests/documentation only.
- An early asynchronous source build `/tmp/scc-base` incorporated the walker
  change while dependencies compiled; it is NOT the baseline used for evidence.

## Regression controls and reproducibility

Loaded the repository-built binary's full `ailang prompt` before source fixtures;
reported teaching prompt version v0.16.11 (repository language v0.57.0).

`SCC_FIXTURE_DIR=/tmp/scc-fixtures go test ./internal/elaborate -run TestSCCShadowEffects -count=1`
exports all twelve source fixtures. From `/tmp/scc-fixtures`, run both binary paths
with `check scc/r<case>.ail`. The committed `scc-shadow-controls.tsv` records source
hashes, exit statuses and missing-IO diagnostics for each case.

Rows 1/5/6 and the additional let-discard row 12 change reject→accept. Rows 2–4,
7–9 remain accepted; 10/11 remain rejected with `Missing effects` and `IO`.
Row 5 uses `each(\t. f(t), r)`: the abbreviated design-table identity callback
`\t. t` returns int rather than the required unit and is independently ill-typed.
No standalone showcase is added; effects are explicit, contracts and inline tests
are skipped because Go tests assert compiler acceptance and rejection instead.

Pattern tests compare surface binders with core binders after elaboration;
legacy `ConsPattern` is normalized to the parser's `ConstructorPattern("::")`
for that comparison, because elaboratePattern does not accept the legacy node.
Scope tests were observed failing in fourteen cases before the walker change.
Graph tests preserve genuine cycles; emission tests require `core.LetRec` for a
bodyless module letrec. Existing scope/module resolution matrices pass unchanged.
New regression tests also pass twenty consecutive runs.

## Repository corpus

Input is exactly `git ls-files -z '*.ail'`: 810 tracked files. Each is checked
from the nearest enclosing `ailang.toml` directory, or the repository root if
there is no manifest. Each binary invocation is
`nice perl -e 'alarm 120; exec @ARGV' <binary> check <relative-path>`.
Four worker threads maximum, each checking base then candidate sequentially.
The committed `scc-shadow-corpus.tsv` contains every path, package root, source
SHA256 and both exit statuses. The controls above establish non-vacuity.

Final result: 810 files, zero exit-status flips, zero timeouts. 592 remain accepted and 218 retain pre-existing rejection. No newly accepted or newly rejected repository paths.

## Operational limitations and release follow-up

The session-start helper printed a jq dependency-expression error and lacked bc;
its summary was not treated as test evidence. Direct make/go checks were used.
Cloud sync reported `skipped:no-task-id`; no remote publication is claimed.

The broader test gate encounters `TestRuntimeNestedStopUsesRemainingDeadline`
and `TestProcessExec_TimeoutKillsGrandchildren` failures in `internal/effects`.
Both reproduce independently in a focused run. `go list -deps ./internal/effects`
contains neither elaborate nor pipeline; this package's code and dependencies
are unchanged by the sprint. These failures do not arise from SCC traversal.

Before release, the operator still owns the approved out-of-repository sweep
(other checkouts, deduplicated eval programs, executor MCP serve-api smoke).
The coordinator should route this implementation to the independent sprint
evaluator; no self-evaluation or merge approval is claimed here.

## Verification results

- `go test ./internal/elaborate -count=1`: PASS.
- New regression tests with `-count=20`: PASS.
- Existing module-let and lexical-scope matrices: PASS unchanged.
- `make fmt-check`, `make check-file-sizes`, `make check-changelog`: PASS.
- `go vet ./internal/elaborate`: PASS.
- `make verify-examples`: PASS, 265 passed / 0 failed / 10 skipped;
  247 manifest modules / 0 drift / 0 missing-on-disk.
- `make test-changed`: FAIL (exit 2), exclusively the two process-cleanup tests
  listed above. All other core packages pass, including pipeline (322.951 s).
- `make lint`: PASS, zero issues (golangci-lint v2.11.4, built with Go 1.26.10).
  Installed via existing `make install-lint` with `GOTOOLCHAIN=local`; no compiler
  or system toolchain was installed.

The exact focused/core failures and other command summaries are preserved in
`scc-shadow-checks.log`. M5's all-gates-pass criterion remains unsatisfied due to
the unchanged effects tests; this is an explicit verification exception, not a
claimed green gate.
