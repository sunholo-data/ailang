# Sprint Plan: M-CODEX-SUBSCRIPTION-LANE

Refs #903 (related deadline umbrella: #1259)

**Design:** [m-codex-subscription-lane.md](m-codex-subscription-lane.md)
**Status:** Phase 1 (M1–M2) implemented and validated, 2026-10-09. This run stops after the local commit; D8 and Phase 2 remain separate runs. See [Phase 1 report](m-codex-subscription-lane-phase1-report.md).
**Duration:** 6 engineering days (Phase 1: 2; Phase 2: 4) plus a half-day D8 measurement step, excluding release waiting, the D8 ruling and external ops work.

## Execution is split into three separate runs

This sprint is **not** executed end to end in one run.

| Run | Milestones | Starts when | Ends with |
|-----|-----------|-------------|-----------|
| **Phase 1** | M1, M2 | this plan merges | executor commits locally (messages carry `Refs #903`), reports, then **STOPs explicitly**. It does not start D8 or any Phase 2 milestone. |
| **D8 step** | D8 (measurement only, no code) | Phase 1 is done (may run before Phase 1 merges) | a written measurement, then **STOP / BLOCKED** for the maintainer's ruling. The executor does not pick an arm. |
| **Phase 2** | M3–M6 | Phase 1 has **merged** on `dev` **and** the D8 ruling is recorded in the design doc | Phase 2 work committed locally, `Refs #903`. |

A Phase 2 run that finds Phase 1 unmerged or the D8 ruling absent stops immediately and reports BLOCKED. No `models.yml` migration (M5) and no `internal/ai/chatgpt/` removal (M6) happen before that ruling.
**Targets:** v0.52.6 first; v0.53.0 second.
**Risk:** low for Phase 1; high for Phase 2 migration and shared credentials.

## Scope and evidence

The 2026-10-08 maintainer comment on [#903](https://github.com/sunholo-data/ailang/issues/903#issuecomment-6066595893) rules option 3: immediate fail-loud routing plus deadline, followed by CLI-backed replacement. Issue body and ruling were read through the GitHub API on 2026-10-08. Do not open a duplicate issue; all commits and the PR body use `Refs #903`, never an auto-close keyword while only Phase 1 is complete.

Current source still guesses codex as OpenAI and constructs the chatgpt HTTP client without a deadline. The existing design already contains systemic caller analysis. The stale document is already marked superseded in the approved design commit; only verify that banner. Direct factory call sites currently found: cmd/ailang/ai_handlers.go (registry and direct), internal/eval_harness/ai_provider.go. Mission quorum uses the executor provider independently. Re-audit before implementation, including aliases and apiserver hosting.

Velocity analysis script (7 days) found only the design commit 838e3b1f in this shallow checkout; no measured implementation LOC/day is available. Older changelog entries printed by the script are not recent velocity. Planning capacity is an explicit estimate: 1,200 changed LOC including tests/docs over 6 days, 200 LOC/day. The extra Phase 1 day buffers shared preflight and handler regression work beyond the design's one-day estimate. LOC estimates describe work, including moves/deletions, rather than net repository growth.

## Registry reuse audit


Searches: codex (no packages), oauth, ai. Inspected sunholo/external_backend@0.2.0 and sunholo/oauth@0.1.0 with pkg info/docs. These are AILANG libraries: external_backend handles single JSON stdout, not NDJSON AI providers; oauth implements installed-app grants, not sanctioned Codex CLI auth. Reuse the existing Go executor, parser, auth bootstrap and cost taxonomy; no registry dependency fits this host-level work.

Every milestone records action `none` for registry packages, with existing Go components reused where applicable. No new OAuth protocol or NDJSON parser is planned.

## Milestones

### M1: Reject codex guesses and verify caller separation (~140 LOC)

**Dependencies:** None
**Duration:** 1 day(s).
**Files to create/update:** `internal/ai/config.go`, `internal/ai/provider_test.go`, `internal/ai/factory/factory.go`, `internal/ai/factory/factory_test.go`, `cmd/ailang/ai_handlers_test.go`, `internal/eval_harness/ai_provider.go`, `internal/executor/motoko/provider_preflight.go`, `internal/executor/motoko/provider_preflight_test.go`.

**Acceptance criteria:**

- [x] Re-audit GuessProvider, ProviderFromString, EnvVarForProvider, factory.New and envpolicy callers before edits; record mission spawn-recipe pins reaching the codex executor rather than the AI effect.
- [x] ProviderCodex sentinel covers codex* guesses and string parsing; no API-key variable is associated with it; direct and registry-fallback construction pin the actionable #903 error pointing to chatgpt/.
- [x] Motoko preflight checks subscription credentials and classifies the sentinel as subscription; mission executor routing and explicit provider: openai registry rows retain their behavior.
- [x] Existing provider-prefix fixtures and coordinator spawn-recipe regression tests pass.

**Examples:** Phase 1 uses routing and hung-stream Go fixtures plus README migration examples; no new language behavior.

### M2: Bound interim client and prepare Phase 1 release (~140 LOC)

**Dependencies:** M1
**Duration:** 1 day(s).
**Files to create/update:** `internal/ai/chatgpt/client.go`, `internal/ai/chatgpt/client_test.go`, `internal/executor/codex/README.md`, `design_docs/planned/m-codex-billing-lane-resolution.md`, a changelog fragment `changelogs/unreleased/YYYY-MM-DD-codex-fail-loud-and-chatgpt-timeout.md` (never `changelogs/v0.32-current.md`).

**Acceptance criteria:**

- [x] Default HTTP deadline is 10 minutes, WithTimeout overrides it, and a continuously streaming httptest response terminates at the configured deadline; earlier context cancellation also works.
- [x] No parallel timeout flag or env variable is introduced; #1259 absorption is documented.
- [x] README describes subscription-first auth and the explicitly metered alternative; preserve the already-landed supersession banner rather than duplicating it.
- [x] A1–A5 and the focused checks below pass; Phase 1 can ship independently before any Phase 2 code is merged.
- [x] **Phase 1 STOP.** The executor commits M1–M2 locally with `Refs #903`, reports the Phase 1 result, and stops. It does not begin D8 or M3 in the same run.

**Examples:** Phase 1 uses routing and hung-stream Go fixtures plus README migration examples; no new language behavior.

### D8: Measure the motoko-chatgpt row's tool use, then STOP for a ruling (~60 LOC, write-up only)

**Dependencies:** M2 (separate run; may happen before Phase 1 merges)
**Duration:** 0.5 day.
**Files to create/update:** `design_docs/planned/v0_52_6/m-codex-subscription-lane-d8-measurement.md` (new). No code, no `models.yml` change.

**Data source:** the per-run eval result JSON for model `motoko-chatgpt-gpt-6-1-sol` (under `eval_results/`, including any baselines) — the `agent_tool_calls` count and the `agent_transcript` field, whose `tool_call: <tool> <args>` lines are written by the motoko parser from the session's `native_tool_calls` events (M-MOTOKO-OBS-TRANSCRIPT). Where the row was used in a mission, add the chain records from `ailang chains` whose agent_id names the row. Record which files/chains were read, their dates and the run count; if no runs exist, say so — that is itself the measurement.

**Acceptance criteria:**

- [ ] For every run found: tool-call count, which tools, and whether the run used the harness tool loop (`std/ai` `Step` with tools) or was effectively single-turn.
- [ ] The write-up states the two options from the design (a: delegate the loop to codex as one `Generate`; b: move the tool-loop arm to a sanctioned metered lane) and what the measurement implies for each, without choosing.
- [ ] **STOP / BLOCKED** for the maintainer's D8 ruling. The executor records the sprint as blocked on D8 and ends the run.

### M3: Adapt the existing executor to the AI provider contract (~350 LOC)

**Dependencies:** M2, D8 ruling (Phase 2 run; starts only after Phase 1 has merged and the ruling is recorded)
**Duration:** 2 day(s).
**Files to create/update:** `internal/executor/codex/aieffect/provider.go`, `internal/executor/codex/aieffect/provider_test.go`, `internal/mission/quorum/agentic_provider.go`.

**Acceptance criteria:**

- [ ] Generate uses the existing codex executor and its NDJSON parser with bounded time/turns, cancellation, and the read-only question tool set; no coordinator import from the adapter.
- [ ] Strip codex: from pinned models; bare codex names reach the CLI unchanged and fail loudly there; prompt, final text, token/cache usage, finish reason and failures map correctly.
- [ ] Step with harness tools returns CodeToolsNotSupported; tool-free Step delegates to Generate; images return CodeCapabilityNotSupported; no streaming interface is claimed.
- [ ] Fake-executor fixtures cover success, failure, malformed or missing output, cancellation, model normalization and unsupported requests; no subprocess starts for rejected capabilities.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

### M4: Enforce subscription auth, serialization and cost provenance (~240 LOC)

**Dependencies:** M3
**Duration:** 1 day(s).
**Files to create/update:** `internal/executor/codex/aieffect/auth.go`, `internal/executor/codex/aieffect/auth_test.go`, `internal/executor/codex/aieffect/provider.go`, `internal/executor/cost.go`, `internal/executor/cost_test.go`, `internal/executor/motoko/provider_preflight.go`, `internal/dispatch/cloudrun/dispatcher.go`.

**Acceptance criteria:**

- [ ] Read-only credential validation moves to the platform layer, honors CODEX_HOME, and refuses missing, expired or API-key credentials with CodeAuthFailed; CLI owns refresh.
- [ ] A process-wide mutex keyed by canonical auth path serializes validation plus execution; canceled queued calls exit without spawning and locks release after failure.
- [ ] Subprocess environment excludes metered-key authority; subscription classification yields CostListPriceEquivalent with real tokens, and absent rates yield unknown provenance rather than fabricated spend.
- [ ] Audit the codex/oauth job selection. In-process locking alone is explicitly insufficient for multiple processes or workers sharing one credential; record the fleet serialization (or isolated-credential) work as a **follow-up in the sprint report**. This is not a pass condition for M4.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

### M5: Wire hosts and resolve motoko migration before replacement (~160 LOC)

**Dependencies:** M4
**Duration:** 0.5 day(s).
**Files to create/update:** `internal/ai/factory/factory.go`, `internal/ai/factory/factory_test.go`, `cmd/ailang/ai_handlers.go`, `internal/eval_harness/ai_provider.go`, `internal/modelreg/models.yml`.

**Acceptance criteria:**

- [ ] Inject a codex constructor without a core-to-platform import; both handler construction paths and eval harness wire it; unwired builds return a named registration error.
- [ ] Repeat the factory.New grep including aliases and indirect callers; audit apiserver effect hosting and wire every supported host or document a deliberate fail-loud build.
- [ ] Apply the **recorded** D8 ruling (from the D8 step) to `models.yml`; if the ruling is not recorded in the design doc, do not touch `models.yml` — STOP and report BLOCKED. Harness tool-loop rows use whichever arm the ruling picked (sanctioned metered lane or delegate-to-codex restructure).
- [ ] Preserve historical registry identity as appropriate and verify changed row lookup, auth-lane classification and external references; migration has no silent reroute.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

### M6: Remove direct backend and validate Phase 2 release (~170 LOC)

**Dependencies:** M5
**Duration:** 0.5 day(s).
**Files to create/update:** `internal/ai/chatgpt/client.go`, `internal/ai/chatgpt/auth.go`, `internal/ai/factory/factory.go`, `internal/observatory/mission_rollup.go`, `docs/docs/guides/ai-routing.md`, `docs/docs/guides/mission-model-fleet.md`, `examples/ai/codex_generate.ail`, `examples/ai/README.md`, a changelog fragment `changelogs/unreleased/YYYY-MM-DD-codex-subscription-lane.md` (never `changelogs/v0.32-current.md`).

**Acceptance criteria:**

- [ ] Only after the D8 ruling is recorded and M5 has applied it: remove direct-backend client and obsolete credential implementation after references migrate; retain ProviderChatGPT as a loud deprecation error naming codex: and retain historical CanonicalQuotaBucket folding.
- [ ] No production path calls chatgpt.com/backend-api/codex; chatgpt/ deprecation, tool refusal and history rollup regressions pass.
- [ ] Read ailang prompt before writing the example; ailang check validates examples/ai/codex_generate.ail; document required auth and single-turn contract plus both release migration notes.
- [ ] Write the live-probe procedure for the maintainer (AI-effect codex probe with OPENAI_API_KEY absent: subscription auth, real token counts, CostListPriceEquivalent, refreshed quota observations; one run of the migrated motoko row). The probe is **maintainer-run evidence**, attached later; it is not executor acceptance and its absence does not fail M6.
- [ ] Focused tests, formatting, lint and architecture gates pass; fleet serialization is a report follow-up before shared-credential rollout, not an M6 pass condition.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

## Daily execution and gates

Phase 1 run:
- Day 1: M1 caller audit, routing and preflight tests, then implementation.
- Day 2: M2 streamed deadline tests, auth README, Phase 1 checks. Commit locally with `Refs #903` and **STOP**.

D8 run (separate):
- Half day: D8 measurement write-up, then **STOP / BLOCKED** for the ruling.

Phase 2 run (separate; only after Phase 1 merged and D8 ruled):
- Day 3: M3 executor seam, request mapping, unsupported tools/images and failures.
- Day 4: Complete M3 bounds/cancellation.
- Day 5: M4 credential validation, queued cancellation, auth-path serialization and cost accounting.
- Day 6: M5 host wiring and the ruled migration; M6 removal, examples, maintainer probe procedure and checks.

Design approval supplied in the handoff ratifies D3, D6 and D7. D8 was ruled *deferred* on 2026-10-08: take the measurement first and bring it back; the executor stops there and does not choose. No migration choice is inferred from elapsed time. Ops work is external to this checkout; record it as a report follow-up without claiming that a local mutex enforces fleet serialization.

## Verification and success metrics

Use targeted Go package tests after each milestone (ai/factory/chatgpt, motoko, coordinator routing, codex/aieffect, executor cost, modelreg and observatory as touched). For changed packages, inspect coverage of the new behavior: every acceptance branch needs a meaningful regression; a repository percentage is not a substitute for deadline, cancellation, serialization or billing tests.

At each phase boundary run focused `go test` on the touched packages (Phase 1: `./internal/ai/... ./internal/executor/motoko/... ./internal/eval_harness/... ./cmd/ailang/...` with `-run` filters for the handler tests; Phase 2 adds `./internal/executor/... ./internal/modelreg/... ./internal/observatory/...`) plus `make test-core`, `make fmt`, `make lint`, `make check-boundaries`, and `make check-architecture-closure`. Do **not** run the full `make test` locally: in the executor's RAM-backed /tmp it has crashed with SIGBUS. The full suite runs in CI on the PR. Confirm target availability in this checkout before execution and use the repository's documented equivalent if a target was renamed. Phase 2 also checks the example; the credentialed probe is maintainer-run evidence (see M6). Record actual command results without credentials.

Remaining risks: #1259 may land first (use its unified timeout mechanism if available); CLI schema drift (reuse executor parser); host injection omissions (loud unregistered error); different paths to the same credential (canonicalize lock keys); multi-process/fleet concurrency (ops evidence); tool-loop migration (D8 gate). Streaming, write-tool exposure, MCP bridge and generic timeout CLI changes remain outside this sprint.

## Coordinator handoff

Artifacts below are ready for review. The sprint JSON passes `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-CODEX-SUBSCRIPTION-LANE` (re-checked at review, 2026-10-09). No implementation tests were run for this planning-only change. Per sprint-planner/resources/coordinator.md, merging the coordinator sprint-plan PR approves the plan and triggers sprint-executor **for Phase 1 only** (M1–M2), which then stops. D8 and Phase 2 are later, separate runs (see the table at the top).

Executor rules:
- Re-run `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-CODEX-SUBSCRIPTION-LANE` before starting each run and after every sprint JSON update.
- The executor cannot push or merge; it commits locally and the coordinator raises the PR. Commit messages and the PR body carry `Refs #903`.
- Every changelog step writes a fragment `changelogs/unreleased/YYYY-MM-DD-<slug>.md`; never `changelogs/v0.32-current.md`.
