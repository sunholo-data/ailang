package feedbackgate

import (
	"context"
	"errors"
	"testing"
)

// fakeShadow returns a canned ShadowResult (or error) and counts calls.
type fakeShadow struct {
	res   ShadowResult
	err   error
	calls int
}

func (f *fakeShadow) Run(_ context.Context, _ Input) (ShadowResult, error) {
	f.calls++
	if f.err != nil {
		return ShadowResult{}, f.err
	}
	return f.res, nil
}

func shadowRes(genuine, injection float64, category, value string) ShadowResult {
	var r ShadowResult
	r.Model = "typesafe/jev-1.13-20260917"
	r.Genuine.P = genuine
	r.Injection.P = injection
	r.Category.Choice = category
	r.Category.Confidence = 0.9
	r.Value.Label = value
	return r
}

// The shadow verdict must mirror classifierVerdict branch for branch, with
// p >= 0.5 as "true". Same order: injection, value none, not genuine,
// category mismatch, dispatch.
func TestShadowVerdictMirrorsClassifierMatrix(t *testing.T) {
	in := flaggedInput() // category "bug"
	cases := []struct {
		name       string
		res        ShadowResult
		wantAction string
		wantReason string
	}{
		{"genuine matching category dispatches", shadowRes(0.92, 0.02, "bug", "high"), ActionDispatch, ReasonPassed},
		{"injection at threshold rejects", shadowRes(0.02, 0.5, "spam", "low"), ActionReject, ReasonClassifierInjection},
		{"injection wins over everything", shadowRes(0.99, 0.99, "bug", "high"), ActionReject, ReasonClassifierInjection},
		{"value none files before genuine (order matters: both would file, the REASON must be no_value)", shadowRes(0.1, 0.0, "bug", "none"), ActionFile, ReasonClassifierNoValue},
		{"not genuine files", shadowRes(0.49, 0.0, "bug", "low"), ActionFile, ReasonClassifierNotGenuine},
		{"category mismatch files", shadowRes(0.9, 0.0, "feature", "high"), ActionFile, ReasonClassifierMismatch},
		{"model error files as classifier_error", func() ShadowResult { r := shadowRes(0.9, 0, "bug", "high"); r.Error = "Transport(timeout)"; return r }(), ActionFile, ReasonClassifierError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, r := shadowVerdict(in, tc.res)
			if a != tc.wantAction || r != tc.wantReason {
				t.Fatalf("got %q/%q, want %q/%q", a, r, tc.wantAction, tc.wantReason)
			}
		})
	}
}

// The shadow NEVER changes the applied verdict, agrees/disagrees is recorded,
// and it runs on exactly the messages the classifier saw.
func TestShadowRidesBesideHaikuWithoutChangingTheVerdict(t *testing.T) {
	haikuDispatch := `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"high","reasoning":"real"}`

	t.Run("agreeing shadow", func(t *testing.T) {
		prov := &countingProvider{text: haikuDispatch}
		sh := &fakeShadow{res: shadowRes(0.9, 0.01, "bug", "high")}
		cfg := FeedbackGateConfig{ShadowMode: ShadowOpenRouter}.normalized()
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
		if err != nil {
			t.Fatal(err)
		}
		if v.Action != ActionDispatch || v.Shadow == nil || !v.Shadow.Agrees || v.Shadow.WouldAction != ActionDispatch {
			t.Fatalf("verdict=%+v shadow=%+v", v, v.Shadow)
		}
		if sh.calls != 1 {
			t.Fatalf("shadow called %d times, want 1", sh.calls)
		}
	})

	t.Run("disagreeing shadow does not flip the action", func(t *testing.T) {
		prov := &countingProvider{text: haikuDispatch}
		sh := &fakeShadow{res: shadowRes(0.1, 0.95, "spam", "none")} // Jev says: injection → reject
		cfg := FeedbackGateConfig{ShadowMode: ShadowDirect}.normalized()
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, _ := applyClassifier(context.Background(), flaggedInput(), cfg)
		if v.Action != ActionDispatch {
			t.Fatalf("shadow changed the action to %q", v.Action)
		}
		if v.Shadow == nil || v.Shadow.Agrees || v.Shadow.WouldAction != ActionReject {
			t.Fatalf("shadow=%+v", v.Shadow)
		}
	})

	t.Run("shadow runner failure is recorded, never propagated", func(t *testing.T) {
		prov := &countingProvider{text: haikuDispatch}
		sh := &fakeShadow{err: errors.New("subprocess exploded")}
		cfg := FeedbackGateConfig{ShadowMode: ShadowOpenRouter}.normalized()
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
		if err != nil || v.Action != ActionDispatch {
			t.Fatalf("err=%v action=%q", err, v.Action)
		}
		if v.Shadow == nil || v.Shadow.Err == "" || v.Shadow.WouldAction != ActionFile {
			t.Fatalf("shadow=%+v", v.Shadow)
		}
	})

	t.Run("shadow also runs when Haiku itself fails, on the same message", func(t *testing.T) {
		prov := &countingProvider{err: errors.New("provider down")}
		sh := &fakeShadow{res: shadowRes(0.9, 0.01, "bug", "high")}
		cfg := FeedbackGateConfig{ShadowMode: ShadowOpenRouter}.normalized()
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, _ := applyClassifier(context.Background(), flaggedInput(), cfg)
		if v.Action != ActionFile || v.Reason != ReasonClassifierError {
			t.Fatalf("haiku failure must still fail closed: %+v", v)
		}
		if v.Shadow == nil || v.Shadow.WouldAction != ActionDispatch || v.Shadow.Agrees {
			t.Fatalf("shadow=%+v", v.Shadow)
		}
	})

	t.Run("nil provider (prod coordinator: no Anthropic key) still runs the shadow and fails closed", func(t *testing.T) {
		sh := &fakeShadow{res: shadowRes(0.9, 0.01, "bug", "high")}
		cfg := FeedbackGateConfig{ShadowMode: ShadowOpenRouter}.normalized()
		cfg.Classifier = NewClassifier(nil, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
		if err != nil || v.Action != ActionFile || v.Reason != ReasonClassifierError {
			t.Fatalf("nil provider must file/classifier_error: err=%v v=%+v", err, v)
		}
		if v.Shadow == nil || sh.calls != 1 || v.Shadow.WouldAction != ActionDispatch || v.Shadow.Agrees {
			t.Fatalf("shadow did not run beside the nil-provider path: calls=%d shadow=%+v", sh.calls, v.Shadow)
		}
	})

	t.Run("off by default: no runner call, no Shadow on the verdict", func(t *testing.T) {
		prov := &countingProvider{text: haikuDispatch}
		sh := &fakeShadow{res: shadowRes(0.9, 0.01, "bug", "high")}
		cfg := FeedbackGateConfig{}.normalized() // ShadowMode empty
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		v, _ := applyClassifier(context.Background(), flaggedInput(), cfg)
		if v.Shadow != nil || sh.calls != 0 {
			t.Fatalf("shadow ran while off: calls=%d shadow=%+v", sh.calls, v.Shadow)
		}
	})

	t.Run("agent senders and unflagged messages never reach the shadow either", func(t *testing.T) {
		prov := &countingProvider{text: haikuDispatch}
		sh := &fakeShadow{res: shadowRes(0.9, 0.01, "bug", "high")}
		cfg := FeedbackGateConfig{ShadowMode: ShadowOpenRouter}.normalized()
		cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)
		cfg.Shadow = sh
		in := flaggedInput()
		in.From = agentSenderPrefix + "sprint-executor"
		v, _ := applyClassifier(context.Background(), in, cfg)
		if v.Shadow != nil || sh.calls != 0 || prov.calls != 0 {
			t.Fatalf("agent bypass leaked to a model: shadow calls=%d provider calls=%d", sh.calls, prov.calls)
		}
	})
}

// ParseShadowOutput takes the LAST JSON line: the pipeline prints chatter
// (type-check / effect-check lines, MOD010 warnings) before the row.
func TestParseShadowOutputTakesLastJSONLine(t *testing.T) {
	out := "→ Type checking...\n→ Effect checking...\nWARNING MOD010 (relaxed): ...\n✓ Running feedback_shadow.ail\n" +
		`{"transport":"direct","model":"jev-1.13.0","id":"","degraded":false,"degraded_why":"","latency_ms":694,"input_tokens":611,"output_tokens":90,"cost_usd":0,"list_price_usd":0.000025662,"genuine":{"p":0.92},"injection":{"p":0.02},"category":{"choice":"bug","confidence":1,"probabilities":{"bug":1,"docs":0,"feature":0,"limitation":0,"spam":0}},"value":{"score":2.81,"label":"high","confidence":0.83,"probabilities":{"0":0,"1":0.02,"2":0.15,"3":0.83}},"error":""}` + "\n"
	r, err := ParseShadowOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	if r.Model != "jev-1.13.0" || r.Genuine.P != 0.92 || r.Category.Choice != "bug" || r.Value.Label != "high" || r.LatencyMs != 694 {
		t.Fatalf("parsed %+v", r)
	}
	if _, err := ParseShadowOutput("→ Type checking...\nError: something\n"); err == nil {
		t.Fatal("no JSON line must be an error, not an empty result")
	}
}
