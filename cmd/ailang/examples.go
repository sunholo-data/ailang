package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ExampleManifest represents the examples/manifest.json structure
type ExampleManifest struct {
	Schema        string           `json:"schema"`
	SchemaVersion string           `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Generator     string           `json:"generator"`
	Examples      []ExampleEntry   `json:"examples"`
}

// ExampleEntry represents a single example in the manifest
type ExampleEntry struct {
	Path        string            `json:"path"`
	Status      string            `json:"status"`
	Tags        []string          `json:"tags"`
	Description string            `json:"description"`
	Expected    *ExpectedOutput   `json:"expected,omitempty"`
}

// ExpectedOutput defines the expected output for an example
type ExpectedOutput struct {
	Stdout   string `json:"stdout,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// examplesCommand implements `ailang examples` subcommand
func examplesCommand(args []string) {
	if len(args) == 0 {
		printExamplesHelp()
		return
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		examplesListCommand(subargs)
	case "search":
		examplesSearchCommand(subargs)
	case "show":
		examplesShowCommand(subargs)
	case "tags":
		examplesTagsCommand(subargs)
	case "help", "--help", "-h":
		printExamplesHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown examples subcommand: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, "Run 'ailang examples help' for usage.")
		os.Exit(1)
	}
}

// examplesListCommand lists all examples with optional tag filtering
func examplesListCommand(args []string) {
	listFlags := flag.NewFlagSet("examples list", flag.ExitOnError)
	tagsFlag := listFlags.String("tags", "", "Filter by tags (comma-separated)")
	statusFlag := listFlags.String("status", "working", "Filter by status: working, broken, all")
	jsonFlag := listFlags.Bool("json", false, "Output as JSON")

	if err := listFlags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	manifest, err := loadExamplesManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
		os.Exit(1)
	}

	// Filter examples
	var filtered []ExampleEntry
	filterTags := []string{}
	if *tagsFlag != "" {
		filterTags = strings.Split(*tagsFlag, ",")
		for i := range filterTags {
			filterTags[i] = strings.TrimSpace(filterTags[i])
		}
	}

	for _, ex := range manifest.Examples {
		// Status filter
		if *statusFlag != "all" && ex.Status != *statusFlag {
			continue
		}
		// Tags filter
		if len(filterTags) > 0 && !hasAnyTag(ex.Tags, filterTags) {
			continue
		}
		filtered = append(filtered, ex)
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(filtered)
		return
	}

	// Print human-readable list
	fmt.Printf("📚 AILANG Examples (%d/%d)\n\n", len(filtered), len(manifest.Examples))
	for _, ex := range filtered {
		statusIcon := "✅"
		if ex.Status != "working" {
			statusIcon = "⚠️"
		}
		fmt.Printf("%s %s\n", statusIcon, ex.Path)
		if ex.Description != "" {
			fmt.Printf("   %s\n", ex.Description)
		}
		if len(ex.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(ex.Tags, ", "))
		}
		fmt.Println()
	}
}

// examplesSearchCommand searches examples by content or description
func examplesSearchCommand(args []string) {
	searchFlags := flag.NewFlagSet("examples search", flag.ExitOnError)
	limitFlag := searchFlags.Int("limit", 10, "Maximum results to return")
	jsonFlag := searchFlags.Bool("json", false, "Output as JSON")

	if err := searchFlags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if searchFlags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: search query required")
		fmt.Fprintln(os.Stderr, "Usage: ailang examples search \"<query>\"")
		os.Exit(1)
	}

	query := strings.ToLower(searchFlags.Arg(0))

	// Find examples directory
	examplesPath, err := findExamplesDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load manifest for metadata
	manifest, _ := loadExamplesManifest()
	manifestMap := make(map[string]ExampleEntry)
	for _, ex := range manifest.Examples {
		manifestMap[ex.Path] = ex
	}

	// Search both manifest (descriptions/tags) and file content
	type searchResult struct {
		path        string
		score       float64
		matchSource string // "description", "tags", "content"
	}

	var results []searchResult
	queryWords := strings.Fields(query)

	// Walk examples/runnable/ for .ail files
	runnablePath := filepath.Join(examplesPath, "runnable")
	filepath.Walk(runnablePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".ail") {
			return nil
		}

		filename := filepath.Base(path)
		meta, hasMeta := manifestMap[filename]

		var score float64
		var matchSource string

		// Check tags (highest priority)
		if hasMeta {
			for _, tag := range meta.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					score = 1.0
					matchSource = "tag"
					break
				}
				for _, qw := range queryWords {
					if strings.Contains(strings.ToLower(tag), qw) {
						score = maxFloat(score, 0.9)
						matchSource = "tag"
					}
				}
			}
		}

		// Check description
		if hasMeta && meta.Description != "" {
			descLower := strings.ToLower(meta.Description)
			if strings.Contains(descLower, query) {
				score = maxFloat(score, 0.95)
				matchSource = "description"
			} else {
				for _, qw := range queryWords {
					if strings.Contains(descLower, qw) {
						score = maxFloat(score, 0.7)
						if matchSource == "" {
							matchSource = "description"
						}
					}
				}
			}
		}

		// Check file content
		content, err := os.ReadFile(path)
		if err == nil {
			contentLower := strings.ToLower(string(content))
			if strings.Contains(contentLower, query) {
				score = maxFloat(score, 0.8)
				if matchSource == "" {
					matchSource = "content"
				}
			} else {
				matchCount := 0
				for _, qw := range queryWords {
					if strings.Contains(contentLower, qw) {
						matchCount++
					}
				}
				if matchCount > 0 {
					s := float64(matchCount) / float64(len(queryWords)) * 0.6
					score = maxFloat(score, s)
					if matchSource == "" {
						matchSource = "content"
					}
				}
			}
		}

		if score > 0 {
			results = append(results, searchResult{
				path:        filename,
				score:       score,
				matchSource: matchSource,
			})
		}

		return nil
	})

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Limit results
	if len(results) > *limitFlag {
		results = results[:*limitFlag]
	}

	if *jsonFlag {
		// Convert to JSON-friendly format
		type jsonResult struct {
			Path        string   `json:"path"`
			Score       float64  `json:"score"`
			MatchSource string   `json:"match_source"`
			Description string   `json:"description,omitempty"`
			Tags        []string `json:"tags,omitempty"`
		}
		var jsonResults []jsonResult
		for _, r := range results {
			jr := jsonResult{
				Path:        r.path,
				Score:       r.score,
				MatchSource: r.matchSource,
			}
			if meta, ok := manifestMap[r.path]; ok {
				jr.Description = meta.Description
				jr.Tags = meta.Tags
			}
			jsonResults = append(jsonResults, jr)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(jsonResults)
		return
	}

	// Print results
	fmt.Printf("🔍 Search: %q\n", searchFlags.Arg(0))
	fmt.Printf("   Found %d examples\n\n", len(results))

	for i, r := range results {
		meta, hasMeta := manifestMap[r.path]

		fmt.Printf("%d. %s (%.2f, %s)\n", i+1, r.path, r.score, r.matchSource)
		if hasMeta && meta.Description != "" {
			fmt.Printf("   %s\n", meta.Description)
		}
		if hasMeta && len(meta.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(meta.Tags, ", "))
		}
		fmt.Println()
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// examplesShowCommand shows the content of a specific example
func examplesShowCommand(args []string) {
	showFlags := flag.NewFlagSet("examples show", flag.ExitOnError)
	runFlag := showFlags.Bool("run", false, "Run the example after showing")
	expectedFlag := showFlags.Bool("expected", false, "Show expected output only")

	if err := showFlags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if showFlags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: example name required")
		fmt.Fprintln(os.Stderr, "Usage: ailang examples show <name>")
		os.Exit(1)
	}

	name := showFlags.Arg(0)

	// Find the example file
	examplesPath, err := findExamplesDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add .ail extension if missing
	if !strings.HasSuffix(name, ".ail") {
		name = name + ".ail"
	}

	// Check runnable/ first, then examples/
	examplePath := filepath.Join(examplesPath, "runnable", name)
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		examplePath = filepath.Join(examplesPath, name)
	}

	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: example not found: %s\n", name)
		fmt.Fprintln(os.Stderr, "Use 'ailang examples list' to see available examples.")
		os.Exit(1)
	}

	// Load manifest for metadata
	manifest, _ := loadExamplesManifest()
	var meta *ExampleEntry
	for _, ex := range manifest.Examples {
		if ex.Path == name {
			meta = &ex
			break
		}
	}

	if *expectedFlag && meta != nil && meta.Expected != nil {
		fmt.Print(meta.Expected.Stdout)
		return
	}

	// Read and display the file
	content, err := os.ReadFile(examplePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading example: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📄 %s\n", name)
	if meta != nil {
		if meta.Description != "" {
			fmt.Printf("   %s\n", meta.Description)
		}
		if len(meta.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(meta.Tags, ", "))
		}
	}
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(string(content))
	fmt.Println(strings.Repeat("─", 60))

	if meta != nil && meta.Expected != nil {
		fmt.Println("\n📤 Expected output:")
		fmt.Println(meta.Expected.Stdout)
	}

	// Run if requested
	if *runFlag {
		fmt.Println("\n🚀 Running example...")
		fmt.Println(strings.Repeat("─", 60))

		// Determine caps from content (simple heuristic)
		caps := detectCaps(string(content))

		cmd := exec.Command("ailang", "run", "--entry", "main", "--caps", caps, examplePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

// examplesTagsCommand lists all available tags
func examplesTagsCommand(args []string) {
	manifest, err := loadExamplesManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
		os.Exit(1)
	}

	// Count tags
	tagCounts := make(map[string]int)
	for _, ex := range manifest.Examples {
		if ex.Status != "working" {
			continue
		}
		for _, tag := range ex.Tags {
			tagCounts[tag]++
		}
	}

	// Sort by count
	type tagCount struct {
		tag   string
		count int
	}
	var tags []tagCount
	for t, c := range tagCounts {
		tags = append(tags, tagCount{t, c})
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].count > tags[j].count
	})

	fmt.Println("🏷️  Available tags:")
	fmt.Println()
	for _, tc := range tags {
		fmt.Printf("  %-25s (%d examples)\n", tc.tag, tc.count)
	}
	fmt.Println()
	fmt.Println("Use: ailang examples list --tags <tag>")
}

// Helper functions

func loadExamplesManifest() (*ExampleManifest, error) {
	examplesPath, err := findExamplesDir()
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(examplesPath, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest ExampleManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

func findExamplesDir() (string, error) {
	candidates := []string{
		"examples",
		"../examples",
		"../../examples",
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("examples directory not found")
}

func hasAnyTag(exampleTags, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, et := range exampleTags {
			if strings.EqualFold(et, ft) {
				return true
			}
		}
	}
	return false
}

func detectCaps(content string) string {
	caps := []string{}

	if strings.Contains(content, "std/io") || strings.Contains(content, "println") || strings.Contains(content, "print(") {
		caps = append(caps, "IO")
	}
	if strings.Contains(content, "std/fs") || strings.Contains(content, "readFile") || strings.Contains(content, "writeFile") {
		caps = append(caps, "FS")
	}
	if strings.Contains(content, "std/net") || strings.Contains(content, "httpGet") || strings.Contains(content, "httpPost") {
		caps = append(caps, "Net")
	}
	if strings.Contains(content, "std/ai") || strings.Contains(content, "_ai_call") {
		caps = append(caps, "AI")
	}
	if strings.Contains(content, "Clock") {
		caps = append(caps, "Clock")
	}

	if len(caps) == 0 {
		caps = append(caps, "IO") // Default
	}

	return strings.Join(caps, ",")
}

func printExamplesHelp() {
	fmt.Println("Usage: ailang examples <command> [options]")
	fmt.Println()
	fmt.Println("Search and explore working AILANG code examples.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list           List all examples (with optional tag/status filter)")
	fmt.Println("  search         Search examples by content or description")
	fmt.Println("  show           Display a specific example with metadata")
	fmt.Println("  tags           List all available tags")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang examples list                    # List all working examples")
	fmt.Println("  ailang examples list --tags recursion   # Filter by tag")
	fmt.Println("  ailang examples list --status all       # Include broken examples")
	fmt.Println()
	fmt.Println("  ailang examples search \"pattern matching\"")
	fmt.Println("  ailang examples search \"filter a list\"")
	fmt.Println("  ailang examples search \"json\" --limit 5")
	fmt.Println()
	fmt.Println("  ailang examples show adt_option         # Show example code")
	fmt.Println("  ailang examples show adt_option --run   # Show and execute")
	fmt.Println("  ailang examples show fold_reduce --expected  # Just show expected output")
	fmt.Println()
	fmt.Println("  ailang examples tags                    # List all tags")
	fmt.Println()
	fmt.Println("Use --json flag for machine-readable output.")
}
