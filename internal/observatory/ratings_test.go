package observatory

import (
	"context"
	"database/sql"
	"testing"
	"time"

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
	if err := SaveRatings(ctx, db, "standard", mr, br, mt, bt); err != nil {
		t.Fatalf("SaveRatings: %v", err)
	}

	models, err := LoadModelRatings(ctx, db, "standard")
	if err != nil {
		t.Fatalf("LoadModelRatings: %v", err)
	}
	if len(models) != 2 || models[0].ModelID != "strong" {
		t.Fatalf("model order/contents wrong: %+v", models)
	}
	if models[0].Rating != 2100 || models[0].NTrials != 37 || models[0].KFactor != 32 || models[0].Mode != "standard" {
		t.Errorf("strong row wrong: %+v", models[0])
	}

	benches, err := LoadBenchmarkRatings(ctx, db, "standard")
	if err != nil {
		t.Fatalf("LoadBenchmarkRatings: %v", err)
	}
	if len(benches) != 2 || benches[0].BenchmarkID != "hard" || benches[1].BenchmarkID != "easy" {
		t.Fatalf("benchmark order wrong (hardest first expected): %+v", benches)
	}

	// Upsert: re-saving the same mode updates in place, no duplicate rows.
	if err := SaveRatings(ctx, db, "standard", map[string]float64{"strong": 2200}, nil, map[string]int{"strong": 50}, nil); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	models, _ = LoadModelRatings(ctx, db, "standard")
	if len(models) != 2 {
		t.Fatalf("upsert created a duplicate row: %d models", len(models))
	}
	for _, m := range models {
		if m.ModelID == "strong" && (m.Rating != 2200 || m.NTrials != 50) {
			t.Errorf("upsert did not update strong: %+v", m)
		}
	}

	// Mode separation: an 'agent' rating for the SAME model coexists, not clobbers.
	if err := SaveRatings(ctx, db, "agent", map[string]float64{"strong": 1700}, nil, nil, nil); err != nil {
		t.Fatalf("save agent: %v", err)
	}
	agent, _ := LoadModelRatings(ctx, db, "agent")
	if len(agent) != 1 || agent[0].Rating != 1700 || agent[0].Mode != "agent" {
		t.Fatalf("agent rating wrong/missing: %+v", agent)
	}
	std, _ := LoadModelRatings(ctx, db, "standard")
	if len(std) != 2 {
		t.Errorf("agent save clobbered standard rows: %d", len(std))
	}
	for _, m := range std {
		if m.ModelID == "strong" && m.Rating != 2200 {
			t.Errorf("standard 'strong' changed after agent save: %+v", m)
		}
	}
}

// TestAppendTrialHistory_Idempotency verifies that re-appending the same trials
// (with the same trial_id) is a no-op enforced by the INSERT OR IGNORE constraint
// (M-EVAL-ROLLING-ELO M2). This is how the persist path achieves idempotency when
// re-running over the same eval corpus.
func TestAppendTrialHistory_Idempotency(t *testing.T) {
	db := ratingsTestDB(t)
	ctx := context.Background()

	now := time.Now()
	entry := TrialHistoryEntry{
		TrialID:           "trial_001", // This trial ID must be unique (PRIMARY KEY)
		BenchmarkID:       "bench_x",
		ModelID:           "model_a",
		Mode:              "standard",
		Outcome:           1, // pass
		PromptVersion:     "v1.0",
		CompilerVersion:   "v0.35.0",
		BenchRatingBefore: 1500.0,
		ModelRatingBefore: 1500.0,
		BenchRatingAfter:  1480.0,
		ModelRatingAfter:  1520.0,
		RecordedAt:        now,
	}

	// First append: should insert successfully.
	if err := AppendTrialHistory(ctx, db, []TrialHistoryEntry{entry}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	// Second append of the SAME trial: should be a no-op due to INSERT OR IGNORE.
	// The function should not error, and no duplicate row should appear.
	if err := AppendTrialHistory(ctx, db, []TrialHistoryEntry{entry}); err != nil {
		t.Fatalf("second append (idempotent re-persist): %v", err)
	}

	// Verify only ONE row exists (not two).
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM trial_history WHERE trial_id = ?", "trial_001").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("idempotency failed: expected 1 row, got %d (INSERT OR IGNORE did not prevent duplicate)", count)
	}

	// Verify the data is correct.
	var e TrialHistoryEntry
	err := db.QueryRow(`
		SELECT trial_id, benchmark_id, model_id, mode, outcome, prompt_version,
		       compiler_version, benchmark_rating_before, model_rating_before,
		       benchmark_rating_after, model_rating_after, recorded_at
		FROM trial_history WHERE trial_id = ?
	`, "trial_001").Scan(
		&e.TrialID, &e.BenchmarkID, &e.ModelID, &e.Mode, &e.Outcome,
		&e.PromptVersion, &e.CompilerVersion,
		&e.BenchRatingBefore, &e.ModelRatingBefore,
		&e.BenchRatingAfter, &e.ModelRatingAfter, &e.RecordedAt,
	)
	if err != nil {
		t.Fatalf("query inserted row: %v", err)
	}
	if e.TrialID != "trial_001" || e.BenchmarkID != "bench_x" || e.Outcome != 1 {
		t.Errorf("persisted data incorrect: %+v", e)
	}
}

// TestAppendTrialHistory_VersionStamping verifies that compiler_version and
// prompt_version are recorded correctly from the trial entry (M2 requirement).
func TestAppendTrialHistory_VersionStamping(t *testing.T) {
	db := ratingsTestDB(t)
	ctx := context.Background()

	entries := []TrialHistoryEntry{
		{
			TrialID:         "trial_v1",
			BenchmarkID:     "bench",
			ModelID:         "model",
			Mode:            "standard",
			Outcome:         1,
			PromptVersion:   "prompt_v2.1",
			CompilerVersion: "v0.35.0-dev",
			RecordedAt:      time.Now(),
		},
		{
			TrialID:         "trial_v2",
			BenchmarkID:     "bench",
			ModelID:         "model",
			Mode:            "agent",
			Outcome:         0,
			PromptVersion:   "prompt_v2.2",
			CompilerVersion: "v0.34.0",
			RecordedAt:      time.Now(),
		},
	}

	if err := AppendTrialHistory(ctx, db, entries); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Verify both versions are stamped correctly.
	var compVer string
	db.QueryRow("SELECT compiler_version FROM trial_history WHERE trial_id = ?", "trial_v1").Scan(&compVer)
	if compVer != "v0.35.0-dev" {
		t.Errorf("trial_v1 compiler_version: want v0.35.0-dev, got %q", compVer)
	}

	var promptVer string
	db.QueryRow("SELECT prompt_version FROM trial_history WHERE trial_id = ?", "trial_v2").Scan(&promptVer)
	if promptVer != "prompt_v2.2" {
		t.Errorf("trial_v2 prompt_version: want prompt_v2.2, got %q", promptVer)
	}
}
