package observatory

import (
	"context"
	"testing"
	"time"
)

// SUM over zero rows is NULL in SQL, and database/sql refuses to scan NULL into
// an int. Before the COALESCEs, an empty chains table — or a ?hours= window with
// no chains in it — made GET /api/chains/stats answer 500 (found by the
// dashboard contract test, internal/server/contract_test.go).
func TestAggregates_EmptyTablesAreZeroNotAnError(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	counts, err := store.GetChainStatusCounts(ctx, nil)
	if err != nil {
		t.Fatalf("GetChainStatusCounts on an empty table: %v", err)
	}
	if counts.Total != 0 || counts.Completed != 0 || counts.Active != 0 || counts.Pending != 0 || counts.Failed != 0 {
		t.Errorf("empty table: got %+v, want all zero", counts)
	}

	future := time.Now().Add(24 * time.Hour)
	if _, err := store.GetChainStatusCounts(ctx, &future); err != nil {
		t.Fatalf("GetChainStatusCounts with a window that matches nothing: %v", err)
	}

	stats, err := store.GetChainStats(ctx)
	if err != nil {
		t.Fatalf("GetChainStats on an empty table: %v", err)
	}
	if stats.TotalChains != 0 || stats.ActiveChains != 0 || stats.FailedChains != 0 {
		t.Errorf("empty table: got %+v, want zeros", stats)
	}

	backend, err := NewSQLiteBackend(store.DB())
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}
	summary, err := backend.GetFilteredMetricsSummary(ctx, &ControlPlaneFilter{Provider: "no-such-provider"}, nil)
	if err != nil {
		t.Fatalf("GetFilteredMetricsSummary matching no spans: %v", err)
	}
	if summary.ErrorCount != 0 || summary.TotalSpans != 0 {
		t.Errorf("no matching spans: got %+v, want zeros", summary)
	}
}
