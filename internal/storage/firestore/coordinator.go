package firestore

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo/ailang/internal/coordinator"
)

const (
	collTasks     = "tasks"
	collApprovals = "approvals"
	collMeta      = "_meta"

	costCountersDocID = "cost_counters"
	costSyncInterval  = 5 * time.Minute
	statsCacheTTL     = 30 * time.Second
)

// CoordinatorStore implements coordinator.Store backed by Firestore.
type CoordinatorStore struct {
	client *Client

	// In-memory cost tracking — avoids full collection scan on every budget check.
	costMu       sync.RWMutex
	costCounters map[string]float64 // provider -> total cost
	costLoaded   bool               // true after initial bootstrap
	costDirty    bool               // true if in-memory state diverges from Firestore
	costCancel   context.CancelFunc // stops the background sync goroutine

	// Cached task stats — avoids full collection scan on every status API call.
	statsMu     sync.RWMutex
	cachedStats *coordinator.TaskStats
	statsExpiry time.Time
}

// NewCoordinatorStore creates a new Firestore-backed coordinator store.
func NewCoordinatorStore(client *Client) *CoordinatorStore {
	return &CoordinatorStore{
		client:       client,
		costCounters: make(map[string]float64),
	}
}

// StartCostSync bootstraps the in-memory cost counter from Firestore and starts
// a background goroutine that writes dirty counters back every 5 minutes.
func (s *CoordinatorStore) StartCostSync(ctx context.Context) {
	s.bootstrapCostCounters(ctx)

	syncCtx, cancel := context.WithCancel(ctx)
	s.costCancel = cancel

	go func() {
		ticker := time.NewTicker(costSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-syncCtx.Done():
				// Final flush on shutdown.
				flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
				s.flushCostCounters(flushCtx)
				flushCancel()
				return
			case <-ticker.C:
				s.flushCostCounters(syncCtx)
			}
		}
	}()
}

// StopCostSync stops the background sync goroutine and performs a final flush.
func (s *CoordinatorStore) StopCostSync() {
	if s.costCancel != nil {
		s.costCancel()
		s.costCancel = nil
	}
}

// bootstrapCostCounters loads the _meta/cost_counters doc. If it doesn't exist,
// falls back to a one-time full collection scan to seed the counters.
func (s *CoordinatorStore) bootstrapCostCounters(ctx context.Context) {
	doc, err := s.client.Doc(collMeta, costCountersDocID).Get(ctx)
	if err == nil {
		s.costMu.Lock()
		for k, v := range doc.Data() {
			if cost, ok := v.(float64); ok {
				s.costCounters[k] = cost
			}
		}
		s.costLoaded = true
		s.costMu.Unlock()
		return
	}

	// _meta doc doesn't exist — one-time full scan to bootstrap.
	log.Println("[cost-sync] bootstrapping cost counters from full scan")
	costs, err := s.fullScanCostByProvider(ctx)
	if err != nil {
		log.Printf("[cost-sync] bootstrap full scan failed: %v", err)
		s.costMu.Lock()
		s.costLoaded = true // mark loaded even on error to avoid blocking
		s.costMu.Unlock()
		return
	}

	s.costMu.Lock()
	s.costCounters = costs
	s.costLoaded = true
	s.costDirty = true // flush to create the _meta doc
	s.costMu.Unlock()
}

// flushCostCounters writes dirty in-memory counters to _meta/cost_counters.
func (s *CoordinatorStore) flushCostCounters(ctx context.Context) {
	s.costMu.RLock()
	if !s.costDirty {
		s.costMu.RUnlock()
		return
	}
	snapshot := make(map[string]interface{}, len(s.costCounters))
	for k, v := range s.costCounters {
		snapshot[k] = v
	}
	s.costMu.RUnlock()

	_, err := s.client.Doc(collMeta, costCountersDocID).Set(ctx, snapshot)
	if err != nil {
		log.Printf("[cost-sync] flush failed: %v", err)
		return
	}

	s.costMu.Lock()
	s.costDirty = false
	s.costMu.Unlock()
}

// addCost increments the in-memory cost counter for a provider.
func (s *CoordinatorStore) addCost(provider string, cost float64) {
	if provider == "" || cost == 0 {
		return
	}
	s.costMu.Lock()
	s.costCounters[provider] += cost
	s.costDirty = true
	s.costMu.Unlock()
}

// fullScanCostByProvider does the original full collection scan (used only for bootstrap).
func (s *CoordinatorStore) fullScanCostByProvider(ctx context.Context) (map[string]float64, error) {
	iter := s.client.Collection(collTasks).Documents(ctx)
	defer iter.Stop()

	costs := make(map[string]float64)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		provider, _ := data["provider"].(string)
		cost, _ := data["cost"].(float64)
		if provider != "" {
			costs[provider] += cost
		}
	}
	return costs, nil
}

// Close stops the cost sync goroutine and closes the underlying client.
func (s *CoordinatorStore) Close() error {
	s.StopCostSync()
	return s.client.Close()
}

// --- Task CRUD ---

func (s *CoordinatorStore) CreateTask(ctx context.Context, task *coordinator.TaskRecord) error {
	data := taskToMap(task)
	_, err := s.client.Doc(collTasks, task.ID).Set(ctx, data)
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) GetTask(ctx context.Context, id string) (*coordinator.TaskRecord, error) {
	doc, err := s.client.Doc(collTasks, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, err
	}
	return mapToTask(doc.Data()), nil
}

func (s *CoordinatorStore) UpdateTask(ctx context.Context, task *coordinator.TaskRecord) error {
	data := taskToMap(task)
	_, err := s.client.Doc(collTasks, task.ID).Set(ctx, data, firestore.MergeAll)
	return err
}

func (s *CoordinatorStore) DeleteTask(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Delete(ctx)
	return err
}

// --- Task Queries ---

func (s *CoordinatorStore) ListTasks(ctx context.Context, filter *coordinator.TaskFilter) ([]*coordinator.TaskRecord, error) {
	q := s.client.Collection(collTasks).Query

	if len(filter.Status) > 0 {
		statuses := make([]interface{}, len(filter.Status))
		for i, st := range filter.Status {
			statuses[i] = string(st)
		}
		if len(statuses) == 1 {
			q = q.Where("status", "==", statuses[0])
		} else {
			q = q.Where("status", "in", statuses)
		}
	}

	if filter.Provider != "" {
		q = q.Where("provider", "==", filter.Provider)
	}

	if filter.Workspace != "" {
		q = q.Where("workspace", "==", filter.Workspace)
	}

	if filter.Since != nil {
		q = q.Where("created_at", ">=", *filter.Since)
	}

	if filter.Until != nil {
		q = q.Where("created_at", "<=", *filter.Until)
	}

	// Ordering
	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	dir := firestore.Asc
	if filter.OrderDesc {
		dir = firestore.Desc
	}
	q = q.OrderBy(orderBy, dir)

	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	q = q.Limit(limit)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var tasks []*coordinator.TaskRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, mapToTask(doc.Data()))
	}
	return tasks, nil
}

// GetTaskStats returns aggregate task statistics, served from an in-memory cache
// with a 30-second TTL. This avoids a full collection scan on every API call.
func (s *CoordinatorStore) GetTaskStats(ctx context.Context) (*coordinator.TaskStats, error) {
	s.statsMu.RLock()
	if s.cachedStats != nil && time.Now().Before(s.statsExpiry) {
		stats := s.cachedStats
		s.statsMu.RUnlock()
		return stats, nil
	}
	s.statsMu.RUnlock()

	// Cache miss — do a full scan and cache the result.
	stats, err := s.fullScanTaskStats(ctx)
	if err != nil {
		return nil, err
	}

	s.statsMu.Lock()
	s.cachedStats = stats
	s.statsExpiry = time.Now().Add(statsCacheTTL)
	s.statsMu.Unlock()

	return stats, nil
}

// invalidateStatsCache clears the cached stats so the next call re-scans.
// Called on every state transition to keep the cache fresh.
func (s *CoordinatorStore) invalidateStatsCache() {
	s.statsMu.Lock()
	s.cachedStats = nil
	s.statsMu.Unlock()
}

// fullScanTaskStats performs the original full collection scan.
func (s *CoordinatorStore) fullScanTaskStats(ctx context.Context) (*coordinator.TaskStats, error) {
	iter := s.client.Collection(collTasks).Documents(ctx)
	defer iter.Stop()

	stats := &coordinator.TaskStats{
		ByType:      make(map[string]int),
		ByProvider:  make(map[string]*coordinator.DetailedStats),
		ByWorkspace: make(map[string]*coordinator.DetailedStats),
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		task := mapToTask(data)

		stats.TotalTasks++
		switch task.Status {
		case coordinator.TaskStatusPending:
			stats.PendingTasks++
		case coordinator.TaskStatusRunning, coordinator.TaskStatusQueued:
			stats.RunningTasks++
		case coordinator.TaskStatusPendingApproval:
			stats.PendingApprovals++
		case coordinator.TaskStatusCompleted:
			stats.CompletedTasks++
		case coordinator.TaskStatusFailed:
			stats.FailedTasks++
		}

		stats.ByType[string(task.Type)]++
		stats.TotalCost += task.Cost
		stats.TotalTokens += task.TokensUsed

		if task.Provider != "" {
			ds, ok := stats.ByProvider[task.Provider]
			if !ok {
				ds = &coordinator.DetailedStats{}
				stats.ByProvider[task.Provider] = ds
			}
			ds.Count++
			ds.CostUSD += task.Cost
			ds.InputTokens += task.InputTokens
			ds.OutputTokens += task.OutputTokens
		}

		if task.Workspace != "" {
			ds, ok := stats.ByWorkspace[task.Workspace]
			if !ok {
				ds = &coordinator.DetailedStats{}
				stats.ByWorkspace[task.Workspace] = ds
			}
			ds.Count++
			ds.CostUSD += task.Cost
			ds.InputTokens += task.InputTokens
			ds.OutputTokens += task.OutputTokens
		}
	}
	return stats, nil
}

// --- Task State Transitions ---

func (s *CoordinatorStore) MarkTaskQueued(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusQueued)},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	now := time.Now()
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusRunning)},
		{Path: "provider", Value: provider},
		{Path: "worktree_id", Value: worktreeID},
		{Path: "started_at", Value: now},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *coordinator.ExecuteResult) error {
	now := time.Now()
	updates := []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPendingApproval)},
		{Path: "worktree_path", Value: worktreePath},
		{Path: "base_branch", Value: baseBranch},
		{Path: "base_commit", Value: baseCommit},
		{Path: "completed_at", Value: now},
	}
	if result != nil {
		updates = append(updates,
			firestore.Update{Path: "output", Value: result.Output},
			firestore.Update{Path: "cost", Value: result.Cost},
			firestore.Update{Path: "tokens_used", Value: result.TokensUsed},
			firestore.Update{Path: "input_tokens", Value: result.InputTokens},
			firestore.Update{Path: "output_tokens", Value: result.OutputTokens},
			firestore.Update{Path: "session_id", Value: result.SessionID},
			firestore.Update{Path: "duration", Value: int64(result.Duration)},
		)
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, updates)
	if err != nil {
		return err
	}
	s.invalidateStatsCache()

	// Update in-memory cost counter.
	if result != nil && result.Cost > 0 {
		task, taskErr := s.GetTask(ctx, id)
		if taskErr == nil && task != nil {
			s.addCost(task.Provider, result.Cost)
		}
	}
	return nil
}

func (s *CoordinatorStore) MarkTaskCompleted(ctx context.Context, id string, result *coordinator.ExecuteResult) error {
	now := time.Now()
	updates := []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusCompleted)},
		{Path: "completed_at", Value: now},
	}
	if result != nil {
		updates = append(updates,
			firestore.Update{Path: "output", Value: result.Output},
			firestore.Update{Path: "cost", Value: result.Cost},
			firestore.Update{Path: "tokens_used", Value: result.TokensUsed},
			firestore.Update{Path: "input_tokens", Value: result.InputTokens},
			firestore.Update{Path: "output_tokens", Value: result.OutputTokens},
			firestore.Update{Path: "session_id", Value: result.SessionID},
			firestore.Update{Path: "duration", Value: int64(result.Duration)},
		)
		if !result.Success {
			updates = append(updates, firestore.Update{Path: "error", Value: result.Error})
		}
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, updates)
	if err != nil {
		return err
	}
	s.invalidateStatsCache()

	// Update in-memory cost counter.
	if result != nil && result.Cost > 0 {
		task, taskErr := s.GetTask(ctx, id)
		if taskErr == nil && task != nil {
			s.addCost(task.Provider, result.Cost)
		}
	}
	return nil
}

func (s *CoordinatorStore) MarkTaskFailed(ctx context.Context, id string, taskErr error) error {
	now := time.Now()
	errMsg := ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusFailed)},
		{Path: "error", Value: errMsg},
		{Path: "completed_at", Value: now},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskRejected(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusRejected)},
		{Path: "completed_at", Value: time.Now()},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskCancelled(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusCancelled)},
		{Path: "completed_at", Value: time.Now()},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) RequeueTask(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPending)},
		{Path: "started_at", Value: nil},
		{Path: "completed_at", Value: nil},
		{Path: "error", Value: ""},
		{Path: "output", Value: ""},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) ResetTaskToPending(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPending)},
		{Path: "worktree_id", Value: ""},
		{Path: "provider", Value: ""},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

// --- Duplicate Detection ---

func (s *CoordinatorStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, _ float64) (*coordinator.TaskRecord, error) {
	// Firestore doesn't support bitwise operations, so we do exact fingerprint match
	iter := s.client.Collection(collTasks).
		Where("fingerprint", "==", int64(fingerprint)).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapToTask(doc.Data()), nil
}

func (s *CoordinatorStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "fingerprint", Value: int64(fingerprint)},
	})
	return err
}

// --- Thread & Chain Linking ---

func (s *CoordinatorStore) SetTaskThreadID(ctx context.Context, id, threadID string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "thread_id", Value: threadID},
	})
	return err
}

func (s *CoordinatorStore) UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "chain_id", Value: chainID},
		{Path: "stage_id", Value: stageID},
	})
	return err
}

func (s *CoordinatorStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return "", "", "", err
	}
	// By convention, agent_id == inbox in agent config
	return task.AgentID, task.AgentID, task.Title, nil
}

// --- GitHub Integration ---

func (s *CoordinatorStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "github_issue", Value: issueNum},
	})
	return err
}

func (s *CoordinatorStore) SetTaskStage(ctx context.Context, id string, stage coordinator.TaskStage) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "stage", Value: string(stage)},
	})
	return err
}

func (s *CoordinatorStore) SetTaskDesignDocPath(ctx context.Context, id, path string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "design_doc_path", Value: path},
	})
	return err
}

func (s *CoordinatorStore) SetTaskSprintPlanPath(ctx context.Context, id, path string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "sprint_plan_path", Value: path},
	})
	return err
}

func (s *CoordinatorStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*coordinator.TaskRecord, error) {
	iter := s.client.Collection(collTasks).
		Where("github_issue", "==", issueNum).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var tasks []*coordinator.TaskRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, mapToTask(doc.Data()))
	}
	return tasks, nil
}

func (s *CoordinatorStore) GetTasksByStage(ctx context.Context, stage coordinator.TaskStage) ([]*coordinator.TaskRecord, error) {
	iter := s.client.Collection(collTasks).
		Where("stage", "==", string(stage)).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var tasks []*coordinator.TaskRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, mapToTask(doc.Data()))
	}
	return tasks, nil
}

// --- Resource Metrics ---

func (s *CoordinatorStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "peak_cpu", Value: peakCPU},
		{Path: "peak_memory_mb", Value: peakMemory},
	})
	return err
}

// --- Budget Tracking ---

// GetCostByProvider returns total cost per provider from the in-memory cache.
// Zero Firestore reads on the hot path — counters are synced in the background.
func (s *CoordinatorStore) GetCostByProvider() (map[string]float64, error) {
	s.costMu.RLock()
	defer s.costMu.RUnlock()

	// If counters haven't been loaded yet (StartCostSync not called), fall back to full scan.
	if !s.costLoaded {
		s.costMu.RUnlock()
		costs, err := s.fullScanCostByProvider(context.Background())
		s.costMu.RLock()
		return costs, err
	}

	result := make(map[string]float64, len(s.costCounters))
	for k, v := range s.costCounters {
		result[k] = v
	}
	return result, nil
}

// --- Approval Requests ---

func (s *CoordinatorStore) CreateApprovalRequest(ctx context.Context, req *coordinator.ApprovalRequestRecord) error {
	data := approvalToMap(req)
	_, err := s.client.Doc(collApprovals, req.ID).Set(ctx, data)
	return err
}

func (s *CoordinatorStore) GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error) {
	doc, err := s.client.Doc(collApprovals, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("approval not found: %s", id)
		}
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "pending").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no pending approval for task: %s", taskID)
	}
	if err != nil {
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) GetApprovalRequestByTaskAnyStatus(ctx context.Context, taskID string) (*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		OrderBy("created_at", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no approval for task: %s", taskID)
	}
	if err != nil {
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) ListPendingApprovals(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("status", "==", "pending").
		OrderBy("created_at", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

func (s *CoordinatorStore) ListResolvedApprovals(ctx context.Context, limit int) ([]*coordinator.ApprovalRequestRecord, error) {
	q := s.client.Collection(collApprovals).
		Where("status", "in", []string{"approved", "rejected"}).
		OrderBy("resolved_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

func (s *CoordinatorStore) ResolveApprovalRequest(ctx context.Context, id, resolveStatus, resolvedBy string) error {
	now := time.Now()
	_, err := s.client.Doc(collApprovals, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: resolveStatus},
		{Path: "resolved_by", Value: resolvedBy},
		{Path: "resolved_at", Value: now},
	})
	return err
}

func (s *CoordinatorStore) ResolveApprovalRequestByTask(ctx context.Context, taskID, resolveStatus, resolvedBy string) error {
	// Find the pending approval for this task, then resolve it
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "pending").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("no pending approval for task: %s", taskID)
	}
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "status", Value: resolveStatus},
		{Path: "resolved_by", Value: resolvedBy},
		{Path: "resolved_at", Value: now},
	})
	return err
}

func (s *CoordinatorStore) MarkApprovalHandoffsTriggered(ctx context.Context, taskID string) error {
	// Find approval for task and mark handoffs as triggered
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "approved").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil // No approval to update
	}
	if err != nil {
		return err
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "handoffs_triggered", Value: true},
	})
	return err
}

func (s *CoordinatorStore) ListApprovedMergeHandoffsWithoutTrigger(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error) {
	// Find approved merge_handoff approvals where handoffs haven't been triggered
	iter := s.client.Collection(collApprovals).
		Where("status", "==", "approved").
		Where("type", "==", "merge_handoff").
		Where("handoffs_triggered", "==", false).
		Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

// --- Cleanup ---

func (s *CoordinatorStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	iter := s.client.Collection(collTasks).
		Where("created_at", "<", cutoff).
		Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *CoordinatorStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-staleThreshold)

	// Find running/queued tasks older than threshold
	count := 0
	for _, taskStatus := range []string{"running", "queued"} {
		iter := s.client.Collection(collTasks).
			Where("status", "==", taskStatus).
			Where("created_at", "<", cutoff).
			Documents(ctx)

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return count, err
			}
			_, err = doc.Ref.Update(ctx, []firestore.Update{
				{Path: "status", Value: string(coordinator.TaskStatusCancelled)},
				{Path: "error", Value: "recovered: stale task from previous daemon run"},
				{Path: "completed_at", Value: time.Now()},
			})
			if err != nil {
				iter.Stop()
				return count, err
			}
			count++
		}
		iter.Stop()
	}
	return count, nil
}

func (s *CoordinatorStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	iter := s.client.Collection(collTasks).
		Where("status", "==", string(coordinator.TaskStatusFailed)).
		Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		_, err = doc.Ref.Update(ctx, []firestore.Update{
			{Path: "status", Value: string(coordinator.TaskStatusPending)},
			{Path: "error", Value: ""},
			{Path: "started_at", Value: nil},
			{Path: "completed_at", Value: nil},
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// --- Event Storage ---

func (s *CoordinatorStore) StoreTaskEvent(ctx context.Context, event *coordinator.TaskEventRecord) error {
	data := eventToMap(event)
	// Use subcollection: /tasks/{task_id}/events/{auto_id}
	_, _, err := s.client.Collection(collTasks).Doc(event.TaskID).Collection("events").Add(ctx, data)
	return err
}

func (s *CoordinatorStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*coordinator.TaskEventRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	iter := s.client.Collection(collTasks).Doc(taskID).Collection("events").
		OrderBy("created_at", firestore.Asc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var events []*coordinator.TaskEventRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		events = append(events, mapToEvent(doc.Data(), taskID))
	}
	return events, nil
}

// Compile-time check that CoordinatorStore implements coordinator.Store.
var _ coordinator.Store = (*CoordinatorStore)(nil)
