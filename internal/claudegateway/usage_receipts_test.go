package claudegateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

const receiptUsageJSON = `{"input_tokens":100,"output_tokens":10,"cache_read_input_tokens":11,"cache_creation_input_tokens":12,"cache_creation":{"ephemeral_5m_input_tokens":5,"ephemeral_1h_input_tokens":7}}`

func receiptStream(complete bool) string {
	body := "event: message_start\ndata: " + `{"type":"message_start","message":{"id":"msg-receipt","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":0,"cache_read_input_tokens":11,"cache_creation_input_tokens":12,"cache_creation":{"ephemeral_5m_input_tokens":5,"ephemeral_1h_input_tokens":7}}}}` + "\n\n" +
		"event: content_block_start\ndata: " + `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"provider-content-private"}}` + "\n\n" +
		"event: content_block_stop\ndata: " + `{"type":"content_block_stop","index":0}` + "\n\n" +
		"event: message_delta\ndata: " + `{"type":"message_delta","usage":{"output_tokens":10}}` + "\n\n"
	if complete {
		body += "event: message_stop\ndata: " + `{"type":"message_stop"}` + "\n\n"
	}
	return body
}

func TestUsageReceiptRequiresDurableSettlementAndPreservesVerifiedCounts(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "message", true: "stream"}[stream], func(t *testing.T) {
			response := `{"id":"msg-receipt","type":"message","model":"claude-haiku-5-5","content":[{"type":"text","text":"provider-content-private"}],"usage":` + receiptUsageJSON + `}`
			ctype := "application/json"
			if stream {
				response, ctype = receiptStream(true), "text/event-stream"
			}
			g, ledger, cap := fixture(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {ctype}, "Request-Id": {"provider-receipt"}, "X-Private": {"header-private"}}, Body: io.NopCloser(strings.NewReader(response))}, nil
			})
			wantUsage := modelreg.ClaudeUsage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 11, CacheWrite5mTokens: 5, CacheWrite1hTokens: 7}
			_, revision, err := modelreg.ReserveClaudeRequest("claude-haiku-5-5", 10)
			if err != nil {
				t.Fatal(err)
			}
			cost, err := modelreg.ClaudeUsageCostAtRevision("claude-haiku-5-5", wantUsage, revision)
			if err != nil {
				t.Fatal(err)
			}
			var receipts []UsageReceipt
			g.Receipt = func(receipt UsageReceipt) {
				s, err := ledger.Status(context.Background(), AccountID)
				if err != nil || s.Reserved != 0 || int64(s.Settled) != cost {
					t.Fatalf("receipt before durable settlement: %+v %v", s, err)
				}
				receipts = append(receipts, receipt)
			}
			body := strings.Replace(plainRequest, "hello", "prompt-private", 1)
			if stream {
				body = strings.TrimSuffix(body, "}") + `,"stream":true}`
			}
			w := invoke(g, cap, body)
			if w.Code != 200 || len(receipts) != 1 {
				t.Fatalf("missing success receipt: status=%d receipts=%+v body=%s", w.Code, receipts, w.Body.String())
			}
			r := receipts[0]
			if r.Event != "credit_usage_settled" || r.AccountID != AccountID || r.TaskID != "task-a" || r.RequestID != w.Header().Get("X-Ailang-Credit-Request") || r.RequestID == "" || r.ProviderRequestID != "provider-receipt" || r.MessageID != "msg-receipt" || r.Model != "claude-haiku-5-5" || r.PricingRevision != revision || r.CostMicroUSD != cost || r.Usage != wantUsage {
				t.Fatalf("receipt lost verified metadata: %+v", r)
			}
			encoded, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"prompt-private", "provider-content-private", "header-private", "real-provider-test-key", "job-token", cap, "usage_evidence", "last_delta"} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatalf("receipt contains private/raw data %q: %s", forbidden, encoded)
				}
			}
		})
	}
}

func TestUnverifiedOrUncommittedResponsesNeverProduceSuccessReceipt(t *testing.T) {
	for _, scenario := range []string{"settlement failure", "truncated stream", "invalid usage", "unpriceable totals"} {
		t.Run(scenario, func(t *testing.T) {
			response, ctype := goodResponse, "application/json"
			if scenario == "truncated stream" {
				response, ctype = receiptStream(false), "text/event-stream"
			}
			if scenario == "invalid usage" {
				response = strings.Replace(goodResponse, `"input_tokens":100`, `"input_tokens":"invalid"`, 1)
			}
			if scenario == "unpriceable totals" {
				response = strings.Replace(goodResponse, `"input_tokens":100`, `"input_tokens":1000001`, 1)
			}
			g, ledger, cap := fixture(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {ctype}}, Body: io.NopCloser(strings.NewReader(response))}, nil
			})
			if scenario == "settlement failure" {
				g.Authority = faultAuthority{Authority: ledger, settleFail: true}
			}
			receipts, failures := 0, 0
			g.Receipt = func(UsageReceipt) { receipts++ }
			g.Diagnostic = func(UsageDiagnostic) { failures++ }
			body := plainRequest
			if scenario == "truncated stream" {
				body = strings.TrimSuffix(body, "}") + `,"stream":true}`
			}
			invoke(g, cap, body)
			s, err := ledger.Status(context.Background(), AccountID)
			if receipts != 0 || failures != 1 || err != nil || s.Settled != 0 || s.Reserved <= 0 || s.Unresolved != s.Reserved {
				t.Fatalf("false receipt or changed failure accounting: receipts=%d diagnostics=%d status=%+v err=%v", receipts, failures, s, err)
			}
		})
	}
}

func TestDefaultUsageReceiptLogIsTypedAndSanitizesProviderIDs(t *testing.T) {
	var output bytes.Buffer
	oldOutput, oldFlags, oldPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() { log.SetOutput(oldOutput); log.SetFlags(oldFlags); log.SetPrefix(oldPrefix) }()
	g, _, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "Request-Id": {"header-private/token"}}, Body: io.NopCloser(strings.NewReader(strings.Replace(goodResponse, "msg-test", "provider-content-private/token", 1)))}, nil
	})
	w := invoke(g, cap, plainRequest)
	var receipt UsageReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil || w.Code != 200 || receipt.Event != "credit_usage_settled" || receipt.ProviderRequestID != "invalid" || receipt.MessageID != "invalid" {
		t.Fatalf("unsafe/unstructured receipt: %s status=%d err=%v", output.String(), w.Code, err)
	}
	for _, private := range []string{"header-private", "provider-content-private", "real-provider-test-key", "job-token", cap} {
		if strings.Contains(output.String(), private) {
			t.Fatalf("receipt logs private data %q", private)
		}
	}
}
