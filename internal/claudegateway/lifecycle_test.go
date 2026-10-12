package claudegateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func fixture(t *testing.T, transport roundTripFunc) (*Gateway, *creditbudget.Engine, string) {
	t.Helper()
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ctx := context.Background()
	a := creditbudget.New(creditbudget.NewMemoryStore(), func() time.Time { return now })
	if err := a.Configure(ctx, creditbudget.Policy{AccountID: AccountID, Organization: "test-org", Workspace: "test-workspace", OperatingCeiling: 190 * creditbudget.USD, DailyCeiling: 6 * creditbudget.USD, TaskCeiling: 2 * creditbudget.USD, MaxTasks: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Confirm(ctx, creditbudget.Confirmation{AccountID: AccountID, GrantID: "cycle-2026-10", Amount: 200 * creditbudget.USD, Available: 200 * creditbudget.USD, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(48 * time.Hour), Operator: "operator", Evidence: "verified grant"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetEnabled(ctx, AccountID, true, "operator", "reviewed test"); err != nil {
		t.Fatal(err)
	}
	until := now.Add(50 * time.Minute)
	if err := a.AdmitTask(ctx, creditbudget.Task{AccountID: AccountID, ID: "task-a", AttemptID: "attempt-a", JobIdentity: "job-sa", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * creditbudget.USD, LeaseUntil: until}); err != nil {
		t.Fatal(err)
	}
	key := []byte(strings.Repeat("k", 32))
	cap, err := SignCapability(key, Capability{AccountID: AccountID, TaskID: "task-a", AttemptID: "attempt-a", JobIdentity: "job-sa", ExpiresAt: until})
	if err != nil {
		t.Fatal(err)
	}
	g := &Gateway{Authority: a, ProviderKey: "real-provider-test-key", SigningKey: key, Client: &http.Client{Transport: transport}, Now: func() time.Time { return now }, Authenticate: func(_ context.Context, token string) (string, error) {
		if token == "job-token" {
			return "job-sa", nil
		}
		return "", errors.New("bad token")
	}}
	return g, a, cap
}
func invoke(g *Gateway, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "https://gateway.test/v1/messages?beta=true", strings.NewReader(body))
	r.Header.Set("x-api-key", token)
	r.Header.Set("Authorization", "Bearer job-token")
	r.Header.Set("anthropic-version", "2023-06-01")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	return w
}

const plainRequest = `{"model":"claude-haiku-5-5","max_tokens":10,"messages":[{"role":"user","content":"hello"}]}`
const goodResponse = `{"id":"msg-test","type":"message","model":"claude-haiku-5-5","usage":{"input_tokens":100,"output_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`

func TestEverySendHasReservationAndOnlyGatewayKey(t *testing.T) {
	var a *creditbudget.Engine
	calls := 0
	g, ledger, cap := fixture(t, func(r *http.Request) (*http.Response, error) {
		calls++
		s, err := a.Status(context.Background(), AccountID)
		if err != nil || s.Reserved <= 0 {
			t.Fatalf("send without reservation: %+v %v", s, err)
		}
		if r.URL.String() != upstreamURL || r.Header.Get("x-api-key") != "real-provider-test-key" || r.Header.Get("Authorization") != "" {
			t.Fatal("origin/credential isolation violated")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(goodResponse))}, nil
	})
	a = ledger
	for i := 0; i < 2; i++ {
		w := invoke(g, cap, plainRequest)
		if w.Code != 200 {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
	s, err := a.Status(context.Background(), AccountID)
	if err != nil || s.Reserved != 0 || s.Settled != 30 || calls != 2 {
		t.Fatalf("settlement %+v calls%d err%v", s, calls, err)
	}
}
func TestAmbiguousOutcomeRetainsExposureAndBlocksNextSend(t *testing.T) {
	for _, scenario := range []string{"transport", "bad-status", "bad-usage", "truncate", "redirect"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
				calls++
				if scenario == "transport" {
					return nil, errors.New("connection lost")
				}
				status := 200
				content := goodResponse
				ctype := "application/json"
				header := http.Header{}
				switch scenario {
				case "bad-status":
					status = 429
				case "redirect":
					status = 307
					header.Set("Location", "https://evil.test")
				case "bad-usage":
					content = strings.ReplaceAll(goodResponse, `"output_tokens":10`, `"output_tokens":999`)
				case "truncate":
					ctype = "text/event-stream"
					content = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg-test\",\"model\":\"claude-haiku-5-5\",\"usage\":{\"input_tokens\":100,\"output_tokens\":0}}}\n\n"
				}
				header.Set("Content-Type", ctype)
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(content))}, nil
			})
			body := plainRequest
			if scenario == "truncate" {
				body = strings.TrimSuffix(body, "}") + `,"stream":true}`
			}
			invoke(g, cap, body)
			s, err := a.Status(context.Background(), AccountID)
			if err != nil || s.Reserved <= 0 || s.Unresolved != s.Reserved || !s.ReconciliationRequired {
				t.Fatalf("lost exposure: %+v %v", s, err)
			}
			invoke(g, cap, body)
			if calls != 1 {
				t.Fatalf("ambiguous request retried upstream %d", calls)
			}
		})
	}
}
func TestUnsupportedRequestsAndWrongIdentityHaveZeroExposure(t *testing.T) {
	g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) { t.Fatal("unsupported forwarded"); return nil, nil })
	bodies := []string{strings.TrimSuffix(plainRequest, "}") + `,"service_tier":"priority"}`, strings.ReplaceAll(plainRequest, `"model":"claude-haiku-5-5"`, `"model":"claude-opus-4-8"`), strings.TrimSuffix(plainRequest, "}") + `,"context_management":{}}`, strings.TrimSuffix(plainRequest, "}") + `,"tools":[{"type":"web_search_20250305","name":"web_search"}]}`, strings.TrimSuffix(plainRequest, "}") + `,"max_tokens":9}`}
	for _, body := range bodies {
		w := invoke(g, cap, body)
		if w.Code != 400 {
			t.Fatalf("unsupported %d %s", w.Code, w.Body.String())
		}
	}
	g.Authenticate = func(context.Context, string) (string, error) { return "wrong-sa", nil }
	if w := invoke(g, cap, plainRequest); w.Code != 403 {
		t.Fatal(w.Code)
	}
	s, _ := a.Status(context.Background(), AccountID)
	if s.Reserved != 0 || s.Settled != 0 {
		t.Fatalf("unsupported exposure %+v", s)
	}
}
func TestStreamingRelaysPingsAndSettlesOnlyCompleteUsage(t *testing.T) {
	stream := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg-test\",\"model\":\"claude-haiku-5-5\",\"usage\":{\"input_tokens\":100,\"output_tokens\":0}}}\n\nevent: ping\ndata: {\"type\":\"ping\"}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":10}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}, nil
	})
	w := invoke(g, cap, strings.TrimSuffix(plainRequest, "}")+`,"stream":true}`)
	if w.Body.String() != stream || !w.Flushed {
		t.Fatal("stream changed or buffered")
	}
	s, _ := a.Status(context.Background(), AccountID)
	if s.Reserved != 0 || s.Settled != 15 {
		t.Fatalf("settlement %+v", s)
	}
	duplicate := strings.Replace(stream, "event: message_stop", "event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":10}}\n\nevent: message_stop", 1)
	du, _, derr := relayStream(httptest.NewRecorder(), strings.NewReader(duplicate), "claude-haiku-5-5", 10)
	if derr != nil || du.OutputTokens != 10 {
		t.Fatalf("cumulative duplicate charged twice: %+v %v", du, derr)
	}
	for _, bad := range []string{strings.ReplaceAll(stream, `"output_tokens":10`, `"output_tokens":-1`), strings.ReplaceAll(stream, "event: message_stop", "event: message_delta"), strings.TrimSuffix(stream, "\n\n")} {
		_, _, err := relayStream(httptest.NewRecorder(), strings.NewReader(bad), "claude-haiku-5-5", 10)
		if err == nil {
			t.Fatal("invalid stream usage settled")
		}
	}
}
