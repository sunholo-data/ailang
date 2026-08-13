package observatory

import (
	"context"
	"fmt"
	"os"
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
	// Status is the stage's OWN outcome (M-MISSION-LOOP-UNIFIED-TELEMETRY M2).
	// One of the ChainStageStatus vocabulary: pending, running, awaiting_approval,
	// completed, failed.
	//
	// It is deliberately PER-STAGE and never defaulted to "completed": blanket-
	// completing an iteration's stages would satisfy "no stage remains pending"
	// while hiding a stage that genuinely failed.
	//
	// EMPTY IS VALID and means "not supplied" — the stage keeps CreateStage's
	// pending default, which is exactly today's behaviour. The mission-control
	// skill and this CLI ship independently, so a payload written before Status
	// existed must keep working rather than have an outcome invented for it.
	Status string `json:"status,omitempty"`
}

// validStageStatuses is the accepted Status vocabulary. An unrecognised value is
// REJECTED rather than coerced: a status is an outcome claim, and silently
// mapping an unknown one onto "completed" is the same failure-hiding this
// milestone exists to prevent.
var validStageStatuses = map[string]ChainStageStatus{
	string(StageStatusPending):          StageStatusPending,
	string(StageStatusRunning):          StageStatusRunning,
	string(StageStatusAwaitingApproval): StageStatusAwaitingApproval,
	string(StageStatusCompleted):        StageStatusCompleted,
	string(StageStatusFailed):           StageStatusFailed,
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
		if st.Status != "" {
			if _, ok := validStageStatuses[st.Status]; !ok {
				return fmt.Errorf("iteration post: stage %q has unknown status %q (valid: pending, running, awaiting_approval, completed, failed; omit for today's default)", st.Role, st.Status)
			}
		}
	}
	return nil
}

// IterationSink is the narrow write surface an iteration post needs
// (M-MISSION-LOOP-UNIFIED-TELEMETRY M3). Every method on it is already part of
// observatory.Backend, so the local SQLite Store AND the Firestore
// ObservatoryStore both satisfy it — which is what makes the write path
// NODE-GENERIC: the node picks a sink, the poster does not know which node it is
// running on.
type IterationSink interface {
	CreateChain(ctx context.Context, req *ChainCreateRequest) (*ExecutionChain, error)
	CreateStage(ctx context.Context, req *StageCreateRequest) (*ChainStage, error)
	UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error
	UpdateStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error
	UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error
}

// IterationModelSink is the OPTIONAL extension for recording a stage's model.
// UpdateStageEvalAssessment is a *Store method and is deliberately NOT on the
// Backend interface, so the Firestore observatory does not implement it. A sink
// that cannot record models is not an error — the local leg of a dual-write
// still records them, so the datum is not lost system-wide — but it is never
// silent either: PostIterationTo says so on stderr.
type IterationModelSink interface {
	UpdateStageEvalAssessment(ctx context.Context, stageID string, assessment *EvalAssessment) error
}

// PostIteration writes one iteration chain and its stages to the LOCAL observatory
// SQLite store. It is the offline-first default (`ailang chains` reads SQLite
// directly); PostIterationTo is the same write against any sink.
func PostIteration(ctx context.Context, backend *SQLiteBackend, p *IterationPost) (string, error) {
	return PostIterationTo(ctx, backend.Store(), p)
}

// PostIterationTo writes one iteration chain and its stages to sink. The chain and
// each stage are created; a per-stage failure aborts and returns an error (the
// caller spools the WHOLE post for retry). The quota bucket is encoded into
// agent_id.
func PostIterationTo(ctx context.Context, store IterationSink, p *IterationPost) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
		SourceRef:  p.Source,
	})
	if err != nil {
		return "", fmt.Errorf("create iteration chain: %w", err)
	}

	// Chain totals are denormalized counters that nothing else credits for a
	// mission iteration, which is why iter-190 read $0.0000 while its stages held
	// $0.1077. Accumulate here and post ONE UpdateChainMetrics after the stages.
	var totalCost float64
	var totalTokens int

	// A sink that cannot record models (Firestore) drops them; count and report
	// rather than dropping quietly.
	modelSink, canRecordModel := store.(IterationModelSink)
	modelsDropped := 0

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
			switch {
			case canRecordModel:
				if err := modelSink.UpdateStageEvalAssessment(ctx, stage.ID, &EvalAssessment{
					Model:    st.Model,
					EvalMode: "mission",
				}); err != nil {
					return chain.ID, fmt.Errorf("update stage %d model: %w", i, err)
				}
			default:
				modelsDropped++
			}
		}
		// Status LAST, so completed_at lands after the stage is fully credited.
		// Only when supplied — see IterationStage.Status on why an absent status
		// is left pending rather than assumed complete.
		if st.Status != "" {
			if err := store.UpdateStageStatus(ctx, stage.ID, validStageStatuses[st.Status]); err != nil {
				return chain.ID, fmt.Errorf("update stage %d status: %w", i, err)
			}
		}

		totalCost += st.CostUSD
		totalTokens += st.TokensIn + st.TokensOut
	}

	// Aggregate the stages into the chain total. Turns are not modelled per
	// iteration stage, so 0 is the honest value rather than a guess.
	if totalCost != 0 || totalTokens != 0 {
		if err := store.UpdateChainMetrics(ctx, chain.ID, totalCost, totalTokens, 0); err != nil {
			return chain.ID, fmt.Errorf("aggregate chain metrics: %w", err)
		}
	}

	if modelsDropped > 0 {
		fmt.Fprintf(os.Stderr, "chains post-iteration: %d stage model(s) not recorded on this sink (%T does not implement eval_assessment); cost rate resolution for %s relies on the local leg\n",
			modelsDropped, store, p.Source)
	}

	return chain.ID, nil
}
