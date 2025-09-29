// validate_manifest.go validates examples against the manifest.
// It ensures documentation stays in sync with reality.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sunholo/ailang/internal/manifest"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

type ValidationResult struct {
	Path    string
	Status  manifest.Status
	Passed  bool
	Message string
	Output  string
}

func main() {
	var (
		manifestPath = flag.String("manifest", "examples/manifest.json", "Path to manifest file")
		examplesDir  = flag.String("dir", "examples", "Examples directory")
		updateFlag   = flag.Bool("update", false, "Update manifest with actual results")
		verboseFlag  = flag.Bool("verbose", false, "Verbose output")
		ciMode       = flag.Bool("ci", false, "CI mode (fail on any mismatch)")
	)
	flag.Parse()

	// Load manifest
	m, err := manifest.Load(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to load manifest: %v\n", red("Error:"), err)
		os.Exit(1)
	}

	fmt.Printf("%s AILANG Example Validator\n", bold("🔍"))
	fmt.Printf("Manifest: %s\n", *manifestPath)
	fmt.Printf("Examples: %d (Working: %d, Broken: %d, Experimental: %d)\n\n",
		m.Statistics.Total, m.Statistics.Working, m.Statistics.Broken, m.Statistics.Experimental)

	// Validate each example
	var results []ValidationResult
	failed := 0

	// Set deterministic environment
	os.Setenv("LC_ALL", "C.UTF-8")
	os.Setenv("TZ", "UTC")
	os.Setenv("AILANG_SEED", "0")

	for _, example := range m.Examples {
		result := validateExample(*examplesDir, example, *verboseFlag)
		results = append(results, result)

		if !result.Passed {
			failed++
			fmt.Printf("%s %s: %s\n", red("✗"), result.Path, result.Message)
			if *verboseFlag && result.Output != "" {
				fmt.Printf("  Output:\n%s\n", indent(result.Output, "    "))
			}
		} else {
			fmt.Printf("%s %s\n", green("✓"), result.Path)
		}

		// Check header in file matches manifest
		if err := validateHeader(*examplesDir, example); err != nil {
			fmt.Printf("  %s Header mismatch: %v\n", yellow("⚠"), err)
			if *ciMode {
				failed++
			}
		}
	}

	// Summary
	fmt.Printf("\n%s\n", strings.Repeat("─", 60))
	passed := len(results) - failed
	fmt.Printf("Results: %s passed, %s failed\n",
		green(fmt.Sprintf("%d", passed)),
		red(fmt.Sprintf("%d", failed)))

	// Update manifest if requested
	if *updateFlag {
		fmt.Printf("\n%s Updating manifest...\n", cyan("→"))
		updateManifest(m, results)
		m.GeneratedAt = time.Now().UTC()
		if err := m.Save(*manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to save manifest: %v\n", red("Error:"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Manifest updated\n", green("✓"))
	}

	// Generate README section
	readmePath := filepath.Join(filepath.Dir(*manifestPath), "..", "README.md")
	if err := updateREADME(readmePath, m); err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to update README: %v\n", yellow("Warning:"), err)
	} else {
		fmt.Printf("%s README status table updated\n", green("✓"))
	}

	// Exit with error in CI mode if any failures
	if *ciMode && failed > 0 {
		os.Exit(1)
	}
}

func validateExample(dir string, example manifest.Example, verbose bool) ValidationResult {
	result := ValidationResult{
		Path:   example.Path,
		Status: example.Status,
		Passed: true,
	}

	examplePath := filepath.Join(dir, example.Path)

	// Check if file exists
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		result.Passed = false
		result.Message = "File not found"
		return result
	}

	// Skip experimental examples
	if example.Status == manifest.StatusExperimental {
		result.Message = "Skipped (experimental)"
		return result
	}

	// Run the example
	cmd := exec.Command("bin/ailang", "run", examplePath)

	// Set environment if specified
	if example.Environment != nil {
		if example.Environment.Seed != 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("AILANG_SEED=%d", example.Environment.Seed))
		}
		if example.Environment.Locale != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("LC_ALL=%s", example.Environment.Locale))
		}
		if example.Environment.Timezone != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TZ=%s", example.Environment.Timezone))
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	// Combine output for debugging
	result.Output = fmt.Sprintf("STDOUT:\n%s\nSTDERR:\n%s\nExit: %d",
		stdout.String(), stderr.String(), exitCode)

	// Validate based on status
	switch example.Status {
	case manifest.StatusWorking:
		if example.Expected == nil {
			result.Passed = false
			result.Message = "Missing expected output in manifest"
			return result
		}

		// Check exit code
		if exitCode != example.Expected.ExitCode {
			result.Passed = false
			result.Message = fmt.Sprintf("Exit code mismatch: got %d, expected %d",
				exitCode, example.Expected.ExitCode)
			return result
		}

		// Check stdout (normalize line endings)
		expectedStdout := normalizeOutput(example.Expected.Stdout)
		actualStdout := normalizeOutput(stdout.String())
		if expectedStdout != actualStdout {
			result.Passed = false
			result.Message = fmt.Sprintf("Stdout mismatch:\n  Expected: %q\n  Got: %q",
				expectedStdout, actualStdout)
			return result
		}

		// Check stderr
		expectedStderr := normalizeOutput(example.Expected.Stderr)
		actualStderr := normalizeOutput(stderr.String())
		if expectedStderr != actualStderr {
			result.Passed = false
			result.Message = fmt.Sprintf("Stderr mismatch:\n  Expected: %q\n  Got: %q",
				expectedStderr, actualStderr)
			return result
		}

		result.Message = "Output matches expected"

	case manifest.StatusBroken:
		if example.Expected == nil {
			result.Passed = false
			result.Message = "Missing expected error in manifest"
			return result
		}

		// Should fail with non-zero exit
		if exitCode == 0 {
			result.Passed = false
			result.Message = "Expected failure but succeeded"
			return result
		}

		// Check error pattern if specified
		if example.Expected.ErrorPattern != "" {
			pattern := regexp.MustCompile(example.Expected.ErrorPattern)
			combined := stdout.String() + stderr.String()
			if !pattern.MatchString(combined) {
				result.Passed = false
				result.Message = fmt.Sprintf("Error pattern not found: %s",
					example.Expected.ErrorPattern)
				return result
			}
		}

		result.Message = "Failed as expected"
	}

	return result
}

func validateHeader(dir string, example manifest.Example) error {
	examplePath := filepath.Join(dir, example.Path)
	content, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}

	// Look for header comment
	lines := strings.Split(string(content), "\n")
	var headerJSON string
	inHeader := false

	for _, line := range lines {
		if strings.HasPrefix(line, "--! {") {
			inHeader = true
			headerJSON = strings.TrimPrefix(line, "--! ")
		} else if inHeader && strings.HasPrefix(line, "--!") {
			headerJSON += strings.TrimPrefix(line, "--!")
		} else if inHeader && !strings.HasPrefix(line, "--") {
			break
		}
	}

	if headerJSON == "" {
		// No header is OK for working examples
		if example.Status == manifest.StatusWorking {
			return nil
		}
		return fmt.Errorf("missing header for %s example", example.Status)
	}

	// Parse header
	var header map[string]interface{}
	if err := json.Unmarshal([]byte(headerJSON), &header); err != nil {
		return fmt.Errorf("invalid header JSON: %w", err)
	}

	// Validate status matches
	if status, ok := header["status"].(string); ok {
		if manifest.Status(status) != example.Status {
			return fmt.Errorf("status mismatch: header=%s, manifest=%s", status, example.Status)
		}
	}

	// For broken examples, check error code
	if example.Status == manifest.StatusBroken && example.Broken != nil {
		if code, ok := header["error_code"].(string); ok {
			if code != example.Broken.ErrorCode {
				return fmt.Errorf("error code mismatch: header=%s, manifest=%s",
					code, example.Broken.ErrorCode)
			}
		}
	}

	return nil
}

func updateManifest(m *manifest.Manifest, results []ValidationResult) {
	for _, result := range results {
		example, found := m.FindExample(result.Path)
		if !found {
			continue
		}

		// Update status based on result
		if result.Passed {
			if example.Status == manifest.StatusBroken && result.Status == manifest.StatusBroken {
				// Still broken as expected
			} else if example.Status == manifest.StatusBroken {
				// Was broken, now working!
				fmt.Printf("  %s %s is now working!\n", green("🎉"), result.Path)
				example.Status = manifest.StatusWorking
				example.Broken = nil
			}
		} else {
			if example.Status == manifest.StatusWorking {
				// Was working, now broken
				fmt.Printf("  %s %s is now broken\n", red("⚠"), result.Path)
				example.Status = manifest.StatusBroken
				example.Broken = &manifest.BrokenInfo{
					Reason:    result.Message,
					ErrorCode: "PAR001", // Default, should be extracted from actual error
				}
			}
		}
	}

	// Recalculate statistics
	m.UpdateStatistics()
}

func updateREADME(readmePath string, m *manifest.Manifest) error {
	// Read current README
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}

	// Find the example status section
	readme := string(content)
	statusSection := m.GenerateREADMESection()

	// Look for existing section markers
	startMarker := "## Example Status"
	endMarker := "_Last updated:"

	startIdx := strings.Index(readme, startMarker)
	if startIdx == -1 {
		// Add new section before first ## or at end
		firstSection := strings.Index(readme, "\n## ")
		if firstSection != -1 {
			readme = readme[:firstSection] + "\n" + statusSection + "\n" + readme[firstSection:]
		} else {
			readme += "\n" + statusSection
		}
	} else {
		// Replace existing section
		endIdx := strings.Index(readme[startIdx:], endMarker)
		if endIdx == -1 {
			// Find next section
			nextSection := strings.Index(readme[startIdx+1:], "\n## ")
			if nextSection != -1 {
				endIdx = nextSection
			} else {
				endIdx = len(readme) - startIdx
			}
		} else {
			// Include the end marker line
			endLineIdx := strings.Index(readme[startIdx+endIdx:], "\n")
			if endLineIdx != -1 {
				endIdx += endLineIdx
			}
		}
		readme = readme[:startIdx] + statusSection + readme[startIdx+endIdx:]
	}

	return os.WriteFile(readmePath, []byte(readme), 0644)
}

func normalizeOutput(s string) string {
	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Trim trailing whitespace
	s = strings.TrimRight(s, " \t\n")
	return s
}

func indent(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
