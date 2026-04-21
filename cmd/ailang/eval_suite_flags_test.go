package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval_analysis"
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

	// Sum must equal total (each benchmark has exactly one tier).
	if got := len(smoke) + len(core) + len(stretch) + len(vision); got != len(all) {
		t.Errorf("tier counts sum to %d, want %d (tier-per-benchmark invariant)", got, len(all))
	}

	// Distribution target post-M5 (added polymorphic_ord_defaulting +
	// typed_refusal): 15/21/11/6 ±2. Kept as drift detector.
	checkTierCount(t, "smoke", len(smoke), 15, 2)
	checkTierCount(t, "core", len(core), 21, 2)
	checkTierCount(t, "stretch", len(stretch), 11, 2)
	checkTierCount(t, "vision", len(vision), 6, 2)

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
		name                        string
		args                        []string
		wantTags, wantSat, wantWins bool
	}{
		{"none", []string{"eval-matrix", "dir", "v1"}, false, false, false},
		{"by-tags only", []string{"eval-matrix", "dir", "v1", "--by-tags"}, true, false, false},
		{"show-saturated only", []string{"eval-matrix", "dir", "v1", "--show-saturated"}, false, true, false},
		{"ailang-wins only", []string{"eval-matrix", "dir", "v1", "--ailang-wins"}, false, false, true},
		{"all three", []string{"eval-matrix", "dir", "v1", "--by-tags", "--show-saturated", "--ailang-wins"}, true, true, true},
		{"unknown flag ignored", []string{"eval-matrix", "dir", "v1", "--by-tags", "--unknown"}, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTags, gotSat, gotWins := parseMatrixFlags(func(i int) string { return tc.args[i] }, len(tc.args))
			if gotTags != tc.wantTags || gotSat != tc.wantSat || gotWins != tc.wantWins {
				t.Errorf("parseMatrixFlags(%v) = (%v, %v, %v), want (%v, %v, %v)",
					tc.args, gotTags, gotSat, gotWins, tc.wantTags, tc.wantSat, tc.wantWins)
			}
		})
	}
}

// TestLoadBenchmarkTags loads the real benchmark directory and asserts
// every benchmark has 1-3 tags (invariant already enforced by
// TestAllBenchmarksHaveTierAndTags in eval_harness; this is the CLI-side
// smoke that the primitive sees them too).
func TestLoadBenchmarkTags(t *testing.T) {
	tags := eval_analysis.LoadBenchmarkTags("../../benchmarks")
	if len(tags) == 0 {
		t.Skip("no benchmarks found at ../../benchmarks")
	}
	for id, ts := range tags {
		if len(ts) < 1 || len(ts) > 3 {
			t.Errorf("benchmark %s: %d tags (want 1-3)", id, len(ts))
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
