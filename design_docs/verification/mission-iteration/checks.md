# Mission iteration verification — 2026-09-07

Implementation source: `77dc7287e7bce225e64062452598f0694ac361fc` on `sprint/mission-iteration`, isolated checkout `/private/tmp/ailang-mission-iteration`.

**Implementation checks pass. Whole sprint remains partial:** M1–M5 complete; M6 needs an unowned, execution-approved Docs canary item and its bound approval artifacts. Live adoption has not run. See [activation packet](canary-activation.md) and [independent evaluation](evaluation.md).

## Reproducible checks

Use `GOCACHE=/private/tmp/ailang-iteration-go-cache`; lint additionally used `GOLANGCI_LINT_CACHE=/private/tmp/ailang-iteration-lint-cache`.

| Check | Result / retained log |
|---|---|
| `GOFLAGS=-p=2 make test` | PASS, exit 0; 128 Go packages, 8,390 top-level Go PASS records and 88 Pi extension tests. `/private/tmp/mission-final-test-p2.log` |
| `go test -race ./internal/mission/iteration ./internal/mission/dispatch ./internal/coordinator -run 'TestIteration|TestMission|TestAdmission|TestDispatch|TestNoFallback' -count=1` | PASS; `/private/tmp/mission-final-race.log` |
| Focused runtime and CLI integration | PASS; `/private/tmp/mission-repaired-focused.log` |
| `make lint fmt-check build check-changelog` | PASS; `/private/tmp/mission-final-quality.log` |
| `make test-launchd-drivers check-boundaries check-file-sizes` | PASS, Bash 3.2 suite, no boundary violations; `/private/tmp/mission-final-shell-boundaries.log` |
| Final build at committed implementation source | PASS; `/private/tmp/mission-canary-build.log` |

The initial sandbox full-suite attempt could not bind loopback HTTP fixture sockets. Authorized local reruns enabled loopback. The first unrestricted default-parallel run hit process-start timing failures in adapter fixtures and the existing SMT timeout fixture. Reducing Go package concurrency to two produced a complete passing run without weakening assertions. The shell suite initially found an existing static preflight-drain wiring assertion no longer recognized an inline conditional; the conditional was expanded, preserving legacy invocation and opt-in exclusion, then the whole suite passed.

An inherited lint failure compared identical calls in `msg_id_suffix_test.go`; the same determinism assertion now evaluates each call into a separate variable. No production coordinator behavior changed for that repair.

## Fault matrix and provider counts

All counts below are injected local executors, never provider inference.

| Boundary | Proven result |
|---|---|
| Useful executor + independent evaluator | Two total calls; committed artifact/check evidence; repeated completed invocation zero new calls |
| Quota denied before construction / after child claim | Zero calls while waiting; same item/deadline/request on resume |
| Availability changes after preflight | Zero calls, reason `availability` rather than quota |
| Abrupt process exit while prepared | Resume completes with two calls; no dispatched work was repeated |
| Abrupt exit after dispatch marker or synced finished receipt before DB completion | Reconciliation, zero resume calls |
| Abrupt exit after DB completion or acceptance | One remaining evaluator call; author never repeated |
| Cancellation during execution | No late acceptance; retained ambiguity holds mission admission |
| Expired parent/stage, including prepared quota wait | Durable failure or reconciliation; no deadline reset/redispatch |
| Symlink source/ancestor, foreign clone, borrowed worktree admin directory | Zero provider calls; repeated identity check before dispatch |
| `needs_decision` protocol | Waiting decision, exact evidence retained, no repeat execution |
| Final evaluator unknown metered cost / cap violation | No final acceptance |
| Git ancestry, scope, authority, dirty tree, stale judge, failed checks | Acceptance refused with retained available evidence |
| 1 MiB aggregate check output | Later checks still execute; exit codes and truncation explicit; acceptance remains bounded |
| Claude / Pi / Codex cancel, timeout, token and cost termination | Fifteen real-adapter fake-process cases stop owned descendants and preserve unrelated process |
| Foreign-project driver entry | Exactly one binary call, original cwd, expected registry path, no legacy provider probe/controller retry |

Representative red-first evidence exists for admission API absence, runtime API absence, process descendants surviving prior leader-only termination, workspace substitutions, and stage timeout reset. `/private/tmp/mission-driver-red.log` is not valid red evidence because that first run overlapped a source edit; only the final green shell fixture is claimed.

## Build artifact

- Binary: `/private/tmp/ailang-mission-iteration/bin/ailang`
- Source commit: `77dc7287e7bce225e64062452598f0694ac361fc`
- SHA-256: `1a062252d21d2e055a1b0552cfc836a9ffe634ddf27916484f81a5691c3fd2e8`

## Explicit limits and deviations

- Runtime uses the concrete local `*coordinator.SQLiteStore` rather than the proposed narrow interface. The additive store and CLI/service boundary are retained; distributed storage remains deferred.
- Existing quota observers remain bounded by their own cleanup/probe timeouts (Ollama observer up to five seconds); an expired stage does not dispatch inference, but a one-second stage need not return precisely within one second.
- Resource guards observe usage boundaries, not provider billing transactions. Unknown metered cost prevents acceptance; there is no quota reservation.
- No auto-merge, remote worker, autonomous selector, mission onboarding, decision ingestion, or outbox implementation is claimed.
- Confirmed-stop reconciliation exists in the store API but has no iteration CLI attestation verb yet. Ambiguous cancellation remains held; unattended adoption needs an operator route for that attestation.
- The main attended checkout and its unrelated uncommitted work were preserved. No messages were acknowledged/sent and no production schedule, env, or fleet binary was installed.
