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

	// Read current README
	readmeContent, err := os.ReadFile("README.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading README: %v\n", err)
		os.Exit(1)
	}

	// Update README with new status
	updatedContent := updateReadmeStatus(string(readmeContent), statusTable)

	// Write updated README
	if err := os.WriteFile("README.md", []byte(updatedContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing README: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("README updated successfully")
}

func generateStatusTable(report reporttypes.VerificationReport) string {
	var sb strings.Builder

	// Add badges
	sb.WriteString("## Status\n\n")
	sb.WriteString("![Examples](https://img.shields.io/badge/examples-")
	if report.Failed == 0 {
		sb.WriteString(fmt.Sprintf("%d%%20passing-brightgreen", report.Passed))
	} else {
		sb.WriteString(fmt.Sprintf("%d%%20passing%%20%d%%20failing-red", report.Passed, report.Failed))
	}
	sb.WriteString(".svg)\n\n")

	// Add summary
	sb.WriteString("### Example Verification Status\n\n")
	sb.WriteString(fmt.Sprintf("*Last updated: %s*\n\n", report.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("**Summary:** %d passed, %d failed, %d skipped (Total: %d)\n\n",
		report.Passed, report.Failed, report.Skipped, report.TotalExamples))

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
				if len(firstLine) > 50 {
					firstLine = firstLine[:47] + "..."
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

func updateReadmeStatus(content, statusTable string) string {
	// Look for markers in README
	startMarker := "<!-- EXAMPLES_STATUS_START -->"
	endMarker := "<!-- EXAMPLES_STATUS_END -->"

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 {
		// Markers not found, add them after the main title
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				// Found main title, insert after it
				newLines := append(lines[:i+1],
					"",
					startMarker,
					statusTable,
					endMarker,
				)
				newLines = append(newLines, lines[i+1:]...)
				return strings.Join(newLines, "\n")
			}
		}
		// No main title found, prepend
		return startMarker + "\n" + statusTable + "\n" + endMarker + "\n\n" + content
	}

	// Replace content between markers
	before := content[:startIdx+len(startMarker)]
	after := content[endIdx:]
	return before + "\n" + statusTable + "\n" + after
}
