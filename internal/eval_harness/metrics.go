package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunMetrics captures the results of a single benchmark run
type RunMetrics struct {
	ID             string    `json:"id"`
	Lang           string    `json:"lang"`
	Model          string    `json:"model"`
	Executor       string    `json:"executor,omitempty"` // Executor used: "claude", "gemini", etc. (agent mode only)
	Seed           int64     `json:"seed"`
	InputTokens    int       `json:"input_tokens"`            // Prompt tokens (recorded but not primary metric)
	OutputTokens   int       `json:"output_tokens"`           // Generated code tokens (PRIMARY METRIC; reasoning excluded)
	ReasonTokens   int       `json:"reason_tokens,omitempty"` // Hidden reasoning/thinking tokens (billed as output)
	TotalTokens    int       `json:"total_tokens"`            // Total for billing (includes reasoning)
	CostUSD        float64   `json:"cost_usd"`
	CompileOk      bool      `json:"compile_ok"`
	RuntimeOk      bool      `json:"runtime_ok"`
	StdoutOk       bool      `json:"stdout_ok"`
	DurationMs     int64     `json:"duration_ms"`    // Total time (startup + compile + execution)
	CompileMs      int64     `json:"compile_ms"`     // Time spent in compilation (if separate)
	ExecuteMs      int64     `json:"execute_ms"`     // Time spent in execution (if measurable)
	ErrorCategory  string    `json:"error_category"` // compile_error | runtime_error | logic_error | none
	Stdout         string    `json:"stdout,omitempty"`
	Stderr         string    `json:"stderr,omitempty"`
	ExpectedStdout string    `json:"expected_stdout,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Code           string    `json:"code,omitempty"` // Generated code (optional, for debugging)

	// Source-constraint violations (constrained-construction benchmarks).
	// Non-empty means the code was rejected BEFORE execution.
	ConstraintViolations []string `json:"constraint_violations,omitempty"`

	// Self-repair metrics (M-EVAL-LOOP)
	FirstAttemptOk  bool   `json:"first_attempt_ok"`            // Did first attempt succeed?
	RepairUsed      bool   `json:"repair_used"`                 // Did we attempt a repair?
	RepairOk        bool   `json:"repair_ok"`                   // Did repair succeed?
	ErrCode         string `json:"err_code,omitempty"`          // Error code from taxonomy (PAR_001, etc.)
	RepairTokensIn  int    `json:"repair_tokens_in,omitempty"`  // Input tokens for repair attempt
	RepairTokensOut int    `json:"repair_tokens_out,omitempty"` // Output tokens for repair attempt

	// Prompt versioning (M-EVAL-LOOP)
	PromptVersion string `json:"prompt_version,omitempty"` // Prompt version used (v0.3.0-hints, etc.)

	// Reproducibility (M-EVAL-LOOP)
	BinaryHash string   `json:"binary_hash,omitempty"` // SHA256 of ailang binary
	StdlibHash string   `json:"stdlib_hash,omitempty"` // SHA256 of stdlib
	Caps       []string `json:"caps,omitempty"`        // Capabilities granted

	// Contract verification results (M-CONTRACT-EVAL)
	VerifyOk        bool   `json:"verify_ok"`             // All contracts verified
	VerifyVerified  int    `json:"verify_verified"`       // Count of verified functions
	VerifyCounterex int    `json:"verify_counterexample"` // Count of counterexamples
	VerifySkipped   int    `json:"verify_skipped"`        // Count of skipped functions
	VerifyErrors    int    `json:"verify_errors"`         // Count of Z3 errors
	VerifyJSON      string `json:"verify_json,omitempty"` // Full ai-check JSON output

	// Agent mode KPIs (M-EVAL-AGENT)
	AgentTurns     int `json:"agent_turns,omitempty"`      // Number of conversation turns (agent mode only)
	AgentToolCalls int `json:"agent_tool_calls,omitempty"` // Tool invocations (agent mode only; validates agentic behavior — the "0 tool calls" signal)
	// AgentToolHistogram breaks the scalar count down by tool name (e.g.
	// {"bash":5,"edit":2}). Nil when the executor didn't capture names. Answers
	// "did the agent reach for a discoverable tool like `ailang fmt`" — the scalar
	// count can't. Added M-EVAL-TOOL-HISTOGRAM (2026-07-22).
	AgentToolHistogram map[string]int `json:"agent_tool_histogram,omitempty"`
	AgentTranscript    string         `json:"agent_transcript,omitempty"` // Full Claude conversation transcript (agent mode only)
	EvalMode           string         `json:"eval_mode,omitempty"`        // Evaluation mode: "standard" or "agent"

	// Context-compaction telemetry (M-AILANG-SEMANTIC-CONTEXT, v0.26.0). Leading
	// indicator of convergence thrash — the agent loop compacting (and so erasing)
	// its own working memory mid-run, forcing re-reads/rewrites. All zero = none.
	CompactionCount     int `json:"compaction_count,omitempty"`      // Total context-compaction events
	CompactionFirstStep int `json:"first_compaction_step,omitempty"` // Step of first compaction (0 = none)
	CompactionMaxLevel  int `json:"compaction_level_max,omitempty"`  // Highest structural compaction level (0 = none)

	// Experimental condition (M-CONTRACT-EVAL conditions dimension)
	Condition string `json:"condition,omitempty"` // Experimental condition: "baseline", "contract", "z3_guided", "full"

	// μRAG state for this run (M-BRAIN-MICRORAG)
	// Values: "on" | "off" | "auto" | "disabled" | "" (legacy / not set).
	// Lets eval-report break results down with vs. without JIT knowledge injection.
	MicroragState string `json:"microrag_state,omitempty"`

	// fmt-hook A/B state for this run (M-EVAL-FMT-WEAKMODEL-AB).
	// Values: "on" | "off" | "" (non-agent / not set). The resolved arm — the
	// ONLY per-arm difference in the fmt weak-model experiment — banked so the
	// required config diff between ON and OFF arms is reviewable from the results.
	FmtHookState string `json:"fmt_hook_state,omitempty"`
	// Per-turn fmt PostToolUse hook reality (M-EVAL-FMT-WEAKMODEL-AB). Each entry
	// is one observed format_ail.sh run classified formatted / deferred / error.
	// Empty on the OFF arm (and on ON runs where the hook never fired). Powers
	// M3's treatment-delivery-rate metric vs the ~8% fail-closed refusal baseline.
	FmtHookEvents []FmtHookEvent `json:"fmt_hook_events,omitempty"`

	// Cross-harness grouping (M-EVAL-CROSS-HARNESS)
	// Populated from models.yml model_family field. Enables --group-by=model-family
	// in eval-matrix to compare same model across different harnesses (e.g. claude vs opencode).
	ModelFamily string `json:"model_family,omitempty"`

	// Cost-and-speed budget metrics (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
	// Populated from executor.Result via AgentBenchmarkResult. Zero values mean "not measured".
	CostKilledAt   float64 `json:"cost_killed_at,omitempty"`   // > 0 if execution stopped because cost budget exceeded
	FirstAttemptMs int64   `json:"first_attempt_ms,omitempty"` // ms from task start to first solution submission
	SuccessAtMs    int64   `json:"success_at_ms,omitempty"`    // ms from task start to first passing solution (-1 = never)
	TokensPerSec   float64 `json:"tokens_per_sec,omitempty"`   // OutputTokens / generation_seconds

	// Executor finish reason (M-EVAL-SWEET-SPOT, v0.19.0). Promoted from
	// executor ProviderData (e.g. motoko_finish_reason="cost_exhausted") and
	// from agent runners on max-turns exit ("step_exhausted"). Empty when
	// the executor didn't surface a finish signal. Used by CategorizeAgentError.
	FinishReason string `json:"finish_reason,omitempty"`

	// Trial number for N-trial release smoke (M-EVAL-OS-LONGITUDINAL Phase 3, v0.23.0).
	// 1 = first trial (default, single-trial mode). 2+ = subsequent trials when
	// --trials N > 1. Used by SummarizeRotation to group multi-trial outcomes
	// into a per-benchmark pass_rate / token-distribution summary.json.
	// Zero is treated as 1 (backward compat with pre-Phase-3 result files).
	Trial int `json:"trial,omitempty"`
}

// EvalMode constants
const (
	EvalModeStandard = "standard" // Standard 0-shot + self-repair evaluation
	EvalModeAgent    = "agent"    // Agent-based evaluation with multi-turn interaction
)

// ErrorCategory constants
const (
	ErrorCategoryNone    = "none"
	ErrorCategoryCompile = "compile_error"
	ErrorCategoryRuntime = "runtime_error"
	ErrorCategoryLogic   = "logic_error"
	ErrorCategoryAPI     = "api_error"    // API call failed — fallback when no more specific cause is known
	ErrorCategoryVerify  = "verify_error" // Contract verification failed (M-CONTRACT-EVAL)
	// Generated source violated the benchmark's source_constraints (checked
	// before execution — the code never ran). Constrained-construction class.
	ErrorCategoryConstraint = "constraint_violation"
	// The model's API-level safety layer declined the prompt (e.g. Anthropic
	// stop_reason "refusal", empty content, HTTP 200). Distinct from
	// api_error: the infrastructure worked; the model would not answer.
	ErrorCategoryRefused = "refused"

	// Typed failure categories (M-EVAL-SWEET-SPOT, v0.19.0). Replace blanket
	// api_error attribution where the cause is actually identifiable. The
	// distinction matters for capability scoring: a model that times out or
	// runs out of turns may still be capable given more budget, while a
	// quota-exhausted run says nothing about the model's capability.
	ErrorCategoryTimeout        = "timeout"         // Wall-clock or context deadline exceeded
	ErrorCategoryQuotaExhausted = "quota_exhausted" // Provider account/key cap reached (e.g. OpenRouter monthly limit)
	ErrorCategoryRateLimit      = "rate_limit"      // 429 — transient, distinct from monthly cap
	ErrorCategoryCostKilled     = "cost_killed"     // Eval-side $ budget exceeded (motoko cost_exhausted, future executor caps)
	ErrorCategoryStepExhausted  = "step_exhausted"  // Agent ran out of turns / step budget without success
	ErrorCategoryThrashAborted  = "thrash_aborted"  // M-EVAL-OS-LONGITUDINAL Phase 1: cumulative tokens exceeded MaxTokensPerBench (free $0 models)
	// M-EVAL-MEM-GUARD: generated code exceeded the AILANG_EVAL_MAX_RSS
	// resident-memory cap and its process group was killed. A model failure
	// (unbounded allocation), banked instead of crashing the host — see the
	// 2026-07-20 rig kernel panic.
	ErrorCategoryResourceLimit = "resource_limit"
)

// CategorizeError determines the error category based on execution results
func CategorizeError(compileOk, runtimeOk, stdoutOk bool) string {
	switch {
	case !compileOk:
		return ErrorCategoryCompile
	case !runtimeOk:
		return ErrorCategoryRuntime
	case !stdoutOk:
		return ErrorCategoryLogic
	default:
		return ErrorCategoryNone
	}
}

// MetricsLogger handles writing metrics to JSON files
type MetricsLogger struct {
	outputDir string
}

// NewMetricsLogger creates a new metrics logger
func NewMetricsLogger(outputDir string) *MetricsLogger {
	return &MetricsLogger{
		outputDir: outputDir,
	}
}

// maxFieldSize is the maximum size for any single string field in the JSON output.
// This is a safety net to prevent oversized result files even if upstream limits fail.
const maxFieldSize = 1 * 1024 * 1024 // 1 MB

// truncateField truncates a string to maxFieldSize with a truncation marker
func truncateField(s string) string {
	if len(s) <= maxFieldSize {
		return s
	}
	return s[:maxFieldSize] + "\n\n[TRUNCATED - field exceeded 1 MB limit]"
}

// Log writes a RunMetrics to a JSON file
func (l *MetricsLogger) Log(m *RunMetrics) error {
	// Safety truncation: prevent any single field from producing oversized JSON
	m.Stdout = truncateField(m.Stdout)
	m.Stderr = truncateField(m.Stderr)
	m.Code = truncateField(m.Code)
	m.AgentTranscript = truncateField(m.AgentTranscript)
	m.VerifyJSON = truncateField(m.VerifyJSON)

	// M-BRAIN-MICRORAG: backstop population so direct struct-literal call sites
	// (e.g. agent paths in cmd/ailang/eval_benchmark.go) cannot silently drop
	// the μRAG state. Auto-derive from env if the caller didn't set it.
	if m.MicroragState == "" {
		m.MicroragState = MicroragModeAuto.ResolvedState()
	}

	// Determine subdirectory based on eval mode
	var targetDir string
	switch m.EvalMode {
	case EvalModeStandard:
		targetDir = filepath.Join(l.outputDir, "standard")
	case EvalModeAgent:
		targetDir = filepath.Join(l.outputDir, "agent")
	default:
		// Legacy: no eval_mode field, use root directory
		targetDir = l.outputDir
	}

	// Nest by condition if set (e.g., eval_results/agent/z3_guided/)
	if m.Condition != "" {
		targetDir = filepath.Join(targetDir, m.Condition)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename: <id>[_trialN]_<lang>_<model>_<timestamp>.json
	// Sanitize model name: replace colons with underscores (Windows compatibility).
	// Trial > 1 (M-EVAL-OS-LONGITUDINAL Phase 3) injects "_trialN" between the
	// benchmark id and the language. Trial == 0 or 1 keeps the legacy filename
	// shape so existing aggregation tools (eval-summary, eval-matrix) work
	// unchanged on single-trial runs.
	sanitizedModel := strings.ReplaceAll(m.Model, ":", "_")
	var filename string
	if m.Trial > 1 {
		filename = fmt.Sprintf("%s_trial%d_%s_%s_%d.json",
			m.ID, m.Trial, m.Lang, sanitizedModel, m.Timestamp.Unix())
	} else {
		filename = fmt.Sprintf("%s_%s_%s_%d.json",
			m.ID, m.Lang, sanitizedModel, m.Timestamp.Unix())
	}
	path := filepath.Join(targetDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// CalculateCostWithBreakdown calculates cost using separate input/output token counts
// This provides accurate pricing based on models.yml configuration
// Returns 0.0 if model not found - FAIL LOUDLY, NO SILENT FALLBACKS
func CalculateCostWithBreakdown(model string, inputTokens, outputTokens int) float64 {
	// CRITICAL: GlobalModelsConfig MUST be initialized
	if GlobalModelsConfig == nil {
		// Return 0 to make it obvious something is wrong
		// Better to see $0.00 in reports than trust wrong data
		return 0.0
	}

	cost, err := GlobalModelsConfig.CalculateCostForModel(model, inputTokens, outputTokens)
	if err != nil {
		// Model not found in config - return 0 to force investigation
		// NO SILENT FALLBACKS - we want to know when pricing is missing
		return 0.0
	}

	return cost
}

// NewRunMetrics creates a new RunMetrics with timestamp and error category.
// MicroragState is auto-populated from the inherited env so every metrics
// emission honours the eval-suite --microrag flag (M-BRAIN-MICRORAG).
func NewRunMetrics(id, lang, model string, seed int64) *RunMetrics {
	return &RunMetrics{
		ID:            id,
		Lang:          lang,
		Model:         model,
		Seed:          seed,
		Timestamp:     time.Now(),
		MicroragState: MicroragModeAuto.ResolvedState(),
	}
}
