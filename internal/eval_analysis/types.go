package eval_analysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BenchmarkResult represents the result of a single benchmark execution
// This mirrors the JSON structure from internal/eval_harness/metrics.go
type BenchmarkResult struct {
	ID            string    `json:"id"`
	Lang          string    `json:"lang"`
	Model         string    `json:"model"`
	Executor      string    `json:"executor,omitempty"` // Executor used: "claude", "gemini", etc. (agent mode)
	Seed          int64     `json:"seed"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	CostUSD       float64   `json:"cost_usd"`
	CompileOk     bool      `json:"compile_ok"`
	RuntimeOk     bool      `json:"runtime_ok"`
	StdoutOk      bool      `json:"stdout_ok"`
	DurationMs    int64     `json:"duration_ms"`
	CompileMs     int64     `json:"compile_ms"`
	ExecuteMs     int64     `json:"execute_ms"`
	ErrorCategory string    `json:"error_category"`
	Stdout        string    `json:"stdout,omitempty"`
	Stderr        string    `json:"stderr,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Code          string    `json:"code,omitempty"`

	// Self-repair metrics (M-EVAL-LOOP)
	FirstAttemptOk  bool   `json:"first_attempt_ok"`
	RepairUsed      bool   `json:"repair_used"`
	RepairOk        bool   `json:"repair_ok"`
	ErrCode         string `json:"err_code,omitempty"`
	RepairTokensIn  int    `json:"repair_tokens_in,omitempty"`
	RepairTokensOut int    `json:"repair_tokens_out,omitempty"`

	// Prompt versioning
	PromptVersion string `json:"prompt_version,omitempty"`

	// Agent evaluation metrics (M-EVAL-AGENT)
	EvalMode        string `json:"eval_mode,omitempty"`        // "standard" or "agent"
	Condition       string `json:"condition,omitempty"`        // Experimental condition: "baseline", "agent_prompt", etc.
	AgentTurns      int    `json:"agent_turns,omitempty"`      // Number of conversation turns
	AgentTranscript string `json:"agent_transcript,omitempty"` // Full session log

	// Reproducibility
	BinaryHash string   `json:"binary_hash,omitempty"`
	StdlibHash string   `json:"stdlib_hash,omitempty"`
	Caps       []string `json:"caps,omitempty"`

	// Cross-harness comparison (M-EVAL-CROSS-HARNESS)
	// Logical model family for grouping paired harness results.
	// e.g. "claude-sonnet-4-6" shared by "claude" and "opencode" executors.
	ModelFamily string `json:"model_family,omitempty"`

	// Refusal detection (M-EVAL-SUITE-PREP M4): populated at load time
	// by DetectRefusal() scanning stdout+stderr. Not written by eval_harness,
	// purely a read-side annotation so historical results inherit it.
	RefusalDetected bool `json:"refusal_detected,omitempty"`

	// Cost-and-speed budget metrics (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
	// Zero values mean "not measured" — preserves byte-identical replay of
	// pre-v0.15.1 baselines (additive schema only).
	CostKilledAt   float64 `json:"cost_killed_at,omitempty"`   // > 0 if execution stopped because cost budget exceeded
	FirstAttemptMs int64   `json:"first_attempt_ms,omitempty"` // ms from task start to first solution submission
	SuccessAtMs    int64   `json:"success_at_ms,omitempty"`    // ms from task start to first passing solution (-1 = never)
	TokensPerSec   float64 `json:"tokens_per_sec,omitempty"`   // OutputTokens / generation_seconds
}

// Baseline represents a stored baseline with metadata
type Baseline struct {
	Version         string             `json:"version"`
	Timestamp       time.Time          `json:"timestamp"`
	Model           string             `json:"model"`
	Languages       string             `json:"languages"`
	SelfRepair      bool               `json:"self_repair"`
	TotalBenchmarks int                `json:"total_benchmarks"`
	SuccessCount    int                `json:"success_count"`
	FailCount       int                `json:"fail_count"`
	MatrixFile      string             `json:"matrix_file"`
	GitCommit       string             `json:"git_commit"`
	GitBranch       string             `json:"git_branch"`
	Results         []*BenchmarkResult `json:"-"` // Loaded separately
}

// ComparisonReport contains structured diff between two benchmark runs
type ComparisonReport struct {
	BaselineLabel string
	NewLabel      string
	Baseline      *Baseline
	New           *Baseline

	// Changes
	Fixed         []*BenchmarkChange
	Broken        []*BenchmarkChange
	StillPassing  []*BenchmarkResult
	StillFailing  []*BenchmarkResult
	NewBenchmarks []*BenchmarkResult
	Removed       []*BenchmarkResult

	// Aggregates
	BaselineSuccessRate float64
	NewSuccessRate      float64
	SuccessRateDelta    float64
	TotalBaselineBench  int
	TotalNewBench       int
}

// BenchmarkChange represents a benchmark that changed status
type BenchmarkChange struct {
	ID             string
	Lang           string
	Model          string
	BaselineStatus bool // true = passing, false = failing
	NewStatus      bool
	BaselineError  string
	NewError       string
}

// PerformanceMatrix contains aggregated performance data
type PerformanceMatrix struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	TotalRuns int       `json:"total_runs"`

	// Overall aggregates
	Aggregates Aggregates `json:"aggregates"`

	// Breakdown by dimension
	Models         map[string]*ModelStats     `json:"models"`
	Benchmarks     map[string]*BenchmarkStats `json:"benchmarks"`
	ErrorCodes     []*ErrorCodeStats          `json:"error_codes"`
	Languages      map[string]*LanguageStats  `json:"languages"`
	PromptVersions map[string]*PromptStats    `json:"prompt_versions,omitempty"`
}

// Aggregates contains overall performance statistics
type Aggregates struct {
	ZeroShotSuccess   float64 `json:"0-shot_success"`      // First attempt success rate
	FinalSuccess      float64 `json:"final_success"`       // After repair success rate
	RepairUsed        int     `json:"repair_used"`         // Number of repairs attempted
	RepairSuccessRate float64 `json:"repair_success_rate"` // Repair success rate
	TotalTokens       int     `json:"total_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
}

// ModelStats contains per-model performance
type ModelStats struct {
	TotalRuns       int                       `json:"total_runs"`
	Aggregates      Aggregates                `json:"aggregates"`
	Benchmarks      map[string]*BenchmarkRun  `json:"benchmarks"`
	BaselineVersion string                    `json:"baseline_version,omitempty"` // Which baseline these results came from
	Languages       map[string]*LanguageStats `json:"languages,omitempty"`        // Per-language breakdown for this model
}

// BenchmarkStats contains per-benchmark performance
type BenchmarkStats struct {
	TotalRuns   int      `json:"total_runs"`
	SuccessRate float64  `json:"success_rate"`
	AvgTokens   float64  `json:"avg_tokens"`
	Languages   []string `json:"languages"`
}

// LanguageStats contains per-language performance
type LanguageStats struct {
	TotalRuns   int     `json:"total_runs"`
	SuccessRate float64 `json:"success_rate"`
	AvgTokens   float64 `json:"avg_tokens"`
}

// PromptStats contains per-prompt-version performance
type PromptStats struct {
	TotalRuns       int     `json:"total_runs"`
	ZeroShotSuccess float64 `json:"0-shot_success"`
	FinalSuccess    float64 `json:"final_success"`
	AvgTokens       float64 `json:"avg_tokens"`
}

// ErrorCodeStats contains per-error-code statistics
type ErrorCodeStats struct {
	Code          string  `json:"code"`
	Count         int     `json:"count"`
	RepairSuccess float64 `json:"repair_success"`
}

// BenchmarkRun contains single benchmark execution stats
type BenchmarkRun struct {
	Success        bool `json:"success"`
	FirstAttemptOk bool `json:"first_attempt_ok"`
	RepairUsed     bool `json:"repair_used"`
	Tokens         int  `json:"tokens"`
}

// SummaryEntry is a simplified record for JSONL export
type SummaryEntry struct {
	ID             string  `json:"id"`
	Lang           string  `json:"lang"`
	Model          string  `json:"model"`
	Executor       string  `json:"executor,omitempty"` // Executor used: "claude", "gemini" (agent mode)
	Seed           int64   `json:"seed"`
	PromptVersion  string  `json:"prompt_version,omitempty"`
	FirstAttemptOk bool    `json:"first_attempt_ok"`
	RepairUsed     bool    `json:"repair_used"`
	RepairOk       bool    `json:"repair_ok"`
	ErrCode        string  `json:"err_code,omitempty"`
	CompileOk      bool    `json:"compile_ok"`
	RuntimeOk      bool    `json:"runtime_ok"`
	StdoutOk       bool    `json:"stdout_ok"`
	ErrorCategory  string  `json:"error_category"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	CostUSD        float64 `json:"cost_usd"`
	DurationMs     int64   `json:"duration_ms"`
	Timestamp      string  `json:"timestamp"`
	Stderr         string  `json:"stderr,omitempty"`
	// Agent evaluation fields (M-EVAL-AGENT)
	EvalMode   string `json:"eval_mode,omitempty"`   // "standard" or "agent"
	Condition  string `json:"condition,omitempty"`   // Experimental condition: "baseline", "agent_prompt", etc.
	AgentTurns int    `json:"agent_turns,omitempty"` // Number of conversation turns
}

// TierLanguageStats holds per-language aggregate metrics for one tier.
// Used in TierAggregate.LanguageStats and TierHistoryPoint.LanguageStats
// to surface data for all eval languages (python, ailang, javascript, go, …).
type TierLanguageStats struct {
	Runs        int     `json:"runs"`
	Pass        int     `json:"pass"`
	SuccessRate float64 `json:"success_rate"`
	RepairDelta float64 `json:"repair_delta,omitempty"`
	AvgCostUSD  float64 `json:"avg_cost_usd,omitempty"`
	APIErrors   int     `json:"api_errors,omitempty"`
}

// TierAggregate contains per-tier pass-rate metrics. Populated by
// ExportBenchmarkJSON from the tier field attached to each benchmark
// result (resolved via the benchmark YAML's tier). The Core tier pass
// rate is the dashboard headline metric per M-EVAL-SUITE-PREP M6.
type TierAggregate struct {
	TotalRuns         int     `json:"total_runs"`
	AILANGRuns        int     `json:"ailang_runs"`
	PythonRuns        int     `json:"python_runs"`
	AILANGSuccessRate float64 `json:"ailang_success_rate"`
	PythonSuccessRate float64 `json:"python_success_rate"`
	BenchmarkCount    int     `json:"benchmark_count"` // unique benchmark IDs in this tier

	// Generic per-language breakdown — includes all eval languages (python,
	// ailang, javascript, go, …). The typed AILANG*/Python* fields above
	// remain for backward compatibility with existing dashboard consumers.
	LanguageStats map[string]*TierLanguageStats `json:"language_stats,omitempty"`

	// M-DASH-V2: per-tier × per-model breakdown so charts can filter
	// time-series data to this tier. Outer key is model name, inner key is
	// language. Nil when the tier has no runs.
	ModelStats map[string]map[string]*ModelDimensionStats `json:"model_stats,omitempty"`

	// M-DASH-V2: API reliability per tier. Splits by language so dashboards
	// can show "how many gemini-3-1-pro AILANG runs on core tier returned
	// api_error?" separately from Python.
	APIErrorCount  int `json:"api_error_count"`
	AILANGAPIError int `json:"ailang_api_error"`
	PythonAPIError int `json:"python_api_error"`

	// M-DASH-V2: refusal count per tier (RefusalDetected at load time).
	RefusalCount int `json:"refusal_count"`

	// M-DASH-V2: self-repair efficacy and cost for this tier. RepairDelta =
	// final pass rate − first-attempt pass rate; answers "does self-repair
	// help more on hard tiers?". AvgCostUSD split by language lets callers
	// tell whether stretch is 3× pricier on AILANG specifically.
	AILANGRepairDelta float64 `json:"ailang_repair_delta"`
	PythonRepairDelta float64 `json:"python_repair_delta"`
	AILANGAvgCostUSD  float64 `json:"ailang_avg_cost_usd"`
	PythonAvgCostUSD  float64 `json:"python_avg_cost_usd"`
}

// ModelDimensionStats is the per-(model, language) cross-section used in
// both TierAggregate.ModelStats and TagAggregate.ModelStats. Shape matches
// what the time-series chart reads from history.modelStats[model][lang]
// so the frontend can swap data sources cleanly.
type ModelDimensionStats struct {
	SuccessRate   float64 `json:"successRate"`
	TotalRuns     int     `json:"totalRuns"`
	AvgTokens     float64 `json:"avgTokens"`
	APIErrorCount int     `json:"apiErrorCount,omitempty"`
	RefusalCount  int     `json:"refusalCount,omitempty"`
}

// TierHistoryPoint is a per-tier snapshot inside a single history entry.
// Lets PerModelTrend filter the time series to e.g. just the Core tier so
// the chart updates when TierToggle changes — not just the hero row.
type TierHistoryPoint struct {
	AILANGSuccessRate float64                                    `json:"ailang_success_rate"`
	PythonSuccessRate float64                                    `json:"python_success_rate"`
	AILANGRuns        int                                        `json:"ailang_runs"`
	PythonRuns        int                                        `json:"python_runs"`
	BenchmarkCount    int                                        `json:"benchmark_count"`
	ModelStats        map[string]map[string]*ModelDimensionStats `json:"modelStats,omitempty"`
	// Generic per-language breakdown for all eval languages (python, ailang,
	// javascript, go, …). The typed AILANG*/Python* fields remain for
	// backward compatibility.
	LanguageStats map[string]*TierLanguageStats `json:"language_stats,omitempty"`
}

// SuiteEvent is a timeline annotation (benchmark additions, taxonomy
// changes, etc.) loaded from benchmarks/events.yml. Rendered as a dashed
// ReferenceLine on every time-series chart.
type SuiteEvent struct {
	Version      string   `json:"version" yaml:"version"`
	Label        string   `json:"label" yaml:"label"`
	Kind         string   `json:"kind" yaml:"kind"` // "benchmark_add" | "benchmark_remove" | "taxonomy" | "prompt"
	Color        string   `json:"color,omitempty" yaml:"color,omitempty"`
	AffectsTiers []string `json:"affects_tiers,omitempty" yaml:"affects_tiers,omitempty"` // if set, event only renders when one of these tiers is selected
}

// DashboardJSON represents the structure of docs/static/benchmarks/latest.json
// This is the single source of truth for the dashboard frontend
type DashboardJSON struct {
	Version    string                   `json:"version"`
	Timestamp  string                   `json:"timestamp"`
	TotalRuns  int                      `json:"totalRuns"`
	Aggregates map[string]interface{}   `json:"aggregates"`
	Tiers      map[string]TierAggregate `json:"tiers,omitempty"` // Per-tier aggregates: smoke/core/stretch/vision
	// M-DASH-V2: per-tag aggregates (12 canonical tags) with per-model
	// cross-sections so the dashboard can narrow the charts to a tag.
	Tags        map[string]*TagAggregate `json:"tags,omitempty"`
	Models      map[string]interface{}   `json:"models"`
	AgentModels map[string]interface{}   `json:"agentModels,omitempty"` // Agent-only models (separate from standard)
	Benchmarks  map[string]interface{}   `json:"benchmarks"`
	Languages   map[string]interface{}   `json:"languages"` // map[language]->stats
	Executors   map[string]interface{}   `json:"executors"` // map[executor]->agent stats (claude, gemini)
	// M-BENCHMARK-SECTION: harness-grouped aggregates for cross-harness comparison page.
	// Keys are agent_cli values ("claude", "gemini", "opencode", "codex").
	Harnesses map[string]interface{} `json:"harnesses,omitempty"`
	History   []HistoryEntry         `json:"history"`
	// M-DASH-V2: suite-change annotations rendered as ReferenceLine on every
	// time-series chart. Sourced from benchmarks/events.yml.
	Events []SuiteEvent `json:"events,omitempty"`
}

// HistoryEntry represents a single version's data in the history array
type HistoryEntry struct {
	Version       string                 `json:"version"`
	Timestamp     string                 `json:"timestamp"`
	SuccessRate   float64                `json:"successRate"`
	TotalRuns     int                    `json:"totalRuns"`
	SuccessCount  int                    `json:"successCount"`
	Languages     string                 `json:"languages"`
	LanguageStats map[string]interface{} `json:"languageStats,omitempty"`
	ModelStats    map[string]interface{} `json:"modelStats,omitempty"` // Per-model, per-language stats for trend charts
	// M-DASH-V2: per-tier snapshots. Lets the time-series chart filter to
	// one tier retroactively (pre-v0.14.0 baselines use the CURRENT tier
	// mapping — docs describe this as an approximation).
	Tiers map[string]*TierHistoryPoint `json:"tiers,omitempty"`
}

// Validate checks if a DashboardJSON structure is valid
func (d *DashboardJSON) Validate() error {
	if d.Version == "" {
		return fmt.Errorf("version required")
	}
	if d.Timestamp == "" {
		return fmt.Errorf("timestamp required")
	}
	if len(d.History) == 0 {
		return fmt.Errorf("history must have at least one entry")
	}

	// Check for duplicate versions in history (normalized: "v0.9.0" == "0.9.0")
	seen := make(map[string]bool)
	for _, entry := range d.History {
		norm := strings.TrimPrefix(entry.Version, "v")
		if seen[norm] {
			return fmt.Errorf("duplicate version in history: %s", entry.Version)
		}
		seen[norm] = true
	}

	return nil
}

// ToSummaryEntry converts a BenchmarkResult to a SummaryEntry for JSONL export
func (r *BenchmarkResult) ToSummaryEntry() *SummaryEntry {
	return &SummaryEntry{
		ID:             r.ID,
		Lang:           r.Lang,
		Model:          r.Model,
		Executor:       r.Executor,
		Seed:           r.Seed,
		PromptVersion:  r.PromptVersion,
		FirstAttemptOk: r.FirstAttemptOk,
		RepairUsed:     r.RepairUsed,
		RepairOk:       r.RepairOk,
		ErrCode:        r.ErrCode,
		CompileOk:      r.CompileOk,
		RuntimeOk:      r.RuntimeOk,
		StdoutOk:       r.StdoutOk,
		ErrorCategory:  r.ErrorCategory,
		InputTokens:    r.InputTokens,
		OutputTokens:   r.OutputTokens,
		TotalTokens:    r.TotalTokens,
		CostUSD:        r.CostUSD,
		DurationMs:     r.DurationMs,
		Timestamp:      r.Timestamp.Format(time.RFC3339),
		Stderr:         r.Stderr,
		EvalMode:       r.EvalMode,
		Condition:      r.Condition,
		AgentTurns:     r.AgentTurns,
	}
}

// MarshalJSON implements custom JSON marshaling for JSONL (single-line)
func (s *SummaryEntry) MarshalJSON() ([]byte, error) {
	type Alias SummaryEntry
	return json.Marshal((*Alias)(s))
}
