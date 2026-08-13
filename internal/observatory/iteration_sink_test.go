package observatory

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
)

// M-MISSION-LOOP-UNIFIED-TELEMETRY M2/M3 — sink-level coverage of the iteration
// write path.
//
// The read-back tests in iteration_post_test.go go through real SQLite, which
// needs cgo. These exercise the SAME semantics through a fake sink so the
// accounting rules stay covered on any toolchain, and so the NODE-GENERIC claim
// is asserted rather than assumed: PostIterationTo must behave identically
// against a sink that is not the local SQLite store.

// fakeSink models the subset of store semantics PostIterationTo depends on,
// mirroring store_chains.go: CreateStage defaults to pending, UpdateStageMetrics
// ACCUMULATES, and a stage transitioning to completed bumps the chain's
// stages_completed counter.
type fakeSink struct {
	chains map[string]*ExecutionChain
	stages map[string]*ChainStage
	models map[string]string
	// calls is the ordered method log, so ordering claims are asserted.
	calls []string
	// failOn makes the named method return an error (outage simulation).
	failOn string
	nextID int
}

func newFakeSink() *fakeSink {
	return &fakeSink{
		chains: map[string]*ExecutionChain{},
		stages: map[string]*ChainStage{},
		models: map[string]string{},
	}
}

func (f *fakeSink) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%d", prefix, f.nextID)
}

func (f *fakeSink) record(method string) error {
	f.calls = append(f.calls, method)
	if f.failOn == method {
		return fmt.Errorf("fakeSink: %s unavailable", method)
	}
	return nil
}

func (f *fakeSink) CreateChain(_ context.Context, req *ChainCreateRequest) (*ExecutionChain, error) {
	if err := f.record("CreateChain"); err != nil {
		return nil, err
	}
	c := &ExecutionChain{ID: f.id("chain"), SourceType: req.SourceType, SourceRef: req.SourceRef}
	f.chains[c.ID] = c
	return c, nil
}

func (f *fakeSink) CreateStage(_ context.Context, req *StageCreateRequest) (*ChainStage, error) {
	if err := f.record("CreateStage"); err != nil {
		return nil, err
	}
	s := &ChainStage{
		ID:       f.id("stage"),
		ChainID:  req.ChainID,
		AgentID:  req.AgentID,
		Provider: req.Provider,
		Status:   StageStatusPending, // the default this milestone exists to move off
	}
	f.stages[s.ID] = s
	return s, nil
}

func (f *fakeSink) UpdateStageMetrics(_ context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, _ string) error {
	if err := f.record("UpdateStageMetrics"); err != nil {
		return err
	}
	s, ok := f.stages[stageID]
	if !ok {
		return fmt.Errorf("fakeSink: stage not found: %s", stageID)
	}
	s.Cost += cost
	s.TokensIn += tokensIn
	s.TokensOut += tokensOut
	s.Turns += turns
	s.ToolCalls += toolCalls
	s.DurationMs += durationMs
	return nil
}

func (f *fakeSink) UpdateStageStatus(_ context.Context, stageID string, status ChainStageStatus) error {
	if err := f.record("UpdateStageStatus"); err != nil {
		return err
	}
	s, ok := f.stages[stageID]
	if !ok {
		return fmt.Errorf("fakeSink: stage not found: %s", stageID)
	}
	s.Status = status
	if status == StageStatusCompleted {
		if c, ok := f.chains[s.ChainID]; ok {
			c.StagesCompleted++
		}
	}
	return nil
}

func (f *fakeSink) UpdateChainMetrics(_ context.Context, id string, cost float64, tokens, turns int) error {
	if err := f.record("UpdateChainMetrics"); err != nil {
		return err
	}
	c, ok := f.chains[id]
	if !ok {
		return fmt.Errorf("fakeSink: chain not found: %s", id)
	}
	c.TotalCost += cost
	c.TotalTokens += tokens
	c.TotalTurns += turns
	return nil
}

// modelFakeSink additionally records stage models (what SQLite can do and
// Firestore cannot).
type modelFakeSink struct{ *fakeSink }

func (f *modelFakeSink) UpdateStageEvalAssessment(_ context.Context, stageID string, a *EvalAssessment) error {
	if err := f.record("UpdateStageEvalAssessment"); err != nil {
		return err
	}
	f.models[stageID] = a.Model
	return nil
}

var (
	_ IterationSink      = (*fakeSink)(nil)
	_ IterationSink      = (*modelFakeSink)(nil)
	_ IterationModelSink = (*modelFakeSink)(nil)
)

func (f *fakeSink) stageByAgentID(agentID string) *ChainStage {
	for _, s := range f.stages {
		if s.AgentID == agentID {
			return s
		}
	}
	return nil
}

// iter190Post is the real measured shape: 4 stages, 3 providers, two stages
// carrying cost with ZERO tokens.
func iter190Post() *IterationPost {
	return &IterationPost{
		Source: "manual:mission:v1/iter-190",
		Stages: []IterationStage{
			{Role: "controller", Provider: "anthropic", QuotaBucket: "opus", Status: "completed"},
			{Role: "designer", Provider: "codex", QuotaBucket: "codex", Status: "completed"},
			{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5.6-sol", CostUSD: 0.0570, Status: "completed"},
			{Role: "quorum-r2", Provider: "openrouter", Model: "gemini-3.1-pro", CostUSD: 0.0507, Status: "completed"},
		},
	}
}

func TestPostIterationTo_StatusIsPerStage(t *testing.T) {
	sink := &modelFakeSink{newFakeSink()}
	post := iter190Post()
	post.Stages[1].Status = "failed" // the designer stage genuinely failed

	chainID, err := PostIterationTo(context.Background(), sink, post)
	if err != nil {
		t.Fatalf("PostIterationTo: %v", err)
	}

	want := map[string]ChainStageStatus{
		"controller (quota:opus)": StageStatusCompleted,
		"designer (quota:codex)":  StageStatusFailed,
		"quorum-r1":               StageStatusCompleted,
		"quorum-r2":               StageStatusCompleted,
	}
	for agentID, wantStatus := range want {
		st := sink.stageByAgentID(agentID)
		if st == nil {
			t.Fatalf("stage %q not written", agentID)
		}
		if st.Status != wantStatus {
			t.Errorf("stage %q status = %q, want %q", agentID, st.Status, wantStatus)
		}
	}

	// stages_completed must exclude the failed stage — a blanket transition would
	// read 4 here and hide the failure.
	if got := sink.chains[chainID].StagesCompleted; got != 3 {
		t.Errorf("StagesCompleted = %d, want 3 (the failed stage must not count)", got)
	}
}

func TestPostIterationTo_ChainTotalEqualsSumOfStages(t *testing.T) {
	sink := &modelFakeSink{newFakeSink()}
	post := iter190Post()
	// One metered stage that DOES carry tokens, so the assertion covers both the
	// cost-without-tokens rows and a normal one.
	post.Stages[3].TokensIn = 9000
	post.Stages[3].TokensOut = 1000

	chainID, err := PostIterationTo(context.Background(), sink, post)
	if err != nil {
		t.Fatalf("PostIterationTo: %v", err)
	}

	var stageCost float64
	var stageTokens int
	for _, s := range sink.stages {
		stageCost += s.Cost
		stageTokens += s.TokensIn + s.TokensOut
	}
	chain := sink.chains[chainID]
	if math.Abs(chain.TotalCost-stageCost) > 1e-9 {
		t.Errorf("chain TotalCost = %v, want %v (sum of stages)", chain.TotalCost, stageCost)
	}
	if chain.TotalTokens != stageTokens {
		t.Errorf("chain TotalTokens = %d, want %d (sum of stages)", chain.TotalTokens, stageTokens)
	}
	// The measured regression: iter-190 held $0.1077 and reported $0.0000.
	if math.Abs(chain.TotalCost-0.1077) > 1e-9 {
		t.Errorf("chain TotalCost = %v, want 0.1077 (the iter-190 figure)", chain.TotalCost)
	}
}

func TestPostIterationTo_OmittedStatusStaysPending(t *testing.T) {
	sink := &modelFakeSink{newFakeSink()}
	post := iter190Post()
	for i := range post.Stages {
		post.Stages[i].Status = "" // an unversioned payload from an older skill
	}

	chainID, err := PostIterationTo(context.Background(), sink, post)
	if err != nil {
		t.Fatalf("PostIterationTo on an unversioned payload: %v", err)
	}
	for _, s := range sink.stages {
		if s.Status != StageStatusPending {
			t.Errorf("stage %q status = %q, want pending (no status supplied)", s.AgentID, s.Status)
		}
	}
	for _, c := range sink.calls {
		if c == "UpdateStageStatus" {
			t.Error("UpdateStageStatus called for a payload that supplied no status")
		}
	}
	// Aggregation is independent of status and still happens.
	if math.Abs(sink.chains[chainID].TotalCost-0.1077) > 1e-9 {
		t.Errorf("TotalCost = %v, want 0.1077", sink.chains[chainID].TotalCost)
	}
}

// TestPostIterationTo_StatusWrittenAfterMetrics pins the ordering: the status
// transition stamps completed_at, so it must land after the stage is credited.
func TestPostIterationTo_StatusWrittenAfterMetrics(t *testing.T) {
	sink := &modelFakeSink{newFakeSink()}
	if _, err := PostIterationTo(context.Background(), sink, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5.6-sol", CostUSD: 0.05, TokensIn: 10, TokensOut: 5, Status: "completed"}},
	}); err != nil {
		t.Fatalf("PostIterationTo: %v", err)
	}

	got := strings.Join(sink.calls, ",")
	want := "CreateChain,CreateStage,UpdateStageMetrics,UpdateStageEvalAssessment,UpdateStageStatus,UpdateChainMetrics"
	if got != want {
		t.Errorf("call order = %q, want %q", got, want)
	}
}

// TestPostIterationTo_SinkWithoutModelSupport is the node-generic case: the
// Firestore observatory does not implement eval_assessment. Dropping the model
// must not fail the post (the cloud leg would never succeed) and must not be
// silent (PostIterationTo reports it on stderr).
func TestPostIterationTo_SinkWithoutModelSupport(t *testing.T) {
	sink := newFakeSink() // no UpdateStageEvalAssessment
	chainID, err := PostIterationTo(context.Background(), sink, iter190Post())
	if err != nil {
		t.Fatalf("PostIterationTo against a model-less sink: %v", err)
	}
	if math.Abs(sink.chains[chainID].TotalCost-0.1077) > 1e-9 {
		t.Errorf("TotalCost = %v, want 0.1077 — accounting must not depend on model support", sink.chains[chainID].TotalCost)
	}
	for _, c := range sink.calls {
		if c == "UpdateStageEvalAssessment" {
			t.Error("UpdateStageEvalAssessment called on a sink that does not implement it")
		}
	}
}

// TestPostIterationTo_WriteFailureIsReported keeps the spool contract honest: a
// mid-post failure must surface as an error so the caller buffers the WHOLE post.
func TestPostIterationTo_WriteFailureIsReported(t *testing.T) {
	for _, method := range []string{"CreateChain", "CreateStage", "UpdateStageMetrics", "UpdateStageStatus", "UpdateChainMetrics"} {
		t.Run(method, func(t *testing.T) {
			sink := &modelFakeSink{newFakeSink()}
			sink.failOn = method
			if _, err := PostIterationTo(context.Background(), sink, iter190Post()); err == nil {
				t.Errorf("PostIterationTo = nil error, want a failure when %s is unavailable", method)
			}
		})
	}
}

func TestPostIterationTo_ValidationRejectsBadPost(t *testing.T) {
	sink := &modelFakeSink{newFakeSink()}
	if _, err := PostIterationTo(context.Background(), sink, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{{Role: "controller", Status: "finished"}},
	}); err == nil {
		t.Error("PostIterationTo accepted an unknown stage status; want a loud rejection")
	}
	if len(sink.calls) != 0 {
		t.Errorf("validation ran after writes began: %v", sink.calls)
	}
}
