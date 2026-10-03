package effects

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func writeFixtures(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixtures.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// #1497: text, step and image requests replay their fixtures; a miss is a
// loud typed error naming the sha256, never the default {"kind":"Wait"}.
func TestFixtureAIHandler_ReplayAndMiss(t *testing.T) {
	png := base64.StdEncoding.EncodeToString(stubPNG)
	p := writeFixtures(t, `{
	  "version": 1,
	  "fixtures": [
	    {"kind": "text", "prompt": "hello", "response": "GRIMNAR: Well met."},
	    {"kind": "text", "sha256": "`+FixtureKey("by-hash")+`", "response": "{\"kind\":\"Move\"}"},
	    {"kind": "text", "prompt": "use a tool", "response": "", "tool_calls": [{"id": "c1", "name": "lookup", "arguments": "{\"q\":1}"}]},
	    {"kind": "text", "prompt": "flaky", "error": {"code": "RateLimit", "message": "slow down", "retryable": true}},
	    {"kind": "image", "prompt": "a red cube", "base64": "`+png+`", "mime_type": "image/png"}
	  ]
	}`)
	f, err := LoadStubFixtures(p)
	if err != nil {
		t.Fatal(err)
	}
	h := NewFixtureAIHandler(f)

	if got, err := h.Call("hello"); err != nil || got != "GRIMNAR: Well met." {
		t.Fatalf("Call: %q %v", got, err)
	}
	if got, err := h.CallJson("by-hash", `{"type":"object"}`); err != nil || got != `{"kind":"Move"}` {
		t.Fatalf("CallJson by sha256: %q %v", got, err)
	}
	resp, err := h.Step("m", []ai.Message{{Role: "user", Content: "first"}, {Role: "user", Content: "use a tool"}}, nil)
	if err != nil || resp.FinishReason != "tool_calls" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "lookup" {
		t.Fatalf("Step: %+v %v", resp, err)
	}
	_, err = h.Call("flaky")
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) || aiErr.Code != "RateLimit" || !aiErr.Retryable {
		t.Fatalf("error fixture: %v", err)
	}
	if got, err := h.CallImageBase64("a red cube", ""); err != nil || !strings.Contains(got, png) || !strings.Contains(got, `"mime_type":"image/png"`) {
		t.Fatalf("CallImageBase64: %q %v", got, err)
	}
	out := filepath.Join(t.TempDir(), "sub", "cube.png")
	if _, err := h.CallImage("a red cube", out, ""); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(out); string(b) != string(stubPNG) {
		t.Fatal("CallImage wrote the wrong bytes")
	}

	// Misses: every op, never the default stub response.
	_, err = h.Call("unknown prompt")
	if !errors.As(err, &aiErr) || !strings.Contains(err.Error(), "E_AI_STUB_FIXTURE_MISS") ||
		!strings.Contains(err.Error(), FixtureKey("unknown prompt")) {
		t.Fatalf("text miss: %v", err)
	}
	if _, err := h.CallImageBase64("hello", ""); err == nil || !strings.Contains(err.Error(), "no image fixture") {
		t.Fatalf("kinds are separate key spaces: %v", err)
	}
	if _, err := h.Step("m", []ai.Message{{Role: "user", Content: "nope"}}, nil); err == nil {
		t.Fatal("step miss must fail")
	}
	if _, err := h.StepWithStream("m", nil, nil, nil, nil); err == nil {
		t.Fatal("stream miss must fail")
	}
}

// The miss reaches AILANG as Err(AIError) through the Result ops.
func TestFixtureAIHandler_MissThroughCallResult(t *testing.T) {
	f, err := LoadStubFixtures(writeFixtures(t, `{"version":1,"fixtures":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.AI = NewAIContext(NewFixtureAIHandler(f))
	v, err := aiCallResult(ctx, []eval.Value{&eval.StringValue{Value: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Err" {
		t.Fatalf("want Err, got %v", v)
	}
	rec := tv.Fields[0].(*eval.RecordValue)
	if msg := rec.Fields["message"].(*eval.StringValue).Value; !strings.Contains(msg, "E_AI_STUB_FIXTURE_MISS") {
		t.Fatalf("message: %s", msg)
	}
}

func TestLoadStubFixtures_Rejects(t *testing.T) {
	cases := map[string]string{
		"bad version":    `{"version":2,"fixtures":[]}`,
		"unknown field":  `{"version":1,"fixtures":[{"kind":"text","prompt":"a","response":"b","extra":1}]}`,
		"unknown kind":   `{"version":1,"fixtures":[{"kind":"video","prompt":"a"}]}`,
		"no key":         `{"version":1,"fixtures":[{"kind":"text","response":"b"}]}`,
		"both keys":      `{"version":1,"fixtures":[{"kind":"text","prompt":"a","sha256":"` + FixtureKey("a") + `","response":"b"}]}`,
		"bad sha":        `{"version":1,"fixtures":[{"kind":"text","sha256":"ABC","response":"b"}]}`,
		"no response":    `{"version":1,"fixtures":[{"kind":"text","prompt":"a"}]}`,
		"bad base64":     `{"version":1,"fixtures":[{"kind":"image","prompt":"a","base64":"!!","mime_type":"image/png"}]}`,
		"image no mime":  `{"version":1,"fixtures":[{"kind":"image","prompt":"a","base64":"AAAA"}]}`,
		"duplicate":      `{"version":1,"fixtures":[{"kind":"text","prompt":"a","response":"1"},{"kind":"text","sha256":"` + FixtureKey("a") + `","response":"2"}]}`,
		"error no code":  `{"version":1,"fixtures":[{"kind":"text","prompt":"a","error":{"message":"m"}}]}`,
		"error+response": `{"version":1,"fixtures":[{"kind":"text","prompt":"a","response":"r","error":{"code":"Timeout"}}]}`,
		"not json":       `{`,
	}
	for name, body := range cases {
		if _, err := LoadStubFixtures(writeFixtures(t, body)); err == nil {
			t.Errorf("%s: want a load error", name)
		}
	}
	if _, err := LoadStubFixtures(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing file: want an error")
	}
}
