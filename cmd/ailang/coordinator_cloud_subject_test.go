package main

import (
	"strings"
	"testing"
)

// The subject must be a PURE function of the directive: same input, same output,
// no model and no network. That is the property the old boilerplate was
// protecting, and none of it is given up here — only the uninformative part.

func TestSummarizeDirective_TheRealCases(t *testing.T) {
	tests := []struct {
		name      string
		directive string
		want      string
	}{
		{
			// PR #1143, the case that prompted this: a structured request whose
			// payload was printed verbatim into the subject line.
			name:      "structured request pulls out the human field",
			directive: `{"project":"ailang","target":"ailang","request":"Route OpenRouter traffic through the EU region"}`,
			want:      "Route OpenRouter traffic through the EU region",
		},
		{
			// PR #1033: prose already reads well and must be left alone.
			name:      "prose passes through",
			directive: "`ailang lock` stamps a wall-clock generated_at",
			want:      "`ailang lock` stamps a wall-clock generated_at",
		},
		{
			name:      "an already-prefixed directive is not double-prefixed",
			directive: "Task task-88a9fa95: the lockfile is not deterministic",
			want:      "the lockfile is not deterministic",
		},
		{
			name:      "only the first line becomes the subject",
			directive: "Fix the retry backoff\n\nThe daemon currently sleeps uncancellably, which\nmakes shutdown take 8 seconds.",
			want:      "Fix the retry backoff",
		},
		{
			name:      "whitespace is collapsed",
			directive: "  Fix   the\tretry   backoff  ",
			want:      "Fix the retry backoff",
		},
		{
			name:      "empty yields empty, so the caller can fall back",
			directive: "   \n  ",
			want:      "",
		},
		{
			// A directive that merely starts with a brace is prose.
			name:      "invalid JSON is treated as prose",
			directive: `{this is not json, it is a sentence}`,
			want:      `{this is not json, it is a sentence}`,
		},
		{
			// Pretty-printed JSON would yield "{" if lines were split first.
			name: "pretty-printed JSON still finds the field",
			directive: `{
  "workflow": "design-document-v1",
  "request": "Scope design requests per project"
}`,
			want: "Scope design requests per project",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizeDirective(tt.directive); got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeDirective_IsDeterministic(t *testing.T) {
	// Go randomises map iteration order, so a JSON directive carrying SEVERAL
	// candidate fields would produce a different subject per run if the key were
	// chosen by iteration rather than by the fixed precedence list. Same commit,
	// different subject, is a worse failure than a dull one.
	directive := `{"goal":"g","task":"t","summary":"s","title":"ti","request":"r","description":"d"}`
	first := summarizeDirective(directive)
	if first != "r" {
		t.Fatalf("expected the highest-precedence key (request), got %q", first)
	}
	for i := 0; i < 200; i++ {
		if got := summarizeDirective(directive); got != first {
			t.Fatalf("run %d gave %q, want the stable %q", i, got, first)
		}
	}
}

func TestSummarizeDirective_IsBounded(t *testing.T) {
	// The old subject had no length bound at all — the entire prompt went on one
	// line. Git's convention is 72; the budget leaves room for the prefix.
	long := "Redesign the coordinator approval pipeline so that every handoff edge " +
		"records its own provenance and the evaluator verdict travels with the approval record"
	got := summarizeDirective(long)
	if len([]rune(got)) > directiveSubjectMax+1 { // +1 for the ellipsis
		t.Errorf("subject is %d runes, want <= %d: %q", len([]rune(got)), directiveSubjectMax+1, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("a truncated subject must show it was truncated: %q", got)
	}
	// Never mid-word.
	trimmed := strings.TrimSuffix(got, "…")
	if strings.HasPrefix(long, trimmed) && len(trimmed) < len(long) {
		next := long[len(trimmed)]
		if next != ' ' {
			t.Errorf("truncated mid-word before %q: %q", string(next), got)
		}
	}
}

func TestStripTaskPrefix_LeavesProseAlone(t *testing.T) {
	// "Task ..." prose containing a colon much later is a sentence, not a
	// prefix. Stripping it would silently delete the first clause.
	prose := "Task scheduling is broken when the queue drains faster than the producer: the worker exits"
	if got := stripTaskPrefix(prose); got != prose {
		t.Errorf("prose was mangled:\n got  %q\n want %q", got, prose)
	}
	if got := stripTaskPrefix("Task task-abc123: the real subject"); got != "the real subject" {
		t.Errorf("a genuine prefix must be stripped, got %q", got)
	}
}

func TestAgentPRTitle_FallsBackToTheTaskID(t *testing.T) {
	// Descriptive when it can be…
	got := agentPRTitle("design-doc-creator", "task-080f4657", `{"request":"EU routing for OpenRouter"}`)
	if got != "[agent] design-doc-creator: EU routing for OpenRouter" {
		t.Errorf("got %q", got)
	}
	// …and never worse than what it replaced when it cannot.
	got = agentPRTitle("design-doc-creator", "task-080f4657", "")
	if got != "[agent] design-doc-creator: task-080f4657" {
		t.Errorf("fallback must keep the task id, got %q", got)
	}
}

func TestAgentCommitSubject_FallsBackToTheTaskID(t *testing.T) {
	if got := agentCommitSubject("task-1", "Fix the thing"); got != "Fix the thing" {
		t.Errorf("got %q", got)
	}
	if got := agentCommitSubject("task-1", ""); got != "Task task-1" {
		t.Errorf("fallback got %q", got)
	}
}
