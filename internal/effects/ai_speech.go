package effects

// AI.callSpeech (#1495): text-to-speech returning Result[bytes, AIError].
// The bytes are raw PCM, s16le, mono, 24 kHz (ai.SpeechSampleRate) — the
// format std/audio's wavFromPcm/encode take. Speech is an OPTIONAL handler
// capability (AISpeechHandler), so handlers that predate it keep compiling
// and a handler without it fails with CapabilityNotSupported.

import (
	"encoding/binary"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// AISpeechHandler is implemented by AI handlers that can synthesise speech.
// CallSpeech returns s16le mono PCM at ai.SpeechSampleRate.
type AISpeechHandler interface {
	CallSpeech(text, voice, style, options string) ([]byte, error)
}

// CallSpeech dispatches to the handler's speech capability.
func (c *AIContext) CallSpeech(text, voice, style, options string) ([]byte, error) {
	if c.handler == nil {
		return nil, ErrNoAIHandler
	}
	sh, ok := c.handler.(AISpeechHandler)
	if !ok {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			fmt.Sprintf("callSpeech: the configured AI handler (%T) does not support text-to-speech", c.handler), false)
	}
	return sh.CallSpeech(text, voice, style, options)
}

// Stub tone: 250 ms of a 400 Hz triangle wave at 24 kHz, amplitude 8192,
// generated with integer arithmetic only so it is byte-identical on every
// platform. Independent of the request: the stub proves the wiring and the
// format, fixtures (kind "speech") supply realistic audio.
const (
	stubToneSamples = ai.SpeechSampleRate / 4 // 250 ms
	stubTonePeriod  = ai.SpeechSampleRate / 400
	stubToneAmp     = 8192
)

// StubSpeechPCM is the deterministic PCM every --ai-stub callSpeech returns.
func StubSpeechPCM() []byte {
	out := make([]byte, 2*stubToneSamples)
	half := stubTonePeriod / 2
	for i := 0; i < stubToneSamples; i++ {
		ph := i % stubTonePeriod
		var v int
		if ph < half {
			v = -stubToneAmp + 2*stubToneAmp*ph/half
		} else {
			v = stubToneAmp - 2*stubToneAmp*(ph-half)/half
		}
		binary.LittleEndian.PutUint16(out[2*i:], uint16(int16(v)))
	}
	return out
}

// CallSpeech returns the stub tone.
func (h *StubAIHandler) CallSpeech(text, voice, style, options string) ([]byte, error) {
	if _, err := ai.ParseSpeechOptions(options); err != nil {
		return nil, err
	}
	return StubSpeechPCM(), nil
}

func init() {
	RegisterOp("AI", "callSpeech", aiCallSpeech)
}

// aiCallSpeech implements AI.callSpeech(text, voice, style, options)
// -> Result[bytes, AIError].
func aiCallSpeech(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callSpeech: expected 4 arguments, got %d", len(args))
	}
	strs := make([]string, 4)
	for i, name := range []string{"text", "voice", "style", "options"} {
		s, ok := args[i].(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("E_AI_TYPE_ERROR: callSpeech: expected string %s, got %T", name, args[i])
		}
		strs[i] = s.Value
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}
	traceArgs := []string{truncateForTrace(strs[0]), strs[1], truncateForTrace(strs[2])}
	pcm, err := ctx.AI.CallSpeech(strs[0], strs[1], strs[2], strs[3])
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("callSpeech", traceArgs, fmt.Sprintf(errResultPrefix, aiErr.Code), ctx.AI.LastRoutingMetadata())
		return makeAIErrorResultRecord(aiErr), nil
	}
	ctx.RecordAIEffect("callSpeech", traceArgs, fmt.Sprintf("pcm:%d bytes", len(pcm)), ctx.AI.LastRoutingMetadata())
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.BytesValue{Value: pcm, MimeType: "audio/L16;rate=24000"}},
	}, nil
}
