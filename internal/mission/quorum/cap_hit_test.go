package quorum

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func cappedModel() *eval_harness.ModelConfig {
	mc := cheapModel()
	mc.MaxOutputTokens = 65536
	return mc
}

func TestRunReviewerWith_CapHit(t *testing.T) {
	const cutOffReject = `{"verdict":"reject","strongest_objection":"the design omits`
	const malformed = `{"verdict":`
	const validPass = `{"verdict":"pass","strongest_objection":"closest concern","catch":"verify the limit"}`
	for _, tc := range []struct {
		name        string
		raw         string
		finish      string
		tokens      int
		zeroBudget  bool
		valid       bool
		salvaged    bool
		want        []string
		unwanted    []string
		parserError bool
	}{
		{
			name: "a", tokens: 65536,
			want: []string{
				"truncated", "inferred", "65536", `"oc-glm-5-3"`, "max_output_tokens",
				"finish_reason not reported; inferred from tokens_out 65536 >= max_output_tokens 65536",
				`the response consumed the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3"`,
			},
		},
		{
			name: "a2", tokens: 65537,
			want: []string{"truncated at 65537"},
		},
		{
			name: "b", raw: cutOffReject, finish: "length", tokens: 65536, salvaged: true,
			want: []string{
				"finish_reason=length", "reported by provider",
				`consumed the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3"`,
			},
			unwanted: []string{"inferred"},
		},
		{
			name: "b2", raw: cutOffReject, finish: "length", tokens: 40000, salvaged: true,
			want: []string{
				"finish_reason=length", "reported by provider",
				`provider reported finish_reason=length at 40000 tokens, below the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3" — a provider-side or context-window limit`,
			},
			unwanted: []string{"consumed", "inferred"},
		},
		{
			name: "b3", raw: cutOffReject, finish: "length", tokens: 16384, zeroBudget: true, salvaged: true,
			want: []string{
				"finish_reason=length", "reported by provider",
				`provider reported finish_reason=length at 16384 tokens; models.yml entry "oc-glm-5-3" declares no max_output_tokens — a provider-side or context-window limit`,
			},
			unwanted: []string{"consumed", "below the declared", "inferred"},
		},
		{
			name: "c", raw: malformed, finish: "stop", tokens: 65536,
			parserError: true, unwanted: []string{"truncated"},
		},
		{
			name: "d", raw: malformed, tokens: 65535,
			parserError: true, unwanted: []string{"truncated"},
		},
		{
			name: "f", raw: malformed, tokens: 20, zeroBudget: true,
			parserError: true, unwanted: []string{"truncated"},
		},
		{
			name: "e", raw: validPass, tokens: 65536, valid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := cappedModel()
			if tc.zeroBudget {
				mc = cheapModel()
			}
			stub := &stubCaller{
				raw: tc.raw,
				resp: &ai.Response{
					Model: "glm-5.3:cloud", InputTokens: 100,
					OutputTokens: tc.tokens, FinishReason: tc.finish,
				},
			}
			got := runReviewerWith(stub, mc, &ReviewerOutcome{Model: "oc-glm-5-3"}, "doc.md", "body", DefaultMaxCostUSD)
			if stub.gotSystem == "" {
				t.Fatal("stub was not called; preflight prevented the test")
			}
			if tc.valid {
				if !got.Present || got.AbsentReason != "" || got.Err != "" || got.Result == nil || got.Result.Verdict != VerdictPass {
					t.Fatalf("valid pass at cap must remain present with no absence/error and pass result; got %+v", got)
				}
				return
			}
			if got.Present || got.AbsentReason != ReasonInvalid || got.Result != nil {
				t.Fatalf("expected absent invalid with no result; got %+v", got)
			}
			if tc.salvaged {
				if got.Salvaged == nil || got.Salvaged.Verdict != VerdictReject {
					t.Fatalf("cut-off reject must be salvaged; got %+v", got.Salvaged)
				}
			} else if got.Salvaged != nil {
				t.Fatalf("expected nil salvage; got %+v", got.Salvaged)
			}
			if tc.parserError {
				_, err := ParseReviewResult(tc.raw)
				if err == nil {
					t.Fatal("fixture must fail parsing")
				}
				if got.Err != err.Error() {
					t.Errorf("ordinary parse error changed: got %q, want %q", got.Err, err.Error())
				}
			} else if !strings.HasPrefix(got.Err, "reviewer output truncated at ") || !strings.Contains(got.Err, ". Parse error: ") {
				t.Errorf("expected truncation diagnostic with parse error; got %q", got.Err)
			}
			for _, want := range tc.want {
				if !strings.Contains(got.Err, want) {
					t.Errorf("error missing %q; got %q", want, got.Err)
				}
			}
			for _, unwanted := range tc.unwanted {
				if strings.Contains(got.Err, unwanted) {
					t.Errorf("error must exclude %q; got %q", unwanted, got.Err)
				}
			}
		})
	}
}
