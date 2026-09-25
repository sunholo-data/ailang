package quorum

import (
	"reflect"
	"strings"
	"testing"
)

var roster = []string{"gpt6-astra", "gemini-3-1-pro", "oc-glm-5-3", ClaudeReviewerID}

func TestSeatReviewers_AuthorsVendorSitsOut(t *testing.T) {
	cases := []struct {
		author      string
		wantBenched []string
	}{
		{"claude:claude-opus-5-5", []string{ClaudeReviewerID}},
		{"opus", []string{ClaudeReviewerID}},
		{"codex:gpt-6-astra", []string{"gpt6-astra"}},
		{"pi:ollama/glm-5.3:cloud", []string{"oc-glm-5-3"}},
		{"gemini-3-1-pro", []string{"gemini-3-1-pro"}},
		{"pi:ollama/deepseek-v4-flash:0731-cloud", nil}, // no deepseek seat: all four review
		{"something-unrecognised", nil},
	}
	for _, c := range cases {
		seated, benched := SeatReviewers(roster, c.author)
		if !reflect.DeepEqual(benched, c.wantBenched) {
			t.Errorf("author %q: benched %v, want %v", c.author, benched, c.wantBenched)
		}
		if len(seated)+len(benched) != len(roster) {
			t.Errorf("author %q: seats lost: %v + %v", c.author, seated, benched)
		}
		for _, m := range seated {
			if av := VendorOf(c.author); av != "" && VendorOf(m) == av {
				t.Errorf("author %q: %s reviews its own vendor's doc", c.author, m)
			}
		}
	}
}

func TestVendorOf_LaneIdsResolveByTheirModel(t *testing.T) {
	if v := VendorOf("pi:openrouter/moonshotai/kimi-k3"); v != "moonshot" {
		t.Errorf("pi lane vendor = %q, want moonshot", v)
	}
	for _, id := range []string{"pi:ollama/foo", "opencode:bar", "motoko:baz"} {
		if v := VendorOf(id); v != "" {
			t.Errorf("harness prefix alone read as vendor %q in %q", v, id)
		}
	}
	if v := VendorOf("codex:gpt-6-astra"); v != "openai" {
		t.Errorf("codex lane vendor = %q, want openai", v)
	}
}

func seat(model string, present bool, v Verdict) *ReviewerOutcome {
	if !present {
		return &ReviewerOutcome{Model: model, AbsentReason: ReasonBudget}
	}
	return &ReviewerOutcome{Model: model, Present: true, Result: &ReviewResult{Verdict: v, StrongestObjection: "x"}}
}

func TestRecallBenched_OnlyWhenEverySeatedReviewerIsAbsent(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{seat("a", false, ""), seat("b", true, VerdictPass)}}
	ran := RecallBenched(q, []string{ClaudeReviewerID}, "d", "b", 0.3, func(string, string, string, float64) *ReviewerOutcome {
		t.Fatal("benched seat recalled although an independent reviewer was present")
		return nil
	})
	if ran {
		t.Fatal("RecallBenched reported running")
	}
}

func TestRecallBenched_ControllerVoteDoesNotCountAsIndependent(t *testing.T) {
	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "fine"}
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{seat("a", false, ""), seat("b", false, "")}, ControllerInSession: ctrl}
	q.Synthesis = synthesize(q.Reviewers, ctrl)
	ran := RecallBenched(q, []string{ClaudeReviewerID}, "d", "b", 0.3, func(m, _, _ string, _ float64) *ReviewerOutcome {
		return seat(m, true, VerdictReject)
	})
	if !ran {
		t.Fatal("benched seat not recalled with every independent reviewer absent")
	}
	last := q.Reviewers[len(q.Reviewers)-1]
	if last.Tier != TierAuthorVendor {
		t.Errorf("recalled seat tier = %q, want %q", last.Tier, TierAuthorVendor)
	}
	if q.Synthesis.Verdict != SynthBlocked {
		t.Errorf("a recalled reject must block, got %s", q.Synthesis.Verdict)
	}
	if !strings.Contains(MarkdownBlock(q), "SAME VENDOR AS THE AUTHOR") {
		t.Error("markdown does not label the recalled seat")
	}
}

func TestRecallBenched_NothingBenchedMeansZeroSignalStillBlocks(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{seat("a", false, "")}}
	q.Synthesis = synthesize(q.Reviewers, nil)
	if RecallBenched(q, nil, "d", "b", 0.3, nil) || q.Synthesis.Verdict != SynthBlocked {
		t.Fatalf("synthesis = %s, want blocked with nothing to recall", q.Synthesis.Verdict)
	}
}
