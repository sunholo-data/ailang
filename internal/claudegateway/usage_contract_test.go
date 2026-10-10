package claudegateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// These are synthetic protocol fixtures, not receipts for the live canary.
// The official SDK replaces cumulative whole-message counters in message_delta.
func TestCumulativeStreamingUsageSettlesFinalTotals(t *testing.T) {
	for _, tc := range []struct {
		name, delta string
		want        modelreg.ClaudeUsage
	}{
		{"updated totals", `{"input_tokens":101,"output_tokens":10,"cache_read_input_tokens":21}`, modelreg.ClaudeUsage{InputTokens: 101, OutputTokens: 10, CacheReadTokens: 21, CacheWrite5mTokens: 3, CacheWrite1hTokens: 4}},
		{"nullable optional totals", `{"input_tokens":null,"output_tokens":10,"cache_read_input_tokens":null,"cache_creation_input_tokens":null,"server_tool_use":null,"output_tokens_details":null}`, modelreg.ClaudeUsage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 20, CacheWrite5mTokens: 3, CacheWrite1hTokens: 4}},
		{"inclusive output breakdown", `{"output_tokens":10,"output_tokens_details":{"thinking_tokens":6}}`, modelreg.ClaudeUsage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 20, CacheWrite5mTokens: 3, CacheWrite1hTokens: 4}},
		{"updated TTL evidence", `{"output_tokens":10,"cache_creation_input_tokens":9,"cache_creation":{"ephemeral_5m_input_tokens":5,"ephemeral_1h_input_tokens":4}}`, modelreg.ClaudeUsage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 20, CacheWrite5mTokens: 5, CacheWrite1hTokens: 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-contract","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0,"cache_read_input_tokens":20,"cache_creation_input_tokens":7,"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":4},"output_tokens_details":null}}}`)
			stream := start + usageEvent("message_delta", `{"type":"message_delta","usage":`+tc.delta+`}`) + usageEvent("message_stop", `{"type":"message_stop"}`)
			g, authority, capability := fixture(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}, nil
			})
			invoke(g, capability, strings.TrimSuffix(plainRequest, "}")+`,"stream":true}`)
			cost, err := modelreg.ClaudeUsageCost("claude-haiku-5-5", tc.want)
			if err != nil {
				t.Fatal(err)
			}
			status, err := authority.Status(context.Background(), AccountID)
			if err != nil || int64(status.Settled) != cost || status.Reserved != 0 || status.ReconciliationRequired {
				t.Fatalf("final cumulative totals not settled: %+v want cost=%d err=%v", status, cost, err)
			}
		})
	}
}

func TestMultipleCumulativeEventsPreserveLatestNonNullTotals(t *testing.T) {
	stream := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-contract","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0}}}`) +
		usageEvent("message_delta", `{"type":"message_delta","usage":{"input_tokens":101,"output_tokens":5,"cache_read_input_tokens":21}}`) +
		usageEvent("message_delta", `{"type":"message_delta","usage":{"input_tokens":null,"output_tokens":10,"cache_read_input_tokens":null}}`) +
		usageEvent("message_stop", `{"type":"message_stop"}`)
	var trace usageTrace
	usage, id, err := relayStreamWithTrace(httptest.NewRecorder(), strings.NewReader(stream), "claude-haiku-5-5", 10, &trace)
	if err != nil || id != "msg-contract" || usage != (modelreg.ClaudeUsage{InputTokens: 101, OutputTokens: 10, CacheReadTokens: 21}) {
		t.Fatalf("latest cumulative totals lost or added twice: %+v id=%s err=%v", usage, id, err)
	}
	if trace.LastVerified == nil || *trace.LastVerified != usage || trace.Final["input_tokens"] != nil || !trace.Stopped {
		t.Fatalf("latest verified observations lost from diagnostic: %+v", trace)
	}
}

func TestMalformedBillingCountersRetainExposure(t *testing.T) {
	for _, usage := range []string{
		`{"input_tokens":null,"output_tokens":1}`,
		`{"input_tokens":1,"output_tokens":null}`,
		`{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":1,"cache_creation":{"ephemeral_5m_input_tokens":1,"ephemeral_1h_input_tokens":null}}`,
		`{"input_tokens":1,"output_tokens":1,"output_tokens_details":{"thinking_tokens":null}}`,
		`{"input_tokens":1,"output_tokens":1,"output_tokens_details":{"thinking_tokens":2}}`,
		`{"input_tokens":1,"output_tokens":1,"output_tokens_details":{"unpriced_tokens":1}}`,
		`{"input_tokens":1,"output_tokens":1,"cache_creation":{"ephemeral_5m_input_tokens":null}}`,
		`{"input_tokens":1,"output_tokens":1,"cache_creation":{"unsupported_ttl":0}}`,
		`{"input_tokens":1,"output_tokens":1,"server_tool_use":{"web_search_requests":null}}`,
		`{"input_tokens":1,"output_tokens":1,"server_tool_use":{"unpriced_tool":0}}`,
	} {
		t.Run(usage, func(t *testing.T) {
			assertUsageFailureHoldsCredit(t, `{"id":"msg-test","type":"message","model":"claude-haiku-5-5","usage":`+usage+`}`, false)
		})
	}
	start := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-test","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0,"cache_creation_input_tokens":7,"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":4}}}}`)
	for _, delta := range []string{
		`{"output_tokens":null}`,
		`{"output_tokens":10,"cache_creation_input_tokens":9}`,
		`{"output_tokens":10,"cache_creation_input_tokens":9,"cache_creation":null}`,
	} {
		t.Run(delta, func(t *testing.T) {
			assertUsageFailureHoldsCredit(t, start+usageEvent("message_delta", `{"type":"message_delta","usage":`+delta+`}`)+usageEvent("message_stop", `{"type":"message_stop"}`), true)
		})
	}
}
