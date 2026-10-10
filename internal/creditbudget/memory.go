package creditbudget

import (
	"context"
	"encoding/json"
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
