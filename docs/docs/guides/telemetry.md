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
| Anthropic | `anthropic.generate` | `ai.model`, `ai.tokens_in`, `ai.tokens_out`, `http.status_code` |
| OpenAI | `openai.generate` | `ai.model`, `ai.api_type` (chat/responses), `ai.tokens_*` |
| Gemini | `gemini.generate` | `ai.model`, `ai.auth_type` (api_key/adc), `ai.tokens_*` |
| Ollama | `ollama.generate` | `ai.model`, `ai.endpoint` |

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

### Task Attributes

| Attribute | Description |
|-----------|-------------|
| `task.id` | Unique task identifier |
| `task.type` | Task type: `bug`, `feature`, `docs`, etc. |
| `task.stage` | Pipeline stage: `design`, `sprint`, `implement` |
| `task.success` | Boolean success status |
| `task.duration_ms` | Duration in milliseconds |

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

## Zero Overhead When Disabled

When no telemetry environment variables are set:
- No exporters are initialized
- Tracers return no-op spans
- Zero performance impact
- No external connections made

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
