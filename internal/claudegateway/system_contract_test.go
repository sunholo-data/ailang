package claudegateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnverifiedCLIBetasStillNeverReserveOrForward(t *testing.T) {
	g, ledger, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unverified CLI beta forwarded")
		return nil, nil
	})
	for _, beta := range []string{"interleaved-thinking-2025-05-14", "mid-conversation-system-2026-04-07", "claude-code-20250219", "effort-2025-11-24"} {
		r := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(plainRequest))
		r.Header.Set("x-api-key", cap)
		r.Header.Set("Authorization", "Bearer job-token")
		r.Header.Set("anthropic-version", "2023-06-01")
		r.Header.Set("anthropic-beta", beta)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("unverified beta %s accepted: %d", beta, w.Code)
		}
	}
	s, err := ledger.Status(context.Background(), AccountID)
	if err != nil || s.Reserved != 0 || s.Settled != 0 {
		t.Fatalf("unverified beta changed spending: %+v %v", s, err)
	}
}

func TestMidConversationSystemUsesExistingReservation(t *testing.T) {
	for _, content := range []string{`"Answer briefly"`, `[{"type":"text","text":"Answer briefly","cache_control":{"type":"ephemeral","ttl":"1h"}}]`} {
		t.Run(content, func(t *testing.T) {
			calls := 0
			g, ledger, cap := fixture(t, func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(goodResponse))}, nil
			})
			body := strings.Replace(plainRequest, `"content":"hello"}]`, `"content":"hello"},{"role":"system","content":`+content+`}]`, 1)
			w := invoke(g, cap, body)
			if w.Code != 200 {
				t.Fatalf("standard system input rejected: %d %s", w.Code, w.Body.String())
			}
			s, err := ledger.Status(context.Background(), AccountID)
			if err != nil || calls != 1 || s.Settled != 15 || s.Reserved != 0 {
				t.Fatalf("system request accounting: %+v calls=%d err=%v", s, calls, err)
			}
		})
	}
}

func TestSystemMessagesCannotEnableUnreviewedFeatures(t *testing.T) {
	g, ledger, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unsupported system feature forwarded")
		return nil, nil
	})
	for _, message := range []string{
		`{"role":"system","content":"text","clear_at":"next_user_message"}`,
		`{"role":"system","content":"text","output_config":{"effort":"high"}}`,
		`{"role":"system","content":[{"type":"tool_addition","tool":{"name":"paid"}}]}`,
		`{"role":"system","content":[{"type":"tool_use","id":"x","name":"x","input":{}}]}`,
		`{"role":"system","content":[{"type":"thinking","thinking":"x","signature":"x"}]}`,
	} {
		body := strings.Replace(plainRequest, `"content":"hello"}]`, `"content":"hello"},`+message+`]`, 1)
		if w := invoke(g, cap, body); w.Code != 400 {
			t.Fatalf("unsupported system request accepted: %s status=%d", message, w.Code)
		}
	}
	s, err := ledger.Status(context.Background(), AccountID)
	if err != nil || s.Reserved != 0 || s.Settled != 0 {
		t.Fatalf("unsupported system feature changed spending: %+v %v", s, err)
	}
}
