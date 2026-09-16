# Sprint Plan: M-PI-HARNESS-UPGRADE

**Design doc**: [m-pi-harness-upgrade.md](m-pi-harness-upgrade.md) · **Created**: 2026-09-16 · **Freeze**: all items ratified (Mark, 2026-09-16) — pin is **0.85.1**
**Duration**: 3 days (~21h) · **Risk**: medium — the cutover is mechanical; the risk is the 0.85.1 differential surfacing a wire change the 0.84.4 rows did not see
**Sprint ID**: `M-PI-HARNESS-UPGRADE` · **Followed by**: [M-AGENT-AILANG-ONLY-EXECUTION](m-agent-ailang-only-execution.md)

## Goal

Both planes on `@earendil-works/pi-coding-agent@0.85.1`, with the harness version banked on every agent-mode row **before** the images move, and the boundary recorded before the first post-upgrade eval is banked.

## Current status

- Nothing from the design doc is implemented. `pi.go` is 799 lines, parser unchanged since the doc was written (re-read 2026-09-16): `HealthCheck` still discards `pi --version` (`health.go`), the `"Write"`/`"Edit"` comparison at `pi.go:294` is still dead, no `executor_version` anywhere in `RunMetrics`.
- Fixtures under `internal/executor/pi/testdata/` are the 0.70.2 pair (`fizzbuzz.ndjson`, `tool_use.ndjson`).
- Dockerfiles: `agent-pi` and `agent-eval` on `@mariozechner` 0.73.1 (pinned by D10); `resident` on `@earendil-works` 0.84.4.
- Rig: `pi --version` = 0.85.1 (`/opt/homebrew/bin/pi`), so every 0.85.1 capture can be made locally.
- Velocity (7d): the S1–S4 simplification program ran ~150 LOC/day of executor/config refactor with tests; this sprint is ~450 LOC + fixtures, fits 3 days.

## Milestones

Ordered so the instrument lands before the change it measures.

### M1 — Bank the harness version (~3h, ~90 LOC) ✅ 2026-09-16
**Files**: `internal/executor/executor.go`, `internal/executor/pi/pi.go`, `internal/executor/pi/health.go`, `internal/eval_harness/agent_runner.go`, `internal/eval_harness/agent_runner_multi.go`, `cmd/ailang/eval_benchmark_agent.go`, `internal/eval_harness/metrics.go`, tests
1. `Result.ExecutorVersion string` — the identity string the CLI reports (`<package>@<semver>` for pi).
2. `PiExecutor` captures `pi --version` stdout once (lazy, cached on the struct) and stamps every `Result`. `HealthCheck` keeps its existing contract and reuses the capture.
3. Plumb through `AgentBenchmarkResult.ExecutorVersion` → `RunMetrics.ExecutorVersion` (`executor_version`, `omitempty`), documented: **absent ⇒ unmeasured, never "the current one"**.
4. Other executors: capture where the CLI reports a version cheaply (claude/codex/opencode `--version`); one line each, optional per the design doc's Deferred Decisions — do pi first, others if the pattern is uniform.
**Accept**: a Result banked with no version reports absent (field omitted), not `""`; `TestHealthCheck_WithFakeBinary` proves the captured string equals the fake's output.

### M2 — Fail loud on the drift that matters (~5h, ~140 LOC) ✅ 2026-09-16 (incl. M3.0 differential — V33)
**Files**: `internal/executor/pi/pi.go`, `pi_test.go`, `health.go`, `scripts/mission_pi_run.sh`
1. Expected-version assertion: `pi.ExpectedPackage = "@earendil-works/pi-coding-agent"`, `pi.ExpectedVersion = "0.85.1"`; `HealthCheck` errors naming both when they differ. Fix the install hint (still names `@mariozechner`).
2. Unknown-event counter → `ProviderData["pi_unknown_events"] = {type: count}`.
3. Fatal path (D4): assistant `message_end` with **no `usage`** → run fails with `FinishReason = executor.FinishError` and a distinct error string prefix `pi: message_end without usage`, mapped to a new named `error_category` (not `api_error`).
4. Recognise `agent_settled`, `auto_retry_start`/`auto_retry_end`: bank `pi_retries` `{count, max_attempts, exhausted}` into `ProviderData`; a run whose `attempt == maxAttempts` is a named outcome.
5. `max` added to `validPiThinkingLevels`.
6. `mission_pi_run.sh` snapshot (V29): keep a rolling window of the last N `message_update` deltas, or correct the stated guarantee — whichever the script's consumer needs; read the consumer first.
7. Lowercase tool names at `pi.go:294` (`write`/`edit`).
**Accept**: replay fixture with an injected unknown event type banks `pi_unknown_events`; replay fixture with a usage-less assistant `message_end` fails with the named error; version mismatch test errors naming both strings.

### M3 — Wire the new signals + fixtures (~4h, ~120 LOC + fixtures) ✅ 2026-09-16
**Files**: `internal/executor/pi/pi.go`, `pi_test.go`, `testdata/v0_73_1/*.ndjson`, `testdata/v0_85_1/*.ndjson`
0. **Re-capture the differential on 0.85.1** (V7 method, on the rig): fizzbuzz + tool-use directives; diff event types and field sets against the doc's 0.84.4 table. Any new row = new work item logged in the design doc's Verification Log before continuing.
1. `usage.reasoning` → `Result.ReasonTokens`, `OutputTokens = output − reasoning` (D5); assert the identity in a test.
2. `rawStopReason` → `ProviderData`; consulted by `normalizePiFinishReason` only when `stopReason` is unrecognised.
3. Fixtures: record a genuine **0.73.1** pair (from a `docker run` of the current `agent-pi` image, or the V7 capture) and a **0.85.1** pair incl. one with non-zero `usage.reasoning`; delete the 0.70.2 pair (D6). Dual-version replay test asserts identical `Result` for the shared fields.
**Accept**: `reason_tokens > 0` on the reasoning fixture with `output + reason == usage.output`; both version dirs replay green; `testdata/fizzbuzz.ndjson` (0.70.2) is gone.

### M4 — Move the images (~4h, ~60 LOC of Dockerfile/cloudbuild) ✅ 2026-09-16 — dev verified by registry read-back; **prod rolls on the next release (tag + promote)**
**Files**: `docker/Dockerfile.agent-pi`, `docker/Dockerfile.agent-eval`, `docker/Dockerfile.agent-base`, `docker/resident/Dockerfile`, `tools/pi-extensions/sandbox/index.ts`, `cloudbuild-dev.yaml`, `docs/internal/harness-upgrade-runbook.md`
1. `PI_PACKAGE=@earendil-works/pi-coding-agent`, `PI_VERSION=0.85.1`, `npm uninstall -g @mariozechner/pi-coding-agent || true` before install (the `EEXIST` trap); build-time assertion `pi --version | grep -qx "$PI_VERSION"`.
2. `resident/Dockerfile`: bump 0.84.4 → 0.85.1; confirm the same-package reinstall path does not reintroduce `EEXIST` and the `--session-id` capability assertion still passes.
3. `agent-base`: pin the nodesource Node 22.x and add the ≥22.19.0 assertion (D8).
4. **No** workspace trust file (D7): negative container test — a sentinel extension under `<workspace>/.pi/extensions/` must not execute; positive probe = a real `tool_execution_*` event from `quota_report`. Neither step `allowFailure: true`.
5. `sandbox/index.ts` value import → `@earendil-works`.
6. Verify by reading the pushed image config (`history[].created_by`) per the runbook method, both dev and prod, for `agent-pi`, `agent-pi-go`, `agent-eval`, `agent-eval-go`, `resident-pi`.
**Accept**: a deliberately wrong `PI_VERSION` fails the build; a Node below the floor fails the build; read-back shows 0.85.1 on every listed image.

### M5 — Boundary bookkeeping + re-baseline (~5h, docs + one eval run) ✅ 2026-09-16 — boundary recorded; rig step sized from existing rows (no new spend); no cloud pi eval lane exists
**Files**: `design_docs/v1-mission.md` (charter), `docs/internal/harness-upgrade-runbook.md`, memory, `docs/docs/guides/evaluation/` caveats
1. Record the boundary (date, both versions, which images, which rows carry `executor_version`) **before** any post-upgrade eval banks.
2. Comparator set (agent picks; must have pre-boundary rows on both planes): run under 0.85.1, report with `ailang eval-paired`.
3. Annotate pre-boundary rows — never re-bank (D2).
**Accept**: runbook + charter + memory updated in the same commit as the first `executor_version`-carrying row lands; `eval-paired` report committed.

## Day plan

| Day | Work |
|---|---|
| 1 | M1 (am) · M2.1–2.5 (pm) |
| 2 | M2.6–2.7 · M3.0 differential · M3.1–3.3 |
| 3 | M4 · M5 |

## Success metrics
- All six design-doc success metrics, measured as stated there
- `make test-core` and `go test ./internal/executor/... ./internal/eval_harness/...` green after every milestone
- `make check-pi-wire-budget` PASS on 0.85.1
- CHANGELOG entry under `changelogs/v0.32-current.md`

## Risks
| Risk | Mitigation |
|---|---|
| 0.85.1 differential shows a new schema change | M3.0 runs before M3.1; a change becomes a logged work item, not a silent skip |
| `EEXIST` on the same-package reinstall in resident | M4.2 checks it explicitly; the uninstall line stays unconditional |
| Cloud Build step for containers cannot run locally | Assertions are Dockerfile `RUN` lines so `docker build` reproduces them on the rig |
| Comparator run spends real quota | Core tier only, one plane at a time, `--models` explicit (eval-suite has no safe no-op) |

## Outcome (2026-09-16)

| | Planned | Actual |
|---|---|---|
| Duration | 3 days | 1 attended session (~4h wall) |
| LOC | ~450 | ~1,020 incl. fixtures (5 NDJSON captures) and the acceptance script |
| Commits | — | `95f844b42` M1 · `0328cd9df` M2 · `c7464a28e` M3 · `5ef7a2b24` M4 · `9e52d9a13` M5 |
| Cloud Build | — | `4a6ac6f6` dev SUCCESS; `test-agent-pi` 12/12 in-container; read-back: 5/5 dev images `@earendil-works@0.85.1`, Node `22.23.2` |

**Residuals, stated:**
- **Prod images** (`agent-pi` 0.73.1, `agent-eval` 0.73.1, `resident-pi` 0.84.4) move on the next release tag + promote — the design's success metric 1 is met for dev only until then.
- `make check-pi-wire-budget` INCONCLUSIVE ×3 — not this lane: the prod observatory's newest `LLM Generation` span is 05:40Z and four OpenRouter runs since never ingested (Broadcast → observatory ingest outage).
- The rig's own `~/.pi/agent/extensions` holds 3 of 13 embedded extensions (`ailang pi status` → 12 MISSING). Found, not fixed.
- The rig's installed `ailang` binary predates M1–M3 (`⚠ Binary may be stale`); `make quick-install` when the shared tree is clean, so rig rows start banking `executor_version` and the version assertion goes live there.
- codex/opencode `--version` capture — two lines + result stamping each via `executor.VersionProbe`; deferred per the design doc.
- No cloud pi **eval** lane exists (cloud pi = coordinator jobs), so D2's cross-plane comparator is N/A; the rig boundary was sized from existing rows instead.
