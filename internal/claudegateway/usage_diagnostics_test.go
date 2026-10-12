package claudegateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUnresolvedDiagnosticsContainOnlyBoundedBillingEvidence(t *testing.T) {
	start := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-contract","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0}}}`)
	content := usageEvent("content_block_start", `{"type":"content_block_start","content_block":{"type":"text","text":"PRIVATE-PROMPT-CONTENT"}}`)
	delta := usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":10,"unpriced_private_field":"SECRET-PROVIDER-VALUE"}}`)
	g, _, capability := fixture(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}, "Request-Id": {"req-contract"}}, Body: io.NopCloser(strings.NewReader(start + content + delta))}, nil
	})
	var diagnostic UsageDiagnostic
	calls := 0
	g.Diagnostic = func(d UsageDiagnostic) { calls++; diagnostic = d }
	invoke(g, capability, strings.TrimSuffix(plainRequest, "}")+`,"stream":true}`)
	if calls != 1 || diagnostic.ProviderRequestID != "req-contract" || diagnostic.MessageID != "msg-contract" || diagnostic.RequestID == "" || diagnostic.Stage != "stream_usage" || diagnostic.Reason != "unsupported stream usage category" || !diagnostic.UnresolvedMarked || !diagnostic.Stream.Started || !diagnostic.Stream.FinalSeen || diagnostic.Stream.Stopped {
		t.Fatalf("missing failure evidence: %+v", diagnostic)
	}
	encoded, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"PRIVATE-PROMPT-CONTENT", "SECRET-PROVIDER-VALUE", "unpriced_private_field", g.ProviderKey, capability} {
		if private != "" && strings.Contains(string(encoded), private) {
			t.Fatal("diagnostic leaked content, arbitrary fields or credentials")
		}
	}
	if diagnostic.Stream.Start["input_tokens"] != int64(100) || diagnostic.Stream.Final["output_tokens"] != int64(10) {
		t.Fatalf("billing counters missing: %+v", diagnostic.Stream)
	}
}

func TestDiagnosticRejectsArbitraryErrorsAndIDs(t *testing.T) {
	if diagnosticReason(errors.New("PRIVATE-UPSTREAM-ERROR")) != "unverifiable_response" {
		t.Fatal("arbitrary error text leaked")
	}
	if diagnosticID(strings.Repeat("a", 129)) != "invalid" || diagnosticID("req\nPRIVATE") != "invalid" || diagnosticID("req-safe_1") != "req-safe_1" {
		t.Fatal("unbounded provider identifier")
	}
	observation := usageObservation(json.RawMessage(`{"input_tokens":"PRIVATE","output_tokens":null,"cache_creation":{"ephemeral_5m_input_tokens":7,"secret":"PRIVATE"}}`))
	encoded, err := json.Marshal(observation)
	if err != nil || strings.Contains(string(encoded), "PRIVATE") || strings.Contains(string(encoded), "secret") {
		t.Fatal("arbitrary usage values leaked")
	}
}
