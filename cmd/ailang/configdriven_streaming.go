package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// init registers the AI.streamCall effect operation.
//
// Lives in cmd/ailang rather than internal/effects to avoid an import cycle:
// internal/ai/configdriven imports internal/telemetry which imports
// internal/effects (for span emission). Putting the bridge in cmd/ailang
// inverts the dependency — cmd/ailang already imports both effects and
// configdriven without conflict.
//
// effects.RegisterOp is a runtime map insert, so registration here at
// package init runs after effects's own init has populated built-in ops;
// no ordering hazard.
func init() {
	effects.RegisterOp("AI", "streamCall", aiStreamCall)
}

// aiStreamCall implements AI.streamCall(provider, model, messages_json) ->
// Result[StreamConn, StreamError].
//
// Dispatches AI streaming through the M-AI-PROVIDER-CONFIG registry — the
// caller passes a registered provider name, never a URL or API key. URL +
// auth + body shape come from the [[ai_provider]] config harvested at CLI
// startup.
//
// Effect signature on the AILANG side: ! {AI, Stream, Net} — AI for cap +
// budget tracking (D11 compliance with M-AI-PROVIDER-CONFIG); Stream for the
// SSE event-loop machinery; Net for the underlying HTTP POST.
//
// Built-in providers (openai, anthropic, gemini, ollama, openrouter) are NOT
// reachable through this op — they have their own streaming code paths in
// future milestones, and v1 routes only config-driven providers here. Calls
// to a built-in name return ProviderNotFound.
func aiStreamCall(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: streamCall: expected 3 arguments (provider, model, messages_json), got %d", len(args))
	}
	providerVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: streamCall: expected string for provider, got %T", args[0])
	}
	modelVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: streamCall: expected string for model, got %T", args[1])
	}
	messagesVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: streamCall: expected string for messages_json, got %T", args[2])
	}

	providerName := providerVal.Value
	model := modelVal.Value
	messagesJSON := messagesVal.Value

	// Streaming requires both AI cap (for budget) and Stream cap (for the
	// SSE event loop). The AI handler bridges to the Stream layer here so
	// budget tracking applies to streaming calls identically to non-streaming.
	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	// Look up the registered config-driven provider. Built-ins are NOT in
	// this registry by design (D4 — built-ins stay built-in); a built-in
	// name returns ProviderNotFound here, which is the correct error for
	// streaming-via-config. Streaming for built-ins is out of scope for v1.
	registered, ok := ai.GlobalProviderRegistry.Lookup(providerName)
	if !ok {
		return makeStreamError("ProviderNotFound",
			fmt.Sprintf("AI provider %q is not registered as a config-driven provider. "+
				"Add an [[ai_provider]] block to ailang.toml or check installed packages.",
				providerName)), nil
	}

	// Only config-driven providers have a Spec() — built-in providers
	// implement ai.Provider differently and are out of scope for v1
	// streaming. Type-assert to access the streaming sub-block.
	configProvider, ok := registered.(*configdriven.Provider)
	if !ok {
		return makeStreamError("ProviderNotFound",
			fmt.Sprintf("provider %q is not a config-driven provider; v1 streaming only supports providers declared via [[ai_provider]]",
				providerName)), nil
	}
	spec := configProvider.Spec()

	// Construct the SSE-POST request: URL, body (with stream:true injected
	// for OpenAI-shaped providers), and headers (auth + custom). All env-var
	// references resolve at call time.
	streamReq, perr := configdriven.BuildStreamRequest(spec, model, messagesJSON)
	if perr != nil {
		return makeStreamError("ConnectionFailed", perr.Message), nil
	}

	// Trace event before the network call. Mirrors aiCall's trace structure
	// so streaming/non-streaming AI calls produce same-shaped trace data.
	ctx.RecordAIEffect("streamCall",
		[]string{providerName, model, truncateAIArg(messagesJSON)},
		"<streaming>",
		nil, // routing metadata not applicable to config-driven providers (D11)
	)

	// Build the StreamSSEPost args: [url: string, body: string, config: record{headers}].
	headersList := make([]eval.Value, 0, len(streamReq.Headers))
	for _, h := range streamReq.Headers {
		headersList = append(headersList, &eval.RecordValue{
			Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: h.Name},
				"value": &eval.StringValue{Value: h.Value},
			},
		})
	}
	configRec := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"headers": &eval.ListValue{Elements: headersList},
		},
	}

	return effects.StreamSSEPost(ctx, []eval.Value{
		&eval.StringValue{Value: streamReq.URL},
		&eval.StringValue{Value: streamReq.Body},
		configRec,
	})
}

// makeStreamError mirrors effects.makeStreamErr (which is unexported).
// Constructs an Err(StreamError variant(message)) ADT value.
func makeStreamError(variant, msg string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Err",
		Fields: []eval.Value{
			&eval.TaggedValue{
				CtorName: variant,
				Fields:   []eval.Value{&eval.StringValue{Value: msg}},
			},
		},
	}
}

// truncateAIArg mirrors the trace-arg truncation in internal/effects/ai.go.
// 256 chars matches the existing convention.
func truncateAIArg(s string) string {
	const maxLen = 256
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
