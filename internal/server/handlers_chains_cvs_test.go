package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// cvsStage is a compact banked-stage spec for the HTTP KPI tests.
type cvsStage struct {
	cost      float64
	tokensIn  int
	tokensOut int
	model     string
	compile   bool
	runtime   bool
	stdout    bool
	verifyOk  bool
	verified  int
	counterex int
	skipped   int
	errors    int
}

// newKPITestServer builds a server backed by an in-memory observatory with a
// banked cohort under sourceRef, and returns it plus the same *Store so a test
// can also invoke the rollup directly (to prove HTTP == direct-rollup identity).
func newKPITestServer(t *testing.T, sourceRef string, stages []cvsStage) (*Server, *observatory.Store) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := observatory.MigrateWithVersion(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	backend, err := observatory.NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	store := backend.Store()

	ctx := context.Background()
	chain, err := store.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType: observatory.ChainSourceEvalSuite,
		SourceRef:  sourceRef,
	})
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	for i, s := range stages {
		stage, err := store.CreateStage(ctx, &observatory.StageCreateRequest{ChainID: chain.ID, AgentID: "eval-agent"})
		if err != nil {
			t.Fatalf("create stage %d: %v", i, err)
		}
		model := s.model
		if model == "" {
			model = "claude-sonnet-4-5"
		}
		a := &observatory.EvalAssessment{
			BenchmarkID: "b", Model: model, Language: "ailang", EvalMode: "agent",
			CompileOk: s.compile, RuntimeOk: s.runtime, StdoutOk: s.stdout,
			VerifyOk: s.verifyOk, VerifyVerified: s.verified,
			VerifyCounterex: s.counterex, VerifySkipped: s.skipped, VerifyErrors: s.errors,
		}
		if err := store.UpdateStageEvalAssessment(ctx, stage.ID, a); err != nil {
			t.Fatalf("bank assessment %d: %v", i, err)
		}
		if err := store.UpdateStageMetrics(ctx, stage.ID, s.cost, s.tokensIn, s.tokensOut, 0, 0, 0, ""); err != nil {
			t.Fatalf("bank metrics %d: %v", i, err)
		}
	}

	return &Server{obsBackend: backend}, store
}

func verifiedSuccessStage(cost float64) cvsStage {
	return cvsStage{cost: cost, compile: true, runtime: true, stdout: true, verifyOk: true, verified: 2}
}

// TestHTTP_CostPerVerifiedSuccess_MatchesDirectRollup proves the HTTP handler
// returns exactly what the observatory rollup computes (field-for-field).
func TestHTTP_CostPerVerifiedSuccess_MatchesDirectRollup(t *testing.T) {
	srv, store := newKPITestServer(t, "v1.0/agent/baseline", []cvsStage{
		verifiedSuccessStage(0.10),
		verifiedSuccessStage(0.20),
		{cost: 0.90, compile: false}, // failed paid run — cost counts, not a success
	})

	// Direct rollup (the CLI serializes this same struct).
	want, err := store.CostPerVerifiedSuccess(context.Background(), observatory.CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("direct rollup: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chains/stats?cost_per_verified_success=true&baseline=v1.0", nil)
	rec := httptest.NewRecorder()
	srv.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got observatory.CostPerVerifiedSuccessResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode http body: %v (%s)", err, rec.Body.String())
	}

	// Compare on the stable fields (GeneratedAt differs by wall-clock).
	if got.Available != want.Available ||
		got.VerifiedSuccesses != want.VerifiedSuccesses ||
		got.TotalRuns != want.TotalRuns ||
		got.KnownCostUSD != want.KnownCostUSD ||
		got.CostPerVerifiedSuccessUSD != want.CostPerVerifiedSuccessUSD ||
		got.ReportedCostUSD != want.ReportedCostUSD ||
		got.EstimatedCostUSD != want.EstimatedCostUSD ||
		got.QuotaStages != want.QuotaStages ||
		got.UnknownStages != want.UnknownStages ||
		got.IncompleteData != want.IncompleteData {
		t.Errorf("HTTP result differs from direct rollup:\n http:   %+v\n direct: %+v", got, want)
	}

	// Sanity: numerator includes the failed run (0.10+0.20+0.90=1.20 / 2 = 0.60).
	if math.Abs(got.KnownCostUSD-1.20) > 1e-9 || math.Abs(got.CostPerVerifiedSuccessUSD-0.60) > 1e-9 {
		t.Errorf("expected known=1.20 kpi=0.60, got known=%v kpi=%v", got.KnownCostUSD, got.CostPerVerifiedSuccessUSD)
	}
}

// TestHTTP_CostPerVerifiedSuccess_UnknownCostUnavailable proves an unknown-cost
// cohort is served as available=false (HTTP 200, Incomplete state) not a silent $0.
func TestHTTP_CostPerVerifiedSuccess_UnknownCostUnavailable(t *testing.T) {
	srv, _ := newKPITestServer(t, "v1.0/agent/baseline", []cvsStage{
		verifiedSuccessStage(0.10),
		// token-bearing, unresolvable model, no cost => unknown
		{cost: 0, tokensIn: 500, tokensOut: 500, model: "not-a-real-model-xyz",
			compile: true, runtime: true, stdout: true, verifyOk: true, verified: 1},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/chains/stats?cost_per_verified_success=true&baseline=v1.0", nil)
	rec := httptest.NewRecorder()
	srv.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got observatory.CostPerVerifiedSuccessResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Available {
		t.Error("expected available=false for unknown cost")
	}
	if got.Reason != observatory.CVSReasonUnknownCost {
		t.Errorf("expected reason unknown_cost, got %q", got.Reason)
	}
}

// TestHTTP_CostPerVerifiedSuccess_ZeroDenominatorUnavailable proves a cohort with
// paid runs but no verified success is unavailable (never divide-by-zero / $0).
func TestHTTP_CostPerVerifiedSuccess_ZeroDenominatorUnavailable(t *testing.T) {
	srv, _ := newKPITestServer(t, "v1.0/agent/baseline", []cvsStage{
		{cost: 0.30, compile: true, runtime: true, stdout: true}, // unverified pass
	})
	req := httptest.NewRequest(http.MethodGet, "/api/chains/stats?cost_per_verified_success=true&baseline=v1.0", nil)
	rec := httptest.NewRecorder()
	srv.handleChainsStats(rec, req)

	var got observatory.CostPerVerifiedSuccessResult
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Available {
		t.Error("expected available=false for zero denominator")
	}
	if got.Reason != observatory.CVSReasonZeroDenominator {
		t.Errorf("expected reason zero_denominator, got %q", got.Reason)
	}
}

// TestHTTP_CostPerVerifiedSuccess_MissingBaselineIsBadRequest proves the baseline
// param is required (no silent default cohort).
func TestHTTP_CostPerVerifiedSuccess_MissingBaselineIsBadRequest(t *testing.T) {
	srv, _ := newKPITestServer(t, "v1.0/agent/baseline", []cvsStage{verifiedSuccessStage(0.10)})
	req := httptest.NewRequest(http.MethodGet, "/api/chains/stats?cost_per_verified_success=true", nil)
	rec := httptest.NewRecorder()
	srv.handleChainsStats(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing baseline, got %d", rec.Code)
	}
}

// TestHTTP_ChainsStats_UnchangedWithoutKpiParam proves the existing stats
// response is unregressed when the KPI param is absent.
func TestHTTP_ChainsStats_UnchangedWithoutKpiParam(t *testing.T) {
	srv, _ := newKPITestServer(t, "v1.0/agent/baseline", []cvsStage{verifiedSuccessStage(0.10)})
	req := httptest.NewRequest(http.MethodGet, "/api/chains/stats", nil)
	rec := httptest.NewRecorder()
	srv.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Existing keys present; the KPI object is NOT injected into the default response.
	if _, ok := got["total_chains"]; !ok {
		t.Error("expected existing total_chains field")
	}
	if _, ok := got["cost_per_verified_success_usd"]; ok {
		t.Error("KPI must not leak into the default stats response")
	}
}
