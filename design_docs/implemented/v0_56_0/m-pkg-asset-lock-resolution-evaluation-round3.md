# PR #1849 — final independent evaluation, round 3

Integration confirmed: PR #1849 merged normally at `9dc6890919aa001deebee4361719456ee087440f` after acceptance. The archival deduction is retained as originally scored.
**PASS: 95/100. No hard failures.** Reviewed exact head `a24f2ca1c25e6dc03938515a1bc660815c26e247`.

The implementation meets the approved M-PKG-ASSET-LOCK-RESOLUTION contract. Locked assets select the dependency version from the consuming context, retain versioned missing-target tombstones, and never fall through to a newer cached installation. Source-time targets survive runner/batch execution, budget scopes, context cloning and Engine context replacement. Root-package assets, AILANG_PACKAGE_ROOT and deployed no-lock semver fallback behavior remain intact after #1964 integration.

## Score

| Category | Score | Evidence |
|---|---:|---|
| Tests | 20/20 | Exact-head Linux/Windows required tests and builds pass; root's authorized native full evaluator completed successfully on clean exact a24. No exclusions. |
| Lint | 10/10 | Native evaluator lint/size gates green; new TODO/HACK/FIXME count 0. |
| Acceptance | 30/30 | All four milestone contracts demonstrated by actual asset bytes, tombstones, malformed locks, snapshots, budget/clone and API/runner/embed behavior. |
| Code quality | 15/15 | Small layered snapshot/accessor helpers; no package-global resolution state, new TODOs or listed rubric quality deductions. The duplicate fail-closed manifest read is only a nonblocking note. |
| Documentation | 10/15 | Changelog/example/reference material is complete, including the two repaired wording findings. Design still says Planned with unchecked boxes and companion plan pending evaluation; deduct 5 under the existing rubric until root post-PASS archival. |
| Design fidelity | 10/10 | Immutable context-specific targets and missing-lock tombstones follow the approved design; existing fallback/authority behavior preserved. |

## Repaired findings and independent evidence

Historical round1 **FAIL51** and round2 **FAIL68** remain preserved unchanged. Round2 found a real public API bypass: malformed lock discovery failed through Call but CallPreserveFloats returned newer fallback bytes with nil error. The identical early discovery-error guard now protects Load, CallPrepared (and Call), and CallPreserveFloats; CallJSON delegates through Call. The documented REPL-style Eval path has no module lock context.

The isolated independent repro now returns nil plus named lock-parse errors from both public call paths. It was rerun after #1964 at af6dca29; later a24 changes are test/metadata only and leave those production guards unchanged. Logs retain the original red result and repaired results. Public PreserveFloats tests also read actual older locked bytes on cold/warm calls and after SetEffContext replacement.

Coverage repair preserves those meaningful assertions. The existing TestGoFiles-only submitted-profile filter dropped the prior external-only test package. Moving the same regression in-package makes the helper profile survive unchanged: reviewer cmp verified raw/derived profiles byte-identical. Actual exact-head Sonar is **PASS, 94.9% new coverage against unchanged 80%**, with 0 new duplication and no quality gate policy changes.

The portable corrupt-cache regression first demonstrates a valid HOME asset path, then requires named Err under an invalid configured cache root. It proves root exclusivity while permitting OS-specific error wording. The prior Windows wording failure and unrelated #1926 PID readiness race are preserved in raw logs; neither was hidden with a skip or production fallback.

## Terminal gates

Root ran the explicitly authorized native fallback on a clean worktree at the exact full SHA above, rebuilt binary stamped to that SHA, and the repaired evaluator script from the approved small-fixes path:

`EVAL_LOCAL_GATES=1 GOFLAGS=-p=1 bash .../small-fixes/.agents/skills/sprint-evaluator/scripts/evaluate_sprint.sh M-PKG-ASSET-LOCK-RESOLUTION`

`1849-native-evaluation.log` ends at 2026-10-29T09:45:16Z with valid JSON: tests_pass=true, lint_clean=true, file_sizes_ok=true, todo_hack_count=0, no hard fails. Session 52021 exited 0. Required remote Linux/Windows checks also passed; exact-head Sonar gate JSON records 94.9%. Two optional hosted Mac build jobs remained queued; the operator-authorized actual native full gate supplies the required local evidence. This report does not falsely label queued hosted jobs green.

Other measured gates include changed/core tests, full seven-package integration suites, boundaries/architecture, sizes, changelog, examples/manifest and all 49 stable stdlib interfaces. This reviewer did not repeat a broad suite or edit implementation. Parent owns normal merge and truthful unreleased document/plan archival after this PASS.

Source acceptance checkpoint: `pr1849-round3-source-acceptance.md`. Raw independent repro: `pr1849-review-repro-af6dca29.log`; remote Sonar: `pr1849-a24-sonar-gate.json`; native: `1849-native-evaluation.log`. All under `/tmp/ailang-backlog-20261029/`.

EVALUATION_VERDICT: PASS score=95
