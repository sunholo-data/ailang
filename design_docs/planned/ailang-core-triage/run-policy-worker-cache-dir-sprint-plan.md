# Sprint Plan: Restricted worker compile-cache isolation

Refs #1547

- **Sprint:** M-RUN-POLICY-WORKER-CACHE
- **Date:** 2026-10-08
- **Status:** Planned; design approved by coordinator handoff, execution awaits sprint-plan approval.
- **Target:** v0.52.6 (checkout v0.52.5)
- **Duration:** 2 days, approximately 10 hours including 25% contingency.
- **Risk:** Medium: subprocess cleanup and filesystem trust boundary.
- **Design:** [run-policy-worker-cache-dir.md](run-policy-worker-cache-dir.md)
- **Neighbour:** [run-policy-result-line-forgeable.md](run-policy-result-line-forgeable.md) (#1548 / PR #1679).

## Goal and current status

Restricted `run --policy` must forward a non-empty operator `AILANG_CACHE_DIR` and otherwise compile into a private per-run cache outside `fs_sandbox`. Explicit operator paths inside the sandbox remain permitted with a warning naming the poisoning risk and `fs_deny_write` mitigation. Thus the outside-sandbox invariant is for the default; operator overrides are deliberate exceptions.

Read issue #1547 and its comments through the GitHub API on 2026-10-08: open, zero comments. The issue reports the ignored override; triage verified it live on origin/dev `658ff76a3`. Current code still omits the variable from `workerEnvAllow`. The approved design audits the pipeline cache, policy-tool children, and motoko cache isolation; adopt the existing policy-tool private-temp pattern without changing compiler cache format or unrelated environment variables.

The seven-day velocity script found only one planning commit in this shallow checkout and no usable implementation LOC/day measurement. The current changelog records policy hardening and recent CLI fixes but no comparable timed metric. The 2-day estimate is therefore a conservative task estimate, not measured velocity: 280 LOC total (140/day planning capacity), with most effort in lifecycle regressions.

## Decisions and scope

- Empty and unset `AILANG_CACHE_DIR` both select the private default; test value rather than `RawSet` presence. Non-empty operator values win and their directories are never removed by the supervisor.
- Create default cache after stdout/stderr/control pipe setup, immediately before `cmd.Start`; explicitly remove it before any refusal that calls `os.Exit`, including start failure and unsafe placement. Defer removal on normal, worker-error, timeout, and output-cap returns, after the worker lifecycle finishes.
- If supervisor `TMPDIR` places the generated directory inside the sandbox (including a symlink alias), remove it and refuse with a clear explanation. Do not fall back silently or warn-and-proceed for generated defaults.
- Use symlink-aware containment. `entryInsideSandbox` resolves existing paths, but an operator path may not yet exist: resolve its nearest existing ancestor and append missing components before testing containment, or reuse an existing equivalent helper. Test a missing cache leaf under a symlink into the sandbox. Keep this limited to cache placement; do not alter entry admission semantics.
- Only compilation in the worker writes the generated outside cache; the confined program's FS handler cannot write there. `trusted_host` retains full environment passthrough and existing defaults.
- Warning text begins with `warning:` and never `policy`. Reuse established refusal handling for errors and check compatibility with #1548's authoritative result channel. No changes to result-channel parsing are part of this sprint.
- Exclude persistent shared caching, cache-format verification, removal of old in-sandbox caches, and #1548 implementation.

## Registry reuse audit

`ailang pkg search cache` and `ailang pkg search sandbox` both returned no packages (local CLI warned it may be stale). M1 and M2 each choose **none**: this is Go supervisor process/environment handling, not an importable AILANG capability. Reuse `internal/policytool/cli_ops.go`'s lifecycle pattern and existing policy test helpers. No candidates require `pkg info` or `pkg docs`; no new package dependency.

## Milestones

### M1: Cache environment and supervisor lifecycle (~140 LOC)

**Estimate:** 55 implementation + 85 unit-test LOC; Day 1, 5 hours including contingency.
**Dependencies:** None.
**Files:** `cmd/ailang/run_policy_supervise.go`, `cmd/ailang/run_policy_hardening_test.go`; a small focused cache-placement test file under `cmd/ailang/` if needed.

Add the allowlist entry and thread the selected default into `workerEnv`, updating all callers. Introduce the smallest testable cache lifecycle helper needed to exercise creation/placement/start-failure cleanup without adding public CLI knobs. Preserve credential filtering and use the existing supervisor deadline.

- [ ] Restricted workers receive exactly one effective cache value; non-empty operator value wins, empty/unset uses the injected default.
- [ ] Trusted-host environment passthrough and provider-scoped credential tests remain unchanged in behavior.
- [ ] Generated cache directories are private, outside the sandbox, and removed on normal exit, start failure, timeout, output-cap termination, and worker failure.
- [ ] Invalid temporary base or in-sandbox TMPDIR produces explicit refusal; generated directories are removed before refusal.
- [ ] Relative, existing symlink, and missing-leaf symlink operator paths are classified correctly; in-sandbox operator values warn once and proceed.

**Risk:** `refusePolicy` uses `os.Exit`, bypassing deferred cleanup. Arrange allocations after pipe setup and explicitly clean before refusal. Tests must prove cleanup, not only assert helper return values.

### M2: End-to-end regressions and operator documentation (~140 LOC)

**Estimate:** 115 integration-test + 25 documentation LOC; Day 2, 5 hours including contingency.
**Dependencies:** M1.
**Files:** `cmd/ailang/run_policy_supervise_test.go`, `docs/docs/reference/env-vars.md`, `docs/docs/guides/agent-tool-policy.md`, `changelogs/v0.32-current.md`.

Use `buildAilang`, `runAilangBin`, `writePolicy`, and existing test fixtures. Capture default temp directories using a dedicated outside-sandbox TMPDIR and verify no `ailang-policy-cache-*` survives. Preserve parent env deliberately with test-local overrides so ambient operator configuration cannot hide regressions.

- [ ] Unset and explicitly empty overrides complete restricted runs without creating sandbox `.ailang/`, and leave no generated cache directory.
- [ ] External override produces `<dir>/compile`, survives the run, and leaves no sandbox `.ailang/`.
- [ ] In-sandbox override emits exactly one `warning:` line naming poison risk and mitigation while the admitted run succeeds; trusted-host behavior remains compatible.
- [ ] In-sandbox and symlinked TMPDIR cases refuse, leave no generated directory, and do not execute the program.
- [ ] Existing timeout, output-limit, admission, output-preservation, and credential suites pass; cleanup exercised on worker-error paths.
- [ ] Documentation describes cold default compilation, cleanup, override ownership, warning exception, and unsafe-TMPDIR refusal; changelog references #1547.

**Example coverage:** No new language feature or permanent `.ail` example is required. Reuse the existing policy-test example fixtures and the issue's runnable CLI repro. If adding/editing `.ail` fixtures, first obtain `ailang prompt`, then check them with `ailang check` under the appropriate policy context.
**Risk:** #1548 modifies the same supervisor. Reconcile its current code when executing and run its output tests; cache diagnostics must not impersonate authoritative result lines.

## Validation and success metrics

Run focused `go test ./cmd/ailang -run 'TestWorkerEnv|TestRunPolicy' -count=1`, then the full `go test ./cmd/ailang` suite. Format changed Go files and run `make lint` and `make test` before completion. Run `make check-boundaries` if execution introduces any cross-package dependency. Planning itself does not claim implementation tests have passed.

Coverage target is behavioral: every new environment/placement branch and every post-allocation exit path listed above has a regression, with no regression in existing policy suites. Record cold-run latency for the small repro and compare with an external persistent override; do not gate correctness on an arbitrary performance threshold. No new permanent files in the sandbox and no leaked generated temp caches are permitted.

## Execution handoff and PR body

Use the plan and `.ailang/state/sprints/sprint_M-RUN-POLICY-WORKER-CACHE.json` after sprint-plan approval. Both milestones start unexecuted. The coordinator's merge/approval workflow routes the artifacts to sprint-executor; this planning task does not self-approve or begin implementation. Do not open a new issue or close #1548.

Suggested plan PR body:

> Refs #1547
>
> Plans restricted worker cache forwarding and private per-run default isolation, with explicit unsafe-TMPDIR refusal, cleanup regressions, and operator documentation. Includes two milestones over two days and machine-readable progress state. Cross-links the neighbouring #1548 result-channel design without implementing it.
>
> Validation: sprint JSON schema, milestone dependencies, estimates, and populated registry decisions checked. Implementation tests are scheduled in the plan.
