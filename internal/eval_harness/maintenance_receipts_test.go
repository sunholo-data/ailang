package eval_harness

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestMaintenanceReceiptsDoNotInventCoverageOrMoney(t *testing.T) {
	zero, out := 0, 9
	amount := "0.000000123456789"
	r := &executor.Result{CostUSD: 1.23456789, InputTokens: 123, UsageEvidence: &executor.UsageEvidence{
		Source: "pi-ndjson", Version: "settled-message-usage/1", StreamDigest: strings.Repeat("a", 64),
		Observations: []executor.UsageObservation{{ImportID: "message-1", EventDigest: strings.Repeat("b", 64), Input: &zero, Output: &out, Contract: "exclusive/1", OutputScheme: "combined", CostAmount: &amount, CostBasis: "list_price"}},
	}}
	x, err := MaintenanceUsageReceipts("attempt", strings.Repeat("c", 64), "model", r)
	if err != nil {
		t.Fatal(err)
	}
	if len(x) != 2 || x[0].ChildrenComplete || x[1].Tokens.Input == nil || *x[1].Tokens.Input != 0 || x[1].Tokens.Reasoning != nil {
		t.Fatalf("invented presence/coverage: %+v", x)
	}
	if *x[1].Cost.Amount != amount || x[1].Cost.Basis != "list_price" {
		t.Fatalf("rounded/relabelled money: %+v", x[1].Cost)
	}
	b, _ := json.Marshal(x)
	if strings.Contains(string(b), "1.23456789") {
		t.Fatal("aggregate float became exact receipt")
	}
}

func TestMaintenanceAggregateOnlyHasUncheckedSummary(t *testing.T) {
	x, err := MaintenanceUsageReceipts("attempt", strings.Repeat("c", 64), "model", &executor.Result{InputTokens: 20, CostUSD: 0.0})
	if err != nil {
		t.Fatal(err)
	}
	if len(x) != 1 || x[0].Role != "summary" || x[0].ChildrenComplete || x[0].Tokens.Input != nil || x[0].Cost.Amount != nil {
		t.Fatalf("fabricated receipt: %+v", x)
	}
}
