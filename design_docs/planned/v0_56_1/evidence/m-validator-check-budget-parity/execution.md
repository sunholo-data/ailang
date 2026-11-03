# Execution evidence

Executed on coordinator/task-76187768 (the provided isolated work branch),
2026-11-03. The handoff's older coordinator/task-9bf807ed was not checked out.

## Recorded route ruling

The design's Design Freeze records **Mark, attended 2026-11-02: release tag,
not an emergency rebuild**. It also records deployed 896 MiB cap and celestial
publish HTTP 200 at 19:03 UTC. This resolves the sprint plan's stale OPEN gate.
Those are inherited operational records, not new registry observations here.
This run neither deploys nor publishes; worlds acceptance is not verified.

## Verification

- Existing toolchain found at /usr/local/go/bin/go: Go 1.26.10 linux/amd64.
  Added its directory to command PATH; installed no toolchain or packages.
- Sprint JSON validator passed. Session-start script ran; its next-feature jq
  expression errors on dependency indexing, but prints M1. GitHub sync failed.
- Cloud branch sync: skipped:no-task-id on coordinator/task-76187768.
- M1/M2 tests failed to compile before implementation as expected (missing env,
  stats and context APIs); metadata/handler tests likewise failed on missing fields.
- Standalone bounded/cap/procRSS/env/context tests passed, including unchanged
  observed=600 MiB, memory/time watchdogs and ordinary child-failure behavior.
- make fmt completed. Architecture boundaries, 800-line limits and changelog
  checks passed. make lint could not run: golangci-lint is absent; installation
  is prohibited by unattended execution instructions.
- Focused package and test-core results are recorded below after completion.
  Historical base c60fa199 is absent from this shallow checkout, so the supplied
  test-changed invocation fell back to HEAD. Both touched Go packages were included
  through current uncommitted changes; committed bounded changes are covered by
  the validator package tests. CI owns the full repository suite.

## Performance and environment limits

Docker and gcloud are unavailable. No image execution, staging refusal, deployed
image identity, production cap-source measurement or live metadata inspection was
possible. Release deployment now passes its actual tagged image in VALIDATOR_IMAGE;
unconfigured deployments explicitly report missing identity instead of inventing it.
The Docker command was inspected and covered by rendering assertions, not run.

Existing flat_self_import fixture, installed AILANG v0.56.1 (commit 05c860a),
linux/amd64, GOMAXPROCS=2, AILANG_NO_CACHE=1:

| GOMEMLIMIT | child exit | observed wall |
|---|---|---|
| unset | 0 | 194 ms |
| 832MiB | 0 | 124 ms |

These single smoke runs are not a GC-heavy benchmark, use no measured RSS and
cannot demonstrate pacing benefit; order/cache effects remain. No celestial or
other GC-heavy package artifact is available locally. Paced/unpaced heavy-package
peak RSS and wall evidence remains pending operator access. The wall cap stays
200 seconds. Docker cannot establish gVisor equivalence.

## Scope

Bare-file checks without a manifest and the verification child are unchanged.
No .ail source was written or edited, so no syntax prompt gate applies.
Successful stats cover check-only child duration after dependency preparation;
missing RSS remains omitted. The handler test captures the actual metadata bytes
passed to the upload path without contacting GCS, and distinguishes 422 from 400.

## Final gate results

- Both touched packages passed: cmd/registry-validator (196.356 s), internal/pkg
  (166.260 s), including build-from-worktree CLI integration fixtures.
- Additional standalone bounded tests after the unavailable-RSS test addition
  passed (1.193 s); this covers zero RSS reporting without fabricating samples.
- Final make fmt-check, check-boundaries, check-file-sizes and check-changelog
  passed. Lint remains unavailable, not claimed as passed.
- The new handler test persists 123456789 bytes / 987 ms through the upload seam
  and checks budget=422 versus ordinary compilation=400.

- go vet ./cmd/registry-validator ./internal/pkg passed.
- make test-changed's core suite failed in unchanged internal/effects:
  TestRuntimeNestedStopUsesRemainingDeadline and
  TestProcessExec_TimeoutKillsGrandchildren report surviving descendants. An
  isolated rerun of exactly those tests reproduced both failures (4.962 s).
  No internal/effects code was changed; these failures are outside sprint scope.
  The broader gate is not claimed green. Other core results are noted below.
- Full-package tests started before edits finished are not treated as pristine
  baseline evidence. Redundant overlapping test invocations were stopped; the
  completed touched-package runs above are from the retained make gate.

## Independent evaluation

The skill-directed message handoff was refused because no sprint-evaluator agent
is registered on this environment's messaging plane. A local independent sub-agent
reviewed the implementation with the sprint-evaluator skill instead. It found no
concrete implementation defect and awarded acceptance 30/30, but returned FAIL,
65/100: tests 0/20 (required gate red), lint 0/10 (unavailable), acceptance 30/30,
code quality 15/15, docs 10/15 (stale design status), fidelity 10/10. The stale
design status has now been corrected. No revised score is claimed.

The required make test-changed run finished with exit 2: the two internal/effects
failures above remained; pipeline, format, stdlib and other reported core packages
passed. Clear these verification blockers in an equipped environment before merge.
No unrelated process-runtime implementation was changed to bypass the gate.
