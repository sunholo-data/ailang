package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo/ailang/internal/executor"
	// Import to trigger init() registration
	_ "github.com/sunholo/ailang/internal/executor/claude"
)

// ClaudeCodeProvider executes tasks using Claude Code CLI (headless mode).
// This uses internal/executor/claude which wraps the `claude` CLI tool.
type ClaudeCodeProvider struct {
	exec executor.Executor
}

// NewClaudeCodeProvider creates a new Claude Code provider
func NewClaudeCodeProvider() (*ClaudeCodeProvider, error) {
	exec, err := executor.GlobalFactory().GetExecutor("claude")
	if err != nil {
		return nil, fmt.Errorf("failed to get claude executor: %w", err)
	}

	return &ClaudeCodeProvider{
		exec: exec,
	}, nil
}

// Name returns the provider name
func (p *ClaudeCodeProvider) Name() string {
	return "claude-code"
}

// CanHandle returns true for tasks that Claude Code can handle well
func (p *ClaudeCodeProvider) CanHandle(task *AnalyzedTask) bool {
	// Claude Code is best for coding tasks that need file editing
	switch task.Type {
	case TaskTypeBugFix, TaskTypeFeature, TaskTypeRefactor, TaskTypeTest:
		return true
	default:
		// Can handle anything as a fallback
		return true
	}
}

// Execute runs a task using Claude Code CLI
func (p *ClaudeCodeProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	result := &ExecuteResult{
		Provider: p.Name(),
	}

	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Build directive from task
	directive := buildDirective(task)

	if opts.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("DRY RUN: Would execute Claude Code with directive:\n%s", directive)
		result.Duration = time.Since(start)
		return result, nil
	}

	// Create executor task
	execTask := &executor.Task{
		ID:        task.Task.ID,
		Directive: directive,
		Workspace: opts.Workspace,
		Timeout:   opts.Timeout,
		Model:     opts.Model,
	}

	// Execute using Claude Code CLI
	execResult, err := p.exec.Execute(ctx, execTask)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Claude Code execution error: %v", err)
		return result, nil
	}

	result.Success = execResult.Success
	result.Output = execResult.Output
	result.Error = execResult.Error
	result.Cost = execResult.CostUSD
	result.InputTokens = execResult.InputTokens
	result.OutputTokens = execResult.OutputTokens
	result.TokensUsed = execResult.InputTokens + execResult.OutputTokens
	result.FilesCreated = execResult.FilesCreated
	result.FilesModified = execResult.FilesModified

	return result, nil
}

// buildDirective creates a directive string for the executor
func buildDirective(task *AnalyzedTask) string {
	// Start with the task content
	directive := task.Task.Content

	// Add type-specific context
	switch task.Type {
	case TaskTypeBugFix:
		directive = fmt.Sprintf("BUG FIX REQUEST:\n\n%s\n\nPlease fix this bug. Make sure to:\n1. Identify the root cause\n2. Implement the fix\n3. Add or update tests\n4. Run the test suite to verify", directive)
	case TaskTypeFeature:
		directive = fmt.Sprintf("FEATURE REQUEST:\n\n%s\n\nPlease implement this feature. Make sure to:\n1. Follow existing code patterns\n2. Add comprehensive tests\n3. Update documentation if needed\n4. Run the test suite to verify", directive)
	case TaskTypeRefactor:
		directive = fmt.Sprintf("REFACTORING REQUEST:\n\n%s\n\nPlease refactor the code. Make sure to:\n1. Maintain existing behavior\n2. Keep tests passing\n3. Improve code quality", directive)
	case TaskTypeTest:
		directive = fmt.Sprintf("TESTING REQUEST:\n\n%s\n\nPlease add or improve tests. Make sure to:\n1. Cover edge cases\n2. Use existing test patterns\n3. Verify all tests pass", directive)
	}

	return directive
}
