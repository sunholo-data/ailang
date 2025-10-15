//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/scripts/internal/reporttypes"
)

func main() {
	// Read the verification report
	reportFile, err := os.Open("examples_report.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading report: %v\n", err)
		fmt.Println("Run 'make verify-examples' first")
		os.Exit(1)
	}
	defer reportFile.Close()

	var report reporttypes.VerificationReport
	if err := json.NewDecoder(reportFile).Decode(&report); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	updated := 0
	for _, result := range report.Results {
		if result.Status == "failed" {
			filePath := filepath.Join("examples", result.File)
			if err := addWarningHeader(filePath); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", result.File, err)
			} else {
				fmt.Printf("Added warning to %s\n", result.File)
				updated++
			}
		}
	}

	fmt.Printf("\nUpdated %d files with warning headers\n", updated)
}

func addWarningHeader(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Check if warning already exists
	if strings.Contains(contentStr, "WARNING: This example is currently broken") {
		return nil // Already has warning
	}

	// Add warning header
	warning := `-- ⚠️ WARNING: This example is currently broken
-- This file demonstrates planned features that are not yet implemented.
-- It will fail if you try to run it with 'ailang run'.
-- For working examples, see: hello.ail, simple.ail, arithmetic.ail, lambda_expressions.ail

`

	// If file starts with a comment, add after it
	if strings.HasPrefix(contentStr, "--") {
		// Find the first non-comment line
		lines := strings.Split(contentStr, "\n")
		i := 0
		for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "--") {
			i++
		}
		// Insert warning after existing comments
		newLines := append(lines[:i], strings.Split(warning, "\n")...)
		newLines = append(newLines, lines[i:]...)
		contentStr = strings.Join(newLines, "\n")
	} else {
		// Add warning at the beginning
		contentStr = warning + contentStr
	}

	return os.WriteFile(filename, []byte(contentStr), 0644)
}
