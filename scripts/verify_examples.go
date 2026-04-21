//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/scripts/internal/reporttypes"
)

func main() {
	// Parse flags
	allExamples := false
	threshold := 0.0
	format := "plain"

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json":
			format = "json"
		case "--markdown":
			format = "markdown"
		case "--all":
			allExamples = true
		case "--trace":
			useTrace = true
		case "--update-baselines":
			useTrace = true
			updateBaselines = true
		case "--threshold":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%f", &threshold)
				i++
			}
		case "--parallel", "-p":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &parallelism)
				i++
			} else {
				parallelism = runtime.NumCPU()
			}
		case "--sequential":
			parallelism = 1
		}
	}

	// Set global flag for findAllExamples
	useAllExamples = allExamples

	if updateBaselines {
		runUpdateBaselines()
		return
	}

	switch format {
	case "json":
		verifyExamplesJSON(threshold)
	case "markdown":
		verifyExamplesMarkdown()
	default:
		verifyExamplesPlain(threshold)
	}
}

var (
	useAllExamples  = false
	useTrace        = false
	updateBaselines = false
	parallelism     = 8 // max concurrent example runs
)

// skippedExamples lists files that are intentionally excluded from
// verify-examples because they exercise behavior that cannot succeed
// under a generic runner (e.g. deliberate non-zero process exit).
// Paths are relative to the repo root.
var skippedExamples = map[string]string{
	"examples/runnable/exit_code.ail": "intentionally exits with non-zero code (42) to demo std/io exit()",
}

// ailangBinary returns the path to a pre-built ailang binary.
// Falls back to "go run ./cmd/ailang" if no binary found.
func ailangBinary() (cmd string, args []string, usesGoRun bool) {
	// Check for bin/ailang (built by make build)
	if _, err := os.Stat("bin/ailang"); err == nil {
		return "bin/ailang", []string{"run"}, false
	}
	// Check for ./ailang
	if _, err := os.Stat("ailang"); err == nil {
		return "./ailang", []string{"run"}, false
	}
	// Fallback to go run
	return "go", []string{"run", "./cmd/ailang", "run"}, true
}

const tracesDir = "examples/traces"

func runExample(filename string) reporttypes.ExampleResult {
	start := time.Now()
	result := reporttypes.ExampleResult{
		File: filename,
	}

	// Skip non-.ail files
	if !strings.HasSuffix(filename, ".ail") {
		result.Status = "skipped"
		result.Duration = time.Since(start)
		return result
	}

	// Skip files on the explicit exclusion list (e.g. intentional non-zero exits).
	if reason, ok := skippedExamples[filepath.ToSlash(filename)]; ok {
		result.Status = "skipped"
		result.Error = reason
		result.Duration = time.Since(start)
		return result
	}

	// Read file to detect capabilities and entrypoint
	content, err := os.ReadFile(filename)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	fileContent := string(content)

	// Detect required capabilities
	// Effect rows can list multiple effects: ! {IO, FS, Net}
	// We check for the effect name anywhere in an effect row (after "! {") plus import patterns
	caps := []string{}
	needsAIStub := false

	type capSpec struct {
		name    string
		imports []string // import paths that imply this cap
		isAI    bool     // special handling for AI stub
	}
	capSpecs := []capSpec{
		{name: "IO", imports: []string{"import std/io"}},
		{name: "FS", imports: []string{"import std/fs", "import std/zip"}},
		{name: "Net", imports: []string{"import std/net"}},
		{name: "Clock", imports: []string{"import std/clock"}},
		{name: "Rand", imports: []string{"import std/rand"}},
		{name: "Env", imports: []string{"import std/env"}},
		{name: "AI", imports: []string{"import std/ai"}, isAI: true},
		{name: "Debug", imports: []string{"import std/debug"}},
		{name: "Process", imports: []string{"import std/process"}},
		{name: "Stream", imports: []string{"import std/stream"}},
	}

	for _, spec := range capSpecs {
		found := false
		// Check effect rows: ! {IO, FS} — the cap name appears after "! {"
		if strings.Contains(fileContent, "! {"+spec.name) || strings.Contains(fileContent, ", "+spec.name) {
			found = true
		}
		// Check builtin references
		if strings.Contains(fileContent, "_"+strings.ToLower(spec.name)+"_") {
			found = true
		}
		// Check import patterns
		for _, imp := range spec.imports {
			if strings.Contains(fileContent, imp) {
				found = true
				break
			}
		}
		if found {
			caps = append(caps, spec.name)
			if spec.isAI {
				needsAIStub = true
			}
		}
	}

	// Detect entrypoint (look for export func NAME)
	// Prefer zero-arg functions as entrypoints
	entrypoint := "main"
	if strings.Contains(fileContent, "export func") && !strings.Contains(fileContent, "export func main") {
		// Try to find a zero-arg exported function first
		lines := strings.Split(fileContent, "\n")
		candidates := []string{}

		for _, line := range lines {
			if strings.Contains(line, "export func") {
				// Extract function name and check if zero-arg
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "func" && i+1 < len(parts) {
						nameWithParen := parts[i+1]
						if idx := strings.Index(nameWithParen, "("); idx > 0 {
							name := nameWithParen[:idx]
							// Check if it's zero-arg: "func name() ->"
							if strings.HasPrefix(nameWithParen[idx:], "()") {
								// Zero-arg function - use immediately
								entrypoint = name
								goto done
							}
							// Non-zero-arg - save as fallback
							candidates = append(candidates, name)
						}
						break
					}
				}
			}
		}

		// If no zero-arg function found, use first exported function
		if entrypoint == "main" && len(candidates) > 0 {
			entrypoint = candidates[0]
		}
	}
done:

	// Build command using pre-built binary when available (avoids go run overhead)
	binCmd, baseArgs, _ := ailangBinary()
	args := append([]string{}, baseArgs...)
	if len(caps) > 0 {
		args = append(args, "--caps", strings.Join(caps, ","))
	}
	if needsAIStub {
		args = append(args, "--ai-stub")
	}
	if entrypoint != "main" {
		args = append(args, "--entry", entrypoint)
	}
	// When tracing, add --emit-trace jsonl and --quiet
	// Status output goes to stderr, trace JSONL goes to stdout
	if useTrace {
		args = append(args, "--emit-trace", "jsonl", "--quiet")
	}
	args = append(args, filename)

	cmd := exec.Command(binCmd, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result.Duration = time.Since(start)
	result.Output = stdout.String()

	if err != nil {
		result.Status = "failed"
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
	} else {
		// Success - check for actual errors in stderr (not just DEBUG output)
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "Error:") || strings.Contains(stderrStr, "error:") {
			result.Status = "failed"
			result.Error = stderrStr
		} else {
			result.Status = "passed"
		}
	}

	// Trace verification (only for passing examples)
	if useTrace && result.Status == "passed" {
		traceVerify(&result, filename, stdout.String())
	}

	return result
}

// extractJSONL filters raw output to only JSONL lines (starting with '{').
// Program output (print/println) is interleaved with trace events on stdout.
func extractJSONL(raw string) string {
	var jsonLines []string
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			jsonLines = append(jsonLines, line)
		}
	}
	return strings.Join(jsonLines, "\n")
}

// traceVerify captures trace data and compares against baseline if one exists.
func traceVerify(result *reporttypes.ExampleResult, filename, traceOutput string) {
	// Extract only JSONL lines (program print output is interleaved)
	jsonlOnly := extractJSONL(traceOutput)

	// Count trace events
	lines := strings.Split(strings.TrimSpace(jsonlOnly), "\n")
	eventCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "{") {
			eventCount++
		}
	}
	result.TraceEvents = eventCount

	if eventCount == 0 {
		result.TraceStatus = "error"
		return
	}

	// Score the trace using ailang export-training --score --json
	score := scoreTrace(jsonlOnly)
	result.TraceScore = score

	// Check for baseline
	baselinePath := traceBaselinePath(filename)
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		result.TraceStatus = "new"
		return
	}

	// Compare: read baseline and compare event types/function names (ignore timestamps)
	baselineData, err := os.ReadFile(baselinePath)
	if err != nil {
		result.TraceStatus = "error"
		return
	}

	if tracesMatch(string(baselineData), jsonlOnly) {
		result.TraceStatus = "match"
	} else {
		result.TraceStatus = "mismatch"
	}
}

// tracesMatch compares two traces ignoring timestamps and durations.
// Returns true if the event sequence (types, function names, args) matches.
func tracesMatch(baseline, current string) bool {
	baselineEvents := parseTraceEvents(baseline)
	currentEvents := parseTraceEvents(current)

	if len(baselineEvents) != len(currentEvents) {
		return false
	}

	for i := range baselineEvents {
		if baselineEvents[i] != currentEvents[i] {
			return false
		}
	}
	return true
}

// parseTraceEvents extracts a normalized sequence of event signatures for comparison.
// Each signature includes event type + key fields (ignoring timestamps/durations).
func parseTraceEvents(traceData string) []string {
	var signatures []string
	for _, line := range strings.Split(strings.TrimSpace(traceData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		eventType, _ := evt["event"].(string)
		sig := eventType

		// Add function name, args, and result for function events.
		// Record field ordering is now deterministic (sorted by key in Value.String()).
		if fn, ok := evt["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				sig += ":" + name
			}
			if args, ok := fn["args"].([]interface{}); ok {
				for _, a := range args {
					sig += ":" + fmt.Sprint(a)
				}
			}
			if result, ok := fn["result"].(string); ok {
				sig += "=" + result
			}
		}

		// Add module name for module events
		if mod, ok := evt["module"].(map[string]interface{}); ok {
			if name, ok := mod["name"].(string); ok {
				sig += ":" + name
			}
		}

		// Add effect info for effect events
		if eff, ok := evt["effect"].(map[string]interface{}); ok {
			if name, ok := eff["effect_name"].(string); ok {
				sig += ":" + name
			}
			if op, ok := eff["op_name"].(string); ok {
				sig += "." + op
			}
		}

		signatures = append(signatures, sig)
	}
	return signatures
}

// scoreTrace runs the scoring algorithm on trace data.
// Returns the quality score (0.0-1.0).
func scoreTrace(traceData string) float64 {
	// Write trace to temp file and score it
	tmpFile, err := os.CreateTemp("", "trace-score-*.jsonl")
	if err != nil {
		return 0
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(traceData); err != nil {
		tmpFile.Close()
		return 0
	}
	tmpFile.Close()

	// Run ailang export-training --score --json <file>
	cmd := exec.Command("go", "run", "./cmd/ailang", "export-training", "--score", "--json", tmpFile.Name())
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return 0
	}

	// Parse JSON output to extract score
	var scores []struct {
		Score struct {
			Total float64 `json:"total"`
		} `json:"score"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &scores); err != nil {
		return 0
	}
	if len(scores) > 0 {
		return scores[0].Score.Total
	}
	return 0
}

// traceBaselinePath returns the path to the baseline trace for a given example.
func traceBaselinePath(examplePath string) string {
	// examples/runnable/hello.ail -> examples/traces/hello.jsonl
	base := filepath.Base(examplePath)
	name := strings.TrimSuffix(base, ".ail")
	return filepath.Join(tracesDir, name+".jsonl")
}

// runUpdateBaselines generates fresh baseline traces for all passing examples.
func runUpdateBaselines() {
	files, err := findAllExamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding examples: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	// Ensure traces directory exists
	if err := os.MkdirAll(tracesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating traces directory: %v\n", err)
		os.Exit(1)
	}

	updated := 0
	skippedCount := 0
	failedCount := 0

	fmt.Println("Updating trace baselines")
	fmt.Println("========================")

	for _, file := range files {
		displayName := strings.TrimPrefix(file, "examples/")
		fmt.Printf("  %s... ", displayName)

		result := runExample(file)

		if result.Status != "passed" {
			fmt.Printf("SKIP (%s)\n", result.Status)
			if result.Status == "failed" {
				failedCount++
			} else {
				skippedCount++
			}
			continue
		}

		if result.TraceEvents == 0 {
			fmt.Printf("SKIP (no trace events)\n")
			skippedCount++
			continue
		}

		// Write filtered JSONL to baseline file (strip program print output)
		baselinePath := traceBaselinePath(file)
		jsonlData := extractJSONL(result.Output)
		if err := os.WriteFile(baselinePath, []byte(jsonlData+"\n"), 0644); err != nil {
			fmt.Printf("ERROR: %v\n", err)
			failedCount++
			continue
		}

		fmt.Printf("OK (%d events, score %.2f)\n", result.TraceEvents, result.TraceScore)
		updated++
	}

	fmt.Printf("\nUpdated %d baselines (%d skipped, %d failed)\n", updated, skippedCount, failedCount)
}

// computeTraceSummary aggregates trace results across all examples.
func computeTraceSummary(results []reporttypes.ExampleResult) *reporttypes.TraceSummary {
	summary := &reporttypes.TraceSummary{}
	totalScore := 0.0

	for _, r := range results {
		if r.TraceStatus == "" {
			continue
		}
		summary.Traced++
		totalScore += r.TraceScore

		switch r.TraceStatus {
		case "match":
			summary.Matches++
		case "mismatch":
			summary.Mismatches++
		case "new":
			summary.NewTraces++
		case "error":
			summary.Errors++
		}
	}

	if summary.Traced > 0 {
		summary.AvgScore = totalScore / float64(summary.Traced)
	}
	return summary
}

// findAllExamples finds all .ail files in examples/ directory
// By default, only checks examples/runnable/ (CI mode)
// With --all flag, checks all example directories
func findAllExamples() ([]string, error) {
	if useAllExamples {
		return findAllExamplesLegacy()
	}

	var files []string
	runnableDir := filepath.Join("examples", "runnable")

	// Check if runnable directory exists
	if _, err := os.Stat(runnableDir); os.IsNotExist(err) {
		// Fallback to old behavior if runnable/ doesn't exist yet
		return findAllExamplesLegacy()
	}

	err := filepath.Walk(runnableDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// findAllExamplesLegacy recursively finds all .ail files in the examples directory
// Used as fallback for backward compatibility
func findAllExamplesLegacy() ([]string, error) {
	var files []string
	err := filepath.Walk("examples", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// runExamplesParallel runs examples concurrently with a semaphore.
// Results are returned in the same order as files.
func runExamplesParallel(files []string) []reporttypes.ExampleResult {
	results := make([]reporttypes.ExampleResult, len(files))
	if parallelism <= 1 {
		// Sequential fallback
		for i, file := range files {
			results[i] = runExample(file)
		}
		return results
	}

	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	for i, file := range files {
		wg.Add(1)
		go func(idx int, f string) {
			defer wg.Done()
			sem <- struct{}{}
			results[idx] = runExample(f)
			<-sem
		}(i, file)
	}
	wg.Wait()
	return results
}

func verifyExamplesPlain(threshold float64) {
	files, err := findAllExamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding examples: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	binCmd, _, _ := ailangBinary()
	fmt.Printf("Verifying AILANG Examples (%d files, parallelism=%d, binary=%s)\n", len(files), parallelism, binCmd)
	fmt.Println("=========================")

	start := time.Now()
	allResults := runExamplesParallel(files)
	elapsed := time.Since(start)

	passed := 0
	failed := 0
	skipped := 0

	for i, result := range allResults {
		displayName := strings.TrimPrefix(files[i], "examples/")
		switch result.Status {
		case "passed":
			suffix := ""
			if useTrace && result.TraceStatus != "" {
				suffix = fmt.Sprintf(" [trace:%s]", result.TraceStatus)
			}
			fmt.Printf("✓ %s (%.2fs)%s\n", displayName, result.Duration.Seconds(), suffix)
			passed++
		case "failed":
			fmt.Printf("✗ %s (%.2fs)\n", displayName, result.Duration.Seconds())
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", strings.TrimSpace(result.Error))
			}
			failed++
		case "skipped":
			fmt.Printf("- %s SKIP\n", displayName)
			skipped++
		}
	}

	total := passed + failed + skipped
	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100.0
	}

	fmt.Printf("\nSummary (%.1fs wall time):\n", elapsed.Seconds())
	fmt.Printf("  Total: %d\n", total)
	fmt.Printf("  Passed: %d\n", passed)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Skipped: %d\n", skipped)

	if useTrace {
		summary := computeTraceSummary(allResults)
		fmt.Printf("\nTrace Determinism:\n")
		fmt.Printf("  Traced: %d\n", summary.Traced)
		fmt.Printf("  Matches: %d\n", summary.Matches)
		fmt.Printf("  Mismatches: %d\n", summary.Mismatches)
		fmt.Printf("  New (no baseline): %d\n", summary.NewTraces)
		fmt.Printf("  Errors: %d\n", summary.Errors)
		fmt.Printf("  Avg Score: %.2f\n", summary.AvgScore)
	}

	// One-line summary (useful for CI)
	fmt.Printf("\nExamples: %d/%d passed (%.1f%%)\n", passed, total, passRate)

	// Threshold check
	if threshold > 0 && passRate < threshold {
		fmt.Fprintf(os.Stderr, "\n❌ FAIL: Pass rate %.1f%% is below threshold %.1f%%\n", passRate, threshold)
		os.Exit(1)
	} else if threshold > 0 {
		fmt.Printf("✅ PASS: Pass rate %.1f%% meets threshold %.1f%%\n", passRate, threshold)
	}

	if failed > 0 {
		os.Exit(1)
	}
}

func verifyExamplesJSON(threshold float64) {
	files, err := findAllExamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding examples: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	report := reporttypes.VerificationReport{
		Timestamp: time.Now(),
		Results:   []reporttypes.ExampleResult{},
	}

	allResults := runExamplesParallel(files)
	for i, result := range allResults {
		result.File = strings.TrimPrefix(files[i], "examples/")
		report.Results = append(report.Results, result)

		switch result.Status {
		case "passed":
			report.Passed++
		case "failed":
			report.Failed++
		case "skipped":
			report.Skipped++
		}
	}

	report.TotalExamples = len(report.Results)

	if useTrace {
		report.TraceSummary = computeTraceSummary(report.Results)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Threshold check
	passRate := 0.0
	if report.TotalExamples > 0 {
		passRate = float64(report.Passed) / float64(report.TotalExamples) * 100.0
	}

	if threshold > 0 && passRate < threshold {
		fmt.Fprintf(os.Stderr, "Pass rate %.1f%% is below threshold %.1f%%\n", passRate, threshold)
		os.Exit(1)
	}

	if report.Failed > 0 {
		os.Exit(1)
	}
}

func verifyExamplesMarkdown() {
	files, err := findAllExamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding examples: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	var passed, failed, skipped []string
	allResults := runExamplesParallel(files)

	for i, result := range allResults {
		displayName := strings.TrimPrefix(files[i], "examples/")

		switch result.Status {
		case "passed":
			passed = append(passed, displayName)
		case "failed":
			failed = append(failed, displayName)
		case "skipped":
			skipped = append(skipped, displayName)
		}
	}

	fmt.Println("## Example Status")
	fmt.Println()
	fmt.Println("### Working Examples ✅")
	if len(passed) > 0 {
		for _, f := range passed {
			fmt.Printf("- `%s`\n", f)
		}
	} else {
		fmt.Println("*None*")
	}

	fmt.Println()
	fmt.Println("### Failing Examples ❌")
	if len(failed) > 0 {
		for _, f := range failed {
			fmt.Printf("- `%s`\n", f)
		}
	} else {
		fmt.Println("*None*")
	}

	fmt.Println()
	fmt.Println("### Skipped Examples ⏭️")
	if len(skipped) > 0 {
		for _, f := range skipped {
			fmt.Printf("- `%s`\n", f)
		}
	} else {
		fmt.Println("*None*")
	}

	if useTrace {
		summary := computeTraceSummary(allResults)
		fmt.Println()
		fmt.Println("### Trace Determinism")
		fmt.Printf("- Traced: %d\n", summary.Traced)
		fmt.Printf("- Matches: %d\n", summary.Matches)
		fmt.Printf("- Mismatches: %d\n", summary.Mismatches)
		fmt.Printf("- New (no baseline): %d\n", summary.NewTraces)
		fmt.Printf("- Avg Score: %.2f\n", summary.AvgScore)
	}

	fmt.Println()
	fmt.Printf("**Summary:** %d passed, %d failed, %d skipped (Total: %d)\n",
		len(passed), len(failed), len(skipped), len(passed)+len(failed)+len(skipped))

	if len(failed) > 0 {
		os.Exit(1)
	}
}
