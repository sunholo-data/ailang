package coordinator

import (
	"fmt"
	"hash/fnv"
	"time"
)

// shadowTaskID is the child task a primary's shadow runs as. Deterministic, so
// a second dispatch of the same primary finds the child instead of making two.
func shadowTaskID(primaryID string) string { return primaryID + "-shadow" }

// shadowSampleFraction maps a task ID to a stable point in [0, 1). The sampling
// decision is a pure function of the task ID and the agent's sample_rate, so it
// can be recomputed from the record (axiom A1) rather than stored.
func shadowSampleFraction(taskID string) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(taskID))
	return float64(h.Sum64()>>11) / float64(uint64(1)<<53)
}

// shadowSampled reports whether a task is shadowed at rate. Rate 1 shadows
// every task and rate 0 none.
func shadowSampled(taskID string, rate float64) bool {
	return shadowSampleFraction(taskID) < rate
}

// shadowChild is the child record: the primary's directive and agent, and
// nothing that links it into the fleet's pipelines. No MessageID or ThreadID
// (nothing posts to the user's thread), no GitHub issue (nothing comments), and
// no StageID or ChainID, so applyMetrics skips it and chain costs never include
// shadow spend (design V18).
func shadowChild(primary *TaskRecord, s *ShadowConfig) *TaskRecord {
	return &TaskRecord{
		ID:         shadowTaskID(primary.ID),
		ShadowOf:   primary.ID,
		Inputs:     append([]TaskInput(nil), primary.Inputs...),
		Title:      primary.Title,
		Content:    primary.Content,
		Type:       primary.Type,
		Kind:       primary.Kind,
		Source:     primary.Source,
		Priority:   primary.Priority,
		Status:     TaskStatusPending,
		Provider:   s.Provider,
		AgentID:    primary.AgentID,
		Workspace:  primary.Workspace,
		BaseBranch: primary.BaseBranch,
		CreatedAt:  time.Now(),
	}
}

// shadowDispatchParams derives the shadow's dispatch from the primary's
// finished one. It takes the primary by value and returns a copy, so the
// primary's parameters cannot change. Everything that would let the run reach
// outside its directory is cleared here, as a second gate behind the job's own
// AILANG_TASK_SHADOW handling (M3).
func shadowDispatchParams(primary DispatchParams, childID string, agent *AgentConfig) DispatchParams {
	s := agent.Shadow
	p := primary
	p.TaskID = childID
	p.Inputs = append([]TaskInput(nil), primary.Inputs...)
	p.Provider = s.Provider
	p.ExecutorVariant = s.ExecutorVariant
	if s.Model != "" {
		p.Model = s.Model
	}
	if s.MaxCostUSD > 0 {
		p.MaxCostUSD = s.MaxCostUSD
	}
	p.Shadow = true
	p.PushBranch = ""
	p.AutoMerge, p.AutoMergeCode = false, false
	p.AutoMergeRequiredChecks = nil
	p.AutoMergeApproverSecret, p.AutoMergeApproverIdentity = "", ""
	p.PRLabels, p.MergeStarts = nil, nil
	// A user-provided key belongs to the user's own task, not to our comparison.
	p.APIKey = ""
	p.AuthMode = agent.AuthMode
	return p
}

// dispatchShadow runs after the primary's dispatch succeeded. It never fails
// the primary: every problem here is logged and leaves the primary exactly as
// it was. A shadow is not retried; a failed one is marked failed.
func (d *Daemon) dispatchShadow(primary *TaskRecord, agent *AgentConfig, primaryParams DispatchParams) {
	if d.cloudDispatcher == nil || agent == nil || agent.Shadow == nil || !agent.Shadow.Enabled || primary.ShadowOf != "" {
		return
	}
	s := agent.Shadow
	if !shadowSampled(primary.ID, s.SampleRate) {
		d.logger.Printf("Shadow: task %s not sampled (fraction %.4f >= rate %.4f)", primary.ID, shadowSampleFraction(primary.ID), s.SampleRate)
		return
	}
	if lane := credentialLane(agent, s.Provider); lane != "" {
		d.logger.Printf("Shadow: task %s not shadowed: provider %s leases credential lane %s, which a shadow must not contend for", primary.ID, s.Provider, lane)
		return
	}
	childID := shadowTaskID(primary.ID)
	if existing, err := d.taskStore.GetTask(d.ctx, childID); err == nil && existing != nil {
		d.logger.Printf("Shadow: %s already exists (status %s); not dispatching again", childID, existing.Status)
		return
	}
	if err := d.taskStore.CreateTask(d.ctx, shadowChild(primary, s)); err != nil {
		d.logger.Printf("Shadow: task %s: create %s: %v", primary.ID, childID, err)
		return
	}
	if err := d.taskStore.MarkTaskQueued(d.ctx, childID); err != nil {
		d.logger.Printf("Shadow: task %s: claim %s: %v", primary.ID, childID, err)
		return
	}
	if err := d.cloudDispatcher.Dispatch(d.ctx, shadowDispatchParams(primaryParams, childID, agent)); err != nil {
		d.failShadow(childID, fmt.Errorf("shadow dispatch: %w", err))
		return
	}
	d.logger.Printf("Shadow: task %s → %s on %s/%s", primary.ID, childID, s.Provider, s.ExecutorVariant)
}

// failShadow marks a shadow child failed without posting anywhere: a shadow
// has no thread, and a failed shadow is a fact for the report, not a message.
func (d *Daemon) failShadow(childID string, err error) {
	d.logger.Printf("Shadow: %s failed: %v", childID, err)
	if mErr := d.taskStore.MarkTaskFailed(d.ctx, childID, err); mErr != nil {
		d.logger.Printf("Warning: failed to mark shadow %s failed: %v", childID, mErr)
	}
}
