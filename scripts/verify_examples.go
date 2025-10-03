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
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/scripts/internal/reporttypes"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		verifyExamplesJSON()
	} else if len(os.Args) > 1 && os.Args[1] == "--markdown" {
		verifyExamplesMarkdown()
	} else {
		verifyExamplesPlain()
	}
}

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

	// Skip known documentation files
	if strings.Contains(filename, "_demo") || strings.Contains(filename, "_test") ||
		strings.Contains(filename, "_trace") || strings.Contains(filename, "_session") {
		result.Status = "skipped"
		result.Duration = time.Since(start)
		return result
	}

	cmd := exec.Command("go", "run", "cmd/ailang/main.go", "run", filename)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

	return result
}

// findAllExamples recursively finds all .ail files in the examples directory
func findAllExamples() ([]string, error) {
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

func verifyExamplesPlain() {
	files, err := findAllExamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding examples: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	passed := 0
	failed := 0
	skipped := 0

	fmt.Println("Verifying AILANG Examples")
	fmt.Println("=========================")

	for _, file := range files {
		// Show relative path from examples/ for better clarity
		displayName := strings.TrimPrefix(file, "examples/")
		fmt.Printf("Testing %s... ", displayName)

		result := runExample(file)

		switch result.Status {
		case "passed":
			fmt.Printf("✓ PASS (%.2fs)\n", result.Duration.Seconds())
			passed++
		case "failed":
			fmt.Printf("✗ FAIL (%.2fs)\n", result.Duration.Seconds())
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", strings.TrimSpace(result.Error))
			}
			failed++
		case "skipped":
			fmt.Printf("- SKIP\n")
			skipped++
		}
	}

	fmt.Println("\nSummary:")
	fmt.Printf("  Total: %d\n", passed+failed+skipped)
	fmt.Printf("  Passed: %d\n", passed)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Skipped: %d\n", skipped)

	if failed > 0 {
		os.Exit(1)
	}
}

func verifyExamplesJSON() {
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

	for _, file := range files {
		result := runExample(file)
		// Use relative path from examples/ for cleaner output
		result.File = strings.TrimPrefix(file, "examples/")
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

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
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

	for _, file := range files {
		// Use relative path from examples/ for better clarity
		displayName := strings.TrimPrefix(file, "examples/")
		result := runExample(file)

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

	fmt.Println()
	fmt.Printf("**Summary:** %d passed, %d failed, %d skipped (Total: %d)\n",
		len(passed), len(failed), len(skipped), len(passed)+len(failed)+len(skipped))

	if len(failed) > 0 {
		os.Exit(1)
	}
}
