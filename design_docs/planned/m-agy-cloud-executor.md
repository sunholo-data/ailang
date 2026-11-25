# M-AGY-CLOUD: agy on the cloud executor plane, on the Google AI subscription

**Status**: Phases 0–2 live; Phase 3 (D6) rolled out 2026-11-25. ailang v0.61.1 built and promoted `agent-agy`/`agent-agy-go`; ailang-multivac `83a882f` (dev + prod) sets `enable_agy` in both envs and adds the twin agents `sprint-planner-agy` / `sprint-executor-agy` (gemini-4-neon-high) beside the codex pair. Prod secret `ailang-agy-oauth-token` seeded from dev. Dev smoke (task-b8155536): subscription auth, lease, token write-back and release verified; the Google pace gate then refused ("weekly usage 34.0% exceeds the 15%/day pace line 15.6%; not cloning", $0). **No agy task has completed in the cloud yet** — rerun the dev and prod smoke once the shared Google week is under the line.
**Target**: v0.60.x
**Priority**: P1
**Estimated**: Phase 0 ~1 day incl. sign-in tooling (one cloud sign-in by Mark); Phase 1 ~2 days; Phase 2 ~1 day (multivac terraform + images); Phase 3 is a routing decision, not work
**Dependencies**: [M-AGY-EXECUTOR](m-agy-executor.md) (built 2026-11-23: executor, pinned installer, sign-in tooling, `google` ration). Reuses the codex cloud subscription machinery from [M-CODEX-SUBSCRIPTION-LANE, operational addendum](v0_52_6/m-codex-subscription-lane.md#operational-addendum-three-oauth-profiles-on-one-account-2026-10-09).

## Scope ruling (Mark, attended 2026-11-24)

> "I don't want the api key, we have that covered already in managed agents etc. we have subscription working for other executors, right?"

So this doc covers the **subscription route only**. The parent doc's Future Work item "Cloud Pillar 2 for the API-key route" is dropped: metered Gemini in the cloud is `gemini_agents` / managed agents. Nothing here creates or grants `GEMINI_AGENTS_API_KEY` to an agy job.

And yes, subscription credentials already work for two cloud executors (V1, V2), which is the precedent this doc follows:

| Executor | Cloud credential | Refresh behaviour | Concurrency guard |
|---|---|---|---|
| claude | `claude setup-token`, cloud-only holder, raw `sk-ant-oat…` in Secret Manager `ailang-claude-code-oauth-token` | **None**: a long-lived token, no refresh token rides along (ailang `07c7599bb`) | Not needed: nothing rotates |
| codex | ChatGPT `auth.json` from a dedicated cloud-only `codex login`, secret `ailang-codex-auth-json` | **Rotates**: codex rewrites the file; the job writes it back as a new secret version | Exclusive Firestore lease per secret (`internal/credentiallease`), dispatcher lane gate (`credential_lane.go`), quarantine when process termination is unconfirmed |
| **agy (this doc)** | Cloud-only `agy` sign-in, token file in Secret Manager | **Rewrites its file on every access-token refresh** (V6). The refresh token itself did not change across one observed refresh (V7) | The codex machinery, generalised (below) |

agy has no long-lived-token mode like `claude setup-token` (V8), so agy is a codex-shaped problem, not a claude-shaped one.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | An LLM executor like every other one; no language change |
| A2: Replayability | 0 | Same stream-json banking as the local executor |
| A3: Effect Legibility | +1 | The credential's lifecycle (lease, install, write-back, quarantine) becomes one explicit, logged path shared by codex and agy instead of a codex-only special case |
| A4: Explicit Authority | +1 | The job gets one named secret and an isolated `HOME`; no API key, no ambient `~/.gemini`. The cloud sign-in is its own holder, never a copy of the rig's |
| A5: Bounded Verification | 0 | No effect |
| A6: Safe Concurrency | +1 | Exclusive lease per rotating credential, enforced across job variants and retries, with termination proof before release |
| A7: Machines First | +1 | Adds flat-rate Google capacity to the cloud fleet, which today has no Google editing lane on subscription |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +1 | Runs bank `cost_provenance: list-price-equivalent`; the job reads `agy -p /quota` before running so cloud spend is paced against the same account the rig rations |
| A10: Composability | +1 | Fits the executor contract; one more variant in the dispatcher table |
| A11: Structured Failure | +1 | Over-ration, sign-in lost, lease held and termination unconfirmed each fail with a named reason before the repo is cloned |
| A12: System Boundary | 0 | One more pinned, checksum-verified vendor CLI in an image |

**Net Score: +7** → **Decision: proceed, gated on Phase 0**

### Hard Violation Check

- [x] A1: no nondeterminism beyond the LLM itself
- [x] A3: credential writes are explicit secret versions, logged, never silent
- [x] A4: no API key, no shared holder; the secret is granted to the agy jobs' service account only
- [x] A7: motivation is fleet capacity

## Problem Statement

**Current state:**
- agy runs only on the rig: the stapledon and World mission loops' executor fallbacks (2026-11-23/24) and attended evals. The cloud fleet has no way to use the Google AI Ultra subscription.
- The cloud's Google options are metered: `gemini_agents` (Developer API key, bills `ailang-gemini-agents`) and Vertex reviewers in design-quorum. Both are per-token.
- The Ultra plan is paid flat-rate and, outside the two mission fallback rungs, mostly idle: weekly Gemini usage read 0% on 2026-11-24 after a window reset.

**Impact:**
- Cloud agents that run codex today have nothing behind them when the ChatGPT bucket is dry. A second subscription lane in the cloud is capacity we already pay for.
- Doing this naively (copying the rig's token into a secret) would reproduce the two outages the claude and codex lanes already paid for: one shared refresh state, and the holder that refreshes last logs out the other (2026-10-07: 16 h of failed loop fires; 2026-10-09: cloud claude failed on turn 1).

## Goals

**Primary goal:** a cloud executor job that runs agy on the Ultra subscription with its own sign-in, never logs out the rig, and never runs two holders on one rotating credential.

**Success metrics:**
- A dev-plane task dispatched to the agy variant completes with `cost_provenance: list-price-equivalent`, and the secret has a new version afterwards if agy refreshed.
- The rig's mission loops' agy sign-in still works after ≥3 cloud runs that each refreshed (no cross-holder logout).
- A second agy task dispatched while one runs is held pending by the dispatcher (not launched to fail on the lease).
- With the `google` pace line exceeded, a cloud agy job refuses before cloning, with the ration reason in its completion.
- `go test ./cmd/ailang/ ./internal/coordinator/ ./internal/executor/agy/ ./internal/executor/codex/` green, including the existing codex lease tests unchanged in behaviour.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. Cloud credential holder: a separate cloud-only `agy` sign-in on the same Ultra account | A copied token shares refresh state with the rig (the 10-07 / 10-09 outages) | **human** (recommended: separate sign-in, the codex three-profile contract) | design | high |
| D2. Concurrency: one cloud agy run at a time (one leased secret) vs a pool of N cloud sign-ins | One lease serialises cloud agy work; a pool multiplies holders on one account and the provider's limit on authorisations is unpublished | **human** (recommended: one lease now, like codex. If P3 confirms the refresh token never rotates, concurrent holders cannot revoke each other and the lease can later be relaxed to allow N concurrent jobs on one secret, with write-back still serialised) | design | med |
| D3. Ration: cloud runs share the account's `google` pace line (default 15% of the weekly window per day since it started, V14) with the rig, checked in-job via `agy -p /quota` | The allowance is per account, so cloud and rig spend one budget; an unpaced cloud lane could starve the mission fallbacks | **human** (recommended: one shared pace line, cloud checks it before cloning, fail-closed) | design | med |
| D4. Sandbox posture inside Cloud Run — **resolved by P5: keep `--sandbox`** | `--sandbox` may not work in a Cloud Run job container (P5). Without it, the container is the boundary | **human ratifies** after Phase 0 data | Phase 1 | med |
| D5. Generalise the codex rotating-credential code rather than copy it | Two copies of lease/write-back/quarantine logic drift; a bug fixed in one stays in the other | agent (audit-before-patching rule) | design | med |
| D6. Which cloud agents get agy, and as pin or fallback | Decides real usage and what a dry bucket costs | **human** | Phase 3 | low |

### Design Freeze

- [x] D1 — separate cloud-only sign-in (Mark) — ratified 2026-11-24
- [x] D2 — one leased credential to start (Mark) — ratified 2026-11-24
- [x] D3 — shared pace line, in-job check (Mark) — ratified 2026-11-24
- [x] Phase 0 closed 2026-11-24: P1–P7 all measured

## Solution Design

### Overview

Treat agy exactly like cloud codex: a cloud-only sign-in kept in one Secret Manager secret, leased exclusively per execution, installed into an isolated `HOME`, written back after the task if agy rewrote it, quarantined when the process cannot be proven stopped. The codex-specific code that does this today becomes a provider-neutral **rotating file credential** with two instances, codex and agy.

### Architecture

**1. Rotating file credential (generalised from codex).** `cmd/ailang/coordinator_cloud_codexauth.go` holds `codexCredential`: canonical secret name, Firestore lease (`acquire` → `heartbeat` → `release` | `quarantine`), install, baseline, write-back of changed bytes, quarantine on `ErrProcessTerminationUnconfirmed`. Only four things in it are codex-specific:

| Seam | codex | agy |
|---|---|---|
| Secret env var | `AILANG_CODEX_AUTH_SECRET` | `AILANG_AGY_AUTH_SECRET` (new) |
| Validate a credential | `codex.ParseSubscriptionAuth` (refuses api-key mode) | `agy.ParseSubscriptionToken` (new): JSON with `token.refresh_token`, `auth_method`; refuses anything else |
| Install / locate | `codex.InstallSubscriptionAuth`, `codex.InstalledAuthPath` (`$CODEX_HOME/auth.json`) | `agy.InstallSubscriptionToken`, `agy.InstalledTokenPath` (`<route HOME>/.gemini/antigravity-cli/antigravity-oauth-token`, mode 0600) |
| Home precondition | explicit isolated `CODEX_HOME` | explicit `AILANG_AGY_HOME_ROOT` |

These become a small interface (`rotatingCredentialSpec`) and the lease/write-back/quarantine logic is shared. `installCodexCredential` becomes `installRotatingCredential(ctx, provider, project)` returning nil for providers with no rotating credential.

**2. Termination proof in the agy executor.** The lease may only be released after the refresh-capable process is gone. codex returns `ErrProcessTerminationUnconfirmed` when it cannot join its process; the agy executor calls `proctree.Kill` and ignores the result (V9). It gains the same contract: after kill, wait for the group with a bounded join; if the join fails, return a sentinel the cloud entry point maps to `quarantine`. The sentinel moves to `internal/executor` so codex and agy share one error.

**3. Dispatcher lane.** `credentialLane("agy")` returns `agy-subscription`, so a second agy task waits pending instead of launching to die on "credential already owned" (the codex lesson of 2026-10-12/13). The lane key derives from the provider; both agy variants share it.

**4. In-job ration check (D3).** After the lease and install, before the clone, the job runs `agy -p /quota` under the installed `HOME` (no model call, ~3 s, V10) and evaluates it with the same pacing the rig uses (`AILANG_GOOGLE_DAILY_PACE`).

**Where each half lives.** Spawning and evaluating are split, because the process-control machinery is in `internal/executor` and the pace logic is in `internal/mission`:
- **Spawn:** `agy.RunQuota(ctx, home) ([]byte, error)`, new in `internal/executor/agy`, uses the executor's own bounded runner (`proctree.Configure`, hard deadline, bounded group join) and returns `executor.ErrProcessTerminationUnconfirmed` when the join fails. `cmd/ailang` calls it while holding the lease.
- **Evaluate:** `mission.EvaluateGoogleQuota(out, runErr, home, pace, now)`, the existing `evaluateGoogleQuota` exported unchanged (V13). It spawns nothing.

`internal/mission`'s own `runAgyQuota` (the rig path, no lease) is untouched; the cloud path never calls it. **The quota read is itself a refresh-capable agy process**, so it runs under the full credential contract, exactly like the task's own agy run:
- It is started through the same bounded runner as the executor (process group, hard deadline, bounded join), and a failed join returns `ErrProcessTerminationUnconfirmed` → quarantine.
- Every exit after it — over the line, unreadable, or ok-then-task — goes through `finish()`: ownership heartbeat, write-back if the token file changed, then release. There is no "release without write-back" path once any agy process has run. The only bare release left is the existing one for failures *before* the first agy process (fetch or install failed).

Over the line or unreadable → the job publishes the ration reason, runs `finish()`, and the task is banked as an ordinary failure before the clone. Because quota is per account, the rig's mission ration sees cloud spend on its next read without any shared ledger.

**This is admission, not reservation.** `/quota` reflects spend already made, so a rig fire and a cloud job that check at the same moment can both pass and both run. That is already true between the rig's own loops today (stapledon and World each check independently). The overshoot is bounded by the work in flight when the line is crossed, at most one cloud agy task (D2's single lease) plus the rig fires that started in that window, and the 2-point start margin (V14) exists to absorb it. A shared reservation ledger across the two planes is a non-goal unless Phase 3 usage shows the overshoot matters.

**Pace semantics** are the evaluator the rig already runs, unchanged (V14): the weekly window's allowance is `pace × days since the window started` (start = reported reset − 7 days), capped at 100%. The job is over when weekly used > allowance, when headroom is under the 2-point start margin (except for an unused window), or when either window is exhausted. The five-hour window is not paced. So with the default 15%/day, the line is 15% one day into the window, 30% on day two, 100% from day 6⅔.

**5. Images and jobs.** `docker/Dockerfile.agent-agy` (FROM `agent-base`, installs pinned agy via `tools/agy/install.sh` with Linux values, `ENV AILANG_IMAGE_PROVIDER=agy`, `AILANG_AGY_HOME_ROOT=/home/ailang/.agy`) and `Dockerfile.agent-agy-go` (adds the Go toolchain, mirroring `agent-codex-go`). Dispatcher variants `agy` and `agy-go` → provider `agy`. In `ailang-multivac`: two `google_cloud_run_v2_job` resources cloned from the codex pair, one secret `ailang-agy-oauth-token` (accessor + version-adder for the agent service account only), image builds in `cloudbuild-images.yaml`.

**6. Cloud sign-in tooling (D1, built in Phase 0 as P0a).** `tools/agy/signin.sh --profile cloud` signs in under a fresh, separate `HOME` on the rig (never the loops' HOME, never `~/.gemini`), then `--upload <project>` pipes the token file into `gcloud secrets versions add … --data-file=-` and deletes the local copy. The value is never printed.

### Implementation Plan

**Phase 0: measurement spike** (~1 day; needs Mark for one sign-in). Each P-item closes one PENDING row. Run as a one-off job in the **dev** project, not prod. The credential tooling comes first, so the spike's sign-in already has D1's guarantees:
- [x] P0a (b5ede5610) `tools/agy/signin.sh --profile cloud`: sign in under a fresh `HOME` (never the loops' HOME, never `~/.gemini`); `--upload <project> <secret>` pipes the token file into `gcloud secrets versions add … --data-file=-`, never prints it, deletes the local copy
- [x] P0b (2026-11-24) dev secret `ailang-dev-agy-oauth-token` and SA `ailang-dev-agy-spike` (no project roles; accessor + version-adder on that secret only), created by hand, labelled `purpose=m-agy-cloud-phase0` with accessor + version-adder for the spike job's service account only (the terraform version is Phase 2)
- [x] P0c no image needed: `tools/agy/cloud_spike_run.sh` runs `tools/agy/cloud_spike.sh` in the stock `google-cloud-cli:slim` image as job `ailang-dev-agy-spike`; Linux SHA512s pinned in `install.sh`. Dry run before sign-in (execution `ailang-dev-agy-spike-xhx27`): pinned 1.5.15 installs and runs on Debian 13 x86_64; unauthenticated agy waits 60 s for the sign-in then exits 1 (no hang), and offers a paste-the-code sign-in, so P2's in-container fallback exists
- [x] P1 (execution `ailang-dev-agy-spike-j9trc`, 2026-11-24: rc 0 in 5 s, events init/step_update/result, status SUCCESS) Linux agy 1.5.15 runs headless `-p` with `--output-format stream-json` in a Debian container: no hang (the #318 class), same event schema as our fixtures
- [x] P2 (same run: the uploaded macOS-minted token served Neon from the file, no keyring) A token file from a macOS sign-in works under a fresh Linux `HOME` with no keyring daemon (file credential, not keyring). **Fallback if not:** sign in inside the Linux container instead (agy prints a sign-in URL when unauthenticated: "Authentication required. Please visit the URL to log in", seen 2026-11-23) and upload from there; P0a gains an `--in-container` mode
- [x] P3 (12:16 run, token expired 12:14: agy refreshed in the container, the file changed, the refresh token did NOT, and the spike wrote it back as secret version 2) Refresh-token rotation (V7): does `token.refresh_token` change when agy refreshes?
- [x] P4 (`make agy-check` on the rig passed after the cloud sign-in existed AND after the cloud copy refreshed) Coexistence: after the cloud copy refreshes several times, the rig loops' sign-in still works, and vice versa
- [x] P5 (same run + negative control: `echo > out.txt` in the workspace succeeded; `echo > /var/tmp/outside.txt` failed `Read-only file system`, file absent. The sandbox is ENFORCED in a Cloud Run job, so **D4 = keep `--sandbox`**, the rig posture) `--sandbox` inside a Cloud Run job container (the agent jobs do not set `execution_environment`, V15, so whatever the default gives): works, or fails with the sandbox configuration error our executor already detects (feeds D4)
- [x] P6 (same run: /proc scan found no agy-spawned process) After agy exits, nothing it spawned keeps running (no lingering process that could refresh after the lease is released)
- [x] P7 (same run: weekly reset 2026-12-01T06:56:36Z, the rig's window) `agy -p /quota` from the cloud `HOME` reports the same windows as the rig's (same account)

**Phase 1: code (ailang)** (~2 days)
- [ ] Generalise `codexCredential` to a rotating-file credential with a per-provider spec; codex behaviour unchanged (existing tests pass unmodified)
- [ ] `agy` subscription token: parse/validate, install, path; refuse an API-route settings file under the cloud HOME
- [ ] Shared `executor.ErrProcessTerminationUnconfirmed`; agy executor joins its process group with a bound and returns it on failure
- [ ] `credentialLane("agy")`, dispatcher variants `agy`/`agy-go`, `AILANG_AGY_AUTH_SECRET` in the config registry
- [ ] `agy.RunQuota` (bounded, termination sentinel) + exported `mission.EvaluateGoogleQuota`; in-job `google` ration check before clone in `cmd/ailang`
- [ ] `tools/agy/install.sh` Linux arm64 value if Phase 0 pinned only amd64
- [ ] Dockerfiles `agent-agy`, `agent-agy-go`

**Phase 2: infrastructure (ailang-multivac)** (~1 day)
- [ ] Secret `ailang-agy-oauth-token` (dev and prod), IAM for the agent SA only
- [ ] Jobs `agent-executor-agy`, `agent-executor-agy-go`; image builds
- [ ] Dev-plane smoke: one task end to end; secret version count before/after; rig loops' agy probe still passes

**Phase 3: routing** (decision, D6)
- [ ] Mark picks the first cloud agent(s) and whether agy is a pin or a fallback

### Files to Modify/Create

**New files:**
- `cmd/ailang/coordinator_cloud_rotatingcred.go` - provider-neutral lease/install/write-back/quarantine, moved out of the codex file (~250 LOC, mostly moved)
- `internal/executor/agy/subscription_token.go` - parse, install, path for the agy token file (~90 LOC)
- `internal/executor/agy/quota.go` - `RunQuota`: bounded `agy -p /quota` under a given HOME, termination sentinel (~60 LOC)
- `internal/executor/agy/subscription_token_test.go` - accepts a real-shaped token, refuses API-route and malformed files, writes 0600 (~120 LOC)
- `docker/Dockerfile.agent-agy` - pinned agy on agent-base (~25 LOC)
- `docker/Dockerfile.agent-agy-go` - adds the Go toolchain (~15 LOC)

**Modified files:**
- `cmd/ailang/coordinator_cloud_codexauth.go` - reduced to the codex spec (~-200 LOC)
- `cmd/ailang/coordinator_cloud.go` - call `installRotatingCredential`; in-job ration check for agy (~30 LOC)
- `internal/executor/agy/agy.go` - bounded process-group join, termination sentinel (~40 LOC)
- `internal/executor/codex/daemon.go` - sentinel moves to `internal/executor` (~5 LOC)
- `internal/coordinator/credential_lane.go` - `agy` lane (~5 LOC)
- `internal/mission/google_quota.go` - export `EvaluateGoogleQuota` (rename only) (~5 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` - variants `agy`, `agy-go` (~6 LOC)
- `internal/config/providers.go` - `AILANG_AGY_AUTH_SECRET` (~8 LOC)
- `tools/agy/install.sh` - Linux pins (~10 LOC)
- `tools/agy/signin.sh` - `--profile cloud`, `--upload` (~40 LOC)
- `../ailang-multivac/terraform/cloud_run_jobs.tf` - two jobs, one secret, IAM (~200 LOC, cloned from codex)
- `../ailang-multivac/cloudbuild-images.yaml` - two image builds (~30 LOC)

### Conflict Surface (shared machinery this overrides)

The rotating-credential refactor touches code the codex cloud lane depends on every day. What must keep working:

| Behaviour | Where it lives today | Guard |
|---|---|---|
| codex lease acquired before the first credential read, held through preflight and execution | `installCodexCredential` → `restore` | existing tests in `cmd/ailang/coordinator_cloud_codexauth_test.go` pass unmodified |
| codex write-back only of changed bytes, after an ownership heartbeat | `persistOwned` | same |
| quarantine on panic and on `ErrProcessTerminationUnconfirmed` | `coordinator_cloud.go` defer, `finalizeCodexCredential` | same, plus a test that the moved sentinel still matches with `errors.Is` |
| dispatcher holds a second codex task while one runs or its retry window is open | `credentialLaneHolder` | existing lane tests; new test that an agy task does not hold the codex lane and vice versa |
| `resolveContainerProvider` refuses an image/provider mismatch | `coordinator_cloud_preflight.go` | new variants declared in `providersForVariant`; `checkVariantProviderAgreement` test row for `agy` |

Intentional change: the shared code's log lines and errors name the provider instead of "Codex".

## Examples

### Example 1: a cloud agy task, healthy

```
execute-job: credential lease acquired (agy-subscription, owner <execution>:<nonce>)
execute-job: agy auth = Google AI subscription (secret ailang-agy-oauth-token)
execute-job: google ration ok — Google AI subscription usage 12.0% within the 15%/day pace line 30.0%   # day 2 of the weekly window
... clone, run agy ...
execute-job: agy credential durably persisted        # only if agy rewrote the token file
execute-job: credential lease released
```

### Example 2: two agy tasks approved together

The dispatcher launches the first. The second stays `pending` (held by `credentialLaneHolder`, as codex tasks are today; the exact log wording is whatever that function emits) and is dispatched on a later tick, instead of launching and failing with "credential already owned" (codex, 2026-10-12/13).

### Example 3: over ration (day 2 of the weekly window, line 30%)

```
execute-job: google ration over — Google weekly usage 31.0% exceeds the 15%/day pace line 30.0%; not cloning
execute-job: agy credential durably persisted        # only if the /quota read refreshed the token
execute-job: credential lease released
```

## Success Criteria

- [ ] Phase 0 rows all measured and recorded in the Verification Log
- [ ] codex cloud tests pass unchanged after the refactor
- [ ] Dev-plane agy task completes; secret version added iff the token changed
- [ ] Rig loops' agy probe passes after ≥3 refreshing cloud runs
- [ ] Concurrent agy dispatch is held, not failed
- [ ] Over-ration job refuses before clone and still writes back a token its `/quota` read refreshed
- [ ] Changelog fragment; `internal/executor/agy/README.md` cloud section; `docs/internal/EXECUTOR_SHAPE.md` notes rotating credentials

## Testing Strategy

- Unit: spec-driven tests of the shared credential code run once per provider (codex, agy) against a fake lease store and fake secret reader/writer: acquire-before-read, unchanged token → no write, changed → one version, write failure → quarantine, termination unconfirmed → quarantine.
- agy executor: mock agy that ignores SIGTERM and forks a grandchild; the executor must return the termination sentinel rather than success.
- Ration path: a fake `/quota` runner that rewrites the token file and reports over-pace → exactly one secret version written, then release; a `/quota` runner that will not die → quarantine, no release. The same pace table the rig's `TestGoogleQuota_DailyPace` uses, run through `EvaluateGoogleQuota`.
- Dispatcher: lane table tests for `agy` alongside `codex`.
- Live: Phase 0 job in dev; Phase 2 smoke in dev. No prod run before both.

## Non-Goals

- **The API-key route in the cloud** (Mark, 2026-11-24). Metered Gemini stays in `gemini_agents`.
- **Using the subscription token from anything but Google's `agy` binary.** The Ultra terms forbid third-party clients on Antigravity OAuth ([[project-google-ai-ultra-assessment]]); pi/opencode never see this secret.
- **A cross-plane quota reservation ledger.** Admission against the account's own reading, with a start margin, as the rig does (Architecture §4).
- **A cross-provider cloud fallback chain.** Which agents get agy, and whether a dry codex bucket re-dispatches to agy, is D6 and the cloud plane's own fallback work.
- **Running agy on the rig differently.** The local executor and mission lane are unchanged.
- **agy `--remote-agent`.** agy 1.5.15 can run conversations on Google-hosted remote agents. That moves execution off our container into Google's sandbox, the managed-agents shape, and would need its own patch round trip. Not pursued here.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Google rotates the refresh token on refresh (P3) and a write-back is lost | High if it rotates: the next job fails sign-in. One sample says it does not (V7), which would make a lost write-back cost only a stale access token | Write-back after every task, quarantine on any write failure (codex design); `make agy-signin --profile cloud` is the recovery |
| Two holders on one account interfere (P4) | High: the rig's loops lose agy | Separate sign-ins (D1); Phase 0 measures coexistence before any rollout |
| agy hangs non-interactively on Linux (#318 class, P1) | Med | Executor's hard deadline and group kill already exist; P1 decides whether the lane is viable at all |
| `--sandbox` unavailable in a Cloud Run job (P5) | Med | D4: container as boundary, with the deny list kept |
| Cloud usage starves the mission fallbacks | Med | One pace line read from the account itself (D3) |
| Ultra terms change for automated use | Med | Recheck antigravity.google terms at Phase 2; the lane is one terraform variable away from off |

## Verification Log

| # | Claim | Evidence | Status |
|---|---|---|---|
| V1 | Cloud claude runs on a subscription token from Secret Manager, no refresh token | memory `reference-claude-credentials-three-holders`; ailang `07c7599bb` | Confirmed |
| V2 | Cloud codex runs on a ChatGPT `auth.json` from secret `ailang-codex-auth-json`, leased and written back | `cmd/ailang/coordinator_cloud_codexauth.go` (`installCodexCredential`, `restore`, `persistOwned`, `finish`); `../ailang-multivac/terraform/cloud_run_jobs.tf` lines 509–807 (both codex jobs reference `codex_auth_json`) | Confirmed (read 2026-11-24) |
| V3 | The lease is provider-specific today: `credentialLane` returns a lane only for `codex` | `internal/coordinator/credential_lane.go:54-58` | Confirmed |
| V4 | No cloud image or job installs agy | `grep -ril 'agy\|antigravity'` over `docker/`, `../ailang-multivac/terraform/` and `../ailang-multivac/cloudbuild*.yaml`: the only hits are bytes inside `.terraform/` provider binaries | Confirmed |
| V5 | A pinned Linux build of 1.5.15 exists for amd64 and arm64 | HEAD `…/1.5.15-5514532022517760/linux-x64/cli_linux_x64.tar.gz` → 200, `linux-arm/cli_linux_arm64.tar.gz` → 200. The manifest itself now serves 1.5.16 (SHA512 published only for latest, so 1.5.15 Linux hashes are computed at pin time) | Confirmed |
| V6 | agy rewrites its token file when it refreshes the access token | token file born 2026-11-23 15:08, last written 2026-11-24 08:53 with a one-hour `expires_in`; keys `token.{access_token,refresh_token,expiry,expires_in,token_type}`, `auth_method`, `id_token_email` (values not read) | Confirmed |
| V7 | Whether the refresh token itself changes on refresh | 2026-11-24: refresh-token SHA-256 prefix `0faa64cf0c75` at 09:30; the file was rewritten by a refresh at 09:52:15 (new one-hour expiry); prefix afterwards `0faa64cf0c75`. **Not rotated across one refresh.** Second sample: rewritten again at 10:47:49, refresh token still the same. P3 repeats it in the cloud HOME | Confirmed: not rotated (twice on the rig, once in the cloud) |
| V8 | agy has no long-lived non-refreshing token mode like `claude setup-token` | `agy --help` (1.5.15) lists no token-minting option; auth is the interactive sign-in or a `GEMINI_API_KEY` | Confirmed for the CLI; docs re-read in Phase 0 |
| V9 | The agy executor does not confirm process termination | `internal/executor/agy/agy.go` calls `proctree.Kill(cmd)` (lines 250–304), which discards `KillGroup`'s error (`internal/proctree/proctree.go:40-44`); codex returns `ErrProcessTerminationUnconfirmed` (`internal/executor/codex/process_lifecycle.go:32`) | Confirmed |
| V10 | `agy -p /quota` reads the account's windows with no model call | m-agy-executor P4; `internal/mission/google_quota.go` | Confirmed (macOS); Linux = P7 |
| V11 | The rig has no container runtime, so Phase 0 runs in the dev project | `command -v docker podman colima orb container`: none | Confirmed |
| V13 | The quota evaluator is not callable from `cmd/ailang` with an explicit HOME | `internal/mission/google_quota.go`: `evaluateGoogleQuota` unexported (line 130); `ObserveGoogleQuota(paths, now)` derives the HOME from `paths.Home` (line 107) | Confirmed |
| V14 | Pace semantics the cloud check reuses | `internal/mission/google_quota.go` `evaluateGoogleQuota` (lines 130–211): allowance = `pace × days` since `reset − 7d`, capped 100; over if exhausted, used > allowance, or headroom < `StartMargin("google")` (2, `internal/mission/quota_margin.go`) unless unused; five-hour window allowance 100. Pinned by `TestGoogleQuota_DailyPace` (day 2: 20% ok, 29% over on margin, 31% over) | Confirmed |
| V15 | The agent jobs leave the Cloud Run execution environment at its default | `grep -n execution_environment ../ailang-multivac/terraform/cloud_run_jobs.tf`: no hits | Confirmed |
| V12 | Ultra terms allow Google's own CLI headless, forbid third-party clients | memory `project-google-ai-ultra-assessment` (terms clause 6, fetched 2026-11-23) | Confirmed 2026-11-23; recheck at Phase 2 |

## Quorum

Triggers fired: #1 (design-freeze items D1–D3), #2 (overrides the codex credential machinery), #4 (premises about agy and Google OAuth we do not control).

**Round 0 (2026-11-24, $0.22): blocked 3/3.** All three reviewers found the same real defect: the over-ration path released the lease after an agy `/quota` process had run, skipping write-back and termination proof. One also found the pace line stated inconsistently. Both fixed (Architecture §4, V14).

**Round 1 ($0.17): blocked 3/3**, on three new and correct objections, all closed in text after the round:
- Phase 0 needed sign-in tooling scheduled for Phase 1 → moved into Phase 0 (P0a–P0c).
- The quota subprocess had no located owner of its lifecycle → `agy.RunQuota` in the executor package spawns, `mission.EvaluateGoogleQuota` only evaluates.
- Rig and cloud can both pass the pace check at once → stated as admission-not-reservation with the overshoot bound, and a cross-plane ledger listed as a non-goal; P2 gained an in-container sign-in fallback.

The skill's re-quorum-once guardrail is spent, so these round-1 fixes are **unreviewed**. Mark ratifies D1–D3 against them, or asks for a fresh quorum.

## Related Documents

- [M-AGY-EXECUTOR](m-agy-executor.md) — the local executor; this is its deferred "Pillar 2", subscription route only
- [M-CODEX-SUBSCRIPTION-LANE, operational addendum](v0_52_6/m-codex-subscription-lane.md#operational-addendum-three-oauth-profiles-on-one-account-2026-10-09) — the three-profile contract and cloud lease this doc generalises
- [M-CODEX-OAUTH-PROFILES sprint](../implemented/v0_53_1/m-codex-oauth-profiles-sprint-plan.md) — how the codex version was built and rolled out
