# Sprint Plan: Codex Provider/Billing Lane Resolution

## Summary

Make `codex:` unambiguously executor-only: guessed AI-effect routing rejects it with one typed, actionable diagnostic, while explicit `models.yml` OpenAI rows continue to reach the metered API deliberately.

**Design:** `design_docs/planned/m-codex-billing-lane-resolution.md`  
**Issue:** #903  
**Duration:** 2 days (about 10 focused hours)  
**Estimated change:** 220 LOC (55 implementation, 135 tests, 30 release/migration documentation)  
**Dependencies:** M-MODEL-REGISTRY-SINGLE-SOURCE (implemented)  
**Risk level:** Medium — small code change on a billing-sensitive routing boundary

## Current Status Analysis

- The repository is at v0.42.0; the design's v0.39.0 target is historical. Land in the next minor release rather than editing an already released changelog.
- `internal/ai/config.go` currently groups `codex*` with `gpt*`, `o1*`, and `o3*`, returning `ProviderOpenAI`. `internal/ai/provider_test.go` locks in the defective behavior with `codex-max -> openai`.
- There are two live fallback consumers of `GuessProvider`: CLI AI-effect construction in `cmd/ailang/ai_handlers.go` and eval adapter construction in `internal/eval_harness/ai_provider.go`. They currently return different untyped errors.
- Explicit registry routing already bypasses guessing. `internal/modelreg/models.yml` contains OpenAI rows including `gpt5-2-codex` and `codex-mini-latest`; these are deliberate metered lanes and must remain valid.
- The migration scan found no repository `.ail` program or model registry key using `codex:` as an AI-effect model. Mission role values and `internal/dashboard_transforms/approval_authority.ail` use executor pins and are out of scope.
- Seven-day commit history contains only the v0.42.0 release, so LOC/day is not a meaningful velocity sample. The estimate uses the design's two-day bound and current call-site/test surface, with roughly 45% contingency for diagnostic propagation and full-suite verification.

## Binding Decisions

1. Add a dedicated typed resolution error in `internal/ai` (for example `*ModelResolutionError`) rather than a fake `ProviderCodex` or a `ProviderError` naming a provider that does not exist. It carries the model and stable executor-only guidance and supports `errors.As` at both consumers.
2. `GuessProvider` remains a pure `ProviderType` lookup and returns `""` for names beginning with `codex`, including `codex:` and bare `codex-*`. A helper converts the empty result into the typed error so CLI and eval paths cannot drift in wording.
3. Registry rows remain authoritative. A model resolved with explicit `provider: openai` never calls the guesser and remains a valid metered API lane.
4. No changes to `internal/executor/codex`, `auth_bootstrap.go`, `mission-control.sh`, credential loading, or `~/.codex/auth.json`.

## Milestones

### M1 — Shared fail-loud routing contract

**Estimated:** 35 implementation + 55 tests = 90 LOC, 0.5 day  
**Files:** `internal/ai/config.go`, `internal/ai/provider_test.go` (or a focused `config_test.go`)

Tasks:

- Write red table tests for `codex:gpt-5-codex`, mixed-case `CODEX:...`, and bare `codex-max`, expecting no guessed provider.
- Add a positive-control table for `gpt-5-codex`, `gpt-*`, `o1-*`, and `o3-*`, which continue to guess OpenAI because they use API-model vocabulary rather than the executor prefix.
- Introduce the typed model-resolution diagnostic and a shared resolver/helper that returns it when no provider can be inferred. Its message must state that `codex:` is an executor-lane pin, ChatGPT subscription OAuth is unavailable to the AI effect, metered OpenAI will not be selected silently, and the remedies are the coordinator/provider-executor lane or an explicit registry row.
- Remove the `strings.HasPrefix(lower, "codex")` guess arm. Do not introduce `ProviderCodex`.

Acceptance criteria:

- `ai.GuessProvider("codex:gpt-5-codex") == ""` and `ai.GuessProvider("codex-max") == ""`.
- `ai.GuessProvider("gpt-5-codex") == ai.ProviderOpenAI` remains true.
- The new resolution helper returns an error discoverable via `errors.As` and preserves the original model string.
- `go test ./internal/ai/... -count=1` passes.

### M2 — Enforce the contract at both fallback consumers

**Estimated:** 20 implementation + 60 tests = 80 LOC, 0.75 day  
**Files:** `cmd/ailang/ai_handlers.go`, `cmd/ailang/ai_handlers_test.go` (new or focused existing test), `internal/eval_harness/ai_provider.go`, `internal/eval_harness/ai_provider_test.go`

Tasks:

- Replace the CLI direct-path generic error with the shared typed resolution error after config-driven `name/model` lookup has had its existing opportunity.
- Replace the eval adapter's `unsupported provider` error with the same typed error.
- Add construction-level tests proving neither path reaches `factory.New`, reads an API key, or installs a handler for `codex:`. Assert error type and the stable billing guidance, not only a substring.
- Add registry precedence tests using an explicit OpenAI model config whose friendly/API name contains `codex`; prove `setupAIHandlerFromConfig` and `newProviderAdapter(..., ProviderOpenAI)` bypass guessing and construct the intended OpenAI lane. Use test credentials/endpoints only; no network call.

Acceptance criteria:

- CLI AI-effect setup for `codex:gpt-5-codex` fails before any provider/client construction with the typed executor-only diagnostic.
- Eval adapter construction for the same input fails with the identical typed diagnostic.
- Explicit `provider: openai` construction for a codex-family API model succeeds without consulting `GuessProvider`.
- Existing config-driven `name/model`, OpenRouter vendor/model, Ollama, Anthropic, Google, and OpenAI routing tests remain green.
- `go test ./cmd/ailang ./internal/eval_harness ./internal/ai/... -count=1` passes.

### M3 — Migration record and verification

**Estimated:** 0 implementation + 20 tests/verification + 30 docs = 50 LOC, 0.75 day  
**Files:** current unreleased changelog selected at execution time; design status/verification notes when landing

Tasks:

- Re-run the repository migration scan for AI-effect `codex:` usages. Classify mission/coordinator role pins as executor-only and therefore unaffected; migrate any newly discovered AI-effect use to an executor task or explicit registry model.
- Add the next-minor-release changelog note: AI-effect `codex:` now fails loudly instead of silently selecting metered OpenAI; use the executor lane for ChatGPT-subscription billing or an explicit `models.yml` OpenAI row for deliberate API billing.
- Run focused tests, `make test`, `make lint`, and `make check-boundaries`. Confirm the diff does not touch executor auth or mission-control billing guards.
- Move/update the design artifact according to the normal implemented-doc release workflow only after all gates pass.

Acceptance criteria:

- Migration scan has no unexplained AI-effect `codex:` usage.
- Changelog names the behavior break and both migration choices.
- `make test`, `make lint`, and `make check-boundaries` pass, or every unrelated pre-existing failure is reproduced against the base commit and recorded.
- `git diff -- internal/executor/codex tools/launchd/mission-control.sh` is empty.

## Execution Order

Day 1: M1 red/green tests and typed contract; begin M2 CLI and eval consumer tests.  
Day 2: finish M2 registry precedence controls; complete M3 migration scan, changelog, and repository gates.

M1 must land before M2. M3 starts only after both construction paths and the registry positive control are green.

## Success Metrics

- No guessed `codex*` name can select `ProviderOpenAI`.
- Both live fallback consumers expose one typed, actionable error contract.
- Explicit registry OpenAI rows for codex-family API models remain functional.
- Zero changes to subscription credential handling or executor routing.
- Issue #903 is referenced by intermediate commits and closed only by the final landing commit (`Fixes #903`).

## Risks and Mitigations

- **Over-broad rejection of legitimate API models:** retain `gpt-*` OpenAI guessing and prove explicit registry precedence for `codex-*` API names.
- **Diagnostic drift between consumers:** centralize construction of the typed error in `internal/ai`; consumer tests compare type and stable fields/message.
- **Accidental credential/network dependency in tests:** test construction only with injected/test provider configuration; no generation call.
- **Config-driven provider regression:** preserve its lookup before emitting the resolution error and retain existing dispatch tests.
- **Release-target drift:** choose the current unreleased changelog at execution time; do not back-edit v0.39.0 or v0.42.0 release records.

## Out of Scope

- A provider that shells out to `codex exec`.
- Reading, refreshing, or translating Codex OAuth credentials.
- Any executor authentication, coordinator spawn, or mission-control guard change.
- Compatibility mapping from `codex:` to metered OpenAI.

## Handoff Gate

This plan is ready for human review. Per repository workflow, implementation begins only after the user explicitly says **execute sprint**, at which point `sprint-executor` owns the JSON progress file and TDD sequence.
