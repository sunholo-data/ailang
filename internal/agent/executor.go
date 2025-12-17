package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/eval_harness"
	"github.com/sunholo/ailang/internal/executor"

	// Register executors via init()
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
)

// DirectiveExecutor executes user directives using Claude Code
type DirectiveExecutor struct {
	config        eval_harness.AgentBenchmarkConfig
	workspaceBase string
}

// NewDirectiveExecutor creates a new executor with default configuration
func NewDirectiveExecutor(workspaceBase string) *DirectiveExecutor {
	config := eval_harness.DefaultAgentConfig()

	// Override defaults for agent execution
	config.TimeoutSeconds = 300 // 5 minutes for directives
	config.WorkspaceDir = workspaceBase
	config.ClaudeModel = "haiku" // Fast and cost-effective for most directives

	return &DirectiveExecutor{
		config:        config,
		workspaceBase: workspaceBase,
	}
}

// DirectiveResult contains the result of executing a directive
type DirectiveResult struct {
	Success      bool       // Overall success (completed without errors)
	DurationMS   int        // Execution time in milliseconds
	NumTurns     int        // Number of Claude turns
	Cost         float64    // Total cost in USD
	SessionID    string     // Claude session ID
	Transcript   string     // Full conversation transcript
	Output       string     // Final result text from Claude
	Error        string     // Error message if failed
	Workspace    string     // Path to workspace directory
	FilesCreated []string   // List of files created (relative to workspace)
	TokensUsed   TokenUsage // Token usage breakdown
}

// TokenUsage captures token metrics
type TokenUsage struct {
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
}

// Execute executes a directive using Claude Code in a fresh workspace
func (e *DirectiveExecutor) Execute(directive string) (*DirectiveResult, error) {
	return e.ExecuteInWorkspace(directive, "")
}

// ExecuteInWorkspace executes a directive in a specific workspace directory
// If workspacePath is empty, creates a fresh workspace
func (e *DirectiveExecutor) ExecuteInWorkspace(directive string, workspacePath string) (*DirectiveResult, error) {
	var workspace string
	var createdFreshWorkspace bool

	if workspacePath != "" {
		// Use specified workspace
		workspace = workspacePath
		// Verify it exists
		if _, err := os.Stat(workspace); os.IsNotExist(err) {
			return nil, fmt.Errorf("specified workspace does not exist: %s", workspace)
		}
	} else {
		// Create unique workspace for this execution
		workspace = filepath.Join(e.config.WorkspaceDir, fmt.Sprintf("directive_%s_%d",
			uuid.New().String()[:8], time.Now().Unix()))
		createdFreshWorkspace = true

		if err := os.MkdirAll(workspace, 0755); err != nil {
			return nil, fmt.Errorf("failed to create workspace: %w", err)
		}

		// Create minimal .git folder so Claude treats workspace as standalone project
		// This prevents Claude from walking up and finding the parent AILANG repo
		gitDir := filepath.Join(workspace, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create .git folder: %w", err)
		}
	}

	// Keep workspaces by default for inspection via UI
	// Set CLEANUP_AGENT_WORKSPACES=1 to auto-delete after execution
	// Only clean up fresh workspaces, not user-specified ones
	if createdFreshWorkspace && os.Getenv("CLEANUP_AGENT_WORKSPACES") == "1" {
		defer os.RemoveAll(workspace)
	}

	// Create minimal spec for eval harness
	spec := &eval_harness.BenchmarkSpec{
		ID:      "directive_" + uuid.New().String()[:8],
		Timeout: e.config.TimeoutSeconds,
	}

	// Execute directive via eval harness
	// No system prompt - directives are self-contained instructions
	// Use directive as task prompt
	result, err := eval_harness.RunHeadlessSessionStreaming(
		spec,
		"",        // systemPrompt (empty for directives)
		directive, // taskPrompt (the user's directive)
		workspace,
		e.config,
	)

	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// List files created in workspace
	filesCreated, err := listFiles(workspace)
	if err != nil {
		// Log warning but don't fail - file listing is informational
		filesCreated = []string{}
	}

	// Convert ClaudeHeadlessResult to DirectiveResult
	return &DirectiveResult{
		Success:      !result.IsError && result.Subtype == "success",
		DurationMS:   result.DurationMS,
		NumTurns:     result.NumTurns,
		Cost:         result.TotalCostUSD,
		SessionID:    result.SessionID,
		Transcript:   result.Transcript,
		Output:       result.Result,
		Error:        getErrorMessage(result),
		Workspace:    workspace,
		FilesCreated: filesCreated,
		TokensUsed: TokenUsage{
			InputTokens:              result.Usage.InputTokens,
			OutputTokens:             result.Usage.OutputTokens,
			CacheReadInputTokens:     result.Usage.CacheReadInputTokens,
			CacheCreationInputTokens: result.Usage.CacheCreationInputTokens,
		},
	}, nil
}

// ExecuteWithModel executes a directive using a specific Claude model
func (e *DirectiveExecutor) ExecuteWithModel(directive string, model string) (*DirectiveResult, error) {
	// Temporarily override model
	originalModel := e.config.ClaudeModel
	e.config.ClaudeModel = model
	defer func() {
		e.config.ClaudeModel = originalModel
	}()

	return e.Execute(directive)
}

// getErrorMessage extracts error message from Claude result
func getErrorMessage(result *eval_harness.ClaudeHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" || result.Subtype == "timeout" {
		return result.Result
	}
	return ""
}

// Maximum files to collect (prevents memory issues with large workspaces)
const maxFilesToCollect = 500

// Directories to skip when listing files (commonly huge)
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"coverage":     true,
}

// ExecuteWithExecutor executes a directive using the executor factory
// This uses the new multi-executor support (Gemini by default)
// Set AILANG_EXECUTOR=claude to use Claude Code instead
func (e *DirectiveExecutor) ExecuteWithExecutor(directive string, workspacePath string) (*DirectiveResult, error) {
	ctx := context.Background()

	// Get the default executor (Gemini 3 Flash by default, or AILANG_EXECUTOR env)
	exec, err := executor.GlobalFactory().GetDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}

	// Set up workspace
	workspace := workspacePath
	if workspace == "" {
		workspace = filepath.Join(e.workspaceBase, fmt.Sprintf("directive_%s_%d",
			uuid.New().String()[:8], time.Now().Unix()))

		if err := os.MkdirAll(workspace, 0755); err != nil {
			return nil, fmt.Errorf("failed to create workspace: %w", err)
		}

		// Create minimal .git folder
		gitDir := filepath.Join(workspace, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create .git folder: %w", err)
		}
	}

	// Execute using the executor interface
	task := &executor.Task{
		ID:        "directive_" + uuid.New().String()[:8],
		Directive: directive,
		Workspace: workspace,
		Timeout:   time.Duration(e.config.TimeoutSeconds) * time.Second,
	}

	result, err := exec.Execute(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// List files created in workspace
	filesCreated, err := listFiles(workspace)
	if err != nil {
		filesCreated = []string{}
	}

	return &DirectiveResult{
		Success:      result.Success,
		DurationMS:   result.DurationMS,
		NumTurns:     result.NumTurns,
		Cost:         result.CostUSD,
		SessionID:    result.SessionID,
		Transcript:   result.Transcript,
		Output:       result.Output,
		Error:        result.Error,
		Workspace:    workspace,
		FilesCreated: filesCreated,
		TokensUsed: TokenUsage{
			InputTokens:              result.InputTokens,
			OutputTokens:             result.OutputTokens,
			CacheReadInputTokens:     result.CacheReadInputTokens,
			CacheCreationInputTokens: result.CacheCreationInputTokens,
		},
	}, nil
}

// GetDefaultExecutorName returns the name of the default executor
func GetDefaultExecutorName() string {
	if envExec := os.Getenv("AILANG_EXECUTOR"); envExec != "" {
		return envExec
	}
	return "gemini" // Default to Gemini 3 Flash
}

// listFiles recursively lists files in a directory (relative paths), capped at maxFilesToCollect
func listFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Stop if we've collected enough files
		if len(files) >= maxFilesToCollect {
			return filepath.SkipAll
		}

		// Skip commonly large directories
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from workspace root
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	// filepath.SkipAll returns an error, but it's expected
	if err == filepath.SkipAll {
		err = nil
	}

	return files, err
}
