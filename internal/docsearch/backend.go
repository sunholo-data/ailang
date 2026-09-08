package docsearch

import "context"

// SearchBackend answers documentation searches. Implementations are selected
// by callers before a search is started.
type SearchBackend interface {
	Search(context.Context, SearchOptions) ([]SearchResult, SearchStats, error)
}

// LocalBackend preserves the existing filesystem search implementation.
type LocalBackend struct{}

func (LocalBackend) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, SearchStats, error) {
	return Search(ctx, opts)
}
