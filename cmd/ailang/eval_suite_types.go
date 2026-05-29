package main

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// evalTracer is the OpenTelemetry tracer for eval harness instrumentation.
// Declared in eval_suite.go (kept there to avoid init-order issues with the
// suite span that uses it directly).

// SuiteResult captures the result of a single benchmark run in the suite
type SuiteResult struct {
	BenchmarkID string
	Language    string
	Model       string
	Success     bool
	Error       error
}

// Job represents a single benchmark task
type Job struct {
	Model     string
	Benchmark string
	Language  string
	Condition string // Experimental condition: "baseline", "contract", "z3_guided", "full", or "" for legacy
	Trial     int    // M-EVAL-OS-LONGITUDINAL Phase 3: 1 = default single-trial; 2+ when --trials N > 1
}

// EvalChainContext holds observatory chain state for agent eval runs.
// When non-nil, benchmark results are stored as chain stages in observatory.db.
type EvalChainContext struct {
	Store   *observatory.Store
	ChainID string
}

// expandModelSuite resolves a --models argument. If the value is a single
// token matching a known suite name (e.g. "agent_suite", "benchmark_suite",
// "extended_suite", "dev_models"), it expands to the composite from
// models.yml. Otherwise the value is split on commas and trimmed.
func expandModelSuite(value string, cfg *eval_harness.ModelsConfig) []string {
	trimmed := strings.TrimSpace(value)
	if cfg != nil && !strings.Contains(trimmed, ",") {
		switch trimmed {
		case "agent_suite":
			if len(cfg.AgentSuite) > 0 {
				return cfg.AgentSuite
			}
		case "benchmark_suite":
			if len(cfg.BenchmarkSuite) > 0 {
				return cfg.BenchmarkSuite
			}
		case "extended_suite":
			if len(cfg.ExtendedSuite) > 0 {
				return cfg.ExtendedSuite
			}
		case "dev_models":
			if len(cfg.DevModels) > 0 {
				return cfg.DevModels
			}
		case "ollama_suite":
			if len(cfg.OllamaSuite) > 0 {
				return cfg.OllamaSuite
			}
		case "harness_suite":
			if len(cfg.HarnessSuite) > 0 {
				return cfg.HarnessSuite
			}
		case "lang_harness_suite":
			if len(cfg.LangHarnessSuite) > 0 {
				return cfg.LangHarnessSuite
			}
		}
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
