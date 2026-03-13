package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/messaging"
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

// --- Subcommands ---

func runCacheSearch(args []string) {
	fs := flag.NewFlagSet("cache search", flag.ExitOnError)
	scope := fs.String("scope", "both", "Brain tier: both, user, project")
	ns := fs.String("namespace", "", "Filter by namespace")
	limit := fs.Int("limit", 10, "Maximum results")
	context := fs.String("context", "", "Comma-separated file paths to find related knowledge")
	fs.Parse(args)

	query := strings.Join(fs.Args(), " ")
	if query == "" && *context == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache search <query> [--scope user|project|both] [--namespace NS] [--limit N]")
		fmt.Fprintln(os.Stderr, "       ailang cache search --context FILE1,FILE2")
		os.Exit(1)
	}

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	start := time.Now()

	// If --context provided, build query from file paths
	if *context != "" {
		files := strings.Split(*context, ",")
		if query == "" {
			query = strings.Join(files, " ")
		}
	}

	// SimHash search + text search, merge results
	queryHash := builtins.SimHash(query)
	simResults := store.Search(*ns, queryHash, *limit, parseScope(*scope))
	textResults := store.SearchText(query, *ns, *limit, parseScope(*scope))

	// Merge: deduplicate by key, prefer SimHash score
	seen := make(map[string]bool)
	var merged []effects.BrainSearchResult
	for _, r := range simResults {
		seen[r.Frame.Key] = true
		merged = append(merged, r)
	}
	for _, r := range textResults {
		if !seen[r.Frame.Key] {
			merged = append(merged, r)
		}
	}

	if len(merged) > *limit {
		merged = merged[:*limit]
	}

	elapsed := time.Since(start)

	if len(merged) == 0 {
		fmt.Println("No results found.")
	} else {
		for i, r := range merged {
			age := formatAgeMillis(r.Frame.UpdatedAt)
			content := truncateClean(r.Frame.Content, 80)
			fmt.Printf("  %d. [%s] [%s] %s (score: %.2f)\n", i+1, r.Tier, r.Frame.Namespace, r.Frame.Key, r.Score)
			if content != "" {
				fmt.Printf("     %s — %s\n", age, content)
			}
		}
	}

	fmt.Printf("\nscope=%s results=%d query_ms=%d\n", *scope, len(merged), elapsed.Milliseconds())
}

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

func runCachePut(args []string) {
	fs := flag.NewFlagSet("cache put", flag.ExitOnError)
	ns := fs.String("ns", "learnings", "Namespace")
	content := fs.String("content", "", "Text content for search/similarity")
	scope := fs.String("scope", "project", "Brain tier: user, project")
	source := fs.String("source", "cli", "Source identifier")
	ttlStr := fs.String("ttl", "", "Time-to-live (e.g., 7d, 30d, 90d)")
	embed := fs.Bool("embed", false, "Compute and store embedding vector alongside SimHash")
	fs.Parse(args)

	key := strings.Join(fs.Args(), "_")
	if key == "" || *content == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache put <key> --content \"text\" [--ns NAME] [--scope user|project] [--embed]")
		os.Exit(1)
	}

	var opts []effects.CacheOption
	if *embed {
		embedder := createEmbedder()
		if embedder != nil {
			opts = append(opts, effects.WithEmbedder(embedder))
		}
	}

	store, err := openBrainStoreWithOpts(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	frame := effects.BrainFrame{
		Key:       key,
		Namespace: *ns,
		Value:     []byte(*content),
		SimHash:   builtins.SimHash(*content),
		Content:   *content,
		Version:   1,
		Source:    *source,
	}

	if *ttlStr != "" {
		var d humanDuration
		if err := d.Set(*ttlStr); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid TTL: %v\n", err)
			os.Exit(1)
		}
		expires := time.Now().Add(time.Duration(d)).UnixMilli()
		frame.ExpiresAt = &expires
	}

	if err := store.Put(frame, parseScope(*scope)); err != nil {
		fmt.Fprintf(os.Stderr, "Error storing frame: %v\n", err)
		os.Exit(1)
	}

	status := fmt.Sprintf("Stored: %s [%s] (scope: %s, simhash: %d", key, *ns, *scope, frame.SimHash)
	if frame.EmbeddingDim > 0 {
		status += fmt.Sprintf(", embedding: %d-dim %s", frame.EmbeddingDim, frame.EmbedModel)
	}
	status += ")"
	fmt.Println(status)
}

func runCachePutResolution(args []string) {
	fs := flag.NewFlagSet("cache put-resolution", flag.ExitOnError)
	commitMsg := fs.String("commit-msg", "", "Commit message")
	diffSummary := fs.String("diff-summary", "", "Git diff stat summary")
	files := fs.String("files", "", "Comma-separated changed files")
	source := fs.String("source", "hook:commit", "Source identifier")
	fs.Parse(args)

	if *commitMsg == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache put-resolution --commit-msg \"msg\" [--diff-summary \"...\"] [--files \"f1,f2\"]")
		os.Exit(1)
	}

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Build content from commit info
	content := *commitMsg
	if *diffSummary != "" {
		content += "\n\nFiles changed:\n" + *diffSummary
	}
	if *files != "" {
		content += "\n\nFiles: " + *files
	}

	// Generate key from timestamp
	key := fmt.Sprintf("resolution_%d", time.Now().UnixMilli())

	frame := effects.BrainFrame{
		Key:       key,
		Namespace: "resolutions",
		Value:     []byte(content),
		SimHash:   builtins.SimHash(content),
		Content:   content,
		Version:   1,
		Source:    *source,
	}

	if err := store.Put(frame, effects.ScopeProject); err != nil {
		fmt.Fprintf(os.Stderr, "Error storing resolution: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Resolution stored: %s\n", key)
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

func runCacheExport(args []string) {
	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	encoder := json.NewEncoder(os.Stdout)

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
		frames := tier.cache.ListRecent("", 100000) // all frames
		for _, f := range frames {
			record := map[string]interface{}{
				"tier":       tier.name,
				"key":        f.Key,
				"namespace":  f.Namespace,
				"value":      string(f.Value),
				"simhash":    f.SimHash,
				"content":    f.Content,
				"version":    f.Version,
				"created_at": f.CreatedAt,
				"updated_at": f.UpdatedAt,
				"source":     f.Source,
			}
			if f.ExpiresAt != nil {
				record["expires_at"] = *f.ExpiresAt
			}
			encoder.Encode(record)
		}
	}
}

func runCacheImport(args []string) {
	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	var count int
	for scanner.Scan() {
		var record map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			fmt.Fprintf(os.Stderr, "Skipping invalid JSON line: %v\n", err)
			continue
		}

		tier := "project"
		if t, ok := record["tier"].(string); ok {
			tier = t
		}

		frame := effects.BrainFrame{
			Key:       getString(record, "key"),
			Namespace: getString(record, "namespace"),
			Value:     []byte(getString(record, "value")),
			SimHash:   getInt64(record, "simhash"),
			Content:   getString(record, "content"),
			Version:   int(getInt64(record, "version")),
			CreatedAt: getInt64(record, "created_at"),
			UpdatedAt: getInt64(record, "updated_at"),
			Source:    getString(record, "source"),
		}
		if exp, ok := record["expires_at"].(float64); ok {
			e := int64(exp)
			frame.ExpiresAt = &e
		}

		store.Put(frame, parseScope(tier))
		count++
	}

	fmt.Printf("Imported %d frame(s)\n", count)
}

func runCacheEmbed(args []string) {
	fs := flag.NewFlagSet("cache embed", flag.ExitOnError)
	ns := fs.String("namespace", "", "Only backfill in this namespace")
	scope := fs.String("scope", "both", "Brain tier: both, user, project")
	fs.Parse(args)

	embedder := createEmbedder()
	if embedder == nil {
		fmt.Fprintln(os.Stderr, "Error: no embedder available. Set AILANG_EMBED_PROVIDER or configure brain.embedding in config.")
		os.Exit(1)
	}

	store, err := openBrainStoreWithOpts(effects.WithEmbedder(embedder))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	fmt.Printf("Backfilling embeddings with %s (dim=%d)...\n", embedder.ModelName(), embedder.Dimension())

	var totalProcessed, totalErrors int

	sc := parseScope(*scope)
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
		if sc != effects.ScopeBoth && string(sc) != tier.name {
			continue
		}

		tier.cache.SetEmbedder(embedder)
		processed, errors := tier.cache.BackfillEmbeddings(*ns)
		totalProcessed += processed
		totalErrors += errors
		if processed > 0 || errors > 0 {
			fmt.Printf("  %s: embedded %d frames (%d errors)\n", tier.name, processed, errors)
		}
	}

	if totalProcessed == 0 && totalErrors == 0 {
		fmt.Println("All frames already have embeddings (nothing to backfill).")
	} else {
		fmt.Printf("Done: %d embedded, %d errors\n", totalProcessed, totalErrors)
	}
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
  put-resolution      Store a commit resolution frame
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
