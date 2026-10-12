package claudegateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func nestedAdmissionBody(feature string) string {
	return strings.TrimSuffix(strings.Replace(plainRequest, `"max_tokens":10`, `"max_tokens":2048`, 1), "}") + "," + feature + "}"
}

// These are authenticated requests, so rejection must precede both reservation
// and the provider transport. A JSON error alone would not prove that boundary.
func TestMalformedNestedContractsHaveNoBillingSideEffects(t *testing.T) {
	cases := []struct{ name, feature string }{
		{"thinking must be object", `"thinking":[]`},
		{"thinking type must be string", `"thinking":{"type":false}`},
		{"enabled requires budget", `"thinking":{"type":"enabled"}`},
		{"adaptive cannot set legacy budget", `"thinking":{"type":"adaptive","budget_tokens":1024}`},
		{"disabled cannot set budget", `"thinking":{"type":"disabled","budget_tokens":1024}`},
		{"thinking budget below minimum", `"thinking":{"type":"enabled","budget_tokens":1023}`},
		{"thinking budget consumes output ceiling", `"thinking":{"type":"enabled","budget_tokens":2048}`},
		{"thinking budget must be integer", `"thinking":{"type":"enabled","budget_tokens":1024.5}`},
		{"thinking display unreviewed", `"thinking":{"type":"adaptive","display":"full"}`},
		{"disabled cannot select display", `"thinking":{"type":"disabled","display":"omitted"}`},
		{"output config must be object", `"output_config":true`},
		{"effort must be string", `"output_config":{"effort":5}`},
		{"tool choice must be object", `"tool_choice":"auto"`},
		{"named tool must have name", `"tool_choice":{"type":"tool"}`},
		{"named tool cannot be empty", `"tool_choice":{"type":"tool","name":""}`},
		{"named tool name must be string", `"tool_choice":{"type":"tool","name":[]}`},
		{"automatic choice cannot select named tool", `"tool_choice":{"type":"auto","name":"paid"}`},
		{"parallel tool option must be boolean", `"tool_choice":{"type":"any","disable_parallel_tool_use":"false"}`},
		{"metadata must be object", `"metadata":[]`},
		{"metadata identity must be string", `"metadata":{"user_id":123}`},
		{"metadata identity over bound", `"metadata":{"user_id":"` + strings.Repeat("u", 4097) + `"}`},
		{"tools must be array", `"tools":{}`},
		{"strict tool flag must be boolean", `"tools":[{"name":"x","input_schema":{},"strict":"true"}]`},
		{"unreviewed tool billing feature", `"tools":[{"name":"x","input_schema":{},"defer_loading":true}]`},
		{"stream flag must be boolean", `"stream":"true"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertRequestRejectedBeforeBilling(t, nestedAdmissionBody(tc.feature))
		})
	}
}

func assertRequestRejectedBeforeBilling(t *testing.T, body string) {
	t.Helper()
	calls := 0
	g, ledger, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		calls++
		return nil, fmt.Errorf("invalid request reached provider")
	})
	before, err := ledger.Status(context.Background(), AccountID)
	if err != nil {
		t.Fatal(err)
	}
	w := invoke(g, cap, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("wanted schema rejection, got %d: %s", w.Code, w.Body.String())
	}
	after, err := ledger.Status(context.Background(), AccountID)
	if err != nil || calls != 0 || after.Reserved != before.Reserved || after.Settled != before.Settled || after.Unresolved != before.Unresolved || after.ReconciliationRequired {
		t.Fatalf("rejected request changed billing: calls=%d before=%+v after=%+v err=%v", calls, before, after, err)
	}
}

func TestAmbiguousJSONAndContentCannotReachProvider(t *testing.T) {
	cases := []struct{ name, body string }{
		{"truncated nested array", `{"model":"claude-haiku-5-5","messages":[{"content":[`},
		{"truncated nested object", `{"model":"claude-haiku-5-5","thinking":{"type":`},
		{"duplicate nested schema key", nestedAdmissionBody(`"tools":[{"name":"x","input_schema":{"properties":{"limit":{"type":"integer","type":"string"}}}}]`)},
		{"duplicate nested input key", strings.Replace(plainRequest, `"content":"hello"`, `"content":[{"type":"tool_use","id":"x","name":"x","input":{"cost":1,"cost":100}}]`, 1)},
		{"second JSON document", plainRequest + `{}`},
		{"null root", `null`},
		{"array root", `[]`},
		{"max tokens missing", strings.Replace(plainRequest, `"max_tokens":10,`, "", 1)},
		{"max tokens zero", strings.Replace(plainRequest, `"max_tokens":10`, `"max_tokens":0`, 1)},
		{"max tokens over supported bound", strings.Replace(plainRequest, `"max_tokens":10`, `"max_tokens":128001`, 1)},
		{"messages must be array", strings.Replace(plainRequest, `[{"role":"user","content":"hello"}]`, `{}`, 1)},
		{"messages cannot be empty", strings.Replace(plainRequest, `[{"role":"user","content":"hello"}]`, `[]`, 1)},
		{"message must be object", strings.Replace(plainRequest, `[{"role":"user","content":"hello"}]`, `[false]`, 1)},
		{"role unreviewed", strings.Replace(plainRequest, `"role":"user"`, `"role":"developer"`, 1)},
		{"role must be string", strings.Replace(plainRequest, `"role":"user"`, `"role":1`, 1)},
		{"content must be text or blocks", strings.Replace(plainRequest, `"content":"hello"`, `"content":4`, 1)},
		{"tool result cannot hide document", strings.Replace(plainRequest, `"content":"hello"`, `"content":[{"type":"tool_result","tool_use_id":"x","content":[{"type":"document","source":{"type":"url","url":"https://paid.test"}}]}]`, 1)},
		{"content cache cannot select unbounded ttl", strings.Replace(plainRequest, `"content":"hello"`, `"content":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral","ttl":"24h"}}]`, 1)},
		{"top level system cannot hide document", nestedAdmissionBody(`"system":[{"type":"document"}]`)},
		{"top level cache unknown contract", nestedAdmissionBody(`"cache_control":{"type":"persistent"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { assertRequestRejectedBeforeBilling(t, tc.body) })
	}
}

// Reviewed nested choices must still run under the ordinary reservation and
// settlement path. The mock transport checks the exact submitted contract.
func TestReviewedNestedChoicesKeepOrdinaryBilling(t *testing.T) {
	for name, feature := range map[string]string{
		"enabled thinking":    `"thinking":{"type":"enabled","budget_tokens":1024,"display":"summarized"}`,
		"adaptive thinking":   `"thinking":{"type":"adaptive","display":"omitted"}`,
		"disabled thinking":   `"thinking":{"type":"disabled"}`,
		"named tool":          `"tool_choice":{"type":"tool","name":"local","disable_parallel_tool_use":true}`,
		"automatic tools":     `"tool_choice":{"type":"auto","disable_parallel_tool_use":false}`,
		"metadata at bound":   `"metadata":{"user_id":"` + strings.Repeat("u", 4096) + `"}`,
		"empty output config": `"output_config":{}`,
		"strict client tool":  `"tools":[{"name":"local","input_schema":{"type":"object"},"strict":true}]`,
	} {
		t.Run(name, func(t *testing.T) {
			body := nestedAdmissionBody(feature)
			calls := 0
			g, ledger, cap := fixture(t, func(r *http.Request) (*http.Response, error) {
				calls++
				raw, err := io.ReadAll(r.Body)
				if err != nil || string(raw) != body {
					t.Fatalf("nested request changed before send: %s %v", raw, err)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(goodResponse))}, nil
			})
			w := invoke(g, cap, body)
			s, err := ledger.Status(context.Background(), AccountID)
			if w.Code != 200 || err != nil || calls != 1 || s.Settled != 15 || s.Reserved != 0 || s.Unresolved != 0 {
				t.Fatalf("reviewed request billing: status=%d calls=%d ledger=%+v err=%v body=%s", w.Code, calls, s, err, w.Body.String())
			}
		})
	}
}
