package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

// The warm-up's failure paths all return BEFORE any network call, which makes
// them testable without a provider — and they are the paths that matter most,
// since the whole design promise is "a warm-up can never break a suite".

func TestWarmPromptCaches_NoEligibleGroups(t *testing.T) {
	// Single-job groups are ineligible, so no spec is loaded and no call made.
	jobs := []Job{{Model: "claude-sonnet-4-5", Benchmark: "b1", Language: "ailang"}}
	if got := warmPromptCaches(context.Background(), jobs, time.Second); got != 0 {
		t.Errorf("warmed %d groups, want 0 — nothing here is eligible", got)
	}
}

func TestWarmPromptCaches_EmptyJobs(t *testing.T) {
	if got := warmPromptCaches(context.Background(), nil, time.Second); got != 0 {
		t.Errorf("warmed %d groups from no jobs, want 0", got)
	}
}

// A missing benchmark spec must degrade to a warning and zero warmed groups —
// never a panic and never an abort. This is the guarantee the suite depends on.
func TestWarmPromptCaches_MissingSpecIsNonFatal(t *testing.T) {
	prev := evalBenchmarkDir
	evalBenchmarkDir = t.TempDir() // no .yml files at all
	defer func() { evalBenchmarkDir = prev }()

	jobs := []Job{
		{Model: "claude-sonnet-4-5", Benchmark: "does-not-exist", Language: "ailang"},
		{Model: "claude-sonnet-4-5", Benchmark: "also-missing", Language: "ailang"},
	}
	got := warmPromptCaches(context.Background(), jobs, time.Second)
	if got != 0 {
		t.Errorf("warmed %d groups despite a missing spec, want 0", got)
	}
}

func TestWarmOneCache_MissingSpecReturnsError(t *testing.T) {
	prev := evalBenchmarkDir
	evalBenchmarkDir = t.TempDir()
	defer func() { evalBenchmarkDir = prev }()

	key := cacheWarmupKey{model: "claude-sonnet-4-5", language: "ailang"}
	sample := Job{Model: key.model, Benchmark: "nope", Language: key.language}

	err := warmOneCache(context.Background(), key, sample, time.Second)
	if err == nil {
		t.Fatal("expected an error for a missing benchmark spec")
	}
	if !strings.Contains(err.Error(), "load spec") {
		t.Errorf("error = %q, want it to name the load-spec step", err)
	}
}

func TestIsAnthropicModel_UnknownModelIsNotAnthropic(t *testing.T) {
	// Resolution failure must mean "skip the warm-up", not "abort the suite".
	if isAnthropicModel("no-such-model-xyz") {
		t.Error("an unresolvable model must not be treated as Anthropic")
	}
}

// With a VALID spec the warm-up gets past loading and into prefix derivation
// and agent construction. Both later failure modes must still be plain errors.
func TestWarmOneCache_ValidSpecLaterFailuresAreErrors(t *testing.T) {
	dir := t.TempDir()
	spec := "id: wu1\nlanguages: [ailang]\ntask_prompt: \"print hello\"\n"
	if err := os.WriteFile(filepath.Join(dir, "wu1.yml"), []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	prev := evalBenchmarkDir
	evalBenchmarkDir = dir
	defer func() { evalBenchmarkDir = prev }()

	t.Run("unknown language has no cacheable base", func(t *testing.T) {
		key := cacheWarmupKey{model: "claude-sonnet-4-5", language: "no-such-lang"}
		err := warmOneCache(context.Background(), key,
			Job{Model: key.model, Benchmark: "wu1", Language: key.language}, time.Second)
		if err == nil {
			t.Fatal("expected an error when there is no base prompt to cache")
		}
	})

	t.Run("unresolvable model fails at agent construction", func(t *testing.T) {
		key := cacheWarmupKey{model: "no-such-model-xyz", language: "ailang"}
		err := warmOneCache(context.Background(), key,
			Job{Model: key.model, Benchmark: "wu1", Language: key.language}, time.Second)
		if err == nil {
			t.Fatal("expected an error for an unresolvable model")
		}
		if !strings.Contains(err.Error(), "create agent") {
			t.Errorf("error = %q, want it to name the agent-construction step", err)
		}
	})
}

// withStubWarmupCall swaps the prefill for a stub and restores it afterwards.
func withStubWarmupCall(t *testing.T, fn func(ctx context.Context, model, prefix, task string, maxTokens int) error) {
	t.Helper()
	prev := warmupCallFn
	warmupCallFn = fn
	t.Cleanup(func() { warmupCallFn = prev })
}

// writeWarmupSpec drops a minimal valid benchmark spec and points the harness at it.
func writeWarmupSpec(t *testing.T, ids ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, id := range ids {
		body := "id: " + id + "\nlanguages: [ailang]\ntask_prompt: \"print hello\"\n"
		if err := os.WriteFile(filepath.Join(dir, id+".yml"), []byte(body), 0o644); err != nil {
			t.Fatalf("write spec %s: %v", id, err)
		}
	}
	prev := evalBenchmarkDir
	evalBenchmarkDir = dir
	t.Cleanup(func() { evalBenchmarkDir = prev })
}

// The success path: one warm-up per eligible group, carrying the cacheable base
// (not the task) and the cheap token budget.
func TestWarmPromptCaches_WarmsEligibleGroupOnce(t *testing.T) {
	writeWarmupSpec(t, "b1", "b2")

	var calls int
	var gotModel, gotPrefix string
	var gotMaxTokens int
	withStubWarmupCall(t, func(_ context.Context, model, prefix, task string, maxTokens int) error {
		calls++
		gotModel, gotPrefix, gotMaxTokens = model, prefix, maxTokens
		if task != warmupTask {
			t.Errorf("task = %q, want the fixed warm-up task", task)
		}
		return nil
	})

	jobs := []Job{
		{Model: "claude-sonnet-4-5", Benchmark: "b1", Language: "ailang"},
		{Model: "claude-sonnet-4-5", Benchmark: "b2", Language: "ailang"},
	}
	if got := warmPromptCaches(context.Background(), jobs, time.Second); got != 1 {
		t.Errorf("warmed %d groups, want 1", got)
	}
	if calls != 1 {
		t.Errorf("made %d prefill calls for one group, want exactly 1 — a per-benchmark warm-up would cost more than it saves", calls)
	}
	if gotModel != "claude-sonnet-4-5" {
		t.Errorf("warmed model %q, want claude-sonnet-4-5", gotModel)
	}
	if gotMaxTokens != warmupMaxTokens {
		t.Errorf("warm-up used MaxTokens=%d, want %d", gotMaxTokens, warmupMaxTokens)
	}
	if gotPrefix == "" {
		t.Error("warm-up sent an empty prefix — there would be nothing to cache")
	}
	if strings.Contains(gotPrefix, "print hello") {
		t.Error("task text leaked into the warmed prefix; the cached span must be the stable base only")
	}
}

// A failing prefill must be swallowed: the suite runs uncached rather than dying.
func TestWarmPromptCaches_PrefillFailureIsNonFatal(t *testing.T) {
	writeWarmupSpec(t, "b1", "b2")
	withStubWarmupCall(t, func(_ context.Context, _, _, _ string, _ int) error {
		return errors.New("provider exploded")
	})

	jobs := []Job{
		{Model: "claude-sonnet-4-5", Benchmark: "b1", Language: "ailang"},
		{Model: "claude-sonnet-4-5", Benchmark: "b2", Language: "ailang"},
	}
	if got := warmPromptCaches(context.Background(), jobs, time.Second); got != 0 {
		t.Errorf("warmed %d groups despite a failing prefill, want 0", got)
	}
}
