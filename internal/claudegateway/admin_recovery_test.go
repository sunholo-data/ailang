package claudegateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

// The domain authority owns the atomic financial rules. This seam proves only
// the authenticated operator can reach it and body identity cannot replace auth.
type recoveryAuthority struct {
	creditbudget.Authority
	conservative []creditbudget.ConservativeDebit
	external     []creditbudget.ExternalDebit
	queries      []string
	err          error
}

func (a *recoveryAuthority) ConservativeDebit(_ context.Context, d creditbudget.ConservativeDebit) error {
	a.conservative = append(a.conservative, d)
	return a.err
}
func (a *recoveryAuthority) ExternalDebit(_ context.Context, d creditbudget.ExternalDebit) error {
	a.external = append(a.external, d)
	return a.err
}
func (a *recoveryAuthority) Requests(_ context.Context, account, task string) ([]creditbudget.Request, error) {
	a.queries = append(a.queries, account+"/"+task)
	return []creditbudget.Request{{Reservation: creditbudget.Reservation{AccountID: account, TaskID: task, RequestID: "request-a", Amount: 1320000}, State: "unresolved"}}, a.err
}

func recoveryAdminFixture(t *testing.T) (*Gateway, *recoveryAuthority) {
	t.Helper()
	g, ledger, _ := fixture(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("admin recovery reached provider")
		return nil, nil
	})
	a := &recoveryAuthority{Authority: ledger}
	g.Authority = a
	g.Operators = map[string]bool{"operator": true}
	g.Coordinators = map[string]string{"coordinator": "job-sa"}
	g.Authenticate = func(_ context.Context, token string) (string, error) {
		if token == "invalid" {
			return "", errors.New("invalid identity")
		}
		return token, nil
	}
	return g, a
}
func callRecoveryAdmin(g *Gateway, actor, method, action, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/admin/credits/"+AccountID+"/"+action, strings.NewReader(body))
	if actor != "" {
		r.Header.Set("Authorization", "Bearer "+actor)
	}
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	return w
}

const conservativeRecoveryJSON = `{"AccountID":"anthropic-api-credits","GrantID":"grant-a","RequestID":"request-a","ExpectedAmount":1320000,"Evidence":"original unknown outcome"}`
const externalRecoveryJSON = `{"AccountID":"anthropic-api-credits","GrantID":"grant-a","DebitID":"diagnostic-a","Amount":1000040,"Kind":"conservative","Evidence":"retained diagnostic exposure"}`

func TestRecoveryRoutesRequireOperatorAndDeriveActor(t *testing.T) {
	g, a := recoveryAdminFixture(t)
	for _, action := range []string{"conservative-debit", "external-debit", "requests?task_id=task-a"} {
		method, body := "POST", conservativeRecoveryJSON
		if action == "external-debit" {
			body = externalRecoveryJSON
		}
		if strings.HasPrefix(action, "requests") {
			method, body = "GET", ""
		}
		for _, actor := range []string{"", "invalid", "coordinator", "job-sa"} {
			w := callRecoveryAdmin(g, actor, method, action, body)
			if w.Code != 401 && w.Code != 403 {
				t.Fatalf("%s reached operator route %s: %d", actor, action, w.Code)
			}
		}
		if w := callRecoveryAdmin(g, "operator", method, action, body); w.Code != 200 {
			t.Fatal(action, w.Code, w.Body.String())
		}
	}
	if len(a.conservative) != 1 || a.conservative[0].Operator != "operator" || a.conservative[0].ExpectedAmount != 1320000 || len(a.external) != 1 || a.external[0].Operator != "operator" || len(a.queries) != 1 || a.queries[0] != AccountID+"/task-a" {
		t.Fatalf("wrong identity or scope: %+v", a)
	}
}

func TestRecoveryStrictPayloadsNeverReachAuthority(t *testing.T) {
	g, a := recoveryAdminFixture(t)
	for _, action := range []string{"conservative-debit", "external-debit"} {
		body := conservativeRecoveryJSON
		if action == "external-debit" {
			body = externalRecoveryJSON
		}
		for _, bad := range []string{
			strings.TrimSuffix(body, "}") + `,"Operator":"forged"}`,
			strings.TrimSuffix(body, "}") + `,"untrusted_flag":true}`,
			strings.TrimSuffix(body, "}") + `,"GrantID":"other"}`,
			strings.Replace(body, AccountID, "other-account", 1),
			body + `{}`,
			`[]`,
		} {
			if w := callRecoveryAdmin(g, "operator", "POST", action, bad); w.Code != 400 {
				t.Fatal(action, w.Code, w.Body.String())
			}
		}
		if w := callRecoveryAdmin(g, "operator", "GET", action, ""); w.Code != 404 {
			t.Fatal("unsafe method accepted", w.Code)
		}
		if w := callRecoveryAdmin(g, "operator", "POST", action+"?operator=forged", body); w.Code != 400 {
			t.Fatal("unexpected mutation query accepted", w.Code)
		}
		if w := callRecoveryAdmin(g, "operator", "POST", action, strings.Repeat(" ", 16385)); w.Code != 400 {
			t.Fatal("oversized mutation accepted", w.Code)
		}
	}
	for _, query := range []string{"requests", "requests?task_id=bad/path", "requests?task_id=task-a&task_id=task-b", "requests?task_id=task-a&account=other", "requests?task_id=task-a&bad%", "requests?task_id=..", "requests?task_id=%0Aevil"} {
		if w := callRecoveryAdmin(g, "operator", "GET", query, ""); w.Code != 400 {
			t.Fatal(query, w.Code)
		}
	}
	if len(a.conservative)+len(a.external)+len(a.queries) != 0 {
		t.Fatal("invalid operation reached authority")
	}
}

func TestRecoveryDomainRefusalAndRequestInspectionFailLoudly(t *testing.T) {
	g, a := recoveryAdminFixture(t)
	a.err = errors.New("stale grant or unsafe recovery")
	for _, action := range []string{"conservative-debit", "external-debit", "requests?task_id=task-a"} {
		method, body := "POST", conservativeRecoveryJSON
		if action == "external-debit" {
			body = externalRecoveryJSON
		}
		if strings.HasPrefix(action, "requests") {
			method, body = "GET", ""
		}
		if w := callRecoveryAdmin(g, "operator", method, action, body); w.Code != 409 || !strings.Contains(w.Body.String(), "stale grant or unsafe recovery") {
			t.Fatal(action, w.Code, w.Body.String())
		}
	}
	a.err = nil
	w := callRecoveryAdmin(g, "operator", "GET", "requests?task_id=task-a", "")
	var requests []creditbudget.Request
	if err := json.Unmarshal(w.Body.Bytes(), &requests); err != nil || len(requests) != 1 || requests[0].Amount != 1320000 || requests[0].State != "unresolved" || requests[0].Actual != 0 || requests[0].UpstreamID != "" {
		t.Fatal(w.Body.String(), err)
	}
}
