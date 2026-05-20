// agent_runner_multi.go - Multi-executor support for agent benchmarks
// Enables running benchmarks with different AI coding agents (Claude, Gemini, etc.)

package eval_harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/eval_harness/langreg"
	"github.com/sunholo-data/ailang/internal/executor"

	// Register executors via init()
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
)

// MultiExecutorConfig extends AgentBenchmarkConfig with executor selection
type MultiExecutorConfig struct {
	AgentBenchmarkConfig

	// ExecutorName specifies which executor to use (e.g., "claude", "codex")
	// If empty, uses the model's agent_cli from models.yml
	ExecutorName string

	// ModelName is the model to use (e.g., "claude-sonnet-4-5", "gemini-3-flash")
	ModelName string

	// ConfigKey is the models.yml lookup key for per-model config (e.g., "opencode-gemma4-e4b").
	// When set, overrides ModelName for timeout/config lookups. Needed when ModelName is the
	// resolved API model name (e.g., "ollama/gemma4:e4b") rather than the models.yml key.
	ConfigKey string

	// ExtraHandler is an additional event handler composed with the debug handler.
	// Used for ObservatoryWriter to capture structured tool calls during streaming.
	// When nil, only the debug handler is used (no behavior change).
	ExtraHandler executor.EventHandler
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
		return nil, fmt.Errorf("no executor configured for model %q: "+
			"add an executor mapping in models.yml or specify --executor explicitly", config.ModelName)
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
		lang, langErr := langreg.Get(language)
		if langErr != nil {
			return nil, fmt.Errorf("unknown language %q: %w", language, langErr)
		}
		solutionPath = filepath.Join(workspace, lang.SolutionFilename())
		placeholder = "# TODO: Write your solution here\n"
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

	// Apply per-model TTFT / generation timeouts from models.yml
	lookupKey := config.ModelName
	if config.ConfigKey != "" {
		lookupKey = config.ConfigKey
	}
	if GlobalModelsConfig != nil {
		if cfg, ok := GlobalModelsConfig.Models[lookupKey]; ok {
			if cfg.TTFTTimeoutSeconds > 0 {
				task.TTFTTimeout = time.Duration(cfg.TTFTTimeoutSeconds) * time.Second
			}
			if cfg.GenerationTimeoutSeconds > 0 {
				task.IdleTimeout = time.Duration(cfg.GenerationTimeoutSeconds) * time.Second
			}
			if cfg.GCPProject != "" {
				task.GCPProject = cfg.GCPProject
			}
			if cfg.GCPLocation != "" {
				task.GCPLocation = cfg.GCPLocation
			}
			// M-EVAL-COST-AND-SPEED-BUDGETS (v0.16.0): populate Task.Budget
			// from the resolved per-model cost ceiling. Wall-clock Timeout
			// is bumped to the budgets:hard_timeout_secs default (600s) to
			// give cost the primary gate; cost.go does the kill-on-exceed.
			if maxCost := cfg.ResolvedMaxCostUSD(); maxCost > 0 {
				task.Budget = executor.NewCostBudget(maxCost, cfg.Pricing.InputPer1K, cfg.Pricing.OutputPer1K)
			}
			// Per-benchmark spec.Timeout (e.g. csv_to_json's 180s override) wins.
			// Otherwise per-model budgets:hard_timeout_secs takes precedence over
			// the CLI agent-timeout default (typically 60s) — the whole point of
			// this milestone is that wall-clock is a safety net, not a cost proxy.
			if spec.Timeout == 0 {
				if hardSecs := cfg.ResolvedHardTimeoutSecs(); hardSecs > 0 {
					task.Timeout = time.Duration(hardSecs) * time.Second
				}
			}
		}
	}

	// Wrap handler to capture TTFT (time from task start to first output event)
	ttftTracker := &ttftEventHandler{start: time.Now()}
	var handler executor.EventHandler = &compositeEventHandler{
		primary:   &debugEventHandler{},
		secondary: ttftTracker,
	}
	if config.ExtraHandler != nil {
		handler = &compositeEventHandler{
			primary:   handler,
			secondary: config.ExtraHandler,
		}
	}

	// Execute with streaming
	result, err := exec.ExecuteStreaming(ctx, task, handler)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Link observatory session to chain stage if handler supports it
	if linker, ok := config.ExtraHandler.(interface{ LinkToStage() }); ok {
		linker.LinkToStage()
	}

	// Check for executor-level failure (crash, timeout, non-zero exit).
	// Executors return (Result{Success:false}, nil) on these failures --
	// the error is in Result.Error, NOT the Go error return value.
	// We check this BEFORE agentic validation to provide clear "executor crashed"
	// errors instead of misleading "non-agentic result" messages.
	if !result.Success && result.Error != "" {
		return nil, fmt.Errorf("executor %q failed for model %q: %s",
			executorName, modelName, result.Error)
	}

	// Validate agent behavior - NO SILENT FALLBACKS
	// Agent mode must produce multi-turn agentic behavior, not 0-shot text generation
	if result.NumTurns <= 1 && result.ToolCallCount == 0 {
		return nil, fmt.Errorf("executor %q produced non-agentic result: "+
			"%d turns, %d tool calls. This looks like 0-shot generation, not agent mode. "+
			"Check that the CLI is configured for agentic coding (tool use, file editing). "+
			"Model: %s", executorName, result.NumTurns, result.ToolCallCount, modelName)
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
		ToolCallCount: result.ToolCallCount,
		Error:         result.Error,
		SessionID:     result.SessionID,
		Result:        result.Output,
		Usage:         TokenUsage{InputTokens: result.InputTokens, OutputTokens: result.OutputTokens},
		TTFTSeconds:   ttftTracker.seconds,
		ModelFamily: func() string {
			if GlobalModelsConfig != nil {
				if cfg, ok := GlobalModelsConfig.Models[lookupKey]; ok {
					return cfg.ModelFamily
				}
			}
			return ""
		}(),
		SolutionCode:  string(solutionCode),
		SessionLog:    result.Transcript,
		PromptVersion: promptVersion,
		CompileOk:     validation.CompileOk,
		RuntimeOk:     validation.RuntimeOk,
		StdoutOk:      validation.StdoutOk,
		Stdout:        validation.Stdout,
		Stderr:        validation.Stderr,

		// M-EVAL-COST-AND-SPEED-BUDGETS (v0.15.1): propagate speed/cost metrics
		// from executor.Result so they land in BenchmarkResult JSON output.
		CostKilledAt:   result.CostKilledAt,
		FirstAttemptMs: result.FirstAttemptMs,
		SuccessAtMs:    result.SuccessAtMs,
		TokensPerSec:   result.TokensPerSec,

		// M-EVAL-SWEET-SPOT (v0.19.0): pass through the executor finish signal
		// so cmd/ailang/eval_benchmark.go can promote it via CategorizeAgentError.
		FinishReason: result.FinishReason,
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

	// Run solution based on language via the runner registry.
	if language == "python" {
		return runPythonSolution(solutionPath, spec)
	}
	if language == "ailang" {
		return runAILANGSolution(string(solutionContent), spec)
	}

	// JS, Go, and future languages: use GetRunnerWithContext.
	r, runErr := GetRunnerWithContext(context.Background(), language, spec, "")
	if runErr != nil {
		return ValidationResult{Stderr: fmt.Sprintf("no runner for %q: %v", language, runErr)}
	}
	runResult, runRunErr := r.Run(string(solutionContent), 30*time.Second)
	if runRunErr != nil {
		return ValidationResult{Stderr: fmt.Sprintf("validation runner error: %v", runRunErr)}
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

// runPythonSolution executes and validates a Python solution using the
// uv-managed pinned Python runtime.
//
// Wires spec.CliArgs into the subprocess invocation and spec.Stdin onto
// the subprocess's stdin. Without these, every benchmark whose Python
// solution reads sys.argv[1] or sys.stdin will fail at runtime even
// though the generated code is correct — see
// design_docs/planned/v0_21_0/m-eval-agent-python-stdio-wiring.md for the
// full context (v0.20.0 cli_args + pipeline benchmarks exposed this).
func runPythonSolution(solutionPath string, spec *BenchmarkSpec) ValidationResult {
	// newPythonCommand is variadic — pass solution path + spec's CLI args
	// in the same slice so they all land after `uv run --python 3.12 --`.
	args := append([]string{solutionPath}, spec.CliArgs...)
	cmd, uvErr := newPythonCommand(args...)
	if uvErr != nil {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    uvErr.Error(),
		}
	}
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
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
	// Pass spec to runner so input_files, cli_args, and stdin are available
	// in the validation workspace (not just in the agent workspace).
	runner := NewAILANGRunnerWithTask(context.Background(), "", spec.Caps, "", spec)
	runResult, err := runner.Run(solutionCode, 10*time.Second)
	if err != nil {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    fmt.Sprintf("validation runner error: %v", err),
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

// compositeEventHandler wraps two EventHandlers, calling both for each event.
type compositeEventHandler struct {
	primary   executor.EventHandler
	secondary executor.EventHandler
}

func (c *compositeEventHandler) OnTurnStart(turnNum int) {
	c.primary.OnTurnStart(turnNum)
	c.secondary.OnTurnStart(turnNum)
}

func (c *compositeEventHandler) OnText(text string) {
	c.primary.OnText(text)
	c.secondary.OnText(text)
}

func (c *compositeEventHandler) OnToolUse(toolName string, input string) {
	c.primary.OnToolUse(toolName, input)
	c.secondary.OnToolUse(toolName, input)
}

func (c *compositeEventHandler) OnToolResult(toolName string, output string) {
	c.primary.OnToolResult(toolName, output)
	c.secondary.OnToolResult(toolName, output)
}

func (c *compositeEventHandler) OnTurnEnd(turnNum int) {
	c.primary.OnTurnEnd(turnNum)
	c.secondary.OnTurnEnd(turnNum)
}

func (c *compositeEventHandler) OnError(err error) {
	c.primary.OnError(err)
	c.secondary.OnError(err)
}

// ttftEventHandler records the elapsed time until the first output event (text or tool use).
type ttftEventHandler struct {
	start   time.Time
	seconds float64
	once    sync.Once
}

func (h *ttftEventHandler) record() {
	h.once.Do(func() { h.seconds = time.Since(h.start).Seconds() })
}

func (h *ttftEventHandler) OnTurnStart(int)             {}
func (h *ttftEventHandler) OnText(string)               { h.record() }
func (h *ttftEventHandler) OnToolUse(string, string)    { h.record() }
func (h *ttftEventHandler) OnToolResult(string, string) {}
func (h *ttftEventHandler) OnTurnEnd(int)               {}
func (h *ttftEventHandler) OnError(error)               {}
