package quorum

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// stubCaller is a JSONCaller test double: it returns a canned raw JSON string
// and response, or a canned error, without touching any real provider.
type stubCaller struct {
	raw  string
	resp *ai.Response
	err  error
	// captured inputs for assertion
	gotSystem string
	gotSchema string
}

func (s *stubCaller) CallJSON(sysPrompt, userPrompt, schema string) (string, *ai.Response, error) {
	s.gotSystem = sysPrompt
	s.gotSchema = schema
	if s.err != nil {
		return "", nil, s.err
	}
	return s.raw, s.resp, nil
}

// cheapModel is a synthetic pricing config keeping estimateCost well under any
// realistic cap so budget tests are deterministic.
func cheapModel() *eval_harness.ModelConfig {
	return &eval_harness.ModelConfig{
		APIName:  "test-model",
		Provider: "openai",
		Pricing:  eval_harness.Pricing{InputPer1K: 0.001, OutputPer1K: 0.002},
	}
}

func TestParseReviewResult_SchemaConformance(t *testing.T) {
	raw := `{"verdict":"reject","strongest_objection":"premise unverified: claims file X exists","catch":"verify internal/foo.go before planning"}`
	r, err := ParseReviewResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Verdict != VerdictReject {
		t.Errorf("verdict = %q, want reject", r.Verdict)
	}
	if r.StrongestObjection == "" || r.Catch == "" {
		t.Errorf("required fields empty: %+v", r)
	}
}

// TestReviewSchema_OpenAIStrictInvariant guards the regression fixed in mission
// iteration 41: OpenAI's strict json_schema mode rejects any schema whose
// "properties" contains a key absent from "required" (400 "'required' ... must
// include every key in properties"). The original reviewSchema left
// proposed_fix out of "required" to make it optional, which silently knocked
// every OpenAI reviewer out of the quorum (degrading it to solo-gemini). The
// fix keeps proposed_fix optional via a nullable type that IS in "required".
// This test fails if anyone reintroduces a property that is not also required.
func TestReviewSchema_OpenAIStrictInvariant(t *testing.T) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal([]byte(reviewSchema), &schema); err != nil {
		t.Fatalf("reviewSchema is not valid JSON: %v", err)
	}
	reqSet := make(map[string]bool, len(schema.Required))
	for _, k := range schema.Required {
		reqSet[k] = true
	}
	for prop := range schema.Properties {
		if !reqSet[prop] {
			t.Errorf("property %q is not in \"required\" — OpenAI strict json_schema mode will 400 and drop the reviewer from the quorum", prop)
		}
	}
	// Cross-provider guard: every property type must be a single JSON string
	// (not a ["string","null"] union). Vertex/Gemini's response_schema rejects
	// union types ("Proto field is not repeating"), so a union that satisfies
	// OpenAI would knock Gemini out. proposed_fix stays optional by CONVENTION
	// ("" sentinel), not by a nullable type.
	for prop, rawProp := range schema.Properties {
		var p struct {
			Type json.RawMessage `json:"type"`
		}
		if err := json.Unmarshal(rawProp, &p); err != nil {
			t.Fatalf("property %q malformed: %v", prop, err)
		}
		if strings.HasPrefix(strings.TrimSpace(string(p.Type)), "[") {
			t.Errorf("property %q type = %s is a union; Vertex/Gemini rejects union types — use a single type", prop, p.Type)
		}
	}
}

// TestParseReviewResult_NullProposedFixPreservesContract confirms the contract
// invariant behind the strict-mode fix: a reviewer that emits proposed_fix:null
// (the "no fix" case) still parses and validates, mapping to the Go zero value.
func TestParseReviewResult_NullProposedFixPreservesContract(t *testing.T) {
	raw := `{"verdict":"reject","strongest_objection":"premise unverified","catch":"verify foo.go","proposed_fix":null}`
	r, err := ParseReviewResult(raw)
	if err != nil {
		t.Fatalf("null proposed_fix must validate (optional-via-null contract): %v", err)
	}
	if r.ProposedFix != "" {
		t.Errorf("proposed_fix = %q, want \"\" (JSON null → Go zero value)", r.ProposedFix)
	}
}

func TestParseReviewResult_StripsCodeFences(t *testing.T) {
	raw := "```json\n{\"verdict\":\"pass\",\"strongest_objection\":\"minor: naming\",\"catch\":\"consider renaming\"}\n```"
	r, err := ParseReviewResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Verdict != VerdictPass {
		t.Errorf("verdict = %q, want pass", r.Verdict)
	}
}

func TestValidateReviewResult_EmptyObjectionIsHardError(t *testing.T) {
	// A reject with an empty strongest_objection is the LGTM-bias failure we
	// guard against — MUST be an error, never a coerced pass.
	cases := []*ReviewResult{
		{Verdict: VerdictReject, StrongestObjection: "", Catch: "x"},
		{Verdict: VerdictPass, StrongestObjection: "  ", Catch: "x"},
		{Verdict: VerdictPass, StrongestObjection: "y", Catch: ""},
		{Verdict: Verdict("maybe"), StrongestObjection: "y", Catch: "z"},
	}
	for i, c := range cases {
		if err := ValidateReviewResult(c); err == nil {
			t.Errorf("case %d: expected hard error for %+v, got nil", i, c)
		}
	}
}

// TestRunReviewer_UnknownModelIsNamedAbsence proves an unknown model id is
// reported with the semantically correct reason "unknown-model" (not "auth"),
// while staying loud: Present=false with a named reason.
func TestRunReviewer_UnknownModelIsNamedAbsence(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Skipf("models.yml unavailable: %v", err)
	}
	got := RunReviewer("definitely-not-a-real-model-id", "doc.md", "body", DefaultMaxCostUSD)
	if got.Present {
		t.Fatalf("expected absent for unknown model id")
	}
	if got.AbsentReason != ReasonUnknownModel {
		t.Errorf("absent reason = %q, want %q", got.AbsentReason, ReasonUnknownModel)
	}
	if got.Err == "" {
		t.Errorf("unknown-model absence must still carry an error (named, not silent)")
	}
}

func TestRunReviewerWith_ValidVerdict(t *testing.T) {
	stub := &stubCaller{
		raw:  `{"verdict":"pass","strongest_objection":"none material","catch":"double-check the cost cap"}`,
		resp: &ai.Response{InputTokens: 1000, OutputTokens: 200},
	}
	out := &ReviewerOutcome{Model: "test-model"}
	got := runReviewerWith(stub, cheapModel(), out, "doc.md", "some body", DefaultMaxCostUSD)

	if !got.Present {
		t.Fatalf("expected Present, got absent: %s", got.Err)
	}
	if got.Result.Verdict != VerdictPass {
		t.Errorf("verdict = %q, want pass", got.Result.Verdict)
	}
	// cost = 1000/1000*0.001 + 200/1000*0.002 = 0.001 + 0.0004 = 0.0014
	if got.CostUSD < 0.0013 || got.CostUSD > 0.0015 {
		t.Errorf("cost = %f, want ~0.0014", got.CostUSD)
	}
	// system prompt + schema must have been passed through
	if !strings.Contains(stub.gotSystem, "REJECT by default") {
		t.Errorf("reviewer system prompt not passed to caller")
	}
	if !strings.Contains(stub.gotSchema, "strongest_objection") {
		t.Errorf("schema not passed to caller")
	}
}

func TestRunReviewerWith_MalformedResponseIsInvalidAbsence(t *testing.T) {
	stub := &stubCaller{
		raw:  `{"verdict":"reject","strongest_objection":"","catch":""}`, // gate violation
		resp: &ai.Response{InputTokens: 100, OutputTokens: 10},
	}
	out := &ReviewerOutcome{Model: "test-model"}
	got := runReviewerWith(stub, cheapModel(), out, "doc.md", "body", DefaultMaxCostUSD)

	if got.Present {
		t.Fatalf("expected absent for gate-violating response")
	}
	if got.AbsentReason != ReasonInvalid {
		t.Errorf("absent reason = %q, want %q", got.AbsentReason, ReasonInvalid)
	}
}

func TestRunReviewerWith_UnreachableProvider(t *testing.T) {
	stub := &stubCaller{err: errors.New("connection refused")}
	out := &ReviewerOutcome{Model: "test-model"}
	got := runReviewerWith(stub, cheapModel(), out, "doc.md", "body", DefaultMaxCostUSD)

	if got.Present {
		t.Fatalf("expected absent for unreachable provider")
	}
	if got.AbsentReason != ReasonUnreachable {
		t.Errorf("absent reason = %q, want %q", got.AbsentReason, ReasonUnreachable)
	}
}

func TestRunReviewerWith_BudgetCapRefusalZeroSpend(t *testing.T) {
	// A model priced so the pre-flight estimate blows the cap. The stub would
	// error if called — proving zero spend on refusal.
	pricey := &eval_harness.ModelConfig{
		APIName:  "pricey",
		Provider: "openai",
		Pricing:  eval_harness.Pricing{InputPer1K: 1.0, OutputPer1K: 1.0}, // est >> cap
	}
	stub := &stubCaller{err: errors.New("MUST NOT BE CALLED")}
	out := &ReviewerOutcome{Model: "pricey"}
	// A body large enough that even the floor-scaled estimate blows the cap at
	// $1/1K pricing.
	bigBody := strings.Repeat("x", 8000)
	got := runReviewerWith(stub, pricey, out, "doc.md", bigBody, DefaultMaxCostUSD)

	if got.Present {
		t.Fatalf("expected budget refusal")
	}
	if got.AbsentReason != ReasonBudget {
		t.Errorf("absent reason = %q, want %q", got.AbsentReason, ReasonBudget)
	}
	if got.CostUSD != 0 {
		t.Errorf("budget refusal must be zero spend, got $%f", got.CostUSD)
	}
	if stub.gotSchema != "" {
		t.Errorf("caller was invoked despite budget refusal (non-zero spend risk)")
	}
}
