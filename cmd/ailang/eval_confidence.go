package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"

	_ "github.com/mattn/go-sqlite3"
)

// defaultObservatoryDB resolves ~/.ailang/state/observatory.db (where the ELO
// rating tables live). Returns "" if the home dir can't be determined.
func defaultObservatoryDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ailang", "state", "observatory.db")
}

// selectBenchmarksByConfidence returns up to max benchmarks worth re-running for
// the given mode, read from a ratings DB (M-EVAL-RATING-EFFICIENCY part 2, M3).
// It drops SATURATED benchmarks (Trivial band — every model passes, so another
// trial yields no information) and ranks the rest by proximity to the field's
// median model rating: a benchmark sitting near where the models actually are
// discriminates them best, so one more trial there moves belief the most.
func selectBenchmarksByConfidence(dbPath, mode string, max int) ([]string, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open ratings db %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	benches, err := observatory.LoadBenchmarkRatings(ctx, db, mode)
	if err != nil {
		return nil, err
	}
	if len(benches) == 0 {
		return nil, fmt.Errorf("no %s benchmark ratings in %s — seed it first:\n  eval-elo <baseline_dir> --mode %s --persist %s", mode, dbPath, mode, dbPath)
	}
	models, err := observatory.LoadModelRatings(ctx, db, mode)
	if err != nil {
		return nil, err
	}

	median := 1500.0
	if n := len(models); n > 0 {
		rs := make([]float64, 0, n)
		for _, m := range models {
			rs = append(rs, m.Rating)
		}
		sort.Float64s(rs)
		median = rs[n/2]
	}

	type cand struct {
		id   string
		dist float64
	}
	cands := make([]cand, 0, len(benches))
	for _, b := range benches {
		if eval_harness.Band(b.Rating) == "Trivial" {
			continue // saturated-easy: no discrimination
		}
		d := b.Rating - median
		if d < 0 {
			d = -d
		}
		cands = append(cands, cand{b.BenchmarkID, d})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].dist < cands[j].dist })

	out := make([]string, 0, len(cands))
	for i, c := range cands {
		if max > 0 && i >= max {
			break
		}
		out = append(out, c.id)
	}
	return out, nil
}
