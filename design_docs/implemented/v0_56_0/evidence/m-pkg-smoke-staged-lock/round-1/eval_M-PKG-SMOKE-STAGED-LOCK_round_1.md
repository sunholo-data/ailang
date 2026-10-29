# Independent evaluation: M-PKG-SMOKE-STAGED-LOCK — FAIL

**Formal result: FAIL (CI hard gate), 75/100. Provisional source score 95/100 is not a PASS.** PR [#2013](https://github.com/sunholo-data/ailang/pull/2013), exact head `0853c9fc15f3daf5f9e304e6e451e78c21fd899c`, remains OPEN/unmerged. No further retry is authorized.

## Why acceptance is blocked

The existing aggregate CI gate ends `CI_GATE=fail` at 14:01:21 UTC, 26 checks / 0 pending, after exactly one parent-authorized retry. Both actual Ubuntu failures are the **same unchanged #1926 PTY PID-readiness race**, not a new source defect in this package-staging implementation:

| Attempt/job | Sole failed test | Raw log | Successful changed packages in the same broad run |
|---|---|---|---|
| [1 / 123479693310](https://github.com/sunholo-data/ailang/actions/runs/40883138509/job/123479693310) | `internal/runner TestWorkerTerminalPTYLifecycle/eval/sigterm` | `ci-build-ubuntu-failure.log`, 4,590,430 bytes | cmd/ailang 346.613s; internal/pkg 18.030s |
| [2 / 123488317858](https://github.com/sunholo-data/ailang/actions/runs/40883138509/job/123488317858) | Same exact test/race | `ci-build-ubuntu-attempt2-failure.log`, 4,594,234 bytes | cmd/ailang 352.579s; internal/pkg 18.427s |

Both traces end at Python `worker_pid = int(open(pidfile).read())` with `ValueError: invalid literal for int() with base 10: ''`. The own PR has **zero internal/runner diff** from actual dev. Actual run 40883138509 attempt 2 is terminal FAILURE: Ubuntu failed, Windows and one macOS sibling were fail-fast-cancelled, other macOS succeeded, release bundle skipped. Cancelled jobs are not reported as passed. Both complete raw logs and SHA256s are preserved in `ci-two-failures-manifest.json`; no failure was erased or waived.

External blocker reference, taken from already-collected reports only: operator-held [PR #1926](https://github.com/sunholo-data/ailang/pull/1926), PROD `task-5755c3e3` / `apr-5755c3e3`, type `merge_handoff`, `agent_actionable:false`, `execution_authorized:false`. Existing committed operator-pending-review.md row 14 records that merging would dispatch despite absent execution authorization. This evaluator did not review or modify that implementation/card, dispatch an agent, change source/timeouts, or initiate either retry. Root owns its operator routing; do not bypass it to merge 2013.

## Successful evidence and source qualification

Main exact-head [Linux tests](https://github.com/sunholo-data/ailang/actions/runs/40883138953/job/123479695145), [Windows tests](https://github.com/sunholo-data/ailang/actions/runs/40883138953/job/123479865041), [docs build](https://github.com/sunholo-data/ailang/actions/runs/40883138302/job/123479847654), [lint](https://github.com/sunholo-data/ailang/actions/runs/40883138953/job/123479694413), [Sonar](https://sonarcloud.io/dashboard?id=sunholo-data_ailang&pullRequest=2013), CodeQL, vulnerability and other primary/static gates succeeded. Final PR snapshot is `pr2013-final-failed.json`; complete top-level, inline and formal reviews were empty. The aggregate skill gate still fails because of the separate Build matrix.

Owner's actual make-stamped source 37c957b3a passed changed packages (cmd 172.825s/pkg 19.353s) plus **all core packages**, lint 0, boundaries, home-isolation, sizes, changelog and fmt. Exact final 0853 has identical complete cmd/internal/std/go.mod/go.sum and Go/AIL tree; later integration of actual dba1c499 and owned links/state was docs-only. These are actual qualified-source proofs, not a repeated local whole-suite run. `final-source-identity.json` and `local-evidence-manifest.json` preserve hashes/provenance.

All 21 scoped criteria have independent source/test/log anchors in `acceptance-matrix-final.json`; generator passes flags were not used as proof. The actual primary Linux/Windows gate plus local scoped/core evidence verifies the scoped acceptance rows. The separate aggregate CI failure remains the hard Tests/verdict gate; it is not hidden by acceptance points.

## Implementation review findings

M1 stages the published registry manifest and regenerates its execution lock, reuses validated previous metadata only through existing D2 matched-git carry-forward, retains canonical authority refusals and exact source/asset coherence, and leaves author inputs intact. Silent structured rewrite facts keep quality JSON clean. Malformed/unreadable prior locks fail before child execution.

M2 initially had genuine independent finding D1: an empty-output signal death lost its actual cause. Exact actual independent RED was preserved, repair dade1527b and identical-control GREEN were verified, as was a real full-stage/process regression. Timeout priority, byte-safe 8192/2048 caps, schema, NotRun and timer placement remain unchanged. Signal-only test explicitly qualifies Windows: Go 1.26.10 source uses TerminateProcess(...,1); common start/empty/timeout/bounds controls run, and actual Windows CI succeeded.

M3 warns only after successful existing lock save, observes actual registry/git Source, sorts stderr output and does not resolve, fetch, write another lock, alter precedence or change return codes. Actual CLI lock-drift/direct-path/conflict controls pass.

M4 proof was strengthened after independent review: CLI-specific smoke must first reject the actual matching author-local pair (exit 23), then succeeds only with REGISTRY_SOURCE + REGISTRY_SOURCE_ASSET + STAGED_ROOT through real quality and publish --dry-run. Exact-version unavailable/newer-cache refuses canonically. Observed mock reads/zero writes, no message/coordinator DB, absent-Z3 portability, unchanged author bytes and actual human/JSON/schema proof pass. Guide/file-set comments match current tarball behavior without a new file-set policy.

## Base 100 scoring

| Category | Points | Basis |
|---|---:|---|
| Tests | 0/20 | Aggregate CI hard-failed twice; primary gate successes do not waive it |
| Lint | 10/10 | Actual local lint 0 and CI lint SUCCESS |
| Acceptance | 30/30 | 21 independently mapped scoped criteria, primary Linux/Windows/native proofs; aggregate failure counted above |
| Code quality | 10/15 | Literal physical inclusive FuncDecl >50 lines, including tests: -2/function capped -5; no other deduction |
| Documentation | 15/15 | Changelog, accurate implementation-complete/pending-review design, guide and approved controlled AIL fixtures |
| Design fidelity | 10/10 | Approved staging/D2/diagnostic/warning contracts preserved |
| **Total** |**75/100 — FAIL** | Hard failure overrides the numeric >=70 threshold |

The provisional source score is 95/100 if the Tests gate were green; **it is not a pass or permission to merge**. Concrete function examples: RewritePathDeps 57 physical lines, newSmokeCLIProof 60, TestSmokeStage_GitCarryForward 88; `functions-over50-final.json` records the complete inventory. All files meet 800-line gate, no new TODO/HACK/FIXME, milestone passes/timestamps/notes complete, plan has no pending milestones.

Standalone examples/runnable is **N/A by the approved plan**: a faithful example requires a controlled two-package registry/layout, supplied by effect-annotated internal AIL fixtures plus the guide and real executable tests. No new example requirement was invented. Compiler-regression bonus is N/A because own changes are host package/CLI tooling, with no compiler/exec infrastructure modification; performance bonus is N/A because there is no performance goal. Score uses base 100 with no conditional bonus.

## Disposition

Keep #2013 unmerged. No further retries, source changes, policy/threshold/timeouts or approval actions by evaluator. After the separately gated external blocker is legitimately resolved, evaluate the actual resulting source/head and terminal CI before issuing PASS. Root owns formal acceptance/merge and later design/companion archival. Reports and scratch controls are the evaluator's only writes, all under /tmp.
