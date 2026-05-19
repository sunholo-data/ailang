package effects

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// DOM Effect — Cognitive OS substrate (M-COG-RUNTIME, v0.21.x)
// ============================================================================
//
// The DOM effect provides scoped, structured browser-DOM mutation + event
// subscription for the reflective Cognitive OS runtime. The handler interface
// mirrors the step-pattern established by AIHandler (Call/Step/StepWithStream)
// in ai.go — see design_docs/planned/v0_21_0/m-cog-runtime.md for rationale.
//
// Key design principles (locked by the umbrella design freeze):
//   - Scoped regions only: agents address DOM by RegionID, not raw element IDs
//   - Structured patches: typed ADT, no raw HTML/JS injection
//   - Subscribe streams events back into AILANG — makes the runtime reflective
//   - Pluggable handler: cmd/wasm/effects.go sets a JS-backed handler at runtime;
//     internal/effects/dom.go is platform-neutral.
//
// Cognitive event log persistence (PatchApplied / DOMEvent events) lands in M2;
// here we just expose the handler interface + stub for tests.

// ErrNoDOMHandler is returned when DOM ops are invoked without a configured handler.
//
// Configuration paths:
//   - Browser (WASM): set by cmd/wasm/effects.go via the WasmREPL init
//   - Tests: ctx.DOM = NewDOMContext(NewStubDOMHandler())
//   - Native CLI: not configured → ops return this error (DOM is browser-only)
var ErrNoDOMHandler = errors.New("no DOM handler configured — DOM is only available in WASM/browser runtimes; configure a StubDOMHandler for tests")

// RegionID names an agent-scoped DOM region. Each agent in a multi-agent
// topology has its own region; agents cannot patch outside their region.
type RegionID string

// DOMNodeID names a node within a DOM region. Assigned by the handler when
// patches are applied — content-hashed in the browser host (M3) for replay
// determinism. The stub uses sequential IDs for test determinism.
type DOMNodeID string

// ============================================================================
// DOMPatch ADT — structured patches (no raw HTML/JS)
// ============================================================================

// DOMPatch is the sum type of all structured DOM mutations agents can express.
// New variants must be enumerated in decodeDOMPatch and stdlib/std/dom.ail.
type DOMPatch interface {
	isDOMPatch()
}

// PatchAddPanel creates a new titled panel in the agent's region.
type PatchAddPanel struct {
	Title   string
	Content string
}

func (PatchAddPanel) isDOMPatch() {}

// PatchUpdateNode replaces the content of an existing node.
type PatchUpdateNode struct {
	Node    DOMNodeID
	Content string
}

func (PatchUpdateNode) isDOMPatch() {}

// PatchRemoveNode removes a node by ID.
type PatchRemoveNode struct {
	Node DOMNodeID
}

func (PatchRemoveNode) isDOMPatch() {}

// PatchAddTimeline creates a timeline visualization (for trace replay etc.).
type PatchAddTimeline struct {
	Title string
}

func (PatchAddTimeline) isDOMPatch() {}

// PatchResult is the typed return of ApplyPatch.
//
// DOMNodeID is the handler-assigned identifier (content-hashed in browser host
// for replay determinism; sequential in stub). BudgetRemaining = -1 means
// unbounded (no budget configured); 0 means the next call will trap.
type PatchResult struct {
	NodeID          DOMNodeID
	BudgetRemaining int
}

// BatchResult is the typed return of ApplyBatch — N node IDs for N patches.
//
// ApplyBatch is atomic: either all patches apply or none do. A partial-failure
// shape would defeat the replay-determinism guarantee.
type BatchResult struct {
	NodeIDs         []DOMNodeID
	BudgetRemaining int
}

// ============================================================================
// DOMEvent ADT — events streamed back via Subscribe
// ============================================================================

// DOMEvent is the sum type of events that flow from the browser back into
// AILANG via DOMHandler.Subscribe. New variants must be enumerated in
// encodeDOMEvent and stdlib/std/dom.ail.
type DOMEvent interface {
	isDOMEvent()
}

// EventClick fires when a user clicks a node.
type EventClick struct {
	Node DOMNodeID
}

func (EventClick) isDOMEvent() {}

// EventInput fires when input field content changes.
type EventInput struct {
	Node  DOMNodeID
	Value string
}

func (EventInput) isDOMEvent() {}

// ============================================================================
// DOMHandler — pluggable step-pattern interface
// ============================================================================

// DOMHandler is the runtime contract for DOM mutation + event subscription.
//
// The interface mirrors the AIHandler step pattern (ai.go:29) — see
// "Handler Interfaces — Step Pattern Alignment" in the design doc.
//
// Subscribe is the critical asymmetry-breaker vs. simpler Call patterns:
// it makes the runtime reflective by streaming DOM events back into AILANG.
// Without Subscribe, the DOM is write-only and the runtime can't observe
// its own environment.
type DOMHandler interface {
	// ApplyPatch applies a single patch atomically and returns the assigned
	// NodeId + remaining budget. Returns an error for malformed patches or
	// budget exhaustion (which surfaces as a typed CapabilityExceeded event
	// in the cognitive event log — M2).
	ApplyPatch(region RegionID, patch DOMPatch) (*PatchResult, error)

	// ApplyBatch applies a sequence transactionally — all patches succeed or
	// none do. Mirrors the multi-message shape of AIHandler.Step.
	ApplyBatch(region RegionID, patches []DOMPatch) (*BatchResult, error)

	// Subscribe registers a long-lived callback that fires for each DOM event
	// matching eventTypes in the named region. The returned cancel function
	// unregisters the callback and must be called to avoid resource leaks
	// (browser host holds a JS reference for the callback's lifetime).
	//
	// Analogous to AIHandler.StepWithStream's onChunk, but long-lived rather
	// than per-call.
	Subscribe(region RegionID, eventTypes []string, onEvent func(DOMEvent)) (cancel func(), err error)
}

// ============================================================================
// DOMContext — stored on EffContext.DOM
// ============================================================================

// DOMContext holds the active DOM handler. Created once per evaluation and
// attached to EffContext (see context.go: EffContext.DOM).
//
// Thread-safety: DOMContext is designed for single-threaded use within one
// evaluator. Concurrent calls require the handler to provide its own
// synchronization.
type DOMContext struct {
	handler DOMHandler
}

// NewDOMContext creates a context with the given handler.
func NewDOMContext(handler DOMHandler) *DOMContext {
	return &DOMContext{handler: handler}
}

// ApplyPatch delegates to the configured handler with a nil-handler check.
func (c *DOMContext) ApplyPatch(region RegionID, patch DOMPatch) (*PatchResult, error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoDOMHandler
	}
	return c.handler.ApplyPatch(region, patch)
}

// ApplyBatch delegates to the configured handler with a nil-handler check.
func (c *DOMContext) ApplyBatch(region RegionID, patches []DOMPatch) (*BatchResult, error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoDOMHandler
	}
	return c.handler.ApplyBatch(region, patches)
}

// Subscribe delegates to the configured handler with a nil-handler check.
func (c *DOMContext) Subscribe(region RegionID, eventTypes []string, onEvent func(DOMEvent)) (func(), error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoDOMHandler
	}
	return c.handler.Subscribe(region, eventTypes, onEvent)
}

// ============================================================================
// StubDOMHandler — deterministic in-memory handler for tests
// ============================================================================

// StubDOMHandler is a deterministic in-memory DOMHandler for tests.
//
// Mirrors StubAIHandler (ai.go:226): records every patch + batch applied so
// tests can assert handler behavior, assigns sequential node IDs for replay
// determinism, and provides FireEvent for simulating browser events.
type StubDOMHandler struct {
	mu          sync.Mutex
	Applied     []StubAppliedPatch
	Batches     []StubAppliedBatch
	nextNodeID  int
	subscribers []*stubSubscriber
}

// StubAppliedPatch records one ApplyPatch call.
type StubAppliedPatch struct {
	Region RegionID
	Patch  DOMPatch
}

// StubAppliedBatch records one ApplyBatch call.
type StubAppliedBatch struct {
	Region  RegionID
	Patches []DOMPatch
}

type stubSubscriber struct {
	region     RegionID
	eventTypes []string
	onEvent    func(DOMEvent)
}

// NewStubDOMHandler creates an empty stub handler.
func NewStubDOMHandler() *StubDOMHandler {
	return &StubDOMHandler{}
}

// ApplyPatch records the call and returns a sequential node ID.
func (h *StubDOMHandler) ApplyPatch(region RegionID, patch DOMPatch) (*PatchResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Applied = append(h.Applied, StubAppliedPatch{Region: region, Patch: patch})
	h.nextNodeID++
	return &PatchResult{
		NodeID:          DOMNodeID(fmt.Sprintf("stub_node_%d", h.nextNodeID)),
		BudgetRemaining: -1, // -1 = unbounded in stub
	}, nil
}

// ApplyBatch records the call and assigns sequential node IDs.
func (h *StubDOMHandler) ApplyBatch(region RegionID, patches []DOMPatch) (*BatchResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Batches = append(h.Batches, StubAppliedBatch{Region: region, Patches: patches})
	ids := make([]DOMNodeID, len(patches))
	for i := range patches {
		h.nextNodeID++
		ids[i] = DOMNodeID(fmt.Sprintf("stub_node_%d", h.nextNodeID))
	}
	return &BatchResult{NodeIDs: ids, BudgetRemaining: -1}, nil
}

// Subscribe registers the callback. The returned cancel function removes it.
func (h *StubDOMHandler) Subscribe(region RegionID, eventTypes []string, onEvent func(DOMEvent)) (func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub := &stubSubscriber{region: region, eventTypes: eventTypes, onEvent: onEvent}
	h.subscribers = append(h.subscribers, sub)
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		for i, s := range h.subscribers {
			if s == sub {
				h.subscribers = append(h.subscribers[:i], h.subscribers[i+1:]...)
				return
			}
		}
	}
	return cancel, nil
}

// FireEvent dispatches an event to all matching subscribers — for tests.
//
// Only subscribers whose region matches receive the event. Event-type
// filtering is left to subscribers (the stub fires unconditionally; the
// real handler will filter in the browser before crossing the JS-Go boundary).
func (h *StubDOMHandler) FireEvent(region RegionID, event DOMEvent) {
	h.mu.Lock()
	subs := make([]*stubSubscriber, len(h.subscribers))
	copy(subs, h.subscribers)
	h.mu.Unlock()
	for _, s := range subs {
		if s.region == region {
			s.onEvent(event)
		}
	}
}

// ============================================================================
// Ops — registered with the effect registry; called from AILANG code
// ============================================================================

// init registers DOM effect operations.
//
// Bare ops give raw Go-error semantics. Result-returning variants
// (applyPatchResult / applyBatchResult) wrap success and failure into
// AILANG Result[T, CognitionError] — mirroring the aiCallResult /
// aiStepResult pattern in ai_step.go so AILANG callers get a uniform
// shape instead of raw panics on missing handlers.
func init() {
	RegisterOp("DOM", "applyPatch", domApplyPatch)
	RegisterOp("DOM", "applyBatch", domApplyBatch)
	RegisterOp("DOM", "applyPatchResult", domApplyPatchResult)
	RegisterOp("DOM", "applyBatchResult", domApplyBatchResult)
	RegisterOp("DOM", "subscribe", domSubscribe)
}

// ============================================================================
// Subscribe op — M-COG-RUNTIME-BROWSER M4
// ============================================================================
//
// AILANG signature (in std/dom.ail):
//
//	subscribe(region: string, eventTypes: [string], callback: (DOMEvent) -> ()) -> () ! DOM
//
// Wires the AILANG callback through the underlying DOMHandler, with
// arrival events queued onto ctx.Cog for later dispatch via _cog_drain.
// The cancel function (returned by handler.Subscribe) is currently
// retained on the handler side; M5 may expose it as an explicit
// unsubscribe builtin if/when cancellation matters in AILANG-side code.
//
// Browser path: WasmDOMHandler.Subscribe (cmd/wasm/effects_cognition.go)
// registers a JS-side onmessage callback; when DOM events fire in the
// scoped region, the JS host enqueues envelopes that the Go-side
// onEvent closure relays into ctx.Cog.Enqueue.
//
// Native path: StubDOMHandler can fire events via its FireEvent helper;
// onEvent is invoked synchronously from the same goroutine.
func domSubscribe(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: subscribe: expected 3 args (region, eventTypes, callback), got %d", len(args))
	}
	region, err := domStringArg(args[0], "region")
	if err != nil {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: subscribe: %w", err)
	}
	eventTypesVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: subscribe: expected eventTypes as list[string], got %T", args[1])
	}
	eventTypes := make([]string, 0, len(eventTypesVal.Elements))
	for i, e := range eventTypesVal.Elements {
		s, ok := e.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("E_DOM_TYPE_ERROR: subscribe: eventTypes[%d] not a string", i)
		}
		eventTypes = append(eventTypes, s.Value)
	}
	callback := args[2] // AILANG closure (eval.Value); passed verbatim to ctx.Cog.Enqueue
	if ctx.Cog == nil {
		return nil, fmt.Errorf("E_DOM_NO_COG: subscribe: ctx.Cog not configured — wire one via NewCogContext()")
	}

	// onEvent runs on whatever goroutine fires the event (transport,
	// JS bridge, test harness). We forward to ctx.Cog.Enqueue which is
	// goroutine-safe; the AILANG callback then fires in _cog_drain on
	// the evaluator's goroutine.
	onEvent := func(ev DOMEvent) {
		ctx.Cog.Enqueue(callback, encodeDOMEvent(ev))
	}
	if _, err := ctx.DOM.Subscribe(RegionID(region), eventTypes, onEvent); err != nil {
		return nil, err
	}
	return &eval.UnitValue{}, nil
}

// encodeDOMEvent serializes a DOMEvent into an AILANG record for callback
// invocation. Variant shape:
//
//	EventClick     -> {kind: "Click", node: string}
//	EventInput     -> {kind: "Input", node: string, value: string}
func encodeDOMEvent(ev DOMEvent) eval.Value {
	switch v := ev.(type) {
	case EventClick:
		return &eval.RecordValue{
			Fields: map[string]eval.Value{
				"kind": &eval.StringValue{Value: "Click"},
				"node": &eval.StringValue{Value: string(v.Node)},
			},
		}
	case EventInput:
		return &eval.RecordValue{
			Fields: map[string]eval.Value{
				"kind":  &eval.StringValue{Value: "Input"},
				"node":  &eval.StringValue{Value: string(v.Node)},
				"value": &eval.StringValue{Value: v.Value},
			},
		}
	default:
		return &eval.RecordValue{
			Fields: map[string]eval.Value{
				"kind": &eval.StringValue{Value: "Unknown"},
			},
		}
	}
}

// domApplyPatch implements DOM.applyPatch(region: string, patch: DOMPatch) -> {node_id: string, budget_remaining: int}
//
// The patch variant is encoded as an AILANG variant value (TaggedValue) with
// CtorName matching the patch type — see decodeDOMPatch for the wire format.
func domApplyPatch(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyPatch: expected 2 arguments, got %d", len(args))
	}
	region, err := domStringArg(args[0], "region")
	if err != nil {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyPatch: %w", err)
	}
	patch, err := decodeDOMPatch(args[1])
	if err != nil {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyPatch: %w", err)
	}
	res, err := ctx.DOM.ApplyPatch(RegionID(region), patch)
	if err != nil {
		return nil, err
	}
	return encodePatchResult(res), nil
}

// domApplyBatch implements DOM.applyBatch(region: string, patches: list[DOMPatch]) -> {node_ids: list[string], budget_remaining: int}
func domApplyBatch(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyBatch: expected 2 arguments, got %d", len(args))
	}
	region, err := domStringArg(args[0], "region")
	if err != nil {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyBatch: %w", err)
	}
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyBatch: expected list of patches, got %T", args[1])
	}
	patches := make([]DOMPatch, len(list.Elements))
	for i, e := range list.Elements {
		p, err := decodeDOMPatch(e)
		if err != nil {
			return nil, fmt.Errorf("E_DOM_TYPE_ERROR: applyBatch[%d]: %w", i, err)
		}
		patches[i] = p
	}
	res, err := ctx.DOM.ApplyBatch(RegionID(region), patches)
	if err != nil {
		return nil, err
	}
	return encodeBatchResult(res), nil
}

// ============================================================================
// Codecs — AILANG values <-> Go types
// ============================================================================

// domStringArg extracts a string argument by name. Local helper to avoid
// importing the broader ai_step.go helper (which uses RecordValue fields).
func domStringArg(v eval.Value, name string) (string, error) {
	s, ok := v.(*eval.StringValue)
	if !ok {
		return "", fmt.Errorf("expected %s as string, got %T", name, v)
	}
	return s.Value, nil
}

// decodeDOMPatch reads an AILANG variant value as a DOMPatch.
//
// Wire format: TaggedValue with CtorName matching the patch type and Fields
// holding StringValue/DOMNodeID arguments. New patch variants must be added here
// in lockstep with the variant's definition above and the AILANG bindings in
// stdlib/std/dom.ail.
func decodeDOMPatch(v eval.Value) (DOMPatch, error) {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return nil, fmt.Errorf("expected DOMPatch variant, got %T", v)
	}
	switch tv.CtorName {
	case "AddPanel":
		if len(tv.Fields) != 2 {
			return nil, fmt.Errorf("AddPanel: expected 2 fields (title, content), got %d", len(tv.Fields))
		}
		return PatchAddPanel{
			Title:   domStringField(tv.Fields[0]),
			Content: domStringField(tv.Fields[1]),
		}, nil
	case "UpdateNode":
		if len(tv.Fields) != 2 {
			return nil, fmt.Errorf("UpdateNode: expected 2 fields (node, content), got %d", len(tv.Fields))
		}
		return PatchUpdateNode{
			Node:    DOMNodeID(domStringField(tv.Fields[0])),
			Content: domStringField(tv.Fields[1]),
		}, nil
	case "RemoveNode":
		if len(tv.Fields) != 1 {
			return nil, fmt.Errorf("RemoveNode: expected 1 field (node), got %d", len(tv.Fields))
		}
		return PatchRemoveNode{Node: DOMNodeID(domStringField(tv.Fields[0]))}, nil
	case "AddTimeline":
		if len(tv.Fields) != 1 {
			return nil, fmt.Errorf("AddTimeline: expected 1 field (title), got %d", len(tv.Fields))
		}
		return PatchAddTimeline{Title: domStringField(tv.Fields[0])}, nil
	default:
		return nil, fmt.Errorf("unknown DOMPatch variant: %q", tv.CtorName)
	}
}

// domStringField extracts a string from a TaggedValue field; returns ""
// for non-string or nil fields (forgiving — schema rigor is the type
// checker's job).
func domStringField(v eval.Value) string {
	if s, ok := v.(*eval.StringValue); ok {
		return s.Value
	}
	return ""
}

// encodePatchResult builds the AILANG record returned by applyPatch.
func encodePatchResult(r *PatchResult) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"node_id":          &eval.StringValue{Value: string(r.NodeID)},
			"budget_remaining": &eval.IntValue{Value: r.BudgetRemaining},
		},
	}
}

// encodeBatchResult builds the AILANG record returned by applyBatch.
func encodeBatchResult(r *BatchResult) eval.Value {
	elems := make([]eval.Value, len(r.NodeIDs))
	for i, id := range r.NodeIDs {
		elems[i] = &eval.StringValue{Value: string(id)}
	}
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"node_ids":         &eval.ListValue{Elements: elems},
			"budget_remaining": &eval.IntValue{Value: r.BudgetRemaining},
		},
	}
}

// ============================================================================
// Cognitive OS shared Result helpers — used by both DOM and Msg
// ============================================================================
//
// Cognitive OS effects (DOM, Msg, later SharedMem / SemanticSearch) share a
// uniform {code, message} error shape — simpler than AIError because they
// don't carry provider-classification semantics. Result-returning op variants
// produce `Result[T, CognitionError]` where T is the effect-specific success
// record.
//
// Error codes are stable strings; downstream consumers can pattern-match on
// them. The set is intentionally small in M1 and grows as new failure modes
// surface in M2 (CapabilityExceeded, transport errors).

// Cognition error codes (stable string constants for pattern matching).
const (
	CogErrCodeNoHandler        = "NO_HANDLER"        // handler not configured for this runtime
	CogErrCodeInvalidArgs      = "INVALID_ARGS"      // arg shape/type mismatch
	CogErrCodeNoMessage        = "NO_MESSAGE"        // recv on empty mailbox (stub-only currently)
	CogErrCodeInternal         = "INTERNAL"          // catch-all for unexpected handler failures
	CogErrCodeBudgetExceeded   = "BUDGET_EXCEEDED"   // capability budget hit (M2 will set this)
	CogErrCodeTransportFailure = "TRANSPORT_FAILURE" // transport-layer fault (M-COG-MESH)
)

// makeCognitionErrResult wraps a {code, message} record in Err(...).
//
// Shape matches the AILANG side: Result[T, {code: string, message: string}].
// Uses the shared makeErrResult helper (env.go) for the Result tag itself —
// only the inner error record is cognition-specific.
//
// Distinct from makeAIErrorResultRecord (which carries a third "retryable"
// field) — cognitive failures are categorically structural, not provider-
// transient, so retryability isn't a property of the error.
func makeCognitionErrResult(code, message string) eval.Value {
	return makeErrResult(&eval.RecordValue{
		Fields: map[string]eval.Value{
			"code":    &eval.StringValue{Value: code},
			"message": &eval.StringValue{Value: message},
		},
	})
}

// classifyDOMError maps Go errors from the handler into stable error codes.
// Currently only one classification (ErrNoDOMHandler); extends in M2 as
// budget enforcement and transport faults arrive.
func classifyDOMError(err error) (code, message string) {
	switch err {
	case ErrNoDOMHandler:
		return CogErrCodeNoHandler, err.Error()
	default:
		return CogErrCodeInternal, err.Error()
	}
}

// ============================================================================
// Result-returning op variants — Ok(record) / Err({code, message})
// ============================================================================

// domApplyPatchResult implements DOM.applyPatchResult(region, patch)
// -> Result[{node_id, budget_remaining}, {code, message}].
//
// Unlike domApplyPatch, this never returns a raw Go error to the evaluator:
// arg-shape errors, missing handler, and handler failures all surface as
// Err({code, message}) so AILANG callers can match without panicking.
func domApplyPatchResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return makeCognitionErrResult(CogErrCodeInvalidArgs,
			fmt.Sprintf("applyPatchResult: expected 2 arguments, got %d", len(args))), nil
	}
	region, err := domStringArg(args[0], "region")
	if err != nil {
		return makeCognitionErrResult(CogErrCodeInvalidArgs, err.Error()), nil
	}
	patch, err := decodeDOMPatch(args[1])
	if err != nil {
		return makeCognitionErrResult(CogErrCodeInvalidArgs, err.Error()), nil
	}
	res, err := ctx.DOM.ApplyPatch(RegionID(region), patch)
	if err != nil {
		code, msg := classifyDOMError(err)
		return makeCognitionErrResult(code, msg), nil
	}
	return makeOkResult(encodePatchResult(res)), nil
}

// domApplyBatchResult implements DOM.applyBatchResult(region, patches)
// -> Result[{node_ids, budget_remaining}, {code, message}].
func domApplyBatchResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return makeCognitionErrResult(CogErrCodeInvalidArgs,
			fmt.Sprintf("applyBatchResult: expected 2 arguments, got %d", len(args))), nil
	}
	region, err := domStringArg(args[0], "region")
	if err != nil {
		return makeCognitionErrResult(CogErrCodeInvalidArgs, err.Error()), nil
	}
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return makeCognitionErrResult(CogErrCodeInvalidArgs,
			fmt.Sprintf("applyBatchResult: expected list of patches, got %T", args[1])), nil
	}
	patches := make([]DOMPatch, len(list.Elements))
	for i, e := range list.Elements {
		p, err := decodeDOMPatch(e)
		if err != nil {
			return makeCognitionErrResult(CogErrCodeInvalidArgs,
				fmt.Sprintf("applyBatchResult[%d]: %s", i, err.Error())), nil
		}
		patches[i] = p
	}
	res, err := ctx.DOM.ApplyBatch(RegionID(region), patches)
	if err != nil {
		code, msg := classifyDOMError(err)
		return makeCognitionErrResult(code, msg), nil
	}
	return makeOkResult(encodeBatchResult(res)), nil
}
