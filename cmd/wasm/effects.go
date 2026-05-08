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
			msg = args[0].String()
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

// WasmAIHandler implements effects.AIHandler via a JS callback.
// It bridges the Go AIHandler interface to a JavaScript function
// that can call external APIs (e.g., Gemini Flash via fetch).
type WasmAIHandler struct {
	callback js.Value
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

// Step is not supported in WASM — multi-turn tool-loop requires server-side
// orchestration and is not available in the browser sandbox.
func (h *WasmAIHandler) Step(_ string, _ []ai.Message, _ []ai.ToolSchema) (*ai.Response, error) {
	return nil, fmt.Errorf("ai.step not supported in WASM environment")
}

// StepWithCache is the cache-aware variant (M-AI-PROMPT-CACHING v0.18.4).
// Same status as Step — not supported in WASM. Cache hints are ignored.
func (h *WasmAIHandler) StepWithCache(_ string, _ []ai.Message, _ []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return nil, fmt.Errorf("ai.stepWithCache not supported in WASM environment")
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

// setAIHandler: ailangSetAIHandler(fn)
// Convenience wrapper that configures the AI handler on EffContext.AI
// and also registers the JS callback in the effect registry.
func setAIHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetAIHandler requires 1 argument: callback function",
		}
	}

	callback := args[0]

	// Set the dedicated AI handler (used by aiCall in effects/ai.go)
	handler := &WasmAIHandler{callback: callback}
	replInstance.repl.SetAIHandler(handler)

	return map[string]interface{}{
		"success": true,
	}
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
