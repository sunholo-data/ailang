package server

import "testing"

// The server's self-tracing filter keeps the hook/exec/chains traffic and
// skips health, the socket, OTLP ingest (a loop), the polling reads and the
// observatory API.
func TestShouldTraceRequest(t *testing.T) {
	skip := []string{
		"/", "/health", "/ws", "/api/approvals", "/api/version", "/api/inbox", "/api/coordinator/events",
		"/v1/traces", "/v1/metrics", "/api/observatory/spans", "/api/controlplane/exec-hierarchy",
		"/favicon.ico", "/logo.png",
	}
	trace := []string{
		"/api/hooks/claude", "/api/exec/sessions", "/api/telemetry", "/api/chains",
		"/api/coordinator/tasks/t-1/events", "/api/approvals/secret-1", "/benchmarks/latest.json",
	}
	for _, p := range skip {
		if shouldTraceRequest(p) {
			t.Errorf("shouldTraceRequest(%q) = true, want false", p)
		}
	}
	for _, p := range trace {
		if !shouldTraceRequest(p) {
			t.Errorf("shouldTraceRequest(%q) = false, want true", p)
		}
	}
}
