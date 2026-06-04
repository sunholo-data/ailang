package eval_harness

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockHTTPURLToken is the placeholder a benchmark's task_prompt uses in place of a
// real external URL. When present, the harness starts a local mock server (see
// StartHTTPMock) and substitutes this token with the live server URL for the run.
//
// Rationale (M-EVAL-NETWORK-MOCK-FIXTURE): a benchmark that calls a real third-party
// service (e.g. httpbin.org) produces non-deterministic verdicts — in the v0.23.0
// run, httpbin returned 503 to 5 of 7 models and 200 to 2, with identical code. A
// local mock makes such benchmarks deterministic, offline, and concurrency-safe.
const MockHTTPURLToken = "{{MOCK_HTTP_URL}}"

// PromptUsesHTTPMock reports whether a task prompt references the mock-URL token.
func PromptUsesHTTPMock(taskPrompt string) bool {
	return strings.Contains(taskPrompt, MockHTTPURLToken)
}

// StartHTTPMock starts a local, ephemeral-port HTTP server that mimics the subset
// of httpbin.org used by network benchmarks: any request returns 200 with a JSON
// envelope echoing the request headers, the parsed JSON body, and the path.
//
// The server binds 127.0.0.1:0 (ephemeral) so concurrent benchmark runs never
// collide. The caller MUST Close() the returned server when the benchmark run
// finishes (success, failure, or timeout) — use defer.
func StartHTTPMock() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		// Echo the posted JSON if it parses; otherwise pass it through as a string.
		var parsed any
		if len(body) > 0 {
			if err := json.Unmarshal(body, &parsed); err != nil {
				parsed = string(body)
			}
		}

		// Flatten headers (httpbin uses single-string values, last wins).
		headers := make(map[string]string, len(r.Header))
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[len(v)-1]
			}
		}

		resp := map[string]any{
			"headers": headers, // includes custom headers like X-Test-Header
			"json":    parsed,  // echoes the posted JSON body
			"method":  r.Method,
			"url":     r.URL.String(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // deterministic 200, every time
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(handler)
}
