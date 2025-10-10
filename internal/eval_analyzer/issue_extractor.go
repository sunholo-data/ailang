package eval_analyzer

import (
	"fmt"
	"regexp"
	"strings"
)

// ErrorPattern represents a recognized error pattern with extraction rules
type ErrorPattern struct {
	Name        string
	Regex       *regexp.Regexp
	Category    string
	Severity    string
	Description string
}

// ParsedError contains extracted information from an error message
type ParsedError struct {
	Pattern    string            // Which pattern matched
	Category   string            // Error category
	Context    string            // Surrounding code context
	Suggestion string            // Suggested fix (if known)
	Metadata   map[string]string // Extracted fields (e.g., "missing_feature": "recursion")
}

// ErrorExtractor parses stderr and code to identify specific issues
type ErrorExtractor struct {
	patterns []ErrorPattern
}

// NewErrorExtractor creates a new error extractor with predefined patterns
func NewErrorExtractor() *ErrorExtractor {
	patterns := []ErrorPattern{
		// AILANG-specific patterns
		{
			Name:        "recursion_not_implemented",
			Regex:       regexp.MustCompile(`(?i)(recursion|recursive|self.reference).*(not|un).*(implemented|supported)`),
			Category:    "missing_feature",
			Severity:    "high",
			Description: "Code attempts to use recursion which is not yet implemented",
		},
		{
			Name:        "pattern_guards_unsupported",
			Regex:       regexp.MustCompile(`pattern.*(guard|if).*(not|un).*(supported|implemented)`),
			Category:    "missing_feature",
			Severity:    "medium",
			Description: "Code uses pattern matching guards (if conditions) which are not supported",
		},
		{
			Name:        "error_propagation_missing",
			Regex:       regexp.MustCompile(`\?.*operator.*(not|un).*(supported|implemented)`),
			Category:    "missing_feature",
			Severity:    "medium",
			Description: "Code uses '?' error propagation operator which is not implemented",
		},
		{
			Name:        "parse_error_syntax",
			Regex:       regexp.MustCompile(`parse error.*expected\s+(\w+).*got\s+(\w+)`),
			Category:    "syntax_error",
			Severity:    "high",
			Description: "AI generated invalid syntax",
		},
		{
			Name:        "type_mismatch",
			Regex:       regexp.MustCompile(`type.*(mismatch|error).*expected\s+(\w+).*got\s+(\w+)`),
			Category:    "type_error",
			Severity:    "high",
			Description: "Type inference failed or types don't unify",
		},
		{
			Name:        "unbound_variable",
			Regex:       regexp.MustCompile(`unbound.*variable.*['"]?(\w+)['"]?`),
			Category:    "compile_error",
			Severity:    "high",
			Description: "Reference to undefined variable",
		},
		{
			Name:        "module_not_found",
			Regex:       regexp.MustCompile(`module.*not.found.*['"]?([\w/]+)['"]?`),
			Category:    "compile_error",
			Severity:    "medium",
			Description: "Import references non-existent module",
		},
		{
			Name:        "effect_capability_missing",
			Regex:       regexp.MustCompile(`capability.*['"]?(\w+)['"]?.*not.granted`),
			Category:    "runtime_error",
			Severity:    "high",
			Description: "Code attempts effect without required capability",
		},
		{
			Name:        "deep_let_nesting",
			Regex:       regexp.MustCompile(`let.*nesting.*exceeded|let.*depth.*limit`),
			Category:    "limitation",
			Severity:    "medium",
			Description: "Let expressions nested beyond 3 levels",
		},

		// Python patterns (for comparison)
		{
			Name:        "python_syntax_error",
			Regex:       regexp.MustCompile(`SyntaxError:`),
			Category:    "syntax_error",
			Severity:    "high",
			Description: "Invalid Python syntax",
		},
		{
			Name:        "python_name_error",
			Regex:       regexp.MustCompile(`NameError:.*['"]?(\w+)['"]?`),
			Category:    "runtime_error",
			Severity:    "high",
			Description: "Undefined Python variable",
		},
		{
			Name:        "python_type_error",
			Regex:       regexp.MustCompile(`TypeError:`),
			Category:    "type_error",
			Severity:    "high",
			Description: "Python type mismatch",
		},
	}

	return &ErrorExtractor{
		patterns: patterns,
	}
}

// Extract analyzes stderr and returns parsed error information
func (e *ErrorExtractor) Extract(stderr, code string) *ParsedError {
	// Try each pattern
	for _, pattern := range e.patterns {
		if pattern.Regex.MatchString(stderr) {
			matches := pattern.Regex.FindStringSubmatch(stderr)

			metadata := make(map[string]string)

			// Extract captured groups into metadata
			if len(matches) > 1 {
				for i, match := range matches[1:] {
					metadata[fmt.Sprintf("group_%d", i)] = match
				}
			}

			// Try to extract code context
			context := extractContext(code, stderr)

			// Generate suggestion based on pattern
			suggestion := generateSuggestion(pattern.Name, metadata)

			return &ParsedError{
				Pattern:    pattern.Name,
				Category:   pattern.Category,
				Context:    context,
				Suggestion: suggestion,
				Metadata:   metadata,
			}
		}
	}

	// No pattern matched, return generic error
	return &ParsedError{
		Pattern:  "unknown",
		Category: "unknown_error",
		Context:  truncate(stderr, 200),
		Metadata: make(map[string]string),
	}
}

// extractContext attempts to find the relevant code section from error message
func extractContext(code, stderr string) string {
	// Look for line numbers in stderr
	lineNumRegex := regexp.MustCompile(`line\s+(\d+)`)
	matches := lineNumRegex.FindStringSubmatch(stderr)

	if len(matches) > 1 {
		// TODO: Extract specific line from code
		// For now, just return first few lines
		lines := strings.Split(code, "\n")
		if len(lines) > 5 {
			return strings.Join(lines[:5], "\n") + "\n..."
		}
		return code
	}

	// No line number, return first significant part of code
	lines := strings.Split(code, "\n")
	significant := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") && !strings.HasPrefix(trimmed, "//") {
			significant = append(significant, line)
			if len(significant) >= 5 {
				break
			}
		}
	}

	if len(significant) > 0 {
		return strings.Join(significant, "\n")
	}

	return truncate(code, 500)
}

// generateSuggestion provides a fix suggestion based on the error pattern
func generateSuggestion(patternName string, metadata map[string]string) string {
	suggestions := map[string]string{
		"recursion_not_implemented":  "Rewrite using iteration or list operations. Recursion support is planned for v0.3.0+",
		"pattern_guards_unsupported": "Extract guard condition into separate if-expression. Pattern guards are planned (M-R3)",
		"error_propagation_missing":  "Use explicit match/case handling instead of ? operator. Error propagation is planned for v0.3.0+",
		"parse_error_syntax":         "Review AILANG syntax guide in prompts/v0.3.0.md. Ensure using correct let/func/module syntax",
		"type_mismatch":              "Add explicit type annotations. Check type class constraints (Num, Eq, Ord)",
		"unbound_variable":           "Ensure variable is defined before use. Check imports and function parameters",
		"module_not_found":           "Verify module path. Standard library is in std/ (e.g., std/io, std/fs)",
		"effect_capability_missing":  "Add required capability: ailang run --caps IO,FS --entry main file.ail",
		"deep_let_nesting":           "Refactor to use block expressions {...} instead of deeply nested let bindings",
		"python_syntax_error":        "Review Python syntax",
		"python_name_error":          "Define variable before use",
		"python_type_error":          "Check type compatibility",
	}

	if suggestion, ok := suggestions[patternName]; ok {
		return suggestion
	}

	return "Review error message and code for potential issues"
}

// EnhanceIssueReport adds detailed error analysis to an issue report
func (e *ErrorExtractor) EnhanceIssueReport(issue *IssueReport) {
	// Parse error messages to identify common patterns
	patternCounts := make(map[string]int)
	var suggestions []string

	for i, errMsg := range issue.ErrorMessages {
		code := ""
		if i < len(issue.Examples) {
			code = issue.Examples[i]
		}

		parsed := e.Extract(errMsg, code)

		patternCounts[parsed.Pattern]++

		if parsed.Suggestion != "" && !contains(suggestions, parsed.Suggestion) {
			suggestions = append(suggestions, parsed.Suggestion)
		}
	}

	// Find most common pattern
	maxCount := 0
	mostCommon := ""
	for pattern, count := range patternCounts {
		if count > maxCount {
			maxCount = count
			mostCommon = pattern
		}
	}

	// Update issue title if we identified a specific pattern
	if mostCommon != "unknown" && mostCommon != "" {
		issue.Title = humanizePatternName(mostCommon) + " in " + issue.Lang
	}

	// Store suggestions (would add to IssueReport struct if we extend it)
	// For now, we can add to the category field for debugging
	_ = suggestions // TODO: Store as metadata somewhere accessible
}

// humanizePatternName converts pattern name to human-readable title
func humanizePatternName(pattern string) string {
	// Remove underscores and title case
	words := strings.Split(pattern, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}
