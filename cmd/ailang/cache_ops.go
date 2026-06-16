package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embedprefix"
)

func runCacheSearch(args []string) {
	fs := flag.NewFlagSet("cache search", flag.ExitOnError)
	scope := fs.String("scope", "both", "Brain tier: both, user, project")
	ns := fs.String("namespace", "", "Filter by namespace")
	limit := fs.Int("limit", 10, "Maximum results")
	context := fs.String("context", "", "Comma-separated file paths to find related knowledge")
	cosineOnly := fs.Bool("cosine", false, "Force cosine (embedding) search only")
	simhashOnly := fs.Bool("simhash", false, "Force SimHash search only (fast path)")
	jsonOut := fs.Bool("json", false, "Output results as JSON (for tool consumption, e.g., micro-rag engine)")
	fs.Parse(args)

	query := strings.Join(fs.Args(), " ")
	if query == "" && *context == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache search <query> [--cosine] [--simhash] [--scope both|user|project]")
		os.Exit(1)
	}

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	start := time.Now()

	if *context != "" {
		files := strings.Split(*context, ",")
		if query == "" {
			query = strings.Join(files, " ")
		}
	}

	queryHash := builtins.SimHash(query)
	sc := parseScope(*scope)

	var merged []effects.BrainSearchResult

	switch {
	case *cosineOnly:
		// Cosine-only: need embedder to compute query embedding
		embedder := createEmbedder()
		if embedder == nil {
			fmt.Fprintln(os.Stderr, "Error: --cosine requires an embedder. Set AILANG_EMBED_PROVIDER.")
			os.Exit(1)
		}
		queryEmb, err := embedprefix.EmbedWithRole(embedder, embedprefix.RoleQuery, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error computing query embedding: %v\n", err)
			os.Exit(1)
		}
		merged = store.SearchByEmbedding(queryEmb, *ns, *limit, sc)

	case *simhashOnly:
		// SimHash-only fast path
		merged = store.Search(*ns, queryHash, *limit, sc)

	default:
		// Three-tier: cosine (if embedder available) > SimHash > text
		var queryEmb []float32
		embedder := createEmbedder()
		if embedder != nil {
			if emb, err := embedprefix.EmbedWithRole(embedder, embedprefix.RoleQuery, query); err == nil {
				queryEmb = emb
			}
		}
		merged = store.SearchThreeTier(query, queryHash, queryEmb, *ns, *limit, sc)
	}

	elapsed := time.Since(start)

	if *jsonOut {
		emitCacheSearchJSON(merged, *scope, elapsed)
		return
	}

	if len(merged) == 0 {
		fmt.Println("No results found.")
	} else {
		for i, r := range merged {
			age := formatAgeMillis(r.Frame.UpdatedAt)
			content := truncateClean(r.Frame.Content, 80)
			embTag := ""
			if r.Frame.EmbeddingDim > 0 {
				embTag = " [emb]"
			}
			fmt.Printf("  %d. [%s] [%s] %s (score: %.2f)%s\n", i+1, r.Tier, r.Frame.Namespace, r.Frame.Key, r.Score, embTag)
			if content != "" {
				fmt.Printf("     %s — %s\n", age, content)
			}
		}
	}

	mode := "three-tier"
	if *cosineOnly {
		mode = "cosine"
	} else if *simhashOnly {
		mode = "simhash"
	}
	fmt.Printf("\nmode=%s scope=%s results=%d query_ms=%d\n", mode, *scope, len(merged), elapsed.Milliseconds())
}

// emitCacheSearchJSON prints search results as a stable JSON envelope.
// Consumed by the micro-rag engine and any other tool that needs structured
// search output. Field names are deliberately stable — changing them is a
// breaking change for downstream consumers.
func emitCacheSearchJSON(results []effects.BrainSearchResult, scope string, elapsed time.Duration) {
	type resultJSON struct {
		Tier         string  `json:"tier"`
		Namespace    string  `json:"namespace"`
		Key          string  `json:"key"`
		Score        float64 `json:"score"`
		Content      string  `json:"content"`
		UpdatedAt    int64   `json:"updated_at_ms"`
		EmbeddingDim int     `json:"embedding_dim"`
		Source       string  `json:"source"`
	}
	type envelope struct {
		Scope   string       `json:"scope"`
		Count   int          `json:"count"`
		QueryMs int64        `json:"query_ms"`
		Results []resultJSON `json:"results"`
	}
	out := envelope{Scope: scope, Count: len(results), QueryMs: elapsed.Milliseconds()}
	out.Results = make([]resultJSON, len(results))
	for i, r := range results {
		out.Results[i] = resultJSON{
			Tier:         r.Tier,
			Namespace:    r.Frame.Namespace,
			Key:          r.Frame.Key,
			Score:        r.Score,
			Content:      r.Frame.Content,
			UpdatedAt:    r.Frame.UpdatedAt,
			EmbeddingDim: r.Frame.EmbeddingDim,
			Source:       r.Frame.Source,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
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
	embed := fs.Bool("embed", false, "Compute and store embedding vector alongside SimHash")
	enrichContent := fs.String("enrich", "", "Extra content to store alongside commit metadata (e.g., design doc text)")
	fs.Parse(args)

	if *commitMsg == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang cache put-resolution --commit-msg \"msg\" [--diff-summary \"...\"] [--files \"f1,f2\"] [--embed] [--enrich \"content\"]")
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

	// Build content from commit info
	content := *commitMsg
	if *diffSummary != "" {
		content += "\n\nFiles changed:\n" + *diffSummary
	}
	if *files != "" {
		content += "\n\nFiles: " + *files
	}
	if *enrichContent != "" {
		content += "\n\n---\n" + *enrichContent
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

	status := fmt.Sprintf("Resolution stored: %s", key)
	if frame.EmbeddingDim > 0 {
		status += fmt.Sprintf(" (embedding: %d-dim %s)", frame.EmbeddingDim, frame.EmbedModel)
	}
	fmt.Println(status)
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
			record := effects.ExportFrameRecord(f, tier.name)
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
		effects.ImportFrameEmbedding(record, &frame)

		store.Put(frame, parseScope(tier))
		count++
	}

	fmt.Printf("Imported %d frame(s)\n", count)
}

func runCachePutVector(args []string) {
	fs := flag.NewFlagSet("cache put-vector", flag.ExitOnError)
	ns := fs.String("ns", "vectors", "Namespace")
	scope := fs.String("scope", "project", "Brain tier")
	model := fs.String("model", "manual", "Embedding model name")
	fs.Parse(args)

	// Read JSON from stdin: {"key": "k", "embedding": [0.1, 0.2, ...], "payload": {...}}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var input struct {
		Key       string          `json:"key"`
		Embedding []float32       `json:"embedding"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if input.Key == "" || len(input.Embedding) == 0 {
		fmt.Fprintln(os.Stderr, "JSON must have 'key' and 'embedding' fields")
		fmt.Fprintln(os.Stderr, `Example: echo '{"key":"v1","embedding":[0.1,0.2],"payload":{"type":"task"}}' | ailang cache put-vector`)
		os.Exit(1)
	}

	store, err := openBrainStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening brain: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	payload := input.Payload
	if payload == nil {
		payload = []byte("{}")
	}

	cache := store.Project
	if parseScope(*scope) == effects.ScopeUser {
		cache = store.User
	}
	if cache == nil {
		fmt.Fprintln(os.Stderr, "Error: specified brain tier is not available")
		os.Exit(1)
	}

	if err := cache.PutVector(input.Key, *ns, input.Embedding, *model, payload); err != nil {
		fmt.Fprintf(os.Stderr, "Error storing vector: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Stored vector: %s [%s] (dim=%d, model=%s)\n", input.Key, *ns, len(input.Embedding), *model)
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
