package quorum

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"testing"
)

var rolePool = []string{"gpt6-astra", "gpt6-1-sol", "gemini-4-argon", "oc-glm-5-3", "oc-kimi-k3"}

func TestQuorumRoleRefusal(t *testing.T) {
	for _, allAbsent := range []bool{false, true} {
		t.Run(fmt.Sprintf("allAbsent=%v", allAbsent), func(t *testing.T) {
			sel := SelectReviewers(rolePool, "opus", "doc.md", DefaultSeats)
			if !reflect.DeepEqual(sel.Refused, []string{"gpt6-astra"}) {
				t.Errorf("refused = %v, want astra only", sel.Refused)
			}
			for _, group := range [][]string{sel.Primary, sel.Reserve, sel.Benched} {
				if hasCall(group, "gpt6-astra") {
					t.Errorf("retired astra remains runnable: %+v", sel)
				}
			}
			up := map[string]bool{"gpt6-astra": true}
			for _, m := range rolePool[1:] {
				up[m] = !allAbsent
			}
			spy := &poolRunner{up: up}
			q := RunSeatedQuorum("doc.md", "body", "ts", sel, DefaultSeats, 0.3, nil, spy.run)
			if hasCall(spy.calls, "gpt6-astra") {
				t.Errorf("runner called refused astra: %v", spy.calls)
			}
			found := 0
			for _, o := range q.Reviewers {
				if o.Model == "gpt6-astra" {
					found++
					if o.Present || o.AbsentReason != ReasonNotAllowlisted || o.CostUSD != 0 || o.TokensIn != 0 || o.TokensOut != 0 || o.Result != nil {
						t.Errorf("refused artifact must be absent, not-allowlisted, zero spend: %+v", o)
					}
				}
			}
			if found != 1 {
				t.Errorf("refused artifact count = %d, want 1", found)
			}
			var notes []AbsentNote
			for _, note := range q.Synthesis.AbsentReviewers {
				if note.Model == "gpt6-astra" {
					notes = append(notes, note)
				}
			}
			if !reflect.DeepEqual(notes, []AbsentNote{{Model: "gpt6-astra", Reason: ReasonNotAllowlisted}}) {
				t.Errorf("synthesis must name refused astra: %+v", q.Synthesis.AbsentReviewers)
			}
			if allAbsent && q.Synthesis.Verdict != SynthBlocked {
				t.Errorf("all allowed reviewers absent must block: %+v", q.Synthesis)
			}
		})
	}
}

func TestQuorumRoleAuthorVendor(t *testing.T) {
	sel := SelectReviewers(rolePool, "codex:gpt-6.1-sol", "doc.md", DefaultSeats)
	if !reflect.DeepEqual(sel.Refused, []string{"gpt6-astra"}) {
		t.Errorf("author arm refused = %v, want astra only", sel.Refused)
	}
	if hasCall(sel.Benched, "gpt6-astra") {
		t.Errorf("refuse before benching: retired astra is recallable: %+v", sel)
	}
	if !reflect.DeepEqual(sel.Benched, []string{"gpt6-1-sol"}) {
		t.Errorf("only allowed author-vendor model may be benched: %+v", sel)
	}
	spy := &poolRunner{up: map[string]bool{"gpt6-astra": true, "gpt6-1-sol": true}}
	q := RunSeatedQuorum("doc.md", "body", "ts", sel, DefaultSeats, 0.3, nil, spy.run)
	if hasCall(spy.calls, "gpt6-astra") || !hasCall(spy.calls, "gpt6-1-sol") {
		t.Errorf("fallback must recall only allowed model: %v", spy.calls)
	}
	for _, o := range q.Reviewers {
		if o.Model == "gpt6-astra" && (o.Present || o.Tier != "" || o.AbsentReason != ReasonNotAllowlisted) {
			t.Errorf("refused author-vendor artifact: %+v", o)
		}
	}
}

func TestQuorumDefaultPoolControl(t *testing.T) {
	// Frozen CLI default literal; do not import or change design_quorum.go.
	defaults := []string{"gpt6-1-sol", "gemini-4-argon", "oc-glm-5-3", "oc-kimi-k3", ClaudeReviewerID}
	for _, author := range []string{"opus", "codex:gpt-6.1-sol", "gemini-4-argon", "oc-glm-5-3", "oc-kimi-k3", "unknown"} {
		for doc := 0; doc < 20; doc++ {
			for _, seats := range []int{0, 1, 3, 6} {
				key := fmt.Sprintf("design_docs/planned/control-%d.md", doc)
				got := SelectReviewers(defaults, author, key, seats)
				want := frozenBaseSelection(defaults, author, key, seats)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("default pool changed author=%s doc=%s seats=%d: got %+v want %+v", author, key, seats, got, want)
				}
			}
		}
	}
}

// frozenBaseSelection is the pre-allowlist algorithm, independent of the new
// selector and policy. Keep it frozen so the default-pool control detects drift.
func frozenBaseSelection(pool []string, author, docKey string, seats int) Selection {
	var sel Selection
	av := VendorOf(author)
	var eligible []string
	for _, m := range pool {
		if av != "" && VendorOf(m) == av {
			sel.Benched = append(sel.Benched, m)
		} else {
			eligible = append(eligible, m)
		}
	}
	if len(eligible) == 0 {
		return sel
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(docKey))
	off := int(h.Sum32() % uint32(len(eligible)))
	rotated := append(append([]string{}, eligible[off:]...), eligible[:off]...)
	seen := map[string]bool{}
	var rest []string
	for _, m := range rotated {
		v := VendorOf(m)
		if len(sel.Primary) < seats && (v == "" || !seen[v]) {
			sel.Primary = append(sel.Primary, m)
			seen[v] = true
		} else {
			rest = append(rest, m)
		}
	}
	for len(sel.Primary) < seats && len(rest) > 0 {
		sel.Primary, rest = append(sel.Primary, rest[0]), rest[1:]
	}
	sel.Reserve = rest
	return sel
}
