package coordinator

import (
	"strings"
	"sync"
)

// TaskAnalyzer analyzes and classifies tasks
type TaskAnalyzer struct {
	fingerprints        map[uint64]string // fingerprint -> task ID
	similarityThreshold float64
	mu                  sync.RWMutex
}

// NewTaskAnalyzer creates a new task analyzer
func NewTaskAnalyzer(similarityThreshold float64) *TaskAnalyzer {
	if similarityThreshold <= 0 || similarityThreshold > 1 {
		similarityThreshold = 0.8 // Default 80% similarity
	}

	return &TaskAnalyzer{
		fingerprints:        make(map[uint64]string),
		similarityThreshold: similarityThreshold,
	}
}

// Analyze processes a task and returns an AnalyzedTask
func (a *TaskAnalyzer) Analyze(task *Task) *AnalyzedTask {
	analyzed := &AnalyzedTask{
		Task:     task,
		Type:     classifyTaskType(task.Content),
		Keywords: extractKeywords(task.Content),
	}

	// Calculate fingerprint for duplicate detection
	analyzed.Fingerprint = simhash(task.Content)

	// Check for duplicates
	a.mu.RLock()
	for fp, taskID := range a.fingerprints {
		if hammingSimilarity(analyzed.Fingerprint, fp) >= a.similarityThreshold {
			analyzed.DuplicateOf = taskID
			break
		}
	}
	a.mu.RUnlock()

	// Register this task's fingerprint if not a duplicate
	if analyzed.DuplicateOf == "" {
		a.mu.Lock()
		a.fingerprints[analyzed.Fingerprint] = task.ID
		a.mu.Unlock()
	}

	return analyzed
}

// classifyTaskType determines the task type from content
func classifyTaskType(content string) TaskType {
	lower := strings.ToLower(content)

	// Bug/fix indicators
	bugKeywords := []string{"bug", "fix", "error", "crash", "broken", "issue", "problem", "fail", "wrong"}
	for _, kw := range bugKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeBugFix
		}
	}

	// Test indicators - check before feature
	testKeywords := []string{"test", "coverage", "spec", "unittest", "integration test"}
	for _, kw := range testKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeTest
		}
	}

	// Docs indicators - check before feature
	docsKeywords := []string{"document", "docs", "readme", "comment", "example", "guide", "tutorial"}
	for _, kw := range docsKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeDocs
		}
	}

	// Research indicators - check before feature
	researchKeywords := []string{"research", "investigate", "explore", "analyze", "compare", "evaluate", "benchmark"}
	for _, kw := range researchKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeResearch
		}
	}

	// Refactor indicators
	refactorKeywords := []string{"refactor", "cleanup", "reorganize", "restructure", "simplify", "optimize"}
	for _, kw := range refactorKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeRefactor
		}
	}

	// Feature indicators
	featureKeywords := []string{"add", "implement", "create", "new", "feature", "support", "enable"}
	for _, kw := range featureKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeFeature
		}
	}

	return TaskTypeUnknown
}

// extractKeywords extracts significant keywords from content
func extractKeywords(content string) []string {
	// Simple tokenization
	words := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	// Filter stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "been": true, "be": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"this": true, "that": true, "these": true, "those": true, "it": true,
		"i": true, "you": true, "he": true, "she": true, "we": true, "they": true,
		"me": true, "him": true, "her": true, "us": true, "them": true,
		"my": true, "your": true, "his": true, "its": true, "our": true, "their": true,
		"when": true, "where": true, "what": true, "which": true, "who": true, "how": true,
		"if": true, "then": true, "else": true, "so": true, "than": true, "can": true,
		"not": true, "no": true, "yes": true, "all": true, "any": true, "some": true,
	}

	keywords := make([]string, 0)
	seen := make(map[string]bool)

	for _, word := range words {
		// Skip short words, stop words, and duplicates
		if len(word) < 3 || stopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
	}

	// Limit to top 10 keywords
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

// simhash computes a SimHash fingerprint for a string
// SimHash is a locality-sensitive hashing algorithm
func simhash(content string) uint64 {
	// Tokenize
	words := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	// Initialize vector
	var v [64]int

	// For each word, compute hash and update vector
	for _, word := range words {
		if len(word) < 2 {
			continue
		}

		h := fnv1a64(word)
		for i := uint(0); i < 64; i++ {
			if (h>>i)&1 == 1 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	// Convert vector to hash
	var fingerprint uint64
	for i := uint(0); i < 64; i++ {
		if v[i] > 0 {
			fingerprint |= 1 << i
		}
	}

	return fingerprint
}

// fnv1a64 computes FNV-1a hash for a string
func fnv1a64(s string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}

// hammingSimilarity computes similarity between two fingerprints
// Returns a value between 0 and 1
func hammingSimilarity(a, b uint64) float64 {
	// XOR to find differing bits
	diff := a ^ b

	// Count differing bits (Hamming distance)
	distance := 0
	for diff != 0 {
		distance++
		diff &= diff - 1
	}

	// Convert to similarity (1 - normalized distance)
	return 1.0 - float64(distance)/64.0
}

// CalculatePriority calculates task priority based on keywords and type
func CalculatePriority(analyzed *AnalyzedTask) int {
	basePriority := 5 // Default medium priority

	// Adjust by task type
	switch analyzed.Type {
	case TaskTypeBugFix:
		basePriority = 2 // High priority for bugs
	case TaskTypeFeature:
		basePriority = 5
	case TaskTypeTest:
		basePriority = 6
	case TaskTypeDocs:
		basePriority = 7
	case TaskTypeResearch:
		basePriority = 8
	case TaskTypeRefactor:
		basePriority = 6
	}

	// Check for priority boosters in keywords
	boosterKeywords := map[string]int{
		"urgent":    -3,
		"critical":  -3,
		"important": -2,
		"asap":      -2,
		"security":  -2,
		"breaking":  -2,
		"blocker":   -3,
	}

	// Check for priority reducers
	reducerKeywords := map[string]int{
		"minor":      2,
		"small":      1,
		"trivial":    2,
		"nice":       1,
		"eventually": 2,
		"sometime":   2,
	}

	for _, kw := range analyzed.Keywords {
		if adj, ok := boosterKeywords[kw]; ok {
			basePriority += adj
		}
		if adj, ok := reducerKeywords[kw]; ok {
			basePriority += adj
		}
	}

	// Clamp to valid range
	if basePriority < 1 {
		basePriority = 1
	}
	if basePriority > 10 {
		basePriority = 10
	}

	return basePriority
}

// ClearFingerprints clears all stored fingerprints (useful for testing)
func (a *TaskAnalyzer) ClearFingerprints() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.fingerprints = make(map[uint64]string)
}

// FingerprintCount returns the number of stored fingerprints
func (a *TaskAnalyzer) FingerprintCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.fingerprints)
}
