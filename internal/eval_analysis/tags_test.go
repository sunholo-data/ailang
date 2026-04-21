package eval_analysis

import (
	"strings"
	"testing"
)

// TestRefusalPatternsAtLeastFour is the M4 acceptance guard: the sprint
// spec mandates ≥4 refusal patterns. If the list is trimmed below that
// bar accidentally, this test fails loudly.
func TestRefusalPatternsAtLeastFour(t *testing.T) {
	patterns := RefusalPatterns()
	if len(patterns) < 4 {
		t.Fatalf("want ≥4 refusal patterns, got %d: %v", len(patterns), patterns)
	}
	// Verify the four canonical patterns from the sprint doc are present.
	required := []string{"apologies", "i cannot", "i'm sorry, but", "as an ai"}
	for _, r := range required {
		found := false
		for _, p := range patterns {
			if strings.EqualFold(p, r) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required pattern %q missing from RefusalPatterns()", r)
		}
	}
}

func TestDetectRefusal(t *testing.T) {
	cases := []struct {
		name   string
		code   string
		stderr string
		stdout string
		want   bool
	}{
		{"clean code", "print('ok')", "", "ok\n", false},
		{"refusal in code", "# I cannot help with that task.", "", "", true},
		{"refusal in stderr", "", "SyntaxError: Apologies, I'm unable to assist.", "", true},
		{"refusal in stdout", "", "", "As an AI language model I refuse", true},
		{"case insensitive", "I CANNOT perform this action", "", "", true},
		{"false positive guard (cant is not i cant)", "def cant(): pass", "", "", false},
		{"sorry-but", "I'm sorry, but I can't write malware.", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectRefusal(tc.code, tc.stderr, tc.stdout)
			if got != tc.want {
				t.Errorf("DetectRefusal(%q, %q, %q) = %v, want %v",
					tc.code, tc.stderr, tc.stdout, got, tc.want)
			}
		})
	}
}

func TestGroupByTags(t *testing.T) {
	tags := map[string][]string{
		"fizzbuzz": {"algorithmic", "string_algo"},
		"fib":      {"recursion", "algorithmic"},
		"refuser":  {"algorithmic"},
	}
	results := []*BenchmarkResult{
		{ID: "fizzbuzz", Lang: "ailang", StdoutOk: true},
		{ID: "fizzbuzz", Lang: "python", StdoutOk: true},
		{ID: "fib", Lang: "ailang", StdoutOk: true},
		{ID: "fib", Lang: "python", StdoutOk: false},
		{ID: "refuser", Lang: "ailang", StdoutOk: false, RefusalDetected: true},
		{ID: "refuser", Lang: "python", StdoutOk: false, RefusalDetected: true},
	}

	report := GroupByTags(results, tags)

	// algorithmic spans fizzbuzz + fib + refuser(excluded), so 2 per lang.
	alg := report.Aggregates["algorithmic"]
	if alg == nil {
		t.Fatal("algorithmic aggregate missing")
	}
	if alg.AILANGTotal != 2 || alg.PythonTotal != 2 {
		t.Errorf("algorithmic totals = (ailang=%d, python=%d), want 2/2 (refusal excluded)",
			alg.AILANGTotal, alg.PythonTotal)
	}
	// AILANG 2/2 pass, Python 1/2 pass -> delta = +0.5.
	if alg.AILANGPass != 2 || alg.PythonPass != 1 {
		t.Errorf("algorithmic pass = (ailang=%d, python=%d), want 2/1",
			alg.AILANGPass, alg.PythonPass)
	}
	if alg.Delta < 0.49 || alg.Delta > 0.51 {
		t.Errorf("algorithmic delta = %v, want ~0.5", alg.Delta)
	}

	// recursion only applies to fib -> 1 per lang.
	rec := report.Aggregates["recursion"]
	if rec == nil {
		t.Fatal("recursion aggregate missing")
	}
	if rec.AILANGTotal != 1 || rec.PythonTotal != 1 {
		t.Errorf("recursion totals = (ailang=%d, python=%d), want 1/1",
			rec.AILANGTotal, rec.PythonTotal)
	}

	// Tags slice must be sorted.
	for i := 1; i < len(report.Tags); i++ {
		if report.Tags[i-1] > report.Tags[i] {
			t.Errorf("report.Tags not sorted: %v", report.Tags)
		}
	}
}

func TestDetectAILANGOnlyWins(t *testing.T) {
	results := []*BenchmarkResult{
		// fizzbuzz: AILANG wins on 3 models, Python fails -> pattern.
		{ID: "fizzbuzz", Lang: "ailang", Model: "gpt5", StdoutOk: true},
		{ID: "fizzbuzz", Lang: "python", Model: "gpt5", StdoutOk: false},
		{ID: "fizzbuzz", Lang: "ailang", Model: "claude", StdoutOk: true},
		{ID: "fizzbuzz", Lang: "python", Model: "claude", StdoutOk: false},
		{ID: "fizzbuzz", Lang: "ailang", Model: "gemini", StdoutOk: true},
		{ID: "fizzbuzz", Lang: "python", Model: "gemini", StdoutOk: false},
		// adt_option: AILANG wins on only 1 model -> not a pattern.
		{ID: "adt_option", Lang: "ailang", Model: "gpt5", StdoutOk: true},
		{ID: "adt_option", Lang: "python", Model: "gpt5", StdoutOk: false},
		// balanced: both pass -> no win.
		{ID: "balanced", Lang: "ailang", Model: "gpt5", StdoutOk: true},
		{ID: "balanced", Lang: "python", Model: "gpt5", StdoutOk: true},
		// refuser: AILANG pass, Python refuse -> excluded.
		{ID: "refuser", Lang: "ailang", Model: "gpt5", StdoutOk: true},
		{ID: "refuser", Lang: "python", Model: "gpt5", StdoutOk: false, RefusalDetected: true},
	}

	report := DetectAILANGOnlyWins(results)

	if got, want := len(report.Wins), 4; got != want {
		t.Errorf("len(Wins) = %d, want %d (3 fizzbuzz models + 1 adt_option)", got, want)
	}
	if got := report.PerBenchmark["fizzbuzz"]; got != 3 {
		t.Errorf("fizzbuzz per-benchmark = %d, want 3", got)
	}
	if got := report.PerBenchmark["adt_option"]; got != 1 {
		t.Errorf("adt_option per-benchmark = %d, want 1", got)
	}
	if _, ok := report.PerBenchmark["refuser"]; ok {
		t.Errorf("refuser should be excluded due to Python refusal")
	}
	if got, want := len(report.Patterns), 1; got != want {
		t.Errorf("len(Patterns) = %d (%v), want %d (fizzbuzz only)",
			got, report.Patterns, want)
	}
	if len(report.Patterns) > 0 && report.Patterns[0] != "fizzbuzz" {
		t.Errorf("Patterns[0] = %q, want fizzbuzz", report.Patterns[0])
	}
}

func TestDetectSaturation(t *testing.T) {
	// Two baselines, two benchmarks: fizzbuzz saturates both, fib only saturates v2.
	b1 := &Baseline{
		Version: "v1",
		Results: []*BenchmarkResult{
			{ID: "fizzbuzz", Lang: "ailang", Model: "gpt5", StdoutOk: true},
			{ID: "fizzbuzz", Lang: "python", Model: "gpt5", StdoutOk: true},
			{ID: "fib", Lang: "ailang", Model: "gpt5", StdoutOk: false},
			{ID: "fib", Lang: "python", Model: "gpt5", StdoutOk: true},
		},
	}
	b2 := &Baseline{
		Version: "v2",
		Results: []*BenchmarkResult{
			{ID: "fizzbuzz", Lang: "ailang", Model: "gpt5", StdoutOk: true},
			{ID: "fizzbuzz", Lang: "python", Model: "gpt5", StdoutOk: true},
			{ID: "fib", Lang: "ailang", Model: "gpt5", StdoutOk: true},
			{ID: "fib", Lang: "python", Model: "gpt5", StdoutOk: true},
		},
	}

	sat := DetectSaturation([]*Baseline{b1, b2}, 2)
	if len(sat) != 1 || sat[0].ID != "fizzbuzz" {
		ids := make([]string, len(sat))
		for i, s := range sat {
			ids[i] = s.ID
		}
		t.Errorf("DetectSaturation = %v, want [fizzbuzz]", ids)
	}
	if len(sat) > 0 && len(sat[0].BaselinesSeen) != 2 {
		t.Errorf("BaselinesSeen = %v, want 2 entries", sat[0].BaselinesSeen)
	}
}

func TestDetectSaturationSkipsPythonOnlyBaseline(t *testing.T) {
	// Legacy Python-only baseline should be ignored entirely.
	b := &Baseline{
		Version: "legacy",
		Results: []*BenchmarkResult{
			{ID: "fizzbuzz", Lang: "python", Model: "gpt5", StdoutOk: true},
		},
	}
	if got := DetectSaturation([]*Baseline{b}, 1); got != nil {
		t.Errorf("expected nil (python-only baseline skipped), got %v", got)
	}
}
