package observatory

import (
	"context"
	"strings"
	"testing"
)

// Quota-lane token accounting (M-QUOTA-RATIONING-ROUTING M2).
//
// The defect these pin: 4,979 quota stages recorded zero tokens BY DESIGN, because
// `tokens > 0` is the fleet's structural marker for "metered and priceable" and a quota
// lane must post 0/0 to stay out of the cost estimator. So nothing could answer "how
// much of the codex bucket have we spent?" — the one question a ration exists to answer.
// QuotaTokens is a SEPARATE field precisely so fixing the ration cannot corrupt the cost.

// A quota lane's spend must read back, without touching the metered fields.
func TestPostIteration_QuotaTokensReadBackWithoutMeteredTokens(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-338"
	post.Stages[1].QuotaTokens = 999_376 // the measured iter-338 codex controller session

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}
	st := readStages(t, backend, chainID)["designer (quota:codex)"]
	if st == nil {
		t.Fatal("quota stage not found")
	}
	if st.QuotaTokens != 999_376 {
		t.Errorf("quota_tokens read back as %d, want 999376", st.QuotaTokens)
	}
	// The whole point: the estimator's marker must stay clear.
	if st.TokensIn != 0 || st.TokensOut != 0 {
		t.Errorf("quota stage polluted the metered fields: in=%d out=%d, want 0/0", st.TokensIn, st.TokensOut)
	}
	if st.Cost != 0 {
		t.Errorf("quota stage reported cost %v, want 0 — a subscription run was never billed", st.Cost)
	}
}

// A quota stage's tokens must NOT inflate the chain's metered total, or every mission
// iteration's cost KPI silently absorbs subscription spend.
func TestPostIteration_QuotaTokensStayOutOfTheChainTotal(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-339"
	post.Stages[1].QuotaTokens = 500_000
	post.Stages[2].TokensIn = 100
	post.Stages[2].TokensOut = 50

	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}
	chain, err := backend.Store().GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	if chain.TotalTokens != 150 {
		t.Errorf("chain total tokens = %d, want 150 (metered only; quota spend must not be summed here)", chain.TotalTokens)
	}
}

// The converse guard. A stage carrying BOTH would be counted once by the estimator and
// once by the ration, as if it were two runs.
func TestIterationPost_ValidateRejectsQuotaTokensWithoutABucket(t *testing.T) {
	p := &IterationPost{
		Source: "mission:v1/iter-1",
		Stages: []IterationStage{
			{Role: "quorum-r1", Model: "gpt-5", TokensIn: 10, TokensOut: 5, QuotaTokens: 4242},
		},
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("a metered stage carrying quota_tokens was accepted; it would be double-counted")
	}
	if !strings.Contains(err.Error(), "quota_bucket") {
		t.Errorf("error %q does not say what is wrong", err)
	}
	// Control: the same stage without quota_tokens must validate, proving the rejection
	// comes from the field and not from the rest of the shape.
	p.Stages[0].QuotaTokens = 0
	if err := p.Validate(); err != nil {
		t.Fatalf("control failed: %v", err)
	}
}

func TestIterationPost_ValidateRejectsNegativeQuotaTokens(t *testing.T) {
	p := &IterationPost{
		Source: "mission:v1/iter-1",
		Stages: []IterationStage{{Role: "designer", QuotaBucket: "codex", QuotaTokens: -1}},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("negative quota_tokens accepted")
	}
	p.Stages[0].QuotaTokens = 1
	if err := p.Validate(); err != nil {
		t.Fatalf("control failed: %v", err)
	}
}

// A zero write is indistinguishable from never reporting — every quota stage already
// reads zero — so the store must refuse it rather than record a false "measured 0".
func TestUpdateStageQuotaTokens_RefusesZeroAndUnknownStage(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := iter190Post()
	post.Source = "mission:v1/iter-340"
	chainID, err := PostIteration(ctx, backend, post)
	if err != nil {
		t.Fatalf("PostIteration: %v", err)
	}
	stageID := readStages(t, backend, chainID)["designer (quota:codex)"].ID

	if err := backend.UpdateStageQuotaTokens(ctx, stageID, 0); err == nil {
		t.Error("a zero quota-token write was accepted")
	}
	if err := backend.UpdateStageQuotaTokens(ctx, "no-such-stage", 100); err == nil {
		t.Error("a write against an unknown stage silently succeeded; the ledger would be short a whole stage with no error anywhere")
	}
	// Control.
	if err := backend.UpdateStageQuotaTokens(ctx, stageID, 100); err != nil {
		t.Errorf("control write failed: %v", err)
	}
}

// The rollup's ration sum must come from quota_tokens. Summing TokensIn+TokensOut
// measured zero for all 4,979 quota stages, which is the bug this milestone closes.
func TestMissionRollup_QuotaTokensAreSummedFromTheQuotaField(t *testing.T) {
	backend := newIterationBackend(t)
	ctx := context.Background()

	post := &IterationPost{
		Source: "mission:v1/iter-341",
		Stages: []IterationStage{
			{Role: "controller", Provider: "anthropic", QuotaBucket: "opus", QuotaTokens: 1000, Status: "completed"},
			{Role: "designer", Provider: "openai", QuotaBucket: "codex", QuotaTokens: 2000, Status: "completed"},
			{Role: "planner", Provider: "openai", QuotaBucket: "codex-oauth", QuotaTokens: 500, Status: "completed"},
			{Role: "quorum", Provider: "openrouter", Model: "gpt-5", CostUSD: 0.05, TokensIn: 700, TokensOut: 300, Status: "completed"},
		},
	}
	if _, err := PostIteration(ctx, backend, post); err != nil {
		t.Fatalf("PostIteration: %v", err)
	}

	rollups, err := backend.Store().GetMissionRollups(ctx, nil, "mission:v1", 10)
	if err != nil {
		t.Fatalf("MissionRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("rollups = %d, want 1", len(rollups))
	}
	got := rollups[0].QuotaTokensByBucket

	// codex and codex-oauth fold to one canonical bucket, because they are one
	// subscription — a ration that splits them measures each half as within budget.
	if got["codex"] != 2500 {
		t.Errorf("codex quota tokens = %d, want 2500 (codex + codex-oauth)", got["codex"])
	}
	if got["anthropic"] != 1000 {
		t.Errorf("anthropic quota tokens = %d, want 1000", got["anthropic"])
	}
	// The metered stage must not appear in the ration sum at all.
	for bucket, n := range got {
		if bucket == "openrouter" || n == 1000+2000+500+1000 {
			t.Errorf("metered stage leaked into the ration sum: %s=%d", bucket, n)
		}
	}
	// Control: the metered stage IS counted by the cost rollup, so the two systems are
	// both live and merely separate.
	if rollups[0].Rollup.ReportedCost == 0 && rollups[0].Rollup.ReportedTokens == 0 {
		t.Error("the metered stage vanished from the cost rollup entirely")
	}
}
