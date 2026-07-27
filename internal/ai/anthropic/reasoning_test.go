package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"errors"

	"github.com/sunholo-data/ailang/internal/ai"
)

func captureBody(t *testing.T, captured *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*captured = string(b)
		resp := messagesResponse{
			ID: "m", Type: "message", Role: "assistant", Model: "claude",
			Content:    []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 1, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// TestAnthropic_Golden_NoReasoning_ByteIdentical (AC14): unset request body
// carries NO thinking key (byte-identical to pre-v0.31.0; omitempty pointer),
// and no output_config either. Covers both generations: on an adaptive-style
// model an omitted block means the model's own default (thinking ON for Opus 5
// / Sonnet 5), which is intended — the harness must not start injecting a
// control just because the default changed.
func TestAnthropic_Golden_NoReasoning_ByteIdentical(t *testing.T) {
	for _, model := range []string{"claude-sonnet-4-5", "claude-opus-4-8", "claude-opus-5"} {
		t.Run(model, func(t *testing.T) {
			var body string
			srv := captureBody(t, &body)
			defer srv.Close()

			c := NewClient("k", WithBaseURL(srv.URL))
			_, err := c.Generate(context.Background(), &ai.Request{
				Model: model, SystemPrompt: "sys", UserPrompt: "hi", MaxTokens: 2048,
			})
			if err != nil {
				t.Fatalf("Generate error = %v", err)
			}
			if strings.Contains(body, "thinking") {
				t.Fatalf("unset request body leaked thinking block: %s", body)
			}
			if strings.Contains(body, "output_config") {
				t.Fatalf("unset request body leaked output_config: %s", body)
			}
		})
	}
}

// TestAnthropic_ThinkingWireShape_PerGeneration pins the EXACT request-body
// controls per model generation. This is the regression guard for the hard 400
// Anthropic returns when thinking.budget_tokens reaches an Opus 4.7-or-later
// model: budget_tokens must never appear for an adaptive-generation model, and
// the adaptive shape must never appear for a budget-generation model.
//
// It marshals the real messagesRequest rather than inspecting the structs, so a
// JSON tag or omitempty regression fails here too.
func TestAnthropic_ThinkingWireShape_PerGeneration(t *testing.T) {
	effort := func(e string, budget int) ai.ReasoningDecision {
		return ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: e, Budget: budget, BudgetSet: true}
	}

	tests := []struct {
		name string
		// model is the model id the decision is being applied to.
		model string
		dec   ai.ReasoningDecision
		// want is the marshaled subset of the request body carrying the
		// reasoning controls ("" = neither key present).
		want string
		// wantErrSubstr, when set, requires a rejection containing it.
		wantErrSubstr string
	}{
		// --- no control: body unchanged on every generation ---------------
		{name: "none/budget-gen", model: "claude-sonnet-4-6", dec: ai.ReasoningDecision{Kind: ai.ReasoningNone}},
		{name: "none/adaptive-gen", model: "claude-opus-5", dec: ai.ReasoningDecision{Kind: ai.ReasoningNone}},

		// --- budget generation: unchanged pre-v0.31.0 shape ---------------
		{
			name: "budget-gen/high", model: "claude-sonnet-4-6",
			dec:  effort(ai.ReasoningEffortHigh, 16384),
			want: `"thinking":{"type":"enabled","budget_tokens":16384}`,
		},
		{
			name: "budget-gen/explicit-budget", model: "claude-opus-4-6",
			dec:  ai.ReasoningDecision{Kind: ai.ReasoningBudgetKind, Budget: 1024, BudgetSet: true},
			want: `"thinking":{"type":"enabled","budget_tokens":1024}`,
		},
		{
			// Disablement by omission — the only form this generation has.
			name: "budget-gen/off", model: "claude-haiku-4-5",
			dec: effort(ai.ReasoningEffortOff, 0),
		},
		{
			// Dated snapshot ids resolve to the same generation as the alias.
			name: "budget-gen/dated-snapshot", model: "claude-sonnet-4-5-20250929",
			dec:  effort(ai.ReasoningEffortMedium, 4096),
			want: `"thinking":{"type":"enabled","budget_tokens":4096}`,
		},

		// --- adaptive generation: budget_tokens must NEVER appear ---------
		{
			name: "adaptive-gen/high/opus-5", model: "claude-opus-5",
			dec:  effort(ai.ReasoningEffortHigh, 16384),
			want: `"thinking":{"type":"adaptive"},"output_config":{"effort":"high"}`,
		},
		{
			name: "adaptive-gen/low/opus-4-8", model: "claude-opus-4-8",
			dec:  effort(ai.ReasoningEffortLow, 1024),
			want: `"thinking":{"type":"adaptive"},"output_config":{"effort":"low"}`,
		},
		{
			name: "adaptive-gen/medium/sonnet-5", model: "claude-sonnet-5",
			dec:  effort(ai.ReasoningEffortMedium, 4096),
			want: `"thinking":{"type":"adaptive"},"output_config":{"effort":"medium"}`,
		},
		{
			name: "adaptive-gen/high/opus-4-7", model: "claude-opus-4-7",
			dec:  effort(ai.ReasoningEffortHigh, 16384),
			want: `"thinking":{"type":"adaptive"},"output_config":{"effort":"high"}`,
		},
		{
			// EXPLICIT disabled: omitting the block would leave thinking ON by
			// default on this generation, i.e. the opposite of what was asked.
			name: "adaptive-gen/off/opus-5", model: "claude-opus-5",
			dec:  effort(ai.ReasoningEffortOff, 0),
			want: `"thinking":{"type":"disabled"}`,
		},
		{
			name: "adaptive-gen/high/fable-5", model: "claude-fable-5",
			dec:  effort(ai.ReasoningEffortHigh, 16384),
			want: `"thinking":{"type":"adaptive"},"output_config":{"effort":"high"}`,
		},

		// --- rejections instead of provider 400s --------------------------
		{
			// Fable/Mythos think unconditionally; explicit disabled is a 400.
			name: "adaptive-gen/off/fable-5-rejected", model: "claude-fable-5",
			dec:           effort(ai.ReasoningEffortOff, 0),
			wantErrSubstr: "runs thinking unconditionally",
		},
		{
			name: "unregistered-generation-rejected", model: "claude-9-not-real",
			dec:           effort(ai.ReasoningEffortHigh, 16384),
			wantErrSubstr: "no registered thinking-control generation",
		},
		{
			// Belt and braces: ResolveReasoning rejects a positive explicit
			// budget on an adaptive model first, but if one ever reaches the
			// builder it must not be silently downgraded to a wire shape.
			name: "adaptive-gen/explicit-budget-rejected", model: "claude-opus-5",
			dec:           ai.ReasoningDecision{Kind: ai.ReasoningBudgetKind, Budget: 8192, BudgetSet: true},
			wantErrSubstr: "needs a qualitative effort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, oc, err := thinkingConfigFor(tt.dec, tt.model)

			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("thinkingConfigFor(%s) = (%+v, %+v, nil), want error containing %q",
						tt.model, tb, oc, tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("thinkingConfigFor(%s) error = %v", tt.model, err)
			}

			body, mErr := json.Marshal(messagesRequest{
				Model:        tt.model,
				MaxTokens:    64000,
				Messages:     []messageContent{{Role: "user", Content: "hi"}},
				Thinking:     tb,
				OutputConfig: oc,
			})
			if mErr != nil {
				t.Fatalf("marshal: %v", mErr)
			}
			got := string(body)

			if tt.want == "" {
				if strings.Contains(got, "thinking") || strings.Contains(got, "output_config") {
					t.Fatalf("body carries reasoning controls, want none: %s", got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("body = %s\nwant substring %s", got, tt.want)
			}
			// The core invariant: the removed knob must not reach a 4.7+ model.
			if style, ok := ai.AnthropicThinkingStyleFor(tt.model); ok && style.Adaptive {
				if strings.Contains(got, "budget_tokens") {
					t.Fatalf("adaptive-generation model %s got budget_tokens (hard 400 at Anthropic): %s",
						tt.model, got)
				}
			}
		})
	}
}

// TestAnthropic_StepRequest_SharesThinkingShape pins that the Step/StreamStep
// path emits the same per-generation controls as Generate — the two request
// builders are separate structs and drifted independently before.
func TestAnthropic_StepRequest_SharesThinkingShape(t *testing.T) {
	req := &ai.Request{Model: "claude-opus-5", UserPrompt: "hi", MaxTokens: 64000}
	dec := ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: ai.ReasoningEffortHigh, Budget: 16384, BudgetSet: true}

	apiReq, err := buildStepRequest(req, dec)
	if err != nil {
		t.Fatalf("buildStepRequest error = %v", err)
	}
	body, mErr := json.Marshal(apiReq)
	if mErr != nil {
		t.Fatalf("marshal: %v", mErr)
	}
	got := string(body)
	if !strings.Contains(got, `"thinking":{"type":"adaptive"}`) {
		t.Errorf("step body missing adaptive thinking: %s", got)
	}
	if !strings.Contains(got, `"output_config":{"effort":"high"}`) {
		t.Errorf("step body missing output_config effort: %s", got)
	}
	if strings.Contains(got, "budget_tokens") {
		t.Errorf("step body sent budget_tokens to an adaptive-generation model: %s", got)
	}
}

// TestAnthropic_StrictBudget_BeforeDefaulting (AC11): an enabled thinking budget
// with MaxTokens unset (the client would otherwise silently substitute 4096) is
// rejected BEFORE that defaulting, and no request is dispatched.
func TestAnthropic_StrictBudget_BeforeDefaulting(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	// budget 4096 with MaxTokens unset: the 4096 default must NOT rescue it.
	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "claude", UserPrompt: "hi",
		Options: map[string]any{"thinking_budget_tokens": 4096},
	})
	if !errors.Is(err, ai.ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("error = %v, want ErrReasoningBudgetExceedsMaxTokens", err)
	}
	if hit {
		t.Fatalf("request dispatched; validation must precede the MaxTokens=4096 defaulting")
	}

	// budget == MaxTokens is also rejected (strict >).
	_, err = c.Generate(context.Background(), &ai.Request{
		Model: "claude", UserPrompt: "hi", MaxTokens: 4096,
		Options: map[string]any{"thinking_budget_tokens": 4096},
	})
	if !errors.Is(err, ai.ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("budget==maxtokens: error = %v, want ErrReasoningBudgetExceedsMaxTokens", err)
	}
}
