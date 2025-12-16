package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/docsearch"
)

// docsSearchCommand implements `ailang docs search` subcommand
// Searches design docs using SimHash (fast) or neural embeddings (--neural)
func docsSearchCommand(args []string) {
	searchFlags := flag.NewFlagSet("docs search", flag.ExitOnError)

	// Flags
	pathFlag := searchFlags.String("path", "", "Document corpus path (default: design_docs)")
	subdirFlag := searchFlags.String("subdir", "", "Filter by subdirectory pattern")
	streamFlag := searchFlags.String("stream", "", "Filter by stream: planned, implemented, all (alias for --subdir, design_docs only)")
	neuralFlag := searchFlags.Bool("neural", false, "Use neural embeddings for semantic search")
	neuralCandidates := searchFlags.Int("neural-candidates", 0, "Max candidates for neural search (default: max(200, 20*limit))")
	limitFlag := searchFlags.Int("limit", 10, "Maximum results to return")
	jsonFlag := searchFlags.Bool("json", false, "Output results as JSON")
	helpFlag := searchFlags.Bool("help", false, "Show help for docs search")

	// Cache management flags
	cacheInfoFlag := searchFlags.Bool("cache-info", false, "Show embedding cache statistics")
	cleanupFlag := searchFlags.Bool("cleanup", false, "Remove orphaned cache entries")
	rebuildFlag := searchFlags.Bool("rebuild", false, "Force rebuild of all embeddings")

	if err := searchFlags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *helpFlag {
		printDocsSearchHelp()
		return
	}

	// Handle cache management commands (no query needed)
	if *cacheInfoFlag {
		showCacheInfo(*pathFlag)
		return
	}
	if *cleanupFlag {
		runCacheCleanup(*pathFlag)
		return
	}

	// Need a query for search
	if searchFlags.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: search query required\n", red("Error"))
		fmt.Fprintln(os.Stderr, "\nUsage: ailang docs search \"<query>\" [flags]")
		os.Exit(1)
	}

	query := searchFlags.Arg(0)

	// Determine docs path
	var docsPath string
	var err error
	if *pathFlag != "" {
		// User specified path
		docsPath, err = filepath.Abs(*pathFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid path: %v\n", red("Error"), err)
			os.Exit(1)
		}
		if info, statErr := os.Stat(docsPath); statErr != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s: path not found or not a directory: %s\n", red("Error"), *pathFlag)
			os.Exit(1)
		}
	} else {
		// Default to design_docs
		docsPath, err = findDesignDocsDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
	}

	// Handle --stream as alias for --subdir (backwards compatibility)
	subdir := *subdirFlag
	if *streamFlag != "" && subdir == "" {
		// Map stream values to subdir patterns
		switch *streamFlag {
		case "planned":
			subdir = "planned"
		case "implemented":
			subdir = "implemented"
		case "all":
			subdir = ""
		default:
			subdir = *streamFlag // Allow arbitrary values
		}
	}

	// Calculate neural candidates default
	candidates := *neuralCandidates
	if candidates == 0 {
		candidates = max(200, 20*(*limitFlag))
	}

	// Create search options
	opts := docsearch.SearchOptions{
		Query:            query,
		DocsPath:         docsPath,
		Subdir:           subdir,
		Neural:           *neuralFlag,
		NeuralCandidates: candidates,
		Limit:            *limitFlag,
		JSON:             *jsonFlag,
		Rebuild:          *rebuildFlag,
	}

	// Run search
	results, stats, err := docsearch.Search(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Output results
	if *jsonFlag {
		printJSONResults(results, stats)
	} else {
		printResults(query, results, stats, *neuralFlag)
	}
}

// findDesignDocsDir finds the design_docs directory
func findDesignDocsDir() (string, error) {
	// Check common locations
	candidates := []string{
		"design_docs",
		"../design_docs",
		"../../design_docs",
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("design_docs directory not found")
}

// printResults displays search results in human-readable format
func printResults(query string, results []docsearch.SearchResult, stats docsearch.SearchStats, neural bool) {
	// Print header
	if neural {
		fmt.Printf("🔍 Neural search: %q\n", query)
		fmt.Printf("   SimHash candidates: %d (from %d total docs)\n", stats.SimHashCandidates, stats.TotalDocs)
		fmt.Printf("   Embeddings: %d computed, %d reused (model: %s)\n",
			stats.EmbeddingsComputed, stats.EmbeddingsReused, stats.EmbeddingModel)
	} else {
		fmt.Printf("🔍 SimHash search: %q\n", query)
		fmt.Printf("   Scanned: %d docs\n", stats.TotalDocs)
	}
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No matching documents found.")
		return
	}

	// Print results
	for i, r := range results {
		// Show relative path for cleaner output
		relPath := r.Path
		if strings.Contains(relPath, "design_docs") {
			parts := strings.Split(relPath, "design_docs")
			if len(parts) > 1 {
				relPath = "design_docs" + parts[len(parts)-1]
			}
		}

		fmt.Printf("%d. %s (%.2f)\n", i+1, relPath, r.Score)
		if r.Title != "" {
			fmt.Printf("   %s\n", r.Title)
		}
	}
}

// printJSONResults outputs results as JSON
func printJSONResults(results []docsearch.SearchResult, stats docsearch.SearchStats) {
	fmt.Println("{")
	fmt.Printf("  \"stats\": {\n")
	fmt.Printf("    \"total_docs\": %d,\n", stats.TotalDocs)
	fmt.Printf("    \"simhash_candidates\": %d,\n", stats.SimHashCandidates)
	fmt.Printf("    \"embeddings_computed\": %d,\n", stats.EmbeddingsComputed)
	fmt.Printf("    \"embeddings_reused\": %d,\n", stats.EmbeddingsReused)
	fmt.Printf("    \"embedding_model\": %q,\n", stats.EmbeddingModel)
	fmt.Printf("    \"search_time_ms\": %d\n", stats.SearchTimeMs)
	fmt.Printf("  },\n")
	fmt.Printf("  \"results\": [\n")
	for i, r := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		fmt.Printf("    {\"path\": %q, \"title\": %q, \"score\": %.4f}%s\n",
			r.Path, r.Title, r.Score, comma)
	}
	fmt.Printf("  ]\n")
	fmt.Println("}")
}

// printDocsSearchHelp prints help for docs search command
func printDocsSearchHelp() {
	fmt.Println("Usage: ailang docs search [flags] \"<query>\"")
	fmt.Println()
	fmt.Println("Search documentation using SimHash or neural embeddings.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --path <dir>         Document corpus path (default: design_docs)")
	fmt.Println("  --subdir <pattern>   Filter by subdirectory (e.g., 'guides', 'planned')")
	fmt.Println("  --stream <stream>    Alias for --subdir (backwards compatibility)")
	fmt.Println("  --neural             Use neural embeddings for semantic search (requires Ollama)")
	fmt.Println("  --neural-candidates  Max candidates for neural search (default: max(200, 20*limit))")
	fmt.Println("  --limit <n>          Maximum results to return (default: 10)")
	fmt.Println("  --json               Output results as JSON")
	fmt.Println("  --rebuild            Force rebuild of all embeddings (ignore cache)")
	fmt.Println("  --help               Show this help message")
	fmt.Println()
	fmt.Println("Cache Management:")
	fmt.Println("  --cache-info         Show embedding cache statistics")
	fmt.Println("  --cleanup            Remove orphaned cache entries (files that no longer exist)")
	fmt.Println()
	fmt.Println("Note: Flags must come BEFORE the query.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang docs search \"parser error\"")
	fmt.Println("  ailang docs search --stream planned \"type inference\"")
	fmt.Println("  ailang docs search --path docs \"semantic caching\"")
	fmt.Println("  ailang docs search --path docs --subdir guides \"getting started\"")
	fmt.Println("  ailang docs search --neural \"semantic search\"")
	fmt.Println("  ailang docs search --limit 5 --json \"builtin\"")
	fmt.Println("  ailang docs search --cache-info")
	fmt.Println("  ailang docs search --cleanup")
	fmt.Println()
	fmt.Println("Neural Search:")
	fmt.Println("  When using --neural, embeddings are computed lazily only for the")
	fmt.Println("  bounded SimHash candidate set. Embeddings are cached per-corpus with")
	fmt.Println("  model version tagging - subsequent searches reuse cached embeddings.")
	fmt.Println("  Content changes are detected via SHA256 hash - stale entries auto-recompute.")
}

// max returns the larger of a or b
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FrameMetadata stores embedding information for lazy caching
type FrameMetadata struct {
	EmbeddingModel     string    `json:"embedding_model,omitempty"`
	EmbeddingUpdatedAt time.Time `json:"embedding_updated_at,omitempty"`
	EmbeddingDim       int       `json:"embedding_dim,omitempty"`
}

// showCacheInfo displays embedding cache statistics
func showCacheInfo(pathFlag string) {
	// Determine corpus path
	var corpusPath string
	var err error
	if pathFlag != "" {
		corpusPath, err = filepath.Abs(pathFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid path: %v\n", red("Error"), err)
			os.Exit(1)
		}
	} else {
		corpusPath, err = findDesignDocsDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
	}

	info, err := docsearch.GetCacheInfo(corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Println("📦 Embedding Cache Info:")
	fmt.Printf("   Corpus: %s\n", info.CorpusPath)
	fmt.Printf("   Cache file: %s\n", info.CacheFile)
	fmt.Printf("   Model: %s\n", info.Model)
	fmt.Printf("   Entries: %d\n", info.EntryCount)
	fmt.Printf("   Cache size: %.2f KB\n", float64(info.CacheSize)/1024)
	if !info.LastUpdated.IsZero() {
		fmt.Printf("   Last updated: %s\n", info.LastUpdated.Format("2006-01-02 15:04:05"))
	}
	if info.OrphanedCount > 0 {
		fmt.Printf("\n   ⚠️  Orphaned entries: %d (use --cleanup to remove)\n", info.OrphanedCount)
	}
}

// runCacheCleanup removes orphaned cache entries
func runCacheCleanup(pathFlag string) {
	// Determine corpus path
	var corpusPath string
	var err error
	if pathFlag != "" {
		corpusPath, err = filepath.Abs(pathFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid path: %v\n", red("Error"), err)
			os.Exit(1)
		}
	} else {
		corpusPath, err = findDesignDocsDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
	}

	result, err := docsearch.CleanupCache(corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Println("🧹 Cache cleanup:")
	if result.RemovedCount == 0 {
		fmt.Println("   No orphaned entries found.")
	} else {
		fmt.Printf("   Removed %d orphaned entries:\n", result.RemovedCount)
		for _, path := range result.RemovedPaths {
			fmt.Printf("   - %s\n", path)
		}
		fmt.Printf("\n   Cache size: %.2f KB → %.2f KB\n",
			float64(result.OldSize)/1024, float64(result.NewSize)/1024)
	}
}
