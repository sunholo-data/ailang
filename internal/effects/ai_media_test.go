package effects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func makeFileMediaPartVal(uri, mime, resolution, options string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "File",
		Fields: []eval.Value{
			&eval.RecordValue{
				Fields: map[string]eval.Value{
					"uri":        &eval.StringValue{Value: uri},
					"mime":       &eval.StringValue{Value: mime},
					"resolution": &eval.StringValue{Value: resolution},
					"options":    &eval.StringValue{Value: options},
				},
			},
		},
	}
}

func makeImageMediaPartVal(source, mime, resolution string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Image",
		Fields: []eval.Value{
			&eval.RecordValue{
				Fields: map[string]eval.Value{
					"source":     &eval.StringValue{Value: source},
					"mime":       &eval.StringValue{Value: mime},
					"resolution": &eval.StringValue{Value: resolution},
				},
			},
		},
	}
}

func TestAICallMediaResult_StubDeterministic(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())

	args := []eval.Value{
		&eval.StringValue{Value: "Extract totals"},
		&eval.ListValue{Elements: []eval.Value{
			makeFileMediaPartVal("gs://eu-bucket/inv.pdf", "application/pdf", "low", `{"page_options":{"start_page":1,"end_page":1},"content_layers":["page_text"]}`),
			makeImageMediaPartVal("iVBORw0KGgo=", "image/png", "medium"),
		}},
		&eval.StringValue{Value: ""},
	}
	res, err := aiCallMediaResult(ctx, args)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tv, ok := res.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %#v", res)
	}
	rec := tv.Fields[0].(*eval.RecordValue)
	if got := rec.Fields["input_tokens"].(*eval.IntValue).Value; got <= 0 {
		t.Errorf("input_tokens = %d, want > 0", got)
	}
	if got := rec.Fields["output_tokens"].(*eval.IntValue).Value; got <= 0 {
		t.Errorf("output_tokens = %d, want > 0", got)
	}
}

func TestAICallMediaResult_ValidationErrorSurfacesTypedAIError(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())

	args := []eval.Value{
		&eval.StringValue{Value: "Extract"},
		&eval.ListValue{Elements: []eval.Value{
			makeFileMediaPartVal("gs://eu-bucket/inv.pdf", "application/pdf", "ultra", ""),
		}},
		&eval.StringValue{Value: ""},
	}
	res, err := aiCallMediaResult(ctx, args)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tv := res.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tv.CtorName)
	}
	errRec := tv.Fields[0].(*eval.RecordValue)
	if code := errRec.Fields["code"].(*eval.StringValue).Value; code != ai.CodeSchemaValidation {
		t.Errorf("code = %q, want %q", code, ai.CodeSchemaValidation)
	}
	if retryable := errRec.Fields["retryable"].(*eval.BoolValue).Value; retryable {
		t.Errorf("retryable = true, want false")
	}
}

func TestFixtureAIHandler_MediaFixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixtures.json")
	content := `{
  "version": 1,
  "fixtures": [
    {
      "kind": "media",
      "prompt": "Summarize page 1",
      "response": "{\"total\": 123}",
      "input_tokens": 312,
      "output_tokens": 18
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	sf, err := LoadStubFixtures(path)
	if err != nil {
		t.Fatalf("LoadStubFixtures: %v", err)
	}
	h := NewFixtureAIHandler(sf)
	resp, err := h.CallMediaResult("Summarize page 1", []ai.MediaPart{{
		Kind:    ai.MediaKindFile,
		Source:  "gs://eu-bucket/doc.pdf",
		Mime:    "application/pdf",
		Options: `{"page_options":{"start_page":1,"end_page":1}}`,
	}}, "")
	if err != nil {
		t.Fatalf("CallMediaResult: %v", err)
	}
	if resp.Text != `{"total": 123}` || resp.InputTokens != 312 || resp.OutputTokens != 18 {
		t.Errorf("unexpected resp: %+v", resp)
	}
}
