// agent_runner_multi.go - Multi-executor support for agent benchmarks
// Enables running benchmarks with different AI coding agents (Claude, Gemini, etc.)

package eval_harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"

	// Register executors via init()
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
)

// MultiExecutorConfig extends AgentBenchmarkConfig with executor selection
type MultiExecutorConfig struct {
	AgentBenchmarkConfig

	// ExecutorName specifies which executor to use (e.g., "claude", "gemini")
	// If empty, uses the model's agent_cli from models.yml
	ExecutorName string

	// ModelName is the model to use (e.g., "claude-sonnet-4-5", "gemini-3-flash")
	ModelName string
}

// RunAgentBenchmarkWithExecutor runs a benchmark using the specified executor
// This enables comparing performance across different AI coding agents
func RunAgentBenchmarkWithExecutor(spec *BenchmarkSpec, config MultiExecutorConfig, language string) (*AgentBenchmarkResult, error) {
	if language == "" {
		language = "ailang"
	}

	// Determine which executor to use
	executorName := config.ExecutorName
	modelName := config.ModelName // Use provided model name (e.g., "gemini-3-flash-preview")

	// If executor not provided, look up from model config
	if executorName == "" && GlobalModelsConfig != nil {
		var err error
		executorName, modelName, err = GlobalModelsConfig.GetExecutorForModel(config.ModelName)
		if err != nil {
			return nil, fmt.Errorf("failed to get executor for model %s: %w", config.ModelName, err)
		}
	}

	if executorName == "" {
		executorName = "claude" // Default fallback
	}

	// Get the executor from factory
	exec, err := executor.GlobalFactory().GetExecutor(executorName)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor %s: %w", executorName, err)
	}

	// Health check
	ctx := context.Background()
	if err := exec.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("executor %s health check failed: %w", executorName, err)
	}

	// Create isolated workspace
	workspace := filepath.Join(config.WorkspaceDir, fmt.Sprintf("%s_%s_%s_%d",
		spec.ID, language, executorName, os.Getpid()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create minimal .git folder
	gitDir := filepath.Join(workspace, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .git folder: %w", err)
	}

	// Only cleanup workspace if DEBUG_AGENT is not set
	if os.Getenv("DEBUG_AGENT") == "" {
		defer os.RemoveAll(workspace)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace preserved: %s\n", workspace)
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Executor: %s, Model: %s\n", executorName, modelName)
	}

	// Create solution file placeholder
	var solutionPath string
	var placeholder string

	if language == "ailang" {
		benchmarkDir := filepath.Join(workspace, "benchmark")
		if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create benchmark dir: %w", err)
		}
		solutionPath = filepath.Join(benchmarkDir, "solution.ail")
		placeholder = `module benchmark/solution
// DO NOT CHANGE THE MODULE DECLARATION ABOVE!
// TODO: Add your solution code below

`
	} else {
		solutionPath = filepath.Join(workspace, "solution.py")
		placeholder = "# TODO: Write your Python solution here\n"
	}

	// Generate prompts
	systemPrompt, taskPrompt, promptVersion, err := GenerateAgentPromptsWithSystemPrompt(spec, config.AgentBenchmarkConfig, language, "", solutionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompts: %w", err)
	}

	if err := os.WriteFile(solutionPath, []byte(placeholder), 0644); err != nil {
		return nil, fmt.Errorf("failed to create solution placeholder: %w", err)
	}

	// Set up timeout - use spec timeout if set, otherwise use config default
	timeoutSeconds := config.TimeoutSeconds
	if spec.Timeout > 0 {
		timeoutSeconds = spec.Timeout
	}

	// Build task for executor
	task := &executor.Task{
		ID:           fmt.Sprintf("%s_%s", spec.ID, uuid.New().String()[:8]),
		Directive:    taskPrompt,
		SystemPrompt: systemPrompt,
		Workspace:    workspace,
		Timeout:      time.Duration(timeoutSeconds) * time.Second,
		Model:        modelName,
	}

	// Execute with streaming
	result, err := exec.ExecuteStreaming(ctx, task, &debugEventHandler{})
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Read solution code
	solutionCode, err := os.ReadFile(solutionPath)
	if err != nil {
		solutionCode = []byte{}
	}

	// Validate solution
	validation := validateSolution(result, spec, workspace, language, solutionPath)

	// Calculate success
	success := validation.CompileOk && validation.RuntimeOk && validation.StdoutOk

	return &AgentBenchmarkResult{
		BenchmarkID:   spec.ID,
		Executor:      executorName,
		Success:       success,
		Iterations:    result.NumTurns,
		Cost:          result.CostUSD,
		DurationMS:    result.DurationMS,
		NumTurns:      result.NumTurns,
		Error:         result.Error,
		SessionID:     result.SessionID,
		Result:        result.Output,
		Usage:         TokenUsage{InputTokens: result.InputTokens, OutputTokens: result.OutputTokens},
		SolutionCode:  string(solutionCode),
		SessionLog:    result.Transcript,
		PromptVersion: promptVersion,
		CompileOk:     validation.CompileOk,
		RuntimeOk:     validation.RuntimeOk,
		StdoutOk:      validation.StdoutOk,
		Stdout:        validation.Stdout,
		Stderr:        validation.Stderr,
	}, nil
}

// validateSolution checks if the solution is correct
func validateSolution(result *executor.Result, spec *BenchmarkSpec, workspace, language, solutionPath string) ValidationResult {
	if !result.Success {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    result.Error,
		}
	}

	// Check if solution file exists
	solutionContent, err := os.ReadFile(solutionPath)
	if err != nil || len(solutionContent) == 0 {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    fmt.Sprintf("Solution file not found or empty: %v", err),
		}
	}

	// Run solution based on language
	if language == "python" {
		return runPythonSolution(solutionPath, spec)
	}

	return runAILANGSolution(string(solutionContent), spec)
}

// runPythonSolution executes and validates a Python solution
func runPythonSolution(solutionPath string, spec *BenchmarkSpec) ValidationResult {
	cmd := exec.Command("python3", solutionPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ValidationResult{
			CompileOk: true,
			RuntimeOk: false,
			StdoutOk:  false,
			Stdout:    string(output),
			Stderr:    fmt.Sprintf("Python execution failed: %v", err),
		}
	}
	stdoutOk := CompareOutput(spec.ExpectedOut, string(output))
	return ValidationResult{
		CompileOk: true,
		RuntimeOk: true,
		StdoutOk:  stdoutOk,
		Stdout:    string(output),
	}
}

// runAILANGSolution executes and validates an AILANG solution
func runAILANGSolution(solutionCode string, spec *BenchmarkSpec) ValidationResult {
	runner := NewAILANGRunner("", spec.Caps)
	runResult, err := runner.Run(solutionCode, 10*time.Second)
	if err != nil {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    err.Error(),
		}
	}

	stdoutOk := runResult.RuntimeOk && CompareOutput(spec.ExpectedOut, runResult.Stdout)
	return ValidationResult{
		CompileOk: runResult.CompileOk,
		RuntimeOk: runResult.RuntimeOk,
		StdoutOk:  stdoutOk,
		Stdout:    runResult.Stdout,
		Stderr:    runResult.Stderr,
	}
}

// debugEventHandler prints streaming events when DEBUG_AGENT is set
type debugEventHandler struct{}

func (h *debugEventHandler) OnTurnStart(turnNum int) {
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "\n[TURN %d] ==========================================\n", turnNum)
	}
}

func (h *debugEventHandler) OnText(text string) {
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprint(os.Stderr, text)
	}
}

func (h *debugEventHandler) OnToolUse(toolName string, input string) {
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[TOOL] %s\n", toolName)
	}
}

func (h *debugEventHandler) OnToolResult(toolName string, output string) {
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[RESULT] %s\n", toolName)
	}
}

func (h *debugEventHandler) OnTurnEnd(turnNum int) {
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "\n[END TURN %d]\n", turnNum)
	}
}

func (h *debugEventHandler) OnError(err error) {
	fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
}
