# M-CODEX-SUBSCRIPTION-LANE Phase 1 report

Refs #903; absorbs the interim ChatGPT HTTP deadline portion of #1259.

## Pre-edit caller re-audit (2026-10-09)

Searched all Go sources for GuessProvider, ProviderFromString, EnvVarForProvider,
factory.New and factory.NewProvider (including aliases); inspected the caller bodies.

| Caller | Routing and Phase 1 consequence |
| --- | --- |
| cmd/ailang/ai_handlers.go | Registry path honors explicit provider; direct/registry-miss path guesses then calls factory.New. Both must refuse the codex sentinel. |
| internal/eval_harness/ai_provider.go and ai_agent.go | Explicit registry provider wins; inferred codex must fail through the shared factory. |
| cmd/ailang/exec.go | NewProvider delegates to New, including the config-driven hook; built-in codex refusal must precede that hook. |
| cmd/ailang/coordinator_lifecycle.go | Explicit anthropic NewProvider call, unaffected. |
| internal/mission/quorum/call.go | Explicit registry provider allowlist (openai/google/ollama), then NewProvider; codex already refused. |
| internal/executor/motoko/provider_preflight.go | Shared guess now returns codex sentinel; check subscription credential and classify subscription instead of demanding OPENAI_API_KEY. |
| internal/executor/envpolicy.go | ProviderCredentialVars uses guess plus env mapping; codex carries no metered key authority. |
| cmd/ailang/eval_cache_warmup.go | Anthropic string parsing check only; unaffected. |
| internal/apiserver | No factory constructor call; NewWithConfig passes the supplied EffCtx into embed.Engine.SetEffContext. Provider construction belongs to the caller; no additional construction seam in Phase 1. |

Mission pins such as codex:gpt-5.6-sol identify an executor plus model, not an
AI-effect provider. Coordinator dispatch_provider.go resolves executor variants;
provider_executor.go registers and constructs the codex executor separately.
Mission quorum's agentic_provider.go calls coordinator.NewExecutorProvider.
The agents fixture, dispatch-provider fixtures and modelreg executor identity
fixtures verify this separation. Re-read modelreg/identity.go and the mission
control gate-3-route.md cross-provider spawn recipe (provider_executor path). Explicit provider: openai registry rows remain metered,
including gpt5-2-codex; no registry rows or mission recipes change in Phase 1.

The pre-existing billing-lane resolution document already has the supersession
banner; preserve it. D8 measurement and the migration ruling are separate runs.

## Implementation

- ProviderCodex represents all codex* guesses and case-insensitive provider
  parsing; EnvVarForProvider grants it no API-key variable. Factory New and
  NewProvider refuse it before config-driven fallback, with a pinned,
  non-retryable CapabilityNotSupported error naming #903 and chatgpt/<model>.
- Both CLI construction paths and the eval adapter inherit the shared refusal.
  Tests also construct the shipped gpt5-2-codex row with explicit openai and
  verify explicit openai overrides a codex-shaped inferred name.
- Motoko preflight checks the existing read-only subscription credential loader
  for codex and chatgpt, refusing missing/API-key logins even with a metered key;
  motokoAuthLane labels both subscription. Existing executor routing stays intact.
- The interim ChatGPT HTTP client defaults to a 10-minute total response
  deadline. WithTimeout overrides it. Continuous SSE output cannot reset that
  deadline; Generate, Step, StreamStep and earlier caller cancellation have
  httptest coverage, with no credentials or external provider requests.
- #1259's interim ChatGPT HTTP deadline is absorbed here. There is no new
  timeout CLI flag or environment variable, and no claim to resolve the rest
  of that umbrella. The supersession banner was verified and left intact.

## Validation evidence

Tests were written before implementation: codex guess, factory fallback,
subscription preflight and eval routing tests failed against the original code.
The initial timeout test build failed because WithTimeout did not exist.
After implementation the regressions pass, including repeated streamed-deadline
checks. New tests require no external binaries and use portable paths.

| Check | Result |
| --- | --- |
| Sprint JSON validator | PASS before execution and after state updates |
| go test ./internal/ai/... ./internal/executor/motoko/... ./internal/eval_harness/... -run 'Codex\|Provider\|ClientTimeout\|ClientStreaming\|GuessProvider' | PASS |
| go test ./cmd/ailang -run 'AIHandlerCodex' | PASS, including the shipped explicit openai registry row |
| Coordinator/modelreg/mission routing fixtures (ResolveDispatchProvider, ProviderForVariant, CloudAgents, ResolveModel, Resolve_AgentModelName, RenderEnv) | PASS |
| executor DefaultProfiles_OnlyTheLaneCredential and PolicyGrantsWorkerCredentials fixtures | PASS |
| ChatGPT ClientTimeout/ClientStreaming tests, count=3 | PASS |
| All internal/ai and motoko package tests | PASS with isolated cloud config |
| make fmt and git diff --check | PASS |
| make check-boundaries and make check-architecture-closure | PASS |
| make check-file-sizes | PASS |
| make test-core | PASS with CGO_ENABLED=1 and temporary compiler |
| make lint | PASS: 0 issues on bounded-memory retry |

The cloud environment initially omitted Go from PATH, make, jq, a C compiler,
ps and uv. Temporary tools outside the repository restored Go/make/jq, lint,
process inspection and a Zig C compiler for CGO SQLite checks. No environment
setup changes were added to the repository. The first factory baseline failed
TestNew_GoogleLanes because local cloud configuration leaked into its fixture;
AILANG_NO_METADATA=1 and an empty AILANG_CONFIG override made that suite pass.

The unfiltered eval-harness package run still fails the pre-existing Python
runner tests because uv is missing and TestMemWatchdogKillsGrandchild because
its process-group teardown sees a member after 10 seconds. The focused provider
suite passes. Initial CGO-disabled core checks failed SQLite brain fixtures;
the final core run passes with CGO_ENABLED=1 and the temporary compiler.
The first lint run was killed with exit 137 while CGO compilation ran in
parallel; the retry passes with 0 issues using GOMAXPROCS=2 and GOMEMLIMIT=1GiB.

The approved plan overrides the generic skill's full-suite checkpoint scripts:
no full make test was run (its documented RAM-backed /tmp SIGBUS risk); focused
checks, core, formatting, lint and architecture gates are used instead.

## Phase boundary and follow-ups

Phase 1 only: M1 and M2. STOP after the local Refs #903 commit; do not push or
merge. D8 measurement is a separate run and requires the maintainer's migration
ruling afterward. M3-M6, registry migration and direct-backend removal remain
unstarted. Fleet serialization / isolated credential streams for shared Codex
logins remain an external ops follow-up for Phase 2; Phase 1 claims no fleet lock.

## Coordinator artifacts

Branch: coordinator/task-6e2afe00

Files created:

- changelogs/unreleased/2026-10-09-codex-fail-loud-and-chatgpt-timeout.md
- cmd/ailang/ai_handlers_test.go
- internal/ai/chatgpt/client_test.go
- internal/eval_harness/ai_provider_codex_test.go
- design_docs/planned/v0_52_6/m-codex-subscription-lane-phase1-report.md

Files modified:

- internal/ai/config.go
- internal/ai/provider_test.go
- internal/ai/factory/factory.go
- internal/ai/factory/factory_test.go
- internal/ai/chatgpt/client.go
- internal/executor/motoko/provider_preflight.go
- internal/executor/motoko/provider_preflight_test.go
- internal/executor/codex/README.md
- .ailang/state/sprints/sprint_M-CODEX-SUBSCRIPTION-LANE.json
- design_docs/planned/v0_52_6/m-codex-subscription-lane-sprint-plan.md
