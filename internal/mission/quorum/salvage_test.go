package quorum

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/ai"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

const F1 = `{"verdict":"reject","strongest_objection":"the table row a \| b is unescaped","catch":"x","proposed_fix":""}`
const F2 = `{"verdict":"reject","strongest_objection":"asserts \"GOOS\" absent","catch":"the test greps "GOARCH" in ci.yml","proposed_fix":""}`
const F3 = `{"verdict":"reject","strongest_objection":"cut off mid-sent`

func TestSalvageDiagnostics(t *testing.T) {
	t.Run("offset", func(t *testing.T) {
		for i, raw := range []string{F1, F2, F3} {
			_, err := ParseReviewResult(raw)
			want := fmt.Sprintf("at byte %d", []int{61, 94, 59}[i])
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("F%d: error %v lacks %q", i+1, err, want)
			}
		}
	})
	t.Run("centred_window", func(t *testing.T) {
		raw := strings.Replace(F2, "asserts", strings.Repeat("a", 300)+"asserts", 1)
		_, err := ParseReviewResult(raw)
		if err == nil || !strings.Contains(err.Error(), "GOARCH") || !strings.Contains(err.Error(), "at byte 394") {
			t.Fatalf("break past byte 300 not located: %v", err)
		}
	})
	t.Run("trimmed_fences", func(t *testing.T) {
		_, err := ParseReviewResult(" \n```json\n" + F1 + "\n```\n ")
		if err == nil || !strings.Contains(err.Error(), "at byte 61 of 108") {
			t.Fatalf("offset not relative to parser bytes: %v", err)
		}
	})
	t.Run("utf8_and_clamping", func(t *testing.T) {
		// A syntax break within a long multibyte value puts both window ends in runes.
		b := []byte(`{"x":"` + strings.Repeat("界", 160) + `"x"` + strings.Repeat("界", 160) + `"}`)
		for _, off := range []int64{1, 248, int64(len(b))} {
			got := describeJSONError(&json.SyntaxError{Offset: off}, b)
			var a, z int
			marker := strings.Index(got, "window [")
			if marker < 0 {
				t.Fatalf("missing window: %s", got)
			}
			if _, err := fmt.Sscanf(got[marker:], "window [%d:%d]", &a, &z); err != nil {
				t.Fatal(err)
			}
			if a < 0 || z > len(b) || a > z || !utf8.Valid(b[a:z]) {
				t.Fatalf("invalid UTF-8 window [%d:%d]", a, z)
			}
			start, end := max(0, int(off)-120), min(len(b), int(off)+120)
			for start > 0 && !utf8.RuneStart(b[start]) {
				start--
			}
			for end < len(b) && !utf8.RuneStart(b[end]) {
				end--
			}
			if a != start || z != end {
				t.Fatalf("window [%d:%d], want [%d:%d]", a, z, start, end)
			}
			quoted := got[strings.Index(got, "]: ")+3 : len(got)-1]
			decoded, err := strconv.Unquote(quoted)
			if err != nil || decoded != string(b[a:z]) {
				t.Fatalf("window did not render its bytes: %q %v", quoted, err)
			}
		}
	})
	t.Run("non_syntax", func(t *testing.T) {
		b := []byte(strings.Repeat("a", 250))
		got := describeJSONError(errors.New("type mismatch"), b)
		if got != fmt.Sprintf("type mismatch (raw: %.200q)", b) {
			t.Fatalf("fallback changed: %s", got)
		}
	})
	t.Run("claude_envelope", func(t *testing.T) {
		raw := `{"result":"` + strings.Repeat("a", 300) + `"GOARCH"}`
		stubClaude(t, raw, nil)
		got := RunClaudeSubscriptionReviewer(ClaudeReviewerModel, "doc.md", "body")
		if !strings.Contains(got.Err, "at byte 313") || !strings.Contains(got.Err, "window [") || !strings.Contains(got.Err, "GOARCH") {
			t.Fatalf("envelope diagnostic: %s", got.Err)
		}
	})
}

func TestSalvageVerdict(t *testing.T) {
	cases := []struct {
		name, raw string
		verdict   Verdict
		objection string
		offset    int64
	}{
		{"fixtures/F1", F1, VerdictReject, "", 61},
		{"fixtures/F2", F2, VerdictReject, `asserts "GOOS" absent`, 94},
		{"fixtures/F3", F3, VerdictReject, "", 59},
		{"unconfirmed", `{"verdict":"reject","strongest_objection":"the test greps "GOARCH" x"}`, VerdictReject, "", 72},
		{"nested_only", `{"x":{"verdict":"reject"}}`, "", "", -1},
		{"after_nested_object", `{"meta":{"k":1},"verdict":"reject","strongest_objection":"after nested","catch":"x`, VerdictReject, "after nested", 79},
		{"after_nested_array", `{"notes":["a",{"verdict":"reject"}],"verdict":"reject","strongest_objection":"top","catch":"x`, VerdictReject, "top", 88},
		{"prose/plain", `I would reject this`, "", "", 0},
		{"prose/object", `Prose {"verdict":"reject"}`, "", "", 0},
		{"prose/array", `[{"verdict":"reject"}]`, "", "", -1},
		{"key_value_state", `{"other":"verdict","reject":"pass","verdict":"pass","catch":"x`, VerdictPass, "", 59},
		{"duplicates_and_pass/broken_pass", `{"verdict":"pass","catch":"x`, VerdictPass, "", 26},
		{"duplicates_and_pass/pass_reject", `{"verdict":"pass","verdict":"reject","catch":"x`, VerdictReject, "", 45},
		{"duplicates_and_pass/reject_pass", `{"verdict":"reject","verdict":"pass","catch":"x`, VerdictReject, "", 45},
		{"duplicates_and_pass/unknown", `{"verdict":"REJECT"}`, "", "", -1},
		{"duplicates_and_pass/non_string", `{"verdict":true}`, "", "", -1},
		{"gate_violation", `{"verdict":"reject","strongest_objection":"","catch":""}`, VerdictReject, "", -1},
		{"confirmed_at_root_close", `{"verdict":"reject","strongest_objection":"root"}`, VerdictReject, "root", -1},
		{"fenced", " \n```json\n" + F2 + "\n```\n ", VerdictReject, `asserts "GOOS" absent`, 94},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := salvageVerdict(c.raw)
			if c.verdict == "" {
				if got != nil {
					t.Fatalf("wanted nil, got %+v", got)
				}
				return
			}
			if got == nil || got.Verdict != c.verdict || got.StrongestObjection != c.objection {
				t.Fatalf("salvage = %+v, want verdict=%s objection=%q", got, c.verdict, c.objection)
			}
			b := []byte(stripCodeFences(strings.TrimSpace(c.raw)))
			var dummy any
			err := json.Unmarshal(b, &dummy)
			var se *json.SyntaxError
			off := int64(-1)
			if errors.As(err, &se) {
				off = se.Offset
			}
			if got.Offset != off {
				t.Fatalf("offset=%d, want Unmarshal offset %d", got.Offset, off)
			}
			if strings.HasPrefix(c.name, "fixtures/") && got.Offset != c.offset {
				t.Fatalf("fixture offset=%d, want %d", got.Offset, c.offset)
			}
		})
	}
}

func TestSalvageSynthesis(t *testing.T) {
	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "controller passes"}
	t.Run("reject", func(t *testing.T) {
		for _, objection := range []string{"real objection", ""} {
			absent := &ReviewerOutcome{Model: "B", AbsentReason: ReasonInvalid, Salvaged: &SalvagedVerdict{Verdict: VerdictReject, StrongestObjection: objection}}
			absent.CostUSD, absent.TokensIn, absent.TokensOut = 0.01, 100, 20
			s := synthesize([]*ReviewerOutcome{present("A", VerdictPass, "concern"), absent}, ctrl)
			want := "B [salvaged from invalid response]: " + objection
			if objection == "" {
				want = "B [salvaged from invalid response]: strongest_objection unrecoverable, see reviewer error"
			}
			if s.Verdict != SynthBlocked || len(s.BlockingObjections) != 1 || s.BlockingObjections[0] != want {
				t.Fatalf("salvaged reject not folded: %+v, want objection %q", s, want)
			}
			b, err := json.Marshal(s.AbsentReviewers)
			if err != nil || !strings.Contains(string(b), `"salvaged_verdict":"reject"`) {
				t.Fatalf("absent audit lost: %s %v", b, err)
			}
			if s.TotalCostUSD < 0.01 || s.TotalTokensIn < 100 || s.TotalTokensOut < 20 {
				t.Fatalf("absent accounting lost: %+v", s)
			}
		}
	})
	t.Run("pass_not_signal", func(t *testing.T) {
		absent := &ReviewerOutcome{Model: "B", AbsentReason: ReasonInvalid, Salvaged: &SalvagedVerdict{Verdict: VerdictPass}}
		s := synthesize([]*ReviewerOutcome{absent}, ctrl)
		if s.Verdict != SynthBlocked || len(s.BlockingObjections) != 1 || s.BlockingObjections[0] != zeroSignalObjection {
			t.Fatalf("salvaged pass became signal: %+v", s)
		}
		b, _ := json.Marshal(s.AbsentReviewers)
		if !strings.Contains(string(b), `"salvaged_verdict":"pass"`) {
			t.Fatalf("pass audit missing: %s", b)
		}
		if presentCount(&QuorumResult{Reviewers: []*ReviewerOutcome{absent}}) != 0 {
			t.Fatal("salvaged pass became a seat")
		}
		s = synthesize([]*ReviewerOutcome{present("A", VerdictPass, "concern"), absent}, ctrl)
		if s.Verdict != SynthProceed || len(s.BlockingObjections) != 0 {
			t.Fatalf("audit pass became veto: %+v", s)
		}
	})
}

const salvageValidPass = `{"verdict":"pass","strongest_objection":"closest concern","catch":"check it"}`
const salvageGateViolation = `{"verdict":"reject","strongest_objection":"","catch":"","proposed_fix":""}`

func salvageProductionOutcome(model, raw, finish string) *ReviewerOutcome {
	return runReviewerWith(&stubCaller{raw: raw, resp: &ai.Response{InputTokens: 100, OutputTokens: 20, FinishReason: finish}}, cheapModel(), &ReviewerOutcome{Model: model}, "doc.md", "body", DefaultMaxCostUSD)
}

func assertSalvagedAbsent(t *testing.T, got *ReviewerOutcome, v Verdict) {
	t.Helper()
	if got.Present || got.AbsentReason != ReasonInvalid || got.Result != nil || got.Salvaged == nil || got.Salvaged.Verdict != v {
		t.Fatalf("invalid outcome lost salvage or became present: %+v", got)
	}
}

func TestSalvageProduction(t *testing.T) {
	for _, c := range []struct{ name, raw, finish string }{
		{"plain", F2, ""}, {"truncated", F3, "length"}, {"gate_violation", salvageGateViolation, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := salvageProductionOutcome("B", c.raw, c.finish)
			assertSalvagedAbsent(t, got, VerdictReject)
			if c.finish == "length" && !strings.Contains(got.Err, "finish_reason=length") {
				t.Fatalf("lost explicit truncation: %s", got.Err)
			}
			if c.name == "gate_violation" && got.Salvaged.Offset != -1 {
				t.Fatalf("gate offset=%d, want -1", got.Salvaged.Offset)
			}
			runner := func(model, _, _ string, _ float64) *ReviewerOutcome {
				if model == "B" {
					return salvageProductionOutcome(model, c.raw, c.finish)
				}
				return salvageProductionOutcome(model, salvageValidPass, "")
			}
			q := RunSeatedQuorum("doc.md", "body", "ts", Selection{Primary: []string{"A", "B", "C"}, Reserve: []string{"D"}}, 3, DefaultMaxCostUSD, &ControllerReview{Verdict: VerdictPass}, runner)
			if q.Synthesis.Verdict != SynthBlocked || len(q.Reviewers) != 4 || presentCount(q) != 3 || q.Reviewers[3].Model != "D" || !q.Reviewers[3].Present {
				t.Fatalf("seated veto/refill lost: synthesis=%+v reviewers=%+v", q.Synthesis, q.Reviewers)
			}
			if len(q.Synthesis.AbsentReviewers) != 1 || q.Synthesis.AbsentReviewers[0].Model != "B" || q.Synthesis.AbsentReviewers[0].Reason != ReasonInvalid {
				t.Fatalf("B absence lost: %+v", q.Synthesis.AbsentReviewers)
			}
			assertSalvagedAbsent(t, q.Reviewers[1], VerdictReject)
		})
	}
	t.Run("pass_refills_seat", func(t *testing.T) {
		runner := func(model, _, _ string, _ float64) *ReviewerOutcome {
			if model == "B" {
				return salvageProductionOutcome(model, `{"verdict":"pass","catch":"x`, "")
			}
			return salvageProductionOutcome(model, salvageValidPass, "")
		}
		q := RunSeatedQuorum("doc.md", "body", "ts", Selection{Primary: []string{"A", "B", "C"}, Reserve: []string{"D"}}, 3, DefaultMaxCostUSD, nil, runner)
		assertSalvagedAbsent(t, q.Reviewers[1], VerdictPass)
		if q.Synthesis.Verdict != SynthProceed || presentCount(q) != 3 || len(q.Reviewers) != 4 || q.Reviewers[3].Model != "D" {
			t.Fatalf("audit pass counted as seat or veto: %+v", q)
		}
	})
}

func TestSalvageClaude(t *testing.T) {
	t.Run("inner_verdict", func(t *testing.T) {
		envelope, err := json.Marshal(struct {
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
		}{false, F2})
		if err != nil {
			t.Fatal(err)
		}
		stubClaude(t, string(envelope), nil)
		got := RunClaudeSubscriptionReviewer(ClaudeReviewerModel, "doc.md", "body")
		assertSalvagedAbsent(t, got, VerdictReject)
		if got.Salvaged.StrongestObjection != `asserts "GOOS" absent` || got.Salvaged.Offset != 94 {
			t.Fatalf("inner salvage wrong: %+v", got.Salvaged)
		}
	})
	t.Run("envelope", func(t *testing.T) {
		stubClaude(t, `{"verdict":"reject","result":"x`, nil)
		got := RunClaudeSubscriptionReviewer(ClaudeReviewerModel, "doc.md", "body")
		if got.Present || got.Result != nil || got.AbsentReason != ReasonInvalid || got.Salvaged != nil || !strings.Contains(got.Err, "at byte 31") || !strings.Contains(got.Err, "window [") {
			t.Fatalf("transport envelope treated as verdict or undiagnosed: %+v", got)
		}
	})
}

func TestSalvageCompatibility(t *testing.T) {
	t.Run("omitempty", func(t *testing.T) {
		for _, value := range []any{&ReviewerOutcome{Model: "B", AbsentReason: ReasonBudget}, AbsentNote{Model: "B", Reason: ReasonBudget}} {
			b, err := json.Marshal(value)
			if err != nil || strings.Contains(string(b), "salvaged") {
				t.Fatalf("nil salvage changed old JSON: %s %v", b, err)
			}
		}
		old := `{"doc":"old.md","iso_ts":"ts","reviewers":[{"model":"A","cost_usd":0.01,"tokens_in":10,"tokens_out":2,"present":true,"result":{"verdict":"pass","strongest_objection":"concern","catch":"c"},"landed":null},{"model":"B","cost_usd":0,"tokens_in":0,"tokens_out":0,"present":false,"absent_reason":"budget","error":"cap","landed":null}],"synthesis":{"verdict":"proceed","blocking_objections":null,"absent_reviewers":[{"model":"B","reason":"budget"}],"total_cost_usd":0.01,"total_tokens_in":10,"total_tokens_out":2}}`
		var q QuorumResult
		if err := json.Unmarshal([]byte(old), &q); err != nil {
			t.Fatal(err)
		}
		if q.Doc != "old.md" || len(q.Reviewers) != 2 || !q.Reviewers[0].Present || q.Reviewers[0].Result.Verdict != VerdictPass || q.Reviewers[1].Salvaged != nil || q.Synthesis.Verdict != SynthProceed || q.Synthesis.AbsentReviewers[0].Reason != ReasonBudget {
			t.Fatalf("old artifact changed: %+v", q)
		}
		marshaled, err := json.Marshal(q)
		if err != nil {
			t.Fatal(err)
		}
		var before, after any
		if err := json.Unmarshal([]byte(old), &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(marshaled, &after); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("old artifact roundtrip changed: %s", marshaled)
		}
	})
	t.Run("markdown", func(t *testing.T) {
		for _, c := range []struct {
			v      Verdict
			suffix string
		}{
			{VerdictReject, "; salvaged REJECT — blocks"},
			{VerdictPass, "; salvaged PASS — audit only (not counted)"},
			{"", ""},
		} {
			o := &ReviewerOutcome{Model: "B", AbsentReason: ReasonInvalid}
			if c.v != "" {
				o.Salvaged = &SalvagedVerdict{Verdict: c.v}
			}
			got := MarkdownBlock(&QuorumResult{Doc: "doc.md", Reviewers: []*ReviewerOutcome{o}})
			want := "- `B` → **ABSENT** (invalid) — degraded to N-1, not a silent pass" + c.suffix + "\n"
			if !strings.Contains(got, want) {
				t.Fatalf("missing exact absent label %q: %s", want, got)
			}
		}
	})
}
