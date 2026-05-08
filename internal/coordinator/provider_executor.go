package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	// Import to trigger init() registration for all executor packages
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
	_ "github.com/sunholo-data/ailang/internal/executor/codex"
	_ "github.com/sunholo-data/ailang/internal/executor/gemini"
	_ "github.com/sunholo-data/ailang/internal/executor/motoko"
	_ "github.com/sunholo-data/ailang/internal/executor/opencode"
	_ "github.com/sunholo-data/ailang/internal/executor/pi"
)

// ExecutorProvider wraps any executor.Executor as a coordinator Provider.
// This is the unified provider for all CLI-based agentic executors (Claude Code,
// Gemini CLI, Codex, etc.). Adding a new executor only requires creating the
// executor package with an init() registration — no coordinator changes needed.
type ExecutorProvider struct {
	exec         executor.Executor
	executorName string // "claude", "gemini", "codex", etc.
}

// NewExecutorProvider creates an ExecutorProvider for the named executor.
// The executor must be registered in the global executor factory (via init()).
func NewExecutorProvider(executorName string) (*ExecutorProvider, error) {
	exec, err := executor.GlobalFactory().GetExecutor(executorName)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s executor: %w", executorName, err)
	}

	return &ExecutorProvider{
		exec:         exec,
		executorName: executorName,
	}, nil
}

// Name returns the provider name (e.g., "claude-code", "gemini-cli").
func (p *ExecutorProvider) Name() string {
	return p.executorName + "-cli"
}

// ExecutorName returns the underlying executor name (e.g., "claude", "gemini").
func (p *ExecutorProvider) ExecutorName() string {
	return p.executorName
}

// CanHandle returns true — executor-based providers can handle any coding task.
func (p *ExecutorProvider) CanHandle(task *AnalyzedTask) bool {
	return true
}

// Execute runs a task using the underlying executor CLI.
func (p *ExecutorProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
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
		result.Output = fmt.Sprintf("DRY RUN: Would execute %s with directive:\n%s", p.executorName, directive)
		result.Duration = time.Since(start)
		return result, nil
	}

	// Build system prompt from meta-prompt + task type + agent config
	systemPrompt := BuildSystemPrompt(task.Type, opts.AgentConfig)

	// Create executor task
	execTask := &executor.Task{
		ID:              task.Task.ID,
		ParentTaskID:    task.Task.ParentTaskID, // M-TASK-HIERARCHY: propagate from coordinator task
		Directive:       directive,
		SystemPrompt:    systemPrompt,
		Workspace:       opts.Workspace,
		Timeout:         opts.Timeout,
		IdleTimeout:     opts.IdleTimeout,
		Model:           opts.Model,
		Effort:          opts.Effort,
		PluginDirs:      opts.PluginDirs,                    // M-CLOUD-PLUGIN-SKILLS: pass plugin dirs to executor
		Plugins:         convertPluginsConfig(opts.Plugins), // M-CLOUD-PLUGIN-SKILLS: third-party plugins
		Metadata:        make(map[string]string),
		Iteration:       task.Task.Iteration, // M-TRANSCRIPT: feedback loop iteration
		ResumeSessionID: task.Task.SessionID, // M-TRANSCRIPT: resume session if iteration > 1
	}

	// Pass Observatory context for trace linking (M-TASK-HIERARCHY)
	if opts.ObservatoryContext != nil {
		execTask.Metadata["ailang.task_id"] = opts.ObservatoryContext.TaskID
		execTask.Metadata["ailang.agent_id"] = opts.ObservatoryContext.AgentID
		execTask.Metadata["ailang.assignment_id"] = opts.ObservatoryContext.AssignmentID
		execTask.Metadata["ailang.workspace_id"] = opts.ObservatoryContext.WorkspaceID

		// Chain context for unified hierarchy (M-CHAINS-SIMPLIFY)
		if opts.ObservatoryContext.ChainID != "" {
			execTask.Metadata["ailang.chain_id"] = opts.ObservatoryContext.ChainID
			execTask.Metadata["chain_id"] = opts.ObservatoryContext.ChainID
		}
		if opts.ObservatoryContext.StageID != "" {
			execTask.Metadata["ailang.stage_id"] = opts.ObservatoryContext.StageID
			execTask.Metadata["stage_id"] = opts.ObservatoryContext.StageID
		}
		if opts.ObservatoryContext.MessageID != "" {
			execTask.Metadata["ailang.message_id"] = opts.ObservatoryContext.MessageID
			execTask.Metadata["message_id"] = opts.ObservatoryContext.MessageID
		}
	}

	// For questions, use read-only tools (no file modifications)
	if task.Task.Kind == "question" {
		execTask.AllowedTools = []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"}
	}

	// Execute using the CLI executor — use streaming if handler provided
	var execResult *executor.Result
	var err error
	if opts.EventHandler != nil {
		execResult, err = p.exec.ExecuteStreaming(ctx, execTask, opts.EventHandler)
	} else {
		execResult, err = p.exec.Execute(ctx, execTask)
	}
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("%s execution error: %v", p.executorName, err)
		return result, nil
	}

	result.Success = execResult.Success
	result.Output = execResult.Output
	result.Error = execResult.Error
	result.Cost = execResult.CostUSD
	result.InputTokens = execResult.InputTokens
	result.OutputTokens = execResult.OutputTokens
	result.TokensUsed = execResult.InputTokens + execResult.OutputTokens
	result.NumTurns = execResult.NumTurns
	result.ToolCallCount = execResult.ToolCallCount
	result.FilesCreated = execResult.FilesCreated
	result.FilesModified = execResult.FilesModified
	result.SessionID = execResult.SessionID // For agent-to-agent handoffs

	return result, nil
}

// convertPluginsConfig converts coordinator.PluginsConfig to executor.PluginsConfig.
// Returns nil if input is nil.
func convertPluginsConfig(pc *PluginsConfig) *executor.PluginsConfig {
	if pc == nil {
		return nil
	}
	return &executor.PluginsConfig{
		Marketplaces: pc.Marketplaces,
		Install:      pc.Install,
	}
}

// buildDirective returns the directive for the executor.
// The directive is already fully constructed by BuildDirectiveFromConfig()
// in stage_execution.go (skill invocation, output markers, etc.).
// Task type context and commit instructions are now in the system prompt
// (see meta_prompt.go:BuildSystemPrompt) to avoid burying skill invocations.
func buildDirective(task *AnalyzedTask) string {
	return task.Task.Content
}
