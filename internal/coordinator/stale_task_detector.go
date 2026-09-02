package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// StaleTaskDetector periodically checks for tasks stuck in queued/running status
// and marks them as failed after their timeout expires. This catches cases where:
//   - Cloud Run Job container fails to start (image pull, OOM, quota)
//   - Pub/Sub completion publish fails
//   - Job process crashes before publishing completion
//
// Only runs in cloud mode (COORDINATOR_MODE=cloud).
// Local mode uses RecoverStaleTasks at startup instead.
type StaleTaskDetector struct {
	store         Store
	agentRegistry *AgentRegistry
	msgStore      messaging.MessageStore
	obsBackend    observatory.Backend // may be nil; chain closure is skipped when absent
	logger        *log.Logger
	interval      time.Duration // Check interval (default: 2 min)

	// reDispatch, when set, re-runs an infra-class failure on the next chain
	// link. Nil means "report only" — the detector then behaves exactly as it
	// did before M3, which is the safe default for any caller that has not
	// opted in. This detector is the only component that may hold it.
	reDispatch func(ctx context.Context, task *TaskRecord) error

	// TTL cache for stale task query results.
	cacheMu     sync.RWMutex
	cachedTasks []*TaskRecord
	cacheExpiry time.Time
}

// WithReDispatcher opts this detector in as the sole re-dispatcher of
// infra-class failures. The callback must compare-and-set AttemptCount so two
// coordinator instances cannot both spend an execution.
func (d *StaleTaskDetector) WithReDispatcher(f func(ctx context.Context, task *TaskRecord) error) *StaleTaskDetector {
	d.reDispatch = f
	return d
}

// WithObservatory attaches the observatory backend so a timed-out task also
// closes its execution chain. Without it, chain closure is skipped (and said so
// once), which is the pre-existing behaviour.
func (d *StaleTaskDetector) WithObservatory(b observatory.Backend) *StaleTaskDetector {
	d.obsBackend = b
	return d
}

// NewStaleTaskDetector creates a detector that checks for stale tasks periodically.
func NewStaleTaskDetector(store Store, agentRegistry *AgentRegistry, msgStore messaging.MessageStore, logger *log.Logger) *StaleTaskDetector {
	return &StaleTaskDetector{
		store:         store,
		agentRegistry: agentRegistry,
		msgStore:      msgStore,
		logger:        logger,
		interval:      2 * time.Minute,
	}
}

// Run starts the periodic stale task check. Blocks until ctx is cancelled.
func (d *StaleTaskDetector) Run(ctx context.Context) {
	d.logger.Printf("stale task detector: started (interval=%v)", d.interval)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Println("stale task detector: stopped")
			return
		case <-ticker.C:
			d.detectAndMarkStale(ctx)
		}
	}
}

// detectAndMarkStale finds tasks in queued/running status that have exceeded
// their timeout and marks them as failed.
func (d *StaleTaskDetector) detectAndMarkStale(ctx context.Context) {
	tasks, err := d.getCachedOrQueryTasks(ctx)
	if err != nil {
		d.logger.Printf("stale task detector: query error: %v", err)
		return
	}

	for _, task := range tasks {
		timeout := d.getTaskTimeout(task)
		age, known := d.getTaskAge(task)
		if !known {
			// Loud, and every pass: this is a writer defect upstream (a task
			// persisted with no created_at), and staying quiet about it is how it
			// survived long enough to be discovered from its blast radius.
			d.logger.Printf("stale task detector: task %s has no created_at and no started_at — "+
				"age unknowable, NOT marking stale (upstream writer defect)", task.ID)
			continue
		}

		if age <= timeout {
			continue
		}

		// M-COORDINATOR-EXECUTION-TRUST M3: a task that timed out without ever
		// publishing a completion is the INFRASTRUCTURE class — the container
		// died, was OOM-killed or was preempted. In-container retry is
		// impossible by definition here, which is exactly why this tier exists.
		//
		// This detector is the SOLE re-dispatcher. Three other components can
		// also move a task toward a terminal state (the completion handler, the
		// stranded-approval sweep, the worktree sweep — design doc V23); any
		// second one gaining this power would breach the cap and duplicate work
		// nondeterministically.
		if ShouldReDispatch(task, age, known) {
			if d.reDispatch != nil {
				// Compare-and-set on the persisted counter: a loser LOGS and does
				// nothing. Deliberately not a lock — a silent loser is
				// indistinguishable from a component that never ran.
				if err := d.reDispatch(ctx, task); err != nil {
					d.logger.Printf("stale task detector: task %s not re-dispatched (attempt %d/%d): %v",
						task.ID, task.AttemptCount+1, MaxTaskExecutions, err)
				} else {
					d.logger.Printf("stale task detector: task %s re-dispatched on chain link %d (attempt %d/%d)",
						task.ID, task.ChainLinkIndex+1, task.AttemptCount+1, MaxTaskExecutions)
					d.cacheMu.Lock()
					d.cachedTasks = nil
					d.cacheMu.Unlock()
					continue
				}
			}
		}

		errMsg := fmt.Sprintf("task timed out: no completion received within %v of being queued (age=%v)", timeout, age)
		d.logger.Printf("stale task detector: marking task %s as failed: %s", task.ID, errMsg)

		if err := d.store.MarkTaskFailed(ctx, task.ID, fmt.Errorf("%s", errMsg)); err != nil {
			d.logger.Printf("stale task detector: failed to mark task %s: %v", task.ID, err)
			continue
		}

		// Invalidate cache since we changed task state.
		d.cacheMu.Lock()
		d.cachedTasks = nil
		d.cacheMu.Unlock()

		d.closeChainForFailedTask(ctx, task, errMsg)
		d.postFailureNotification(ctx, task, errMsg)
	}
}

// closeChainForFailedTask drives the task's execution chain to a terminal state.
//
// MarkTaskFailed moved only the TASK. Nothing reconciled the chain, and chain
// closure otherwise happens solely on the approval/rejection path
// (approval_processor.go) — which a job that died before completing never
// reaches. So a chain opened at dispatch stayed `active` forever.
//
// Measured 2026-08-31 in prod: 92 of 99 chains were `active`, the oldest since
// 2026-04-27. That makes "running right now" and "died in April" the same
// reading, which is worse than no status at all — a stuck chain is invisible
// precisely because it looks busy.
func (d *StaleTaskDetector) closeChainForFailedTask(ctx context.Context, task *TaskRecord, errMsg string) {
	if d.obsBackend == nil || task.ChainID == "" {
		return
	}
	if err := d.obsBackend.UpdateChainStatus(ctx, task.ChainID, observatory.ChainStatusFailed); err != nil {
		// Loud, not fatal: the task IS failed and the notification still goes out.
		// A chain left open is a reporting defect, not a lost outcome.
		d.logger.Printf("stale task detector: task %s marked failed but chain %s could not be closed: %v",
			task.ID, task.ChainID, err)
		return
	}
	d.logger.Printf("stale task detector: closed chain %s as failed (task %s: %s)", task.ChainID, task.ID, errMsg)
}

// getCachedOrQueryTasks returns queued/running tasks, using a 90-second TTL cache
// to avoid repeated Firestore queries within the same detection window.
func (d *StaleTaskDetector) getCachedOrQueryTasks(ctx context.Context) ([]*TaskRecord, error) {
	d.cacheMu.RLock()
	if d.cachedTasks != nil && time.Now().Before(d.cacheExpiry) {
		tasks := d.cachedTasks
		d.cacheMu.RUnlock()
		return tasks, nil
	}
	d.cacheMu.RUnlock()

	tasks, err := d.store.ListTasks(ctx, &TaskFilter{
		Status: []TaskStatus{TaskStatusQueued, TaskStatusRunning},
	})
	if err != nil {
		return nil, err
	}

	d.cacheMu.Lock()
	d.cachedTasks = tasks
	d.cacheExpiry = time.Now().Add(90 * time.Second)
	d.cacheMu.Unlock()

	return tasks, nil
}

// getTaskTimeout returns the timeout for a task, using agent config with 1.5x safety margin.
func (d *StaleTaskDetector) getTaskTimeout(task *TaskRecord) time.Duration {
	if d.agentRegistry != nil {
		if agent := d.agentRegistry.GetAgentByID(task.AgentID); agent != nil {
			return time.Duration(float64(agent.GetEffectiveTimeout()) * 1.5)
		}
	}
	// Default: 90 minutes (1.5x Cloud Run's 60-minute default)
	return 90 * time.Minute
}

// getTaskAge reports how long the task has been outstanding, and whether that
// is knowable at all.
//
// NO SILENT FALLBACK. A task with neither StartedAt nor CreatedAt used to fall
// through to time.Since(zero) — about 292 years — so every such task exceeded
// every timeout and was killed on the first tick after dispatch. Measured in
// prod 2026-08-31: task-a855b349 and task-133e933b were both written with
// created_at = null, marked "timed out ... within 22m30s (age=2562047h47m16s)"
// roughly 57s after being queued, and each failure notice then fed a dispatch
// loop. An unknown age is a data defect to report, never a timeout to act on.
func (d *StaleTaskDetector) getTaskAge(task *TaskRecord) (time.Duration, bool) {
	if task.StartedAt != nil && !task.StartedAt.IsZero() {
		return time.Since(*task.StartedAt), true
	}
	if !task.CreatedAt.IsZero() {
		return time.Since(task.CreatedAt), true
	}
	return 0, false
}

// postFailureNotification posts a failure message to the agent's inbox
// so external clients (portal, sidecar) can detect the failure.
func (d *StaleTaskDetector) postFailureNotification(ctx context.Context, task *TaskRecord, errMsg string) {
	if d.msgStore == nil {
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"task_id":  task.ID,
		"agent_id": task.AgentID,
		"status":   "failed",
		"error":    errMsg,
		"source":   "stale_task_detector",
	})

	// Agent ID is not an inbox name for package agents — resolve the real inbox.
	toInbox := task.AgentID
	if inbox, ok := d.agentRegistry.InboxForAgent(task.AgentID); ok {
		toInbox = inbox
	}

	msg := &messaging.InboxMessage{
		FromAgent:     task.AgentID,
		ToInbox:       toInbox,
		MessageType:   "completion",
		Title:         fmt.Sprintf("Task %s: failed (timeout)", task.ID),
		Payload:       string(payload),
		CorrelationID: task.MessageID,
	}

	if err := d.msgStore.InsertInboxMessage(msg); err != nil {
		d.logger.Printf("stale task detector: failed to post failure notification for task %s: %v",
			task.ID, err)
	}
}

// IsCloudMode returns true if running in cloud mode.
func IsCloudMode() bool {
	return os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud
}
