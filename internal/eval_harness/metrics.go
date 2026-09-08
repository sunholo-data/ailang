package eval_harness

import (
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/executor"
)

// RunMetrics captures the results of a single benchmark run
type RunMetrics struct {
	ID           string `json:"id"`
	Lang         string `json:"lang"`
	Model        string `json:"model"`
	Executor     string `json:"executor,omitempty"` // Executor used: "claude", "gemini", etc. (agent mode only)
	Seed         int64  `json:"seed"`
	InputTokens  int    `json:"input_tokens"`            // Prompt tokens (recorded but not primary metric)
	OutputTokens int    `json:"output_tokens"`           // Generated code tokens (PRIMARY METRIC; reasoning excluded)
	ReasonTokens int    `json:"reason_tokens,omitempty"` // Hidden reasoning/thinking tokens (billed as output)
	// CacheReadInputTokens / CacheCreationInputTokens record prompt-cache
	// activity (M-ANTHROPIC-CACHE-HIT-RATE M3). Before v0.31.0 these were
	// modelled on ai.Response but never persisted, so NONE of the 28,139 banked
	// result files carried them and our own cache hit rate was unmeasurable from
	// our own data — the low rate had to be reported to us from outside.
	//
	// omitempty keeps pre-v0.31.0 baselines parsing unchanged (absent reads as 0)
	// and keeps rows for providers without cache reporting free of noise.
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	// CacheAccounted distinguishes "this run genuinely had no cache reads" from
	// "nobody recorded them" — omitempty makes both look like 0 otherwise, and
	// that ambiguity is not cosmetic: agent mode never populated the cache fields
	// before 2026-08-11, so treating absent-as-zero would score every historical
	// row as 100% fresh and make post-fix runs look artificially cheaper. Set it
	// wherever the executor path reports cache usage, EVEN WHEN THE VALUE IS 0
	// (a local ollama model with no cache is a real zero, not a missing one).
	CacheAccounted bool    `json:"cache_accounted,omitempty"`
	TotalTokens    int     `json:"total_tokens"` // Total for billing (includes reasoning)
	CostUSD        float64 `json:"cost_usd"`
	// CostProvenance labels CostUSD: "metered" (an account was genuinely
	// charged), "list-price-equivalent" (real arithmetic over real tokens, but
	// a subscription/OAuth lane covered the run and nobody was billed),
	// "free-local" (on-device, no marginal cost), or "unknown".
	//
	// Added 2026-07-30. omitempty keeps earlier baselines parsing unchanged —
	// but an ABSENT value means unmeasured, NOT metered. Any aggregate that
	// claims metered dollars must filter on this, not assume it.
	CostProvenance string `json:"cost_provenance,omitempty"`
	CompileOk      bool   `json:"compile_ok"`
	RuntimeOk      bool   `json:"runtime_ok"`
	StdoutOk       bool   `json:"stdout_ok"`
	DurationMs     int64  `json:"duration_ms"` // Total time (startup + compile + execution)
	CompileMs      int64  `json:"compile_ms"`  // Time spent in compilation (if separate)
	ExecuteMs      int64  `json:"execute_ms"`  // Time spent in execution (if measurable)

	// LLM generation latency (M-LYCEUM-PROVIDER M3 route A/B). llm_wall_ms is
	// the client-observed wall time of the generation HTTP call(s) behind this
	// row — first attempt, PLUS the repair attempt when one produced the
	// persisted code, and the FAILED call's wall time on api_error rows (the
	// "did the gateway die at 30s or 5min?" datum). ttft_ms is
	// time-to-first-token, only measurable on streaming transports (ollama);
	// 0/absent means UNMEASURED, not instant. OpenRouter's server-side TTFT
	// lives in the broadcast observatory instead.
	LLMWallMs      int64     `json:"llm_wall_ms,omitempty"`
	TTFTMs         int64     `json:"ttft_ms,omitempty"`
	ErrorCategory  string    `json:"error_category"` // compile_error | runtime_error | logic_error | none
	Stdout         string    `json:"stdout,omitempty"`
	Stderr         string    `json:"stderr,omitempty"`
	ExpectedStdout string    `json:"expected_stdout,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Code           string    `json:"code,omitempty"` // Generated code (optional, for debugging)

	// ResolvedProfile / ResolvedExtensions record what the SUBJECT reports it
	// actually loaded, from its own step-0 broadcast — not what we asked for.
	// Banking them is what makes a row auditable after the fact; asserting on
	// them in-flight without recording them (as M4 first did) leaves no evidence
	// behind.
	ResolvedProfile    string `json:"resolved_profile,omitempty"`
	ResolvedExtensions string `json:"resolved_extensions,omitempty"`

	// Validity marks whether this row is a MEASUREMENT at all, as opposed to a
	// failure to measure (dead subject, harness error, wrong config). NIL means
	// valid — every row banked before v0.31.0 lacks the field, and treating
	// absent as invalid would erase all of that history. See validity.go.
	Validity *Validity `json:"validity,omitempty"`

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

	// M-EVAL-FAILURE-ATTRIBUTION: the program computed every expected value but
	// dressed the output — `treeToList(tree): [1, 2, 3]` where `[1, 2, 3]` was
	// asked for. Byte-exact stdout is deliberate and stays the contract, so this
	// is still a FAIL and still attributed to the model; the split exists purely
	// so that "could not solve it" and "solved it, ignored the output contract"
	// stop being the same number. The second is a prompt-teaching signal, not a
	// capability one.
	//
	// Frequency is currently LOW and unmeasured at scale: observed on
	// tree_transformation_pipeline (ran as or-deepseek-v4-pro-0813 before that id
	// was folded into or-deepseek-v4-pro by the 2026-08-13 repoint; 2026-08-13 core
	// run), and IsOutputFormatFailure reclassifies 0 of the 72 logic_error rows
	// in the v0.32.0 baseline. That zero is the point — an earlier, looser
	// heuristic claimed 17 such rows and every one it found was a genuine wrong
	// answer (contract_rle_roundtrip expects a final "true", produced "1q"),
	// which is exactly the over-matching this category must never do.
	ErrorCategoryOutputFormat = "output_format"

	// The model spent its whole output budget thinking and emitted no answer:
	// content empty, output_tokens == reasoning_tokens. HTTP 200, no provider
	// error, and — critically — a finish_reason that is either absent (the
	// stream was cancelled) or a perfectly normal "stop".
	//
	// Distinct from ErrorCategoryRefused, which is also "empty content, HTTP 200":
	// a refusal means the safety layer declined, while this means the model
	// engaged with the task and never came back. The reasoning text is present
	// and on-topic; one measured instance reasoned correctly about all six target
	// sites of its sprint before stalling.
	//
	// Distinct from ErrorCategoryTimeout because the cause is identifiable and
	// lives in the model, not the clock — more wall-clock does not obviously fix
	// it, and it should not be read as "slow but capable".
	//
	// MEASURED 2026-08-26 from OpenRouter Broadcast traces (the provider's own
	// side of the wire, ingested into the prod observatory). Across the whole
	// 08-18..08-22 corpus, 3 of 173 generations had no finish_reason; ALL THREE
	// matched this signature, and the other 170 all carried content or tool_calls.
	// It is not model-specific: deepseek-v4-flash-0731 under pi and z-ai/glm-5.2
	// under OpenCode both produced it, on different provider hosts.
	//
	// NOTE for anyone tempted to fix this with reasoning.max_tokens: OpenRouter's
	// third-party upstreams do NOT enforce it (probed 2026-07-19, recorded on the
	// or-glm-5-2 entry in models.yml). Output headroom is the only enforced lever.
	ErrorCategoryReasoningStall = "reasoning_stall"
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

	// M-EVAL-MEASUREMENT-CONTRACT: a row that failed for an unidentifiable
	// reason is not a measurement. Applied HERE, at the single point every
	// banked row passes through, rather than at each construction site — the
	// framework's first version wired only its consumers and left this gap, so
	// harness crashes were banked as valid model failures for six weeks. See
	// applyValidityBackstop for what depended on that being wrong.
	m.applyValidityBackstop()

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
	// CRITICAL: modelreg.GlobalModelsConfig MUST be initialized
	if modelreg.GlobalModelsConfig == nil {
		// Return 0 to make it obvious something is wrong
		// Better to see $0.00 in reports than trust wrong data
		return 0.0
	}

	cost, err := modelreg.GlobalModelsConfig.CalculateCostForModel(model, inputTokens, outputTokens)
	if err != nil {
		// Model not found in config - return 0 to force investigation
		// NO SILENT FALLBACKS - we want to know when pricing is missing
		return 0.0
	}

	return cost
}

// CalculateCostWithCache is CalculateCostWithBreakdown for callers that know the
// run's prompt-cache reads. inputTokens must be FRESH input, disjoint from
// cacheReadTokens. Same no-silent-fallback stance: an unpriced model returns 0.
func CalculateCostWithCache(model string, inputTokens, outputTokens, cacheReadTokens int) float64 {
	if modelreg.GlobalModelsConfig == nil {
		return 0.0
	}

	cost, err := modelreg.GlobalModelsConfig.CalculateCostForModelWithCache(model, inputTokens, outputTokens, cacheReadTokens)
	if err != nil {
		return 0.0
	}

	return cost
}

// standardModeCostProvenance classifies a standard-mode row's cost.
//
// Standard mode reaches providers over their metered HTTP APIs via an API key
// (internal/ai), so a priced model is genuinely billed — unlike agent mode,
// where the codex and claude CLIs run on subscriptions. A zero-rate model is
// on-device and free. An unresolvable model yields unknown rather than a guess,
// matching CalculateCostWithBreakdown's no-silent-fallback stance.
func standardModeCostProvenance(model string) string {
	if modelreg.GlobalModelsConfig == nil {
		return string(executor.CostProvenanceUnknown)
	}
	cfg, ok := modelreg.GlobalModelsConfig.Models[model]
	if !ok {
		return string(executor.CostProvenanceUnknown)
	}
	if cfg.Pricing.InputPer1K == 0 && cfg.Pricing.OutputPer1K == 0 {
		return string(executor.CostFreeLocal)
	}
	// D1 (Mark-ratified 2026-08-26): Ollama Cloud is a subscription lane, so its
	// non-zero prices are IMPUTED from a metered twin rather than charged. Saying
	// `metered` would claim a spend that never happened.
	if IsOllamaCloudRoute(cfg.APIName) ||
		(cfg.AgentModelName != nil && IsOllamaCloudRoute(*cfg.AgentModelName)) {
		return string(executor.CostListPriceEquivalent)
	}
	// Same reasoning for Anthropic on the OAuth lane (M-EVAL-STANDARD-OAUTH):
	// standard mode can now authenticate with a subscription access token, in
	// which case the priced arithmetic is real but nobody was charged. Calling
	// that `metered` would invent spend — the exact defect this field exists to
	// prevent. Reads the same resolver the client used, so the label cannot
	// disagree with the lane that actually ran.
	if cfg.Provider == string(ai.ProviderAnthropic) && ai.AnthropicLaneIsOAuth() {
		return string(executor.CostListPriceEquivalent)
	}
	return string(executor.CostMetered)
}

// NewRunMetrics creates a new RunMetrics with timestamp and error category.
// MicroragState is auto-populated from the inherited env so every metrics
// emission honours the eval-suite --microrag flag (M-BRAIN-MICRORAG).
func NewRunMetrics(id, lang, model string, seed int64) *RunMetrics {
	return &RunMetrics{
		ID:             id,
		Lang:           lang,
		Model:          model,
		Seed:           seed,
		Timestamp:      time.Now(),
		MicroragState:  MicroragModeAuto.ResolvedState(),
		CostProvenance: standardModeCostProvenance(model),
	}
}

// FreshTokens returns the tokens this run actually made the provider process:
// uncached input plus generated output (reasoning included, since upstream bills
// it at the output rate). Cache reads are EXCLUDED.
//
// Why the token KPI excludes them (Mark, 2026-08-11): a cache read costs ~20% of
// a fresh token, so counting it as one full token makes a run that caches well
// look more expensive than one that does not — the metric would penalise exactly
// the behaviour we want. Measured on the pi lane: total_tokens reads ~3.9x the
// real work on an AILANG run at a 75% hit rate, and AILANG is hit hardest of all
// because its large teaching prompt is the most cacheable thing in the system.
//
// ok is false when this row predates cache accounting, i.e. the split is unknown
// rather than zero. Callers MUST NOT treat that as an all-fresh run: every
// agent-mode row before 2026-08-11 would then report inflated fresh tokens and
// make later runs look better for free. Exclude those rows and say how many.
func (m *RunMetrics) FreshTokens() (tokens int, ok bool) {
	if !m.CacheAccounted {
		return 0, false
	}
	fresh := m.InputTokens - m.CacheReadInputTokens - m.CacheCreationInputTokens
	if fresh < 0 {
		// InputTokens is cache-INCLUSIVE by construction, so this cannot happen
		// unless a producer changed that contract. Clamp rather than emit a
		// negative token count, and refuse to vouch for it.
		return 0, false
	}
	return fresh + m.OutputTokens + m.ReasonTokens, true
}
