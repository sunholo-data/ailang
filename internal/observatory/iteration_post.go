package observatory

import (
	"context"
	"fmt"
	"time"
)

// Mission-loop iteration ingest (M-MISSION-COST-CHAINS, M2).
//
// The mission loop's spenders (headless claude, Agent-tool subs, codex, managed
// agents, quorum reviewers) never appeared in `ailang chains` — 48h of the heaviest
// fleet activity produced zero mission chains. M2 posts ONE chain per mission
// iteration whose stages are the loop's units of spend, so the M1 classifier can
// roll up mission cost.
//
// Design: `ailang chains` is offline-first (direct SQLite), and there is no HTTP
// endpoint for stage metrics — so PostIteration writes DIRECTLY to the observatory
// backend (CreateChain + CreateStage + UpdateStageMetrics + eval_assessment for the
// model). The CLI wraps this with a bounded+LOUD spool (see spool.go) so a DB
// failure never blocks the iteration.

// IterationStage is one spend unit of a mission iteration.
type IterationStage struct {
	// Role is the loop role (controller, sprint-executor, evaluator, …). Stored as
	// the stage agent_id, optionally suffixed with the quota bucket for quota lanes.
	Role string `json:"role"`
	// Provider is the executor provider (anthropic, codex, managed_agents, …).
	Provider string `json:"provider,omitempty"`
	// Model is the model that ran (metered lanes). Recorded via eval_assessment so
	// the M1 classifier can resolve a rate. Empty for quota lanes.
	Model string `json:"model,omitempty"`
	// CostUSD is the metered dollar cost (0 for quota lanes).
	CostUSD float64 `json:"cost_usd"`
	// TokensIn/TokensOut are token counts if known. Quota lanes MUST post 0/0 so
	// M1's tokens>0 estimation gate excludes them structurally (no schema marker).
	TokensIn  int `json:"tokens_in"`
	TokensOut int `json:"tokens_out"`
	// QuotaBucket, when set (fable|opus|sonnet|…), marks a subscription/quota lane.
	// It is encoded into the free-text agent_id as "<role> (quota:<bucket>)" so it
	// is visible in `chains view` with NO schema change.
	QuotaBucket string `json:"quota_bucket,omitempty"`
}

// IterationPost is one mission iteration to be posted as a chain.
type IterationPost struct {
	// Source is the chain source_ref, e.g. "mission:v1/iter-42"
	// (portable: "mission:<name>/iter-<N>").
	Source string `json:"source"`
	// SpooledAt records when this was buffered (set by the spool; informational).
	SpooledAt time.Time        `json:"spooled_at,omitempty"`
	Stages    []IterationStage `json:"stages"`
}

// agentIDFor builds the stage agent_id, encoding the quota bucket in free text for
// quota lanes so it is visible without a schema column.
func (st IterationStage) agentIDFor() string {
	if st.QuotaBucket != "" {
		return fmt.Sprintf("%s (quota:%s)", st.Role, st.QuotaBucket)
	}
	return st.Role
}

// Validate checks the post is well-formed before it hits storage (or the spool).
func (p *IterationPost) Validate() error {
	if p.Source == "" {
		return fmt.Errorf("iteration post: source is required")
	}
	if len(p.Stages) == 0 {
		return fmt.Errorf("iteration post: at least one stage is required")
	}
	for i, st := range p.Stages {
		if st.Role == "" {
			return fmt.Errorf("iteration post: stage %d has no role", i)
		}
		if st.QuotaBucket != "" && (st.TokensIn != 0 || st.TokensOut != 0 || st.CostUSD != 0) {
			return fmt.Errorf("iteration post: quota-lane stage %q must have zero tokens and cost (subscription spend is bucket-visible, not dollar-faked)", st.Role)
		}
	}
	return nil
}

// PostIteration writes one iteration chain and its stages to the observatory SQLite
// store. The chain and each stage are created; a per-stage failure aborts and
// returns an error (the caller spools the WHOLE post for retry). The quota bucket is
// encoded into agent_id. It takes *SQLiteBackend directly because it writes the
// stage model via eval_assessment (a store-level write not on the Backend interface)
// and because `ailang chains` is offline-first (direct SQLite).
func PostIteration(ctx context.Context, backend *SQLiteBackend, p *IterationPost) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	store := backend.Store()

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
		SourceRef:  p.Source,
	})
	if err != nil {
		return "", fmt.Errorf("create iteration chain: %w", err)
	}

	for i, st := range p.Stages {
		stage, err := store.CreateStage(ctx, &StageCreateRequest{
			ChainID:  chain.ID,
			AgentID:  st.agentIDFor(),
			Provider: Provider(st.Provider),
		})
		if err != nil {
			return chain.ID, fmt.Errorf("create stage %d (%s): %w", i, st.Role, err)
		}
		// Metrics: cost + tokens (quota lanes post zeros, which is a no-op add).
		if st.CostUSD != 0 || st.TokensIn != 0 || st.TokensOut != 0 {
			// "" = provenance not classified by this poster; reads as unknown.
			if err := store.UpdateStageMetrics(ctx, stage.ID, st.CostUSD, st.TokensIn, st.TokensOut, 0, 0, 0, ""); err != nil {
				return chain.ID, fmt.Errorf("update stage %d metrics: %w", i, err)
			}
		}
		// Record the model (metered lanes) so the M1 classifier can resolve a rate.
		if st.Model != "" {
			if err := store.UpdateStageEvalAssessment(ctx, stage.ID, &EvalAssessment{
				Model:    st.Model,
				EvalMode: "mission",
			}); err != nil {
				return chain.ID, fmt.Errorf("update stage %d model: %w", i, err)
			}
		}
	}

	return chain.ID, nil
}
