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
	_ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
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

	// M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP: chain_id and stage_id are populated by
	// cmd/ailang/eval_benchmark.go from the active execution chain. They flow into
	// executor.Task.Metadata which BuildResourceAttributes converts into
	// OTEL_RESOURCE_ATTRIBUTES. Without these, spans emitted by the opencode
	// subprocess can't be linked back to their stage in `ailang chains live`.
	// Empty strings are skipped (coordinator path uses Task.Metadata directly).
	ChainID string
	StageID string
}

// buildChainMetadata builds the executor.Task.Metadata map from the chain/stage
// IDs threaded through MultiExecutorConfig. Empty IDs are omitted (so coordinator-
// driven tasks that populate Task.Metadata directly are not affected).
//
// M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP: the returned map flows into
// BuildResourceAttributes which converts it to OTEL_RESOURCE_ATTRIBUTES.
// Always returns a non-nil map (even if both inputs are empty) so callers
// can append vendor-specific keys later without nil-checking.
func buildChainMetadata(chainID, stageID string) map[string]string {
	m := make(map[string]string, 2)
	if chainID != "" {
		m["chain_id"] = chainID
	}
	if stageID != "" {
		m["stage_id"] = stageID
	}
	return m
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

	// Seed benchmark input files so the agent can actually run/test its solution
	// (e.g. cli_args reads numbers.txt). Mirrors the standard-runner layout.
	if err := seedInputFiles(workspace, spec); err != nil {
		return nil, err
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

	// Detect executors that run the agent in an isolated sandbox without
	// shared filesystem (e.g. managed_agents → Vertex Managed Agents API).
	// For those, append the cross-environment bridge instruction so the
	// agent dumps its solution as a fenced code block in the response —
	// the only channel we can read post-run.
	remoteSandbox := executorHasCapability(exec, executor.CapRemoteSandbox)
	if remoteSandbox {
		systemPrompt += managedAgentsBridgeInstruction
	}

	// Build task for executor.
	//
	// M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP: populate Metadata with chain_id and
	// stage_id when the caller provided them. executor.BuildEnvironment converts
	// these into OTEL_RESOURCE_ATTRIBUTES so the opencode subprocess emits
	// spans tagged with ailang.chain_id / ailang.stage_id — which is what makes
	// `ailang chains live` per-stage join precise (without this, spans land
	// in observatory.db but with NULL chain_id/stage_id).
	task := &executor.Task{
		ID:           fmt.Sprintf("%s_%s", spec.ID, uuid.New().String()[:8]),
		Directive:    maybePrependTrapsCard(taskPrompt),
		SystemPrompt: systemPrompt,
		// Teaching-prompt delivery (M-EVAL-OPENCODE-SYSTEM-PROMPT). The
		// 2026-06-06 delivery experiment found turn-1 concatenation + a small
		// front-loaded traps card beats re-injecting the full prompt every turn
		// via AGENTS.md ("MOVE", which bloated context and scored 1/6 vs 3/6).
		// So persistence now defaults OFF; set AILANG_EVAL_PERSIST_PROMPT=1 to
		// re-enable (opencode → AGENTS.md; executors without a persistent
		// channel ignore it).
		PersistentSystemPrompt: persistentSystemPromptEnabled(),
		Workspace:              workspace,
		Timeout:                time.Duration(timeoutSeconds) * time.Second,
		Model:                  modelName,
		Metadata:               buildChainMetadata(config.ChainID, config.StageID),
		MaxTokensPerBench:      config.MaxTokensPerBench,        // M-EVAL-OS-LONGITUDINAL Phase 1
		MaxOutputTokens:        modelMaxOutputTokens(modelName), // M-OLLAMA-PER-MODEL-MAX-TOKENS
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
			// M-OLLAMA-PER-MODEL-MAX-TOKENS (fix 2026-06-19): resolve
			// max_output_tokens via the SAME registry key as TTFT (lookupKey =
			// ConfigKey) rather than `modelName` (the agent display name, e.g.
			// "ollama/qwen3.6:..."), which is NOT a registry key — GetModel(modelName)
			// missed → returned 0, so the per-model 32768 budget never reached
			// motoko and the wire stayed at the 16384 floor.
			if cfg.MaxOutputTokens > 0 {
				task.MaxOutputTokens = cfg.MaxOutputTokens
			}
			if cfg.GCPProject != "" {
				task.GCPProject = cfg.GCPProject
			}
			if cfg.GCPLocation != "" {
				task.GCPLocation = cfg.GCPLocation
			}
			if cfg.MotokoProfile != "" {
				if task.Metadata == nil {
					task.Metadata = make(map[string]string)
				}
				task.Metadata["motoko_profile"] = cfg.MotokoProfile
			}
			// M-EVAL-COST-AND-SPEED-BUDGETS (v0.16.0): populate Task.Budget
			// from the resolved per-model cost ceiling. Wall-clock Timeout
			// is bumped to the budgets:hard_timeout_secs default (600s) to
			// give cost the primary gate; cost.go does the kill-on-exceed.
			if maxCost := cfg.ResolvedMaxCostUSD(); maxCost > 0 {
				task.Budget = executor.NewCostBudget(maxCost, cfg.Pricing.InputPer1K, cfg.Pricing.OutputPer1K)
			}
			// M-EVAL-LOCAL-OLLAMA (v0.22.0): take the MAX of spec.Timeout and
			// model.HardTimeoutSecs rather than letting spec.Timeout veto. The
			// benchmark spec timeout is cloud-tuned (Sonnet 4.6 speeds); local
			// Ollama models need their own slower budget to apply. For cloud
			// models with cost budgets, the wall-clock bump is a no-op because
			// cost trips first; for local models (pricing=0, no cost gate) this
			// is the only gate, so it must reflect the model's actual speed.
			specT := spec.Timeout
			hardSecs := cfg.ResolvedHardTimeoutSecs()
			effective := specT
			if hardSecs > effective {
				effective = hardSecs
			}
			if effective > 0 {
				task.Timeout = time.Duration(effective) * time.Second
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

	// Cross-environment bridge: for CapRemoteSandbox executors, the agent's
	// file edits happened in an isolated sandbox we can't read. Extract the
	// solution code from result.Output (instructed via managedAgentsBridgeInstruction
	// above) and write it to solutionPath so validateSolution() can read it.
	if remoteSandbox && result != nil && result.Output != "" {
		if path, n, werr := writeSolutionFromResponse(workspace, result.Output); werr != nil {
			// Log but don't abort — validateSolution will report the failure
			// via stdout_ok=false if the file remains a placeholder.
			fmt.Fprintf(os.Stderr, "[managed_agents bridge] write failed: %v\n", werr)
		} else if path != "" && n > 0 {
			result.FilesModified = append(result.FilesModified, path)
			// Count the file write as one tool call so the harness's
			// agentic-result gate (NumTurns>1 || ToolCallCount>0) accepts
			// the run. This is the eval harness's policy, not the
			// executor's — see managed_agents_bridge.go.
			if result.ToolCallCount == 0 {
				result.ToolCallCount = 1
			}
		}
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

		// M-AILANG-SEMANTIC-CONTEXT (v0.26.0): context-compaction telemetry.
		CompactionCount:     result.CompactionCount,
		CompactionFirstStep: result.CompactionFirstStep,
		CompactionMaxLevel:  result.CompactionMaxLevel,
	}, nil
}

// validateSolution checks if the solution is correct
// seedInputFiles writes spec.InputFiles into the agent workspace root, mirroring
// the layout the standard runners use (runner.go:138/320). Without this, an agent
// working on a benchmark that reads a data file (e.g. cli_args reads numbers.txt)
// cannot test its own solution — the file does not exist in its workspace — so it
// submits blind and the task looks far harder in agent mode than in standard mode.
// Files land at the workspace root, which is the agent's cwd AND the cwd the
// AILANG/Python validators run from, so relative reads resolve identically during
// the agent's iteration and during grading.
func seedInputFiles(workspace string, spec *BenchmarkSpec) error {
	if spec == nil {
		return nil
	}
	for name, content := range spec.InputFiles {
		fpath := filepath.Join(workspace, name)
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("failed to create dir for input file %s: %w", name, err)
		}
		if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write input file %s: %w", name, err)
		}
	}
	return nil
}

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
	// Run from the solution's directory and seed input files there so relative
	// cli_args / input_files (e.g. cli_args reads numbers.txt) resolve the same
	// way they do for the agent and the standard runner. newPythonCommand sets no
	// cwd, so without cmd.Dir the args would resolve against the harness process
	// cwd (the repo root) and a correct solution would fail validation.
	workDir := filepath.Dir(solutionPath)
	if err := seedInputFiles(workDir, spec); err != nil {
		return ValidationResult{Stderr: err.Error()}
	}

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
	cmd.Dir = workDir
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

// trapsCardDefaultPath is the built-in location of the dialect-traps card,
// loaded relative to the eval working directory (repo root) — the same
// convention agent_prompt.txt uses. A package var (not const) so tests can
// redirect it at a temp file.
var trapsCardDefaultPath = "prompts/agent/dialect-traps.md"

// maybePrependTrapsCard front-loads the compact "dialect traps" card into the
// turn-1 task message — a tiny, un-buryable reminder of the highest-frequency
// rule violations, distilled from the 14-failure analysis.
//
// Default ON. The 2026-06-06 prompt-delivery experiment (local qwen3.5, n=2)
// showed the card sharply cuts flailing (symbolic_diff 1/2→2/2, 880k→246k
// tokens) by front-loading the import/syntax rules the model otherwise misses.
// It loads trapsCardDefaultPath unless AILANG_EVAL_TRAPS_CARD overrides the
// path; set AILANG_EVAL_TRAPS_CARD=off (or 0/false/no/none) to disable. If the
// card file is unreadable the directive is returned unchanged — this is an
// additive salience aid, not a data-integrity path.
func maybePrependTrapsCard(directive string) string {
	path := strings.TrimSpace(os.Getenv("AILANG_EVAL_TRAPS_CARD"))
	switch strings.ToLower(path) {
	case "off", "0", "false", "no", "none":
		return directive // explicitly disabled
	case "":
		path = trapsCardDefaultPath // default on
	}
	card, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[eval] traps card unreadable (%s): %v — continuing without card\n", path, err)
		return directive
	}
	return strings.TrimRight(string(card), "\n") + "\n\n---\n\n" + directive
}

// persistentSystemPromptEnabled reports whether the FULL teaching prompt should
// be delivered via a persistent system-prompt channel (opencode AGENTS.md),
// re-injected every turn, instead of concatenated once into the first user
// message.
//
// Defaults to FALSE. The 2026-06-05/06 prompt-delivery experiment (local
// qwen3.5, n=2) showed re-injecting the full ~22k prompt every turn ("MOVE")
// was the WORST delivery — 1/6 vs 3/6 for turn-1 concatenation — because it
// bloated the context (up to 39 turns / 2.4M tokens) and the model lost the
// signal. Set AILANG_EVAL_PERSIST_PROMPT=1/true/on to re-enable for A/B testing.
func persistentSystemPromptEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AILANG_EVAL_PERSIST_PROMPT"))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// modelMaxOutputTokens returns the registry's declared max_output_tokens for a
// model (its per-request output strength), or 0 if unknown. Forwarded on the Task
// to executors that drive a separate runtime so a reasoning model isn't truncated
// mid-<think> by a small default (M-OLLAMA-PER-MODEL-MAX-TOKENS).
func modelMaxOutputTokens(modelName string) int {
	if GlobalModelsConfig == nil {
		return 0
	}
	if m, err := GlobalModelsConfig.GetModel(modelName); err == nil {
		return m.MaxOutputTokens
	}
	return 0
}
