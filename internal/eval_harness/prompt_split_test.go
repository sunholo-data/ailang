package eval_harness

import (
	"strings"
	"testing"
)

// M-ANTHROPIC-CACHE-HIT-RATE M2.
//
// Splitting the eval prompt into a cacheable base + a volatile task is a
// COST change, not a behaviour change. If the split ever alters the bytes the
// model receives, every v0.30.0 baseline becomes incomparable and the cache win
// is unmeasurable — the exact outcome design decision D1 exists to avoid.
//
// This test is the guard on that invariant.

func TestPromptParts_RejoinIsByteIdentical(t *testing.T) {
	specs := []struct {
		name string
		spec *BenchmarkSpec
	}{
		{
			name: "task_prompt_field",
			spec: &BenchmarkSpec{
				ID:         "b1",
				Languages:  []string{"ailang", "python"},
				TaskPrompt: "Write a program that prints hello.",
			},
		},
		{
			name: "legacy_prompt_field",
			spec: &BenchmarkSpec{
				ID:        "b2",
				Languages: []string{"ailang", "python"},
				Prompt:    "Compute the first 10 primes.",
			},
		},
		{
			name: "no_task_description",
			spec: &BenchmarkSpec{
				ID:        "b3",
				Languages: []string{"ailang", "python"},
			},
		},
	}

	for _, tc := range specs {
		for _, lang := range []string{"ailang", "python"} {
			t.Run(tc.name+"/"+lang, func(t *testing.T) {
				joined := tc.spec.PromptForLanguage(lang)
				base, task := tc.spec.PromptPartsForLanguage(lang)

				if base+task != joined {
					t.Errorf("split prompt does not rejoin byte-identically for %s/%s\n"+
						"  len(base)=%d len(task)=%d len(base+task)=%d len(joined)=%d",
						tc.name, lang, len(base), len(task), len(base+task), len(joined))
				}
			})
		}
	}
}

// TestPromptParts_SplitIsAtTheTaskBoundary: the base must be the reusable part.
// If the task text leaked into the base, the "stable" prefix would differ per
// benchmark and every request would write a fresh cache entry it never reads —
// a silent 1.25x cost increase rather than a saving.
func TestPromptParts_SplitIsAtTheTaskBoundary(t *testing.T) {
	const task = "UNIQUE-TASK-TEXT-b7f3"
	spec := &BenchmarkSpec{
		ID:         "b1",
		Languages:  []string{"ailang"},
		TaskPrompt: task,
	}

	base, taskSection := spec.PromptPartsForLanguage("ailang")

	if strings.Contains(base, task) {
		t.Error("task text leaked into the cacheable base — the prefix would differ per benchmark and never produce a cache hit")
	}
	if !strings.Contains(taskSection, task) {
		t.Errorf("task section %q does not contain the task text", taskSection)
	}
	if !strings.HasPrefix(taskSection, "\n\n## Task\n\n") {
		t.Errorf("task section should start at the '## Task' boundary, got %q", taskSection[:min(40, len(taskSection))])
	}
}

// TestPromptParts_BaseIsStableAcrossBenchmarks is the property that makes
// caching pay: two different benchmarks in the same language must share a
// byte-identical base.
func TestPromptParts_BaseIsStableAcrossBenchmarks(t *testing.T) {
	a := &BenchmarkSpec{ID: "a", Languages: []string{"ailang"}, TaskPrompt: "Task A"}
	b := &BenchmarkSpec{ID: "b", Languages: []string{"ailang"}, TaskPrompt: "Task B totally different"}

	baseA, _ := a.PromptPartsForLanguage("ailang")
	baseB, _ := b.PromptPartsForLanguage("ailang")

	if baseA != baseB {
		t.Errorf("cacheable base differs between benchmarks (len %d vs %d) — nothing would ever hit the cache",
			len(baseA), len(baseB))
	}
	if len(baseA) == 0 {
		t.Error("cacheable base is empty — there is nothing to cache")
	}
}
