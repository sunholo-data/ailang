package effects

import (
	"fmt"
	"sync"
)

// PersistentSharedIndex is the optional durable index and its owned connection.
type PersistentSharedIndex interface {
	SharedIndex
	Close() error
}

// SharedIndexOpener opens an isolated index store.
type SharedIndexOpener func(string) (PersistentSharedIndex, error)

var indexOpenerMu sync.RWMutex
var indexOpener SharedIndexOpener

// RegisterSharedIndexOpener installs the platform backend.
func RegisterSharedIndexOpener(open SharedIndexOpener) {
	indexOpenerMu.Lock()
	defer indexOpenerMu.Unlock()
	indexOpener = open
}

// OpenSharedIndex opens durable storage; no backend is a configuration error.
func OpenSharedIndex(path string) (PersistentSharedIndex, error) {
	indexOpenerMu.RLock()
	open := indexOpener
	indexOpenerMu.RUnlock()
	if open == nil {
		return nil, fmt.Errorf("shared index backend for %s: %w", path, ErrBackendNotRegistered)
	}
	return open(path)
}

// CheckedSharedIndex is an optional operation-local error adapter. Persistent
// backends implement it so storage failures cannot masquerade as empty hits.
type CheckedSharedIndex interface {
	UpsertChecked(namespace, key string, simhash, version, timestamp int64) error
	DeleteChecked(namespace, key string) error
	UpsertWithEmbeddingChecked(namespace, key string, simhash int64, embedding []float64, version, timestamp int64) error
	FindSimilarSimHashChecked(namespace string, querySimHash int64, topK, maxScan int, mode DeterminismMode) ([]SearchResult, error)
	FindSimilarByEmbeddingChecked(namespace string, queryEmbedding []float64, topK, maxScan int, mode DeterminismMode) ([]SearchResult, error)
	EntryCountChecked(namespace string) (int, error)
	NamespacesChecked() ([]string, error)
}

// CheckedIndex adapts legacy in-memory implementations to checked operations.
func CheckedIndex(index SharedIndex) CheckedSharedIndex {
	if checked, ok := index.(CheckedSharedIndex); ok {
		return checked
	}
	return uncheckedIndex{index}
}

type uncheckedIndex struct{ SharedIndex }

func (i uncheckedIndex) UpsertChecked(namespace, key string, simhash, version, timestamp int64) error {
	i.Upsert(namespace, key, simhash, version, timestamp)
	return nil
}
func (i uncheckedIndex) DeleteChecked(namespace, key string) error {
	i.Delete(namespace, key)
	return nil
}
func (i uncheckedIndex) UpsertWithEmbeddingChecked(namespace, key string, simhash int64, embedding []float64, version, timestamp int64) error {
	i.UpsertWithEmbedding(namespace, key, simhash, embedding, version, timestamp)
	return nil
}
func (i uncheckedIndex) FindSimilarSimHashChecked(namespace string, querySimHash int64, topK, maxScan int, mode DeterminismMode) ([]SearchResult, error) {
	return i.FindSimilarSimHash(namespace, querySimHash, topK, maxScan, mode), nil
}
func (i uncheckedIndex) FindSimilarByEmbeddingChecked(namespace string, queryEmbedding []float64, topK, maxScan int, mode DeterminismMode) ([]SearchResult, error) {
	return i.FindSimilarByEmbedding(namespace, queryEmbedding, topK, maxScan, mode), nil
}
func (i uncheckedIndex) EntryCountChecked(namespace string) (int, error) {
	return i.EntryCount(namespace), nil
}
func (i uncheckedIndex) NamespacesChecked() ([]string, error) { return i.Namespaces(), nil }
