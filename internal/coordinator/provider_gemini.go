package coordinator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/ai"
	"github.com/sunholo/ailang/internal/ai/gemini"
	"github.com/sunholo/ailang/internal/executor"
	// Import to trigger init() registration
	_ "github.com/sunholo/ailang/internal/executor/gemini"
)

// GeminiCLIProvider executes tasks using Gemini CLI (agentic coding).
// This uses internal/executor/gemini which wraps the `gemini` CLI tool.
type GeminiCLIProvider struct {
	exec executor.Executor
}

// NewGeminiCLIProvider creates a new Gemini CLI provider
func NewGeminiCLIProvider() (*GeminiCLIProvider, error) {
	exec, err := executor.GlobalFactory().GetExecutor("gemini")
	if err != nil {
		return nil, fmt.Errorf("failed to get gemini executor: %w", err)
	}

	return &GeminiCLIProvider{
		exec: exec,
	}, nil
}

// Name returns the provider name
func (p *GeminiCLIProvider) Name() string {
	return "gemini-cli"
}

// CanHandle returns true for tasks that Gemini CLI can handle
func (p *GeminiCLIProvider) CanHandle(task *AnalyzedTask) bool {
	// Gemini CLI is good for coding tasks
	switch task.Type {
	case TaskTypeBugFix, TaskTypeFeature, TaskTypeRefactor, TaskTypeTest:
		return true
	default:
		return true
	}
}

// Execute runs a task using Gemini CLI
func (p *GeminiCLIProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	result := &ExecuteResult{
		Provider: p.Name(),
	}

	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	directive := buildDirective(task)

	if opts.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("DRY RUN: Would execute Gemini CLI with directive:\n%s", directive)
		result.Duration = time.Since(start)
		return result, nil
	}

	execTask := &executor.Task{
		ID:        task.Task.ID,
		Directive: directive,
		Workspace: opts.Workspace,
		Timeout:   opts.Timeout,
		Model:     opts.Model,
	}

	execResult, err := p.exec.Execute(ctx, execTask)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Gemini CLI execution error: %v", err)
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

// GeminiAPIProvider executes tasks using Gemini API (text generation only).
// This uses internal/ai/gemini for simple research and documentation tasks.
type GeminiAPIProvider struct {
	client ai.Provider
	model  string
}

// NewGeminiAPIProvider creates a new Gemini API provider
func NewGeminiAPIProvider() (*GeminiAPIProvider, error) {
	// Try Vertex AI first (ADC), fall back to API key
	client, err := gemini.NewVertexAIClient("")
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiAPIProvider{
		client: client,
		model:  "gemini-2.0-flash", // Fast model for quick responses
	}, nil
}

// Name returns the provider name
func (p *GeminiAPIProvider) Name() string {
	return "gemini-api"
}

// CanHandle returns true for tasks that Gemini API can handle
func (p *GeminiAPIProvider) CanHandle(task *AnalyzedTask) bool {
	// API is best for research and docs (no file editing needed)
	switch task.Type {
	case TaskTypeDocs, TaskTypeResearch:
		return true
	default:
		return false // Prefer CLI for coding tasks
	}
}

// Execute runs a task using Gemini API
func (p *GeminiAPIProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	result := &ExecuteResult{
		Provider: p.Name(),
	}

	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	prompt := buildAPIPrompt(task)

	if opts.DryRun {
		result.Success = true
		result.Output = fmt.Sprintf("DRY RUN: Would call Gemini API with prompt:\n%s", prompt)
		result.Duration = time.Since(start)
		return result, nil
	}

	// Create context with timeout
	execCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}

	req := &ai.Request{
		Model:      model,
		UserPrompt: prompt,
		MaxTokens:  4096,
	}
	response, err := p.client.Generate(execCtx, req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Gemini API error: %v", err)
		return result, nil
	}

	result.Success = true
	result.Output = response.Text
	result.TokensUsed = response.TotalTokens

	return result, nil
}

// buildAPIPrompt creates a prompt for text generation tasks
func buildAPIPrompt(task *AnalyzedTask) string {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant helping with software development tasks.\n\n")
	sb.WriteString("Task:\n")
	sb.WriteString(task.Task.Content)
	sb.WriteString("\n\n")

	switch task.Type {
	case TaskTypeDocs:
		sb.WriteString("This is a documentation task. Please:\n")
		sb.WriteString("1. Write clear, concise documentation\n")
		sb.WriteString("2. Include code examples where helpful\n")
		sb.WriteString("3. Follow markdown formatting\n")
	case TaskTypeResearch:
		sb.WriteString("This is a research task. Please:\n")
		sb.WriteString("1. Analyze the problem thoroughly\n")
		sb.WriteString("2. Present multiple approaches if applicable\n")
		sb.WriteString("3. Provide recommendations with rationale\n")
	default:
		sb.WriteString("Please help with this task and provide a clear solution.\n")
	}

	return sb.String()
}
