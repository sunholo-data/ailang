package eval_analysis

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// sampleKPI is an available result with representative field values.
func sampleKPI() *observatory.CostPerVerifiedSuccessResult {
	return &observatory.CostPerVerifiedSuccessResult{
		BaselineID:                "v1.0",
		Language:                  "ailang",
		EvalMode:                  "agent",
		SourceRef:                 "v1.0/",
		TotalRuns:                 4,
		PassedRuns:                3,
		VerifiedSuccesses:         2,
		UnverifiedPasses:          1,
		VerificationFailures:      0,
		ReportedCostUSD:           0.30,
		EstimatedCostUSD:          0.10,
		KnownCostUSD:              0.40,
		QuotaStages:               1,
		UnknownStages:             0,
		IncompleteData:            false,
		CostPerVerifiedSuccessUSD: 0.20,
		Available:                 true,
	}
}

// TestHeadlineKpiObject_FieldForFieldIdentical proves the published headline
// object is byte-identical to a direct JSON serialization of the canonical
// struct — i.e. the publisher and the CLI/HTTP surfaces emit the same shape.
func TestHeadlineKpiObject_FieldForFieldIdentical(t *testing.T) {
	res := sampleKPI()

	// What the CLI (--json) and the HTTP handler emit: the struct, verbatim.
	direct, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal struct: %v", err)
	}

	// What the publisher nests under headlineKpis.costPerVerifiedSuccess.
	obj, err := headlineKpiObject(res)
	if err != nil {
		t.Fatalf("headlineKpiObject: %v", err)
	}
	published, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal published obj: %v", err)
	}

	// Compare as normalized generic JSON (map key order is irrelevant).
	var a, b interface{}
	if err := json.Unmarshal(direct, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(published, &b); err != nil {
		t.Fatal(err)
	}
	da, _ := json.Marshal(a)
	db, _ := json.Marshal(b)
	if string(da) != string(db) {
		t.Errorf("published headline object differs from canonical struct:\n CLI/HTTP: %s\n published: %s", da, db)
	}
}

// TestAttachHeadlineKpi_AdditiveDoesNotMutateExisting proves the headline object
// is additive: attaching it leaves models/agentModels/aggregates/ratings byte-stable.
func TestAttachHeadlineKpi_AdditiveDoesNotMutateExisting(t *testing.T) {
	dashboard := &DashboardJSON{
		Version:   "v0.30.0",
		Timestamp: "2026-07-25T12:53:29+02:00",
		TotalRuns: 1810,
		Aggregates: map[string]interface{}{
			"finalSuccess": 0.808839779,
			"totalCostUSD": 98.32670983,
		},
		Models:      map[string]interface{}{"claude-sonnet-4-5": map[string]interface{}{"costPerSuccess": 0.12}},
		AgentModels: map[string]interface{}{"claude-sonnet-4-5": map[string]interface{}{"runs": 100}},
		Ratings:     map[string]interface{}{"agent": map[string]interface{}{"elo": 1500}},
		Benchmarks:  map[string]interface{}{},
		Languages:   map[string]interface{}{},
		Executors:   map[string]interface{}{},
		History:     []HistoryEntry{{Version: "v0.30.0", Timestamp: "2026-07-25T12:53:29+02:00"}},
	}

	// Snapshot the pre-change serialization of each preserved block.
	before := map[string]string{}
	for _, k := range []string{"models", "agentModels", "aggregates", "ratings"} {
		var v interface{}
		switch k {
		case "models":
			v = dashboard.Models
		case "agentModels":
			v = dashboard.AgentModels
		case "aggregates":
			v = dashboard.Aggregates
		case "ratings":
			v = dashboard.Ratings
		}
		b, _ := json.Marshal(v)
		before[k] = string(b)
	}

	if err := AttachHeadlineKpi(dashboard, sampleKPI()); err != nil {
		t.Fatalf("AttachHeadlineKpi: %v", err)
	}

	// The headline object is present...
	if dashboard.HeadlineKpis[HeadlineKpiCostPerVerifiedSuccess] == nil {
		t.Fatal("expected headlineKpis.costPerVerifiedSuccess to be set")
	}
	// ...and every preserved block is byte-stable.
	after := map[string]interface{}{
		"models":      dashboard.Models,
		"agentModels": dashboard.AgentModels,
		"aggregates":  dashboard.Aggregates,
		"ratings":     dashboard.Ratings,
	}
	for k, wantStr := range before {
		b, _ := json.Marshal(after[k])
		if string(b) != wantStr {
			t.Errorf("%s mutated by AttachHeadlineKpi:\n before: %s\n after:  %s", k, wantStr, string(b))
		}
	}

	// Validate() must still pass (schema unbroken).
	if err := dashboard.Validate(); err != nil {
		t.Errorf("dashboard invalid after attach: %v", err)
	}
}

// TestAttachHeadlineKpi_UnavailableIsPublishedNotFabricated proves an
// unavailable KPI is published with available=false (dashboard renders
// Incomplete), never fabricated into an available number.
func TestAttachHeadlineKpi_UnavailableIsPublishedNotFabricated(t *testing.T) {
	res := sampleKPI()
	res.Available = false
	res.IncompleteData = true
	res.UnknownStages = 1
	res.Reason = observatory.CVSReasonUnknownCost

	dashboard := &DashboardJSON{
		Version: "v", Timestamp: "t",
		History: []HistoryEntry{{Version: "v", Timestamp: "t"}},
	}
	if err := AttachHeadlineKpi(dashboard, res); err != nil {
		t.Fatalf("AttachHeadlineKpi: %v", err)
	}
	obj := dashboard.HeadlineKpis[HeadlineKpiCostPerVerifiedSuccess].(map[string]interface{})
	if obj["available"] != false {
		t.Errorf("expected available=false in published object, got %v", obj["available"])
	}
	if obj["reason"] != observatory.CVSReasonUnknownCost {
		t.Errorf("expected reason unknown_cost, got %v", obj["reason"])
	}
}

func TestAttachHeadlineKpi_NilResultErrors(t *testing.T) {
	dashboard := &DashboardJSON{Version: "v", Timestamp: "t", History: []HistoryEntry{{Version: "v", Timestamp: "t"}}}
	if err := AttachHeadlineKpi(dashboard, nil); err == nil {
		t.Error("expected error attaching nil result (no silent fallback)")
	}
}
