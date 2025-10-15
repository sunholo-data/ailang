package eval_analysis

import (
	"encoding/json"
	"time"
)

// BenchmarkResult represents the result of a single benchmark execution
// This mirrors the JSON structure from internal/eval_harness/metrics.go
type BenchmarkResult struct {
	ID            string    `json:"id"`
	Lang          string    `json:"lang"`
	Model         string    `json:"model"`
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

	// Reproducibility
	BinaryHash string   `json:"binary_hash,omitempty"`
	StdlibHash string   `json:"stdlib_hash,omitempty"`
	Caps       []string `json:"caps,omitempty"`
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
}

// ToSummaryEntry converts a BenchmarkResult to a SummaryEntry for JSONL export
func (r *BenchmarkResult) ToSummaryEntry() *SummaryEntry {
	return &SummaryEntry{
		ID:             r.ID,
		Lang:           r.Lang,
		Model:          r.Model,
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
	}
}

// MarshalJSON implements custom JSON marshaling for JSONL (single-line)
func (s *SummaryEntry) MarshalJSON() ([]byte, error) {
	type Alias SummaryEntry
	return json.Marshal((*Alias)(s))
}
