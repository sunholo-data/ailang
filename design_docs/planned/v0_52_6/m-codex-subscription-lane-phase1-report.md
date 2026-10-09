# M-CODEX-SUBSCRIPTION-LANE: Phase 1 implementation

Refs #903; related #1259.

## Result and boundary

M1–M2 implement the approved first release: `codex*` AI-effect guesses resolve to a subscription sentinel and fail at the shared factory with actionable `chatgpt/` guidance. Explicit metered registry rows continue to construct OpenAI clients. The interim ChatGPT client has a 10-minute total HTTP deadline and a `WithTimeout` option; an earlier context deadline still wins, including while SSE heartbeats continue arriving. Motoko preflight validates subscription login and classifies the sentinel as subscription.

This run stops after the Phase 1 local commit. D8 measurement is a separate run. Phase 2 requires Phase 1 merged on dev and a recorded maintainer D8 ruling; no migration or backend removal occurred here.

## Caller audit (2026-10-09)

Read #903 and the 2026-10-08 option-3 ruling through GitHub's issue and comment APIs. Re-audited `GuessProvider`, `ProviderFromString`, `EnvVarForProvider`, factory constructors (including `NewProvider`), and executor environment policy.

- `cmd/ailang/ai_handlers.go`: direct guessing and explicit registry construction both call `factory.New`. New handler regressions pin both refusal paths.
- `internal/eval_harness/ai_provider.go`: fallback inference calls the same factory. New adapter regressions pin refusal with the model name and preserve explicit `ProviderOpenAI` construction.
- `cmd/ailang/exec.go`, `cmd/ailang/coordinator_lifecycle.go`, and `internal/mission/quorum/call.go`: `factory.NewProvider` delegates to the same constructor. Explicit metered providers retain their behavior.
- `internal/executor/envpolicy.go`: `ProviderCredentialVars` derives grants through guessing and env mapping. The sentinel grants no metered key. The Codex executor's own credential grants are independent and unchanged.
- `tools/launchd/resolve-role-spawn.sh` emits `recipe codex:gpt-*`; `internal/mission/dispatch/resolve.go` splits executor pins, and the Codex CLI is selected independently of AI-effect guessing. `internal/coordinator/provider_executor.go` uses the executor factory and imports `internal/executor/codex` for registration; quorum's agentic adapter uses `NewExecutorProvider`. This route never calls the AI provider factory.
- `internal/apiserver` contains no AI provider factory construction; Phase 1 does not add host injection or change its effect hosting.

The stale billing-lane design already has its supersession banner; it was preserved. No `.ail` edits were needed. No new timeout flag or environment variable was introduced; #1259 will absorb the interim constant.

## Validation

- Sprint JSON validator: passed at startup and after progress updates.
- `go test ./internal/ai/... ./internal/executor/motoko/...`: passed with `AILANG_NO_METADATA=1` to isolate this worker's ambient GCP metadata. Initial Google-lane isolation failure was environmental.
- `go test ./internal/ai/chatgpt -run 'TestDeadlineDefaults|TestContinuouslyStreamingDeadline' -count=3`: passed.
- `bash tools/launchd/test_spawn_pin_hook.sh`: 27 passed, 0 failed.
- Coordinator executor registration/provider regressions: passed, including Codex auto-discovery.
- `make test-core`, `make fmt`, `make check-boundaries`, `make check-architecture-closure`, and `git diff --check`: passed.
- CLI handler tests (`TestCodexAIHandlerRefusal` and `TestDeclaredMaxOutputTokens`): passed.
- `go test ./internal/eval_harness/... -skip '^TestMemWatchdogKillsGrandchild$'`: passed; the unrelated container cleanup exception is documented below.
- `make lint`: passed, 0 issues (repository-pinned golangci-lint 2.11.4).

Tooling was restored outside the repository: Go PATH, jq, make, GCC and runtime libraries, ps, uv/Python 3.12, and the repository-pinned golangci-lint 2.11.4. Initial concurrent compiler kills were resolved by lowering compile concurrency and memory limits. Stale temporary build artifacts were removed outside the repository to restore RAM headroom; the final full lint run then passed. The broad harness run exposed the existing `TestMemWatchdogKillsGrandchild` cleanup assertion: the container retains orphan zombies after killing the group; this is independent of the provider change. Its code was left unchanged and its exception is explicit in the final run.

No full `make test` is run locally, per the approved plan; CI owns the full suite. New tests use platform-neutral paths or httptest, without new external binary dependencies.

## Coordinator PR body

Refs #903

Reject `codex*` AI-effect inference instead of silently constructing the metered OpenAI client. Point callers to the interim `chatgpt/` lane, classify Motoko subscription credentials correctly, and bound the ChatGPT HTTP exchange (including streaming bodies) to 10 minutes with an override option. Preserve explicit metered registry entries and the independent Codex CLI executor. This is Phase 1 only; D8 and Phase 2 remain separate gated runs.

## Follow-ups

Run D8 measurement separately, bring the tool-use evidence to the maintainer, and record the ruling before migrating Motoko rows. Phase 2 must audit fleet serialization or isolated credentials before shared-credential rollout; an in-process mutex alone cannot establish that property. Independent sprint evaluation remains the coordinator's next stage.
