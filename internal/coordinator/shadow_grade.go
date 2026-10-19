package coordinator

import (
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// The grade pipeline (M-AILANG-LANE-EXECUTOR-SHADOW M5–M6).
//
// The coordinator only DISPATCHES grading. Everything that executes code or
// calls a model — the mechanical gates, the merged PR's tests, the reviewer —
// runs in a Cloud Run job (cmd/ailang, AILANG_JOB_MODE=shadow_grade, one phase
// per dispatch), which writes its result straight back to the comparison. None
// of that code is in this package, so the daemon cannot execute a repo's code
// by construction (design round-2 objection). The coordinator's own share is
// claiming stages, dispatching, settling what cannot be graded, and reading how
// the primary's PR ended (shadow_merged.go), which is a GitHub read.

const (
	// shadowGradeBatch bounds dispatches per stage per tick, like the task batch.
	shadowGradeBatch = 5
	// shadowGradeStaleAfter settles a job that never reported. The job bounds
	// itself at 15 minutes.
	shadowGradeStaleAfter = 30 * time.Minute
	// shadowGradeJobTimeout is the job's wall-clock, passed as AILANG_TIMEOUT.
	shadowGradeJobTimeout = "15m"
)

// shadowPhase is one job phase: the stage it is dispatched from, the stage
// that claims it, and how a job that never reports is settled.
type shadowPhase struct {
	name      string // AILANG_SHADOW_PHASE
	ready     string
	inFlight  string
	onAbandon func(c *ShadowComparison, reason string)
}

var shadowPhases = []shadowPhase{
	{config.ShadowPhaseMechanical, ShadowGradeNone, ShadowGradeDispatched, func(c *ShadowComparison, reason string) {
		c.GradeState, c.Verdict, c.UngradedReason = ShadowGradeGraded, ShadowVerdictUngraded, reason
	}},
	{config.ShadowPhaseMergedTests, ShadowGradeMergedTests, ShadowGradeMergedTesting, func(c *ShadowComparison, _ string) {
		c.GradeState = ShadowGradeReview // the merged tests are simply absent
	}},
	{config.ShadowPhaseReview, ShadowGradeReview, ShadowGradeReviewing, func(c *ShadowComparison, reason string) {
		c.Review = &ShadowReview{Ungraded: reason}
		c.Verdict, c.UngradedReason = FinalShadowVerdict(c)
		c.GradeState = ShadowGradeDone
	}},
}

// dispatchShadowGrades runs once per cloud tick. It never fails the tick:
// grading is best-effort evidence, not fleet work.
func (d *Daemon) dispatchShadowGrades() {
	store, ok := d.taskStore.(ShadowComparisonStore)
	if !ok || d.cloudDispatcher == nil {
		return
	}
	now := time.Now().UTC()
	d.sweepShadowMerged(store, now)
	d.postWeeklyShadowReport(store, now)
	for _, ph := range shadowPhases {
		if stale, err := store.ListShadowComparisonsByGradeState(d.ctx, ph.inFlight, shadowGradeBatch); err == nil {
			for _, c := range stale {
				if now.Sub(c.GradeDispatchedAt) > shadowGradeStaleAfter {
					d.settleShadow(store, c.ParentID, ph.inFlight, ph.name+"_timeout", ph.onAbandon)
				}
			}
		}
		ready, err := store.ListShadowComparisonsByGradeState(d.ctx, ph.ready, shadowGradeBatch)
		if err != nil {
			d.logger.Printf("Shadow grade: list %s: %v", ph.name, err)
			continue
		}
		for _, c := range ready {
			d.dispatchShadowPhase(store, c, ph, now)
		}
	}
}

func (d *Daemon) dispatchShadowPhase(store ShadowComparisonStore, c *ShadowComparison, ph shadowPhase, now time.Time) {
	agent := d.shadowAgent(c)
	if ph.name == config.ShadowPhaseReview {
		// A review happens only when the layers tie or disagree, and only with a
		// reviewer configured; otherwise the layered verdict is final.
		_, _, needs := LayeredShadowVerdict(c)
		if !needs || agent == nil || agent.Shadow == nil || agent.Shadow.ReviewerModel == "" {
			d.settleShadow(store, c.ParentID, ph.ready, "", func(cur *ShadowComparison, _ string) {
				cur.Verdict, cur.UngradedReason = FinalShadowVerdict(cur)
				cur.GradeState = ShadowGradeDone
			})
			return
		}
	}
	planeDefault := ""
	if d.coordConfig != nil {
		planeDefault = d.coordConfig.DefaultProvider
	}
	params, reason := shadowGradeParams(c, agent, planeDefault, ph.name)
	if reason == "" && ph.name == config.ShadowPhaseReview {
		params.Model, params.MaxCostUSD = agent.Shadow.ReviewerModel, agent.Shadow.reviewerBudget()
		// The reviewer judges both diffs against what was asked: the primary's
		// own directive, read from its task record.
		if t, err := d.taskStore.GetTask(d.ctx, c.ParentID); err == nil && t != nil {
			params.Directive = t.Content
		}
	}
	if reason != "" {
		d.settleShadow(store, c.ParentID, ph.ready, reason, ph.onAbandon)
		return
	}
	won, err := store.UpdateShadowComparison(d.ctx, c.ParentID, func(cur *ShadowComparison) bool {
		if cur.GradeState != ph.ready {
			return false // another instance claimed it
		}
		cur.GradeState, cur.GradeDispatchedAt = ph.inFlight, now
		return true
	})
	if err != nil || !won {
		return
	}
	if err := d.cloudDispatcher.Dispatch(d.ctx, params); err != nil {
		d.settleShadow(store, c.ParentID, ph.inFlight, ph.name+"_dispatch_failed: "+err.Error(), ph.onAbandon)
		return
	}
	d.logger.Printf("Shadow grade: %s → %s job %s", c.ParentID, ph.name, params.TaskID)
}

func (d *Daemon) shadowAgent(c *ShadowComparison) *AgentConfig {
	if d.agentRegistry == nil || c.Primary == nil {
		return nil
	}
	return d.agentRegistry.GetAgentByID(c.Primary.AgentID)
}

// shadowGradeParams builds a grade-job dispatch, or names why it cannot be
// built. It carries no credential beyond the clone and the agent's own
// restricted policy, which every gate runs under.
func shadowGradeParams(c *ShadowComparison, agent *AgentConfig, planeDefault, phase string) (DispatchParams, string) {
	if agent == nil {
		return DispatchParams{}, "agent_unknown"
	}
	if agent.PolicyPath == "" {
		return DispatchParams{}, "no_policy"
	}
	toml, err := os.ReadFile(agent.PolicyPath)
	if err != nil {
		return DispatchParams{}, "no_policy: " + err.Error()
	}
	if lerr := executor.CheckLanePolicy(executor.ToolProfileAILANGOnly, toml); lerr != nil {
		return DispatchParams{}, "no_policy: " + lerr.Error()
	}
	repo := agent.ResolveRepo()
	if repo == "" {
		repo = agent.Workspace
	}
	return DispatchParams{
		TaskID:          c.ParentID + "-" + phase,
		AgentID:         agent.ID,
		Workspace:       agent.Workspace,
		Provider:        ResolveDispatchProvider(agent, planeDefault),
		ExecutorVariant: agent.ExecutorVariant,
		AuthMode:        agent.AuthMode,
		Directive:       fmt.Sprintf("Shadow grade (%s) of %s; no agent runs.", phase, c.ParentID),
		TaskTitle:       "shadow " + phase + " " + c.ParentID,
		RepoURL:         deriveRepoURL(repo),
		Timeout:         shadowGradeJobTimeout,
		ToolPolicy:      executor.ToolProfileAILANGOnly,
		PolicyTOML:      string(toml),
		WorkTier:        string(ResolveWorkTier(agent, "")),
		AcknowledgeOnly: true,
		JobMode:         config.JobModeShadowGrade,
		ShadowParent:    c.ParentID,
		ShadowPhase:     phase,
	}, ""
}

// settleShadow moves a comparison out of stage `from` with fn. It acts only if
// the comparison is still in that stage, so a job result that landed meanwhile
// is never overwritten.
func (d *Daemon) settleShadow(store ShadowComparisonStore, parentID, from, reason string, fn func(*ShadowComparison, string)) {
	_, err := store.UpdateShadowComparison(d.ctx, parentID, func(cur *ShadowComparison) bool {
		if cur.GradeState != from {
			return false
		}
		fn(cur, reason)
		return true
	})
	if err != nil {
		d.logger.Printf("Shadow grade: %s: settling (%s): %v", parentID, reason, err)
		return
	}
	if reason != "" {
		d.logger.Printf("Shadow grade: %s settled: %s", parentID, reason)
	}
}
