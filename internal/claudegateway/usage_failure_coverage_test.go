package claudegateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// A response that cannot prove its charge must retain the entire reservation,
// including the input/cache upper bound, and prevent another provider send.
func assertUsageFailureHoldsCredit(t *testing.T, content string, stream bool) {
	t.Helper()
	calls := 0
	ctype := "application/json"
	request := plainRequest
	if stream {
		ctype = "text/event-stream"
		request = strings.TrimSuffix(request, "}") + `,"stream":true}`
	}
	g, authority, capability := fixture(t, func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {ctype}}, Body: io.NopCloser(strings.NewReader(content))}, nil
	})
	amount, _, err := modelreg.ReserveClaudeRequest("claude-haiku-5-5", 10)
	if err != nil {
		t.Fatal(err)
	}
	invoke(g, capability, request)
	status, err := authority.Status(context.Background(), AccountID)
	if err != nil || status.Reserved != creditbudget.MicroUSD(amount) || status.Unresolved != status.Reserved || status.Settled != 0 || !status.ReconciliationRequired {
		t.Fatalf("unverifiable usage lost exposure: %+v, err=%v", status, err)
	}
	if response := invoke(g, capability, request); response.Code != http.StatusPaymentRequired || calls != 1 {
		t.Fatalf("unresolved spend allowed another send: status=%d calls=%d", response.Code, calls)
	}
}

func TestUnverifiableMessageUsageRetainsFullCredit(t *testing.T) {
	for _, tc := range []struct{ name, usage string }{
		{"missing input", `{"output_tokens":1}`},
		{"missing output", `{"input_tokens":1}`},
		{"negative input", `{"input_tokens":-1,"output_tokens":1}`},
		{"fractional output", `{"input_tokens":1,"output_tokens":1.5}`},
		{"overflow input", `{"input_tokens":9223372036854775808,"output_tokens":1}`},
		{"invalid cache read", `{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":"100"}`},
		{"negative cache creation", `{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":-1}`},
		{"cache TTL category missing", `{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":10}}`},
		{"invalid five minute count", `{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":"10","ephemeral_1h_input_tokens":0}}`},
		{"invalid one hour count", `{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":10,"ephemeral_1h_input_tokens":[]}}`},
		{"nonzero TTL without creation total", `{"input_tokens":1,"output_tokens":1,"cache_creation":{"ephemeral_1h_input_tokens":10}}`},
		{"invalid zero TTL accounting", `{"input_tokens":1,"output_tokens":1,"cache_creation":{"ephemeral_1h_input_tokens":"0"}}`},
		{"invalid server tool accounting", `{"input_tokens":1,"output_tokens":1,"server_tool_use":{"web_search_requests":"0"}}`},
		{"impossible context", `{"input_tokens":1000001,"output_tokens":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message := `{"id":"msg-test","type":"message","model":"claude-haiku-5-5","usage":` + tc.usage + `}`
			assertUsageFailureHoldsCredit(t, message, false)
		})
	}
	for _, tc := range []struct{ name, response string }{
		{"truncated JSON", `{"id":"msg-test"`},
		{"missing identity", strings.Replace(goodResponse, `"msg-test"`, `""`, 1)},
		{"other model", strings.Replace(goodResponse, "claude-haiku-5-5", "claude-sonnet-5-5", 1)},
		{"wrong message type", strings.Replace(goodResponse, `"type":"message"`, `"type":"error"`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) { assertUsageFailureHoldsCredit(t, tc.response, false) })
	}
}

func usageEvent(typ, body string) string {
	return "event: " + typ + "\ndata: " + body + "\n\n"
}

func TestUnverifiableStreamingUsageRetainsFullCredit(t *testing.T) {
	start := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-test","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0}}}`)
	delta := usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":10}}`)
	stop := usageEvent("message_stop", `{"type":"message_stop"}`)
	for _, tc := range []struct{ name, response string }{
		{"malformed event data", "event: message_start\ndata: {\n\n"},
		{"event after stop", start + delta + stop + usageEvent("ping", `{"type":"ping"}`)},
		{"duplicate start", start + start},
		{"invalid message object", usageEvent("message_start", `{"type":"message_start","message":[]}`)},
		{"invalid stream identity", strings.Replace(start, `"msg-test"`, `""`, 1)},
		{"incomplete start usage", strings.Replace(start, `"input_tokens":100,`, "", 1)},
		{"delta before start", delta},
		{"invalid delta usage", start + usageEvent("message_delta", `{"type":"message_delta","usage":[]}`)},
		{"fractional cumulative output", start + usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":1.5}}`)},
		{"output exceeds ceiling", start + strings.Replace(delta, `"output_tokens":10`, `"output_tokens":11`, 1)},
		{"delta missing output", start + usageEvent("message_delta", `{"type":"message_delta","usage":{"input_tokens":100}}`)},
		{"changed input usage", start + usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":10,"input_tokens":101}}`)},
		{"changed cache usage", start + usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":10,"cache_read_input_tokens":1}}`)},
		{"unpriced delta category", start + usageEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":10,"iterations":[]}}`)},
		{"stop without start", stop},
		{"stop without final usage", start + stop},
		{"content before start", usageEvent("content_block_start", `{"type":"content_block_start"}`)},
		{"content after final usage", start + delta + usageEvent("content_block_delta", `{"type":"content_block_delta"}`)},
		{"unknown event", start + usageEvent("provider_extra", `{"type":"provider_extra"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) { assertUsageFailureHoldsCredit(t, tc.response, true) })
	}
}

func TestVerifiedMixedCacheUsageSettlesAtCorrectContextTier(t *testing.T) {
	for _, tc := range []struct {
		name, usage string
		cost        creditbudget.MicroUSD
	}{
		{"short mixed TTL", `{"input_tokens":100,"output_tokens":10,"cache_read_input_tokens":200,"cache_creation_input_tokens":70,"cache_creation":{"ephemeral_5m_input_tokens":30,"ephemeral_1h_input_tokens":40},"server_tool_use":{"web_search_requests":0},"service_tier":"standard","inference_geo":"global"}`, 29},
		{"cache read crosses tier", `{"input_tokens":100,"output_tokens":10,"cache_read_input_tokens":100001,"cache_creation":null,"server_tool_use":null,"inference_geo":""}`, 5076},
		{"explicit zero cache", `{"input_tokens":100,"output_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"server_tool_use":{"web_search_requests":0}}`, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, authority, capability := fixture(t, func(*http.Request) (*http.Response, error) {
				body := `{"id":"msg-test","type":"message","model":"claude-haiku-5-5","usage":` + tc.usage + `}`
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			if response := invoke(g, capability, plainRequest); response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			status, err := authority.Status(context.Background(), AccountID)
			if err != nil || status.Reserved != 0 || status.Unresolved != 0 || status.Settled != tc.cost || status.ReconciliationRequired {
				t.Fatalf("incorrect cache settlement: %+v, want %d, err=%v", status, tc.cost, err)
			}
		})
	}
}

func TestStreamingRepeatedVerifiedUsageIsNotChargedTwice(t *testing.T) {
	usage := `{"input_tokens":100,"output_tokens":0,"cache_read_input_tokens":20,"cache_creation_input_tokens":7,"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":4}}`
	start := usageEvent("message_start", `{"type":"message_start","message":{"id":"msg-cache","model":"claude-haiku-5-5","usage":`+usage+`}}`)
	deltaUsage := strings.TrimSuffix(strings.Replace(usage, `"output_tokens":0`, `"output_tokens":10`, 1), "}") + `,"service_tier":"standard","inference_geo":"global","server_tool_use":{"web_search_requests":0}}`
	stream := ": keepalive\n\n" + start + usageEvent("message_delta", `{"type":"message_delta","usage":`+deltaUsage+`}`) + usageEvent("message_stop", `{"type":"message_stop"}`)
	w := httptest.NewRecorder()
	u, id, err := relayStream(w, strings.NewReader(stream), "claude-haiku-5-5", 10)
	want := modelreg.ClaudeUsage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 20, CacheWrite5mTokens: 3, CacheWrite1hTokens: 4}
	if err != nil || id != "msg-cache" || u != want || w.Body.String() != stream {
		t.Fatalf("repeated usage changed accounting or relay: %+v %q %v", u, id, err)
	}
}

type usageBrokenReader struct{ err error }

func (r usageBrokenReader) Read([]byte) (int, error) { return 0, r.err }

type usageBrokenWriter struct {
	*httptest.ResponseRecorder
	err error
}

func (w usageBrokenWriter) Write([]byte) (int, error) { return 0, w.err }

func TestStreamingIOReturnsNoVerifiableSettlement(t *testing.T) {
	broken := errors.New("stream connection lost")
	for _, tc := range []struct {
		name string
		body io.Reader
		w    http.ResponseWriter
	}{
		{"upstream read", usageBrokenReader{broken}, httptest.NewRecorder()},
		{"downstream write", strings.NewReader("event: ping\n"), usageBrokenWriter{httptest.NewRecorder(), broken}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, id, err := relayStream(tc.w, tc.body, "claude-haiku-5-5", 10)
			if !errors.Is(err, broken) || id != "" {
				t.Fatalf("IO failure became verified usage: id=%q err=%v", id, err)
			}
		})
	}
}

func TestOptionalOutputUsageRetainsAllReportedCacheCategories(t *testing.T) {
	u, err := parseUsage(json.RawMessage(`{"input_tokens":100,"cache_read_input_tokens":20}`), false)
	if err != nil || u != (modelreg.ClaudeUsage{InputTokens: 100, CacheReadTokens: 20}) {
		t.Fatalf("optional output corrupted input usage: %+v %v", u, err)
	}
}
