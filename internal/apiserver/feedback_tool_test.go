package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHandleSubmitFeedback_RateLimited verifies that once a single client IP
// exhausts its burst, subsequent submit_feedback calls short-circuit with the
// structured `rate_limited` error envelope BEFORE reaching the publisher.
//
// This is the contract M-MCP-EDGE-THROTTLE Path A relies on: a flooding
// client never gets to spawn the downstream coordinator → Cloud Run Job →
// Sonnet chain because we deny at the tool boundary.
func TestHandleSubmitFeedback_RateLimited(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	ms := &MCPServer{
		feedbackRL: NewIPRateLimiter(60, 2), // 1/sec, burst 2
	}
	ms.feedbackRL.now = clock.now

	makeReq := func(xff string) *mcp.CallToolRequest {
		return &mcp.CallToolRequest{
			Extra: &mcp.RequestExtra{
				Header: http.Header{"X-Forwarded-For": []string{xff}},
			},
			Params: &mcp.CallToolParamsRaw{
				Name:      "submit_feedback",
				Arguments: json.RawMessage(`{}`),
			},
		}
	}

	ctx := context.Background()

	// Burst (2 calls) is allowed through to the publisher path. The publisher
	// is unconfigured in this test, so we expect a `publisher_unavailable` or
	// `invalid_input` error envelope — anything except `rate_limited`.
	for i := 0; i < 2; i++ {
		res, err := ms.handleSubmitFeedback(ctx, makeReq("1.2.3.4"))
		if err != nil {
			t.Fatalf("call %d: handler returned go error: %v", i+1, err)
		}
		if res == nil || len(res.Content) == 0 {
			t.Fatalf("call %d: empty result", i+1)
		}
		if got := errCodeFromResult(t, res); got == "rate_limited" {
			t.Fatalf("call %d: burst should pass the limiter, got rate_limited", i+1)
		}
	}

	// Third call from the same IP exhausts the bucket → rate_limited.
	res, err := ms.handleSubmitFeedback(ctx, makeReq("1.2.3.4"))
	if err != nil {
		t.Fatalf("3rd call: handler returned go error: %v", err)
	}
	if got := errCodeFromResult(t, res); got != "rate_limited" {
		t.Fatalf("3rd call: expected rate_limited, got %q", got)
	}
	if !res.IsError {
		t.Fatal("rate-limited result must have IsError=true")
	}

	// A different IP gets its own bucket — should pass the limiter even while
	// 1.2.3.4 is throttled.
	res2, err := ms.handleSubmitFeedback(ctx, makeReq("9.9.9.9"))
	if err != nil {
		t.Fatalf("different-ip call: %v", err)
	}
	if got := errCodeFromResult(t, res2); got == "rate_limited" {
		t.Fatal("a fresh IP must have its own bucket; rate_limited is wrong")
	}
}

// TestHandleSubmitFeedback_LimiterDisabledAllowsAll verifies the operator
// escape hatch: when AILANG_RATELIMIT_RPM=0, NewIPRateLimiter returns nil
// and every call passes through unchecked.
func TestHandleSubmitFeedback_LimiterDisabledAllowsAll(t *testing.T) {
	ms := &MCPServer{feedbackRL: nil}
	ctx := context.Background()
	req := &mcp.CallToolRequest{
		Extra: &mcp.RequestExtra{
			Header: http.Header{"X-Forwarded-For": []string{"flood"}},
		},
		Params: &mcp.CallToolParamsRaw{Name: "submit_feedback", Arguments: json.RawMessage(`{}`)},
	}
	for i := 0; i < 50; i++ {
		res, err := ms.handleSubmitFeedback(ctx, req)
		if err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
		if got := errCodeFromResult(t, res); got == "rate_limited" {
			t.Fatalf("call %d: nil limiter must never deny", i+1)
		}
	}
}

// TestHandleSubmitFeedback_NilExtraIsSafe makes sure a request without HTTP
// Extra (e.g. via the stdio transport) still flows through. Stdio MCP has no
// concept of client IP, so we treat it as unrate-limited rather than crash.
func TestHandleSubmitFeedback_NilExtraIsSafe(t *testing.T) {
	ms := &MCPServer{feedbackRL: NewIPRateLimiter(1, 1)}
	ctx := context.Background()
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "submit_feedback", Arguments: json.RawMessage(`{}`)},
		// no Extra
	}
	res, err := ms.handleSubmitFeedback(ctx, req)
	if err != nil {
		t.Fatalf("nil Extra should be safe, got: %v", err)
	}
	if got := errCodeFromResult(t, res); got == "rate_limited" {
		t.Fatal("nil Extra must not be rate-limited (no IP to bucket)")
	}
}

// errCodeFromResult extracts the "error" code from the MCP result envelope.
// Returns empty string if the result is not an error envelope.
func errCodeFromResult(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &payload); err != nil {
		// Non-JSON content (success path) — surface a non-error sentinel
		return ""
	}
	if e, ok := payload["error"].(string); ok {
		return e
	}
	return ""
}

// TestFeedbackRateLimit_EnvVarOverride sanity-checks the env-var helpers.
func TestFeedbackRateLimit_EnvVarOverride(t *testing.T) {
	tests := []struct {
		name      string
		rpmEnv    string
		burstEnv  string
		wantRPM   int
		wantBurst int
	}{
		{"defaults", "", "", defaultFeedbackRPM, defaultFeedbackBurst},
		{"explicit override", "20", "10", 20, 10},
		{"disable via 0", "0", "5", 0, 5},
		{"junk values fall back to defaults", "abc", "xyz", defaultFeedbackRPM, defaultFeedbackBurst},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AILANG_RATELIMIT_RPM", tc.rpmEnv)
			t.Setenv("AILANG_RATELIMIT_BURST", tc.burstEnv)
			if got := feedbackRateLimitRPM(); got != tc.wantRPM {
				t.Errorf("rpm: got %d, want %d", got, tc.wantRPM)
			}
			if got := feedbackRateLimitBurst(); got != tc.wantBurst {
				t.Errorf("burst: got %d, want %d", got, tc.wantBurst)
			}
		})
	}
}

// TestRateLimitedErrorPayload locks down the exact wire shape of the
// rate-limit envelope so MCP clients can pattern-match it.
func TestRateLimitedErrorPayload(t *testing.T) {
	ms := &MCPServer{feedbackRL: NewIPRateLimiter(60, 1)}
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	ms.feedbackRL.now = clock.now

	req := &mcp.CallToolRequest{
		Extra: &mcp.RequestExtra{
			Header: http.Header{"X-Forwarded-For": []string{"1.2.3.4"}},
		},
		Params: &mcp.CallToolParamsRaw{Name: "submit_feedback", Arguments: json.RawMessage(`{}`)},
	}

	// Burn the burst, then the throttled call.
	_, _ = ms.handleSubmitFeedback(context.Background(), req)
	res, _ := ms.handleSubmitFeedback(context.Background(), req)

	if !res.IsError {
		t.Fatal("rate-limited result must set IsError")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &payload); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if payload["error"] != "rate_limited" {
		t.Errorf("error code: got %v, want rate_limited", payload["error"])
	}
	detail, _ := payload["detail"].(string)
	if !strings.Contains(detail, "60s") {
		t.Errorf("detail should mention retry hint (60s), got %q", detail)
	}
}
