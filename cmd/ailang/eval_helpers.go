package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// evalBenchmarkDir is the directory containing benchmark YAML files.
// Set by --benchmark-dir flag; defaults to "benchmarks" (CWD-relative).
var evalBenchmarkDir = "benchmarks"

// evalVerifyFlag enables contract verification during eval.
var evalVerifyFlag bool

// evalVerifyTimeout is the per-function Z3 timeout for contract verification.
var evalVerifyTimeout = eval_harness.DefaultVerifyTimeout

// evalDevtoolsPromptFlag appends the devtools prompt to agent system prompts.
var evalDevtoolsPromptFlag bool

// evalMicroragMode controls AILANG_MICRORAG_ENABLED in subprocess env (M-BRAIN-MICRORAG).
// Default auto: respect inherited environment. on/off force the value.
var evalMicroragMode = eval_harness.MicroragModeAuto

// evalFmtHookMode controls whether the LANDED format_ail.sh PostToolUse fmt hook
// is wired into agent workspaces (M-EVAL-FMT-WEAKMODEL-AB). Default off preserves
// today's behaviour; on is the treatment arm of the fmt A/B.
var evalFmtHookMode = eval_harness.FmtHookModeOff

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

// preservedEvalArtifacts are files in the output dir that are RUN METADATA, not
// per-benchmark results, and so must survive cleanResults.
//
// cleanResults globs "*.json", which used to mean "delete everything". That
// silently destroyed cohort_manifest.json: M4a writes it at freeze time (before
// the run loop, so it records the resolved cohort and chain_id) and cleanResults
// runs a few lines later, so the artifact the run had just printed as release
// evidence pointed at a deleted file on every non---skip-existing run.
// summary.json has the same shape (aggregate, not a result) and is listed for the
// same reason.
var preservedEvalArtifacts = map[string]bool{
	cohortManifestFilename: true,
	"summary.json":         true,
}

// cleanResults removes old per-benchmark result files, preserving run metadata.
func cleanResults(outputDir string) {
	// Remove result JSON files but keep directory structure and run metadata
	pattern := filepath.Join(outputDir, "*.json")
	files, _ := filepath.Glob(pattern)
	for _, f := range files {
		if preservedEvalArtifacts[filepath.Base(f)] {
			continue
		}
		_ = os.Remove(f)
	}
}
