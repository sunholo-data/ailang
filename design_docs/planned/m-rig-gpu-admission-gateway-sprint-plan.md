# Sprint Plan — M-RIG-GPU-ADMISSION-GATEWAY Phase 2

**Design doc:** [m-rig-gpu-admission-gateway.md](m-rig-gpu-admission-gateway.md)  
**Sprint ID:** `M-RIG-GPU-ADMISSION-GATEWAY-P2`  
**Planned at:** `6c2dc3e1` (`coordinator/task-58385fae`), 2026-09-27  
**Phase 1 reference:** `6d39972ff` (reported pushed; not present in this checkout's reachable commit graph)  
**Duration:** 3 engineering days + attended cutover + 7-night soak  
**Risk:** High at cutover; medium before cutover  
**Scope:** Remaining AILANG-side work only. Daneel and eparse delivery are external prerequisites.

## 0. Outcome and non-negotiable order

This sprint makes every AILANG eval lane present its minted rig lease, closes the motoko orphan-registration gap, adds a bypass alarm, and then performs the port cutover under Mark's supervision.

The executor must preserve this sequence:

`D1–D5 ratified` → `Daneel + eparse token-ready` → **M1 clients** → **M2 motoko child registration** → **M3 bypass detector** → `Mark go` → **M4 attended cutover** → **7-night soak**

M1–M3 may be implemented and merged before external readiness. M4 must not begin until every pre-cutover gate is green. No step may silently fall back to direct ollama.

## 1. Current state and planning evidence

- Phase 1 is documented as built: `ailang rig-gate`, AILANG admission policy, streaming proxy/ledger, Go and shell lease minting, and executor-wide `AILANG_RIG_LEASE=none` default.
- `tools/launchd/dev.ailang.rig-gate.plist` exists but is intentionally not installed; its header pins the same cutover order as this plan.
- Local pi and opencode lanes in `internal/modelreg/models.yml` still name `ollama/...`; they do not yet select a leased provider.
- The canonical pi config is `tools/pi-extensions/models.mission.json`; `internal/eval_harness/pi_models_config_test.go` currently recognizes only `openrouter` and `ollama`, so M1 must extend the fixture/test contract for `ollama-rig` without weakening the everyday `ollama` checks.
- Motoko reaches `cmd.Run()` in `internal/executor/motoko/motoko.go`; pi and opencode already use `Start` → `riglock.RegisterChild` → `Wait`.
- `tools/launchd/rig-watchdog.sh` probes ollama and orphan sockets on `127.0.0.1:11434`; after cutover the upstream probe/socket selection must move to `11435`, while the public gateway remains `11434`.
- The repository worktree was clean at planning time. The velocity helper found only the v0.46.0 release commit and no usable LOC/day sample, so estimates below are bottom-up with a 30% integration buffer rather than a fabricated velocity number.

## 2. Preconditions and human gates

### Gate P0 — before execution starts

- [ ] Mark ratifies design freeze D1–D5, including D5's initial **no-pf** posture.
- [ ] Confirm Phase 1 commit `6d39972ff` is an ancestor of the execution branch (or identify its squash/rebase equivalent).
- [ ] Executor records pristine baselines for the named test packages and shell tests before edits.

### Gate P1 — before M4 cutover only

- [ ] Daneel mints the shared token and sends `X-Rig-Lease` on `/api/generate`; cross-repo verification is recorded.
- [ ] eparse sends the token; cross-repo verification is recorded.
- [ ] M1, M2, and M3 are merged, installed on the rig, and their acceptance gates are green.
- [ ] No active GPU job; rig lock is free; rollback commands and last-known-good installed plists are captured.
- [ ] Mark gives an explicit attended go.

If P1 is not green, stop after M3. This is a planned pause, not a reason to weaken admission.

## 3. Milestones

### M1 — Route eval-lane clients through leased providers

**Goal:** pi and opencode local eval lanes send `AILANG_RIG_LEASE` while everyday/mission `ollama` remains usable without that variable.  
**Estimate:** ~90 implementation/config LOC + ~90 test/fixture LOC; 1 day.  
**Depends on:** P0.

**Files:**

- `tools/pi-extensions/models.mission.json`
- the installer/sync path that produces `~/.pi/agent/models.json`
- opencode's canonical config/fixture (including `internal/executor/opencode/testdata/opencode_ollama_config.jsonc` where applicable)
- `internal/modelreg/models.yml`
- `internal/eval_harness/pi_models_config_test.go` and relevant opencode config tests
- harness-upgrade/provenance documentation if required by the banked boundary

**Tasks:**

1. Add a separate pi provider named `ollama-rig`, cloning only the local on-device model definitions needed by pi lanes. Set `headers.X-Rig-Lease` to `${AILANG_RIG_LEASE}`. Do not add templating to the existing `ollama` provider and do not move Ollama Cloud/interactive mission rows.
2. Add the leased opencode provider configuration with `apiKey: "{env:AILANG_RIG_LEASE}"`; preserve `baseURL` at the public gateway `http://127.0.0.1:11434/v1`.
3. Change only on-device pi/opencode eval rows in `internal/modelreg/models.yml` from `ollama/<model>` to `ollama-rig/<model>`. Motoko and Ollama Cloud rows stay on their existing provider because their routing/auth semantics differ.
4. Extend config drift tests so they understand `ollama-rig` as a distinct wire provider but compare its model budgets against the underlying local models. Add negative assertions that everyday `ollama` contains no lease template and local eval rows cannot regress to it.
5. Run one pi and one opencode leased smoke through a capture/fake gateway. Bank both resulting rows and assert non-empty `executor_version`; record this models.yml wire boundary in the harness upgrade ledger/runbook. A row without `executor_version` does not prove the intended harness carried the lease.

**Acceptance criteria:**

- [x] With `AILANG_RIG_LEASE=test-token`, pi config sends `X-Rig-Lease: test-token`; opencode config sends `Authorization: Bearer test-token` through its `apiKey` contract.
- [x] With the variable unset, the everyday `ollama` pi provider remains lease-free; the `ollama-rig` provider uses pi's fail-loud environment template.
- [x] Every on-device pi/opencode eval row uses `ollama-rig/...`; no Cloud, interactive, mission, or motoko row is accidentally moved.
- [x] Canonical config/install drift tests pass and include a mutation that changes one local row back to `ollama/...` and makes the test fail.
- [ ] Rig smoke artifacts with non-empty `executor_version` remain a Gate P1 installation check; the unattended cloud executor has neither pi nor opencode installed and did not fabricate runtime rows.

**Risk:** pi config is both a user-level installed file and a repo fixture. Mitigation: update the canonical source and install path together, diff installed vs canonical, and never hand-edit only `~/.pi/agent/models.json`.

### M2 — Register motoko as a rig-lock child

**Goal:** motoko has the same orphan-reaping registration as pi/opencode.  
**Estimate:** ~25 implementation LOC + ~55 test LOC; 0.5 day.  
**Depends on:** M1.

**Files:** `internal/executor/motoko/motoko.go`, motoko executor tests.

**Tasks:**

1. Replace the blocking `cmd.Run()` with `cmd.Start()`, immediate `riglock.RegisterChild(cmd.Process.Pid, cmd.Path)`, then `cmd.Wait()`.
2. Preserve existing semantics: stdout/stderr wiring, start time, debug diagnostics, parse-after-nonzero-exit behavior, timeout/process-group handling, and exactly one wait.
3. Add an injectable registration seam or subprocess test proving registration occurs after successful start and before wait; prove start failure neither dereferences `cmd.Process` nor registers PID 0.

**Acceptance criteria:**

- [x] A held-lock motoko child appears in the riglock children ledger before the fake child exits.
- [x] No-lock execution remains a no-op for registration.
- [x] Start failure returns without panic; the existing parse-after-nonzero path remains after the single Wait.
- [x] Motoko package tests, riglock tests, formatting, and lint are green. Race execution requires CGO and is unavailable on this cloud worker.

### M3 — Detect traffic that bypasses the gateway

**Goal:** every complete minute is reconciled between ollama's long-request GIN lines and the gateway ledger; excess upstream traffic alerts at most hourly.  
**Estimate:** ~100 shell LOC + ~110 shell-test LOC; 1 day.  
**Depends on:** M2.

**Files:** `tools/launchd/rig-watchdog.sh`, a new focused shell test such as `tools/launchd/test_rig_watchdog_bypass.sh`, and watchdog plist comments/config only if state paths need documenting.

**Contract:**

- Count only `/v1/chat/completions`, `/api/chat`, and `/api/generate`.
- Compare the last **complete** UTC/local minute consistently on both sides; exclude the in-progress minute to avoid false positives from streaming completion timing.
- `ollama_count > gateway_count` is a bypass. Gateway excess is diagnostic drift, not a bypass.
- Persist the last successful reconciliation cursor and last-alert timestamp under `~/.ailang/state/` (or the existing shared state convention), using atomic replacement and a single-run lock so overlapping ticks cannot double-alert.
- Missing/unreadable logs are an observable detector error, not `0`; log the condition and do not claim “no bypass.”
- On bypass, always write a watchdog log line with minute and both counts. Send at most one `controlplane` message per rolling hour, but continue logging each excess minute.

**Acceptance criteria:**

- [x] Fixture table covers equal counts, ollama excess, gateway excess, irrelevant endpoints, minute boundary exclusion, rotated/truncated log, missing ledger, duplicate watchdog invocation, and alert re-arm after 60 minutes.
- [x] First ollama excess logs and messages; a second within the hour logs but does not message; the first after the hour messages again.
- [x] A mutation from `>` to `>=` fails the equal-count test; removing any one endpoint fails its endpoint case; treating a missing ledger as zero fails loudly.
- [x] Tests do not read live rig logs or send real messages; paths, clock, minute, and notifier are injected.

### M4 — Attended cutover, rollback drill, and soak

**Goal:** move ollama to private `:11435`, put rig-gate on public `:11434`, supervise both, and prove the production metrics.  
**Estimate:** ~25 repo config LOC + ~30 test/docs LOC; 0.5 engineering day plus attended window and 7 nights.  
**Depends on:** P1 and explicit Mark go.

**Attended runbook (exact order):**

1. Pause scheduled GPU consumers and verify no active holder/request. Save installed plist copies and current `launchctl print` state for rollback.
2. Change `OLLAMA_HOST` to `127.0.0.1:11435` in both `tools/launchd/dev.ollama.serve.plist` and `~/Library/LaunchAgents/dev.ollama.serve.plist`. Use the `.claude/rules/local-models.md` `plutil -p` diff to prove equality.
3. Change the watchdog's **ollama upstream** health probe and orphan-socket inspection to pinned `127.0.0.1:11435`; retain the #557 IPv4 pin. Do not change eval clients from public `11434`.
4. Reload/restart ollama and prove `/api/tags` responds on `11435` while `11434` is still deliberately absent.
5. Copy/install `dev.ailang.rig-gate.plist`, validate it with `plutil`, bootstrap it, and prove public `11434` proxies to private `11435`.
6. Add rig-gate to watchdog health/re-bootstrap handling. Exercise failure recovery for each service separately: dead gateway is restarted without moving clients direct; dead ollama is restarted on `11435`.
7. Run a leased pi, opencode, and motoko smoke; run an unleased/foreign long request during a held lease and require 423 under 100 ms; verify the ledger and bypass reconciler agree.
8. Resume scheduled consumers only after all smoke gates pass.

**Rollback trigger and procedure:**

- Trigger on failed gateway bootstrap, leased-client rejection, streaming corruption, watchdog restart loop, or unexplained bypass count.
- Pause GPU consumers; boot out rig-gate; restore both ollama plists to `127.0.0.1:11434`; restore watchdog probe/socket target; reload ollama; diff repo/installed copies; record the failure. Never enable an automatic client fallback.

**Seven-night soak:**

- Nightly record: gateway admitted/refused counts by lease label, upstream-vs-ledger reconciliation, client timeout signatures, nightly validity category, watchdog restarts, and ledger summary.
- Pass after 7 consecutive nights with: zero admitted orphan/unleased long requests while held; zero 4m59s queue-timeout failures; zero bypass alerts; zero `infra_outage` nights; ledger answers who/when/GPU duration. The design doc's 14-night metric remains the post-sprint confirmation target.
- Any bypass alert pauses the soak and brings D5/pf back to Mark; do not silently reset the seven-night clock.

**Acceptance criteria:**

- [ ] Repo and installed ollama plists are `plutil`-equivalent and pin `127.0.0.1:11435`.
- [ ] Public `127.0.0.1:11434` is owned by rig-gate; private `127.0.0.1:11435` is owned by ollama; neither listens on a non-loopback address.
- [ ] Watchdog independently detects and recovers each service without confusing `localhost`/IPv6.
- [ ] Leased three-harness smoke passes, foreign/unleased refusal is <100 ms, and no direct-port request is emitted by an eval client.
- [ ] Seven-night report is attached to the design doc; docs/runbooks/changelog are updated. Phase 2 is not marked complete before this report.

## 4. Verification matrix

| Gate | M1 | M2 | M3 | M4 |
|---|---:|---:|---:|---:|
| Config/registry unit tests | required | — | — | required |
| `go test ./internal/executor/pi ./internal/executor/opencode ./internal/eval_harness` | required | — | — | smoke |
| `go test ./internal/executor/motoko ./internal/riglock` | — | required | — | smoke |
| Focused watchdog shell fixtures | — | — | required | required |
| `make fmt`, `make lint`, `make check-boundaries` | required | required | required | required |
| Live GPU | capture/fake first; one leased smoke | no | no | attended only |

The executor must record baseline failures before attributing them to the sprint. Named-test commands must also assert at least one test ran; rc=0 with “no tests to run” is not evidence.

## 5. Estimates and schedule

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 provider split, registry rewiring, drift tests, version-banked smokes | Local pi/opencode lanes demonstrably send the lease; ordinary ollama stays untouched |
| 2 AM | M2 Start/RegisterChild/Wait + regression tests | Motoko child registered before wait with old exit parsing preserved |
| 2 PM–3 AM | M3 reconciler, durable throttle state, shell fixtures | Bypass excess is detected and hourly controlplane throttle is mutation-tested |
| 3 PM | Integration gates, docs, cutover packet and rollback rehearsal without changing live ports | P1 packet ready for Mark |
| Attended | M4 exact-order port move and three-harness smoke | Both services supervised; refusal/ledger/reconciliation gates green |
| Nights 1–7 | Soak and daily metric capture | Seven consecutive qualifying nights |

**Estimated changed LOC:** ~520 total (implementation/config ~240; tests/fixtures ~280). Operational installed-copy edits and soak records are excluded.

## 6. Definition of done

- M1–M3 code and tests are merged in dependency order.
- M4 has Mark's recorded go and all external prerequisites are evidenced.
- Repo and installed launchd configuration match after cutover.
- Seven-night soak passes; 14-night design metric continues tracking.
- `design_docs/planned/m-rig-gpu-admission-gateway.md`, `changelogs/v0.32-current.md` (current unreleased section), `docs/docs/guides/evaluation/local-ollama.md`, local-ollama eval runbook/skill, and harness upgrade ledger reflect the delivered boundary.
- D5 remains “no pf” only while the detector stays quiet. A detector firing is a human decision point, not executor latitude.

## 7. Executor handoff constraints

This plan is ready for review, not execution. Per repository workflow, Mark/user must approve it and then explicitly say **“execute sprint”** before `sprint-executor` begins. The executor must stop at P1 and again for Mark's attended go; neither pause authorizes live port changes.
