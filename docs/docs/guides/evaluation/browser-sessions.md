---
sidebar_position: 10
title: AI Browser Sessions
---

# AI Browser Sessions

AILANG agent evals can attach an isolated browser through the same Playwright
MCP tool surface in two environments:

| Provider | Intended use | Browser location | Cost field |
|---|---|---|---|
| `local-playwright` | Development, private targets, high-volume iteration | This machine | `null`, source `local-resource-unpriced` |
| `browserbase` | Cloud Run, reproducible hosted sessions, recordings | Browserbase | `null` until separately joined to provider billing |

The first executor integration is Codex. A browser-enabled task sent to an
executor without the `mcp` capability fails before a session is provisioned;
there is no silent fallback to a non-browser run.

## Security model

Each eval receives a new provider session. Local sessions use Playwright MCP's
isolated mode and unique state/artifact directories. Browserbase receives one
new keep-alive session which AILANG explicitly releases after the executor
finishes.

Connection URLs and API keys are held only in opaque in-memory values and the
Codex child environment. Codex configuration forwards the environment variable
name `PLAYWRIGHT_MCP_CDP_ENDPOINT`; the endpoint value is not placed in the
prompt, command arguments, task metadata, result JSON, errors, or trace
attributes.

The browser origin allowlist is application policy, not a complete network
sandbox. Redirects, DNS rebinding, WebSockets, service workers, downloads, and
browser subprocess traffic still require worker-level containment for hostile
targets. Do not give unattended neutral evals a real personal Chrome profile.

## Local setup (recommended development lane)

Requirements:

- Node.js 18 or newer and `npx` on `PATH`;
- Codex CLI authenticated with `codex login`;
- enough memory/process headroom for the requested `--parallel` value.

AILANG pins `@playwright/mcp@0.0.79`, whose package metadata pins Playwright
`1.63.0-alpha-2026-08-05` and its bundled Chromium revision. Confirm the package
before the first eval:

```bash
node --version
npx -y @playwright/mcp@0.0.79 --help
codex --version
```

Run the hermetic fixture serially first:

```bash
ailang eval-suite \
  --agent \
  --models gpt5-4 \
  --langs ailang \
  --benchmarks browser_session_fixture \
  --browser-provider local-playwright \
  --browser-artifacts eval_results/browser-local \
  --parallel 1
```

The fixture points the browser at AILANG's ephemeral loopback HTTP server and
uses exact stdout grading. Confirm the result's tool histogram includes a
Playwright browser tool; exact program output alone is not proof the browser was
used.

Do not infer concurrency from installed RAM. Start at `--parallel 1`, then run
the same fixed fixture at 8, 16, and 24 while watching process count, RSS,
timeouts, disconnects, and artifact completeness. Record the highest clean
step as the worker-image default. The design's 50-session sequential leak run
uses the same CLI rather than special test code:

```bash
ailang eval-suite --agent --models gpt5-4 --langs ailang \
  --benchmarks browser_session_fixture --browser-provider local-playwright \
  --browser-artifacts eval_results/browser-local-capacity --trials 50 --parallel 1

# Repeat with --parallel 8, 16, and 24 against separate output directories.
```

These are release/deployment gates, not ordinary unit tests.

## Browserbase setup (recommended first cloud lane)

Create a Browserbase project and inject credentials at runtime:

```bash
export BROWSERBASE_API_KEY='...'
export BROWSERBASE_PROJECT_ID='...'

ailang eval-suite \
  --agent \
  --models gpt5-4 \
  --langs ailang \
  --benchmarks browser_session_fixture \
  --browser-provider browserbase \
  --browser-region eu-central-1 \
  --browser-artifacts eval_results/browserbase \
  --parallel 1
```

For Cloud Run, bind `BROWSERBASE_API_KEY` from Secret Manager directly into the
job environment. Do not put it in an eval request, benchmark YAML, model row,
container argument, or generated MCP config. `BROWSERBASE_PROJECT_ID` may use
the same secret/config injection pattern.

The adapter uses the documented Sessions API:

- `POST /v1/sessions` to allocate a keep-alive session;
- the returned `connectUrl` through the MCP child environment;
- `GET /v1/sessions/{id}/debug` for a separately controlled inspection ref;
- logs and recording endpoints for artifacts;
- `POST /v1/sessions/{id}` with `REQUEST_RELEASE` for idempotent teardown.

The billable live contract test is opt-in:

```bash
AILANG_BROWSERBASE_LIVE=1 \
  go test ./internal/browser/browserbase -run TestLiveBrowserbaseLifecycle -v
```

It skips unless explicitly enabled. A 20-session Cloud Run soak should be run
only against the final worker image and project quota, with its output retained
as deployment evidence.

## Results and comparison rules

Browser-enabled result JSON adds `browser` with:

- provider and safe session identity;
- browser, MCP, and policy versions;
- chain/stage identity and termination reason;
- action/disconnect/reconnect counts and provider usage;
- artifact inventory and export/cleanup status;
- nullable USD cost plus its source;
- neutral-vs-managed vessel labels.

The enclosing `browser.session` span carries only safe provider/session identity,
termination, artifact counts/completeness, cost source, and the opaque
inspection ref, so existing chain/trace views can link the browser lifecycle
without receiving CDP or debugger credentials.

Never treat `cost.usd: null` as zero. Local compute is unpriced by this layer;
Browserbase cost must be joined from provider billing/usage rather than guessed
from wall time.

OpenAI, Anthropic, and Gemini managed computer-use agents are different eval
vessels because their scaffold and tool policy are part of the product. Rows
with a non-empty `agent_scaffold` or `managed_vessel: true` do not enter neutral
local/Browserbase aggregation by default.

## Failure recovery

Stable browser failures use the `browser_*` categories. On executor error,
timeout, or cancellation, the controller exports what is available and then
calls provider stop with a bounded cleanup context. Export failure and cleanup
failure are recorded separately so the original executor failure is not hidden.

For a suspected Browserbase leak, inspect the session in the provider dashboard
and request release. The adapter also exposes an audit operation for known
session handles. For local failures, verify no `@playwright/mcp`/Chromium child
survives, retain the artifact directory, and remove only the session-owned state
directory—not the operator's Chrome profile.

## Current boundaries

- Codex is the reference MCP executor; other executors must advertise and
  implement the same per-task MCP contract before browser selection is allowed.
- Artifact links are banked, but dashboard rendering/access control is a
  follow-up UI integration.
- AWS AgentCore Browser and self-hosted Steel remain future adapters behind the
  same provider contract.
- Live local 50-session and Browserbase 20-session evidence is intentionally not
  fabricated by CI; record it when deploying the final runtime image.
