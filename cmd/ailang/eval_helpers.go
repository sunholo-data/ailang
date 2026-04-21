package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
)

// evalBenchmarkDir is the directory containing benchmark YAML files.
// Set by --benchmark-dir flag; defaults to "benchmarks" (CWD-relative).
var evalBenchmarkDir = "benchmarks"

// evalVerifyFlag enables contract verification during eval.
var evalVerifyFlag bool

// evalVerifyTimeout is the per-function Z3 timeout for contract verification.
var evalVerifyTimeout = 5 * time.Second

// evalDevtoolsPromptFlag appends the devtools prompt to agent system prompts.
var evalDevtoolsPromptFlag bool

// discoverBenchmarks finds all .yml files in the benchmark directory
func discoverBenchmarks() []string {
	entries, err := os.ReadDir(evalBenchmarkDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not read benchmarks directory: %v\n", err)
		return nil
	}

	var benchmarks []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isBenchmarkMetaFile(entry.Name()) {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml") {
			// Remove extension to get benchmark ID
			name := strings.TrimSuffix(entry.Name(), ".yml")
			name = strings.TrimSuffix(name, ".yaml")
			benchmarks = append(benchmarks, name)
		}
	}
	return benchmarks
}

// isBenchmarkMetaFile reports whether a YAML under benchmarks/ is a
// non-spec meta-file (suite-change log, etc.) that the benchmark scanner
// must skip. Keep this list short — prefer renaming over growing it.
func isBenchmarkMetaFile(name string) bool {
	switch name {
	case "events.yml":
		return true
	}
	return false
}

// parseTierList splits a comma-separated --tier argument and validates each entry
// against eval_harness.ValidTiers. Whitespace is trimmed; empty input returns nil.
func parseTierList(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	validTiers := map[string]bool{}
	for _, t := range eval_harness.ValidTiers {
		validTiers[t] = true
	}
	var tiers []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !validTiers[t] {
			return nil, fmt.Errorf("unknown tier %q (valid: %v)", t, eval_harness.ValidTiers)
		}
		tiers = append(tiers, t)
	}
	return tiers, nil
}

// filterBenchmarksByTier narrows a benchmark ID list to those whose YAML tier
// is in `tiers`. Missing tier fields default to "core" (per M1 back-compat
// rule). Benchmarks that fail to load are dropped with a stderr warning.
func filterBenchmarksByTier(benchmarks []string, tiers []string) []string {
	if len(tiers) == 0 {
		return benchmarks
	}
	keep := map[string]bool{}
	for _, t := range tiers {
		keep[t] = true
	}
	var out []string
	for _, id := range benchmarks {
		specPath := filepath.Join(evalBenchmarkDir, id+".yml")
		spec, err := eval_harness.LoadSpec(specPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", yellow("⚠️"), id, err)
			continue
		}
		if keep[spec.Tier] {
			out = append(out, id)
		}
	}
	return out
}

// checkAPIKeys validates that required API keys are set
func checkAPIKeys(models []string) {
	warnings := []string{}

	for _, model := range models {
		switch {
		case strings.Contains(model, "gpt"):
			if os.Getenv("OPENAI_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s OPENAI_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		case strings.Contains(model, "claude"):
			if os.Getenv("ANTHROPIC_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s ANTHROPIC_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		case strings.Contains(model, "gemini"):
			if os.Getenv("GOOGLE_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s GOOGLE_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		}
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Println(w)
		}
		fmt.Println()
		fmt.Println("Set API keys to run with real models:")
		fmt.Println("  export OPENAI_API_KEY='sk-...'")
		fmt.Println("  export ANTHROPIC_API_KEY='sk-ant-...'")
		fmt.Println("  export GOOGLE_API_KEY='...'")
		fmt.Println()
	}
}

// cleanResults removes old result files
func cleanResults(outputDir string) {
	// Remove JSON files but keep directory structure
	pattern := filepath.Join(outputDir, "*.json")
	files, _ := filepath.Glob(pattern)
	for _, f := range files {
		_ = os.Remove(f)
	}
}
