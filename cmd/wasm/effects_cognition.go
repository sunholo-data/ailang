//go:build js && wasm
// +build js,wasm

package main

// Cognitive OS WASM effect bridges (M-COG-RUNTIME, v0.21.x).
//
// Split out from cmd/wasm/effects.go to keep both files under the
// 800-line AI-maintainability threshold per the M-COG-RUNTIME
// sprint-evaluator follow-up. The bridges here:
//
//   - WasmDOMHandler implements effects.DOMHandler via JS callbacks
//   - WasmMsgHandler implements effects.MsgHandler via JS callbacks
//   - Per-handler setter functions wired as JS globals in main.go
//
// Shared helpers (awaitJSResult, jsGetString, jsGetInt, replInstance)
// live in effects.go and effects_helpers.go.

import (
	"fmt"
	"syscall/js"

	"github.com/sunholo-data/ailang/internal/effects"
)

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

// Subscribe registers a long-lived JS callback for DOM events in the
// scoped region. The JS host (host.js) is responsible for installing
// real DOM event listeners; this method bridges arrivals back into Go
// via a js.Func that calls onEvent (which queues into ctx.Cog).
//
// JS callback signature (registered via ailangSetDOMSubscribeHandler):
//
//	(region, eventTypes) => js.Func({type, node, value?}) wrapper
//
// Cancel: returned function releases the js.Func reference + calls
// the host's unsubscribe-by-id endpoint so the JS-side listener
// detaches. Avoids leaking js.Func across many subscribe/cancel cycles.
func (h *WasmDOMHandler) Subscribe(region effects.RegionID, eventTypes []string, onEvent func(effects.DOMEvent)) (func(), error) {
	if !h.subscribeCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: DOM.subscribe requires a JS handler — call ailangSetDOMSubscribeHandler(fn) first")
	}
	// Wrap onEvent in a js.Func that the JS host invokes on each arrival.
	// The wrapper decodes the {type, node, value?} object back into a
	// typed effects.DOMEvent before passing to onEvent.
	bridgeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 || args[0].Type() != js.TypeObject {
			return nil
		}
		ev := args[0]
		kind := jsGetString(ev, "kind")
		node := effects.DOMNodeID(jsGetString(ev, "node"))
		switch kind {
		case "Click":
			onEvent(effects.EventClick{Node: node})
		case "Input":
			onEvent(effects.EventInput{Node: node, Value: jsGetString(ev, "value")})
		}
		return nil
	})

	// Convert eventTypes to a JS array for the registration call.
	jsTypes := make([]interface{}, len(eventTypes))
	for i, t := range eventTypes {
		jsTypes[i] = t
	}
	subID := h.subscribeCallback.Invoke(string(region), js.ValueOf(jsTypes), bridgeFn)

	// Return a cancel that:
	//   1. Asks JS to detach by ID via the same callback's "unsubscribe" arity
	//   2. Releases the js.Func so the GC can reclaim it
	cancel := func() {
		if h.subscribeCallback.Truthy() {
			// Convention: passing a single string sub-id to the same
			// callback signals "unsubscribe this ID". The JS host handles
			// the dual signature.
			h.subscribeCallback.Invoke(subID)
		}
		bridgeFn.Release()
	}
	return cancel, nil
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

// setDOMSubscribeHandler: ailangSetDOMSubscribeHandler(fn)
//
// fn has a dual signature handled by the JS host:
//   - fn(region, eventTypes, bridgeFn) → returns a sub-id string
//   - fn(subId) → detaches the registration
func setDOMSubscribeHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetDOMSubscribeHandler requires 1 argument: callback function",
		}
	}
	getOrCreateDOMHandler().subscribeCallback = args[0]
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

// Subscribe registers a long-lived JS callback for mailbox arrivals.
// The JS host wires the BroadcastChannel.onmessage; this method bridges
// each arrival back into Go via a js.Func that decodes the envelope
// and calls onMsg (which queues into ctx.Cog).
//
// Cancel: releases the js.Func + tells JS to detach the per-mailbox
// listener registration (by passing the returned sub-id back to the
// same callback — same dual-signature convention as WasmDOMHandler).
func (h *WasmMsgHandler) Subscribe(mailbox effects.Mailbox, onMsg func(effects.Message)) (func(), error) {
	if !h.subscribeCallback.Truthy() {
		return nil, fmt.Errorf("no_handler: Msg.subscribe requires a JS handler — call ailangSetMsgSubscribeHandler(fn) first")
	}
	bridgeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 || args[0].Type() != js.TypeObject {
			return nil
		}
		env := args[0]
		onMsg(effects.Message{
			ID:      effects.MsgID(jsGetString(env, "msg_id")),
			From:    effects.NodeID(jsGetString(env, "from")),
			To:      effects.Mailbox(jsGetString(env, "to")),
			Payload: []byte(jsGetString(env, "payload")),
			Clock:   effects.LamportClock(jsGetInt(env, "clock")),
		})
		return nil
	})
	subID := h.subscribeCallback.Invoke(string(mailbox), bridgeFn)
	cancel := func() {
		if h.subscribeCallback.Truthy() {
			h.subscribeCallback.Invoke(subID)
		}
		bridgeFn.Release()
	}
	return cancel, nil
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

// setMsgSubscribeHandler: ailangSetMsgSubscribeHandler(fn)
// Dual signature (same convention as setDOMSubscribeHandler):
//   - fn(mailbox, bridgeFn) → returns sub-id
//   - fn(subId) → detaches
func setMsgSubscribeHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangSetMsgSubscribeHandler requires 1 argument: callback function",
		}
	}
	getOrCreateMsgHandler().subscribeCallback = args[0]
	return map[string]interface{}{"success": true}
}
