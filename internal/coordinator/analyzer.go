package coordinator

import (
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/simhash"
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

	// Fingerprint for duplicate detection — the one simhash every store
	// persists (internal/simhash). Stored as uint64 here and as int64 in the
	// task stores; the bit pattern is the same, compare with Hamming only.
	//
	// HASH-SPACE NOTE (M-V1-SIMPLIFY-S3 M5, 2026-09-15): before this the
	// coordinator ran its own variant (ASCII-only tokens, 1-char words dropped),
	// so fingerprints written before the switch live in a different space.
	// FindDuplicateTask matches by exact equality inside DedupWindow (24h), so
	// a pre-switch row can fail to suppress a post-switch duplicate for at most
	// that window; it can never false-match. Re-indexing the stored column
	// needs the M3-owned stores (store_sqlite.go, firestore) — Sprint 4.
	analyzed.Fingerprint = uint64(simhash.Hash(task.Content))

	// Check for duplicates
	a.mu.RLock()
	for fp, taskID := range a.fingerprints {
		if simhash.Similarity(int64(analyzed.Fingerprint), int64(fp)) >= a.similarityThreshold {
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

	// Detect capabilities and impact level
	cd := NewCapabilityDetector()
	analyzed.Capabilities = cd.DetectCapabilities(task.Content)
	analyzed.ImpactLevel = cd.ClassifyImpact(analyzed.Capabilities)
	analyzed.EstimatedCost = cd.EstimateTotalCost(analyzed.Capabilities, 0.01) // Base cost $0.01

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
