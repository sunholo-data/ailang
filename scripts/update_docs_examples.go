//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/scripts/internal/reporttypes"
)

func main() {
	// Read the verification report
	reportFile, err := os.Open("examples_report.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading examples report: %v\n", err)
		os.Exit(1)
	}
	defer reportFile.Close()

	var report reporttypes.VerificationReport
	if err := json.NewDecoder(reportFile).Decode(&report); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Generate markdown table
	statusTable := generateStatusTable(report)

	// Update docs/docs/examples.mdx
	docsPath := "docs/docs/examples.mdx"
	docsContent, err := os.ReadFile(docsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading docs examples page: %v\n", err)
		os.Exit(1)
	}

	// Update docs page with new status
	updatedContent := updateExamplesStatus(string(docsContent), statusTable, report)

	// Write updated docs page
	if err := os.WriteFile(docsPath, []byte(updatedContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing docs examples page: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Docs examples page updated successfully")
}

func generateStatusTable(report reporttypes.VerificationReport) string {
	var sb strings.Builder

	// Add badges
	sb.WriteString("## Status\n\n")
	sb.WriteString("![Examples](https://img.shields.io/badge/examples-")
	if report.Failed == 0 {
		sb.WriteString(fmt.Sprintf("%d%%20passing-brightgreen", report.Passed))
	} else {
		sb.WriteString(fmt.Sprintf("%d%%20passing%%20%d%%20failing-", report.Passed, report.Failed))
		// Choose color based on percentage
		percentage := float64(report.Passed) / float64(report.TotalExamples) * 100
		if percentage >= 80 {
			sb.WriteString("green")
		} else if percentage >= 60 {
			sb.WriteString("orange")
		} else {
			sb.WriteString("red")
		}
	}
	sb.WriteString(".svg)\n\n")

	// Add summary
	sb.WriteString("### Example Verification Status\n\n")
	sb.WriteString(fmt.Sprintf("*Last updated: %s (Auto-updated by CI)*\n\n", report.Timestamp.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Summary:** %d passed, %d failed, %d skipped (Total: %d)\n\n",
		report.Passed, report.Failed, report.Skipped, report.TotalExamples))

	// Calculate percentage
	percentage := float64(report.Passed) / float64(report.TotalExamples) * 100
	sb.WriteString(fmt.Sprintf("**Overall: %d/%d examples working (%.0f%%)**\n\n",
		report.Passed, report.TotalExamples, percentage))

	// Create status table with GitHub links
	sb.WriteString("| Example File | Status | Notes |\n")
	sb.WriteString("|--------------|--------|-------|\n")

	const githubBase = "https://github.com/sunholo-data/ailang/blob/dev/examples/"

	for _, result := range report.Results {
		statusIcon := getStatusIcon(result.Status)
		notes := ""
		if result.Status == "failed" && result.Error != "" {
			// Extract first line of error, filtering out stdlib warnings
			lines := strings.Split(result.Error, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// Skip stdlib version mismatch warnings - not useful for users
				if strings.Contains(line, "stdlib version mismatch") {
					continue
				}
				if line != "" {
					if len(line) > 55 {
						line = line[:52] + "..."
					}
					notes = line
					break
				}
			}
		} else if result.Status == "skipped" {
			notes = "Test/demo file"
		}

		// Create GitHub link for the file
		fileLink := fmt.Sprintf("[`%s`](%s%s)", result.File, githubBase, result.File)
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", fileLink, statusIcon, notes))
	}

	return sb.String()
}

func getStatusIcon(status string) string {
	switch status {
	case "passed":
		return "✅ Pass"
	case "failed":
		return "❌ Fail"
	case "skipped":
		return "⏭️ Skip"
	default:
		return "❓ Unknown"
	}
}

func updateExamplesStatus(content, statusTable string, report reporttypes.VerificationReport) string {
	// Look for markers in docs page
	startMarker := "<!-- EXAMPLES_STATUS_START -->"
	endMarker := "<!-- EXAMPLES_STATUS_END -->"

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 {
		fmt.Fprintf(os.Stderr, "Warning: EXAMPLES_STATUS markers not found in docs/docs/examples.mdx\n")
		return content
	}

	// Replace content between markers
	before := content[:startIdx+len(startMarker)]
	after := content[endIdx:]
	result := before + "\n" + statusTable + "\n" + after

	// Also update hardcoded example counts outside the markers
	// Pattern: "97+" or "66+" etc should become dynamic
	totalStr := fmt.Sprintf("%d+", report.TotalExamples)

	// Replace common patterns for example counts
	// "for all 97+ examples" -> "for all {total}+ examples"
	// "97+ verified examples" -> "{total}+ verified examples"
	result = replaceExampleCounts(result, totalStr)

	return result
}

// replaceExampleCounts updates hardcoded example counts to the actual total
func replaceExampleCounts(content, totalStr string) string {
	// Common patterns to replace (be careful not to replace version numbers)
	patterns := []string{
		"for all 66+ examples",
		"for all 97+ examples",
		"66+ example files",
		"97+ example files",
		"66+ verified examples",
		"97+ verified examples",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			// Figure out the replacement based on the pattern
			var replacement string
			if strings.Contains(pattern, "for all") {
				replacement = "for all " + totalStr + " examples"
			} else if strings.Contains(pattern, "example files") {
				replacement = totalStr + " example files"
			} else if strings.Contains(pattern, "verified examples") {
				replacement = totalStr + " verified examples"
			}
			content = strings.ReplaceAll(content, pattern, replacement)
		}
	}

	return content
}
