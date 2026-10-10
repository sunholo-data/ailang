package claudegateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func TestAdminIdentityAndTaskCapabilityScope(t *testing.T) {
	g, a, _ := fixture(t, nil)
	g.PublicURL = "https://gateway.test"
	g.Operators = map[string]bool{"operator": true}
	g.Coordinators = map[string]string{"coordinator": "job-sa"}
	g.Authenticate = func(_ context.Context, token string) (string, error) { return token, nil }
	call := func(actor, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+actor)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		return w
	}
	task := creditbudget.Task{AccountID: AccountID, ID: "task-b", AttemptID: "attempt-b", JobIdentity: "job-sa", Models: []string{"claude-haiku-5-5"}, Ceiling: 2 * creditbudget.USD, LeaseUntil: g.now().Add(50 * time.Minute)}
	raw, _ := json.Marshal(task)
	if w := call("operator", "/admin/tasks", string(raw)); w.Code != 403 {
		t.Fatal("operator impersonated coordinator", w.Code)
	}
	if w := call("coordinator", "/admin/credits/"+AccountID+"/enabled", `{"Enabled":true,"Evidence":"grant"}`); w.Code != 403 {
		t.Fatal("coordinator changed credit policy", w.Code)
	}
	if w := call("coordinator", "/admin/tasks", strings.ReplaceAll(string(raw), "job-sa", "other-sa")); w.Code != 400 {
		t.Fatal("arbitrary job identity admitted", w.Code)
	}
	w := call("coordinator", "/admin/tasks", string(raw))
	if w.Code != 200 {
		t.Fatalf("admit %d %s", w.Code, w.Body.String())
	}
	var reply map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	c, err := VerifyCapability(g.SigningKey, reply["capability"], g.now())
	if err != nil || c.TaskID != "task-b" || c.AttemptID != "attempt-b" || c.JobIdentity != "job-sa" {
		t.Fatalf("wrong binding %+v %v", c, err)
	}
	if w := call("operator", "/admin/credits/"+AccountID+"/enabled", `{"Enabled":false,"Evidence":"kill switch test"}`); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ := a.Status(context.Background(), AccountID)
	if s.Enabled {
		t.Fatal("kill switch ignored")
	}
	if w := call("operator", "/admin/credits/"+AccountID+"/confirm", `{"AccountID":"anthropic-api-credits","Operator":"forged"}`); w.Code != 400 {
		t.Fatal("forged actor accepted")
	}
	if w := call("operator", "/admin/credits/"+AccountID+"/enabled", `{"Enabled":true,"Enabled":false,"Evidence":"duplicate"}`); w.Code != 400 {
		t.Fatal("duplicate JSON accepted")
	}
}
func TestUnsupportedUsageCannotSettle(t *testing.T) {
	for _, raw := range []string{
		`{"input_tokens":1,"output_tokens":1,"iterations":[{}]}`,
		`{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":10}`,
		`{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":1,"ephemeral_1h_input_tokens":1}}`,
		`{"input_tokens":1,"output_tokens":1,"server_tool_use":{"web_search_requests":1}}`,
		`{"input_tokens":1,"output_tokens":1,"service_tier":"priority"}`,
		`{"input_tokens":1,"output_tokens":1,"inference_geo":"us"}`,
		`{"input_tokens":1,"output_tokens":1,"output_tokens":0}`,
	} {
		if _, err := parseUsage(json.RawMessage(raw), true); err == nil {
			t.Fatal("unpriced usage settled", raw)
		}
	}
}

func TestReleaseAttemptFreesSlotWithoutRefund(t *testing.T) {
	g, a, cap := fixture(t, nil)
	amount, revision, err := modelreg.ReserveClaudeRequest("claude-haiku-5-5", 10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Reserve(context.Background(), creditbudget.Reservation{AccountID: AccountID, TaskID: "task-a", AttemptID: "attempt-a", JobIdentity: "job-sa", Model: "claude-haiku-5-5", RequestID: "ambiguous-a", Amount: creditbudget.MicroUSD(amount), PricingRevision: revision, Deadline: g.now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err = a.MarkForwarded(context.Background(), AccountID, "ambiguous-a"); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/tasks/release", strings.NewReader("{}"))
	r.Header.Set("x-api-key", cap)
	r.Header.Set("Authorization", "Bearer job-token")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("release %d %s", w.Code, w.Body.String())
	}
	s, _ := a.Status(context.Background(), AccountID)
	if s.ActiveTasks != 0 || s.Reserved != creditbudget.MicroUSD(amount) || s.Forwarding != creditbudget.MicroUSD(amount) {
		t.Fatalf("release refunded exposure or held slot: %+v", s)
	}
}

func TestOperatorPromotionRequiresSettledCanaryAndNeverRenewsGrant(t *testing.T) {
	g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(goodResponse))}, nil
	})
	g.Operators = map[string]bool{"operator": true}
	baseAuth := g.Authenticate
	g.Authenticate = func(ctx context.Context, token string) (string, error) {
		if token == "operator-token" {
			return "operator", nil
		}
		return baseAuth(ctx, token)
	}
	promote := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/admin/credits/"+AccountID+"/promote", strings.NewReader(`{"Evidence":"provider usage and billing matched"}`))
		r.Header.Set("Authorization", "Bearer operator-token")
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		return w
	}
	if w := promote(); w.Code != 409 {
		t.Fatal("promoted without canary", w.Code)
	}
	invoke(g, cap, plainRequest)
	before, _ := a.Status(context.Background(), AccountID)
	for i := 0; i < 2; i++ {
		if w := promote(); w.Code != 200 {
			t.Fatalf("promotion %d %s", w.Code, w.Body.String())
		}
	}
	after, _ := a.Status(context.Background(), AccountID)
	if !after.CanaryComplete || after.Grant.GrantID != before.Grant.GrantID || after.Settled != before.Settled || after.Settled != 15 || after.CanarySettled != 15 {
		t.Fatalf("promotion changed allowance: %+v", after)
	}
}
