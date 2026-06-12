package observatory

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func ratingsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	v, err := MigrateWithVersion(db)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if v < 16 {
		t.Fatalf("expected schema >= v16, got v%d", v)
	}
	return db
}

func TestMigrateWithVersion_V16CreatesRatingTables(t *testing.T) {
	db := ratingsTestDB(t)
	for _, table := range []string{"benchmark_ratings", "model_ratings", "trial_history"} {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing after v16 migration: %v", table, err)
		}
	}
	// ValidateSchema must now accept the rating tables.
	if err := ValidateSchema(db); err != nil {
		t.Errorf("ValidateSchema failed after v16: %v", err)
	}
}

func TestSaveRatings_RoundTrip(t *testing.T) {
	db := ratingsTestDB(t)
	ctx := context.Background()

	mr := map[string]float64{"strong": 2100, "weak": 1400}
	br := map[string]float64{"hard": 1950, "easy": 1200}
	mt := map[string]int{"strong": 37, "weak": 37}
	bt := map[string]int{"hard": 11, "easy": 11}
	if err := SaveRatings(ctx, db, mr, br, mt, bt); err != nil {
		t.Fatalf("SaveRatings: %v", err)
	}

	models, err := LoadModelRatings(ctx, db)
	if err != nil {
		t.Fatalf("LoadModelRatings: %v", err)
	}
	if len(models) != 2 || models[0].ModelID != "strong" {
		t.Fatalf("model order/contents wrong: %+v", models)
	}
	if models[0].Rating != 2100 || models[0].NTrials != 37 || models[0].KFactor != 32 {
		t.Errorf("strong row wrong: %+v", models[0])
	}

	benches, err := LoadBenchmarkRatings(ctx, db)
	if err != nil {
		t.Fatalf("LoadBenchmarkRatings: %v", err)
	}
	if len(benches) != 2 || benches[0].BenchmarkID != "hard" || benches[1].BenchmarkID != "easy" {
		t.Fatalf("benchmark order wrong (hardest first expected): %+v", benches)
	}

	// Upsert: re-saving with a new rating updates in place, no duplicate rows.
	if err := SaveRatings(ctx, db, map[string]float64{"strong": 2200}, nil, map[string]int{"strong": 50}, nil); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	models, _ = LoadModelRatings(ctx, db)
	if len(models) != 2 {
		t.Fatalf("upsert created a duplicate row: %d models", len(models))
	}
	for _, m := range models {
		if m.ModelID == "strong" && (m.Rating != 2200 || m.NTrials != 50) {
			t.Errorf("upsert did not update strong: %+v", m)
		}
	}
}
