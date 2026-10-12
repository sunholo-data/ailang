package coordinator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/creditbudget"
	"gopkg.in/yaml.v3"
)

func TestCreditRegistryBudgetOverridesLegacyProviderBudget(t *testing.T) {
	var agent AgentConfig
	if err := yaml.Unmarshal([]byte(`id: claude-credit-canary
provider: claude
auth_mode: apikey
model: claude-haiku-5-5
credit_account: anthropic-api-credits
credit_gateway_url: https://gateway
credit_job_identity: guarded@example.iam.gserviceaccount.com
credit_max_cost_usd: 2
`), &agent); err != nil {
		t.Fatal(err)
	}
	// The production provider budget is intentionally larger for legacy lanes.
	p := DispatchParams{TaskID: "canary", Provider: agent.Provider, AuthMode: agent.AuthMode, Model: agent.Model, MaxCostUSD: 150}
	applyCloudCreditAgentConfig(&p, &agent)
	client := &http.Client{Transport: admissionTransport(func(r *http.Request) (*http.Response, error) {
		var task creditbudget.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Fatal(err)
		}
		if task.Ceiling != 2_000_000 || task.JobIdentity != agent.CreditJobIdentity {
			t.Fatalf("registry budget/identity not admitted: %+v", task)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"capability":"scoped","gateway_url":"https://gateway","account_id":"anthropic-api-credits"}`))}, nil
	})}
	if _, err := AdmitCloudCreditTask(context.Background(), p, client, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
}

func TestCreditRegistryBudgetNeverInheritsInvalidOrMissingLimit(t *testing.T) {
	for _, budget := range []string{"", "credit_max_cost_usd: 0", "credit_max_cost_usd: -1", "credit_max_cost_usd: 2.01", "credit_max_cost_usd: .nan", "credit_max_cost_usd: .inf"} {
		t.Run(budget, func(t *testing.T) {
			var agent AgentConfig
			if err := yaml.Unmarshal([]byte("credit_account: anthropic-api-credits\ncredit_gateway_url: https://gateway\ncredit_job_identity: guarded@example.iam.gserviceaccount.com\n"+budget), &agent); err != nil {
				t.Fatal(err)
			}
			p := DispatchParams{TaskID: "canary", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 1.5}
			applyCloudCreditAgentConfig(&p, &agent)
			sends := 0
			client := &http.Client{Transport: admissionTransport(func(*http.Request) (*http.Response, error) { sends++; return nil, nil })}
			_, err := AdmitCloudCreditTask(context.Background(), p, client, func(context.Context, string) (string, error) {
				t.Fatal("invalid budget reached token acquisition")
				return "", nil
			})
			if err == nil || sends != 0 {
				t.Fatalf("invalid/missing registry budget authorized admission: %v, sends %d", err, sends)
			}
		})
	}
}

func TestCreditRegistryBudgetLeavesLegacyLanesUnchanged(t *testing.T) {
	for _, mode := range []string{"oauth", "apikey"} {
		var agent AgentConfig
		if err := yaml.Unmarshal([]byte("auth_mode: "+mode+"\ncredit_max_cost_usd: 2"), &agent); err != nil {
			t.Fatal(err)
		}
		p := DispatchParams{Provider: "claude", AuthMode: mode, MaxCostUSD: 150, APIKey: "existing-request-key"}
		applyCloudCreditAgentConfig(&p, &agent)
		if p.MaxCostUSD != 150 || p.APIKey != "existing-request-key" || p.AuthMode != mode {
			t.Fatalf("legacy lane changed: %+v", p)
		}
	}
}
