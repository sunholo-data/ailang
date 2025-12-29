package coordinator

import (
	"context"
	"testing"
	"time"
)

// MockProvider is a test provider that returns configurable results
type MockProvider struct {
	name       string
	canHandle  bool
	result     *ExecuteResult
	err        error
	executions int
}

func NewMockProvider(name string, canHandle bool, success bool) *MockProvider {
	return &MockProvider{
		name:      name,
		canHandle: canHandle,
		result: &ExecuteResult{
			Success:  success,
			Provider: name,
			Output:   "mock output",
		},
	}
}

func (p *MockProvider) Name() string {
	return p.name
}

func (p *MockProvider) CanHandle(task *AnalyzedTask) bool {
	return p.canHandle
}

func (p *MockProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	p.executions++
	if p.err != nil {
		return nil, p.err
	}
	p.result.Duration = time.Millisecond * 100
	return p.result, nil
}

func TestTaskExecutorSelectProvider(t *testing.T) {
	tests := []struct {
		name         string
		taskType     TaskType
		providers    []Provider
		wantProvider string
	}{
		{
			name:     "coding task uses claude-code",
			taskType: TaskTypeBugFix,
			providers: []Provider{
				NewMockProvider("claude-code", true, true),
				NewMockProvider("gemini-api", true, true),
			},
			wantProvider: "claude-code",
		},
		{
			name:     "docs task prefers gemini-api",
			taskType: TaskTypeDocs,
			providers: []Provider{
				NewMockProvider("gemini-api", true, true),
				NewMockProvider("claude-code", false, true),
			},
			wantProvider: "gemini-api",
		},
		{
			name:     "falls back to available provider",
			taskType: TaskTypeUnknown,
			providers: []Provider{
				NewMockProvider("only-provider", true, true),
			},
			wantProvider: "only-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTaskExecutor(tt.providers...)
			task := &AnalyzedTask{
				Task: &Task{ID: "test-1", Content: "test task"},
				Type: tt.taskType,
			}

			result, err := te.Execute(context.Background(), task, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Provider != tt.wantProvider {
				t.Errorf("got provider %q, want %q", result.Provider, tt.wantProvider)
			}
		})
	}
}

func TestTaskExecutorNoProviders(t *testing.T) {
	te := NewTaskExecutor() // No providers
	task := &AnalyzedTask{
		Task: &Task{ID: "test-1", Content: "test task"},
		Type: TaskTypeBugFix,
	}

	result, err := te.Execute(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure with no providers")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestTaskExecutorDryRun(t *testing.T) {
	mock := NewMockProvider("test", true, true)
	te := NewTaskExecutor(mock)
	task := &AnalyzedTask{
		Task: &Task{ID: "test-1", Content: "test task"},
		Type: TaskTypeBugFix,
	}

	opts := &ExecuteOptions{DryRun: true}
	result, err := te.Execute(context.Background(), task, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Even with dry run, the mock provider is called
	if !result.Success {
		t.Error("expected success")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		errMsg string
		want   bool
	}{
		{"rate limit exceeded", true},
		{"status code 429", true},
		{"connection timeout", true},
		{"network error", true},
		{"500 internal server error", true},
		{"502 bad gateway", true},
		{"syntax error in code", false},
		{"invalid input", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			got := isRetryable(tt.errMsg)
			if got != tt.want {
				t.Errorf("isRetryable(%q) = %v, want %v", tt.errMsg, got, tt.want)
			}
		})
	}
}

func TestBuildDirective(t *testing.T) {
	tests := []struct {
		taskType TaskType
		content  string
		contains string
	}{
		{TaskTypeBugFix, "fix the bug", "BUG FIX REQUEST"},
		{TaskTypeFeature, "add feature", "FEATURE REQUEST"},
		{TaskTypeRefactor, "refactor code", "REFACTORING REQUEST"},
		{TaskTypeTest, "add tests", "TESTING REQUEST"},
		{TaskTypeDocs, "write docs", "write docs"}, // No special prefix for docs
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			task := &AnalyzedTask{
				Task: &Task{Content: tt.content},
				Type: tt.taskType,
			}
			directive := buildDirective(task)
			if !containsSubstring(directive, tt.contains) {
				t.Errorf("directive missing %q, got: %s", tt.contains, directive)
			}
		})
	}
}

func TestBuildAPIPrompt(t *testing.T) {
	tests := []struct {
		taskType TaskType
		contains string
	}{
		{TaskTypeDocs, "documentation task"},
		{TaskTypeResearch, "research task"},
		{TaskTypeUnknown, "help with this task"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			task := &AnalyzedTask{
				Task: &Task{Content: "test content"},
				Type: tt.taskType,
			}
			prompt := buildAPIPrompt(task)
			if !containsSubstring(prompt, tt.contains) {
				t.Errorf("prompt missing %q, got: %s", tt.contains, prompt)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
