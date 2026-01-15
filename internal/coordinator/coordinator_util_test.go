package coordinator

import (
	"strings"
	"testing"
)

// TestCapabilityDetectorBasic tests basic capability detection
func TestCapabilityDetectorBasic(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedCapTypes []CapabilityType
		shouldHaveNoCaps bool
	}{
		{
			name:             "empty content",
			content:          "",
			shouldHaveNoCaps: true,
		},
		{
			name:             "whitespace only",
			content:          "   \n\t  ",
			shouldHaveNoCaps: true,
		},
		{
			name:    "IO capability detection with print",
			content: "Print information to stdout",
			expectedCapTypes: []CapabilityType{
				CapabilityIO,
			},
		},
		{
			name:    "FS capability detection with write",
			content: "Write data to the system",
			expectedCapTypes: []CapabilityType{
				CapabilityFS,
			},
		},
		{
			name:    "multiple capabilities",
			content: "Write to disk and print results",
			expectedCapTypes: []CapabilityType{
				CapabilityIO,
				CapabilityFS,
			},
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := cd.DetectCapabilities(tt.content)

			if tt.shouldHaveNoCaps {
				if len(caps) != 0 {
					t.Errorf("expected no capabilities, got %v", caps)
				}
				return
			}

			// Check that expected capabilities are present
			capMap := make(map[CapabilityType]bool)
			for _, c := range caps {
				capMap[c.Type] = true
			}

			for _, expected := range tt.expectedCapTypes {
				if !capMap[expected] {
					t.Errorf("expected capability %s not found in %v",
						expected, caps)
				}
			}
		})
	}
}

// TestCapabilityDetectorEdgeCases tests edge cases for capability detection
func TestCapabilityDetectorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []CapabilityType
	}{
		{
			name:     "case insensitive IO detection (uppercase)",
			content:  "PRINT TO CONSOLE",
			expected: []CapabilityType{CapabilityIO},
		},
		{
			name:     "case insensitive IO detection (mixed)",
			content:  "Print important data",
			expected: []CapabilityType{CapabilityIO},
		},
		{
			name:     "network capability with http",
			content:  "Make an HTTP request to the server",
			expected: []CapabilityType{CapabilityNet},
		},
		{
			name:     "clock capability detection",
			content:  "Sleep for 5 seconds",
			expected: []CapabilityType{CapabilityClock},
		},
		{
			name:     "environment capability",
			content:  "Read environment secret values",
			expected: []CapabilityType{CapabilityEnv},
		},
		{
			name:     "shell capability (high risk)",
			content:  "Execute bash commands",
			expected: []CapabilityType{CapabilityShell},
		},
		{
			name:     "AI capability detection",
			content:  "Call the LLM API using claude",
			expected: []CapabilityType{CapabilityAI},
		},
		{
			name:     "debug capability",
			content:  "Debug and trace the execution",
			expected: []CapabilityType{CapabilityDebug},
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := cd.DetectCapabilities(tt.content)

			if len(caps) < len(tt.expected) {
				capTypes := make([]CapabilityType, len(caps))
				for i, c := range caps {
					capTypes[i] = c.Type
				}
				t.Errorf("expected at least capabilities %v, got %v",
					tt.expected, capTypes)
				return
			}

			// Check that all expected capabilities are present
			capMap := make(map[CapabilityType]bool)
			for _, c := range caps {
				capMap[c.Type] = true
			}

			for _, exp := range tt.expected {
				if !capMap[exp] {
					t.Errorf("expected capability %s not found", exp)
				}
			}
		})
	}
}

// TestCapabilityDetectorStress tests the detector with large/complex content
func TestCapabilityDetectorStress(t *testing.T) {
	tests := []struct {
		name           string
		contentBuilder func() string
		minExpected    int
	}{
		{
			name: "repeated keywords",
			contentBuilder: func() string {
				result := ""
				for i := 0; i < 100; i++ {
					result += "print output "
				}
				return result
			},
			minExpected: 1, // Should still detect IO
		},
		{
			name: "mixed keywords in sentences",
			contentBuilder: func() string {
				return "Please print to stdout, " +
					"write to files, " +
					"make HTTP calls, " +
					"use sleep for timing, " +
					"check secret env values, " +
					"and execute bash scripts"
			},
			minExpected: 5,
		},
		{
			name: "very long content",
			contentBuilder: func() string {
				result := ""
				for i := 0; i < 1000; i++ {
					result += "This sentence requires printing to output. "
				}
				return result
			},
			minExpected: 1,
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := tt.contentBuilder()
			caps := cd.DetectCapabilities(content)

			if len(caps) < tt.minExpected {
				t.Errorf("expected at least %d capabilities, got %d",
					tt.minExpected, len(caps))
			}
		})
	}
}

// TestClassifyTaskTypeEdgeCases tests edge cases for task type classification
func TestClassifyTaskTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected TaskType
	}{
		{
			name:     "empty string",
			content:  "",
			expected: TaskTypeUnknown,
		},
		{
			name:     "only whitespace",
			content:  "   \n\t  ",
			expected: TaskTypeUnknown,
		},
		{
			name:     "bug fix with multiple keywords",
			content:  "Fix the bug in error handling code",
			expected: TaskTypeBugFix,
		},
		{
			name:     "test with multiple keywords",
			content:  "Add unit tests and test coverage",
			expected: TaskTypeTest,
		},
		{
			name:     "docs priority",
			content:  "Document and add comments",
			expected: TaskTypeDocs,
		},
		{
			name:     "research investigation",
			content:  "Investigate and research alternatives",
			expected: TaskTypeResearch,
		},
		{
			name:     "refactor optimization",
			content:  "Refactor and optimize the code",
			expected: TaskTypeRefactor,
		},
		{
			name:     "feature implementation",
			content:  "Implement new feature support",
			expected: TaskTypeFeature,
		},
		{
			name:     "case insensitivity",
			content:  "FIX THE BUG IN PARSER",
			expected: TaskTypeBugFix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyTaskType(tt.content)
			if got != tt.expected {
				t.Errorf("classifyTaskType(%q) = %v, want %v",
					tt.content, got, tt.expected)
			}
		})
	}
}

// TestExtractKeywordsEdgeCases tests edge cases for keyword extraction
func TestExtractKeywordsEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		minExpectedLength int
		shouldBeEmpty     bool
	}{
		{
			name:              "empty string",
			content:           "",
			minExpectedLength: 0,
			shouldBeEmpty:     true,
		},
		{
			name:              "only short words",
			content:           "a b c d e",
			minExpectedLength: 0,
			shouldBeEmpty:     true,
		},
		{
			name:              "mixed short and long words",
			content:           "a b c database server network",
			minExpectedLength: 3,
			shouldBeEmpty:     false,
		},
		{
			name:              "repeated words",
			content:           "parser parser lexer lexer",
			minExpectedLength: 2,
			shouldBeEmpty:     false,
		},
		{
			name:              "very long keyword",
			content:           "supercalifragilisticexpialidocious",
			minExpectedLength: 1,
			shouldBeEmpty:     false,
		},
		{
			name:              "mixed case keywords",
			content:           "Parser LExer PARSER lexer",
			minExpectedLength: 1,
			shouldBeEmpty:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeywords(tt.content)

			if tt.shouldBeEmpty {
				if len(got) != 0 {
					t.Errorf("expected empty keywords, got %v", got)
				}
			} else if len(got) < tt.minExpectedLength {
				t.Errorf("expected at least %d keywords, got %d: %v",
					tt.minExpectedLength, len(got), got)
			}
		})
	}
}

// TestNewCapabilityDetector tests the constructor
func TestNewCapabilityDetector(t *testing.T) {
	cd := NewCapabilityDetector()
	if cd == nil {
		t.Errorf("NewCapabilityDetector returned nil")
	}

	// Verify it's functional
	caps := cd.DetectCapabilities("random neutral text with no keywords")
	if caps == nil {
		// Nil slice is acceptable if no capabilities detected
		t.Logf("DetectCapabilities returned nil for no-capability content")
	}
	// Just ensure it doesn't panic
}

// TestCapabilityDetectorPathExtraction tests file path extraction
func TestCapabilityDetectorPathExtraction(t *testing.T) {
	tests := []struct {
		name    string
		content string
		hasPath bool
	}{
		{
			name:    "content with quoted path",
			content: `Edit the file "/path/to/config.json"`,
			hasPath: true,
		},
		{
			name:    "content without quoted path",
			content: "Edit the configuration file",
			hasPath: false, // No quotes, so wildcard
		},
		{
			name:    "multiple quoted paths",
			content: `Copy "/source/file.txt" to "/dest/file.txt"`,
			hasPath: true,
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := cd.DetectCapabilities(tt.content)

			// Find FS capability
			var fsCap *Capability
			for i := range caps {
				if caps[i].Type == CapabilityFS {
					fsCap = &caps[i]
					break
				}
			}

			if fsCap == nil {
				t.Errorf("FS capability not detected")
				return
			}

			if tt.hasPath {
				// Check that at least one path was extracted
				if len(fsCap.Paths) == 0 {
					t.Errorf("expected paths to be extracted, got %v", fsCap.Paths)
				}
			}
		})
	}
}

// TestClassifyImpact tests impact classification
func TestClassifyImpact(t *testing.T) {
	tests := []struct {
		name           string
		capabilities   []Capability
		expectedImpact string
	}{
		{
			name:           "no capabilities",
			capabilities:   []Capability{},
			expectedImpact: "low",
		},
		{
			name: "IO only (low risk)",
			capabilities: []Capability{
				{Type: CapabilityIO},
			},
			expectedImpact: "low",
		},
		{
			name: "FS access (medium risk)",
			capabilities: []Capability{
				{Type: CapabilityFS},
			},
			expectedImpact: "medium",
		},
		{
			name: "Shell access (high risk)",
			capabilities: []Capability{
				{Type: CapabilityShell},
			},
			expectedImpact: "high",
		},
		{
			name: "AI access (high risk)",
			capabilities: []Capability{
				{Type: CapabilityAI},
			},
			expectedImpact: "high",
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cd.ClassifyImpact(tt.capabilities)
			if got != tt.expectedImpact {
				t.Errorf("ClassifyImpact() = %v, want %v", got, tt.expectedImpact)
			}
		})
	}
}

// TestFormatImpact tests impact formatting
func TestFormatImpact(t *testing.T) {
	tests := []struct {
		name          string
		capabilities  []Capability
		shouldContain string
	}{
		{
			name:          "no capabilities",
			capabilities:  []Capability{},
			shouldContain: "Low risk",
		},
		{
			name: "IO capability",
			capabilities: []Capability{
				{Type: CapabilityIO},
			},
			shouldContain: "Console I/O",
		},
		{
			name: "Shell capability",
			capabilities: []Capability{
				{Type: CapabilityShell, Paths: []string{"*"}},
			},
			shouldContain: "HIGH RISK",
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cd.FormatImpact(tt.capabilities)
			if !strings.Contains(got, tt.shouldContain) {
				t.Errorf("FormatImpact() = %q, should contain %q", got, tt.shouldContain)
			}
		})
	}
}

// TestEstimateTotalCost tests cost estimation
func TestEstimateTotalCost(t *testing.T) {
	tests := []struct {
		name              string
		capabilities      []Capability
		baseExecutionCost float64
		expectedMin       float64
		expectedMax       float64
	}{
		{
			name:              "no capabilities",
			capabilities:      []Capability{},
			baseExecutionCost: 0.10,
			expectedMin:       0.10,
			expectedMax:       0.10,
		},
		{
			name: "with budget capability",
			capabilities: []Capability{
				{Type: CapabilityBudget, BudgetDelta: 0.50},
			},
			baseExecutionCost: 0.10,
			expectedMin:       0.60,
			expectedMax:       0.60,
		},
		{
			name: "multiple capabilities",
			capabilities: []Capability{
				{Type: CapabilityBudget, BudgetDelta: 0.50},
				{Type: CapabilityBudget, BudgetDelta: 0.30},
			},
			baseExecutionCost: 0.10,
			expectedMin:       0.89, // Allow for floating point precision
			expectedMax:       0.91,
		},
	}

	cd := NewCapabilityDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cd.EstimateTotalCost(tt.capabilities, tt.baseExecutionCost)
			if got < tt.expectedMin || got > tt.expectedMax {
				t.Errorf("EstimateTotalCost() = %v, want between %v and %v",
					got, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// BenchmarkCapabilityDetection benchmarks capability detection performance
func BenchmarkCapabilityDetection(b *testing.B) {
	cd := NewCapabilityDetector()
	content := "Print debug information to IO, write to files with FS, make HTTP requests for network, and sleep with clock"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cd.DetectCapabilities(content)
	}
}

// BenchmarkTaskClassification benchmarks task type classification
func BenchmarkTaskClassification(b *testing.B) {
	content := "Fix the bug in the parser that causes crashes when processing large files"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classifyTaskType(content)
	}
}

// BenchmarkKeywordExtraction benchmarks keyword extraction
func BenchmarkKeywordExtraction(b *testing.B) {
	content := "Implement a comprehensive solution to detect capabilities and classify tasks for the autonomous agent system"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractKeywords(content)
	}
}
