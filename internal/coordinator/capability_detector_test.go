package coordinator

import (
	"testing"
)

func TestCapabilityDetector_DetectCapabilities(t *testing.T) {
	cd := NewCapabilityDetector()

	tests := []struct {
		name     string
		content  string
		wantCaps []CapabilityType
	}{
		{
			name:     "no capabilities for simple question",
			content:  "what is the meaning of life",
			wantCaps: nil,
		},
		{
			name:     "FS capability for file operations",
			content:  "create a new file called test.go",
			wantCaps: []CapabilityType{CapabilityFS},
		},
		{
			name:     "FS capability for edit operations",
			content:  "edit the function to add error handling",
			wantCaps: []CapabilityType{CapabilityFS},
		},
		{
			name:     "Net capability for API operations",
			content:  "call the https://api.example.com endpoint",
			wantCaps: []CapabilityType{CapabilityNet},
		},
		{
			name:     "Net capability for fetch operations",
			content:  "fetch data from the server",
			wantCaps: []CapabilityType{CapabilityNet},
		},
		{
			name:     "Shell capability for npm",
			content:  "run npm install",
			wantCaps: []CapabilityType{CapabilityFS, CapabilityShell}, // install triggers FS too
		},
		{
			name:     "Shell capability for git",
			content:  "use git to commit the changes",
			wantCaps: []CapabilityType{CapabilityShell},
		},
		{
			name:     "Shell capability for docker",
			content:  "build the docker image",
			wantCaps: []CapabilityType{CapabilityShell},
		},
		{
			name:     "Shell capability for make",
			content:  "run make test",
			wantCaps: []CapabilityType{CapabilityShell},
		},
		{
			name:     "Budget capability for refactoring",
			content:  "refactor the entire codebase to use the new pattern",
			wantCaps: []CapabilityType{CapabilityFS, CapabilityBudget}, // refactor triggers both FS and Budget
		},
		{
			name:    "multiple capabilities",
			content: "download the file from https://example.com and save it",
			wantCaps: []CapabilityType{CapabilityFS, CapabilityNet},
		},
		// New AILANG effect capabilities
		{
			name:     "IO capability for print operations",
			content:  "print the result to the console",
			wantCaps: []CapabilityType{CapabilityIO},
		},
		{
			name:     "Clock capability for sleep operations",
			content:  "sleep for 5 seconds then resume",
			wantCaps: []CapabilityType{CapabilityClock},
		},
		{
			name:     "Env capability for getenv operations",
			content:  "use getenv to get the secret",
			wantCaps: []CapabilityType{CapabilityEnv},
		},
		{
			name:     "AI capability for LLM operations",
			content:  "use claude to generate a summary",
			wantCaps: []CapabilityType{CapabilityAI},
		},
		{
			name:     "Debug capability for debugging",
			content:  "add a breakpoint here",
			wantCaps: []CapabilityType{CapabilityDebug},
		},
		{
			name:     "multiple AILANG capabilities",
			content:  "print the clock timestamp to console",
			wantCaps: []CapabilityType{CapabilityIO, CapabilityClock},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := cd.DetectCapabilities(tt.content)

			if len(caps) != len(tt.wantCaps) {
				t.Errorf("got %d capabilities, want %d", len(caps), len(tt.wantCaps))
				t.Logf("got: %v", caps)
				return
			}

			// Check each expected capability is present
			for _, wantCap := range tt.wantCaps {
				found := false
				for _, cap := range caps {
					if cap.Type == wantCap {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing capability %s in result", wantCap)
				}
			}
		})
	}
}

func TestCapabilityDetector_ClassifyImpact(t *testing.T) {
	cd := NewCapabilityDetector()

	tests := []struct {
		name       string
		caps       []Capability
		wantImpact string
	}{
		{
			name:       "no capabilities is low risk",
			caps:       nil,
			wantImpact: "low",
		},
		{
			name:       "empty capabilities is low risk",
			caps:       []Capability{},
			wantImpact: "low",
		},
		{
			name:       "shell is high risk",
			caps:       []Capability{{Type: CapabilityShell}},
			wantImpact: "high",
		},
		{
			name:       "shell + FS is still high risk",
			caps:       []Capability{{Type: CapabilityFS}, {Type: CapabilityShell}},
			wantImpact: "high",
		},
		{
			name:       "network is medium risk",
			caps:       []Capability{{Type: CapabilityNet}},
			wantImpact: "medium",
		},
		{
			name:       "budget is medium risk",
			caps:       []Capability{{Type: CapabilityBudget}},
			wantImpact: "medium",
		},
		{
			name:       "FS only is medium risk",
			caps:       []Capability{{Type: CapabilityFS}},
			wantImpact: "medium",
		},
		// New AILANG effect capabilities
		{
			name:       "AI is high risk",
			caps:       []Capability{{Type: CapabilityAI}},
			wantImpact: "high",
		},
		{
			name:       "Env is medium risk",
			caps:       []Capability{{Type: CapabilityEnv}},
			wantImpact: "medium",
		},
		{
			name:       "IO only is low risk",
			caps:       []Capability{{Type: CapabilityIO}},
			wantImpact: "low",
		},
		{
			name:       "Clock only is low risk",
			caps:       []Capability{{Type: CapabilityClock}},
			wantImpact: "low",
		},
		{
			name:       "Debug only is low risk",
			caps:       []Capability{{Type: CapabilityDebug}},
			wantImpact: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cd.ClassifyImpact(tt.caps)
			if got != tt.wantImpact {
				t.Errorf("ClassifyImpact() = %v, want %v", got, tt.wantImpact)
			}
		})
	}
}

func TestCapabilityDetector_EstimateCost(t *testing.T) {
	cd := NewCapabilityDetector()

	// Simple content should be $0.10
	simple := "fix the bug"
	if cost := cd.estimateCost(simple); cost != 0.10 {
		t.Errorf("simple content cost = %v, want 0.10", cost)
	}

	// Medium content (>20 words) should be $0.20
	medium := "please analyze the code and find all the places where we need to add error handling and then update the functions accordingly"
	if cost := cd.estimateCost(medium); cost != 0.20 {
		t.Errorf("medium content cost = %v, want 0.20", cost)
	}

	// Complex content (>50 words) should be $0.50
	complex := "please perform a comprehensive analysis of the entire codebase including all modules and packages to identify any potential issues with the current implementation and then create a detailed report with recommendations for improvements including code quality metrics performance optimizations security vulnerabilities and best practices compliance across all files and directories in addition please review the test coverage documentation completeness and architectural decisions to ensure everything meets our high quality standards"
	if cost := cd.estimateCost(complex); cost != 0.50 {
		t.Errorf("complex content cost = %v, want 0.50", cost)
	}
}

func TestCapabilityDetector_ExtractPaths(t *testing.T) {
	cd := NewCapabilityDetector()

	// Test quoted path extraction
	content := `modify the file "internal/parser/parser.go" and update the test`
	paths := cd.extractPaths(content)
	if len(paths) != 1 || paths[0] != "internal/parser/parser.go" {
		t.Errorf("extractPaths() = %v, want [internal/parser/parser.go]", paths)
	}

	// Test wildcard fallback
	content2 := "modify all the files in the project"
	paths2 := cd.extractPaths(content2)
	if len(paths2) != 1 || paths2[0] != "*" {
		t.Errorf("extractPaths() = %v, want [*]", paths2)
	}
}

func TestCapabilityDetector_ExtractURLs(t *testing.T) {
	cd := NewCapabilityDetector()

	// Test URL extraction
	content := "fetch data from https://api.example.com/v1/users and save it"
	urls := cd.extractURLs(content)
	if len(urls) != 1 || urls[0] != "https://api.example.com/v1/users" {
		t.Errorf("extractURLs() = %v, want [https://api.example.com/v1/users]", urls)
	}

	// Test wildcard fallback
	content2 := "make an api request to the server"
	urls2 := cd.extractURLs(content2)
	if len(urls2) != 1 || urls2[0] != "*" {
		t.Errorf("extractURLs() = %v, want [*]", urls2)
	}
}

func TestCapabilityDetector_FormatImpact(t *testing.T) {
	cd := NewCapabilityDetector()

	tests := []struct {
		name string
		caps []Capability
		want string
	}{
		{
			name: "no capabilities",
			caps: nil,
			want: "Low risk - read-only operations",
		},
		{
			name: "FS with wildcard",
			caps: []Capability{{Type: CapabilityFS, Paths: []string{"*"}}},
			want: "May modify files in workspace",
		},
		{
			name: "FS with specific path",
			caps: []Capability{{Type: CapabilityFS, Paths: []string{"src/main.go"}}},
			want: "May modify files: src/main.go",
		},
		{
			name: "Shell capability",
			caps: []Capability{{Type: CapabilityShell}},
			want: "HIGH RISK - Can execute arbitrary shell commands",
		},
		{
			name: "Net capability",
			caps: []Capability{{Type: CapabilityNet}},
			want: "May make network requests",
		},
		{
			name: "Budget capability",
			caps: []Capability{{Type: CapabilityBudget}},
			want: "May incur additional costs",
		},
		// New AILANG effect capabilities
		{
			name: "IO capability",
			caps: []Capability{{Type: CapabilityIO}},
			want: "Console I/O",
		},
		{
			name: "Clock capability",
			caps: []Capability{{Type: CapabilityClock}},
			want: "Time/scheduling operations",
		},
		{
			name: "Env capability",
			caps: []Capability{{Type: CapabilityEnv}},
			want: "May access environment variables",
		},
		{
			name: "AI capability",
			caps: []Capability{{Type: CapabilityAI}},
			want: "HIGH RISK - External AI/LLM API calls",
		},
		{
			name: "Debug capability",
			caps: []Capability{{Type: CapabilityDebug}},
			want: "Debugging operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cd.FormatImpact(tt.caps)
			if got != tt.want {
				t.Errorf("FormatImpact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskAnalyzer_AnalyzeWithCapabilities(t *testing.T) {
	analyzer := NewTaskAnalyzer(0.8)

	task := &Task{
		ID:      "test-1",
		Content: "run npm install and then edit the package.json file",
	}

	analyzed := analyzer.Analyze(task)

	// Should detect FS (edit, file) and Shell (npm, install)
	if len(analyzed.Capabilities) < 2 {
		t.Errorf("expected at least 2 capabilities, got %d", len(analyzed.Capabilities))
	}

	// Should be high risk due to shell
	if analyzed.ImpactLevel != "high" {
		t.Errorf("expected high impact level, got %s", analyzed.ImpactLevel)
	}

	// Should have estimated cost
	if analyzed.EstimatedCost <= 0 {
		t.Errorf("expected positive estimated cost, got %f", analyzed.EstimatedCost)
	}
}

// TestCapabilityDetection_VerboseDemo demonstrates capability detection with different messages
// Run with: go test -v -run TestCapabilityDetection_VerboseDemo ./internal/coordinator/...
func TestCapabilityDetection_VerboseDemo(t *testing.T) {
	analyzer := NewTaskAnalyzer(0.8)
	cd := NewCapabilityDetector()

	testCases := []struct {
		name    string
		content string
	}{
		{"Simple Question", "What is the best approach for implementing caching?"},
		{"File Operations", "Please read the config.json file and update the database settings"},
		{"Network API", "Call the API at https://api.example.com/users and parse the response"},
		{"Shell Commands", "Run npm install and then execute make build to compile"},
		{"Large Refactor", "Refactor the entire authentication module to use OAuth2"},
		{"Multi-Capability", "Edit /etc/hosts, fetch https://api.example.com, run docker build, and refactor auth"},
	}

	t.Log("\n=== Capability Detection Demo ===")
	for _, tc := range testCases {
		task := &Task{ID: "demo", Content: tc.content}
		analyzed := analyzer.Analyze(task)

		capTypes := make([]string, len(analyzed.Capabilities))
		for i, cap := range analyzed.Capabilities {
			capTypes[i] = string(cap.Type)
		}

		t.Logf("\n📋 %s", tc.name)
		t.Logf("   Content: %q", truncateForDemo(tc.content, 60))
		if len(capTypes) == 0 {
			t.Logf("   Capabilities: none")
		} else {
			t.Logf("   Capabilities: %v", capTypes)
		}
		t.Logf("   Impact: %s | Est. Cost: $%.2f", analyzed.ImpactLevel, analyzed.EstimatedCost)
		t.Logf("   Human-readable: %s", cd.FormatImpact(analyzed.Capabilities))
	}
}

func truncateForDemo(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
