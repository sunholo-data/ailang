package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// init registers the AI.streamCall and AI.callStream effect operations.
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
	effects.RegisterOp("AI", "callStream", aiCallStream)
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
	// name produces a structured ProtocolError here. Streaming for
	// built-ins is out of scope for v1.
	//
	// Error encoding: AILANG's std/stream declares only five StreamErrorKind
	// variants (ConnectionFailed, Timeout, BudgetExhausted, ProtocolError,
	// MessageTooLarge). Provider-registry / capability misses are mapped to
	// ProtocolError with a structured "[<code>] <message>" prefix so callers
	// pattern-matching on the variant get the right shape AND callers
	// inspecting the message string can switch on the [code] tag. Adding
	// AI-specific variants to StreamErrorKind was rejected because the type
	// is generic; AI-specific error structure belongs in std/ai/streaming
	// (deferred to v1.1 alongside parseDelta).
	registered, ok := ai.GlobalProviderRegistry.Lookup(providerName)
	if !ok {
		return makeStreamError("ProtocolError",
			fmt.Sprintf("[ProviderNotFound] AI provider %q is not registered as a config-driven provider. Add an [[ai_provider]] block to ailang.toml or check installed packages.",
				providerName)), nil
	}

	// Only config-driven providers have a Spec() — built-in providers
	// implement ai.Provider differently and are out of scope for v1
	// streaming. Type-assert to access the streaming sub-block.
	configProvider, ok := registered.(*configdriven.Provider)
	if !ok {
		return makeStreamError("ProtocolError",
			fmt.Sprintf("[ProviderNotFound] provider %q is not a config-driven provider; v1 streaming only supports providers declared via [[ai_provider]]",
				providerName)), nil
	}
	spec := configProvider.Spec()

	// Construct the SSE-POST request: URL, body (with stream:true injected
	// for OpenAI-shaped providers), and headers (auth + custom). All env-var
	// references resolve at call time.
	streamReq, perr := configdriven.BuildStreamRequest(spec, model, messagesJSON)
	if perr != nil {
		// BuildStreamRequest's pre-flight failures (CapabilityNotSupported,
		// ModelNotAllowed, missing env vars) are configuration violations,
		// not transport failures — map to ProtocolError. The "[code]"
		// prefix is preserved in perr.Message so callers can switch on it.
		return makeStreamError("ProtocolError", perr.Message), nil
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

// ============================================================================
// M-AI-CALL-STREAM-HELPER (v0.15.1): synchronous accumulator wrapper
// ============================================================================
//
// aiCallStream implements AI.callStream(provider, model, messages_json) ->
// Result[string, AIError]. Synchronous accumulator wrapper around the
// streamCall op: opens the connection, drives the SSE event loop in Go,
// accumulates content deltas via the provider's [ai_provider.streaming]
// JSONPath config, and returns a single accumulated string (or AIError).
//
// CRITICAL: this op MUST NOT call ctx.RecordAIEffect — the underlying
// streamCall op (called via effects.Call) already records the trace span.
// Double-recording would emit two AI/streamCall events per logical call,
// breaking the "exactly one span per AI call" snapshot test that
// M-AI-STREAMING-HELPER M1 established.

// aiCallStream is the AI.callStream op handler. Args mirror aiStreamCall.
// Returns Result[string, AIError] as the AILANG-side ADT.
func aiCallStream(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callStream: expected 3 arguments (provider, model, messages_json), got %d", len(args))
	}
	providerVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callStream: expected string for provider, got %T", args[0])
	}
	if _, ok := args[1].(*eval.StringValue); !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callStream: expected string for model, got %T", args[1])
	}
	if _, ok := args[2].(*eval.StringValue); !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callStream: expected string for messages_json, got %T", args[2])
	}

	// Step 1: open the StreamConn via the existing streamCall op. This
	// performs all preflight (provider lookup, capability gate, body build,
	// auth) and emits the AI/streamCall trace span.
	streamCallResult, err := effects.Call(ctx, "AI", "streamCall", args)
	if err != nil {
		return nil, err
	}

	// Step 2: unwrap Result[StreamConn, StreamErrorKind]. On Err, map to AIError.
	tagged, ok := streamCallResult.(*eval.TaggedValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_INTERNAL: streamCall returned non-Result %T", streamCallResult)
	}
	if tagged.CtorName == "Err" {
		// Map StreamErrorKind variant + message to AIError record.
		return wrapErrAsAIError(tagged), nil
	}
	if tagged.CtorName != "Ok" || len(tagged.Fields) == 0 {
		return nil, fmt.Errorf("E_AI_INTERNAL: streamCall returned malformed Result")
	}

	// Step 3: get the StreamConn id and resolve the connection.
	connTagged, ok := tagged.Fields[0].(*eval.TaggedValue)
	if !ok || connTagged.CtorName != "StreamConn" || len(connTagged.Fields) == 0 {
		return nil, fmt.Errorf("E_AI_INTERNAL: streamCall Ok did not contain StreamConn(int)")
	}
	connIDVal, ok := connTagged.Fields[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_INTERNAL: StreamConn id is not int")
	}
	connID := int(connIDVal.Value)

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return makeAIErrorResult("ConnectionFailed",
			fmt.Sprintf("stream connection %d not found after open (already closed?)", connID),
			false), nil
	}

	// Defer disconnect so even error paths clean up.
	defer func() {
		conn.Close()
		ctx.Stream.ReleaseConnection(connID)
	}()

	// Step 4: look up the provider config to get streaming.delta_path +
	// done_sentinel. Lookup must succeed because streamCall just used it;
	// failure here is an internal invariant violation.
	registered, ok := ai.GlobalProviderRegistry.Lookup(providerVal.Value)
	if !ok {
		return makeAIErrorResult("ProviderNotFound",
			fmt.Sprintf("provider %q disappeared from registry mid-call", providerVal.Value),
			false), nil
	}
	configProvider, ok := registered.(*configdriven.Provider)
	if !ok {
		return makeAIErrorResult("ProviderNotFound",
			fmt.Sprintf("provider %q is not a config-driven provider", providerVal.Value),
			false), nil
	}
	spec := configProvider.Spec()
	deltaPath := spec.Streaming.DeltaPath
	doneSentinel := spec.Streaming.DoneSentinel

	// Step 5: drain events into a Go-side accumulator. Reasoning fields
	// (streaming.reasoning_path) are READ from the JSON but DISCARDED in
	// v0.15.1 — only visible content accumulates. v0.15.2's
	// callStreamWithReasoning will surface both.
	var acc strings.Builder
	var streamErr *aiStreamFailure // set if a Closed/error event terminates the loop with failure

	drainErr := conn.DrainEventsToFunc(func(evt effects.SSEEventInfo) bool {
		switch evt.Kind {
		case "sse_data":
			// Anthropic-style termination: message_stop event ends the stream.
			if evt.EventType == "message_stop" {
				return false
			}
			// OpenAI-style termination: bare "[DONE]" sentinel as the data field.
			if doneSentinel != "" && evt.Data == doneSentinel {
				return false
			}
			// Extract content delta via the configured JSONPath. Skip events
			// where extraction misses (e.g. message_start, content_block_start
			// which carry no text in delta_path).
			if delta := extractDeltaContent(evt.Data, deltaPath); delta != "" {
				acc.WriteString(delta)
			}
			return true
		case "closed":
			// Connection closed cleanly — happens when the upstream finishes
			// without an explicit done_sentinel. Treat as successful end.
			return false
		case "error":
			streamErr = &aiStreamFailure{
				code:      mapStreamErrTypeToAICode(evt.ErrType),
				message:   evt.Data,
				retryable: isAICodeRetryable(evt.ErrType),
			}
			return false
		default:
			// Other event kinds (ping, opened, message, binary) are not
			// expected on an SSE-POST stream from an AI provider; ignore.
			return true
		}
	})

	// Drain returned an error (idle timeout or done channel closed unexpectedly).
	if drainErr != nil {
		return makeAIErrorResult("Timeout", drainErr.Error(), true), nil
	}

	// Stream emitted a typed error event before terminating cleanly.
	if streamErr != nil {
		return makeAIErrorResult(streamErr.code, streamErr.message, streamErr.retryable), nil
	}

	// Happy path: return Ok(accumulated).
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.StringValue{Value: acc.String()}},
	}, nil
}

// aiStreamFailure captures a typed error encountered mid-stream.
type aiStreamFailure struct {
	code      string
	message   string
	retryable bool
}

// extractDeltaContent parses a single SSE data line as JSON and returns the
// content extracted at deltaPath, or "" if the path doesn't resolve. Empty
// string is a normal outcome (the event might be a content_block_start with
// no text yet) — callers treat empty deltas as no-ops.
//
// Uses encoding/json + a tiny path walker rather than the configdriven
// extractPath because deltaPath uses ailang.toml-syntax (e.g.
// "$.choices[0].delta.content") and we already have that parser. To keep
// this file dependency-light, we re-implement the minimal walker inline.
func extractDeltaContent(rawJSON, deltaPath string) string {
	if deltaPath == "" || rawJSON == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return ""
	}
	val, err := walkJSONPath(parsed, deltaPath)
	if err != nil || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// walkJSONPath is a minimal JSONPath subset matching the syntax in
// ailang.toml's [ai_provider.streaming.delta_path]:
//
//	$            -> root
//	$.foo        -> object field "foo"
//	$.foo.bar    -> nested fields
//	$[0]         -> array index
//	$.foo[0].bar -> mixed
//
// No filters, wildcards, or recursive descent. Mirrors the same subset as
// internal/ai/configdriven/jsonpath.go to keep the schema-vs-runtime
// behaviour consistent. Inline here so cmd/ailang doesn't depend on the
// configdriven package's path internals.
func walkJSONPath(root any, path string) (any, error) {
	if path == "$" {
		return root, nil
	}
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("path must start with $: %q", path)
	}
	rest := path[1:]
	cur := root
	for len(rest) > 0 {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			end := strings.IndexAny(rest, ".[")
			var field string
			if end == -1 {
				field, rest = rest, ""
			} else {
				field, rest = rest[:end], rest[end:]
			}
			obj, ok := cur.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected object at field %q", field)
			}
			val, exists := obj[field]
			if !exists {
				return nil, fmt.Errorf("missing field %q", field)
			}
			cur = val
		case '[':
			end := strings.Index(rest, "]")
			if end == -1 {
				return nil, fmt.Errorf("unclosed [")
			}
			idxStr := rest[1:end]
			rest = rest[end+1:]
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array at index %s", idxStr)
			}
			var idx int
			if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
				return nil, fmt.Errorf("non-integer index %q", idxStr)
			}
			if idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("index %d out of bounds", idx)
			}
			cur = arr[idx]
		default:
			return nil, fmt.Errorf("unexpected char %q", rest[0])
		}
	}
	return cur, nil
}

// wrapErrAsAIError converts an Err(StreamErrorKind variant) into an
// Err(AIError record) for the callStream return type. Parses the
// "[<code>]" prefix in ProtocolError messages (set by aiStreamCall and
// BuildStreamRequest) so callers see structured codes like ProviderNotFound
// and CapabilityNotSupported rather than the generic ProtocolError variant.
func wrapErrAsAIError(errResult *eval.TaggedValue) eval.Value {
	if len(errResult.Fields) == 0 {
		return makeAIErrorResult("ConnectionFailed", "unknown error (empty Err)", false)
	}
	innerErr, ok := errResult.Fields[0].(*eval.TaggedValue)
	if !ok {
		return makeAIErrorResult("ConnectionFailed", "unknown error (non-tagged inner)", false)
	}
	streamErrKind := innerErr.CtorName
	var msg string
	if len(innerErr.Fields) > 0 {
		if sv, ok := innerErr.Fields[0].(*eval.StringValue); ok {
			msg = sv.Value
		}
	}

	// ProtocolError messages carry a [<code>] prefix when produced by
	// aiStreamCall or BuildStreamRequest. Parse it back out so the AIError
	// returned to AILANG callers has the specific code.
	code := streamErrKind
	if streamErrKind == "ProtocolError" {
		if extracted, rest := parseCodePrefix(msg); extracted != "" {
			code = extracted
			msg = rest
		}
	}
	return makeAIErrorResult(code, msg, isAICodeRetryable(code))
}

// parseCodePrefix extracts a "[CODE] rest" prefix from a message string.
// Returns ("", message) if no prefix is found.
func parseCodePrefix(msg string) (code, rest string) {
	if !strings.HasPrefix(msg, "[") {
		return "", msg
	}
	end := strings.Index(msg, "]")
	if end < 0 {
		return "", msg
	}
	code = msg[1:end]
	rest = strings.TrimPrefix(msg[end+1:], " ")
	return code, rest
}

// mapStreamErrTypeToAICode maps a streamEvent.errType (set by readLoop /
// stream_sse.go) to an AIError code. Most pass through unchanged; the
// distinction matters when the upstream emits a typed error event mid-stream
// (e.g. retryable 5xx) vs the connection failing at open time.
func mapStreamErrTypeToAICode(errType string) string {
	if errType == "" {
		return "ConnectionFailed"
	}
	return errType
}

// isAICodeRetryable returns the conventional retryable-ness of an AI error
// code. Matches the existing taxonomy in internal/ai/openrouter/types.go
// and similar.
func isAICodeRetryable(code string) bool {
	switch code {
	case "Timeout", "ConnectionFailed":
		return true
	case "BudgetExhausted", "AuthFailed", "ProviderNotFound", "CapabilityNotSupported", "ModelNotAllowed":
		return false
	}
	// Default: assume retryable for unknown codes (caller can override).
	return true
}

// makeAIErrorResult builds an Err(AIError record) AILANG value.
// AIError shape: { code: string, message: string, retryable: bool }
// matching std/ai/streaming.ail's AIError type.
func makeAIErrorResult(code, message string, retryable bool) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Err",
		Fields: []eval.Value{
			&eval.RecordValue{
				Fields: map[string]eval.Value{
					"code":      &eval.StringValue{Value: code},
					"message":   &eval.StringValue{Value: message},
					"retryable": &eval.BoolValue{Value: retryable},
				},
			},
		},
	}
}
