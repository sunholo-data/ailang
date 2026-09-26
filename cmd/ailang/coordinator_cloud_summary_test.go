package main

import (
	"strings"
	"testing"
)

// The conclusion is at the END of a transcript — the beginning is the agent
// reading CLAUDE.md. Taking the head would reliably capture the least useful
// part.
func TestCompletionSummary_KeepsTheConclusion(t *testing.T) {
	transcript := strings.Repeat("[TOOL] reading files\n", 400) +
		"Execution is blocked by the mandatory sprint-executor gate:\n" +
		".ailang/state/sprints/sprint_M-OPENROUTER-EU-ROUTING.json is missing.\n" +
		"**IMPLEMENTATION_COMPLETE:** `false`\n"

	got := completionSummary(transcript)
	if !strings.Contains(got, "is missing") {
		t.Errorf("the conclusion was dropped:\n%s", got)
	}
	if !strings.Contains(got, "IMPLEMENTATION_COMPLETE") {
		t.Errorf("the output markers sit under the conclusion and must survive:\n%s", got)
	}
	if len(got) > completionSummaryMax+120 { // + the truncation notice
		t.Errorf("summary is %d bytes; it rides on EVERY completion", len(got))
	}
}

// A silently truncated explanation is its own small lie.
func TestCompletionSummary_SaysWhenItTruncated(t *testing.T) {
	got := completionSummary(strings.Repeat("x y z\n", 5000))
	if !strings.Contains(got, "truncated") {
		t.Errorf("truncation must be visible:\n%s", got[:80])
	}
}

// Short transcripts pass through whole — no notice, no cut.
func TestCompletionSummary_ShortTranscriptIsVerbatim(t *testing.T) {
	const s = "Done. Added two tests and fixed the off-by-one."
	if got := completionSummary(s); got != s {
		t.Errorf("got %q, want the transcript verbatim", got)
	}
}

func TestCompletionSummary_EmptyIsEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t\n"} {
		if got := completionSummary(in); got != "" {
			t.Errorf("completionSummary(%q) = %q, want empty", in, got)
		}
	}
}

// One enormous line with no newline anywhere: keep it rather than returning
// nothing, because a single-line answer is still an answer.
func TestCompletionSummary_SingleHugeLineSurvives(t *testing.T) {
	got := completionSummary(strings.Repeat("a", completionSummaryMax*3))
	if strings.TrimSpace(strings.TrimPrefix(got, "…(transcript truncated; full text at the artifact path)")) == "" {
		t.Error("a newline-free transcript must still yield text")
	}
}
