package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// cacheCommand handles the 'cache' subcommand for the AILANG brain.
// Two-tier: user (~/.ailang/state/brain.db) + project (.ailang/state/brain.db).
func cacheCommand() {
	if len(os.Args) < 3 {
		printCacheHelp()
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "search":
		runCacheSearch(args)
	case "list", "ls":
		runCacheList(args)
	case "show":
		runCacheShow(args)
	case "put":
		runCachePut(args)
	case "put-resolution":
		runCachePutResolution(args)
	case "promote":
		runCachePromote(args)
	case "stats":
		runCacheStats(args)
	case "gc":
		runCacheGC(args)
	case "export":
		runCacheExport(args)
	case "import":
		runCacheImport(args)
	case "embed":
		runCacheEmbed(args)
	case "put-vector":
		runCachePutVector(args)
	case "compile-clear":
		runCompileCacheClear()
	case "compile-stats":
		runCompileCacheStats()
	case "--help", "-h", "help":
		printCacheHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subCmd)
		printCacheHelp()
		os.Exit(1)
	}
}

// --- Brain store helpers ---

func openBrainStore() (*effects.BrainStore, error) {
	userDB := getUserBrainPath()
	projectDB := getProjectBrainPath()
	return effects.NewBrainStore(userDB, projectDB)
}

func openBrainStoreWithOpts(opts ...effects.CacheOption) (*effects.BrainStore, error) {
	userDB := getUserBrainPath()
	projectDB := getProjectBrainPath()
	return effects.NewBrainStore(userDB, projectDB, opts...)
}

// createEmbedder creates an embedder from config/env.
// Returns nil if no embedder available (graceful fallback).
func createEmbedder() effects.Embedder {
	cfg := messaging.LoadEmbedConfigFromEnv()
	embedder, err := messaging.NewEmbedderFromConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: embedder unavailable (%v), falling back to SimHash\n", err)
		return nil
	}
	return embedder
}

func getUserBrainPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ailang", "state", "brain.db")
}

func getProjectBrainPath() string {
	// Walk up to find .ailang/ directory (project root marker)
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".ailang")); err == nil {
			return filepath.Join(dir, ".ailang", "state", "brain.db")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: use cwd
	return filepath.Join(".ailang", "state", "brain.db")
}

func parseScope(s string) effects.BrainScope {
	switch s {
	case "user":
		return effects.ScopeUser
	case "project":
		return effects.ScopeProject
	default:
		return effects.ScopeBoth
	}
}

// --- Subcommands (simple ops) ---

func runCacheList(args []string) {
	fs := flag.NewFlagSet("cache list", flag.ExitOnError)
	ns := fs.String("namespace", "", "Filter by namespace")
	limit := fs.Int("limit", 10, "Maximum results")
	scope := fs.String("scope", "both", "Brain tier: both, user, project")
	fs.Parse(args)

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	var frames []effects.BrainFrame

	sc := parseScope(*scope)
	if sc == effects.ScopeBoth || sc == effects.ScopeProject {
		if store.Project != nil {
			frames = append(frames, store.Project.ListRecent(*ns, *limit)...)
		}
	}
	if sc == effects.ScopeBoth || sc == effects.ScopeUser {
		if store.User != nil {
			frames = append(frames, store.User.ListRecent(*ns, *limit)...)
		}
	}

	if len(frames) > *limit {
		frames = frames[:*limit]
	}

	if len(frames) == 0 {
		fmt.Println("Brain is empty.")
		return
	}

	for _, f := range frames {
		age := formatAgeMillis(f.UpdatedAt)
		content := truncateClean(f.Content, 70)
		fmt.Printf("  [%s] %s — %s\n", f.Namespace, f.Key, age)
		if content != "" {
			fmt.Printf("     %s\n", content)
		}
	}
	fmt.Printf("\nShowing %d frame(s)\n", len(frames))
}

func runCacheShow(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache show <key>")
		os.Exit(1)
	}
	key := args[0]

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Try project first, then user
	for _, tier := range []struct {
		name  string
		cache *effects.SQLiteSharedCache
	}{
		{"project", store.Project},
		{"user", store.User},
	} {
		if tier.cache == nil {
			continue
		}
		val, ok := tier.cache.Get(key)
		if ok {
			fmt.Printf("Key:       %s\n", key)
			fmt.Printf("Tier:      %s\n", tier.name)
			fmt.Printf("Value:     %s\n", string(val))
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Frame not found: %s\n", key)
	os.Exit(1)
}

func runCachePromote(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache promote <key>")
		os.Exit(1)
	}

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if store.Promote(args[0]) {
		fmt.Printf("Promoted: %s (project → user)\n", args[0])
	} else {
		fmt.Fprintf(os.Stderr, "Frame not found in project brain: %s\n", args[0])
		os.Exit(1)
	}
}

func runCacheStats(args []string) {
	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	allStats := store.Stats()
	embStats := store.EmbeddingStats()

	for tier, stats := range allStats {
		fmt.Printf("━━━ %s brain ━━━\n", tier)
		fmt.Printf("  Total frames: %d\n", stats.TotalFrames)
		if stats.TotalFrames > 0 {
			fmt.Printf("  Oldest: %s\n", formatAgeMillis(stats.OldestFrame))
			fmt.Printf("  Newest: %s\n", formatAgeMillis(stats.NewestFrame))
		}
		for ns, count := range stats.Namespaces {
			fmt.Printf("  [%s]: %d\n", ns, count)
		}
		if es, ok := embStats[tier]; ok {
			pct := 0.0
			if es.Total > 0 {
				pct = float64(es.WithEmbedding) / float64(es.Total) * 100
			}
			fmt.Printf("  With embeddings: %d (%.0f%%)\n", es.WithEmbedding, pct)
			for model, count := range es.Models {
				fmt.Printf("    %s: %d\n", model, count)
			}
		}
		fmt.Println()
	}
}

func runCacheGC(args []string) {
	fs := flag.NewFlagSet("cache gc", flag.ExitOnError)
	var olderThan humanDuration
	fs.Var(&olderThan, "older-than", "Remove frames older than duration (e.g., 30d, 90d)")
	ns := fs.String("namespace", "", "Only GC in this namespace")
	fs.Parse(args)

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	var totalRemoved int64

	for _, tier := range []struct {
		name  string
		cache *effects.SQLiteSharedCache
	}{
		{"project", store.Project},
		{"user", store.User},
	} {
		if tier.cache == nil {
			continue
		}

		// TTL-based GC (always)
		removed, err := tier.cache.GarbageCollect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "GC error (%s): %v\n", tier.name, err)
			continue
		}
		totalRemoved += removed

		// Age-based GC (if --older-than specified)
		if time.Duration(olderThan) > 0 && *ns != "" {
			aged, err := tier.cache.GarbageCollectOlderThan(*ns, time.Duration(olderThan))
			if err != nil {
				fmt.Fprintf(os.Stderr, "GC error (%s, %s): %v\n", tier.name, *ns, err)
				continue
			}
			totalRemoved += aged
		}
	}

	fmt.Printf("Removed %d expired/old frame(s)\n", totalRemoved)
}

// --- Helpers ---

func formatAgeMillis(unixMilli int64) string {
	if unixMilli == 0 {
		return "unknown"
	}
	age := time.Since(time.UnixMilli(unixMilli))
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(age.Hours()/24))
	}
}

func truncateClean(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func printCacheHelp() {
	fmt.Println(`Usage: ailang cache <subcommand> [options]

The AILANG brain — persistent semantic cache for coding knowledge.
Two-tier: user (cross-project) + project (repo-specific).

Subcommands:
  search <query>      Search brain by SimHash similarity + keyword match
  list                List recent frames
  show <key>          Show full frame detail
  put <key>           Store a frame manually
  put-resolution      Store a commit resolution frame (supports --embed, --enrich)
  promote <key>       Copy frame from project → user brain
  embed               Backfill embeddings for frames missing them
  stats               Show brain statistics
  gc                  Garbage collect expired frames
  export              Export all frames as JSONL (stdout)
  import              Import frames from JSONL (stdin)

Search options:
  --scope user|project|both   Brain tier (default: both)
  --namespace NAME            Filter by namespace
  --limit N                   Maximum results (default: 10)
  --context FILE1,FILE2       Find knowledge about these files

Put options:
  --ns NAME          Namespace (default: learnings)
  --content "text"   Text content (required)
  --scope user|project  Brain tier (default: project)
  --ttl 30d          Time-to-live (optional)
  --embed            Compute embedding vector alongside SimHash

Embed options:
  --namespace NAME   Only backfill in this namespace
  --scope both|user|project  Brain tier (default: both)

Examples:
  ailang cache search "type inference bug"
  ailang cache put fix_unify --content "Always check occurs in unification"
  ailang cache put --embed --content "sync.Pool pattern" sync_pool_tip
  ailang cache embed                     # backfill all missing embeddings
  ailang cache embed --namespace learnings
  ailang cache stats
  ailang cache export > backup.jsonl`)
}
