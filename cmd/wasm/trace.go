//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"

	"github.com/sunholo/ailang/internal/trace"
)

// setTraceHandler: ailangSetTraceHandler(callback)
// Registers a JS function that receives trace events in real-time during AILANG execution.
// Each event is delivered as a JS object with fields matching the trace.TraceEvent schema.
// Pass null/undefined to unregister the handler.
func setTraceHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetTraceHandler requires 1 argument: callback function (or null to unregister)",
		}
	}

	ctx := replInstance.repl.GetEffContext()
	if ctx == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "effect context not initialized",
		}
	}

	// Ensure a trace collector exists on the EffContext
	if ctx.Trace == nil {
		ctx.Trace = trace.NewCollector()
	}

	arg := args[0]
	if arg.Type() == js.TypeNull || arg.Type() == js.TypeUndefined {
		// Unregister handler
		ctx.Trace.OnEvent = nil
		return map[string]interface{}{"success": true}
	}

	if arg.Type() != js.TypeFunction {
		return map[string]interface{}{
			"success": false,
			"error":   "argument must be a function, null, or undefined",
		}
	}

	callback := arg
	ctx.Trace.OnEvent = func(evt trace.TraceEvent) {
		jsEvt := traceEventToJS(evt)
		callback.Invoke(jsEvt)
	}

	return map[string]interface{}{"success": true}
}

// traceEventToJS converts a TraceEvent to a JS object using direct field setting.
func traceEventToJS(evt trace.TraceEvent) map[string]interface{} {
	obj := map[string]interface{}{
		"version":      evt.Version,
		"event":        string(evt.Event),
		"timestamp_ns": evt.TimestampNS,
		"depth":        evt.Depth,
	}
	if evt.TraceID != "" {
		obj["trace_id"] = evt.TraceID
	}
	if evt.SpanID != "" {
		obj["span_id"] = evt.SpanID
	}
	if evt.ParentSpanID != "" {
		obj["parent_span_id"] = evt.ParentSpanID
	}

	if evt.Module != nil {
		mod := map[string]interface{}{
			"name": evt.Module.Name,
		}
		if evt.Module.DurationNS != 0 {
			mod["duration_ns"] = evt.Module.DurationNS
		}
		if evt.Module.Caps != nil {
			caps := make([]interface{}, len(evt.Module.Caps))
			for i, c := range evt.Module.Caps {
				caps[i] = c
			}
			mod["caps"] = caps
		}
		obj["module"] = mod
	}

	if evt.Function != nil {
		fn := map[string]interface{}{
			"name": evt.Function.Name,
		}
		if evt.Function.Args != nil {
			fnArgs := make([]interface{}, len(evt.Function.Args))
			for i, a := range evt.Function.Args {
				fnArgs[i] = a
			}
			fn["args"] = fnArgs
		}
		if evt.Function.Result != "" {
			fn["result"] = evt.Function.Result
		}
		if evt.Function.DurationNS != 0 {
			fn["duration_ns"] = evt.Function.DurationNS
		}
		obj["function"] = fn
	}

	if evt.Effect != nil {
		eff := map[string]interface{}{
			"effect_name": evt.Effect.EffectName,
			"op_name":     evt.Effect.OpName,
		}
		if evt.Effect.Args != nil {
			effArgs := make([]interface{}, len(evt.Effect.Args))
			for i, a := range evt.Effect.Args {
				effArgs[i] = a
			}
			eff["args"] = effArgs
		}
		if evt.Effect.Result != "" {
			eff["result"] = evt.Effect.Result
		}
		if evt.Effect.Deterministic != nil {
			eff["deterministic"] = *evt.Effect.Deterministic
		}
		obj["effect"] = eff
	}

	if evt.Contract != nil {
		obj["contract"] = map[string]interface{}{
			"kind":     evt.Contract.Kind,
			"passed":   evt.Contract.Passed,
			"message":  evt.Contract.Message,
			"location": evt.Contract.Location,
			"function": evt.Contract.Function,
		}
	}

	if evt.Budget != nil {
		obj["budget"] = map[string]interface{}{
			"effect":    evt.Budget.Effect,
			"used":      evt.Budget.Used,
			"limit":     evt.Budget.Limit,
			"remaining": evt.Budget.Remaining,
			"physical":  evt.Budget.Physical,
		}
	}

	if evt.Error != nil {
		errObj := map[string]interface{}{
			"message": evt.Error.Message,
		}
		if evt.Error.Location != "" {
			errObj["location"] = evt.Error.Location
		}
		obj["error"] = errObj
	}

	return obj
}
