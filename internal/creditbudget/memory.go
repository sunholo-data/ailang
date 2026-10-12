package creditbudget

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryStore supports deterministic tests and examples only. Cloud admission
// must use a shared durable authority, never this process-local implementation.
type MemoryStore struct {
	mu      sync.Mutex
	records map[string]json.RawMessage
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[string]json.RawMessage)} }
func (m *MemoryStore) Run(ctx context.Context, account string, fn func(Transaction) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tx := &memoryTransaction{account: account, records: m.records, pending: make(map[string]json.RawMessage)}
	if err := fn(tx); err != nil {
		return err
	}
	for key, value := range tx.pending {
		m.records[key] = value
	}
	return nil
}

type memoryTransaction struct {
	account          string
	records, pending map[string]json.RawMessage
}

func (t *memoryTransaction) Get(collection, id string, into any) (bool, error) {
	b, ok := t.records[t.account+"/"+collection+"/"+id]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(b, into)
}
func (t *memoryTransaction) Put(collection, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	t.pending[t.account+"/"+collection+"/"+id] = b
	return nil
}

func (m *MemoryStore) ListRequests(ctx context.Context, accountID, taskID string, limit int) ([]Request, error) {
	if limit < 1 || limit > RequestInspectionLimit {
		return nil, fmt.Errorf("invalid request inspection bound")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := accountID + "/requests/"
	rows := []Request{}
	for key, b := range m.records {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		var r Request
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		if r.TaskID != taskID {
			continue
		}
		rows = append(rows, r)
		if len(rows) > limit {
			return nil, fmt.Errorf("request inspection truncated: more than %d requests", limit)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].RequestID < rows[j].RequestID })
	return rows, nil
}
