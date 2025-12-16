package docsearch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// EmbeddingCache stores document embeddings with model versioning
type EmbeddingCache struct {
	mu       sync.RWMutex
	entries  map[string]*CachedEmbedding // key: doc path
	model    string                      // current model name
	filePath string                      // cache file path
	dirty    bool                        // needs saving
}

// CachedEmbedding stores an embedding with metadata
type CachedEmbedding struct {
	Path      string    `json:"path"`
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmbeddingCacheFile is the JSON structure for persisting the cache
type EmbeddingCacheFile struct {
	Model   string                      `json:"model"`
	Entries map[string]*CachedEmbedding `json:"entries"`
}

// NewEmbeddingCache creates a new embedding cache
func NewEmbeddingCache(model string) *EmbeddingCache {
	// Determine cache file path
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".ailang", "cache")
	_ = os.MkdirAll(cacheDir, 0755) // Best effort, cache works without persistence

	cache := &EmbeddingCache{
		entries:  make(map[string]*CachedEmbedding),
		model:    model,
		filePath: filepath.Join(cacheDir, "doc_embeddings.json"),
	}

	// Try to load existing cache
	cache.load()

	return cache
}

// load reads cache from disk
func (c *EmbeddingCache) load() {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return // No cache file, start fresh
	}

	var file EmbeddingCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return // Invalid cache, start fresh
	}

	// Only load if model matches
	if file.Model == c.model {
		c.entries = file.Entries
	}
	// If model changed, entries remain empty (will be recomputed lazily)
}

// save writes cache to disk
func (c *EmbeddingCache) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	file := EmbeddingCacheFile{
		Model:   c.model,
		Entries: c.entries,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.filePath, data, 0644)
}

// Get retrieves a cached embedding if it exists and model matches
func (c *EmbeddingCache) Get(path string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok {
		return nil, false
	}

	// Check model matches
	if entry.Model != c.model {
		return nil, false
	}

	return entry.Embedding, true
}

// Set stores an embedding in the cache
func (c *EmbeddingCache) Set(path string, embedding []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CachedEmbedding{
		Path:      path,
		Embedding: embedding,
		Model:     c.model,
		UpdatedAt: time.Now(),
	}
	c.dirty = true
}

// Close saves the cache to disk
func (c *EmbeddingCache) Close() error {
	return c.save()
}

// neuralSearchImpl performs embedding-based semantic search
// This is the real implementation of neuralSearch
func neuralSearchImpl(candidates []DocFrame, query string, limit int, stats SearchStats) ([]SearchResult, SearchStats, error) {
	// Load embedding config
	cfg := messaging.LoadEmbedConfigFromEnv()

	// Check if embeddings are disabled
	if cfg.Provider == "none" {
		return nil, stats, fmt.Errorf("embeddings disabled (provider=none)")
	}

	// Create embedder
	embedder, err := messaging.NewOllamaEmbedder(cfg.Ollama)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to create embedder: %w", err)
	}

	modelName := embedder.ModelName()
	stats.EmbeddingModel = modelName

	// Create/load cache
	cache := NewEmbeddingCache(modelName)
	defer cache.Close()

	// Compute query embedding
	queryEmb, err := embedder.Embed(query)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to embed query: %w", err)
	}

	// Process candidates: get or compute embeddings
	type scoredDoc struct {
		doc   DocFrame
		emb   []float32
		score float64
	}

	var scoredDocs []scoredDoc

	for _, doc := range candidates {
		// Check cache first
		if cachedEmb, ok := cache.Get(doc.Path); ok {
			stats.EmbeddingsReused++
			score := messaging.CosineSimilarity(queryEmb, cachedEmb)
			scoredDocs = append(scoredDocs, scoredDoc{
				doc:   doc,
				emb:   cachedEmb,
				score: score,
			})
			continue
		}

		// Compute embedding
		emb, err := embedder.Embed(doc.Content)
		if err != nil {
			// Skip docs that fail to embed
			continue
		}

		// Cache the embedding
		cache.Set(doc.Path, emb)
		stats.EmbeddingsComputed++

		// Compute score
		score := messaging.CosineSimilarity(queryEmb, emb)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:   doc,
			emb:   emb,
			score: score,
		})
	}

	// Sort by score descending
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Take top results
	n := len(scoredDocs)
	if n > limit {
		n = limit
	}

	results := make([]SearchResult, n)
	for i := 0; i < n; i++ {
		results[i] = SearchResult{
			Path:  scoredDocs[i].doc.Path,
			Title: scoredDocs[i].doc.Title,
			Score: scoredDocs[i].score,
		}
	}

	return results, stats, nil
}

func init() {
	// Replace the stub neuralSearch with real implementation
	// This is done via init to avoid circular dependencies
}
