package eval_analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MergeStrategy determines how to handle an issue when similar docs exist
type MergeStrategy string

const (
	StrategyCreate MergeStrategy = "create" // No similar doc exists
	StrategyMerge  MergeStrategy = "merge"  // Update existing doc
	StrategySkip   MergeStrategy = "skip"   // Already well-documented
	StrategyLink   MergeStrategy = "link"   // Related but distinct
)

// SimilarDoc represents a design doc similar to the current issue
type SimilarDoc struct {
	Path            string
	Filename        string
	SimilarityScore float64
	Category        string
	Language        string
	Benchmarks      []string
	Frequency       int
}

// DedupConfig configures deduplication behavior
type DedupConfig struct {
	Enabled          bool
	MergeThreshold   float64 // Similarity % for merging (0.0-1.0)
	ForceNew         bool    // Always create new docs
	SkipWellDocumented bool  // Skip if issue is already comprehensive
}

// DefaultDedupConfig returns default deduplication settings
func DefaultDedupConfig() DedupConfig {
	return DedupConfig{
		Enabled:          true,
		MergeThreshold:   0.75, // 75% similarity
		ForceNew:         false,
		SkipWellDocumented: false,
	}
}

// FindSimilarDesignDocs searches for design docs similar to the given issue
func FindSimilarDesignDocs(issue IssueReport, plannedDir string, config DedupConfig) ([]SimilarDoc, error) {
	if !config.Enabled || config.ForceNew {
		return nil, nil
	}

	// List all existing design docs
	pattern := filepath.Join(plannedDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob design docs: %w", err)
	}

	var similar []SimilarDoc

	for _, file := range files {
		// Skip summary and enhancement docs
		base := filepath.Base(file)
		if strings.HasPrefix(base, "EVAL_ANALYSIS_") || strings.HasPrefix(base, "ENHANCEMENT_") {
			continue
		}

		// Calculate similarity
		doc, score, err := calculateSimilarity(file, issue)
		if err != nil {
			continue // Skip files we can't read
		}

		if score >= config.MergeThreshold {
			doc.SimilarityScore = score
			similar = append(similar, *doc)
		}
	}

	return similar, nil
}

// calculateSimilarity computes how similar a design doc is to an issue
func calculateSimilarity(docPath string, issue IssueReport) (*SimilarDoc, float64, error) {
	content, err := os.ReadFile(docPath)
	if err != nil {
		return nil, 0, err
	}

	contentStr := string(content)

	doc := &SimilarDoc{
		Path:     docPath,
		Filename: filepath.Base(docPath),
	}

	// Parse metadata from doc
	parseDocMetadata(contentStr, doc)

	score := 0.0
	weights := 0.0

	// 1. Category match (weight: 0.3)
	if doc.Category == issue.Category {
		score += 0.3
		weights += 0.3
	} else {
		weights += 0.3
	}

	// 2. Language match (weight: 0.2)
	if doc.Language == issue.Lang {
		score += 0.2
		weights += 0.2
	} else {
		weights += 0.2
	}

	// 3. Benchmark overlap (weight: 0.3)
	benchmarkOverlap := calculateBenchmarkOverlap(doc.Benchmarks, issue.Benchmarks)
	score += 0.3 * benchmarkOverlap
	weights += 0.3

	// 4. Error message similarity (weight: 0.2)
	errorSimilarity := calculateErrorSimilarity(contentStr, issue.ErrorMessages)
	score += 0.2 * errorSimilarity
	weights += 0.2

	if weights > 0 {
		return doc, score / weights, nil
	}

	return doc, 0, nil
}

// parseDocMetadata extracts metadata from a design doc
func parseDocMetadata(content string, doc *SimilarDoc) {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Parse title
		if i == 0 && strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			// Extract language from title (e.g., "AILANG: Runtime Errors")
			if strings.Contains(title, "AILANG") {
				doc.Language = "ailang"
			} else if strings.Contains(title, "PYTHON") {
				doc.Language = "python"
			}
		}

		// Parse category
		if strings.HasPrefix(line, "**Category**:") {
			doc.Category = strings.TrimSpace(strings.TrimPrefix(line, "**Category**:"))
		}

		// Parse frequency
		if strings.HasPrefix(line, "**Frequency**:") {
			fmt.Sscanf(line, "**Frequency**: %d", &doc.Frequency)
		}

		// Parse benchmarks
		if strings.HasPrefix(line, "**Affected Benchmarks**:") {
			benchStr := strings.TrimSpace(strings.TrimPrefix(line, "**Affected Benchmarks**:"))
			doc.Benchmarks = strings.Split(benchStr, ", ")
			for i := range doc.Benchmarks {
				doc.Benchmarks[i] = strings.TrimSpace(doc.Benchmarks[i])
			}
		}
	}
}

// calculateBenchmarkOverlap computes overlap between two benchmark lists
func calculateBenchmarkOverlap(benchmarks1, benchmarks2 []string) float64 {
	if len(benchmarks1) == 0 || len(benchmarks2) == 0 {
		return 0.0
	}

	// Count overlapping benchmarks
	overlap := 0
	for _, b1 := range benchmarks1 {
		for _, b2 := range benchmarks2 {
			if b1 == b2 {
				overlap++
				break
			}
		}
	}

	// Jaccard similarity: intersection / union
	union := len(benchmarks1) + len(benchmarks2) - overlap
	if union == 0 {
		return 0.0
	}

	return float64(overlap) / float64(union)
}

// calculateErrorSimilarity computes similarity between doc errors and issue errors
func calculateErrorSimilarity(docContent string, issueErrors []string) float64 {
	if len(issueErrors) == 0 {
		return 0.0
	}

	// Extract error examples from doc content
	docErrors := extractErrorsFromDoc(docContent)

	if len(docErrors) == 0 {
		return 0.0
	}

	// Count similar errors (fuzzy match)
	matches := 0
	for _, issueErr := range issueErrors {
		for _, docErr := range docErrors {
			if fuzzyErrorMatch(issueErr, docErr) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(issueErrors))
}

// extractErrorsFromDoc extracts error messages from a design doc
func extractErrorsFromDoc(content string) []string {
	var errors []string

	// Look for error blocks (```...```)
	errorBlockRegex := regexp.MustCompile("(?s)```\n(Error:.*?)\n```")
	matches := errorBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			errors = append(errors, match[1])
		}
	}

	return errors
}

// fuzzyErrorMatch checks if two error messages are similar
func fuzzyErrorMatch(err1, err2 string) bool {
	// Normalize: lowercase, remove extra whitespace
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.Join(strings.Fields(s), " ")
		return s
	}

	err1Norm := normalize(err1)
	err2Norm := normalize(err2)

	// Exact match
	if err1Norm == err2Norm {
		return true
	}

	// Contains (one error is substring of other)
	if strings.Contains(err1Norm, err2Norm) || strings.Contains(err2Norm, err1Norm) {
		return true
	}

	// Key phrase match (extract key parts)
	keyPhrases1 := extractKeyPhrases(err1Norm)
	keyPhrases2 := extractKeyPhrases(err2Norm)

	// If 50%+ key phrases overlap, consider similar
	overlap := 0
	for _, kp1 := range keyPhrases1 {
		for _, kp2 := range keyPhrases2 {
			if kp1 == kp2 {
				overlap++
				break
			}
		}
	}

	maxLen := len(keyPhrases1)
	if len(keyPhrases2) > maxLen {
		maxLen = len(keyPhrases2)
	}

	return maxLen > 0 && float64(overlap)/float64(maxLen) >= 0.5
}

// extractKeyPhrases extracts key phrases from an error message
func extractKeyPhrases(err string) []string {
	// Common error message patterns
	patterns := []string{
		`builtin \w+`,
		`expects \w+`,
		`type \w+`,
		`failed to \w+`,
		`cannot \w+`,
		`undefined \w+`,
		`missing \w+`,
	}

	var phrases []string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(err, -1)
		phrases = append(phrases, matches...)
	}

	return phrases
}

// DetermineMergeStrategy decides how to handle an issue given similar docs
func DetermineMergeStrategy(issue IssueReport, similar []SimilarDoc, config DedupConfig) (MergeStrategy, *SimilarDoc) {
	if !config.Enabled || config.ForceNew {
		return StrategyCreate, nil
	}

	if len(similar) == 0 {
		return StrategyCreate, nil
	}

	// Sort by similarity score (highest first)
	bestMatch := &similar[0]
	for i := range similar {
		if similar[i].SimilarityScore > bestMatch.SimilarityScore {
			bestMatch = &similar[i]
		}
	}

	// Decision logic
	score := bestMatch.SimilarityScore

	// Very high similarity (>90%) - merge or skip
	if score >= 0.9 {
		if config.SkipWellDocumented && bestMatch.Frequency >= issue.Frequency {
			// Already well-documented with similar or more failures
			return StrategySkip, bestMatch
		}
		return StrategyMerge, bestMatch
	}

	// High similarity (>75%) - merge
	if score >= config.MergeThreshold {
		return StrategyMerge, bestMatch
	}

	// Moderate similarity (>50%) - link as related
	if score >= 0.5 {
		return StrategyLink, bestMatch
	}

	// Low similarity - create new
	return StrategyCreate, nil
}

// MergeDesignDoc merges new evidence into an existing design doc
func MergeDesignDoc(existingPath string, issue IssueReport, totalFailures int) error {
	// Read existing doc
	content, err := os.ReadFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to read existing doc: %w", err)
	}

	contentStr := string(content)

	// Parse existing frequency
	existingFreq := 0
	freqRegex := regexp.MustCompile(`\*\*Frequency\*\*: (\d+)`)
	if match := freqRegex.FindStringSubmatch(contentStr); len(match) > 1 {
		fmt.Sscanf(match[1], "%d", &existingFreq)
	}

	// Update frequency
	newFreq := existingFreq + issue.Frequency
	contentStr = freqRegex.ReplaceAllString(contentStr, fmt.Sprintf("**Frequency**: %d", newFreq))

	// Update benchmark count
	benchmarkRegex := regexp.MustCompile(`(\d+) benchmark\(s\)`)
	existingBenchmarks := make(map[string]bool)
	for _, line := range strings.Split(contentStr, "\n") {
		if strings.HasPrefix(line, "**Affected Benchmarks**:") {
			benchStr := strings.TrimSpace(strings.TrimPrefix(line, "**Affected Benchmarks**:"))
			for _, b := range strings.Split(benchStr, ", ") {
				existingBenchmarks[strings.TrimSpace(b)] = true
			}
		}
	}

	// Merge benchmarks
	for _, b := range issue.Benchmarks {
		existingBenchmarks[b] = true
	}

	benchmarkList := make([]string, 0, len(existingBenchmarks))
	for b := range existingBenchmarks {
		benchmarkList = append(benchmarkList, b)
	}

	// Update benchmark list in doc
	benchmarkStr := strings.Join(benchmarkList, ", ")
	benchRegex := regexp.MustCompile(`\*\*Affected Benchmarks\*\*: [^\n]+`)
	contentStr = benchRegex.ReplaceAllString(contentStr, fmt.Sprintf("**Affected Benchmarks**: %s", benchmarkStr))

	// Update benchmark count
	contentStr = benchmarkRegex.ReplaceAllString(contentStr, fmt.Sprintf("%d benchmark(s)", len(benchmarkList)))

	// Add update timestamp
	now := time.Now().Format("2006-01-06")
	updateNote := fmt.Sprintf("\n\n**Last Updated**: %s (merged %d new failures)\n", now, issue.Frequency)

	// Insert before "## Evidence from AI Eval" section
	evidenceMarker := "## Evidence from AI Eval"
	if idx := strings.Index(contentStr, evidenceMarker); idx != -1 {
		contentStr = contentStr[:idx] + updateNote + contentStr[idx:]
	} else {
		// Append at end if section not found
		contentStr += updateNote
	}

	// Add new error examples
	if len(issue.ErrorMessages) > 0 && len(issue.Examples) > 0 {
		newExamples := "\n### Additional Examples (Latest Analysis)\n\n"
		for i := 0; i < len(issue.ErrorMessages) && i < 2; i++ {
			code := ""
			if i < len(issue.Examples) {
				code = issue.Examples[i]
			}

			newExamples += fmt.Sprintf("**Error %d:**\n```\n%s\n```\n\n", i+1, truncate(issue.ErrorMessages[i], 500))
			if code != "" {
				newExamples += fmt.Sprintf("**Generated Code:**\n```%s\n%s\n```\n\n---\n\n", issue.Lang, truncate(code, 1000))
			}
		}

		// Insert after existing examples
		exampleSectionEnd := "---\n\n\n## Root Cause Analysis"
		if idx := strings.Index(contentStr, exampleSectionEnd); idx != -1 {
			contentStr = contentStr[:idx] + newExamples + contentStr[idx:]
		}
	}

	// Write updated doc
	if err := os.WriteFile(existingPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write merged doc: %w", err)
	}

	return nil
}
