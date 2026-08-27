package eval_harness

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

//go:embed anchor_v1.json
var anchorV1JSON []byte

// AnchorPanelV1 is the frozen reference panel of benchmark difficulties from v0.32.0 baseline + banked runs.
// When fitting ratings, these benchmark difficulties are held fixed to anchor the ELO scale,
// making ratings comparable across pool compositions (M-EVAL-ROLLING-ELO M1).
var AnchorPanelV1 map[string]float64

// AnchorVersion is the embedded anchor's version label, stamped into published
// rating history so a reader can tell which scale a number was fitted on
// (re-anchoring produces anchor_v2 and this changes with it).
var AnchorVersion string

func init() {
	var anchor struct {
		Version    string             `json:"version"`
		Benchmarks map[string]float64 `json:"benchmarks"`
	}
	if err := json.Unmarshal(anchorV1JSON, &anchor); err != nil {
		// The anchor is an embedded build artifact: if it does not parse, the
		// binary is broken and every anchored fit would silently degrade to an
		// unanchored one — the exact incomparability this file exists to kill.
		// NO SILENT FALLBACKS (CLAUDE.md §2): fail at init, loudly.
		panic(fmt.Sprintf("eval_harness: embedded anchor_v1.json is invalid: %v", err))
	}
	if len(anchor.Benchmarks) == 0 {
		panic("eval_harness: embedded anchor_v1.json has an empty benchmark panel")
	}
	AnchorPanelV1 = anchor.Benchmarks
	AnchorVersion = anchor.Version
}

// DefaultCoverageThreshold is the provisional threshold for gating evaluation decisions.
// Unified to 90% to match the site's ELO_COVERAGE_FRACTION and prevent drift between Go and JavaScript.
const DefaultCoverageThreshold = 0.9

// ELO-style rating engine for the eval rig (M-EVAL-RATING-EFFICIENCY).
//
// Each (model, benchmark) trial is a game: a PASS is a win for the model against
// the benchmark. Iterating the standard ELO update over a set of trials yields a
// capability rating per model and a *derived difficulty* per benchmark — so the
// rig itself decides what "hard" means, instead of hand-assigned tiers.

// DefaultInitialRating is the ELO starting point (chess convention).
//
// This is a FLAT seed applied to every unseen benchmark AND every unseen model,
// regardless of tier. There is deliberately no tier-biased provisional rating:
// a `frontier` benchmark does NOT enter with a higher difficulty than a `core`
// benchmark — its "hard" rating is EMERGENT from pass/fail data as the fit
// converges (a benchmark that frontier models fail rises; one they pass falls).
// The M-EVAL-FRONTIER-TIER design doc's phrasing "enter with a high provisional
// ELO" is aspirational, not literal: seeding frontier benchmarks high would need
// calibration data that does not exist. Do not add tier-biased seeding here
// without that data (guarded by TestFitFromTrials_UnseenBenchmarkEntersFlat and
// TestFitFromTrials_TierAgnostic in ratings_test.go).
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

// FitFromTrialsAnchored fits converged ELO ratings over a static set of trials by
// iterating the update with a decaying step (≈32 → 4), holding designated entities fixed.
// Entities present in fixedBench or fixedModels keep their given rating; deltas to those
// entities are discarded. It is DETERMINISTIC: same trials + same fixed sets ⇒ same ratings.
//
// Usage:
//   - Placement fit (default): fixedBench=anchor_vN, fixedModels=nil → model ratings change, benchmark panel fixed
//   - Direction fit: fixedBench=nil, fixedModels=bridge_strengths → benchmark ratings change, bridge models fixed
//   - Legacy (no anchor): fixedBench=nil, fixedModels=nil → behavior-preserving delegation for existing code
func FitFromTrialsAnchored(trials []Trial, fixedBench, fixedModels map[string]float64) (modelRatings, benchRatings map[string]float64) {
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
	// Apply initial fixed values (if provided, they override the default seed).
	for model, rating := range fixedModels {
		modelRatings[model] = rating
	}
	for bench, rating := range fixedBench {
		benchRatings[bench] = rating
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
			// Apply update only if entity is not fixed (not in the fixed map).
			if _, isFixed := fixedModels[t.Model]; !isFixed {
				modelRatings[t.Model] = nm
			}
			if _, isFixed := fixedBench[t.Bench]; !isFixed {
				benchRatings[t.Bench] = nb
			}
		}
	}
	return modelRatings, benchRatings
}

// FitFromTrials fits converged ELO ratings over a static set of trials by
// iterating the update with a decaying step (≈32 → 4). It is DETERMINISTIC: the
// trials are processed in a fixed (bench, model) order every epoch, so the same
// input always produces the same ratings. Returns model ratings and benchmark
// ratings (benchmark rating = derived difficulty: higher = harder).
//
// This is now a behavior-preserving delegation to FitFromTrialsAnchored with no fixed entities.
func FitFromTrials(trials []Trial) (modelRatings, benchRatings map[string]float64) {
	return FitFromTrialsAnchored(trials, nil, nil)
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
