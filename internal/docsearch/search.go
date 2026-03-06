// Package docsearch provides semantic search over design documentation.
// It implements a two-stage pipeline: SimHash shortlist → lazy embedding → cosine similarity.
package docsearch

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SearchOptions configures a documentation search
type SearchOptions struct {
	Query            string   // Search query text
	DocsPath         string   // Path to document corpus directory
	ExtraPaths       []string // Additional corpus directories to search (e.g., changelogs/)
	Subdir           string   // Filter by subdirectory pattern (e.g., "planned", "guides")
	Neural           bool     // Use neural embeddings (requires Ollama)
	NeuralCandidates int      // Max candidates for neural search
	Limit            int      // Max results to return
	JSON             bool     // Output as JSON
	Rebuild          bool     // Force rebuild of all embeddings (ignore cache)
}

// SearchResult represents a single search match
type SearchResult struct {
	Path  string  // Full path to document
	Title string  // Document title (from # heading)
	Score float64 // Similarity score (0-1)
}

// SearchStats provides search performance metrics
type SearchStats struct {
	TotalDocs          int    // Total docs in corpus
	SimHashCandidates  int    // Candidates from SimHash shortlist
	EmbeddingsComputed int    // New embeddings computed this search
	EmbeddingsReused   int    // Cached embeddings reused
	EmbeddingModel     string // Model used for embeddings
	SearchTimeMs       int64  // Total search time in milliseconds
}

// DocFrame represents a document with its metadata and optional embedding
type DocFrame struct {
	Path               string    // Full path to document
	Title              string    // Document title
	Content            string    // Full text content
	SimHash            uint64    // 64-bit SimHash of content
	Embedding          []float64 // Neural embedding (lazy computed)
	EmbeddingModel     string    // Model used for embedding
	EmbeddingUpdatedAt time.Time // When embedding was computed
}

// Search executes a documentation search with the given options.
// Returns results sorted by score (descending) and search statistics.
// The context controls the overall timeout for neural search — if it expires,
// partial results are returned gracefully (fallback to SimHash).
func Search(ctx context.Context, opts SearchOptions) ([]SearchResult, SearchStats, error) {
	startTime := time.Now()
	stats := SearchStats{}

	// Discover documents from primary path
	docs, err := discoverDocs(opts.DocsPath, opts.Subdir)
	if err != nil {
		return nil, stats, fmt.Errorf("discovering docs: %w", err)
	}

	// Discover documents from extra paths (e.g., changelogs/)
	for _, extra := range opts.ExtraPaths {
		extraDocs, extraErr := discoverDocs(extra, opts.Subdir)
		if extraErr != nil {
			continue // Skip missing/broken extra paths
		}
		docs = append(docs, extraDocs...)
	}

	stats.TotalDocs = len(docs)

	if len(docs) == 0 {
		return nil, stats, nil
	}

	// Stage 1: SimHash shortlist across ALL docs
	queryHash := simhash(opts.Query)
	candidates := simhashShortlist(docs, queryHash, opts.NeuralCandidates)
	stats.SimHashCandidates = len(candidates)

	var results []SearchResult

	if opts.Neural {
		// Group candidates by corpus path, run neural per-corpus, merge results.
		// This keeps embedding caches independent (each corpus gets its own cache file).
		grouped := groupByCorpus(candidates, opts.DocsPath, opts.ExtraPaths)
		var allResults []SearchResult
		for corpus, group := range grouped {
			corpusResults, corpusStats, corpusErr := neuralSearch(ctx, group, opts.Query, corpus, opts.Limit, stats)
			if corpusErr != nil {
				// Fallback to SimHash for this corpus
				corpusResults = simhashResults(group, opts.Limit)
			} else {
				stats.EmbeddingsComputed += corpusStats.EmbeddingsComputed
				stats.EmbeddingsReused += corpusStats.EmbeddingsReused
				if stats.EmbeddingModel == "" {
					stats.EmbeddingModel = corpusStats.EmbeddingModel
				}
			}
			allResults = append(allResults, corpusResults...)
		}
		// Re-sort merged results by score and trim to limit
		sort.Slice(allResults, func(i, j int) bool {
			return allResults[i].Score > allResults[j].Score
		})
		if len(allResults) > opts.Limit {
			allResults = allResults[:opts.Limit]
		}
		results = allResults
		if stats.EmbeddingModel == "" {
			stats.EmbeddingModel = "fallback-simhash"
		}
	} else {
		// SimHash-only search
		results = simhashResults(candidates, opts.Limit)
	}

	stats.SearchTimeMs = time.Since(startTime).Milliseconds()
	return results, stats, nil
}

// discoverDocs finds all markdown files in the docs path, filtered by subdir pattern
func discoverDocs(docsPath, subdir string) ([]DocFrame, error) {
	var docs []DocFrame

	// Debug: track subdir filter
	debug := os.Getenv("DEBUG_DOCSEARCH") == "1"
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] discoverDocs: docsPath=%q subdir=%q\n", docsPath, subdir)
	}

	err := filepath.Walk(docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip common directories that shouldn't be searched
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process markdown files (and .mdx for docusaurus)
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx") {
			return nil
		}

		// Filter by subdir pattern (if specified)
		if subdir != "" {
			// Check if path contains the subdir pattern
			pattern := "/" + subdir + "/"
			if !strings.Contains(path, pattern) {
				return nil
			}
		}

		// Parse document
		doc, err := parseDoc(path)
		if err != nil {
			return nil // Skip unparseable docs
		}

		docs = append(docs, doc)
		return nil
	})

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] discoverDocs: found %d docs\n", len(docs))
	}

	return docs, err
}

// parseDoc reads a markdown file and extracts title and content
func parseDoc(path string) (DocFrame, error) {
	file, err := os.Open(path)
	if err != nil {
		return DocFrame{}, err
	}
	defer file.Close()

	var title string
	var content strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")

		// Extract title from first # heading
		if title == "" && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
		}
	}

	contentStr := content.String()
	return DocFrame{
		Path:    path,
		Title:   title,
		Content: contentStr,
		SimHash: simhash(contentStr),
	}, nil
}

// simhash computes a 64-bit SimHash for the given text
// This is a simplified implementation using MD5 for determinism
func simhash(text string) uint64 {
	// Tokenize: split on whitespace and punctuation
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return 0
	}

	// Vector for bit accumulation
	var v [64]int

	for _, word := range words {
		// Hash each word to 64 bits using MD5 (deterministic)
		hash := md5.Sum([]byte(word))
		var h uint64
		for i := 0; i < 8; i++ {
			h = (h << 8) | uint64(hash[i])
		}

		// Accumulate into vector
		for i := 0; i < 64; i++ {
			if (h>>i)&1 == 1 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	// Convert vector to hash
	var result uint64
	for i := 0; i < 64; i++ {
		if v[i] > 0 {
			result |= 1 << i
		}
	}

	return result
}

// hammingDistance calculates the number of differing bits between two hashes
func hammingDistance(a, b uint64) int {
	x := a ^ b
	count := 0
	for x != 0 {
		count++
		x &= x - 1 // Clear lowest set bit
	}
	return count
}

// simhashShortlist returns the top candidates by SimHash similarity
func simhashShortlist(docs []DocFrame, queryHash uint64, maxCandidates int) []DocFrame {
	type scored struct {
		doc   DocFrame
		dist  int
		score float64
	}

	var scoredDocs []scored
	for _, doc := range docs {
		dist := hammingDistance(queryHash, doc.SimHash)
		score := 1.0 - float64(dist)/64.0
		scoredDocs = append(scoredDocs, scored{doc: doc, dist: dist, score: score})
	}

	// Sort by score descending
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Take top candidates
	n := len(scoredDocs)
	if n > maxCandidates {
		n = maxCandidates
	}

	result := make([]DocFrame, n)
	for i := 0; i < n; i++ {
		result[i] = scoredDocs[i].doc
	}

	return result
}

// simhashResults converts SimHash candidates to search results
func simhashResults(candidates []DocFrame, limit int) []SearchResult {
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]SearchResult, len(candidates))
	for i, doc := range candidates {
		// Score is already implicit from ordering (shortlist sorted by similarity)
		// Use a simple linear decay for display score
		results[i] = SearchResult{
			Path:  doc.Path,
			Title: doc.Title,
			Score: 1.0 - float64(i)*0.05, // Simple decay for display
		}
	}

	return results
}

// groupByCorpus partitions candidates by which corpus directory they belong to.
// This allows neural search to use per-corpus embedding caches.
func groupByCorpus(candidates []DocFrame, primary string, extras []string) map[string][]DocFrame {
	grouped := make(map[string][]DocFrame)
	for _, doc := range candidates {
		corpus := primary // default to primary
		for _, extra := range extras {
			if strings.HasPrefix(doc.Path, extra) {
				corpus = extra
				break
			}
		}
		grouped[corpus] = append(grouped[corpus], doc)
	}
	return grouped
}

// neuralSearch performs embedding-based semantic search (Stage 2-3)
// Delegates to neuralSearchImpl in embed.go. The context bounds the overall
// embedding time — if it expires, partial results are returned.
func neuralSearch(ctx context.Context, candidates []DocFrame, query string, corpus string, limit int, stats SearchStats) ([]SearchResult, SearchStats, error) {
	return neuralSearchImpl(ctx, candidates, query, corpus, limit, stats)
}
