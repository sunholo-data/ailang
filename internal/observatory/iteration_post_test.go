package observatory

import (
	"context"
	"testing"
)

// Mission stage accounting (M-MISSION-LOOP-UNIFIED-TELEMETRY, M2).
//
// The defect these tests pin: a mission iteration spanning three providers read
// back four `pending` stages, 0 tokens against non-zero cost, and a $0.0000 chain
// total while holding $0.1077 across its own stages (measured on
// mission:v1/iter-190). iter190Post below is that exact shape.

func newIterationBackend(t *testing.T) *SQLiteBackend {
	t.Helper()
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("open in-memory observatory: %v", err)
	}
	t.Cleanup(func() { _ = backend.Close() })
	return backend
}

// iter190Post is the real iter-190 shape: 4 stages, 3 providers
// (anthropic/openai/openrouter), two of them carrying cost with ZERO tokens —
// which is what made the aggregate look empty while money was being spent.
func iter190Post() *IterationPost {
	return &IterationPost{
		Source: "mission:v1/iter-190",
		Stages: []IterationStage{
			{Role: "controller", Provider: "anthropic", QuotaBucket: "opus", Status: "completed"},
			{Role: "designer", Provider: "openai", QuotaBucket: "codex", Status: "completed"},
			{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5", CostUSD: 0.0570, Status: "completed"},
			{Role: "quorum-r2", Provider: "openrouter", Model: "gpt-5", CostUSD: 0.0507, Status: "completed"},
		},
	}
}

func readStages(t *testing.T, backend *SQLiteBackend, chainID string) map[string]*ChainStage {
	t.Helper()
	stages, err := backend.Store().GetChainStages(context.Background(), chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read stages: %v", err)
	}
	byAgent := make(map[string]*ChainStage, len(stages))
	for _, st := range stages {
		byAgent[st.AgentID] = st
	}
	return byAgent
}

// TestPostIteration_TerminalStatusReadsBack is the primary M2 criterion: a stage
// that posts an outcome must NOT read back `pending`.
func TestPostIteration_TerminalStatusReadsBack(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	chainID, err := PostIteration(ctx, backend, iter190Post())
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	stages := readStages(t, backend, chainID)
	if len(stages) != 4 {
		t.Fatalf("expected 4 stages, got %d", len(stages))
	}
	for agentID, st := range stages {
		if st.Status != StageStatusCompleted {
			t.Errorf("stage %q: status = %q, want %q", agentID, st.Status, StageStatusCompleted)
		}
		if st.CompletedAt == nil {
			t.Errorf("stage %q: completed_at not set on a terminal stage", agentID)
		}
	}
}

// TestPostIteration_FailedStageReadsBackFailed blocks the shortcut. Setting every
// stage to `completed` would satisfy "nothing stays pending" while hiding real
// failures, so a failed stage must survive the round trip AS FAILED — and the
// chain must not claim success around it.
func TestPostIteration_FailedStageReadsBackFailed(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-191"
	post.Stages[3].Status = "failed" // quorum-r2 fell over

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	stages := readStages(t, backend, chainID)
	if got := stages["quorum-r2"].Status; got != StageStatusFailed {
		t.Errorf("failed stage read back as %q, want %q", got, StageStatusFailed)
	}
	// The other three are untouched — a failure does not smear across the chain.
	for _, agentID := range []string{"controller (quota:opus)", "designer (quota:codex)", "quorum-r1"} {
		if got := stages[agentID].Status; got != StageStatusCompleted {
			t.Errorf("stage %q read back as %q, want %q", agentID, got, StageStatusCompleted)
		}
	}

	chain, err := backend.Store().GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	if chain.Status != ChainStatusFailed {
		t.Errorf("chain with a failed stage: status = %q, want %q", chain.Status, ChainStatusFailed)
	}
}

// TestPostIteration_TokensReadBack asserts the caller-side path rather than
// assuming it: the writer already forwards tokens, so a post that CARRIES them
// must show them. The zeros in iter-190 came from the poster, not from here.
func TestPostIteration_TokensReadBack(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-192"
	post.Stages[2].TokensIn = 18_432
	post.Stages[2].TokensOut = 2_101

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	st := readStages(t, backend, chainID)["quorum-r1"]
	if st.TokensIn != 18_432 || st.TokensOut != 2_101 {
		t.Errorf("tokens read back as in=%d out=%d, want in=18432 out=2101", st.TokensIn, st.TokensOut)
	}
	if st.Cost != 0.0570 {
		t.Errorf("cost read back as %v, want 0.0570", st.Cost)
	}
}

// TestPostIteration_ChainTotalsAggregateStages is the iter-190 headline: the chain
// reported $0.0000 while holding $0.1077 across its stages.
func TestPostIteration_ChainTotalsAggregateStages(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Stages[2].TokensIn, post.Stages[2].TokensOut = 18_432, 2_101
	post.Stages[3].TokensIn, post.Stages[3].TokensOut = 15_004, 1_877

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	chain, err := backend.Store().GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}

	// Sum the stages as stored, so the assertion is "total == sum of parts"
	// rather than a second hand-maintained constant.
	var wantCost float64
	var wantTokens int
	for _, st := range readStages(t, backend, chainID) {
		wantCost += st.Cost
		wantTokens += st.TokensIn + st.TokensOut
	}
	if diff := chain.TotalCost - wantCost; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("chain total_cost = %v, want %v (sum of stages)", chain.TotalCost, wantCost)
	}
	if chain.TotalTokens != wantTokens {
		t.Errorf("chain total_tokens = %d, want %d (sum of stages)", chain.TotalTokens, wantTokens)
	}
	if chain.TotalCost == 0 {
		t.Error("chain total_cost is 0 while its stages hold real spend — the iter-190 defect")
	}
	// Two stages carry cost with zero tokens; that must not zero the money.
	if chain.StagesCompleted != 4 {
		t.Errorf("stages_completed = %d, want 4", chain.StagesCompleted)
	}
	if chain.Status != ChainStatusCompleted {
		t.Errorf("all-completed chain: status = %q, want %q", chain.Status, ChainStatusCompleted)
	}
}

// TestPostIteration_VersionSkew_StatusOmitted pins the compatibility contract: the
// mission-control skill and this CLI ship independently, so a payload written
// before Status existed must keep working and behave exactly as it did then.
func TestPostIteration_VersionSkew_StatusOmitted(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-193"
	for i := range post.Stages {
		post.Stages[i].Status = "" // the pre-v0.33.2 payload
	}

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration with no statuses must not error: %v", err)
	}

	for agentID, st := range readStages(t, backend, chainID) {
		if st.Status != StageStatusPending {
			t.Errorf("stage %q: status = %q, want %q (unchanged legacy behaviour)",
				agentID, st.Status, StageStatusPending)
		}
	}

	// Aggregation is NOT gated on status — an old payload still gets its money
	// counted, which is the half of M2 that works without a skill update.
	chain, err := backend.Store().GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	if diff := chain.TotalCost - 0.1077; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("chain total_cost = %v, want 0.1077", chain.TotalCost)
	}
	// No stage reported an outcome, so the chain must not claim one.
	if chain.Status != ChainStatusActive {
		t.Errorf("chain with no reported outcomes: status = %q, want %q", chain.Status, ChainStatusActive)
	}
}

// TestPostIteration_PartialStatusLeavesChainActive: a mid-flight iteration must
// not read as finished just because some stages are done.
func TestPostIteration_PartialStatusLeavesChainActive(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-194"
	post.Stages[3].Status = "running"

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	stages := readStages(t, backend, chainID)
	if got := stages["quorum-r2"].Status; got != StageStatusRunning {
		t.Errorf("running stage read back as %q, want %q", got, StageStatusRunning)
	}
	chain, err := backend.Store().GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	if chain.Status != ChainStatusActive {
		t.Errorf("chain with a running stage: status = %q, want %q", chain.Status, ChainStatusActive)
	}
}

// TestIterationPost_ValidateStatus: an unknown status is rejected loudly rather
// than coerced — coercion would report an outcome the caller never claimed.
func TestIterationPost_ValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"empty is the version-skew path", "", false},
		{"pending", "pending", false},
		{"running", "running", false},
		{"awaiting_approval", "awaiting_approval", false},
		{"completed", "completed", false},
		{"failed", "failed", false},
		{"unknown word", "done", true},
		{"wrong case", "COMPLETED", true},
		{"chain vocabulary is not stage vocabulary", "pending_approval", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &IterationPost{
				Source: "mission:v1/iter-195",
				Stages: []IterationStage{{Role: "controller", QuotaBucket: "opus", Status: tt.status}},
			}
			err := post.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
