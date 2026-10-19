package coordinator

import (
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// The mechanical grade (M-AILANG-LANE-EXECUTOR-SHADOW M5, design layer 1).
//
// The coordinator only DISPATCHES grading. The grading itself — clone at each
// base commit, apply each diff, run `ailang check` / `ailang test` through the
// policy tool under the agent's restricted policy — runs in a Cloud Run job
// (cmd/ailang, AILANG_JOB_MODE=shadow_grade), which writes its result straight
// back to the comparison. None of that code is in this package, so the daemon
// cannot execute a repo's code by construction (design round-2 objection).

const (
	// shadowGradeBatch bounds dispatches per tick, like the task batch.
	shadowGradeBatch = 5
	// shadowGradeStaleAfter turns a grade job that never reported into
	// ungraded(grade_timeout). The job bounds itself at 15 minutes.
	shadowGradeStaleAfter = 30 * time.Minute
	// shadowGradeJobTimeout is the job's wall-clock, passed as AILANG_TIMEOUT.
	shadowGradeJobTimeout = "15m"
)

// dispatchShadowGrades runs once per cloud tick. It never fails the tick:
// grading is best-effort evidence, not fleet work.
func (d *Daemon) dispatchShadowGrades() {
	store, ok := d.taskStore.(ShadowComparisonStore)
	if !ok || d.cloudDispatcher == nil {
		return
	}
	now := time.Now().UTC()
	if stale, err := store.ListShadowComparisonsByGradeState(d.ctx, ShadowGradeDispatched, shadowGradeBatch); err == nil {
		for _, c := range stale {
			if now.Sub(c.GradeDispatchedAt) > shadowGradeStaleAfter {
				d.markShadowUngraded(store, c.ParentID, ShadowGradeDispatched, "grade_timeout")
			}
		}
	}
	ready, err := store.ListShadowComparisonsByGradeState(d.ctx, ShadowGradeNone, shadowGradeBatch)
	if err != nil {
		d.logger.Printf("Shadow grade: list: %v", err)
		return
	}
	for _, c := range ready {
		d.dispatchShadowGrade(store, c, now)
	}
}

func (d *Daemon) dispatchShadowGrade(store ShadowComparisonStore, c *ShadowComparison, now time.Time) {
	var agent *AgentConfig
	if d.agentRegistry != nil && c.Primary != nil {
		agent = d.agentRegistry.GetAgentByID(c.Primary.AgentID)
	}
	planeDefault := ""
	if d.coordConfig != nil {
		planeDefault = d.coordConfig.DefaultProvider
	}
	params, reason := shadowGradeParams(c, agent, planeDefault)
	if reason != "" {
		d.markShadowUngraded(store, c.ParentID, ShadowGradeNone, reason)
		return
	}
	won, err := store.UpdateShadowComparison(d.ctx, c.ParentID, func(cur *ShadowComparison) bool {
		if cur.GradeState != ShadowGradeNone {
			return false // another instance claimed it
		}
		cur.GradeState, cur.GradeDispatchedAt = ShadowGradeDispatched, now
		return true
	})
	if err != nil || !won {
		return
	}
	if err := d.cloudDispatcher.Dispatch(d.ctx, params); err != nil {
		d.markShadowUngraded(store, c.ParentID, ShadowGradeDispatched, "grade_dispatch_failed: "+err.Error())
		return
	}
	d.logger.Printf("Shadow grade: %s → grade job %s", c.ParentID, params.TaskID)
}

// shadowGradeParams builds the grade job's dispatch, or names why it cannot be
// built. It carries no model, no credential beyond the clone, and the agent's
// own restricted policy, which the grade job runs every gate under.
func shadowGradeParams(c *ShadowComparison, agent *AgentConfig, planeDefault string) (DispatchParams, string) {
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
		TaskID:          c.ParentID + "-grade",
		AgentID:         agent.ID,
		Workspace:       agent.Workspace,
		Provider:        ResolveDispatchProvider(agent, planeDefault),
		ExecutorVariant: agent.ExecutorVariant,
		AuthMode:        agent.AuthMode,
		Directive:       fmt.Sprintf("Grade the shadow comparison of %s (no agent runs).", c.ParentID),
		TaskTitle:       "shadow grade " + c.ParentID,
		RepoURL:         deriveRepoURL(repo),
		Timeout:         shadowGradeJobTimeout,
		ToolPolicy:      executor.ToolProfileAILANGOnly,
		PolicyTOML:      string(toml),
		WorkTier:        string(ResolveWorkTier(agent, "")),
		AcknowledgeOnly: true,
		JobMode:         config.JobModeShadowGrade,
		ShadowParent:    c.ParentID,
	}, ""
}

// markShadowUngraded settles a comparison the grade job will not grade. It only
// moves a comparison out of the state the caller saw, so a late grade that
// landed meanwhile is never overwritten.
func (d *Daemon) markShadowUngraded(store ShadowComparisonStore, parentID, from, reason string) {
	_, err := store.UpdateShadowComparison(d.ctx, parentID, func(cur *ShadowComparison) bool {
		if cur.GradeState != from {
			return false
		}
		cur.GradeState, cur.Verdict, cur.UngradedReason = ShadowGradeGraded, ShadowVerdictUngraded, reason
		return true
	})
	if err != nil {
		d.logger.Printf("Shadow grade: %s: marking ungraded (%s): %v", parentID, reason, err)
		return
	}
	d.logger.Printf("Shadow grade: %s ungraded: %s", parentID, reason)
}
