package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestParseTierList covers the M3 --tier argument parser.
func TestParseTierList(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"whitespace only", "   ", nil, false},
		{"single tier", "smoke", []string{"smoke"}, false},
		{"all four", "smoke,core,stretch,vision", []string{"smoke", "core", "stretch", "vision"}, false},
		{"with whitespace", " smoke , core ", []string{"smoke", "core"}, false},
		{"dropped empty segments", "smoke,,core", []string{"smoke", "core"}, false},
		{"invalid tier", "smoke,bogus", nil, true},
		{"all invalid", "xxx", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTierList(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (got=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseTierList(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestFilterBenchmarksByTier loads real benchmark YAMLs and verifies the
// tier filter produces the expected counts (matches M2 distribution).
func TestFilterBenchmarksByTier(t *testing.T) {
	prev := evalBenchmarkDir
	evalBenchmarkDir = "../../benchmarks"
	t.Cleanup(func() { evalBenchmarkDir = prev })

	all := discoverBenchmarks()
	if len(all) == 0 {
		t.Skip("no benchmarks found at ../../benchmarks")
	}

	smoke := filterBenchmarksByTier(all, []string{"smoke"})
	core := filterBenchmarksByTier(all, []string{"core"})
	stretch := filterBenchmarksByTier(all, []string{"stretch"})
	vision := filterBenchmarksByTier(all, []string{"vision"})
	experimental := filterBenchmarksByTier(all, []string{"experimental"})

	// Sum must equal total (each benchmark has exactly one tier).
	// "experimental" tier added 2026-05-22 for diagnostic probes (probes
	// measure gaps, don't score capability) — included in the sum invariant
	// but NOT in the distribution drift detector below.
	if got := len(smoke) + len(core) + len(stretch) + len(vision) + len(experimental); got != len(all) {
		t.Errorf("tier counts sum to %d, want %d (tier-per-benchmark invariant)", got, len(all))
	}

	// Distribution drift detector. Refreshed 2026-07-08 (second pass): +4
	// constrained-construction / insight-forced benchmarks (emit_exact_bytes,
	// digitless_constants, commonmark_emphasis, binary_strings_1e18) after the
	// sonnet probe showed the first 8 sat at the top of stretch — stretch
	// 22→26. Earlier same day: 8 frontier-class benchmarks (M-EVAL-FRONTIER-TIER
	// authoring phase) moved stretch 14→22; they re-tier to `frontier` when
	// that tier lands. Prior baselines: 23/26/11/9 (2026-06-08), 17/32/11/9
	// (2026-05-20), 15/21/11/6 (post-M5). Kept in sync with the sibling check in
	// internal/eval_harness/spec_test.go. Refresh again when drift outgrows the
	// ±3 envelope (don't widen tolerance — bump the target counts to match
	// reality). Experimental tier is intentionally excluded — probe count grows
	// independently.
	checkTierCount(t, "smoke", len(smoke), 23, 3)
	checkTierCount(t, "core", len(core), 26, 3)
	checkTierCount(t, "stretch", len(stretch), 26, 3)
	checkTierCount(t, "vision", len(vision), 9, 3)

	// Combined filter returns the union.
	smokePlusCore := filterBenchmarksByTier(all, []string{"smoke", "core"})
	if len(smokePlusCore) != len(smoke)+len(core) {
		t.Errorf("smoke+core = %d, want %d (union)", len(smokePlusCore), len(smoke)+len(core))
	}

	// Empty tier list is a pass-through (no filtering applied).
	passthrough := filterBenchmarksByTier(all, nil)
	if len(passthrough) != len(all) {
		t.Errorf("empty tier list filtered %d -> %d, want pass-through", len(all), len(passthrough))
	}
}

func checkTierCount(t *testing.T, name string, got, want, tolerance int) {
	t.Helper()
	if got < want-tolerance || got > want+tolerance {
		t.Errorf("tier %s: count=%d, want %d±%d", name, got, want, tolerance)
	}
}

// TestParseMatrixFlags covers the three M3 report-section flags scanned
// from the positional arg list.
func TestParseMatrixFlags(t *testing.T) {
	cases := []struct {
		name                                     string
		args                                     []string
		wantTags, wantSat, wantWins, wantHarness bool
	}{
		{"none", []string{"eval-matrix", "dir", "v1"}, false, false, false, false},
		{"by-tags only", []string{"eval-matrix", "dir", "v1", "--by-tags"}, true, false, false, false},
		{"show-saturated only", []string{"eval-matrix", "dir", "v1", "--show-saturated"}, false, true, false, false},
		{"ailang-wins only", []string{"eval-matrix", "dir", "v1", "--ailang-wins"}, false, false, true, false},
		{"by-harness only", []string{"eval-matrix", "dir", "v1", "--by-harness"}, false, false, false, true},
		{"all flags", []string{"eval-matrix", "dir", "v1", "--by-tags", "--show-saturated", "--ailang-wins", "--by-harness"}, true, true, true, true},
		{"unknown flag ignored", []string{"eval-matrix", "dir", "v1", "--by-tags", "--unknown"}, true, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTags, gotSat, gotWins, gotHarness := parseMatrixFlags(func(i int) string { return tc.args[i] }, len(tc.args))
			if gotTags != tc.wantTags || gotSat != tc.wantSat || gotWins != tc.wantWins || gotHarness != tc.wantHarness {
				t.Errorf("parseMatrixFlags(%v) = (%v, %v, %v, %v), want (%v, %v, %v, %v)",
					tc.args, gotTags, gotSat, gotWins, gotHarness, tc.wantTags, tc.wantSat, tc.wantWins, tc.wantHarness)
			}
		})
	}
}

// TestLoadBenchmarkTags loads the real benchmark directory and asserts
// every benchmark has 1-3 tags for standard tiers, 1-5 for experimental
// (invariant already enforced by TestAllBenchmarksHaveTierAndTags in
// eval_harness; this is the CLI-side smoke that the primitive sees them too).
func TestLoadBenchmarkTags(t *testing.T) {
	tags := eval_analysis.LoadBenchmarkTags("../../benchmarks")
	if len(tags) == 0 {
		t.Skip("no benchmarks found at ../../benchmarks")
	}
	// Build a tier index alongside the tag map so we can apply the
	// per-tier cap (standard: 1-3; experimental: 1-5).
	specPaths, err := filepath.Glob("../../benchmarks/*.yml")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	tiers := map[string]string{}
	for _, p := range specPaths {
		spec, err := eval_harness.LoadSpec(p)
		if err != nil {
			continue
		}
		tiers[spec.ID] = spec.Tier
	}
	for id, ts := range tags {
		maxTags := 3
		if tiers[id] == "experimental" {
			maxTags = 5
		}
		if len(ts) < 1 || len(ts) > maxTags {
			t.Errorf("benchmark %s: %d tags (want 1-%d for tier %q)", id, len(ts), maxTags, tiers[id])
		}
	}
}

// TestSectionPrintersSmoke verifies the three section printers produce
// deterministic, non-empty output against synthetic results. Captures
// stdout because the printers write directly to os.Stdout.
func TestSectionPrintersSmoke(t *testing.T) {
	results := []*eval_analysis.BenchmarkResult{
		{ID: "fizzbuzz", Lang: "ailang", Model: "m1", StdoutOk: true},
		{ID: "fizzbuzz", Lang: "python", Model: "m1", StdoutOk: true},
		{ID: "contract_bst_validate", Lang: "ailang", Model: "m1", StdoutOk: true},
		{ID: "contract_bst_validate", Lang: "python", Model: "m1", StdoutOk: false},
	}

	t.Run("saturated", func(t *testing.T) {
		out := captureStdout(t, func() { printSaturatedSection(results) })
		// fizzbuzz passes in both langs -> saturated
		if !strings.Contains(out, "fizzbuzz") {
			t.Errorf("saturated section missing fizzbuzz: %q", out)
		}
		// contract_bst_validate fails in python -> not saturated
		if strings.Contains(out, "- `contract_bst_validate`") {
			t.Errorf("saturated section should not list contract_bst_validate: %q", out)
		}
	})

	t.Run("ailang-wins", func(t *testing.T) {
		out := captureStdout(t, func() { printAILANGWinsSection(results) })
		// contract_bst_validate: ailang pass, python fail, same model -> win
		if !strings.Contains(out, "contract_bst_validate") {
			t.Errorf("ailang-wins section missing contract_bst_validate: %q", out)
		}
	})

	t.Run("by-tags without tags dir", func(t *testing.T) {
		// Point at an empty dir; loadBenchmarkTags returns empty, section
		// should print the placeholder header instead of crashing.
		dir := t.TempDir()
		out := captureStdout(t, func() { printTagsSection(results, dir) })
		if !strings.Contains(out, "no benchmark tags loaded") {
			t.Errorf("expected empty-tags placeholder, got: %q", out)
		}
	})
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written. Used because the section printers call fmt.Println
// directly (matches the style of the rest of eval_tools.go).
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- b.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// TestExpandModelSuite verifies all named suites expand correctly from models.yml.
// This guards against adding a suite to models.yml but forgetting to register
// it in expandModelSuite()'s switch statement.
func TestExpandModelSuite(t *testing.T) {
	cfg, err := eval_harness.LoadModelsConfig("../../internal/eval_harness/models.yml")
	if err != nil {
		t.Skipf("models.yml not available: %v", err)
	}
	cases := []struct {
		suite string
		field []string
	}{
		{"agent_suite", cfg.AgentSuite},
		{"extended_suite", cfg.ExtendedSuite},
		{"dev_models", cfg.DevModels},
		{"ollama_suite", cfg.OllamaSuite},
		{"harness_suite", cfg.HarnessSuite},
		{"lang_harness_suite", cfg.LangHarnessSuite},
	}
	for _, tc := range cases {
		t.Run(tc.suite, func(t *testing.T) {
			if len(tc.field) == 0 {
				t.Skipf("suite %q is empty in models.yml, skipping", tc.suite)
			}
			got := expandModelSuite(tc.suite, cfg)
			if len(got) == 0 {
				t.Errorf("expandModelSuite(%q) returned empty — suite registered in models.yml but missing from switch", tc.suite)
			}
			if len(got) != len(tc.field) {
				t.Errorf("expandModelSuite(%q): got %d models, want %d", tc.suite, len(got), len(tc.field))
			}
		})
	}
}
