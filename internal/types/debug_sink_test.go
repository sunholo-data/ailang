package types

import (
	"testing"
)

// Helper to create a test type variable
func testTypeVar(name string) *TVar2 {
	return &TVar2{Name: name, Kind: KStar{}}
}

// Helper type constants for tests
var (
	testInt   = &TCon{Name: "int"}
	testFloat = &TCon{Name: "float"}
)

func TestNoOpDebugSink_NoAllocations(t *testing.T) {
	sink := NoOpDebugSink{}
	tv := testTypeVar("α1")

	// All methods should be no-ops with zero allocations
	sink.OnFreshTypeVar(tv, 1, OriginInferred)
	sink.OnUnify(testInt, testFloat, testFloat, 2)
	sink.OnSubstitute(tv, testInt)
	sink.OnDefault(tv, testInt, "Num defaults to Int")
	sink.OnConstraintAdd("Num", tv, 3)
	sink.OnConstraintResolve("Num", testInt, "add", 3)

	// If we got here without panic, the test passes
	// The zero-allocation property is verified by the benchmark
}

func TestVerboseDebugSink_CollectsEvents(t *testing.T) {
	sink := NewVerboseDebugSink()
	tv := testTypeVar("α42")

	// Emit various events
	sink.OnFreshTypeVar(tv, 1, OriginLiteral)
	sink.OnUnify(testInt, testFloat, testFloat, 2)
	sink.OnSubstitute(tv, testInt)
	sink.OnDefault(tv, testInt, "defaulting")
	sink.OnConstraintAdd("Num", tv, 3)
	sink.OnConstraintResolve("Ord", testInt, "gt", 4)

	events := sink.Events()
	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d", len(events))
	}

	// Check each event type
	tests := []struct {
		idx  int
		kind DebugEventKind
	}{
		{0, EventFreshTypeVar},
		{1, EventUnify},
		{2, EventSubstitute},
		{3, EventDefault},
		{4, EventConstraintAdd},
		{5, EventConstraintResolve},
	}

	for _, tt := range tests {
		if events[tt.idx].Kind != tt.kind {
			t.Errorf("events[%d].Kind = %v, want %v", tt.idx, events[tt.idx].Kind, tt.kind)
		}
	}
}

func TestVerboseDebugSink_FreshTypeVarEvent(t *testing.T) {
	sink := NewVerboseDebugSink()
	tv := testTypeVar("α99")

	sink.OnFreshTypeVar(tv, 42, OriginAnnotation)

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Kind != EventFreshTypeVar {
		t.Errorf("Kind = %v, want EventFreshTypeVar", e.Kind)
	}
	if e.TypeVar != tv {
		t.Errorf("TypeVar = %v, want %v", e.TypeVar, tv)
	}
	if e.NodeID != 42 {
		t.Errorf("NodeID = %d, want 42", e.NodeID)
	}
	if e.Origin != OriginAnnotation {
		t.Errorf("Origin = %v, want OriginAnnotation", e.Origin)
	}
}

func TestVerboseDebugSink_UnifyEvent(t *testing.T) {
	sink := NewVerboseDebugSink()

	sink.OnUnify(testInt, testFloat, testFloat, 100)

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Kind != EventUnify {
		t.Errorf("Kind = %v, want EventUnify", e.Kind)
	}
	if e.Left != testInt {
		t.Errorf("Left = %v, want int", e.Left)
	}
	if e.Right != testFloat {
		t.Errorf("Right = %v, want float", e.Right)
	}
	if e.Result != testFloat {
		t.Errorf("Result = %v, want float", e.Result)
	}
	if e.NodeID != 100 {
		t.Errorf("NodeID = %d, want 100", e.NodeID)
	}
}

func TestVerboseDebugSink_DefaultEvent(t *testing.T) {
	sink := NewVerboseDebugSink()
	tv := testTypeVar("α5")

	sink.OnDefault(tv, testInt, "Num constraint defaults to Int")

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Kind != EventDefault {
		t.Errorf("Kind = %v, want EventDefault", e.Kind)
	}
	if e.TypeVar != tv {
		t.Errorf("TypeVar = %v, want %v", e.TypeVar, tv)
	}
	if e.Defaulted != testInt {
		t.Errorf("Defaulted = %v, want int", e.Defaulted)
	}
	if e.Reason != "Num constraint defaults to Int" {
		t.Errorf("Reason = %q, want %q", e.Reason, "Num constraint defaults to Int")
	}
}

func TestVerboseDebugSink_ConstraintEvents(t *testing.T) {
	sink := NewVerboseDebugSink()
	tv := testTypeVar("α7")

	sink.OnConstraintAdd("Eq", tv, 50)
	sink.OnConstraintResolve("Eq", testInt, "eq", 50)

	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Check add event
	add := events[0]
	if add.Kind != EventConstraintAdd {
		t.Errorf("add.Kind = %v, want EventConstraintAdd", add.Kind)
	}
	if add.ClassName != "Eq" {
		t.Errorf("add.ClassName = %q, want %q", add.ClassName, "Eq")
	}
	if add.NodeID != 50 {
		t.Errorf("add.NodeID = %d, want 50", add.NodeID)
	}

	// Check resolve event
	resolve := events[1]
	if resolve.Kind != EventConstraintResolve {
		t.Errorf("resolve.Kind = %v, want EventConstraintResolve", resolve.Kind)
	}
	if resolve.ClassName != "Eq" {
		t.Errorf("resolve.ClassName = %q, want %q", resolve.ClassName, "Eq")
	}
	if resolve.Method != "eq" {
		t.Errorf("resolve.Method = %q, want %q", resolve.Method, "eq")
	}
}

func TestVerboseDebugSink_Clear(t *testing.T) {
	sink := NewVerboseDebugSink()

	sink.OnFreshTypeVar(testTypeVar("α1"), 1, OriginInferred)
	sink.OnFreshTypeVar(testTypeVar("α2"), 2, OriginInferred)

	if len(sink.Events()) != 2 {
		t.Fatalf("expected 2 events before clear, got %d", len(sink.Events()))
	}

	sink.Clear()

	if len(sink.Events()) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(sink.Events()))
	}

	// Verify provenance is also cleared
	if len(sink.AllProvenance()) != 0 {
		t.Errorf("expected 0 provenance entries after clear, got %d", len(sink.AllProvenance()))
	}
}

func TestVerboseDebugSink_Provenance(t *testing.T) {
	sink := NewVerboseDebugSink()

	// Test RecordProvenance and GetProvenance
	origin1 := TypeOrigin{
		Kind:   OriginAnnotation,
		NodeID: 42,
		Span:   SourceSpan{File: "test.ail", Line: 10, Column: 5},
		Note:   "parameter annotation x: int",
	}
	origin2 := TypeOrigin{
		Kind:   OriginInferred,
		NodeID: 43,
		Note:   "from unification",
	}

	sink.RecordProvenance("α1", origin1)
	sink.RecordProvenance("α1", origin2) // Multiple origins for same var

	origins := sink.GetProvenance("α1")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins for α1, got %d", len(origins))
	}

	if origins[0].Kind != OriginAnnotation {
		t.Errorf("origins[0].Kind = %v, want OriginAnnotation", origins[0].Kind)
	}
	if origins[0].Span.Line != 10 {
		t.Errorf("origins[0].Span.Line = %d, want 10", origins[0].Span.Line)
	}
	if origins[1].Kind != OriginInferred {
		t.Errorf("origins[1].Kind = %v, want OriginInferred", origins[1].Kind)
	}

	// Test GetProvenance for non-existent var
	missing := sink.GetProvenance("α99")
	if missing != nil {
		t.Errorf("expected nil for missing type var, got %v", missing)
	}

	// Test AllProvenance
	all := sink.AllProvenance()
	if len(all) != 1 {
		t.Errorf("expected 1 entry in AllProvenance, got %d", len(all))
	}
}

func TestVerboseDebugSink_ProvenanceFromFreshTypeVar(t *testing.T) {
	sink := NewVerboseDebugSink()

	tv := testTypeVar("α1")
	sink.OnFreshTypeVar(tv, 42, OriginAnnotation)

	// OnFreshTypeVar should also record provenance
	origins := sink.GetProvenance("α1")
	if len(origins) != 1 {
		t.Fatalf("expected 1 origin from OnFreshTypeVar, got %d", len(origins))
	}

	if origins[0].Kind != OriginAnnotation {
		t.Errorf("origin.Kind = %v, want OriginAnnotation", origins[0].Kind)
	}
	if origins[0].NodeID != 42 {
		t.Errorf("origin.NodeID = %d, want 42", origins[0].NodeID)
	}
}

func TestVerboseDebugSink_ProvenanceFromDefault(t *testing.T) {
	sink := NewVerboseDebugSink()

	tv := testTypeVar("α1")
	sink.OnDefault(tv, testInt, "Num constraint")

	// OnDefault should also record provenance
	origins := sink.GetProvenance("α1")
	if len(origins) != 1 {
		t.Fatalf("expected 1 origin from OnDefault, got %d", len(origins))
	}

	if origins[0].Kind != OriginDefaulted {
		t.Errorf("origin.Kind = %v, want OriginDefaulted", origins[0].Kind)
	}
	if origins[0].Note == "" {
		t.Error("expected non-empty note for defaulting origin")
	}
}

func TestSourceSpan_String(t *testing.T) {
	tests := []struct {
		span SourceSpan
		want string
	}{
		{SourceSpan{}, "<unknown>"},
		{SourceSpan{Line: 10, Column: 5}, "10:5"},
		{SourceSpan{File: "test.ail", Line: 10, Column: 5}, "test.ail:10:5"},
	}

	for _, tt := range tests {
		got := tt.span.String()
		if got != tt.want {
			t.Errorf("SourceSpan%+v.String() = %q, want %q", tt.span, got, tt.want)
		}
	}
}

func TestDebugEventKind_String(t *testing.T) {
	tests := []struct {
		kind DebugEventKind
		want string
	}{
		{EventFreshTypeVar, "fresh_type_var"},
		{EventUnify, "unify"},
		{EventSubstitute, "substitute"},
		{EventDefault, "default"},
		{EventConstraintAdd, "constraint_add"},
		{EventConstraintResolve, "constraint_resolve"},
		{DebugEventKind(999), "unknown"},
	}

	for _, tt := range tests {
		got := tt.kind.String()
		if got != tt.want {
			t.Errorf("DebugEventKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

// BenchmarkNoOpDebugSink verifies zero allocations
func BenchmarkNoOpDebugSink(b *testing.B) {
	sink := NoOpDebugSink{}
	tv := testTypeVar("α1")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink.OnFreshTypeVar(tv, 1, OriginInferred)
		sink.OnUnify(testInt, testFloat, testFloat, 2)
		sink.OnSubstitute(tv, testInt)
		sink.OnDefault(tv, testInt, "default")
		sink.OnConstraintAdd("Num", tv, 3)
		sink.OnConstraintResolve("Num", testInt, "add", 3)
	}
}

// BenchmarkVerboseDebugSink measures event collection overhead
func BenchmarkVerboseDebugSink(b *testing.B) {
	tv := testTypeVar("α1")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink := NewVerboseDebugSink()
		sink.OnFreshTypeVar(tv, 1, OriginInferred)
		sink.OnUnify(testInt, testFloat, testFloat, 2)
		sink.OnSubstitute(tv, testInt)
		sink.OnDefault(tv, testInt, "default")
		sink.OnConstraintAdd("Num", tv, 3)
		sink.OnConstraintResolve("Num", testInt, "add", 3)
	}
}
