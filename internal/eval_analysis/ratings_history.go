package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// attachRatingsToHistoryEntry is the export-path wiring: fit this run's anchored
// model ratings and look for the release's stamped direction index next to its
// banked results (eval_results/baselines/<version>/direction_index.json — where
// tools/linking-run.sh writes it).
func attachRatingsToHistoryEntry(entry *HistoryEntry, standard []*BenchmarkResult) {
	if len(standard) == 0 {
		return
	}
	trials := make([]eval_harness.Trial, 0, len(standard))
	for _, r := range standard {
		trials = append(trials, eval_harness.Trial{
			Model: r.Model,
			Bench: r.ID,
			Pass:  r.CompileOk && r.RuntimeOk && r.StdoutOk,
		})
	}
	modelELO, _ := eval_harness.FitFromTrialsAnchored(trials, eval_harness.AnchorPanelV1, nil)
	idxPath := filepath.Join("eval_results", "baselines", entry.Version, "direction_index.json")
	if _, err := os.Stat(idxPath); err != nil {
		idxPath = ""
	}
	AttachRatingsHistory(entry, modelELO, eval_harness.AnchorVersion, idxPath)
}

// directionIndexDoc is the on-disk artifact written by tools/direction-fit.
type directionIndexDoc struct {
	Version         string             `json:"version"`
	PanelVersion    string             `json:"panel_version"`
	IndexOverall    float64            `json:"index_overall"`
	IndexByTier     map[string]float64 `json:"index_by_tier"`
	BridgeStrengths map[string]float64 `json:"bridge_strengths_used"`
	Trials          int                `json:"trials"`
}

// AttachRatingsHistory fills entry.Ratings for one release: the anchored model
// leaderboard from this run's fit, plus the release's stamped language-direction
// index if the linking run produced one (M-EVAL-ROLLING-ELO M4).
//
// The index is READ from the artifact rather than recomputed here, because it is
// a stamped measurement: recomputing it later with today's bridge strengths would
// rewrite history, which is exactly what the design forbids. A missing artifact is
// not an error — releases measured before the linking-run protocol simply carry the
// model half of the series.
func AttachRatingsHistory(entry *HistoryEntry, modelELO map[string]float64, anchorVersion, directionIndexPath string) {
	if len(modelELO) == 0 && directionIndexPath == "" {
		return
	}
	point := &RatingsHistoryPoint{AnchorVersion: anchorVersion}
	if len(modelELO) > 0 {
		point.Models = make(map[string]float64, len(modelELO))
		for m, r := range modelELO {
			point.Models[m] = round1(r)
		}
	}
	if directionIndexPath != "" {
		if raw, err := os.ReadFile(directionIndexPath); err == nil {
			var doc directionIndexDoc
			if json.Unmarshal(raw, &doc) == nil && doc.IndexOverall != 0 {
				point.PanelVersion = doc.PanelVersion
				point.DirectionIndex = round1(doc.IndexOverall)
				point.DirectionByTier = doc.IndexByTier
				point.BridgeStrengths = doc.BridgeStrengths
				point.Trials = doc.Trials
			}
		}
	}
	if point.Models == nil && point.DirectionIndex == 0 {
		return
	}
	entry.Ratings = point
}
