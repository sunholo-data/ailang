package agent

import (
	"fmt"
	"strings"
	"time"
)

// MaxOutputSize is the maximum size of output to include in formatted results (50KB)
const MaxOutputSize = 50 * 1024

// truncateOutput truncates text to maxLen characters, adding a note if truncated
func truncateOutput(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Find a good break point (newline) near the truncation point
	truncated := text[:maxLen]
	if idx := strings.LastIndex(truncated, "\n"); idx > maxLen*3/4 {
		truncated = truncated[:idx]
	}
	lines := strings.Count(text, "\n")
	truncatedLines := strings.Count(truncated, "\n")
	return fmt.Sprintf("%s\n\n... **[Output truncated: showing %d of %d lines, %d of %d bytes]**",
		truncated, truncatedLines, lines, len(truncated), len(text))
}

// FormatResult formats a DirectiveResult as markdown for display in the UI
func FormatResult(result *DirectiveResult) string {
	var sb strings.Builder

	// Header with status
	if result.Success {
		sb.WriteString("## ✅ Directive Completed Successfully\n\n")
	} else {
		sb.WriteString("## ❌ Directive Failed\n\n")
	}

	// Execution metadata
	sb.WriteString("### Execution Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", formatDuration(result.DurationMS)))
	sb.WriteString(fmt.Sprintf("- **Cost**: $%.4f\n", result.Cost))
	sb.WriteString(fmt.Sprintf("- **Turns**: %d\n", result.NumTurns))
	sb.WriteString(fmt.Sprintf("- **Session ID**: `%s`\n", result.SessionID))
	sb.WriteString("\n")

	// Token usage
	if result.TokensUsed.InputTokens > 0 || result.TokensUsed.OutputTokens > 0 {
		sb.WriteString("### Token Usage\n\n")
		sb.WriteString(fmt.Sprintf("- **Input**: %s tokens", formatNumber(result.TokensUsed.InputTokens)))
		if result.TokensUsed.CacheReadInputTokens > 0 {
			sb.WriteString(fmt.Sprintf(" (%s from cache)", formatNumber(result.TokensUsed.CacheReadInputTokens)))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("- **Output**: %s tokens\n", formatNumber(result.TokensUsed.OutputTokens)))
		if result.TokensUsed.CacheCreationInputTokens > 0 {
			sb.WriteString(fmt.Sprintf("- **Cache Created**: %s tokens\n", formatNumber(result.TokensUsed.CacheCreationInputTokens)))
		}
		sb.WriteString("\n")
	}

	// Output/Error (truncated to prevent UI issues with large responses)
	if result.Success {
		if result.Output != "" {
			sb.WriteString("### Result\n\n")
			sb.WriteString(truncateOutput(result.Output, MaxOutputSize))
			sb.WriteString("\n\n")
		}
	} else {
		if result.Error != "" {
			sb.WriteString("### Error\n\n")
			sb.WriteString("```\n")
			sb.WriteString(truncateOutput(result.Error, MaxOutputSize))
			sb.WriteString("\n```\n\n")
		}
	}

	// Files created
	if len(result.FilesCreated) > 0 {
		sb.WriteString("### Files Created\n\n")
		for _, file := range result.FilesCreated {
			sb.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
		sb.WriteString("\n")
	}

	// Workspace location - prominent link for inspection
	if result.Workspace != "" {
		sb.WriteString("### Workspace\n\n")
		sb.WriteString(fmt.Sprintf("📁 [Open workspace folder](%s)\n\n", result.Workspace))
		sb.WriteString(fmt.Sprintf("*Path: `%s`*\n\n", result.Workspace))
	}

	return sb.String()
}

// FormatResultCompact formats a DirectiveResult as a compact single-line summary
func FormatResultCompact(result *DirectiveResult) string {
	status := "✅"
	if !result.Success {
		status = "❌"
	}

	return fmt.Sprintf("%s Completed in %s (cost: $%.4f, %d turns, %d files created)",
		status,
		formatDuration(result.DurationMS),
		result.Cost,
		result.NumTurns,
		len(result.FilesCreated))
}

// FormatResultWithTranscript formats a DirectiveResult including the full transcript
func FormatResultWithTranscript(result *DirectiveResult) string {
	var sb strings.Builder

	// Standard result formatting
	sb.WriteString(FormatResult(result))

	// Add transcript (truncated for large transcripts)
	if result.Transcript != "" {
		sb.WriteString("---\n\n")
		sb.WriteString("### Full Transcript\n\n")
		sb.WriteString("<details>\n")
		sb.WriteString("<summary>Click to expand conversation</summary>\n\n")
		sb.WriteString("```\n")
		sb.WriteString(truncateOutput(result.Transcript, MaxOutputSize))
		sb.WriteString("\n```\n\n")
		sb.WriteString("</details>\n")
	}

	return sb.String()
}

// formatDuration formats milliseconds as a human-readable duration
func formatDuration(ms int) string {
	d := time.Duration(ms) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}

	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	return fmt.Sprintf("%.1fm", d.Minutes())
}

// formatNumber formats a number with thousands separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}

	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}
