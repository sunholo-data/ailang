package eval_harness

import (
	"math"
	"sort"
)

// ELO-style rating engine for the eval rig (M-EVAL-RATING-EFFICIENCY).
//
// Each (model, benchmark) trial is a game: a PASS is a win for the model against
// the benchmark. Iterating the standard ELO update over a set of trials yields a
// capability rating per model and a *derived difficulty* per benchmark — so the
// rig itself decides what "hard" means, instead of hand-assigned tiers.

// DefaultInitialRating is the ELO starting point (chess convention).
const DefaultInitialRating = 1500.0

// Trial is one (model, benchmark) outcome. Pass=true means the model beat the
// benchmark (compile+runtime+stdout all ok).
type Trial struct {
	Model string
	Bench string
	Pass  bool
}

// UpdateTrial applies one ELO update for a model-vs-benchmark game and returns the
// new (model, benchmark) ratings. A PASS is outcome 1.0 for the model; the
// benchmark receives the mirror update so the pair is zero-sum. k is the step.
func UpdateTrial(modelRating, benchRating float64, pass bool, k float64) (newModel, newBench float64) {
	expected := 1.0 / (1.0 + math.Pow(10, (benchRating-modelRating)/400.0))
	actual := 0.0
	if pass {
		actual = 1.0
	}
	delta := k * (actual - expected)
	return modelRating + delta, benchRating - delta
}

// FitFromTrials fits converged ELO ratings over a static set of trials by
// iterating the update with a decaying step (≈32 → 4). It is DETERMINISTIC: the
// trials are processed in a fixed (bench, model) order every epoch, so the same
// input always produces the same ratings. Returns model ratings and benchmark
// ratings (benchmark rating = derived difficulty: higher = harder).
func FitFromTrials(trials []Trial) (modelRatings, benchRatings map[string]float64) {
	modelRatings = make(map[string]float64)
	benchRatings = make(map[string]float64)
	for _, t := range trials {
		if _, ok := modelRatings[t.Model]; !ok {
			modelRatings[t.Model] = DefaultInitialRating
		}
		if _, ok := benchRatings[t.Bench]; !ok {
			benchRatings[t.Bench] = DefaultInitialRating
		}
	}
	ordered := append([]Trial(nil), trials...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Bench != ordered[j].Bench {
			return ordered[i].Bench < ordered[j].Bench
		}
		return ordered[i].Model < ordered[j].Model
	})
	const epochs = 400
	for e := 0; e < epochs; e++ {
		k := 4 + 28*math.Exp(-float64(e)/80.0)
		for _, t := range ordered {
			nm, nb := UpdateTrial(modelRatings[t.Model], benchRatings[t.Bench], t.Pass, k)
			modelRatings[t.Model], benchRatings[t.Bench] = nm, nb
		}
	}
	return modelRatings, benchRatings
}

// Band maps an ELO rating to a difficulty band (descriptive, emergent from data).
func Band(rating float64) string {
	switch {
	case rating < 1300:
		return "Trivial"
	case rating < 1500:
		return "Easy"
	case rating < 1700:
		return "Moderate"
	case rating < 1900:
		return "Hard"
	default:
		return "Very hard"
	}
}
