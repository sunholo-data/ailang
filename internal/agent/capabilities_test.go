package agent

import (
	"strings"
	"testing"
)

func TestDetectCapabilities(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name          string
		directive     string
		expectedTypes []string // Expected capability types
		expectedCount int
		shouldBeEmpty bool
	}{
		{
			name:          "Safe directive - no capabilities",
			directive:     "Explain how AILANG works",
			expectedTypes: []string{},
			expectedCount: 0,
			shouldBeEmpty: true,
		},
		{
			name:          "File system operation",
			directive:     "Create a file called hello.txt",
			expectedTypes: []string{"FS"},
			expectedCount: 1,
		},
		{
			name:          "Network operation",
			directive:     "Fetch data from https://api.example.com",
			expectedTypes: []string{"Net"},
			expectedCount: 1,
		},
		{
			name:          "Shell operation",
			directive:     "Run npm install to install dependencies",
			expectedTypes: []string{"FS", "Shell"},
			expectedCount: 2,
		},
		{
			name:          "High cost operation",
			directive:     "Refactor the entire codebase to use async/await",
			expectedTypes: []string{"FS", "Budget"},
			expectedCount: 2,
		},
		{
			name:          "Multiple capabilities",
			directive:     "Download the API docs from https://example.com and save to docs/ folder",
			expectedTypes: []string{"FS", "Net"},
			expectedCount: 2,
		},
		{
			name:          "Shell with git",
			directive:     "Run git commit -m 'initial commit'",
			expectedTypes: []string{"Shell"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deltas := detector.DetectCapabilities(tt.directive)

			if tt.shouldBeEmpty && len(deltas) != 0 {
				t.Errorf("Expected no capabilities, got %d", len(deltas))
				return
			}

			if !tt.shouldBeEmpty && len(deltas) != tt.expectedCount {
				t.Errorf("Expected %d capabilities, got %d", tt.expectedCount, len(deltas))
			}

			// Check that expected types are present
			foundTypes := make(map[string]bool)
			for _, delta := range deltas {
				foundTypes[delta.CapType] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !foundTypes[expectedType] {
					t.Errorf("Expected capability type %s not found", expectedType)
				}
			}
		})
	}
}

func TestNeedsFileSystem(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		directive string
		expected  bool
	}{
		{"Create a new file", true},
		{"Write to config.yaml", true},
		{"Delete old logs", true},
		{"Read the documentation", true},
		{"Explain how closures work", false},
		{"Calculate fibonacci", false},
	}

	for _, tt := range tests {
		t.Run(tt.directive, func(t *testing.T) {
			result := detector.needsFileSystem(strings.ToLower(tt.directive))
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for directive: %s", tt.expected, result, tt.directive)
			}
		})
	}
}

func TestNeedsNetwork(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		directive string
		expected  bool
	}{
		{"Fetch data from API", true},
		{"Download https://example.com/data.json", true},
		{"Make a POST request", true},
		{"Create a webhook handler", true},
		{"Explain HTTP methods", true}, // False positive OK - contains "http"
		{"Create a local server", false},
	}

	for _, tt := range tests {
		t.Run(tt.directive, func(t *testing.T) {
			result := detector.needsNetwork(strings.ToLower(tt.directive))
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for directive: %s", tt.expected, result, tt.directive)
			}
		})
	}
}

func TestNeedsShell(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		directive string
		expected  bool
	}{
		{"Run bash command ls -la", true},
		{"Execute npm install", true},
		{"Run git status", true},
		{"Install the dependencies", true},
		{"Make the build", true},
		{"Explain shell scripting", true}, // False positive OK - contains "shell"
		{"Create a Makefile", true},       // False positive OK - contains "make"
		{"Calculate a sum", false},        // No shell keywords
	}

	for _, tt := range tests {
		t.Run(tt.directive, func(t *testing.T) {
			result := detector.needsShell(strings.ToLower(tt.directive))
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for directive: %s", tt.expected, result, tt.directive)
			}
		})
	}
}

func TestNeedsHighCost(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		directive string
		expected  bool
	}{
		{"Refactor the entire codebase", true},
		{"Analyze entire project structure", true},
		{"Comprehensive security audit", true},
		{"Migrate all files to new format", true},
		{"Create a simple function", false},
		{"Fix this bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.directive, func(t *testing.T) {
			result := detector.needsHighCost(strings.ToLower(tt.directive))
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for directive: %s", tt.expected, result, tt.directive)
			}
		})
	}
}

func TestExtractPaths(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name      string
		directive string
		expected  []string
	}{
		{
			name:      "Quoted path",
			directive: `Create "src/main.go" file`,
			expected:  []string{"src/main.go"},
		},
		{
			name:      "Multiple quoted paths",
			directive: `Copy "src/old.go" to "src/new.go"`,
			expected:  []string{"src/old.go", "src/new.go"},
		},
		{
			name:      "No paths - wildcard",
			directive: "Create some files",
			expected:  []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.extractPaths(tt.directive)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d paths, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected path %s, got %s", expected, result[i])
				}
			}
		})
	}
}

func TestExtractURLs(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name      string
		directive string
		expected  []string
	}{
		{
			name:      "Single URL",
			directive: "Fetch data from https://api.example.com",
			expected:  []string{"https://api.example.com"},
		},
		{
			name:      "HTTP and HTTPS",
			directive: "Get http://old.com and https://new.com",
			expected:  []string{"http://old.com", "https://new.com"},
		},
		{
			name:      "No URLs - wildcard",
			directive: "Make some API calls",
			expected:  []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.extractURLs(tt.directive)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d URLs, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected URL %s, got %s", expected, result[i])
				}
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name      string
		directive string
		minCost   float64
		maxCost   float64
	}{
		{
			name:      "Simple directive",
			directive: "Create file",
			minCost:   0.10,
			maxCost:   0.10,
		},
		{
			name:      "Medium directive - 17 words",
			directive: "Create a new function that calculates the factorial of a number recursively with proper error handling",
			minCost:   0.10, // 17 words < 20, so $0.10
			maxCost:   0.10,
		},
		{
			name: "Complex directive - 35 words",
			directive: "Refactor the entire authentication system to use JWT tokens instead of session cookies, update all API endpoints, " +
				"write comprehensive tests, update documentation, and ensure backward compatibility with existing clients using feature flags",
			minCost: 0.20, // 35 words: > 20 but < 50, so $0.20
			maxCost: 0.20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.estimateCost(tt.directive)
			if result < tt.minCost || result > tt.maxCost {
				t.Errorf("Expected cost between $%.2f and $%.2f, got $%.2f", tt.minCost, tt.maxCost, result)
			}
		})
	}
}

func TestFormatProposal(t *testing.T) {
	detector := NewCapabilityDetector()

	directive := "Download API docs from https://example.com and save to docs/api/"
	deltas := detector.DetectCapabilities(directive)

	proposal := detector.FormatProposal(deltas)

	expectedParts := []string{"File system", "Network"}
	for _, part := range expectedParts {
		if !strings.Contains(proposal, part) {
			t.Errorf("Proposal missing expected part: %s\nGot: %s", part, proposal)
		}
	}
}

func TestFormatImpact(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name            string
		directive       string
		expectedPhrases []string
	}{
		{
			name:            "File system impact",
			directive:       "Create hello.txt",
			expectedPhrases: []string{"modify files"},
		},
		{
			name:            "Network impact",
			directive:       "Fetch https://example.com",
			expectedPhrases: []string{"network requests"},
		},
		{
			name:            "Shell impact - high risk",
			directive:       "Run bash script",
			expectedPhrases: []string{"HIGH RISK", "shell commands"},
		},
		{
			name:            "Safe directive",
			directive:       "Explain how functions work",
			expectedPhrases: []string{"Low risk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deltas := detector.DetectCapabilities(tt.directive)
			impact := detector.FormatImpact(deltas)

			for _, phrase := range tt.expectedPhrases {
				if !strings.Contains(impact, phrase) {
					t.Errorf("Impact missing expected phrase: %s\nGot: %s", phrase, impact)
				}
			}
		})
	}
}

func TestCalculateTotalCost(t *testing.T) {
	detector := NewCapabilityDetector()

	tests := []struct {
		name        string
		directive   string
		baseCost    float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "Simple file operation",
			directive:   "Create hello.txt",
			baseCost:    0.005, // Base Claude execution
			expectedMin: 0.005,
			expectedMax: 0.01,
		},
		{
			name:        "High cost refactoring - 7 words",
			directive:   "Refactor the entire codebase with comprehensive changes",
			baseCost:    0.010,
			expectedMin: 0.11, // Base $0.01 + budget delta $0.10 (7 words)
			expectedMax: 0.12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deltas := detector.DetectCapabilities(tt.directive)
			total := detector.CalculateTotalCost(deltas, tt.baseCost)

			if total < tt.expectedMin || total > tt.expectedMax {
				t.Errorf("Expected cost between $%.3f and $%.3f, got $%.3f",
					tt.expectedMin, tt.expectedMax, total)
			}
		})
	}
}
