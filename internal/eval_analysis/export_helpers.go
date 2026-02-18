package eval_analysis

import "strings"

// formatBenchmarkName converts snake_case benchmark IDs to Title Case for display
func formatBenchmarkName(id string) string {
	// Convert snake_case to Title Case
	words := strings.Split(id, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// formatModelName shortens long model names for table display
func formatModelName(name string) string {
	// Shorten long model names for table display
	switch {
	case strings.Contains(name, "claude-sonnet-4-6"):
		return "Claude Sonnet 4.6"
	case strings.Contains(name, "claude-sonnet-4-5"):
		return "Claude Sonnet 4.5"
	case strings.Contains(name, "gpt-4o-mini"):
		return "GPT-4o Mini"
	case strings.Contains(name, "gpt-4"):
		return "GPT-4"
	case strings.Contains(name, "gemini-2-5-pro"):
		return "Gemini 2.5 Pro"
	default:
		return name
	}
}
