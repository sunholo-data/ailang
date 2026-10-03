package effects

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func speechArgs(text, voice, style, options string) []eval.Value {
	return []eval.Value{
		&eval.StringValue{Value: text}, &eval.StringValue{Value: voice},
		&eval.StringValue{Value: style}, &eval.StringValue{Value: options},
	}
}

func okBytes(t *testing.T, v eval.Value) []byte {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Ok" {
		t.Fatalf("want Ok(bytes), got %v", v)
	}
	b, ok := tv.Fields[0].(*eval.BytesValue)
	if !ok {
		t.Fatalf("want BytesValue, got %T", tv.Fields[0])
	}
	return b.Value
}

func errCode(t *testing.T, v eval.Value) string {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Err" {
		t.Fatalf("want Err(AIError), got %v", v)
	}
	return tv.Fields[0].(*eval.RecordValue).Fields["code"].(*eval.StringValue).Value
}

// #1495: the stub tone is a fixed, platform-independent 250 ms of s16le
// mono 24 kHz PCM. The digest pins it so a change is a deliberate act.
func TestStubSpeechPCM_Deterministic(t *testing.T) {
	pcm := StubSpeechPCM()
	if len(pcm) != 2*ai.SpeechSampleRate/4 {
		t.Fatalf("len = %d, want 250 ms of 16-bit mono at 24 kHz", len(pcm))
	}
	var peak, nonzero int
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(pcm[i:])))
		if v != 0 {
			nonzero++
		}
		if v > peak {
			peak = v
		}
	}
	if nonzero < len(pcm)/4 || peak < 4096 {
		t.Fatalf("tone is not audible: peak %d, nonzero samples %d", peak, nonzero)
	}
	sum := sha256.Sum256(pcm)
	const want = "7951caf474022a571d92ab9c6ae754621b17cc9a6fd705a6219e2cb7783a3c27"
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("stub tone digest = %s, want %s", got, want)
	}
}

func TestAICallSpeech_Stub(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.AI = NewAIContext(NewStubAIHandler())
	v, err := aiCallSpeech(ctx, speechArgs("Well met.", "Kore", "Say gruffly", ""))
	if err != nil {
		t.Fatal(err)
	}
	if got := okBytes(t, v); string(got) != string(StubSpeechPCM()) {
		t.Fatal("stub speech is not the stub tone")
	}
	// Strict options even in the stub: a typo is not silently ignored.
	v, _ = aiCallSpeech(ctx, speechArgs("x", "Kore", "", `{"rate":16000}`))
	if code := errCode(t, v); code != ai.CodeSchemaValidation {
		t.Fatalf("bad options: %s", code)
	}
}

// No handler → Err(ProviderNotFound); a handler without speech →
// Err(CapabilityNotSupported). Neither crashes the host.
func TestAICallSpeech_NoHandlerAndUnsupported(t *testing.T) {
	ctx := NewEffContext(nil)
	v, err := aiCallSpeech(ctx, speechArgs("x", "", "", ""))
	if err != nil || errCode(t, v) != ai.CodeProviderNotFound {
		t.Fatalf("no handler: %v %v", v, err)
	}
	ctx.AI = NewAIContext(textOnlyHandler{NewStubAIHandler()})
	v, err = aiCallSpeech(ctx, speechArgs("x", "", "", ""))
	if err != nil || errCode(t, v) != ai.CodeCapabilityNotSupported {
		t.Fatalf("unsupported: %v %v", v, err)
	}
	if _, err := aiCallSpeech(ctx, speechArgs("x", "", "", "")[:3]); err == nil {
		t.Fatal("arity error expected")
	}
}

// textOnlyHandler hides the stub's CallSpeech, like a handler written before #1495.
type textOnlyHandler struct{ s *StubAIHandler }

func (h textOnlyHandler) Call(i string) (string, error)        { return h.s.Call(i) }
func (h textOnlyHandler) CallJson(i, s string) (string, error) { return h.s.CallJson(i, s) }
func (h textOnlyHandler) CallImageBase64(p, o string) (string, error) {
	return h.s.CallImageBase64(p, o)
}
func (h textOnlyHandler) CallImage(p, out, o string) (string, error) {
	return h.s.CallImage(p, out, o)
}
func (h textOnlyHandler) Step(m string, ms []ai.Message, ts []ai.ToolSchema) (*ai.Response, error) {
	return h.s.Step(m, ms, ts)
}
func (h textOnlyHandler) StepWithCache(m string, ms []ai.Message, ts []ai.ToolSchema, c []ai.CacheBreakpoint) (*ai.Response, error) {
	return h.s.StepWithCache(m, ms, ts, c)
}
func (h textOnlyHandler) StepWithStream(m string, ms []ai.Message, ts []ai.ToolSchema, c []ai.CacheBreakpoint, f func(ai.StreamChunk)) (*ai.Response, error) {
	return h.s.StepWithStream(m, ms, ts, c, f)
}

// #1495 + #1497: speech fixtures replay PCM keyed by {style,text,voice};
// a miss (here: a different voice) is an error, not the stub tone.
func TestFixtureAIHandler_Speech(t *testing.T) {
	pcm := []byte{1, 0, 2, 0, 3, 0}
	b64 := base64.StdEncoding.EncodeToString(pcm)
	canon := SpeechFixtureCanonical("Well met <friend>", "Kore", "")
	if canon != `{"style":"","text":"Well met <friend>","voice":"Kore"}` {
		t.Fatalf("canonical form drifted: %s", canon)
	}
	p := writeFixtures(t, `{"version":1,"fixtures":[
	  {"kind":"speech","request":{"text":"Well met <friend>","voice":"Kore","style":""},"pcm_base64":"`+b64+`"},
	  {"kind":"speech","sha256":"`+FixtureKey(SpeechFixtureCanonical("Bye", "Puck", "Say sadly"))+`","pcm_base64":"`+b64+`"}
	]}`)
	f, err := LoadStubFixtures(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.AI = NewAIContext(NewFixtureAIHandler(f))
	for _, args := range [][3]string{{"Well met <friend>", "Kore", ""}, {"Bye", "Puck", "Say sadly"}} {
		v, err := aiCallSpeech(ctx, speechArgs(args[0], args[1], args[2], ""))
		if err != nil {
			t.Fatal(err)
		}
		if got := okBytes(t, v); string(got) != string(pcm) {
			t.Fatalf("%v: pcm = %v", args, got)
		}
	}
	_, err = NewFixtureAIHandler(f).CallSpeech("Well met <friend>", "Puck", "", "")
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) || !strings.Contains(err.Error(), "no speech fixture") {
		t.Fatalf("miss: %v", err)
	}

	for name, body := range map[string]string{
		"odd pcm":        `{"version":1,"fixtures":[{"kind":"speech","request":{"text":"a","voice":"","style":""},"pcm_base64":"AAAA"}]}`,
		"prompt key":     `{"version":1,"fixtures":[{"kind":"speech","prompt":"a","pcm_base64":"AAA="}]}`,
		"request on txt": `{"version":1,"fixtures":[{"kind":"text","request":{"text":"a","voice":"","style":""},"response":"r"}]}`,
		"no pcm":         `{"version":1,"fixtures":[{"kind":"speech","request":{"text":"a","voice":"","style":""}}]}`,
	} {
		if _, err := LoadStubFixtures(writeFixtures(t, body)); err == nil {
			t.Errorf("%s: want a load error", name)
		}
	}
}
