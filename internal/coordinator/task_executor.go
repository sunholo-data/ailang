package coordinator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/executor"
)

// TaskExecutor orchestrates task execution across multiple providers.
// It routes tasks to the appropriate provider based on task type and
// provider capabilities.
type TaskExecutor struct {
	providers      []Provider
	defaultCoding  Provider // Default for coding tasks (bug-fix, feature, refactor, test)
	defaultSimple  Provider // Default for simple tasks (docs, research)
	scriptProvider Provider // Provider for deterministic script execution (v0.6.4+)
}

// NewTaskExecutor creates a new task executor with the given providers
func NewTaskExecutor(providers ...Provider) *TaskExecutor {
	te := &TaskExecutor{
		providers: providers,
	}

	// Set defaults based on provider type
	for _, p := range providers {
		if _, ok := p.(*ExecutorProvider); ok && te.defaultCoding == nil {
			// ExecutorProvider = CLI-based agentic coding (Claude, Gemini, Codex, etc.)
			te.defaultCoding = p
		}
		if _, ok := p.(*GeminiAPIProvider); ok && te.defaultSimple == nil {
			// API-based provider for simple text generation tasks
			te.defaultSimple = p
		}
	}

	return te
}

// DefaultTaskExecutor creates a TaskExecutor with all available providers.
// It discovers executor-based providers from the global executor factory,
// so adding a new executor (e.g., codex) only requires registering it via init().
func DefaultTaskExecutor() (*TaskExecutor, error) {
	var providers []Provider

	// Discover all registered executors from the factory
	for _, name := range executor.GlobalFactory().ListAvailable() {
		provider, err := NewExecutorProvider(name)
		if err != nil {
			continue // Skip executors that aren't available (e.g., binary not found)
		}
		providers = append(providers, provider)
	}

	// Try to create Gemini API provider (API-based, not executor-based)
	geminiAPI, err := NewGeminiAPIProvider()
	if err == nil {
		providers = append(providers, geminiAPI)
	}

	// Script provider is always available (no external dependencies)
	scriptProvider := NewScriptProvider()

	if len(providers) == 0 {
		// Even with no AI providers, scripts can still work
		te := NewTaskExecutor()
		te.scriptProvider = scriptProvider
		return te, nil
	}

	te := NewTaskExecutor(providers...)
	te.scriptProvider = scriptProvider
	return te, nil
}

// Execute runs a task using the best available provider
func (te *TaskExecutor) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Check for script invoke type first (v0.6.4+)
	// Script execution is determined by InvokeConfig, not task type
	if opts.InvokeConfig != nil && opts.InvokeConfig.Type == "script" {
		if te.scriptProvider != nil {
			fmt.Printf("[DEBUG] TaskExecutor: Using script provider for task (invoke.type=script)\n")
			return te.scriptProvider.Execute(ctx, task, opts)
		}
		return &ExecuteResult{
			Success: false,
			Error:   "script provider not available",
		}, nil
	}

	// Find the best AI provider for this task
	provider := te.selectProvider(task)
	if provider == nil {
		return &ExecuteResult{
			Success: false,
			Error:   "no provider available for this task type",
		}, nil
	}

	// Debug logging for provider selection
	hasEventHandler := opts.EventHandler != nil
	fmt.Printf("[DEBUG] TaskExecutor: Selected provider '%s' for task type '%s' (EventHandler: %v)\n",
		provider.Name(), task.Type, hasEventHandler)

	// Execute the task
	return provider.Execute(ctx, task, opts)
}

// ExecuteWithRetry runs a task with retry logic
func (te *TaskExecutor) ExecuteWithRetry(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions, maxRetries int) (*ExecuteResult, error) {
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	var lastResult *ExecuteResult
	baseDelay := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := te.Execute(ctx, task, opts)
		if err != nil {
			return nil, err
		}

		lastResult = result
		if result.Success {
			return result, nil
		}

		// Check if error is retryable
		if !isRetryable(result.Error) {
			return result, nil
		}
	}

	return lastResult, nil
}

// selectProvider chooses the best provider for a task
func (te *TaskExecutor) selectProvider(task *AnalyzedTask) Provider {
	// First, try to find a provider that explicitly handles this task type
	for _, p := range te.providers {
		if p.CanHandle(task) {
			return p
		}
	}

	// Fall back to defaults based on task type
	switch task.Type {
	case TaskTypeBugFix, TaskTypeFeature, TaskTypeRefactor, TaskTypeTest:
		if te.defaultCoding != nil {
			return te.defaultCoding
		}
	case TaskTypeDocs, TaskTypeResearch:
		if te.defaultSimple != nil {
			return te.defaultSimple
		}
	}

	// Last resort: return first available provider
	if len(te.providers) > 0 {
		return te.providers[0]
	}

	return nil
}

// ListProviders returns the names of all registered providers
func (te *TaskExecutor) ListProviders() []string {
	names := make([]string, len(te.providers))
	for i, p := range te.providers {
		names[i] = p.Name()
	}
	return names
}

// isRetryable checks if an error should trigger a retry
func isRetryable(errMsg string) bool {
	if errMsg == "" {
		return false
	}

	// Our own execution timeouts should NOT be retried —
	// the agent was given its full configured timeout (v0.8.1)
	if strings.HasPrefix(errMsg, "timeout after") {
		return false
	}

	// Rate limiting
	if contains(errMsg, "rate limit", "429", "too many requests") {
		return true
	}

	// Temporary network errors
	if contains(errMsg, "timeout", "connection", "network") {
		return true
	}

	// Server errors
	if contains(errMsg, "500", "502", "503", "504", "internal server error") {
		return true
	}

	return false
}

// contains checks if s contains any of the substrings
func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
