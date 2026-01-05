# M-GEMINI-TRACE: Gemini CLI Trace Investigation

**Status:** Investigation Required
**Priority:** Medium
**Sprint:** v0.6.4
**Created:** 2026-01-05

## Problem Statement

Gemini CLI traces are not appearing in AILANG Observatory despite:
1. Telemetry being enabled (`enabled: true`)
2. Target set to GCP (`target: "gcp"`)
3. GCP project configured (`multivac-internal-dev`)
4. Composite backend enabled (local + GCP Trace)

When running Gemini CLI, we see confirmation that telemetry is being exported:
```
Creating GCP exporters with projectId: multivac-internal-dev using ADC
```

However, when querying Observatory traces, only `local` source traces appear - no GCP-sourced Gemini traces.

## What We Know

### Confirmed Working
- Gemini CLI executes successfully via skill scripts
- Gemini CLI telemetry settings are correct (`~/.gemini/settings.json`)
- AILANG Observatory composite backend initializes with GCP remote
- OTLP receiver accepts traces from Claude Code and AILANG services
- GCP Cloud Logging receives Gemini CLI events (confirmed via console)

### Observed Issues
1. **Cloud Logging vs Cloud Trace**: Gemini CLI may export to Cloud Logging but not Cloud Trace
2. **Service Name Mismatch**: Unknown what service name Gemini CLI uses
3. **Trace Format**: GCP Trace API may return traces in format we don't recognize
4. **Quota Exhaustion**: Hit 300 reads/minute limit during investigation (60s cache added)

### Key Discovery
Found Gemini CLI events in **Cloud Logging** (not Cloud Trace):
```json
{
  "event.name": "gen_ai.client.inference.operation.details",
  "gen_ai.request.model": "gemini-2.5-flash",
  "gen_ai.input.messages": "[user prompt]",
  "gen_ai.output.messages": "[model response]"
}
```

## Investigation Tasks

### Phase 1: Verify GCP Trace Export (30 min)

- [ ] Check if Gemini CLI actually exports to Cloud Trace (not just Logging)
- [ ] Query Cloud Trace API directly for traces in last hour
- [ ] Identify Gemini CLI's service name in traces
- [ ] Check if traces exist but with different filter criteria

```bash
# Query Cloud Trace via gcloud alpha
gcloud alpha trace traces list --project=multivac-internal-dev --limit=10

# Or via API directly
curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://cloudtrace.googleapis.com/v1/projects/multivac-internal-dev/traces?pageSize=10"
```

### Phase 2: Debug GCP Backend Query (30 min)

- [ ] Add verbose logging to `backend_gcp.go` ListTraces
- [ ] Log raw GCP API response before conversion
- [ ] Check if traces are being filtered out during conversion
- [ ] Verify cache invalidation is working

### Phase 3: Alternative Approaches (1 hr)

If Gemini CLI doesn't export to Cloud Trace, consider alternatives:

#### Option A: Import from Cloud Logging
- Create LogsBackend that queries Cloud Logging API
- Parse gen_ai.* events into Span format
- Add as additional remote in composite backend

#### Option B: Direct OTLP Export
- Configure Gemini CLI with `useCollector: true`
- Point OTLP endpoint directly to Observatory (`localhost:1957`)
- Bypass GCP entirely for local development

#### Option C: OpenTelemetry Collector
- Deploy local OTEL Collector
- Receive from Gemini CLI
- Forward to both GCP and Observatory

## Technical Details

### Gemini CLI Telemetry Settings
```json
// ~/.gemini/settings.json
{
  "telemetry": {
    "enabled": true,
    "target": "gcp",
    "logPrompts": true
  }
}
```

### Observatory Composite Backend
```go
// internal/server/server.go:152
compositeBackend, err := observatory.NewCompositeBackend(observatory.CompositeConfig{
    Local:   sqliteBackend,
    Remotes: []observatory.Backend{gcpBackend},
})
```

### GCP Backend Query
```go
// internal/observatory/backend_gcp.go:190
req := &tracepb.ListTracesRequest{
    ProjectId: b.projectID,
    View:      tracepb.ListTracesRequest_ROOTSPAN,
    StartTime: timestampProto(startTime),
    EndTime:   timestampProto(endTime),
    PageSize:  int32(limit),
}
```

## Expected Outcome

After investigation, we should:
1. Understand exactly where Gemini CLI telemetry goes (Trace vs Logging vs Monitoring)
2. Have a working solution to view Gemini CLI traces in Observatory
3. Document the correct configuration in gemini-cli-helper skill

## References

- [Gemini CLI Telemetry Docs](https://geminicli.com/docs/cli/telemetry/#logs-and-metrics)
- [GCP Cloud Trace API](https://cloud.google.com/trace/docs/reference/v1/rest)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- Related skill: `.claude/skills/gemini-cli-helper/`
