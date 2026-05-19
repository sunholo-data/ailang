//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// jsRejectionToString extracts a human-readable string from a JS rejection
// value. Plain strings pass through; Error objects yield their .message
// field; other objects yield their .toString() result. This avoids the
// "<object>" fallback from js.Value.String() on Type=Object values.
func jsRejectionToString(v js.Value) string {
	switch v.Type() {
	case js.TypeString:
		return v.String()
	case js.TypeObject:
		// Prefer .message (standard Error objects).
		if m := v.Get("message"); m.Type() == js.TypeString {
			s := m.String()
			if s != "" {
				return s
			}
		}
		// Fall back to JS .toString() (catches custom error shapes).
		if v.Get("toString").Truthy() {
			return v.Call("toString").String()
		}
		return "<object>"
	default:
		return v.String()
	}
}

// awaitJSResult handles both sync returns and Promise returns from JS callbacks.
// If the result has a .then method (i.e. is a Promise), it awaits resolution.
func awaitJSResult(result js.Value) (js.Value, error) {
	if result.Type() == js.TypeObject && result.Get("then").Truthy() {
		return awaitPromise(result)
	}
	return result, nil
}

// awaitPromise blocks the current goroutine until the JS Promise resolves or rejects.
// This works in Go WASM because js.FuncOf callbacks run as goroutines,
// so blocking one goroutine doesn't freeze the browser event loop.
func awaitPromise(promise js.Value) (js.Value, error) {
	ch := make(chan js.Value, 1)
	errCh := make(chan error, 1)

	thenFn := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			ch <- args[0]
		} else {
			ch <- js.Undefined()
		}
		return nil
	})
	defer thenFn.Release()

	catchFn := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		msg := "effect handler error"
		if len(args) > 0 {
			msg = jsRejectionToString(args[0])
		}
		errCh <- fmt.Errorf("%s", msg)
		return nil
	})
	defer catchFn.Release()

	promise.Call("then", thenFn).Call("catch", catchFn)

	select {
	case val := <-ch:
		return val, nil
	case err := <-errCh:
		return js.Undefined(), err
	}
}

// registerJSEffectHandler overrides an effect's operations with JS callbacks.
// handlers is a JS object: {opName: function, ...}
// Each callback receives AILANG values converted to JS primitives and should
// return a value (sync) or a Promise (async).
func registerJSEffectHandler(effectName string, handlers js.Value) {
	keys := js.Global().Get("Object").Call("keys", handlers)
	for i := 0; i < keys.Length(); i++ {
		opName := keys.Index(i).String()
		callback := handlers.Get(opName)

		// Capture for closure
		cb := callback
		effects.RegisterOp(effectName, opName, func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			// Convert AILANG args to JS values
			jsArgs := make([]interface{}, len(args))
			for i, arg := range args {
				jsArgs[i] = ailangValueToJS(arg)
			}
			// Call JS callback
			result := cb.Invoke(jsArgs...)
			// Await if Promise
			resolved, err := awaitJSResult(result)
			if err != nil {
				return nil, err
			}
			// Convert JS result back to AILANG value
			return jsToAILANGValue(resolved), nil
		})
	}
}

// WasmAIHandler implements effects.AIHandler via JS callbacks.
// It bridges the Go AIHandler interface to JavaScript functions
// that can call external APIs (e.g., Anthropic / OpenRouter via fetch).
//
// Five callbacks total — one per AIHandler method:
//   - callback:                 ai.call / ai.callJson / ai.callImage*
//   - stepCallback:             ai.step (M-WASM-AI-STEP-BYO-KEY)
//   - stepWithCacheCallback:    ai.stepWithCache (M-WASM-AI-STEP-BYO-KEY)
//   - stepWithStreamCallback:   ai.stepWithStream (M-WASM-AI-STEP-BYO-KEY)
//
// Each is registered separately via its own ailangSet*Handler global.
// A nil/missing callback for a given method returns a typed error
// rather than panicking, so AILANG-side code sees a normal AIError.
type WasmAIHandler struct {
	callback               js.Value
	stepCallback           js.Value
	stepWithCacheCallback  js.Value
	stepWithStreamCallback js.Value
}

// Call invokes the JS callback with the input string.
// Supports both sync returns and Promise returns (async).
func (h *WasmAIHandler) Call(input string) (string, error) {
	result := h.callback.Invoke(input)
	resolved, err := awaitJSResult(result)
	if err != nil {
		return "", err
	}
	return resolved.String(), nil
}

// CallJson invokes the JS callback requesting JSON output.
// WASM delegates to the same JS callback — structured output is not natively enforced.
func (h *WasmAIHandler) CallJson(input string, schema string) (string, error) {
	return h.Call(input)
}

// CallImage is not supported in WASM — image generation requires server-side file I/O.
func (h *WasmAIHandler) CallImage(prompt, outputPath, options string) (string, error) {
	return "", fmt.Errorf("image generation not supported in WASM environment")
}

// CallImageBase64 delegates to the JS callback for image generation.
// The JS host is responsible for returning JSON: {"base64": "...", "mime_type": "..."}
func (h *WasmAIHandler) CallImageBase64(prompt, options string) (string, error) {
	return h.Call(prompt)
}

// Step invokes the registered JS step callback with the model + messages +
// tools. The callback should fetch the provider directly (BYO-key from
// localStorage) and return a JS object with the response shape.
// (M-WASM-AI-STEP-BYO-KEY, v0.19.0.)
//
// Expected JS callback signature:
//
//	async (model, messages, tools) => ({
//	  message: { role, content, tool_calls, tool_call_id },
//	  tool_calls: [...],
//	  finish_reason: "stop" | "tool_calls" | ...,
//	  input_tokens: N, output_tokens: N,
//	  cache_read_input_tokens: N, cache_creation_input_tokens: N
//	})
//
// Returns a clean Go error when no handler was registered (callable via
// ailangSetAIStepHandler). The aiStep effect op upstream classifies that
// into Err(AIError{code:"no_handler"}) for the AILANG caller.
func (h *WasmAIHandler) Step(model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error) {
	if !h.stepCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: ai.step requires a JS handler — call ailangSetAIStepHandler(fn) first")
	}
	jsModel := js.ValueOf(model)
	jsMsgs := js.ValueOf(messagesToJSCompat(messages))
	jsTools := js.ValueOf(toolsToJSCompat(tools))
	result := h.stepCallback.Invoke(jsModel, jsMsgs, jsTools)
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return jsToResponse(resolved)
}

// StepWithCache mirrors Step but adds cache_breakpoints — opt-in
// prompt-cache hints (Anthropic's cache_control). The JS callback
// receives a 4th arg as a JS array of {position, ttl} objects.
// Empty cache_breakpoints serialize to a JS empty array, not null.
// (M-WASM-AI-STEP-BYO-KEY, v0.19.0.)
func (h *WasmAIHandler) StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint) (*ai.Response, error) {
	if !h.stepWithCacheCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: ai.stepWithCache requires a JS handler — call ailangSetAIStepWithCacheHandler(fn) first")
	}
	jsModel := js.ValueOf(model)
	jsMsgs := js.ValueOf(messagesToJSCompat(messages))
	jsTools := js.ValueOf(toolsToJSCompat(tools))
	jsBreakpoints := js.ValueOf(cacheBreakpointsToJSCompat(cacheBreakpoints))
	result := h.stepWithCacheCallback.Invoke(jsModel, jsMsgs, jsTools, jsBreakpoints)
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return jsToResponse(resolved)
}

// StepWithStream mirrors StepWithCache but adds a per-chunk callback. The
// JS handler receives the wrapped Go callback as its 5th arg (as a JS
// function) and fires it for each StreamChunk variant — ContentDelta,
// ThinkingDelta, Usage — by passing a JS object {kind, text} or
// {kind, input_tokens, output_tokens, cache_read_input_tokens,
// cache_creation_input_tokens} at end-of-stream.
//
// The JS-side onChunk wrapper is freed via funcWrapper.Release() once
// the JS handler resolves its Promise, preventing a memory leak from
// long-lived AILANG closures.
// (M-WASM-AI-STEP-BYO-KEY, v0.19.0.)
func (h *WasmAIHandler) StepWithStream(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	if !h.stepWithStreamCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: ai.stepWithStream requires a JS handler — call ailangSetAIStepWithStreamHandler(fn) first")
	}
	// Wrap the Go callback as a JS function. The JS handler invokes this
	// per chunk; we translate the JS object to the appropriate StreamChunk
	// variant and call onChunk.
	funcWrapper := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		chunk := jsToStreamChunk(args[0])
		if chunk != nil {
			onChunk(chunk)
		}
		return nil
	})
	defer funcWrapper.Release()

	jsModel := js.ValueOf(model)
	jsMsgs := js.ValueOf(messagesToJSCompat(messages))
	jsTools := js.ValueOf(toolsToJSCompat(tools))
	jsBreakpoints := js.ValueOf(cacheBreakpointsToJSCompat(cacheBreakpoints))
	result := h.stepWithStreamCallback.Invoke(jsModel, jsMsgs, jsTools, jsBreakpoints, funcWrapper)
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return jsToResponse(resolved)
}

// ─── JS conversion helpers (M-WASM-AI-STEP-BYO-KEY) ───────────────────
// Pure-Go conversion helpers (messagesToJSCompat, toolsToJSCompat,
// cacheBreakpointsToJSCompat) live in effects_helpers.go without a build
// tag so they can be unit-tested on the host. The js.Value-consuming
// helpers below stay here, gated by //go:build js && wasm.

// jsToResponse decodes the JS handler's response object into *ai.Response.
// Tolerates two tool_calls layouts: top-level (`v.tool_calls`) or nested
// under message (`v.message.tool_calls`). Token-count fields default to
// zero when absent. Returns an error when the JS value isn't an object
// (e.g., the handler accidentally returned a string error).
func jsToResponse(v js.Value) (*ai.Response, error) {
	if v.Type() != js.TypeObject {
		return nil, fmt.Errorf("step response must be a JS object, got %s (value=%v)", v.Type(), v)
	}

	var text string
	msg := v.Get("message")
	if msg.Type() == js.TypeObject {
		if c := msg.Get("content"); c.Type() == js.TypeString {
			text = c.String()
		}
	}

	// Prefer top-level tool_calls (provider canonical), fall back to nested.
	toolCallsJS := v.Get("tool_calls")
	if !toolCallsJS.Truthy() && msg.Type() == js.TypeObject {
		toolCallsJS = msg.Get("tool_calls")
	}
	toolCalls := jsToToolCalls(toolCallsJS)

	return &ai.Response{
		Text:                     text,
		ToolCalls:                toolCalls,
		InputTokens:              jsGetInt(v, "input_tokens"),
		OutputTokens:             jsGetInt(v, "output_tokens"),
		CacheReadInputTokens:     jsGetInt(v, "cache_read_input_tokens"),
		CacheCreationInputTokens: jsGetInt(v, "cache_creation_input_tokens"),
		FinishReason:             jsGetString(v, "finish_reason"),
		Model:                    jsGetString(v, "model"),
	}, nil
}

// jsToToolCalls decodes a JS array of tool-call objects. Each entry may be
// in either of two shapes (provider variation):
//
//	{id: "...", name: "...", arguments: "..."}            (flat — the AILANG canonical shape)
//	{id: "...", function: {name: "...", arguments: "..."}} (OpenAI nested)
//
// Returns nil for non-array inputs (handles missing-field gracefully).
func jsToToolCalls(v js.Value) []ai.ToolCall {
	if !v.Truthy() {
		return nil
	}
	arrayCtor := js.Global().Get("Array")
	if arrayCtor.Truthy() && !v.InstanceOf(arrayCtor) {
		return nil
	}
	n := v.Length()
	out := make([]ai.ToolCall, 0, n)
	for i := 0; i < n; i++ {
		entry := v.Index(i)
		if entry.Type() != js.TypeObject {
			continue
		}
		tc := ai.ToolCall{
			ID: jsGetString(entry, "id"),
		}
		// Try nested {function: {name, arguments}} first
		fn := entry.Get("function")
		if fn.Type() == js.TypeObject {
			tc.Name = jsGetString(fn, "name")
			tc.Arguments = jsGetString(fn, "arguments")
		} else {
			// Flat shape
			tc.Name = jsGetString(entry, "name")
			tc.Arguments = jsGetString(entry, "arguments")
		}
		out = append(out, tc)
	}
	return out
}

// jsToStreamChunk decodes a JS chunk object into an ai.StreamChunk variant.
// Discriminator: `kind` field, one of "ContentDelta" | "ThinkingDelta" | "Usage".
// Returns nil for unknown kinds (caller skips the chunk silently).
func jsToStreamChunk(v js.Value) ai.StreamChunk {
	if v.Type() != js.TypeObject {
		return nil
	}
	switch jsGetString(v, "kind") {
	case "ContentDelta":
		return ai.StreamContentDelta{Text: jsGetString(v, "text")}
	case "ThinkingDelta":
		return ai.StreamThinkingDelta{Text: jsGetString(v, "text")}
	case "Usage":
		return ai.StreamUsage{
			InputTokens:              jsGetInt(v, "input_tokens"),
			OutputTokens:             jsGetInt(v, "output_tokens"),
			CacheReadInputTokens:     jsGetInt(v, "cache_read_input_tokens"),
			CacheCreationInputTokens: jsGetInt(v, "cache_creation_input_tokens"),
		}
	}
	return nil
}

// jsGetString safely extracts a string field from a JS object. Returns "" for
// missing/non-string fields.
func jsGetString(v js.Value, key string) string {
	f := v.Get(key)
	if f.Type() == js.TypeString {
		return f.String()
	}
	return ""
}

// jsGetInt safely extracts an integer field from a JS object. Returns 0 for
// missing/non-numeric fields. Truncates floats to int.
func jsGetInt(v js.Value, key string) int {
	f := v.Get(key)
	if f.Type() == js.TypeNumber {
		return int(f.Float())
	}
	return 0
}

// ailangValueToJS converts an AILANG eval.Value to a JS-compatible interface{}.
func ailangValueToJS(v eval.Value) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case *eval.IntValue:
		return val.Value
	case *eval.FloatValue:
		return val.Value
	case *eval.StringValue:
		return val.Value
	case *eval.BoolValue:
		return val.Value
	case *eval.BytesValue:
		// Convert to Uint8Array for JS interop (binary data stays binary)
		arr := js.Global().Get("Uint8Array").New(len(val.Value))
		js.CopyBytesToJS(arr, val.Value)
		return arr
	case *eval.UnitValue:
		return nil
	case *eval.FunctionValue:
		// Wrap AILANG closure as callable JS function (M-WASM-STREAM-BRIDGE)
		// This enables patterns like onEvent(conn, handler) where AILANG passes
		// a closure to be invoked by JS on each WebSocket message.
		closure := val
		return js.FuncOf(func(this js.Value, jsArgs []js.Value) interface{} {
			ailangArgs := make([]eval.Value, len(jsArgs))
			for i, jsArg := range jsArgs {
				ailangArgs[i] = jsToAILANGValue(jsArg)
			}
			result, err := replInstance.repl.ApplyClosure(closure, ailangArgs)
			if err != nil {
				js.Global().Get("console").Call("error", "AILANG closure error: "+err.Error())
				return nil
			}
			return ailangValueToJS(result)
		})
	case *eval.TaggedValue:
		// Convert ADT to JS object with _ctor/_fields convention (M-WASM-STREAM-BRIDGE Phase 2)
		// This enables round-trip: AILANG ADT → JS object → pattern match in JS or back to AILANG.
		obj := js.Global().Get("Object").New()
		obj.Set("_ctor", val.CtorName)
		fieldsArr := js.Global().Get("Array").New(len(val.Fields))
		for i, f := range val.Fields {
			fieldsArr.SetIndex(i, ailangValueToJS(f))
		}
		obj.Set("_fields", fieldsArr)
		return obj
	case *eval.ListValue:
		arr := js.Global().Get("Array").New(len(val.Elements))
		for i, elem := range val.Elements {
			arr.SetIndex(i, ailangValueToJS(elem))
		}
		return arr
	case *eval.RecordValue:
		obj := js.Global().Get("Object").New()
		for k, v := range val.Fields {
			obj.Set(k, ailangValueToJS(v))
		}
		return obj
	default:
		// Fallback: string representation
		return formatValue(v)
	}
}

// jsToAILANGValue converts a JS value to an AILANG eval.Value.
func jsToAILANGValue(v js.Value) eval.Value {
	switch v.Type() {
	case js.TypeNumber:
		f := v.Float()
		if f == float64(int(f)) && f >= -1e15 && f <= 1e15 {
			return &eval.IntValue{Value: int(f)}
		}
		return &eval.FloatValue{Value: f}
	case js.TypeString:
		return &eval.StringValue{Value: v.String()}
	case js.TypeBoolean:
		return &eval.BoolValue{Value: v.Bool()}
	case js.TypeNull, js.TypeUndefined:
		return &eval.UnitValue{}
	default:
		// Check for Uint8Array (binary data from JS)
		if v.InstanceOf(js.Global().Get("Uint8Array")) {
			length := v.Get("length").Int()
			buf := make([]byte, length)
			js.CopyBytesToGo(buf, v)
			return &eval.BytesValue{Value: buf}
		}
		// Check for ADT convention: {_ctor: "Name", _fields: [...]}
		// This enables JS effect handlers to return properly-typed ADT values
		// that AILANG code can pattern-match on (e.g., Ok(StreamConn(1))).
		ctorVal := v.Get("_ctor")
		if ctorVal.Type() == js.TypeString {
			ctor := ctorVal.String()
			var fields []eval.Value
			fieldsVal := v.Get("_fields")
			if fieldsVal.Type() != js.TypeUndefined && fieldsVal.Type() != js.TypeNull {
				for i := 0; i < fieldsVal.Length(); i++ {
					fields = append(fields, jsToAILANGValue(fieldsVal.Index(i)))
				}
			}
			return &eval.TaggedValue{
				CtorName: ctor,
				Fields:   fields,
			}
		}
		// Check for Array → ListValue
		if v.InstanceOf(js.Global().Get("Array")) {
			elements := make([]eval.Value, v.Length())
			for i := 0; i < v.Length(); i++ {
				elements[i] = jsToAILANGValue(v.Index(i))
			}
			return &eval.ListValue{Elements: elements}
		}
		// Plain object → RecordValue
		if v.Type() == js.TypeObject {
			keys := js.Global().Get("Object").Call("keys", v)
			fields := make(map[string]eval.Value, keys.Length())
			for i := 0; i < keys.Length(); i++ {
				key := keys.Index(i).String()
				fields[key] = jsToAILANGValue(v.Get(key))
			}
			return &eval.RecordValue{Fields: fields}
		}
		// Fallback: string representation
		return &eval.StringValue{Value: v.String()}
	}
}

// --- JS Global Functions ---

// setEffectHandler: ailangSetEffectHandler(effectName, {opName: fn, ...})
// Overrides any effect's operations with JS callbacks and auto-grants the capability.
func setEffectHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetEffectHandler requires 2 arguments: effectName and handlers object",
		}
	}

	effectName := args[0].String()
	handlers := args[1]

	if effectName == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "effect name cannot be empty",
		}
	}

	// Register JS callbacks as effect ops
	registerJSEffectHandler(effectName, handlers)

	// Auto-grant the capability
	replInstance.repl.GrantCapability(effectName)

	return map[string]interface{}{
		"success": true,
	}
}

// wasmAIHandlerSingleton is a process-wide WasmAIHandler that all four
// setters (ailangSetAIHandler + ailangSetAIStep*Handler) mutate. Without
// this, calling e.g. ailangSetAIStepHandler after ailangSetAIHandler would
// replace the entire handler and clobber the previously-registered call
// callback. The singleton lets users register the four callbacks
// independently in any order.
//
// The first set-call also wires the singleton into the REPL via
// replInstance.repl.SetAIHandler. Subsequent set-calls just mutate the
// singleton in-place; the REPL still holds the same pointer.
var wasmAIHandlerSingleton *WasmAIHandler

// getOrCreateAIHandler returns the singleton WasmAIHandler, creating and
// wiring it to the REPL on first call.
func getOrCreateAIHandler() *WasmAIHandler {
	if wasmAIHandlerSingleton == nil {
		wasmAIHandlerSingleton = &WasmAIHandler{}
		replInstance.repl.SetAIHandler(wasmAIHandlerSingleton)
	}
	return wasmAIHandlerSingleton
}

// setAIHandler: ailangSetAIHandler(fn)
// Wires the JS callback for ai.call / ai.callJson / ai.callImage*.
// Mutates the singleton WasmAIHandler — safe to call before/after the
// step setters.
func setAIHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetAIHandler requires 1 argument: callback function",
		}
	}
	getOrCreateAIHandler().callback = args[0]
	return map[string]interface{}{"success": true}
}

// setAIStepHandler: ailangSetAIStepHandler(fn)
// Wires the JS callback for ai.step. The callback receives
// (model, messages, tools) and returns a Promise resolving to the
// step response object. M-WASM-AI-STEP-BYO-KEY (v0.19.0).
func setAIStepHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetAIStepHandler requires 1 argument: callback function",
		}
	}
	getOrCreateAIHandler().stepCallback = args[0]
	return map[string]interface{}{"success": true}
}

// setAIStepWithCacheHandler: ailangSetAIStepWithCacheHandler(fn)
// Wires the JS callback for ai.stepWithCache. The callback receives
// (model, messages, tools, cache_breakpoints) and returns a Promise
// resolving to the step response object. M-WASM-AI-STEP-BYO-KEY (v0.19.0).
func setAIStepWithCacheHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetAIStepWithCacheHandler requires 1 argument: callback function",
		}
	}
	getOrCreateAIHandler().stepWithCacheCallback = args[0]
	return map[string]interface{}{"success": true}
}

// setAIStepWithStreamHandler: ailangSetAIStepWithStreamHandler(fn)
// Wires the JS callback for ai.stepWithStream. The callback receives
// (model, messages, tools, cache_breakpoints, onChunk) and returns a
// Promise resolving to the final step response. The 5th arg is a JS
// function the handler invokes per chunk; AILANG translates each
// invocation back to a StreamChunk variant. M-WASM-AI-STEP-BYO-KEY (v0.19.0).
func setAIStepWithStreamHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetAIStepWithStreamHandler requires 1 argument: callback function",
		}
	}
	getOrCreateAIHandler().stepWithStreamCallback = args[0]
	return map[string]interface{}{"success": true}
}

// grantCapability: ailangGrantCapability(name)
// Grants a named capability to the effect context.
func grantCapability(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangGrantCapability requires 1 argument: capability name",
		}
	}

	name := args[0].String()
	replInstance.repl.GrantCapability(name)

	return map[string]interface{}{
		"success": true,
	}
}

// ============================================================================
// Cognitive OS — DOM effect bridge (M-COG-RUNTIME, v0.21.x)
// ============================================================================
//
// WasmDOMHandler implements effects.DOMHandler via JS callbacks. Each
// step-pattern method (ApplyPatch / ApplyBatch / Subscribe) wires to its
// own JS callback so the browser host can implement them independently
// — matching the WasmAIHandler / setAIHandler+setAIStepHandler split.
//
// Subscribe is the trickiest: it must register a long-lived JS callback
// and return a cancel function that frees the JS reference. The cancel
// function calls back into JS via the host-provided cancel registry.

// WasmDOMHandler implements effects.DOMHandler.
//
// Each callback is registered separately via its own ailangSet*Handler
// global. A nil/missing callback returns a typed Go error so AILANG-side
// Result-returning ops surface as Err({code: "NO_HANDLER", ...}).
type WasmDOMHandler struct {
	applyPatchCallback js.Value
	applyBatchCallback js.Value
	subscribeCallback  js.Value
}

// ApplyPatch invokes the JS applyPatch callback synchronously (or awaits
// a Promise) and decodes the response into a *PatchResult.
//
// JS callback signature:
//
//	(region, patch) => ({node_id: "...", budget_remaining: N})
//
// where `patch` is the structured DOMPatch ADT serialized as a tagged JS
// object: {ctor: "AddPanel", fields: ["title", "content"]} (etc.).
func (h *WasmDOMHandler) ApplyPatch(region effects.RegionID, patch effects.DOMPatch) (*effects.PatchResult, error) {
	if !h.applyPatchCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: DOM.applyPatch requires a JS handler — call ailangSetDOMApplyPatchHandler(fn) first")
	}
	jsPatch := domPatchToJS(patch)
	result := h.applyPatchCallback.Invoke(string(region), jsPatch)
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return &effects.PatchResult{
		NodeID:          effects.DOMNodeID(jsGetString(resolved, "node_id")),
		BudgetRemaining: jsGetInt(resolved, "budget_remaining"),
	}, nil
}

// ApplyBatch invokes the JS applyBatch callback. Transactional semantics
// are the JS host's responsibility — Go-side just decodes the response.
//
// JS callback signature:
//
//	(region, [patch1, patch2, ...]) => ({node_ids: ["..."], budget_remaining: N})
func (h *WasmDOMHandler) ApplyBatch(region effects.RegionID, patches []effects.DOMPatch) (*effects.BatchResult, error) {
	if !h.applyBatchCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: DOM.applyBatch requires a JS handler — call ailangSetDOMApplyBatchHandler(fn) first")
	}
	jsPatches := make([]interface{}, len(patches))
	for i, p := range patches {
		jsPatches[i] = domPatchToJS(p)
	}
	result := h.applyBatchCallback.Invoke(string(region), js.ValueOf(jsPatches))
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	idsJS := resolved.Get("node_ids")
	ids := []effects.DOMNodeID{}
	if idsJS.Truthy() {
		for i := 0; i < idsJS.Length(); i++ {
			ids = append(ids, effects.DOMNodeID(idsJS.Index(i).String()))
		}
	}
	return &effects.BatchResult{
		NodeIDs:         ids,
		BudgetRemaining: jsGetInt(resolved, "budget_remaining"),
	}, nil
}

// Subscribe is intentionally NOT wired in this iteration.
//
// The lifecycle (long-lived JS callback ↔ Go onEvent closure ↔ cancel)
// requires careful management to avoid leaking js.Func references. M2's
// scheduler is the primary consumer; wiring lands when the scheduler
// arrives. For now, returns a typed no_handler error so M1's Subscribe
// callers see a clean failure.
func (h *WasmDOMHandler) Subscribe(region effects.RegionID, eventTypes []string, onEvent func(effects.DOMEvent)) (func(), error) {
	return nil, fmt.Errorf("not_implemented: DOM.subscribe wiring lands in M2 alongside the deterministic scheduler")
}

// domPatchToJS serializes a Go-side DOMPatch into a JS-friendly tagged
// object: {ctor, fields}. The browser host pattern-matches on ctor.
//
// New patch variants must be added in lockstep with effects/dom.go and
// std/dom.ail.
func domPatchToJS(p effects.DOMPatch) interface{} {
	switch v := p.(type) {
	case effects.PatchAddPanel:
		return map[string]interface{}{
			"ctor":   "AddPanel",
			"fields": []interface{}{v.Title, v.Content},
		}
	case effects.PatchUpdateNode:
		return map[string]interface{}{
			"ctor":   "UpdateNode",
			"fields": []interface{}{string(v.Node), v.Content},
		}
	case effects.PatchRemoveNode:
		return map[string]interface{}{
			"ctor":   "RemoveNode",
			"fields": []interface{}{string(v.Node)},
		}
	case effects.PatchAddTimeline:
		return map[string]interface{}{
			"ctor":   "AddTimeline",
			"fields": []interface{}{v.Title},
		}
	default:
		return map[string]interface{}{
			"ctor":   "Unknown",
			"fields": []interface{}{},
		}
	}
}

// wasmDOMHandlerSingleton is the process-wide handler; the three setters
// mutate fields on it (matches wasmAIHandlerSingleton pattern).
var wasmDOMHandlerSingleton *WasmDOMHandler

func getOrCreateDOMHandler() *WasmDOMHandler {
	if wasmDOMHandlerSingleton == nil {
		wasmDOMHandlerSingleton = &WasmDOMHandler{}
		replInstance.repl.SetDOMHandler(wasmDOMHandlerSingleton)
	}
	return wasmDOMHandlerSingleton
}

// setDOMApplyPatchHandler: ailangSetDOMApplyPatchHandler(fn)
func setDOMApplyPatchHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetDOMApplyPatchHandler requires 1 argument: callback function",
		}
	}
	getOrCreateDOMHandler().applyPatchCallback = args[0]
	return map[string]interface{}{"success": true}
}

// setDOMApplyBatchHandler: ailangSetDOMApplyBatchHandler(fn)
func setDOMApplyBatchHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetDOMApplyBatchHandler requires 1 argument: callback function",
		}
	}
	getOrCreateDOMHandler().applyBatchCallback = args[0]
	return map[string]interface{}{"success": true}
}

// ============================================================================
// Cognitive OS — Msg effect bridge (M-COG-RUNTIME, v0.21.x)
// ============================================================================
//
// WasmMsgHandler implements effects.MsgHandler via JS callbacks. The JS
// host decides the transport (LocalWorker / BroadcastChannel / future
// WebSocket) — Go-side just calls the callback. This is the "transport-
// independence" property at runtime: agent code doesn't change between
// transports because the handler interface is uniform.

// WasmMsgHandler implements effects.MsgHandler.
type WasmMsgHandler struct {
	sendCallback      js.Value
	recvCallback      js.Value
	subscribeCallback js.Value
}

// Send invokes the JS send callback. JS callback signature:
//
//	(to, payload) => ({msg_id: "...", clock: N, budget_remaining: N})
func (h *WasmMsgHandler) Send(to effects.Mailbox, payload []byte) (*effects.SendResult, error) {
	if !h.sendCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: Msg.send requires a JS handler — call ailangSetMsgSendHandler(fn) first")
	}
	result := h.sendCallback.Invoke(string(to), string(payload))
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return &effects.SendResult{
		MsgID:           effects.MsgID(jsGetString(resolved, "msg_id")),
		Clock:           effects.LamportClock(jsGetInt(resolved, "clock")),
		BudgetRemaining: jsGetInt(resolved, "budget_remaining"),
	}, nil
}

// Recv invokes the JS recv callback. JS callback signature:
//
//	(mailbox) => ({msg_id, from, to, payload, clock}) | Promise<...>
//
// A Promise return lets the host block until a message arrives in the
// async case (BroadcastChannel, WebSocket). The Go side awaits via
// awaitJSResult, which handles both sync and async.
func (h *WasmMsgHandler) Recv(mailbox effects.Mailbox) (*effects.Message, error) {
	if !h.recvCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: Msg.recv requires a JS handler — call ailangSetMsgRecvHandler(fn) first")
	}
	result := h.recvCallback.Invoke(string(mailbox))
	resolved, err := awaitJSResult(result)
	if err != nil {
		return nil, err
	}
	return &effects.Message{
		ID:      effects.MsgID(jsGetString(resolved, "msg_id")),
		From:    effects.NodeID(jsGetString(resolved, "from")),
		To:      effects.Mailbox(jsGetString(resolved, "to")),
		Payload: []byte(jsGetString(resolved, "payload")),
		Clock:   effects.LamportClock(jsGetInt(resolved, "clock")),
	}, nil
}

// Subscribe is deferred to M2 alongside the deterministic scheduler —
// same rationale as WasmDOMHandler.Subscribe.
func (h *WasmMsgHandler) Subscribe(mailbox effects.Mailbox, onMsg func(effects.Message)) (func(), error) {
	return nil, fmt.Errorf("not_implemented: Msg.subscribe wiring lands in M2 alongside the deterministic scheduler")
}

var wasmMsgHandlerSingleton *WasmMsgHandler

func getOrCreateMsgHandler() *WasmMsgHandler {
	if wasmMsgHandlerSingleton == nil {
		wasmMsgHandlerSingleton = &WasmMsgHandler{}
		// Self defaults to "wasm_repl"; can be overridden via a future
		// ailangSetMsgSelf setter when multi-instance topologies arrive.
		replInstance.repl.SetMsgHandler(wasmMsgHandlerSingleton, effects.NodeID("wasm_repl"))
	}
	return wasmMsgHandlerSingleton
}

// setMsgSendHandler: ailangSetMsgSendHandler(fn)
func setMsgSendHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetMsgSendHandler requires 1 argument: callback function",
		}
	}
	getOrCreateMsgHandler().sendCallback = args[0]
	return map[string]interface{}{"success": true}
}

// setMsgRecvHandler: ailangSetMsgRecvHandler(fn)
func setMsgRecvHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetMsgRecvHandler requires 1 argument: callback function",
		}
	}
	getOrCreateMsgHandler().recvCallback = args[0]
	return map[string]interface{}{"success": true}
}
