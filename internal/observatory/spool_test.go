package observatory

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func newTestSpool(t *testing.T) (*Spool, *bytes.Buffer) {
	t.Helper()
	sp := NewSpool(filepath.Join(t.TempDir(), "spool.jsonl"))
	var warnBuf bytes.Buffer
	sp.SetWarnWriter(&warnBuf)
	return sp, &warnBuf
}

func meteredPost(source string) *IterationPost {
	return &IterationPost{
		Source: source,
		Stages: []IterationStage{{Role: "executor", Model: "claude-sonnet-4-5", CostUSD: 0.1, TokensIn: 100, TokensOut: 50}},
	}
}

// Every buffering event is LOUD.
func TestSpool_AppendIsLoud(t *testing.T) {
	sp, warn := newTestSpool(t)
	if err := sp.Append(meteredPost("mission:v1/iter-1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	if !strings.Contains(warn.String(), "BUFFERING") {
		t.Errorf("expected loud BUFFERING notice, got: %q", warn.String())
	}
	if sp.Len() != 1 {
		t.Errorf("expected 1 buffered entry, got %d", sp.Len())
	}
}

// Entry cap is enforced with drop-oldest + a loud OVERFLOW notice.
func TestSpool_EntryCapDropsOldestLoudly(t *testing.T) {
	sp, warn := newTestSpool(t)
	sp.MaxEntries = 3

	for i := 0; i < 5; i++ {
		if err := sp.Append(meteredPost("mission:v1/iter-" + string(rune('a'+i)))); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	if got := sp.Len(); got != 3 {
		t.Fatalf("expected cap of 3 entries, got %d", got)
	}
	if !strings.Contains(warn.String(), "OVERFLOW") || !strings.Contains(warn.String(), "dropped") {
		t.Errorf("expected loud OVERFLOW/dropped notice, got: %q", warn.String())
	}

	// The retained entries are the NEWEST (drop-oldest): iter-c, iter-d, iter-e.
	entries, err := sp.Drain()
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(entries) != 3 || entries[0].Source != "mission:v1/iter-c" || entries[2].Source != "mission:v1/iter-e" {
		t.Errorf("drop-oldest wrong: got %v", func() []string {
			var s []string
			for _, e := range entries {
				s = append(s, e.Source)
			}
			return s
		}())
	}
}

// Size cap is enforced with drop-oldest.
func TestSpool_SizeCapDropsOldest(t *testing.T) {
	sp, warn := newTestSpool(t)
	sp.MaxEntries = 1000 // don't let entry cap fire
	sp.MaxBytes = 400    // small enough to force size-based drops

	for i := 0; i < 20; i++ {
		_ = sp.Append(meteredPost("mission:v1/iter-with-a-longish-source-" + string(rune('a'+i))))
	}
	// The persisted file must stay under (or near) the byte cap via drop-oldest.
	if sp.Len() >= 20 {
		t.Errorf("size cap did not drop anything: %d entries retained", sp.Len())
	}
	if !strings.Contains(warn.String(), "size cap") {
		t.Errorf("expected loud size-cap notice, got: %q", warn.String())
	}
}

// Drain clears the spool.
func TestSpool_DrainClears(t *testing.T) {
	sp, _ := newTestSpool(t)
	_ = sp.Append(meteredPost("mission:v1/iter-1"))
	if _, err := sp.Drain(); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if sp.Len() != 0 {
		t.Errorf("spool not cleared after drain: %d", sp.Len())
	}
	// Second drain on empty spool is a no-op.
	entries, err := sp.Drain()
	if err != nil || entries != nil {
		t.Errorf("empty drain should be nil/nil, got %v / %v", entries, err)
	}
}

// PostIteration writes a chain + stages the M1 classifier can then roll up.
func TestPostIteration_WritesChainAndStages(t *testing.T) {
	ensurePricingLoaded(t)
	db, store := setupTestDB(t)
	defer db.Close()
	backend := &SQLiteBackend{store: store}
	ctx := context.Background()

	post := &IterationPost{
		Source: "mission:v1/iter-99",
		Stages: []IterationStage{
			{Role: "codex-executor", Provider: "codex", Model: "claude-sonnet-4-5", CostUSD: 0.42, TokensIn: 1000, TokensOut: 500},
			{Role: "controller", QuotaBucket: "opus"},  // quota lane
			{Role: "evaluator", QuotaBucket: "sonnet"}, // quota lane
		},
	}
	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}
	if chainID == "" {
		t.Fatal("expected a chain ID")
	}

	// The mission rollup must see: metered $0.42 reported, 2 quota stages with buckets.
	rollups, err := store.GetMissionRollups(ctx, nil, "mission:", 5)
	if err != nil {
		t.Fatalf("GetMissionRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("expected 1 mission, got %d", len(rollups))
	}
	mr := rollups[0]
	if mr.Mission != "mission:v1" {
		t.Errorf("mission key: got %q", mr.Mission)
	}
	if mr.Rollup.ReportedCost != 0.42 {
		t.Errorf("metered total: got $%f, want $0.42", mr.Rollup.ReportedCost)
	}
	if mr.QuotaByBucket["opus"] != 1 || mr.QuotaByBucket["sonnet"] != 1 {
		t.Errorf("quota buckets: got %v", mr.QuotaByBucket)
	}
}

// A quota-lane stage with non-zero tokens/cost is rejected (subscription spend is
// bucket-visible, not dollar-faked).
func TestIterationPost_ValidateRejectsQuotaWithTokens(t *testing.T) {
	p := &IterationPost{
		Source: "mission:v1/iter-1",
		Stages: []IterationStage{{Role: "controller", QuotaBucket: "opus", TokensIn: 5}},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for quota lane with tokens")
	}
}
