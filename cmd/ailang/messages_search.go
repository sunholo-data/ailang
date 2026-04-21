package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Semantic search and deduplication commands for messages

// runMessagesSearch handles the 'messages search' subcommand
func runMessagesSearch(args []string) {
	fs := flag.NewFlagSet("messages search", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	threshold := fs.Float64("threshold", 0.40, "Minimum similarity (0.0-1.0)")
	limit := fs.Int("limit", 20, "Maximum results")
	maxScan := fs.Int("max-scan", 1000, "Maximum messages to scan")
	neural := fs.Bool("neural", false, "Use neural embeddings via Ollama (requires Ollama running)")
	simhash := fs.Bool("simhash", false, "Force SimHash mode (default)")
	space := fs.String("space", "", "Search a specific envelope space (intent, code, context, skill, resolution)")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: search query required\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang messages search \"query\" [flags]\n")
		os.Exit(1)
	}

	query := fs.Arg(0)

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Resolve inbox (use inference if not specified)
	resolvedInbox := *inbox
	if resolvedInbox == "" {
		resolvedInbox = inferInbox()
		if resolvedInbox != "" {
			fmt.Printf("Using inbox: %s (inferred from repo)\n", cyan(resolvedInbox))
		}
	}

	// Determine search mode
	useNeural := *neural && !*simhash
	scoreKind := "simhash"
	if *space != "" {
		// Envelope space search implies neural
		useNeural = true
		scoreKind = "envelope:" + *space
		fmt.Printf("Using envelope search (space: %s)\n", cyan(*space))
	} else if useNeural {
		scoreKind = "embedding"
		fmt.Printf("Using neural search via %s\n", cyan("Ollama"))
	}

	opts := messaging.SearchOptions{
		Query:         query,
		Threshold:     *threshold,
		Limit:         *limit,
		MaxScan:       *maxScan,
		Inbox:         resolvedInbox,
		UseNeural:     useNeural,
		EnvelopeSpace: *space,
	}

	hits, err := store.SemanticSearch(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(hits, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(hits) == 0 {
		fmt.Println("No messages found matching query.")
		printSearchFooter("SQLite", scoreKind, 0, *threshold)
		return
	}

	fmt.Printf("\n%s \"%s\":\n\n", bold("Search results for"), query)
	for i, hit := range hits {
		fmt.Printf("%d. [%.0f%% match] ", i+1, hit.Score*100)
		printInboxMessage(hit.Message, false)
	}

	printSearchFooter("SQLite", scoreKind, len(hits), *threshold)
}

// runMessagesDedupe handles the 'messages dedupe' subcommand
func runMessagesDedupe(args []string) {
	fs := flag.NewFlagSet("messages dedupe", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	threshold := fs.Float64("threshold", 0.95, "Minimum similarity for duplicates (0.0-1.0)")
	apply := fs.Bool("apply", false, "Actually mark duplicates (default: report only)")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Resolve inbox
	resolvedInbox := *inbox
	if resolvedInbox == "" {
		resolvedInbox = inferInbox()
		if resolvedInbox != "" {
			fmt.Printf("Using inbox: %s (inferred from repo)\n", cyan(resolvedInbox))
		}
	}

	groups, err := store.FindDuplicates(resolvedInbox, *threshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(groups, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(groups) == 0 {
		fmt.Println("No duplicate messages found.")
		return
	}

	// Print report
	totalDuplicates := 0
	for _, g := range groups {
		totalDuplicates += len(g.Duplicates)
	}

	fmt.Printf("\n%s\n\n", bold("Duplicate Report"))
	fmt.Printf("Found %s duplicate groups (%s messages)\n\n",
		yellow(fmt.Sprintf("%d", len(groups))),
		yellow(fmt.Sprintf("%d", totalDuplicates)))

	for i, group := range groups {
		fmt.Printf("Group %d (%.0f%% similar, %d duplicates):\n", i+1, group.MinScore*100, len(group.Duplicates))
		fmt.Printf("  %s Keep: %s\n", green("*"), group.Representative.MessageID)
		fmt.Printf("    %s\n", dim(group.Representative.Title))
		for _, dup := range group.Duplicates {
			fmt.Printf("  %s Archive: %s\n", red("-"), dup.MessageID)
			fmt.Printf("    %s\n", dim(dup.Title))
		}
		fmt.Println()
	}

	if !*apply {
		fmt.Printf("Run with %s to mark duplicates.\n", bold("--apply"))
		return
	}

	// Apply deduplication
	if err := store.ApplyDuplicates(groups, "cli_dedupe"); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Marked %d messages as duplicates.\n", green("Done!"), totalDuplicates)
}

// printSearchFooter prints the explainability footer for semantic queries
func printSearchFooter(backend, scoreKind string, results int, threshold float64) {
	fmt.Printf("\n%s backend=%s score=%s results=%d threshold=%.2f\n",
		dim("---"), backend, scoreKind, results, threshold)
}

// inferInbox attempts to infer the inbox name from the current environment
func inferInbox() string {
	// Try git repo name
	repoName := getGitRepoName()
	if repoName != "" {
		// Map known repo names to inboxes
		inboxMap := map[string]string{
			"ailang": "ailang_core",
		}
		if mapped, ok := inboxMap[repoName]; ok {
			return mapped
		}
		return repoName
	}
	return ""
}

// getGitRepoName returns the current git repository name (folder name of repo root)
func getGitRepoName() string {
	// Check if we're in a git repo
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up to find .git directory
	for dir := cwd; dir != filepath.Dir(dir) && dir != "."; {
		if _, err := os.Stat(dir + "/.git"); err == nil {
			// Found git root - return the directory name
			return filepath.Base(dir)
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
