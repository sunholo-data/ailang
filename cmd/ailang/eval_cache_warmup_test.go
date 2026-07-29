package main

import "testing"

// M-ANTHROPIC-CACHE-HIT-RATE D4: the warm-up must fire exactly where it pays and
// nowhere else. Getting the grouping wrong is expensive in both directions —
// warming a single-job group is a pure ~1.25x loss, and failing to warm a real
// group forfeits ~34 percentage points of the saving.
func TestGroupJobsForWarmup(t *testing.T) {
	t.Run("skips single-job groups", func(t *testing.T) {
		// One benchmark for this model: the write could never be read back.
		jobs := []Job{{Model: "claude-sonnet-4-5", Benchmark: "b1", Language: "ailang"}}
		if got := groupJobsForWarmup(jobs); len(got) != 0 {
			t.Errorf("got %d groups, want 0 — a single call cannot read back its own cache write", len(got))
		}
	})

	t.Run("skips non-anthropic models", func(t *testing.T) {
		// Other providers cache automatically server-side; there is no
		// client-visible write step to pre-warm.
		jobs := []Job{
			{Model: "gpt5", Benchmark: "b1", Language: "ailang"},
			{Model: "gpt5", Benchmark: "b2", Language: "ailang"},
		}
		for k := range groupJobsForWarmup(jobs) {
			if k.model == "gpt5" {
				t.Error("non-Anthropic model should not be warmed")
			}
		}
	})

	t.Run("separates groups by language", func(t *testing.T) {
		// The teaching prompt is per-language, so ailang and python do not share
		// a cacheable prefix and must be warmed independently.
		jobs := []Job{
			{Model: "m", Benchmark: "b1", Language: "ailang"},
			{Model: "m", Benchmark: "b2", Language: "ailang"},
			{Model: "m", Benchmark: "b1", Language: "python"},
			{Model: "m", Benchmark: "b2", Language: "python"},
		}
		counts := map[cacheWarmupKey]int{}
		for _, j := range jobs {
			counts[cacheWarmupKey{model: j.Model, language: j.Language}]++
		}
		if len(counts) != 2 {
			t.Fatalf("expected 2 (model,language) groups, got %d", len(counts))
		}
		for k, n := range counts {
			if n != 2 {
				t.Errorf("group %v has %d jobs, want 2", k, n)
			}
		}
	})

	t.Run("picks one representative per group", func(t *testing.T) {
		jobs := []Job{
			{Model: "m", Benchmark: "b1", Language: "ailang"},
			{Model: "m", Benchmark: "b2", Language: "ailang"},
			{Model: "m", Benchmark: "b3", Language: "ailang"},
		}
		counts := map[cacheWarmupKey]int{}
		for _, j := range jobs {
			counts[cacheWarmupKey{model: j.Model, language: j.Language}]++
		}
		if len(counts) != 1 {
			t.Errorf("3 jobs on one model+language should be ONE warm-up group, got %d", len(counts))
		}
	})
}
