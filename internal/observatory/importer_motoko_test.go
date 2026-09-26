package observatory

import (
	"context"
	"path/filepath"
	"testing"
)

// The importer shares the executor's SessionEvent decoder (M-V1-SIMPLIFY-S3
// M5). Its private copy had no cache-token fields, so an imported chain
// reported cache_read=0 for a run the adapter had measured at 128 per step.
// The fixture is the executor's own: a thinking event carrying 128 cache-read
// tokens on its second step (the other 128 in the fixture is the
// run_summary usage block), one tool call, a run_summary.
func TestImportMotokoSessionKeepsCacheTokens(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	fixture := filepath.Join("..", "executor", "motoko", "testdata", "session_success.jsonl")
	res, err := store.ImportMotokoSession(context.Background(), fixture)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.CacheReadTokens != 128 {
		t.Errorf("CacheReadTokens = %d, want 128 (step 1's cache_read_input_tokens)", res.CacheReadTokens)
	}
	if res.ToolCalls != 1 {
		t.Errorf("ToolCalls = %d, want 1", res.ToolCalls)
	}
	if res.Status != "completed" || res.FinishReason != "stop" {
		t.Errorf("status/finish = %s/%s, want completed/stop", res.Status, res.FinishReason)
	}
	if res.TokensIn == 0 || res.PeakInput == 0 {
		t.Errorf("token totals not decoded: in=%d peak=%d", res.TokensIn, res.PeakInput)
	}

	// Idempotent: a second import of the same run replaces, not duplicates.
	again, err := store.ImportMotokoSession(context.Background(), fixture)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if again.ChainID != res.ChainID {
		t.Errorf("re-import changed the chain id: %s vs %s", again.ChainID, res.ChainID)
	}
	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM execution_chains WHERE id = ?`, res.ChainID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("chain rows = %d, want 1", n)
	}
}
