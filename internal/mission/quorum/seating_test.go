package quorum

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

var pool = []string{"gpt6-astra", "gemini-3-1-pro", "oc-glm-5-3", "oc-kimi-k3", ClaudeReviewerID}

func TestSelectReviewers_AuthorsVendorNeverSeated(t *testing.T) {
	cases := []struct {
		author      string
		wantBenched []string
	}{
		{"claude:claude-opus-5-5", []string{ClaudeReviewerID}},
		{"opus", []string{ClaudeReviewerID}},
		{"codex:gpt-6-astra", []string{"gpt6-astra"}},
		{"pi:ollama/glm-5.3:cloud", []string{"oc-glm-5-3"}},
		{"pi:openrouter/moonshotai/kimi-k3", []string{"oc-kimi-k3"}},
		{"pi:ollama/deepseek-v4-flash:0731-cloud", nil},
		{"something-unrecognised", nil},
	}
	for _, c := range cases {
		for doc := 0; doc < 20; doc++ {
			sel := SelectReviewers(pool, c.author, fmt.Sprintf("doc-%d.md", doc), DefaultSeats)
			if !reflect.DeepEqual(sel.Benched, c.wantBenched) {
				t.Fatalf("author %q: benched %v, want %v", c.author, sel.Benched, c.wantBenched)
			}
			if len(sel.Primary) != DefaultSeats {
				t.Fatalf("author %q: %d seats, want %d", c.author, len(sel.Primary), DefaultSeats)
			}
			if len(sel.Primary)+len(sel.Reserve)+len(sel.Benched) != len(pool) {
				t.Fatalf("author %q: pool members lost: %+v", c.author, sel)
			}
			for _, m := range append(append([]string{}, sel.Primary...), sel.Reserve...) {
				if av := VendorOf(c.author); av != "" && VendorOf(m) == av {
					t.Fatalf("author %q: %s would review its own vendor's doc", c.author, m)
				}
			}
		}
	}
}

func TestSelectReviewers_StablePerDocAndRotatesAcrossDocs(t *testing.T) {
	a := SelectReviewers(pool, "opus", "design_docs/planned/x.md", 3)
	b := SelectReviewers(pool, "opus", "design_docs/planned/x.md", 3)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same doc seated differently across rounds: %v vs %v", a.Primary, b.Primary)
	}
	seatedEver := map[string]bool{}
	for i := 0; i < 40; i++ {
		for _, m := range SelectReviewers(pool, "opus", fmt.Sprintf("d%d.md", i), 3).Primary {
			seatedEver[m] = true
		}
	}
	for _, m := range pool {
		if m != ClaudeReviewerID && !seatedEver[m] {
			t.Errorf("%s never seated across 40 docs — the pool does not rotate", m)
		}
	}
}

func TestSelectReviewers_PrefersDistinctVendors(t *testing.T) {
	p := []string{"gpt6-astra", "gpt5-6-sol", "gemini-3-1-pro", "oc-glm-5-3"}
	for i := 0; i < 20; i++ {
		sel := SelectReviewers(p, "opus", fmt.Sprintf("d%d.md", i), 3)
		vendors := map[string]bool{}
		for _, m := range sel.Primary {
			if vendors[VendorOf(m)] {
				t.Fatalf("two %s seats (%v) while a distinct vendor was available", VendorOf(m), sel.Primary)
			}
			vendors[VendorOf(m)] = true
		}
	}
}

func seat(model string, present bool, v Verdict) *ReviewerOutcome {
	if !present {
		return &ReviewerOutcome{Model: model, AbsentReason: ReasonBudget}
	}
	return &ReviewerOutcome{Model: model, Present: true, Result: &ReviewResult{Verdict: v, StrongestObjection: "x"}}
}

// poolRunner answers for the models in up; everyone else is absent. It records calls.
type poolRunner struct {
	mu    sync.Mutex
	up    map[string]bool
	calls []string
}

func (f *poolRunner) run(m, _, _ string, _ float64) *ReviewerOutcome {
	f.mu.Lock()
	f.calls = append(f.calls, m)
	f.mu.Unlock()
	return seat(m, f.up[m], VerdictPass)
}

func TestRunSeatedQuorum_ReplacesAbsentSeatsFromTheReserve(t *testing.T) {
	sel := Selection{Primary: []string{"a", "b", "c"}, Reserve: []string{"d", "e"}, Benched: []string{"z"}}
	f := &poolRunner{up: map[string]bool{"a": true, "c": true, "d": true, "e": true, "z": true}}
	q := RunSeatedQuorum("doc", "body", "ts", sel, 3, 0.3, nil, f.run)
	if n := presentCount(q); n != 3 {
		t.Fatalf("present = %d, want 3 (b replaced by d)", n)
	}
	for _, m := range f.calls {
		if m == "e" || m == "z" {
			t.Errorf("%s was called although three seats had already answered", m)
		}
	}
}

func TestRunSeatedQuorum_RecallsTheAuthorsVendorOnlyWhenNobodyAnswers(t *testing.T) {
	sel := Selection{Primary: []string{"a", "b"}, Reserve: []string{"c"}, Benched: []string{"z"}}
	f := &poolRunner{up: map[string]bool{"z": true}}
	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "the controller is Anthropic too"}
	q := RunSeatedQuorum("doc", "body", "ts", sel, 3, 0.3, ctrl, f.run)
	last := q.Reviewers[len(q.Reviewers)-1]
	if last.Model != "z" || last.Tier != TierAuthorVendor {
		t.Fatalf("last reviewer = %s tier %q, want the benched seat labelled %q", last.Model, last.Tier, TierAuthorVendor)
	}
	if !strings.Contains(MarkdownBlock(q), "SAME VENDOR AS THE AUTHOR") {
		t.Error("markdown does not label the recalled seat")
	}
	if len(q.Synthesis.AbsentReviewers) != 3 {
		t.Errorf("absent = %v, want a, b, c named", q.Synthesis.AbsentReviewers)
	}
}

func TestRunSeatedQuorum_OneIndependentVerdictKeepsTheAuthorBenched(t *testing.T) {
	sel := Selection{Primary: []string{"a", "b"}, Benched: []string{"z"}}
	f := &poolRunner{up: map[string]bool{"a": true, "z": true}}
	RunSeatedQuorum("doc", "body", "ts", sel, 3, 0.3, nil, f.run)
	for _, m := range f.calls {
		if m == "z" {
			t.Fatal("author's vendor recalled although an independent reviewer answered")
		}
	}
}

func TestRunSeatedQuorum_NothingAnswersBlocks(t *testing.T) {
	f := &poolRunner{up: map[string]bool{}}
	q := RunSeatedQuorum("doc", "body", "ts", Selection{Primary: []string{"a"}, Benched: []string{"z"}}, 3, 0.3, nil, f.run)
	if q.Synthesis.Verdict != SynthBlocked {
		t.Fatalf("synthesis = %s, want blocked on zero signal", q.Synthesis.Verdict)
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
