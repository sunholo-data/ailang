package claudegateway

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestUnknownNestedFeaturesNeverReserveOrForward(t *testing.T) {
	g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unknown nested feature forwarded")
		return nil, nil
	})
	for _, feature := range []string{
		`"tools":[{"name":"x","input_schema":{"type":"object"},"cache_control":{"type":"ephemeral","ttl":"10h"}}]`,
		`"thinking":{"type":"unpriced_future_mode"}`,
		`"thinking":{"type":"adaptive","future_billing":"premium"}`,
		`"output_config":{"effort":"future_premium"}`,
		`"output_config":{"unpriced_feature":true}`,
		`"tool_choice":{"type":"future_paid_tool"}`,
		`"metadata":{"future_billing":true}`,
	} {
		w := invoke(g, cap, strings.TrimSuffix(plainRequest, "}")+","+feature+"}")
		if w.Code != 400 {
			t.Fatalf("accepted %s: %d", feature, w.Code)
		}
	}
	body := strings.Replace(plainRequest, `"content":"hello"`, `"content":[{"type":"text","text":"hello","future_paid_feature":true}]`, 1)
	if w := invoke(g, cap, body); w.Code != 400 {
		t.Fatalf("unknown content field accepted: %d", w.Code)
	}
	body = strings.Replace(plainRequest, `"role":"user"`, `"role":"user","future_paid_feature":true`, 1)
	if w := invoke(g, cap, body); w.Code != 400 {
		t.Fatalf("unknown message field accepted: %d", w.Code)
	}
	status, _ := a.Status(context.Background(), AccountID)
	if status.Reserved != 0 || status.Settled != 0 {
		t.Fatal("unsupported features reserved credit")
	}

}
