package firestore

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/api/iterator"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

const (
	collTasks     = "tasks"
	collApprovals = "approvals"
	collMeta      = "_meta"

	costCountersDocID = "cost_counters"
	costSyncInterval  = 5 * time.Minute
	statsCacheTTL     = 300 * time.Second // 5 min (M-COST2: was 30s)
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
// Bounded to 10000 documents to limit Firestore read costs (M-COST2).
func (s *CoordinatorStore) fullScanCostByProvider(ctx context.Context) (map[string]float64, error) {
	const limit = 10000
	iter := s.client.Collection(collTasks).Limit(limit).Documents(ctx)
	defer iter.Stop()

	costs := make(map[string]float64)
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		count++
		data := doc.Data()
		provider, _ := data["provider"].(string)
		cost, _ := data["cost"].(float64)
		if provider != "" {
			costs[provider] += cost
		}
	}
	if count >= limit {
		log.Printf("WARNING: fullScanCostByProvider hit limit of %d documents — costs may be incomplete", limit)
	}
	return costs, nil
}

// Close stops the cost sync goroutine and closes the underlying client.
func (s *CoordinatorStore) Close() error {
	s.StopCostSync()
	return s.client.Close()
}

// invalidateStatsCache clears the cached stats so the next call re-scans.
// Called on every state transition to keep the cache fresh.
func (s *CoordinatorStore) invalidateStatsCache() {
	s.statsMu.Lock()
	s.cachedStats = nil
	s.statsMu.Unlock()
}

// fullScanTaskStats performs a full collection scan to compute task statistics.
// Bounded to 10000 documents to limit Firestore read costs (M-COST2).
func (s *CoordinatorStore) fullScanTaskStats(ctx context.Context) (*coordinator.TaskStats, error) {
	const limit = 10000
	iter := s.client.Collection(collTasks).Limit(limit).Documents(ctx)
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
	if stats.TotalTasks >= limit {
		log.Printf("WARNING: fullScanTaskStats hit limit of %d documents — stats may be incomplete", limit)
	}
	return stats, nil
}

// Compile-time check that CoordinatorStore implements coordinator.Store.
var _ coordinator.Store = (*CoordinatorStore)(nil)
