package coordinator

import (
	"context"
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/creditbudget"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCloudCreditLaneFailsBeforeAdmission(t *testing.T) {
	for _, p := range []DispatchParams{
		{CreditAccount: "anthropic-api-credits", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 0},
		{CreditAccount: "anthropic-api-credits", Provider: "claude", AuthMode: "oauth", Model: "claude-haiku-5-5", MaxCostUSD: 2},
		{CreditAccount: "anthropic-api-credits", Provider: "claude", AuthMode: "apikey", Model: "opus", MaxCostUSD: 2},
		{CreditAccount: "anthropic-api-credits", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 2, APIKey: "user-key"},
	} {
		if err := ValidateCloudCreditLane(p); err == nil {
			t.Fatalf("accepted incoherent guarded lane: %+v", p)
		}
	}
	if err := ValidateCloudCreditLane(DispatchParams{Provider: "claude", AuthMode: "oauth"}); err != nil {
		t.Fatal(err)
	}
}
func TestAdmissionRequiresHTTPS(t *testing.T) {
	// Non-HTTPS endpoints never receive a capability or identity.
	_, err := AdmitCloudCreditTask(context.Background(), DispatchParams{CreditAccount: "anthropic-api-credits", CreditGatewayURL: "http://invalid"}, &http.Client{}, nil)
	if err == nil {
		t.Fatal("insecure gateway accepted")
	}
}

type admissionTransport func(*http.Request) (*http.Response, error)

func (f admissionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCreditAdmissionHasDistinctAttemptAndScopedModel(t *testing.T) {
	attempts := map[string]bool{}
	p := DispatchParams{TaskID: "task", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 2, CreditAccount: "anthropic-api-credits", CreditGatewayURL: "https://gateway", CreditJobIdentity: "job@example.com"}
	c := &http.Client{Transport: admissionTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Serverless-Authorization") != "Bearer identity" || r.Header.Get("Authorization") != "Bearer identity" {
			t.Fatal("identity headers missing")
		}
		var task creditbudget.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Fatal(err)
		}
		if task.Ceiling != 2000000 || task.JobIdentity != p.CreditJobIdentity || len(task.Models) != 1 || task.Models[0] != p.Model || task.AttemptID == "" || attempts[task.AttemptID] {
			t.Fatal(task)
		}
		attempts[task.AttemptID] = true
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"capability":"scoped","gateway_url":"https://gateway","account_id":"anthropic-api-credits"}`))}, nil
	})}
	for i := 0; i < 2; i++ {
		if _, err := AdmitCloudCreditTask(context.Background(), p, c, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreditReleaseIsScopedAndAuthenticated(t *testing.T) {
	c := &http.Client{Transport: admissionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/tasks/release" || r.Method != http.MethodPost || r.Header.Get("x-api-key") != "signed-capability" || r.Header.Get("Authorization") != "Bearer identity" || r.Header.Get("X-Serverless-Authorization") != "Bearer identity" {
			t.Fatal("invalid scoped release transport")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "{}" {
			t.Fatal("release must derive the attempt from its signed capability")
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	err := ReleaseCloudCreditTask(context.Background(), CloudCreditCapability{AccountID: "anthropic-api-credits", GatewayURL: "https://gateway", Capability: "signed-capability"}, c, func(context.Context, string) (string, error) { return "identity", nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreditTaskBudgetNeverRoundsUp(t *testing.T) {
	p := DispatchParams{TaskID: "task", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 0.0000001, CreditAccount: "anthropic-api-credits", CreditGatewayURL: "https://gateway", CreditJobIdentity: "job@example.com"}
	if err := ValidateCloudCreditLane(p); err == nil {
		t.Fatal("sub-micro task ceiling accepted")
	}
	p.MaxCostUSD = 0.0000019
	c := &http.Client{Transport: admissionTransport(func(r *http.Request) (*http.Response, error) {
		var task creditbudget.Task
		json.NewDecoder(r.Body).Decode(&task)
		if task.Ceiling != 1 {
			t.Fatalf("task ceiling rounded upward: %d", task.Ceiling)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"capability":"scoped","gateway_url":"https://gateway","account_id":"anthropic-api-credits"}`))}, nil
	})}
	if _, err := AdmitCloudCreditTask(context.Background(), p, c, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
}
