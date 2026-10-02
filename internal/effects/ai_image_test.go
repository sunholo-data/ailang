package effects

// Per-call image model resolution and reference-conditioned image ops
// (#1496/#1500).

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The real handler and the --ai-stub handler both carry the refs capability.
var (
	_ AIHandlerWithImageRefs = (*ai.Handler)(nil)
	_ AIHandlerWithImageRefs = (*StubAIHandler)(nil)
)

// imageRecordingHandler captures the options/refs each image call receives.
// It embeds fakeStepHandler for the rest of AIHandler and implements the
// optional AIHandlerWithImageRefs interface.
type imageRecordingHandler struct {
	fakeStepHandler
	calls       int
	lastOptions string
	lastRefs    []ai.ImagePart
}

func (h *imageRecordingHandler) CallImage(_, outputPath, options string) (string, error) {
	h.calls++
	h.lastOptions = options
	return outputPath, nil
}

func (h *imageRecordingHandler) CallImageBase64(_, options string) (string, error) {
	h.calls++
	h.lastOptions = options
	return `{"base64":"","mime_type":"image/png"}`, nil
}

func (h *imageRecordingHandler) CallImageWithRefs(_ string, refs []ai.ImagePart, outputPath, options string) (string, error) {
	h.calls++
	h.lastOptions, h.lastRefs = options, refs
	return outputPath, nil
}

func (h *imageRecordingHandler) CallImageBase64WithRefs(_ string, refs []ai.ImagePart, options string) (string, error) {
	h.calls++
	h.lastOptions, h.lastRefs = options, refs
	return `{"base64":"","mime_type":"image/png"}`, nil
}

// friendlyResolver mimics cmd/ailang makeModelResolver: one known friendly
// name, one cross-provider name, everything else passes through.
func friendlyResolver(calls *int) ModelResolver {
	return func(model string) (string, error) {
		*calls++
		switch model {
		case "pro-image":
			return "google/gemini-3-pro-image", nil
		case "gemini-2-5-flash-image":
			return "", ai.NewAIError(ai.CodeModelNotAllowed, "per-call model resolves to provider google, bound is openrouter", false)
		}
		return model, nil
	}
}

func optionsModel(t *testing.T, options string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(options), &m); err != nil {
		t.Fatalf("options not JSON: %q", options)
	}
	s, _ := m["model"].(string)
	return s
}

func TestAIContext_CallImage_ResolvesFriendlyModel(t *testing.T) {
	var resolves int
	h := &imageRecordingHandler{}
	c := NewAIContext(h).WithModelResolver(friendlyResolver(&resolves))

	if _, err := c.CallImage("p", "out.png", `{"model":"pro-image","aspect_ratio":"1:1"}`); err != nil {
		t.Fatalf("CallImage: %v", err)
	}
	if got := optionsModel(t, h.lastOptions); got != "google/gemini-3-pro-image" {
		t.Errorf("handler saw model %q, want resolved api_name", got)
	}
	if !strings.Contains(h.lastOptions, `"aspect_ratio":"1:1"`) {
		t.Errorf("other keys lost: %s", h.lastOptions)
	}

	if _, err := c.CallImageBase64("p", `{"model":"pro-image"}`); err != nil {
		t.Fatalf("CallImageBase64: %v", err)
	}
	if got := optionsModel(t, h.lastOptions); got != "google/gemini-3-pro-image" {
		t.Errorf("base64 handler saw model %q", got)
	}
}

func TestAIContext_CallImage_CrossProviderModelRejected(t *testing.T) {
	var resolves int
	h := &imageRecordingHandler{}
	c := NewAIContext(h).WithModelResolver(friendlyResolver(&resolves))

	checks := []func() error{
		func() error { _, err := c.CallImage("p", "o.png", `{"model":"gemini-2-5-flash-image"}`); return err },
		func() error { _, err := c.CallImageBase64("p", `{"model":"gemini-2-5-flash-image"}`); return err },
		func() error {
			_, err := c.CallImageBase64WithRefs("p", []ai.ImagePart{{Source: "cmVm", Mime: "image/png"}}, `{"model":"gemini-2-5-flash-image"}`)
			return err
		},
	}
	for i, check := range checks {
		err := check()
		var aiErr *ai.AIError
		if !errors.As(err, &aiErr) || aiErr.Code != ai.CodeModelNotAllowed {
			t.Errorf("case %d: err = %v, want CodeModelNotAllowed", i, err)
		}
	}
	if h.calls != 0 {
		t.Errorf("handler called %d times despite rejected model", h.calls)
	}
}

func TestAIContext_CallImage_NoModelKeySkipsResolver(t *testing.T) {
	var resolves int
	h := &imageRecordingHandler{}
	c := NewAIContext(h).WithModelResolver(friendlyResolver(&resolves))
	if _, err := c.CallImage("p", "o.png", `{"aspect_ratio":"16:9"}`); err != nil {
		t.Fatalf("CallImage: %v", err)
	}
	if resolves != 0 {
		t.Errorf("resolver called %d times without a model key", resolves)
	}
	if h.lastOptions != `{"aspect_ratio":"16:9"}` {
		t.Errorf("options rewritten without a model key: %s", h.lastOptions)
	}
}

func TestAIContext_CallImage_NilResolverPassthrough(t *testing.T) {
	h := &imageRecordingHandler{}
	c := NewAIContext(h)
	opts := `{"model":"pro-image"}`
	if _, err := c.CallImage("p", "o.png", opts); err != nil {
		t.Fatalf("CallImage: %v", err)
	}
	if h.lastOptions != opts {
		t.Errorf("options = %s, want passthrough %s", h.lastOptions, opts)
	}
}

func TestAIContext_CallImageWithRefs_UnsupportedHandlerFailsLoudly(t *testing.T) {
	c := NewAIContext(&fakeStepHandler{}) // no AIHandlerWithImageRefs
	_, err := c.CallImageWithRefs("p", []ai.ImagePart{{Source: "cmVm", Mime: "image/png"}}, "o.png", "{}")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err = %v, want not-supported", err)
	}
	_, err = c.CallImageBase64WithRefs("p", nil, "{}")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("base64 err = %v, want not-supported", err)
	}
}

func refsList(parts ...ai.ImagePart) *eval.ListValue {
	elems := make([]eval.Value, 0, len(parts))
	for _, p := range parts {
		elems = append(elems, &eval.RecordValue{Fields: map[string]eval.Value{
			"source": &eval.StringValue{Value: p.Source},
			"mime":   &eval.StringValue{Value: p.Mime},
		}})
	}
	return &eval.ListValue{Elements: elems}
}

func TestAIOp_CallImageBase64WithRefs_DecodesRefs(t *testing.T) {
	h := &imageRecordingHandler{}
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(h)

	args := []eval.Value{
		&eval.StringValue{Value: "same person, smiling"},
		refsList(ai.ImagePart{Source: "cmVm", Mime: "image/png"}, ai.ImagePart{Source: "data:image/jpeg;base64,eA==", Mime: "image/jpeg"}),
		&eval.StringValue{Value: `{"model":"m"}`},
	}
	out, err := Call(ctx, "AI", "callImageBase64WithRefs", args)
	if err != nil {
		t.Fatalf("callImageBase64WithRefs: %v", err)
	}
	if s, ok := out.(*eval.StringValue); !ok || !strings.Contains(s.Value, "mime_type") {
		t.Errorf("result = %v", out)
	}
	if len(h.lastRefs) != 2 || h.lastRefs[0].Source != "cmVm" || h.lastRefs[1].Mime != "image/jpeg" {
		t.Errorf("refs = %+v", h.lastRefs)
	}
}

func TestAIOp_CallImageWithRefs_RejectsEmptySource(t *testing.T) {
	h := &imageRecordingHandler{}
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(h)
	args := []eval.Value{
		&eval.StringValue{Value: "p"},
		refsList(ai.ImagePart{Source: "", Mime: "image/png"}),
		&eval.StringValue{Value: "o.png"},
		&eval.StringValue{Value: "{}"},
	}
	if _, err := Call(ctx, "AI", "callImageWithRefs", args); err == nil || !strings.Contains(err.Error(), "empty source") {
		t.Fatalf("err = %v, want empty-source error", err)
	}
	if h.calls != 0 {
		t.Error("handler called despite invalid ref")
	}
}

// --ai-stub: deterministic, no network, for both refs variants.
func TestStubAIHandler_WithRefs_Deterministic(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())
	refs := refsList(ai.ImagePart{Source: "cmVm", Mime: "image/png"})

	out := filepath.Join(t.TempDir(), "nested", "stub.png")
	got, err := Call(ctx, "AI", "callImageWithRefs", []eval.Value{
		&eval.StringValue{Value: "p"}, refs, &eval.StringValue{Value: out}, &eval.StringValue{Value: "{}"},
	})
	if err != nil {
		t.Fatalf("stub callImageWithRefs: %v", err)
	}
	if got.(*eval.StringValue).Value != out {
		t.Errorf("returned %v, want %s", got, out)
	}
	if b, err := os.ReadFile(out); err != nil || len(b) == 0 {
		t.Errorf("stub did not write the PNG: %v", err)
	}

	a, err := Call(ctx, "AI", "callImageBase64WithRefs", []eval.Value{&eval.StringValue{Value: "p"}, refs, &eval.StringValue{Value: "{}"}})
	if err != nil {
		t.Fatalf("stub callImageBase64WithRefs: %v", err)
	}
	b, _ := Call(ctx, "AI", "callImageBase64", []eval.Value{&eval.StringValue{Value: "p"}, &eval.StringValue{Value: "{}"}})
	if a.(*eval.StringValue).Value != b.(*eval.StringValue).Value {
		t.Error("stub refs variant differs from plain callImageBase64")
	}
}
