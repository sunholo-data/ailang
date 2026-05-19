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

// NodeID names a node within a DOM region. Assigned by the handler when
// patches are applied — content-hashed in the browser host (M3) for replay
// determinism. The stub uses sequential IDs for test determinism.
type NodeID string

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
	Node    NodeID
	Content string
}

func (PatchUpdateNode) isDOMPatch() {}

// PatchRemoveNode removes a node by ID.
type PatchRemoveNode struct {
	Node NodeID
}

func (PatchRemoveNode) isDOMPatch() {}

// PatchAddTimeline creates a timeline visualization (for trace replay etc.).
type PatchAddTimeline struct {
	Title string
}

func (PatchAddTimeline) isDOMPatch() {}

// PatchResult is the typed return of ApplyPatch.
//
// NodeID is the handler-assigned identifier (content-hashed in browser host
// for replay determinism; sequential in stub). BudgetRemaining = -1 means
// unbounded (no budget configured); 0 means the next call will trap.
type PatchResult struct {
	NodeID          NodeID
	BudgetRemaining int
}

// BatchResult is the typed return of ApplyBatch — N node IDs for N patches.
//
// ApplyBatch is atomic: either all patches apply or none do. A partial-failure
// shape would defeat the replay-determinism guarantee.
type BatchResult struct {
	NodeIDs         []NodeID
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
	Node NodeID
}

func (EventClick) isDOMEvent() {}

// EventInput fires when input field content changes.
type EventInput struct {
	Node  NodeID
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
		NodeID:          NodeID(fmt.Sprintf("stub_node_%d", h.nextNodeID)),
		BudgetRemaining: -1, // -1 = unbounded in stub
	}, nil
}

// ApplyBatch records the call and assigns sequential node IDs.
func (h *StubDOMHandler) ApplyBatch(region RegionID, patches []DOMPatch) (*BatchResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Batches = append(h.Batches, StubAppliedBatch{Region: region, Patches: patches})
	ids := make([]NodeID, len(patches))
	for i := range patches {
		h.nextNodeID++
		ids[i] = NodeID(fmt.Sprintf("stub_node_%d", h.nextNodeID))
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
// Bare ops (applyPatch, applyBatch) — these are the M1-shippable surface.
// Result-returning variants (applyPatchResult etc.) land in Day 4 alongside
// the manifest generator, when the typed-error shape is locked.
func init() {
	RegisterOp("DOM", "applyPatch", domApplyPatch)
	RegisterOp("DOM", "applyBatch", domApplyBatch)
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
// holding StringValue/NodeID arguments. New patch variants must be added here
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
			Node:    NodeID(domStringField(tv.Fields[0])),
			Content: domStringField(tv.Fields[1]),
		}, nil
	case "RemoveNode":
		if len(tv.Fields) != 1 {
			return nil, fmt.Errorf("RemoveNode: expected 1 field (node), got %d", len(tv.Fields))
		}
		return PatchRemoveNode{Node: NodeID(domStringField(tv.Fields[0]))}, nil
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
