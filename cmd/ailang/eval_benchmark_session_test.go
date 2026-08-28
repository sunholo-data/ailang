package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func TestRegisterStandardEvalSession_ResolvesOpenRouterSessionID(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "observatory.db")
	store, err := observatory.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("open observatory store: %v", err)
	}
	defer store.Close()

	chain, err := store.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType:    observatory.ChainSourceEvalSuite,
		SourceRef:     "standard-session-registration-test",
		WorkspacePath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create eval chain: %v", err)
	}
	stage, err := store.CreateStage(ctx, &observatory.StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "eval-standard:test-benchmark",
	})
	if err != nil {
		t.Fatalf("create eval stage: %v", err)
	}

	registerStandardEvalSession(ctx, &EvalChainContext{Store: store, ChainID: chain.ID}, stage.ID, t.TempDir())

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		t.Fatalf("open observatory resolver backend: %v", err)
	}
	defer backend.Close()
	gotChainID, gotStageID := backend.LookupChainBySessionID(ctx, chain.ID)
	if gotChainID != chain.ID || gotStageID != stage.ID {
		t.Fatalf("LookupChainBySessionID(%q) = (%q, %q), want (%q, %q)",
			chain.ID, gotChainID, gotStageID, chain.ID, stage.ID)
	}
}
