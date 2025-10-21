//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/scripts/internal/reporttypes"
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
	updatedContent := updateExamplesStatus(string(docsContent), statusTable)

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

	// Create status table
	sb.WriteString("| Example File | Status | Notes |\n")
	sb.WriteString("|--------------|--------|-------|\n")

	for _, result := range report.Results {
		statusIcon := getStatusIcon(result.Status)
		notes := ""
		if result.Status == "failed" && result.Error != "" {
			// Extract first line of error
			lines := strings.Split(result.Error, "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				if len(firstLine) > 60 {
					firstLine = firstLine[:57] + "..."
				}
				notes = firstLine
			}
		} else if result.Status == "skipped" {
			notes = "Test/demo file"
		}

		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", result.File, statusIcon, notes))
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

func updateExamplesStatus(content, statusTable string) string {
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
	return before + "\n" + statusTable + "\n" + after
}
