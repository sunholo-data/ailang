package quorum

import (
	"encoding/json"
	"errors"
	"fmt"
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
