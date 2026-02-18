//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
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
		// Object/array fallback: convert to string
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
