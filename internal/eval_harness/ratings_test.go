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
