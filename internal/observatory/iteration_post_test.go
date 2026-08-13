package observatory

import (
	"context"
	"math"
	"testing"
)

// M-MISSION-LOOP-UNIFIED-TELEMETRY M2 — mission stage accounting.
//
// Two defects with DIFFERENT owners, which is why they get separate tests:
//
//   - writer-side: PostIteration created stages and never transitioned them, so
//     every stage kept CreateStage's StageStatusPending default;
//   - aggregation: the chain total was never credited, so iter-190 read $0.0000
//     while holding $0.1077 of stage cost.
//
// The load-bearing test here is TestPostIteration_FailedStageReadsBackFailed:
// blanket-completing every stage would satisfy "no stage remains pending" AND
// hide real failures. That is an acceptance criterion, not a preference.

// postToMemory posts p to a fresh in-memory observatory and returns the chain id
// plus the store, so tests assert on what was READ BACK rather than on what the
// poster believed it wrote.
func postToMemory(t *testing.T, p *IterationPost) (string, *Store) {
	t.Helper()
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("open in-memory observatory: %v", err)
	}
	t.Cleanup(func() { _ = backend.Close() })

	chainID, err := PostIteration(context.Background(), backend, p)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}
	return chainID, backend.Store()
}

// stagesByRole reads the chain's stages back and indexes them by agent_id.
func stagesByRole(t *testing.T, store *Store, chainID string) map[string]*ChainStage {
	t.Helper()
	stages, err := store.GetChainStages(context.Background(), chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("GetChainStages: %v", err)
	}
	out := make(map[string]*ChainStage, len(stages))
	for _, st := range stages {
		out[st.AgentID] = st
	}
	return out
}

func TestPostIteration_TerminalStatusReadsBack(t *testing.T) {
	chainID, store := postToMemory(t, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{
			{Role: "controller", Provider: "anthropic", QuotaBucket: "opus", Status: "completed"},
			{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5.6-sol", CostUSD: 0.0570, Status: "completed"},
		},
	})

	for _, agentID := range []string{"controller (quota:opus)", "quorum-r1"} {
		st, ok := stagesByRole(t, store, chainID)[agentID]
		if !ok {
			t.Fatalf("stage %q not found", agentID)
		}
		if st.Status != StageStatusCompleted {
			t.Errorf("stage %q status = %q, want %q (a posted terminal status must not read back pending)",
				agentID, st.Status, StageStatusCompleted)
		}
	}
}

// TestPostIteration_FailedStageReadsBackFailed is the criterion that blocks the
// shortcut: setting every stage to completed would satisfy "no stage remains
// pending" while hiding real failures.
func TestPostIteration_FailedStageReadsBackFailed(t *testing.T) {
	chainID, store := postToMemory(t, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{
			{Role: "controller", QuotaBucket: "opus", Status: "completed"},
			{Role: "designer", QuotaBucket: "codex", Status: "failed"},
			{Role: "quorum-r1", Provider: "openrouter", CostUSD: 0.0570, Status: "completed"},
		},
	})

	got := stagesByRole(t, store, chainID)
	if st := got["designer (quota:codex)"]; st == nil || st.Status != StageStatusFailed {
		var have ChainStageStatus
		if st != nil {
			have = st.Status
		}
		t.Errorf("failed stage status = %q, want %q — a failed stage must still say so", have, StageStatusFailed)
	}
	// And the surrounding stages keep their own outcome (no blanket transition).
	if st := got["controller (quota:opus)"]; st == nil || st.Status != StageStatusCompleted {
		t.Errorf("controller status = %v, want completed", st)
	}

	// stages_completed counts the completed stages only, not the failed one.
	chain, err := store.GetChain(context.Background(), chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("GetChain: %v", err)
	}
	if chain.StagesCompleted != 2 {
		t.Errorf("chain.StagesCompleted = %d, want 2 (the failed stage must not count as completed)", chain.StagesCompleted)
	}
}

func TestPostIteration_TokensReadBack(t *testing.T) {
	chainID, store := postToMemory(t, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{
			{Role: "executor", Provider: "codex", Model: "gpt-5.6-sol", CostUSD: 0.42, TokensIn: 18000, TokensOut: 2000, Status: "completed"},
		},
	})

	st := stagesByRole(t, store, chainID)["executor"]
	if st == nil {
		t.Fatal("executor stage not found")
	}
	if st.TokensIn != 18000 || st.TokensOut != 2000 {
		t.Errorf("stage tokens = %d in / %d out, want 18000 / 2000", st.TokensIn, st.TokensOut)
	}
	if math.Abs(st.Cost-0.42) > 1e-9 {
		t.Errorf("stage cost = %v, want 0.42", st.Cost)
	}
}

// TestPostIteration_ChainTotalEqualsSumOfStages is the iter-190 regression
// fixture: 4 stages, 3 providers, and two stages carrying cost with ZERO tokens
// (the real measured shape). The chain reported $0.0000 while its stages held
// $0.1077.
func TestPostIteration_ChainTotalEqualsSumOfStages(t *testing.T) {
	stages := []IterationStage{
		{Role: "controller", Provider: "anthropic", QuotaBucket: "opus", Status: "completed"},
		{Role: "designer", Provider: "codex", QuotaBucket: "codex", Status: "completed"},
		{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5.6-sol", CostUSD: 0.0570, Status: "completed"},
		{Role: "quorum-r2", Provider: "openrouter", Model: "gemini-3.1-pro", CostUSD: 0.0507, TokensIn: 9000, TokensOut: 1000, Status: "completed"},
	}
	chainID, store := postToMemory(t, &IterationPost{Source: "manual:mission:v1/iter-190", Stages: stages})

	var wantCost float64
	var wantTokens int
	for _, st := range stages {
		wantCost += st.CostUSD
		wantTokens += st.TokensIn + st.TokensOut
	}

	chain, err := store.GetChain(context.Background(), chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("GetChain: %v", err)
	}
	if math.Abs(chain.TotalCost-wantCost) > 1e-9 {
		t.Errorf("chain.TotalCost = %v, want %v (sum of stage costs)", chain.TotalCost, wantCost)
	}
	if chain.TotalTokens != wantTokens {
		t.Errorf("chain.TotalTokens = %d, want %d (sum of stage tokens)", chain.TotalTokens, wantTokens)
	}

	// Cross-check against what the stages themselves read back, so the assertion
	// is "total == sum of stored stages", not "total == sum of my fixture".
	var storedCost float64
	var storedTokens int
	for _, st := range stagesByRole(t, store, chainID) {
		storedCost += st.Cost
		storedTokens += st.TokensIn + st.TokensOut
	}
	if math.Abs(chain.TotalCost-storedCost) > 1e-9 || chain.TotalTokens != storedTokens {
		t.Errorf("chain total (%v, %d) != sum of stored stages (%v, %d)",
			chain.TotalCost, chain.TotalTokens, storedCost, storedTokens)
	}
}

// TestPostIteration_OmittedStatusKeepsTodaysBehaviour covers version skew: the
// mission-control skill and the CLI ship independently, so a payload written
// before Status existed must keep working and must not be silently invented.
func TestPostIteration_OmittedStatusKeepsTodaysBehaviour(t *testing.T) {
	chainID, store := postToMemory(t, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{
			{Role: "controller", QuotaBucket: "opus"},
			{Role: "quorum-r1", Provider: "openrouter", CostUSD: 0.0570},
		},
	})

	for agentID, st := range stagesByRole(t, store, chainID) {
		if st.Status != StageStatusPending {
			t.Errorf("stage %q status = %q, want %q — an omitted status must not be invented",
				agentID, st.Status, StageStatusPending)
		}
	}
	// Aggregation still happens: it does not depend on status being supplied.
	chain, err := store.GetChain(context.Background(), chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("GetChain: %v", err)
	}
	if math.Abs(chain.TotalCost-0.0570) > 1e-9 {
		t.Errorf("chain.TotalCost = %v, want 0.0570", chain.TotalCost)
	}
}

func TestIterationPost_ValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"empty is allowed (version skew)", "", false},
		{"pending", "pending", false},
		{"running", "running", false},
		{"awaiting_approval", "awaiting_approval", false},
		{"completed", "completed", false},
		{"failed", "failed", false},
		{"unknown status is rejected loudly", "finished", true},
		{"case mismatch is rejected loudly", "COMPLETED", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IterationPost{
				Source: "mission:v1/iter-191",
				Stages: []IterationStage{{Role: "controller", QuotaBucket: "opus", Status: tt.status}},
			}
			err := p.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil, want an error for status %q", tt.status)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil for status %q", err, tt.status)
			}
		})
	}
}
