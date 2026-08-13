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
	// Status is the stage's outcome, from the existing ChainStageStatus vocabulary:
	// pending | running | awaiting_approval | completed | failed.
	//
	// A stage that FAILED must post "failed". Posting "completed" for every stage
	// would satisfy "nothing stays pending" while hiding real failures — that is the
	// wrong fix, and the reason this field is free-form outcome rather than a bool.
	//
	// EMPTY IS VALID and means "not reported": the stage keeps CreateStage's
	// `pending` default, i.e. exactly today's behaviour. The mission-control skill
	// and this CLI ship independently, so a payload written before this field
	// existed must keep working.
	Status string `json:"status,omitempty"`
	// ID pins this stage's identity across stores; see IterationPost.ChainID.
	// Normally empty on the wire and BACKFILLED by the first write.
	ID string `json:"id,omitempty"`
}

// IterationPost is one mission iteration to be posted as a chain.
type IterationPost struct {
	// Source is the chain source_ref, e.g. "mission:v1/iter-42"
	// (portable: "mission:<name>/iter-<N>").
	Source string `json:"source"`
	// SpooledAt records when this was buffered (set by the spool; informational).
	SpooledAt time.Time        `json:"spooled_at,omitempty"`
	Stages    []IterationStage `json:"stages"`
	// ChainID pins the chain identity so the SAME iteration written to a second
	// store (local + remote, M3) carries the SAME id — otherwise a span carrying
	// that id cannot join the remote copy, and the remote record is an island.
	//
	// Normally empty on the wire: PostIteration BACKFILLS it (and each stage's ID)
	// from what it wrote, so the caller can hand the same post to the next target,
	// and so a spooled entry replays with the identity it already has. A caller
	// MAY set it deliberately to pin an iteration's id up front.
	ChainID string `json:"chain_id,omitempty"`
}

// agentIDFor builds the stage agent_id, encoding the quota bucket in free text for
// quota lanes so it is visible without a schema column.
func (st IterationStage) agentIDFor() string {
	if st.QuotaBucket != "" {
		return fmt.Sprintf("%s (quota:%s)", st.Role, st.QuotaBucket)
	}
	return st.Role
}

// stageStatus resolves a posted status to the store vocabulary. An EMPTY status
// resolves to ("", nil) — "leave the stage at its CreateStage default", which is
// the version-skew path for payloads written before Status existed. An unknown
// status is an error rather than a silent coercion: coercing it would report an
// outcome the caller never claimed.
func (st IterationStage) stageStatus() (ChainStageStatus, error) {
	switch ChainStageStatus(st.Status) {
	case "":
		return "", nil
	case StageStatusPending, StageStatusRunning, StageStatusAwaitingApproval,
		StageStatusCompleted, StageStatusFailed:
		return ChainStageStatus(st.Status), nil
	}
	return "", fmt.Errorf("unknown status %q (want one of: %s, %s, %s, %s, %s)",
		st.Status, StageStatusPending, StageStatusRunning, StageStatusAwaitingApproval,
		StageStatusCompleted, StageStatusFailed)
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
		if _, err := st.stageStatus(); err != nil {
			return fmt.Errorf("iteration post: stage %d (%s): %w", i, st.Role, err)
		}
	}
	return nil
}

// PostIteration writes one iteration chain and its stages to an observatory
// backend. The chain and each stage are created, each stage's reported outcome is
// applied, and the stage costs/tokens are rolled up into the chain total; a
// per-stage failure aborts and returns an error (the caller spools the WHOLE post
// for retry). The quota bucket is encoded into agent_id.
//
// It takes the Backend INTERFACE, not *SQLiteBackend: the same iteration is posted
// to the local store and, when a node is configured for it, to a remote one
// (M-MISSION-LOOP-UNIFIED-TELEMETRY M3). One writer, two targets — a second
// implementation would be a second set of bugs.
//
// SIDE EFFECT: it backfills p.ChainID and each p.Stages[i].ID with what it wrote,
// so the next target writes the SAME identity and a spooled retry replays it.
func PostIteration(ctx context.Context, backend Backend, p *IterationPost) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}

	chain, err := backend.CreateChain(ctx, &ChainCreateRequest{
		ID:         p.ChainID, // empty = generate; set = replaying a known identity
		SourceType: ChainSourceManual,
		SourceRef:  p.Source,
	})
	if err != nil {
		return "", fmt.Errorf("create iteration chain: %w", err)
	}
	// Backfill so a second target (and the spool) reuse this identity.
	p.ChainID = chain.ID

	// Chain totals are denormalized, and nothing else writes them for a mission
	// chain — so an iteration that holds real per-stage spend reported $0.0000 as
	// its total (measured on mission:v1/iter-190). Accumulate as we go and roll up
	// once, after every stage has landed.
	var (
		totalCost   float64
		totalTokens int
		anyFailed   bool
		allTerminal = true
	)

	for i, st := range p.Stages {
		stage, err := backend.CreateStage(ctx, &StageCreateRequest{
			ID:       st.ID, // empty = generate; set = replaying a known identity
			ChainID:  chain.ID,
			AgentID:  st.agentIDFor(),
			Provider: Provider(st.Provider),
		})
		if err != nil {
			return chain.ID, fmt.Errorf("create stage %d (%s): %w", i, st.Role, err)
		}
		p.Stages[i].ID = stage.ID
		// Metrics: cost + tokens (quota lanes post zeros, which is a no-op add).
		if st.CostUSD != 0 || st.TokensIn != 0 || st.TokensOut != 0 {
			// "" = provenance not classified by this poster; reads as unknown.
			if err := backend.UpdateStageMetrics(ctx, stage.ID, st.CostUSD, st.TokensIn, st.TokensOut, 0, 0, 0, ""); err != nil {
				return chain.ID, fmt.Errorf("update stage %d metrics: %w", i, err)
			}
		}
		// Record the model (metered lanes) so the M1 classifier can resolve a rate.
		if st.Model != "" {
			if err := backend.UpdateStageEvalAssessment(ctx, stage.ID, &EvalAssessment{
				Model:    st.Model,
				EvalMode: "mission",
			}); err != nil {
				return chain.ID, fmt.Errorf("update stage %d model: %w", i, err)
			}
		}
		// Outcome. Validate already rejected an unknown status, so the only error
		// here is a store failure. An empty status leaves the CreateStage default.
		status, _ := st.stageStatus()
		if status != "" {
			if err := backend.UpdateStageStatus(ctx, stage.ID, status); err != nil {
				return chain.ID, fmt.Errorf("update stage %d status: %w", i, err)
			}
		}

		totalCost += st.CostUSD
		totalTokens += st.TokensIn + st.TokensOut
		switch status {
		case StageStatusFailed:
			anyFailed = true
		case StageStatusCompleted:
		default:
			// Unreported, running, or awaiting approval: the iteration is not
			// finished, so the chain must not claim an outcome.
			allTerminal = false
		}
	}

	// Roll the stages up into the chain. UpdateChainMetrics is additive and the
	// chain was created in this call, so this is the sum, not a re-add.
	if totalCost != 0 || totalTokens != 0 {
		if err := backend.UpdateChainMetrics(ctx, chain.ID, totalCost, totalTokens, 0); err != nil {
			return chain.ID, fmt.Errorf("aggregate chain metrics: %w", err)
		}
	}

	// Chain outcome, derived ONLY when every stage reported a terminal one — a post
	// that omits Status (or is still mid-flight) leaves the chain `active`, which is
	// today's behaviour. One failed stage fails the iteration: a mission iteration
	// that lost a stage did not succeed, and "completed with a failed stage inside"
	// is the reading this whole milestone exists to prevent.
	if allTerminal {
		chainStatus := ChainStatusCompleted
		if anyFailed {
			chainStatus = ChainStatusFailed
		}
		if err := backend.UpdateChainStatus(ctx, chain.ID, chainStatus); err != nil {
			return chain.ID, fmt.Errorf("set chain status: %w", err)
		}
	}

	return chain.ID, nil
}
