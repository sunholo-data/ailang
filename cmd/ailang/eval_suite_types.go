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

// modelSuiteResolvers maps a `--models` SUITE TOKEN to its accessor on
// ModelsConfig. This is the SINGLE source of truth for "which tokens name a
// suite", shared by:
//   - expandModelSuite        — resolves the token to its members
//   - isModelSuiteToken       — records the token in the M4a cohort manifest
//
// One table, so a suite cannot become resolvable but unrecorded in the manifest
// (which would make a frozen cohort's provenance silently incomplete).
var modelSuiteResolvers = map[string]func(*eval_harness.ModelsConfig) []string{
	"agent_suite":        func(c *eval_harness.ModelsConfig) []string { return c.AgentSuite },
	"benchmark_suite":    func(c *eval_harness.ModelsConfig) []string { return c.BenchmarkSuite },
	"extended_suite":     func(c *eval_harness.ModelsConfig) []string { return c.ExtendedSuite },
	"dev_models":         func(c *eval_harness.ModelsConfig) []string { return c.DevModels },
	"ollama_suite":       func(c *eval_harness.ModelsConfig) []string { return c.OllamaSuite },
	"harness_suite":      func(c *eval_harness.ModelsConfig) []string { return c.HarnessSuite },
	"lang_harness_suite": func(c *eval_harness.ModelsConfig) []string { return c.LangHarnessSuite },
}

// isModelSuiteToken reports whether value is a known suite name.
func isModelSuiteToken(value string) bool {
	_, ok := modelSuiteResolvers[strings.TrimSpace(value)]
	return ok
}

// expandModelSuite resolves a --models argument. If the value is a single
// token matching a known suite name (e.g. "agent_suite", "benchmark_suite",
// "extended_suite", "dev_models"), it expands to the composite from
// models.yml. Otherwise the value is split on commas and trimmed.
func expandModelSuite(value string, cfg *eval_harness.ModelsConfig) []string {
	trimmed := strings.TrimSpace(value)
	if cfg != nil && !strings.Contains(trimmed, ",") {
		if resolve, ok := modelSuiteResolvers[trimmed]; ok {
			if members := resolve(cfg); len(members) > 0 {
				return members
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
