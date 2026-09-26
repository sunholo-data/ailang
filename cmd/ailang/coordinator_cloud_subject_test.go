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

// Measured 2026-09-13 on PRs #1145-#1149: every title was the raw payload,
//
//	[agent] design-doc-creator: {"workflow":"design-document-v1","project":"ail…
//
// even though subjectFromJSON existed and the payload carried a perfectly good
// `request`. By the time a directive reaches here it is rarely pure JSON — the
// skill path appends "Invoke the <skill> skill to complete this task." after it
// — and json.Unmarshal rejects trailing content outright ("invalid character
// 'I' after top-level value"), so the JSON arm returned "" and the first-line
// fallback printed the whole single-line blob.
//
// The JSON extractor has to survive whatever the wrapper wraps around it.
func TestSummarizeDirective_JSONWithTrailingWrapperText(t *testing.T) {
	// Exactly what buildSkillDirectiveWithConfig produces.
	directive := `{"workflow":"design-document-v1","project":"ailang","request":"Suppress cascading parser errors after the first failure."}

Invoke the design-doc-creator skill to complete this task.
Return DESIGN_DOC_PATH: followed by the path.`

	got := summarizeDirective(directive)
	if strings.Contains(got, "{") || strings.Contains(got, "workflow") {
		t.Fatalf("the payload leaked into the subject: %q", got)
	}
	if !strings.HasPrefix(got, "Suppress cascading parser errors") {
		t.Errorf("want the request field, got %q", got)
	}

	// And the whole point: the PR title a person reads.
	title := agentPRTitle("design-doc-creator", "task-087ae04b", directive)
	if strings.Contains(title, "design-document-v1") {
		t.Errorf("title still carries the payload: %q", title)
	}
	if !strings.Contains(title, "Suppress cascading parser errors") {
		t.Errorf("title should say what the change is, got %q", title)
	}
}

// Trailing content must not turn NON-JSON prose into a bad extraction either.
func TestSummarizeDirective_BraceProseIsStillProse(t *testing.T) {
	got := summarizeDirective("{ this is not json } and never was")
	if got != "{ this is not json } and never was" {
		t.Errorf("prose that merely starts with a brace must survive intact, got %q", got)
	}
}

// TestTaskSubject_PrefersTheCarriedTitle is the third shape of one mistake.
//
// The directive reaching the job is the TEMPLATE-WRAPPED prompt, so its first
// line is harness boilerplate. Measured 2026-09-14, PR #62:
//
//	[agent] pkg-sunholo-testing-utils: You are an autonomous AIL…
//
// The 2026-09-13 fix handled the JSON shape of the same error. Deriving a
// description from a prompt yields a new wrong answer every time the prompt
// template changes, so the title is now carried instead.
func TestTaskSubject_PrefersTheCarriedTitle(t *testing.T) {
	wrapped := "You are an autonomous AILANG package agent.\n\nTask:\n`sunholo/testing_utils` 0.1.1 does not compile."
	t.Setenv("AILANG_TASK_TITLE", "sunholo/testing_utils 0.1.1: ++ on strings — does not compile on v0.38.5")

	got := taskSubject(wrapped)
	if strings.Contains(got, "autonomous") {
		t.Errorf("the subject names the harness, not the change: %q", got)
	}
	if !strings.Contains(got, "testing_utils") {
		t.Errorf("the carried title must win: %q", got)
	}
}

// With no carried title — an older dispatcher — derivation is still the
// fallback, unchanged.
func TestTaskSubject_FallsBackToDerivationWhenNoTitleIsCarried(t *testing.T) {
	t.Setenv("AILANG_TASK_TITLE", "")
	got := taskSubject("Fix the parser cascade\n\nmore context here")
	if got != "Fix the parser cascade" {
		t.Errorf("derivation must still work for a dispatcher that sends no title: %q", got)
	}
}

// A carried title is human-written and occasionally long. Unbounded subjects
// were the original bug, so the same budget applies to both sources.
func TestTaskSubject_CarriedTitleIsBounded(t *testing.T) {
	t.Setenv("AILANG_TASK_TITLE", strings.Repeat("words that go on ", 20))
	got := taskSubject("")
	if len([]rune(got)) > directiveSubjectMax+1 { // +1 for the ellipsis
		t.Errorf("a carried title must be bounded like a derived one: %d runes", len([]rune(got)))
	}
}

// Whitespace-only is not a title.
func TestTaskSubject_BlankCarriedTitleDoesNotWin(t *testing.T) {
	t.Setenv("AILANG_TASK_TITLE", "   \n\t ")
	if got := taskSubject("Real directive first line"); got != "Real directive first line" {
		t.Errorf("a blank title must not beat a usable directive: %q", got)
	}
}
