package observatory

import (
	"context"
	"math"
	"testing"
)

// floatEq compares two dollar amounts with a small tolerance.
func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// ===== Cost-Per-Verified-Success KPI Tests (M-COST-PER-SUCCESS-KPI, M1) =====
//
// These tests pin the two hardest rules BEFORE any wiring:
//  1. The VERIFIED-success predicate (defined exactly once, in the rollup).
//  2. The numerator = reported+estimated via CostRollup.TotalKnownCost(),
//     including the cost of FAILED runs; unknown cost / zero denominator ⇒
//     available=false / incomplete_data=true (never $0 or a stale value).

// cvsStageSpec is a compact description of one banked eval stage for a fixture.
type cvsStageSpec struct {
	benchmarkID string
	model       string
	cost        float64 // stored self-reported cost (0 ⇒ classifier estimates or quota)
	tokensIn    int
	tokensOut   int
	compileOk   bool
	runtimeOk   bool
	stdoutOk    bool
	verifyOk    bool
	verified    int
	counterex   int
	skipped     int
	errors      int
}

// bankCohort creates one eval chain with the given source_ref and banks every
// stage spec against it, returning the source_ref for cohort filtering.
func bankCohort(t *testing.T, store *Store, sourceRef string, specs []cvsStageSpec) {
	t.Helper()
	ctx := context.Background()

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceEvalSuite,
		SourceRef:  sourceRef,
	})
	if err != nil {
		t.Fatalf("failed to create cohort chain: %v", err)
	}

	for i, s := range specs {
		stage, err := store.CreateStage(ctx, &StageCreateRequest{
			ChainID: chain.ID,
			AgentID: "eval-agent",
		})
		if err != nil {
			t.Fatalf("failed to create stage %d: %v", i, err)
		}
		assessment := &EvalAssessment{
			BenchmarkID:     s.benchmarkID,
			Model:           s.model,
			Language:        "ailang",
			EvalMode:        "agent",
			CompileOk:       s.compileOk,
			RuntimeOk:       s.runtimeOk,
			StdoutOk:        s.stdoutOk,
			VerifyOk:        s.verifyOk,
			VerifyVerified:  s.verified,
			VerifyCounterex: s.counterex,
			VerifySkipped:   s.skipped,
			VerifyErrors:    s.errors,
		}
		if err := store.UpdateStageEvalAssessment(ctx, stage.ID, assessment); err != nil {
			t.Fatalf("failed to bank assessment %d: %v", i, err)
		}
		if err := store.UpdateStageMetrics(ctx, stage.ID, s.cost, s.tokensIn, s.tokensOut, 0, 0, 0, ""); err != nil {
			t.Fatalf("failed to bank metrics %d: %v", i, err)
		}
	}
}

// aVerified is a fully-verified success spec (reported cost).
func aVerified(bench string, cost float64) cvsStageSpec {
	return cvsStageSpec{
		benchmarkID: bench, model: "claude-sonnet-4-5", cost: cost,
		compileOk: true, runtimeOk: true, stdoutOk: true,
		verifyOk: true, verified: 2, counterex: 0, skipped: 0, errors: 0,
	}
}

func TestCostPerVerifiedSuccess_VerifiedSuccessCounted(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{
		aVerified("safe_sub", 0.10),
		aVerified("safe_div", 0.20),
	})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0",
		SourceRef:  "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Available {
		t.Fatalf("expected available=true, got incomplete=%v", res.IncompleteData)
	}
	if res.VerifiedSuccesses != 2 {
		t.Errorf("expected 2 verified successes, got %d", res.VerifiedSuccesses)
	}
	if res.TotalRuns != 2 {
		t.Errorf("expected 2 total runs, got %d", res.TotalRuns)
	}
	// numerator = 0.10 + 0.20 = 0.30, denom 2 ⇒ 0.15
	if !floatEq(res.KnownCostUSD, 0.30) {
		t.Errorf("expected known cost 0.30, got %v", res.KnownCostUSD)
	}
	if !floatEq(res.CostPerVerifiedSuccessUSD, 0.15) {
		t.Errorf("expected KPI 0.15, got %v", res.CostPerVerifiedSuccessUSD)
	}
}

func TestCostPerVerifiedSuccess_Table(t *testing.T) {
	cases := []struct {
		name string
		spec cvsStageSpec
		// expectations about how this single row is classified
		isVerifiedSuccess bool
		isUnverifiedPass  bool
		isVerifyFailure   bool
	}{
		{
			name:              "failed paid run (reported cost, no pass, no verify)",
			spec:              cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.50, compileOk: false},
			isVerifiedSuccess: false, isUnverifiedPass: false, isVerifyFailure: false,
		},
		{
			name: "stdout-only pass (no verification block)",
			spec: cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.10,
				compileOk: true, runtimeOk: true, stdoutOk: true},
			isVerifiedSuccess: false, isUnverifiedPass: true, isVerifyFailure: false,
		},
		{
			name:              "verified success",
			spec:              aVerified("b", 0.10),
			isVerifiedSuccess: true, isUnverifiedPass: false, isVerifyFailure: false,
		},
		{
			name: "verify_ok with zero verified functions (no proved obligation)",
			spec: cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.10,
				compileOk: true, runtimeOk: true, stdoutOk: true,
				verifyOk: true, verified: 0},
			// verify_ok but nothing proved ⇒ NOT a verified success; it's a pass with
			// missing/empty verification evidence ⇒ unverified pass.
			isVerifiedSuccess: false, isUnverifiedPass: true, isVerifyFailure: false,
		},
		{
			name: "counterexample (verification failure)",
			spec: cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.10,
				compileOk: true, runtimeOk: true, stdoutOk: true,
				verifyOk: true, verified: 1, counterex: 1},
			isVerifiedSuccess: false, isUnverifiedPass: false, isVerifyFailure: true,
		},
		{
			name: "skipped obligation (verification failure)",
			spec: cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.10,
				compileOk: true, runtimeOk: true, stdoutOk: true,
				verifyOk: true, verified: 1, skipped: 1},
			isVerifiedSuccess: false, isUnverifiedPass: false, isVerifyFailure: true,
		},
		{
			name: "verifier error (verification failure)",
			spec: cvsStageSpec{benchmarkID: "b", model: "claude-sonnet-4-5", cost: 0.10,
				compileOk: true, runtimeOk: true, stdoutOk: true,
				verifyOk: true, verified: 1, errors: 1},
			isVerifiedSuccess: false, isUnverifiedPass: false, isVerifyFailure: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, store := setupTestDB(t)
			defer db.Close()
			bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{tc.spec})

			res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
				BaselineID: "v1.0", SourceRef: "v1.0/",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.TotalRuns != 1 {
				t.Fatalf("expected 1 total run, got %d", res.TotalRuns)
			}
			gotVS := res.VerifiedSuccesses == 1
			gotUP := res.UnverifiedPasses == 1
			gotVF := res.VerificationFailures == 1
			if gotVS != tc.isVerifiedSuccess {
				t.Errorf("verified success: got %v want %v", gotVS, tc.isVerifiedSuccess)
			}
			if gotUP != tc.isUnverifiedPass {
				t.Errorf("unverified pass: got %v want %v", gotUP, tc.isUnverifiedPass)
			}
			if gotVF != tc.isVerifyFailure {
				t.Errorf("verify failure: got %v want %v", gotVF, tc.isVerifyFailure)
			}
		})
	}
}

func TestCostPerVerifiedSuccess_NumeratorIncludesFailedRunCost(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// 1 verified success ($0.10) + 1 failed paid run ($0.90). Numerator MUST
	// include the failed run: 1.00 / 1 verified = 1.00 (NOT 0.10).
	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{
		aVerified("ok", 0.10),
		{benchmarkID: "fail", model: "claude-sonnet-4-5", cost: 0.90, compileOk: false},
	})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Available {
		t.Fatalf("expected available")
	}
	if !floatEq(res.KnownCostUSD, 1.00) {
		t.Errorf("expected numerator 1.00 (includes failed-run cost), got %v", res.KnownCostUSD)
	}
	if !floatEq(res.CostPerVerifiedSuccessUSD, 1.00) {
		t.Errorf("expected KPI 1.00, got %v", res.CostPerVerifiedSuccessUSD)
	}
	if res.VerifiedSuccesses != 1 {
		t.Errorf("expected 1 verified success, got %d", res.VerifiedSuccesses)
	}
}

func TestCostPerVerifiedSuccess_ReportedVsEstimatedProvenance(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// One reported ($0.20) and one estimated-cost verified success. The estimated
	// one has cost==0 but tokens + a resolvable pricing model.
	specReported := aVerified("rep", 0.20)
	specEstimated := aVerified("est", 0)
	specEstimated.model = "claude-sonnet-4-5" // resolves to a metered rate in models.yml
	specEstimated.tokensIn = 100000
	specEstimated.tokensOut = 100000

	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{specReported, specEstimated})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Available {
		t.Fatalf("expected available, incomplete=%v unknown=%d", res.IncompleteData, res.UnknownStages)
	}
	if res.VerifiedSuccesses != 2 {
		t.Errorf("expected 2 verified successes, got %d", res.VerifiedSuccesses)
	}
	if !floatEq(res.ReportedCostUSD, 0.20) {
		t.Errorf("expected reported 0.20, got %v", res.ReportedCostUSD)
	}
	if res.EstimatedCostUSD <= 0 {
		t.Errorf("expected positive estimated cost, got %v", res.EstimatedCostUSD)
	}
	if !floatEq(res.KnownCostUSD, res.ReportedCostUSD+res.EstimatedCostUSD) {
		t.Errorf("known cost must equal reported+estimated")
	}
}

func TestCostPerVerifiedSuccess_QuotaStageIsZeroNotUnknown(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// A verified success on a quota lane (no tokens, no cost) contributes $0 but
	// is available and countable — quota is NOT unknown.
	quota := aVerified("quota", 0)
	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{quota})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Available {
		t.Fatalf("quota-only cohort should be available (KPI $0), got incomplete=%v", res.IncompleteData)
	}
	if res.QuotaStages != 1 {
		t.Errorf("expected 1 quota stage, got %d", res.QuotaStages)
	}
	if res.UnknownStages != 0 {
		t.Errorf("expected 0 unknown stages, got %d", res.UnknownStages)
	}
	if !floatEq(res.CostPerVerifiedSuccessUSD, 0) {
		t.Errorf("expected KPI $0 for quota-only, got %v", res.CostPerVerifiedSuccessUSD)
	}
}

func TestCostPerVerifiedSuccess_UnknownCostMakesUnavailable(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// A token-bearing stage with an unresolvable model ⇒ unknown cost ⇒ the whole
	// KPI is unavailable / incomplete (NEVER silently $0).
	unknown := aVerified("unk", 0)
	unknown.model = "totally-not-a-real-model-xyz"
	unknown.tokensIn = 500
	unknown.tokensOut = 500

	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{
		aVerified("ok", 0.10),
		unknown,
	})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Available {
		t.Errorf("expected unavailable when a stage has unknown cost")
	}
	if !res.IncompleteData {
		t.Errorf("expected incomplete_data=true")
	}
	if res.UnknownStages != 1 {
		t.Errorf("expected 1 unknown stage, got %d", res.UnknownStages)
	}
}

func TestCostPerVerifiedSuccess_ZeroDenominatorUnavailable(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Paid runs, but NO verified successes ⇒ zero denominator ⇒ unavailable.
	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{
		{benchmarkID: "p", model: "claude-sonnet-4-5", cost: 0.30,
			compileOk: true, runtimeOk: true, stdoutOk: true}, // unverified pass
	})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Available {
		t.Errorf("expected unavailable for zero denominator")
	}
	if res.VerifiedSuccesses != 0 {
		t.Errorf("expected 0 verified successes, got %d", res.VerifiedSuccesses)
	}
	// KPI must NOT be emitted as $0 or infinity when denom is zero.
	if res.CostPerVerifiedSuccessUSD != 0 || res.Available {
		// value may be 0 internally but Available must be false (guarded above)
	}
}

func TestCostPerVerifiedSuccess_EmptyCohortUnavailable(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "no-such-cohort/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Available {
		t.Errorf("expected unavailable for empty cohort")
	}
	if res.TotalRuns != 0 {
		t.Errorf("expected 0 runs, got %d", res.TotalRuns)
	}
}

func TestCostPerVerifiedSuccess_CohortFilterExcludesOtherSourceRefs(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// A v1.0 cohort and an unrelated mission cohort. The filter must only see v1.0.
	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{aVerified("a", 0.10)})
	bankCohort(t, store, "mission:v1/iter-42", []cvsStageSpec{aVerified("b", 5.00)})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalRuns != 1 {
		t.Errorf("expected 1 run (cohort-scoped), got %d", res.TotalRuns)
	}
	if !floatEq(res.KnownCostUSD, 0.10) {
		t.Errorf("expected cohort-scoped cost 0.10 (mission spend excluded), got %v", res.KnownCostUSD)
	}
}

// ===== Eval assessment JSON round-trip / backward-compat (M1 §3.2) =====

func TestEvalAssessment_VerifyFieldsRoundTrip(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, stage := createTestEvalChainWithStage(t, store)
	in := &EvalAssessment{
		BenchmarkID: "b", Model: "claude-sonnet-4-5", Language: "ailang", EvalMode: "agent",
		CompileOk: true, RuntimeOk: true, StdoutOk: true,
		VerifyOk: true, VerifyVerified: 3, VerifyCounterex: 1, VerifySkipped: 2, VerifyErrors: 4,
	}
	if err := store.UpdateStageEvalAssessment(ctx, stage.ID, in); err != nil {
		t.Fatalf("bank: %v", err)
	}
	got, err := store.GetStageEvalAssessment(ctx, stage.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.VerifyVerified != 3 || got.VerifyCounterex != 1 || got.VerifySkipped != 2 || got.VerifyErrors != 4 || !got.VerifyOk {
		t.Errorf("verify fields did not round-trip: %+v", got)
	}
}

func TestEvalAssessment_HistoricalRowIsVerificationMissing(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// A historical row: passes stdout but has no verification block at all
	// (all zeros). It must be treated as verification_missing ⇒ unverified pass,
	// NEVER a verified success.
	bankCohort(t, store, "v1.0/agent/baseline", []cvsStageSpec{
		{benchmarkID: "hist", model: "claude-sonnet-4-5", cost: 0.10,
			compileOk: true, runtimeOk: true, stdoutOk: true},
	})

	res, err := computeCostPerVerifiedSuccess(context.Background(), store, CostPerVerifiedSuccessOptions{
		BaselineID: "v1.0", SourceRef: "v1.0/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.VerifiedSuccesses != 0 {
		t.Errorf("historical (no-verify) row must NOT count as verified, got %d", res.VerifiedSuccesses)
	}
	if res.UnverifiedPasses != 1 {
		t.Errorf("expected 1 unverified pass, got %d", res.UnverifiedPasses)
	}
}
