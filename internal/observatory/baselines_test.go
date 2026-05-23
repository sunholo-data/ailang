package observatory

import (
	"context"
	"database/sql"
	"math"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// newBaselineTestDB opens an in-memory observatory DB and runs migrations.
// Returns the DB plus a teardown.
func newBaselineTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, func() { _ = db.Close() }
}

// TestGetEvalBaseline_MissingRowReturnsNil covers the bootstrap case:
// no row yet -> (nil, nil). Caller must handle nil and fall back to
// fixed threshold.
func TestGetEvalBaseline_MissingRowReturnsNil(t *testing.T) {
	db, teardown := newBaselineTestDB(t)
	defer teardown()
	ctx := context.Background()

	got, err := GetEvalBaseline(ctx, db, "model-x", "bench-y")
	if err != nil {
		t.Fatalf("GetEvalBaseline: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing row, got %+v", got)
	}
}

// TestUpdatePassedTrial_CreatesAndUpdates exercises the UPSERT path:
// first call creates the row; subsequent calls extend the running stats.
func TestUpdatePassedTrial_CreatesAndUpdates(t *testing.T) {
	db, teardown := newBaselineTestDB(t)
	defer teardown()
	ctx := context.Background()

	// First passing trial: 100K tokens. n=1, mean=100000, stddev=0.
	if err := UpdatePassedTrial(ctx, db, "m", "b", 100000); err != nil {
		t.Fatalf("first update: %v", err)
	}
	b, err := GetEvalBaseline(ctx, db, "m", "b")
	if err != nil || b == nil {
		t.Fatalf("get after first update: err=%v baseline=%+v", err, b)
	}
	if b.NPassTrials != 1 {
		t.Errorf("after 1 update: n=%d, want 1", b.NPassTrials)
	}
	if b.MeanTokens != 100000 {
		t.Errorf("after 1 update: mean=%v, want 100000", b.MeanTokens)
	}
	if b.StddevTokens != 0 {
		t.Errorf("after 1 update: stddev=%v, want 0 (n=1)", b.StddevTokens)
	}

	// Second passing trial: 120K. n=2, mean=110000, stddev=sqrt(200M)=14142.135.
	if err := UpdatePassedTrial(ctx, db, "m", "b", 120000); err != nil {
		t.Fatalf("second update: %v", err)
	}
	b, _ = GetEvalBaseline(ctx, db, "m", "b")
	if b.NPassTrials != 2 {
		t.Errorf("after 2 updates: n=%d, want 2", b.NPassTrials)
	}
	if math.Abs(b.MeanTokens-110000) > 0.01 {
		t.Errorf("after 2 updates: mean=%v, want 110000", b.MeanTokens)
	}
	// Sample stddev: sqrt(sum((x-mean)^2) / (n-1)) = sqrt(2e8 / 1) = ~14142.135
	if math.Abs(b.StddevTokens-14142.135623730952) > 0.01 {
		t.Errorf("after 2 updates: stddev=%v, want ~14142.14", b.StddevTokens)
	}
}

// TestUpdatePassedTrial_WelfordConvergence verifies the online algorithm
// converges to the same mean and (sample) stddev as the naive batch formula.
// Run 10 PASS samples drawn from a known distribution, then assert.
func TestUpdatePassedTrial_WelfordConvergence(t *testing.T) {
	db, teardown := newBaselineTestDB(t)
	defer teardown()
	ctx := context.Background()

	samples := []int{100000, 110000, 95000, 105000, 115000, 90000, 120000, 100000, 110000, 105000}
	for _, s := range samples {
		if err := UpdatePassedTrial(ctx, db, "model-conv", "fizzbuzz", s); err != nil {
			t.Fatalf("update with %d: %v", s, err)
		}
	}
	b, _ := GetEvalBaseline(ctx, db, "model-conv", "fizzbuzz")

	// Reference: batch-compute mean and sample stddev.
	var sum float64
	for _, s := range samples {
		sum += float64(s)
	}
	wantMean := sum / float64(len(samples))
	var sumSq float64
	for _, s := range samples {
		d := float64(s) - wantMean
		sumSq += d * d
	}
	wantStddev := math.Sqrt(sumSq / float64(len(samples)-1))

	if math.Abs(b.MeanTokens-wantMean) > 0.5 {
		t.Errorf("Welford mean = %v, batch mean = %v (delta %v)", b.MeanTokens, wantMean, b.MeanTokens-wantMean)
	}
	if math.Abs(b.StddevTokens-wantStddev) > 0.5 {
		t.Errorf("Welford stddev = %v, batch stddev = %v (delta %v)", b.StddevTokens, wantStddev, b.StddevTokens-wantStddev)
	}
	if b.NPassTrials != 10 {
		t.Errorf("NPassTrials = %d, want 10", b.NPassTrials)
	}
}

// TestComputeAdaptiveThreshold_BootstrapFallsBackToFixed:
// during bootstrap (n < BootstrapPassesRequired), the function MUST
// return the fixed fallback, regardless of mean/stddev content.
func TestComputeAdaptiveThreshold_BootstrapFallsBackToFixed(t *testing.T) {
	cases := []struct {
		name string
		b    *EvalBaseline
		fix  int
		want int
	}{
		{"nil baseline (no row yet)", nil, 500000, 500000},
		{"n=0 (zero rows after a delete)", &EvalBaseline{NPassTrials: 0, MeanTokens: 100000}, 500000, 500000},
		{"n=4 (just under bootstrap)", &EvalBaseline{NPassTrials: 4, MeanTokens: 100000, StddevTokens: 5000}, 500000, 500000},
		{"fixed=0 during bootstrap means no abort", nil, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeAdaptiveThreshold(tc.b, AdaptiveThresholdSigmas, tc.fix)
			if got != tc.want {
				t.Errorf("ComputeAdaptiveThreshold = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestComputeAdaptiveThreshold_PostBootstrapUsesMeanPlusSigma:
// once n >= 5, the function returns ceil(mean + sigmas * stddev) and
// IGNORES the fixed fallback.
func TestComputeAdaptiveThreshold_PostBootstrapUsesMeanPlusSigma(t *testing.T) {
	b := &EvalBaseline{
		NPassTrials:  5,
		MeanTokens:   100000,
		StddevTokens: 10000,
	}
	// 100000 + 2.0*10000 = 120000
	got := ComputeAdaptiveThreshold(b, 2.0, 500000)
	if got != 120000 {
		t.Errorf("got %d, want 120000 (mean + 2*stddev)", got)
	}

	// With sigmas = 1.5: 100000 + 15000 = 115000
	if got := ComputeAdaptiveThreshold(b, 1.5, 0); got != 115000 {
		t.Errorf("got %d at 1.5σ, want 115000", got)
	}
}

// TestComputeAdaptiveThreshold_CeilingRounding ensures fractional means
// round UP to avoid under-budgeting borderline runs.
func TestComputeAdaptiveThreshold_CeilingRounding(t *testing.T) {
	b := &EvalBaseline{NPassTrials: 5, MeanTokens: 100.1, StddevTokens: 0}
	// 100.1 + 0 = 100.1 -> ceil = 101
	got := ComputeAdaptiveThreshold(b, AdaptiveThresholdSigmas, 0)
	if got != 101 {
		t.Errorf("got %d, want 101 (ceil(100.1))", got)
	}
}
