package eval_harness

import (
	"math"
	"testing"
)

func TestUpdateTrial_Mirror_ZeroSum(t *testing.T) {
	nm, nb := UpdateTrial(1500, 1500, true, 32)
	dM, dB := nm-1500, nb-1500
	if math.Abs(dM+dB) > 1e-9 {
		t.Errorf("update not zero-sum: model %+.3f benchmark %+.3f", dM, dB)
	}
}

func TestUpdateTrial_CoinFlip(t *testing.T) {
	// Equal ratings, a win → +k/2 model, -k/2 benchmark.
	nm, nb := UpdateTrial(1500, 1500, true, 32)
	if math.Abs(nm-1516) > 1e-6 || math.Abs(nb-1484) > 1e-6 {
		t.Errorf("coin-flip got model=%.3f bench=%.3f, want 1516/1484", nm, nb)
	}
}

func TestUpdateTrial_ExpectedWinIsSmall_UpsetIsLarge(t *testing.T) {
	// Strong model beats easy benchmark: small rating gain (expected).
	strongM, _ := UpdateTrial(1900, 1300, true, 32)
	strongDelta := strongM - 1900
	// Weak model beats hard benchmark: large rating gain (upset).
	weakM, _ := UpdateTrial(1300, 1900, true, 32)
	weakDelta := weakM - 1300
	if !(strongDelta > 0 && weakDelta > 0) {
		t.Fatalf("both wins should raise the model: strong=%.3f weak=%.3f", strongDelta, weakDelta)
	}
	if !(weakDelta > 10*strongDelta) {
		t.Errorf("upset (%.3f) should dwarf the expected win (%.3f)", weakDelta, strongDelta)
	}
}

func TestUpdateTrial_LossDropsModel(t *testing.T) {
	nm, nb := UpdateTrial(1500, 1500, false, 32)
	if !(nm < 1500 && nb > 1500) {
		t.Errorf("a loss should drop the model and raise the benchmark: model=%.3f bench=%.3f", nm, nb)
	}
}

func TestFitFromTrials_SeparatesByCapabilityAndDifficulty(t *testing.T) {
	// strong passes both; weak passes easy, fails hard.
	trials := []Trial{
		{"strong", "easy", true}, {"strong", "hard", true},
		{"weak", "easy", true}, {"weak", "hard", false},
	}
	mr, br := FitFromTrials(trials)
	if !(mr["strong"] > mr["weak"]) {
		t.Errorf("strong (%.0f) should outrate weak (%.0f)", mr["strong"], mr["weak"])
	}
	if !(br["hard"] > br["easy"]) {
		t.Errorf("hard (%.0f) should outrate easy (%.0f) in difficulty", br["hard"], br["easy"])
	}
}

func TestFitFromTrials_Deterministic(t *testing.T) {
	trials := []Trial{
		{"a", "x", true}, {"b", "x", false}, {"a", "y", false}, {"b", "y", true},
	}
	m1, b1 := FitFromTrials(trials)
	m2, b2 := FitFromTrials(trials)
	for k := range m1 {
		if m1[k] != m2[k] {
			t.Errorf("non-deterministic model rating for %s: %.6f vs %.6f", k, m1[k], m2[k])
		}
	}
	for k := range b1 {
		if b1[k] != b2[k] {
			t.Errorf("non-deterministic benchmark rating for %s: %.6f vs %.6f", k, b1[k], b2[k])
		}
	}
}

// TestFitFromTrials_UnseenBenchmarkEntersFlat is the M-EVAL-FRONTIER-TIER (M3)
// regression: a never-before-seen benchmark id (e.g. a newly-added frontier
// benchmark) enters the ELO fit at DefaultInitialRating (1500) with NO tier-biased
// seeding — the rating engine treats every benchmark identically regardless of
// tier. Its derived difficulty then RISES on FAIL and FALLS on PASS, exactly like
// any other benchmark, so a frontier benchmark's "hard" rating is emergent from
// pass/fail data, not seeded. Guards against anyone adding tier-biased provisional
// seeding without recalibrating (there is no data to calibrate it — see the
// ratings.go DefaultInitialRating comment).
func TestFitFromTrials_UnseenBenchmarkEntersFlat(t *testing.T) {
	// A single model, one trial, against a brand-new frontier-style benchmark id.
	// With one trial the fit stays close to the seed, so we can observe the
	// direction of the update from the flat 1500 start.
	failTrials := []Trial{{"m", "frontier_new_bench", false}}
	passTrials := []Trial{{"m", "frontier_new_bench", true}}

	_, brFail := FitFromTrials(failTrials)
	_, brPass := FitFromTrials(passTrials)

	// The unseen benchmark must have been present in the output (it was seeded
	// generically at DefaultInitialRating with no tier-specific branch).
	if _, ok := brFail["frontier_new_bench"]; !ok {
		t.Fatalf("unseen benchmark not present in fitted ratings: %v", brFail)
	}

	// A FAIL (model does not beat the benchmark) should push benchmark difficulty
	// ABOVE the flat seed; a PASS should push it BELOW.
	if !(brFail["frontier_new_bench"] > DefaultInitialRating) {
		t.Errorf("FAIL should raise difficulty above seed %.0f, got %.1f",
			DefaultInitialRating, brFail["frontier_new_bench"])
	}
	if !(brPass["frontier_new_bench"] < DefaultInitialRating) {
		t.Errorf("PASS should lower difficulty below seed %.0f, got %.1f",
			DefaultInitialRating, brPass["frontier_new_bench"])
	}
}

// TestFitFromTrials_TierAgnostic confirms the ELO fit does not special-case any
// benchmark by name/tier: two benchmarks with identical pass/fail patterns get
// (near-)identical difficulty ratings, regardless of what a frontier vs core
// naming might imply. They are not bit-identical only because the deterministic
// (bench, model) processing order interleaves updates to the SHARED model
// ratings differently — a consequence of ordering, NOT of any tier/name branch
// (there is none). The tiny residual (<0.5 ELO here) proves the treatment is
// identical up to that ordering. (M-EVAL-FRONTIER-TIER M3.)
func TestFitFromTrials_TierAgnostic(t *testing.T) {
	trials := []Trial{
		// "frontier_x" and "core_y" see the exact same outcomes.
		{"strong", "frontier_x", true}, {"weak", "frontier_x", false},
		{"strong", "core_y", true}, {"weak", "core_y", false},
	}
	_, br := FitFromTrials(trials)
	const eps = 1.0 // ELO points; ordering-only residual, must be far below a band width (200)
	if diff := br["frontier_x"] - br["core_y"]; diff > eps || diff < -eps {
		t.Errorf("tier-agnostic fit violated: frontier_x=%.6f core_y=%.6f (identical trials must give near-identical difficulty; diff %.6f > %.1f implies name/tier special-casing)",
			br["frontier_x"], br["core_y"], diff, eps)
	}
}

func TestBand(t *testing.T) {
	cases := []struct {
		r    float64
		want string
	}{
		{1200, "Trivial"}, {1299.9, "Trivial"}, {1300, "Easy"}, {1499, "Easy"},
		{1500, "Moderate"}, {1699, "Moderate"}, {1700, "Hard"}, {1899, "Hard"},
		{1900, "Very hard"}, {2300, "Very hard"},
	}
	for _, c := range cases {
		if got := Band(c.r); got != c.want {
			t.Errorf("Band(%.1f) = %q, want %q", c.r, got, c.want)
		}
	}
}
