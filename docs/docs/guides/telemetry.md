---
sidebar_position: 20
title: Telemetry & Tracing
description: OpenTelemetry integration for distributed tracing and observability
---

# Telemetry & Tracing

AILANG includes comprehensive OpenTelemetry (OTEL) instrumentation for distributed tracing and observability. This enables integration with standard observability backends like Google Cloud Trace, Grafana, Honeycomb, Jaeger, and more.

## Quick Start

### Google Cloud Trace (Recommended for GCP)

```bash
# Set your GCP project (uses Application Default Credentials)
export GOOGLE_CLOUD_PROJECT=your-project-id

# Start services
ailang serve
ailang coordinator start
```

View traces at: https://console.cloud.google.com/traces/explorer?project=your-project-id

### Generic OTLP (Jaeger, Grafana, Honeycomb, etc.)

```bash
# Set OTLP collector endpoint
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Start services
ailang serve
ailang coordinator start
```

### Dual Export (Both GCP + OTLP)

Send traces to **both** Google Cloud Trace and another backend simultaneously:

```bash
# Configure both
export GOOGLE_CLOUD_PROJECT=your-project-id
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Traces go to both destinations
ailang serve
```

## CLI Trace Commands

AILANG includes a built-in CLI for querying traces from Google Cloud Trace:

### Check Status

```bash
# See current telemetry configuration
ailang trace status
```

Output:
```
Telemetry Configuration Status
────────────────────────────────────────
Google Cloud Project: multivac-internal-dev
OTLP Endpoint:        (not set)

Mode: Google Cloud Trace
View traces: https://console.cloud.google.com/traces/explorer?project=multivac-internal-dev
```

### List Recent Traces

```bash
# List last 10 traces (default)
ailang trace list

# Customize time range and limit
ailang trace list --hours 2 --limit 20

# Filter by span name
ailang trace list --filter "ailang run"

# JSON output for scripting
ailang trace list --json
```

### View Trace Details

```bash
# View full trace hierarchy with timing
ailang trace view <trace-id>
```

Example output:
```
Trace: 5d359e6d157ba7e726aca8a7600a3bfe
Spans: 5
────────────────────────────────────────────────────────────
ailang run: examples/runnable/factorial.ail (2.065ms)
  └─ compile: examples/runnable/factorial.ail (1.458ms)
  └─ compile.load (358µs)
  └─ compile.topo_sort (84µs)
  └─ compile.modules (859µs)
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `GOOGLE_CLOUD_PROJECT` | GCP project for Cloud Trace | `my-project` |
| `OTLP_GOOGLE_CLOUD_PROJECT` | Telemetry-specific GCP project (takes precedence) | `telemetry-project` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint | `http://localhost:4318` |
| `OTEL_ENVIRONMENT` | Deployment environment | `production`, `staging`, `development` |

**Priority for GCP project:**
1. `OTLP_GOOGLE_CLOUD_PROJECT` (if set)
2. `GOOGLE_CLOUD_PROJECT` (fallback)

This matches the [Gemini CLI telemetry convention](https://geminicli.com/docs/cli/telemetry/).

## Instrumented Components

All components emit traces automatically when telemetry is configured:

### Compiler Pipeline

The compilation pipeline emits detailed spans for each phase:

**Single-file/REPL compilation:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `compile.pipeline` | Parent span for entire compilation | `file.path`, `file.size_bytes`, `is_repl` |
| `compile.parse` | Parsing phase | `ast.nodes` (count) |
| `compile.elaborate` | Surface→Core elaboration | - |
| `compile.typecheck` | Type checking | - |
| `compile.validate` | CoreTypeInfo validation | - |
| `compile.lower` | Operator lowering | - |

**Module compilation:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `compile.module_pipeline` | Parent span for module compilation | `file.path`, `file.size_bytes` |
| `compile.load` | Module loading | `modules.loaded` (count) |
| `compile.topo_sort` | Topological sort | `modules.sorted` (count) |
| `compile.modules` | Compile all modules | `modules.count` |

### Eval Harness (`ailang eval-suite`)

The benchmark evaluation system emits spans for suite execution and individual benchmarks:

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `eval.suite` | Parent span for entire benchmark run | `eval.models`, `eval.benchmarks`, `eval.languages`, `eval.total_runs`, `eval.agent_mode`, `eval.success_count`, `eval.fail_count`, `eval.success_rate` |
| `eval.benchmark` | Individual benchmark execution | `benchmark.id`, `benchmark.model`, `benchmark.language`, `benchmark.seed`, `benchmark.success`, `benchmark.duration_ms`, `benchmark.input_tokens`, `benchmark.output_tokens`, `benchmark.cost_usd` |

**v0.6.3+ Enhanced Attributes:**

For successful benchmarks:
- `code.preview` - First 100 chars of generated code
- `code.hash` - 8-char hash for deduplication

For failed benchmarks:
- `error.summary` - Truncated error message
- `error.category` - Error classification

For standard mode (with repair):
- `benchmark.repair_successful` - Whether self-repair succeeded

### Messaging System (`ailang messages`)

Message operations emit spans for observability:

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `messages.send` | Create/insert message | `message.to_inbox`, `message.from_agent`, `message.type`, `message.category`, `message.id` |
| `messages.list` | List messages with filters | `list.inbox`, `list.unread_only`, `list.collapsed`, `list.limit`, `list.result_count` |
| `messages.read` | Read single message by ID | `message.id`, `message.from_agent`, `message.to_inbox`, `message.type` |
| `messages.search` | Semantic search | `search.query`, `search.use_neural`, `search.threshold`, `search.limit`, `search.inbox`, `search.result_count` |
| `messages.ack` | Mark message as read | `message.id`, `message.new_status` |
| `messages.unack` | Mark message as unread | `message.id`, `message.new_status` |
| `messages.cleanup` | Delete old/expired messages | `cleanup.older_than`, `cleanup.expired_only`, `cleanup.dry_run`, `cleanup.deleted_count` |
| `messages.github_sync` | Import issues from GitHub | `github.repo`, `sync.dry_run`, `github.issues_found`, `sync.imported`, `sync.skipped` |

### REPL (`ailang repl`)

Interactive REPL sessions emit session-level and input-level spans:

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `repl.session` | Parent span for entire REPL session | `session.id`, `version`, `session.input_count`, `session.duration_ms` |
| `repl.input` | Individual user input evaluation | `input.type` (command/expression), `input.text` (truncated 200 chars), `input.number` |

**Span Hierarchy:**
```
repl.session (duration of interactive session)
  └─ repl.input #1 (first user input)
  └─ repl.input #2 (second user input)
  └─ ... (subsequent inputs)
```

Session metrics are finalized when the REPL exits, capturing total input count and session duration.

### Check Command (`ailang check`)

The type checking command emits spans for file/directory verification:

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `ailang.check` | Root span for check operation | `file.path`, `timeout_ms`, `is_directory` |
| `check.result` | Check outcome with pass/fail | `passed` (bool), `errors.count`, `timed_out` (if timeout occurred) |

**Span Hierarchy:**
```
ailang.check (root span)
  └─ check.result (with pass/fail and error counts)
  └─ compile.* (compilation phases from compiler pipeline)
```

When using `--timeout`, the `timed_out` attribute is set if compilation exceeds the limit.

### Server (`ailang serve`)

- HTTP request/response spans via `otelhttp` middleware
- Automatic status codes, latency, and path attributes
- Filters out `/health` and `/ws` endpoints

### Coordinator (`ailang coordinator start`)

- Task lifecycle spans: `coordinator.execute_task`
- Attributes: `task.id`, `task.type`, `task.stage`
- Token and cost tracking per task

### Executors

**Claude Executor:**
- Span: `claude.execute`
- Attributes: `executor.model`, `task.workspace`, `session.id`
- Token counts: `task.tokens_in`, `task.tokens_out`
- Cost: `task.cost_usd`

**Gemini Executor:**
- Span: `gemini.execute`
- Same attributes as Claude executor

### AI Providers

All AI providers emit spans for API calls:

| Provider | Span Name | Key Attributes |
|----------|-----------|----------------|
| Anthropic | `anthropic.generate` | `ai.model`, `ai.tokens_in`, `ai.tokens_out`, `http.status_code`, `ai.prompt_preview`, `ai.response_preview`, `ai.finish_reason` |
| OpenAI | `openai.generate` | `ai.model`, `ai.api_type` (chat/responses), `ai.tokens_*`, `ai.prompt_preview`, `ai.response_preview` |
| Gemini | `gemini.generate` | `ai.model`, `ai.auth_type` (api_key/adc), `ai.tokens_*`, `ai.prompt_preview`, `ai.response_preview` |
| Ollama | `ollama.generate` | `ai.model`, `ai.endpoint`, `ai.prompt_preview`, `ai.response_preview` |

## Telemetry Helpers (v0.6.3+)

The `internal/telemetry` package provides helper functions for safe, consistent span attributes:

### Truncate

Safely truncate strings for span attributes, preserving UTF-8 boundaries:

```go
import "github.com/sunholo/ailang/internal/telemetry"

// Truncate to 100 chars, adding "..." if truncated
preview := telemetry.Truncate(longString, 100)
// "Hello, 世界..." (never breaks in middle of multi-byte chars)
```

### CategorizeError

Categorize errors for filtering and aggregation:

```go
category := telemetry.CategorizeError(err)
// Returns: "network", "timeout", "auth", "rate_limit", "parse", "type", "runtime", or "unknown"
```

### ShortHash

Generate deterministic short hashes for deduplication:

```go
hash := telemetry.ShortHash(codeString)
// Returns 8-char hex string like "a1b2c3d4"
```

### LineSnippet

Extract source code context around a line number:

```go
snippet := telemetry.LineSnippet(sourceCode, lineNumber, 60)
// Returns up to 60 chars of the specified line
```

## Example: Local Jaeger Setup

```bash
# Start Jaeger with OTLP collector
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# Configure AILANG
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Start services
ailang serve &
ailang coordinator start

# View traces at http://localhost:16686
```

## Example: Google Cloud Trace + Local Jaeger

```bash
# Dual export - both GCP and local Jaeger
export GOOGLE_CLOUD_PROJECT=my-gcp-project
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Traces go to BOTH destinations
ailang serve
```

## Integration Tests

Run the integration tests to verify your Google Cloud Trace setup:

```bash
# Set project
export GOOGLE_CLOUD_PROJECT=your-project-id

# Run tests
go test -tags=integration -v -run TestGoogleCloudTrace ./internal/telemetry/...
```

Tests create sample traces and verify they export correctly.

## Native CLI Telemetry

Both Claude Code and Gemini CLI have native OTEL support:

### Claude Code

```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

### Gemini CLI

Configure in `~/.gemini/settings.json`:
```json
{
  "telemetry": {
    "enabled": true,
    "endpoint": "http://localhost:4318"
  }
}
```

## Trace Attributes Reference

### Common Attributes

All spans include:
- `service.name` - Service identifier (e.g., `ailang-server`, `ailang-coordinator`)
- `service.version` - AILANG version
- `deployment.environment` - From `OTEL_ENVIRONMENT` (default: `development`)
- `process.runtime.name` - `go`
- `process.runtime.version` - Go version

### AI-Specific Attributes

| Attribute | Description |
|-----------|-------------|
| `ai.provider` | Provider name: `anthropic`, `openai`, `gemini`, `ollama` |
| `ai.model` | Model ID (e.g., `claude-sonnet-4-5`, `gpt-5`) |
| `ai.tokens_in` | Input tokens |
| `ai.tokens_out` | Output tokens |
| `ai.tokens_total` | Total tokens |
| `ai.cost_usd` | Estimated cost in USD |
| `ai.prompt_preview` | First 100 chars of prompt (v0.6.3+) |
| `ai.response_preview` | First 100 chars of response (v0.6.3+) |
| `ai.finish_reason` | Why generation stopped: `end_turn`, `max_tokens`, etc. (v0.6.3+) |

### Task Attributes

| Attribute | Description |
|-----------|-------------|
| `task.id` | Unique task identifier |
| `task.type` | Task type: `bug`, `feature`, `docs`, etc. |
| `task.stage` | Pipeline stage: `design`, `sprint`, `implement` |
| `task.success` | Boolean success status |
| `task.duration_ms` | Duration in milliseconds |

### Error Context Attributes (v0.6.3+)

When errors occur, spans include rich debugging context:

| Attribute | Description |
|-----------|-------------|
| `error.message` | Truncated error message (max 200 chars, UTF-8 safe) |
| `error.category` | Error type: `network`, `timeout`, `auth`, `rate_limit`, `parse`, `type`, `runtime`, `unknown` |
| `error.location` | Position in source: `line:column` format |
| `error.snippet` | Source code around error (max 60 chars) |
| `error.summary` | Short error description for failed benchmarks |

### Code Context Attributes (v0.6.3+)

Eval benchmarks include code analysis attributes:

| Attribute | Description |
|-----------|-------------|
| `code.preview` | First 100 chars of generated code |
| `code.hash` | Short hash (8 chars) for deduplication |
| `benchmark.repair_successful` | Whether self-repair succeeded (standard mode) |

### CLI Run Attributes (v0.6.3+)

The `ailang run` command includes:

| Attribute | Description |
|-----------|-------------|
| `file.path` | Path to the executed file |
| `entry.function` | Entry point function name |
| `caps.granted` | List of granted capabilities (e.g., `["IO", "FS"]`) |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Your Application                         │
├─────────────────────────────────────────────────────────────┤
│  Server    │  Coordinator  │  Executors  │  AI Providers    │
│  (HTTP)    │  (Tasks)      │  (Claude/   │  (Anthropic/     │
│            │               │   Gemini)   │   OpenAI/etc)    │
└────────────┴───────────────┴─────────────┴──────────────────┘
                              │
                    otel.Tracer("...")
                              │
                    ┌─────────▼─────────┐
                    │  TracerProvider   │
                    │  (Global)         │
                    └─────────┬─────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
    ┌─────────▼────┐  ┌───────▼────┐  ┌───────▼──────┐
    │ GCP Exporter │  │ OTLP       │  │ (disabled)   │
    │ Cloud Trace  │  │ Exporter   │  │ No-op        │
    └──────────────┘  └────────────┘  └──────────────┘
              │               │
              ▼               ▼
    Google Cloud        Jaeger/Grafana/
    Console             Honeycomb/etc
```

## Performance Overhead

### When Disabled (Default)

When no telemetry environment variables are set:
- No exporters are initialized
- Tracers return no-op spans
- **~2-5 nanoseconds** per span call (just a nil check)
- Zero memory allocations
- No external connections made

This is negligible - you can leave the instrumentation in place without any measurable impact.

### When Enabled (Production Use)

When `GOOGLE_CLOUD_PROJECT` or `OTEL_EXPORTER_OTLP_ENDPOINT` is set:
- **~50-200 microseconds** per compilation (all phases combined)
- **~100-500 microseconds** per AI API call
- Spans are batched and exported asynchronously
- Export happens in background goroutines (doesn't block your code)

**Production recommendations:**
- Use sampling for high-throughput services (e.g., 10% of requests)
- Batch exporters are enabled by default
- Set `OTEL_ENVIRONMENT=production` for environment tagging

### Overhead Breakdown by Component

| Component | Spans per Operation | Typical Overhead |
|-----------|---------------------|------------------|
| Compiler Pipeline | 5-7 spans | ~100μs |
| Eval Harness | 2 spans per benchmark | ~50μs |
| Messaging | 1 span per operation | ~20μs |
| REPL Session | 1 session + N input spans | ~30μs per input |
| Check Command | 2 spans (root + result) | ~40μs |
| AI Providers | 1 span per API call | ~30μs |

**Note:** Actual overhead depends on your OTEL collector. Local Jaeger adds ~10μs, while cloud exports (GCP, Honeycomb) add ~50-100μs due to batching and network I/O.

## Troubleshooting

### Traces not appearing in GCP?

1. Verify ADC credentials: `gcloud auth application-default login`
2. Check project: `gcloud config get-value project`
3. Verify permissions: Cloud Trace Agent role required
4. Run integration test: `go test -tags=integration ./internal/telemetry/...`

### OTLP connection refused?

1. Check collector is running: `curl http://localhost:4318/v1/traces`
2. Verify endpoint URL includes protocol: `http://` not just `localhost:4318`
3. Check firewall/port access

### No spans from AI providers?

1. Verify telemetry is initialized before AI calls
2. Check spans are ending: `defer span.End()` in all paths
3. Enable debug logging: set `OTEL_LOG_LEVEL=debug`
