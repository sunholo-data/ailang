package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"

	_ "github.com/mattn/go-sqlite3"
)

// TestSelectBenchmarksByConfidence verifies the M3 selection: saturated (Trivial)
// benchmarks are dropped, and the rest are ranked by proximity to the median
// model rating, capped at max.
func TestSelectBenchmarksByConfidence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ratings.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := observatory.MigrateWithVersion(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()

	// Models sit around 1600 (median). Benchmarks: one Trivial (excluded), and
	// three discriminating at increasing distance from the median.
	models := map[string]float64{"a": 1500, "b": 1600, "c": 1700}
	benches := map[string]float64{
		"saturated_easy": 1100, // Trivial -> dropped
		"near":           1620, // closest to median 1600
		"mid":            1800,
		"far":            2100,
	}
	if err := observatory.SaveRatings(ctx, db, "standard", models, benches, nil, nil); err != nil {
		t.Fatalf("save: %v", err)
	}
	_ = db.Close()

	got, err := selectBenchmarksByConfidence(dbPath, "standard", 2)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	// max=2, Trivial excluded, nearest-first → ["near", "mid"].
	want := []string{"near", "mid"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
	for _, id := range got {
		if id == "saturated_easy" {
			t.Errorf("Trivial benchmark should have been excluded: %v", got)
		}
	}
}
