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

// Subscribe is intentionally NOT wired in this iteration.
//
// The lifecycle (long-lived JS callback ↔ Go onEvent closure ↔ cancel)
// requires careful management to avoid leaking js.Func references. M2's
// scheduler is the primary consumer; wiring lands when the scheduler
// arrives (M-COG-RUNTIME-BROWSER Phase 4). For now, returns a typed
// no_handler error so callers see a clean failure.
func (h *WasmDOMHandler) Subscribe(region effects.RegionID, eventTypes []string, onEvent func(effects.DOMEvent)) (func(), error) {
	return nil, fmt.Errorf("not_implemented: DOM.subscribe wiring lands in M-COG-RUNTIME-BROWSER alongside the deterministic scheduler")
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

// Subscribe is deferred to M-COG-RUNTIME-BROWSER alongside the JS
// scheduler that consumes it — same rationale as WasmDOMHandler.Subscribe.
func (h *WasmMsgHandler) Subscribe(mailbox effects.Mailbox, onMsg func(effects.Message)) (func(), error) {
	return nil, fmt.Errorf("not_implemented: Msg.subscribe wiring lands in M-COG-RUNTIME-BROWSER alongside the deterministic scheduler")
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
