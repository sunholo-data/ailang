# M-PI-HARNESS-UPGRADE — move the fleet off abandoned pi 0.73.1, and make the boundary visible in the data

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1 — no outage today, but every day the split widens the un-annotated stretch of the benchmark record
**Estimated**: ~3 days (2 days build, 1 day re-baseline + boundary bookkeeping)
**Dependencies**: none blocking. Parent decision: `design_docs/planned/m-resident-agent-instances.md` D10, which pinned 0.73.1 and explicitly deferred this migration.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Both planes run one pinned harness version; today `npm install -g <abandoned pkg>` is a floating install reproducible only by the accident that upstream stopped publishing. The score is **contingent on D8**: the same Dockerfile installs Node via an unpinned `setup_22.x`, and pinning pi while leaving Node floating would be a half-measure, so both land together |
| A2: Replayability | +1 | Fixtures re-recorded at the pinned version, and the version is banked on every row, so a replay knows which harness produced it |
| A3: Effect Legibility | 0 | No AILANG effect surface touched |
| A4: Explicit Authority | 0 | No new ambient authority — and this is now a *tested* zero. An earlier draft proposed seeding container trust for the workspace root; that was withdrawn as a real authority expansion (D7), and a negative test now asserts that workspace-supplied extensions do **not** execute |
| A5: Bounded Verification | 0 | No type-checking surface touched |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | `rawStopReason` and `usage.reasoning` give the error categoriser two machine-readable signals it does not have today |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Reasoning tokens become measurable on the pi lane for the first time; today they are silently folded into `output` and `reason_tokens` is always 0 |
| A10: Composability | 0 | Executor interface unchanged |
| A11: Structured Failure | +1 | Replaces silent event-skipping with a counted, banked, fail-loud contract |
| A12: System Boundary | +1 | Makes the pi CLI an explicitly versioned, asserted boundary instead of an implicit one |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — the change *removes* an unpinned install
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): optimises for machine-readable provenance, not human convenience

---

## Problem Statement

**Current State (all measured 2026-09-03 — see Verification Log):**

1. **The fleet runs a package that upstream abandoned four months ago.** `@mariozechner/pi-coding-agent` last published **0.73.1 on 2026-05-07** and carries an npm deprecation notice: *"please use @earendil-works/pi-coding-agent instead going forward"*. Development continued as `@earendil-works/pi-coding-agent` — **43 releases**, latest **0.84.4** (2026-08-28).

2. **Cloud and rig are on different pi majors, and have been since 2026-08-31.** Every prod image (`agent-pi`, `agent-pi-go`, `agent-eval`, `agent-eval-go`, `resident-pi`) contains the literal layer `RUN npm install -g @mariozechner/pi-coding-agent` → 0.73.1. The rig's `pi` on `PATH` is `@earendil-works/pi-coding-agent@0.84.4`, installed 2026-08-31 10:47. `internal/executor/pi/pi.go` execs whatever `pi` is on `PATH` ([pi.go:57](../../../internal/executor/pi/pi.go#L57)), so **the local lane and the cloud lane have run different harnesses for three days**. That is the exact shape of the comparison this program exists to make ([the local-agentic vs cloud-frontier thesis](../m-ailang-native-harness.md)), and it is currently confounded.

3. **The boundary is invisible in the data.** `RunMetrics` records `prompt_version` but **no executor/harness version field at all** ([metrics.go:17-95](../../../internal/eval_harness/metrics.go#L17)). `PiExecutor.HealthCheck` runs `pi --version` and *discards the output* ([pi.go:528](../../../internal/executor/pi/pi.go#L528)). So nothing in the banked corpus (28,139 result files as of v0.31.0, per [metrics.go:29](../../../internal/eval_harness/metrics.go#L29)) says which pi produced them, and nothing would say it after this upgrade either. This is the same failure that produced the three un-annotated ollama boundaries; the fix is cheap and belongs in this sprint.

4. **The parser degrades silently under drift.** `parsePiEvent` skips lines it cannot parse and the event switch ignores unknown `type`s ([pi.go:242-343](../../../internal/executor/pi/pi.go#L242)). A schema change therefore shows up as *quietly missing metrics*, not as an error — which is precisely what CLAUDE.md's "no silent fallbacks" rule forbids.

**Impact:**

- Any cross-plane pi comparison drawn since 2026-08-31 is comparing two harness versions.
- The pi lane cannot report reasoning tokens (`reason_tokens` is always 0 there), so `IsReasoningStall` ([ai_agent.go:115](../../../internal/eval_harness/ai_agent.go#L115)) can never fire on a pi run, and pi-lane cost carries the same defect that invalidated the v0.30.0 baseline.
- We are one hypothetical `0.73.2` publish away from the whole fleet moving silently. herdr, in the same directory, is pinned by version **and** sha256 for exactly this reason ([docker/resident/Dockerfile:19-25](../../../docker/resident/Dockerfile#L19)).

**What is NOT the problem:** 0.73.1 is not broken, and there is no live incident. D10 already pinned it, so the images are reproducible as of commit `08da6ceea`. This doc is about closing the split deliberately, not about firefighting.

---

## Measured schema drift (0.73.1 → 0.84.4)

Not from the changelog — from **two live captures of the same prompt**, same model (`ollama/glm-5.3-flash:cloud`), same flags, one per version (V7). The 0.73.1 build was installed into a scratch dir; the rig's global install was not disturbed.

| Event / field | 0.73.1 | 0.84.4 | Does `pi.go` read it? | Consequence |
|---|---|---|---|---|
| `session.{type,id,cwd,timestamp,version}` | present | **unchanged** | yes (`SessionID`) | none |
| `turn_start`, `agent_start` | `{type}` | **unchanged** | yes | none |
| `message_update.message` | present (full cumulative message) | **REMOVED** | no | none — but it is why banked NDJSON grew quadratically |
| `message_update.assistantMessageEvent.partial` | present on every delta | **REMOVED** | no | none directly; see below |
| `message_update.assistantMessageEvent` (`text_delta.delta`) | present | **unchanged** | **yes** — the transcript is built from these | safe |
| `message_update.usage` (top level) | absent | **NEW**, cumulative-per-message | no | opportunity: live token tally without waiting for `message_end` |
| `assistantMessageEvent.toolcall_start` | `{contentIndex, partial, type}` | `{contentIndex, **id**, **toolName**, type}` | no | tool identity now available at first delta |
| `thinking_start/delta/end` | not emitted in the 0.73.1 capture | emitted | no | **uncertain** — may be content-dependent rather than a schema change; treat as "observed", not "new" |
| `message_end.message.usage` (assistant) | present | **unchanged, still per-message (not cumulative)** | **yes** — summed per assistant message | safe: the existing sum stays correct (V10) |
| `message_end.message.stopReason` | present | **unchanged** | **yes** | safe |
| `message_end/turn_end.message.rawStopReason` | absent | **NEW** (`"tool_calls"`, `"stop"` — the provider's own value) | no | opportunity for error categorisation |
| `usage.reasoning` | **absent** | **NEW**, and documented as *a subset of `output`* | no | must be wired **and subtracted** (see D5) |
| `message_start.message.{responseId,responseModel}` | absent | NEW | no | none |
| `agent_end` | `{type, messages}` | `{type, messages, **willRetry**}` | yes (no-op branch) | a run ending `willRetry: true` is **not final** |
| `agent_settled` | absent | **NEW terminal-ish event** | no (skipped) | the real "done" signal now |
| `auto_retry_start` / `auto_retry_end` | absent | **NEW** | no (skipped) | pi now retries internally: wall-clock and cost inflate with no event we record |
| `tool_execution_start/update/end` | `{toolCallId,toolName,args,result,isError}` | **unchanged** | yes | safe |
| StopReason vocabulary | `stop`, `toolUse` observed | + `pending` (on `message_start`), `deferred` in the type union | partially | `normalizePiFinishReason` passes unknowns through verbatim — fail-visible, not silent. Acceptable |
| Wire size, `message_update` | avg **1043 B**, max 1222 B | avg **274 B**, max 365 B | — | on a 20-token reply. The 0.73.1 form grows with the message; this is the quadratic replay that `scripts/mission_pi_run.sh` was written to filter around (its header records the ~28 GB implied budget and the size ceiling that followed) |

**Headline: the fields `pi.go` actually reads all survive.** The upgrade is not a rewrite. The danger is the opposite of the one D10 feared — not that the parser breaks loudly, but that it keeps working while three *new* signals go unread and one class of event (`auto_retry_*`) changes what a run's duration and cost mean without any of our instruments noticing.

### Non-wire drift

- **CLI flags** (V16): every flag `buildPiArgs` emits (`--mode json`, `--model`, `--no-session`, `-p`, `--thinking`, `--tools`, `--no-tools`) still exists in 0.84.4. `--thinking` gains a level, `max`, that `validPiThinkingLevels` ([pi.go:616](../../../internal/executor/pi/pi.go#L616)) rejects — fail-loud, but wrong.
- **Extension API** (V14): all four symbols the suite imports (`ExtensionAPI`, `BashOperations`, `createBashTool`, `getAgentDir`) are still exported from the package root, and all eight `pi.*` methods and all seven subscribed event names still exist. The suite in `cmd/ailang/pi_assets/` **already imports its types from `@earendil-works`** — it is written against 0.84.x and running on 0.73.1 today.
- **One real runtime break** (V15): `tools/pi-extensions/sandbox/index.ts:55` imports *values* (not types) from `@mariozechner/pi-coding-agent`. Type imports erase; value imports do not. This file must be renamed or it fails to resolve.
- **Node floor** (V17): pi ≥0.75.0 requires Node ≥22.19.0. `docker/Dockerfile.agent-base:29` installs nodesource `setup_22.x` — satisfied today, unpinned tomorrow.
- **≥0.84 trust gate** (V25–V27, measured, not inherited from the runbook): in an isolated HOME with no `trust.json`, a sentinel extension in `<workspace>/.pi/extensions/` **did not execute**, `rc=0`, and **stderr was completely empty** — the silent-disable is real. Adding `{"<workspace>": true}` to `~/.pi/agent/trust.json` made it execute. **But a sentinel in `~/.pi/agent/extensions/` executed in both runs, with no trust decision at all** — and that global dir is exactly where `ailang pi install` materializes the suite ([pi_setup.go:116](../../../cmd/ailang/pi_setup.go#L116)). So the fleet images need no trust file; see D7.

---

## Goals

**Primary Goal:** Put both planes on one pinned, non-abandoned pi (`@earendil-works/pi-coding-agent@0.84.4`) without a single benchmark comparison silently spanning the boundary.

**Success Metrics:**

1. Every prod and dev image's *pushed config* (read back from the registry, not inferred from a green build) shows `@earendil-works/pi-coding-agent@0.84.4`.
2. Every agent-mode banked row carries `executor_version` — and a row without one is distinguishable from a row that had none to report.
3. The pi parser reports a non-zero unknown-event count in the banked row instead of skipping silently; a run whose assistant `message_end` carries no usage fails loudly rather than banking a zero cost.
4. `reason_tokens > 0` appears on at least one pi-lane row with a reasoning model, and `output_tokens + reason_tokens` equals pi's `usage.output` exactly (no double count).
5. The boundary date is recorded in the charter, in `docs/internal/harness-upgrade-runbook.md`, and in memory, before the first post-upgrade eval is banked.
6. `make check-pi-wire-budget` PASSes, and the extension-execution probe (real `tool_execution_*` events, not a `pi list` and not a substring match) passes **inside a container**, not just on the rig.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1** — Cut over both planes to 0.84.4, rather than pinning cloud at 0.73.1 indefinitely or dual-running | Determines whether the program's central local-vs-cloud comparison is single-harness. Reverting later means re-baselining twice | human | design | high |
| **D2** — Historical rows are **annotated, never re-banked**; a fixed comparator set is re-run post-upgrade to size the shift | Any other choice either destroys history or spends a full baseline. Follows the v0.30.0 precedent | human | design | high |
| **D3** — Add `executor_version` (+ package identity) to `RunMetrics`, `omitempty`, absent ⇒ *unmeasured*, never *"the current one"* | Banked-data schema. Get the absent-vs-zero semantics wrong and every historical row silently claims a version it never ran | human | design | high |
| **D4** — Strictness policy: which drift is **fatal** (abort the run) vs **recorded** (banked counter) | Too strict and a harmless new event type kills a nightly; too loose and we rebuild the silent-skip we are removing | agent, within the rule below | design | med |
| **D5** — `usage.reasoning` is a **subset of `output`**, so bank `output_tokens = usage.output − usage.reasoning` and `reason_tokens = usage.reasoning` | `executor.Result.ReasonTokens` is contractually disjoint from `OutputTokens` ([executor.go:205-214](../../../internal/executor/executor.go#L205)). Wire it naively and every post-upgrade pi row double-counts reasoning in cost | agent | compile | med |
| **D6** — Record a genuine 0.73.1 fixture pair for the overlap window (the existing pair is 0.70.2, V21), delete it once the images cut over | Deleting early leaves the still-live 0.73.1 cloud path untested; keeping them forever contradicts the testing policy | agent | compile | low |
| **D8** — Pin and assert the image's Node version against pi's ≥22.19.0 floor, in this sprint | A1 is scored +1 for removing an unpinned install; leaving a second unpinned install in the same file that can silently break pi at runtime undercuts both the score and the goal | human | design | med |
| **D7** — **Do not** grant container trust to the workspace root. Rely on the image-owned global extension dir, and add a negative test that workspace extensions stay inert | The workspace in a fleet job is a cloned repo — potentially from an untrusted branch. Trusting it would let repo-supplied TypeScript execute inside every container beside `bash`. Measured (V25/V27): the global dir already loads with no trust decision, so the expansion buys nothing | human | design | high |

**D4's rule (so the agent has a bright line):** a *missing field that a banked metric depends on* is fatal (loud error, run marked failed with a distinct `error_category`). An *unrecognised event type* is recorded — counted, named, and banked in `ProviderData` — never fatal. Rationale: an unknown event is upstream adding a feature; a missing `usage` on an assistant `message_end` is our cost record being wrong, and a wrong number is worse than no number.

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — cut over both planes to 0.84.4 (vs. staying pinned at 0.73.1)
- [ ] **D2** — annotate-and-re-baseline-a-comparator-set, not a full re-bank and not a history rewrite
- [ ] **D3** — `executor_version` lands in the banked schema this sprint, with absent ⇒ unmeasured
- [ ] **D7** — containers do **not** trust the workspace root; the suite ships in the image-owned global dir and a negative test proves workspace extensions stay inert
- [ ] **D8** — the image's Node version is **pinned and asserted** at build time against pi's 22.19.0 floor; a floating `setup_22.x` is not acceptable in a sprint whose whole thesis is that unpinned installs are the defect

---

## Solution Design

### Overview

Five milestones, ordered so that **the instrument lands before the change it is meant to measure**. M1 and M2 are useful even if D1 is rejected; M3–M5 are the cutover.

### Architecture

The pi CLI is an external, independently-versioned system boundary with **four in-repo consumers** (V22):

| Consumer | Reads | Affected? |
|---|---|---|
| `internal/executor/pi/pi.go` | full NDJSON → `executor.Result` | **yes — primary** |
| `scripts/mission_pi_run.sh` | **substring** match on `"type":"message_update"`; reads no sub-field of it. Greps `"type":"agent_end"` and `"type":"tool_execution_end"` | yes — but not the way it looks: the filter keeps working, the **snapshot** it feeds silently loses its value (V29) |
| `docker/resident/lib/pi.mjs` | `message_update.assistantMessageEvent.text_delta.delta`, `tool_execution_start.{toolCallId,toolName}`, `message_end.message.{usage,stopReason}` | **no — run against 0.84.4 in this session** (V28), not cross-referenced |
| `cmd/ailang/pi_assets/ail-fmt-autolint.ts` | `tool_execution_end` via the extension API, not the wire | no |

**Components:**

- **Version assertion** — a compiled-in expected `(package, version)` for pi, checked at executor construction against `pi --version`, whose output is currently thrown away.
- **Strict-ish event parser** — same switch, plus an unknown-type counter and a fail-loud path for missing usage.
- **Banked provenance** — `executor_version` on `RunMetrics`, populated from the assertion.
- **Image pin** — `@earendil-works/pi-coding-agent@0.84.4` with a build-time assertion that the installed version equals the pin.

### Implementation Plan

**M1: Bank the harness version** (~3 hours)
1. Add `ExecutorVersion string` to `executor.Result` and `executor_version string` (`omitempty`) to `RunMetrics`; document absent ⇒ unmeasured in the same voice as `CacheAccounted`.
2. `PiExecutor` captures `pi --version` output (already invoked, currently discarded) and stamps it on every `Result`.
3. Same for the other executors where the CLI reports a version — one line each; do not defer, this is the whole point.
4. Test: a run banked with no version reports absent, not `""`-as-current.

**M2: Fail loud on the drift that matters** (~5 hours)
1. Expected-version assertion in `PiExecutor.HealthCheck` + the executor's construction path; mismatch is an error naming both versions, not a warning. Fix the install hint — it still names the abandoned package.
2. Count unrecognised `ev.Type` values; bank `{name: count}` under `ProviderData["pi_unknown_events"]`.
3. Fatal path per D4: an assistant `message_end` with no `usage` fails the run with a distinct error category rather than contributing zero cost.
4. Recognise `agent_settled` and `auto_retry_start`/`auto_retry_end` explicitly — count retries into `ProviderData` so an inflated duration has a visible cause. **The retry loop is bounded** (V31): `maxRetries` defaults to **3**, is enforced in `agent-session.js:431`/`:2285`, and every `auto_retry_start` carries `maxAttempts` on the wire — so record `maxAttempts` alongside the count, and treat a run where `attempt` reaches `maxAttempts` as a distinct, named outcome rather than a slow success. Wall-clock is independently bounded by the executor's own hard/TTFT/idle timer triple ([pi.go:159-172](../../../internal/executor/pi/pi.go#L159)), which is unchanged by this upgrade.
5. Add `max` to `validPiThinkingLevels`.
6. **`scripts/mission_pi_run.sh`'s forensic snapshot** (V29): the script keeps the newest `message_update` in a snapshot file so "a run killed mid-turn still has full forensics." At 0.73.1 that record was the full cumulative message; at 0.84.4 it is a ~274-byte delta, so the guarantee quietly becomes false. Either keep a rolling window of the last N deltas, or correct the comment — but do not leave a stated guarantee that the wire no longer supports. Its `message_update` **filter** and its `agent_end`/`tool_execution_end` greps are unaffected (substring matches on fields that survive).
7. Fix the dead `"Write"`/`"Edit"` comparison at [pi.go:293](../../../internal/executor/pi/pi.go#L293): pi's builtin tools are lowercase (`read`, `bash`, `edit`, `write`) in **both** versions (V11), so `first_attempt_ms` on the pi lane has always fallen through to first-text. Pre-existing, in the blast radius, one line.

**M3: Wire the new signals** (~3 hours)
1. `usage.reasoning` → `Result.ReasonTokens`, with `OutputTokens = output − reasoning` (D5). Assert the identity in a test.
2. `rawStopReason` → `ProviderData`, and consult it in `normalizePiFinishReason` only when `stopReason` is unrecognised.
3. Fixtures. **The current pair is pi 0.70.2** (V21) — there is no 0.73.1 pair to "keep", and an earlier draft of this doc said otherwise. Record **both** afresh: a 0.73.1 pair (the version the cloud actually runs during the overlap) under `testdata/v0_73_1/`, and a 0.84.4 pair under `testdata/v0_84_4/`. Both captures already exist from the V7 differential and can seed them. Delete the 0.70.2 pair once the 0.73.1 pair replaces it — it pins a version no plane has run for months (D6).

**M4: Move the images** (~4 hours)
1. `docker/Dockerfile.agent-pi` and `Dockerfile.agent-eval`: `PI_PACKAGE=@earendil-works/pi-coding-agent`, `PI_VERSION=0.84.4`, plus the `npm uninstall -g @mariozechner/pi-coding-agent || true` that `docker/resident/Dockerfile:51` already does — and for the reason it documents: both packages publish a `pi` bin and npm refuses to clobber another package's link (`EEXIST`), which is how the resident's first attempt failed.
1b. **`resident-pi` needs no Dockerfile change but is in scope for verification** (V30). Its Dockerfile already pins `@earendil-works@0.84.4` and asserts on the **capability** (`--session-id`), not the version string; prod simply has not rebuilt since. Two consequences to check, not assume: once `agent-pi` installs earendil, the resident's `npm uninstall @mariozechner` becomes a no-op and its install becomes a same-package reinstall — that must not reintroduce `EEXIST`; and its capability assertion must still pass on the new base. Both prod and dev `resident-pi` appear in the read-back criteria below.
2. **Assert the pin in the build**: `pi --version | grep -qx "$PI_VERSION"` — a build that installs the wrong version must fail, not warn. The resident's pin failed on `npm error EEXIST` and shipped the old pi under a green build on 2026-09-02 — written down in [cloudbuild-dev.yaml:232-240](../../../cloudbuild-dev.yaml#L232) because it already misled someone; the same trap is live here.
3. **Node is the second unpinned install in the same Dockerfile, and this sprint closes it too** (D8). `setup_22.x` resolves to whatever 22.x is current at build time, so a fresh build can in principle drop below pi's 22.19.0 floor and break pi at *runtime* under a green build — the exact failure shape this doc treats as a measured risk. Deliverable, concrete: pin the nodesource package to an explicit 22.x, and assert it at build time so a wrong version fails the build rather than shipping:
   ```dockerfile
   RUN node -e 'const [a,b]=process.versions.node.split(".").map(Number); if(a<22||(a===22&&b<19)) { console.error(`node ${process.versions.node} < 22.19.0 (pi floor)`); process.exit(1); }'
   ```
4. **Do NOT ship a container trust file** (D7). `RUN ailang pi install` already writes the suite to `~/.pi/agent/extensions/`, which V25/V27 prove loads headless with no trust decision. Add a **negative** container test: a sentinel extension planted at `<workspace>/.pi/extensions/` must NOT execute, alongside the positive `quota_report` probe. Without the negative test, "extensions work" and "the workspace can run code in our containers" look identical.
5. `tools/pi-extensions/sandbox/index.ts`: rename the value import to `@earendil-works/pi-coding-agent`.
6. **Verify by reading the pushed image config**, not by a green build: fetch the amd64 manifest, then the config blob (`curl -sL` — the blob endpoint 302s), and read `history[].created_by`. Method in the Verification Log.

**M5: Boundary bookkeeping and re-baseline** (~6 hours)
1. Record the boundary in the charter, `docs/internal/harness-upgrade-runbook.md`, memory, and the eval-data caveats — **before** banking a post-upgrade run.
2. Run the fixed comparator set on both sides of the boundary; report with `ailang eval-paired`, not aggregate pass rates.
3. Annotate the pre-boundary rows. Do not re-bank them (D2).

### Files to Modify/Create

**New files:**
- `internal/executor/pi/testdata/v0_84_4/fizzbuzz.ndjson` — re-recorded fixture (~40 lines)
- `internal/executor/pi/testdata/v0_84_4/tool_use.ndjson` — re-recorded fixture (~40 lines)
- `internal/executor/pi/testdata/v0_84_4/reasoning.ndjson` — fixture with non-zero `usage.reasoning`, for the disjointness assertion (~40 lines)

**Modified files:**
- `internal/executor/pi/pi.go` — version capture + assertion, unknown-event counter, fatal-missing-usage, reasoning subtraction, `rawStopReason`, `max` level, lowercase tool names (~+120/−15)
- `internal/executor/executor.go` — `Result.ExecutorVersion` (~+8)
- `internal/eval_harness/metrics.go` — `executor_version` with absent ⇒ unmeasured (~+12)
- `internal/executor/pi/pi_test.go` — dual-version replay, reasoning-disjointness assertion (~+90)
- `docker/Dockerfile.agent-pi` — package + version + uninstall + assertion (~+10/−4)
- `docker/Dockerfile.agent-eval` — same (~+8/−3)
- `docker/Dockerfile.agent-base` — pin the nodesource Node version and add the build-time floor assertion above; today line 29 is `setup_22.x`, unpinned (~+6/−1)
- `tools/pi-extensions/sandbox/index.ts` — value import rename (~+2/−2)
- `docs/internal/harness-upgrade-runbook.md` — fleet section + the registry-config verification method (~+30)
- `scripts/mission_pi_run.sh` — snapshot window or corrected guarantee (V29); note the filter's upstream cause is fixed (~+15/−4)

---

## Examples

### Example 1: a harness boundary that the data can see

**Before** — the banked row cannot answer "which pi ran this?":
```json
{ "model": "...", "executor": "pi", "prompt_version": "v0.3.0-hints",
  "output_tokens": 1840, "reason_tokens": 0, "cost_usd": 0.021 }
```

**After:**
```json
{ "model": "...", "executor": "pi", "executor_version": "@earendil-works/pi-coding-agent@0.84.4",
  "prompt_version": "v0.3.0-hints",
  "output_tokens": 1204, "reason_tokens": 636, "cost_usd": 0.021,
  "provider_data": { "pi_unknown_events": {"queue_update": 2}, "pi_auto_retries": 0 } }
```
`output_tokens` drops not because the model wrote less, but because 636 reasoning tokens were always in there and are now named. `cost_usd` is unchanged — reasoning bills at the output rate. This is exactly the shift that D5's subtraction rule exists to keep honest, and exactly why D2 forbids pooling across the boundary.

### Example 2: drift that used to be silent

**Before** — upstream renames `tool_execution_end` and the run still "succeeds":
```
tool calls recorded: 0     (the switch matched nothing; every line was skipped)
result: success, 0 tools, transcript intact
```

**After:**
```
ERROR pi: assistant message_end carried no usage (harness drift?)
      expected @earendil-works/pi-coding-agent@0.84.4, running @earendil-works/pi-coding-agent@0.85.0
      unknown events seen: tool_result_end×7
```

---

## Conflict Surface

Not strictly required (no `internal/parser|types|codegen` files), written anyway — the rule has earned it twice.

1. **What position does this extend?** The pi NDJSON event stream, and the banked agent-mode row schema.
2. **What else lives there?** Three other consumers of the same stream (table above) and every historical banked row that lacks the new field.
3. **How is it disambiguated?** By event `type` (unchanged discriminator) and, for the schema, by `omitempty` + an explicit absent ⇒ unmeasured reading — the `CacheAccounted` precedent, which exists because absent-as-zero once made every historical row look 100% cache-fresh.
4. **What MUST still work post-change** (fixtures verified to exist):
   - `internal/executor/pi/testdata/fizzbuzz.ndjson` replay → same metrics as today
   - `internal/executor/pi/testdata/tool_use.ndjson` replay → same tool counts as today
   - `scripts/test_mission_pi_run.sh` → unchanged (its assertions are substring greps on `agent_end` / `tool_execution_end`, both of which survive — V29)
   - `docker/resident/lib/pi.mjs` against 0.84.4 → already verified live (D10)
   - every pre-boundary banked row still parses, with `executor_version` absent
5. **What deliberately changes?**
   - A pi run whose assistant `message_end` carries no usage stops being a silent zero-cost success and becomes a loud failure. Any consumer relying on that silence is relying on a bug.
   - `mission_pi_run.sh`'s snapshot stops being a full-message forensic record (V29). This is upstream's change, not ours; the deliverable is that the script stops *claiming* otherwise.
6. **Two consumers examined and found unaffected, with the reason stated** (rather than asserted):
   - `cmd/ailang/pi_assets/ail-fmt-autolint.ts` reads `tool_execution_end` through the **extension API**, not the wire. All seven event names the suite subscribes to still exist at 0.84.4 (V14), so the wire changes cannot reach it.
   - `docker/resident/lib/pi.mjs` reads four fields, all verified present in a live 0.84.4 run through the module itself (V28).
7. **Out of scope, recorded so it is not mistaken for drift:** `pi.mjs` sets `state.usage` from the *last* `message_end` rather than summing across turns, so a multi-turn resident run under-reports tokens. That behaviour is identical at 0.73.1 — a pre-existing resident bug, not something this upgrade introduces or should fix.

---

## Success Criteria

- [ ] Both prod and dev `agent-pi`, `agent-pi-go`, `agent-eval`, `agent-eval-go` **and `resident-pi`** image configs read back from Artifact Registry show `@earendil-works/pi-coding-agent@0.84.4` — the goal is *no* prod image left on the abandoned package, and V3 shows `resident-pi` is currently one of them
- [ ] A container-run extension probe emits real `tool_execution_start`/`_end` events naming `quota_report` (not `pi list`, not a substring match)
- [ ] **Negative:** a sentinel extension planted at `<workspace>/.pi/extensions/` does **not** execute in that same container, and no `trust.json` exists in the image (D7)
- [ ] A container built from the pinned Dockerfile reports `node --version` ≥ 22.19.0, and a deliberately wrong pin **fails the build** rather than shipping (D8)
- [ ] `make check-pi-wire-budget` PASSes post-cutover
- [ ] `executor_version` present on every new agent-mode row; absent on replayed historical rows and read as unmeasured
- [ ] `output_tokens + reason_tokens == usage.output` asserted in a test with a reasoning model fixture
- [ ] Boundary date recorded in charter + runbook + memory before the first post-upgrade bank
- [ ] Comparator set re-run and reported via `ailang eval-paired`
- [ ] All tests passing; documentation updated (CHANGELOG, runbook, `docs/LIMITATIONS.md` if the comparability caveat lands there)

## Testing Strategy

**Unit tests:**
- Dual-version fixture replay (0.73.1 and 0.84.4) producing identical `Result` metrics for the fields both versions carry
- Reasoning disjointness: `OutputTokens + ReasonTokens == usage.output`
- Version-assertion mismatch produces an error naming both versions
- Assistant `message_end` without `usage` → run fails with the distinct category, not a zero
- Unknown event type → counted and banked, run still succeeds

**Integration tests:**
- Live `pi --mode json` capture on the rig, asserted against the fixture shape
- One benchmark end-to-end on the pi lane, `executor_version` present in the banked JSON
- Node floor: build the image and assert the version; separately prove the assertion fires by building with a deliberately low pin
- A synthetic `auto_retry_start` stream reaching `maxAttempts` produces the named outcome, not a slow success

**Manual testing:**
- The runbook's four post-checks, run **inside a built container** as well as on the rig
- Registry config read-back for each of the four images

## Deferred Decisions

- Which comparator benchmarks form the re-baseline set — **agent may choose**, must be a set with pre-boundary rows on both planes, and must be stated in the sprint plan before it runs
- Whether `rawStopReason` also feeds `CategorizeAgentError` — **agent may choose**; banking it is mandatory, consuming it is optional this sprint
- Whether the other executors' version capture lands in the same commit or a follow-up — **agent may choose**; the pi one is not optional
- Exact `error_category` name for the missing-usage failure — **agent may choose**, must not reuse `api_error` (the "cause unknown" catch-all)

## Non-Goals

- **Chasing pi latest.** We pin 0.84.4, the version the rig has been running and the extension suite is written against. Later versions are a future runbook-driven upgrade.
- **Migrating opencode, codex, gemini, or claude.** Same good idea, separate change — the runbook is explicit that upgrades are one tool per session.
- **Re-running the full benchmark ladder.** D2 chooses a comparator set precisely to avoid this.
- **Rewriting the parser to be schema-generic.** The four fields we read are stable across eleven minors; a generic parser would trade a measured risk for an unmeasured one.
- **Rewriting history.** Pre-boundary rows are annotated, never re-banked.
- **Enabling project-local `.pi/extensions` inside fleet containers.** Measured as unnecessary (V27) and, for a workspace that is a cloned repo, an authority expansion (D7). If a future design needs it, it needs its own authority model, not a line in a migration doc.

## Timeline

**Day 1** (~8 hours): M1 + M2 — version banking and fail-loud parser. Independently valuable; ships even if D1 stalls.
**Day 2** (~7 hours): M3 + M4 — new signals, fixtures, image pins, registry read-back verification.
**Day 3** (~6 hours): M5 — boundary bookkeeping, comparator re-baseline, paired report.

**Total: ~21 hours across 3 days**

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A green Cloud Build ships an image whose pi pin silently failed | **measured, happened 2026-09-02** | high | Build-time `pi --version` assertion + read the pushed config back from the registry. A green build says nothing |
| Post-upgrade pi-lane cost/tokens shift and get read as a model change | high | high | D5 subtraction + D2 annotation + `executor_version` on every row |
| The ≥0.84 trust gate silently disables project-local extensions in containers | **confirmed (V25)** | low | Not applicable to us: the suite ships in the image-owned global dir, which loads unconditionally (V27). Verified by the positive probe |
| Someone "fixes" the trust gate later by trusting the workspace, handing repo-supplied code execution inside every container | med | **high** | D7 is a human design-freeze item, and the negative sentinel test fails loudly if the trust ever gets granted |
| The mid-turn forensic snapshot silently degrades from a full message to a 274-byte delta | **confirmed (V29)** | med | M2.7 either keeps a rolling delta window or corrects the script's stated guarantee — it must not keep claiming "full forensics" |
| `auto_retry_*` inflates duration/cost with no visible cause | med | med | Count retries into `ProviderData` (M2.4) |
| Upstream publishes 0.85.x mid-sprint | med | low | Version is pinned and asserted; a mismatch is loud |
| The 0.73.1 comparison arm disappears before the cloud images cut over | med | med | D6 keeps the old fixtures through the overlap window |
| `thinking_*` difference is content-dependent, not schema | — | low | Recorded as *observed*, not *changed*; re-check with a reasoning model before relying on it |

---

## Verification Log

Every load-bearing claim, with the command that proved it. All run 2026-09-03.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | `@mariozechner/pi-coding-agent` is abandoned at 0.73.1 | npm registry JSON | last publish 2026-05-07; `deprecated: "please use @earendil-works/pi-coding-agent instead going forward"` | 
| V2 | Upstream continues as `@earendil-works` | npm registry JSON | 43 versions, latest 0.84.4 (2026-08-28) |
| V3 | All prod images run the unpinned old package | AR manifest → config blob → `history[].created_by`, project `ailang-multivac` | `agent-pi`, `agent-pi-go`, `agent-eval`, `agent-eval-go`, `resident-pi`, all built 2026-09-02T19:08Z, all `npm install -g @mariozechner/pi-coding-agent` |
| V4 | Dev plane matches; dev `resident-pi` alone carries the new pin | same, project `ailang-multivac-dev` | `agent-pi` 06:27Z unpinned; `resident-pi` 06:46Z has `PI_PACKAGE=@earendil-works PI_VERSION=0.84.4` over a 0.73.1 base |
| V5 | The rig moved on 2026-08-31 | `ls -la /opt/homebrew/lib/node_modules` | `@earendil-works` dir mtime 2026-08-31 10:47; `@mariozechner` dir present but empty |
| V6 | `pi.go` execs whatever is on PATH | read | [pi.go:57](../../../internal/executor/pi/pi.go#L57), :76, :124 |
| V7 | Wire-schema drift table | two live captures, same prompt/model/flags, 0.73.1 installed to a scratch dir | see table above |
| V8 | `usage.reasoning` is new in 0.84.4 | usage key sets from both captures | 0.73.1 lacks it; 0.84.4 has it |
| V9 | `rawStopReason` is new | same | 0.84.4 only: `"stop"`, `"tool_calls"` |
| V10 | `message_end` usage is per-message, not cumulative | usage values across the 0.84.4 capture | turn 1 = 248/15, turn 2 = 272/23 — existing summation stays correct |
| V11 | The `"Write"`/`"Edit"` check is dead code | tool names in both captures + `dist/core/tools/*.js` | `bash`; builtins are `read`/`bash`/`edit`/`write`, lowercase |
| V12 | **Negative existence:** no harness/executor version on banked rows | `grep -n "version" internal/eval_harness/metrics.go` | only `prompt_version` |
| V13 | `ReasonTokens` must be disjoint from `OutputTokens`, and pi's is not | [executor.go:205-214](../../../internal/executor/executor.go#L205) + pi-ai `types.d.ts:272-277` | contract says disjoint; pi documents reasoning as *a subset of output* → subtract |
| V14 | The extension API surface survives | `dist/index.d.ts` + `dist/core/extensions/types.d.ts` | all 4 symbols exported; all 8 `pi.*` methods and all 7 subscribed events present |
| V15 | One real runtime break | `grep "^import" tools/pi-extensions/sandbox/index.ts` | line 55 imports *values* from `@mariozechner`; `cmd/ailang/pi_assets/*.ts` already import types from `@earendil-works` |
| V16 | Every CLI flag we pass still exists | `pi --help` at 0.84.4 | all present; `--thinking` gains `max`, absent from `validPiThinkingLevels` |
| V17 | Node floor | CHANGELOG 0.75.0 + `Dockerfile.agent-base:29` | pi needs ≥22.19.0; image installs nodesource `setup_22.x`, unpinned |
| V18 | The wire-budget gate passes at 0.84.4 | `make check-pi-wire-budget` | PASS (declared 32000, actual 32000) |
| V19 | Session-id parsing survives | `dist/core/session-manager.d.ts:5-12` + capture | `SessionHeader {type:"session", id, …}` unchanged |
| V20 | `HealthCheck` discards the version | [pi.go:520-533](../../../internal/executor/pi/pi.go#L520) | runs `pi --version`, keeps only the exit code; error text still names the abandoned package |
| V21 | Fixtures are older than either version | `pi_test.go:645` | recorded at pi **0.70.2** |
| V22 | Four in-repo consumers of the stream | repo-wide grep for `assistantMessageEvent|tool_execution_end|message_end` | `pi.go`, `docker/resident/lib/pi.mjs`, `scripts/mission_pi_run.sh` (+ its test), `ail-fmt-autolint.ts` (API, not wire) |
| V23 | **Negative existence:** no existing doc covers the fleet upgrade | `create_planned_doc.sh` dual search | all neural matches < 0.45; D10 explicitly defers it |
| V24 | This repo's Dockerfiles are the control for the fleet images | `ailang-multivac/cloudbuild-images.yaml:41-51, 200-213` | images build from `git clone --depth 1 --branch dev <ailang repo>` |
| V25 | **The ≥0.84 trust gate really does silently disable project-local extensions** | isolated `HOME`, sentinel extension at `<ws>/.pi/extensions/` writing a marker file on load, `pi --mode json --no-session --no-tools -p` run inside `<ws>` with **no** `trust.json` | marker **absent**; `rc=0`; **stderr empty**; no mention of trust or extensions anywhere in the output |
| V26 | Seeding `trust.json` is what lifts the gate | same setup + `~/.pi/agent/trust.json` = `{"<canonical ws path>": true}`, re-run | marker **present** — the project extension executed. Confirms the mitigation works, and therefore that it *is* an authority grant |
| V27 | **The global extension dir needs no trust decision** | second sentinel at `~/.pi/agent/extensions/` in the same two runs | executed in **both** runs, with and without `trust.json`. `ailang pi install` writes exactly here ([pi_setup.go:116](../../../cmd/ailang/pi_setup.go#L116)) → the containers need no trust file (D7) |
| V28 | **The resident's own parser handles 0.84.4** — measured here, not cross-referenced to D10 | `import("docker/resident/lib/pi.mjs").runPi({model:"ollama/glm-5.3-flash:cloud", tools:["bash"], …})` against the real 0.84.4 binary | `exitCode=0`, 26 events, `text="DONE"`, `toolCalls=[{id:"call_luo68no8",name:"bash"}]`, `stopReason="stop"`, `usage` populated **including `reasoning`**. All four fields it reads survived |
| V29 | What `mission_pi_run.sh` actually does with `message_update` — and the guarantee that breaks | read [scripts/mission_pi_run.sh:31-38, 155-163, 233-240](../../../scripts/mission_pi_run.sh) | It **substring-matches** `"type":"message_update"` in awk and routes those lines to a snapshot file; it parses **no sub-field**, so the REMOVED `message`/`partial` cannot break it. Its `agent_end` and `tool_execution_end` greps also survive. **But** its stated purpose — "a run killed mid-turn still has full forensics" — depended on that snapshot being the full cumulative message (avg 1043 B at 0.73.1); at 0.84.4 it is a lone delta (avg 274 B, V7). The filter survives; the guarantee does not |
| V30 | `resident-pi` needs no Dockerfile change — only a rebuild | read [docker/resident/Dockerfile:41-56](../../../docker/resident/Dockerfile#L41) | already `PI_PACKAGE=@earendil-works PI_VERSION=0.84.4`, with the old package uninstalled first (both publish a `pi` bin; npm refuses to clobber another package's link — the documented `EEXIST` cause) and the result asserted on the **capability** (`--help | grep -- --session-id`), not the version string. Prod (V3) simply predates the commit |
| V31 | **pi's internal auto-retry is bounded** | `dist/core/settings-manager.js:595`, `dist/core/agent-session.js:431, 2285, 2294` | `maxRetries` defaults to **3**; the loop stops at `_retryAttempt >= settings.maxRetries`; `maxAttempts` is emitted on every `auto_retry_start`. Independently, wall-clock is bounded by the executor's own hard/TTFT/idle triple ([pi.go:159-172](../../../internal/executor/pi/pi.go#L159)) — 30 s TTFT, 3 min idle, hard timeout from config |
| V32 | Node is genuinely unpinned today | read [docker/Dockerfile.agent-base:29](../../../docker/Dockerfile.agent-base#L29) | `curl -fsSL https://deb.nodesource.com/setup_22.x \| bash -` — resolves to whatever 22.x is current at build time. Satisfies pi's 22.19.0 floor today; nothing enforces that it will tomorrow (D8) |

**Registry read-back method** (for M4.6, because it is not obvious and the blob endpoint redirects):
```bash
TOK=$(gcloud auth print-access-token)
# 1. index → amd64 manifest digest;  2. manifest → config digest;  3. config blob (needs -L)
curl -sL -H "Authorization: Bearer $TOK" \
  "https://europe-west1-docker.pkg.dev/v2/<project>/ailang/<image>/blobs/<config-digest>" \
  | python3 -c "import json,sys;[print(h.get('created_by','')) for h in json.load(sys.stdin)['history']]"
```

---

## Quorum

**Triggers fired: 3 of 4** — (1) design-freeze items exist (D1–D3, D7); (3) it touches cost/KPI semantics and the banked-data schema; (4) load-bearing premises concern an external system we do not control.

**Round 0 (2026-09-03, `gpt5-6-sol` + `gemini-3-1-pro` + `oc-glm-5-2`): BLOCKED, 3/3 reject.** All three objections were accepted and are now closed by measurement, not by argument:

| Reviewer | Objection | Resolution |
|---|---|---|
| gpt5-6-sol | M4.4 seeded container trust for the workspace root — an authority expansion — while A4 was scored 0 and called "unchanged" | **Withdrawn entirely.** V27 shows the global dir loads with no trust decision, so the grant bought nothing. Now D7 (human freeze item), a negative sentinel test, an honest A4 justification, and a Non-Goal |
| gemini-3-1-pro | Neither the silent-disable failure mode nor the `trust.json` mitigation was verified | **V25 / V26** — ran both arms in an isolated `HOME`. Silent-disable confirmed (marker absent, rc=0, stderr empty); mitigation confirmed |
| oc-glm-5-2 | `pi.mjs` "already verified (D10)" was a cross-reference, not evidence; `mission_pi_run.sh` was hand-waved despite two REMOVED fields | **V28** — ran the resident module itself against 0.84.4. **V29** — read the script: it substring-matches and parses no sub-field, so it survives; but its *forensic-snapshot guarantee* does not, which is now a work item (M2.6) |

**Round 1 (2026-09-03, same three reviewers): BLOCKED, 3/3 reject — on three entirely new grounds, all correct, all now closed:**

| Reviewer | Objection | Resolution |
|---|---|---|
| gpt5-6-sol | Scope contradiction: V3 proves prod `resident-pi` is on the abandoned package, but M4 touched only two Dockerfiles and the success criteria omitted it — leaving a verified prod consumer on 0.73.1 after a "fleet-wide" migration | **V30** — `resident-pi`'s Dockerfile already carries the pin; prod merely predates the commit. Added M4.1b (with the two follow-on checks a rebuild-only path still needs: no re-introduced `EEXIST`, capability assertion still passing) and put `resident-pi` in the read-back criteria |
| gemini-3-1-pro | (a) The new upstream auto-retry loop was recorded but never shown to be **bounded**. (b) M3.3/D6 said "keep the 0.73.1 fixtures" while V21 proves the existing fixtures are **0.70.2** — an internal contradiction | **V31** — `maxRetries` defaults to 3, enforced in two places, `maxAttempts` on the wire, and wall-clock independently bounded by the executor's own timer triple. **Fixture contradiction was real and is fixed**: record a genuine 0.73.1 pair (the V7 capture seeds it) and retire the 0.70.2 pair |
| oc-glm-5-2 | The Node floor was acknowledged as unpinned but was in no freeze item, no success criterion, and no concrete mechanism — while A1 was scored +1 for *removing* an unpinned install, leaving a second one in the same file | **V32** + **D8** — Node pin and a build-time floor assertion are now a mandatory deliverable with the exact check written out, a success criterion including a negative build, and the A1 justification explicitly made contingent on D8 |

**Two rounds, six distinct objections, six accepted.** Round 0's were premise gaps; round 1's were scope and internal-consistency gaps that only became visible once the premises were solid. Per the re-quorum-once guardrail this doc now goes to a human rather than grinding a third round: the remaining risk is not an unverified premise but a judgement call about scope (D1, D2, D7, D8), which is the human's to make. A third round is available on request and would cost ~$0.11.

## Related Documents

**Implemented (may inform design):**
- [m-exec-pi-harness](../../implemented/v0_14_2/m-exec-pi-harness.md) — where the pi executor and its NDJSON parser came from
- [m-dx-pi-harness-sprint-plan2](../../implemented/v0_35_0/m-dx-pi-harness-sprint-plan2.md) — the embedded extension suite's distribution model

**Planned (check for overlap):**
- [m-resident-agent-instances](../m-resident-agent-instances.md) — **parent decision (D10)**. It pinned 0.73.1 and deferred this migration; this doc executes it. The resident moved alone because it *needs* `--session-id`; the job executor did not
- [m-dx-session-protocol-gate](m-dx-session-protocol-gate.md) — the session gate rides on the extension API this upgrade moves
- [m-eval-validity-discipline](../m-eval-validity-discipline.md) — the boundary-annotation discipline D2 follows

## References

- `docs/internal/harness-upgrade-runbook.md` — the rig procedure, the four post-checks, and the ≥0.84 trust gate
- `docs/docs/guides/debugging.md` — the 0.73.1 quadratic-replay note that 0.84.0 fixes upstream
- pi CHANGELOG 0.84.0 "Breaking Changes" — the `message_update` change, upstream's own account
- CLAUDE.md §2 — no silent fallbacks (the rule M2 enforces)

## Future Work

- Pin every executor CLI by version **and** digest, the way herdr already is
- A weekly CI check that the pushed image's pi matches the Dockerfile pin — the 2026-09-02 failure would have been caught in a day
- Consume `rawStopReason` in `CategorizeAgentError` to shrink the `api_error` catch-all

---

**Document created**: 2026-09-03
**Last updated**: 2026-09-03
