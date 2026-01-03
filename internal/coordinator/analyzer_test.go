package coordinator

import (
	"testing"
)

func TestClassifyTaskType(t *testing.T) {
	tests := []struct {
		content  string
		wantType TaskType
	}{
		// Bug/fix detection
		{"Fix the bug in parser", TaskTypeBugFix},
		{"There's an error in the code", TaskTypeBugFix},
		{"The build is broken", TaskTypeBugFix},
		{"Crash when running tests", TaskTypeBugFix},
		{"Something is wrong with imports", TaskTypeBugFix},

		// Test detection
		{"Add tests for the parser", TaskTypeTest},
		{"Improve test coverage", TaskTypeTest},
		{"Write unit tests", TaskTypeTest},
		{"Add integration test for API", TaskTypeTest},

		// Docs detection
		{"Document the API", TaskTypeDocs},
		{"Update the README", TaskTypeDocs},
		{"Add comments to the code", TaskTypeDocs},
		{"Write a tutorial for beginners", TaskTypeDocs},

		// Research detection
		{"Research alternatives to X", TaskTypeResearch},
		{"Investigate the performance characteristics", TaskTypeResearch},
		{"Explore different approaches", TaskTypeResearch},
		{"Benchmark the implementation", TaskTypeResearch},

		// Refactor detection
		{"Refactor the module system", TaskTypeRefactor},
		{"Cleanup old code", TaskTypeRefactor},
		{"Simplify the request handling", TaskTypeRefactor},
		{"Optimize the parser", TaskTypeRefactor},

		// Feature detection
		{"Add support for generics", TaskTypeFeature},
		{"Implement the new syntax", TaskTypeFeature},
		{"Create a new command", TaskTypeFeature},
		{"Enable dark mode", TaskTypeFeature},

		// Unknown (no keywords match)
		{"Something vague", TaskTypeUnknown},
		{"", TaskTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := classifyTaskType(tt.content)
			if got != tt.wantType {
				t.Errorf("classifyTaskType(%q) = %v, want %v", tt.content, got, tt.wantType)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		content      string
		wantKeywords []string
		wantMinCount int
	}{
		{
			content:      "Fix the parser bug in lexer module",
			wantKeywords: []string{"fix", "parser", "bug", "lexer", "module"},
			wantMinCount: 5,
		},
		{
			content:      "Add new feature for authentication and authorization",
			wantKeywords: []string{"add", "new", "feature", "authentication", "authorization"},
			wantMinCount: 5,
		},
		{
			content:      "The quick brown fox jumps over the lazy dog",
			wantKeywords: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
			wantMinCount: 5,
		},
		{
			content:      "a b c", // All too short
			wantKeywords: []string{},
			wantMinCount: 0,
		},
		{
			content:      "", // Empty
			wantKeywords: []string{},
			wantMinCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := extractKeywords(tt.content)

			if len(got) < tt.wantMinCount {
				t.Errorf("extractKeywords(%q) returned %d keywords, want at least %d", tt.content, len(got), tt.wantMinCount)
			}

			// Check that expected keywords are present
			gotSet := make(map[string]bool)
			for _, kw := range got {
				gotSet[kw] = true
			}

			for _, want := range tt.wantKeywords {
				if !gotSet[want] {
					t.Errorf("extractKeywords(%q) missing expected keyword %q, got %v", tt.content, want, got)
				}
			}
		})
	}
}

func TestExtractKeywordsLimit(t *testing.T) {
	// Create content with many words
	content := "word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12 word13 word14 word15"
	keywords := extractKeywords(content)

	if len(keywords) > 10 {
		t.Errorf("extractKeywords should limit to 10 keywords, got %d", len(keywords))
	}
}

func TestSimhash(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"simple", "hello world"},
		{"code", "func main() { fmt.Println(hello) }"},
		{"empty", ""},
		{"single", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic and returns consistent results
			h1 := simhash(tt.content)
			h2 := simhash(tt.content)
			if h1 != h2 {
				t.Errorf("simhash(%q) not consistent: %d != %d", tt.content, h1, h2)
			}
		})
	}
}

func TestSimhashSimilarity(t *testing.T) {
	// Similar texts should have similar hashes
	text1 := "fix the bug in the parser module"
	text2 := "fix the bug in the lexer module"
	text3 := "completely different content about cats and dogs"

	h1 := simhash(text1)
	h2 := simhash(text2)
	h3 := simhash(text3)

	sim12 := hammingSimilarity(h1, h2)
	sim13 := hammingSimilarity(h1, h3)

	// Similar texts should have higher similarity
	if sim12 < sim13 {
		t.Errorf("Expected similar texts to have higher similarity: sim(%q, %q) = %f, sim(%q, %q) = %f",
			text1, text2, sim12, text1, text3, sim13)
	}
}

func TestHammingSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b uint64
		want float64
	}{
		{"identical", 0xFFFF, 0xFFFF, 1.0},
		{"all different", 0x0, 0xFFFFFFFFFFFFFFFF, 0.0},
		{"half different", 0x00000000FFFFFFFF, 0xFFFFFFFF00000000, 0.0},
		{"one bit different", 0x0, 0x1, 63.0 / 64.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hammingSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("hammingSimilarity(%x, %x) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestTaskAnalyzerDuplicateDetection(t *testing.T) {
	analyzer := NewTaskAnalyzer(0.9) // 90% similarity threshold

	// Analyze first task
	task1 := &Task{ID: "task-1", Content: "Fix the critical bug in the parser module"}
	analyzed1 := analyzer.Analyze(task1)

	if analyzed1.DuplicateOf != "" {
		t.Errorf("First task should not be a duplicate, got DuplicateOf=%q", analyzed1.DuplicateOf)
	}

	// Analyze very similar task
	task2 := &Task{ID: "task-2", Content: "Fix the critical bug in the parser module please"}
	analyzed2 := analyzer.Analyze(task2)

	if analyzed2.DuplicateOf != "task-1" {
		t.Errorf("Similar task should be detected as duplicate of task-1, got DuplicateOf=%q", analyzed2.DuplicateOf)
	}

	// Analyze different task
	task3 := &Task{ID: "task-3", Content: "Add new feature for authentication system"}
	analyzed3 := analyzer.Analyze(task3)

	if analyzed3.DuplicateOf != "" {
		t.Errorf("Different task should not be a duplicate, got DuplicateOf=%q", analyzed3.DuplicateOf)
	}
}

func TestTaskAnalyzerClearFingerprints(t *testing.T) {
	analyzer := NewTaskAnalyzer(0.8)

	// Add some tasks
	analyzer.Analyze(&Task{ID: "1", Content: "task one"})
	analyzer.Analyze(&Task{ID: "2", Content: "task two"})

	if analyzer.FingerprintCount() == 0 {
		t.Error("Expected fingerprints to be stored")
	}

	analyzer.ClearFingerprints()

	if analyzer.FingerprintCount() != 0 {
		t.Error("Expected fingerprints to be cleared")
	}
}

func TestCalculatePriority(t *testing.T) {
	tests := []struct {
		name         string
		taskType     TaskType
		keywords     []string
		wantPriority int
		wantMin      int
		wantMax      int
	}{
		{
			name:     "bug fix base priority",
			taskType: TaskTypeBugFix,
			keywords: []string{},
			wantMin:  1,
			wantMax:  3,
		},
		{
			name:     "urgent bug",
			taskType: TaskTypeBugFix,
			keywords: []string{"urgent", "blocker"},
			wantMin:  1,
			wantMax:  1,
		},
		{
			name:     "minor feature",
			taskType: TaskTypeFeature,
			keywords: []string{"minor", "small"},
			wantMin:  6,
			wantMax:  8,
		},
		{
			name:     "docs low priority",
			taskType: TaskTypeDocs,
			keywords: []string{"eventually"},
			wantMin:  8,
			wantMax:  10,
		},
		{
			name:     "security critical",
			taskType: TaskTypeUnknown,
			keywords: []string{"security", "critical"},
			wantMin:  1,
			wantMax:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzed := &AnalyzedTask{
				Task:     &Task{},
				Type:     tt.taskType,
				Keywords: tt.keywords,
			}

			got := CalculatePriority(analyzed)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculatePriority() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCalculatePriorityClamp(t *testing.T) {
	// Test that priority is clamped to 1-10 range
	analyzed := &AnalyzedTask{
		Task:     &Task{},
		Type:     TaskTypeBugFix,
		Keywords: []string{"urgent", "critical", "blocker", "asap", "security", "breaking"},
	}

	got := CalculatePriority(analyzed)
	if got < 1 {
		t.Errorf("Priority should be at least 1, got %d", got)
	}

	// Test upper bound
	analyzed2 := &AnalyzedTask{
		Task:     &Task{},
		Type:     TaskTypeDocs,
		Keywords: []string{"minor", "trivial", "eventually", "sometime"},
	}

	got2 := CalculatePriority(analyzed2)
	if got2 > 10 {
		t.Errorf("Priority should be at most 10, got %d", got2)
	}
}

func TestNewTaskAnalyzerDefaults(t *testing.T) {
	// Test invalid threshold defaults to 0.8
	analyzer := NewTaskAnalyzer(0)
	if analyzer.similarityThreshold != 0.8 {
		t.Errorf("Expected default threshold 0.8, got %f", analyzer.similarityThreshold)
	}

	analyzer2 := NewTaskAnalyzer(-1)
	if analyzer2.similarityThreshold != 0.8 {
		t.Errorf("Expected default threshold 0.8, got %f", analyzer2.similarityThreshold)
	}

	analyzer3 := NewTaskAnalyzer(1.5)
	if analyzer3.similarityThreshold != 0.8 {
		t.Errorf("Expected default threshold 0.8, got %f", analyzer3.similarityThreshold)
	}

	// Valid threshold should be used
	analyzer4 := NewTaskAnalyzer(0.5)
	if analyzer4.similarityThreshold != 0.5 {
		t.Errorf("Expected threshold 0.5, got %f", analyzer4.similarityThreshold)
	}
}
