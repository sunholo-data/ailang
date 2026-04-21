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
	logger        *log.Logger
	interval      time.Duration // Check interval (default: 2 min)

	// TTL cache for stale task query results.
	cacheMu     sync.RWMutex
	cachedTasks []*TaskRecord
	cacheExpiry time.Time
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
		age := d.getTaskAge(task)

		if age <= timeout {
			continue
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

		d.postFailureNotification(ctx, task, errMsg)
	}
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

// getTaskAge returns how long a task has been in its current status.
func (d *StaleTaskDetector) getTaskAge(task *TaskRecord) time.Duration {
	if task.StartedAt != nil {
		return time.Since(*task.StartedAt)
	}
	return time.Since(task.CreatedAt)
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

	msg := &messaging.InboxMessage{
		FromAgent:     task.AgentID,
		ToInbox:       task.AgentID,
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
