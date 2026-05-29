package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

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
// with a 5-minute TTL. This avoids a full collection scan on every API call.
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
