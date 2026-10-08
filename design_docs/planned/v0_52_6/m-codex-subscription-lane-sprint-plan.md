# Sprint Plan: M-CODEX-SUBSCRIPTION-LANE

Refs #903 (related deadline umbrella: #1259)

**Design:** [m-codex-subscription-lane.md](m-codex-subscription-lane.md)
**Status:** Ready for sprint-plan review; implementation not started.
**Duration:** 6 engineering days (Phase 1: 2; Phase 2: 4), excluding release waiting, D8 ruling and external ops work.
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

- [ ] Re-audit GuessProvider, ProviderFromString, EnvVarForProvider, factory.New and envpolicy callers before edits; record mission spawn-recipe pins reaching the codex executor rather than the AI effect.
- [ ] ProviderCodex sentinel covers codex* guesses and string parsing; no API-key variable is associated with it; direct and registry-fallback construction pin the actionable #903 error pointing to chatgpt/.
- [ ] Motoko preflight checks subscription credentials and classifies the sentinel as subscription; mission executor routing and explicit provider: openai registry rows retain their behavior.
- [ ] Existing provider-prefix fixtures and coordinator spawn-recipe regression tests pass.

**Examples:** Phase 1 uses routing and hung-stream Go fixtures plus README migration examples; no new language behavior.

### M2: Bound interim client and prepare Phase 1 release (~140 LOC)

**Dependencies:** M1
**Duration:** 1 day(s).
**Files to create/update:** `internal/ai/chatgpt/client.go`, `internal/ai/chatgpt/client_test.go`, `internal/executor/codex/README.md`, `design_docs/planned/m-codex-billing-lane-resolution.md`, `changelogs/`.

**Acceptance criteria:**

- [ ] Default HTTP deadline is 10 minutes, WithTimeout overrides it, and a continuously streaming httptest response terminates at the configured deadline; earlier context cancellation also works.
- [ ] No parallel timeout flag or env variable is introduced; #1259 absorption is documented.
- [ ] README describes subscription-first auth and the explicitly metered alternative; preserve the already-landed supersession banner rather than duplicating it.
- [ ] A1–A5 and required repository checks pass; Phase 1 can ship independently before any Phase 2 code is merged.

**Examples:** Phase 1 uses routing and hung-stream Go fixtures plus README migration examples; no new language behavior.

### M3: Adapt the existing executor to the AI provider contract (~350 LOC)

**Dependencies:** M2
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
- [ ] Audit the codex/oauth job selection and link an ops-repo serialization task or isolated-credential evidence. In-process locking alone is explicitly insufficient for multiple processes or workers sharing one credential.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

### M5: Wire hosts and resolve motoko migration before replacement (~160 LOC)

**Dependencies:** M4
**Duration:** 0.5 day(s).
**Files to create/update:** `internal/ai/factory/factory.go`, `internal/ai/factory/factory_test.go`, `cmd/ailang/ai_handlers.go`, `internal/eval_harness/ai_provider.go`, `internal/modelreg/models.yml`.

**Acceptance criteria:**

- [ ] Inject a codex constructor without a core-to-platform import; both handler construction paths and eval harness wire it; unwired builds return a named registration error.
- [ ] Repeat the factory.New grep including aliases and indirect callers; audit apiserver effect hosting and wire every supported host or document a deliberate fail-loud build.
- [ ] Measure tool usage for every motoko-chatgpt-* row, record the evidence and obtain the D8 human ruling before changing models.yml. Harness tool-loop rows must use a sanctioned metered lane or an approved delegate-to-codex restructure.
- [ ] Preserve historical registry identity as appropriate and verify changed row lookup, auth-lane classification and external references; migration has no silent reroute.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

### M6: Remove direct backend and validate Phase 2 release (~170 LOC)

**Dependencies:** M5
**Duration:** 0.5 day(s).
**Files to create/update:** `internal/ai/chatgpt/client.go`, `internal/ai/chatgpt/auth.go`, `internal/ai/factory/factory.go`, `internal/observatory/mission_rollup.go`, `docs/docs/guides/ai-routing.md`, `docs/docs/guides/mission-model-fleet.md`, `examples/ai/codex_generate.ail`, `examples/ai/README.md`, `changelogs/`.

**Acceptance criteria:**

- [ ] Remove direct-backend client and obsolete credential implementation after references migrate; retain ProviderChatGPT as a loud deprecation error naming codex: and retain historical CanonicalQuotaBucket folding.
- [ ] No production path calls chatgpt.com/backend-api/codex; chatgpt/ deprecation, tool refusal and history rollup regressions pass.
- [ ] Read ailang prompt before writing the example; ailang check validates examples/ai/codex_generate.ail; document required auth and single-turn contract plus both release migration notes.
- [ ] Run a live AI-effect codex probe with OPENAI_API_KEY absent: subscription auth, real token counts, CostListPriceEquivalent and refreshed quota observations are evidenced. Verify one run of the approved migrated motoko row.
- [ ] Required tests, formatting, lint and architecture gates pass; fleet serialization evidence is required before shared-credential rollout.

**Examples:** M3–M5 use fake-executor/auth/host fixtures; M6 creates and verifies `examples/ai/codex_generate.ail` for the public lane.

## Daily execution and gates

- Day 1: M1 caller audit, routing and preflight tests, then implementation.
- Day 2: M2 streamed deadline tests, auth README, Phase 1 checks and release review. Stop Phase 2 merge until Phase 1 ships.
- Day 3: M3 executor seam, request mapping, unsupported tools/images and failures.
- Day 4: Complete M3 bounds/cancellation; collect D8 run evidence and prepare the ops serialization task for parallel human review.
- Day 5: M4 credential validation, queued cancellation, auth-path serialization and cost accounting.
- Day 6: M5 host wiring and approved migration; M6 removal, examples, live probe and release checks. If D8 or fleet evidence is pending, leave Phase 2 incomplete and record the blocker; Phase 1 remains independently shippable.

Design approval supplied in the handoff ratifies D3, D6 and D7. D8 still requires measured usage and a human choice; unchecked design boxes are not evidence of an unapproved design. No migration choice is inferred from elapsed time. Ops work is external to this checkout; record a linked task without claiming that a local mutex enforces fleet serialization.

## Verification and success metrics

Use targeted Go package tests after each milestone (ai/factory/chatgpt, motoko, coordinator routing, codex/aieffect, executor cost, modelreg and observatory as touched). For changed packages, inspect coverage of the new behavior: every acceptance branch needs a meaningful regression; a repository percentage is not a substitute for deadline, cancellation, serialization or billing tests.

At each phase release boundary run `make test`, `make test-core`, `make fmt`, `make lint`, `make check-boundaries`, and `make check-architecture-closure`. Confirm target availability in this checkout before execution and use the repository's documented equivalent if a target was renamed. Phase 2 also checks the example and runs the credentialed probe; record actual command results, token/provenance evidence and quota observation timestamps without credentials. Missing credentials block only the live verification and Phase 2 release, not fixture work or Phase 1.

Remaining risks: #1259 may land first (use its unified timeout mechanism if available); CLI schema drift (reuse executor parser); host injection omissions (loud unregistered error); different paths to the same credential (canonicalize lock keys); multi-process/fleet concurrency (ops evidence); tool-loop migration (D8 gate). Streaming, write-tool exposure, MCP bridge and generic timeout CLI changes remain outside this sprint.

## Coordinator handoff

Artifacts below are ready for review. JSON syntax, milestone IDs, dependencies, LOC totals and per-milestone registry decisions were validated with Python. The repository shell validator could not run because jq is absent (its reported syntax failure is a missing-tool result). No implementation tests were run for this planning-only change. Per sprint-planner/resources/coordinator.md, merging the coordinator sprint-plan PR approves the plan and triggers sprint-executor. This planning task does not start implementation or self-approve that PR. Executor should read both phases and the JSON, preserve the independent Phase 1 release, and report unmet Phase 2 gates. PR body must contain `Refs #903`.
