package docsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	corpus   string                      // corpus path (for per-corpus caching)
	filePath string                      // cache file path
	dirty    bool                        // needs saving
}

// CachedEmbedding stores an embedding with metadata
type CachedEmbedding struct {
	Path        string    `json:"path"`
	Embedding   []float32 `json:"embedding"`
	Model       string    `json:"model"`
	ContentHash string    `json:"content_hash"` // SHA256 of content for staleness detection
	UpdatedAt   time.Time `json:"updated_at"`
}

// EmbeddingCacheFile is the JSON structure for persisting the cache
type EmbeddingCacheFile struct {
	Model   string                      `json:"model"`
	Entries map[string]*CachedEmbedding `json:"entries"`
}

// NewEmbeddingCache creates a new embedding cache for a specific corpus
func NewEmbeddingCache(model, corpus string) *EmbeddingCache {
	// Determine cache file path
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".ailang", "cache", "embeddings")
	_ = os.MkdirAll(cacheDir, 0755) // Best effort, cache works without persistence

	// Generate corpus hash for cache filename
	corpusHash := hashCorpusPath(corpus)
	cacheFile := fmt.Sprintf("%s.json", corpusHash)

	cache := &EmbeddingCache{
		entries:  make(map[string]*CachedEmbedding),
		model:    model,
		corpus:   corpus,
		filePath: filepath.Join(cacheDir, cacheFile),
	}

	// Try to load existing cache
	cache.load()

	return cache
}

// hashCorpusPath generates a short hash of the corpus path for cache filenames
func hashCorpusPath(corpus string) string {
	h := sha256.Sum256([]byte(corpus))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 hex chars
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

// Get retrieves a cached embedding if it exists, model matches, and content is fresh
func (c *EmbeddingCache) Get(path, contentHash string) ([]float32, bool) {
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

	// Check content hash matches (if provided and stored)
	if contentHash != "" && entry.ContentHash != "" && entry.ContentHash != contentHash {
		return nil, false // Content changed, need to recompute
	}

	return entry.Embedding, true
}

// Set stores an embedding in the cache with content hash
func (c *EmbeddingCache) Set(path string, embedding []float32, contentHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CachedEmbedding{
		Path:        path,
		Embedding:   embedding,
		Model:       c.model,
		ContentHash: contentHash,
		UpdatedAt:   time.Now(),
	}
	c.dirty = true
}

// hashContent computes SHA256 hash of content for staleness detection
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// Close saves the cache to disk
func (c *EmbeddingCache) Close() error {
	return c.save()
}

// scoredDoc pairs a document with its embedding and similarity score.
type scoredDoc struct {
	doc   DocFrame
	emb   []float32
	score float64
}

// neuralSearchImpl performs embedding-based semantic search.
// The context bounds the overall operation — if it expires mid-embedding,
// results computed so far are returned (graceful degradation).
func neuralSearchImpl(ctx context.Context, candidates []DocFrame, query string, corpus string, limit int, stats SearchStats) ([]SearchResult, SearchStats, error) {
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

	// Create/load cache (per-corpus)
	cache := NewEmbeddingCache(modelName, corpus)
	defer cache.Close()

	// Compute query embedding
	queryEmb, err := embedder.Embed(query)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to embed query: %w", err)
	}

	// Process candidates: get or compute embeddings
	var scoredDocs []scoredDoc
	total := len(candidates)

	for i, doc := range candidates {
		// Check if overall timeout has fired — return partial results
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "⏱ Neural search timeout after %d/%d embeddings — returning partial results\n", i, total)
			return collectResults(scoredDocs, limit), stats, nil
		default:
		}

		// Compute content hash for staleness detection
		docHash := hashContent(doc.Content)

		// Check cache first (validates model AND content hash)
		if cachedEmb, ok := cache.Get(doc.Path, docHash); ok {
			stats.EmbeddingsReused++
			score := messaging.CosineSimilarity(queryEmb, cachedEmb)
			scoredDocs = append(scoredDocs, scoredDoc{
				doc:   doc,
				emb:   cachedEmb,
				score: score,
			})
			continue
		}

		// Show progress for embedding computation (slow operation)
		fmt.Fprintf(os.Stderr, "⏳ Embedding %d/%d: %s\n", i+1, total, truncatePath(doc.Path, 50))

		// Compute embedding (cache miss or stale)
		emb, err := embedder.Embed(doc.Content)
		if err != nil {
			// Skip docs that fail to embed
			continue
		}

		// Cache the embedding with content hash
		cache.Set(doc.Path, emb, docHash)
		stats.EmbeddingsComputed++

		// Compute score
		score := messaging.CosineSimilarity(queryEmb, emb)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:   doc,
			emb:   emb,
			score: score,
		})
	}

	return collectResults(scoredDocs, limit), stats, nil
}

// collectResults sorts scored docs by score and returns the top N as SearchResults.
func collectResults(scoredDocs []scoredDoc, limit int) []SearchResult {
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

	return results
}

// CacheInfo contains cache statistics
type CacheInfo struct {
	CorpusPath    string
	CacheFile     string
	Model         string
	EntryCount    int
	CacheSize     int64
	LastUpdated   time.Time
	OrphanedCount int
}

// CleanupResult contains cleanup operation results
type CleanupResult struct {
	RemovedCount int
	RemovedPaths []string
	OldSize      int64
	NewSize      int64
}

// GetCacheInfo returns cache statistics for a corpus
func GetCacheInfo(corpusPath string) (*CacheInfo, error) {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".ailang", "cache", "embeddings")
	corpusHash := hashCorpusPath(corpusPath)
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s.json", corpusHash))

	info := &CacheInfo{
		CorpusPath: corpusPath,
		CacheFile:  cacheFile,
	}

	// Check if cache file exists
	stat, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		return info, nil // Empty cache
	}
	if err != nil {
		return nil, fmt.Errorf("stat cache file: %w", err)
	}
	info.CacheSize = stat.Size()

	// Load cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("read cache file: %w", err)
	}

	var file EmbeddingCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse cache file: %w", err)
	}

	info.Model = file.Model
	info.EntryCount = len(file.Entries)

	// Find latest update time and count orphaned entries
	for path, entry := range file.Entries {
		if entry.UpdatedAt.After(info.LastUpdated) {
			info.LastUpdated = entry.UpdatedAt
		}
		// Check if file still exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			info.OrphanedCount++
		}
	}

	return info, nil
}

// CleanupCache removes orphaned entries from the cache
func CleanupCache(corpusPath string) (*CleanupResult, error) {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".ailang", "cache", "embeddings")
	corpusHash := hashCorpusPath(corpusPath)
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s.json", corpusHash))

	result := &CleanupResult{}

	// Check if cache file exists
	stat, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		return result, nil // Nothing to clean
	}
	if err != nil {
		return nil, fmt.Errorf("stat cache file: %w", err)
	}
	result.OldSize = stat.Size()

	// Load cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("read cache file: %w", err)
	}

	var file EmbeddingCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse cache file: %w", err)
	}

	// Find and remove orphaned entries
	newEntries := make(map[string]*CachedEmbedding)
	for path, entry := range file.Entries {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.RemovedCount++
			result.RemovedPaths = append(result.RemovedPaths, path)
		} else {
			newEntries[path] = entry
		}
	}

	if result.RemovedCount == 0 {
		result.NewSize = result.OldSize
		return result, nil
	}

	// Save cleaned cache
	file.Entries = newEntries
	newData, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, newData, 0644); err != nil {
		return nil, fmt.Errorf("write cache file: %w", err)
	}

	newStat, _ := os.Stat(cacheFile)
	if newStat != nil {
		result.NewSize = newStat.Size()
	}

	return result, nil
}

// truncatePath shortens a path for display, keeping the end
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

func init() {
	// Replace the stub neuralSearch with real implementation
	// This is done via init to avoid circular dependencies
}

// WarmupResult contains statistics from a cache warmup operation.
type WarmupResult struct {
	TotalDocs     int
	AlreadyCached int
	NewlyEmbedded int
	Failed        int
	Model         string
}

// WarmupCache pre-computes embeddings for all docs in a corpus, populating the cache.
// This prevents cold-cache hangs during interactive neural search.
// The context controls the overall timeout — warmup stops gracefully if it expires.
// If quiet is true, progress is not printed to stderr.
func WarmupCache(ctx context.Context, docsPath string, quiet bool) (*WarmupResult, error) {
	result := &WarmupResult{}

	// Discover all docs
	docs, err := discoverDocs(docsPath, "")
	if err != nil {
		return nil, fmt.Errorf("discovering docs: %w", err)
	}
	result.TotalDocs = len(docs)

	if len(docs) == 0 {
		return result, nil
	}

	// Load embedding config
	cfg := messaging.LoadEmbedConfigFromEnv()
	if cfg.Provider == "none" {
		return nil, fmt.Errorf("embeddings disabled (provider=none)")
	}

	// Create embedder
	embedder, err := messaging.NewOllamaEmbedder(cfg.Ollama)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	modelName := embedder.ModelName()
	result.Model = modelName

	// Load cache
	cache := NewEmbeddingCache(modelName, docsPath)
	defer cache.Close()

	for i, doc := range docs {
		// Check context
		select {
		case <-ctx.Done():
			if !quiet {
				fmt.Fprintf(os.Stderr, "⏱ Warmup stopped after %d/%d docs (timeout)\n", i, len(docs))
			}
			return result, nil
		default:
		}

		docHash := hashContent(doc.Content)

		// Skip if already cached and fresh
		if _, ok := cache.Get(doc.Path, docHash); ok {
			result.AlreadyCached++
			continue
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "⏳ Warming %d/%d: %s\n", i+1, len(docs), truncatePath(doc.Path, 50))
		}

		emb, err := embedder.Embed(doc.Content)
		if err != nil {
			result.Failed++
			continue
		}

		cache.Set(doc.Path, emb, docHash)
		result.NewlyEmbedded++
	}

	return result, nil
}
