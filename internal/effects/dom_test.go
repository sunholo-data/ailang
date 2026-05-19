package effects

import (
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Op registration — proves init() wired the registry
// ============================================================================

func TestDOMOpsRegistered(t *testing.T) {
	required := []string{"applyPatch", "applyBatch"}
	domOps, ok := Registry["DOM"]
	if !ok {
		t.Fatalf("Registry[\"DOM\"] not initialized — init() did not register ops")
	}
	for _, op := range required {
		if _, ok := domOps[op]; !ok {
			t.Errorf("Registry[\"DOM\"][%q] missing — RegisterOp call absent from init()", op)
		}
	}
}

// ============================================================================
// Nil-handler behavior — typed error, not panic
// ============================================================================

func TestDOMContext_NilHandler_ReturnsErr(t *testing.T) {
	// nil DOMContext (the typical case in native CLI runtimes)
	var ctx *DOMContext
	_, err := ctx.ApplyPatch("region", PatchAddPanel{Title: "t"})
	if err == nil {
		t.Fatal("expected ErrNoDOMHandler from nil context, got nil")
	}
	if err != ErrNoDOMHandler {
		t.Errorf("expected ErrNoDOMHandler, got %v", err)
	}
}

func TestDOMContext_NilUnderlyingHandler_ReturnsErr(t *testing.T) {
	// non-nil context but nil handler (defensive case)
	ctx := &DOMContext{handler: nil}
	_, err := ctx.ApplyPatch("region", PatchAddPanel{})
	if err != ErrNoDOMHandler {
		t.Errorf("expected ErrNoDOMHandler, got %v", err)
	}
	_, err = ctx.ApplyBatch("region", nil)
	if err != ErrNoDOMHandler {
		t.Errorf("expected ErrNoDOMHandler from ApplyBatch, got %v", err)
	}
	_, err = ctx.Subscribe("region", nil, nil)
	if err != ErrNoDOMHandler {
		t.Errorf("expected ErrNoDOMHandler from Subscribe, got %v", err)
	}
}

// ============================================================================
// StubDOMHandler — records patches and supports event simulation
// ============================================================================

func TestStubDOMHandler_ApplyPatch_AssignsSequentialNodeID(t *testing.T) {
	h := NewStubDOMHandler()
	r1, err := h.ApplyPatch("agent_a", PatchAddPanel{Title: "First"})
	if err != nil {
		t.Fatalf("ApplyPatch 1 failed: %v", err)
	}
	r2, err := h.ApplyPatch("agent_a", PatchAddPanel{Title: "Second"})
	if err != nil {
		t.Fatalf("ApplyPatch 2 failed: %v", err)
	}
	if r1.NodeID == r2.NodeID {
		t.Errorf("expected distinct node IDs across calls, got %q == %q", r1.NodeID, r2.NodeID)
	}
	if len(h.Applied) != 2 {
		t.Errorf("expected 2 recorded patches, got %d", len(h.Applied))
	}
	if h.Applied[0].Region != "agent_a" {
		t.Errorf("expected region recorded, got %q", h.Applied[0].Region)
	}
}

func TestStubDOMHandler_ApplyBatch_AssignsNIDs(t *testing.T) {
	h := NewStubDOMHandler()
	patches := []DOMPatch{
		PatchAddPanel{Title: "p1"},
		PatchAddPanel{Title: "p2"},
		PatchAddPanel{Title: "p3"},
	}
	res, err := h.ApplyBatch("agent_a", patches)
	if err != nil {
		t.Fatalf("ApplyBatch failed: %v", err)
	}
	if len(res.NodeIDs) != 3 {
		t.Fatalf("expected 3 node IDs, got %d", len(res.NodeIDs))
	}
	// All distinct
	seen := map[DOMNodeID]bool{}
	for _, id := range res.NodeIDs {
		if seen[id] {
			t.Errorf("duplicate node ID %q", id)
		}
		seen[id] = true
	}
}

func TestStubDOMHandler_Subscribe_FireEvent(t *testing.T) {
	h := NewStubDOMHandler()
	received := []DOMEvent{}
	var mu sync.Mutex

	cancel, err := h.Subscribe("agent_a", []string{"click"}, func(e DOMEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	h.FireEvent("agent_a", EventClick{Node: "node_42"})
	h.FireEvent("agent_a", EventInput{Node: "node_43", Value: "hello"})
	h.FireEvent("other_region", EventClick{Node: "node_44"}) // should not deliver

	mu.Lock()
	got := len(received)
	mu.Unlock()
	if got != 2 {
		t.Errorf("expected 2 events delivered to agent_a subscriber, got %d", got)
	}

	// Cancel — further events should not fire
	cancel()
	h.FireEvent("agent_a", EventClick{Node: "node_45"})
	mu.Lock()
	got = len(received)
	mu.Unlock()
	if got != 2 {
		t.Errorf("expected cancel to stop delivery, got %d events after cancel", got)
	}
}

// ============================================================================
// Codecs — AILANG variant <-> Go DOMPatch round-trip via ops
// ============================================================================

func TestDOMApplyPatch_AddPanel_RoundTrip(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	patch := &eval.TaggedValue{
		CtorName: "AddPanel",
		Fields: []eval.Value{
			&eval.StringValue{Value: "Failure Heatmap"},
			&eval.StringValue{Value: "<svg>...</svg>"},
		},
	}
	result, err := domApplyPatch(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		patch,
	})
	if err != nil {
		t.Fatalf("domApplyPatch failed: %v", err)
	}

	rec, ok := result.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", result)
	}
	if _, ok := rec.Fields["node_id"]; !ok {
		t.Error("result missing node_id field")
	}
	if _, ok := rec.Fields["budget_remaining"]; !ok {
		t.Error("result missing budget_remaining field")
	}

	// Verify handler observed the typed patch
	if len(h.Applied) != 1 {
		t.Fatalf("expected handler to record 1 patch, got %d", len(h.Applied))
	}
	addPanel, ok := h.Applied[0].Patch.(PatchAddPanel)
	if !ok {
		t.Fatalf("expected PatchAddPanel, got %T", h.Applied[0].Patch)
	}
	if addPanel.Title != "Failure Heatmap" {
		t.Errorf("title not decoded: %q", addPanel.Title)
	}
	if addPanel.Content != "<svg>...</svg>" {
		t.Errorf("content not decoded: %q", addPanel.Content)
	}
}

func TestDOMApplyPatch_UpdateNode_RoundTrip(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	patch := &eval.TaggedValue{
		CtorName: "UpdateNode",
		Fields: []eval.Value{
			&eval.StringValue{Value: "node_99"},
			&eval.StringValue{Value: "new content"},
		},
	}
	_, err := domApplyPatch(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		patch,
	})
	if err != nil {
		t.Fatalf("domApplyPatch failed: %v", err)
	}
	upd, ok := h.Applied[0].Patch.(PatchUpdateNode)
	if !ok {
		t.Fatalf("expected PatchUpdateNode, got %T", h.Applied[0].Patch)
	}
	if upd.Node != "node_99" {
		t.Errorf("node not decoded: %q", upd.Node)
	}
	if upd.Content != "new content" {
		t.Errorf("content not decoded: %q", upd.Content)
	}
}

func TestDOMApplyPatch_RemoveNode_RoundTrip(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	_, err := domApplyPatch(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		&eval.TaggedValue{
			CtorName: "RemoveNode",
			Fields:   []eval.Value{&eval.StringValue{Value: "node_77"}},
		},
	})
	if err != nil {
		t.Fatalf("domApplyPatch failed: %v", err)
	}
	rm, ok := h.Applied[0].Patch.(PatchRemoveNode)
	if !ok {
		t.Fatalf("expected PatchRemoveNode, got %T", h.Applied[0].Patch)
	}
	if rm.Node != "node_77" {
		t.Errorf("node not decoded: %q", rm.Node)
	}
}

func TestDOMApplyPatch_UnknownVariant_ReturnsError(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	_, err := domApplyPatch(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		&eval.TaggedValue{CtorName: "BogusPatch", Fields: nil},
	})
	if err == nil {
		t.Fatal("expected error for unknown variant, got nil")
	}
}

func TestDOMApplyPatch_WrongArgCount_ReturnsError(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	_, err := domApplyPatch(ctx, []eval.Value{&eval.StringValue{Value: "region"}})
	if err == nil {
		t.Fatal("expected error for missing patch arg, got nil")
	}
}

func TestDOMApplyBatch_RoundTrip(t *testing.T) {
	h := NewStubDOMHandler()
	ctx := &EffContext{DOM: NewDOMContext(h)}

	patches := &eval.ListValue{
		Elements: []eval.Value{
			&eval.TaggedValue{CtorName: "AddPanel", Fields: []eval.Value{
				&eval.StringValue{Value: "Panel A"},
				&eval.StringValue{Value: "content a"},
			}},
			&eval.TaggedValue{CtorName: "AddTimeline", Fields: []eval.Value{
				&eval.StringValue{Value: "Timeline X"},
			}},
		},
	}
	result, err := domApplyBatch(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		patches,
	})
	if err != nil {
		t.Fatalf("domApplyBatch failed: %v", err)
	}
	rec := result.(*eval.RecordValue)
	idList := rec.Fields["node_ids"].(*eval.ListValue)
	if len(idList.Elements) != 2 {
		t.Errorf("expected 2 node IDs in result, got %d", len(idList.Elements))
	}
	if len(h.Batches) != 1 {
		t.Fatalf("expected 1 recorded batch, got %d", len(h.Batches))
	}
	if len(h.Batches[0].Patches) != 2 {
		t.Errorf("batch should have 2 patches, got %d", len(h.Batches[0].Patches))
	}
}
